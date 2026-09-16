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
	handler := NewServerWithConfig(service, store, processingPublisher{service}, ServerConfig{DefaultWorkspaceID: "demo", AdminAPIToken: "admin-token"}).Handler(http.NotFoundHandler())
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

func TestAdminEndpointsRequireBearerToken(t *testing.T) {
	store := testStore(t)
	defer store.Close()
	service := NewService(store, Config{})
	handler := NewServerWithConfig(service, store, processingPublisher{service}, ServerConfig{DefaultWorkspaceID: "demo", AdminAPIToken: "admin-token"}).Handler(http.NotFoundHandler())
	for _, token := range []string{"", "Bearer wrong"} {
		request := httptest.NewRequest(http.MethodGet, "/api/provider-connections", nil)
		request.Header.Set("Authorization", token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("token %q status=%d, want %d", token, response.Code, http.StatusUnauthorized)
		}
	}
	request := httptest.NewRequest(http.MethodGet, "/api/provider-connections", nil)
	request.Header.Set("Authorization", "Bearer admin-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("valid token status=%d body=%s", response.Code, response.Body.String())
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
