const state = { deployments: [], filters: { environment: "", status: "", search: "", hours: 24 } };
const $ = (selector) => document.querySelector(selector);
const dialog = $("#deployment-dialog");
let toastTimer;

const escapeHTML = (value = "") => String(value).replace(/[&<>'"]/g, (character) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#039;", '"': "&quot;" }[character]));
const title = (value = "") => value.split("-").map((word) => word[0]?.toUpperCase() + word.slice(1)).join(" ");
const shortSHA = (value = "") => value.slice(0, 7);

function timeAgo(value) {
  const delta = Math.max(0, Date.now() - new Date(value).getTime());
  const minutes = Math.round(delta / 60000);
  if (minutes < 2) return "just now";
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.round(hours / 24)}d ago`;
}

function statusPill(status) {
  return `<span class="status-pill ${escapeHTML(status)}"><i></i>${escapeHTML(title(status))}</span>`;
}

function visibleDeployments() {
  const query = state.filters.search.trim().toLowerCase();
  return state.deployments.filter((deployment) => !query || [deployment.repository, deployment.branch, deployment.commit_sha, deployment.actor].join(" ").toLowerCase().includes(query));
}

function render() {
  const items = visibleDeployments();
  const failures = items.filter((deployment) => deployment.status === "failed");
  $("#failure-number").textContent = String(failures.length).padStart(2, "0");
  $("#alert-count").textContent = failures.length;
  $("#result-count").textContent = `${items.length} deployment${items.length === 1 ? "" : "s"} in selected range`;
  renderFailureRail(failures);
  renderTable(items);
}

function renderFailureRail(failures) {
  const target = $("#failure-list");
  if (!failures.length) {
    target.innerHTML = `<div class="empty-row">No failed deployments in this range.</div>`;
    return;
  }
  target.innerHTML = failures.slice(0, 3).map((deployment) => `
    <button class="incident-row" type="button" data-deployment="${escapeHTML(deployment.id)}" aria-label="Inspect failed deployment ${escapeHTML(deployment.repository)}">
      <span class="incident-time"><i></i>${timeAgo(deployment.started_at)}</span>
      <span class="incident-body">
        <span class="incident-title"><strong>${escapeHTML(deployment.repository)}</strong><code>${escapeHTML(shortSHA(deployment.commit_sha))}</code></span>
        <p>${escapeHTML(title(deployment.failure_category || "provider failure"))} · ${escapeHTML(deployment.environment)} · ${escapeHTML(deployment.failure_summary || "Review deployment details")}</p>
      </span>
      <span class="incident-cta">Inspect <svg aria-hidden="true"><use href="#i-arrow"/></svg></span>
    </button>`).join("");
}

function renderTable(items) {
  const target = $("#deployment-table");
  if (!items.length) {
    target.innerHTML = `<tr><td colspan="7"><div class="empty-row">No deployments match these filters.</div></td></tr>`;
    return;
  }
  target.innerHTML = items.map((deployment) => `
    <tr>
      <td><div class="deployment-name"><button type="button" data-deployment="${escapeHTML(deployment.id)}">${escapeHTML(deployment.repository)}</button><span>${escapeHTML(deployment.branch)} · ${escapeHTML(shortSHA(deployment.commit_sha))}</span></div></td>
      <td><span class="environment">${escapeHTML(deployment.environment)}</span></td>
      <td>${statusPill(deployment.status)}</td>
      <td><span class="duration">${escapeHTML(deployment.duration || "in progress")}</span></td>
      <td><span class="started">${escapeHTML(timeAgo(deployment.started_at))}</span></td>
      <td><span class="provider">${escapeHTML(providerName(deployment.provider))}</span></td>
      <td><button class="open-button" type="button" data-deployment="${escapeHTML(deployment.id)}" aria-label="Open ${escapeHTML(deployment.repository)}"><svg aria-hidden="true"><use href="#i-chevron"/></svg></button></td>
    </tr>`).join("");
}

function providerName(value) {
  return ({ github: "GitHub Actions", gitlab: "GitLab CI", circleci: "CircleCI", vercel: "Vercel", netlify: "Netlify", "aws-codepipeline": "AWS CodePipeline" })[value] || value;
}

async function loadDeployments() {
  const params = new URLSearchParams({ limit: "50" });
  if (state.filters.environment) params.set("environment", state.filters.environment);
  if (state.filters.status) params.set("status", state.filters.status);
  if (state.filters.hours) params.set("start", new Date(Date.now() - state.filters.hours * 3600000).toISOString());
  try {
    const response = await fetch(`/api/deployments?${params}`);
    if (!response.ok) throw new Error("The deployment feed is unavailable.");
    const data = await response.json();
    state.deployments = data.items || [];
    $("#sync-status").textContent = "Updated now";
    render();
  } catch (error) {
    $("#sync-status").textContent = "Feed unavailable";
    $("#failure-list").innerHTML = `<div class="error-row">${escapeHTML(error.message)}</div>`;
    $("#deployment-table").innerHTML = `<tr><td colspan="7"><div class="error-row">${escapeHTML(error.message)}</div></td></tr>`;
    $("#result-count").textContent = "No data loaded";
  }
}

function providerBadge(provider) {
  return ({ github: "GH", gitlab: "GL", circleci: "CI", vercel: "V", netlify: "N", "aws-codepipeline": "AWS" })[provider] || provider.slice(0, 3).toUpperCase();
}

function renderProviderHealth(providers, connected) {
  const target = $("#provider-health-list");
  const active = new Set(connected);
  target.innerHTML = providers.map((provider) => `<li><span class="provider-badge ${escapeHTML(provider)}">${escapeHTML(providerBadge(provider))}</span><span>${escapeHTML(providerName(provider))}</span><small><i class="status-dot ${active.has(provider) ? "success" : "muted"}"></i>${active.has(provider) ? "Configured" : "Not configured"}</small></li>`).join("");
}

async function loadHealth() {
  try {
    const response = await fetch("/api/health");
    const data = await response.json();
    $("#api-health").textContent = data.api || "unavailable";
    $("#database-health").textContent = data.database || "unavailable";
    $("#queue-health").textContent = data.queue || "unavailable";
    $("#dlq-count").textContent = String(data.dead_letter_events ?? "—");
    $("#system-status").textContent = data.status === "ok" ? "All systems reporting" : "System attention needed";
    $("#system-status-dot").className = `status-dot ${data.status === "ok" ? "success" : "muted"}`;
    renderProviderHealth(data.providers || [], data.connected_providers || []);
    if (!response.ok) throw new Error("One or more runtime dependencies are unavailable.");
  } catch (error) {
    $("#system-status").textContent = "Runtime health unavailable";
    $("#system-status-dot").className = "status-dot muted";
    $("#runtime-note").textContent = error.message;
  }
}

async function openDeployment(id) {
  try {
    const response = await fetch(`/api/deployments/${encodeURIComponent(id)}`);
    if (!response.ok) throw new Error("Deployment details are unavailable.");
    const deployment = await response.json();
    dialog.innerHTML = detailMarkup(deployment);
    dialog.showModal();
    dialog.querySelector(".dialog-close").focus();
  } catch (error) { showToast(error.message); }
}

function detailMarkup(deployment) {
  const checks = deployment.checks?.length ? deployment.checks.map((check) => `
    <div class="check"><span class="check-mark ${escapeHTML(check.status)}">${check.status === "success" ? '<svg aria-hidden="true"><use href="#i-check"/></svg>' : check.status === "failed" ? '<svg aria-hidden="true"><use href="#i-x"/></svg>' : '<svg aria-hidden="true"><use href="#i-clock"/></svg>'}</span><span class="check-name"><strong>${escapeHTML(check.name)}</strong><span>${escapeHTML(check.summary || title(check.status))}</span></span><time>${escapeHTML(title(check.status))}</time></div>`).join("") : `<p class="empty-logs">No checks were supplied by this provider.</p>`;
  const logs = deployment.log_lines?.length ? deployment.log_lines.map((line, index) => `<div class="log-line"><span>${index + 1}</span><code>${escapeHTML(line)}</code></div>`).join("") : `<p class="empty-logs">No sanitized log lines are available.</p>`;
  const providerLink = deployment.provider_url ? `<a class="provider-link" href="${escapeHTML(deployment.provider_url)}" target="_blank" rel="noreferrer">Open in ${escapeHTML(providerName(deployment.provider))}<svg aria-hidden="true"><use href="#i-external"/></svg></a>` : "";
  return `<div class="dialog-shell"><section class="dialog-main"><header class="dialog-header"><div><p class="section-note">${escapeHTML(providerName(deployment.provider))} deployment</p><h2 id="dialog-title">${escapeHTML(deployment.repository)}</h2></div><button class="dialog-close" type="button" aria-label="Close deployment details"><svg aria-hidden="true"><use href="#i-x"/></svg></button></header><section class="dialog-section">${deployment.status === "failed" ? `<div class="failure-summary"><strong>${escapeHTML(title(deployment.failure_category || "failure"))}</strong><br>${escapeHTML(deployment.failure_summary)}</div>` : `<div class="failure-summary" style="color:#b7dec7;border-color:#305443;border-left-color:#76be98;background:#16251d">This deployment is ${escapeHTML(deployment.status)}.</div>`}</section><section class="dialog-section"><h3>Checks</h3><div class="check-list">${checks}</div></section><section class="dialog-section"><h3>Sanitized log excerpt</h3><div class="log-viewer">${logs}</div></section></section><aside class="dialog-aside"><h3>Release context</h3><dl class="dialog-meta"><div><dt>Branch</dt><dd>${escapeHTML(deployment.branch)}</dd></div><div><dt>Commit</dt><dd>${escapeHTML(deployment.commit_sha)}</dd></div><div><dt>Environment</dt><dd>${escapeHTML(deployment.environment)}</dd></div><div><dt>Actor</dt><dd>${escapeHTML(deployment.actor)}</dd></div><div><dt>Duration</dt><dd>${escapeHTML(deployment.duration || "In progress")}</dd></div><div><dt>Original provider state</dt><dd>${escapeHTML(deployment.original_status)}</dd></div></dl>${providerLink}</aside></div>`;
}

function showToast(message) {
  const toast = $("#toast");
  toast.textContent = message;
  toast.classList.add("show");
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => toast.classList.remove("show"), 3500);
}

$("#environment-filter").addEventListener("change", (event) => { state.filters.environment = event.target.value; loadDeployments(); });
$("#range-filter").addEventListener("change", (event) => { state.filters.hours = Number(event.target.value); loadDeployments(); });
$("#status-filter").addEventListener("change", (event) => { state.filters.status = event.target.value; loadDeployments(); });
$("#search-filter").addEventListener("input", (event) => { state.filters.search = event.target.value; render(); });
$("#focus-search").addEventListener("click", () => { $("#search-filter").scrollIntoView({ behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth", block: "center" }); $("#search-filter").focus({ preventScroll: true }); });
$("#user-menu").addEventListener("click", () => showToast("This dashboard does not hold administrative credentials."));
$("#reset-filters").addEventListener("click", () => { state.filters = { environment: "", status: "", search: "", hours: 24 }; $("#environment-filter").value = ""; $("#range-filter").value = "24"; $("#status-filter").value = ""; $("#search-filter").value = ""; loadDeployments(); });
document.addEventListener("click", (event) => { const trigger = event.target.closest("[data-deployment]"); if (trigger) openDeployment(trigger.dataset.deployment); if (event.target.closest(".dialog-close")) dialog.close(); });
dialog.addEventListener("click", (event) => { if (event.target === dialog) dialog.close(); });
document.querySelectorAll(".nav-link").forEach((link) => link.addEventListener("click", () => { document.querySelectorAll(".nav-link").forEach((item) => item.classList.remove("active")); link.classList.add("active"); }));

loadDeployments();
loadHealth();
