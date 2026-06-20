package websocket

import (
	"testing"
	"time"
)

// TestHubRegisterUnregister tests that the Hub correctly adds and removes
// clients from its clients map via the register and unregister channels.
func TestHubRegisterUnregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		hub:  hub,
		conn: nil, // Not needed for this test — we're only testing Hub logic
		send: make(chan []byte, 256),
	}

	// Register the client.
	hub.register <- client

	// Give the Hub's Run goroutine time to process the register event.
	time.Sleep(50 * time.Millisecond)

	if _, exists := hub.clients[client]; !exists {
		t.Error("expected client to be registered in hub, but it was not found")
	}

	// Unregister the client.
	hub.unregister <- client

	// Give the Hub's Run goroutine time to process the unregister event.
	time.Sleep(50 * time.Millisecond)

	if _, exists := hub.clients[client]; exists {
		t.Error("expected client to be unregistered from hub, but it still exists")
	}
}

// TestHubBroadcast tests that the Hub broadcasts a message to all connected
// clients via their send channels.
func TestHubBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create and register two clients.
	client1 := &Client{
		hub:  hub,
		conn: nil,
		send: make(chan []byte, 256),
	}
	client2 := &Client{
		hub:  hub,
		conn: nil,
		send: make(chan []byte, 256),
	}

	hub.register <- client1
	hub.register <- client2

	// Give the Hub's Run goroutine time to process register events.
	time.Sleep(50 * time.Millisecond)

	if len(hub.clients) != 2 {
		t.Errorf("expected 2 clients in hub, got %d", len(hub.clients))
	}

	// Broadcast a message.
	message := []byte(`{"type":"sensor_update","vehicle_id":"v1"}`)
	hub.broadcast <- message

	// Give the Hub's Run goroutine time to process the broadcast.
	time.Sleep(50 * time.Millisecond)

	// Both clients should receive the message on their send channels.
	select {
	case msg := <-client1.send:
		if string(msg) != string(message) {
			t.Errorf("client1 received wrong message: got %q, want %q", string(msg), string(message))
		}
	default:
		t.Error("client1 did not receive the broadcast message")
	}

	select {
	case msg := <-client2.send:
		if string(msg) != string(message) {
			t.Errorf("client2 received wrong message: got %q, want %q", string(msg), string(message))
		}
	default:
		t.Error("client2 did not receive the broadcast message")
	}

	// Unregister client1.
	hub.unregister <- client1
	time.Sleep(50 * time.Millisecond)

	// Broadcast again — only client2 should receive it.
	message2 := []byte(`{"type":"sensor_update","vehicle_id":"v2"}`)
	hub.broadcast <- message2

	time.Sleep(50 * time.Millisecond)

	// client1's send channel should be closed after unregister.
	select {
	case _, ok := <-client1.send:
		if ok {
			t.Error("client1 should not receive messages after unregister")
		}
		// Channel was closed, which is expected behavior.
	default:
		// No message received, which is also fine.
	}

	select {
	case msg := <-client2.send:
		if string(msg) != string(message2) {
			t.Errorf("client2 received wrong message: got %q, want %q", string(msg), string(message2))
		}
	default:
		t.Error("client2 did not receive the second broadcast message")
	}
}

// TestHubBroadcastDropsSlowClient tests that the Hub drops messages for
// clients whose send buffer is full, and disconnects them.
func TestHubBroadcastDropsSlowClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create a client with a very small send buffer (1 slot).
	slowClient := &Client{
		hub:  hub,
		conn: nil,
		send: make(chan []byte, 1), // Tiny buffer — will fill quickly
	}

	hub.register <- slowClient
	time.Sleep(50 * time.Millisecond)

	// Fill the client's send buffer.
	slowClient.send <- []byte("filler")

	// Broadcast a message. The Hub should detect that slowClient's
	// buffer is full, close its send channel, and remove it from clients.
	hub.broadcast <- []byte("overflow_message")
	time.Sleep(50 * time.Millisecond)

	// The slow client should have been removed from the hub.
	if _, exists := hub.clients[slowClient]; exists {
		t.Error("expected slow client to be removed from hub after buffer overflow")
	}
}