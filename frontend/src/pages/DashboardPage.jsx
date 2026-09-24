import React, { useEffect, useRef, useState } from "react";
import {
  Bell,
  Cable,
  LayoutDashboard,
  Menu,
  PanelLeftClose,
  RefreshCw,
  Rocket,
} from "lucide-react";
import { Link, NavLink, Navigate, useLocation } from "react-router-dom";
import { api } from "../api";
import ProfileMenu from "../components/ProfileMenu";
import { BrandMark, DeployPulseLogo, TerminalBadge } from "../components/ui";

const providers = {
  github: "GitHub Actions",
  gitlab: "GitLab CI",
  circleci: "CircleCI",
  vercel: "Vercel",
  netlify: "Netlify",
  "aws-codepipeline": "AWS CodePipeline",
};

const pages = {
  "/app": "Overview",
  "/app/deployments": "Deployments",
  "/app/alerts": "Alerts",
  "/app/connections": "Connections",
};

const navItems = [
  { to: "/app", label: "Overview", icon: LayoutDashboard, end: true },
  { to: "/app/deployments", label: "Deployments", icon: Rocket },
  { to: "/app/alerts", label: "Alerts", icon: Bell },
  { to: "/app/connections", label: "Connections", icon: Cable },
];

const pretty = (value = "") => value.replace(/[-_]/g, " ").replace(/\b\w/g, (character) => character.toUpperCase());
const ago = (value) => {
  const minutes = Math.max(0, Math.round((Date.now() - new Date(value).getTime()) / 60000));
  return minutes < 2 ? "just now" : minutes < 60 ? `${minutes}m ago` : minutes < 1440 ? `${Math.round(minutes / 60)}h ago` : `${Math.round(minutes / 1440)}d ago`;
};

function Status({ value }) {
  return <TerminalBadge tone={value}>{pretty(value)}</TerminalBadge>;
}

function Sidebar({ collapsed, health, onToggle }) {
  return <aside className="sidebar">
    <div className="sidebar-head">
      <Link className="brand" to="/app" aria-label="Deploy Pulse overview">
        <BrandMark />
        <DeployPulseLogo />
      </Link>
      <button
        className="sidebar-toggle"
        type="button"
        aria-label={collapsed ? "Expand sidebar" : "Minimize sidebar"}
        aria-expanded={!collapsed}
        onClick={onToggle}
      >
        {collapsed ? <Menu aria-hidden="true" /> : <PanelLeftClose aria-hidden="true" />}
      </button>
    </div>
    <nav className="nav-list" aria-label="Workspace navigation">
      {navItems.map(({ to, label, icon: Icon, end }) => <NavLink
        key={to}
        className="nav-link"
        to={to}
        end={end}
        aria-label={collapsed ? label : undefined}
        title={collapsed ? label : undefined}
      >
        <Icon aria-hidden="true" />
        <span className="nav-label">{label}</span>
      </NavLink>)}
    </nav>
    <div className="sidebar-bottom">
      <p title={health?.status === "ok" ? "All systems reporting" : "Checking runtime health"}>
        <span className={`status-dot ${health?.status === "ok" ? "success" : "muted"}`} />
        <span className="sidebar-status-label">{health?.status === "ok" ? "All systems reporting" : "Checking runtime health"}</span>
      </p>
    </div>
  </aside>;
}

function Topbar({ busy, pageTitle, onRefresh }) {
  return <header className="topbar">
    <div className="crumb"><span className="live-dot" />Live workspace <span className="slash">/</span><strong>{pageTitle}</strong></div>
    <div className="top-actions">
      <span className="sync-status">{busy ? "Syncing" : "Updated"}</span>
      <button className="icon-button" aria-label={`Refresh ${pageTitle.toLowerCase()}`} onClick={onRefresh} disabled={busy}>
        <RefreshCw aria-hidden="true" />
      </button>
      <ProfileMenu />
    </div>
  </header>;
}

function ScopeControls({ filters, setFilters, showStatus = false }) {
  return <div className="scope-controls">
    <label>Environment<select value={filters.environment} onChange={(event) => setFilters({ ...filters, environment: event.target.value })}>
      <option value="">All environments</option><option value="production">Production</option><option value="staging">Staging</option>
    </select></label>
    {showStatus && <label>Status<select value={filters.status} onChange={(event) => setFilters({ ...filters, status: event.target.value })}>
      <option value="">Every status</option><option value="failed">Failed</option><option value="running">Running</option><option value="queued">Queued</option><option value="success">Successful</option>
    </select></label>}
    <label>Range<select value={filters.days} onChange={(event) => setFilters({ ...filters, days: Number(event.target.value) })}>
      <option value="1">Last 24 hours</option><option value="7">Last 7 days</option><option value="30">Last 30 days</option>
    </select></label>
  </div>;
}

function OverviewPage({ analytics, deployments, filters, health, openDetail, setFilters }) {
  const failed = deployments.filter((deployment) => deployment.status === "failed");
  const summary = analytics?.summary || {};
  return <>
    <section className="intro">
      <div><p className="section-note">Release watch</p><h1>{failed.length ? `${failed.length} deployment${failed.length === 1 ? " needs" : "s need"} attention.` : "Release activity is stable."}</h1><p className="intro-copy">{summary.total || 0} deployments in the selected window · {Number(summary.success_rate || 0).toFixed(1)}% success rate.</p></div>
      <ScopeControls filters={filters} setFilters={setFilters} />
    </section>
    <section className="focus-grid">
      <article className="failure-rail">
        <div className="panel-heading"><div><p className="section-note">Needs attention</p><h2>Failure rail</h2></div><span className="failure-number">{String(failed.length).padStart(2, "0")}</span></div>
        <div className="rail-list">{failed.length ? failed.slice(0, 3).map((deployment) => <button className="incident-row" key={deployment.id} onClick={() => openDetail(deployment)}><span className="incident-time"><i />{ago(deployment.started_at)}</span><span className="incident-body"><span className="incident-title"><strong>{deployment.repository}</strong><code>{deployment.commit_sha?.slice(0, 7)}</code></span><p>{pretty(deployment.failure_category || "provider failure")} · {deployment.environment}</p></span><span className="incident-cta">Inspect</span></button>) : <div className="empty-row">No failed deployments in this range.</div>}</div>
        <div className="rail-footer">Failure count reflects the selected range.</div>
      </article>
      <article className="pulse-panel">
        <div className="panel-heading"><div><p className="section-note">Last {filters.days} day{filters.days === 1 ? "" : "s"}</p><h2>Deployment pulse</h2></div><span className="chart-legend"><i />Successful releases <b>{summary.success || 0}</b></span></div>
        <div className="chart" aria-label="Deployment trend"><div className="grid-line" /><div className="grid-line" /><div className="grid-line" /><div className="chart-bars">{(analytics?.days || []).map((day) => <span key={day.date} style={{ height: `${Math.max(4, (day.total / Math.max(1, summary.total || 1)) * 100)}%` }} title={`${day.date}: ${day.total} deployments`} />)}</div></div>
        <div className="chart-axis"><span>Earlier</span><span>Now</span></div>
      </article>
    </section>
    <dl className="health-strip"><div><dt>API</dt><dd>{health?.api || "Checking"}</dd><small>service liveness</small></div><div><dt>Database</dt><dd>{health?.database || "Checking"}</dd><small>PostgreSQL readiness</small></div><div><dt>Queue</dt><dd>{health?.queue || "Checking"}</dd><small>Redis readiness</small></div><div><dt>Recovery queue</dt><dd>{health?.dead_letter_events ?? "—"}</dd><small>dead-letter events</small></div></dl>
  </>;
}

function DeploymentsPage({ busy, deployments, filters, nextCursor, openDetail, onLoadMore, setFilters }) {
  return <>
    <section className="intro">
      <div><p className="section-note">Deployment history</p><h1>Every release, in one timeline.</h1><p className="intro-copy">Search and filter deployment activity, then inspect checks and sanitized logs.</p></div>
      <ScopeControls filters={filters} setFilters={setFilters} showStatus />
    </section>
    <section className="deployment-section">
      <div className="section-header"><div><h2>Recent releases</h2></div><p className="result-count">{deployments.length}{nextCursor ? "+" : ""} deployments</p></div>
      <div className="table-tools"><label className="search-field">Search deployments<input type="search" value={filters.q} onChange={(event) => setFilters({ ...filters, q: event.target.value })} placeholder="Repository, provider, actor…" /></label></div>
      <div className="table-frame"><table><thead><tr><th>Deployment</th><th>Environment</th><th>Status</th><th>Duration</th><th>Started</th><th>Provider</th></tr></thead><tbody>{deployments.length ? deployments.map((deployment) => <tr key={deployment.id}><td><div className="deployment-name"><button onClick={() => openDetail(deployment)}>{deployment.repository}</button><span>{deployment.branch} · {deployment.commit_sha?.slice(0, 7)} · {deployment.actor}</span></div></td><td>{deployment.environment}</td><td><Status value={deployment.status} /></td><td>{deployment.duration || "in progress"}</td><td>{ago(deployment.started_at)}</td><td>{providers[deployment.provider] || deployment.provider}</td></tr>) : <tr><td colSpan="6"><div className="empty-row">{busy ? "Loading deployment history" : "No deployments match these filters."}</div></td></tr>}</tbody></table></div>
      {nextCursor && <div className="list-actions"><button className="load-more" disabled={busy} onClick={onLoadMore}>{busy ? "Loading…" : "Load more deployments"}</button></div>}
    </section>
  </>;
}

function AlertsPage({ deadLetters, onReload, onToast, rules }) {
  const [rule, setRule] = useState({ repository: "", environment: "", channel: "slack", target: "" });
  async function submitRule(event) {
    event.preventDefault();
    try {
      await api("/api/notification-rules", { method: "POST", headers: { "Content-Type": "application/json", Origin: window.location.origin }, body: JSON.stringify({ ...rule, status: "failed" }) });
      setRule({ repository: "", environment: "", channel: "slack", target: "" });
      onToast("Notification rule created.");
      onReload();
    } catch (error) { onToast(error.message); }
  }
  return <>
    <section className="intro"><div><p className="section-note">Alert operations</p><h1>Route failures to the right place.</h1><p className="intro-copy">Create notification rules and retry provider events that exhausted automatic delivery.</p></div></section>
    <section className="admin-grid">
      <article className="admin-panel">
        <h2>Notification rules</h2><p>Send a failure alert for a repository and environment.</p>
        <form onSubmit={submitRule}><label>Repository<input required value={rule.repository} onChange={(event) => setRule({ ...rule, repository: event.target.value })} /></label><label>Environment<input required value={rule.environment} onChange={(event) => setRule({ ...rule, environment: event.target.value })} /></label><label>Channel<select value={rule.channel} onChange={(event) => setRule({ ...rule, channel: event.target.value })}><option>slack</option><option>email</option></select></label><label>Target<input required value={rule.target} onChange={(event) => setRule({ ...rule, target: event.target.value })} /></label><button className="primary-button">Create rule</button></form>
        <div className="compact-list">{rules.length ? rules.map((item) => <div className="compact-item" key={item.id}><strong>{item.repository} · {item.environment}</strong><span>{item.target}</span></div>) : <span className="admin-empty">No notification rules yet.</span>}</div>
      </article>
      <article className="admin-panel">
        <h2>Dead-letter recovery</h2><p>Events that reached five attempts and need operator action.</p>
        {deadLetters.length ? deadLetters.map((item) => <div className="dead-letter-item" key={item.id}><span>{providers[item.provider] || item.provider}</span><span className="dead-letter-error" title={item.error}>{item.error}</span><button className="reprocess-button" onClick={async () => { try { await api(`/api/dead-letter-events/${item.id}/reprocess`, { method: "POST", headers: { Origin: window.location.origin } }); onToast("Event queued for reprocessing."); onReload(); } catch (error) { onToast(error.message); } }}>Reprocess</button></div>) : <span className="admin-empty">Recovery queue is empty.</span>}
      </article>
    </section>
  </>;
}

function ConnectionsPage({ connections, onReload, onToast, webhookBaseURL, workspaceID }) {
  const [connection, setConnection] = useState({ provider: "github", name: "", secret: "" });
  async function submitConnection(event) {
    event.preventDefault();
    try {
      await api("/api/provider-connections", { method: "POST", headers: { "Content-Type": "application/json", Origin: window.location.origin }, body: JSON.stringify(connection) });
      setConnection({ ...connection, name: "", secret: "" });
      onToast("Provider connection saved.");
      onReload();
    } catch (error) { onToast(error.message); }
  }
  return <>
    <section className="intro"><div><p className="section-note">Provider setup</p><h1>Connect your release sources.</h1><p className="intro-copy">Provider secrets are encrypted before storage and are never returned by the API.</p></div></section>
    <section className="admin-grid connections-grid">
      <article className="admin-panel"><h2>New connection</h2><p>Add a provider credential to start receiving deployment events.</p><form onSubmit={submitConnection}><label>Provider<select value={connection.provider} onChange={(event) => setConnection({ ...connection, provider: event.target.value })}>{Object.keys(providers).map((key) => <option key={key} value={key}>{providers[key]}</option>)}</select></label><label>Connection name<input required value={connection.name} onChange={(event) => setConnection({ ...connection, name: event.target.value })} /></label><label className="form-span">Provider secret<input required type="password" autoComplete="new-password" value={connection.secret} onChange={(event) => setConnection({ ...connection, secret: event.target.value })} /></label><button className="primary-button">Save connection</button></form>{webhookBaseURL && workspaceID && <div className="webhook-url"><strong>Payload URL</strong><code>{webhookBaseURL}{connection.provider}/{encodeURIComponent(workspaceID)}</code><small>Set this URL and the same secret in your provider. GitHub: select the workflow_run event and application/json. A public HTTPS URL is required.</small></div>}</article>
      <article className="admin-panel"><h2>Connected providers</h2><p>Credentials currently available to this workspace.</p><div className="compact-list">{connections.length ? connections.map((item) => <div className="compact-item" key={item.id}><strong>{providers[item.provider] || item.provider}</strong><span>{item.name}</span></div>) : <span className="admin-empty">No provider connections yet.</span>}</div></article>
    </section>
  </>;
}

function DetailDialog({ deployment, onClose }) {
  const dialog = useRef(null);
  const previousFocus = useRef(document.activeElement);
  useEffect(() => {
    dialog.current?.showModal();
    return () => { dialog.current?.close(); previousFocus.current?.focus?.(); };
  }, []);
  return <dialog ref={dialog} className="deployment-dialog" onCancel={(event) => { event.preventDefault(); onClose(); }}>
    <div className="dialog-header"><div><p className="section-note">{providers[deployment.provider] || deployment.provider} deployment</p><h2>{deployment.repository}</h2></div><button className="dialog-close" aria-label="Close deployment details" onClick={onClose}>×</button></div>
    <section className="dialog-section"><div className={deployment.status === "failed" ? "failure-summary" : "success-summary"}>{deployment.status === "failed" ? deployment.failure_summary || "Review deployment details" : `This deployment is ${pretty(deployment.status)}.`}</div></section>
    <section className="dialog-section"><h3>Checks</h3>{deployment.checks?.length ? deployment.checks.map((check) => <div className="check" key={check.id}><span className={`check-mark ${check.status}`}>{check.status === "success" ? "✓" : "!"}</span><span className="check-name"><strong>{check.name}</strong><span>{check.summary}</span></span><time>{pretty(check.status)}</time></div>) : <p className="empty-logs">No checks were supplied by this provider.</p>}</section>
    <section className="dialog-section"><h3>Sanitized log excerpt</h3><div className="log-viewer">{deployment.log_lines?.length ? deployment.log_lines.map((line, index) => <div className="log-line" key={index}><span>{index + 1}</span><code>{line}</code></div>) : <p className="empty-logs">No sanitized logs available.</p>}</div></section>
  </dialog>;
}

export default function DashboardPage() {
  const location = useLocation();
  const pageTitle = pages[location.pathname];
  const main = useRef(null);
  const [collapsed, setCollapsed] = useState(false);
  const [deployments, setDeployments] = useState([]);
  const [nextCursor, setNextCursor] = useState("");
  const [analytics, setAnalytics] = useState(null);
  const [health, setHealth] = useState(null);
  const [rules, setRules] = useState([]);
  const [connections, setConnections] = useState([]);
  const [webhookSetup, setWebhookSetup] = useState({ baseURL: "", workspaceID: "" });
  const [deadLetters, setDeadLetters] = useState([]);
  const [filters, setFilters] = useState({ environment: "", status: "", q: "", days: 1 });
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [selected, setSelected] = useState(null);
  const [toast, setToast] = useState("");

  const deploymentQuery = (append = false) => {
    const query = new URLSearchParams({ limit: "50", start: new Date(Date.now() - filters.days * 86400000).toISOString() });
    if (filters.environment) query.set("environment", filters.environment);
    if (filters.status) query.set("status", filters.status);
    if (filters.q) query.set("q", filters.q);
    if (append && nextCursor) query.set("cursor", nextCursor);
    return query;
  };

  async function loadDeployments(append = false) {
    const data = await api(`/api/deployments?${deploymentQuery(append)}`);
    setDeployments((current) => append ? [...current, ...(data.items || [])] : (data.items || []));
    setNextCursor(data.next_cursor || "");
  }

  async function loadPage() {
    if (!pageTitle) return;
    setBusy(true);
    setError("");
    const requests = [api("/api/health").then(setHealth)];
    if (location.pathname === "/app") requests.push(
      loadDeployments(),
      api(`/api/analytics/deployments?days=${filters.days}${filters.environment ? `&environment=${encodeURIComponent(filters.environment)}` : ""}`).then(setAnalytics),
    );
    if (location.pathname === "/app/deployments") requests.push(loadDeployments());
    if (location.pathname === "/app/alerts") requests.push(
      api("/api/notification-rules").then((data) => setRules(data.items || [])),
      api("/api/dead-letter-events").then((data) => setDeadLetters(data.items || [])),
    );
    if (location.pathname === "/app/connections") requests.push(
      api("/api/provider-connections").then((data) => { setConnections(data.items || []); setWebhookSetup({ baseURL: data.webhook_base_url || "", workspaceID: data.workspace_id || "" }); }),
    );
    const results = await Promise.allSettled(requests);
    const failedRequest = results.find((result) => result.status === "rejected");
    if (failedRequest) setError(failedRequest.reason.message);
    setBusy(false);
  }

  useEffect(() => { loadPage(); }, [location.pathname, filters.environment, filters.status, filters.days, filters.q]);
  useEffect(() => { main.current?.focus({ preventScroll: true }); }, [location.pathname]);
  useEffect(() => { if (!toast) return; const timer = window.setTimeout(() => setToast(""), 4000); return () => window.clearTimeout(timer); }, [toast]);

  if (!pageTitle) return <Navigate to="/app" replace />;

  async function openDetail(deployment) {
    try { setSelected(await api(`/api/deployments/${encodeURIComponent(deployment.id)}`)); }
    catch (requestError) { setToast(requestError.message); }
  }

  async function loadMore() {
    setBusy(true);
    try { await loadDeployments(true); }
    catch (requestError) { setError(requestError.message); }
    finally { setBusy(false); }
  }

  const content = location.pathname === "/app"
    ? <OverviewPage analytics={analytics} deployments={deployments} filters={filters} health={health} openDetail={openDetail} setFilters={setFilters} />
    : location.pathname === "/app/deployments"
      ? <DeploymentsPage busy={busy} deployments={deployments} filters={filters} nextCursor={nextCursor} openDetail={openDetail} onLoadMore={loadMore} setFilters={setFilters} />
      : location.pathname === "/app/alerts"
        ? <AlertsPage deadLetters={deadLetters} onReload={loadPage} onToast={setToast} rules={rules} />
        : <ConnectionsPage connections={connections} onReload={loadPage} onToast={setToast} webhookBaseURL={webhookSetup.baseURL} workspaceID={webhookSetup.workspaceID} />;

  return <div className={`app-shell${collapsed ? " sidebar-collapsed" : ""}`}>
    <Sidebar collapsed={collapsed} health={health} onToggle={() => setCollapsed((value) => !value)} />
    <section className="page-shell">
      <Topbar busy={busy} pageTitle={pageTitle} onRefresh={() => loadPage().then(() => setToast(`${pageTitle} refreshed.`))} />
      <main id="main" ref={main} tabIndex="-1">
        {error && <div className="inline-error" role="alert">{error}<button onClick={loadPage}>Try again</button></div>}
        {content}
      </main>
    </section>
    {selected && <DetailDialog deployment={selected} onClose={() => setSelected(null)} />}
    {toast && <div className="toast show" role="status" aria-live="polite">{toast}</div>}
  </div>;
}
