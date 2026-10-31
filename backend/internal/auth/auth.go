// Package auth handles password hashing, JWT tokens and the request principal.
package auth

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword returns a bcrypt hash.
func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

// CheckPassword reports whether pw matches hash.
func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// Tokens issues and verifies HS256 access tokens.
type Tokens struct {
	secret []byte
	ttl    time.Duration
}

// NewTokens creates a token issuer.
func NewTokens(secret string, ttl time.Duration) *Tokens {
	return &Tokens{secret: []byte(secret), ttl: ttl}
}

// Issue returns a signed token for userID.
func (t *Tokens) Issue(userID int64) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(t.ttl)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
}

// Parse validates a token and returns its user id.
func (t *Tokens) Parse(tok string) (int64, error) {
	var claims jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(tok, &claims, func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return 0, err
	}
	id, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, errors.New("bad subject")
	}
	return id, nil
}

// Scope restricts which factories' data a user may see.
type Scope struct {
	All        bool
	FactoryIDs []int64
}

// Allows reports whether the scope includes factoryID.
func (s Scope) Allows(factoryID int64) bool {
	if s.All {
		return true
	}
	for _, id := range s.FactoryIDs {
		if id == factoryID {
			return true
		}
	}
	return false
}

// Principal is the authenticated caller.
type Principal struct {
	UserID      int64
	Username    string
	DisplayName string
	StaffNo     string
	IsAdmin     bool
	Scope       Scope
}

type ctxKey struct{}

// WithPrincipal stores p in ctx.
func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

// FromContext returns the principal, or nil for anonymous requests.
func FromContext(ctx context.Context) *Principal {
	p, _ := ctx.Value(ctxKey{}).(*Principal)
	return p
}
