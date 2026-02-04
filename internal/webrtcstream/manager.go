// Package webrtcstream manages WebRTC peer connections for screen streaming
package webrtcstream

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kbinani/screenshot"
	"github.com/pion/webrtc/v3"
	"golang.org/x/image/draw"
)

const maxMessageSize = 63 * 1024 // 63KB, safely under 64KB SCTP limit

// Stream represents an active WebRTC screen streaming session
type Stream struct {
	ID             string
	PeerConnection *webrtc.PeerConnection
	DataChannel    *webrtc.DataChannel
	cancel         context.CancelFunc
	targetFPS      int
	jpegQuality    int
	monitorIndex   int
}

// Manager manages multiple WebRTC streams
type Manager struct {
	streams map[string]*Stream
	mu      sync.RWMutex
}

// NewManager creates a new WebRTC stream manager
func NewManager() *Manager {
	return &Manager{
		streams: make(map[string]*Stream),
	}
}

// Start creates a new WebRTC peer connection and returns an offer SDP
func (m *Manager) Start(sessionID string, targetFPS, jpegQuality, monitorIndex int) (streamID, offerSDP string, err error) {
	// Apply defaults
	if targetFPS <= 0 {
		targetFPS = 8
	}
	if jpegQuality <= 0 {
		jpegQuality = 60
	}

	// Validate monitor index
	numDisplays := screenshot.NumActiveDisplays()
	if monitorIndex < 0 || monitorIndex >= numDisplays {
		return "", "", fmt.Errorf("invalid monitor_index %d, have %d displays", monitorIndex, numDisplays)
	}

	// Create peer connection with no ICE servers (LAN only)
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{}, // Empty for LAN-only
	}

	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return "", "", fmt.Errorf("failed to create peer connection: %w", err)
	}

	// Create data channel for frames
	dc, err := pc.CreateDataChannel("frames", nil)
	if err != nil {
		pc.Close()
		return "", "", fmt.Errorf("failed to create data channel: %w", err)
	}

	streamID = uuid.New().String()

	stream := &Stream{
		ID:             streamID,
		PeerConnection: pc,
		DataChannel:    dc,
		targetFPS:      targetFPS,
		jpegQuality:    jpegQuality,
		monitorIndex:   monitorIndex,
	}

	// Set up data channel open handler to start capture
	dc.OnOpen(func() {
		log.Printf("[INFO] WebRTC stream %s: data channel opened, starting capture", streamID)
		ctx, cancel := context.WithCancel(context.Background())
		stream.cancel = cancel
		go stream.captureLoop(ctx)
	})

	dc.OnClose(func() {
		log.Printf("[INFO] WebRTC stream %s: data channel closed", streamID)
		if stream.cancel != nil {
			stream.cancel()
		}
	})

	// Create offer
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		pc.Close()
		return "", "", fmt.Errorf("failed to create offer: %w", err)
	}

	// Set local description
	if err := pc.SetLocalDescription(offer); err != nil {
		pc.Close()
		return "", "", fmt.Errorf("failed to set local description: %w", err)
	}

	// Wait for ICE gathering to complete (non-trickle)
	gatherComplete := webrtc.GatheringCompletePromise(pc)
	select {
	case <-gatherComplete:
		// ICE gathering complete
		log.Printf("[INFO] WebRTC stream %s: ICE gathering complete", streamID)
	case <-time.After(5 * time.Second):
		pc.Close()
		return "", "", fmt.Errorf("ICE gathering timeout")
	}

	// Get the complete SDP with candidates
	offerSDP = pc.LocalDescription().SDP

	// Store stream
	m.mu.Lock()
	m.streams[streamID] = stream
	m.mu.Unlock()

	return streamID, offerSDP, nil
}

// Complete sets the remote description (answer) for a stream
func (m *Manager) Complete(streamID, answerSDP string) error {
	m.mu.RLock()
	stream, ok := m.streams[streamID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("stream not found: %s", streamID)
	}

	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  answerSDP,
	}

	if err := stream.PeerConnection.SetRemoteDescription(answer); err != nil {
		return fmt.Errorf("failed to set remote description: %w", err)
	}

	log.Printf("[INFO] WebRTC stream %s: remote description set, connection established", streamID)
	return nil
}

// Stop closes a stream and cleans up resources
func (m *Manager) Stop(streamID string) error {
	m.mu.Lock()
	stream, ok := m.streams[streamID]
	if ok {
		delete(m.streams, streamID)
	}
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("stream not found: %s", streamID)
	}

	if stream.cancel != nil {
		stream.cancel()
	}

	log.Printf("[INFO] WebRTC stream %s: stopped", streamID)
	return stream.PeerConnection.Close()
}

// captureLoop captures screen frames and sends them over the data channel
func (s *Stream) captureLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second / time.Duration(s.targetFPS))
	defer ticker.Stop()

	log.Printf("[INFO] WebRTC stream %s: capture loop started (fps=%d, quality=%d, monitor=%d)",
		s.ID, s.targetFPS, s.jpegQuality, s.monitorIndex)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[INFO] WebRTC stream %s: capture loop stopped (context cancelled)", s.ID)
			return
		case <-ticker.C:
			if err := s.captureAndSend(); err != nil {
				log.Printf("[WARN] WebRTC stream %s: capture/send error: %v", s.ID, err)
				// Stop on send error (connection closed)
				return
			}
		}
	}
}

// scaleImage downscales src by the given factor (0..1].
func scaleImage(src image.Image, factor float64) image.Image {
	srcBounds := src.Bounds()
	newW := int(float64(srcBounds.Dx()) * factor)
	newH := int(float64(srcBounds.Dy()) * factor)
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, srcBounds, draw.Over, nil)
	return dst
}

// captureAndSend captures a single frame and sends it
func (s *Stream) captureAndSend() error {
	// Capture screen
	bounds := screenshot.GetDisplayBounds(s.monitorIndex)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return fmt.Errorf("capture failed: %w", err)
	}

	// Scale down to fit within SCTP message limit.
	// Start at 50% resolution, reduce further if still too large.
	scale := 0.5
	quality := s.jpegQuality

	for attempts := 0; attempts < 4; attempts++ {
		scaled := scaleImage(img, scale)

		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, scaled, &jpeg.Options{Quality: quality}); err != nil {
			return fmt.Errorf("jpeg encode failed: %w", err)
		}

		if buf.Len() <= maxMessageSize {
			return s.DataChannel.Send(buf.Bytes())
		}

		// Still too large — reduce scale and quality
		scale *= 0.7
		if quality > 20 {
			quality -= 10
		}
	}

	return fmt.Errorf("frame too large after scaling attempts")
}
