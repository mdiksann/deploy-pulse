package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"deploypulse/internal/app"
)

func main() {
	role := env("ROLE", "api")
	if len(os.Args) > 1 {
		role = os.Args[1]
	}
	if role != "api" && role != "worker" && role != "migrate" {
		log.Fatalf("unknown role %q (use api, worker, or migrate)", role)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	store, err := app.OpenStore(ctx, env("DATABASE_URL", "deploy-pulse.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	if role == "migrate" {
		if err := store.Migrate(ctx); err != nil {
			log.Fatal(err)
		}
		return
	}
	production := env("APP_ENV", "development") == "production"
	service := app.NewService(store, app.Config{
		WebhookSecrets: webhookSecrets(),
		SecretNames:    split(env("SECRET_NAMES", "TOKEN,PASSWORD,API_KEY,PRIVATE_KEY")),
		EncryptionKey:  os.Getenv("ENCRYPTION_KEY"),
		Production:     production,
	})
	if err := service.ValidateConfig(); err != nil {
		log.Fatal(err)
	}
	frontendOrigins := env("FRONTEND_ORIGINS", "http://localhost:3000")
	queue, err := app.OpenRedisStream(os.Getenv("REDIS_URL"))
	if err != nil {
		log.Fatal("REDIS_URL is required for api and worker: ", err)
	}
	defer queue.Close()
	if role == "worker" {
		if err := app.NewWorker(service, store, queue).Run(ctx); err != nil && err != context.Canceled {
			log.Fatal(err)
		}
		return
	}
	if env("DEMO_DATA", "false") == "true" {
		if err := service.SeedDemo(ctx, env("DEFAULT_WORKSPACE_ID", "demo")); err != nil {
			log.Fatal(err)
		}
	}
	server := &http.Server{
		Addr:              env("ADDR", ":8080"),
		Handler:           app.NewServerWithConfig(service, store, queue, app.ServerConfig{DefaultWorkspaceID: env("DEFAULT_WORKSPACE_ID", "demo"), Production: production, FrontendOrigins: splitNonEmpty(frontendOrigins), APIPublicURL: env("API_PUBLIC_URL", "http://localhost:8080")}).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Printf("Deploy Pulse API listening on %s", env("API_PUBLIC_URL", "http://localhost"+server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	<-ctx.Done()
	shutdownCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
	defer stop()
	_ = server.Shutdown(shutdownCtx)
}

func webhookSecrets() map[string]string {
	secrets := make(map[string]string, len(app.SortedProviders()))
	dev := os.Getenv("DEV_WEBHOOK_SECRET")
	for _, provider := range app.SortedProviders() {
		key := "WEBHOOK_SECRET_" + strings.ToUpper(strings.ReplaceAll(provider, "-", "_"))
		secrets[provider] = env(key, dev)
	}
	if relay := os.Getenv("AWS_RELAY_SECRET"); relay != "" {
		secrets["aws-codepipeline"] = relay
	}
	return secrets
}
func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func split(value string) []string {
	parts := strings.Split(value, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func splitNonEmpty(value string) []string {
	parts := split(value)
	filtered := parts[:0]
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return filtered
}
