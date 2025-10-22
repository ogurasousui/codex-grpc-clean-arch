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
