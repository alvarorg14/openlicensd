package store

import (
	"os"
	"testing"
)

func TestApplyPoolConfigOverridesURL(t *testing.T) {
	t.Parallel()

	poolCfg, err := parsePoolConfig("postgres://example?pool_max_conns=5", PoolConfig{})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	applyPoolConfig(poolCfg, PoolConfig{
		MaxConns: 10,
	})

	if poolCfg.MaxConns != 10 {
		t.Fatalf("max conns=%d want 10", poolCfg.MaxConns)
	}
}

func TestApplyPoolConfigZeroLeavesURL(t *testing.T) {
	t.Parallel()

	poolCfg, err := parsePoolConfig("postgres://example?pool_max_conns=5", PoolConfig{})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	if poolCfg.MaxConns != 5 {
		t.Fatalf("max conns=%d want 5 from URL", poolCfg.MaxConns)
	}
}

func TestApplyPoolConfigStatementTimeout(t *testing.T) {
	t.Parallel()

	poolCfg, err := parsePoolConfig("postgres://example", PoolConfig{
		StatementTimeoutSeconds: 30,
	})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	got := poolCfg.ConnConfig.RuntimeParams["statement_timeout"]
	if got != "30000" {
		t.Fatalf("statement_timeout=%q want 30000", got)
	}
}

func TestNewWithPoolMaxConns(t *testing.T) {
	databaseURL := os.Getenv("OPENLICENSD_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("OPENLICENSD_DATABASE_URL not set")
	}

	ctx := t.Context()
	st, err := NewWithPool(ctx, databaseURL, PoolConfig{MaxConns: 2})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer st.Close()

	if got := st.PoolStat().MaxConns(); got != 2 {
		t.Fatalf("max conns=%d want 2", got)
	}
}
