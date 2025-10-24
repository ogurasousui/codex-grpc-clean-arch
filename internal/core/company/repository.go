package company

import "context"

// Repository は会社エンティティの永続化を行うインターフェースです。
type Repository interface {
	Create(ctx context.Context, company *Company) (*Company, error)
	Update(ctx context.Context, company *Company) (*Company, error)
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (*Company, error)
	FindByCode(ctx context.Context, code string) (*Company, error)
	List(ctx context.Context, filter ListCompaniesFilter) ([]*Company, string, error)
}

// ListCompaniesFilter は一覧取得時の検索条件を表します。
type ListCompaniesFilter struct {
	Limit  int
	Offset int
	Status *Status
}
