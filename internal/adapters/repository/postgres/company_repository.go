package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/core/company"
	pgdb "github.com/ogurasousui/codex-grpc-clean-arch/internal/platform/db/postgres"
)

const companyUniqueViolationCode = "23505"

// CompanyRepository は PostgreSQL を利用した会社永続化の実装です。
type CompanyRepository struct {
	pool pgdb.Queryer
}

// NewCompanyRepository は CompanyRepository を生成します。
func NewCompanyRepository(pool pgdb.Queryer) *CompanyRepository {
	return &CompanyRepository{pool: pool}
}

// Create は会社を新規作成します。
func (r *CompanyRepository) Create(ctx context.Context, c *company.Company) (*company.Company, error) {
	exec := pgdb.QueryerFromContext(ctx, r.pool)
	row := exec.QueryRow(ctx, `
        INSERT INTO companies (name, code, status, description, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, name, code, status, description, created_at, updated_at
    `, c.Name, c.Code, c.Status, nullableString(c.Description), c.CreatedAt, c.UpdatedAt)

	created, err := scanCompany(row)
	if err != nil {
		return nil, translateCompanyPgError(err)
	}
	return created, nil
}

// Update は会社情報を更新します。
func (r *CompanyRepository) Update(ctx context.Context, c *company.Company) (*company.Company, error) {
	exec := pgdb.QueryerFromContext(ctx, r.pool)
	row := exec.QueryRow(ctx, `
        UPDATE companies
           SET name = $1,
               code = $2,
               status = $3,
               description = $4,
               updated_at = $5
         WHERE id = $6
        RETURNING id, name, code, status, description, created_at, updated_at
    `, c.Name, c.Code, c.Status, nullableString(c.Description), c.UpdatedAt, c.ID)

	updated, err := scanCompany(row)
	if err != nil {
		return nil, translateCompanyPgError(err)
	}
	return updated, nil
}

// Delete は会社を削除します。
func (r *CompanyRepository) Delete(ctx context.Context, id string) error {
	exec := pgdb.QueryerFromContext(ctx, r.pool)
	tag, err := exec.Exec(ctx, `DELETE FROM companies WHERE id = $1`, id)
	if err != nil {
		return translateCompanyPgError(err)
	}
	if tag.RowsAffected() == 0 {
		return company.ErrCompanyNotFound
	}
	return nil
}

// FindByID は ID で会社を取得します。
func (r *CompanyRepository) FindByID(ctx context.Context, id string) (*company.Company, error) {
	exec := pgdb.QueryerFromContext(ctx, r.pool)
	row := exec.QueryRow(ctx, `
        SELECT id, name, code, status, description, created_at, updated_at
          FROM companies
         WHERE id = $1
         LIMIT 1
    `, id)

	found, err := scanCompany(row)
	if err != nil {
		return nil, translateCompanyPgError(err)
	}
	return found, nil
}

// FindByCode はコードで会社を取得します。
func (r *CompanyRepository) FindByCode(ctx context.Context, code string) (*company.Company, error) {
	exec := pgdb.QueryerFromContext(ctx, r.pool)
	row := exec.QueryRow(ctx, `
        SELECT id, name, code, status, description, created_at, updated_at
          FROM companies
         WHERE code = $1
         LIMIT 1
    `, code)

	found, err := scanCompany(row)
	if err != nil {
		return nil, translateCompanyPgError(err)
	}
	return found, nil
}

// List は会社の一覧を取得します。
func (r *CompanyRepository) List(ctx context.Context, filter company.ListCompaniesFilter) ([]*company.Company, string, error) {
	if filter.Limit <= 0 {
		return nil, "", company.ErrInvalidPageSize
	}
	if filter.Offset < 0 {
		return nil, "", company.ErrInvalidPageToken
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
        SELECT id, name, code, status, description, created_at, updated_at
          FROM companies` + whereClause + `
         ORDER BY created_at DESC, id DESC
         LIMIT ` + limitPlaceholder + `
        OFFSET ` + offsetPlaceholder + `
    `

	exec := pgdb.QueryerFromContext(ctx, r.pool)
	rows, err := exec.Query(ctx, query, args...)
	if err != nil {
		return nil, "", translateCompanyPgError(err)
	}
	defer rows.Close()

	var companies []*company.Company
	for rows.Next() {
		found, err := scanCompany(rows)
		if err != nil {
			return nil, "", translateCompanyPgError(err)
		}
		companies = append(companies, found)
	}

	if err := rows.Err(); err != nil {
		return nil, "", translateCompanyPgError(err)
	}

	var nextToken string
	if len(companies) > filter.Limit {
		nextToken = strconv.Itoa(filter.Offset + filter.Limit)
		companies = companies[:filter.Limit]
	}

	return companies, nextToken, nil
}

func scanCompany(row pgx.Row) (*company.Company, error) {
	var (
		id                   string
		name                 string
		code                 string
		status               string
		description          sql.NullString
		createdAt, updatedAt time.Time
	)

	if err := row.Scan(&id, &name, &code, &status, &description, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, company.ErrCompanyNotFound
		}
		return nil, err
	}

	var descPtr *string
	if description.Valid {
		desc := description.String
		descPtr = &desc
	}

	return &company.Company{
		ID:          id,
		Name:        name,
		Code:        code,
		Status:      company.Status(status),
		Description: descPtr,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

func translateCompanyPgError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == companyUniqueViolationCode {
			return company.ErrCodeAlreadyExists
		}
	}
	return err
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}
