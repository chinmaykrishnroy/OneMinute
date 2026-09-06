//go:build integration

package social_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"example.com/encounter/apps/server/internal/auth"
	"example.com/encounter/apps/server/internal/discovery"
	"example.com/encounter/apps/server/internal/social"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestProfileConnectionReportAndBlockLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db, err := pgxpool.New(ctx, os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ids := make([]string, 2)
	for index, name := range []string{"Social One", "Social Two"} {
		if err := db.QueryRow(ctx, "INSERT INTO users DEFAULT VALUES RETURNING id::text").Scan(&ids[index]); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(ctx, "INSERT INTO profiles(user_id,display_name) VALUES($1,$2)", ids[index], name); err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		_, _ = db.Exec(context.Background(), "DELETE FROM reports WHERE reporter_user_id=ANY($1::uuid[]) OR reported_user_id=ANY($1::uuid[])", ids)
		_, _ = db.Exec(context.Background(), "DELETE FROM connections WHERE user_low=ANY($1::uuid[]) OR user_high=ANY($1::uuid[])", ids)
		_, _ = db.Exec(context.Background(), "DELETE FROM encounters WHERE user_low=ANY($1::uuid[]) OR user_high=ANY($1::uuid[])", ids)
		_, _ = db.Exec(context.Background(), "DELETE FROM users WHERE id=ANY($1::uuid[])", ids)
	}()
	repo := social.Repository{DB: db}
	updated, err := repo.UpdateProfile(ctx, ids[0], social.Profile{DisplayName: "Social One", Bio: "Here to learn.", CountryCode: "IN", Interests: []string{"ai", "music"}, Languages: []string{"en", "hi"}})
	if err != nil || updated.Bio != "Here to learn." || len(updated.Interests) != 2 || len(updated.Languages) != 2 {
		t.Fatalf("profile update: %+v err=%v", updated, err)
	}
	matchID := testUUID()
	connectionID, err := (discovery.Repository{DB: db}).PersistConnection(ctx, discovery.MatchState{ID: matchID, UserA: ids[0], UserB: ids[1], Intent: "new_friends", StartedAt: time.Now().Add(-time.Minute).UnixMilli()})
	if err != nil {
		t.Fatal(err)
	}
	connections, err := repo.Connections(ctx, ids[1])
	if err != nil || len(connections) != 1 || connections[0].Person.Bio != "Here to learn." {
		t.Fatalf("connections: %+v err=%v", connections, err)
	}
	handler := &social.Handler{Repo: repo, Authenticate: func(r *http.Request) (auth.User, error) {
		if id := r.Header.Get("X-Test-User"); id != "" {
			return auth.User{ID: id}, nil
		}
		return auth.User{}, errors.New("not authenticated")
	}, Origin: "https://example.test"}
	mux := http.NewServeMux()
	handler.Register(mux)
	if got := request(mux, "GET", "/v1/profile", "", "", "").Code; got != http.StatusUnauthorized {
		t.Fatalf("anonymous profile: %d", got)
	}
	if got := request(mux, "PATCH", "/v1/profile", `{}`, ids[0], "https://hostile.test").Code; got != http.StatusForbidden {
		t.Fatalf("hostile profile update: %d", got)
	}
	bogusConnection := "20000000-0000-4000-8000-000000000000"
	if got := request(mux, "POST", "/v1/blocks", `{"targetUserId":"`+ids[1]+`","connectionId":"`+bogusConnection+`"}`, ids[0], "https://example.test").Code; got != http.StatusForbidden {
		t.Fatalf("unrelated block: %d", got)
	}
	reportBody := `{"targetUserId":"` + ids[0] + `","connectionId":"` + connectionID + `","category":"spam","details":"test report"}`
	if response := request(mux, "POST", "/v1/reports", reportBody, ids[1], "https://example.test"); response.Code != http.StatusNoContent {
		t.Fatalf("connection report: %d %s", response.Code, response.Body.String())
	}
	if err := repo.Block(ctx, ids[1], ids[0]); err != nil {
		t.Fatal(err)
	}
	if active, err := repo.ActivePair(ctx, ids[0], ids[1]); err != nil || active {
		t.Fatalf("block did not end connection: active=%v err=%v", active, err)
	}
	blocks, err := repo.Blocks(ctx, ids[1])
	if err != nil || len(blocks) != 1 || blocks[0].ID != ids[0] {
		t.Fatalf("blocks: %+v err=%v", blocks, err)
	}
	if err := repo.Unblock(ctx, ids[1], ids[0]); err != nil {
		t.Fatal(err)
	}
}

func testUUID() string { return "10000000-0000-4000-8000-" + time.Now().UTC().Format("060102150405") }

func request(handler http.Handler, method, path, body, userID, origin string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	if userID != "" {
		r.Header.Set("X-Test-User", userID)
	}
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}
