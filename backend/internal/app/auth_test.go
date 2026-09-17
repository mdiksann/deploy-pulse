package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSignupAndLoginWithoutEmailVerification(t *testing.T) {
	store := testStore(t)
	defer store.Close()
	server := NewServerWithConfig(NewService(store, Config{}), store, nil, ServerConfig{DefaultWorkspaceID: "demo"})
	handler := server.Handler(http.NotFoundHandler())
	bad := httptest.NewRequest(http.MethodPost, "http://example.com/api/auth/signup", strings.NewReader(`{"email":"not-an-email","password":"short"}`))
	bad.Header.Set("Content-Type", "application/json")
	badResponse := httptest.NewRecorder()
	handler.ServeHTTP(badResponse, bad)
	if badResponse.Code != http.StatusBadRequest {
		t.Fatalf("invalid signup status=%d", badResponse.Code)
	}

	password, err := passwordHash("password-123")
	if err != nil {
		t.Fatal(err)
	}
	_, created, err := store.CreatePendingUser(context.Background(), "auth@example.com", password, "demo")
	if err != nil || !created {
		t.Fatalf("create user: %v", err)
	}
	login := httptest.NewRequest(http.MethodPost, "http://example.com/api/auth/login", bytes.NewBufferString(`{"email":"auth@example.com","password":"password-123"}`))
	login.Header.Set("Content-Type", "application/json")
	loginResponse := httptest.NewRecorder()
	handler.ServeHTTP(loginResponse, login)
	if loginResponse.Code != http.StatusOK || len(loginResponse.Result().Cookies()) != 1 {
		t.Fatalf("unverified login status=%d", loginResponse.Code)
	}
	if strings.Contains(loginResponse.Body.String(), password) {
		t.Fatal("password hash leaked in login response")
	}
}
