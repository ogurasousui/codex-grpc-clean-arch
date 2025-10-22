package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/core/user"
)

const uniqueViolationCode = "23505"

// UserRepository は PostgreSQL を利用したユーザー永続化の実装です。
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository は UserRepository を生成します。
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create はユーザーを新規作成します。
func (r *UserRepository) Create(ctx context.Context, u *user.User) (*user.User, error) {
	row := r.pool.QueryRow(ctx, `
        INSERT INTO users (email, name, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, email, name, status, created_at, updated_at
    `, u.Email, u.Name, u.Status, u.CreatedAt, u.UpdatedAt)

	created, err := scanUser(row)
	if err != nil {
		return nil, translatePgError(err)
	}
	return created, nil
}

// Update はユーザー情報を更新します。
func (r *UserRepository) Update(ctx context.Context, u *user.User) (*user.User, error) {
	row := r.pool.QueryRow(ctx, `
        UPDATE users
           SET name = $1,
               status = $2,
               updated_at = $3
         WHERE id = $4
        RETURNING id, email, name, status, created_at, updated_at
    `, u.Name, u.Status, u.UpdatedAt, u.ID)

	updated, err := scanUser(row)
	if err != nil {
		return nil, translatePgError(err)
	}
	return updated, nil
}

// Delete はユーザーを削除します。
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return translatePgError(err)
	}
	if tag.RowsAffected() == 0 {
		return user.ErrUserNotFound
	}
	return nil
}

// FindByID はIDでユーザーを取得します。
func (r *UserRepository) FindByID(ctx context.Context, id string) (*user.User, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, email, name, status, created_at, updated_at
          FROM users
         WHERE id = $1
         LIMIT 1
    `, id)

	found, err := scanUser(row)
	if err != nil {
		return nil, translatePgError(err)
	}
	return found, nil
}

// FindByEmail はメールアドレスでユーザーを取得します。
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, email, name, status, created_at, updated_at
          FROM users
         WHERE email = $1
         LIMIT 1
    `, email)

	found, err := scanUser(row)
	if err != nil {
		return nil, translatePgError(err)
	}
	return found, nil
}

func scanUser(row pgx.Row) (*user.User, error) {
	var (
		id                   string
		email                string
		name                 string
		status               string
		createdAt, updatedAt time.Time
	)

	if err := row.Scan(&id, &email, &name, &status, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}

	return &user.User{
		ID:        id,
		Email:     email,
		Name:      name,
		Status:    user.Status(status),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func translatePgError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == uniqueViolationCode {
			return user.ErrEmailAlreadyExists
		}
	}
	return err
}
