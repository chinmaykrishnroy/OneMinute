package relay

import (
	"github.com/pion/turn/v5"
	"testing"
)

func TestRejectOutOfRangeRequestedPort(t *testing.T) {
	g := &Generator{&turn.RelayAddressGeneratorPortRange{MinPort: 49160, MaxPort: 49180}}
	for _, port := range []int{1, 3478, 49159, 49181, 65535} {
		if _, _, err := g.AllocatePacketConn(turn.AllocateListenerConfig{RequestedPort: port}); err == nil {
			t.Fatalf("accepted %d", port)
		}
	}
	if _, _, err := g.AllocateListener(turn.AllocateListenerConfig{}); err == nil {
		t.Fatal("TCP relay unexpectedly enabled")
	}
}
