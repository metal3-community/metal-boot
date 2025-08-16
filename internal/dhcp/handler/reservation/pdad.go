package reservation

import (
	"context"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/insomniacslk/dhcp/dhcpv4"
)

// PDADDetector implements Passive Duplicate Address Detection
type PDADDetector struct {
	interfaceName string
	log           logr.Logger
	ipUsage       map[string]*IPUsageInfo
	mu            sync.RWMutex
	handle        *pcap.Handle
	stopCh        chan struct{}
	running       bool
}

// IPUsageInfo tracks IP address usage information
type IPUsageInfo struct {
	MAC         net.HardwareAddr
	FirstSeen   time.Time
	LastSeen    time.Time
	PacketCount int
	IsStatic    bool // Whether this is a known static reservation
}

// NewPDADDetector creates a new passive duplicate address detector
func NewPDADDetector(interfaceName string, log logr.Logger) *PDADDetector {
	return &PDADDetector{
		interfaceName: interfaceName,
		log:           log,
		ipUsage:       make(map[string]*IPUsageInfo),
		stopCh:        make(chan struct{}),
	}
}

// Start begins passive monitoring of network traffic
func (p *PDADDetector) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return nil
	}

	// Open the network interface for packet capture
	handle, err := pcap.OpenLive(p.interfaceName, 65536, true, pcap.BlockForever)
	if err != nil {
		return err
	}

	// Set BPF filter to capture only relevant traffic
	// Monitor ARP and DHCP traffic
	filter := "arp or (udp and (port 67 or port 68))"
	if err := handle.SetBPFFilter(filter); err != nil {
		handle.Close()
		return err
	}

	p.handle = handle
	p.running = true

	go p.monitorTraffic(ctx)

	p.log.Info("Started Passive Duplicate Address Detection", "interface", p.interfaceName)
	return nil
}

// Stop stops the passive monitoring
func (p *PDADDetector) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return
	}

	close(p.stopCh)
	if p.handle != nil {
		p.handle.Close()
	}
	p.running = false

	p.log.Info("Stopped Passive Duplicate Address Detection")
}

// monitorTraffic continuously monitors network traffic
func (p *PDADDetector) monitorTraffic(ctx context.Context) {
	packetSource := gopacket.NewPacketSource(p.handle, p.handle.LinkType())

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		case packet := <-packetSource.Packets():
			if packet == nil {
				continue
			}
			p.processPacket(packet)
		}
	}
}

// processPacket analyzes individual packets for IP usage
func (p *PDADDetector) processPacket(packet gopacket.Packet) {
	// Process ARP packets
	if arpLayer := packet.Layer(layers.LayerTypeARP); arpLayer != nil {
		arp := arpLayer.(*layers.ARP)
		p.processARPPacket(arp)
	}

	// Process DHCP packets
	if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
		udp := udpLayer.(*layers.UDP)
		if udp.SrcPort == 68 || udp.DstPort == 68 || udp.SrcPort == 67 || udp.DstPort == 67 {
			if payload := packet.ApplicationLayer(); payload != nil {
				p.processDHCPPacket(payload.Payload())
			}
		}
	}
}

// processARPPacket analyzes ARP packets for IP usage
func (p *PDADDetector) processARPPacket(arp *layers.ARP) {
	if arp.Operation == layers.ARPReply || arp.Operation == layers.ARPRequest {
		srcIP := net.IP(arp.SourceProtAddress)
		srcMAC := net.HardwareAddr(arp.SourceHwAddress)

		if !srcIP.IsUnspecified() && len(srcMAC) > 0 {
			p.recordIPUsage(srcIP.String(), srcMAC, false)
		}
	}
}

// processDHCPPacket analyzes DHCP packets for IP assignments
func (p *PDADDetector) processDHCPPacket(payload []byte) {
	// Parse DHCP packet using dhcpv4
	pkt, err := dhcpv4.FromBytes(payload)
	if err != nil {
		return
	}

	// Record client IP usage
	if !pkt.ClientIPAddr.IsUnspecified() {
		p.recordIPUsage(pkt.ClientIPAddr.String(), pkt.ClientHWAddr, true)
	}

	// Record your IP usage (for DHCP offers/acks)
	if !pkt.YourIPAddr.IsUnspecified() {
		p.recordIPUsage(pkt.YourIPAddr.String(), pkt.ClientHWAddr, true)
	}
}

// recordIPUsage records IP address usage information
func (p *PDADDetector) recordIPUsage(ip string, mac net.HardwareAddr, isStatic bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()

	if existing, exists := p.ipUsage[ip]; exists {
		// Check for MAC address conflict
		if existing.MAC.String() != mac.String() {
			p.log.Info("Potential IP conflict detected",
				"ip", ip,
				"existing_mac", existing.MAC.String(),
				"new_mac", mac.String(),
				"existing_first_seen", existing.FirstSeen,
				"existing_last_seen", existing.LastSeen,
			)
		}
		existing.LastSeen = now
		existing.PacketCount++
		if isStatic {
			existing.IsStatic = true
		}
	} else {
		p.ipUsage[ip] = &IPUsageInfo{
			MAC:         make(net.HardwareAddr, len(mac)),
			FirstSeen:   now,
			LastSeen:    now,
			PacketCount: 1,
			IsStatic:    isStatic,
		}
		copy(p.ipUsage[ip].MAC, mac)
	}
}

// HasConflict checks if an IP address has a conflict
func (p *PDADDetector) HasConflict(ip netip.Addr, expectedMAC net.HardwareAddr) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	usage, exists := p.ipUsage[ip.String()]
	if !exists {
		return false
	}

	// Check if the MAC addresses match
	if usage.MAC.String() != expectedMAC.String() {
		// Consider the age of the conflict
		if time.Since(usage.LastSeen) < 5*time.Minute {
			return true
		}
	}

	return false
}

// GetIPUsage returns usage information for an IP address
func (p *PDADDetector) GetIPUsage(ip string) (*IPUsageInfo, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	usage, exists := p.ipUsage[ip]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	return &IPUsageInfo{
		MAC:         append(net.HardwareAddr(nil), usage.MAC...),
		FirstSeen:   usage.FirstSeen,
		LastSeen:    usage.LastSeen,
		PacketCount: usage.PacketCount,
		IsStatic:    usage.IsStatic,
	}, true
}

// CleanupOldEntries removes old IP usage entries
func (p *PDADDetector) CleanupOldEntries(maxAge time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for ip, usage := range p.ipUsage {
		if usage.LastSeen.Before(cutoff) {
			delete(p.ipUsage, ip)
		}
	}
}
