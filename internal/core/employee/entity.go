package employee

import "time"

// Status は社員の状態を表します。
type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

// Employee は社員エンティティです。
type Employee struct {
	ID           string
	CompanyID    string
	EmployeeCode string
	Email        *string
	LastName     string
	FirstName    string
	Status       Status
	HiredAt      *time.Time
	TerminatedAt *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
