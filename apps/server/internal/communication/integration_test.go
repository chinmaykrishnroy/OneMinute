//go:build integration

package communication

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"example.com/encounter/apps/server/internal/auth"
	"example.com/encounter/apps/server/internal/discovery"
	"example.com/encounter/apps/server/internal/social"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func TestDurableCommunicationLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := pgxpool.New(ctx, os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	options, err := redis.ParseURL(os.Getenv("TEST_REDIS_URL"))
	if err != nil {
		t.Fatal(err)
	}
	cache := redis.NewClient(options)
	defer cache.Close()
	ids := make([]string, 3)
	for i := range ids {
		if err := db.QueryRow(ctx, "INSERT INTO users DEFAULT VALUES RETURNING id::text").Scan(&ids[i]); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(ctx, "INSERT INTO profiles(user_id,display_name) VALUES($1,'Communication test')", ids[i]); err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		_, _ = db.Exec(context.Background(), "DELETE FROM connections WHERE user_low=ANY($1::uuid[]) OR user_high=ANY($1::uuid[])", ids)
		_, _ = db.Exec(context.Background(), "DELETE FROM encounters WHERE user_low=ANY($1::uuid[]) OR user_high=ANY($1::uuid[])", ids)
		_, _ = db.Exec(context.Background(), "DELETE FROM users WHERE id=ANY($1::uuid[])", ids)
		for _, id := range ids {
			cache.Del(context.Background(), "communication:busy:"+id)
		}
	}()
	connection, err := (discovery.Repository{DB: db}).PersistConnection(ctx, discovery.MatchState{ID: uuid.NewString(), UserA: ids[0], UserB: ids[1], Intent: "new_friends", StartedAt: time.Now().Add(-time.Minute).UnixMilli()})
	if err != nil {
		t.Fatal(err)
	}
	h := &Handler{DB: db, Redis: cache, Origin: "https://example.test", Authenticate: func(r *http.Request) (auth.User, error) {
		if id := r.Header.Get("X-Test-User"); id != "" {
			return auth.User{ID: id, DisplayName: "Test caller"}, nil
		}
		return auth.User{}, errors.New("unauthenticated")
	}}
	mux := http.NewServeMux()
	h.Register(mux)
	request := func(method, path, user string, body any, want int) *httptest.ResponseRecorder {
		t.Helper()
		encoded, _ := json.Marshal(body)
		r := httptest.NewRequest(method, path, bytes.NewReader(encoded))
		r.Header.Set("X-Test-User", user)
		r.Header.Set("Origin", h.Origin)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, w.Code, want, w.Body.String())
		}
		return w
	}
	request("GET", "/v1/settings", "", nil, 401)
	w := request("GET", "/v1/settings", ids[0], nil, 200)
	var settings Settings
	_ = json.Unmarshal(w.Body.Bytes(), &settings)
	if settings.Theme != "system" || !settings.ReadReceipts {
		t.Fatalf("wrong defaults: %+v", settings)
	}
	settings.Theme = "dark"
	request("PUT", "/v1/settings", ids[0], settings, 200)
	if got := h.getSettings(ctx, ids[0]).Theme; got != "dark" {
		t.Fatalf("theme not durable: %s", got)
	}
	hostile := httptest.NewRequest("PUT", "/v1/settings", bytes.NewBufferString(`{}`))
	hostile.Header.Set("X-Test-User", ids[0])
	hostile.Header.Set("Origin", "https://hostile.test")
	out := httptest.NewRecorder()
	mux.ServeHTTP(out, hostile)
	if out.Code != 403 {
		t.Fatal("cross-origin mutation allowed")
	}
	path := "/v1/connections/" + connection
	request("GET", path+"/messages", ids[2], nil, 403)
	request("POST", path+"/messages", ids[0], map[string]any{"body": "", "clientId": uuid.NewString()}, 400)
	body := map[string]any{"body": "Hello, safely saved", "clientId": uuid.NewString()}
	w = request("POST", path+"/messages", ids[0], body, 200)
	var first Message
	if err := json.Unmarshal(w.Body.Bytes(), &first); err != nil {
		t.Fatal(err)
	}
	w = request("POST", path+"/messages", ids[0], body, 200)
	var duplicate Message
	_ = json.Unmarshal(w.Body.Bytes(), &duplicate)
	if duplicate.ID != first.ID {
		t.Fatal("retry duplicated message")
	}
	var count int
	_ = db.QueryRow(ctx, "SELECT count(*) FROM messages WHERE connection_id=$1", connection).Scan(&count)
	if count != 1 {
		t.Fatalf("saved %d messages", count)
	}
	w = request("GET", path+"/messages", ids[1], nil, 200)
	var listing struct{ Messages []Message }
	_ = json.Unmarshal(w.Body.Bytes(), &listing)
	if len(listing.Messages) != 1 || listing.Messages[0].Body != first.Body {
		t.Fatal("history unavailable to recipient")
	}
	request("POST", path+"/receipt", ids[0], map[string]any{"id": first.ID, "read": true}, 400)
	sub := cache.Subscribe(ctx, "communication:user:"+ids[0])
	defer sub.Close()
	if _, err := sub.Receive(ctx); err != nil {
		t.Fatal(err)
	}
	request("POST", path+"/receipt", ids[1], map[string]any{"id": first.ID, "read": true}, 200)
	evCtx, stop := context.WithTimeout(ctx, time.Second)
	event, err := sub.ReceiveMessage(evCtx)
	stop()
	if err != nil || !bytes.Contains([]byte(event.Payload), []byte(`"type":"receipt"`)) {
		t.Fatalf("receipt not broadcast: %v", err)
	}
	request("POST", path+"/receipt", ids[1], map[string]any{"id": first.ID, "read": true}, 200)
	quiet, stop := context.WithTimeout(ctx, 150*time.Millisecond)
	_, err = sub.ReceiveMessage(quiet)
	stop()
	if err == nil {
		t.Fatal("duplicate receipt broadcast causes fetch loop")
	}
	settings.ReadReceipts = false
	settings.Typing = false
	request("PUT", "/v1/settings", ids[1], settings, 200)
	request("POST", path+"/messages", ids[0], map[string]any{"body": "Second", "clientId": uuid.NewString()}, 200)
	request("GET", path+"/messages?before="+jsonNumber(first.ID), ids[1], nil, 200)
	request("POST", path+"/typing", ids[1], nil, 200)
	w = request("GET", "/v1/notifications", ids[1], nil, 200)
	var notices []map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &notices)
	if len(notices) != 3 {
		t.Fatalf("expected connection and two message notices, got %d", len(notices))
	}
	request("POST", "/v1/notifications/read", ids[1], nil, 200)
	w = request("GET", "/v1/conversations", ids[1], nil, 200)
	var inbox struct {
		Connections []struct {
			Preview string `json:"preview"`
			Unread  int    `json:"unread"`
		} `json:"connections"`
	}
	if json.Unmarshal(w.Body.Bytes(), &inbox) != nil || len(inbox.Connections) != 1 || inbox.Connections[0].Preview != "Second" || inbox.Connections[0].Unread != 0 {
		t.Fatal("conversation preview or unread state incorrect")
	}
	w = request("POST", path+"/calls", ids[0], map[string]bool{"video": false}, 200)
	var call Call
	_ = json.Unmarshal(w.Body.Bytes(), &call)
	defer cache.Del(context.Background(), "communication:call:"+call.ID)
	request("POST", path+"/calls", ids[1], map[string]bool{"video": true}, 409)
	action := "/v1/calls/" + call.ID
	request("POST", action, ids[2], map[string]any{"type": "accept"}, 404)
	request("POST", action, ids[0], map[string]any{"type": "accept"}, 403)
	request("POST", action, ids[1], map[string]any{"type": "accept"}, 200)
	request("POST", action, ids[1], map[string]any{"type": "accept"}, 409)
	if ttl := cache.TTL(ctx, "communication:busy:"+ids[0]).Val(); ttl < 70*time.Second {
		t.Fatalf("busy lease not renewed on acceptance: %s", ttl)
	}
	request("POST", action, ids[0], map[string]any{"type": "offer", "payload": map[string]string{"type": "offer", "sdp": "v=0"}}, 200)
	request("POST", action, ids[0], map[string]any{"type": "answer", "payload": map[string]string{"type": "answer", "sdp": "v=0"}}, 403)
	request("POST", action, ids[1], map[string]any{"type": "answer", "payload": map[string]string{"type": "answer", "sdp": "v=0"}}, 200)
	request("POST", action, ids[1], map[string]any{"type": "media", "payload": map[string]bool{"video": true, "audio": true}}, 200)
	request("POST", action, ids[0], map[string]any{"type": "heartbeat"}, 200)
	if err := (social.Repository{DB: db}).Block(ctx, ids[1], ids[0]); err != nil {
		t.Fatal(err)
	}
	request("POST", path+"/messages", ids[0], map[string]any{"body": "Forbidden", "clientId": uuid.NewString()}, 403)
	request("GET", path+"/messages", ids[0], nil, 403)
	request("POST", path+"/calls", ids[0], map[string]bool{"video": true}, 403)
	request("POST", action, ids[0], map[string]any{"type": "heartbeat"}, 200)
	if cache.Exists(ctx, "communication:call:"+call.ID, "communication:busy:"+ids[0], "communication:busy:"+ids[1]).Val() != 0 {
		t.Fatal("blocked call retained leases")
	}
	w = request("GET", "/v1/notifications", ids[0], nil, 200)
	_ = json.Unmarshal(w.Body.Bytes(), &notices)
	if len(notices) != 0 {
		t.Fatal("blocked notifications visible")
	}
}

func jsonNumber(n int64) string { b, _ := json.Marshal(n); return string(b) }
