//go:build integration

package discovery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"example.com/encounter/apps/server/internal/auth"
	"example.com/encounter/internal/ice"
	"github.com/coder/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func TestAuthenticatedCrossInstanceDiscovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db, err := pgxpool.New(ctx, os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	redisOptions, err := redis.ParseURL(os.Getenv("TEST_REDIS_URL"))
	if err != nil {
		t.Fatal(err)
	}
	cache := redis.NewClient(redisOptions)
	defer cache.Close()
	prefix := "test:discovery:" + randomID() + ":"
	defer deletePrefix(context.Background(), cache, prefix)

	users := map[string]auth.User{}
	for _, name := range []string{"Ada", "Lin", "Blocked A", "Blocked B"} {
		var user auth.User
		if err := db.QueryRow(ctx, "INSERT INTO users DEFAULT VALUES RETURNING id::text").Scan(&user.ID); err != nil {
			t.Fatal(err)
		}
		user.DisplayName = name
		if _, err := db.Exec(ctx, "INSERT INTO profiles(user_id,display_name) VALUES ($1,$2)", user.ID, name); err != nil {
			t.Fatal(err)
		}
		users[name] = user
	}
	defer func() {
		for _, user := range users {
			_, _ = db.Exec(context.Background(), "DELETE FROM users WHERE id=$1", user.ID)
		}
	}()
	authenticate := func(r *http.Request) (auth.User, error) {
		cookie, err := r.Cookie("test_session")
		if err != nil {
			return auth.User{}, auth.ErrUnauthenticated
		}
		user, ok := users[cookie.Value]
		if !ok {
			return auth.User{}, auth.ErrUnauthenticated
		}
		return user, nil
	}
	root, stop := context.WithCancel(context.Background())
	defer stop()
	handler := &Handler{
		Root: root, Store: Store{Redis: cache, Prefix: prefix}, Repo: Repository{DB: db},
		Authenticate: authenticate, Origin: "https://example.test",
		ICE: ice.SharedSecretProvider{Secret: "01234567890123456789012345678901", URLs: []string{"turn:localhost:3478"}, TTL: time.Minute},
	}
	muxA, muxB := http.NewServeMux(), http.NewServeMux()
	handler.Register(muxA)
	handler.Register(muxB)
	serverA, serverB := httptest.NewServer(muxA), httptest.NewServer(muxB)
	defer serverA.Close()
	defer serverB.Close()

	assertRejectedSocket(t, ctx, serverA.URL, "", handler.Origin, http.StatusUnauthorized)
	assertRejectedSocket(t, ctx, serverA.URL, "Ada", "https://hostile.test", http.StatusForbidden)
	a := connectDiscovery(t, ctx, serverA.URL, handler.Origin, "Ada")
	b := connectDiscovery(t, ctx, serverB.URL, handler.Origin, "Lin")
	defer a.CloseNow()
	defer b.CloseNow()
	readType(t, ctx, a, "connection.ready")
	readType(t, ctx, b, "connection.ready")
	preferences := Preferences{Intent: "tech_ideas", Languages: []string{"en"}, Interests: []string{"ai", "music"}}
	sendEvent(t, ctx, a, event("queue.join", "", preferences))
	readType(t, ctx, a, "queue.joined")
	sendEvent(t, ctx, b, event("queue.join", "", preferences))
	matchA := readUntil(t, ctx, a, "match.found")
	matchB := readUntil(t, ctx, b, "match.found")
	if matchA.MatchID == "" || matchA.MatchID != matchB.MatchID {
		t.Fatalf("users received different matches: %q %q", matchA.MatchID, matchB.MatchID)
	}
	var found struct {
		Peer            Profile  `json:"peer"`
		SharedInterests []string `json:"sharedInterests"`
		Offerer         bool     `json:"offerer"`
		StartedAt       int64    `json:"startedAt"`
		ExpiresAt       int64    `json:"expiresAt"`
	}
	if json.Unmarshal(matchA.Payload, &found) != nil || found.Peer.ID != users["Lin"].ID || len(found.SharedInterests) != 2 || found.ExpiresAt-found.StartedAt != encounterLength.Milliseconds() {
		t.Fatalf("unexpected match context: %+v", found)
	}
	offer := event("webrtc.offer", matchA.MatchID, map[string]string{"type": "offer", "sdp": "v=0\r\n"})
	sendEvent(t, ctx, a, offer)
	if forwarded := readUntil(t, ctx, b, "webrtc.offer"); forwarded.MatchID != matchA.MatchID {
		t.Fatal("cross-instance signal used the wrong match")
	}
	sendEvent(t, ctx, a, event("webrtc.ice", randomID(), map[string]string{"candidate": "forged"}))
	if got := readUntil(t, ctx, a, "error"); got.MatchID == matchA.MatchID {
		t.Fatal("forged match was accepted")
	}
	a.Close(websocket.StatusNormalClosure, "test disconnect")
	readUntil(t, ctx, b, "peer.disconnected")
	a2 := connectDiscovery(t, ctx, serverA.URL, handler.Origin, "Ada")
	defer a2.CloseNow()
	readType(t, ctx, a2, "connection.ready")
	recovered := readUntil(t, ctx, a2, "match.found")
	if recovered.MatchID != matchA.MatchID {
		t.Fatal("reconnect did not revalidate and recover the active match")
	}
	var recoveredContext struct {
		Offerer         bool     `json:"offerer"`
		Intent          string   `json:"intent"`
		SharedInterests []string `json:"sharedInterests"`
	}
	if json.Unmarshal(recovered.Payload, &recoveredContext) != nil || recoveredContext.Offerer != found.Offerer || recoveredContext.Intent != "tech_ideas" || len(recoveredContext.SharedInterests) != 2 {
		t.Fatalf("recovered match context changed: %+v", recoveredContext)
	}
	readUntil(t, ctx, b, "peer.reconnected")
	sendEvent(t, ctx, a2, event("match.extend", matchA.MatchID, struct{}{}))
	if pending := readAny(t, ctx, a2); pending.Type != "match.extend_pending" {
		t.Fatalf("expected private extension pending event, got %+v", pending)
	}
	sendEvent(t, ctx, b, event("match.extend", matchA.MatchID, struct{}{}))
	readUntil(t, ctx, a2, "match.extended")
	readUntil(t, ctx, b, "match.extended")
	sendEvent(t, ctx, b, event("match.skip", matchA.MatchID, struct{}{}))
	readUntil(t, ctx, b, "match.ended")
	readUntil(t, ctx, a2, "match.ended")
	sendEvent(t, ctx, a2, event("queue.join", "", preferences))
	readUntil(t, ctx, a2, "queue.joined")
	sendEvent(t, ctx, b, event("queue.join", "", preferences))
	rematchA := readUntil(t, ctx, a2, "match.found")
	rematchB := readUntil(t, ctx, b, "match.found")
	if rematchA.MatchID == "" || rematchA.MatchID != rematchB.MatchID {
		t.Fatalf("recent pair was not used as the only-candidate fallback: %q %q", rematchA.MatchID, rematchB.MatchID)
	}
	sendEvent(t, ctx, b, event("match.leave", rematchB.MatchID, struct{}{}))
	readUntil(t, ctx, a2, "match.ended")
	readUntil(t, ctx, b, "match.ended")

	if _, err := db.Exec(ctx, "INSERT INTO blocks(blocker_user_id,blocked_user_id) VALUES ($1,$2)", users["Blocked A"].ID, users["Blocked B"].ID); err != nil {
		t.Fatal(err)
	}
	c := connectDiscovery(t, ctx, serverA.URL, handler.Origin, "Blocked A")
	d := connectDiscovery(t, ctx, serverB.URL, handler.Origin, "Blocked B")
	defer c.CloseNow()
	defer d.CloseNow()
	readType(t, ctx, c, "connection.ready")
	readType(t, ctx, d, "connection.ready")
	sendEvent(t, ctx, c, event("queue.join", "", preferences))
	readType(t, ctx, c, "queue.joined")
	sendEvent(t, ctx, d, event("queue.join", "", preferences))
	readUntil(t, ctx, d, "queue.joined")
	time.Sleep(100 * time.Millisecond)
	if _, matched, err := handler.Store.CurrentMatch(ctx, users["Blocked A"].ID); err != nil || matched {
		t.Fatalf("blocked pair matched: matched=%v err=%v", matched, err)
	}
}

func TestAtomicTwoUserClaim(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	options, err := redis.ParseURL(os.Getenv("TEST_REDIS_URL"))
	if err != nil {
		t.Fatal(err)
	}
	cache := redis.NewClient(options)
	defer cache.Close()
	prefix := "test:discovery:" + randomID() + ":"
	defer deletePrefix(context.Background(), cache, prefix)
	store := Store{Redis: cache, Prefix: prefix}
	a, b := randomID(), randomID()
	connectionA, connectionB := randomID(), randomID()
	if err := store.Connect(ctx, a, connectionA); err != nil {
		t.Fatal(err)
	}
	if err := store.Connect(ctx, b, connectionB); err != nil {
		t.Fatal(err)
	}
	prefs := Preferences{Intent: "new_friends", Languages: []string{"en"}}
	if err := store.Enqueue(ctx, a, connectionA, prefs, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := store.Enqueue(ctx, b, connectionB, prefs, time.Now()); err != nil {
		t.Fatal(err)
	}
	var successes atomic.Int32
	var group sync.WaitGroup
	for range 20 {
		group.Add(1)
		go func() {
			defer group.Done()
			ok, err := store.Claim(ctx, a, b, connectionA, connectionB, randomID(), "new_friends", []string{"music"}, []byte(`{"version":1}`), []byte(`{"version":1}`), time.Now(), false)
			if err != nil {
				t.Error(err)
				return
			}
			if ok {
				successes.Add(1)
			}
		}()
	}
	group.Wait()
	if successes.Load() != 1 {
		t.Fatalf("expected one atomic claim, got %d", successes.Load())
	}
}

func TestAuthoritativeExpiryWinsAfterDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	options, err := redis.ParseURL(os.Getenv("TEST_REDIS_URL"))
	if err != nil {
		t.Fatal(err)
	}
	cache := redis.NewClient(options)
	defer cache.Close()
	prefix := "test:lifecycle:" + randomID() + ":"
	defer deletePrefix(context.Background(), cache, prefix)
	store := Store{Redis: cache, Prefix: prefix}
	a, b, connectionA, connectionB, matchID := randomID(), randomID(), randomID(), randomID(), randomID()
	for user, connection := range map[string]string{a: connectionA, b: connectionB} {
		if err := store.Connect(ctx, user, connection); err != nil {
			t.Fatal(err)
		}
		if err := store.Enqueue(ctx, user, connection, Preferences{Intent: "new_friends", Languages: []string{"en"}}, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	if ok, err := store.Claim(ctx, a, b, connectionA, connectionB, matchID, "new_friends", nil, []byte(`{"version":1}`), []byte(`{"version":1}`), time.Now(), false); err != nil || !ok {
		t.Fatalf("claim: ok=%v err=%v", ok, err)
	}
	past := time.Now().Add(-time.Second)
	if err := cache.HSet(ctx, prefix+"match:"+matchID, "expiresAt", past.UnixMilli()).Err(); err != nil {
		t.Fatal(err)
	}
	if err := cache.ZAdd(ctx, prefix+"deadlines", redis.Z{Score: float64(past.UnixMilli()), Member: matchID}).Err(); err != nil {
		t.Fatal(err)
	}
	var extendResult int
	var extendErr error
	var group sync.WaitGroup
	group.Add(2)
	go func() {
		defer group.Done()
		extendResult, extendErr = store.Extend(ctx, a, connectionA, matchID, time.Now())
	}()
	go func() { defer group.Done(); _ = store.Sweep(ctx, time.Now()) }()
	group.Wait()
	if extendErr != nil || (extendResult != -1 && extendResult != 0) {
		t.Fatalf("expired extension result=%d err=%v", extendResult, extendErr)
	}
	if _, exists, err := store.CurrentMatch(ctx, a); err != nil || exists {
		t.Fatalf("expired match survived: exists=%v err=%v", exists, err)
	}

	c, d, connectionC, connectionD, graceMatch := randomID(), randomID(), randomID(), randomID(), randomID()
	for user, connection := range map[string]string{c: connectionC, d: connectionD} {
		if err := store.Connect(ctx, user, connection); err != nil {
			t.Fatal(err)
		}
		if err := store.Enqueue(ctx, user, connection, Preferences{Intent: "new_friends", Languages: []string{"en"}}, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	if ok, err := store.Claim(ctx, c, d, connectionC, connectionD, graceMatch, "new_friends", nil, []byte(`{"version":1}`), []byte(`{"version":1}`), time.Now(), false); err != nil || !ok {
		t.Fatalf("grace claim: ok=%v err=%v", ok, err)
	}
	if err := store.Disconnect(ctx, c, connectionC, time.Now().Add(-reconnectGrace-time.Second)); err != nil {
		t.Fatal(err)
	}
	newConnection := randomID()
	if err := store.Connect(ctx, c, newConnection); err != nil {
		t.Fatal(err)
	}
	if ok, err := store.Reconnect(ctx, c, newConnection, graceMatch, time.Now()); err != nil || ok {
		t.Fatalf("reconnect after grace: ok=%v err=%v", ok, err)
	}
	if err := store.Sweep(ctx, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, exists, err := store.CurrentMatch(ctx, d); err != nil || exists {
		t.Fatalf("disconnect-expired match survived: exists=%v err=%v", exists, err)
	}
}

func connectDiscovery(t *testing.T, ctx context.Context, server, origin, token string) *websocket.Conn {
	t.Helper()
	conn, response, err := websocket.Dial(ctx, strings.Replace(server, "http:", "ws:", 1)+"/v1/discovery/ws", &websocket.DialOptions{HTTPHeader: http.Header{"Origin": []string{origin}, "Cookie": []string{"test_session=" + token}}})
	if err != nil {
		status := 0
		if response != nil {
			status = response.StatusCode
		}
		t.Fatalf("dial failed: status=%d err=%v", status, err)
	}
	return conn
}

func assertRejectedSocket(t *testing.T, ctx context.Context, server, token, origin string, status int) {
	t.Helper()
	conn, response, err := websocket.Dial(ctx, strings.Replace(server, "http:", "ws:", 1)+"/v1/discovery/ws", &websocket.DialOptions{HTTPHeader: http.Header{"Origin": []string{origin}, "Cookie": []string{"test_session=" + token}}})
	if conn != nil {
		conn.CloseNow()
	}
	if err == nil || response == nil || response.StatusCode != status {
		t.Fatalf("socket rejection: status=%v err=%v", response, err)
	}
}

func sendEvent(t *testing.T, ctx context.Context, conn *websocket.Conn, value Envelope) {
	t.Helper()
	data, _ := json.Marshal(value)
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatal(err)
	}
}

func readType(t *testing.T, ctx context.Context, conn *websocket.Conn, kind string) Envelope {
	t.Helper()
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var value Envelope
	if json.Unmarshal(data, &value) != nil || value.Type != kind {
		t.Fatalf("wanted %s, got %s", kind, data)
	}
	return value
}

func readUntil(t *testing.T, ctx context.Context, conn *websocket.Conn, kind string) Envelope {
	t.Helper()
	for range 5 {
		value := readAny(t, ctx, conn)
		if value.Type == kind {
			return value
		}
	}
	t.Fatalf("event %s not received", kind)
	return Envelope{}
}

func readAny(t *testing.T, ctx context.Context, conn *websocket.Conn) Envelope {
	t.Helper()
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var value Envelope
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func deletePrefix(ctx context.Context, client *redis.Client, prefix string) {
	var cursor uint64
	for {
		keys, next, err := client.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return
		}
		if len(keys) > 0 {
			_ = client.Unlink(ctx, keys...).Err()
		}
		cursor = next
		if cursor == 0 {
			return
		}
	}
}
