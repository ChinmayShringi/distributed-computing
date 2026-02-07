// Package transfer manages download ticket lifecycle for the bulk HTTP plane.
package transfer

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// Ticket represents a one-time-use download authorization.
type Ticket struct {
	Token     string
	FilePath  string // absolute path on disk (already validated)
	Filename  string // basename for Content-Disposition
	SizeBytes int64
	ExpiresAt time.Time
}

// Manager is a thread-safe ticket store.
type Manager struct {
	mu      sync.Mutex
	tickets map[string]*Ticket
	ttl     time.Duration
}

// NewManager creates a new ticket manager with the given default TTL.
func NewManager(ttl time.Duration) *Manager {
	return &Manager{
		tickets: make(map[string]*Ticket),
		ttl:     ttl,
	}
}

// Create mints a new download ticket.
// filePath must be the absolute, validated path on disk.
// filename is the basename used for Content-Disposition.
func (m *Manager) Create(filePath, filename string, sizeBytes int64) (*Ticket, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(b)

	ticket := &Ticket{
		Token:     token,
		FilePath:  filePath,
		Filename:  filename,
		SizeBytes: sizeBytes,
		ExpiresAt: time.Now().Add(m.ttl),
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.purgeExpiredLocked()
	m.tickets[token] = ticket
	return ticket, nil
}

// Consume retrieves and atomically deletes a ticket.
// Returns nil if the token is invalid, already used, or expired.
func (m *Manager) Consume(token string) *Ticket {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.purgeExpiredLocked()

	ticket, ok := m.tickets[token]
	if !ok {
		return nil
	}

	if time.Now().After(ticket.ExpiresAt) {
		delete(m.tickets, token)
		return nil
	}

	delete(m.tickets, token)
	return ticket
}

// purgeExpiredLocked removes all expired tickets. Caller must hold m.mu.
func (m *Manager) purgeExpiredLocked() {
	now := time.Now()
	for token, ticket := range m.tickets {
		if now.After(ticket.ExpiresAt) {
			delete(m.tickets, token)
		}
	}
}
