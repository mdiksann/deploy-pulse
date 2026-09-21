import "./styles/dashboard.css";
import "./styles/auth.css";
import "./styles/landing.css";
import "./styles/tailwind.css";
import "./styles/ascii-theme.css";
import React, { useState } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, Link, Navigate, Route, Routes, useNavigate } from "react-router-dom";
import { api } from "./api";
import { AuthProvider, useAuth } from "./auth";
import { DeployPulseLogo } from "./components/ui";
import DemoOne from "./components/ui/demo";
import SonarGridDemo from "./components/ui/sonar-grid-demo";
import DashboardPage from "./pages/DashboardPage";
import { LandingPage } from "./pages/LandingPage";
import ProfilePage from "./pages/ProfilePage";

function Field({ label, error, ...props }) {
  const id = props.id || props.name;
  return <label className="auth-field" htmlFor={id}>
    <span>{label}</span>
    <input id={id} aria-invalid={Boolean(error)} aria-describedby={error ? `${id}-error` : undefined} {...props} />
    {error && <small id={`${id}-error`} className="field-error">{error}</small>}
  </label>;
}

function AuthLayout({ eyebrow, title, copy, children }) {
  return <main className="auth-shell">
    <aside className="auth-visual">
      <Link className="auth-console-brand" to="/"><DeployPulseLogo /></Link>
      <div className="auth-visual-copy"><h2>A clearer view of what shipped.</h2><p>Follow your deployments, investigate failed checks, and keep your team informed.</p></div>
      <p className="auth-visual-status">GitHub Actions, GitLab CI, CircleCI, Vercel, Netlify, and AWS CodePipeline.</p>
    </aside>
    <section className="auth-card"><Link className="auth-back" to="/">Back to Deploy Pulse</Link><p className="section-note">{eyebrow}</p><h1>{title}</h1><p className="auth-copy">{copy}</p>{children}</section>
  </main>;
}

function useFormState() {
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  return { error, setError, busy, setBusy };
}

function LoginPage() {
  const { user, setUser } = useAuth();
  const navigate = useNavigate();
  const form = useFormState();
  const [fields, setFields] = useState({ email: "", password: "" });
  if (user) return <Navigate to="/app" replace />;

  async function submit(event) {
    event.preventDefault();
    form.setError("");
    if (!fields.email || !fields.password) return form.setError("Enter your email and password.");
    form.setBusy(true);
    try {
      const data = await api("/api/auth/login", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(fields) });
      setUser(data.user);
      navigate("/app", { replace: true });
    } catch (error) { form.setError(error.message); }
    finally { form.setBusy(false); }
  }

  return <AuthLayout eyebrow="Welcome back" title="Sign in to release watch." copy="Track deployment health and operate your workspace from one quiet control room.">
    <form className="auth-form" onSubmit={submit} noValidate>
      <div className="form-error" role="alert" aria-live="polite">{form.error}</div>
      <Field label="Email" type="email" name="email" autoComplete="email" value={fields.email} onChange={(event) => setFields({ ...fields, email: event.target.value })} />
      <Field label="Password" type="password" name="password" autoComplete="current-password" value={fields.password} onChange={(event) => setFields({ ...fields, password: event.target.value })} />
      <button className="primary-button auth-submit" disabled={form.busy}>{form.busy ? "Signing in…" : "Sign in"}</button>
    </form>
    <p className="auth-footer">Need an account? <Link to="/signup">Create one</Link></p>
  </AuthLayout>;
}

function SignupPage() {
  const { user } = useAuth();
  const navigate = useNavigate();
  const form = useFormState();
  const [fields, setFields] = useState({ email: "", password: "", confirm: "" });
  const [created, setCreated] = useState(false);
  if (user) return <Navigate to="/app" replace />;

  async function submit(event) {
    event.preventDefault();
    form.setError("");
    if (!fields.email || fields.password.length < 8) return form.setError("Use a valid email and a password of at least 8 characters.");
    if (fields.password !== fields.confirm) return form.setError("Passwords do not match.");
    form.setBusy(true);
    try {
      await api("/api/auth/signup", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ email: fields.email, password: fields.password }) });
      setCreated(true);
    } catch (error) { form.setError(error.message); }
    finally { form.setBusy(false); }
  }

  if (created) return <AuthLayout eyebrow="Account ready" title="You can sign in now." copy="Your Deploy Pulse workspace is ready. No email verification is required."><button className="secondary-button auth-submit" onClick={() => navigate("/login")}>Continue to sign in</button></AuthLayout>;

  return <AuthLayout eyebrow="Create account" title="Keep every release in view." copy="Create an account and start operating your deployment workspace immediately.">
    <form className="auth-form" onSubmit={submit} noValidate>
      <div className="form-error" role="alert">{form.error}</div>
      <Field label="Email" type="email" name="email" autoComplete="email" value={fields.email} onChange={(event) => setFields({ ...fields, email: event.target.value })} />
      <Field label="Password" type="password" name="password" autoComplete="new-password" minLength="8" value={fields.password} onChange={(event) => setFields({ ...fields, password: event.target.value })} />
      <Field label="Confirm password" type="password" name="confirm" autoComplete="new-password" value={fields.confirm} onChange={(event) => setFields({ ...fields, confirm: event.target.value })} />
      <button className="primary-button auth-submit" disabled={form.busy}>{form.busy ? "Creating account…" : "Create account"}</button>
    </form>
    <p className="auth-footer">Already have an account? <Link to="/login">Sign in</Link></p>
  </AuthLayout>;
}

function Guard({ children }) {
  const { user, loading } = useAuth();
  if (loading) return <main className="auth-shell"><div className="auth-card">Loading session…</div></main>;
  return user ? children : <Navigate to="/login" replace />;
}

export function App() {
  return <BrowserRouter><AuthProvider><Routes>
    <Route path="/" element={<LandingPage />} />
    <Route path="/hero-ascii" element={<DemoOne />} />
    <Route path="/sonar-grid" element={<SonarGridDemo />} />
    <Route path="/login" element={<LoginPage />} />
    <Route path="/signup" element={<SignupPage />} />
    <Route path="/app/profile" element={<Guard><ProfilePage /></Guard>} />
    <Route path="/app/*" element={<Guard><DashboardPage /></Guard>} />
    <Route path="*" element={<Navigate to="/" replace />} />
  </Routes></AuthProvider></BrowserRouter>;
}

if (document.getElementById("root")) createRoot(document.getElementById("root")).render(<App />);
