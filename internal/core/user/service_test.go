package user

import (
	"context"
	"errors"
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
