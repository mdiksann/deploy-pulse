package app

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWebhookRejectsInvalidSignatureAndProcessesValidDelivery(t *testing.T) {
	store := testStore(t)
	defer store.Close()
	service := NewService(store, Config{WebhookSecrets: map[string]string{"github": "test-secret"}})
	handler := NewServerWithConfig(service, store, processingPublisher{service}, ServerConfig{DefaultWorkspaceID: "demo"}).Handler(http.NotFoundHandler())
	payload, err := os.ReadFile(filepath.Join("testdata", "github.json"))
	if err != nil {
		t.Fatal(err)
	}
	invalid := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(payload))
	invalid.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, invalid)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("invalid signature status=%d", response.Code)
	}
	var events int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM webhook_events`).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if events != 0 {
		t.Fatalf("invalid webhook persisted %d events", events)
	}
	valid := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(payload))
	valid.Header.Set("Content-Type", "application/json")
	valid.Header.Set("X-Hub-Signature-256", "sha256="+signature("test-secret", payload))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, valid)
	if response.Code != http.StatusAccepted {
		t.Fatalf("valid signature status=%d body=%s", response.Code, response.Body.String())
	}
	deadline := time.Now().Add(time.Second)
	for {
		items, _, listErr := store.ListDeployments(context.Background(), "demo", ListFilter{})
		if listErr != nil {
			t.Fatal(listErr)
		}
		if len(items) == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("accepted webhook was not processed")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestAdminEndpointsRequireSessionAndRejectBearerToken(t *testing.T) {
	store := testStore(t)
	defer store.Close()
	service := NewService(store, Config{})
	password, _ := passwordHash("password-123")
	user, _, err := store.CreatePendingUser(context.Background(), "admin@example.com", password, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.exec(context.Background(), `UPDATE users SET email_verified_at=? WHERE id=?`, time.Now().UTC().Format(time.RFC3339Nano), user.ID); err != nil {
		t.Fatal(err)
	}
	handler := NewServerWithConfig(service, store, processingPublisher{service}, ServerConfig{DefaultWorkspaceID: "demo"}).Handler(http.NotFoundHandler())
	login := httptest.NewRequest(http.MethodPost, "http://example.com/api/auth/login", bytes.NewBufferString(`{"email":"admin@example.com","password":"password-123"}`))
	login.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, login)
	if response.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", response.Code, response.Body.String())
	}
	cookie := response.Result().Cookies()[0]
	request := httptest.NewRequest(http.MethodGet, "http://example.com/api/provider-connections", nil)
	request.AddCookie(cookie)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("session status=%d body=%s", response.Code, response.Body.String())
	}
	bearer := httptest.NewRequest(http.MethodGet, "/api/provider-connections", nil)
	bearer.Header.Set("Authorization", "Bearer password-123")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, bearer)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("bearer status=%d", response.Code)
	}
}

type processingPublisher struct{ service *Service }

func (p processingPublisher) Publish(ctx context.Context, eventID string) error {
	return p.service.Process(ctx, eventID)
}
func (processingPublisher) Healthy(context.Context) error { return nil }

func signature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
