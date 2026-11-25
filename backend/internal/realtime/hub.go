// Package realtime implements the Release 3 WebSocket hub for chat presence,
// typing and message events. It is deliberately independent of internal/chat:
// the chat package injects an access checker and publishes through this hub.
package realtime

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Aswanidev-vs/promiseland/backend/internal/config"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/gorilla/websocket"
)

// Connection and frame limits. A client that falls more than
// maxDroppedEvents behind is closed so publishers never block on it.
const (
	sendBufferSize   = 64
	maxDroppedEvents = 16
	maxFrameBytes    = 1 << 20 // 1 MiB inbound JSON frames
)

// Event is the envelope pushed to WebSocket clients. Chat handlers publish
// message.new, message.deleted, reaction.updated and typing events; the hub
// publishes presence on connect/disconnect.
type Event struct {
	Type           string  `json:"type"`
	Payload        any     `json:"payload"`
	ChannelID      int64   `json:"channel_id,omitempty"`
	ConversationID int64   `json:"conversation_id,omitempty"`
	UserIDs        []int64 `json:"user_ids,omitempty"`
}

// Conn is a single authenticated WebSocket connection. Its fields are
// private: callers only pass a *Conn back to Hub.Subscribe.
type Conn struct {
	hub    *Hub
	ws     *websocket.Conn
	userID int64

	send      chan []byte
	done      chan struct{}
	writeDone chan struct{}

	writeMu   sync.Mutex
	closeOnce sync.Once
	drops     atomic.Int64

	// Guarded by hub.mu.
	channels      map[int64]struct{}
	conversations map[int64]struct{}
}

// Hub is the process-wide registry of live connections and subscriptions.
// All maps are guarded by mu; Publish never blocks on a client.
type Hub struct {
	mu sync.RWMutex

	logger        *slog.Logger
	deps          *shared.Deps
	accessChecker func(userID, channelID, conversationID int64) bool

	conns         map[*Conn]struct{}
	channels      map[int64]map[*Conn]struct{}
	conversations map[int64]map[*Conn]struct{}
	users         map[int64]map[*Conn]struct{}
}

var (
	defaultMu  sync.Mutex
	defaultHub *Hub
)

// NewHub builds an empty hub. A nil logger falls back to slog.Default.
func NewHub(logger *slog.Logger) *Hub {
	if logger == nil {
		logger = slog.Default()
	}
	return &Hub{
		logger:        logger,
		conns:         make(map[*Conn]struct{}),
		channels:      make(map[int64]map[*Conn]struct{}),
		conversations: make(map[int64]map[*Conn]struct{}),
		users:         make(map[int64]map[*Conn]struct{}),
	}
}

// Default returns the process-wide singleton used by the root router and by
// chat handlers. It is safe to call concurrently.
func Default() *Hub {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	if defaultHub == nil {
		defaultHub = NewHub(nil)
	}
	return defaultHub
}

// SetDeps supplies the dependency bundle used for pl_session lookups during
// the upgrade handshake. Call it once at startup, before serving traffic.
func (h *Hub) SetDeps(d *shared.Deps) {
	h.mu.Lock()
	h.deps = d
	h.mu.Unlock()
}

// SetAccessChecker installs the chat-package membership check. A nil checker
// allows every subscription (useful before chat is registered or in tests).
func (h *Hub) SetAccessChecker(fn func(userID, channelID, conversationID int64) bool) {
	h.mu.Lock()
	h.accessChecker = fn
	h.mu.Unlock()
}

// Subscribe registers conn for a channel or conversation scope and for the
// user's per-user fan-out set. It re-checks the injected access checker so no
// caller can bypass subscription authorization.
func (h *Hub) Subscribe(conn *Conn, userID int64, channelID, conversationID int64) {
	if conn == nil {
		return
	}
	select {
	case <-conn.done:
		return
	default:
	}
	if channelID <= 0 && conversationID <= 0 {
		return
	}
	h.mu.RLock()
	checker := h.accessChecker
	h.mu.RUnlock()
	if checker != nil && !checker(userID, channelID, conversationID) {
		h.logger.Warn("realtime subscription denied",
			"user_id", userID, "channel_id", channelID, "conversation_id", conversationID)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if userID > 0 {
		if h.users[userID] == nil {
			h.users[userID] = make(map[*Conn]struct{})
		}
		h.users[userID][conn] = struct{}{}
	}
	if channelID > 0 {
		if h.channels[channelID] == nil {
			h.channels[channelID] = make(map[*Conn]struct{})
		}
		h.channels[channelID][conn] = struct{}{}
		conn.channels[channelID] = struct{}{}
	}
	if conversationID > 0 {
		if h.conversations[conversationID] == nil {
			h.conversations[conversationID] = make(map[*Conn]struct{})
		}
		h.conversations[conversationID][conn] = struct{}{}
		conn.conversations[conversationID] = struct{}{}
	}
}

// unsubscribe removes a single scope; per-user delivery remains active until
// the connection closes.
func (h *Hub) unsubscribe(conn *Conn, channelID, conversationID int64) {
	if conn == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if channelID > 0 {
		delete(h.channels[channelID], conn)
		if len(h.channels[channelID]) == 0 {
			delete(h.channels, channelID)
		}
		delete(conn.channels, channelID)
	}
	if conversationID > 0 {
		delete(h.conversations[conversationID], conn)
		if len(h.conversations[conversationID]) == 0 {
			delete(h.conversations, conversationID)
		}
		delete(conn.conversations, conversationID)
	}
}

// removeConn drops every reference to a closed connection.
func (h *Hub) removeConn(conn *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for channelID := range conn.channels {
		delete(h.channels[channelID], conn)
		if len(h.channels[channelID]) == 0 {
			delete(h.channels, channelID)
		}
	}
	for conversationID := range conn.conversations {
		delete(h.conversations[conversationID], conn)
		if len(h.conversations[conversationID]) == 0 {
			delete(h.conversations, conversationID)
		}
	}
	delete(h.users[conn.userID], conn)
	if len(h.users[conn.userID]) == 0 {
		delete(h.users, conn.userID)
	}
	delete(h.conns, conn)
}

// Publish marshals ev once and fans it out to the union of the requested
// scopes: explicit UserIDs, channel subscribers and conversation subscribers.
// A presence event without any scope is broadcast to all live connections.
// Sends are non-blocking; slow clients are dropped and eventually closed.
func (h *Hub) Publish(ev Event) {
	data, err := json.Marshal(ev)
	if err != nil {
		h.logger.Warn("realtime event marshal failed", "type", ev.Type)
		return
	}

	targets := make(map[*Conn]struct{})
	h.mu.RLock()
	if ev.Type == "presence" && ev.ChannelID == 0 && ev.ConversationID == 0 && len(ev.UserIDs) == 0 {
		for conn := range h.conns {
			targets[conn] = struct{}{}
		}
	} else {
		for _, userID := range ev.UserIDs {
			for conn := range h.users[userID] {
				targets[conn] = struct{}{}
			}
		}
		if ev.ChannelID > 0 {
			for conn := range h.channels[ev.ChannelID] {
				targets[conn] = struct{}{}
			}
		}
		if ev.ConversationID > 0 {
			for conn := range h.conversations[ev.ConversationID] {
				targets[conn] = struct{}{}
			}
		}
	}
	h.mu.RUnlock()

	for conn := range targets {
		conn.enqueue(data)
	}
}

// publishPresence emits the payload shape {"userId":N,"status":"..."}.
func (h *Hub) publishPresence(userID int64, status string) {
	if userID <= 0 {
		return
	}
	h.Publish(Event{
		Type:    "presence",
		Payload: presencePayload{UserID: userID, Status: status},
	})
}

type presencePayload struct {
	UserID int64  `json:"userId"`
	Status string `json:"status"`
}

func (c *Conn) enqueue(data []byte) {
	select {
	case <-c.done:
		return
	default:
	}
	select {
	case c.send <- data:
	case <-c.done:
	default:
		// Never block a publisher (and therefore REST chat handlers) on a
		// client that is not reading. Close persistently slow clients so their
		// pumps and subscriptions cannot accumulate.
		if c.drops.Add(1) > maxDroppedEvents {
			c.hub.logger.Warn("closing slow realtime client", "user_id", c.userID)
			c.shutdown()
		}
	}
}

// shutdown is idempotent and may be called from any pump or from Publish.
func (c *Conn) shutdown() {
	c.closeOnce.Do(func() {
		close(c.done)
		if c.ws != nil {
			_ = c.ws.Close()
		}
		c.hub.removeConn(c)
		c.hub.publishPresence(c.userID, "offline")
	})
}

// depsSnapshot reads the dependency bundle without holding a lock while the
// (potentially database-backed) session lookup runs.
func (h *Hub) depsSnapshot() *shared.Deps {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.deps
}

// allowedOrigin accepts same-origin requests and the configured CORS origin.
// WebSocket handshakes carry a browser Origin header, never credentials in
// query strings.
func (h *Hub) allowedOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true // non-browser clients, same as gorilla's default policy
	}
	d := h.depsSnapshot()
	if d == nil {
		return false
	}
	allowed := strings.TrimSpace(d.Config.CORSOrigin)
	if allowed == "" {
		allowed = config.DefaultCORSOrigin
	}
	// A wildcard never authorises a foreign origin here: WebSocket handshakes
	// carry the session cookie, so only same-host and the exact configured
	// origin are accepted.
	if sameHost(origin, r.Host) {
		return true
	}
	allowed = strings.TrimRight(allowed, "/")
	return allowed != "*" && strings.TrimRight(origin, "/") == allowed
}

// sameHost reports whether an Origin's host (including explicit port) matches
// the request Host. The scheme is ignored so wss/ws clients behind TLS match
// their https/http web origin.
func sameHost(origin, host string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	switch u.Scheme {
	case "http", "https", "ws", "wss":
	default:
		return false
	}
	return u.Host != "" && strings.EqualFold(u.Host, host)
}
