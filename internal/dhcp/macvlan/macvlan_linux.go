//go:build linux

// Package macvlan creates a per-replica macvlan interface for the DHCP
// listener, ported from Tinkerbell smee's replica-friendly interface setup.
//
// A pod without hostNetwork cannot see DHCP broadcasts on the physical L2
// segment, and multiple replicas binding port 67 would otherwise stomp each
// other. Each replica creates a bridge-mode macvlan on the host's uplink,
// moves it into its own network namespace, and binds the DHCP socket to it
// with SO_BINDTODEVICE (plus SO_REUSEPORT from the DHCP conn helper), so any
// number of replicas can coexist while all seeing broadcast traffic.
//
// Requires CAP_NET_ADMIN, CAP_SYS_ADMIN, and hostPID=true (the host netns is
// entered via /proc/1/ns/net).
package macvlan

import (
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/go-logr/logr"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"golang.org/x/sys/unix"
)

var ifaceNameRe = regexp.MustCompile(`^mv[0-9a-f]{13}$`)

const (
	ifacePrefix = "mv"
	// ifaceAddr is a dummy link-local address so the interface is usable as
	// a packet source.
	ifaceAddr     = "169.254.1.1/32"
	podUIDEnv     = "MY_POD_UID"
	hostNetNSPath = "/proc/1/ns/net"
)

// InterfaceName derives this replica's unique macvlan name from the pod UID
// (Kubernetes downward API, fieldPath metadata.uid): "mv" + first 13 hex
// chars, exactly filling Linux's 15-char IFNAMSIZ budget.
func InterfaceName() (string, error) {
	uid := strings.ReplaceAll(os.Getenv(podUIDEnv), "-", "")
	if uid == "" {
		return "", fmt.Errorf(
			"%s environment variable not set (expose metadata.uid via the downward API)",
			podUIDEnv,
		)
	}
	if len(uid) < 13 {
		return "", fmt.Errorf("%s too short to derive an interface name: %q", podUIDEnv, uid)
	}
	return ifacePrefix + uid[:13], nil
}

// Setup creates the macvlan interface in the host network namespace, moves
// it into this pod's namespace, brings it up, and returns a cleanup function
// that tears it down. sourceIface is the host parent interface; empty means
// auto-detect from the host's default route.
func Setup(log logr.Logger, sourceIface, ifaceName string) (func(), error) {
	// Network namespaces are per OS thread; everything below must stay on
	// one thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	podNS, err := netns.Get()
	if err != nil {
		return nil, fmt.Errorf("get pod netns: %w", err)
	}
	defer podNS.Close()

	hostNS, err := netns.GetFromPath(hostNetNSPath)
	if err != nil {
		return nil, fmt.Errorf("get host netns from %s (requires hostPID=true): %w",
			hostNetNSPath, err)
	}
	defer hostNS.Close()

	if err := netns.Set(hostNS); err != nil {
		return nil, fmt.Errorf("enter host netns: %w", err)
	}
	// Best effort to end up back in the pod namespace regardless of the exit
	// path.
	defer func() { _ = netns.Set(podNS) }()

	purgeStale(log)

	if sourceIface == "" {
		sourceIface, err = defaultRouteInterface()
		if err != nil {
			return nil, fmt.Errorf("auto-detect macvlan source interface: %w", err)
		}
	}

	parent, err := netlink.LinkByName(sourceIface)
	if err != nil {
		return nil, fmt.Errorf("find source interface %q in host netns: %w", sourceIface, err)
	}

	mv := &netlink.Macvlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:        ifaceName,
			ParentIndex: parent.Attrs().Index,
		},
		Mode: netlink.MACVLAN_MODE_BRIDGE,
	}
	if err := netlink.LinkAdd(mv); err != nil {
		return nil, fmt.Errorf("create macvlan %q on %q: %w", ifaceName, sourceIface, err)
	}

	created, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("find created macvlan %q: %w", ifaceName, err)
	}

	if err := netlink.LinkSetNsFd(created, int(podNS)); err != nil {
		_ = netlink.LinkDel(created)
		return nil, fmt.Errorf("move macvlan %q into pod netns: %w", ifaceName, err)
	}

	if err := netns.Set(podNS); err != nil {
		return nil, fmt.Errorf("re-enter pod netns: %w", err)
	}

	podLink, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("find macvlan %q in pod netns: %w", ifaceName, err)
	}
	if err := netlink.LinkSetUp(podLink); err != nil {
		return nil, fmt.Errorf("bring up macvlan %q: %w", ifaceName, err)
	}

	parsed, err := netlink.ParseAddr(ifaceAddr)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", ifaceAddr, err)
	}
	parsed.Flags |= unix.IFA_F_NOPREFIXROUTE
	if err := netlink.AddrAdd(podLink, parsed); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, fmt.Errorf("add %s to macvlan %q: %w", ifaceAddr, ifaceName, err)
	}

	// IFA_F_NOPREFIXROUTE suppresses the 255.255.255.255/32 RTN_BROADCAST
	// entry the kernel would otherwise add to the local routing table.
	// Without it, fib_lookup for 255.255.255.255 constrained to this
	// interface (via SO_BINDTODEVICE) finds nothing and sendmsg fails with
	// EINVAL. Add it explicitly so broadcast DHCP replies succeed while the
	// pod routing table stays free of a 169.254.0.0/16 prefix route.
	bcastRoute := &netlink.Route{
		LinkIndex: podLink.Attrs().Index,
		Dst:       &net.IPNet{IP: net.IPv4bcast, Mask: net.CIDRMask(32, 32)},
		Type:      unix.RTN_BROADCAST,
		Scope:     netlink.SCOPE_LINK,
		Protocol:  unix.RTPROT_KERNEL,
		Table:     unix.RT_TABLE_LOCAL,
	}
	if err := netlink.RouteAdd(bcastRoute); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, fmt.Errorf("add broadcast route for %q: %w", ifaceName, err)
	}

	log.Info("macvlan interface ready",
		"interface", ifaceName, "source", sourceIface, "address", ifaceAddr)

	return func() { teardown(log, ifaceName) }, nil
}

// purgeStale removes leftover mv* interfaces in the host namespace. A live
// macvlan has been moved into a pod namespace; when that namespace is
// destroyed the kernel returns the interface to the parent namespace, so
// anything matching the pattern still in the host namespace is an orphan.
// Must be called with the current thread in the host namespace.
func purgeStale(log logr.Logger) {
	links, err := netlink.LinkList()
	if err != nil {
		log.Error(err, "list host links while purging stale macvlans")
		return
	}
	for _, l := range links {
		if ifaceNameRe.MatchString(l.Attrs().Name) {
			if err := netlink.LinkDel(l); err != nil {
				log.Error(err, "delete stale macvlan", "interface", l.Attrs().Name)
			} else {
				log.Info("deleted stale macvlan", "interface", l.Attrs().Name)
			}
		}
	}
}

func teardown(log logr.Logger, ifaceName string) {
	link, err := netlink.LinkByName(ifaceName)
	if err != nil {
		var notFound netlink.LinkNotFoundError
		if !errors.As(err, &notFound) {
			log.Error(err, "find macvlan for teardown", "interface", ifaceName)
		}
		return
	}
	if err := netlink.LinkDel(link); err != nil {
		log.Error(err, "delete macvlan", "interface", ifaceName)
		return
	}
	log.Info("macvlan interface removed", "interface", ifaceName)
}

// DefaultRouteInterface returns the name of the interface carrying the IPv4
// default route in the current network namespace. Even without macvlan, the
// DHCP socket should bind an explicit interface so the kernel has a valid
// source when sending 255.255.255.255 broadcast replies; an unbound socket
// on a multi-homed/CNI pod can fail sendmsg with EINVAL.
func DefaultRouteInterface() (string, error) {
	return defaultRouteInterface()
}

func defaultRouteInterface() (string, error) {
	routes, err := netlink.RouteList(nil, unix.AF_INET)
	if err != nil {
		return "", fmt.Errorf("list routes: %w", err)
	}
	for _, r := range routes {
		isDefault := r.Dst == nil ||
			(r.Dst.IP.Equal(net.IPv4(0, 0, 0, 0)) && r.Gw != nil)
		if !isDefault {
			continue
		}
		iface, err := net.InterfaceByIndex(r.LinkIndex)
		if err != nil {
			continue
		}
		return iface.Name, nil
	}
	return "", errors.New("no IPv4 default route found")
}
