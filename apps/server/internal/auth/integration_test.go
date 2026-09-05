//go:build integration

package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"example.com/encounter/apps/server/internal/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

type verifier struct{ identity auth.Identity }

func (v verifier) Verify(_ context.Context, credential, nonce string) (auth.Identity, error) {
	if credential != "signed-test-token" || nonce == "" {
		return auth.Identity{}, context.Canceled
	}
	return v.identity, nil
}

func TestSessionLifecycle(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("TEST_DATABASE_URL required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	subject := "integration-" + time.Now().UTC().Format("20060102150405.000000000")
	defer pool.Exec(context.Background(), "DELETE FROM users WHERE id=(SELECT user_id FROM external_identities WHERE provider='google' AND provider_subject=$1)", subject)
	h := &auth.Handler{Repo: auth.Repository{DB: pool}, Verifier: verifier{auth.Identity{Subject: subject, Name: "Integration Person", Picture: "https://example.com/avatar.png", EmailVerified: true}}, Origin: "https://example.test", ClientID: "client.apps.googleusercontent.com", Secure: true}
	mux := http.NewServeMux()
	h.Register(mux)

	config := serve(mux, "GET", "/v1/auth/config", "", nil, "")
	if config.Code != http.StatusOK {
		t.Fatalf("config: %d", config.Code)
	}
	var cfg struct {
		Nonce string `json:"nonce"`
	}
	if err := json.Unmarshal(config.Body.Bytes(), &cfg); err != nil {
		t.Fatal(err)
	}
	nonceCookie := findCookie(t, config, "__Host-encounter_login_nonce")
	if !nonceCookie.HttpOnly || !nonceCookie.Secure || nonceCookie.SameSite != http.SameSiteLaxMode {
		t.Fatal("nonce cookie flags missing")
	}
	body := `{"credential":"signed-test-token","nonce":"` + cfg.Nonce + `"}`
	wrong := serve(mux, "POST", "/v1/auth/google", body, nonceCookie, "https://hostile.test")
	if wrong.Code != http.StatusForbidden {
		t.Fatalf("hostile origin: %d", wrong.Code)
	}
	login := serve(mux, "POST", "/v1/auth/google", body, nonceCookie, h.Origin)
	if login.Code != http.StatusOK {
		t.Fatalf("login: %d %s", login.Code, login.Body.String())
	}
	var first auth.User
	if err := json.Unmarshal(login.Body.Bytes(), &first); err != nil {
		t.Fatal(err)
	}
	if !first.NewUser || !first.GoogleEmailVerified || first.ID == "" {
		t.Fatalf("unexpected user: %+v", first)
	}
	session := findCookie(t, login, "__Host-encounter_session")
	if !session.HttpOnly || !session.Secure || session.SameSite != http.SameSiteLaxMode || len(session.Value) < 40 {
		t.Fatal("session cookie is not hardened")
	}
	var stored string
	if err := pool.QueryRow(ctx, "SELECT encode(secret_hash,'hex') FROM sessions WHERE user_id=$1", first.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored, session.Value) {
		t.Fatal("raw session secret stored")
	}
	me := serve(mux, "GET", "/v1/auth/me", "", session, "")
	if me.Code != http.StatusOK {
		t.Fatalf("me: %d", me.Code)
	}
	logout := serve(mux, "POST", "/v1/auth/logout", "", session, h.Origin)
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout: %d", logout.Code)
	}
	if after := serve(mux, "GET", "/v1/auth/me", "", session, ""); after.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session accepted: %d", after.Code)
	}

	config2 := serve(mux, "GET", "/v1/auth/config", "", nil, "")
	_ = json.Unmarshal(config2.Body.Bytes(), &cfg)
	login2 := serve(mux, "POST", "/v1/auth/google", `{"credential":"signed-test-token","nonce":"`+cfg.Nonce+`"}`, findCookie(t, config2, "__Host-encounter_login_nonce"), h.Origin)
	var returning auth.User
	_ = json.Unmarshal(login2.Body.Bytes(), &returning)
	if login2.Code != http.StatusOK || returning.NewUser || returning.ID != first.ID {
		t.Fatalf("returning identity changed: %d %+v", login2.Code, returning)
	}
}

func serve(handler http.Handler, method, path, body string, cookie *http.Cookie, origin string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		r.AddCookie(cookie)
	}
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

func findCookie(t *testing.T, response *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %s missing", name)
	return nil
}
