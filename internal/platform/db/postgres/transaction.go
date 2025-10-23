package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// transactionContextKey はコンテキストにトランザクションを格納するためのキーです。
type transactionContextKey struct{}

var txContextKey = transactionContextKey{}

// TransactionManager は pgx を用いたトランザクション制御を提供します。
type txStarter interface {
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

// TransactionManager は pgx を用いたトランザクション制御を提供します。
type TransactionManager struct {
	pool txStarter
}

// NewTransactionManager は TransactionManager を生成します。
func NewTransactionManager(pool txStarter) *TransactionManager {
	if pool == nil {
		return nil
	}
	return &TransactionManager{pool: pool}
}

// WithinReadOnly は読み取り専用トランザクションを開始し、fn を実行します。
func (m *TransactionManager) WithinReadOnly(ctx context.Context, fn func(context.Context) error) error {
	if m == nil {
		return fn(ctx)
	}
	return m.within(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly}, fn)
}

// WithinReadWrite は読み書きトランザクションを開始し、fn を実行します。
func (m *TransactionManager) WithinReadWrite(ctx context.Context, fn func(context.Context) error) error {
	if m == nil {
		return fn(ctx)
	}
	return m.within(ctx, pgx.TxOptions{AccessMode: pgx.ReadWrite}, fn)
}

func (m *TransactionManager) within(ctx context.Context, opts pgx.TxOptions, fn func(context.Context) error) error {
	if fn == nil {
		return fmt.Errorf("postgres: transaction function is required")
	}

	if _, ok := txFromContext(ctx); ok {
		return fn(ctx)
	}

	tx, err := m.pool.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("postgres: begin tx: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	txCtx := contextWithTx(ctx, tx)

	if err := fn(txCtx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			return errors.Join(err, fmt.Errorf("postgres: rollback: %w", rbErr))
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		if !errors.Is(err, pgx.ErrTxClosed) {
			if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
				return errors.Join(fmt.Errorf("postgres: commit: %w", err), fmt.Errorf("postgres: rollback after commit failure: %w", rbErr))
			}
		}
		return fmt.Errorf("postgres: commit: %w", err)
	}

	committed = true
	return nil
}

func contextWithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txContextKey, tx)
}

func txFromContext(ctx context.Context) (pgx.Tx, bool) {
	if ctx == nil {
		return nil, false
	}
	tx, ok := ctx.Value(txContextKey).(pgx.Tx)
	return tx, ok
}

// QueryerFromContext はコンテキスト内にトランザクションが存在すればそれを返し、存在しなければ fallback を返します。
func QueryerFromContext(ctx context.Context, fallback Queryer) Queryer {
	if tx, ok := txFromContext(ctx); ok {
		return tx
	}
	return fallback
}

// Queryer は pgx.Tx および pgxpool.Pool と互換性のあるクエリ実行インターフェースです。
type Queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}
