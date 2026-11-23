package realtime

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/gorilla/websocket"
)

// Keepalive and write timing. Clients must answer a server ping within
// pongWait or their read deadline expires and the connection is reaped.
const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	pingWait   = 5 * time.Second
)

// Handler returns the /ws upgrade handler. Mount it at the root of the chi
// router (outside /api/v1); nginx proxies /ws to it.
func (h *Hub) Handler() http.Handler {
	upgrader := websocket.Upgrader{
		HandshakeTimeout: pingWait,
		ReadBufferSize:   4096,
		WriteBufferSize:  4096,
		CheckOrigin:      h.allowedOrigin,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			shared.WriteError(w, http.StatusMethodNotAllowed, shared.CodeMethodNotAllowed, "Method not allowed.")
			return
		}
		d := h.depsSnapshot()
		if d == nil {
			shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
			return
		}
		// The session token is only ever read from the pl_session cookie by
		// shared.Deps; query-string credentials are deliberately ignored.
		userID, ok := d.SessionUserID(r)
		if !ok || userID <= 0 {
			shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
			return
		}

		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return // gorilla has already written the HTTP error response
		}

		conn := newConn(h, ws, userID)
		h.mu.Lock()
		h.conns[conn] = struct{}{}
		// Per-user fan-out (typing/presence) must reach every device of the
		// user, even before it subscribes to a channel or conversation.
		if h.users[userID] == nil {
			h.users[userID] = make(map[*Conn]struct{})
		}
		h.users[userID][conn] = struct{}{}
		h.mu.Unlock()

		go conn.writePump()
		h.publishPresence(userID, "online")

		conn.readPump()
		<-conn.writeDone
	})
}

func newConn(h *Hub, ws *websocket.Conn, userID int64) *Conn {
	return &Conn{
		hub:           h,
		ws:            ws,
		userID:        userID,
		send:          make(chan []byte, sendBufferSize),
		done:          make(chan struct{}),
		writeDone:     make(chan struct{}),
		channels:      make(map[int64]struct{}),
		conversations: make(map[int64]struct{}),
	}
}

// readPump is the only goroutine reading from the socket. It parses client
// subscribe/unsubscribe/ping frames and enforces the access checker.
func (c *Conn) readPump() {
	defer c.shutdown()

	c.ws.SetReadLimit(maxFrameBytes)
	_ = c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error {
		return c.ws.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			return
		}
		c.handleFrame(data)
	}
}

// writePump is the only goroutine writing application messages and control
// pings. A write mutex keeps the socket's single-writer contract explicit.
func (c *Conn) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	defer close(c.writeDone)
	defer c.shutdown()

	for {
		select {
		case data := <-c.send:
			if !c.write(websocket.TextMessage, data) {
				return
			}
		case <-ticker.C:
			if !c.write(websocket.PingMessage, nil) {
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *Conn) write(messageType int, data []byte) bool {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(messageType, data) == nil
}

// clientFrame models the small set of frames clients may send. Anything else
// (including malformed JSON) is ignored rather than closing the connection.
type clientFrame struct {
	Type           string `json:"type"`
	ChannelID      int64  `json:"channel_id"`
	ConversationID int64  `json:"conversation_id"`
}

func (c *Conn) handleFrame(data []byte) {
	var frame clientFrame
	if err := json.Unmarshal(data, &frame); err != nil {
		return
	}
	switch frame.Type {
	case "subscribe", "unsubscribe":
		// Exactly one positive scope is required; negative IDs and
		// "scope":0 frames are malformed and silently ignored.
		if frame.ChannelID < 0 || frame.ConversationID < 0 {
			return
		}
		if (frame.ChannelID > 0) == (frame.ConversationID > 0) {
			return
		}
		if frame.Type == "subscribe" {
			c.hub.Subscribe(c, c.userID, frame.ChannelID, frame.ConversationID)
			return
		}
		c.hub.unsubscribe(c, frame.ChannelID, frame.ConversationID)
	case "ping":
		c.enqueue([]byte(`{"type":"pong"}`))
	}
}
