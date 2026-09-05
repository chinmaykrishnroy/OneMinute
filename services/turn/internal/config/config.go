package config

import (
	"errors"
	"net"
	"os"
	"strconv"
)

type Config struct {
	Environment, ListenAddr, HealthAddr, Realm, Secret string
	PublicIP                                           net.IP
	RelayMin, RelayMax                                 uint16
	AllowPrivate                                       bool
}

func Load() (Config, error) {
	c := Config{Environment: get("APP_ENV", "development"), ListenAddr: get("TURN_LISTEN_ADDR", ":3478"), HealthAddr: get("TURN_HEALTH_ADDR", ":8081"), Realm: get("TURN_REALM", "local.encounter"), Secret: os.Getenv("TURN_SECRET"), PublicIP: net.ParseIP(os.Getenv("TURN_PUBLIC_IP"))}
	if c.Environment != "development" && c.Environment != "test" && c.Environment != "production" {
		return c, errors.New("invalid APP_ENV")
	}
	if len(c.Secret) < 32 || c.Secret == "replace-with-at-least-32-random-bytes" {
		return c, errors.New("TURN_SECRET must contain at least 32 random characters")
	}
	if c.PublicIP == nil || c.PublicIP.To4() == nil || c.PublicIP.IsUnspecified() || c.PublicIP.IsMulticast() {
		return c, errors.New("TURN_PUBLIC_IP must be a concrete IPv4 address")
	}
	min, e1 := strconv.ParseUint(get("TURN_RELAY_MIN", "49160"), 10, 16)
	max, e2 := strconv.ParseUint(get("TURN_RELAY_MAX", "49180"), 10, 16)
	if e1 != nil || e2 != nil || min < 1024 || max < min || max == 65535 {
		return c, errors.New("invalid TURN relay range (1024..65534)")
	}
	c.RelayMin, c.RelayMax = uint16(min), uint16(max)
	var err error
	c.AllowPrivate, err = strconv.ParseBool(get("TURN_ALLOW_PRIVATE_PEERS", "false"))
	if err != nil {
		return c, errors.New("invalid TURN_ALLOW_PRIVATE_PEERS")
	}
	if c.Environment == "production" && (c.AllowPrivate || !PublicPeer(c.PublicIP)) {
		return c, errors.New("production TURN requires a public IP and private peers disabled")
	}
	return c, nil
}
func PublicPeer(ip net.IP) bool {
	// Includes shared-address space, reserved/documentation ranges and metadata endpoints.
	if ip == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	for _, block := range []string{"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "2001:db8::/32"} {
		_, subnet, _ := net.ParseCIDR(block)
		if subnet.Contains(ip) {
			return false
		}
	}
	return true
}
func get(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
