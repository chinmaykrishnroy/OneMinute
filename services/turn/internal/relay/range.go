// Package relay enforces the configured port range even for requested-port allocations.
package relay

import (
	"errors"
	"net"

	"github.com/pion/turn/v5"
)

type Generator struct {
	*turn.RelayAddressGeneratorPortRange
}

func (g *Generator) AllocatePacketConn(c turn.AllocateListenerConfig) (net.PacketConn, net.Addr, error) {
	if c.RequestedPort != 0 && (c.RequestedPort < int(g.MinPort) || c.RequestedPort > int(g.MaxPort)) {
		return nil, nil, errors.New("requested relay port is outside configured range")
	}
	return g.RelayAddressGeneratorPortRange.AllocatePacketConn(c)
}
func (g *Generator) AllocateListener(c turn.AllocateListenerConfig) (net.Listener, net.Addr, error) {
	// Browser TURN/TCP uses UDP relay allocations. RFC6062 TCP relays are out of scope.
	return nil, nil, errors.New("TCP relay allocations are disabled")
}
func (g *Generator) AllocateConn(c turn.AllocateConnConfig) (net.Conn, error) {
	return nil, errors.New("TCP relay connections are disabled")
}
