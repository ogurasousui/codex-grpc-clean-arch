package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	companypb "github.com/ogurasousui/codex-grpc-clean-arch/internal/adapters/grpc/gen/company/v1"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/core/company"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type stubCompanyUseCase struct {
	createInput company.CreateCompanyInput
	createErr   error
	createOut   *company.Company

	getInput company.GetCompanyInput
	getErr   error
	getOut   *company.Company

	listInput company.ListCompaniesInput
	listErr   error
	listOut   *company.ListCompaniesResult

	updateInput company.UpdateCompanyInput
	updateErr   error
	updateOut   *company.Company

	deleteInput company.DeleteCompanyInput
	deleteErr   error
}

func (s *stubCompanyUseCase) CreateCompany(ctx context.Context, in company.CreateCompanyInput) (*company.Company, error) {
	s.createInput = in
	return s.createOut, s.createErr
}

func (s *stubCompanyUseCase) GetCompany(ctx context.Context, in company.GetCompanyInput) (*company.Company, error) {
	s.getInput = in
	return s.getOut, s.getErr
}

func (s *stubCompanyUseCase) ListCompanies(ctx context.Context, in company.ListCompaniesInput) (*company.ListCompaniesResult, error) {
	s.listInput = in
	return s.listOut, s.listErr
}

func (s *stubCompanyUseCase) UpdateCompany(ctx context.Context, in company.UpdateCompanyInput) (*company.Company, error) {
	s.updateInput = in
	return s.updateOut, s.updateErr
}

func (s *stubCompanyUseCase) DeleteCompany(ctx context.Context, in company.DeleteCompanyInput) error {
	s.deleteInput = in
	return s.deleteErr
}

func TestCompanyGrpcHandler_CreateCompany(t *testing.T) {
	t.Parallel()

	desc := " Description "
	now := time.Now()
	stub := &stubCompanyUseCase{
		createOut: &company.Company{
			ID:          "company-1",
			Name:        "Example",
			Code:        "example",
			Status:      company.StatusActive,
			Description: &desc,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	handler := NewCompanyGrpcHandler(stub)

	req := &companypb.CreateCompanyRequest{Name: "Example", Code: "example", Description: wrapperspb.String("test")}
	resp, err := handler.CreateCompany(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateCompany returned error: %v", err)
	}

	if stub.createInput.Name != "Example" || stub.createInput.Code != "example" {
		t.Fatalf("expected inputs to be passed through, got %+v", stub.createInput)
	}

	if resp.GetCompany().GetId() != "company-1" {
		t.Fatalf("expected id company-1, got %s", resp.GetCompany().GetId())
	}
}

func TestCompanyGrpcHandler_CreateCompany_ErrorMapping(t *testing.T) {
	t.Parallel()

	stub := &stubCompanyUseCase{createErr: company.ErrCodeAlreadyExists}
	handler := NewCompanyGrpcHandler(stub)

	_, err := handler.CreateCompany(context.Background(), &companypb.CreateCompanyRequest{})
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("expected AlreadyExists, got %v", status.Code(err))
	}
}

func TestCompanyGrpcHandler_GetCompany_Success(t *testing.T) {
	t.Parallel()

	now := time.Now()
	stub := &stubCompanyUseCase{
		getOut: &company.Company{
			ID:        "company-1",
			Name:      "Example",
			Code:      "example",
			Status:    company.StatusActive,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	handler := NewCompanyGrpcHandler(stub)

	resp, err := handler.GetCompany(context.Background(), &companypb.GetCompanyRequest{Id: "company-1"})
	if err != nil {
		t.Fatalf("GetCompany returned error: %v", err)
	}

	if stub.getInput.ID != "company-1" {
		t.Fatalf("expected ID to be passed through, got %s", stub.getInput.ID)
	}

	if resp.GetCompany().GetId() != "company-1" {
		t.Fatalf("expected response id company-1, got %s", resp.GetCompany().GetId())
	}
}

func TestCompanyGrpcHandler_ListCompanies_StatusTranslation(t *testing.T) {
	t.Parallel()

	now := time.Now()
	stub := &stubCompanyUseCase{
		listOut: &company.ListCompaniesResult{
			Companies: []*company.Company{{
				ID:        "company-1",
				Name:      "Example",
				Code:      "example",
				Status:    company.StatusInactive,
				CreatedAt: now,
				UpdatedAt: now,
			}},
		},
	}

	handler := NewCompanyGrpcHandler(stub)
	resp, err := handler.ListCompanies(context.Background(), &companypb.ListCompaniesRequest{
		PageSize: 2,
		Status:   companypb.CompanyStatus_COMPANY_STATUS_INACTIVE,
	})
	if err != nil {
		t.Fatalf("ListCompanies returned error: %v", err)
	}

	if stub.listInput.PageSize != 2 {
		t.Fatalf("expected page size 2, got %d", stub.listInput.PageSize)
	}

	if stub.listInput.Status == nil || *stub.listInput.Status != company.StatusInactive {
		t.Fatalf("expected status filter to convert to domain inactive")
	}

	if len(resp.GetCompanies()) != 1 {
		t.Fatalf("expected one company in response")
	}
}

func TestCompanyGrpcHandler_UpdateCompany(t *testing.T) {
	t.Parallel()

	now := time.Now()
	stub := &stubCompanyUseCase{
		updateOut: &company.Company{
			ID:        "company-1",
			Name:      "Updated",
			Code:      "updated",
			Status:    company.StatusActive,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	handler := NewCompanyGrpcHandler(stub)
	name := wrapperspb.String("Updated")
	code := wrapperspb.String("updated")
	description := wrapperspb.String("desc")

	resp, err := handler.UpdateCompany(context.Background(), &companypb.UpdateCompanyRequest{
		Id:          "company-1",
		Name:        name,
		Code:        code,
		Status:      companypb.CompanyStatus_COMPANY_STATUS_ACTIVE,
		Description: description,
	})
	if err != nil {
		t.Fatalf("UpdateCompany returned error: %v", err)
	}

	if stub.updateInput.Name == nil || *stub.updateInput.Name != "Updated" {
		t.Fatalf("expected name to be passed through")
	}

	if stub.updateInput.Code == nil || *stub.updateInput.Code != "updated" {
		t.Fatalf("expected code to be passed through")
	}

	if stub.updateInput.Description == nil || *stub.updateInput.Description != "desc" {
		t.Fatalf("expected description to be passed through")
	}

	if resp.GetCompany().GetName() != "Updated" {
		t.Fatalf("expected updated name, got %s", resp.GetCompany().GetName())
	}
}

func TestCompanyGrpcHandler_DeleteCompany_Error(t *testing.T) {
	t.Parallel()

	stub := &stubCompanyUseCase{deleteErr: company.ErrCompanyNotFound}
	handler := NewCompanyGrpcHandler(stub)

	_, err := handler.DeleteCompany(context.Background(), &companypb.DeleteCompanyRequest{Id: "missing"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", status.Code(err))
	}
}

func TestCompanyGrpcHandler_ValidatesNilRequest(t *testing.T) {
	t.Parallel()

	handler := NewCompanyGrpcHandler(&stubCompanyUseCase{})

	_, err := handler.CreateCompany(context.Background(), nil)
	if !isInvalidArgument(err) {
		t.Fatalf("expected invalid argument for create")
	}

	_, err = handler.GetCompany(context.Background(), nil)
	if !isInvalidArgument(err) {
		t.Fatalf("expected invalid argument for get")
	}

	_, err = handler.ListCompanies(context.Background(), nil)
	if !isInvalidArgument(err) {
		t.Fatalf("expected invalid argument for list")
	}

	_, err = handler.UpdateCompany(context.Background(), nil)
	if !isInvalidArgument(err) {
		t.Fatalf("expected invalid argument for update")
	}

	_, err = handler.DeleteCompany(context.Background(), nil)
	if !isInvalidArgument(err) {
		t.Fatalf("expected invalid argument for delete")
	}
}

func isInvalidArgument(err error) bool {
	st, ok := status.FromError(err)
	return ok && st.Code() == codes.InvalidArgument
}

func TestToDomainCompanyStatus_Invalid(t *testing.T) {
	t.Parallel()

	if _, err := toDomainCompanyStatus(companypb.CompanyStatus(99)); !errors.Is(err, company.ErrInvalidStatus) {
		t.Fatalf("expected ErrInvalidStatus, got %v", err)
	}
}
