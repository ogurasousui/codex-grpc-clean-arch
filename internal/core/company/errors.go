package company

import "errors"

var (
	// ErrCompanyNotFound は会社が存在しない場合に返却されます。
	ErrCompanyNotFound = errors.New("company not found")
	// ErrCodeAlreadyExists はコード重複時に返却されます。
	ErrCodeAlreadyExists = errors.New("code already exists")
	// ErrInvalidName は会社名が不正な場合に返却されます。
	ErrInvalidName = errors.New("invalid name")
	// ErrInvalidCode は会社コードが不正な場合に返却されます。
	ErrInvalidCode = errors.New("invalid code")
	// ErrInvalidStatus はステータスが不正な場合に返却されます。
	ErrInvalidStatus = errors.New("invalid status")
	// ErrInvalidID は ID が不正な場合に返却されます。
	ErrInvalidID = errors.New("invalid id")
	// ErrInvalidPageSize は一覧取得時のページサイズが不正な場合に返却されます。
	ErrInvalidPageSize = errors.New("invalid page size")
	// ErrInvalidPageToken は一覧取得時のページトークンが不正な場合に返却されます。
	ErrInvalidPageToken = errors.New("invalid page token")
)
