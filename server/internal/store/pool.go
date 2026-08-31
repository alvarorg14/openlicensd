package store

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolConfig holds optional pgxpool overrides. Zero values leave pgx defaults
// (or values already parsed from the database URL query string).
type PoolConfig struct {
	MaxConns                int
	MinConns                int
	MaxConnIdleMinutes      int
	StatementTimeoutSeconds int
}

func applyPoolConfig(poolCfg *pgxpool.Config, cfg PoolConfig) {
	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = int32(cfg.MaxConns)
	}
	if cfg.MinConns > 0 {
		poolCfg.MinConns = int32(cfg.MinConns)
	}
	if cfg.MaxConnIdleMinutes > 0 {
		poolCfg.MaxConnIdleTime = time.Duration(cfg.MaxConnIdleMinutes) * time.Minute
	}
	if cfg.StatementTimeoutSeconds > 0 {
		if poolCfg.ConnConfig.RuntimeParams == nil {
			poolCfg.ConnConfig.RuntimeParams = make(map[string]string)
		}
		ms := cfg.StatementTimeoutSeconds * 1000
		poolCfg.ConnConfig.RuntimeParams["statement_timeout"] = strconv.Itoa(ms)
	}
}

func logPoolConfig(poolCfg *pgxpool.Config) {
	statementTimeout := poolCfg.ConnConfig.RuntimeParams["statement_timeout"]
	slog.Default().Info("database pool configured",
		slog.Int("max_conns", int(poolCfg.MaxConns)),
		slog.Int("min_conns", int(poolCfg.MinConns)),
		slog.Duration("max_conn_idle_time", poolCfg.MaxConnIdleTime),
		slog.String("statement_timeout_ms", statementTimeout),
	)
}

func parsePoolConfig(databaseURL string, cfg PoolConfig) (*pgxpool.Config, error) {
	poolCfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}
	applyPoolConfig(poolCfg, cfg)
	return poolCfg, nil
}
