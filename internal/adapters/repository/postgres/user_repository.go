package postgres

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/core/user"
)

const uniqueViolationCode = "23505"

type pgxPool interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// UserRepository は PostgreSQL を利用したユーザー永続化の実装です。
type UserRepository struct {
	pool pgxPool
}

// NewUserRepository は UserRepository を生成します。
func NewUserRepository(pool pgxPool) *UserRepository {
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

// List はユーザーの一覧を取得します。
func (r *UserRepository) List(ctx context.Context, filter user.ListUsersFilter) ([]*user.User, string, error) {
	if filter.Limit <= 0 {
		return nil, "", user.ErrInvalidPageSize
	}
	if filter.Offset < 0 {
		return nil, "", user.ErrInvalidPageToken
	}

	limitWithBuffer := filter.Limit + 1

	args := make([]any, 0, 3)
	conditions := make([]string, 0, 1)

	if filter.Status != nil {
		placeholder := "$" + strconv.Itoa(len(args)+1)
		conditions = append(conditions, "status = "+placeholder)
		args = append(args, *filter.Status)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	limitPlaceholder := "$" + strconv.Itoa(len(args)+1)
	args = append(args, limitWithBuffer)
	offsetPlaceholder := "$" + strconv.Itoa(len(args)+1)
	args = append(args, filter.Offset)

	query := `
        SELECT id, email, name, status, created_at, updated_at
          FROM users` + whereClause + `
         ORDER BY created_at DESC, id DESC
         LIMIT ` + limitPlaceholder + `
        OFFSET ` + offsetPlaceholder + `
    `

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", translatePgError(err)
	}
	defer rows.Close()

	var users []*user.User
	for rows.Next() {
		found, err := scanUser(rows)
		if err != nil {
			return nil, "", translatePgError(err)
		}
		users = append(users, found)
	}

	if err := rows.Err(); err != nil {
		return nil, "", translatePgError(err)
	}

	var nextToken string
	if len(users) > filter.Limit {
		nextToken = strconv.Itoa(filter.Offset + filter.Limit)
		users = users[:filter.Limit]
	}

	return users, nextToken, nil
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
