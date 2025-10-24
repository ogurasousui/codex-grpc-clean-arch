package company

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

var codePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// Service は会社に関するユースケースをまとめます。
type Service struct {
	repo  Repository
	clock Clock
	tx    TransactionManager
}

// UseCase は会社ユースケースの公開インターフェースです。
type UseCase interface {
	CreateCompany(ctx context.Context, in CreateCompanyInput) (*Company, error)
	GetCompany(ctx context.Context, in GetCompanyInput) (*Company, error)
	ListCompanies(ctx context.Context, in ListCompaniesInput) (*ListCompaniesResult, error)
	UpdateCompany(ctx context.Context, in UpdateCompanyInput) (*Company, error)
	DeleteCompany(ctx context.Context, in DeleteCompanyInput) error
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

// CreateCompanyInput は会社作成時の入力です。
type CreateCompanyInput struct {
	Name        string
	Code        string
	Description *string
}

// UpdateCompanyInput は会社更新時の入力です。
type UpdateCompanyInput struct {
	ID          string
	Name        *string
	Code        *string
	Status      *Status
	Description *string
}

// DeleteCompanyInput は会社削除時の入力です。
type DeleteCompanyInput struct {
	ID string
}

// GetCompanyInput は会社取得時の入力です。
type GetCompanyInput struct {
	ID string
}

// ListCompaniesInput は一覧取得時の入力です。
type ListCompaniesInput struct {
	PageSize  int
	PageToken string
	Status    *Status
}

// ListCompaniesResult は一覧取得結果を表します。
type ListCompaniesResult struct {
	Companies     []*Company
	NextPageToken string
}

// CreateCompany は新しい会社を作成します。
func (s *Service) CreateCompany(ctx context.Context, in CreateCompanyInput) (*Company, error) {
	name, err := normalizeName(in.Name)
	if err != nil {
		return nil, err
	}

	code, err := normalizeCode(in.Code)
	if err != nil {
		return nil, err
	}

	description := normalizeDescription(in.Description)

	var created *Company
	if err := s.tx.WithinReadWrite(ctx, func(txCtx context.Context) error {
		if err := s.ensureCodeNotExists(txCtx, code); err != nil {
			return err
		}

		now := s.clock.Now()
		company := &Company{
			Name:        name,
			Code:        code,
			Status:      StatusActive,
			Description: description,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		result, err := s.repo.Create(txCtx, company)
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

// UpdateCompany は会社情報を更新します。
func (s *Service) UpdateCompany(ctx context.Context, in UpdateCompanyInput) (*Company, error) {
	if strings.TrimSpace(in.ID) == "" {
		return nil, fmt.Errorf("id: %w", ErrInvalidID)
	}

	var updated *Company
	if err := s.tx.WithinReadWrite(ctx, func(txCtx context.Context) error {
		existing, err := s.repo.FindByID(txCtx, in.ID)
		if err != nil {
			return err
		}

		if in.Name != nil {
			name, err := normalizeName(*in.Name)
			if err != nil {
				return err
			}
			existing.Name = name
		}

		if in.Code != nil {
			code, err := normalizeCode(*in.Code)
			if err != nil {
				return err
			}
			if code != existing.Code {
				if err := s.ensureCodeNotExists(txCtx, code); err != nil {
					return err
				}
				existing.Code = code
			}
		}

		if in.Status != nil {
			if !isValidStatus(*in.Status) {
				return ErrInvalidStatus
			}
			existing.Status = *in.Status
		}

		if in.Description != nil {
			existing.Description = normalizeDescription(in.Description)
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

// DeleteCompany は会社を削除します。
func (s *Service) DeleteCompany(ctx context.Context, in DeleteCompanyInput) error {
	if strings.TrimSpace(in.ID) == "" {
		return fmt.Errorf("id: %w", ErrInvalidID)
	}

	return s.tx.WithinReadWrite(ctx, func(txCtx context.Context) error {
		return s.repo.Delete(txCtx, in.ID)
	})
}

// GetCompany は ID で会社を取得します。
func (s *Service) GetCompany(ctx context.Context, in GetCompanyInput) (*Company, error) {
	if strings.TrimSpace(in.ID) == "" {
		return nil, fmt.Errorf("id: %w", ErrInvalidID)
	}

	var company *Company
	if err := s.tx.WithinReadOnly(ctx, func(txCtx context.Context) error {
		result, err := s.repo.FindByID(txCtx, in.ID)
		if err != nil {
			return err
		}
		company = result
		return nil
	}); err != nil {
		return nil, err
	}

	return company, nil
}

// ListCompanies は会社の一覧を取得します。
func (s *Service) ListCompanies(ctx context.Context, in ListCompaniesInput) (*ListCompaniesResult, error) {
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
		companies []*Company
		nextToken string
	)

	if err := s.tx.WithinReadOnly(ctx, func(txCtx context.Context) error {
		resultCompanies, token, err := s.repo.List(txCtx, ListCompaniesFilter{
			Limit:  limit,
			Offset: offset,
			Status: statusPtr,
		})
		if err != nil {
			return err
		}
		companies = resultCompanies
		nextToken = token
		return nil
	}); err != nil {
		return nil, err
	}

	return &ListCompaniesResult{
		Companies:     companies,
		NextPageToken: nextToken,
	}, nil
}

func (s *Service) ensureCodeNotExists(ctx context.Context, code string) error {
	company, err := s.repo.FindByCode(ctx, code)
	if err != nil && !errors.Is(err, ErrCompanyNotFound) {
		return err
	}
	if company != nil {
		return ErrCodeAlreadyExists
	}
	return nil
}

func normalizeName(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ErrInvalidName
	}
	return trimmed, nil
}

func normalizeCode(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ErrInvalidCode
	}

	lower := strings.ToLower(trimmed)
	if !codePattern.MatchString(lower) {
		return "", ErrInvalidCode
	}

	return lower, nil
}

func normalizeDescription(raw *string) *string {
	if raw == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil
	}

	desc := trimmed
	return &desc
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
