package handler

import (
	"context"
	"fmt"
	"strings"
	"time"

	employeepb "github.com/ogurasousui/codex-grpc-clean-arch/internal/adapters/grpc/gen/employee/v1"
	userpb "github.com/ogurasousui/codex-grpc-clean-arch/internal/adapters/grpc/gen/user/v1"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/core/employee"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const dateLayout = "2006-01-02"

// EmployeeGrpcHandler は EmployeeService の gRPC 実装です。
type EmployeeGrpcHandler struct {
	svc employee.UseCase
	employeepb.UnimplementedEmployeeServiceServer
}

// NewEmployeeGrpcHandler は EmployeeGrpcHandler を生成します。
func NewEmployeeGrpcHandler(svc employee.UseCase) *EmployeeGrpcHandler {
	return &EmployeeGrpcHandler{svc: svc}
}

// CreateEmployee は社員を作成します。
func (h *EmployeeGrpcHandler) CreateEmployee(ctx context.Context, req *employeepb.CreateEmployeeRequest) (*employeepb.CreateEmployeeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if strings.TrimSpace(req.GetUserId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	hiredAt, err := parseDateValue(req.HiredAt)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("hired_at: %v", err))
	}

	terminatedAt, err := parseDateValue(req.TerminatedAt)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("terminated_at: %v", err))
	}

	var statusPtr *employee.Status
	if req.GetStatus() != employeepb.EmployeeStatus_EMPLOYEE_STATUS_UNSPECIFIED {
		domainStatus, err := toEmployeeDomainStatus(req.GetStatus())
		if err != nil {
			return nil, toStatusError(err)
		}
		statusPtr = &domainStatus
	}

	created, err := h.svc.CreateEmployee(ctx, employee.CreateEmployeeInput{
		CompanyID:    req.GetCompanyId(),
		EmployeeCode: req.GetEmployeeCode(),
		UserID:       req.GetUserId(),
		Status:       statusPtr,
		HiredAt:      hiredAt,
		TerminatedAt: terminatedAt,
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	return &employeepb.CreateEmployeeResponse{Employee: toProtoEmployee(created)}, nil
}

// UpdateEmployee は社員情報を更新します。
func (h *EmployeeGrpcHandler) UpdateEmployee(ctx context.Context, req *employeepb.UpdateEmployeeRequest) (*employeepb.UpdateEmployeeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	var codePtr *string
	if req.EmployeeCode != nil {
		value := req.EmployeeCode.GetValue()
		codePtr = &value
	}

	var userIDPtr *string
	if req.UserId != nil {
		value := req.UserId.GetValue()
		userIDPtr = &value
	}

	var statusPtr *employee.Status
	if req.GetStatus() != employeepb.EmployeeStatus_EMPLOYEE_STATUS_UNSPECIFIED {
		domainStatus, err := toEmployeeDomainStatus(req.GetStatus())
		if err != nil {
			return nil, toStatusError(err)
		}
		statusPtr = &domainStatus
	}

	hiredAt, hiredSet, err := parseDateUpdateValue(req.HiredAt)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("hired_at: %v", err))
	}

	terminatedAt, terminatedSet, err := parseDateUpdateValue(req.TerminatedAt)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("terminated_at: %v", err))
	}

	updated, err := h.svc.UpdateEmployee(ctx, employee.UpdateEmployeeInput{
		ID:              req.GetId(),
		EmployeeCode:    codePtr,
		UserID:          userIDPtr,
		Status:          statusPtr,
		HiredAt:         hiredAt,
		HiredAtSet:      hiredSet,
		TerminatedAt:    terminatedAt,
		TerminatedAtSet: terminatedSet,
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	return &employeepb.UpdateEmployeeResponse{Employee: toProtoEmployee(updated)}, nil
}

// DeleteEmployee は社員を削除します。
func (h *EmployeeGrpcHandler) DeleteEmployee(ctx context.Context, req *employeepb.DeleteEmployeeRequest) (*employeepb.DeleteEmployeeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if err := h.svc.DeleteEmployee(ctx, employee.DeleteEmployeeInput{ID: req.GetId()}); err != nil {
		return nil, toStatusError(err)
	}

	return &employeepb.DeleteEmployeeResponse{}, nil
}

// GetEmployee は社員を取得します。
func (h *EmployeeGrpcHandler) GetEmployee(ctx context.Context, req *employeepb.GetEmployeeRequest) (*employeepb.GetEmployeeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	found, err := h.svc.GetEmployee(ctx, employee.GetEmployeeInput{ID: req.GetId()})
	if err != nil {
		return nil, toStatusError(err)
	}

	return &employeepb.GetEmployeeResponse{Employee: toProtoEmployee(found)}, nil
}

// ListEmployees は社員の一覧を取得します。
func (h *EmployeeGrpcHandler) ListEmployees(ctx context.Context, req *employeepb.ListEmployeesRequest) (*employeepb.ListEmployeesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	var statusPtr *employee.Status
	if req.GetStatus() != employeepb.EmployeeStatus_EMPLOYEE_STATUS_UNSPECIFIED {
		domainStatus, err := toEmployeeDomainStatus(req.GetStatus())
		if err != nil {
			return nil, toStatusError(err)
		}
		statusPtr = &domainStatus
	}

	result, err := h.svc.ListEmployees(ctx, employee.ListEmployeesInput{
		CompanyID: req.GetCompanyId(),
		PageSize:  int(req.GetPageSize()),
		PageToken: req.GetPageToken(),
		Status:    statusPtr,
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	protoEmployees := make([]*employeepb.Employee, 0, len(result.Employees))
	for _, emp := range result.Employees {
		protoEmployees = append(protoEmployees, toProtoEmployee(emp))
	}

	return &employeepb.ListEmployeesResponse{
		Employees:     protoEmployees,
		NextPageToken: result.NextPageToken,
	}, nil
}

func toProtoEmployee(emp *employee.Employee) *employeepb.Employee {
	if emp == nil {
		return nil
	}

	return &employeepb.Employee{
		Id:           emp.ID,
		CompanyId:    emp.CompanyID,
		EmployeeCode: emp.EmployeeCode,
		UserId:       emp.UserID,
		Status:       toEmployeeProtoStatus(emp.Status),
		HiredAt:      timePointerToWrapper(emp.HiredAt),
		TerminatedAt: timePointerToWrapper(emp.TerminatedAt),
		CreatedAt:    timestamppb.New(emp.CreatedAt),
		UpdatedAt:    timestamppb.New(emp.UpdatedAt),
		User:         toProtoUserSummary(emp.User),
	}
}

func toEmployeeProtoStatus(status employee.Status) employeepb.EmployeeStatus {
	switch status {
	case employee.StatusActive:
		return employeepb.EmployeeStatus_EMPLOYEE_STATUS_ACTIVE
	case employee.StatusInactive:
		return employeepb.EmployeeStatus_EMPLOYEE_STATUS_INACTIVE
	default:
		return employeepb.EmployeeStatus_EMPLOYEE_STATUS_UNSPECIFIED
	}
}

func toEmployeeDomainStatus(status employeepb.EmployeeStatus) (employee.Status, error) {
	switch status {
	case employeepb.EmployeeStatus_EMPLOYEE_STATUS_ACTIVE:
		return employee.StatusActive, nil
	case employeepb.EmployeeStatus_EMPLOYEE_STATUS_INACTIVE:
		return employee.StatusInactive, nil
	case employeepb.EmployeeStatus_EMPLOYEE_STATUS_UNSPECIFIED:
		return "", nil
	default:
		return "", employee.ErrInvalidStatus
	}
}

func toProtoUserSummary(snapshot *employee.UserSnapshot) *employeepb.UserSummary {
	if snapshot == nil {
		return nil
	}

	return &employeepb.UserSummary{
		Id:        snapshot.ID,
		Email:     snapshot.Email,
		Name:      snapshot.Name,
		Status:    toUserProtoStatus(snapshot.Status),
		CreatedAt: timestamppb.New(snapshot.CreatedAt),
		UpdatedAt: timestamppb.New(snapshot.UpdatedAt),
	}
}

func timePointerToWrapper(value *time.Time) *wrapperspb.StringValue {
	if value == nil {
		return nil
	}
	return wrapperspb.String(value.Format(dateLayout))
}

func parseDateValue(value *wrapperspb.StringValue) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(value.GetValue())
	if trimmed == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation(dateLayout, trimmed, time.UTC)
	if err != nil {
		return nil, fmt.Errorf("invalid format, expected YYYY-MM-DD")
	}
	return &t, nil
}

func parseDateUpdateValue(value *wrapperspb.StringValue) (*time.Time, bool, error) {
	if value == nil {
		return nil, false, nil
	}
	trimmed := strings.TrimSpace(value.GetValue())
	if trimmed == "" {
		return nil, true, nil
	}
	t, err := time.ParseInLocation(dateLayout, trimmed, time.UTC)
	if err != nil {
		return nil, false, fmt.Errorf("invalid format, expected YYYY-MM-DD")
	}
	return &t, true, nil
}

func toUserProtoStatus(status string) userpb.UserStatus {
	switch strings.ToLower(status) {
	case "active":
		return userpb.UserStatus_USER_STATUS_ACTIVE
	case "inactive":
		return userpb.UserStatus_USER_STATUS_INACTIVE
	default:
		return userpb.UserStatus_USER_STATUS_UNSPECIFIED
	}
}
