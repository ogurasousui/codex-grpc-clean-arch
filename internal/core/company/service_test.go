package company

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"
)

type stubClock struct {
	now time.Time
}

func (s *stubClock) Now() time.Time {
	return s.now
}

type fakeRepo struct {
	companies map[string]*Company
	order     []string
	seq       int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{companies: make(map[string]*Company)}
}

func (r *fakeRepo) Create(_ context.Context, company *Company) (*Company, error) {
	for _, c := range r.companies {
		if c.Code == company.Code {
			return nil, ErrCodeAlreadyExists
		}
	}
	clone := cloneCompany(company)
	r.seq++
	id := fmt.Sprintf("company-%d", r.seq)
	clone.ID = id
	r.companies[id] = clone
	r.order = append(r.order, id)
	return cloneCompany(clone), nil
}

func (r *fakeRepo) Update(_ context.Context, company *Company) (*Company, error) {
	if _, ok := r.companies[company.ID]; !ok {
		return nil, ErrCompanyNotFound
	}
	for _, c := range r.companies {
		if c.ID != company.ID && c.Code == company.Code {
			return nil, ErrCodeAlreadyExists
		}
	}
	r.companies[company.ID] = cloneCompany(company)
	return cloneCompany(company), nil
}

func (r *fakeRepo) Delete(_ context.Context, id string) error {
	if _, ok := r.companies[id]; !ok {
		return ErrCompanyNotFound
	}
	delete(r.companies, id)
	for i, existingID := range r.order {
		if existingID == id {
			r.order = append(r.order[:i], r.order[i+1:]...)
			break
		}
	}
	return nil
}

func (r *fakeRepo) FindByID(_ context.Context, id string) (*Company, error) {
	company, ok := r.companies[id]
	if !ok {
		return nil, ErrCompanyNotFound
	}
	return cloneCompany(company), nil
}

func (r *fakeRepo) FindByCode(_ context.Context, code string) (*Company, error) {
	for _, company := range r.companies {
		if company.Code == code {
			return cloneCompany(company), nil
		}
	}
	return nil, ErrCompanyNotFound
}

func (r *fakeRepo) List(_ context.Context, filter ListCompaniesFilter) ([]*Company, string, error) {
	var filtered []*Company
	for _, id := range r.order {
		company := r.companies[id]
		if filter.Status != nil && company.Status != *filter.Status {
			continue
		}
		filtered = append(filtered, cloneCompany(company))
	}

	if filter.Offset > len(filtered) {
		return []*Company{}, "", nil
	}

	end := filter.Offset + filter.Limit
	if end > len(filtered) {
		end = len(filtered)
	}

	page := filtered[filter.Offset:end]

	var nextToken string
	if end < len(filtered) {
		nextToken = strconv.Itoa(end)
	}

	return page, nextToken, nil
}

func cloneCompany(company *Company) *Company {
	if company == nil {
		return nil
	}
	copy := *company
	if company.Description != nil {
		desc := *company.Description
		copy.Description = &desc
	}
	return &copy
}

func TestService_CreateCompany_Success(t *testing.T) {
	t.Parallel()

	desc := "  Leading company description "
	clk := &stubClock{now: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
	repo := newFakeRepo()
	svc := NewService(repo, clk, nil)

	created, err := svc.CreateCompany(context.Background(), CreateCompanyInput{
		Name:        "  Example Inc.  ",
		Code:        " Example-Inc ",
		Description: &desc,
	})
	if err != nil {
		t.Fatalf("CreateCompany returned error: %v", err)
	}

	if created.Name != "Example Inc." {
		t.Fatalf("expected trimmed name, got %q", created.Name)
	}

	if created.Code != "example-inc" {
		t.Fatalf("expected normalized code 'example-inc', got %s", created.Code)
	}

	if created.Description == nil || *created.Description != "Leading company description" {
		t.Fatalf("expected trimmed description, got %+v", created.Description)
	}

	if created.Status != StatusActive {
		t.Fatalf("expected status active, got %s", created.Status)
	}

	if !created.CreatedAt.Equal(clk.now) || !created.UpdatedAt.Equal(clk.now) {
		t.Fatalf("expected timestamps to use clock, got %v and %v", created.CreatedAt, created.UpdatedAt)
	}
}

func TestService_CreateCompany_InvalidCode(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	svc := NewService(repo, &stubClock{now: time.Now()}, nil)

	if _, err := svc.CreateCompany(context.Background(), CreateCompanyInput{Name: "Test", Code: "Invalid Code"}); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("expected ErrInvalidCode, got %v", err)
	}
}

func TestService_CreateCompany_DuplicateCode(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	svc := NewService(repo, &stubClock{now: time.Now()}, nil)

	if _, err := svc.CreateCompany(context.Background(), CreateCompanyInput{Name: "Test", Code: "dup"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := svc.CreateCompany(context.Background(), CreateCompanyInput{Name: "Another", Code: "DUP"}); !errors.Is(err, ErrCodeAlreadyExists) {
		t.Fatalf("expected ErrCodeAlreadyExists, got %v", err)
	}
}

func TestService_UpdateCompany_Success(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	clk := &stubClock{now: time.Now()}
	svc := NewService(repo, clk, nil)

	created, err := svc.CreateCompany(context.Background(), CreateCompanyInput{Name: "Test", Code: "test"})
	if err != nil {
		t.Fatalf("CreateCompany error: %v", err)
	}

	newName := "New Name"
	newCode := "new-code"
	inactive := StatusInactive
	empty := ""
	clk.now = clk.now.Add(time.Hour)

	updated, err := svc.UpdateCompany(context.Background(), UpdateCompanyInput{
		ID:          created.ID,
		Name:        &newName,
		Code:        &newCode,
		Status:      &inactive,
		Description: &empty,
	})
	if err != nil {
		t.Fatalf("UpdateCompany returned error: %v", err)
	}

	if updated.Name != newName {
		t.Fatalf("expected name %s, got %s", newName, updated.Name)
	}

	if updated.Code != newCode {
		t.Fatalf("expected code %s, got %s", newCode, updated.Code)
	}

	if updated.Status != StatusInactive {
		t.Fatalf("expected status inactive, got %s", updated.Status)
	}

	if updated.Description != nil {
		t.Fatalf("expected description cleared, got %+v", updated.Description)
	}

	if !updated.UpdatedAt.Equal(clk.now) {
		t.Fatalf("expected updated timestamp to match clock, got %v", updated.UpdatedAt)
	}
}

func TestService_UpdateCompany_InvalidCode(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	svc := NewService(repo, &stubClock{now: time.Now()}, nil)

	created, err := svc.CreateCompany(context.Background(), CreateCompanyInput{Name: "Test", Code: "valid-code"})
	if err != nil {
		t.Fatalf("CreateCompany error: %v", err)
	}

	invalid := "INVALID CODE"
	_, err = svc.UpdateCompany(context.Background(), UpdateCompanyInput{ID: created.ID, Code: &invalid})
	if !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("expected ErrInvalidCode, got %v", err)
	}
}

func TestService_UpdateCompany_DuplicateCode(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	svc := NewService(repo, &stubClock{now: time.Now()}, nil)

	first, err := svc.CreateCompany(context.Background(), CreateCompanyInput{Name: "First", Code: "first"})
	if err != nil {
		t.Fatalf("CreateCompany error: %v", err)
	}

	second, err := svc.CreateCompany(context.Background(), CreateCompanyInput{Name: "Second", Code: "second"})
	if err != nil {
		t.Fatalf("CreateCompany error: %v", err)
	}

	newCode := first.Code
	_, err = svc.UpdateCompany(context.Background(), UpdateCompanyInput{ID: second.ID, Code: &newCode})
	if !errors.Is(err, ErrCodeAlreadyExists) {
		t.Fatalf("expected ErrCodeAlreadyExists, got %v", err)
	}
}

func TestService_DeleteCompany_InvalidID(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	svc := NewService(repo, &stubClock{now: time.Now()}, nil)

	err := svc.DeleteCompany(context.Background(), DeleteCompanyInput{ID: ""})
	if !errors.Is(err, ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}

func TestService_GetCompany_Success(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	svc := NewService(repo, &stubClock{now: time.Now()}, nil)

	created, err := svc.CreateCompany(context.Background(), CreateCompanyInput{Name: "Test", Code: "test"})
	if err != nil {
		t.Fatalf("CreateCompany error: %v", err)
	}

	found, err := svc.GetCompany(context.Background(), GetCompanyInput{ID: created.ID})
	if err != nil {
		t.Fatalf("GetCompany returned error: %v", err)
	}

	if found.ID != created.ID {
		t.Fatalf("expected ID %s, got %s", created.ID, found.ID)
	}
}

func TestService_GetCompany_InvalidID(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	svc := NewService(repo, &stubClock{now: time.Now()}, nil)

	if _, err := svc.GetCompany(context.Background(), GetCompanyInput{ID: "   "}); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}

func TestService_ListCompanies_Defaults(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	svc := NewService(repo, &stubClock{now: time.Now()}, nil)

	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("Company %d", i)
		code := fmt.Sprintf("company-%d", i)
		if _, err := svc.CreateCompany(context.Background(), CreateCompanyInput{Name: name, Code: code}); err != nil {
			t.Fatalf("CreateCompany error: %v", err)
		}
	}

	result, err := svc.ListCompanies(context.Background(), ListCompaniesInput{})
	if err != nil {
		t.Fatalf("ListCompanies returned error: %v", err)
	}

	if len(result.Companies) != 3 {
		t.Fatalf("expected 3 companies, got %d", len(result.Companies))
	}

	if result.NextPageToken != "" {
		t.Fatalf("expected no next token, got %s", result.NextPageToken)
	}
}

func TestService_ListCompanies_Pagination(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	svc := NewService(repo, &stubClock{now: time.Now()}, nil)

	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("Company %d", i)
		code := fmt.Sprintf("company-%d", i)
		if _, err := svc.CreateCompany(context.Background(), CreateCompanyInput{Name: name, Code: code}); err != nil {
			t.Fatalf("CreateCompany error: %v", err)
		}
	}

	result, err := svc.ListCompanies(context.Background(), ListCompaniesInput{PageSize: 2})
	if err != nil {
		t.Fatalf("ListCompanies returned error: %v", err)
	}

	if len(result.Companies) != 2 {
		t.Fatalf("expected 2 companies, got %d", len(result.Companies))
	}

	if result.NextPageToken != "2" {
		t.Fatalf("expected next token 2, got %s", result.NextPageToken)
	}
}

func TestService_ListCompanies_PageSizeValidation(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	svc := NewService(repo, &stubClock{now: time.Now()}, nil)

	_, err := svc.ListCompanies(context.Background(), ListCompaniesInput{PageSize: maxListPageSize + 1})
	if !errors.Is(err, ErrInvalidPageSize) {
		t.Fatalf("expected ErrInvalidPageSize, got %v", err)
	}
}

func TestService_ListCompanies_PageTokenValidation(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	svc := NewService(repo, &stubClock{now: time.Now()}, nil)

	_, err := svc.ListCompanies(context.Background(), ListCompaniesInput{PageToken: "abc"})
	if !errors.Is(err, ErrInvalidPageToken) {
		t.Fatalf("expected ErrInvalidPageToken, got %v", err)
	}
}

func TestService_ListCompanies_FilterByStatus(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	svc := NewService(repo, &stubClock{now: time.Now()}, nil)

	if _, err := svc.CreateCompany(context.Background(), CreateCompanyInput{Name: "Active", Code: "active"}); err != nil {
		t.Fatalf("CreateCompany error: %v", err)
	}

	inactiveCompany, err := svc.CreateCompany(context.Background(), CreateCompanyInput{Name: "Inactive", Code: "inactive"})
	if err != nil {
		t.Fatalf("CreateCompany error: %v", err)
	}

	inactive := StatusInactive
	if _, err := svc.UpdateCompany(context.Background(), UpdateCompanyInput{ID: inactiveCompany.ID, Status: &inactive}); err != nil {
		t.Fatalf("UpdateCompany error: %v", err)
	}

	result, err := svc.ListCompanies(context.Background(), ListCompaniesInput{Status: &inactive})
	if err != nil {
		t.Fatalf("ListCompanies returned error: %v", err)
	}

	if len(result.Companies) != 1 {
		t.Fatalf("expected 1 company, got %d", len(result.Companies))
	}

	if result.Companies[0].Status != StatusInactive {
		t.Fatalf("expected inactive status, got %s", result.Companies[0].Status)
	}
}
