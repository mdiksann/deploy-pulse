package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDeploymentAnalyticsBucketsStatusesAndWorkspace(t *testing.T) {
	store := testStore(t)
	defer store.Close()
	now := time.Now().UTC()
	for _, item := range []Deployment{
		deploymentForTest("analytics-success", StatusSuccess, now.Add(-2*time.Hour)),
		deploymentForTest("analytics-failed", StatusFailed, now.Add(-3*time.Hour)),
		deploymentForTest("analytics-running", StatusRunning, now.Add(-4*time.Hour)),
	} {
		if _, err := store.SaveDeployment(context.Background(), item, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	other := deploymentForTest("other-workspace", StatusSuccess, now.Add(-time.Hour))
	other.WorkspaceID = "other"
	if _, err := store.SaveDeployment(context.Background(), other, nil, nil); err != nil {
		t.Fatal(err)
	}
	data, err := store.DeploymentAnalytics(context.Background(), "workspace", 1)
	if err != nil {
		t.Fatal(err)
	}
	if data.Summary.Total != 3 || data.Summary.Success != 1 || data.Summary.Failed != 1 || data.Days[0].Other != 1 {
		t.Fatalf("unexpected analytics: %+v", data)
	}
	if data.Summary.SuccessRate != 33.33333333333333 {
		t.Fatalf("success rate=%v", data.Summary.SuccessRate)
	}
}

func TestAdminLoginSessionAndCSRF(t *testing.T) {
	store := testStore(t)
	defer store.Close()
	password, err := passwordHash("password-123")
	if err != nil {
		t.Fatal(err)
	}
	user, _, err := store.CreatePendingUser(context.Background(), "admin@example.com", password, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.exec(context.Background(), `UPDATE users SET email_verified_at=? WHERE id=?`, time.Now().UTC().Format(time.RFC3339Nano), user.ID); err != nil {
		t.Fatal(err)
	}
	server := NewServerWithConfig(NewService(store, Config{}), store, processingPublisher{}, ServerConfig{DefaultWorkspaceID: "demo"})
	handler := server.Handler(http.NotFoundHandler())
	login := httptest.NewRequest(http.MethodPost, "http://example.com/api/auth/login", bytes.NewBufferString(`{"email":"admin@example.com","password":"password-123"}`))
	login.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, login)
	if response.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", response.Code, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected session cookie: %+v", cookies)
	}
	me := httptest.NewRequest(http.MethodGet, "http://example.com/api/auth/me", nil)
	me.AddCookie(cookies[0])
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, me)
	if response.Code != http.StatusOK {
		t.Fatalf("me status=%d", response.Code)
	}
	mutate := httptest.NewRequest(http.MethodPost, "http://example.com/api/notification-rules", bytes.NewBufferString(`{"status":"failed","channel":"slack","target":"#ops"}`))
	mutate.Header.Set("Content-Type", "application/json")
	mutate.AddCookie(cookies[0])
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, mutate)
	if response.Code != http.StatusForbidden {
		t.Fatalf("csrf status=%d", response.Code)
	}
	logout := httptest.NewRequest(http.MethodPost, "http://example.com/api/auth/logout", nil)
	logout.AddCookie(cookies[0])
	logout.Header.Set("Origin", "http://example.com")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, logout)
	if response.Code != http.StatusOK {
		t.Fatalf("logout status=%d", response.Code)
	}
	me = httptest.NewRequest(http.MethodGet, "http://example.com/api/auth/me", nil)
	me.AddCookie(cookies[0])
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, me)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session status=%d", response.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
}
