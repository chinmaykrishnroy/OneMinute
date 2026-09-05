package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/api/idtoken"
)

const sessionLifetime = 30 * 24 * time.Hour

type Identity struct {
	Subject, Name, Picture string
	EmailVerified          bool
}

type Verifier interface {
	Verify(context.Context, string, string) (Identity, error)
}

type GoogleVerifier struct{ Audience string }

func (v GoogleVerifier) Verify(ctx context.Context, credential, nonce string) (Identity, error) {
	payload, err := idtoken.Validate(ctx, credential, v.Audience)
	if err != nil {
		return Identity{}, errors.New("invalid Google credential")
	}
	claimNonce, _ := payload.Claims["nonce"].(string)
	if nonce == "" || subtle.ConstantTimeCompare([]byte(claimNonce), []byte(nonce)) != 1 {
		return Identity{}, errors.New("invalid Google nonce")
	}
	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)
	emailVerified, _ := payload.Claims["email_verified"].(bool)
	if payload.Subject == "" {
		return Identity{}, errors.New("Google subject missing")
	}
	return Identity{Subject: payload.Subject, Name: name, Picture: picture, EmailVerified: emailVerified}, nil
}

type User struct {
	ID                  string `json:"id"`
	DisplayName         string `json:"displayName"`
	AvatarURL           string `json:"avatarUrl"`
	NewUser             bool   `json:"newUser"`
	GoogleEmailVerified bool   `json:"googleEmailVerified"`
}

type Repository struct{ DB *pgxpool.Pool }

func (r Repository) Login(ctx context.Context, identity Identity, secretHash []byte, expires time.Time) (User, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", "google:"+identity.Subject); err != nil {
		return User{}, err
	}
	var user User
	err = tx.QueryRow(ctx, `SELECT u.id::text,p.display_name,p.avatar_url,e.email_verified FROM external_identities e JOIN users u ON u.id=e.user_id JOIN profiles p ON p.user_id=u.id WHERE e.provider='google' AND e.provider_subject=$1`, identity.Subject).Scan(&user.ID, &user.DisplayName, &user.AvatarURL, &user.GoogleEmailVerified)
	if errors.Is(err, pgx.ErrNoRows) {
		user.NewUser = true
		name := strings.TrimSpace(identity.Name)
		if name == "" {
			name = "New person"
		}
		if len([]rune(name)) > 80 {
			name = string([]rune(name)[:80])
		}
		if err = tx.QueryRow(ctx, "INSERT INTO users DEFAULT VALUES RETURNING id::text").Scan(&user.ID); err != nil {
			return User{}, err
		}
		if _, err = tx.Exec(ctx, "INSERT INTO external_identities(provider,provider_subject,user_id,email_verified) VALUES ('google',$1,$2,$3)", identity.Subject, user.ID, identity.EmailVerified); err != nil {
			return User{}, err
		}
		if _, err = tx.Exec(ctx, "INSERT INTO profiles(user_id,display_name,avatar_url) VALUES ($1,$2,$3)", user.ID, name, identity.Picture); err != nil {
			return User{}, err
		}
		user.DisplayName, user.AvatarURL = name, identity.Picture
		user.GoogleEmailVerified = identity.EmailVerified
	} else if err != nil {
		return User{}, err
	}
	command, err := tx.Exec(ctx, "UPDATE users SET last_seen_at=now() WHERE id=$1 AND status='active'", user.ID)
	if err != nil || command.RowsAffected() != 1 {
		return User{}, errors.New("account unavailable")
	}
	if _, err = tx.Exec(ctx, "UPDATE external_identities SET last_login_at=now(),email_verified=$2 WHERE provider='google' AND provider_subject=$1", identity.Subject, identity.EmailVerified); err != nil {
		return User{}, err
	}
	user.GoogleEmailVerified = identity.EmailVerified
	if _, err = tx.Exec(ctx, "INSERT INTO sessions(user_id,secret_hash,expires_at) VALUES ($1,$2,$3)", user.ID, secretHash, expires); err != nil {
		return User{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return user, nil
}

func (r Repository) Current(ctx context.Context, secretHash []byte) (User, error) {
	var user User
	err := r.DB.QueryRow(ctx, `UPDATE sessions s SET last_seen_at=now() FROM users u,profiles p,external_identities e WHERE s.secret_hash=$1 AND s.revoked_at IS NULL AND s.expires_at>now() AND u.id=s.user_id AND u.status='active' AND p.user_id=u.id AND e.user_id=u.id AND e.provider='google' RETURNING u.id::text,p.display_name,p.avatar_url,e.email_verified`, secretHash).Scan(&user.ID, &user.DisplayName, &user.AvatarURL, &user.GoogleEmailVerified)
	return user, err
}

func (r Repository) Revoke(ctx context.Context, secretHash []byte) error {
	_, err := r.DB.Exec(ctx, "UPDATE sessions SET revoked_at=COALESCE(revoked_at,now()) WHERE secret_hash=$1", secretHash)
	return err
}

type Handler struct {
	Repo             Repository
	Verifier         Verifier
	Origin, ClientID string
	Secure           bool
	Now              func() time.Time
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/auth/config", h.config)
	mux.HandleFunc("POST /v1/auth/google", h.google)
	mux.HandleFunc("GET /v1/auth/me", h.me)
	mux.HandleFunc("POST /v1/auth/logout", h.logout)
}

func (h *Handler) config(w http.ResponseWriter, _ *http.Request) {
	nonce, err := randomToken(24)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not initialize sign-in")
		return
	}
	h.setCookie(w, h.nonceCookie(), nonce, 10*time.Minute)
	writeJSON(w, http.StatusOK, map[string]any{"enabled": h.ClientID != "", "clientId": h.ClientID, "nonce": nonce})
}

func (h *Handler) google(w http.ResponseWriter, r *http.Request) {
	if !h.validOrigin(r) {
		writeError(w, http.StatusForbidden, "origin rejected")
		return
	}
	if h.ClientID == "" || h.Verifier == nil {
		writeError(w, http.StatusServiceUnavailable, "Google sign-in is not configured")
		return
	}
	var body struct {
		Credential string `json:"credential"`
		Nonce      string `json:"nonce"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	cookie, err := r.Cookie(h.nonceCookie())
	if err != nil || len(body.Nonce) < 20 || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(body.Nonce)) != 1 {
		writeError(w, http.StatusForbidden, "login nonce rejected")
		return
	}
	identity, err := h.Verifier.Verify(r.Context(), body.Credential, body.Nonce)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Google sign-in failed")
		return
	}
	secret, err := randomToken(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}
	user, err := h.Repo.Login(r.Context(), identity, hash(secret), h.now().Add(sessionLifetime))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}
	h.setCookie(w, h.sessionCookie(), secret, sessionLifetime)
	h.clearCookie(w, h.nonceCookie())
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(h.sessionCookie())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "not signed in")
		return
	}
	user, err := h.Repo.Current(r.Context(), hash(cookie.Value))
	if err != nil {
		h.clearCookie(w, h.sessionCookie())
		writeError(w, http.StatusUnauthorized, "not signed in")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if !h.validOrigin(r) {
		writeError(w, http.StatusForbidden, "origin rejected")
		return
	}
	if cookie, err := r.Cookie(h.sessionCookie()); err == nil {
		_ = h.Repo.Revoke(r.Context(), hash(cookie.Value))
	}
	h.clearCookie(w, h.sessionCookie())
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) validOrigin(r *http.Request) bool { return r.Header.Get("Origin") == h.Origin }
func (h *Handler) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}
func (h *Handler) sessionCookie() string {
	if h.Secure {
		return "__Host-encounter_session"
	}
	return "encounter_session"
}
func (h *Handler) nonceCookie() string {
	if h.Secure {
		return "__Host-encounter_login_nonce"
	}
	return "encounter_login_nonce"
}
func (h *Handler) setCookie(w http.ResponseWriter, name, value string, lifetime time.Duration) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", MaxAge: int(lifetime.Seconds()), Expires: h.now().Add(lifetime), Secure: h.Secure, HttpOnly: true, SameSite: http.SameSiteLaxMode})
}
func (h *Handler) clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{Name: name, Path: "/", MaxAge: -1, Expires: time.Unix(1, 0), Secure: h.Secure, HttpOnly: true, SameSite: http.SameSiteLaxMode})
}
func randomToken(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func hash(value string) []byte { sum := sha256.Sum256([]byte(value)); return sum[:] }
func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return err
	}
	return ensureEOF(dec)
}
func ensureEOF(dec *json.Decoder) error {
	var extra any
	err := dec.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("multiple values")
	}
	return err
}
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
