package lib

import (
	"errors"
	"fmt"
	"sync"
)

// User represents a user in the system
type User struct {
	Username string
	Password string
	Roles    []string // Privileges associated with the user
}

// AuthManager handles user authentication and session management
type AuthManager struct {
	users map[string]*User
	mu    sync.Mutex
}

// NewAuthManager initializes an AuthManager
func NewAuthManager() *AuthManager {
	return &AuthManager{
		users: make(map[string]*User),
	}
}

// CreateUser adds a new user to the system
func (am *AuthManager) CreateUser(username, password string) error {
	am.mu.Lock()
	defer am.mu.Unlock()
	if _, exists := am.users[username]; exists {
		return errors.New("user already exists")
	}
	am.users[username] = &User{
		Username: username,
		Password: password,
	}
	return nil
}

// AlterUser changes the password for a user
func (am *AuthManager) AlterUser(username, newPassword string) error {
	am.mu.Lock()
	defer am.mu.Unlock()
	user, exists := am.users[username]
	if !exists {
		return errors.New("user not found")
	}
	user.Password = newPassword
	return nil
}

// DropUser removes a user from the system
func (am *AuthManager) DropUser(username string) error {
	am.mu.Lock()
	defer am.mu.Unlock()
	if _, exists := am.users[username]; !exists {
		return errors.New("user not found")
	}
	delete(am.users, username)
	return nil
}

// Grant adds a role/privilege to a user
func (am *AuthManager) Grant(username, role string) error {
	am.mu.Lock()
	defer am.mu.Unlock()
	user, exists := am.users[username]
	if !exists {
		return errors.New("user not found")
	}
	user.Roles = append(user.Roles, role)
	return nil
}

// Revoke removes a role/privilege from a user
func (am *AuthManager) Revoke(username, role string) error {
	am.mu.Lock()
	defer am.mu.Unlock()
	user, exists := am.users[username]
	if !exists {
		return errors.New("user not found")
	}
	for i, r := range user.Roles {
		if r == role {
			user.Roles = append(user.Roles[:i], user.Roles[i+1:]...)
			return nil
		}
	}
	return errors.New("role not found")
}

// Connect verifies user credentials and establishes a session
func (am *AuthManager) Connect(username, password string) error {
	am.mu.Lock()
	defer am.mu.Unlock()
	user, exists := am.users[username]
	if !exists || user.Password != password {
		return errors.New("invalid credentials")
	}
	fmt.Println("User connected:", username)
	return nil
}

// Disconnect ends a user's session
func (am *AuthManager) Disconnect(username string) {
	fmt.Println("User disconnected:", username)
}
