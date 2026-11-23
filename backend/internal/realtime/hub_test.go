package realtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"strings"
	"testing"
	"time"

	"github.com/Aswanidev-vs/promiseland/backend/internal/config"
	"github.com/Aswanidev-vs/promiseland/backend/internal/db"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/Aswanidev-vs/promiseland/backend/internal/testutil"
	"github.com/gorilla/websocket"
)

type testEnv struct {
	hub    *Hub
	d      *shared.Deps
	srv    *httptest.Server
	origin string
}

func newTestEnv(t *testing.T, checker func(int64, int64, int64) bool) *testEnv {
	t.Helper()
	database, _ := testutil.TempDB(t, "realtime.db")
	if err := db.Migrate(database, nil); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := &shared.Deps{
		DB:     database,
		Log:    logger,
		Config: config.Config{CORSOrigin: "http://localhost:5173"},
	}
	hub := NewHub(logger)
	hub.SetDeps(d)
	if checker != nil {
		hub.SetAccessChecker(checker)
	}
	srv := httptest.NewServer(hub.Handler())
	t.Cleanup(srv.Close)
	return &testEnv{hub: hub, d: d, srv: srv, origin: d.Config.CORSOrigin}
}

func (e *testEnv) wsURL() string {
	return "ws" + strings.TrimPrefix(e.srv.URL, "http") + "/ws"
}

func (e *testEnv) session(t *testing.T, username string) (int64, *http.Cookie) {
	t.Helper()
	res, err := e.d.DB.Exec(`
		INSERT INTO users(email, password_hash, display_name, username)
		VALUES (?, 'hash', ?, ?)`, username+"@example.test", username, username)
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	token := "session-" + username
	sum := sha256.Sum256([]byte(token))
	_, err = e.d.DB.Exec(`
		INSERT INTO sessions(token_hash, user_id, expires_at)
		VALUES (?, ?, datetime('now', '+1 day'))`, hex.EncodeToString(sum[:]), id)
	if err != nil {
		t.Fatal(err)
	}
	return id, &http.Cookie{Name: shared.SessionCookieName, Value: token, Path: "/"}
}

// testClient consumes frames in a dedicated goroutine. Gorilla sockets become
// unusable after a read timeout, so tests never set read deadlines on them;
// timeouts happen on a Go channel instead.
type testClient struct {
	raw    *websocket.Conn
	events chan Event
	closed chan error
}

func (e *testEnv) dialRaw(t *testing.T, cookie *http.Cookie, origin string) *websocket.Conn {
	t.Helper()
	header := http.Header{}
	if cookie != nil {
		header.Set("Cookie", cookie.String())
	}
	if origin == "" {
		origin = e.origin
	}
	header.Set("Origin", origin)
	conn, _, err := websocket.DefaultDialer.Dial(e.wsURL(), header)
	if err != nil {
		t.Fatalf("dial /ws: %v", err)
	}
	return conn
}

func (e *testEnv) client(t *testing.T, cookie *http.Cookie) *testClient {
	t.Helper()
	ws := e.dialRaw(t, cookie, "")
	c := &testClient{
		raw:    ws,
		events: make(chan Event, 256),
		closed: make(chan error, 1),
	}
	go func() {
		defer close(c.events)
		for {
			_, data, err := ws.ReadMessage()
			if err != nil {
				c.closed <- err
				return
			}
			var ev Event
			if err := json.Unmarshal(data, &ev); err != nil {
				continue
			}
			c.events <- ev
		}
	}()
	t.Cleanup(func() { _ = ws.Close() })
	return c
}

func (c *testClient) send(t *testing.T, frame any) {
	t.Helper()
	if err := c.raw.WriteJSON(frame); err != nil {
		t.Fatalf("write frame: %v", err)
	}
}

func (c *testClient) sendRaw(t *testing.T, data string) {
	t.Helper()
	if err := c.raw.WriteMessage(websocket.TextMessage, []byte(data)); err != nil {
		t.Fatal(err)
	}
}

// wait returns the first event matching match, or fails after timeout.
func (c *testClient) wait(t *testing.T, match func(Event) bool, timeout time.Duration, description string) Event {
	t.Helper()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case ev, ok := <-c.events:
			if !ok {
				t.Fatalf("client closed while waiting for %s: %v", description, <-c.closed)
			}
			if match(ev) {
				return ev
			}
		case err := <-c.closed:
			t.Fatalf("client closed while waiting for %s: %v", description, err)
		case <-timer.C:
			t.Fatalf("timed out waiting for %s", description)
		}
	}
}

func (c *testClient) expectNone(t *testing.T, match func(Event) bool, duration time.Duration, description string) {
	t.Helper()
	timer := time.NewTimer(duration)
	defer timer.Stop()
	for {
		select {
		case ev, ok := <-c.events:
			if !ok {
				t.Fatalf("client closed while checking %s: %v", description, <-c.closed)
			}
			if match(ev) {
				t.Fatalf("received unexpected %s: %+v", description, ev)
			}
		case <-timer.C:
			return
		}
	}
}

func byType(t string) func(Event) bool {
	return func(ev Event) bool { return ev.Type == t }
}

func presenceOf(userID int64, status string) func(Event) bool {
	return func(ev Event) bool {
		if ev.Type != "presence" {
			return false
		}
		payload, ok := ev.Payload.(map[string]any)
		return ok && payload["userId"] == float64(userID) && payload["status"] == status
	}
}

// serverConn returns the single server-side connection currently registered.
func (e *testEnv) serverConn(t *testing.T) *Conn {
	t.Helper()
	e.hub.mu.RLock()
	defer e.hub.mu.RUnlock()
	if len(e.hub.conns) != 1 {
		t.Fatalf("want 1 server connection, got %d", len(e.hub.conns))
	}
	for conn := range e.hub.conns {
		return conn
	}
	return nil
}

func (e *testEnv) serverConnPair(t *testing.T, userA, userB int64) (*Conn, *Conn) {
	t.Helper()
	e.hub.mu.RLock()
	defer e.hub.mu.RUnlock()
	if len(e.hub.conns) != 2 {
		t.Fatalf("want 2 server connections, got %d", len(e.hub.conns))
	}
	var connA, connB *Conn
	for conn := range e.hub.conns {
		switch conn.userID {
		case userA:
			connA = conn
		case userB:
			connB = conn
		}
	}
	if connA == nil || connB == nil {
		t.Fatalf("could not match server connections to users %d and %d", userA, userB)
	}
	return connA, connB
}

func (e *testEnv) hasSubscription(channelID, conversationID int64, conn *Conn) bool {
	e.hub.mu.RLock()
	defer e.hub.mu.RUnlock()
	if channelID > 0 {
		_, ok := e.hub.channels[channelID][conn]
		return ok
	}
	_, ok := e.hub.conversations[conversationID][conn]
	return ok
}

func waitFor(t *testing.T, cond func() bool, message string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal(message)
}

func TestUpgradeRejectsUnauthenticatedAndForbiddenOrigin(t *testing.T) {
	e := newTestEnv(t, nil)
	_, cookie := e.session(t, "alice")

	tests := []struct {
		name   string
		cookie *http.Cookie
		origin string
		query  string
		want   int
	}{
		{name: "no session cookie", origin: e.origin, want: http.StatusUnauthorized},
		{name: "token in query string only", origin: e.origin, query: "?token=" + neturl.QueryEscape(cookie.Value), want: http.StatusUnauthorized},
		{name: "foreign origin", cookie: cookie, origin: "https://evil.example", want: http.StatusForbidden},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			header := http.Header{"Origin": []string{tc.origin}}
			if tc.cookie != nil {
				header.Set("Cookie", tc.cookie.String())
			}
			_, resp, err := websocket.DefaultDialer.Dial(e.wsURL()+tc.query, header)
			if err == nil {
				t.Fatal("expected handshake to fail")
			}
			if resp == nil || resp.StatusCode != tc.want {
				t.Fatalf("want HTTP %d, got %v", tc.want, resp)
			}
		})
	}

	// A same-origin browser Origin (host matching the request Host) is accepted.
	host := strings.TrimPrefix(e.srv.URL, "http://")
	header := http.Header{
		"Origin": []string{"http://" + host},
		"Cookie": []string{cookie.String()},
	}
	conn, _, err := websocket.DefaultDialer.Dial(e.wsURL(), header)
	if err != nil {
		t.Fatalf("same-origin handshake: %v", err)
	}
	_ = conn.Close()

	resp, err := http.Post(e.srv.URL+"/ws", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("POST /ws: want 405, got %d", resp.StatusCode)
	}
}

func TestSubscribeAccessControlAndMalformedFrames(t *testing.T) {
	var allowedUser int64
	e := newTestEnv(t, func(userID, channelID, conversationID int64) bool {
		return userID == allowedUser && (channelID == 1 || conversationID == 2)
	})
	allowedUser, cookie := e.session(t, "bob")
	client := e.client(t, cookie)
	conn := e.serverConn(t)

	client.send(t, map[string]any{"type": "subscribe", "channel_id": 1})
	waitFor(t, func() bool { return e.hasSubscription(1, 0, conn) }, "authorized channel subscription missing")

	// Denied by the injected access checker.
	client.send(t, map[string]any{"type": "subscribe", "channel_id": 99})
	time.Sleep(50 * time.Millisecond)
	if e.hasSubscription(99, 0, conn) {
		t.Fatal("denied channel subscription was registered")
	}

	client.send(t, map[string]any{"type": "subscribe", "conversation_id": 2})
	waitFor(t, func() bool { return e.hasSubscription(0, 2, conn) }, "authorized conversation subscription missing")

	// Malformed frames are ignored: broken JSON, both scopes, unknown type,
	// zero scope, a negative ID, and a binary frame.
	badFrames := []string{
		`{not json`,
		`{"type":"subscribe","channel_id":1,"conversation_id":2}`,
		`{"type":"explode","channel_id":5}`,
		`{"type":"subscribe"}`,
		`{"type":"subscribe","channel_id":-1}`,
	}
	for _, raw := range badFrames {
		client.sendRaw(t, raw)
	}
	if err := client.raw.WriteMessage(websocket.BinaryMessage, []byte{0xff, 0xfe}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	e.hub.mu.RLock()
	channels := len(e.hub.channels)
	conversations := len(e.hub.conversations)
	e.hub.mu.RUnlock()
	if channels != 1 || conversations != 1 {
		t.Fatalf("malformed frames changed subscriptions: channels=%d conversations=%d", channels, conversations)
	}

	client.send(t, map[string]any{"type": "unsubscribe", "channel_id": 1})
	waitFor(t, func() bool { return !e.hasSubscription(1, 0, conn) }, "unsubscribe did not remove channel")
	e.hub.mu.RLock()
	_, stillUser := e.hub.users[allowedUser][conn]
	e.hub.mu.RUnlock()
	if !stillUser {
		t.Fatal("unsubscribe must keep the per-user connection registered")
	}

	client.send(t, map[string]any{"type": "ping"})
	client.wait(t, byType("pong"), time.Second, "pong reply")
}

func TestPublishDeliversOnlyToMatchingScope(t *testing.T) {
	e := newTestEnv(t, nil)
	idA, cookieA := e.session(t, "alice")
	idB, cookieB := e.session(t, "bob")
	a := e.client(t, cookieA)
	b := e.client(t, cookieB)
	connA, connB := e.serverConnPair(t, idA, idB)

	a.send(t, map[string]any{"type": "subscribe", "channel_id": 1})
	b.send(t, map[string]any{"type": "subscribe", "channel_id": 2})
	waitFor(t, func() bool { return e.hasSubscription(1, 0, connA) && e.hasSubscription(2, 0, connB) }, "subscriptions missing")

	e.hub.Publish(Event{Type: "message.new", ChannelID: 1, Payload: map[string]any{"id": 11}})
	got := a.wait(t, byType("message.new"), time.Second, "channel message for A")
	if got.ChannelID != 1 {
		t.Fatalf("channel_id = %d, want 1", got.ChannelID)
	}
	b.expectNone(t, byType("message.new"), 150*time.Millisecond, "channel-1 message")

	e.hub.Publish(Event{Type: "message.new", ConversationID: 7, Payload: map[string]any{"id": 12}})
	a.expectNone(t, byType("message.new"), 150*time.Millisecond, "unsubscribed conversation message")
	b.expectNone(t, byType("message.new"), 150*time.Millisecond, "unsubscribed conversation message")

	b.send(t, map[string]any{"type": "subscribe", "conversation_id": 7})
	waitFor(t, func() bool { return e.hasSubscription(0, 7, connB) }, "conversation subscription missing")
	e.hub.Publish(Event{Type: "message.new", ConversationID: 7, Payload: map[string]any{"id": 13}})
	b.wait(t, byType("message.new"), time.Second, "conversation message for B")
	a.expectNone(t, byType("message.new"), 150*time.Millisecond, "conversation-7 message")
}

func TestPublishFansOutToEveryConnectionOfAUser(t *testing.T) {
	e := newTestEnv(t, nil)
	idA, cookieA := e.session(t, "alice")
	_, cookieB := e.session(t, "bob")
	a1 := e.client(t, cookieA)
	a2 := e.client(t, cookieA)
	b := e.client(t, cookieB)

	e.hub.Publish(Event{
		Type:    "typing",
		UserIDs: []int64{idA},
		Payload: map[string]any{"isTyping": true},
	})
	a1.wait(t, byType("typing"), time.Second, "typing on device 1")
	a2.wait(t, byType("typing"), time.Second, "typing on device 2")
	b.expectNone(t, byType("typing"), 150*time.Millisecond, "typing for another user")
}

func TestPresenceBroadcastOnConnectAndDisconnect(t *testing.T) {
	e := newTestEnv(t, nil)
	aliceID, cookieA := e.session(t, "alice")
	bobID, cookieB := e.session(t, "bob")
	a := e.client(t, cookieA)
	a.wait(t, presenceOf(aliceID, "online"), time.Second, "own online presence")

	b := e.client(t, cookieB)
	a.wait(t, presenceOf(bobID, "online"), time.Second, "bob online presence")

	if err := b.raw.Close(); err != nil {
		t.Fatal(err)
	}
	a.wait(t, presenceOf(bobID, "offline"), 2*time.Second, "bob offline presence")
}

func TestSlowClientDoesNotBlockPublish(t *testing.T) {
	e := newTestEnv(t, nil)
	id, cookie := e.session(t, "slow")
	ws := e.dialRaw(t, cookie, "")
	t.Cleanup(func() { _ = ws.Close() })

	// A connection with a one-slot queue and no write pump simulates a client
	// that stopped reading, without depending on OS socket buffer sizes.
	conn := &Conn{
		hub:           e.hub,
		ws:            ws,
		userID:        id,
		send:          make(chan []byte, 1),
		done:          make(chan struct{}),
		writeDone:     make(chan struct{}),
		channels:      make(map[int64]struct{}),
		conversations: make(map[int64]struct{}),
	}
	e.hub.Subscribe(conn, id, 42, 0)

	start := time.Now()
	for i := 0; i < 200; i++ {
		e.hub.Publish(Event{Type: "message.new", ChannelID: 42, Payload: map[string]any{"seq": i}})
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("Publish blocked for %s on a slow client", elapsed)
	}
	waitFor(t, func() bool {
		select {
		case <-conn.done:
			return true
		default:
			return false
		}
	}, "slow client was not closed")
	e.hub.mu.RLock()
	_, registered := e.hub.conns[conn]
	e.hub.mu.RUnlock()
	if registered {
		t.Fatal("closed slow client remained registered")
	}
}

func TestDefaultIsProcessWideSingleton(t *testing.T) {
	if Default() != Default() {
		t.Fatal("Default() returned two different hubs")
	}
}
