package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
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

// Service はユーザーに関するユースケースをまとめます。
type Service struct {
	repo  Repository
	clock Clock
	tx    TransactionManager
}

// UseCase はユーザーユースケースの公開インターフェースです。
type UseCase interface {
	CreateUser(ctx context.Context, in CreateUserInput) (*User, error)
	UpdateUser(ctx context.Context, in UpdateUserInput) (*User, error)
	DeleteUser(ctx context.Context, in DeleteUserInput) error
	GetUser(ctx context.Context, in GetUserInput) (*User, error)
	ListUsers(ctx context.Context, in ListUsersInput) (*ListUsersResult, error)
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

// CreateUserInput はユーザー作成時の入力です。
type CreateUserInput struct {
	Email string
	Name  string
}

// UpdateUserInput はユーザー更新時の入力です。
type UpdateUserInput struct {
	ID     string
	Name   *string
	Status *Status
}

// DeleteUserInput はユーザー削除時の入力です。
type DeleteUserInput struct {
	ID string
}

// GetUserInput はユーザー取得時の入力です。
type GetUserInput struct {
	ID string
}

// ListUsersInput は一覧取得時の入力です。
type ListUsersInput struct {
	PageSize  int
	PageToken string
	Status    *Status
}

// ListUsersResult は一覧取得結果を表します。
type ListUsersResult struct {
	Users         []*User
	NextPageToken string
}

// CreateUser は新しいユーザーを作成します。
func (s *Service) CreateUser(ctx context.Context, in CreateUserInput) (*User, error) {
	email, err := normalizeEmail(in.Email)
	if err != nil {
		return nil, ErrInvalidEmail
	}

	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrInvalidName
	}

	var created *User
	if err := s.tx.WithinReadWrite(ctx, func(txCtx context.Context) error {
		if err := s.ensureEmailNotExists(txCtx, email); err != nil {
			return err
		}

		now := s.clock.Now()
		u := &User{
			Email:     email,
			Name:      name,
			Status:    StatusActive,
			CreatedAt: now,
			UpdatedAt: now,
		}

		result, err := s.repo.Create(txCtx, u)
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

// UpdateUser はユーザー情報を更新します。
func (s *Service) UpdateUser(ctx context.Context, in UpdateUserInput) (*User, error) {
	if strings.TrimSpace(in.ID) == "" {
		return nil, fmt.Errorf("id: %w", ErrInvalidID)
	}

	var updated *User
	if err := s.tx.WithinReadWrite(ctx, func(txCtx context.Context) error {
		existing, err := s.repo.FindByID(txCtx, in.ID)
		if err != nil {
			return err
		}

		if in.Name != nil {
			updatedName := strings.TrimSpace(*in.Name)
			if updatedName == "" {
				return ErrInvalidName
			}
			existing.Name = updatedName
		}

		if in.Status != nil {
			if !isValidStatus(*in.Status) {
				return ErrInvalidStatus
			}
			existing.Status = *in.Status
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

// DeleteUser はユーザーを削除します。
func (s *Service) DeleteUser(ctx context.Context, in DeleteUserInput) error {
	if strings.TrimSpace(in.ID) == "" {
		return fmt.Errorf("id: %w", ErrInvalidID)
	}
	return s.tx.WithinReadWrite(ctx, func(txCtx context.Context) error {
		return s.repo.Delete(txCtx, in.ID)
	})
}

// GetUser は ID でユーザーを取得します。
func (s *Service) GetUser(ctx context.Context, in GetUserInput) (*User, error) {
	if strings.TrimSpace(in.ID) == "" {
		return nil, fmt.Errorf("id: %w", ErrInvalidID)
	}
	var found *User
	if err := s.tx.WithinReadOnly(ctx, func(txCtx context.Context) error {
		result, err := s.repo.FindByID(txCtx, in.ID)
		if err != nil {
			return err
		}
		found = result
		return nil
	}); err != nil {
		return nil, err
	}
	return found, nil
}

// ListUsers はユーザーの一覧を取得します。
func (s *Service) ListUsers(ctx context.Context, in ListUsersInput) (*ListUsersResult, error) {
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
		users     []*User
		nextToken string
	)

	if err := s.tx.WithinReadOnly(ctx, func(txCtx context.Context) error {
		resultUsers, token, err := s.repo.List(txCtx, ListUsersFilter{
			Limit:  limit,
			Offset: offset,
			Status: statusPtr,
		})
		if err != nil {
			return err
		}
		users = resultUsers
		nextToken = token
		return nil
	}); err != nil {
		return nil, err
	}

	return &ListUsersResult{
		Users:         users,
		NextPageToken: nextToken,
	}, nil
}

func (s *Service) ensureEmailNotExists(ctx context.Context, email string) error {
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return err
	}
	if user != nil {
		return ErrEmailAlreadyExists
	}
	return nil
}

func normalizeEmail(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ErrInvalidEmail
	}

	addr, err := mail.ParseAddress(trimmed)
	if err != nil {
		return "", ErrInvalidEmail
	}

	return strings.ToLower(addr.Address), nil
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
