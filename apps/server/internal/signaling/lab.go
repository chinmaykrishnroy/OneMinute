// Package signaling contains a development-only networking room. It is not application identity.
package signaling

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"time"

	"example.com/encounter/internal/ice"
	"github.com/coder/websocket"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

type Lab struct {
	Root   context.Context
	Redis  *redis.Client
	Origin string
	ICE    ice.Provider
}

func (s *Lab) Register(mux *http.ServeMux) {
	mux.HandleFunc("/v1/lab/rooms", s.rooms)
	mux.HandleFunc("GET /v1/lab/ws", s.socket)
}
func (s *Lab) origin(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("Origin") != s.Origin {
		http.Error(w, "origin not allowed", http.StatusForbidden)
		return false
	}
	w.Header().Set("Access-Control-Allow-Origin", s.Origin)
	w.Header().Set("Vary", "Origin")
	return true
}
func (s *Lab) rooms(w http.ResponseWriter, r *http.Request) {
	if !s.origin(w, r) {
		return
	}
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(204)
		return
	}
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST, OPTIONS")
		http.Error(w, "method not allowed", 405)
		return
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	count, err := s.Redis.Eval(ctx, `local n=redis.call('INCR',KEYS[1]);if n==1 then redis.call('EXPIRE',KEYS[1],60) end;return n`, []string{"lab:rate:" + host}).Int()
	if err != nil {
		http.Error(w, "temporarily unavailable", 503)
		return
	}
	if count > 10 {
		http.Error(w, "too many rooms", 429)
		return
	}
	id := randomID()
	_, err = s.Redis.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.HSet(ctx, "lab:room:"+id, "created", "1")
		p.Expire(ctx, "lab:room:"+id, 10*time.Minute)
		return nil
	})
	if err != nil {
		http.Error(w, "temporarily unavailable", 503)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"roomId": id})
}

func (s *Lab) socket(w http.ResponseWriter, r *http.Request) {
	if !s.origin(w, r) {
		return
	}
	// The exact configured Origin was checked above, including scheme and port.
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer conn.CloseNow()
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	stop := context.AfterFunc(s.Root, cancel)
	defer stop()
	conn.SetReadLimit(64 << 10)
	initial, done := context.WithTimeout(ctx, 10*time.Second)
	_, data, err := conn.Read(initial)
	done()
	if err != nil {
		return
	}
	first, err := Decode(data)
	if err != nil || first.Type != "room.join" {
		send(ctx, conn, Event("error", "", map[string]string{"code": "invalid_join"}))
		return
	}
	var join struct {
		RoomID string `json:"roomId"`
	}
	_ = json.Unmarshal(first.Payload, &join)
	roomID, connectionID := join.RoomID, randomID()
	sub := s.Redis.Subscribe(ctx, "lab:conn:"+connectionID)
	defer sub.Close()
	if _, err := sub.Receive(ctx); err != nil {
		return
	}
	// Subscriber is live before claiming a slot, so a peer cannot race the subscription.
	result, err := s.Redis.Eval(ctx, joinScript, []string{"lab:room:" + roomID}, connectionID, roomID).Int()
	if err != nil || result == 0 {
		send(ctx, conn, Event("error", roomID, map[string]string{"code": "room_unavailable"}))
		return
	}
	defer func() {
		cleanup, c := context.WithTimeout(context.Background(), 3*time.Second)
		defer c()
		_ = s.Redis.Eval(cleanup, leaveScript, []string{"lab:room:" + roomID}, connectionID, roomID).Err()
	}()
	config, err := s.ICE.Configuration(connectionID, time.Now())
	if err != nil {
		return
	}
	if err := send(ctx, conn, Event("connection.ready", roomID, config)); err != nil {
		return
	}
	go func() {
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				return
			case message, ok := <-sub.Channel(redis.WithChannelSize(128)):
				if !ok {
					return
				}
				deadline, c := context.WithTimeout(ctx, 5*time.Second)
				err := conn.Write(deadline, websocket.MessageText, []byte(message.Payload))
				c()
				if err != nil {
					return
				}
			}
		}
	}()
	limiter := rate.NewLimiter(30, 100)
	for {
		deadline, c := context.WithTimeout(ctx, 35*time.Second)
		kind, body, err := conn.Read(deadline)
		c()
		if err != nil {
			return
		}
		if kind != websocket.MessageText || !limiter.Allow() {
			return
		}
		e, err := Decode(body)
		if err != nil || (e.MatchID != "" && e.MatchID != roomID) {
			send(ctx, conn, Event("error", roomID, map[string]string{"code": "invalid_event"}))
			return
		}
		switch e.Type {
		case "presence.heartbeat":
			n, err := s.Redis.Expire(ctx, "lab:presence:"+connectionID, 40*time.Second).Result()
			if err != nil || !n {
				return
			}
			exists, err := s.Redis.Exists(ctx, "lab:room:"+roomID).Result()
			if err != nil || exists == 0 {
				send(ctx, conn, Event("match.ended", roomID, map[string]string{"reason": "room_expired"}))
				return
			}
		case "match.leave":
			return
		case "webrtc.offer", "webrtc.answer", "webrtc.ice":
			if e.MatchID != roomID {
				return
			}
			payload, _ := json.Marshal(e)
			n, err := s.Redis.Eval(ctx, signalScript, []string{"lab:room:" + roomID}, connectionID, string(payload)).Int()
			if err != nil || n == 0 {
				send(ctx, conn, Event("error", roomID, map[string]string{"code": "peer_unavailable"}))
				return
			}
		default:
			send(ctx, conn, Event("error", roomID, map[string]string{"code": "invalid_event"}))
			return
		}
	}
}
func send(ctx context.Context, c *websocket.Conn, e Envelope) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	deadline, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return c.Write(deadline, websocket.MessageText, data)
}
func randomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(errors.New("random source unavailable"))
	}
	return hex.EncodeToString(b)
}

const joinScript = `
if redis.call('EXISTS',KEYS[1])==0 then return 0 end
local a=redis.call('HGET',KEYS[1],'a')
local b=redis.call('HGET',KEYS[1],'b')
if b then return 0 end
if a and redis.call('EXISTS','lab:presence:'..a)==0 then return 0 end
redis.call('SET','lab:presence:'..ARGV[1],1,'EX',40)
if not a then redis.call('HSET',KEYS[1],'a',ARGV[1]);return 1 end
redis.call('HSET',KEYS[1],'b',ARGV[1])
redis.call('PUBLISH','lab:conn:'..a,cjson.encode({version=1,type='match.found',matchId=ARGV[2],payload={offerer=true}}))
redis.call('PUBLISH','lab:conn:'..ARGV[1],cjson.encode({version=1,type='match.found',matchId=ARGV[2],payload={offerer=false}}))
return 2
`
const signalScript = `
local a=redis.call('HGET',KEYS[1],'a');local b=redis.call('HGET',KEYS[1],'b')
if not a or not b then return 0 end
local peer
if ARGV[1]==a then peer=b elseif ARGV[1]==b then peer=a else return 0 end
if redis.call('EXISTS','lab:presence:'..peer)==0 or redis.call('EXISTS','lab:presence:'..ARGV[1])==0 then return 0 end
redis.call('PUBLISH','lab:conn:'..peer,ARGV[2]);return 1
`
const leaveScript = `
local a=redis.call('HGET',KEYS[1],'a');local b=redis.call('HGET',KEYS[1],'b')
local peer
if ARGV[1]==a then peer=b elseif ARGV[1]==b then peer=a else return 0 end
redis.call('DEL',KEYS[1],'lab:presence:'..ARGV[1])
if peer then redis.call('PUBLISH','lab:conn:'..peer,cjson.encode({version=1,type='match.ended',matchId=ARGV[2],payload={reason='peer_disconnected'}})) end
return 1
`
