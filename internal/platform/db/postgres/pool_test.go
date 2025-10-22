package postgres

import (
	"testing"
	"time"

	"github.com/ogurasousui/codex-grpc-clean-arch/internal/platform/config"
)

func TestBuildPoolConfig(t *testing.T) {
	t.Parallel()

	dbCfg := config.DatabaseConfig{
		Host:            "localhost",
		Port:            15432,
		User:            "user",
		Password:        "pass",
		Name:            "db",
		SSLMode:         "disable",
		MaxOpenConns:    20,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	poolCfg, err := BuildPoolConfig(dbCfg)
	if err != nil {
		t.Fatalf("BuildPoolConfig returned error: %v", err)
	}

	if poolCfg.MaxConns != 20 {
		t.Errorf("expected MaxConns 20, got %d", poolCfg.MaxConns)
	}

	if poolCfg.MinConns != 5 {
		t.Errorf("expected MinConns 5, got %d", poolCfg.MinConns)
	}

	if poolCfg.MaxConnLifetime != 30*time.Minute {
		t.Errorf("unexpected MaxConnLifetime: %v", poolCfg.MaxConnLifetime)
	}

	if poolCfg.MaxConnIdleTime != 10*time.Minute {
		t.Errorf("unexpected MaxConnIdleTime: %v", poolCfg.MaxConnIdleTime)
	}

	if poolCfg.ConnConfig.Database != "db" {
		t.Errorf("expected database db, got %s", poolCfg.ConnConfig.Database)
	}
}
