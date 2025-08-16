package reservation

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"

	"github.com/go-logr/logr"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/metal3-community/metal-boot/internal/backend"
	"github.com/metal3-community/metal-boot/internal/dhcp"
	"github.com/metal3-community/metal-boot/internal/dhcp/arp"
	"github.com/metal3-community/metal-boot/internal/dhcp/data"
	oteldhcp "github.com/metal3-community/metal-boot/internal/dhcp/otel"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/net/ipv4"
)

const tracerName = "github.com/metal3-community/metal-boot"

// setDefaults will update the Handler struct to have default values so as
// to avoid panic for nil pointers and such.
func (h *Handler) setDefaults() {
	if h.Backend == nil {
		h.Backend = noop{}
	}
	if h.Log.GetSink() == nil {
		h.Log = logr.Discard()
	}
	// Initialize ARP detector if interface is configured
	if h.ARPDetector == nil && h.InterfaceName != "" {
		h.ARPDetector = arp.NewConflictDetector(h.InterfaceName, h.Log)
	}
	// Initialize lease manager from backend if not already set
	if h.LeaseBackend == nil {
		h.LeaseBackend = CreateLeaseManagerFromBackend(h.Backend)
	}
}

// Handle responds to DHCP messages with DHCP server options.
func (h *Handler) Handle(ctx context.Context, conn *ipv4.PacketConn, p data.Packet) {
	h.setDefaults()
	if p.Pkt == nil {
		h.Log.Error(
			errors.New("incoming packet is nil"),
			"not able to respond when the incoming packet is nil",
		)
		return
	}
	upeer, ok := p.Peer.(*net.UDPAddr)
	if !ok {
		h.Log.Error(
			errors.New("peer is not a UDP connection"),
			"not able to respond when the peer is not a UDP connection",
		)
		return
	}
	if upeer == nil {
		h.Log.Error(errors.New("peer is nil"), "not able to respond when the peer is nil")
		return
	}
	if conn == nil {
		h.Log.Error(
			errors.New("connection is nil"),
			"not able to respond when the connection is nil",
		)
		return
	}

	var ifName string
	if p.Md != nil {
		ifName = p.Md.IfName
	}
	log := h.Log.WithValues(
		"mac",
		p.Pkt.ClientHWAddr.String(),
		"xid",
		p.Pkt.TransactionID.String(),
		"interface",
		ifName,
	)
	tracer := otel.Tracer(tracerName)
	var span trace.Span
	ctx, span = tracer.Start(
		ctx,
		fmt.Sprintf("DHCP Packet Received: %v", p.Pkt.MessageType().String()),
		trace.WithAttributes(h.encodeToAttributes(p.Pkt, "request")...),
		trace.WithAttributes(attribute.String("DHCP.peer", p.Peer.String())),
		trace.WithAttributes(attribute.String("DHCP.server.ifname", ifName)),
	)

	defer span.End()

	var reply *dhcpv4.DHCPv4
	switch mt := p.Pkt.MessageType(); mt {
	case dhcpv4.MessageTypeDiscover:
		d, n, err := h.readBackend(ctx, p.Pkt.ClientHWAddr)
		if err != nil {
			if hardwareNotFound(err) {
				span.SetStatus(codes.Ok, "no reservation found")
				return
			}
			log.Info("error reading from backend", "error", err)
			span.SetStatus(codes.Error, err.Error())

			return
		}
		if d.Disabled {
			log.Info(
				"DHCP is disabled for this MAC address, no response sent",
				"type",
				p.Pkt.MessageType().String(),
			)
			span.SetStatus(codes.Ok, "disabled DHCP response")

			return
		}

		// Temporarily disable IP conflict checking to debug DECLINE issue
		// if h.hasIPConflict(ctx, d.IPAddress) {
		// 	log.Info(
		// 		"IP address conflict detected, declining offer",
		// 		"ip", d.IPAddress.String(),
		// 		"type", p.Pkt.MessageType().String(),
		// 	)
		// 	span.SetStatus(codes.Ok, "IP conflict detected, no offer sent")
		// 	return
		// }

		log.Info("received DHCP packet", "type", p.Pkt.MessageType().String())
		reply = h.updateMsg(ctx, p.Pkt, d, n, dhcpv4.MessageTypeOffer)
		log = log.WithValues("type", dhcpv4.MessageTypeOffer.String())
	case dhcpv4.MessageTypeRequest:
		d, n, err := h.readBackend(ctx, p.Pkt.ClientHWAddr)
		if err != nil {
			if hardwareNotFound(err) {
				span.SetStatus(codes.Ok, "no reservation found")
				return
			}
			log.Info("error reading from backend", "error", err)
			span.SetStatus(codes.Error, err.Error())

			return
		}
		if d.Disabled {
			log.Info(
				"DHCP is disabled for this MAC address, no response sent",
				"type",
				p.Pkt.MessageType().String(),
			)
			span.SetStatus(codes.Ok, "disabled DHCP response")

			return
		}

		// Temporarily disable IP conflict checking to debug DECLINE issue
		// if h.hasIPConflict(ctx, d.IPAddress) {
		// 	log.Info(
		// 		"IP address conflict detected, sending NAK",
		// 		"ip", d.IPAddress.String(),
		// 		"type", p.Pkt.MessageType().String(),
		// 	)
		// 	// Send DHCP NAK to inform client of conflict
		// 	reply = h.createNAK(p.Pkt, "IP address conflict detected")
		// 	log = log.WithValues("type", dhcpv4.MessageTypeNak.String())
		// } else {
		log.Info("received DHCP packet", "type", p.Pkt.MessageType().String())
		reply = h.updateMsg(ctx, p.Pkt, d, n, dhcpv4.MessageTypeAck)
		log = log.WithValues("type", dhcpv4.MessageTypeAck.String())
		// }
		span.SetStatus(codes.Ok, "processed request")
	case dhcpv4.MessageTypeDecline:
		// Handle DHCP DECLINE properly
		h.handleDecline(ctx, p.Pkt, log)
		span.SetStatus(codes.Ok, "processed decline, no response required")
		return
	case dhcpv4.MessageTypeRelease:
		// Release the lease from the backend
		log.Info(
			"received DHCP release packet, no response required, all IPs are host reservations",
			"type",
			p.Pkt.MessageType().String(),
		)
		span.SetStatus(codes.Ok, "received release, no response required")

		return
	default:
		log.Info("received unknown message type", "type", p.Pkt.MessageType().String())
		span.SetStatus(codes.Error, "received unknown message type")

		return
	}

	if bf := reply.BootFileName; bf != "" {
		log = log.WithValues("bootFileName", bf)
	}
	if ns := reply.ServerIPAddr; ns != nil {
		log = log.WithValues("nextServer", ns.String())
	}
	if ci := p.Pkt.ClientIPAddr; ci != nil {
		log = log.WithValues("clientIP", ci)
	}

	// Debug: Log the original peer address and packet details
	log.Info("DEBUG: Packet details",
		"peer", p.Peer.String(),
		"clientIP", p.Pkt.ClientIPAddr.String(),
		"giaddr", p.Pkt.GatewayIPAddr.String(),
		"yourIP", reply.YourIPAddr.String(),
		"broadcastFlag", p.Pkt.IsBroadcast(),
		"messageType", p.Pkt.MessageType().String())

	// Handle destination based on DHCP RFC 2131
	var dst net.Addr
	if p.Pkt.IsBroadcast() {
		// Client requested broadcast response
		dst = &net.UDPAddr{
			IP:   net.IPv4bcast,
			Port: 68,
		}
		log.Info("Using broadcast destination due to broadcast flag")
	} else {
		// Client can receive unicast - send to relay or assigned IP
		if !p.Pkt.GatewayIPAddr.IsUnspecified() && !p.Pkt.GatewayIPAddr.Equal(net.IPv4zero) {
			// Send via relay agent
			dst = &net.UDPAddr{
				IP:   p.Pkt.GatewayIPAddr,
				Port: 67, // DHCP server port for relay
			}
			log.Info("Using relay destination", "giaddr", p.Pkt.GatewayIPAddr.String())
		} else {
			// Send directly to assigned IP
			dst = &net.UDPAddr{
				IP:   reply.YourIPAddr,
				Port: 68,
			}
			log.Info("Using direct unicast destination", "yourIP", reply.YourIPAddr.String())
		}
	}
	log = log.WithValues("ipAddress", reply.YourIPAddr.String(), "destination", dst.String())
	cm := &ipv4.ControlMessage{}
	if p.Md != nil {
		cm.IfIndex = p.Md.IfIndex
	}

	if _, err := conn.WriteTo(reply.ToBytes(), cm, dst); err != nil {
		log.Error(err, "failed to send DHCP")
		span.SetStatus(codes.Error, err.Error())

		return
	}

	log.Info("sent DHCP response")
	span.SetAttributes(h.encodeToAttributes(reply, "reply")...)
	span.SetStatus(codes.Ok, "sent DHCP response")
}

// replyDestination determines the destination address for the DHCP reply.
// If the giaddr is set, then the reply should be sent to the giaddr.
// Otherwise, the reply should be sent to the direct peer.
//
// From page 22 of https://www.ietf.org/rfc/rfc2131.txt:
// "If the 'giaddr' field in a DHCP message from a client is non-zero,
// the server sends any return messages to the 'DHCP server' port on
// the BOOTP relay agent whose address appears in 'giaddr'.".
func replyDestination(directPeer net.Addr, giaddr net.IP) net.Addr {
	if !giaddr.IsUnspecified() && giaddr != nil {
		return &net.UDPAddr{IP: giaddr, Port: dhcpv4.ServerPort}
	}

	return directPeer
}

// readBackend encapsulates the backend read and opentelemetry handling.
func (h *Handler) readBackend(
	ctx context.Context,
	mac net.HardwareAddr,
) (*data.DHCP, *data.Netboot, error) {
	h.setDefaults()

	tracer := otel.Tracer(tracerName)
	ctx, span := tracer.Start(ctx, "Hardware data get")
	defer span.End()

	d, n, err := h.Backend.GetByMac(ctx, mac)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		return nil, nil, err
	}

	span.SetAttributes(d.EncodeToAttributes()...)
	span.SetAttributes(n.EncodeToAttributes()...)
	span.SetStatus(codes.Ok, "done reading from backend")

	return d, n, nil
}

// updateMsg handles updating DHCP packets with the data from the backend.
func (h *Handler) updateMsg(
	ctx context.Context,
	pkt *dhcpv4.DHCPv4,
	d *data.DHCP,
	n *data.Netboot,
	msgType dhcpv4.MessageType,
) *dhcpv4.DHCPv4 {
	h.setDefaults()
	mods := []dhcpv4.Modifier{
		dhcpv4.WithMessageType(msgType),
		dhcpv4.WithGeneric(dhcpv4.OptionServerIdentifier, h.IPAddr.AsSlice()),
		dhcpv4.WithServerIP(h.IPAddr.AsSlice()),
	}

	// Preserve broadcast flag from client request
	if pkt.IsBroadcast() {
		mods = append(mods, dhcpv4.WithBroadcast(true))
	}

	mods = append(mods, h.setDHCPOpts(ctx, pkt, d)...)

	if h.Netboot.Enabled && dhcp.IsNetbootClient(pkt) == nil {
		mods = append(mods, h.setNetworkBootOpts(ctx, pkt, n))
	}
	// We ignore the error here because:
	// 1. it's only non-nil if the generation of a transaction id (XID) fails.
	// 2. We always use the clients transaction id (XID) in responses. See dhcpv4.WithReply().
	reply, _ := dhcpv4.NewReplyFromRequest(pkt, mods...)

	return reply
}

// encodeToAttributes takes a DHCP packet and returns opentelemetry key/value attributes.
func (h *Handler) encodeToAttributes(d *dhcpv4.DHCPv4, namespace string) []attribute.KeyValue {
	h.setDefaults()
	a := &oteldhcp.Encoder{Log: h.Log}

	return a.Encode(d, namespace, oteldhcp.AllEncoders()...)
}

// hardwareNotFound returns true if the error is from a hardware record not being found.
func hardwareNotFound(err error) bool {
	type hardwareNotFound interface {
		NotFound() bool
	}
	te, ok := err.(hardwareNotFound)
	return ok && te.NotFound()
}

// hasIPConflict checks if an IP address has conflicts using both lease tracking and ARP detection.
func (h *Handler) hasIPConflict(ctx context.Context, ip netip.Addr) bool {
	h.setDefaults()

	ipStr := ip.String()

	// First check if IP is marked as declined in our lease manager
	if h.LeaseBackend != nil && h.LeaseBackend.IsIPDeclined(ipStr) {
		h.Log.V(1).Info("IP is marked as declined", "ip", ipStr)
		return true
	}

	// Then check for active ARP conflicts
	if h.ARPDetector != nil {
		if h.ARPDetector.IsIPInUse(ip.AsSlice()) {
			h.Log.Info("ARP conflict detected for IP", "ip", ipStr)
			// Mark this IP as declined for future reference
			if h.LeaseBackend != nil {
				if err := h.LeaseBackend.MarkIPDeclined(ipStr); err != nil {
					h.Log.Error(err, "failed to mark IP as declined", "ip", ipStr)
				}
			}
			return true
		}
	}

	return false
}

// handleDecline processes DHCP DECLINE messages by marking the IP as declined.
func (h *Handler) handleDecline(ctx context.Context, pkt *dhcpv4.DHCPv4, log logr.Logger) {
	h.setDefaults()

	// Extract requested IP from option 50 (Requested IP Address)
	requestedIP := pkt.RequestedIPAddress()
	if requestedIP == nil {
		log.Info("DHCP DECLINE received but no requested IP found")
		return
	}

	ipStr := requestedIP.String()
	log.Info(
		"processing DHCP DECLINE",
		"declined_ip",
		ipStr,
		"client_mac",
		pkt.ClientHWAddr.String(),
	)

	// Mark the IP as declined in our lease manager
	if h.LeaseBackend != nil {
		if err := h.LeaseBackend.MarkIPDeclined(ipStr); err != nil {
			log.Error(err, "failed to mark IP as declined", "ip", ipStr)
			return
		}
		log.Info("marked IP as declined", "ip", ipStr)
	} else {
		log.Info("no lease backend configured, cannot track declined IP", "ip", ipStr)
	}

	// For static reservations, we don't reassign IPs - the same MAC always gets the same IP
	// The decline is likely due to the client detecting its own previous usage
	log.Info("DECLINE processed for static reservation - same MAC will get same IP on next request",
		"mac", pkt.ClientHWAddr.String(),
		"declined_ip", ipStr)

	// Optionally verify the conflict with ARP
	if h.ARPDetector != nil {
		if addr, err := netip.ParseAddr(ipStr); err == nil {
			if h.ARPDetector.IsIPInUse(addr.AsSlice()) {
				log.Info("ARP conflict confirmed for declined IP", "ip", ipStr)
			} else {
				log.Info("no ARP conflict detected for declined IP (possible client quirk)", "ip", ipStr)
			}
		}
	}
}

// createNAK creates a DHCP NAK response to reject a client's request.
func (h *Handler) createNAK(pkt *dhcpv4.DHCPv4, message string) *dhcpv4.DHCPv4 {
	h.setDefaults()

	mods := []dhcpv4.Modifier{
		dhcpv4.WithMessageType(dhcpv4.MessageTypeNak),
		dhcpv4.WithGeneric(dhcpv4.OptionServerIdentifier, h.IPAddr.AsSlice()),
		dhcpv4.WithServerIP(h.IPAddr.AsSlice()),
	}

	// Add an error message if supported
	if message != "" {
		mods = append(mods, dhcpv4.WithGeneric(dhcpv4.OptionMessage, []byte(message)))
	}

	// Create NAK response using client's transaction ID
	reply, _ := dhcpv4.NewReplyFromRequest(pkt, mods...)

	return reply
}

// reassignIPAfterDecline handles IP reassignment after a DHCP DECLINE using the BackendWriter interface.
func (h *Handler) reassignIPAfterDecline(
	ctx context.Context,
	writer backend.BackendWriter,
	mac net.HardwareAddr,
	declinedIP net.IP,
	log logr.Logger,
) error {
	// First mark the declined IP in our lease backend if available
	if h.LeaseBackend != nil {
		if err := h.LeaseBackend.MarkIPDeclined(declinedIP.String()); err != nil {
			log.Error(err, "failed to mark declined IP in lease backend", "ip", declinedIP.String())
		}
	}

	// Get the current data for this MAC to preserve it
	currentDHCP, currentNetboot, err := h.Backend.GetByMac(ctx, mac)
	if err != nil {
		// If we can't get current data, we can't reassign safely
		return fmt.Errorf("failed to get current data for MAC %s: %w", mac.String(), err)
	}

	// If there's no current data or the current IP is the declined IP,
	// we need to assign a new IP. This should trigger automatic assignment
	// in backends that support it (like dnsmasq with auto-assignment enabled).
	if currentDHCP == nil || currentDHCP.IPAddress.String() == declinedIP.String() {
		// Create new DHCP data with empty IP to trigger auto-assignment
		newDHCP := &data.DHCP{
			MACAddress: mac,
			// Leave IPAddress empty to trigger auto-assignment
			Hostname:  fmt.Sprintf("auto-%s", mac.String()),
			LeaseTime: 604800, // Default 1 week
		}

		// Preserve existing netboot configuration if available
		if currentNetboot == nil {
			currentNetboot = &data.Netboot{
				AllowNetboot: true, // Enable netboot for auto-assigned devices
			}
		}

		// Use Put to assign new IP and save to lease file
		if err := writer.Put(ctx, mac, newDHCP, currentNetboot); err != nil {
			return fmt.Errorf("failed to reassign IP using backend writer: %w", err)
		}

		log.Info("triggered IP reassignment via backend writer",
			"mac", mac.String(),
			"declined_ip", declinedIP.String())
		return nil
	}

	// Current IP is different from declined IP, no reassignment needed
	log.Info("current IP differs from declined IP, no reassignment needed",
		"mac", mac.String(),
		"current_ip", currentDHCP.IPAddress.String(),
		"declined_ip", declinedIP.String())
	return nil
}
