package ice

import (
	"testing"
	"time"
)

func TestCredentialExpiry(t *testing.T) {
	now := time.Unix(1800000000, 0)
	p := SharedSecretProvider{Secret: "01234567890123456789012345678901", URLs: []string{"turn:localhost:3478?transport=udp"}, TTL: 10 * time.Minute}
	c, err := p.Configuration("test-subject", now)
	if err != nil {
		t.Fatal(err)
	}
	s := c.ICEServers[0]
	if !ValidUsername(s.Username, now) || ValidUsername(s.Username, c.ExpiresAt) {
		t.Fatal("credentials must expire at the exact expiry boundary")
	}
	if s.Credential != Password(p.Secret, s.Username) {
		t.Fatal("password mismatch")
	}
	for _, invalid := range []string{"", "not-a-time:user", "1800000060", "1800000060:", "1800007200:user"} {
		if ValidUsername(invalid, now) {
			t.Errorf("accepted %q", invalid)
		}
	}
}
func TestCredentialConfiguration(t *testing.T) {
	for _, p := range []SharedSecretProvider{{Secret: "short", TTL: time.Minute}, {Secret: "01234567890123456789012345678901", TTL: 2 * time.Hour}} {
		if _, err := p.Configuration("user", time.Now()); err == nil {
			t.Fatal("accepted invalid provider")
		}
	}
}
