package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

const (
	// DefaultPort is the default UDP port for discovery broadcasts (same as gRPC)
	DefaultPort = 50051
	// BroadcastInterval is how often to broadcast presence
	BroadcastInterval = 5 * time.Second
	// StaleTimeout is how long before a device is considered stale
	StaleTimeout = 30 * time.Second
	// CleanupInterval is how often to check for stale devices
	CleanupInterval = 10 * time.Second
)

// Callback is called when devices are discovered or leave
type Callback interface {
	OnDeviceDiscovered(device *DeviceAnnounce)
	OnDeviceLeft(deviceID string)
}

// Service handles UDP broadcast discovery
type Service struct {
	port       int
	selfDevice *DeviceAnnounce
	callback   Callback
	seedPeers  []*net.UDPAddr // Known peers for cross-subnet discovery

	conn       *net.UDPConn
	broadcasts []*net.UDPAddr

	// Track last-seen times for stale detection
	lastSeen map[string]time.Time
	mu       sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewService creates a new discovery service
func NewService(port int, selfDevice *DeviceAnnounce, callback Callback) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		port:       port,
		selfDevice: selfDevice,
		callback:   callback,
		seedPeers:  make([]*net.UDPAddr, 0),
		lastSeen:   make(map[string]time.Time),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// AddSeedPeer adds a known peer address for cross-subnet discovery
func (s *Service) AddSeedPeer(addr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		return fmt.Errorf("invalid seed peer address %s: %w", addr, err)
	}
	s.seedPeers = append(s.seedPeers, udpAddr)
	return nil
}

// Start begins discovery (binds socket, starts goroutines)
func (s *Service) Start() error {
	// Bind UDP socket for listening on all interfaces
	listenAddr := &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: s.port,
	}
	conn, err := net.ListenUDP("udp4", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to bind UDP port %d: %w", s.port, err)
	}
	s.conn = conn

	// Setup broadcast addresses for ALL valid interfaces
	// This ensures we reach the wifi network even if docker0 is found first
	bcastIPs := detectBroadcastAddresses()
	if len(bcastIPs) == 0 {
		bcastIPs = []net.IP{net.IPv4bcast} // Fallback to 255.255.255.255
	}

	s.broadcasts = make([]*net.UDPAddr, 0, len(bcastIPs))
	for _, ip := range bcastIPs {
		addr := &net.UDPAddr{
			IP:   ip,
			Port: s.port,
		}
		s.broadcasts = append(s.broadcasts, addr)
		log.Printf("[INFO] Using broadcast address: %s:%d", addr.IP, addr.Port)
	}

	// Set buffer sizes
	if err := conn.SetWriteBuffer(MaxMessageSize * 10); err != nil {
		log.Printf("[WARN] discovery: failed to set write buffer: %v", err)
	}
	if err := conn.SetReadBuffer(MaxMessageSize * 10); err != nil {
		log.Printf("[WARN] discovery: failed to set read buffer: %v", err)
	}

	// Start goroutines
	s.wg.Add(3)
	go s.listenLoop()
	go s.announceLoop()
	go s.cleanupLoop()

	log.Printf("[INFO] Discovery service started on UDP port %d", s.port)
	return nil
}

// Stop gracefully shuts down the service
func (s *Service) Stop() {
	// Send LEAVE broadcast before stopping
	s.broadcastMessage(MessageTypeLeave)

	s.cancel()
	if s.conn != nil {
		s.conn.Close()
	}
	s.wg.Wait()
	log.Printf("[INFO] Discovery service stopped")
}

// listenLoop receives UDP broadcasts from other devices
func (s *Service) listenLoop() {
	defer s.wg.Done()

	buf := make([]byte, MaxMessageSize)
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		// Set read deadline to allow periodic ctx check
		s.conn.SetReadDeadline(time.Now().Add(1 * time.Second))

		n, addr, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // Normal timeout, check ctx and retry
			}
			if s.ctx.Err() != nil {
				return // Service stopping
			}
			log.Printf("[WARN] discovery: read error: %v", err)
			continue
		}

		// Parse message
		var msg DiscoveryMessage
		if err := json.Unmarshal(buf[:n], &msg); err != nil {
			log.Printf("[DEBUG] discovery: invalid message from %s: %v", addr, err)
			continue
		}

		// Ignore our own broadcasts
		if msg.Device.DeviceID == s.selfDevice.DeviceID {
			continue
		}

		s.handleMessage(&msg, addr)
	}
}

// handleMessage processes a received discovery message
func (s *Service) handleMessage(msg *DiscoveryMessage, sourceAddr *net.UDPAddr) {
	// Fix device addresses that use 0.0.0.0 or 127.0.0.1
	// Replace with actual source IP from UDP packet
	sourceIP := sourceAddr.IP.String()

	// Fix GrpcAddr if it's using invalid/local addresses
	if msg.Device.GrpcAddr != "" {
		host, port, err := net.SplitHostPort(msg.Device.GrpcAddr)
		if err == nil {
			// Replace 0.0.0.0 or 127.0.0.1 with actual source IP
			if host == "0.0.0.0" || host == "127.0.0.1" || host == "localhost" {
				msg.Device.GrpcAddr = net.JoinHostPort(sourceIP, port)
				log.Printf("[DEBUG] discovery: fixed GrpcAddr from %s to %s", host, msg.Device.GrpcAddr)
			}
		}
	}

	// Fix HttpAddr if it's using invalid/local addresses
	if msg.Device.HttpAddr != "" {
		host, port, err := net.SplitHostPort(msg.Device.HttpAddr)
		if err == nil {
			if host == "0.0.0.0" || host == "127.0.0.1" || host == "localhost" {
				msg.Device.HttpAddr = net.JoinHostPort(sourceIP, port)
			}
		}
	}

	switch msg.Type {
	case MessageTypeAnnounce:
		s.mu.Lock()
		_, known := s.lastSeen[msg.Device.DeviceID]
		s.lastSeen[msg.Device.DeviceID] = time.Now()
		s.mu.Unlock()

		if !known {
			log.Printf("[INFO] discovery: found new device %s (%s) at %s",
				msg.Device.DeviceName, msg.Device.DeviceID[:8], msg.Device.GrpcAddr)
		}

		s.callback.OnDeviceDiscovered(&msg.Device)

	case MessageTypeLeave:
		s.mu.Lock()
		delete(s.lastSeen, msg.Device.DeviceID)
		s.mu.Unlock()

		log.Printf("[INFO] discovery: device %s (%s) left",
			msg.Device.DeviceName, msg.Device.DeviceID[:8])

		s.callback.OnDeviceLeft(msg.Device.DeviceID)
	}
}

// announceLoop periodically broadcasts our presence
func (s *Service) announceLoop() {
	defer s.wg.Done()

	// Broadcast immediately on startup
	s.broadcastMessage(MessageTypeAnnounce)

	ticker := time.NewTicker(BroadcastInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.broadcastMessage(MessageTypeAnnounce)
		}
	}
}

// broadcastMessage sends a discovery message to all devices on the LAN
func (s *Service) broadcastMessage(msgType MessageType) {
	msg := DiscoveryMessage{
		Type:      msgType,
		Version:   1,
		Timestamp: time.Now().UnixMilli(),
		Device:    *s.selfDevice,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[ERROR] discovery: failed to marshal message: %v", err)
		return
	}

	// Send to all broadcast addresses (subnets)
	for _, bcast := range s.broadcasts {
		if _, err := s.conn.WriteToUDP(data, bcast); err != nil {
			// Don't spam logs - broadcast failures are common on some networks
			if s.ctx.Err() == nil {
				log.Printf("[DEBUG] discovery: broadcast to %s failed: %v", bcast, err)
			}
		}
	}

	// Also send directly to seed peers (cross-subnet)
	for _, peer := range s.seedPeers {
		if _, err := s.conn.WriteToUDP(data, peer); err != nil {
			if s.ctx.Err() == nil {
				log.Printf("[DEBUG] discovery: send to seed peer %s failed: %v", peer, err)
			}
		}
	}
}

// cleanupLoop removes devices that haven't been seen recently
func (s *Service) cleanupLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.purgeStaleDevices()
		}
	}
}

// purgeStaleDevices removes devices not seen for StaleTimeout
func (s *Service) purgeStaleDevices() {
	now := time.Now()
	staleThreshold := now.Add(-StaleTimeout)

	s.mu.Lock()
	defer s.mu.Unlock()

	for deviceID, lastSeen := range s.lastSeen {
		if lastSeen.Before(staleThreshold) {
			delete(s.lastSeen, deviceID)
			s.callback.OnDeviceLeft(deviceID)
			log.Printf("[INFO] discovery: device %s... marked stale (no broadcast for %v)",
				deviceID[:8], StaleTimeout)
		}
	}
}

// GetPort returns the discovery port
func (s *Service) GetPort() int {
	return s.port
}

// detectBroadcastAddresses finds all suitable broadcast addresses
func detectBroadcastAddresses() []net.IP {
	var results []net.IP
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	for _, i := range ifaces {
		// Skip loopback and down interfaces
		if i.Flags&net.FlagLoopback != 0 || i.Flags&net.FlagUp == 0 {
			continue
		}
		// Must support broadcast
		if i.Flags&net.FlagBroadcast == 0 {
			continue
		}

		addrs, err := i.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipNet.IP.To4()
			if ip == nil {
				continue // Skip IPv6
			}

			// Calculate broadcast address: IP | ^Mask
			mask := ipNet.Mask
			broadcast := make(net.IP, len(ip))
			for j := 0; j < len(ip); j++ {
				broadcast[j] = ip[j] | ^mask[j]
			}
			results = append(results, broadcast)
		}
	}
	return results
}
