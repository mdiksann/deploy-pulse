import React, { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../auth";

export default function ProfileMenu() {
  const { user, logout } = useAuth();
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const container = useRef(null);
  const trigger = useRef(null);

  useEffect(() => {
    if (!open) return;
    const dismiss = (event) => {
      if (!container.current?.contains(event.target)) setOpen(false);
    };
    const escape = (event) => {
      if (event.key === "Escape") {
        setOpen(false);
        trigger.current?.focus();
      }
    };
    document.addEventListener("pointerdown", dismiss);
    document.addEventListener("keydown", escape);
    return () => {
      document.removeEventListener("pointerdown", dismiss);
      document.removeEventListener("keydown", escape);
    };
  }, [open]);

  async function signOut() {
    setBusy(true);
    setError("");
    try { await logout(); }
    catch (error) { setError(error.message); }
    finally { setBusy(false); }
  }

  return <div className="profile-menu" ref={container} onBlur={(event) => {
    if (!event.currentTarget.contains(event.relatedTarget)) setOpen(false);
  }}>
    <button ref={trigger} className="avatar" aria-label="Account menu" aria-expanded={open} aria-controls="profile-popover" onClick={() => setOpen(!open)}>{user.email.slice(0, 2).toUpperCase()}</button>
    {open && <div className="profile-popover" id="profile-popover">
      <p>{user.email}</p>
      <Link to="/app/profile" onClick={() => setOpen(false)}>Profile</Link>
      <button onClick={signOut} disabled={busy}>{busy ? "Logging out…" : "Log out"}</button>
      {error && <p className="profile-error" role="alert">{error}</p>}
    </div>}
  </div>;
}
