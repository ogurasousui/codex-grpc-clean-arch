package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_Success(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`server:
  listen_addr: ":50051"

database:
  host: localhost
  port: 15432
  user: user
  password: pass
  name: app
  ssl_mode: disable
  max_open_conns: 10
  max_idle_conns: 5
  conn_max_lifetime: "15m"
  conn_max_idle_time: "5m"
`)

	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server.ListenAddr != ":50051" {
		t.Errorf("unexpected listen addr: %s", cfg.Server.ListenAddr)
	}

	if cfg.Database.ConnMaxLifetime != 15*time.Minute {
		t.Errorf("expected ConnMaxLifetime 15m, got %v", cfg.Database.ConnMaxLifetime)
	}

	if cfg.Database.ConnMaxIdleTime != 5*time.Minute {
		t.Errorf("expected ConnMaxIdleTime 5m, got %v", cfg.Database.ConnMaxIdleTime)
	}
}

func TestLoad_MissingField(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("expected error when required fields are missing")
	}
}

func TestDatabaseConfigDSN_EscapesCredentials(t *testing.T) {
	t.Parallel()

	cfg := DatabaseConfig{
		Host:     "db.local",
		Port:     5432,
		User:     "user@domain",
		Password: "p@ss:word",
		Name:     "app_db",
		SSLMode:  "require",
	}

	dsn := cfg.DSN()

	expected := "postgres://user%40domain:p%40ss%3Aword@db.local:5432/app_db?sslmode=require"
	if dsn != expected {
		t.Fatalf("unexpected DSN. want %s got %s", expected, dsn)
	}
}
