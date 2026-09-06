// Package social owns durable profiles, safety records, and mutual connections.
package social

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"example.com/encounter/apps/server/internal/auth"
	"example.com/encounter/apps/server/internal/discovery"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var interests = map[string]bool{"ai": true, "art": true, "books": true, "films": true, "fitness": true, "gaming": true, "music": true, "nature": true, "photography": true, "science": true, "technology": true, "travel": true}
var reportCategories = map[string]bool{"spam": true, "harassment": true, "sexual_content": true, "hate": true, "violence": true, "underage": true, "other": true}
var languagePattern = regexp.MustCompile(`^[a-z]{2,3}(?:-[A-Z]{2})?$`)

type Profile struct {
	DiscoveryIntent string `json:"discoveryIntent,omitempty"`
	ID          string   `json:"id"`
	DisplayName string   `json:"displayName"`
	AvatarURL   string   `json:"avatarUrl"`
	Bio         string   `json:"bio"`
	CountryCode string   `json:"countryCode"`
	Interests   []string `json:"interests"`
	Languages   []string `json:"languages"`
}
type Connection struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Person    Profile   `json:"person"`
}
type Repository struct{ DB *pgxpool.Pool }

func (r Repository) Profile(ctx context.Context, userID string) (Profile, error) {
	var p Profile
	err := r.DB.QueryRow(ctx, `SELECT u.id::text,p.display_name,p.avatar_url,p.bio,COALESCE(p.country_code,''),
		COALESCE((SELECT array_agg(interest ORDER BY interest) FROM profile_interests WHERE user_id=u.id),'{}'),
		COALESCE((SELECT array_agg(language_code ORDER BY language_code) FROM profile_languages WHERE user_id=u.id),'{}')
		,p.discovery_intent FROM users u JOIN profiles p ON p.user_id=u.id WHERE u.id=$1 AND u.status='active'`, userID).Scan(&p.ID, &p.DisplayName, &p.AvatarURL, &p.Bio, &p.CountryCode, &p.Interests, &p.Languages, &p.DiscoveryIntent)
	return p, err
}

func (r Repository) UpdateProfile(ctx context.Context, userID string, p Profile) (Profile, error) {
	if p.DiscoveryIntent == "" { p.DiscoveryIntent = "surprise_me" }
	p.DisplayName = strings.TrimSpace(p.DisplayName)
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return Profile{}, err
	}
	defer tx.Rollback(ctx)
	command, err := tx.Exec(ctx, `UPDATE profiles SET display_name=$2,bio=$3,country_code=NULLIF($4,''),discovery_intent=$5,updated_at=now() WHERE user_id=$1`, userID, p.DisplayName, p.Bio, p.CountryCode, p.DiscoveryIntent)
	if err != nil || command.RowsAffected() != 1 {
		return Profile{}, errors.New("profile unavailable")
	}
	if _, err = tx.Exec(ctx, "DELETE FROM profile_interests WHERE user_id=$1", userID); err != nil {
		return Profile{}, err
	}
	for _, value := range p.Interests {
		if _, err = tx.Exec(ctx, "INSERT INTO profile_interests(user_id,interest) VALUES($1,$2)", userID, value); err != nil {
			return Profile{}, err
		}
	}
	if _, err = tx.Exec(ctx, "DELETE FROM profile_languages WHERE user_id=$1", userID); err != nil {
		return Profile{}, err
	}
	for _, value := range p.Languages {
		if _, err = tx.Exec(ctx, "INSERT INTO profile_languages(user_id,language_code) VALUES($1,$2)", userID, value); err != nil {
			return Profile{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Profile{}, err
	}
	return r.Profile(ctx, userID)
}

func (r Repository) Connections(ctx context.Context, userID string) ([]Connection, error) {
	rows, err := r.DB.Query(ctx, `SELECT c.id::text,c.created_at,u.id::text,p.display_name,p.avatar_url,p.bio,COALESCE(p.country_code,''),
		COALESCE((SELECT array_agg(interest ORDER BY interest) FROM profile_interests WHERE user_id=u.id),'{}'),COALESCE((SELECT array_agg(language_code ORDER BY language_code) FROM profile_languages WHERE user_id=u.id),'{}')
		FROM connections c JOIN users u ON u.id=CASE WHEN c.user_low=$1 THEN c.user_high ELSE c.user_low END JOIN profiles p ON p.user_id=u.id
		WHERE c.ended_at IS NULL AND ($1=c.user_low OR $1=c.user_high) AND u.status='active' AND NOT EXISTS(SELECT 1 FROM blocks b WHERE (b.blocker_user_id=$1 AND b.blocked_user_id=u.id) OR (b.blocker_user_id=u.id AND b.blocked_user_id=$1)) ORDER BY c.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Connection{}
	for rows.Next() {
		var c Connection
		if err := rows.Scan(&c.ID, &c.CreatedAt, &c.Person.ID, &c.Person.DisplayName, &c.Person.AvatarURL, &c.Person.Bio, &c.Person.CountryCode, &c.Person.Interests, &c.Person.Languages); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r Repository) ConnectionPeer(ctx context.Context, userID, connectionID string) (string, bool, error) {
	var peerID string
	err := r.DB.QueryRow(ctx, `SELECT CASE WHEN user_low=$1 THEN user_high::text ELSE user_low::text END FROM connections WHERE id=$2 AND ended_at IS NULL AND ($1=user_low OR $1=user_high)`, userID, connectionID).Scan(&peerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	return peerID, err == nil, err
}
func (r Repository) ActivePair(ctx context.Context, a, b string) (bool, error) {
	var ok bool
	err := r.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM connections WHERE ended_at IS NULL AND user_low=LEAST($1::uuid,$2::uuid) AND user_high=GREATEST($1::uuid,$2::uuid))`, a, b).Scan(&ok)
	return ok, err
}
func (r Repository) Remove(ctx context.Context, userID, connectionID string) (bool, error) {
	tag, err := r.DB.Exec(ctx, `UPDATE connections SET ended_at=now() WHERE id=$2 AND ended_at IS NULL AND ($1=user_low OR $1=user_high)`, userID, connectionID)
	return err == nil && tag.RowsAffected() == 1, err
}
func (r Repository) Block(ctx context.Context, userID, target string) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO blocks(blocker_user_id,blocked_user_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, userID, target); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE connections SET ended_at=COALESCE(ended_at,now()) WHERE user_low=LEAST($1::uuid,$2::uuid) AND user_high=GREATEST($1::uuid,$2::uuid)`, userID, target); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r Repository) Unblock(ctx context.Context, userID, target string) error {
	_, err := r.DB.Exec(ctx, "DELETE FROM blocks WHERE blocker_user_id=$1 AND blocked_user_id=$2", userID, target)
	return err
}
func (r Repository) Blocks(ctx context.Context, userID string) ([]Profile, error) {
	rows, err := r.DB.Query(ctx, `SELECT u.id::text,p.display_name,p.avatar_url,p.bio,COALESCE(p.country_code,''),'{}'::text[],'{}'::text[] FROM blocks b JOIN users u ON u.id=b.blocked_user_id JOIN profiles p ON p.user_id=u.id WHERE b.blocker_user_id=$1 ORDER BY b.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Profile{}
	for rows.Next() {
		var p Profile
		if err := rows.Scan(&p.ID, &p.DisplayName, &p.AvatarURL, &p.Bio, &p.CountryCode, &p.Interests, &p.Languages); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r Repository) ReportMatch(ctx context.Context, reporter, target, category, details string, match discovery.MatchState) error {
	users := []string{match.UserA, match.UserB}
	sort.Strings(users)
	shared := match.SharedInterests
	if shared == nil {
		shared = []string{}
	}
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `INSERT INTO encounters(id,user_low,user_high,started_at,ended_at,intent,shared_interests,outcome) VALUES($1,$2,$3,$4,now(),$5,$6,'reported') ON CONFLICT(id) DO NOTHING`, match.ID, users[0], users[1], time.UnixMilli(match.StartedAt), match.Intent, shared)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO reports(reporter_user_id,reported_user_id,encounter_id,category,details) VALUES($1,$2,$3,$4,$5) ON CONFLICT(reporter_user_id,encounter_id) WHERE encounter_id IS NOT NULL DO NOTHING`, reporter, target, match.ID, category, details)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r Repository) ReportConnection(ctx context.Context, reporter, target, connectionID, category, details string) error {
	_, err := r.DB.Exec(ctx, `INSERT INTO reports(reporter_user_id,reported_user_id,connection_id,category,details) VALUES($1,$2,$3,$4,$5)
		ON CONFLICT(reporter_user_id,connection_id) WHERE connection_id IS NOT NULL DO NOTHING`, reporter, target, connectionID, category, details)
	return err
}

type Handler struct {
	Repo         Repository
	Store        discovery.Store
	Authenticate func(*http.Request) (auth.User, error)
	Origin       string
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/profile", h.profile)
	mux.HandleFunc("PATCH /v1/profile", h.profile)
	mux.HandleFunc("GET /v1/connections", h.connections)
	mux.HandleFunc("DELETE /v1/connections/{id}", h.remove)
	mux.HandleFunc("GET /v1/blocks", h.blocks)
	mux.HandleFunc("POST /v1/blocks", h.block)
	mux.HandleFunc("DELETE /v1/blocks/{id}", h.unblock)
	mux.HandleFunc("POST /v1/reports", h.report)
}
func (h *Handler) user(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	u, err := h.Authenticate(r)
	if err != nil {
		respond(w, 401, map[string]string{"error": "authentication required"})
		return u, false
	}
	return u, true
}
func (h *Handler) mutation(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	u, ok := h.user(w, r)
	if !ok {
		return u, false
	}
	if r.Header.Get("Origin") != h.Origin {
		respond(w, 403, map[string]string{"error": "origin rejected"})
		return u, false
	}
	return u, true
}
func (h *Handler) profile(w http.ResponseWriter, r *http.Request) {
	u, ok := h.user(w, r)
	if !ok {
		return
	}
	if r.Method == "GET" {
		p, err := h.Repo.Profile(r.Context(), u.ID)
		if err != nil {
			respond(w, 500, map[string]string{"error": "profile unavailable"})
			return
		}
		respond(w, 200, p)
		return
	}
	if r.Header.Get("Origin") != h.Origin {
		respond(w, 403, map[string]string{"error": "origin rejected"})
		return
	}
	var p Profile
	if decode(w, r, &p) != nil || !validProfile(p) {
		respond(w, 400, map[string]string{"error": "invalid profile"})
		return
	}
	updated, err := h.Repo.UpdateProfile(r.Context(), u.ID, p)
	if err != nil {
		respond(w, 500, map[string]string{"error": "profile update failed"})
		return
	}
	respond(w, 200, updated)
}
func (h *Handler) connections(w http.ResponseWriter, r *http.Request) {
	u, ok := h.user(w, r)
	if !ok {
		return
	}
	items, err := h.Repo.Connections(r.Context(), u.ID)
	if err != nil {
		respond(w, 500, map[string]string{"error": "connections unavailable"})
		return
	}
	respond(w, 200, map[string]any{"connections": items})
}
func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	u, ok := h.mutation(w, r)
	if !ok {
		return
	}
	removed, err := h.Repo.Remove(r.Context(), u.ID, r.PathValue("id"))
	if err != nil {
		respond(w, 500, map[string]string{"error": "could not remove connection"})
		return
	}
	if !removed {
		respond(w, 404, map[string]string{"error": "connection not found"})
		return
	}
	w.WriteHeader(204)
}
func (h *Handler) blocks(w http.ResponseWriter, r *http.Request) {
	u, ok := h.user(w, r)
	if !ok {
		return
	}
	items, err := h.Repo.Blocks(r.Context(), u.ID)
	if err != nil {
		respond(w, 500, map[string]string{"error": "blocks unavailable"})
		return
	}
	respond(w, 200, map[string]any{"blocks": items})
}
func (h *Handler) block(w http.ResponseWriter, r *http.Request) {
	u, ok := h.mutation(w, r)
	if !ok {
		return
	}
	var body struct {
		TargetUserID string `json:"targetUserId"`
		MatchID      string `json:"matchId,omitempty"`
		ConnectionID string `json:"connectionId,omitempty"`
	}
	if decode(w, r, &body) != nil || body.TargetUserID == "" || body.TargetUserID == u.ID {
		respond(w, 400, map[string]string{"error": "invalid block"})
		return
	}
	authorized := false
	if body.MatchID != "" {
		m, exists, _ := h.Store.CurrentMatch(r.Context(), u.ID)
		authorized = exists && m.ID == body.MatchID && peer(m, u.ID) == body.TargetUserID
	}
	if !authorized && body.ConnectionID != "" {
		target, exists, _ := h.Repo.ConnectionPeer(r.Context(), u.ID, body.ConnectionID)
		authorized = exists && target == body.TargetUserID
	}
	if !authorized {
		respond(w, 403, map[string]string{"error": "block context rejected"})
		return
	}
	if err := h.Repo.Block(r.Context(), u.ID, body.TargetUserID); err != nil {
		respond(w, 500, map[string]string{"error": "block failed"})
		return
	}
	if body.MatchID != "" {
		_, _ = h.Store.EndForUser(r.Context(), u.ID, body.MatchID, "blocked")
	}
	w.WriteHeader(204)
}
func (h *Handler) unblock(w http.ResponseWriter, r *http.Request) {
	u, ok := h.mutation(w, r)
	if !ok {
		return
	}
	if err := h.Repo.Unblock(r.Context(), u.ID, r.PathValue("id")); err != nil {
		respond(w, 500, map[string]string{"error": "unblock failed"})
		return
	}
	w.WriteHeader(204)
}
func (h *Handler) report(w http.ResponseWriter, r *http.Request) {
	u, ok := h.mutation(w, r)
	if !ok {
		return
	}
	var body struct {
		TargetUserID string `json:"targetUserId"`
		MatchID      string `json:"matchId,omitempty"`
		ConnectionID string `json:"connectionId,omitempty"`
		Category     string `json:"category"`
		Details      string `json:"details"`
	}
	if decode(w, r, &body) != nil || !reportCategories[body.Category] || len([]rune(body.Details)) > 500 {
		respond(w, 400, map[string]string{"error": "invalid report"})
		return
	}
	var err error
	if body.MatchID != "" {
		m, exists, e := h.Store.CurrentMatch(r.Context(), u.ID)
		if e != nil || !exists || m.ID != body.MatchID || peer(m, u.ID) != body.TargetUserID {
			respond(w, 403, map[string]string{"error": "report context rejected"})
			return
		}
		err = h.Repo.ReportMatch(r.Context(), u.ID, body.TargetUserID, body.Category, body.Details, m)
	} else {
		target, exists, e := h.Repo.ConnectionPeer(r.Context(), u.ID, body.ConnectionID)
		if e != nil || !exists || target != body.TargetUserID {
			respond(w, 403, map[string]string{"error": "report context rejected"})
			return
		}
		err = h.Repo.ReportConnection(r.Context(), u.ID, target, body.ConnectionID, body.Category, body.Details)
	}
	if err != nil {
		respond(w, 500, map[string]string{"error": "report failed"})
		return
	}
	w.WriteHeader(204)
}

func peer(m discovery.MatchState, user string) string {
	if m.UserA == user {
		return m.UserB
	}
	return m.UserA
}
func validProfile(p Profile) bool {
	if !map[string]bool{"surprise_me":true,"new_friends":true,"dating":true,"gaming":true,"language_exchange":true,"tech_ideas":true,"professional_networking":true}[p.DiscoveryIntent] { return false }
	p.DisplayName = strings.TrimSpace(p.DisplayName)
	if len([]rune(p.DisplayName)) < 1 || len([]rune(p.DisplayName)) > 80 || len([]rune(p.Bio)) > 500 || (!country(p.CountryCode)) {
		return false
	}
	if len(p.Interests) > 12 || len(p.Languages) > 5 {
		return false
	}
	seen := map[string]bool{}
	for _, v := range p.Interests {
		if !interests[v] || seen["i:"+v] {
			return false
		}
		seen["i:"+v] = true
	}
	for _, v := range p.Languages {
		if !languagePattern.MatchString(v) || seen["l:"+v] {
			return false
		}
		seen["l:"+v] = true
	}
	return true
}
func country(v string) bool {
	if v == "" {
		return true
	}
	return len(v) == 2 && v[0] >= 'A' && v[0] <= 'Z' && v[1] >= 'A' && v[1] <= 'Z'
}
func decode(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON")
	}
	return nil
}
func respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
