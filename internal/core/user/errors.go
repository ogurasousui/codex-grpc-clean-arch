package user

import "errors"

var (
	// ErrUserNotFound はユーザーが存在しない場合に返却されます。
	ErrUserNotFound = errors.New("user not found")
	// ErrEmailAlreadyExists はメールアドレス重複時に返却されます。
	ErrEmailAlreadyExists = errors.New("email already exists")
	// ErrInvalidEmail はメールアドレスが不正な場合に返却されます。
	ErrInvalidEmail = errors.New("invalid email")
	// ErrInvalidName は名前が不正な場合に返却されます。
	ErrInvalidName = errors.New("invalid name")
	// ErrInvalidStatus はステータスが不正な場合に返却されます。
	ErrInvalidStatus = errors.New("invalid status")
	// ErrInvalidID はIDが不正な場合に返却されます。
	ErrInvalidID = errors.New("invalid id")
)
