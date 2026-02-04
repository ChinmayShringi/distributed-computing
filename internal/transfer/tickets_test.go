package transfer

import (
	"testing"
	"time"
)

func TestCreateAndConsume(t *testing.T) {
	mgr := NewManager(60 * time.Second)

	ticket, err := mgr.Create("/tmp/test.txt", "test.txt", 100)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if len(ticket.Token) < 40 {
		t.Fatalf("Token too short: %d chars", len(ticket.Token))
	}

	// Consume should succeed once
	consumed := mgr.Consume(ticket.Token)
	if consumed == nil {
		t.Fatal("Consume returned nil")
	}
	if consumed.FilePath != "/tmp/test.txt" {
		t.Fatalf("Wrong file path: %s", consumed.FilePath)
	}
	if consumed.Filename != "test.txt" {
		t.Fatalf("Wrong filename: %s", consumed.Filename)
	}
	if consumed.SizeBytes != 100 {
		t.Fatalf("Wrong size: %d", consumed.SizeBytes)
	}

	// Second consume should fail (one-time use)
	again := mgr.Consume(ticket.Token)
	if again != nil {
		t.Fatal("Second consume should return nil")
	}
}

func TestExpiredTicket(t *testing.T) {
	mgr := NewManager(1 * time.Millisecond)

	ticket, err := mgr.Create("/tmp/test.txt", "test.txt", 100)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	time.Sleep(5 * time.Millisecond)

	consumed := mgr.Consume(ticket.Token)
	if consumed != nil {
		t.Fatal("Expired ticket should not be consumable")
	}
}

func TestInvalidToken(t *testing.T) {
	mgr := NewManager(60 * time.Second)

	consumed := mgr.Consume("nonexistent-token")
	if consumed != nil {
		t.Fatal("Invalid token should return nil")
	}
}
