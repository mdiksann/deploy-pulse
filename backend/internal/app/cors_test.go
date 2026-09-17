package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCORSAllowsConfiguredFrontendAndRejectsUnknownOrigin(t *testing.T) {
	store := testStore(t)
	defer store.Close()
	server := NewServerWithConfig(NewService(store, Config{}), store, processingPublisher{}, ServerConfig{DefaultWorkspaceID: "demo", FrontendOrigins: []string{"http://localhost:3000"}})
	handler := server.Handler()

	preflight := httptest.NewRequest(http.MethodOptions, "http://localhost:8080/api/auth/login", nil)
	preflight.Header.Set("Origin", "http://localhost:3000")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodPost)
	preflight.Header.Set("Access-Control-Request-Headers", "content-type")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, preflight)
	if response.Code != http.StatusNoContent || response.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("allowed preflight status=%d headers=%v", response.Code, response.Header())
	}
	if response.Header().Get("Access-Control-Allow-Credentials") != "true" || response.Header().Get("Access-Control-Allow-Methods") != "GET, POST, OPTIONS" {
		t.Fatalf("incomplete CORS response: %v", response.Header())
	}

	unknown := httptest.NewRequest(http.MethodOptions, "http://localhost:8080/api/auth/login", nil)
	unknown.Header.Set("Origin", "https://unknown.example")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, unknown)
	if response.Code != http.StatusForbidden || response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("unknown origin status=%d headers=%v", response.Code, response.Header())
	}

}

func TestMutationRejectsOriginOutsideFrontendAllowlist(t *testing.T) {
	store := testStore(t)
	defer store.Close()
	service := NewService(store, Config{})
	server := NewServerWithConfig(service, store, processingPublisher{}, ServerConfig{DefaultWorkspaceID: "demo", FrontendOrigins: []string{"http://localhost:3000"}})
	handler := server.Handler()
	password, err := passwordHash("password-123")
	if err != nil {
		t.Fatal(err)
	}
	user, _, err := store.CreatePendingUser(context.Background(), "mutate@example.com", password, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.exec(context.Background(), `UPDATE users SET email_verified_at=? WHERE id=?`, time.Now().UTC().Format(time.RFC3339Nano), user.ID); err != nil {
		t.Fatal(err)
	}
	login := httptest.NewRequest(http.MethodPost, "http://api.example/api/auth/login", bytes.NewBufferString(`{"email":"mutate@example.com","password":"password-123"}`))
	login.Header.Set("Content-Type", "application/json")
	login.Header.Set("Origin", "http://localhost:3000")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, login)
	if response.Code != http.StatusOK {
		t.Fatalf("login status=%d", response.Code)
	}
	cookie := response.Result().Cookies()[0]
	mutate := httptest.NewRequest(http.MethodPost, "http://api.example/api/notification-rules", bytes.NewBufferString(`{"status":"failed","channel":"slack","target":"#ops"}`))
	mutate.Header.Set("Content-Type", "application/json")
	mutate.Header.Set("Origin", "https://unknown.example")
	mutate.AddCookie(cookie)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, mutate)
	if response.Code != http.StatusForbidden {
		t.Fatalf("unknown mutation status=%d", response.Code)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("unknown mutation received CORS permission")
	}
}
