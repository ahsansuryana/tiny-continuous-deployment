package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/alexedwards/scs/v2"
	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const usernameKey contextKey = "username"

type Auth struct {
	sm       *scs.SessionManager
	username string
	password string
}

func New(sm *scs.SessionManager, username, password string) (*Auth, error) {
	if username == "" || password == "" {
		return nil, errors.New("ADMIN_USERNAME and ADMIN_PASSWORD must be set")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return &Auth{
		sm:       sm,
		username: username,
		password: string(hash),
	}, nil
}

func (a *Auth) Login(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	if username != a.username {
		return errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(a.password), []byte(password)); err != nil {
		return errors.New("invalid credentials")
	}

	if err := a.sm.RenewToken(r.Context()); err != nil {
		return err
	}

	a.sm.Put(r.Context(), "authenticated", true)
	a.sm.Put(r.Context(), "username", a.username)
	return nil
}

func (a *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	a.sm.Destroy(r.Context())
}

func (a *Auth) IsAuthenticated(r *http.Request) bool {
	return a.sm.GetBool(r.Context(), "authenticated")
}

func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		publicPaths := []string{"/login", "/api/", "/static/"}
		for _, p := range publicPaths {
			if r.URL.Path == p || strings.HasPrefix(r.URL.Path, p) {
				next.ServeHTTP(w, r)
				return
			}
		}

		if !a.IsAuthenticated(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		username := a.sm.GetString(r.Context(), "username")
		ctx := context.WithValue(r.Context(), usernameKey, username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *Auth) CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions || r.Method == http.MethodTrace {
			next.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/login" {
			next.ServeHTTP(w, r)
			return
		}

		token := r.FormValue("csrf_token")
		sessionToken, ok := a.sm.Get(r.Context(), "csrf_token").(string)
		if !ok || token == "" || token != sessionToken {
			http.Error(w, "CSRF token invalid", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func GenerateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (a *Auth) EnsureCSRFToken(r *http.Request) string {
	if token, ok := a.sm.Get(r.Context(), "csrf_token").(string); ok && token != "" {
		return token
	}
	token := GenerateCSRFToken()
	a.sm.Put(r.Context(), "csrf_token", token)
	return token
}

func (a *Auth) UpdatePassword(newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	a.password = string(hash)
	return nil
}

func (a *Auth) VerifyCurrentPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(a.password), []byte(password)) == nil
}

func (a *Auth) GetUsername(r *http.Request) string {
	if u, ok := r.Context().Value(usernameKey).(string); ok {
		return u
	}
	return ""
}
