package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr              string
	HTTPReadTimeout       time.Duration
	HTTPReadHeaderTimeout time.Duration
	HTTPWriteTimeout      time.Duration
	HTTPIdleTimeout       time.Duration
	LogLevel              slog.Level
	ShutdownTimeout       time.Duration
	DatabaseURL           string
	DatabaseMaxConns      int32
	DatabaseMinConns      int32
	DatabaseMaxLifetime   time.Duration
	DatabaseConnectTime   time.Duration
	DatabaseQueryTime     time.Duration
}


func Load() (Config, error) {
	return load(os.Getenv)
}

func load(getenv func(string) string) (Config, error) {
	var c Config
	var err error
	if c.HTTPAddr, err = required(getenv, "HTTP_ADDR"); err != nil {
		return c, err
	}
	for _, field := range []struct {
		name string
		out  *time.Duration
	}{
		{"HTTP_READ_TIMEOUT", &c.HTTPReadTimeout},
		{"HTTP_READ_HEADER_TIMEOUT", &c.HTTPReadHeaderTimeout},
		{"HTTP_WRITE_TIMEOUT", &c.HTTPWriteTimeout},
		{"HTTP_IDLE_TIMEOUT", &c.HTTPIdleTimeout},
		{"SHUTDOWN_TIMEOUT", &c.ShutdownTimeout},
		{"DATABASE_MAX_CONN_LIFETIME", &c.DatabaseMaxLifetime},
		{"DATABASE_CONNECT_TIMEOUT", &c.DatabaseConnectTime},
		{"DATABASE_QUERY_TIMEOUT", &c.DatabaseQueryTime},
	} {
		if *field.out, err = duration(getenv, field.name); err != nil {
			return c, err
		}
	}
	level, err := required(getenv, "LOG_LEVEL")
	if err != nil {
		return c, err
	}
	if err := c.LogLevel.UnmarshalText([]byte(strings.ToUpper(level))); err != nil {
		return c, fmt.Errorf("LOG_LEVEL: %w", err)
	}
	if c.DatabaseURL, err = required(getenv, "DATABASE_URL"); err != nil {
		return c, err
	}
	if c.DatabaseMaxConns, err = positiveInt(getenv, "DATABASE_MAX_CONNS"); err != nil {
		return c, err
	}
	if c.DatabaseMinConns, err = positiveInt(getenv, "DATABASE_MIN_CONNS"); err != nil {
		return c, err
	}
	if c.DatabaseMinConns > c.DatabaseMaxConns {
		return c, fmt.Errorf("DATABASE_MIN_CONNS must not exceed DATABASE_MAX_CONNS")
	}
	return c, nil
}

func required(getenv func(string) string, name string) (string, error) {
	value := strings.TrimSpace(getenv(name))
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return value, nil
}

func duration(getenv func(string) string, name string) (time.Duration, error) {
	value, err := required(getenv, name)
	if err != nil {
		return 0, err
	}
	d, err := time.ParseDuration(value)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration (for example 5s)", name)
	}
	return d, nil
}

func positiveInt(getenv func(string) string, name string) (int32, error) {
	value, err := required(getenv, name)
	if err != nil {
		return 0, err
	}
	n, err := strconv.ParseInt(value, 10, 32)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return int32(n), nil
}
