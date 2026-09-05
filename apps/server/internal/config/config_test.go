package config

import "testing"

func TestRequiredDependencies(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("missing dependencies accepted")
	}
}
func TestOriginValidation(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	for _, origin := range []string{"javascript:alert(1)", "https://example.com/path", "https://user:password@example.com", "https://example.com?x=1"} {
		t.Setenv("WEB_ORIGIN", origin)
		if _, err := Load(); err == nil {
			t.Errorf("accepted %s", origin)
		}
	}
	t.Setenv("APP_ENV", "production")
	t.Setenv("WEB_ORIGIN", "http://example.com")
	if _, err := Load(); err == nil {
		t.Fatal("accepted HTTP in production")
	}
}

func TestLabForbiddenInProduction(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("WEB_ORIGIN", "https://example.com")
	t.Setenv("APP_ENV", "production")
	t.Setenv("RTC_LAB_ENABLED", "true")
	if _, err := Load(); err == nil {
		t.Fatal("production lab accepted")
	}
}
