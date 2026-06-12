// Package ws implements a WebSocket fan-out for dashboard clients. Whenever a
// server's metrics or status change, the transport layer calls Broadcast with a
// fresh snapshot and every connected dashboard receives it immediately, giving
// real-time updates without polling.
package ws

import (
	"sync"

	"github.com/gorilla/websocket"
)

// Hub tracks connected dashboard sockets and broadcasts messages to them. Each
// connection carries its auth state so admin sockets receive unmasked frames
// while anonymous sockets receive masked ones.
type Hub struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
}

type client struct {
	conn   *websocket.Conn
	send   chan []byte
	authed bool
}

// New creates an empty Hub.
func New() *Hub {
	return &Hub{clients: make(map[*client]struct{})}
}

// Add registers a websocket connection and starts its writer pump. authed marks
// whether the connection presented a valid admin token (it then receives
// unmasked broadcasts). It blocks reading control frames until the connection
// closes, then deregisters it, so callers should run it for the lifetime of the
// request handler.
func (h *Hub) Add(conn *websocket.Conn, authed bool) {
	c := &client{conn: conn, send: make(chan []byte, 16), authed: authed}

	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()

	// Writer pump: serialize all writes to this conn through its send channel.
	go func() {
		for msg := range c.send {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}()

	// Reader loop: we don't expect inbound data, but reading is required to
	// process pings/pongs and detect disconnects.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}

	h.remove(c)
}

func (h *Hub) remove(c *client) {
	h.mu.Lock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
	}
	h.mu.Unlock()
	c.conn.Close()
}

// Broadcast sends the masked frame to anonymous dashboards and the full
// (unmasked) frame to authenticated ones. Slow clients that cannot keep up are
// dropped rather than blocking the broadcaster.
func (h *Hub) Broadcast(masked, full []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		msg := masked
		if c.authed {
			msg = full
		}
		select {
		case c.send <- msg:
		default:
			// Buffer full: drop this client's pending update; it will catch up
			// on the next broadcast or reconnect.
		}
	}
}

// Count returns the number of connected dashboards.
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// CloseAll closes every connected socket, unblocking their Add reader loops.
// Used to drain the hub during graceful shutdown.
func (h *Hub) CloseAll() {
	h.mu.RLock()
	conns := make([]*websocket.Conn, 0, len(h.clients))
	for c := range h.clients {
		conns = append(conns, c.conn)
	}
	h.mu.RUnlock()
	for _, conn := range conns {
		conn.Close()
	}
}
