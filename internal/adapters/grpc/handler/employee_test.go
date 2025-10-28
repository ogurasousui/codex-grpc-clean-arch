package handler

import (
	"context"
	"testing"
	"time"

	employeepb "github.com/ogurasousui/codex-grpc-clean-arch/internal/adapters/grpc/gen/employee/v1"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/core/employee"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type stubEmployeeUseCase struct {
	createInput employee.CreateEmployeeInput
	createOut   *employee.Employee
	createErr   error

	updateInput employee.UpdateEmployeeInput
	updateOut   *employee.Employee
	updateErr   error

	deleteInput employee.DeleteEmployeeInput
	deleteErr   error

	getInput employee.GetEmployeeInput
	getOut   *employee.Employee
	getErr   error

	listInput employee.ListEmployeesInput
	listOut   *employee.ListEmployeesResult
	listErr   error
}

func (s *stubEmployeeUseCase) CreateEmployee(ctx context.Context, in employee.CreateEmployeeInput) (*employee.Employee, error) {
	s.createInput = in
	return s.createOut, s.createErr
}

func (s *stubEmployeeUseCase) GetEmployee(ctx context.Context, in employee.GetEmployeeInput) (*employee.Employee, error) {
	s.getInput = in
	return s.getOut, s.getErr
}

func (s *stubEmployeeUseCase) ListEmployees(ctx context.Context, in employee.ListEmployeesInput) (*employee.ListEmployeesResult, error) {
	s.listInput = in
	return s.listOut, s.listErr
}

func (s *stubEmployeeUseCase) UpdateEmployee(ctx context.Context, in employee.UpdateEmployeeInput) (*employee.Employee, error) {
	s.updateInput = in
	return s.updateOut, s.updateErr
}

func (s *stubEmployeeUseCase) DeleteEmployee(ctx context.Context, in employee.DeleteEmployeeInput) error {
	s.deleteInput = in
	return s.deleteErr
}

func TestEmployeeGrpcHandler_CreateEmployee_Success(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	stub := &stubEmployeeUseCase{
		createOut: &employee.Employee{
			ID:           "emp-1",
			CompanyID:    "company-1",
			EmployeeCode: "emp-001",
			UserID:       "user-1",
			Status:       employee.StatusActive,
			CreatedAt:    now,
			UpdatedAt:    now,
			User: &employee.UserSnapshot{
				ID:        "user-1",
				Email:     "user@example.com",
				Name:      "Taro Yamada",
				Status:    "active",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	handler := NewEmployeeGrpcHandler(stub)
	resp, err := handler.CreateEmployee(context.Background(), &employeepb.CreateEmployeeRequest{
		CompanyId:    "company-1",
		EmployeeCode: "emp-001",
		UserId:       "user-1",
		HiredAt:      wrapperspb.String("2024-01-01"),
	})
	if err != nil {
		t.Fatalf("CreateEmployee returned error: %v", err)
	}

	if stub.createInput.CompanyID != "company-1" {
		t.Errorf("expected company id to pass through, got %s", stub.createInput.CompanyID)
	}
	if stub.createInput.UserID != "user-1" {
		t.Errorf("expected user id to be passed, got %s", stub.createInput.UserID)
	}
	if stub.createInput.HiredAt == nil || stub.createInput.HiredAt.Format("2006-01-02") != "2024-01-01" {
		t.Errorf("expected hired date parsed, got %+v", stub.createInput.HiredAt)
	}

	if resp.GetEmployee().GetId() != "emp-1" {
		t.Fatalf("expected response id 'emp-1', got %s", resp.GetEmployee().GetId())
	}
	if resp.GetEmployee().GetUser().GetEmail() != "user@example.com" {
		t.Fatalf("expected embedded user email, got %s", resp.GetEmployee().GetUser().GetEmail())
	}
}

func TestEmployeeGrpcHandler_CreateEmployee_InvalidDateFormat(t *testing.T) {
	t.Parallel()

	handler := NewEmployeeGrpcHandler(&stubEmployeeUseCase{})

	_, err := handler.CreateEmployee(context.Background(), &employeepb.CreateEmployeeRequest{
		CompanyId:    "company-1",
		EmployeeCode: "emp-001",
		UserId:       "user-1",
		HiredAt:      wrapperspb.String("2024/01/01"),
	})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("expected invalid argument for date parse, got %v", st.Code())
	}
}

func TestEmployeeGrpcHandler_UpdateEmployee_SetsPointers(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	stub := &stubEmployeeUseCase{
		updateOut: &employee.Employee{
			ID:           "emp-1",
			CompanyID:    "company-1",
			EmployeeCode: "emp-001",
			UserID:       "user-1",
			Status:       employee.StatusActive,
			CreatedAt:    now,
			UpdatedAt:    now,
			User: &employee.UserSnapshot{
				ID:        "user-1",
				Email:     "user@example.com",
				Name:      "Updated User",
				Status:    "active",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
	handler := NewEmployeeGrpcHandler(stub)

	resp, err := handler.UpdateEmployee(context.Background(), &employeepb.UpdateEmployeeRequest{
		Id:           "emp-1",
		EmployeeCode: wrapperspb.String("emp-002"),
		UserId:       wrapperspb.String("user-2"),
		Status:       employeepb.EmployeeStatus_EMPLOYEE_STATUS_INACTIVE,
		HiredAt:      wrapperspb.String(""),
		TerminatedAt: wrapperspb.String("2024-02-01"),
	})
	if err != nil {
		t.Fatalf("UpdateEmployee returned error: %v", err)
	}

	if stub.updateInput.EmployeeCode == nil || *stub.updateInput.EmployeeCode != "emp-002" {
		t.Fatalf("expected employee code pointer to be set")
	}
	if stub.updateInput.UserID == nil || *stub.updateInput.UserID != "user-2" {
		t.Fatalf("expected user id pointer to be set")
	}
	if !stub.updateInput.HiredAtSet || stub.updateInput.HiredAt != nil {
		t.Fatalf("expected hired_at to be explicitly cleared")
	}
	if !stub.updateInput.TerminatedAtSet || stub.updateInput.TerminatedAt == nil {
		t.Fatalf("expected terminated_at to be set")
	}
	if stub.updateInput.Status == nil || *stub.updateInput.Status != employee.StatusInactive {
		t.Fatalf("expected status to be converted to inactive")
	}

	if resp.GetEmployee().GetStatus() != employeepb.EmployeeStatus_EMPLOYEE_STATUS_ACTIVE {
		t.Fatalf("response should echo domain status")
	}
	if resp.GetEmployee().GetUser().GetId() != "user-1" {
		t.Fatalf("expected user snapshot in response")
	}
}

func TestEmployeeGrpcHandler_ListEmployees_ErrorMapping(t *testing.T) {
	t.Parallel()

	stub := &stubEmployeeUseCase{listErr: employee.ErrInvalidCompanyID}
	handler := NewEmployeeGrpcHandler(stub)

	_, err := handler.ListEmployees(context.Background(), &employeepb.ListEmployeesRequest{CompanyId: ""})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got %v", status.Code(err))
	}

	if stub.listInput.CompanyID != "" {
		t.Fatalf("expected company id to be passed even on error")
	}
}

func TestEmployeeGrpcHandler_GetEmployee_NotFound(t *testing.T) {
	t.Parallel()

	stub := &stubEmployeeUseCase{getErr: employee.ErrEmployeeNotFound}
	handler := NewEmployeeGrpcHandler(stub)

	_, err := handler.GetEmployee(context.Background(), &employeepb.GetEmployeeRequest{Id: "missing"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", status.Code(err))
	}
}

func TestEmployeeGrpcHandler_DeleteEmployee_ValidatesRequest(t *testing.T) {
	t.Parallel()

	handler := NewEmployeeGrpcHandler(&stubEmployeeUseCase{})

	_, err := handler.DeleteEmployee(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument for nil request")
	}
}

func TestEmployeeGrpcHandler_ListEmployees_Success(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	stub := &stubEmployeeUseCase{
		listOut: &employee.ListEmployeesResult{
			Employees: []*employee.Employee{
				{
					ID:           "emp-1",
					CompanyID:    "company-1",
					EmployeeCode: "emp-1",
					UserID:       "user-1",
					Status:       employee.StatusActive,
					CreatedAt:    now,
					UpdatedAt:    now,
					User: &employee.UserSnapshot{
						ID:        "user-1",
						Email:     "user@example.com",
						Name:      "Test User",
						Status:    "active",
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
			},
			NextPageToken: "10",
		},
	}

	handler := NewEmployeeGrpcHandler(stub)

	resp, err := handler.ListEmployees(context.Background(), &employeepb.ListEmployeesRequest{CompanyId: "company-1", PageSize: 10})
	if err != nil {
		t.Fatalf("ListEmployees returned error: %v", err)
	}

	if stub.listInput.CompanyID != "company-1" {
		t.Fatalf("expected company id passed to use case")
	}
	if len(resp.GetEmployees()) != 1 {
		t.Fatalf("expected one employee, got %d", len(resp.GetEmployees()))
	}
	if resp.GetNextPageToken() != "10" {
		t.Fatalf("expected next page token '10', got %s", resp.GetNextPageToken())
	}
}

func TestEmployeeGrpcHandler_DeleteEmployee_ErrorMapping(t *testing.T) {
	t.Parallel()

	stub := &stubEmployeeUseCase{deleteErr: employee.ErrEmployeeNotFound}
	handler := NewEmployeeGrpcHandler(stub)

	_, err := handler.DeleteEmployee(context.Background(), &employeepb.DeleteEmployeeRequest{Id: "missing"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", status.Code(err))
	}
	if stub.deleteInput.ID != "missing" {
		t.Fatalf("expected delete input to capture id")
	}
}
