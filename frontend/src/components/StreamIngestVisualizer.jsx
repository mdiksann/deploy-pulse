import React, { useState, useEffect } from "react";

const SAMPLE_EVENTS = [
  { id: "e1", provider: "GitHub", repo: "checkout-service", event: "deployment_status.success", latency: "0.38ms", hmac: "sha256=a92b...ok", status: "success" },
  { id: "e2", provider: "GitLab", repo: "payment-gateway", event: "pipeline.failed", latency: "0.42ms", hmac: "token=gl_sig...ok", status: "failed" },
  { id: "e3", provider: "Vercel", repo: "web-dashboard", event: "deployment.ready", latency: "0.29ms", hmac: "sha1=vc_sig...ok", status: "success" },
  { id: "e4", provider: "AWS", repo: "order-pipeline", event: "execution.running", latency: "0.61ms", hmac: "relay=sig256...ok", status: "running" }
];

export function StreamIngestVisualizer() {
  const [activeIdx, setActiveIdx] = useState(0);

  useEffect(() => {
    const timer = setInterval(() => {
      setActiveIdx((prev) => (prev + 1) % SAMPLE_EVENTS.length);
    }, 3200);
    return () => clearInterval(timer);
  }, []);

  const activeEvent = SAMPLE_EVENTS[activeIdx];

  return (
    <div className="stream-visualizer-container" aria-label="Live Ingestion Stream">
      <div className="stream-hud-bar">
        <div className="hud-metric">
          <span className="hud-label">INGESTION LATENCY</span>
          <strong className="hud-val">{activeEvent.latency}</strong>
        </div>
        <div className="hud-metric">
          <span className="hud-label">SIGNATURE VALIDATION</span>
          <span className="hud-tag-verified">HMAC STRICT</span>
        </div>
      </div>

      <div className="stream-event-cards-track">
        {SAMPLE_EVENTS.map((item, idx) => {
          const isActive = idx === activeIdx;
          return (
            <div
              key={item.id}
              className={`stream-event-card ${isActive ? "active" : ""}`}
              onClick={() => setActiveIdx(idx)}
              role="button"
              tabIndex={0}
            >
              <div className="event-card-left">
                <span className={`event-status-pip ${item.status}`} />
                <div className="event-info">
                  <strong>{item.repo}</strong>
                  <span className="event-meta">{item.provider} &bull; {item.event}</span>
                </div>
              </div>
              <div className="event-card-right">
                <span className="event-hmac-badge">{item.hmac}</span>
                <span className="event-time-badge">{item.latency}</span>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
