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

	row := stubEmployeeRow{scanFn: func(dest ...interface{}) error {
		if len(dest) != 11 {
			return errors.New("unexpected dest length")
		}
		*(dest[0].(*string)) = "emp-1"
		*(dest[1].(*string)) = "company-1"
		*(dest[2].(*string)) = "emp-001"

		emailDest := dest[3].(*sql.NullString)
		emailDest.String = email
		emailDest.Valid = true

		*(dest[4].(*string)) = "Yamada"
		*(dest[5].(*string)) = "Taro"
		*(dest[6].(*string)) = string(employee.StatusActive)

		hiredDest := dest[7].(*sql.NullTime)
		hiredDest.Time = hired
		hiredDest.Valid = true

		termDest := dest[8].(*sql.NullTime)
		termDest.Time = terminated
		termDest.Valid = true

		*(dest[9].(*time.Time)) = createdAt
		*(dest[10].(*time.Time)) = updatedAt
		return nil
	}}

	emp, err := scanEmployee(row)
	if err != nil {
		t.Fatalf("scanEmployee returned error: %v", err)
	}

	if emp.Email == nil || *emp.Email != email {
		t.Fatalf("expected email %s, got %+v", email, emp.Email)
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

	fkErr := &pgconn.PgError{Code: employeeForeignKeyViolationCode}
	if !errors.Is(translateEmployeePgError(fkErr), employee.ErrCompanyNotFound) {
		t.Fatalf("expected fk violation to map to ErrCompanyNotFound")
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
        SELECT id, company_id, employee_code, email, last_name, first_name, status, hired_at, terminated_at, created_at, updated_at
          FROM employees WHERE company_id = $1 AND status = $2
         ORDER BY created_at DESC, id DESC
         LIMIT $3
        OFFSET $4
    `)

	now := time.Now().UTC()
	rows := pgxmock.NewRows([]string{"id", "company_id", "employee_code", "email", "last_name", "first_name", "status", "hired_at", "terminated_at", "created_at", "updated_at"}).
		AddRow("emp-1", "company-1", "emp-1", nil, "Yamada", "Taro", string(employee.StatusActive), nil, nil, now, now).
		AddRow("emp-2", "company-1", "emp-2", nil, "Sato", "Hanako", string(employee.StatusActive), nil, nil, now, now).
		AddRow("emp-3", "company-1", "emp-3", nil, "Suzuki", "Ichiro", string(employee.StatusInactive), nil, nil, now, now)

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
