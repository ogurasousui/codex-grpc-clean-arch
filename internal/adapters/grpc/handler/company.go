package handler

import (
	"context"

	companypb "github.com/ogurasousui/codex-grpc-clean-arch/internal/adapters/grpc/gen/company/v1"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/core/company"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// CompanyGrpcHandler は CompanyService の gRPC 実装です。
type CompanyGrpcHandler struct {
	svc company.UseCase
	companypb.UnimplementedCompanyServiceServer
}

// NewCompanyGrpcHandler は CompanyGrpcHandler を生成します。
func NewCompanyGrpcHandler(svc company.UseCase) *CompanyGrpcHandler {
	return &CompanyGrpcHandler{svc: svc}
}

// CreateCompany は会社を作成します。
func (h *CompanyGrpcHandler) CreateCompany(ctx context.Context, req *companypb.CreateCompanyRequest) (*companypb.CreateCompanyResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	var description *string
	if req.GetDescription() != nil {
		value := req.GetDescription().GetValue()
		description = &value
	}

	created, err := h.svc.CreateCompany(ctx, company.CreateCompanyInput{
		Name:        req.GetName(),
		Code:        req.GetCode(),
		Description: description,
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	return &companypb.CreateCompanyResponse{Company: toProtoCompany(created)}, nil
}

// GetCompany は会社を取得します。
func (h *CompanyGrpcHandler) GetCompany(ctx context.Context, req *companypb.GetCompanyRequest) (*companypb.GetCompanyResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	found, err := h.svc.GetCompany(ctx, company.GetCompanyInput{ID: req.GetId()})
	if err != nil {
		return nil, toStatusError(err)
	}

	return &companypb.GetCompanyResponse{Company: toProtoCompany(found)}, nil
}

// ListCompanies は会社の一覧を取得します。
func (h *CompanyGrpcHandler) ListCompanies(ctx context.Context, req *companypb.ListCompaniesRequest) (*companypb.ListCompaniesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	var statusPtr *company.Status
	if req.GetStatus() != companypb.CompanyStatus_COMPANY_STATUS_UNSPECIFIED {
		domainStatus, err := toDomainCompanyStatus(req.GetStatus())
		if err != nil {
			return nil, toStatusError(err)
		}
		statusPtr = &domainStatus
	}

	result, err := h.svc.ListCompanies(ctx, company.ListCompaniesInput{
		PageSize:  int(req.GetPageSize()),
		PageToken: req.GetPageToken(),
		Status:    statusPtr,
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	protoCompanies := make([]*companypb.Company, 0, len(result.Companies))
	for _, c := range result.Companies {
		protoCompanies = append(protoCompanies, toProtoCompany(c))
	}

	return &companypb.ListCompaniesResponse{
		Companies:     protoCompanies,
		NextPageToken: result.NextPageToken,
	}, nil
}

// UpdateCompany は会社情報を更新します。
func (h *CompanyGrpcHandler) UpdateCompany(ctx context.Context, req *companypb.UpdateCompanyRequest) (*companypb.UpdateCompanyResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	var namePtr *string
	if req.GetName() != nil {
		value := req.GetName().GetValue()
		namePtr = &value
	}

	var codePtr *string
	if req.GetCode() != nil {
		value := req.GetCode().GetValue()
		codePtr = &value
	}

	var statusPtr *company.Status
	if req.GetStatus() != companypb.CompanyStatus_COMPANY_STATUS_UNSPECIFIED {
		domainStatus, err := toDomainCompanyStatus(req.GetStatus())
		if err != nil {
			return nil, toStatusError(err)
		}
		statusPtr = &domainStatus
	}

	var descriptionPtr *string
	if req.GetDescription() != nil {
		value := req.GetDescription().GetValue()
		descriptionPtr = &value
	}

	updated, err := h.svc.UpdateCompany(ctx, company.UpdateCompanyInput{
		ID:          req.GetId(),
		Name:        namePtr,
		Code:        codePtr,
		Status:      statusPtr,
		Description: descriptionPtr,
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	return &companypb.UpdateCompanyResponse{Company: toProtoCompany(updated)}, nil
}

// DeleteCompany は会社を削除します。
func (h *CompanyGrpcHandler) DeleteCompany(ctx context.Context, req *companypb.DeleteCompanyRequest) (*companypb.DeleteCompanyResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if err := h.svc.DeleteCompany(ctx, company.DeleteCompanyInput{ID: req.GetId()}); err != nil {
		return nil, toStatusError(err)
	}

	return &companypb.DeleteCompanyResponse{}, nil
}

func toProtoCompany(c *company.Company) *companypb.Company {
	if c == nil {
		return nil
	}

	var description *wrapperspb.StringValue
	if c.Description != nil {
		description = wrapperspb.String(*c.Description)
	}

	return &companypb.Company{
		Id:          c.ID,
		Name:        c.Name,
		Code:        c.Code,
		Status:      toProtoCompanyStatus(c.Status),
		Description: description,
		CreatedAt:   timestamppb.New(c.CreatedAt),
		UpdatedAt:   timestamppb.New(c.UpdatedAt),
	}
}

func toProtoCompanyStatus(status company.Status) companypb.CompanyStatus {
	switch status {
	case company.StatusActive:
		return companypb.CompanyStatus_COMPANY_STATUS_ACTIVE
	case company.StatusInactive:
		return companypb.CompanyStatus_COMPANY_STATUS_INACTIVE
	default:
		return companypb.CompanyStatus_COMPANY_STATUS_UNSPECIFIED
	}
}

func toDomainCompanyStatus(status companypb.CompanyStatus) (company.Status, error) {
	switch status {
	case companypb.CompanyStatus_COMPANY_STATUS_ACTIVE:
		return company.StatusActive, nil
	case companypb.CompanyStatus_COMPANY_STATUS_INACTIVE:
		return company.StatusInactive, nil
	case companypb.CompanyStatus_COMPANY_STATUS_UNSPECIFIED:
		return "", nil
	default:
		return "", company.ErrInvalidStatus
	}
}
