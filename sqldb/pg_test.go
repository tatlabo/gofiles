package sqldb

import (
	"testing"
	"time"
)

func TestPgxPoolConfig(t *testing.T) {
	config, err := pgxPoolConfig("postgres://golang:golang@localhost:5432/json?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	if config.MinConns != 1 {
		t.Errorf("MinConns = %d, want 0", config.MinConns)
	}
	if config.MaxConns != 12 {
		t.Errorf("MaxConns = %d, want 12", config.MaxConns)
	}
	if config.MaxConnIdleTime != 10*time.Minute {
		t.Errorf("MaxConnIdleTime = %s, want %s", config.MaxConnIdleTime, 10*time.Minute)
	}
}
