package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	userpb "github.com/ogurasousui/codex-grpc-clean-arch/internal/adapters/grpc/gen/user/v1"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/core/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type stubUserUseCase struct {
	createInput user.CreateUserInput
	createErr   error
	createOut   *user.User

	updateInput user.UpdateUserInput
	updateErr   error
	updateOut   *user.User

	deleteInput user.DeleteUserInput
	deleteErr   error

	getInput user.GetUserInput
	getErr   error
	getOut   *user.User

	listInput user.ListUsersInput
	listErr   error
	listOut   *user.ListUsersResult
}

func (s *stubUserUseCase) CreateUser(ctx context.Context, in user.CreateUserInput) (*user.User, error) {
	s.createInput = in
	return s.createOut, s.createErr
}

func (s *stubUserUseCase) UpdateUser(ctx context.Context, in user.UpdateUserInput) (*user.User, error) {
	s.updateInput = in
	return s.updateOut, s.updateErr
}

func (s *stubUserUseCase) DeleteUser(ctx context.Context, in user.DeleteUserInput) error {
	s.deleteInput = in
	return s.deleteErr
}

func (s *stubUserUseCase) GetUser(ctx context.Context, in user.GetUserInput) (*user.User, error) {
	s.getInput = in
	return s.getOut, s.getErr
}

func (s *stubUserUseCase) ListUsers(ctx context.Context, in user.ListUsersInput) (*user.ListUsersResult, error) {
	s.listInput = in
	return s.listOut, s.listErr
}

func TestUserGrpcHandler_CreateUser(t *testing.T) {
	t.Parallel()

	now := time.Now()
	stub := &stubUserUseCase{
		createOut: &user.User{
			ID:        "user-1",
			Email:     "user@example.com",
			Name:      "User",
			Status:    user.StatusActive,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	handler := NewUserGrpcHandler(stub)

	resp, err := handler.CreateUser(context.Background(), &userpb.CreateUserRequest{Email: "user@example.com", Name: "User"})
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	if stub.createInput.Email != "user@example.com" {
		t.Errorf("expected email passed through, got %s", stub.createInput.Email)
	}

	if resp.GetUser().GetId() != "user-1" {
		t.Errorf("expected id user-1, got %s", resp.GetUser().GetId())
	}
}

func TestUserGrpcHandler_CreateUser_ErrorMapping(t *testing.T) {
	t.Parallel()

	stub := &stubUserUseCase{createErr: user.ErrEmailAlreadyExists}
	handler := NewUserGrpcHandler(stub)

	_, err := handler.CreateUser(context.Background(), &userpb.CreateUserRequest{})
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("expected AlreadyExists, got %v", status.Code(err))
	}
}

func TestUserGrpcHandler_UpdateUser_StatusTranslation(t *testing.T) {
	t.Parallel()

	now := time.Now()
	stub := &stubUserUseCase{
		updateOut: &user.User{
			ID:        "user-1",
			Email:     "user@example.com",
			Name:      "Updated",
			Status:    user.StatusInactive,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	handler := NewUserGrpcHandler(stub)
	nameValue := wrapperspb.String("Updated")

	resp, err := handler.UpdateUser(context.Background(), &userpb.UpdateUserRequest{
		Id:     "user-1",
		Name:   nameValue,
		Status: userpb.UserStatus_USER_STATUS_INACTIVE,
	})
	if err != nil {
		t.Fatalf("UpdateUser returned error: %v", err)
	}

	if stub.updateInput.Status == nil || *stub.updateInput.Status != user.StatusInactive {
		t.Fatalf("expected status to be converted to domain inactive")
	}

	if resp.GetUser().GetStatus() != userpb.UserStatus_USER_STATUS_INACTIVE {
		t.Fatalf("expected response status inactive, got %v", resp.GetUser().GetStatus())
	}
}

func TestUserGrpcHandler_DeleteUser_Error(t *testing.T) {
	t.Parallel()

	stub := &stubUserUseCase{deleteErr: user.ErrUserNotFound}
	handler := NewUserGrpcHandler(stub)

	_, err := handler.DeleteUser(context.Background(), &userpb.DeleteUserRequest{Id: "missing"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", status.Code(err))
	}
}

func TestUserGrpcHandler_DeleteUser_ValidatesRequest(t *testing.T) {
	t.Parallel()

	handler := NewUserGrpcHandler(&stubUserUseCase{})

	_, err := handler.DeleteUser(context.Background(), nil)
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument status, got %v", err)
	}
}

func TestToStatusError_DefaultsToInternal(t *testing.T) {
	t.Parallel()

	err := toStatusError(errors.New("unexpected"))
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("expected Internal code, got %v", err)
	}
}

func TestDeleteUserSuccessReturnsEmpty(t *testing.T) {
	t.Parallel()

	stub := &stubUserUseCase{}
	handler := NewUserGrpcHandler(stub)

	resp, err := handler.DeleteUser(context.Background(), &userpb.DeleteUserRequest{Id: "user-1"})
	if err != nil {
		t.Fatalf("DeleteUser returned error: %v", err)
	}

	if resp == nil {
		t.Fatalf("expected non-nil empty response")
	}
}

func TestUserGrpcHandler_GetUser_Success(t *testing.T) {
	t.Parallel()

	now := time.Now()
	stub := &stubUserUseCase{
		getOut: &user.User{
			ID:        "user-1",
			Email:     "user@example.com",
			Name:      "User",
			Status:    user.StatusActive,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	handler := NewUserGrpcHandler(stub)

	resp, err := handler.GetUser(context.Background(), &userpb.GetUserRequest{Id: "user-1"})
	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}

	if stub.getInput.ID != "user-1" {
		t.Fatalf("expected ID to be passed to use case")
	}

	if resp.GetUser().GetId() != "user-1" {
		t.Fatalf("expected response ID user-1, got %s", resp.GetUser().GetId())
	}
}

func TestUserGrpcHandler_GetUser_ErrorMapping(t *testing.T) {
	t.Parallel()

	stub := &stubUserUseCase{getErr: user.ErrUserNotFound}
	handler := NewUserGrpcHandler(stub)

	_, err := handler.GetUser(context.Background(), &userpb.GetUserRequest{Id: "missing"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", status.Code(err))
	}
}

func TestUserGrpcHandler_ListUsers_Success(t *testing.T) {
	t.Parallel()

	now := time.Now()
	stub := &stubUserUseCase{
		listOut: &user.ListUsersResult{
			Users: []*user.User{
				{
					ID:        "user-1",
					Email:     "user1@example.com",
					Name:      "User1",
					Status:    user.StatusActive,
					CreatedAt: now,
					UpdatedAt: now,
				},
				{
					ID:        "user-2",
					Email:     "user2@example.com",
					Name:      "User2",
					Status:    user.StatusInactive,
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
			NextPageToken: "2",
		},
	}

	handler := NewUserGrpcHandler(stub)

	resp, err := handler.ListUsers(context.Background(), &userpb.ListUsersRequest{PageSize: 20, PageToken: "0"})
	if err != nil {
		t.Fatalf("ListUsers returned error: %v", err)
	}

	if stub.listInput.PageSize != 20 {
		t.Fatalf("expected page size 20, got %d", stub.listInput.PageSize)
	}

	if len(resp.GetUsers()) != 2 {
		t.Fatalf("expected 2 users in response, got %d", len(resp.GetUsers()))
	}

	if resp.GetNextPageToken() != "2" {
		t.Fatalf("expected next page token 2, got %s", resp.GetNextPageToken())
	}
}

func TestUserGrpcHandler_ListUsers_StatusFilter(t *testing.T) {
	t.Parallel()

	stub := &stubUserUseCase{
		listOut: &user.ListUsersResult{Users: []*user.User{}},
	}

	handler := NewUserGrpcHandler(stub)

	_, err := handler.ListUsers(context.Background(), &userpb.ListUsersRequest{Status: userpb.UserStatus_USER_STATUS_INACTIVE})
	if err != nil {
		t.Fatalf("ListUsers returned error: %v", err)
	}

	if stub.listInput.Status == nil || *stub.listInput.Status != user.StatusInactive {
		t.Fatalf("expected status to be set to inactive")
	}
}

func TestUserGrpcHandler_ListUsers_ErrorMapping(t *testing.T) {
	t.Parallel()

	stub := &stubUserUseCase{listErr: user.ErrInvalidPageSize}
	handler := NewUserGrpcHandler(stub)

	_, err := handler.ListUsers(context.Background(), &userpb.ListUsersRequest{PageSize: 1000})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}
