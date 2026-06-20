package websocket

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/francisco/fleet-monitor/internal/infrastructure/jwt"
	"github.com/gorilla/websocket"
)

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

const (
	maxMessageSize  = 4096
	writeBufferSize  = 65536
	sendChannelSize  = 256
	pongWait        = 60 * time.Second
	pingPeriod       = (pongWait * 9) / 10
	writeWait        = 10 * time.Second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  maxMessageSize,
	WriteBufferSize: writeBufferSize,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// ServeWS handles a WebSocket upgrade. Auth is done via ?token= query parameter
// because WebSocket upgrade requests can't set Authorization headers.
func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request, tokenService *jwt.TokenService) {
	tokenString := r.URL.Query().Get("token")
	if tokenString == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	claims, err := tokenService.Validate(tokenString)
	if err != nil {
		slog.Warn("WebSocket auth failed", "error", err)
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}

	slog.Info("WebSocket client authenticated", "user_id", claims.Subject, "role", claims.Role)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket upgrade failed", "error", err)
		return
	}

	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, sendChannelSize),
	}
	hub.register <- client

	go client.ReadPump()
	go client.WritePump()
}

// ReadPump reads from the websocket connection. Exits on disconnection or pong timeout.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)

	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		slog.Warn("failed to set read deadline", "error", err)
		return
	}

	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Warn("WebSocket read error", "error", err)
			}
			break
		}
		// Inbound messages are ignored — sensor data comes via REST.
	}
}

// WritePump writes messages to the websocket and sends periodic pings.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				slog.Warn("failed to set write deadline", "error", err)
				return
			}
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				slog.Warn("WebSocket write error", "error", err)
				return
			}

		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				slog.Warn("failed to set write deadline for ping", "error", err)
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				slog.Warn("WebSocket ping error", "error", err)
				return
			}
		}
	}
}