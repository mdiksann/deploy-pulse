import React, { useState, useEffect, useCallback } from "react";
import { OscilloscopeCanvas } from "./OscilloscopeCanvas";
import { playTactileSound, toggleAudio, getAudioState } from "./AudioEngine";

const SCENARIOS = [
  {
    id: "github_success",
    provider: "GitHub Actions",
    badge: "RELEASE",
    tone: "emerald",
    repository: "checkout-service",
    environment: "production",
    event: "deployment_status.success",
    commit: "8f92a1b",
    statusText: "DEPLOYMENT_VERIFIED",
    hmacVerified: true,
    signatureHeader: "X-Hub-Signature-256",
    signatureVal: "sha256=a92b740ef881c201...",
    ingestLatency: "0.38ms",
    redisStream: "deploypulse:events",
    consumerGroup: "deploypulse-workers",
    sanitizedLog: "✓ Building bundle (18.2s)\n✓ Running database schema migration v42\n✓ Traffic shifted to canary instances (100%)\n✓ Health check /readyz returned 200 OK",
    rawPayload: {
      action: "created",
      deployment: { id: 910442, sha: "8f92a1b4429", ref: "main", task: "deploy" },
      repository: { full_name: "platform/checkout-service", private: true },
      secret_token_in_log: "[REDACTED_BY_ENCRYPTION_KEY]"
    }
  },
  {
    id: "gitlab_failed",
    provider: "GitLab CI",
    badge: "ALERT",
    tone: "crimson",
    repository: "payment-gateway",
    environment: "production",
    event: "pipeline.failed",
    commit: "4e18d99",
    statusText: "CHECK_FAILED_FATAL",
    hmacVerified: true,
    signatureHeader: "Webhook-Signature",
    signatureVal: "gl_sig_99014b2d1844...",
    ingestLatency: "0.45ms",
    redisStream: "deploypulse:events",
    consumerGroup: "deploypulse-workers",
    sanitizedLog: "✓ Install dependencies (1.4s)\n✗ Build application container\n  [FATAL] Missing mandatory environment variable:\n  PAYMENTS_API_URL is undefined.\n  Terminating container build with exit code 1.",
    rawPayload: {
      object_kind: "pipeline",
      object_attributes: { id: 44810, status: "failed", ref: "production" },
      project: { path_with_namespace: "finance/payment-gateway" },
      sanitized_secrets: 1
    }
  },
  {
    id: "circleci_reclaim",
    provider: "CircleCI",
    badge: "RETRY",
    tone: "amber",
    repository: "auth-broker",
    environment: "staging",
    event: "workflow.retried",
    commit: "3a00c71",
    statusText: "XAUTOCLAIM_RESOLVED",
    hmacVerified: true,
    signatureHeader: "Circleci-Signature",
    signatureVal: "v1=98c3e8002dfa...",
    ingestLatency: "0.52ms",
    redisStream: "deploypulse:events",
    consumerGroup: "deploypulse-workers",
    sanitizedLog: "⟳ Pending delivery claimed via XAUTOCLAIM (attempt 2/5)\n✓ Database transaction committed\n✓ Worker acknowledged event via XACK\n✓ Status reconciled to healthy",
    rawPayload: {
      type: "workflow-completed",
      workflow: { id: "wf_90214", name: "build-and-test", status: "success" },
      claims_count: 2
    }
  },
  {
    id: "hmac_invalid",
    provider: "Unverified Client",
    badge: "REJECTED",
    tone: "crimson",
    repository: "core-api",
    environment: "unknown",
    event: "webhook.unauthorized",
    commit: "0000000",
    statusText: "SIGNATURE_MISMATCH_401",
    hmacVerified: false,
    signatureHeader: "X-Hub-Signature-256",
    signatureVal: "sha256=invalid_hash_signature",
    ingestLatency: "0.12ms",
    redisStream: "BLOCKED_BEFORE_STREAM",
    consumerGroup: "N/A",
    sanitizedLog: "⚠ HTTP 401 Unauthorized\n  HMAC verification failed for provider payload.\n  Event discarded at ingress boundary.\n  Zero state persisted; no Redis write.",
    rawPayload: {
      error: "invalid signature",
      action: "ingress_drop"
    }
  }
];

export function HeroTelemetryReactor() {
  const [selectedScenarioId, setSelectedScenarioId] = useState("github_success");
  const [activeStage, setActiveStage] = useState(3);
  const [isSimulating, setIsSimulating] = useState(false);
  const [soundOn, setSoundOn] = useState(false);
  const [payloadTab, setPayloadTab] = useState("logs"); // "logs" | "payload" | "curl"
  const [copied, setCopied] = useState(false);

  const currentScenario = SCENARIOS.find((s) => s.id === selectedScenarioId) || SCENARIOS[0];

  const handleSoundToggle = () => {
    const newState = toggleAudio();
    setSoundOn(newState);
    if (newState) {
      playTactileSound("pulse");
    }
  };

  const triggerScenario = useCallback((scenarioId) => {
    setSelectedScenarioId(scenarioId);
    setIsSimulating(true);
    playTactileSound("tick");

    // Cycle through stages with realistic timing
    setActiveStage(0);
    setTimeout(() => {
      setActiveStage(1);
      playTactileSound("tick");
    }, 180);

    setTimeout(() => {
      setActiveStage(2);
      playTactileSound("tick");
    }, 360);

    setTimeout(() => {
      setActiveStage(3);
      setIsSimulating(false);
      const s = SCENARIOS.find((item) => item.id === scenarioId);
      if (s?.tone === "crimson") {
        playTactileSound("error");
      } else {
        playTactileSound("success");
      }
    }, 550);
  }, []);

  const copyCurl = () => {
    const curlCode = `curl -X POST https://api.deploypulse.io/webhooks/github \\\n  -H "X-Hub-Signature-256: sha256=a92b740e..." \\\n  -H "Content-Type: application/json" \\\n  -d '{"action":"deployment_status","status":"success"}'`;
    navigator.clipboard?.writeText(curlCode);
    setCopied(true);
    playTactileSound("tick");
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="telemetry-reactor" role="region" aria-label="Interactive Telemetry Demonstration">
      {/* Top Header Bar / Monochromatic Instrumentation */}
      <div className="reactor-masthead">
        <div className="masthead-left">
          <span className="reactor-crosshair">+</span>
          <span className="reactor-title">SYS://TELEMETRY_ENGINE_V2</span>
          <span className="reactor-indicator">
            <i className={`pulse-led ${currentScenario.tone}`} />
            LIVE SIMULATOR
          </span>
        </div>
        <div className="masthead-right">
          <button
            type="button"
            className={`tactile-toggle-btn ${soundOn ? "active" : ""}`}
            onClick={handleSoundToggle}
            aria-label="Toggle tactile UI feedback audio"
            title="Toggle micro-audio synthesis feedback"
          >
            <span className="toggle-label">AUDIO FEEDBACK</span>
            <span className="toggle-status">{soundOn ? "[ON]" : "[OFF]"}</span>
          </button>
        </div>
      </div>

      {/* Scenario Trigger Dock */}
      <div className="reactor-scenario-dock" aria-label="Simulate deployment events">
        <span className="dock-label">// SIMULATE EVENT DISPATCH:</span>
        <div className="scenario-buttons">
          {SCENARIOS.map((sc) => {
            const isSelected = sc.id === selectedScenarioId;
            return (
              <button
                key={sc.id}
                type="button"
                className={`scenario-btn ${isSelected ? "selected" : ""} ${sc.tone}`}
                onClick={() => triggerScenario(sc.id)}
              >
                <span className="btn-tag">{sc.badge}</span>
                <span className="btn-name">{sc.provider}</span>
                <span className="btn-repo">{sc.repository}</span>
              </button>
            );
          })}
        </div>
      </div>

      {/* Live Pipeline Transit Canvas */}
      <div className="pipeline-transit-zone">
        <div className="pipeline-stages-grid">
          {/* Stage 1: Ingress */}
          <div className={`pipeline-stage ${activeStage >= 0 ? "active" : ""} ${currentScenario.tone}`}>
            <div className="stage-header">
              <span className="stage-num">01</span>
              <span className="stage-name">INGRESS GATEWAY</span>
            </div>
            <div className="stage-meta">
              <span className="mono-badge">HMAC-SHA256</span>
              <span className="stage-stat">
                {currentScenario.hmacVerified ? "✓ VERIFIED" : "✗ REJECTED"}
              </span>
            </div>
          </div>

          <div className={`stage-connector ${activeStage >= 1 ? "flowing" : ""}`} />

          {/* Stage 2: Redis Stream */}
          <div className={`pipeline-stage ${activeStage >= 1 ? "active" : ""} ${currentScenario.tone}`}>
            <div className="stage-header">
              <span className="stage-num">02</span>
              <span className="stage-name">REDIS STREAM</span>
            </div>
            <div className="stage-meta">
              <span className="mono-badge">XADD AT-LEAST-ONCE</span>
              <span className="stage-stat">{currentScenario.ingestLatency}</span>
            </div>
          </div>

          <div className={`stage-connector ${activeStage >= 2 ? "flowing" : ""}`} />

          {/* Stage 3: Worker */}
          <div className={`pipeline-stage ${activeStage >= 2 ? "active" : ""} ${currentScenario.tone}`}>
            <div className="stage-header">
              <span className="stage-num">03</span>
              <span className="stage-name">WORKER CONSUMER</span>
            </div>
            <div className="stage-meta">
              <span className="mono-badge">SANITIZATION</span>
              <span className="stage-stat">REDACT SECRETS</span>
            </div>
          </div>

          <div className={`stage-connector ${activeStage >= 3 ? "flowing" : ""}`} />

          {/* Stage 4: Postgres / DLQ */}
          <div className={`pipeline-stage ${activeStage >= 3 ? "active" : ""} ${currentScenario.tone}`}>
            <div className="stage-header">
              <span className="stage-num">04</span>
              <span className="stage-name">
                {currentScenario.id === "gitlab_failed" ? "DLQ & ALERT" : "POSTGRES STORE"}
              </span>
            </div>
            <div className="stage-meta">
              <span className="mono-badge">DURABLE TX</span>
              <span className="stage-stat">{currentScenario.statusText}</span>
            </div>
          </div>
        </div>

        {/* Live Signal Oscilloscope */}
        <div className="reactor-oscilloscope-container">
          <OscilloscopeCanvas activePulse={isSimulating} tone={currentScenario.tone} />
        </div>
      </div>

      {/* Interactive Payload / Logs Inspector */}
      <div className="reactor-inspector">
        <div className="inspector-tabs">
          <button
            type="button"
            className={`tab-btn ${payloadTab === "logs" ? "active" : ""}`}
            onClick={() => {
              setPayloadTab("logs");
              playTactileSound("tick");
            }}
          >
            // SANITIZED LOG EXCERPT
          </button>
          <button
            type="button"
            className={`tab-btn ${payloadTab === "payload" ? "active" : ""}`}
            onClick={() => {
              setPayloadTab("payload");
              playTactileSound("tick");
            }}
          >
            // INGEST PAYLOAD & CIPHER
          </button>
          <button
            type="button"
            className={`tab-btn ${payloadTab === "curl" ? "active" : ""}`}
            onClick={() => {
              setPayloadTab("curl");
              playTactileSound("tick");
            }}
          >
            // C時代の WEBHOOK CURL
          </button>
        </div>

        <div className="inspector-content">
          {payloadTab === "logs" && (
            <div className="log-viewer">
              <div className="log-header">
                <span className="log-target">
                  <strong>{currentScenario.repository}</strong> @ {currentScenario.environment} [commit: {currentScenario.commit}]
                </span>
                <span className={`status-pill ${currentScenario.tone}`}>
                  {currentScenario.statusText}
                </span>
              </div>
              <pre className="terminal-code">{currentScenario.sanitizedLog}</pre>
            </div>
          )}

          {payloadTab === "payload" && (
            <div className="payload-viewer">
              <div className="crypto-readout">
                <div className="crypto-row">
                  <span className="crypto-key">VERIFICATION HEADER:</span>
                  <span className="crypto-val">{currentScenario.signatureHeader}</span>
                </div>
                <div className="crypto-row">
                  <span className="crypto-key">HMAC SIGNATURE:</span>
                  <span className="crypto-val mono">{currentScenario.signatureVal}</span>
                </div>
                <div className="crypto-row">
                  <span className="crypto-key">INGEST REDIS STREAM:</span>
                  <span className="crypto-val mono">{currentScenario.redisStream}</span>
                </div>
              </div>
              <pre className="terminal-code json">
                {JSON.stringify(currentScenario.rawPayload, null, 2)}
              </pre>
            </div>
          )}

          {payloadTab === "curl" && (
            <div className="curl-viewer">
              <div className="curl-bar">
                <span>SIMULATE SIGNED INGESTION DISPATCH</span>
                <button type="button" className="copy-btn" onClick={copyCurl}>
                  {copied ? "COPIED ✓" : "COPY CURL"}
                </button>
              </div>
              <pre className="terminal-code">
{`curl -X POST https://api.deploypulse.io/webhooks/github \\
  -H "X-Hub-Signature-256: sha256=a92b740ef881c201..." \\
  -H "Content-Type: application/json" \\
  -d '{
    "action": "deployment_status",
    "deployment": {"id": 910442, "sha": "8f92a1b", "ref": "main"},
    "status": "success"
  }'`}
              </pre>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
