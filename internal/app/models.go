package app

import "time"

type Status string

const (
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusSuccess   Status = "success"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
	StatusUnknown   Status = "unknown"
)

type Deployment struct {
	ID              string    `json:"id"`
	WorkspaceID     string    `json:"workspace_id"`
	Provider        string    `json:"provider"`
	ProviderEventID string    `json:"provider_event_id"`
	ProviderURL     string    `json:"provider_url"`
	Repository      string    `json:"repository"`
	Branch          string    `json:"branch"`
	CommitSHA       string    `json:"commit_sha"`
	Environment     string    `json:"environment"`
	Actor           string    `json:"actor"`
	Status          Status    `json:"status"`
	OriginalStatus  string    `json:"original_status"`
	FailureCategory string    `json:"failure_category"`
	FailureSummary  string    `json:"failure_summary"`
	StartedAt       time.Time `json:"started_at"`
	FinishedAt      time.Time `json:"finished_at,omitempty"`
	Duration        string    `json:"duration"`
	CreatedAt       time.Time `json:"created_at"`
}

type DeploymentCheck struct {
	ID           string `json:"id"`
	DeploymentID string `json:"deployment_id"`
	Name         string `json:"name"`
	Status       Status `json:"status"`
	Summary      string `json:"summary"`
	StartedAt    string `json:"started_at"`
	FinishedAt   string `json:"finished_at"`
}

type DeploymentDetail struct {
	Deployment
	Checks   []DeploymentCheck `json:"checks"`
	LogLines []string          `json:"log_lines"`
}

type WebhookEvent struct {
	ID              string
	WorkspaceID     string
	Provider        string
	ProviderEventID string
	Payload         []byte
	PayloadHash     string
	CorrelationID   string
	Status          string
	ReceivedAt      time.Time
}

type DeadLetterEvent struct {
	ID             string    `json:"id"`
	WebhookEventID string    `json:"webhook_event_id"`
	Provider       string    `json:"provider"`
	Error          string    `json:"error"`
	RetryCount     int       `json:"retry_count"`
	RawPayloadRef  string    `json:"raw_payload_ref"`
	CreatedAt      time.Time `json:"created_at"`
}

type NotificationRule struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Repository  string `json:"repository"`
	Environment string `json:"environment"`
	Status      Status `json:"status"`
	Channel     string `json:"channel"`
	Target      string `json:"target"`
}

type ProviderConnection struct {
	ID        string    `json:"id"`
	Provider  string    `json:"provider"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type ListFilter struct {
	Repository  string
	Branch      string
	Environment string
	Provider    string
	Status      string
	Actor       string
	Start       time.Time
	End         time.Time
	Cursor      string
	Limit       int
}
