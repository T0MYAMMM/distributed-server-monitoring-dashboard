// Package auth handles admin password hashing (bcrypt) and stateless session
// tokens (JWT HS256), mirroring the original Flask behaviour.
package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Auth issues and verifies JWTs and hashes/checks passwords.
type Auth struct {
	secret []byte
}

// New returns an Auth bound to the given signing secret.
func New(secret []byte) *Auth { return &Auth{secret: secret} }

// Hash returns a bcrypt hash of the password.
func (a *Auth) Hash(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

// Check verifies a password against a stored bcrypt hash.
func (a *Auth) Check(hash []byte, password string) bool {
	return bcrypt.CompareHashAndPassword(hash, []byte(password)) == nil
}

// IssueToken returns a signed token valid for one day.
func (a *Auth) IssueToken() (string, error) {
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.secret)
}

// Valid reports whether a token string is a well-formed, unexpired token
// signed with our secret.
func (a *Auth) Valid(token string) bool {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return a.secret, nil
	})
	return err == nil && parsed.Valid
}
