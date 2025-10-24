package company

import "time"

// Status は会社の状態を表します。
type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

// Company は会社エンティティです。
type Company struct {
	ID          string
	Name        string
	Code        string
	Status      Status
	Description *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
