import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { App } from "./main";

function response(status, body) { return { ok: status < 400, status, headers: new Headers({ "content-type": "application/json" }), json: async () => body }; }

beforeEach(() => {
  window.history.pushState({}, "", "/login");
  vi.stubGlobal("fetch", vi.fn(async (path, options = {}) => {
    if (path === "/api/auth/me") return response(401, { error: "authentication required" });
    if (path === "/api/auth/login") return response(200, { user: { email: "ops@example.com" } });
    return response(200, {});
  }));
});

describe("authentication flow", () => {
  it("opens the account menu, shows profile details, and logs out", async () => {
    window.history.pushState({}, "", "/app");
    fetch.mockImplementation(async (path) => response(200, path === "/api/auth/me"
      ? { user: { email: "ops@example.com", role: "admin", workspace_id: "default", created_at: "2026-01-10T00:00:00Z" } }
      : {}));
    render(<App/>);
    const trigger = await screen.findByRole("button", { name: "Account menu" });
    fireEvent.click(trigger);
    expect(trigger).toHaveAttribute("aria-expanded", "true");
    expect(fetch).not.toHaveBeenCalledWith("/api/auth/logout", expect.anything());
    fireEvent.keyDown(document, { key: "Escape" });
    expect(trigger).toHaveFocus();
    expect(trigger).toHaveAttribute("aria-expanded", "false");
    fireEvent.click(trigger);
    fireEvent.pointerDown(document.body);
    expect(trigger).toHaveAttribute("aria-expanded", "false");
    fireEvent.click(trigger);
    fireEvent.click(screen.getByRole("link", { name: "Profile" }));
    expect(await screen.findByRole("heading", { name: "Profile" })).toBeInTheDocument();
    expect(window.location.pathname).toBe("/app/profile");
    expect(screen.getByText("ops@example.com")).toBeInTheDocument();
    expect(screen.getByText("admin")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Account menu" }));
    fetch.mockImplementationOnce(async () => response(500, { error: "Unable to log out" }));
    fireEvent.click(screen.getByRole("button", { name: "Log out" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Unable to log out");
    expect(window.location.pathname).toBe("/app/profile");
    fireEvent.click(screen.getByRole("button", { name: "Log out" }));
    expect(await screen.findByRole("heading", { name: /sign in to release watch/i })).toBeInTheDocument();
    expect(fetch).toHaveBeenCalledWith("/api/auth/logout", expect.objectContaining({ method: "POST" }));
  });

  it("guards the profile page for signed-out users", async () => {
    window.history.pushState({}, "", "/app/profile");
    render(<App/>);
    expect(await screen.findByRole("heading", { name: /sign in to release watch/i })).toBeInTheDocument();
  });

  it("shows login and redirects after successful login", async () => {
    render(<App/>);
    await screen.findByRole("heading", { name: /sign in to release watch/i });
    fireEvent.change(screen.getByLabelText("Email"), { target: { value: "ops@example.com" } });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "password-123" } });
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));
    await waitFor(() => expect(window.location.pathname).toBe("/app"));
  });

  it("shows the public landing page with signup and login CTAs", async () => {
    window.history.pushState({}, "", "/");
    render(<App/>);
    expect(await screen.findByRole("heading", { name: /releases in view/i })).toBeInTheDocument();
    expect(screen.getAllByRole("link", { name: /get started/i }).some((link) => link.getAttribute("href") === "/signup")).toBe(true);
    expect(screen.getAllByRole("link", { name: /^sign in$/i }).every((link) => link.getAttribute("href") === "/login")).toBe(true);
    expect(screen.getByRole("heading", { name: /less hunting/i })).toBeInTheDocument();
    expect(document.querySelector(".landing-footer")).toBeInTheDocument();
  });

  it("validates signup password confirmation", async () => {
    window.history.pushState({}, "", "/signup");
    render(<App/>);
    await screen.findByRole("heading", { name: /keep every release/i });
    fireEvent.change(screen.getByLabelText("Email"), { target: { value: "ops@example.com" } });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "password-123" } });
    fireEvent.change(screen.getByLabelText("Confirm password"), { target: { value: "different-123" } });
    fireEvent.click(screen.getByRole("button", { name: "Create account" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Passwords do not match");
  });

  it("confirms account creation without email verification", async () => {
    window.history.pushState({}, "", "/signup");
    render(<App/>);
    await screen.findByRole("heading", { name: /keep every release/i });
    fireEvent.change(screen.getByLabelText("Email"), { target: { value: "new@example.com" } });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "password-123" } });
    fireEvent.change(screen.getByLabelText("Confirm password"), { target: { value: "password-123" } });
    fireEvent.click(screen.getByRole("button", { name: "Create account" }));
    expect(await screen.findByRole("heading", { name: /you can sign in now/i })).toBeInTheDocument();
  });

  it("guards the operations console for signed-out users", async () => {
    window.history.pushState({}, "", "/app");
    render(<App/>);
    expect(await screen.findByRole("heading", { name: /sign in to release watch/i })).toBeInTheDocument();
  });
});
