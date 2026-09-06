package communication

import (
	"context"
	"github.com/google/uuid"
	"net/http"
	"strings"
	"time"
)

type Moment struct {
	ID           string    `json:"id"`
	UserID       string    `json:"userId"`
	Name         string    `json:"name"`
	Avatar       string    `json:"avatarUrl"`
	Body         string    `json:"body"`
	Tone         string    `json:"tone"`
	CreatedAt    time.Time `json:"createdAt"`
	ExpiresAt    time.Time `json:"expiresAt"`
	ConnectionID string    `json:"connectionId"`
}

func (h *Handler) moments(w http.ResponseWriter, r *http.Request) {
	u, ok := h.user(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	if r.Method == "GET" {
		rows, err := h.DB.Query(ctx, `SELECT m.id::text,m.user_id::text,p.display_name,p.avatar_url,m.body,m.tone,m.created_at,m.expires_at,COALESCE(audience.connection_id,'')
 FROM moments m JOIN profiles p ON p.user_id=m.user_id JOIN users author ON author.id=m.user_id
 LEFT JOIN LATERAL(SELECT c.id::text AS connection_id FROM moment_audience a JOIN connections c ON c.id=a.connection_id
 WHERE a.moment_id=m.id AND c.ended_at IS NULL AND ($1=c.user_low OR $1=c.user_high)
 AND NOT EXISTS(SELECT 1 FROM blocks b WHERE (b.blocker_user_id=c.user_low AND b.blocked_user_id=c.user_high) OR (b.blocker_user_id=c.user_high AND b.blocked_user_id=c.user_low)) LIMIT 1) audience ON true
 WHERE m.expires_at>now() AND author.status='active' AND (m.user_id=$1 OR audience.connection_id IS NOT NULL)
 ORDER BY m.created_at DESC LIMIT 100`, u.ID)
		if err != nil {
			reply(w, 500, nil)
			return
		}
		defer rows.Close()
		items := []Moment{}
		for rows.Next() {
			var m Moment
			if rows.Scan(&m.ID, &m.UserID, &m.Name, &m.Avatar, &m.Body, &m.Tone, &m.CreatedAt, &m.ExpiresAt, &m.ConnectionID) != nil {
				reply(w, 500, nil)
				return
			}
			items = append(items, m)
		}
		if rows.Err() != nil {
			reply(w, 500, nil)
			return
		}
		reply(w, 200, items)
		return
	}
	if !h.allow(w, r, u.ID, "moments", 6) {
		return
	}
	var body struct {
		Body string `json:"body"`
		Tone string `json:"tone"`
	}
	if decode(w, r, &body) != nil {
		reply(w, 400, nil)
		return
	}
	body.Body = strings.TrimSpace(body.Body)
	if len([]rune(body.Body)) < 1 || len([]rune(body.Body)) > 600 || (body.Tone != "mint" && body.Tone != "lilac" && body.Tone != "sand") {
		reply(w, 400, nil)
		return
	}
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		reply(w, 500, nil)
		return
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT id FROM users WHERE id=$1 FOR UPDATE", u.ID); err != nil {
		reply(w, 500, nil)
		return
	}
	var count int
	if tx.QueryRow(ctx, "SELECT count(*) FROM moments WHERE user_id=$1 AND expires_at>now()", u.ID).Scan(&count) != nil {
		reply(w, 500, nil)
		return
	}
	if count >= 3 {
		reply(w, 409, map[string]string{"error": "You can share up to three active moments. Remove one or wait for it to expire."})
		return
	}
	var id string
	if tx.QueryRow(ctx, "INSERT INTO moments(user_id,body,tone) VALUES($1,$2,$3) RETURNING id::text", u.ID, body.Body, body.Tone).Scan(&id) != nil {
		reply(w, 500, nil)
		return
	}
	// Snapshot the audience. New connections cannot see an earlier moment;
	// removal, blocking or a new connection ID revokes the original access.
	_, err = tx.Exec(ctx, `INSERT INTO moment_audience(moment_id,connection_id) SELECT $1,c.id FROM connections c WHERE ($2=c.user_low OR $2=c.user_high) AND c.ended_at IS NULL
 AND NOT EXISTS(SELECT 1 FROM blocks b WHERE (b.blocker_user_id=c.user_low AND b.blocked_user_id=c.user_high) OR (b.blocker_user_id=c.user_high AND b.blocked_user_id=c.user_low))`, id, u.ID)
	if err != nil || tx.Commit(ctx) != nil {
		reply(w, 500, nil)
		return
	}
	reply(w, 201, map[string]string{"id": id})
}

func (h *Handler) deleteMoment(w http.ResponseWriter, r *http.Request) {
	u, ok := h.user(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		reply(w, 404, nil)
		return
	}
	result, err := h.DB.Exec(r.Context(), "DELETE FROM moments WHERE id=$1 AND user_id=$2", id, u.ID)
	if err != nil {
		reply(w, 500, nil)
		return
	}
	if result.RowsAffected() == 0 {
		reply(w, 404, nil)
		return
	}
	reply(w, 200, map[string]bool{"ok": true})
}

// Query-time expiry is authoritative even if cleanup is delayed or restarted.
func (h *Handler) RunMomentCleanup(ctx context.Context) {
	timer := time.NewTicker(time.Minute)
	defer timer.Stop()
	for {
		if _, err := h.DB.Exec(ctx, "DELETE FROM moments WHERE expires_at<=now()"); err != nil && ctx.Err() != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
	}
}
