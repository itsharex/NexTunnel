package dashboard

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// AuthConfig configures dashboard authentication.
type AuthConfig struct {
	SecretKey    string
	TokenExpiry  time.Duration
	DefaultAdmin string
	DefaultPass  string
}

// DefaultAuthConfig returns default auth configuration.
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		SecretKey:    "",
		TokenExpiry:  24 * time.Hour,
		DefaultAdmin: "admin",
		DefaultPass:  "",
	}
}

// AuthManager handles user authentication and token management.
type AuthManager struct {
	config AuthConfig
	mu     sync.RWMutex
	users  map[string]*User
	tokens map[string]*tokenInfo
}

type tokenInfo struct {
	UserID    string
	Role      string
	ExpiresAt time.Time
}

// NewAuthManager creates a new auth manager with default admin user.
func NewAuthManager(cfg AuthConfig) *AuthManager {
	a := &AuthManager{
		config: cfg,
		users:  make(map[string]*User),
		tokens: make(map[string]*tokenInfo),
	}
	if cfg.SecretKey == "" {
		cfg.SecretKey = "test-only-empty-secret-rejected-on-server-start"
		a.config = cfg
	}
	if cfg.DefaultAdmin != "" && cfg.DefaultPass != "" {
		if err := a.AddUserWithPassword(&User{
			ID:       "admin-1",
			Username: cfg.DefaultAdmin,
			Role:     "admin",
			Email:    "admin@nextunnel.local",
		}, cfg.DefaultPass); err != nil {
			panic(err)
		}
	}
	return a
}

// Login authenticates a user and returns a token.
func (a *AuthManager) Login(req LoginRequest) (*LoginResponse, error) {
	a.mu.RLock()
	user, ok := a.users[req.Username]
	a.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("invalid credentials")
	}

	if user.PasswordHash == "" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	token := a.generateToken(user.ID)
	expiresAt := time.Now().Add(a.config.TokenExpiry)

	a.mu.Lock()
	a.tokens[token] = &tokenInfo{
		UserID:    user.ID,
		Role:      user.Role,
		ExpiresAt: expiresAt,
	}
	a.mu.Unlock()

	return &LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
	}, nil
}

// ValidateToken checks if a token is valid and returns the associated user info.
func (a *AuthManager) ValidateToken(token string) (*User, error) {
	a.mu.RLock()
	info, ok := a.tokens[token]
	a.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("invalid token")
	}

	if time.Now().After(info.ExpiresAt) {
		a.mu.Lock()
		delete(a.tokens, token)
		a.mu.Unlock()
		return nil, fmt.Errorf("token expired")
	}

	// Find user
	a.mu.RLock()
	for _, u := range a.users {
		if u.ID == info.UserID {
			a.mu.RUnlock()
			return u, nil
		}
	}
	a.mu.RUnlock()

	return nil, fmt.Errorf("user not found")
}

// AddUser adds a new dashboard user.
func (a *AuthManager) AddUser(user *User) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, exists := a.users[user.Username]; exists {
		return fmt.Errorf("user %q already exists", user.Username)
	}
	a.users[user.Username] = user
	return nil
}

// AddUserWithPassword adds a user after hashing the plaintext password.
func (a *AuthManager) AddUserWithPassword(user *User, password string) error {
	if password == "" {
		return fmt.Errorf("password must not be empty")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	cloned := *user
	cloned.PasswordHash = string(hash)
	return a.AddUser(&cloned)
}

// ListUsers returns all users.
func (a *AuthManager) ListUsers() []*User {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]*User, 0, len(a.users))
	for _, u := range a.users {
		result = append(result, u)
	}
	return result
}

// UpdateUserRole changes the role of an existing user.
func (a *AuthManager) UpdateUserRole(username, role string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	user, ok := a.users[username]
	if !ok {
		return fmt.Errorf("user %q not found", username)
	}
	user.Role = role
	return nil
}

// RemoveUser deletes a user by username.
func (a *AuthManager) RemoveUser(username string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.users, username)
}

// ReplaceUsers 用持久化用户表重建内存索引，服务重启后仍可继续登录。
func (a *AuthManager) ReplaceUsers(users []*User) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.users = make(map[string]*User, len(users))
	for _, user := range users {
		if user == nil || user.Username == "" {
			continue
		}
		cloned := *user
		a.users[cloned.Username] = &cloned
	}
}

func (a *AuthManager) generateToken(userID string) string {
	h := hmac.New(sha256.New, []byte(a.config.SecretKey))
	h.Write([]byte(userID))
	h.Write([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return hex.EncodeToString(h.Sum(nil))
}

// AuthMiddleware is an HTTP middleware that validates JWT-like tokens.
func (a *AuthManager) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 静态资源、预检请求、登录和健康检查必须允许匿名访问，其他 API 保持鉴权。
		if !strings.HasPrefix(r.URL.Path, "/api/") ||
			r.Method == http.MethodOptions ||
			r.URL.Path == "/api/v1/auth/login" ||
			r.URL.Path == "/api/v1/health" {
			next.ServeHTTP(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		if auth == "" {
			http.Error(w, `{"error":"missing authorization"}`, http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if token == auth {
			http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
			return
		}

		user, err := a.ValidateToken(token)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusUnauthorized)
			return
		}

		// Set user info in context (simplified)
		r.Header.Set("X-User-ID", user.ID)
		r.Header.Set("X-User-Role", user.Role)
		next.ServeHTTP(w, r)
	})
}
