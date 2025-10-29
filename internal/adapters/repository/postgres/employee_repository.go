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
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/core/employee"
	pgdb "github.com/ogurasousui/codex-grpc-clean-arch/internal/platform/db/postgres"
)

const (
	employeeUniqueViolationCode     = "23505"
	employeeForeignKeyViolationCode = "23503"
	employeeCheckViolationCode      = "23514"
)

// EmployeeRepository は PostgreSQL を利用した社員永続化の実装です。
type EmployeeRepository struct {
	pool pgdb.Queryer
}

// NewEmployeeRepository は EmployeeRepository を生成します。
func NewEmployeeRepository(pool pgdb.Queryer) *EmployeeRepository {
	return &EmployeeRepository{pool: pool}
}

// Create は社員を新規作成します。
func (r *EmployeeRepository) Create(ctx context.Context, e *employee.Employee) (*employee.Employee, error) {
	exec := pgdb.QueryerFromContext(ctx, r.pool)
	row := exec.QueryRow(ctx, `
        WITH inserted AS (
            INSERT INTO employees (company_id, employee_code, user_id, status, hired_at, terminated_at, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
            RETURNING id, company_id, employee_code, user_id, status, hired_at, terminated_at, created_at, updated_at
        )
        SELECT i.id, i.company_id, i.employee_code, i.user_id, i.status, i.hired_at, i.terminated_at, i.created_at, i.updated_at,
               u.id, u.email, u.name, u.status, u.created_at, u.updated_at
          FROM inserted i
          JOIN users u ON u.id = i.user_id
    `,
		e.CompanyID,
		e.EmployeeCode,
		e.UserID,
		string(e.Status),
		nullableTime(e.HiredAt),
		nullableTime(e.TerminatedAt),
		e.CreatedAt,
		e.UpdatedAt,
	)

	created, err := scanEmployee(row)
	if err != nil {
		return nil, translateEmployeePgError(err)
	}
	return created, nil
}

// Update は社員情報を更新します。
func (r *EmployeeRepository) Update(ctx context.Context, e *employee.Employee) (*employee.Employee, error) {
	exec := pgdb.QueryerFromContext(ctx, r.pool)
	row := exec.QueryRow(ctx, `
        WITH updated AS (
            UPDATE employees
               SET employee_code = $1,
                   user_id = $2,
                   status = $3,
                   hired_at = $4,
                   terminated_at = $5,
                   updated_at = $6
             WHERE id = $7
            RETURNING id, company_id, employee_code, user_id, status, hired_at, terminated_at, created_at, updated_at
        )
        SELECT urow.id, urow.company_id, urow.employee_code, urow.user_id, urow.status, urow.hired_at, urow.terminated_at, urow.created_at, urow.updated_at,
               usr.id, usr.email, usr.name, usr.status, usr.created_at, usr.updated_at
          FROM updated urow
          JOIN users usr ON usr.id = urow.user_id
    `,
		e.EmployeeCode,
		e.UserID,
		string(e.Status),
		nullableTime(e.HiredAt),
		nullableTime(e.TerminatedAt),
		e.UpdatedAt,
		e.ID,
	)

	updated, err := scanEmployee(row)
	if err != nil {
		return nil, translateEmployeePgError(err)
	}
	return updated, nil
}

// Delete は社員を削除します。
func (r *EmployeeRepository) Delete(ctx context.Context, id string) error {
	exec := pgdb.QueryerFromContext(ctx, r.pool)
	tag, err := exec.Exec(ctx, `DELETE FROM employees WHERE id = $1`, id)
	if err != nil {
		return translateEmployeePgError(err)
	}
	if tag.RowsAffected() == 0 {
		return employee.ErrEmployeeNotFound
	}
	return nil
}

// FindByID は ID で社員を取得します。
func (r *EmployeeRepository) FindByID(ctx context.Context, id string) (*employee.Employee, error) {
	exec := pgdb.QueryerFromContext(ctx, r.pool)
	row := exec.QueryRow(ctx, `
        SELECT e.id,
               e.company_id,
               e.employee_code,
               e.user_id,
               e.status,
               e.hired_at,
               e.terminated_at,
               e.created_at,
               e.updated_at,
               u.id,
               u.email,
               u.name,
               u.status,
               u.created_at,
               u.updated_at
          FROM employees e
          JOIN users u ON u.id = e.user_id
         WHERE e.id = $1
         LIMIT 1
    `, id)

	found, err := scanEmployee(row)
	if err != nil {
		return nil, translateEmployeePgError(err)
	}
	return found, nil
}

// FindByCompanyAndCode は会社 ID と社員コードで検索します。
func (r *EmployeeRepository) FindByCompanyAndCode(ctx context.Context, companyID, employeeCode string) (*employee.Employee, error) {
	exec := pgdb.QueryerFromContext(ctx, r.pool)
	row := exec.QueryRow(ctx, `
        SELECT e.id,
               e.company_id,
               e.employee_code,
               e.user_id,
               e.status,
               e.hired_at,
               e.terminated_at,
               e.created_at,
               e.updated_at,
               u.id,
               u.email,
               u.name,
               u.status,
               u.created_at,
               u.updated_at
          FROM employees e
          JOIN users u ON u.id = e.user_id
         WHERE e.company_id = $1 AND e.employee_code = $2
         LIMIT 1
    `, companyID, employeeCode)

	found, err := scanEmployee(row)
	if err != nil {
		return nil, translateEmployeePgError(err)
	}
	return found, nil
}

// List は社員の一覧を取得します。
func (r *EmployeeRepository) List(ctx context.Context, filter employee.ListEmployeesFilter) ([]*employee.Employee, string, error) {
	if strings.TrimSpace(filter.CompanyID) == "" {
		return nil, "", employee.ErrInvalidCompanyID
	}
	if filter.Limit <= 0 {
		return nil, "", employee.ErrInvalidPageSize
	}
	if filter.Offset < 0 {
		return nil, "", employee.ErrInvalidPageToken
	}

	limitWithBuffer := filter.Limit + 1

	args := make([]any, 0, 4)
	conditions := make([]string, 0, 2)

	companyPlaceholder := "$" + strconv.Itoa(len(args)+1)
	conditions = append(conditions, "e.company_id = "+companyPlaceholder)
	args = append(args, filter.CompanyID)

	if filter.Status != nil {
		placeholder := "$" + strconv.Itoa(len(args)+1)
		conditions = append(conditions, "e.status = "+placeholder)
		args = append(args, string(*filter.Status))
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
        SELECT e.id,
               e.company_id,
               e.employee_code,
               e.user_id,
               e.status,
               e.hired_at,
               e.terminated_at,
               e.created_at,
               e.updated_at,
               u.id,
               u.email,
               u.name,
               u.status,
               u.created_at,
               u.updated_at
          FROM employees e
          JOIN users u ON u.id = e.user_id` + whereClause + `
         ORDER BY e.created_at DESC, e.id DESC
         LIMIT ` + limitPlaceholder + `
        OFFSET ` + offsetPlaceholder + `
    `

	exec := pgdb.QueryerFromContext(ctx, r.pool)
	rows, err := exec.Query(ctx, query, args...)
	if err != nil {
		return nil, "", translateEmployeePgError(err)
	}
	defer rows.Close()

	employees := make([]*employee.Employee, 0, filter.Limit)
	for rows.Next() {
		emp, err := scanEmployee(rows)
		if err != nil {
			return nil, "", translateEmployeePgError(err)
		}
		employees = append(employees, emp)
	}

	if err := rows.Err(); err != nil {
		return nil, "", translateEmployeePgError(err)
	}

	var nextToken string
	if len(employees) == limitWithBuffer {
		employees = employees[:filter.Limit]
		nextToken = strconv.Itoa(filter.Offset + filter.Limit)
	}

	return employees, nextToken, nil
}

func scanEmployee(row pgx.Row) (*employee.Employee, error) {
	var (
		id           string
		companyID    string
		code         string
		userID       string
		status       string
		hiredAt      sql.NullTime
		terminatedAt sql.NullTime
		createdAt    time.Time
		updatedAt    time.Time
		userJoinedID string
		userEmail    string
		userName     string
		userStatus   string
		userCreated  time.Time
		userUpdated  time.Time
	)

	if err := row.Scan(
		&id,
		&companyID,
		&code,
		&userID,
		&status,
		&hiredAt,
		&terminatedAt,
		&createdAt,
		&updatedAt,
		&userJoinedID,
		&userEmail,
		&userName,
		&userStatus,
		&userCreated,
		&userUpdated,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, employee.ErrEmployeeNotFound
		}
		return nil, err
	}

	var hiredPtr *time.Time
	if hiredAt.Valid {
		t := hiredAt.Time.UTC()
		date := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		hiredPtr = &date
	}

	var terminatedPtr *time.Time
	if terminatedAt.Valid {
		t := terminatedAt.Time.UTC()
		date := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		terminatedPtr = &date
	}

	return &employee.Employee{
		ID:           id,
		CompanyID:    companyID,
		EmployeeCode: code,
		UserID:       userID,
		Status:       employee.Status(status),
		HiredAt:      hiredPtr,
		TerminatedAt: terminatedPtr,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		User: &employee.UserSnapshot{
			ID:        userJoinedID,
			Email:     userEmail,
			Name:      userName,
			Status:    userStatus,
			CreatedAt: userCreated,
			UpdatedAt: userUpdated,
		},
	}, nil
}

func translateEmployeePgError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return employee.ErrEmployeeNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case employeeUniqueViolationCode:
			return employee.ErrEmployeeCodeAlreadyExists
		case employeeForeignKeyViolationCode:
			switch pgErr.ConstraintName {
			case "employees_company_id_fkey":
				return employee.ErrCompanyNotFound
			case "employees_user_id_fkey":
				return employee.ErrUserNotFound
			default:
				return err
			}
		case employeeCheckViolationCode:
			return employee.ErrInvalidDateRange
		}
	}

	return err
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
