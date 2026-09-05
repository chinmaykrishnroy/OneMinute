package config

import (
	"errors"
	"net/url"
	"os"
	"strconv"
)

type Config struct {
	LabEnabled  bool
	Environment string
	HTTPAddr    string
	DatabaseURL string
	RedisURL    string
	WebOrigin   string
}

func Load() (Config, error) {
	c := Config{Environment: value("APP_ENV", "development"), HTTPAddr: value("HTTP_ADDR", ":8080"), DatabaseURL: os.Getenv("DATABASE_URL"), RedisURL: os.Getenv("REDIS_URL"), WebOrigin: value("WEB_ORIGIN", "http://localhost:3000")}
	if c.Environment != "development" && c.Environment != "test" && c.Environment != "production" {
		return c, errors.New("APP_ENV must be development, test, or production")
	}
	if c.DatabaseURL == "" || c.RedisURL == "" {
		return c, errors.New("DATABASE_URL and REDIS_URL are required")
	}
	u, err := url.Parse(c.WebOrigin)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return c, errors.New("WEB_ORIGIN must be an HTTP(S) origin without a path")
	}
	if c.Environment == "production" && u.Scheme != "https" {
		return c, errors.New("production requires an HTTPS WEB_ORIGIN")
	}
	enabled, err := strconv.ParseBool(value("RTC_LAB_ENABLED", "false"))
	if err != nil {
		return c, errors.New("invalid RTC_LAB_ENABLED")
	}
	c.LabEnabled = enabled
	if c.LabEnabled && c.Environment == "production" {
		return c, errors.New("networking lab is forbidden in production")
	}
	return c, nil
}

func value(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
