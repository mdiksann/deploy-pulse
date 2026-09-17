package app

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookieName = "deploypulse_session"
	sessionTTL        = 8 * time.Hour
)

func normalizeEmail(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || !strings.Contains(email, "@") || len(email) > 320 {
		return "", errors.New("enter a valid email address")
	}
	return email, nil
}

func validatePassword(value string) error {
	if len(value) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}

func passwordHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func bcryptCompare(hash, password string) error {
	if hash == "" {
		return errors.New("missing password hash")
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func tokenPair() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	raw := hex.EncodeToString(bytes)
	hash := sha256.Sum256([]byte(raw))
	return raw, hex.EncodeToString(hash[:]), nil
}

func hashToken(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

func newSessionToken() (string, string, error) {
	return tokenPair()
}

func publicUser(user User) User { return user }

func (s *Server) authenticateSession(r *http.Request) (User, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return User{}, false
	}
	user, err := s.store.SessionUser(r.Context(), hashToken(cookie.Value), time.Now().UTC())
	return user, err == nil
}

func (s *Server) requireSession(w http.ResponseWriter, r *http.Request) (User, bool) {
	user, ok := s.authenticateSession(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return User{}, false
	}
	return user, true
}

func (s *Server) issueSession(w http.ResponseWriter, r *http.Request, user User) error {
	raw, hashed, err := newSessionToken()
	if err != nil {
		return err
	}
	expires := time.Now().UTC().Add(sessionTTL)
	if err := s.store.CreateSession(r.Context(), hashed, user.ID, expires); err != nil {
		return err
	}
	httpOnlyCookie := &http.Cookie{Name: sessionCookieName, Value: raw, Path: "/", Expires: expires, MaxAge: int(sessionTTL.Seconds()), HttpOnly: true, Secure: s.config.Production, SameSite: http.SameSiteLaxMode}
	http.SetCookie(w, httpOnlyCookie)
	return nil
}

func userResponse(user User) map[string]any {
	return map[string]any{"authenticated": true, "user": publicUser(user), "role": user.Role, "workspace_id": user.WorkspaceID}
}
