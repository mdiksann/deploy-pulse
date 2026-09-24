package app

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

func TestGitHubWorkflowDeliveriesUseDeliveryID(t *testing.T) {
	store := testStore(t)
	defer store.Close()
	service := NewService(store, Config{WebhookSecrets: map[string]string{"github": "test-secret"}})
	handler := NewServerWithConfig(service, store, processingPublisher{service}, ServerConfig{DefaultWorkspaceID: "demo"}).Handler()
	send := func(deliveryID, body string) int {
		payload := []byte(body)
		request := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(payload))
		request.Header.Set("X-Hub-Signature-256", "sha256="+signature("test-secret", payload))
		request.Header.Set("X-GitHub-Delivery", deliveryID)
		request.Header.Set("X-GitHub-Event", "workflow_run")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response.Code
	}
	running := `{"repository":{"full_name":"example/app"},"workflow_run":{"id":123,"status":"in_progress"}}`
	completed := `{"repository":{"full_name":"example/app"},"workflow_run":{"id":123,"status":"completed","conclusion":"success"}}`
	for _, item := range []struct{ id, body string }{{"delivery-running", running}, {"delivery-completed", completed}, {"delivery-running", running}} {
		if code := send(item.id, item.body); code != http.StatusAccepted {
			t.Fatalf("delivery %s status=%d", item.id, code)
		}
	}
	var events, deployments int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM webhook_events`).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM deployments`).Scan(&deployments); err != nil {
		t.Fatal(err)
	}
	if events != 2 || deployments != 2 {
		t.Fatalf("events=%d deployments=%d, want 2 each", events, deployments)
	}
}

func TestDeploymentReadsRequireOwnWorkspace(t *testing.T) {
	store := testStore(t)
	defer store.Close()
	server := NewServerWithConfig(NewService(store, Config{}), store, processingPublisher{}, ServerConfig{DefaultWorkspaceID: "shared"})
	handler := server.Handler()
	deploymentIDs := map[string]string{}
	for _, item := range []struct{ id, workspace string }{{"first-deploy", "first"}, {"second-deploy", "second"}} {
		deployment := deploymentForTest(item.id, StatusSuccess, time.Now())
		deployment.WorkspaceID = item.workspace
		deploymentIDs[item.id] = deployment.ID
		if _, err := store.SaveDeployment(context.Background(), deployment, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{"/api/deployments", "/api/deployments/" + deploymentIDs["first-deploy"], "/api/analytics/deployments", "/api/health"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("anonymous %s status=%d", path, response.Code)
		}
	}
	var request *http.Request
	var response *httptest.ResponseRecorder
	for _, item := range []struct{ workspace, own, other string }{{"first", "first-deploy", "second-deploy"}, {"second", "second-deploy", "first-deploy"}} {
		user, _, err := store.CreatePendingUser(context.Background(), item.workspace+"@example.com", "hash", item.workspace)
		if err != nil {
			t.Fatal(err)
		}
		login := httptest.NewRecorder()
		if err := server.issueSession(login, httptest.NewRequest(http.MethodPost, "/api/auth/login", nil), user); err != nil {
			t.Fatal(err)
		}
		cookie := login.Result().Cookies()[0]
		request = httptest.NewRequest(http.MethodGet, "/api/deployments", nil)
		request.AddCookie(cookie)
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		var list struct {
			Items []Deployment `json:"items"`
		}
		if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &list) != nil || len(list.Items) != 1 || list.Items[0].ProviderEventID != item.own {
			t.Fatalf("workspace %s list status=%d body=%s", item.workspace, response.Code, response.Body.String())
		}
		request = httptest.NewRequest(http.MethodGet, "/api/deployments/"+deploymentIDs[item.other], nil)
		request.AddCookie(cookie)
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("workspace %s foreign detail status=%d", item.workspace, response.Code)
		}
	}
}

func TestSignupCreatesSeparateWorkspaces(t *testing.T) {
	store := testStore(t)
	defer store.Close()
	handler := NewServerWithConfig(NewService(store, Config{}), store, processingPublisher{}, ServerConfig{DefaultWorkspaceID: "shared"}).Handler()
	for _, email := range []string{"first@example.com", "second@example.com"} {
		body, _ := json.Marshal(map[string]string{"email": email, "password": "password-123"})
		request := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewReader(body))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusCreated {
			t.Fatalf("signup status=%d body=%s", response.Code, response.Body.String())
		}
	}
	first, err := store.UserByEmail(context.Background(), "first@example.com")
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.UserByEmail(context.Background(), "second@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if first.WorkspaceID == second.WorkspaceID || first.WorkspaceID == "shared" || second.WorkspaceID == "shared" {
		t.Fatalf("signups share workspace: %s / %s", first.WorkspaceID, second.WorkspaceID)
	}
}

func TestWorkspaceWebhookUsesStoredSecret(t *testing.T) {
	store := testStore(t)
	defer store.Close()
	service := NewService(store, Config{WebhookSecrets: map[string]string{"github": "default-secret"}, EncryptionKey: "encryption-key"})
	handler := NewServerWithConfig(service, store, processingPublisher{service}, ServerConfig{DefaultWorkspaceID: "shared"}).Handler()
	for _, item := range []struct{ workspace, secret string }{{"first", "first-secret"}, {"second", "second-secret"}} {
		encrypted, err := service.EncryptSecret(item.secret)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.SaveConnection(context.Background(), item.workspace, "github", "GitHub", encrypted); err != nil {
			t.Fatal(err)
		}
	}
	payload := []byte(`{"repository":{"full_name":"example/app"},"workflow_run":{"id":123,"status":"completed","conclusion":"success"}}`)
	send := func(workspace, secret string) int {
		request := httptest.NewRequest(http.MethodPost, "/webhooks/github/"+workspace, bytes.NewReader(payload))
		request.Header.Set("X-Hub-Signature-256", "sha256="+signature(secret, payload))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response.Code
	}
	if code := send("first", "second-secret"); code != http.StatusUnauthorized {
		t.Fatalf("wrong workspace secret status=%d", code)
	}
	if code := send("first", "first-secret"); code != http.StatusAccepted {
		t.Fatalf("correct workspace secret status=%d", code)
	}
	first, _, err := store.ListDeployments(context.Background(), "first", ListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := store.ListDeployments(context.Background(), "second", ListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || len(second) != 0 {
		t.Fatalf("first deployments=%d second deployments=%d", len(first), len(second))
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
