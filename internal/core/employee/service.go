package employee

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Clock は現在時刻を提供します。
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now().UTC()
}

// TransactionManager はトランザクション制御の抽象化です。
type TransactionManager interface {
	WithinReadOnly(ctx context.Context, fn func(context.Context) error) error
	WithinReadWrite(ctx context.Context, fn func(context.Context) error) error
}

type noopTransactionManager struct{}

func (noopTransactionManager) WithinReadOnly(ctx context.Context, fn func(context.Context) error) error {
	if fn == nil {
		return nil
	}
	return fn(ctx)
}

func (noopTransactionManager) WithinReadWrite(ctx context.Context, fn func(context.Context) error) error {
	if fn == nil {
		return nil
	}
	return fn(ctx)
}

const (
	defaultListPageSize = 50
	maxListPageSize     = 200
)

var employeeCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// Service は社員に関するユースケースをまとめます。
type Service struct {
	repo  Repository
	clock Clock
	tx    TransactionManager
}

// UseCase は社員ユースケースの公開インターフェースです。
type UseCase interface {
	CreateEmployee(ctx context.Context, in CreateEmployeeInput) (*Employee, error)
	GetEmployee(ctx context.Context, in GetEmployeeInput) (*Employee, error)
	ListEmployees(ctx context.Context, in ListEmployeesInput) (*ListEmployeesResult, error)
	UpdateEmployee(ctx context.Context, in UpdateEmployeeInput) (*Employee, error)
	DeleteEmployee(ctx context.Context, in DeleteEmployeeInput) error
}

// NewService は Service を生成します。
func NewService(repo Repository, clock Clock, tx TransactionManager) *Service {
	if clock == nil {
		clock = realClock{}
	}
	if tx == nil {
		tx = noopTransactionManager{}
	}
	return &Service{repo: repo, clock: clock, tx: tx}
}

// CreateEmployeeInput は社員作成時の入力です。
type CreateEmployeeInput struct {
	CompanyID    string
	EmployeeCode string
	UserID       string
	Status       *Status
	HiredAt      *time.Time
	TerminatedAt *time.Time
}

// UpdateEmployeeInput は社員更新時の入力です。
type UpdateEmployeeInput struct {
	ID              string
	EmployeeCode    *string
	UserID          *string
	Status          *Status
	HiredAt         *time.Time
	HiredAtSet      bool
	TerminatedAt    *time.Time
	TerminatedAtSet bool
}

// DeleteEmployeeInput は社員削除時の入力です。
type DeleteEmployeeInput struct {
	ID string
}

// GetEmployeeInput は社員取得時の入力です。
type GetEmployeeInput struct {
	ID string
}

// ListEmployeesInput は一覧取得時の入力です。
type ListEmployeesInput struct {
	CompanyID string
	PageSize  int
	PageToken string
	Status    *Status
}

// ListEmployeesResult は一覧取得結果を表します。
type ListEmployeesResult struct {
	Employees     []*Employee
	NextPageToken string
}

// CreateEmployee は新しい社員を作成します。
func (s *Service) CreateEmployee(ctx context.Context, in CreateEmployeeInput) (*Employee, error) {
	companyID, err := normalizeCompanyID(in.CompanyID)
	if err != nil {
		return nil, err
	}

	code, err := normalizeEmployeeCode(in.EmployeeCode)
	if err != nil {
		return nil, err
	}

	userID, err := normalizeUserID(in.UserID)
	if err != nil {
		return nil, err
	}

	hiredAt := normalizeDate(in.HiredAt)
	terminatedAt := normalizeDate(in.TerminatedAt)

	if err := validateEmploymentPeriod(hiredAt, terminatedAt); err != nil {
		return nil, err
	}

	status := StatusActive
	if in.Status != nil {
		if !isValidStatus(*in.Status) {
			return nil, ErrInvalidStatus
		}
		status = *in.Status
	}

	var created *Employee
	if err := s.tx.WithinReadWrite(ctx, func(txCtx context.Context) error {
		if err := s.ensureEmployeeCodeNotExists(txCtx, companyID, code); err != nil {
			return err
		}

		now := s.clock.Now()
		emp := &Employee{
			CompanyID:    companyID,
			EmployeeCode: code,
			UserID:       userID,
			Status:       status,
			HiredAt:      cloneTime(hiredAt),
			TerminatedAt: cloneTime(terminatedAt),
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		result, err := s.repo.Create(txCtx, emp)
		if err != nil {
			return err
		}

		created = result
		return nil
	}); err != nil {
		return nil, err
	}

	return created, nil
}

// UpdateEmployee は社員情報を更新します。
func (s *Service) UpdateEmployee(ctx context.Context, in UpdateEmployeeInput) (*Employee, error) {
	if strings.TrimSpace(in.ID) == "" {
		return nil, fmt.Errorf("id: %w", ErrInvalidID)
	}

	var updated *Employee
	if err := s.tx.WithinReadWrite(ctx, func(txCtx context.Context) error {
		existing, err := s.repo.FindByID(txCtx, in.ID)
		if err != nil {
			return err
		}

		if in.EmployeeCode != nil {
			code, err := normalizeEmployeeCode(*in.EmployeeCode)
			if err != nil {
				return err
			}
			if code != existing.EmployeeCode {
				if err := s.ensureEmployeeCodeNotExists(txCtx, existing.CompanyID, code); err != nil {
					return err
				}
				existing.EmployeeCode = code
			}
		}

		if in.UserID != nil {
			userID, err := normalizeUserID(*in.UserID)
			if err != nil {
				return err
			}
			existing.UserID = userID
		}

		if in.Status != nil {
			if !isValidStatus(*in.Status) {
				return ErrInvalidStatus
			}
			existing.Status = *in.Status
		}

		if in.HiredAtSet {
			existing.HiredAt = cloneTime(normalizeDate(in.HiredAt))
		}

		if in.TerminatedAtSet {
			existing.TerminatedAt = cloneTime(normalizeDate(in.TerminatedAt))
		}

		if err := validateEmploymentPeriod(existing.HiredAt, existing.TerminatedAt); err != nil {
			return err
		}

		existing.UpdatedAt = s.clock.Now()

		result, err := s.repo.Update(txCtx, existing)
		if err != nil {
			return err
		}

		updated = result
		return nil
	}); err != nil {
		return nil, err
	}

	return updated, nil
}

// DeleteEmployee は社員を削除します。
func (s *Service) DeleteEmployee(ctx context.Context, in DeleteEmployeeInput) error {
	if strings.TrimSpace(in.ID) == "" {
		return fmt.Errorf("id: %w", ErrInvalidID)
	}

	return s.tx.WithinReadWrite(ctx, func(txCtx context.Context) error {
		return s.repo.Delete(txCtx, in.ID)
	})
}

// GetEmployee は社員を取得します。
func (s *Service) GetEmployee(ctx context.Context, in GetEmployeeInput) (*Employee, error) {
	if strings.TrimSpace(in.ID) == "" {
		return nil, fmt.Errorf("id: %w", ErrInvalidID)
	}

	var result *Employee
	if err := s.tx.WithinReadOnly(ctx, func(txCtx context.Context) error {
		found, err := s.repo.FindByID(txCtx, in.ID)
		if err != nil {
			return err
		}
		result = found
		return nil
	}); err != nil {
		return nil, err
	}

	return result, nil
}

// ListEmployees は社員の一覧を取得します。
func (s *Service) ListEmployees(ctx context.Context, in ListEmployeesInput) (*ListEmployeesResult, error) {
	companyID, err := normalizeCompanyID(in.CompanyID)
	if err != nil {
		return nil, err
	}

	limit, err := normalizePageSize(in.PageSize)
	if err != nil {
		return nil, err
	}

	offset, err := parsePageToken(in.PageToken)
	if err != nil {
		return nil, err
	}

	var statusPtr *Status
	if in.Status != nil {
		if !isValidStatus(*in.Status) {
			return nil, ErrInvalidStatus
		}
		status := *in.Status
		statusPtr = &status
	}

	var (
		employees []*Employee
		nextToken string
	)

	if err := s.tx.WithinReadOnly(ctx, func(txCtx context.Context) error {
		resultEmployees, token, err := s.repo.List(txCtx, ListEmployeesFilter{
			CompanyID: companyID,
			Status:    statusPtr,
			Limit:     limit,
			Offset:    offset,
		})
		if err != nil {
			return err
		}
		employees = resultEmployees
		nextToken = token
		return nil
	}); err != nil {
		return nil, err
	}

	return &ListEmployeesResult{Employees: employees, NextPageToken: nextToken}, nil
}

func (s *Service) ensureEmployeeCodeNotExists(ctx context.Context, companyID, code string) error {
	emp, err := s.repo.FindByCompanyAndCode(ctx, companyID, code)
	if err != nil && !errors.Is(err, ErrEmployeeNotFound) {
		return err
	}
	if emp != nil {
		return ErrEmployeeCodeAlreadyExists
	}
	return nil
}

func normalizeCompanyID(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ErrInvalidCompanyID
	}
	return trimmed, nil
}

func normalizeEmployeeCode(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ErrInvalidEmployeeCode
	}

	lower := strings.ToLower(trimmed)
	if !employeeCodePattern.MatchString(lower) {
		return "", ErrInvalidEmployeeCode
	}
	return lower, nil
}

func normalizeUserID(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ErrInvalidUserID
	}
	return trimmed, nil
}

func normalizeDate(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}

	normalized := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return &normalized
}

func validateEmploymentPeriod(hiredAt, terminatedAt *time.Time) error {
	if hiredAt == nil || terminatedAt == nil {
		return nil
	}
	if terminatedAt.Before(*hiredAt) {
		return ErrInvalidDateRange
	}
	return nil
}

func cloneTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	clone := *t
	return &clone
}

func isValidStatus(status Status) bool {
	switch status {
	case StatusActive, StatusInactive:
		return true
	default:
		return false
	}
}

func normalizePageSize(pageSize int) (int, error) {
	if pageSize <= 0 {
		return defaultListPageSize, nil
	}
	if pageSize > maxListPageSize {
		return 0, ErrInvalidPageSize
	}
	return pageSize, nil
}

func parsePageToken(token string) (int, error) {
	if strings.TrimSpace(token) == "" {
		return 0, nil
	}

	offset, err := strconv.Atoi(token)
	if err != nil || offset < 0 {
		return 0, ErrInvalidPageToken
	}

	return offset, nil
}
