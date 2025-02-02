package auth

import (
	"sync"
	"time"
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

	// Check username
	if _, exists := r.users[user.Username]; exists {
		return ErrUserExists
	}

	// Check email
	if _, exists := r.usersByEmail[user.Email]; exists {
		return ErrUserExists
	}

	// Clone the user to prevent external modifications
	newUser := &User{
		ID:               uint(len(r.users) + 1), // Simple ID generation
		Username:         user.Username,
		PasswordHash:     user.PasswordHash,
		Email:            user.Email,
		FailedLoginCount: 0,
		Locked:           false,
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

func (r *mockRepository) UpdateLoginAttempts(userID uint, failed bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Find user by ID
	var user *User
	for _, u := range r.users {
		if u.ID == userID {
			user = u
			break
		}
	}
	if user == nil {
		return ErrUserNotFound
	}

	now := time.Now()
	user.LastLoginAttempt = &now

	if failed {
		user.FailedLoginCount++
	} else {
		user.FailedLoginCount = 0
	}

	return nil
}

func (r *mockRepository) LockAccount(userID uint, duration time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Find user by ID
	var user *User
	for _, u := range r.users {
		if u.ID == userID {
			user = u
			break
		}
	}
	if user == nil {
		return ErrUserNotFound
	}

	user.Locked = true
	lockUntil := time.Now().Add(duration)
	user.LockUntil = &lockUntil

	return nil
}

func (r *mockRepository) UnlockAccount(userID uint) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Find user by ID
	var user *User
	for _, u := range r.users {
		if u.ID == userID {
			user = u
			break
		}
	}
	if user == nil {
		return ErrUserNotFound
	}

	user.Locked = false
	user.LockUntil = nil
	user.FailedLoginCount = 0

	return nil
}

func (r *mockRepository) VerifyEmail(userID uint) error {
	return nil
}
