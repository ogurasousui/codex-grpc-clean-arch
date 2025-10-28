package employee

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

type fakeEmployeeRepo struct {
	employees map[string]*Employee
	sequence  int
	order     []string
}

func newFakeEmployeeRepo() *fakeEmployeeRepo {
	return &fakeEmployeeRepo{employees: make(map[string]*Employee)}
}

func (r *fakeEmployeeRepo) Create(_ context.Context, e *Employee) (*Employee, error) {
	for _, existing := range r.employees {
		if existing.CompanyID == e.CompanyID && existing.EmployeeCode == e.EmployeeCode {
			return nil, ErrEmployeeCodeAlreadyExists
		}
	}

	clone := cloneEmployee(e)
	r.sequence++
	id := fmt.Sprintf("emp-%d", r.sequence)
	clone.ID = id
	r.employees[id] = clone
	r.order = append(r.order, id)
	return cloneEmployee(clone), nil
}

func (r *fakeEmployeeRepo) Update(_ context.Context, e *Employee) (*Employee, error) {
	if _, ok := r.employees[e.ID]; !ok {
		return nil, ErrEmployeeNotFound
	}
	for _, existing := range r.employees {
		if existing.ID != e.ID && existing.CompanyID == e.CompanyID && existing.EmployeeCode == e.EmployeeCode {
			return nil, ErrEmployeeCodeAlreadyExists
		}
	}
	r.employees[e.ID] = cloneEmployee(e)
	return cloneEmployee(e), nil
}

func (r *fakeEmployeeRepo) Delete(_ context.Context, id string) error {
	if _, ok := r.employees[id]; !ok {
		return ErrEmployeeNotFound
	}
	delete(r.employees, id)
	for idx, existingID := range r.order {
		if existingID == id {
			r.order = append(r.order[:idx], r.order[idx+1:]...)
			break
		}
	}
	return nil
}

func (r *fakeEmployeeRepo) FindByID(_ context.Context, id string) (*Employee, error) {
	emp, ok := r.employees[id]
	if !ok {
		return nil, ErrEmployeeNotFound
	}
	return cloneEmployee(emp), nil
}

func (r *fakeEmployeeRepo) FindByCompanyAndCode(_ context.Context, companyID, code string) (*Employee, error) {
	for _, emp := range r.employees {
		if emp.CompanyID == companyID && emp.EmployeeCode == code {
			return cloneEmployee(emp), nil
		}
	}
	return nil, ErrEmployeeNotFound
}

func (r *fakeEmployeeRepo) List(_ context.Context, filter ListEmployeesFilter) ([]*Employee, string, error) {
	var filtered []*Employee
	for _, id := range r.order {
		emp := r.employees[id]
		if emp.CompanyID != filter.CompanyID {
			continue
		}
		if filter.Status != nil && emp.Status != *filter.Status {
			continue
		}
		filtered = append(filtered, cloneEmployee(emp))
	}

	if filter.Offset > len(filtered) {
		return []*Employee{}, "", nil
	}

	end := filter.Offset + filter.Limit
	if end > len(filtered) {
		end = len(filtered)
	}

	page := filtered[filter.Offset:end]

	nextToken := ""
	if end < len(filtered) {
		nextToken = strconv.Itoa(end)
	}

	return page, nextToken, nil
}

func cloneEmployee(emp *Employee) *Employee {
	if emp == nil {
		return nil
	}
	copy := *emp
	if emp.Email != nil {
		email := *emp.Email
		copy.Email = &email
	}
	if emp.HiredAt != nil {
		hired := *emp.HiredAt
		copy.HiredAt = &hired
	}
	if emp.TerminatedAt != nil {
		terminated := *emp.TerminatedAt
		copy.TerminatedAt = &terminated
	}
	return &copy
}

func TestService_CreateEmployee_Success(t *testing.T) {
	t.Parallel()

	repo := newFakeEmployeeRepo()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	svc := NewService(repo, &stubClock{now: now}, nil)

	email := "Example@Example.com"
	hired := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)

	created, err := svc.CreateEmployee(context.Background(), CreateEmployeeInput{
		CompanyID:    " company-1 ",
		EmployeeCode: " Emp-001 ",
		Email:        &email,
		LastName:     "  Yamada  ",
		FirstName:    " Taro ",
		HiredAt:      &hired,
	})
	if err != nil {
		t.Fatalf("CreateEmployee returned error: %v", err)
	}

	if created.CompanyID != "company-1" {
		t.Fatalf("expected normalized company id, got %s", created.CompanyID)
	}
	if created.EmployeeCode != "emp-001" {
		t.Fatalf("expected normalized employee code, got %s", created.EmployeeCode)
	}
	if created.Email == nil || *created.Email != "example@example.com" {
		t.Fatalf("expected normalized email, got %+v", created.Email)
	}
	if created.LastName != "Yamada" || created.FirstName != "Taro" {
		t.Fatalf("expected trimmed names, got %s %s", created.LastName, created.FirstName)
	}
	if created.Status != StatusActive {
		t.Fatalf("expected default status active, got %s", created.Status)
	}
	if created.HiredAt == nil || !created.HiredAt.Equal(time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected hired_at: %+v", created.HiredAt)
	}
	if !created.CreatedAt.Equal(now) || !created.UpdatedAt.Equal(now) {
		t.Fatalf("expected timestamps to use clock now")
	}
}

func TestService_CreateEmployee_DuplicateCode(t *testing.T) {
	t.Parallel()

	repo := newFakeEmployeeRepo()
	svc := NewService(repo, &stubClock{now: time.Now().UTC()}, nil)

	if _, err := svc.CreateEmployee(context.Background(), CreateEmployeeInput{
		CompanyID:    "company-1",
		EmployeeCode: "emp-1",
		LastName:     "Test",
		FirstName:    "User",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := svc.CreateEmployee(context.Background(), CreateEmployeeInput{
		CompanyID:    "company-1",
		EmployeeCode: "EMP-1",
		LastName:     "Another",
		FirstName:    "User",
	})
	if !errors.Is(err, ErrEmployeeCodeAlreadyExists) {
		t.Fatalf("expected ErrEmployeeCodeAlreadyExists, got %v", err)
	}
}

func TestService_CreateEmployee_InvalidDateRange(t *testing.T) {
	t.Parallel()

	repo := newFakeEmployeeRepo()
	svc := NewService(repo, &stubClock{now: time.Now().UTC()}, nil)

	hired := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	terminated := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	_, err := svc.CreateEmployee(context.Background(), CreateEmployeeInput{
		CompanyID:    "company-1",
		EmployeeCode: "emp-2",
		LastName:     "Test",
		FirstName:    "User",
		HiredAt:      &hired,
		TerminatedAt: &terminated,
	})
	if !errors.Is(err, ErrInvalidDateRange) {
		t.Fatalf("expected ErrInvalidDateRange, got %v", err)
	}
}

func TestService_UpdateEmployee_Success(t *testing.T) {
	t.Parallel()

	repo := newFakeEmployeeRepo()
	clk := &stubClock{now: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)}
	svc := NewService(repo, clk, nil)

	created, err := svc.CreateEmployee(context.Background(), CreateEmployeeInput{
		CompanyID:    "company-1",
		EmployeeCode: "emp-3",
		LastName:     "Suzuki",
		FirstName:    "Ichiro",
	})
	if err != nil {
		t.Fatalf("CreateEmployee returned error: %v", err)
	}

	clk.now = clk.now.Add(time.Hour)

	newCode := "EMP-999"
	newEmail := "ichiro+update@example.com"
	newLast := "  Sato  "
	newFirst := "  Hanako  "
	newStatus := StatusInactive
	hired := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	terminated := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	updated, err := svc.UpdateEmployee(context.Background(), UpdateEmployeeInput{
		ID:              created.ID,
		EmployeeCode:    &newCode,
		Email:           &newEmail,
		LastName:        &newLast,
		FirstName:       &newFirst,
		Status:          &newStatus,
		HiredAt:         &hired,
		HiredAtSet:      true,
		TerminatedAt:    &terminated,
		TerminatedAtSet: true,
	})
	if err != nil {
		t.Fatalf("UpdateEmployee returned error: %v", err)
	}

	if updated.EmployeeCode != "emp-999" {
		t.Fatalf("expected normalized code in update, got %s", updated.EmployeeCode)
	}
	if updated.Email == nil || *updated.Email != "ichiro+update@example.com" {
		t.Fatalf("expected updated email, got %+v", updated.Email)
	}
	if updated.LastName != "Sato" || updated.FirstName != "Hanako" {
		t.Fatalf("expected trimmed names, got %s %s", updated.LastName, updated.FirstName)
	}
	if updated.Status != StatusInactive {
		t.Fatalf("expected status inactive, got %s", updated.Status)
	}
	if updated.HiredAt == nil || !updated.HiredAt.Equal(hired) {
		t.Fatalf("expected hired date to update, got %+v", updated.HiredAt)
	}
	if updated.TerminatedAt == nil || !updated.TerminatedAt.Equal(terminated) {
		t.Fatalf("expected terminated date to update, got %+v", updated.TerminatedAt)
	}
	if !updated.UpdatedAt.Equal(clk.now) {
		t.Fatalf("expected updated timestamp to use clock")
	}
}

func TestService_UpdateEmployee_InvalidStatus(t *testing.T) {
	t.Parallel()

	repo := newFakeEmployeeRepo()
	svc := NewService(repo, &stubClock{now: time.Now().UTC()}, nil)

	created, err := svc.CreateEmployee(context.Background(), CreateEmployeeInput{
		CompanyID:    "company-1",
		EmployeeCode: "emp-4",
		LastName:     "Foo",
		FirstName:    "Bar",
	})
	if err != nil {
		t.Fatalf("CreateEmployee returned error: %v", err)
	}

	invalidStatus := Status("unknown")
	_, err = svc.UpdateEmployee(context.Background(), UpdateEmployeeInput{ID: created.ID, Status: &invalidStatus})
	if !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestService_ListEmployees_FilterAndPagination(t *testing.T) {
	t.Parallel()

	repo := newFakeEmployeeRepo()
	svc := NewService(repo, &stubClock{now: time.Now().UTC()}, nil)

	// seed
	statuses := []Status{StatusActive, StatusInactive, StatusActive}
	for i := 0; i < 3; i++ {
		status := statuses[i]
		if _, err := svc.CreateEmployee(context.Background(), CreateEmployeeInput{
			CompanyID:    "company-1",
			EmployeeCode: fmt.Sprintf("emp-%d", i),
			LastName:     "Seed",
			FirstName:    fmt.Sprintf("User%d", i),
			Status:       &status,
		}); err != nil {
			t.Fatalf("unexpected seed error: %v", err)
		}
	}

	inactive := StatusInactive
	result, err := svc.ListEmployees(context.Background(), ListEmployeesInput{
		CompanyID: "company-1",
		PageSize:  2,
		Status:    &inactive,
	})
	if err != nil {
		t.Fatalf("ListEmployees returned error: %v", err)
	}
	if len(result.Employees) != 1 {
		t.Fatalf("expected 1 inactive employee, got %d", len(result.Employees))
	}

	active := StatusActive
	page1, err := svc.ListEmployees(context.Background(), ListEmployeesInput{
		CompanyID: "company-1",
		PageSize:  1,
		Status:    &active,
	})
	if err != nil {
		t.Fatalf("ListEmployees active returned error: %v", err)
	}
	if len(page1.Employees) != 1 {
		t.Fatalf("expected first page to have 1 employee, got %d", len(page1.Employees))
	}
	if page1.NextPageToken == "" {
		t.Fatalf("expected next page token")
	}

	page2, err := svc.ListEmployees(context.Background(), ListEmployeesInput{
		CompanyID: "company-1",
		PageSize:  1,
		PageToken: page1.NextPageToken,
		Status:    &active,
	})
	if err != nil {
		t.Fatalf("ListEmployees page2 returned error: %v", err)
	}
	if len(page2.Employees) != 1 {
		t.Fatalf("expected second page to have 1 employee, got %d", len(page2.Employees))
	}

	if page2.NextPageToken != "" {
		page3, err := svc.ListEmployees(context.Background(), ListEmployeesInput{
			CompanyID: "company-1",
			PageSize:  1,
			PageToken: page2.NextPageToken,
			Status:    &active,
		})
		if err != nil {
			t.Fatalf("ListEmployees page3 returned error: %v", err)
		}
		if len(page3.Employees) != 0 {
			t.Fatalf("expected no more employees, got %d", len(page3.Employees))
		}
	}
}

func TestService_ListEmployees_InvalidCompanyID(t *testing.T) {
	t.Parallel()

	repo := newFakeEmployeeRepo()
	svc := NewService(repo, &stubClock{now: time.Now().UTC()}, nil)

	_, err := svc.ListEmployees(context.Background(), ListEmployeesInput{CompanyID: ""})
	if !errors.Is(err, ErrInvalidCompanyID) {
		t.Fatalf("expected ErrInvalidCompanyID, got %v", err)
	}
}
