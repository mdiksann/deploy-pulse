package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Server struct {
	service *Service
	store   *Store
}

func NewServer(service *Service, store *Store) *Server {
	return &Server{service: service, store: store}
}

func (s *Server) Handler(static http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /webhooks/{provider}", s.handleWebhook)
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "providers": SortedProviders()})
	})
	mux.HandleFunc("GET /api/deployments", s.listDeployments)
	mux.HandleFunc("GET /api/deployments/{id}", s.getDeployment)
	mux.HandleFunc("GET /api/notification-rules", s.listRules)
	mux.HandleFunc("POST /api/notification-rules", s.createRule)
	mux.HandleFunc("GET /api/dead-letter-events", s.listDeadLetters)
	mux.HandleFunc("POST /api/dead-letter-events/{id}/reprocess", s.reprocessDeadLetter)
	mux.HandleFunc("GET /api/provider-connections", s.listConnections)
	mux.HandleFunc("POST /api/provider-connections", s.saveConnection)
	mux.Handle("/", static)
	return requestLog(mux)
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	provider := strings.ToLower(r.PathValue("provider"))
	if !supportedProviders[provider] {
		writeError(w, http.StatusNotFound, "unsupported provider")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "payload exceeds 1 MiB")
		return
	}
	if err := s.service.Verify(provider, r.Header, body); err != nil {
		if strings.Contains(err.Error(), "not configured") {
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		writeError(w, http.StatusUnauthorized, "invalid webhook signature")
		return
	}
	providerEventID := ProviderEventID(provider, body)
	hash := sha256.Sum256(body)
	if providerEventID == "" {
		providerEventID = "payload-" + hex.EncodeToString(hash[:12])
	}
	correlationID := r.Header.Get("X-Correlation-ID")
	if correlationID == "" {
		correlationID = uuid.NewString()
	}
	event := WebhookEvent{ID: uuid.NewString(), WorkspaceID: workspaceID(r), Provider: provider, ProviderEventID: providerEventID, Payload: body, PayloadHash: hex.EncodeToString(hash[:]), CorrelationID: correlationID, ReceivedAt: time.Now().UTC()}
	w.Header().Set("X-Correlation-ID", correlationID)
	inserted, err := s.store.RecordWebhook(r.Context(), event)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not record webhook")
		return
	}
	if inserted && !s.service.Enqueue(event.ID) {
		go s.service.process(r.Context(), event.ID)
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": true, "duplicate": !inserted, "correlation_id": correlationID})
}

func (s *Server) listDeployments(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	f := ListFilter{Repository: q.Get("repository"), Branch: q.Get("branch"), Environment: q.Get("environment"), Provider: q.Get("provider"), Status: q.Get("status"), Actor: q.Get("actor"), Cursor: q.Get("cursor"), Limit: limit}
	if value := q.Get("start"); value != "" {
		f.Start, _ = time.Parse(time.RFC3339, value)
	}
	if value := q.Get("end"); value != "" {
		f.End, _ = time.Parse(time.RFC3339, value)
	}
	items, next, err := s.store.ListDeployments(r.Context(), workspaceID(r), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load deployments")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "next_cursor": next})
}

func (s *Server) getDeployment(w http.ResponseWriter, r *http.Request) {
	detail, err := s.store.Deployment(r.Context(), workspaceID(r), r.PathValue("id"))
	if errors.Is(err, io.EOF) {
		writeError(w, http.StatusNotFound, "deployment not found")
		return
	}
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			writeError(w, http.StatusNotFound, "deployment not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load deployment")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) listRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.store.ListRules(r.Context(), workspaceID(r))
	if err != nil {
		writeError(w, 500, "could not load rules")
		return
	}
	writeJSON(w, 200, map[string]any{"items": rules})
}
func (s *Server) createRule(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		writeError(w, http.StatusForbidden, "admin role required")
		return
	}
	var input struct {
		Repository  string `json:"repository"`
		Environment string `json:"environment"`
		Status      Status `json:"status"`
		Channel     string `json:"channel"`
		Target      string `json:"target"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.Repository == "" {
		input.Repository = "*"
	}
	if input.Environment == "" {
		input.Environment = "*"
	}
	if input.Status != "failed" || (input.Channel != "slack" && input.Channel != "email") || input.Target == "" {
		writeError(w, 400, "repository, environment, failed status, channel, and target are required")
		return
	}
	rule, err := s.store.CreateRule(r.Context(), NotificationRule{WorkspaceID: workspaceID(r), Repository: input.Repository, Environment: input.Environment, Status: input.Status, Channel: input.Channel, Target: input.Target})
	if err != nil {
		writeError(w, 500, "could not create rule")
		return
	}
	writeJSON(w, 201, rule)
}

func (s *Server) listDeadLetters(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		writeError(w, 403, "admin role required")
		return
	}
	items, err := s.store.ListDeadLetters(r.Context(), workspaceID(r))
	if err != nil {
		writeError(w, 500, "could not load dead-letter events")
		return
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
func (s *Server) reprocessDeadLetter(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		writeError(w, 403, "admin role required")
		return
	}
	eventID, err := s.store.ReprocessDeadLetter(r.Context(), workspaceID(r), r.PathValue("id"))
	if err != nil {
		writeError(w, 404, "dead-letter event not found")
		return
	}
	if !s.service.Enqueue(eventID) {
		go s.service.process(r.Context(), eventID)
	}
	writeJSON(w, 202, map[string]any{"reprocessed": true})
}

func (s *Server) listConnections(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		writeError(w, 403, "admin role required")
		return
	}
	items, err := s.store.ListConnections(r.Context(), workspaceID(r))
	if err != nil {
		writeError(w, 500, "could not load provider connections")
		return
	}
	writeJSON(w, 200, map[string]any{"items": items, "supported_providers": SortedProviders()})
}
func (s *Server) saveConnection(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		writeError(w, 403, "admin role required")
		return
	}
	var input struct {
		Provider string `json:"provider"`
		Name     string `json:"name"`
		Secret   string `json:"secret"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Provider = strings.ToLower(input.Provider)
	if !supportedProviders[input.Provider] || input.Name == "" || input.Secret == "" {
		writeError(w, 400, "provider, name, and secret are required")
		return
	}
	encrypted, err := s.service.EncryptSecret(input.Secret)
	if err != nil {
		writeError(w, 500, "could not encrypt secret")
		return
	}
	connection, err := s.store.SaveConnection(r.Context(), workspaceID(r), input.Provider, input.Name, encrypted)
	if err != nil {
		writeError(w, 500, "could not save provider connection")
		return
	}
	writeJSON(w, 201, connection)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		writeError(w, 400, "invalid JSON body")
		return false
	}
	return true
}
func workspaceID(r *http.Request) string {
	if id := r.Header.Get("X-Workspace-ID"); id != "" {
		return id
	}
	return "demo"
}
func isAdmin(r *http.Request) bool { return r.Header.Get("X-Role") == "admin" }
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
func requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}
