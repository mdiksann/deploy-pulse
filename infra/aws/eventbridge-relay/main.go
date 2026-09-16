package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
)

type eventBridgeEvent struct {
	Source     string          `json:"source"`
	DetailType string          `json:"detail-type"`
	Detail     json.RawMessage `json:"detail"`
}

func main() { lambda.Start(handle) }

func handle(ctx context.Context, event eventBridgeEvent) error {
	if event.Source != "aws.codepipeline" || event.DetailType != "CodePipeline Pipeline Execution State Change" {
		return errors.New("unexpected EventBridge event")
	}
	webhookURL, secret := os.Getenv("DEPLOYPULSE_WEBHOOK_URL"), os.Getenv("RELAY_SHARED_SECRET")
	if webhookURL == "" || secret == "" {
		return errors.New("DEPLOYPULSE_WEBHOOK_URL and RELAY_SHARED_SECRET are required")
	}
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-DeployPulse-Relay-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	client := &http.Client{Timeout: 8 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return errors.New("Deploy Pulse rejected relay: " + response.Status + " " + string(body))
	}
	return nil
}
