package app

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"hash"
	"net/http"
	"testing"
)

func TestProviderSignatureAdapters(t *testing.T) {
	body := []byte(`{"event":"deployment"}`)
	secrets := map[string]string{"github": "github-secret", "gitlab": "gitlab-secret", "circleci": "circle-secret", "vercel": "vercel-secret", "netlify": "netlify-secret", "aws-codepipeline": "relay-secret"}
	service := NewService(nil, Config{WebhookSecrets: secrets})
	cases := []struct {
		provider string
		header   string
		value    string
	}{
		{"github", "X-Hub-Signature-256", "sha256=" + hmacHex(sha256.New, secrets["github"], body)},
		{"gitlab", "X-Gitlab-Token", secrets["gitlab"]},
		{"circleci", "Circleci-Signature", "v1=" + hmacHex(sha256.New, secrets["circleci"], body)},
		{"vercel", "X-Vercel-Signature", hmacHex(sha1.New, secrets["vercel"], body)},
		{"netlify", "X-Webhook-Signature", netlifyJWS(t, secrets["netlify"], body)},
		{"aws-codepipeline", "X-DeployPulse-Relay-Signature", "sha256=" + hmacHex(sha256.New, secrets["aws-codepipeline"], body)},
	}
	for _, test := range cases {
		t.Run(test.provider, func(t *testing.T) {
			header := make(http.Header)
			header.Set(test.header, test.value)
			if err := service.Verify(test.provider, header, body); err != nil {
				t.Fatal(err)
			}
		})
	}
	wrongHeader := make(http.Header)
	wrongHeader.Set("X-Hub-Signature-256", "sha256="+hmacHex(sha256.New, secrets["vercel"], body))
	if err := service.Verify("vercel", wrongHeader, body); err == nil {
		t.Fatal("Vercel accepted a GitHub signature header")
	}
}

func TestProductionRequiresEncryptionKey(t *testing.T) {
	if err := NewService(nil, Config{Production: true}).ValidateConfig(); err == nil {
		t.Fatal("production accepted an empty encryption key")
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	store := testStore(t)
	defer store.Close()
	if err := store.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	var migrationCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&migrationCount); err != nil {
		t.Fatal(err)
	}
	if migrationCount != len(migrations) {
		t.Fatalf("migrations=%d, want %d", migrationCount, len(migrations))
	}
}

func hmacHex(newHash func() hash.Hash, secret string, body []byte) string {
	mac := hmac.New(newHash, []byte(secret))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func netlifyJWS(t *testing.T, secret string, body []byte) string {
	t.Helper()
	header, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(body)
	claims, err := json.Marshal(map[string]string{"iss": "netlify", "sha256": hex.EncodeToString(hash[:])})
	if err != nil {
		t.Fatal(err)
	}
	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
