//go:build integration

package integration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	repo "github.com/ogurasousui/codex-grpc-clean-arch/internal/adapters/repository/postgres"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/core/user"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/platform/config"
	pg "github.com/ogurasousui/codex-grpc-clean-arch/internal/platform/db/postgres"
)

const (
	migrationsDir = "assets/migrations"
	seedsDir      = "assets/seeds"
)

var projectRoot = func() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to determine project root")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
}()

func TestUserCRUDIntegration(t *testing.T) {
	t.Parallel()

	cfgPath := resolvePath(configPathFromEnv())

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if err := resetMigrations(cfg.Database.DSN(), resolvePath(migrationsDir)); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	if err := applySeeds(cfg.Database.DSN(), resolvePath(seedsDir)); err != nil {
		t.Fatalf("failed to apply seeds: %v", err)
	}

	ctx := context.Background()
	pool, err := pg.NewPool(ctx, cfg.Database)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	userRepo := repo.NewUserRepository(pool)
	svc := user.NewService(userRepo, stubClock{now: time.Now().UTC()}, pg.NewTransactionManager(pool))

	created, err := svc.CreateUser(ctx, user.CreateUserInput{Email: "integration@example.com", Name: "Integration"})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}

	found, err := userRepo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("FindByID error: %v", err)
	}
	if found.Email != created.Email {
		t.Fatalf("expected email %s, got %s", created.Email, found.Email)
	}

	newName := "Updated"
	newStatus := user.StatusInactive
	updated, err := svc.UpdateUser(ctx, user.UpdateUserInput{ID: created.ID, Name: &newName, Status: &newStatus})
	if err != nil {
		t.Fatalf("UpdateUser error: %v", err)
	}
	if updated.Name != newName || updated.Status != newStatus {
		t.Fatalf("update not applied: %+v", updated)
	}

	if err := svc.DeleteUser(ctx, user.DeleteUserInput{ID: created.ID}); err != nil {
		t.Fatalf("DeleteUser error: %v", err)
	}

	if _, err := userRepo.FindByID(ctx, created.ID); !errors.Is(err, user.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserCRUDIntegration_RollbackOnError(t *testing.T) {
	t.Parallel()

	cfgPath := resolvePath(configPathFromEnv())

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if err := resetMigrations(cfg.Database.DSN(), resolvePath(migrationsDir)); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	ctx := context.Background()
	pool, err := pg.NewPool(ctx, cfg.Database)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	tm := pg.NewTransactionManager(pool)
	userRepo := repo.NewUserRepository(pool)

	forcedErr := errors.New("force rollback")
	email := "rollback@example.com"

	err = tm.WithinReadWrite(ctx, func(txCtx context.Context) error {
		now := time.Now().UTC()
		_, err := userRepo.Create(txCtx, &user.User{
			Email:     email,
			Name:      "Rollback",
			Status:    user.StatusActive,
			CreatedAt: now,
			UpdatedAt: now,
		})
		if err != nil {
			return err
		}
		return forcedErr
	})

	if !errors.Is(err, forcedErr) {
		t.Fatalf("expected forced rollback error, got %v", err)
	}

	_, findErr := userRepo.FindByEmail(ctx, email)
	if !errors.Is(findErr, user.ErrUserNotFound) {
		t.Fatalf("expected user to be rolled back, got %v", findErr)
	}
}

func resetMigrations(dsn, dir string) error {
	m, err := migrate.New("file://"+filepath.ToSlash(dir), dsn)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

func applySeeds(dsn, dir string) error {
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	m, err := migrate.New("file://"+filepath.ToSlash(dir), dsn)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

func configPathFromEnv() string {
	if v := os.Getenv("CONFIG_PATH"); v != "" {
		return v
	}
	return "assets/local.yaml"
}

func resolvePath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(projectRoot, p)
}

type stubClock struct {
	now time.Time
}

func (s stubClock) Now() time.Time {
	return s.now
}
