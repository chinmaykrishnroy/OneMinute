//go:build integration

package turn_test

import (
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"example.com/encounter/internal/ice"
	"github.com/pion/turn/v5"
)

func TestTURNTransports(t *testing.T) {
	address, secret, realm := os.Getenv("TEST_TURN_ADDR"), os.Getenv("TURN_SECRET"), os.Getenv("TURN_REALM")
	if address == "" || secret == "" || realm == "" {
		t.Fatal("TEST_TURN_ADDR, TURN_SECRET, TURN_REALM required")
	}
	for _, transport := range []string{"udp", "tcp"} {
		t.Run(transport, func(t *testing.T) {
			var conn net.PacketConn
			if transport == "udp" {
				var err error
				conn, err = net.ListenPacket("udp4", "0.0.0.0:0")
				if err != nil {
					t.Fatal(err)
				}
			} else {
				raw, err := net.DialTimeout("tcp", address, 5*time.Second)
				if err != nil {
					t.Fatal(err)
				}
				conn = turn.NewSTUNConn(raw)
			}
			defer conn.Close()
			username := strconv.FormatInt(time.Now().Add(5*time.Minute).Unix(), 10) + ":integration"
			client, err := turn.NewClient(&turn.ClientConfig{STUNServerAddr: address, TURNServerAddr: address, Conn: conn, Username: username, Password: ice.Password(secret, username), Realm: realm})
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			if err := client.Listen(); err != nil {
				t.Fatal(err)
			}
			mapped, err := client.SendBindingRequest()
			if err != nil {
				t.Fatal(err)
			}
			relay, err := client.Allocate()
			if err != nil {
				t.Fatal(err)
			}
			defer relay.Close()
			_, portString, err := net.SplitHostPort(relay.LocalAddr().String())
			if err != nil {
				t.Fatal(err)
			}
			port, _ := strconv.Atoi(portString)
			min, _ := strconv.Atoi(os.Getenv("TURN_RELAY_MIN"))
			max, _ := strconv.Atoi(os.Getenv("TURN_RELAY_MAX"))
			if port < min || port > max {
				t.Fatalf("relay outside range: %d", port)
			}
			t.Logf("STUN binding and authenticated TURN allocation passed; mapped=%s relay=%s", mapped, relay.LocalAddr())
		})
	}
}
