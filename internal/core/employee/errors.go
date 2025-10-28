package employee

import "errors"

var (
	ErrInvalidID                 = errors.New("employee: invalid id")
	ErrInvalidCompanyID          = errors.New("employee: invalid company id")
	ErrInvalidEmployeeCode       = errors.New("employee: invalid employee code")
	ErrInvalidEmail              = errors.New("employee: invalid email")
	ErrInvalidLastName           = errors.New("employee: invalid last name")
	ErrInvalidFirstName          = errors.New("employee: invalid first name")
	ErrInvalidStatus             = errors.New("employee: invalid status")
	ErrInvalidPageSize           = errors.New("employee: invalid page size")
	ErrInvalidPageToken          = errors.New("employee: invalid page token")
	ErrInvalidDateRange          = errors.New("employee: invalid employment period")
	ErrEmployeeNotFound          = errors.New("employee: not found")
	ErrCompanyNotFound           = errors.New("employee: company not found")
	ErrEmployeeCodeAlreadyExists = errors.New("employee: employee code already exists")
)
