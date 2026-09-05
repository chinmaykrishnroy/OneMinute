//go:build integration

package signaling

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

	"example.com/encounter/internal/ice"
	"github.com/coder/websocket"
	"github.com/redis/go-redis/v9"
)

func testLab(t *testing.T) (*Lab, *redis.Client) {
	t.Helper()
	opts, err := redis.ParseURL(os.Getenv("TEST_REDIS_URL"))
	if err != nil {
		t.Fatal(err)
	}
	client := redis.NewClient(opts)
	t.Cleanup(func() { client.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return &Lab{Root: ctx, Redis: client, Origin: "http://localhost:3000", ICE: ice.SharedSecretProvider{Secret: "01234567890123456789012345678901", URLs: []string{"turn:localhost:3478"}, TTL: time.Minute}}, client
}
func TestAtomicRoomSlots(t *testing.T) {
	_, client := testLab(t)
	ctx := context.Background()
	room := randomID()
	key := "lab:room:" + room
	if err := client.HSet(ctx, key, "created", 1).Err(); err != nil {
		t.Fatal(err)
	}
	defer client.Del(ctx, key)
	var claimed atomic.Int32
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := randomID()
			defer client.Del(ctx, "lab:presence:"+id)
			n, err := client.Eval(ctx, joinScript, []string{key}, id, room).Int()
			if err != nil {
				t.Error(err)
				return
			}
			if n > 0 {
				claimed.Add(1)
				time.Sleep(100 * time.Millisecond)
			}
		}()
	}
	wg.Wait()
	if claimed.Load() != 2 {
		t.Fatalf("expected exactly two claims; got %d", claimed.Load())
	}
}
func TestCrossInstanceSignaling(t *testing.T) {
	lab, client := testLab(t)
	muxA, muxB := http.NewServeMux(), http.NewServeMux()
	lab.Register(muxA)
	lab.Register(muxB)
	serverA, serverB := httptest.NewServer(muxA), httptest.NewServer(muxB)
	defer serverA.Close()
	defer serverB.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	room := randomID()
	key := "lab:room:" + room
	if err := client.HSet(ctx, key, "created", 1).Err(); err != nil {
		t.Fatal(err)
	}
	defer client.Del(context.Background(), key)
	connect := func(server string) *websocket.Conn {
		c, _, err := websocket.Dial(ctx, strings.Replace(server, "http:", "ws:", 1)+"/v1/lab/ws", &websocket.DialOptions{HTTPHeader: http.Header{"Origin": []string{lab.Origin}}})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { c.CloseNow() })
		if err := send(ctx, c, Event("room.join", "", map[string]string{"roomId": room})); err != nil {
			t.Fatal(err)
		}
		return c
	}
	read := func(c *websocket.Conn, kind string) Envelope {
		t.Helper()
		_, b, err := c.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var e Envelope
		if err := json.Unmarshal(b, &e); err != nil {
			t.Fatal(err)
		}
		if e.Type != kind {
			t.Fatalf("wanted %s, got %s", kind, e.Type)
		}
		return e
	}
	a := connect(serverA.URL)
	read(a, "connection.ready")
	b := connect(serverB.URL)
	read(b, "connection.ready")
	read(a, "match.found")
	read(b, "match.found")
	third := connect(serverB.URL)
	read(third, "error")
	if err := send(ctx, a, Event("webrtc.offer", room, map[string]string{"type": "offer", "sdp": "v=0\r\n"})); err != nil {
		t.Fatal(err)
	}
	offer := read(b, "webrtc.offer")
	if offer.MatchID != room {
		t.Fatal("wrong match forwarded")
	}
	if err := send(ctx, a, Event("webrtc.ice", randomID(), map[string]string{"candidate": "candidate:forged"})); err != nil {
		t.Fatal(err)
	}
	read(a, "error")
	read(b, "match.ended")
	request, _ := http.NewRequest("POST", serverA.URL+"/v1/lab/rooms", nil)
	request.Header.Set("Origin", "https://attacker.example")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 403 {
		t.Fatal("hostile HTTP origin accepted")
	}
	c, response, err := websocket.Dial(ctx, strings.Replace(serverA.URL, "http:", "ws:", 1)+"/v1/lab/ws", &websocket.DialOptions{HTTPHeader: http.Header{"Origin": []string{"https://attacker.example"}}})
	if c != nil {
		c.CloseNow()
	}
	if err == nil || response.StatusCode != 403 {
		t.Fatal("hostile websocket origin accepted")
	}
}
