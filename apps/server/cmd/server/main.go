package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"example.com/encounter/apps/server/internal/auth"
	"example.com/encounter/apps/server/internal/communication"
	"example.com/encounter/apps/server/internal/config"
	"example.com/encounter/apps/server/internal/discovery"
	"example.com/encounter/apps/server/internal/signaling"
	"example.com/encounter/apps/server/internal/social"
	"example.com/encounter/apps/server/internal/transport"
	"example.com/encounter/internal/ice"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, log); err != nil {
		log.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return errors.New("invalid DATABASE_URL")
	}
	poolCfg.MaxConns = 10
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return errors.New("could not initialize PostgreSQL pool")
	}
	defer pool.Close()
	redisCfg, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return errors.New("invalid REDIS_URL")
	}
	redisCfg.PoolSize = 10
	cache := redis.NewClient(redisCfg)
	defer cache.Close()
	authHandler := &auth.Handler{Repo: auth.Repository{DB: pool}, Origin: cfg.WebOrigin, ClientID: cfg.GoogleClientID, Secure: strings.HasPrefix(cfg.WebOrigin, "https://")}
	if cfg.GoogleClientID != "" {
		authHandler.Verifier = auth.GoogleVerifier{Audience: cfg.GoogleClientID}
	}
	if len(os.Getenv("TURN_SECRET")) < 32 || os.Getenv("TURN_HOST") == "" {
		return errors.New("TURN_SECRET and TURN_HOST are required")
	}
	host := os.Getenv("TURN_HOST")
	iceProvider := ice.SharedSecretProvider{Secret: os.Getenv("TURN_SECRET"), URLs: []string{"stun:" + host, "turn:" + host + "?transport=udp", "turn:" + host + "?transport=tcp"}, TTL: 10 * time.Minute}
	discoveryHandler := &discovery.Handler{Root: ctx, Store: discovery.Store{Redis: cache}, Repo: discovery.Repository{DB: pool}, Authenticate: authHandler.Authenticate, Origin: cfg.WebOrigin, ICE: iceProvider}
	socialHandler := &social.Handler{Repo: social.Repository{DB: pool}, Store: discoveryHandler.Store, Authenticate: authHandler.Authenticate, Origin: cfg.WebOrigin}
	go discoveryHandler.Run(ctx)
	communicationHandler := &communication.Handler{DB: pool, Redis: cache, Authenticate: authHandler.Authenticate, Origin: cfg.WebOrigin, ICE: iceProvider}
	var register = []func(*http.ServeMux){authHandler.Register, discoveryHandler.Register, socialHandler.Register, communicationHandler.Register}
	if cfg.LabEnabled {
		lab := &signaling.Lab{Root: ctx, Redis: cache, Origin: cfg.WebOrigin, ICE: iceProvider}
		register = append(register, lab.Register)
	}
	handler := transport.Handler(log, map[string]transport.Check{
		"postgres": pool.Ping,
		"redis":    func(ctx context.Context) error { return cache.Ping(ctx).Err() },
		"schema": func(ctx context.Context) error {
			var exists bool
			err := pool.QueryRow(ctx, "SELECT to_regclass('public.profiles') IS NOT NULL AND to_regclass('public.blocks') IS NOT NULL AND to_regclass('public.connections') IS NOT NULL AND to_regclass('public.messages') IS NOT NULL AND to_regclass('public.user_settings') IS NOT NULL AND EXISTS (SELECT 1 FROM pg_extension WHERE extname='vector')").Scan(&exists)
			if err != nil {
				return err
			}
			if !exists {
				return errors.New("migration required")
			}
			return nil
		},
	}, register...)
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	failures := make(chan error, 1)
	go func() { log.Info("server listening", "address", cfg.HTTPAddr); failures <- server.ListenAndServe() }()
	select {
	case err := <-failures:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}
