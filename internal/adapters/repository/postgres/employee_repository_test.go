package postgres

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/core/employee"
	pgxmock "github.com/pashagolub/pgxmock/v4"
)

type stubEmployeeRow struct {
	scanFn func(dest ...interface{}) error
}

func (s stubEmployeeRow) Scan(dest ...interface{}) error {
	return s.scanFn(dest...)
}

func TestScanEmployee_Success(t *testing.T) {
	t.Parallel()

	email := "user@example.com"
	hired := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	terminated := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	createdAt := time.Now().UTC()
	updatedAt := createdAt.Add(time.Minute)
	userCreated := createdAt.Add(-time.Hour)
	userUpdated := updatedAt

	row := stubEmployeeRow{scanFn: func(dest ...interface{}) error {
		if len(dest) != 15 {
			return errors.New("unexpected dest length")
		}
		*(dest[0].(*string)) = "emp-1"
		*(dest[1].(*string)) = "company-1"
		*(dest[2].(*string)) = "emp-001"
		*(dest[3].(*string)) = "user-1"
		*(dest[4].(*string)) = string(employee.StatusActive)

		hiredDest := dest[5].(*sql.NullTime)
		hiredDest.Time = hired
		hiredDest.Valid = true

		termDest := dest[6].(*sql.NullTime)
		termDest.Time = terminated
		termDest.Valid = true

		*(dest[7].(*time.Time)) = createdAt
		*(dest[8].(*time.Time)) = updatedAt

		*(dest[9].(*string)) = "user-1"
		*(dest[10].(*string)) = email
		*(dest[11].(*string)) = "Taro Yamada"
		*(dest[12].(*string)) = "active"
		*(dest[13].(*time.Time)) = userCreated
		*(dest[14].(*time.Time)) = userUpdated
		return nil
	}}

	emp, err := scanEmployee(row)
	if err != nil {
		t.Fatalf("scanEmployee returned error: %v", err)
	}

	if emp.UserID != "user-1" {
		t.Fatalf("expected user id user-1, got %s", emp.UserID)
	}
	if emp.User == nil || emp.User.Email != email {
		t.Fatalf("expected user snapshot email %s", email)
	}
	if emp.HiredAt == nil || !emp.HiredAt.Equal(hired) {
		t.Fatalf("expected hired date, got %+v", emp.HiredAt)
	}
	if emp.TerminatedAt == nil || !emp.TerminatedAt.Equal(terminated) {
		t.Fatalf("expected terminated date, got %+v", emp.TerminatedAt)
	}
}

func TestScanEmployee_NoRows(t *testing.T) {
	t.Parallel()

	row := stubEmployeeRow{scanFn: func(dest ...interface{}) error {
		return pgx.ErrNoRows
	}}

	_, err := scanEmployee(row)
	if !errors.Is(err, employee.ErrEmployeeNotFound) {
		t.Fatalf("expected ErrEmployeeNotFound, got %v", err)
	}
}

func TestTranslateEmployeePgError(t *testing.T) {
	t.Parallel()

	uniqueErr := &pgconn.PgError{Code: employeeUniqueViolationCode}
	if !errors.Is(translateEmployeePgError(uniqueErr), employee.ErrEmployeeCodeAlreadyExists) {
		t.Fatalf("expected unique violation to map to ErrEmployeeCodeAlreadyExists")
	}

	fkCompanyErr := &pgconn.PgError{Code: employeeForeignKeyViolationCode, ConstraintName: "employees_company_id_fkey"}
	if !errors.Is(translateEmployeePgError(fkCompanyErr), employee.ErrCompanyNotFound) {
		t.Fatalf("expected fk violation to map to ErrCompanyNotFound")
	}

	fkUserErr := &pgconn.PgError{Code: employeeForeignKeyViolationCode, ConstraintName: "employees_user_id_fkey"}
	if !errors.Is(translateEmployeePgError(fkUserErr), employee.ErrUserNotFound) {
		t.Fatalf("expected user fk violation to map to ErrUserNotFound")
	}

	checkErr := &pgconn.PgError{Code: employeeCheckViolationCode}
	if !errors.Is(translateEmployeePgError(checkErr), employee.ErrInvalidDateRange) {
		t.Fatalf("expected check violation to map to ErrInvalidDateRange")
	}

	other := errors.New("other")
	if translateEmployeePgError(other) != other {
		t.Fatalf("unexpected translation for generic error")
	}
}

func TestEmployeeRepository_List_WithFilters(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewEmployeeRepository(mock)
	status := employee.StatusActive

	query := regexp.QuoteMeta(`
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
          JOIN users u ON u.id = e.user_id WHERE e.company_id = $1 AND e.status = $2
         ORDER BY e.created_at DESC, e.id DESC
         LIMIT $3
        OFFSET $4
    `)

	now := time.Now().UTC()
	rows := pgxmock.NewRows([]string{"id", "company_id", "employee_code", "user_id", "status", "hired_at", "terminated_at", "created_at", "updated_at", "user_id_join", "user_email", "user_name", "user_status", "user_created_at", "user_updated_at"}).
		AddRow("emp-1", "company-1", "emp-1", "user-1", string(employee.StatusActive), nil, nil, now, now, "user-1", "user1@example.com", "User One", "active", now, now).
		AddRow("emp-2", "company-1", "emp-2", "user-2", string(employee.StatusActive), nil, nil, now, now, "user-2", "user2@example.com", "User Two", "active", now, now).
		AddRow("emp-3", "company-1", "emp-3", "user-3", string(employee.StatusInactive), nil, nil, now, now, "user-3", "user3@example.com", "User Three", "inactive", now, now)

	mock.ExpectQuery(query).
		WithArgs("company-1", string(status), 3, 0).
		WillReturnRows(rows)

	employees, nextToken, err := repo.List(context.Background(), employee.ListEmployeesFilter{
		CompanyID: "company-1",
		Status:    &status,
		Limit:     2,
		Offset:    0,
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(employees) != 2 {
		t.Fatalf("expected 2 employees, got %d", len(employees))
	}
	if nextToken != "2" {
		t.Fatalf("expected next token '2', got %s", nextToken)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
