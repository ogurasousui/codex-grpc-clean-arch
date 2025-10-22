package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/platform/config"
)

func main() {
	var (
		configPath    = flag.String("config", "", "path to config file (defaults to CONFIG_PATH env or assets/local.yaml)")
		migrationsDir = flag.String("dir", "assets/migrations", "directory containing migration files")
	)
	flag.Parse()

	action := "up"
	if flag.NArg() > 0 {
		action = flag.Arg(0)
	}

	cfgPath := effectiveConfigPath(*configPath)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if err := runMigration(action, *migrationsDir, cfg.Database.DSN()); err != nil {
		log.Fatalf("migration %s failed: %v", action, err)
	}

	log.Printf("migration %s completed", action)
}

func effectiveConfigPath(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if env := os.Getenv("CONFIG_PATH"); env != "" {
		return env
	}
	return "assets/local.yaml"
}

func runMigration(action, dir, dsn string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolve path for %s: %w", dir, err)
	}
	absDir = filepath.ToSlash(absDir)

	m, err := migrate.New(fmt.Sprintf("file://%s", absDir), dsn)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}
	defer m.Close()

	switch action {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			return err
		}
		return nil
	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			return err
		}
		return nil
	case "drop":
		return m.Drop()
	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			if err == migrate.ErrNilVersion {
				log.Printf("no migration applied")
				return nil
			}
			return err
		}
		log.Printf("version=%d dirty=%t", version, dirty)
		return nil
	default:
		return fmt.Errorf("unsupported action %q", action)
	}
}
