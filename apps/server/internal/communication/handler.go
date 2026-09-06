// Package communication provides durable connection chat and transient call signaling.
package communication

import (
	"context"
	"encoding/json"
	"errors"
	"example.com/encounter/apps/server/internal/auth"
	"example.com/encounter/internal/ice"
	"github.com/coder/websocket"
	"github.com/google/uuid"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	DB           *pgxpool.Pool
	Redis        *redis.Client
	Authenticate func(*http.Request) (auth.User, error)
	Origin       string
	ICE          ice.SharedSecretProvider
}
type Settings struct {
	Theme         string `json:"theme"`
	Notifications bool   `json:"notifications"`
	Typing        bool   `json:"typing"`
	ReadReceipts  bool   `json:"readReceipts"`
}
type Message struct {
	ID        int64     `json:"id"`
	SenderID  string    `json:"senderId"`
	ClientID  string    `json:"clientId"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

func (h *Handler) Register(m *http.ServeMux) {
	m.HandleFunc("GET /v1/moments", h.moments)
	m.HandleFunc("POST /v1/moments", h.moments)
	m.HandleFunc("DELETE /v1/moments/{id}", h.deleteMoment)
	m.HandleFunc("GET /v1/conversations", h.conversations)
	m.HandleFunc("GET /v1/events/ws", h.events)
	m.HandleFunc("GET /v1/settings", h.settings)
	m.HandleFunc("PUT /v1/settings", h.settings)
	m.HandleFunc("GET /v1/notifications", h.notifications)
	m.HandleFunc("POST /v1/notifications/read", h.readNotifications)
	m.HandleFunc("GET /v1/connections/{id}/messages", h.messages)
	m.HandleFunc("POST /v1/connections/{id}/messages", h.messages)
	m.HandleFunc("POST /v1/connections/{id}/receipt", h.receipt)
	m.HandleFunc("POST /v1/connections/{id}/typing", h.typing)
	m.HandleFunc("POST /v1/connections/{id}/calls", h.invite)
	m.HandleFunc("POST /v1/calls/{id}", h.callAction)
}
func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return e
	}
	if e := d.Decode(new(any)); e != io.EOF {
		return errors.New("trailing data")
	}
	return nil
}
func (h *Handler) user(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	u, e := h.Authenticate(r)
	if e != nil {
		reply(w, 401, map[string]string{"error": "Sign in required"})
		return u, false
	}
	if r.Method != "GET" && r.Header.Get("Origin") != h.Origin {
		reply(w, 403, map[string]string{"error": "Origin rejected"})
		return u, false
	}
	if r.Method != "GET" && !h.allow(w, r, u.ID, "mutations", 360) {
		return u, false
	}
	return u, true
}

// Runtime limits are per authenticated account and shared across instances.
func (h *Handler) allow(w http.ResponseWriter, r *http.Request, user, bucket string, maximum int) bool {
	count, err := h.Redis.Eval(r.Context(), `local n=redis.call('INCR',KEYS[1]);if n==1 then redis.call('EXPIRE',KEYS[1],60) end;return n`, []string{"communication:limit:" + bucket + ":" + user}).Int()
	if err != nil {
		reply(w, 503, map[string]string{"error": "Please try again shortly"})
		return false
	}
	if count > maximum {
		w.Header().Set("Retry-After", "60")
		reply(w, 429, map[string]string{"error": "Please slow down and try again shortly"})
		return false
	}
	return true
}
func (h *Handler) peer(ctx context.Context, user, id string) (string, error) {
	if _, e := uuid.Parse(id); e != nil {
		return "", e
	}
	var peer string
	e := h.DB.QueryRow(ctx, `SELECT CASE WHEN c.user_low=$1 THEN c.user_high::text ELSE c.user_low::text END FROM connections c WHERE c.id=$2 AND c.ended_at IS NULL AND ($1=c.user_low OR $1=c.user_high) AND NOT EXISTS(SELECT 1 FROM blocks b WHERE (b.blocker_user_id=c.user_low AND b.blocked_user_id=c.user_high) OR (b.blocker_user_id=c.user_high AND b.blocked_user_id=c.user_low)) AND NOT EXISTS(SELECT 1 FROM users WHERE id IN(c.user_low,c.user_high) AND status<>'active')`, user, id).Scan(&peer)
	return peer, e
}
func (h *Handler) authorized(w http.ResponseWriter, r *http.Request) (auth.User, string, bool) {
	u, ok := h.user(w, r)
	if !ok {
		return u, "", false
	}
	p, e := h.peer(r.Context(), u.ID, r.PathValue("id"))
	if e != nil {
		reply(w, 403, map[string]string{"error": "Connection unavailable"})
		return u, "", false
	}
	return u, p, true
}
func (h *Handler) emit(ctx context.Context, user string, event any) {
	b, _ := json.Marshal(event)
	_ = h.Redis.Publish(ctx, "communication:user:"+user, b).Err()
}
func (h *Handler) getSettings(ctx context.Context, user string) Settings {
	s := Settings{"system", true, true, true}
	_ = h.DB.QueryRow(ctx, "SELECT theme,notifications,typing,read_receipts FROM user_settings WHERE user_id=$1", user).Scan(&s.Theme, &s.Notifications, &s.Typing, &s.ReadReceipts)
	return s
}
func (h *Handler) settings(w http.ResponseWriter, r *http.Request) {
	u, ok := h.user(w, r)
	if !ok {
		return
	}
	if r.Method == "GET" {
		reply(w, 200, h.getSettings(r.Context(), u.ID))
		return
	}
	var s Settings
	if decode(w, r, &s) != nil || (s.Theme != "system" && s.Theme != "light" && s.Theme != "dark") {
		reply(w, 400, nil)
		return
	}
	_, e := h.DB.Exec(r.Context(), `INSERT INTO user_settings(user_id,theme,notifications,typing,read_receipts) VALUES($1,$2,$3,$4,$5) ON CONFLICT(user_id) DO UPDATE SET theme=$2,notifications=$3,typing=$4,read_receipts=$5`, u.ID, s.Theme, s.Notifications, s.Typing, s.ReadReceipts)
	if e != nil {
		reply(w, 500, nil)
		return
	}
	h.emit(r.Context(), u.ID, map[string]any{"type": "settings.changed", "settings": s})
	reply(w, 200, s)
}
func (h *Handler) messages(w http.ResponseWriter, r *http.Request) {
	u, p, ok := h.authorized(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	id := r.PathValue("id")
	if r.Method == "GET" {
		before, _ := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64)
		rows, e := h.DB.Query(ctx, `SELECT id,sender_id::text,client_id::text,body,created_at FROM messages WHERE connection_id=$1 AND ($2::bigint=0 OR id<$2) ORDER BY id DESC LIMIT 60`, id, before)
		if e != nil {
			reply(w, 500, nil)
			return
		}
		defer rows.Close()
		items := []Message{}
		for rows.Next() {
			var m Message
			if rows.Scan(&m.ID, &m.SenderID, &m.ClientID, &m.Body, &m.CreatedAt) != nil {
				reply(w, 500, nil)
				return
			}
			items = append(items, m)
		}
		if rows.Err() != nil {
			reply(w, 500, nil)
			return
		}
		var delivered, read int64
		_ = h.DB.QueryRow(ctx, "SELECT delivered_id,read_id FROM message_receipts WHERE connection_id=$1 AND user_id=$2", id, p).Scan(&delivered, &read)
		reply(w, 200, map[string]any{"messages": items, "deliveredId": delivered, "readId": read})
		return
	}
	var b struct {
		Body     string `json:"body"`
		ClientID string `json:"clientId"`
	}
	if decode(w, r, &b) != nil {
		reply(w, 400, nil)
		return
	}
	b.Body = strings.TrimSpace(b.Body)
	if _, e := uuid.Parse(b.ClientID); e != nil || len([]rune(b.Body)) < 1 || len([]rune(b.Body)) > 4000 {
		reply(w, 400, nil)
		return
	}
	// Lock the relationship through commit so removal cannot race an authorized send.
	tx, e := h.DB.Begin(ctx)
	if e != nil {
		reply(w, 500, nil)
		return
	}
	defer tx.Rollback(ctx)
	var active bool
	e = tx.QueryRow(ctx, "SELECT ended_at IS NULL FROM connections WHERE id=$1 FOR UPDATE", id).Scan(&active)
	if e != nil || !active {
		reply(w, 403, nil)
		return
	}
	var m Message
	e = tx.QueryRow(ctx, `INSERT INTO messages(connection_id,sender_id,client_id,body) VALUES($1,$2,$3,$4) ON CONFLICT(sender_id,client_id) DO UPDATE SET body=messages.body WHERE messages.connection_id=EXCLUDED.connection_id RETURNING id,sender_id::text,client_id::text,body,created_at`, id, u.ID, b.ClientID, b.Body).Scan(&m.ID, &m.SenderID, &m.ClientID, &m.Body, &m.CreatedAt)
	if e != nil {
		reply(w, 409, nil)
		return
	}
	_, e = tx.Exec(ctx, `INSERT INTO notifications(user_id,connection_id,kind,reference) VALUES($1,$2,'message',$3) ON CONFLICT DO NOTHING`, p, id, strconv.FormatInt(m.ID, 10))
	if e != nil || tx.Commit(ctx) != nil {
		reply(w, 500, nil)
		return
	}
	event := map[string]any{"type": "message.created", "connectionId": id, "message": m}
	h.emit(ctx, p, event)
	h.emit(ctx, u.ID, event)
	reply(w, 200, m)
}
func (h *Handler) receipt(w http.ResponseWriter, r *http.Request) {
	u, p, ok := h.authorized(w, r)
	if !ok {
		return
	}
	var b struct {
		ID   int64 `json:"id"`
		Read bool  `json:"read"`
	}
	if decode(w, r, &b) != nil || b.ID < 1 {
		reply(w, 400, nil)
		return
	}
	ctx := r.Context()
	var valid bool
	_ = h.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM messages WHERE id=$1 AND connection_id=$2 AND sender_id=$3)", b.ID, r.PathValue("id"), p).Scan(&valid)
	if !valid {
		reply(w, 400, nil)
		return
	}
	read := int64(0)
	if b.Read && h.getSettings(ctx, u.ID).ReadReceipts {
		read = b.ID
	}
	changed, e := h.DB.Exec(ctx, `INSERT INTO message_receipts(connection_id,user_id,delivered_id,read_id) VALUES($1,$2,$3,$4) ON CONFLICT(connection_id,user_id) DO UPDATE SET delivered_id=GREATEST(message_receipts.delivered_id,$3),read_id=GREATEST(message_receipts.read_id,$4) WHERE message_receipts.delivered_id<$3 OR message_receipts.read_id<$4`, r.PathValue("id"), u.ID, b.ID, read)
	if e != nil {
		reply(w, 500, nil)
		return
	}
	if b.Read {
		_, _ = h.DB.Exec(ctx, "UPDATE notifications SET read_at=now() WHERE user_id=$1 AND connection_id=$2 AND kind='message' AND reference::bigint<=$3", u.ID, r.PathValue("id"), b.ID)
	}
	if changed.RowsAffected() > 0 {
		h.emit(ctx, p, map[string]any{"type": "receipt", "connectionId": r.PathValue("id"), "deliveredId": b.ID, "readId": read})
	}
	reply(w, 200, map[string]bool{"ok": true})
}
func (h *Handler) typing(w http.ResponseWriter, r *http.Request) {
	u, p, ok := h.authorized(w, r)
	if !ok {
		return
	}
	if h.getSettings(r.Context(), u.ID).Typing {
		h.emit(r.Context(), p, map[string]any{"type": "typing", "connectionId": r.PathValue("id")})
	}
	reply(w, 200, nil)
}
func (h *Handler) notifications(w http.ResponseWriter, r *http.Request) {
	u, ok := h.user(w, r)
	if !ok {
		return
	}
	rows, e := h.DB.Query(r.Context(), `SELECT n.id,n.connection_id::text,n.kind,n.created_at,n.read_at,p.display_name FROM notifications n JOIN connections c ON c.id=n.connection_id JOIN profiles p ON p.user_id=CASE WHEN c.user_low=$1 THEN c.user_high ELSE c.user_low END WHERE n.user_id=$1 AND c.ended_at IS NULL AND NOT EXISTS(SELECT 1 FROM blocks b WHERE (b.blocker_user_id=c.user_low AND b.blocked_user_id=c.user_high) OR (b.blocker_user_id=c.user_high AND b.blocked_user_id=c.user_low)) ORDER BY n.id DESC LIMIT 80`, u.ID)
	if e != nil {
		reply(w, 500, nil)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var connection, kind, name string
		var at time.Time
		var read *time.Time
		if rows.Scan(&id, &connection, &kind, &at, &read, &name) != nil {
			reply(w, 500, nil)
			return
		}
		out = append(out, map[string]any{"id": id, "connectionId": connection, "kind": kind, "createdAt": at, "read": read != nil, "name": name})
	}
	reply(w, 200, out)
}
func (h *Handler) readNotifications(w http.ResponseWriter, r *http.Request) {
	u, ok := h.user(w, r)
	if !ok {
		return
	}
	_, e := h.DB.Exec(r.Context(), "UPDATE notifications SET read_at=now() WHERE user_id=$1 AND read_at IS NULL", u.ID)
	if e != nil {
		reply(w, 500, nil)
		return
	}
	reply(w, 200, nil)
}
func (h *Handler) events(w http.ResponseWriter, r *http.Request) {
	u, ok := h.user(w, r)
	if !ok {
		return
	}
	if r.Header.Get("Origin") != h.Origin {
		reply(w, 403, nil)
		return
	}
	c, e := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{strings.TrimPrefix(strings.TrimPrefix(h.Origin, "https://"), "http://")}})
	if e != nil {
		return
	}
	defer c.CloseNow()
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	c.SetReadLimit(1024)
	go func() {
		for {
			if _, _, e := c.Read(ctx); e != nil {
				cancel()
				return
			}
		}
	}()
	sub := h.Redis.Subscribe(ctx, "communication:user:"+u.ID)
	defer sub.Close()
	if _, e = sub.Receive(ctx); e != nil {
		return
	}
	config, e := h.ICE.Configuration(u.ID, time.Now())
	if e != nil {
		return
	}
	write := func(v any) error {
		b, _ := json.Marshal(v)
		deadline, stop := context.WithTimeout(ctx, 5*time.Second)
		defer stop()
		return c.Write(deadline, websocket.MessageText, b)
	}
	if write(map[string]any{"type": "ready", "user": u, "ice": config, "settings": h.getSettings(ctx, u.ID)}) != nil {
		return
	}
	tick := time.NewTicker(20 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			if _, e := h.Authenticate(r); e != nil {
				return
			}
			if write(map[string]string{"type": "heartbeat"}) != nil {
				return
			}
		case event := <-sub.Channel():
			if event == nil {
				return
			}
			var value any
			if json.Unmarshal([]byte(event.Payload), &value) == nil && write(value) != nil {
				return
			}
		}
	}
}
