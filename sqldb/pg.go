// Package sqldb
//
// # pgx pool settings
package sqldb

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gofor-little/env"
	"github.com/jackc/pgx/v5/pgxpool"
)

func pgEnvConfPath() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "pg.env")
}

func pgConfEnv() (string, error) {
	if err := env.Load(pgEnvConfPath()); err != nil {
		return "", err
	}

	dbHost := env.Get("DB_HOST", "")
	dbName := env.Get("DB_NAME", "")
	dbUser := env.Get("DB_USER", "")
	dbPass := env.Get("DB_PASS", "")
	dbSSL := env.Get("DB_SSL", "disable")
	dbPort := env.Get("DB_PORT", "5432")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", dbUser, dbPass, dbHost, dbPort, dbName, dbSSL), nil
}

func pgxPoolConfig(connStr string) (*pgxpool.Config, error) {
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}

	config.MaxConns = 12                               // Maximum number of connections in the pool
	config.MinConns = 1                                // Minimum number of idle connections to keep open
	config.MaxConnLifetime = time.Hour * 1             // Maximum time a connection can live
	config.MaxConnIdleTime = time.Minute * 10          // Maximum time a connection can sit idle
	config.HealthCheckPeriod = time.Minute * 1         // How often to check if idle connections are alive
	config.ConnConfig.ConnectTimeout = time.Second * 5 // Timeout for establishing a new connection

	return config, nil
}

// PgxPool returns pgx.Pool
func PgxPool() (pool *pgxpool.Pool, err error) {
	connStr, err := pgConfEnv()
	if err != nil {
		return nil, err
	}

	config, err := pgxPoolConfig(connStr)
	if err != nil {
		return nil, err
	}

	pool, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	return pool, nil
}
