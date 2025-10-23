package user

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

func (s stubClock) Now() time.Time {
	return s.now
}

type fakeRepo struct {
	users map[string]*User
	order []string
	seq   int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{users: make(map[string]*User)}
}

func (r *fakeRepo) Create(_ context.Context, user *User) (*User, error) {
	for _, u := range r.users {
		if u.Email == user.Email {
			return nil, ErrEmailAlreadyExists
		}
	}
	r.seq++
	id := "user-" + strconv.Itoa(r.seq)
	copy := *user
	copy.ID = id
	r.users[id] = &copy
	r.order = append(r.order, id)
	return cloneUser(&copy), nil
}

func (r *fakeRepo) Update(_ context.Context, user *User) (*User, error) {
	existing, ok := r.users[user.ID]
	if !ok {
		return nil, ErrUserNotFound
	}
	*existing = *user
	return cloneUser(existing), nil
}

func (r *fakeRepo) Delete(_ context.Context, id string) error {
	if _, ok := r.users[id]; !ok {
		return ErrUserNotFound
	}
	delete(r.users, id)
	for i, existingID := range r.order {
		if existingID == id {
			r.order = append(r.order[:i], r.order[i+1:]...)
			break
		}
	}
	return nil
}

func (r *fakeRepo) FindByID(_ context.Context, id string) (*User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return cloneUser(u), nil
}

func (r *fakeRepo) FindByEmail(_ context.Context, email string) (*User, error) {
	for _, u := range r.users {
		if u.Email == email {
			return cloneUser(u), nil
		}
	}
	return nil, ErrUserNotFound
}

func (r *fakeRepo) List(_ context.Context, filter ListUsersFilter) ([]*User, string, error) {
	var filtered []*User
	for _, id := range r.order {
		u := r.users[id]
		if filter.Status != nil && u.Status != *filter.Status {
			continue
		}
		filtered = append(filtered, cloneUser(u))
	}

	if filter.Offset > len(filtered) {
		return []*User{}, "", nil
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

func cloneUser(u *User) *User {
	if u == nil {
		return nil
	}
	copy := *u
	return &copy
}

func TestService_CreateUser_Success(t *testing.T) {
	t.Parallel()

	clk := stubClock{now: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
	repo := newFakeRepo()
	svc := NewService(repo, &clk)

	input := CreateUserInput{Email: " USER@example.com ", Name: "  John Doe  "}

	created, err := svc.CreateUser(context.Background(), input)
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	if created.Email != "user@example.com" {
		t.Errorf("expected normalized email, got %s", created.Email)
	}

	if created.Name != "John Doe" {
		t.Errorf("expected trimmed name, got %q", created.Name)
	}

	if created.Status != StatusActive {
		t.Errorf("expected status active, got %s", created.Status)
	}

	if created.CreatedAt != clk.now || created.UpdatedAt != clk.now {
		t.Errorf("expected timestamps to use clock, got %v and %v", created.CreatedAt, created.UpdatedAt)
	}
}

func TestService_CreateUser_DuplicateEmail(t *testing.T) {
	t.Parallel()

	clk := stubClock{now: time.Now()}
	repo := newFakeRepo()
	svc := NewService(repo, &clk)

	if _, err := svc.CreateUser(context.Background(), CreateUserInput{Email: "john@example.com", Name: "John"}); err != nil {
		t.Fatalf("unexpected error preparing data: %v", err)
	}

	_, err := svc.CreateUser(context.Background(), CreateUserInput{Email: "JOHN@example.com", Name: "Johnny"})
	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}
}

func TestService_UpdateUser_Success(t *testing.T) {
	t.Parallel()

	clk := stubClock{now: time.Now()}
	repo := newFakeRepo()
	svc := NewService(repo, &clk)

	created, err := svc.CreateUser(context.Background(), CreateUserInput{Email: "user@example.com", Name: "User"})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}

	newName := "New Name"
	newStatus := StatusInactive
	clk.now = clk.now.Add(time.Hour)

	updated, err := svc.UpdateUser(context.Background(), UpdateUserInput{ID: created.ID, Name: &newName, Status: &newStatus})
	if err != nil {
		t.Fatalf("UpdateUser returned error: %v", err)
	}

	if updated.Name != newName {
		t.Errorf("expected name %s, got %s", newName, updated.Name)
	}

	if updated.Status != newStatus {
		t.Errorf("expected status %s, got %s", newStatus, updated.Status)
	}

	if updated.UpdatedAt != clk.now {
		t.Errorf("expected UpdatedAt to use clock, got %v", updated.UpdatedAt)
	}
}

func TestService_UpdateUser_InvalidStatus(t *testing.T) {
	t.Parallel()

	clk := stubClock{now: time.Now()}
	repo := newFakeRepo()
	svc := NewService(repo, &clk)

	created, err := svc.CreateUser(context.Background(), CreateUserInput{Email: "user@example.com", Name: "User"})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}

	invalidStatus := Status("blocked")
	_, err = svc.UpdateUser(context.Background(), UpdateUserInput{ID: created.ID, Status: &invalidStatus})
	if !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestService_DeleteUser_InvalidID(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	clk := stubClock{now: time.Now()}
	svc := NewService(repo, &clk)

	err := svc.DeleteUser(context.Background(), DeleteUserInput{ID: ""})
	if !errors.Is(err, ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}

func TestService_GetUser_Success(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	clk := stubClock{now: time.Now()}
	svc := NewService(repo, &clk)

	created, err := svc.CreateUser(context.Background(), CreateUserInput{Email: "user@example.com", Name: "User"})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}

	found, err := svc.GetUser(context.Background(), GetUserInput{ID: created.ID})
	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}

	if found.ID != created.ID {
		t.Fatalf("expected ID %s, got %s", created.ID, found.ID)
	}
}

func TestService_GetUser_InvalidID(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	clk := stubClock{now: time.Now()}
	svc := NewService(repo, &clk)

	if _, err := svc.GetUser(context.Background(), GetUserInput{ID: "   "}); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}

func TestService_ListUsers_Defaults(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	clk := stubClock{now: time.Now()}
	svc := NewService(repo, &clk)

	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("User %d", i)
		email := fmt.Sprintf("user%d@example.com", i)
		if _, err := svc.CreateUser(context.Background(), CreateUserInput{Email: email, Name: name}); err != nil {
			t.Fatalf("CreateUser error: %v", err)
		}
	}

	result, err := svc.ListUsers(context.Background(), ListUsersInput{})
	if err != nil {
		t.Fatalf("ListUsers returned error: %v", err)
	}

	if len(result.Users) != 3 {
		t.Fatalf("expected 3 users, got %d", len(result.Users))
	}

	if result.NextPageToken != "" {
		t.Fatalf("expected no next token, got %s", result.NextPageToken)
	}
}

func TestService_ListUsers_PageSizeValidation(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	clk := stubClock{now: time.Now()}
	svc := NewService(repo, &clk)

	_, err := svc.ListUsers(context.Background(), ListUsersInput{PageSize: maxListPageSize + 1})
	if !errors.Is(err, ErrInvalidPageSize) {
		t.Fatalf("expected ErrInvalidPageSize, got %v", err)
	}
}

func TestService_ListUsers_PageTokenValidation(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	clk := stubClock{now: time.Now()}
	svc := NewService(repo, &clk)

	_, err := svc.ListUsers(context.Background(), ListUsersInput{PageToken: "abc"})
	if !errors.Is(err, ErrInvalidPageToken) {
		t.Fatalf("expected ErrInvalidPageToken, got %v", err)
	}
}

func TestService_ListUsers_FilterByStatus(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	clk := stubClock{now: time.Now()}
	svc := NewService(repo, &clk)

	if _, err := svc.CreateUser(context.Background(), CreateUserInput{Email: "active@example.com", Name: "Active"}); err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}

	created, err := svc.CreateUser(context.Background(), CreateUserInput{Email: "inactive@example.com", Name: "Inactive"})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}

	inactive := StatusInactive
	if _, err := svc.UpdateUser(context.Background(), UpdateUserInput{ID: created.ID, Status: &inactive}); err != nil {
		t.Fatalf("UpdateUser error: %v", err)
	}

	result, err := svc.ListUsers(context.Background(), ListUsersInput{Status: &inactive})
	if err != nil {
		t.Fatalf("ListUsers returned error: %v", err)
	}

	if len(result.Users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(result.Users))
	}

	if result.Users[0].Status != StatusInactive {
		t.Fatalf("expected inactive status, got %s", result.Users[0].Status)
	}
}
