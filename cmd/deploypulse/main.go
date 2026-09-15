package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"deploypulse/internal/app"
)

//go:embed web/*
var web embed.FS

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	store, err := app.OpenStore(ctx, env("DATABASE_URL", "deploy-pulse.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store, app.Config{WebhookSecrets: webhookSecrets(), SecretNames: split(env("SECRET_NAMES", "TOKEN,PASSWORD,API_KEY,PRIVATE_KEY"))})
	service.Start(ctx)
	defer service.Close()
	if env("DEMO_DATA", "true") == "true" {
		if err := service.SeedDemo(ctx, "demo"); err != nil {
			log.Fatal(err)
		}
	}
	static, err := fs.Sub(web, "web")
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{Addr: env("ADDR", ":8080"), Handler: app.NewServer(service, store).Handler(http.FileServer(http.FS(static))), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("Deploy Pulse listening on http://localhost%s", server.Addr)
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
	secrets := map[string]string{"encryption": env("ENCRYPTION_KEY", "")}
	dev := env("DEV_WEBHOOK_SECRET", "")
	for _, provider := range app.SortedProviders() {
		key := "WEBHOOK_SECRET_" + strings.ToUpper(strings.ReplaceAll(provider, "-", "_"))
		secrets[provider] = env(key, dev)
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
