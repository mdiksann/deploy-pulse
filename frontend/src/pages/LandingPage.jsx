import React, { useState } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../auth";
import { BrandMark } from "../components/ui";
import { apiURL } from "../api";
import { HeroTelemetryReactor } from "../components/HeroTelemetryReactor";
import { ProviderMatrix } from "../components/ProviderMatrix";
import { playTactileSound } from "../components/AudioEngine";

const PROVIDERS = [
  "GitHub Actions",
  "GitLab CI",
  "CircleCI",
  "Vercel",
  "Netlify",
  "AWS CodePipeline"
];

export function LandingPage() {
  const { user } = useAuth();
  const [activeCheckTab, setActiveCheckTab] = useState("failed");

  return (
    <div className="landing-page">
      <a className="skip-link" href="#main-content">
        Skip to content
      </a>

      {/* Top Telemetry HUD Ticker */}
      <div className="telemetry-top-strip" aria-label="System status telemetry">
        <div className="strip-inner">
          <div className="strip-item">
            <span className="live-indicator"><i />STREAM: ACTIVE</span>
          </div>
          <div className="strip-item">
            <span className="mono-stat">INGEST_LATENCY: <strong>0.38ms</strong></span>
          </div>
          <div className="strip-item">
            <span className="mono-stat">HMAC_VERIFICATION: <strong>STRICT_ENFORCED</strong></span>
          </div>
          <div className="strip-item hide-mobile">
            <span className="mono-stat">PROTOCOL: <strong>REDIS_XADD // POSTGRES_TX</strong></span>
          </div>
        </div>
      </div>

      {/* Global Header */}
      <header className="landing-header">
        <Link className="landing-brand" to="/" aria-label="Deploy Pulse home">
          <BrandMark />
          <span className="brand-wordmark">
            DEPLOY<strong>PULSE</strong>
          </span>
        </Link>
        <nav aria-label="Primary navigation" className="landing-nav">
          <a href="#product">// PRODUCT</a>
          <a href="#providers">// PROVIDERS</a>
          <a href="#workflow">// ARCHITECTURE</a>
          <a href="#security">// CRYPTO_BOUNDARIES</a>
        </nav>
        <div className="header-actions">
          <Link className="landing-signin" to={user ? "/app" : "/login"}>
            {user ? "Open workspace" : "Sign in"}
          </Link>
          <Link className="landing-button-sm hide-mobile" to={user ? "/app" : "/signup"}>
            {user ? "Console →" : "Get started"}
          </Link>
        </div>
      </header>

      <main id="main-content">
        {/* Asymmetrical Hero Section with Live Telemetry Reactor */}
        <section className="landing-hero" aria-label="Hero section">
          <div className="hero-grid">
            {/* Left Column: Asymmetric Editorial Masthead */}
            <div className="hero-editorial">
              <div className="hero-badge">
                <span className="badge-bullet">◈</span>
                <p className="product-description">DEPLOYMENT MONITORING FOR PLATFORM TEAMS</p>
              </div>

              <h1 className="hero-title">
                Monitor deployments.
                <br />
                <em>Know what needs you.</em>
              </h1>

              <p className="hero-lede">
                Ingest signed webhooks from every CI/CD provider into a durable, sanitized deployment timeline. 
                Isolate check failures in sub-milliseconds without switching tabs or leaking environment secrets.
              </p>

              <div className="hero-actions">
                <Link
                  className="landing-button"
                  to={user ? "/app" : "/signup"}
                  onClick={() => playTactileSound("tick")}
                >
                  <span className="btn-bracket">[</span>
                  {user ? "Open workspace" : "Get started"}
                  <span className="btn-bracket">]</span>
                </Link>
                <a
                  className="landing-text-link"
                  href="#product"
                  onClick={() => playTactileSound("tick")}
                >
                  Explore the workflow <span aria-hidden="true">↓</span>
                </a>
              </div>

              {/* Technical Footnote & Coordinate Grid */}
              <div className="hero-technical-specs">
                <div className="tech-spec-item">
                  <span className="spec-code">[01 // REDIS STREAMS]</span>
                  <p>At-least-once ingest with XAUTOCLAIM dead-letter recovery.</p>
                </div>
                <div className="tech-spec-item">
                  <span className="spec-code">[02 // ENCRYPTED SECRETS]</span>
                  <p>AES-256 encrypted provider keys with sanitized log excerpts.</p>
                </div>
              </div>

              <p className="hero-footnote">
                Your existing pipelines. A clearer view of every release.
              </p>
            </div>

            {/* Right Column: Interactive Live Telemetry Reactor */}
            <div className="hero-reactor-column">
              <HeroTelemetryReactor />
            </div>
          </div>
        </section>

        {/* Provider Compatibility & Cryptographic Matrix */}
        <section className="provider-strip" id="providers" aria-label="Supported providers">
          <div className="section-label-bar">
            <span className="sec-tag">[02 // COMPATIBILITY]</span>
            <p>Works with the tools you already ship with</p>
            <span className="sec-tag-end">HMAC_V2</span>
          </div>

          <ProviderMatrix />
        </section>

        {/* Product Deep-Dive: Asymmetric Incident Breakdown */}
        <section className="product-section" id="product">
          <div className="section-intro">
            <div className="intro-title-block">
              <span className="section-index-num">03 // ANALYSIS</span>
              <h2>
                Less hunting.
                <br />
                <em>More understanding.</em>
              </h2>
            </div>
            <p>
              A failed release usually means another round of tabs. Keep the deployment, its checks, 
              and the useful part of its logs together in one unified, high-density control surface.
            </p>
          </div>

          <div className="product-story">
            <div className="story-copy">
              <div className="story-badge">// INCIDENT ISOLATION</div>
              <h3>Start with the releases that need attention.</h3>
              <p>
                Find failed deployments by repository and environment instantly. Open the details to see which check failed 
                and inspect a sanitized log excerpt with secrets automatically redacted at the worker boundary.
              </p>
              <div className="story-metrics-grid">
                <div className="metric-box">
                  <span className="metric-val">100%</span>
                  <span className="metric-lbl">Signed Webhook Ingest</span>
                </div>
                <div className="metric-box">
                  <span className="metric-val">&lt; 2ms</span>
                  <span className="metric-lbl">Sanitized Timeline Write</span>
                </div>
              </div>
              <p className="story-note">Filter by status, environment, or time range.</p>
            </div>

            {/* Interactive Release Incident View */}
            <figure className="release-example">
              <figcaption>
                <div className="fig-title">
                  <span className="crosshair">+</span> Inside a deployment
                </div>
                <span>Illustrative incident view</span>
              </figcaption>

              <div className="example-heading">
                <div className="heading-repo-info">
                  <strong>checkout-service</strong>
                  <span className="repo-branch">production / main</span>
                </div>
                <span className="example-failed">Build failed</span>
              </div>

              <div className="example-checks-list">
                <div className="example-check passed">
                  <span className="check-icon" aria-hidden="true">✓</span>
                  <div className="check-detail">
                    <span>Install dependencies</span>
                    <small>Node v20.12 • npm ci (1.2s)</small>
                  </div>
                  <span className="check-status-tag">Passed</span>
                </div>

                <div className="example-check failed">
                  <span className="check-icon" aria-hidden="true">!</span>
                  <div className="check-detail">
                    <span>Build application container</span>
                    <small>Docker build stage 2/4</small>
                  </div>
                  <span className="check-status-tag error">Failed</span>
                </div>
              </div>

              <div className="terminal-excerpt-wrapper">
                <div className="terminal-bar">
                  <span>LOG TRACE // CONTAINER_BUILD</span>
                  <span className="sanitized-badge">SANITIZED</span>
                </div>
                <pre>
{`[FATAL] Missing environment variable:
PAYMENTS_API_URL is undefined.
Checked: production secrets vault (encrypted: AES-256)
Process terminated with exit code 1.`}
                </pre>
              </div>

              <p className="example-caption">The failure, with the context to investigate.</p>
            </figure>
          </div>

          <div className="capability-list">
            <article className="capability-card">
              <div className="card-top">
                <span className="card-num">01</span>
                <span className="card-accent" />
              </div>
              <h3>One release history</h3>
              <p>Compare deployment activity across providers without translating six different status formats or authorization headers.</p>
            </article>

            <article className="capability-card">
              <div className="card-top">
                <span className="card-num">02</span>
                <span className="card-accent" />
              </div>
              <h3>Alerts with a purpose</h3>
              <p>Configure failure notifications for the repositories and environments your platform engineering team actually owns.</p>
            </article>

            <article className="capability-card">
              <div className="card-top">
                <span className="card-num">03</span>
                <span className="card-accent" />
              </div>
              <h3>A path to recovery</h3>
              <p>Review events in the Redis & PostgreSQL Dead Letter Queue (DLQ) and safely reprocess them when systems recover.</p>
            </article>
          </div>
        </section>

        {/* Workflow Pipeline Blueprint */}
        <section className="workflow-section" id="workflow">
          <div className="workflow-intro">
            <span className="sec-tag">[04 // ARCHITECTURE]</span>
            <h2>
              From your pipeline
              <br />
              <em>to your next step.</em>
            </h2>
            <p className="workflow-subtext">
              Zero open inbound ports to your internal infra. Pure signed HTTP webhook ingestion into Redis Streams and atomic PostgreSQL transactions.
            </p>
          </div>

          <ol className="workflow-steps-list">
            <li className="workflow-step-card">
              <div className="step-badge">STAGE 01</div>
              <h3>Connect a provider</h3>
              <p>Add a provider connection and configure provider-specific signed webhook delivery with HMAC verification.</p>
            </li>
            <li className="workflow-step-card">
              <div className="step-badge">STAGE 02</div>
              <h3>Follow your releases</h3>
              <p>Deployment events stream into Redis, get sanitized by worker groups, and form a durable, searchable timeline.</p>
            </li>
            <li className="workflow-step-card">
              <div className="step-badge">STAGE 03</div>
              <h3>Investigate and recover</h3>
              <p>Review failures, configure alerts, and reprocess DLQ events with idempotent guarantees.</p>
            </li>
          </ol>
        </section>

        {/* Cryptographic Security Boundaries */}
        <section className="security-section" id="security">
          <div className="security-hero-block">
            <span className="sec-tag">[05 // ZERO-LEAK SECURITY]</span>
            <h2>
              Useful context.
              <br />
              <em>Careful boundaries.</em>
            </h2>
            <p>Release data deserves the same cryptographic care and boundary protection as the systems it deploys.</p>
          </div>

          <dl className="security-specs-grid">
            <div className="security-spec-card">
              <dt>
                <span className="spec-bullet">◆</span> Signed webhooks
              </dt>
              <dd>Provider-specific HMAC verification before events are written to the stream or database.</dd>
            </div>

            <div className="security-spec-card">
              <dt>
                <span className="spec-bullet">◆</span> Encrypted secrets
              </dt>
              <dd>Provider secrets are encrypted at rest with mandatory ENCRYPTION_KEY and never returned via the API.</dd>
            </div>

            <div className="security-spec-card">
              <dt>
                <span className="spec-bullet">◆</span> Verified access
              </dt>
              <dd>Revocable, database-backed sessions with HttpOnly, SameSite cookies keep workspace access locked.</dd>
            </div>

            <div className="security-spec-card">
              <dt>
                <span className="spec-bullet">◆</span> Sanitized excerpts
              </dt>
              <dd>Inspect useful build and test check context while sensitive tokens and credentials are redacted.</dd>
            </div>
          </dl>
        </section>

        {/* Final CTA / Launch Terminal */}
        <section className="final-cta">
          <div className="cta-container">
            <div className="cta-content">
              <span className="cta-eyebrow">// MISSION_CONTROL_READY</span>
              <h2>
                Give your releases
                <br />
                <em>a place to land.</em>
              </h2>
            </div>
            <div className="cta-action-box">
              <p>Set up your workspace and connect your first provider in under two minutes.</p>
              <Link className="landing-button cta-btn" to={user ? "/app" : "/signup"}>
                {user ? "Open workspace" : "Get started"}
              </Link>
            </div>
          </div>
        </section>
      </main>

      {/* High-Craft Footer */}
      <footer className="landing-footer">
        <div className="footer-top-grid">
          <div className="footer-brand-col">
            <Link className="landing-brand" to="/">
              <BrandMark />
              <span className="brand-wordmark">
                DEPLOY<strong>PULSE</strong>
              </span>
            </Link>
            <p className="footer-motto">High-precision observability for platform and release engineering.</p>
          </div>

          <nav aria-label="Footer navigation" className="footer-nav-grid">
            <div className="footer-col">
              <span className="col-title">// SYSTEM</span>
              <a href="#product">Product</a>
              <a href="#workflow">Workflow</a>
              <a href="#providers">Providers</a>
            </div>
            <div className="footer-col">
              <span className="col-title">// SECURITY</span>
              <a href="#security">Security</a>
              <a href={apiURL("/healthz")}>API health</a>
              <Link to="/login">Sign in</Link>
            </div>
          </nav>
        </div>

        <div className="footer-bottom-bar">
          <span className="footer-credit">Built for the people behind the release.</span>
          <span className="footer-version">DEPLOY_PULSE // V2.4_STABLE // SHA_8F92A</span>
        </div>
      </footer>
    </div>
  );
}
