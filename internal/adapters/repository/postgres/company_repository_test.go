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
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/core/company"
	pgxmock "github.com/pashagolub/pgxmock/v4"
)

type stubCompanyRow struct {
	scanFn func(dest ...interface{}) error
}

func (s stubCompanyRow) Scan(dest ...interface{}) error {
	return s.scanFn(dest...)
}

func TestScanCompany_Success(t *testing.T) {
	t.Parallel()

	desc := "Sample"
	createdAt := time.Now().UTC()
	updatedAt := createdAt.Add(time.Minute)

	row := stubCompanyRow{scanFn: func(dest ...interface{}) error {
		if len(dest) != 7 {
			return errors.New("unexpected dest length")
		}
		*(dest[0].(*string)) = "company-1"
		*(dest[1].(*string)) = "Company"
		*(dest[2].(*string)) = "company"
		*(dest[3].(*string)) = string(company.StatusActive)

		d := dest[4].(*sql.NullString)
		d.String = desc
		d.Valid = true

		*(dest[5].(*time.Time)) = createdAt
		*(dest[6].(*time.Time)) = updatedAt
		return nil
	}}

	c, err := scanCompany(row)
	if err != nil {
		t.Fatalf("scanCompany returned error: %v", err)
	}

	if c.Description == nil || *c.Description != desc {
		t.Fatalf("expected description %s, got %+v", desc, c.Description)
	}
}

func TestScanCompany_NoRows(t *testing.T) {
	t.Parallel()

	row := stubCompanyRow{scanFn: func(dest ...interface{}) error {
		return pgx.ErrNoRows
	}}

	_, err := scanCompany(row)
	if !errors.Is(err, company.ErrCompanyNotFound) {
		t.Fatalf("expected ErrCompanyNotFound, got %v", err)
	}
}

func TestTranslateCompanyPgError(t *testing.T) {
	t.Parallel()

	pgErr := &pgconn.PgError{Code: companyUniqueViolationCode}
	if !errors.Is(translateCompanyPgError(pgErr), company.ErrCodeAlreadyExists) {
		t.Fatalf("expected code already exists error mapping")
	}

	otherErr := errors.New("random")
	if translateCompanyPgError(otherErr) != otherErr {
		t.Fatalf("unexpected translation for generic error")
	}
}

func TestCompanyRepository_List_WithNextToken(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewCompanyRepository(mock)

	query := regexp.QuoteMeta(`
        SELECT id, name, code, status, description, created_at, updated_at
          FROM companies
         ORDER BY created_at DESC, id DESC
         LIMIT $1
        OFFSET $2
    `)

	now := time.Now().UTC()
	rows := pgxmock.NewRows([]string{"id", "name", "code", "status", "description", "created_at", "updated_at"}).
		AddRow("company-1", "Company1", "company-1", string(company.StatusActive), nil, now, now).
		AddRow("company-2", "Company2", "company-2", string(company.StatusActive), nil, now, now).
		AddRow("company-3", "Company3", "company-3", string(company.StatusInactive), nil, now, now)

	mock.ExpectQuery(query).
		WithArgs(3, 0).
		WillReturnRows(rows)

	companies, nextToken, err := repo.List(context.Background(), company.ListCompaniesFilter{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(companies) != 2 {
		t.Fatalf("expected 2 companies, got %d", len(companies))
	}

	if nextToken != "2" {
		t.Fatalf("expected next token '2', got %s", nextToken)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCompanyRepository_List_WithStatusFilter(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewCompanyRepository(mock)
	inactive := company.StatusInactive

	query := regexp.QuoteMeta(`
        SELECT id, name, code, status, description, created_at, updated_at
          FROM companies WHERE status = $1
         ORDER BY created_at DESC, id DESC
         LIMIT $2
        OFFSET $3
    `)

	now := time.Now().UTC()
	rows := pgxmock.NewRows([]string{"id", "name", "code", "status", "description", "created_at", "updated_at"}).
		AddRow("company-5", "Inactive", "inactive", string(company.StatusInactive), nil, now, now)

	mock.ExpectQuery(query).
		WithArgs(inactive, 3, 0).
		WillReturnRows(rows)

	companies, nextToken, err := repo.List(context.Background(), company.ListCompaniesFilter{Limit: 2, Offset: 0, Status: &inactive})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(companies) != 1 {
		t.Fatalf("expected 1 company, got %d", len(companies))
	}

	if nextToken != "" {
		t.Fatalf("expected empty next token, got %s", nextToken)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCompanyRepository_List_InvalidArguments(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewCompanyRepository(mock)

	if _, _, err := repo.List(context.Background(), company.ListCompaniesFilter{Limit: 0, Offset: 0}); !errors.Is(err, company.ErrInvalidPageSize) {
		t.Fatalf("expected ErrInvalidPageSize, got %v", err)
	}

	if _, _, err := repo.List(context.Background(), company.ListCompaniesFilter{Limit: 1, Offset: -1}); !errors.Is(err, company.ErrInvalidPageToken) {
		t.Fatalf("expected ErrInvalidPageToken, got %v", err)
	}
}
