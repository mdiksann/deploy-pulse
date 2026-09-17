import React, { useState } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../auth";
import { BrandMark } from "../components/ui";
import { apiURL } from "../api";
import { StreamIngestVisualizer } from "../components/StreamIngestVisualizer";
import { CapsuleChart } from "../components/CapsuleChart";
import { TiltCard } from "../components/TiltCard";
import { ScrollReveal, AnimatedCounter } from "../components/ScrollReveal";
import { playTactileSound } from "../components/AudioEngine";

const TOPOLOGY_BARS = [
  { label: "GitHub", value: 46, count: 465, color: "#38bdf8" },
  { label: "GitLab", value: 32, count: 320, color: "#06b6d4" },
  { label: "CircleCI", value: 26, count: 260, color: "#22d3ee" },
  { label: "Vercel", value: 19, count: 192, color: "#67e8f9" },
  { label: "AWS", value: 14, count: 140, color: "#7dd3fc" }
];

export function LandingPage() {
  const { user } = useAuth();
  const [activeStep, setActiveStep] = useState(1);
  const [activeTopologyTab, setActiveTopologyTab] = useState(0);

  const scrollToSection = (id) => {
    playTactileSound("tick");
    const target = document.getElementById(id);
    if (target) {
      target.scrollIntoView({ behavior: "smooth" });
    }
  };

  return (
    <div className="landing-page modern-theme cyber-cyan-theme">
      {/* Background Vertical Laser Pinstripes & Atmospheric Glow */}
      <div className="ambient-laser-grid" aria-hidden="true">
        <div className="laser-pinstripe" style={{ left: "10%" }} />
        <div className="laser-pinstripe" style={{ left: "22%" }} />
        <div className="laser-pinstripe" style={{ left: "34%" }} />
        <div className="laser-pinstripe" style={{ left: "50%" }} />
        <div className="laser-pinstripe" style={{ left: "66%" }} />
        <div className="laser-pinstripe" style={{ left: "78%" }} />
        <div className="laser-pinstripe" style={{ left: "90%" }} />
        <div className="ambient-cyan-glow ambient-glow-top" />
        <div className="ambient-cyan-glow ambient-glow-center" />
      </div>

      <a className="skip-link" href="#main-content">
        Skip to content
      </a>

      {/* Floating Modern Header */}
      <div className="floating-header-wrapper">
        <header className="landing-header floating-pill-nav" aria-label="Main Navigation">
          <Link className="landing-brand-pill" to="/" aria-label="Deploy Pulse home">
            <span className="brand-glyph-glow">
              <BrandMark />
            </span>
            <span className="brand-wordmark">
              DEPLOY<strong>PULSE</strong>
            </span>
          </Link>

          <nav aria-label="Primary navigation" className="landing-nav-pill">
            <a href="#hero" onClick={() => playTactileSound("tick")}>How it works?</a>
            <a href="#insights" onClick={() => playTactileSound("tick")}>Features</a>
            <a href="#workflow" onClick={() => playTactileSound("tick")}>Architecture</a>
            <a href="#security" onClick={() => playTactileSound("tick")}>Security</a>
          </nav>

          <div className="header-actions-pill">
            <Link className="landing-signin-pill" to={user ? "/app" : "/login"}>
              {user ? "Console" : "Sign in"}
            </Link>
            <Link className="landing-cta-pill" to={user ? "/app" : "/signup"}>
              {user ? "Open workspace" : "Get started"}
            </Link>
          </div>
        </header>
      </div>

      <main id="main-content">
        {/* ==========================================================================
            HERO SECTION: Glowing Cyan Planetary Horizon & Anchor Box Headline
            ========================================================================== */}
        <section className="landing-hero-modern hero-reference-exact" id="hero" aria-label="Hero Section">
          <div className="hero-content-wrapper">
            {/* Top Eyebrow Tag */}
            <ScrollReveal direction="down" delay={100}>
              <div className="hero-top-badge">
                <span className="badge-sparkle">✦</span>
                <span>Single Workspace CI/CD Observability</span>
                <span className="badge-sparkle">✦</span>
              </div>
            </ScrollReveal>

            {/* Hero Main Headline Matching Reference Image */}
            <ScrollReveal direction="up" delay={200}>
              <h1 className="hero-modern-title reference-headline">
                <span className="headline-row">
                  One-click for{" "}
                  <span className="hero-keyword-box">
                    <span className="anchor-dot tl" />
                    <span className="anchor-dot tr" />
                    <span className="anchor-dot bl" />
                    <span className="anchor-dot br" />
                    release defense
                  </span>
                </span>
                <span className="headline-row accent-cyan">
                  made simple for
                </span>
                <span className="headline-row strong-white">
                  Platform teams
                </span>
              </h1>
            </ScrollReveal>

            {/* Hero Subtitle */}
            <ScrollReveal direction="up" delay={300}>
              <p className="hero-modern-lede">
                Ingest signed webhooks instantly from your CI/CD pipelines, isolate check failures in sub-milliseconds, and keep your production releases safe, fast, and reliable.
              </p>
            </ScrollReveal>

            {/* Hero Pill CTA Button */}
            <ScrollReveal direction="up" delay={400}>
              <div className="hero-modern-actions">
                <Link
                  className="pill-btn pill-btn-cyan-glow"
                  to={user ? "/app" : "/signup"}
                  onClick={() => playTactileSound("tick")}
                >
                  {user ? "Open Console" : "Get started"}
                </Link>
              </div>
            </ScrollReveal>

            {/* Glowing Cyan Planetary Horizon Arc Visual */}
            <div className="hero-planetary-horizon" aria-hidden="true">
              <div className="horizon-ambient-aura" />
              <div className="horizon-arc-glow" />
              <div className="horizon-rim-light" />
              <div className="horizon-inner-curvature" />
              <div className="horizon-vertical-lasers">
                <span className="laser-beam l1" />
                <span className="laser-beam l2" />
                <span className="laser-beam l3" />
                <span className="laser-beam l4" />
                <span className="laser-beam l5" />
                <span className="laser-beam l6" />
                <span className="laser-beam l7" />
              </div>
            </div>

            {/* Bottom Partner Trust Strip */}
            <div className="hero-partner-trust-strip">
              <span className="trust-caption">Trusted across modern engineering stacks</span>
              <div className="partner-logo-cloud">
                <div className="partner-logos-track">
                  <span className="partner-logo-item">● GitHub Actions</span>
                  <span className="partner-logo-item">◆ GitLab CI</span>
                  <span className="partner-logo-item">⦿ CircleCI</span>
                  <span className="partner-logo-item">▲ Vercel</span>
                  <span className="partner-logo-item">⬢ Netlify</span>
                  <span className="partner-logo-item">✦ AWS CodePipeline</span>
                  <span className="partner-logo-item">❖ Slack Alerts</span>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* ==========================================================================
            BENTO INSIGHTS GRID
            ========================================================================== */}
        <section className="bento-insights-section" id="insights" aria-label="Observability Insights">
          <ScrollReveal direction="up">
            <div className="section-title-center">
              <h2>Meet High-Precision Insights</h2>
              <p>Save your platform engineering team&apos;s precious time. DeployPulse replaces the chaos of provider context switching with one normalized timeline.</p>
            </div>
          </ScrollReveal>

          <div className="bento-insights-grid">
            {/* Bento Card 1: Zero-Drop Ingestion Stream Visualizer */}
            <TiltCard className="bento-card bento-card-large bento-stream-card">
              <div className="card-header-split">
                <div>
                  <div className="huge-metric-display">
                    <AnimatedCounter value={99.98} decimals={2} suffix="%" />
                    <span className="metric-dot"> .</span>
                  </div>
                  <span className="metric-sub-label">Ingestion Reliability &middot; HMAC Strictly Enforced</span>
                </div>
                <div className="card-top-tags">
                  <span className="glass-tag">ZERO-DROP INGRESS</span>
                </div>
              </div>

              {/* Real Webhook Stream Visualizer */}
              <div className="bento-stream-viewport">
                <StreamIngestVisualizer />
              </div>

              <div className="bento-card-footer">
                <div className="action-pill-row">
                  <span className="mini-action-pill">
                    <span>◈</span> POST /webhooks/:provider
                  </span>
                  <span className="mini-action-pill">
                    <span>✦</span> HMAC-SHA256 &amp; JWS
                  </span>
                </div>
                <div className="footer-callout">
                  <strong>Signature Ingress Boundary</strong>
                  <p>Every event is cryptographically verified before touching Redis Streams. Invalid signatures return 401 with zero state persisted.</p>
                </div>
              </div>
            </TiltCard>

            {/* Bento Card 2: Release Health & 3D Glowing Capsule Chart */}
            <TiltCard className="bento-card bento-card-large bento-capsule-card">
              <div className="card-header-split">
                <div>
                  <span className="card-eyebrow">// PIPELINE CHECK VELOCITY</span>
                  <h3>Release Check Health &amp; Velocity</h3>
                </div>
                <span className="glass-tag tag-cyan">ALL SYSTEMS OK</span>
              </div>

              {/* Glowing 3D Glass Capsule Bars */}
              <div className="bento-capsule-viewport">
                <CapsuleChart />
              </div>

              <div className="bento-card-footer">
                <div className="footer-callout">
                  <strong>Check Isolation Matrix</strong>
                  <p>Where each build, test, and container check is tracked in real time, exposing failure bottlenecks before they block production.</p>
                </div>
              </div>
            </TiltCard>

            {/* Bento Card 3: Delivery Velocity & Error Redaction Dual Metric Pills */}
            <TiltCard className="bento-card bento-card-small bento-metrics-card">
              <div className="card-header-split">
                <span className="card-eyebrow">// VELOCITY &amp; SECURITY GUARANTEES</span>
              </div>

              <div className="dual-metric-pills">
                <div className="metric-pill-item">
                  <span className="pill-stripe cyan-glow-stripe" />
                  <div className="pill-content">
                    <span className="pill-title">Ingest Latency</span>
                    <strong className="pill-num">
                      <AnimatedCounter value={0.38} decimals={2} suffix="ms" />
                    </strong>
                    <small className="pill-sub">p95 &lt; 1s ACK</small>
                  </div>
                </div>

                <div className="metric-pill-item">
                  <span className="pill-stripe cyan-glow-stripe" />
                  <div className="pill-content">
                    <span className="pill-title">Secret Redaction</span>
                    <strong className="pill-num">
                      <AnimatedCounter value={100} suffix="%" />
                    </strong>
                    <small className="pill-sub">AES-256 Vault Encryption</small>
                  </div>
                </div>
              </div>

              <div className="bento-card-footer">
                <div className="footer-callout">
                  <strong>High-Throughput Delivery Health</strong>
                  <p>Observe deployment frequency across environments with secrets automatically redacted at the worker boundary.</p>
                </div>
              </div>
            </TiltCard>

            {/* Bento Card 4: Multi-Provider Release Topology */}
            <TiltCard className="bento-card bento-card-small bento-topology-card">
              <div className="card-header-split">
                <div>
                  <span className="card-eyebrow">// PROVIDER DISTRIBUTION</span>
                  <h3>Normalized Release Topology</h3>
                </div>
              </div>

              <div className="topology-chart-view">
                <div className="topology-bars-row">
                  {TOPOLOGY_BARS.map((bar, idx) => (
                    <div
                      key={bar.label}
                      className={`topology-bar-item ${activeTopologyTab === idx ? "active" : ""}`}
                      onClick={() => setActiveTopologyTab(idx)}
                      role="button"
                      tabIndex={0}
                    >
                      <div className="top-bar-track">
                        <div
                          className="top-bar-fill"
                          style={{
                            height: `${(bar.value / 50) * 100}%`,
                            backgroundColor: bar.color
                          }}
                        >
                          <span className="top-bar-tooltip">{bar.count}</span>
                        </div>
                      </div>
                      <span className="top-bar-label">{bar.label}</span>
                    </div>
                  ))}
                </div>
              </div>

              <div className="bento-card-footer">
                <div className="topology-pagination-dots">
                  {TOPOLOGY_BARS.map((_, i) => (
                    <span
                      key={i}
                      className={`page-dot ${activeTopologyTab === i ? "active" : ""}`}
                      onClick={() => setActiveTopologyTab(i)}
                    />
                  ))}
                </div>
                <p className="topology-caption">Normalized release timeline across 6 major CI/CD providers without translating different status codes.</p>
              </div>
            </TiltCard>
          </div>
        </section>

        {/* ==========================================================================
            MISSION CONTROL REACTOR SECTION ("Autonomous Deployment Control")
            ========================================================================== */}
        <section className="mission-control-section" id="control" aria-label="Deployment Mission Control">
          <ScrollReveal direction="up">
            <div className="section-title-center">
              <h2>Autonomous Deployment Control</h2>
              <p>Durable stream processing with Redis at-least-once delivery and PostgreSQL atomic transactions.</p>
              <div className="reactor-pill-cta">
                <button
                  type="button"
                  className="pill-btn pill-btn-cyan-sm"
                  onClick={() => scrollToSection("product")}
                >
                  How it works?
                </button>
              </div>
            </div>
          </ScrollReveal>

          <div className="mission-control-canvas">
            {/* Left Column: Live Event Stream Feed Stack */}
            <div className="reactor-feed-column">
              <div className="reactor-stat-header">
                <span className="reactor-stat-lbl">Stream Ingest System</span>
                <strong className="reactor-stat-val">
                  +<AnimatedCounter value={2.7} decimals={1} suffix="k" /> Events
                </strong>
              </div>

              <div className="reactor-feed-stack">
                <div className="feed-card-item">
                  <div className="feed-card-icon">
                    <span className="icon-badge">⇡</span>
                  </div>
                  <div className="feed-card-info">
                    <span className="feed-lbl">Sent &amp; Verified</span>
                    <strong>checkout-service / main</strong>
                    <small>deployment_status.success</small>
                  </div>
                  <div className="feed-card-meta">
                    <span className="feed-amt">0.38ms</span>
                    <span className="feed-token">HMAC-OK</span>
                  </div>
                </div>

                <div className="feed-card-item highlight-card">
                  <div className="feed-card-icon">
                    <span className="icon-badge">🔒</span>
                  </div>
                  <div className="feed-card-info">
                    <span className="feed-lbl">Redacted &amp; Persisted</span>
                    <strong>payment-gateway / prod</strong>
                    <small>secrets sanitized in memory</small>
                  </div>
                  <div className="feed-card-meta">
                    <span className="feed-amt">1,038</span>
                    <span className="feed-token">STREAM</span>
                  </div>
                </div>

                <div className="feed-card-item">
                  <div className="feed-card-icon">
                    <span className="icon-badge">⟳</span>
                  </div>
                  <div className="feed-card-info">
                    <span className="feed-lbl">DLQ Reclaimed</span>
                    <strong>auth-broker / staging</strong>
                    <small>reprocessed via XAUTOCLAIM</small>
                  </div>
                  <div className="feed-card-meta">
                    <span className="feed-amt">4.94s</span>
                    <span className="feed-token">RETRY</span>
                  </div>
                </div>
              </div>
            </div>

            {/* Center/Right: Glowing Step Reactor Dial */}
            <div className="reactor-gauge-column">
              <div className="radial-reactor-gauge">
                <div className="gauge-glow-aura" />
                <svg className="gauge-svg-rings" viewBox="0 0 300 300">
                  <circle cx="150" cy="150" r="130" stroke="rgba(56, 189, 248, 0.2)" strokeWidth="1.5" fill="none" strokeDasharray="4 8" />
                  <circle
                    cx="150"
                    cy="150"
                    r="110"
                    stroke="#38bdf8"
                    strokeWidth="8"
                    fill="none"
                    strokeDasharray="691"
                    strokeDashoffset={691 - (691 * (activeStep / 3))}
                    strokeLinecap="round"
                    transform="rotate(-90 150 150)"
                  />
                  <circle cx="150" cy="150" r="70" stroke="rgba(34, 211, 238, 0.25)" strokeWidth="1" fill="none" />
                </svg>

                <div className="gauge-center-badge">
                  <span className="gauge-spark-icon">⚡</span>
                  <strong>Step 0{activeStep}</strong>
                  <span className="gauge-sub">
                    {activeStep === 1 ? "Ingest & Verify Signature" : activeStep === 2 ? "Redact Secrets & Enqueue" : "Reconcile Status & Alert"}
                  </span>
                </div>

                <div className="gauge-step-switchers">
                  <button
                    type="button"
                    className={`step-nav-btn ${activeStep === 1 ? "active" : ""}`}
                    onClick={() => { setActiveStep(1); playTactileSound("tick"); }}
                  >
                    01
                  </button>
                  <button
                    type="button"
                    className={`step-nav-btn ${activeStep === 2 ? "active" : ""}`}
                    onClick={() => { setActiveStep(2); playTactileSound("tick"); }}
                  >
                    02
                  </button>
                  <button
                    type="button"
                    className={`step-nav-btn ${activeStep === 3 ? "active" : ""}`}
                    onClick={() => { setActiveStep(3); playTactileSound("tick"); }}
                  >
                    03
                  </button>
                </div>
              </div>
            </div>
          </div>

          <div className="reactor-tag-cluster">
            <span className="reactor-pill-tag">✦ 6 Providers Supported</span>
            <span className="reactor-pill-tag">✦ HMAC Signature Verification</span>
            <span className="reactor-pill-tag solid-cyan-active">✦ Zero-Leak Redaction</span>
            <span className="reactor-pill-tag">✦ Redis Streams</span>
            <span className="reactor-pill-tag">✦ Dead-Letter Recovery (DLQ)</span>
            <span className="reactor-pill-tag">✦ Slack &amp; Email Alerts</span>
          </div>
        </section>

        {/* ==========================================================================
            INCIDENT ANALYSIS & LIVE EXCERPT VIEW
            ========================================================================== */}
        <section className="product-section" id="product">
          <ScrollReveal direction="up">
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
          </ScrollReveal>

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
            <TiltCard className="release-example glassmorphic-frame">
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
                    <small>Node v20.12 &bull; npm ci (1.2s)</small>
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
            </TiltCard>
          </div>
        </section>

        {/* ==========================================================================
            WORKFLOW & ARCHITECTURE BLUEPRINT
            ========================================================================== */}
        <section className="workflow-section" id="workflow">
          <ScrollReveal direction="up">
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
          </ScrollReveal>

          <ol className="workflow-steps-list">
            <TiltCard className="workflow-step-card">
              <div className="step-badge">STAGE 01</div>
              <h3>Connect a provider</h3>
              <p>Add a provider connection and configure provider-specific signed webhook delivery with HMAC verification.</p>
            </TiltCard>
            <TiltCard className="workflow-step-card">
              <div className="step-badge">STAGE 02</div>
              <h3>Follow your releases</h3>
              <p>Deployment events stream into Redis, get sanitized by worker groups, and form a durable, searchable timeline.</p>
            </TiltCard>
            <TiltCard className="workflow-step-card">
              <div className="step-badge">STAGE 03</div>
              <h3>Investigate and recover</h3>
              <p>Review failures, configure alerts, and reprocess DLQ events with idempotent guarantees.</p>
            </TiltCard>
          </ol>
        </section>

        {/* ==========================================================================
            CRYPTOGRAPHIC SECURITY BOUNDARIES
            ========================================================================== */}
        <section className="security-section" id="security">
          <ScrollReveal direction="up">
            <div className="security-hero-block">
              <span className="sec-tag">[05 // ZERO-LEAK SECURITY]</span>
              <h2>
                Useful context.
                <br />
                <em>Careful boundaries.</em>
              </h2>
              <p>Release data deserves the same cryptographic care and boundary protection as the systems it deploys.</p>
            </div>
          </ScrollReveal>

          <dl className="security-specs-grid">
            <TiltCard className="security-spec-card">
              <dt>
                <span className="spec-bullet">◆</span> Signed webhooks
              </dt>
              <dd>Provider-specific HMAC verification before events are written to the stream or database.</dd>
            </TiltCard>

            <TiltCard className="security-spec-card">
              <dt>
                <span className="spec-bullet">◆</span> Encrypted secrets
              </dt>
              <dd>Provider secrets are encrypted at rest with mandatory ENCRYPTION_KEY and never returned via the API.</dd>
            </TiltCard>

            <TiltCard className="security-spec-card">
              <dt>
                <span className="spec-bullet">◆</span> Verified access
              </dt>
              <dd>Revocable, database-backed sessions with HttpOnly, SameSite cookies keep workspace access locked.</dd>
            </TiltCard>

            <TiltCard className="security-spec-card">
              <dt>
                <span className="spec-bullet">◆</span> Sanitized excerpts
              </dt>
              <dd>Inspect useful build and test check context while sensitive tokens and credentials are redacted.</dd>
            </TiltCard>
          </dl>
        </section>

        {/* ==========================================================================
            FINAL CTA LAUNCH TERMINAL
            ========================================================================== */}
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
              <Link className="pill-btn pill-btn-cyan-glow cta-btn" to={user ? "/app" : "/signup"}>
                {user ? "Open workspace" : "Get started"}
              </Link>
            </div>
          </div>
        </section>
      </main>

      {/* ==========================================================================
          HIGH-CRAFT GLASSMORPHIC FOOTER
          ========================================================================== */}
      <footer className="landing-footer">
        <div className="footer-top-grid">
          <div className="footer-brand-col">
            <Link className="landing-brand-pill" to="/">
              <span className="brand-glyph-glow"><BrandMark /></span>
              <span className="brand-wordmark">
                DEPLOY<strong>PULSE</strong>
              </span>
            </Link>
            <p className="footer-motto">High-precision observability for platform and release engineering.</p>
          </div>

          <nav aria-label="Footer navigation" className="footer-nav-grid">
            <div className="footer-col">
              <span className="col-title">// SYSTEM</span>
              <a href="#hero">Overview</a>
              <a href="#insights">Features</a>
              <a href="#workflow">Architecture</a>
              <a href="#security">Security</a>
            </div>
            <div className="footer-col">
              <span className="col-title">// WORKSPACE</span>
              <a href={apiURL("/healthz")}>API health</a>
              <Link to="/login">Sign in</Link>
              <Link to="/signup">Register</Link>
            </div>
          </nav>
        </div>

        <div className="footer-bottom-bar">
          <span className="footer-credit">&copy; {new Date().getFullYear()} DeployPulse Platform Engineering. Built for the people behind the release.</span>
          <div className="footer-social-pills">
            <span className="social-pill">𝕏</span>
            <span className="social-pill">in</span>
            <span className="footer-version">V2.9_STABLE</span>
          </div>
        </div>
      </footer>
    </div>
  );
}
