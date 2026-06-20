// Package websocket implements the Hub/Client pattern for real-time broadcasts.
// Slow clients that can't keep up are disconnected to preserve Hub responsiveness.
package websocket

import (
	"log/slog"
)

type Hub struct {
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	clients    map[*Client]bool
}

func NewHub() *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:   make(chan []byte),
		clients:     make(map[*Client]bool),
	}
}

// Run starts the Hub event loop. Blocks; run in a goroutine.
func (h *Hub) Run() {
	slog.Info("WebSocket Hub started")
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			slog.Info("client connected", "total_clients", len(h.clients))

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				slog.Info("client disconnected", "total_clients", len(h.clients))
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client buffer full — disconnect slow client to keep Hub responsive.
					close(client.send)
					delete(h.clients, client)
					slog.Warn("dropped slow client", "total_clients", len(h.clients))
				}
			}
		}
	}
}

// Broadcast sends a message to all connected clients via the Hub's channel.
func (h *Hub) Broadcast(msg []byte) {
	h.broadcast <- msg
}