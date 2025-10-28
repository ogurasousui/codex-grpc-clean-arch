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
	UserID       string
	Status       Status
	HiredAt      *time.Time
	TerminatedAt *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	User         *UserSnapshot
}

// UserSnapshot は社員に紐づくユーザー情報のスナップショットです。
type UserSnapshot struct {
	ID        string
	Email     string
	Name      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}
