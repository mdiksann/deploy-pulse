import React, { useEffect } from "react";
import { Link } from "react-router-dom";
import Lenis from "lenis";
import { useAuth } from "../auth";
import { DeployPulseLogo } from "../components/ui";
import { AsciiAnimationBackground } from "../components/ui/hero-ascii-one";
import { apiURL } from "../api";
import "lenis/dist/lenis.css";

const providers = [
  "GitHub Actions",
  "GitLab CI",
  "CircleCI",
  "Vercel",
  "Netlify",
  "AWS CodePipeline"
];

const workflow = [
  {
    code: "01",
    title: "Receive the event",
    copy: "Provider webhooks are checked for a valid signature before they are accepted."
  },
  {
    code: "02",
    title: "Normalize the release",
    copy: "Different provider payloads become one timeline with a shared status, repository, branch, and environment."
  },
  {
    code: "03",
    title: "Investigate the outcome",
    copy: "Open checks, failure context, and sanitized log excerpts from the release that needs attention."
  }
];

const capabilities = [
  {
    code: "A",
    title: "A single release timeline",
    copy: "Filter deployment history by repository, provider, status, environment, actor, or time range."
  },
  {
    code: "B",
    title: "Useful failure context",
    copy: "See which check failed and inspect the most relevant sanitized log lines without storing raw logs."
  },
  {
    code: "C",
    title: "Operator controls",
    copy: "Configure Slack or email alerts and reprocess events that are waiting in the dead-letter queue."
  }
];

export function LandingPage() {
  const { user } = useAuth();
  const primaryHref = user ? "/app" : "/signup";
  const primaryLabel = user ? "Open console" : "Get started";

  useEffect(() => {
    if (!window.matchMedia || window.matchMedia("(prefers-reduced-motion: reduce)").matches) return undefined;

    const lenis = new Lenis({
      anchors: true,
      autoRaf: true,
      duration: 1.1,
      smoothWheel: true
    });

    return () => lenis.destroy();
  }, []);

  return (
    <div className="landing-page">
      <AsciiAnimationBackground className="landing-sonar-layer" />
      <a className="skip-link" href="#main-content">Skip to content</a>

      <header className="landing-header">
        <Link className="landing-logo" to="/" aria-label="Deploy Pulse home">
          <DeployPulseLogo />
        </Link>
        <span className="header-meta">est. 2025</span>

        <nav className="landing-nav" aria-label="Primary navigation">
          <a href="#features">Features</a>
          <a href="#workflow">Architecture</a>
          <a href="#security">Security</a>
          <Link to="/login">Sign in</Link>
          <Link className="header-cta" to={primaryHref}>{primaryLabel}</Link>
        </nav>
      </header>

      <main id="main-content">
        <section className="landing-hero" id="hero" aria-labelledby="hero-title">
          <div className="hero-copy">
            <p className="section-code">deploy pulse / release watch</p>
            <h1 id="hero-title">Releases in view</h1>
            <p className="hero-lede">
              Signed webhooks, normalized deployment events, and failure context in one quiet control room. Keep production releases visible from commit to recovery.
            </p>
            <div className="hero-actions">
              <Link className="outline-button button-primary" to={primaryHref}>{primaryLabel}</Link>
              <a className="outline-button" href="#workflow">See how it works</a>
            </div>
          </div>
        </section>

        <section className="status-strip" aria-label="System status">
          <span>system.active</span>
          <span>ingestion.active</span>
          <span className="status-spacer" />
          <span className="status-live"><i /> monitoring</span>
          <span>events: live</span>
        </section>

        <section className="landing-section feature-section" id="features" aria-labelledby="features-title">
          <div className="section-heading">
            <div>
              <p className="section-code">01 / features</p>
              <h2 id="features-title">Less hunting.<br />More understanding.</h2>
            </div>
            <p className="section-intro">
              Deploy Pulse gives deployment events a shared shape so teams can follow what shipped and focus on what needs attention.
            </p>
          </div>

          <div className="capability-grid">
            {capabilities.map((item) => (
              <article className="capability" key={item.code}>
                <span className="capability-code">{item.code}</span>
                <h3>{item.title}</h3>
                <p>{item.copy}</p>
              </article>
            ))}
          </div>
        </section>

        <section className="landing-section workflow-section" id="workflow" aria-labelledby="workflow-title">
          <div className="section-heading section-heading-narrow">
            <div>
              <p className="section-code">02 / architecture</p>
              <h2 id="workflow-title">From webhook<br />to clear action.</h2>
            </div>
            <p className="section-intro">The path is deliberately small: verify, process, and make the outcome easy to inspect.</p>
          </div>

          <ol className="workflow-list">
            {workflow.map((item) => (
              <li className="workflow-item" key={item.code}>
                <span className="workflow-code">{item.code}</span>
                <div>
                  <h3>{item.title}</h3>
                  <p>{item.copy}</p>
                </div>
                <span className="workflow-mark" aria-hidden="true">+</span>
              </li>
            ))}
          </ol>
        </section>

        <section className="landing-section security-section" id="security" aria-labelledby="security-title">
          <div className="security-copy">
            <p className="section-code">03 / security boundary</p>
            <h2 id="security-title">Useful signal.<br />Careful boundaries.</h2>
            <p>Release visibility should not mean copying sensitive data everywhere. The project keeps verification and redaction close to the ingestion path.</p>
          </div>
          <dl className="security-list">
            <div><dt>Signed webhooks</dt><dd>Invalid signatures are rejected before persistence.</dd></div>
            <div><dt>Encrypted secrets</dt><dd>Provider secrets are encrypted at rest and never returned by the API.</dd></div>
            <div><dt>Sanitized excerpts</dt><dd>Token, password, and private-key patterns are redacted before logs are stored.</dd></div>
            <div><dt>Session access</dt><dd>Workspace access uses revocable, database-backed HttpOnly sessions.</dd></div>
          </dl>
        </section>

        <section className="provider-section" aria-labelledby="providers-title">
          <div className="provider-heading">
            <p className="section-code">supported inputs</p>
            <h2 id="providers-title">One timeline across the providers you already use.</h2>
          </div>
          <ul className="provider-list">
            {providers.map((provider) => <li key={provider}>{provider}</li>)}
          </ul>
        </section>

        <section className="landing-cta" aria-labelledby="cta-title">
          <div>
            <p className="section-code">release watch / ready</p>
            <h2 id="cta-title">Put every release<br />in view.</h2>
          </div>
          <div className="cta-copy">
            <p>Create a workspace, connect a provider, and start with the events your system already emits.</p>
            <Link className="outline-button button-primary" to={primaryHref}>{primaryLabel}</Link>
          </div>
        </section>
      </main>

      <footer className="landing-footer">
        <div className="footer-main">
          <div>
            <Link className="landing-logo" to="/"><DeployPulseLogo /></Link>
            <p>Deployment observability for the people behind the release.</p>
          </div>
          <nav aria-label="Footer navigation">
            <a href="#features">Features</a>
            <a href="#workflow">Architecture</a>
            <a href="#security">Security</a>
            <a href={apiURL("/healthz")}>API health</a>
          </nav>
        </div>
        <div className="footer-bottom">
          <span>© {new Date().getFullYear()} Deploy Pulse</span>
          <span>single workspace / release watch</span>
        </div>
      </footer>
    </div>
  );
}
