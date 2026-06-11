// Package auth holds the admin authentication use cases: first-run init,
// login, password reset, and bearer-token verification. Credential primitives
// (bcrypt, JWT) live in internal/auth; this service orchestrates them over a
// Repo it defines.
package auth

import (
	"github.com/thomasstefen/server-monitor/internal/domain"
)

// Hasher issues and verifies password hashes and session tokens. The
// internal/auth.Auth type satisfies it.
type Hasher interface {
	Hash(password string) ([]byte, error)
	Check(hash []byte, password string) bool
	IssueToken() (string, error)
	Valid(token string) bool
}

// Repo is the admin-credential persistence the service needs.
type Repo interface {
	IsInitialized() (bool, error)
	SetPasswordHash(hash []byte) error
	PasswordHash() ([]byte, bool, error)
}

// Service implements the admin auth use cases.
type Service struct {
	repo   Repo
	hasher Hasher
}

// New constructs the auth service.
func New(repo Repo, hasher Hasher) *Service {
	return &Service{repo: repo, hasher: hasher}
}

// Initialized reports whether an admin password has been set.
func (s *Service) Initialized() (bool, error) { return s.repo.IsInitialized() }

// Initialize sets the first-run admin password. Returns domain.ErrConflict if
// already initialized, domain.ErrInvalidInput for an empty password.
func (s *Service) Initialize(password string) error {
	if password == "" {
		return domain.ErrInvalidInput
	}
	init, err := s.repo.IsInitialized()
	if err != nil {
		return err
	}
	if init {
		return domain.ErrConflict
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return err
	}
	return s.repo.SetPasswordHash(hash)
}

// Login verifies the password and returns a session token, or
// domain.ErrUnauthorized.
func (s *Service) Login(password string) (string, error) {
	if password == "" {
		return "", domain.ErrInvalidInput
	}
	hash, ok, err := s.repo.PasswordHash()
	if err != nil {
		return "", err
	}
	if !ok || !s.hasher.Check(hash, password) {
		return "", domain.ErrUnauthorized
	}
	return s.hasher.IssueToken()
}

// ResetPassword changes the admin password after verifying the current one.
func (s *Service) ResetPassword(oldPassword, newPassword string) error {
	if oldPassword == "" || newPassword == "" {
		return domain.ErrInvalidInput
	}
	hash, ok, err := s.repo.PasswordHash()
	if err != nil {
		return err
	}
	if !ok || !s.hasher.Check(hash, oldPassword) {
		return domain.ErrUnauthorized
	}
	newHash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return err
	}
	return s.repo.SetPasswordHash(newHash)
}

// ValidToken reports whether a bearer token is valid.
func (s *Service) ValidToken(token string) bool { return s.hasher.Valid(token) }
