package transport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type Check func(context.Context) error

func Handler(log *slog.Logger, checks map[string]Check) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		respond(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		status := http.StatusOK
		result := map[string]string{}
		for name, check := range checks {
			if err := check(ctx); err != nil {
				result[name] = "unavailable"
				status = http.StatusServiceUnavailable
			} else {
				result[name] = "ok"
			}
		}
		respond(w, status, result)
	})
	return Observe(log, mux)
}

func Observe(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := make([]byte, 16)
		_, _ = rand.Read(id)
		requestID := hex.EncodeToString(id)
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		start := time.Now()
		next.ServeHTTP(w, r)
		// Never log query strings, cookies, credentials, or request bodies.
		log.Info("http request", "request_id", requestID, "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(start).Milliseconds())
	})
}

func respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
