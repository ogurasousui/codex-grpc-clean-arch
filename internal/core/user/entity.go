package user

import "time"

// Status はユーザーの状態を表します。
type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

// User はユーザーエンティティです。
type User struct {
	ID        string
	Email     string
	Name      string
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
}
