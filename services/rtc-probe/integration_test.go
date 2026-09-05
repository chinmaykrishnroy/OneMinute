//go:build integration

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

type probeLog struct {
	mu                                sync.Mutex
	channels, media, relays, messages int
	entries                           []string
}

func (p *probeLog) Write(data []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.entries) < 200 && !strings.Contains(string(data), "join this room") {
		p.entries = append(p.entries, string(data))
	}
	if strings.Contains(string(data), `"msg":"DataChannel open"`) {
		p.channels++
	}
	if strings.Contains(string(data), `"msg":"DataChannel message received"`) {
		p.messages++
	}
	if strings.Contains(string(data), `"msg":"receiving media"`) && strings.Contains(string(data), `"kind":"audio"`) {
		p.media++
	}
	if strings.Contains(string(data), `"local_type":"relay"`) && strings.Contains(string(data), `"remote_type":"relay"`) {
		p.relays++
	}
	return len(data), nil
}
func (p *probeLog) complete(relay bool) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.channels >= 2 && p.messages >= 2 && p.media >= 2 && (!relay || p.relays >= 2)
}
func TestPionEndToEnd(t *testing.T) {
	api := os.Getenv("TEST_API_URL")
	if api == "" {
		t.Fatal("TEST_API_URL required")
	}
	const origin = "http://localhost:3000"
	for _, relay := range []bool{false, true} {
		t.Run(fmt.Sprintf("relay_%t", relay), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			req, _ := http.NewRequestWithContext(ctx, "POST", api+"/v1/lab/rooms", nil)
			req.Header.Set("Origin", origin)
			response, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != 201 {
				t.Fatalf("room creation returned %d", response.StatusCode)
			}
			var body struct {
				RoomID string `json:"roomId"`
			}
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			observed := &probeLog{}
			defer func() {
				if t.Failed() {
					observed.mu.Lock()
					defer observed.mu.Unlock()
					t.Logf("events: %s", strings.Join(observed.entries, ""))
				}
			}()
			log := slog.New(slog.NewJSONHandler(observed, nil))
			failures := make(chan error, 2)
			for range 2 {
				go func() { failures <- run(ctx, log, api, origin, body.RoomID, relay, true) }()
			}
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case err := <-failures:
					t.Fatalf("probe exited before transport verification: %v", err)
				case <-ctx.Done():
					t.Fatal("timed out waiting for bidirectional RTP, DataChannels and selected relay pair")
				case <-ticker.C:
					if observed.complete(relay) {
						cancel()
						// Both probe goroutines must finish, ensuring peer/socket cleanup.
						for range 2 {
							select {
							case <-failures:
							case <-time.After(15 * time.Second):
								t.Fatal("probe cleanup stalled")
							}
						}
						t.Log("bidirectional synthetic audio RTP and DataChannels established through real Go signaling")
						return
					}
				}
			}
		})
	}
}
