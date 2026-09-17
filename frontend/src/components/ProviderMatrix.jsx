import React, { useState } from "react";
import { playTactileSound } from "./AudioEngine";

const PROVIDERS = [
  {
    name: "GitHub Actions",
    authHeader: "X-Hub-Signature-256",
    cipher: "HMAC-SHA256",
    retryMode: "XAUTOCLAIM 5x Reclaim",
    latencyP99: "0.38ms",
    description: "Full support for deployment_status, workflow_job, and check_run events with payload verification.",
    activeBadge: "NATIVE INTEGRATION"
  },
  {
    name: "GitLab CI",
    authHeader: "Webhook-Signature / Token",
    cipher: "Signing Token / HMAC",
    retryMode: "Durable Redis Queue",
    latencyP99: "0.42ms",
    description: "Ingests pipeline and job hooks with sanitized environment variables and multi-branch tracking.",
    activeBadge: "ENTERPRISE READY"
  },
  {
    name: "CircleCI",
    authHeader: "Circleci-Signature",
    cipher: "v1 HMAC-SHA256",
    retryMode: "DLQ Re-queue",
    latencyP99: "0.40ms",
    description: "Workflow status changes captured and parsed into consolidated unified release checkpoints.",
    activeBadge: "HIGH THROUGHPUT"
  },
  {
    name: "Vercel",
    authHeader: "X-Vercel-Signature",
    cipher: "HMAC-SHA1",
    retryMode: "At-Least-Once",
    latencyP99: "0.29ms",
    description: "Instant preview and production deploy webhook ingestion with automated alias reconciliation.",
    activeBadge: "EDGE OPTIMIZED"
  },
  {
    name: "Netlify",
    authHeader: "X-Webhook-Signature",
    cipher: "HS256 JWS",
    retryMode: "Signed Verification",
    latencyP99: "0.34ms",
    description: "JWT-based signed webhook authentication for branch deploys and deploy-preview audits.",
    activeBadge: "ZERO-TRUST"
  },
  {
    name: "AWS CodePipeline",
    authHeader: "X-DeployPulse-Relay-Sig",
    cipher: "Internal HMAC-SHA256",
    retryMode: "Relay Gateway",
    latencyP99: "0.55ms",
    description: "Secure relay endpoint for EventBridge and SNS notifications with zero open inbound AWS ports.",
    activeBadge: "CLOUD RELAY"
  }
];

export function ProviderMatrix() {
  const [activeProviderIndex, setActiveProviderIndex] = useState(0);

  const selectProvider = (idx) => {
    setActiveProviderIndex(idx);
    playTactileSound("tick");
  };

  const current = PROVIDERS[activeProviderIndex];

  return (
    <div className="provider-matrix-wrapper">
      <div className="provider-tabs-rail" role="tablist" aria-label="Supported CI/CD Providers">
        {PROVIDERS.map((p, idx) => {
          const isSelected = idx === activeProviderIndex;
          return (
            <button
              key={p.name}
              role="tab"
              aria-selected={isSelected}
              className={`provider-tab-chip ${isSelected ? "active" : ""}`}
              onClick={() => selectProvider(idx)}
            >
              <span className="chip-indicator" />
              <span className="chip-name">{p.name}</span>
            </button>
          );
        })}
      </div>

      <div className="provider-spec-display" role="tabpanel">
        <div className="spec-corner tl">+</div>
        <div className="spec-corner tr">+</div>
        <div className="spec-corner bl">+</div>
        <div className="spec-corner br">+</div>

        <div className="spec-header">
          <div className="spec-title-group">
            <span className="spec-sub">// PROVIDER PROTOCOL SPEC</span>
            <h3>{current.name}</h3>
          </div>
          <span className="spec-badge">{current.activeBadge}</span>
        </div>

        <p className="spec-desc">{current.description}</p>

        <div className="spec-grid">
          <div className="spec-card">
            <span className="spec-label">SIGNATURE AUTH HEADER</span>
            <code className="spec-value">{current.authHeader}</code>
          </div>
          <div className="spec-card">
            <span className="spec-label">CRYPTOGRAPHIC CIPHER</span>
            <code className="spec-value">{current.cipher}</code>
          </div>
          <div className="spec-card">
            <span className="spec-label">RETRY & CONSUMPTION GUARANTEE</span>
            <code className="spec-value">{current.retryMode}</code>
          </div>
          <div className="spec-card">
            <span className="spec-label">INGEST P99 LATENCY</span>
            <code className="spec-value highlight">{current.latencyP99}</code>
          </div>
        </div>
      </div>
    </div>
  );
}
