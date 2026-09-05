package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"example.com/encounter/internal/ice"
	"example.com/encounter/services/turn/internal/config"
	"example.com/encounter/services/turn/internal/relay"
	"github.com/pion/turn/v5"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, log); err != nil {
		log.Error("TURN stopped", "error", err)
		os.Exit(1)
	}
}
func run(ctx context.Context, log *slog.Logger) error {
	c, err := config.Load()
	if err != nil {
		return err
	}
	udp, err := net.ListenPacket("udp4", c.ListenAddr)
	if err != nil {
		return err
	}
	defer udp.Close()
	tcp, err := net.Listen("tcp4", c.ListenAddr)
	if err != nil {
		return err
	}
	defer tcp.Close()
	var accepted, rejected atomic.Uint64
	generator := func() *relay.Generator {
		return &relay.Generator{RelayAddressGeneratorPortRange: &turn.RelayAddressGeneratorPortRange{RelayAddress: c.PublicIP, Address: "0.0.0.0", MinPort: c.RelayMin, MaxPort: c.RelayMax, MaxRetries: 100}}
	}
	permission := func(_ net.Addr, peer net.IP) bool { return c.AllowPrivate || config.PublicPeer(peer) }
	server, err := turn.NewServer(turn.ServerConfig{
		Realm: c.Realm,
		AuthHandler: func(a *turn.RequestAttributes) (string, []byte, bool) {
			if a.Realm != c.Realm || !ice.ValidUsername(a.Username, time.Now()) {
				rejected.Add(1)
				return "", nil, false
			}
			accepted.Add(1)
			return a.Username, turn.GenerateAuthKey(a.Username, c.Realm, ice.Password(c.Secret, a.Username)), true
		},
		PacketConnConfigs: []turn.PacketConnConfig{{PacketConn: udp, RelayAddressGenerator: generator(), PermissionHandler: permission}},
		ListenerConfigs:   []turn.ListenerConfig{{Listener: tcp, RelayAddressGenerator: generator(), PermissionHandler: permission}},
	})
	if err != nil {
		return err
	}
	defer server.Close()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"status\":\"ok\"}"))
	})
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprintf(w, "turn_auth_callbacks_accepted_total %d\nturn_auth_callbacks_rejected_total %d\n", accepted.Load(), rejected.Load())
	})
	health := &http.Server{Addr: c.HealthAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second}
	failures := make(chan error, 1)
	go func() { failures <- health.ListenAndServe() }()
	log.Info("TURN listening", "address", c.ListenAddr, "public_ip", c.PublicIP.String(), "relay_min", c.RelayMin, "relay_max", c.RelayMax)
	select {
	case <-ctx.Done():
	case err := <-failures:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return health.Shutdown(shutdown)
}
