import React from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../auth";
import ProfileMenu from "../components/ProfileMenu";

export default function ProfilePage() {
  const { user } = useAuth();
  return <div className="profile-page">
    <header className="topbar"><Link to="/app">← Dashboard</Link><ProfileMenu /></header>
    <main className="profile-content">
      <div className="intro"><div><p className="section-note">Account</p><h1>Profile</h1><p className="intro-copy">Your account details.</p></div></div>
      <dl className="profile-details">
        <div><dt>Email</dt><dd>{user.email}</dd></div>
        {user.role && <div><dt>Role</dt><dd>{user.role}</dd></div>}
        {user.workspace_id && <div><dt>Workspace</dt><dd>{user.workspace_id}</dd></div>}
        {user.created_at && <div><dt>Member since</dt><dd>{new Date(user.created_at).toLocaleDateString(undefined, { year: "numeric", month: "long", day: "numeric" })}</dd></div>}
      </dl>
    </main>
  </div>;
}
