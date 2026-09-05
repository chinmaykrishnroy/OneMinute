package config

import (
	"net"
	"testing"
)

func TestPrivatePeerPolicy(t *testing.T) {
	for _, s := range []string{"127.0.0.1", "10.1.2.3", "172.16.2.1", "192.168.1.1", "169.254.169.254", "100.100.100.200", "0.0.0.0", "224.1.1.1", "::1", "fc00::1", "2001:db8::1", "255.255.255.255"} {
		if PublicPeer(net.ParseIP(s)) {
			t.Errorf("accepted %s", s)
		}
	}
	if !PublicPeer(net.ParseIP("8.8.8.8")) {
		t.Fatal("public address rejected")
	}
}
func TestRejectUnsafeProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("TURN_SECRET", "01234567890123456789012345678901")
	t.Setenv("TURN_PUBLIC_IP", "127.0.0.1")
	if _, err := Load(); err == nil {
		t.Fatal("accepted local public IP in production")
	}
	t.Setenv("TURN_PUBLIC_IP", "8.8.8.8")
	t.Setenv("TURN_ALLOW_PRIVATE_PEERS", "true")
	if _, err := Load(); err == nil {
		t.Fatal("accepted private peers in production")
	}
}
