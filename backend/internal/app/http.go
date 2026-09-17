package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ServerConfig struct {
	DefaultWorkspaceID string
	Production         bool
	FrontendOrigins    []string
	APIPublicURL       string
}

type Server struct {
	service   *Service
	store     *Store
	publisher Publisher
	config    ServerConfig
}

// NewServer keeps the SQLite/test-friendly defaults. Production uses NewServerWithConfig.
func NewServer(service *Service, store *Store) *Server {
	return NewServerWithConfig(service, store, nil, ServerConfig{DefaultWorkspaceID: "demo"})
}

func NewServerWithConfig(service *Service, store *Store, publisher Publisher, config ServerConfig) *Server {
	if config.DefaultWorkspaceID == "" {
		config.DefaultWorkspaceID = "demo"
	}
	return &Server{service: service, store: store, publisher: publisher, config: config}
}

func (s *Server) Handler(_ ...http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /webhooks/{provider}", s.handleWebhook)
	mux.HandleFunc("GET /healthz", s.live)
	mux.HandleFunc("GET /readyz", s.ready)
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/analytics/deployments", s.analytics)
	mux.HandleFunc("GET /api/deployments", s.listDeployments)
	mux.HandleFunc("GET /api/deployments/{id}", s.getDeployment)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("POST /api/auth/signup", s.signup)
	mux.HandleFunc("GET /api/auth/me", s.me)
	mux.HandleFunc("POST /api/auth/logout", s.logout)
	mux.HandleFunc("GET /api/notification-rules", s.listRules)
	mux.HandleFunc("POST /api/notification-rules", s.createRule)
	mux.HandleFunc("GET /api/dead-letter-events", s.listDeadLetters)
	mux.HandleFunc("POST /api/dead-letter-events/{id}/reprocess", s.reprocessDeadLetter)
	mux.HandleFunc("GET /api/provider-connections", s.listConnections)
	mux.HandleFunc("POST /api/provider-connections", s.saveConnection)
	return requestLog(s.cors(mux))
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api") {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Add("Vary", "Origin")
		origin := r.Header.Get("Origin")
		if s.allowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if r.Method == http.MethodOptions {
			if !s.allowedOrigin(origin) {
				writeError(w, http.StatusForbidden, "origin not allowed")
				return
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) allowedOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	for _, allowed := range s.config.FrontendOrigins {
		if strings.TrimRight(allowed, "/") == strings.TrimRight(origin, "/") {
			return true
		}
	}
	return false
}

func (s *Server) live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Healthy(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	if s.publisher == nil {
		writeError(w, http.StatusServiceUnavailable, "queue unavailable")
		return
	}
	if err := s.publisher.Healthy(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "queue unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	database, queue := "healthy", "healthy"
	if s.store.Healthy(r.Context()) != nil {
		database = "unavailable"
	}
	if s.publisher == nil || s.publisher.Healthy(r.Context()) != nil {
		queue = "unavailable"
	}
	dlq, err := s.store.DeadLetterCount(r.Context(), s.workspaceID())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "health unavailable")
		return
	}
	connected, err := s.store.ConnectedProviders(r.Context(), s.workspaceID())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "health unavailable")
		return
	}
	connectedSet := make(map[string]bool, len(connected))
	for _, provider := range connected {
		connectedSet[provider] = true
	}
	for _, provider := range s.service.ConfiguredProviders() {
		connectedSet[provider] = true
	}
	connected = connected[:0]
	for _, provider := range SortedProviders() {
		if connectedSet[provider] {
			connected = append(connected, provider)
		}
	}
	status := "ok"
	code := http.StatusOK
	if database != "healthy" || queue != "healthy" {
		status, code = "degraded", http.StatusServiceUnavailable
	}
	writeJSON(w, code, map[string]any{"status": status, "api": "healthy", "database": database, "queue": queue, "dead_letter_events": dlq, "providers": SortedProviders(), "connected_providers": connected})
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
	event := WebhookEvent{ID: uuid.NewString(), WorkspaceID: s.workspaceID(), Provider: provider, ProviderEventID: providerEventID, Payload: body, PayloadHash: hex.EncodeToString(hash[:]), CorrelationID: correlationID, ReceivedAt: time.Now().UTC()}
	w.Header().Set("X-Correlation-ID", correlationID)
	inserted, err := s.store.RecordWebhook(r.Context(), event)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not record webhook")
		return
	}
	if inserted {
		if s.publisher == nil || s.publisher.Publish(r.Context(), event.ID) != nil {
			// The database record remains queued. Returning 503 asks the provider to retry its delivery.
			writeError(w, http.StatusServiceUnavailable, "queue unavailable")
			return
		}
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": true, "duplicate": !inserted, "correlation_id": correlationID})
}

func (s *Server) listDeployments(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	f := ListFilter{Repository: q.Get("repository"), Branch: q.Get("branch"), Environment: q.Get("environment"), Provider: q.Get("provider"), Status: q.Get("status"), Actor: q.Get("actor"), Query: q.Get("q"), Cursor: q.Get("cursor"), Limit: limit}
	if value := q.Get("start"); value != "" {
		f.Start, _ = time.Parse(time.RFC3339, value)
	}
	if value := q.Get("end"); value != "" {
		f.End, _ = time.Parse(time.RFC3339, value)
	}
	items, next, err := s.store.ListDeployments(r.Context(), s.workspaceID(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load deployments")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "next_cursor": next})
}

func (s *Server) analytics(w http.ResponseWriter, r *http.Request) {
	days, err := strconv.Atoi(r.URL.Query().Get("days"))
	if err != nil {
		days = 7
	}
	data, err := s.store.DeploymentAnalyticsFiltered(r.Context(), s.workspaceID(), days, r.URL.Query().Get("environment"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load deployment analytics")
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	email, emailErr := normalizeEmail(input.Email)
	if emailErr != nil || validatePassword(input.Password) != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	stored, err := s.store.UserByEmail(r.Context(), email)
	if err != nil || bcryptCompare(stored.PasswordHash, input.Password) != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err := s.issueSession(w, r, stored.User); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}
	writeJSON(w, http.StatusOK, userResponse(stored.User))
}

func firstAuthError(emailErr, passwordErr error) string {
	if emailErr != nil {
		return emailErr.Error()
	}
	return passwordErr.Error()
}

func (s *Server) signup(w http.ResponseWriter, r *http.Request) {
	var input struct{ Email, Password string }
	if !decodeJSON(w, r, &input) {
		return
	}
	email, emailErr := normalizeEmail(input.Email)
	passwordErr := validatePassword(input.Password)
	if emailErr != nil || passwordErr != nil {
		writeError(w, http.StatusBadRequest, firstAuthError(emailErr, passwordErr))
		return
	}
	hash, err := passwordHash(input.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create account")
		return
	}
	_, _, err = s.store.CreatePendingUser(r.Context(), email, hash, s.workspaceID())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create account")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"message": "Account created. You can sign in now."})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	user, ok := s.authenticateSession(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "admin authentication required")
		return
	}
	writeJSON(w, http.StatusOK, userResponse(user))
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		if !s.sameOrigin(r) {
			writeError(w, http.StatusForbidden, "origin required for browser mutation")
			return
		}
		_ = s.store.RevokeSession(r.Context(), hashToken(cookie.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, Expires: time.Unix(1, 0).UTC(), HttpOnly: true, Secure: s.config.Production, SameSite: http.SameSiteLaxMode})
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": false})
}

func (s *Server) getDeployment(w http.ResponseWriter, r *http.Request) {
	detail, err := s.store.Deployment(r.Context(), s.workspaceID(), r.PathValue("id"))
	if errors.Is(err, io.EOF) || strings.Contains(errString(err), "no rows") {
		writeError(w, http.StatusNotFound, "deployment not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load deployment")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (s *Server) listRules(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeAdmin(w, r, false) {
		return
	}
	rules, err := s.store.ListRules(r.Context(), s.workspaceID())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load rules")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": rules})
}

func (s *Server) createRule(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeAdmin(w, r, true) {
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
	if input.Status != StatusFailed || (input.Channel != "slack" && input.Channel != "email") || input.Target == "" {
		writeError(w, http.StatusBadRequest, "repository, environment, failed status, channel, and target are required")
		return
	}
	rule, err := s.store.CreateRule(r.Context(), NotificationRule{WorkspaceID: s.workspaceID(), Repository: input.Repository, Environment: input.Environment, Status: input.Status, Channel: input.Channel, Target: input.Target})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create rule")
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (s *Server) listDeadLetters(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeAdmin(w, r, false) {
		return
	}
	items, err := s.store.ListDeadLetters(r.Context(), s.workspaceID())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load dead-letter events")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) reprocessDeadLetter(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeAdmin(w, r, true) {
		return
	}
	dlqID := r.PathValue("id")
	eventID, err := s.store.PrepareReprocess(r.Context(), s.workspaceID(), dlqID)
	if err != nil {
		writeError(w, http.StatusNotFound, "dead-letter event not found")
		return
	}
	if s.publisher == nil || s.publisher.Publish(r.Context(), eventID) != nil {
		writeError(w, http.StatusServiceUnavailable, "queue unavailable")
		return
	}
	if err := s.store.CompleteReprocess(r.Context(), dlqID); err != nil {
		writeError(w, http.StatusInternalServerError, "event queued but recovery record could not be cleared")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"reprocessed": true})
}

func (s *Server) listConnections(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeAdmin(w, r, false) {
		return
	}
	items, err := s.store.ListConnections(r.Context(), s.workspaceID())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load provider connections")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "supported_providers": SortedProviders()})
}

func (s *Server) saveConnection(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeAdmin(w, r, true) {
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
		writeError(w, http.StatusBadRequest, "provider, name, and secret are required")
		return
	}
	encrypted, err := s.service.EncryptSecret(input.Secret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not encrypt secret")
		return
	}
	connection, err := s.store.SaveConnection(r.Context(), s.workspaceID(), input.Provider, input.Name, encrypted)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save provider connection")
		return
	}
	writeJSON(w, http.StatusCreated, connection)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func (s *Server) workspaceID() string { return s.config.DefaultWorkspaceID }

func (s *Server) isAdmin(r *http.Request) bool {
	_, ok := s.authenticateSession(r)
	return ok
}

func (s *Server) authorizeAdmin(w http.ResponseWriter, r *http.Request, mutate bool) bool {
	if !s.isAdmin(r) {
		writeError(w, http.StatusUnauthorized, "admin authentication required")
		return false
	}
	if mutate {
		if !s.sameOrigin(r) {
			writeError(w, http.StatusForbidden, "invalid request origin")
			return false
		}
	}
	return true
}

func (s *Server) sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	if len(s.config.FrontendOrigins) > 0 {
		return s.allowedOrigin(origin)
	}
	parsed, err := url.Parse(origin)
	return err == nil && parsed.Host != "" && parsed.Host == r.Host
}

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
