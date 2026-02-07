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
	// DefaultPort is the default UDP port for discovery broadcasts
	DefaultPort = 50050
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

	conn      *net.UDPConn
	broadcast *net.UDPAddr

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

	// Setup broadcast address (255.255.255.255:port)
	s.broadcast = &net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: s.port,
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

		s.handleMessage(&msg)
	}
}

// handleMessage processes a received discovery message
func (s *Service) handleMessage(msg *DiscoveryMessage) {
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

	// Send to broadcast address (same subnet only)
	if _, err := s.conn.WriteToUDP(data, s.broadcast); err != nil {
		// Don't spam logs - broadcast failures are common on some networks
		if s.ctx.Err() == nil {
			log.Printf("[DEBUG] discovery: broadcast failed: %v", err)
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
