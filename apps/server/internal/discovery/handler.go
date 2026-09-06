// Package discovery owns authenticated live presence, queueing and distributed match claims.
package discovery

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"time"

	"example.com/encounter/apps/server/internal/auth"
	"example.com/encounter/internal/ice"
	"github.com/coder/websocket"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

type Authenticate func(*http.Request) (auth.User, error)

type Handler struct {
	Root         context.Context
	Store        Store
	Repo         Repository
	Authenticate Authenticate
	Origin       string
	ICE          ice.Provider
	Now          func() time.Time
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/discovery/ws", h.socket)
}

func (h *Handler) Run(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			_ = h.Store.Sweep(ctx, now)
		}
	}
}

func (h *Handler) socket(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Origin") != h.Origin {
		http.Error(w, "origin not allowed", http.StatusForbidden)
		return
	}
	user, err := h.Authenticate(r)
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer conn.CloseNow()
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	stop := context.AfterFunc(h.Root, cancel)
	defer stop()
	conn.SetReadLimit(maxFrameBytes)
	connectionID := randomID()
	sub := h.Store.Redis.Subscribe(ctx, h.Store.channel(user.ID))
	defer sub.Close()
	if _, err := sub.Receive(ctx); err != nil || h.Store.Connect(ctx, user.ID, connectionID) != nil {
		return
	}
	defer func() {
		cleanup, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()
		_ = h.Store.Disconnect(cleanup, user.ID, connectionID, h.now())
	}()
	iceConfig, err := h.ICE.Configuration(user.ID, h.now())
	if err != nil || write(ctx, conn, event("connection.ready", "", map[string]any{"user": user, "ice": iceConfig})) != nil {
		return
	}
	if current, ok, err := h.Store.CurrentMatch(ctx, user.ID); err != nil {
		return
	} else if ok {
		profile, err := h.Repo.Profile(ctx, peer(current, user.ID))
		reconnected, reconnectErr := h.Store.Reconnect(ctx, user.ID, connectionID, current.ID, h.now())
		if err != nil || reconnectErr != nil || !reconnected || write(ctx, conn, event("match.found", current.ID, map[string]any{"peer": profile, "sharedInterests": current.SharedInterests, "intent": current.Intent, "offerer": current.UserA == user.ID, "recovered": true, "state": current.State, "startedAt": current.StartedAt, "expiresAt": current.ExpiresAt})) != nil {
			return
		}
	}
	go h.deliver(ctx, cancel, conn, sub)
	limiter := rate.NewLimiter(30, 100)
	for {
		readCtx, done := context.WithTimeout(ctx, 35*time.Second)
		kind, body, err := conn.Read(readCtx)
		done()
		if err != nil || kind != websocket.MessageText || !limiter.Allow() {
			return
		}
		envelope, err := decode(body)
		if err != nil {
			_ = write(ctx, conn, event("error", "", map[string]string{"code": "invalid_event"}))
			return
		}
		switch envelope.Type {
		case "presence.heartbeat":
			ok, err := h.Store.Heartbeat(ctx, user.ID, connectionID)
			if err != nil || !ok {
				return
			}
		case "queue.join":
			var preferences Preferences
			_ = json.Unmarshal(envelope.Payload, &preferences)
			if err := h.Store.Enqueue(ctx, user.ID, connectionID, preferences, h.now()); err != nil {
				_ = write(ctx, conn, event("error", "", map[string]string{"code": "queue_unavailable"}))
				continue
			}
			if write(ctx, conn, event("queue.joined", "", map[string]any{"preferences": preferences, "joinedAt": h.now().UTC()})) != nil {
				return
			}
			if err := h.tryMatch(ctx, user, connectionID, preferences); err != nil {
				_ = write(ctx, conn, event("error", "", map[string]string{"code": "matchmaking_unavailable"}))
			}
		case "queue.leave":
			if err := h.Store.LeaveQueue(ctx, user.ID, connectionID); err != nil {
				return
			}
			_ = write(ctx, conn, event("queue.left", "", struct{}{}))
		case "webrtc.offer", "webrtc.answer", "webrtc.ice":
			if envelope.MatchID == "" {
				return
			}
			payload, _ := json.Marshal(envelope)
			ok, err := h.Store.Signal(ctx, user.ID, connectionID, envelope.MatchID, payload)
			if err != nil || !ok {
				_ = write(ctx, conn, event("error", envelope.MatchID, map[string]string{"code": "peer_unavailable"}))
			}
		case "match.leave":
			if envelope.MatchID == "" {
				return
			}
			_, _ = h.Store.EndMatch(ctx, user.ID, connectionID, envelope.MatchID, "left")
		case "match.skip":
			if envelope.MatchID == "" {
				return
			}
			_, _ = h.Store.EndMatch(ctx, user.ID, connectionID, envelope.MatchID, "skipped")
		case "match.extend":
			if envelope.MatchID == "" {
				return
			}
			result, err := h.Store.Extend(ctx, user.ID, connectionID, envelope.MatchID, h.now())
			if err != nil || result <= 0 {
				code := "match_unavailable"
				if err != nil {
					code = "lifecycle_error"
				}
				_ = write(ctx, conn, event("error", envelope.MatchID, map[string]string{"code": code}))
				continue
			}
			if result == 1 {
				_ = write(ctx, conn, event("match.extend_pending", envelope.MatchID, struct{}{}))
			}
		}
	}
}

func (h *Handler) deliver(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn, sub *redis.PubSub) {
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-sub.Channel(redis.WithChannelSize(128)):
			if !ok {
				return
			}
			writeCtx, done := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Write(writeCtx, websocket.MessageText, []byte(message.Payload))
			done()
			if err != nil {
				return
			}
		}
	}
}

func (h *Handler) tryMatch(ctx context.Context, user auth.User, connectionID string, preferences Preferences) error {
	candidates, err := h.Store.Candidates(ctx, user.ID, 99)
	if err != nil {
		return err
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left := len(sharedInterests(preferences.Interests, candidates[i].Preferences.Interests))
		right := len(sharedInterests(preferences.Interests, candidates[j].Preferences.Interests))
		if left == right {
			return candidates[i].JoinedAt < candidates[j].JoinedAt
		}
		return left > right
	})
	for _, candidate := range candidates {
		if !compatible(preferences, candidate.Preferences) {
			continue
		}
		recent, err := h.Store.Recent(ctx, user.ID, candidate.ID)
		if err != nil || recent {
			continue
		}
		eligible, err := h.Repo.EligiblePair(ctx, user.ID, candidate.ID)
		if err != nil || !eligible {
			continue
		}
		candidateProfile, err := h.Repo.Profile(ctx, candidate.ID)
		if err != nil {
			continue
		}
		shared := sharedInterests(preferences.Interests, candidate.Preferences.Interests)
		matchID := randomID()
		startedAt := h.now().UTC()
		expiresAt := startedAt.Add(encounterLength)
		lifecycle := map[string]any{"startedAt": startedAt.UnixMilli(), "expiresAt": expiresAt.UnixMilli(), "state": "active"}
		eventForUser, _ := json.Marshal(event("match.found", matchID, merge(lifecycle, map[string]any{"peer": candidateProfile, "sharedInterests": shared, "intent": resolvedIntent(preferences.Intent, candidate.Preferences.Intent), "offerer": true})))
		eventForCandidate, _ := json.Marshal(event("match.found", matchID, merge(lifecycle, map[string]any{"peer": Profile{ID: user.ID, DisplayName: user.DisplayName, AvatarURL: user.AvatarURL}, "sharedInterests": shared, "intent": resolvedIntent(preferences.Intent, candidate.Preferences.Intent), "offerer": false})))
		claimed, err := h.Store.Claim(ctx, user.ID, candidate.ID, connectionID, candidate.ConnectionID, matchID, resolvedIntent(preferences.Intent, candidate.Preferences.Intent), shared, eventForUser, eventForCandidate, startedAt)
		if err != nil {
			return err
		}
		if claimed {
			return nil
		}
	}
	return nil
}

func merge(first, second map[string]any) map[string]any {
	result := make(map[string]any, len(first)+len(second))
	for key, value := range first {
		result[key] = value
	}
	for key, value := range second {
		result[key] = value
	}
	return result
}

func resolvedIntent(first, second string) string {
	if first == "surprise_me" {
		return second
	}
	return first
}

func write(ctx context.Context, conn *websocket.Conn, envelope Envelope) error {
	data, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	writeCtx, done := context.WithTimeout(ctx, 5*time.Second)
	defer done()
	return conn.Write(writeCtx, websocket.MessageText, data)
}

func randomID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		panic(errors.New("random source unavailable"))
	}
	return hex.EncodeToString(value)
}

func (h *Handler) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}
