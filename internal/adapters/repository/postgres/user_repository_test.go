package postgres

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/core/user"
	pgxmock "github.com/pashagolub/pgxmock/v4"
)

type stubRow struct {
	scanFn func(dest ...interface{}) error
}

func (s stubRow) Scan(dest ...interface{}) error {
	return s.scanFn(dest...)
}

func TestScanUser_Success(t *testing.T) {
	t.Parallel()

	createdAt := time.Now().UTC()
	updatedAt := createdAt.Add(time.Minute)

	row := stubRow{scanFn: func(dest ...interface{}) error {
		if len(dest) != 6 {
			return errors.New("unexpected dest length")
		}
		*(dest[0].(*string)) = "user-1"
		*(dest[1].(*string)) = "user@example.com"
		*(dest[2].(*string)) = "User"
		*(dest[3].(*string)) = string(user.StatusActive)
		*(dest[4].(*time.Time)) = createdAt
		*(dest[5].(*time.Time)) = updatedAt
		return nil
	}}

	u, err := scanUser(row)
	if err != nil {
		t.Fatalf("scanUser returned error: %v", err)
	}

	if u.ID != "user-1" || u.Email != "user@example.com" {
		t.Fatalf("unexpected user %+v", u)
	}
}

func TestScanUser_NoRows(t *testing.T) {
	t.Parallel()

	row := stubRow{scanFn: func(dest ...interface{}) error {
		return pgx.ErrNoRows
	}}

	_, err := scanUser(row)
	if !errors.Is(err, user.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestTranslatePgError(t *testing.T) {
	t.Parallel()

	pgErr := &pgconn.PgError{Code: uniqueViolationCode}
	if !errors.Is(translatePgError(pgErr), user.ErrEmailAlreadyExists) {
		t.Fatalf("expected email exists error mapping")
	}

	otherErr := errors.New("random")
	if translatePgError(otherErr) != otherErr {
		t.Fatalf("unexpected translation for generic error")
	}
}

func TestUserRepository_List_WithNextToken(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepository(mock)

	query := regexp.QuoteMeta(`
        SELECT id, email, name, status, created_at, updated_at
          FROM users
         ORDER BY created_at DESC, id DESC
         LIMIT $1
        OFFSET $2
    `)

	now := time.Now().UTC()
	rows := pgxmock.NewRows([]string{"id", "email", "name", "status", "created_at", "updated_at"}).
		AddRow("user-1", "user1@example.com", "User1", string(user.StatusActive), now, now).
		AddRow("user-2", "user2@example.com", "User2", string(user.StatusActive), now, now).
		AddRow("user-3", "user3@example.com", "User3", string(user.StatusInactive), now, now)

	mock.ExpectQuery(query).
		WithArgs(3, 0).
		WillReturnRows(rows)

	users, nextToken, err := repo.List(context.Background(), user.ListUsersFilter{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}

	if nextToken != "2" {
		t.Fatalf("expected next token '2', got %s", nextToken)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUserRepository_List_WithStatusFilter(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepository(mock)
	inactive := user.StatusInactive
	query := regexp.QuoteMeta(`
        SELECT id, email, name, status, created_at, updated_at
          FROM users WHERE status = $1
         ORDER BY created_at DESC, id DESC
         LIMIT $2
        OFFSET $3
    `)

	now := time.Now().UTC()
	rows := pgxmock.NewRows([]string{"id", "email", "name", "status", "created_at", "updated_at"}).
		AddRow("user-5", "inactive@example.com", "Inactive", string(user.StatusInactive), now, now)

	mock.ExpectQuery(query).
		WithArgs(inactive, 3, 0).
		WillReturnRows(rows)

	users, nextToken, err := repo.List(context.Background(), user.ListUsersFilter{Limit: 2, Offset: 0, Status: &inactive})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}

	if nextToken != "" {
		t.Fatalf("expected empty next token, got %s", nextToken)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUserRepository_List_InvalidArguments(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepository(mock)

	if _, _, err := repo.List(context.Background(), user.ListUsersFilter{Limit: 0, Offset: 0}); !errors.Is(err, user.ErrInvalidPageSize) {
		t.Fatalf("expected ErrInvalidPageSize, got %v", err)
	}

	if _, _, err := repo.List(context.Background(), user.ListUsersFilter{Limit: 1, Offset: -1}); !errors.Is(err, user.ErrInvalidPageToken) {
		t.Fatalf("expected ErrInvalidPageToken, got %v", err)
	}
}
