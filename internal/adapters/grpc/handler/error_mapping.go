package handler

import (
	"errors"

	"github.com/ogurasousui/codex-grpc-clean-arch/internal/core/company"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/core/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func toStatusError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, user.ErrInvalidEmail),
		errors.Is(err, user.ErrInvalidName),
		errors.Is(err, user.ErrInvalidStatus),
		errors.Is(err, user.ErrInvalidID),
		errors.Is(err, user.ErrInvalidPageSize),
		errors.Is(err, user.ErrInvalidPageToken),
		errors.Is(err, company.ErrInvalidName),
		errors.Is(err, company.ErrInvalidCode),
		errors.Is(err, company.ErrInvalidStatus),
		errors.Is(err, company.ErrInvalidID),
		errors.Is(err, company.ErrInvalidPageSize),
		errors.Is(err, company.ErrInvalidPageToken):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, user.ErrEmailAlreadyExists), errors.Is(err, company.ErrCodeAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, user.ErrUserNotFound), errors.Is(err, company.ErrCompanyNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
