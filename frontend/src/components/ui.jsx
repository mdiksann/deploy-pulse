import React from "react";

export function BrandMark({ className = "" }) {
  return <span className={`brand-mark ${className}`} aria-hidden="true"><i /><i /><i /></span>;
}

export function DeployPulseLogo({ className = "" }) {
  return <span className={`deploy-pulse-logo ${className}`} aria-label="Deploy Pulse"><span className="deploy-pulse-wordmark"><span>DEPLOY</span><strong>PULSE</strong></span></span>;
}

export function Brand({ className = "", to = "/" }) {
  return <a className={`brand ${className}`} href={to} aria-label="Deploy Pulse home"><DeployPulseLogo /></a>;
}

export function Arrow() {
  return <svg aria-hidden="true" viewBox="0 0 16 16"><path d="M3 8h9m-4-4 4 4-4 4" /></svg>;
}

export function StatusDot({ tone = "success", label }) {
  return <span className={`status-dot ${tone}`} aria-hidden={!label}>{label ? <><i />{label}</> : <i />}</span>;
}

export function TerminalBadge({ children, tone = "neutral" }) {
  return <span className={`terminal-badge ${tone}`}><i />{children}</span>;
}

export function TerminalPanel({ as: Element = "article", className = "", children }) {
  return <Element className={`terminal-panel ${className}`}>{children}</Element>;
}
