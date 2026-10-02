package config

import (
	"strings"
	"testing"
)

func validEnvironment() map[string]string {
	return map[string]string{
		"HTTP_ADDR": ":8080", "HTTP_READ_TIMEOUT": "10s",
		"HTTP_READ_HEADER_TIMEOUT": "5s", "HTTP_WRITE_TIMEOUT": "15s",
		"HTTP_IDLE_TIMEOUT": "60s", "LOG_LEVEL": "info",
		"SHUTDOWN_TIMEOUT": "10s", "DATABASE_URL": "postgres://local/test",
		"DATABASE_MAX_CONNS": "10", "DATABASE_MIN_CONNS": "2",
		"DATABASE_MAX_CONN_LIFETIME": "30m", "DATABASE_CONNECT_TIMEOUT": "5s",
		"DATABASE_QUERY_TIMEOUT": "3s",
	}
}

func TestConfigRejectsMissingDatabaseURL(t *testing.T) {
	env := validEnvironment()
	delete(env, "DATABASE_URL")
	_, err := load(func(key string) string { return env[key] })
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("expected a useful DATABASE_URL error; got %v", err)
	}
}

func TestConfigRejectsImpossiblePoolSize(t *testing.T) {
	env := validEnvironment()
	env["DATABASE_MIN_CONNS"] = "20"
	_, err := load(func(key string) string { return env[key] })
	if err == nil || !strings.Contains(err.Error(), "DATABASE_MIN_CONNS") {
		t.Fatalf("expected pool-size validation; got %v", err)
	}
}

func TestConfigAcceptsValidEnvironment(t *testing.T) {
	env := validEnvironment()
	c, err := load(func(key string) string { return env[key] })
	if err != nil || c.DatabaseMaxConns != 10 || c.DatabaseQueryTime.Seconds() != 3 {
		t.Fatalf("unexpected config: %+v, error: %v", c, err)
	}
}
