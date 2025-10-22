package handler

import (
	"context"
	"errors"

	userpb "github.com/ogurasousui/codex-grpc-clean-arch/internal/adapters/grpc/gen/user/v1"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/core/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// UserGrpcHandler は UserService の gRPC 実装です。
type UserGrpcHandler struct {
	svc user.UseCase
	userpb.UnimplementedUserServiceServer
}

// NewUserGrpcHandler は UserGrpcHandler を生成します。
func NewUserGrpcHandler(svc user.UseCase) *UserGrpcHandler {
	return &UserGrpcHandler{svc: svc}
}

// CreateUser はユーザーを作成します。
func (h *UserGrpcHandler) CreateUser(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.CreateUserResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	created, err := h.svc.CreateUser(ctx, user.CreateUserInput{
		Email: req.GetEmail(),
		Name:  req.GetName(),
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	return &userpb.CreateUserResponse{User: toProtoUser(created)}, nil
}

// UpdateUser はユーザー情報を更新します。
func (h *UserGrpcHandler) UpdateUser(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.UpdateUserResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	var namePtr *string
	if req.Name != nil {
		value := req.Name.GetValue()
		namePtr = &value
	}

	var statusPtr *user.Status
	if req.GetStatus() != userpb.UserStatus_USER_STATUS_UNSPECIFIED {
		domainStatus, err := toDomainStatus(req.GetStatus())
		if err != nil {
			return nil, toStatusError(err)
		}
		statusPtr = &domainStatus
	}

	updated, err := h.svc.UpdateUser(ctx, user.UpdateUserInput{
		ID:     req.GetId(),
		Name:   namePtr,
		Status: statusPtr,
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	return &userpb.UpdateUserResponse{User: toProtoUser(updated)}, nil
}

// DeleteUser はユーザーを削除します。
func (h *UserGrpcHandler) DeleteUser(ctx context.Context, req *userpb.DeleteUserRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if err := h.svc.DeleteUser(ctx, user.DeleteUserInput{ID: req.GetId()}); err != nil {
		return nil, toStatusError(err)
	}

	return &emptypb.Empty{}, nil
}

func toStatusError(err error) error {
	switch {
	case errors.Is(err, user.ErrInvalidEmail), errors.Is(err, user.ErrInvalidName), errors.Is(err, user.ErrInvalidStatus), errors.Is(err, user.ErrInvalidID):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, user.ErrEmailAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, user.ErrUserNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func toProtoUser(u *user.User) *userpb.User {
	if u == nil {
		return nil
	}

	return &userpb.User{
		Id:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Status:    toProtoStatus(u.Status),
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}
}

func toProtoStatus(status user.Status) userpb.UserStatus {
	switch status {
	case user.StatusActive:
		return userpb.UserStatus_USER_STATUS_ACTIVE
	case user.StatusInactive:
		return userpb.UserStatus_USER_STATUS_INACTIVE
	default:
		return userpb.UserStatus_USER_STATUS_UNSPECIFIED
	}
}

func toDomainStatus(status userpb.UserStatus) (user.Status, error) {
	switch status {
	case userpb.UserStatus_USER_STATUS_ACTIVE:
		return user.StatusActive, nil
	case userpb.UserStatus_USER_STATUS_INACTIVE:
		return user.StatusInactive, nil
	case userpb.UserStatus_USER_STATUS_UNSPECIFIED:
		return "", nil
	default:
		return "", user.ErrInvalidStatus
	}
}
