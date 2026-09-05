package transport

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthAndReadiness(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := Handler(log, map[string]Check{"postgres": func(context.Context) error { return errors.New("sensitive connection details") }})
	for _, tc := range []struct {
		path   string
		status int
	}{{"/healthz", 200}, {"/readyz", 503}} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", tc.path, nil))
		if w.Code != tc.status {
			t.Fatalf("%s: %d", tc.path, w.Code)
		}
		if strings.Contains(w.Body.String(), "sensitive") {
			t.Fatal("dependency error leaked")
		}
		if len(w.Header().Get("X-Request-ID")) != 32 {
			t.Fatal("missing request ID")
		}
	}
}
