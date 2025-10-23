package user

import "context"

// Repository はユーザーエンティティの永続化を行うインターフェースです。
type Repository interface {
	Create(ctx context.Context, user *User) (*User, error)
	Update(ctx context.Context, user *User) (*User, error)
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	List(ctx context.Context, filter ListUsersFilter) ([]*User, string, error)
}

// ListUsersFilter は一覧取得時の検索条件を表します。
type ListUsersFilter struct {
	Limit  int
	Offset int
	Status *Status
}
