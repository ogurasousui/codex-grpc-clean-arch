package employee

import "context"

// Repository は社員永続化の抽象です。
type Repository interface {
	Create(ctx context.Context, employee *Employee) (*Employee, error)
	Update(ctx context.Context, employee *Employee) (*Employee, error)
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (*Employee, error)
	FindByCompanyAndCode(ctx context.Context, companyID, employeeCode string) (*Employee, error)
	List(ctx context.Context, filter ListEmployeesFilter) ([]*Employee, string, error)
}

// ListEmployeesFilter は一覧取得用フィルタです。
type ListEmployeesFilter struct {
	CompanyID string
	Status    *Status
	Limit     int
	Offset    int
}
