package handler

import (
	"context"

	userpb "github.com/ogurasousui/codex-grpc-clean-arch/internal/adapters/grpc/gen/user/v1"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/core/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
func (h *UserGrpcHandler) DeleteUser(ctx context.Context, req *userpb.DeleteUserRequest) (*userpb.DeleteUserResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if err := h.svc.DeleteUser(ctx, user.DeleteUserInput{ID: req.GetId()}); err != nil {
		return nil, toStatusError(err)
	}

	return &userpb.DeleteUserResponse{}, nil
}

// GetUser はユーザーを取得します。
func (h *UserGrpcHandler) GetUser(ctx context.Context, req *userpb.GetUserRequest) (*userpb.GetUserResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	found, err := h.svc.GetUser(ctx, user.GetUserInput{ID: req.GetId()})
	if err != nil {
		return nil, toStatusError(err)
	}

	return &userpb.GetUserResponse{User: toProtoUser(found)}, nil
}

// ListUsers はユーザーの一覧を取得します。
func (h *UserGrpcHandler) ListUsers(ctx context.Context, req *userpb.ListUsersRequest) (*userpb.ListUsersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	var statusPtr *user.Status
	if req.GetStatus() != userpb.UserStatus_USER_STATUS_UNSPECIFIED {
		domainStatus, err := toDomainStatus(req.GetStatus())
		if err != nil {
			return nil, toStatusError(err)
		}
		statusPtr = &domainStatus
	}

	result, err := h.svc.ListUsers(ctx, user.ListUsersInput{
		PageSize:  int(req.GetPageSize()),
		PageToken: req.GetPageToken(),
		Status:    statusPtr,
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	protoUsers := make([]*userpb.User, 0, len(result.Users))
	for _, u := range result.Users {
		protoUsers = append(protoUsers, toProtoUser(u))
	}

	return &userpb.ListUsersResponse{
		Users:         protoUsers,
		NextPageToken: result.NextPageToken,
	}, nil
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
