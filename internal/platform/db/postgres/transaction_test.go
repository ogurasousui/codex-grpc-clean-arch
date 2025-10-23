package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	pgxmock "github.com/pashagolub/pgxmock/v4"
)

func TestTransactionManager_ReadWriteCommit(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	tm := NewTransactionManager(mock)

	mock.ExpectBeginTx(pgx.TxOptions{AccessMode: pgx.ReadWrite})
	mock.ExpectCommit()

	err = tm.WithinReadWrite(context.Background(), func(ctx context.Context) error {
		if _, ok := txFromContext(ctx); !ok {
			t.Fatalf("transaction not injected into context")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("WithinReadWrite returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestTransactionManager_ReadOnlyRollbackOnError(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	tm := NewTransactionManager(mock)

	mock.ExpectBeginTx(pgx.TxOptions{AccessMode: pgx.ReadOnly})
	mock.ExpectRollback()

	expectedErr := errors.New("usecase error")
	err = tm.WithinReadOnly(context.Background(), func(ctx context.Context) error {
		if _, ok := txFromContext(ctx); !ok {
			t.Fatalf("transaction not injected into context")
		}
		return expectedErr
	})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestTransactionManager_NestedReuse(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	tm := NewTransactionManager(mock)

	mock.ExpectBeginTx(pgx.TxOptions{AccessMode: pgx.ReadWrite})
	mock.ExpectCommit()

	err = tm.WithinReadWrite(context.Background(), func(ctx context.Context) error {
		return tm.WithinReadOnly(ctx, func(inner context.Context) error {
			if _, ok := txFromContext(inner); !ok {
				t.Fatalf("nested transaction lost context")
			}
			return nil
		})
	})

	if err != nil {
		t.Fatalf("nested transaction returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
