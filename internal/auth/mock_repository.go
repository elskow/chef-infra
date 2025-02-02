package auth

import (
	"sync"
)

type mockRepository struct {
	users        map[string]*User
	usersByEmail map[string]*User
	mu           sync.RWMutex
}

func newMockRepository() Repository {
	return &mockRepository{
		users:        make(map[string]*User),
		usersByEmail: make(map[string]*User),
	}
}

func (r *mockRepository) CreateUser(user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.Username]; exists {
		return ErrUserExists
	}

	if _, exists := r.usersByEmail[user.Email]; exists {
		return ErrUserExists
	}

	newUser := &User{
		ID:           uint(len(r.users) + 1),
		Username:     user.Username,
		PasswordHash: user.PasswordHash,
		Email:        user.Email,
	}

	r.users[user.Username] = newUser
	r.usersByEmail[user.Email] = newUser
	return nil
}

func (r *mockRepository) GetUserByUsername(username string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[username]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (r *mockRepository) GetUserByEmail(email string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.usersByEmail[email]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (r *mockRepository) VerifyEmail(userID uint) error {
	return nil
}
