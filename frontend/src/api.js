let onUnauthorized = () => {};

const apiBaseURL = (import.meta.env.VITE_API_BASE_URL || "").replace(/\/$/, "");
export const apiURL = (path) => `${apiBaseURL}${path.startsWith("/") ? path : `/${path}`}`;

export class ApiError extends Error {
  constructor(message, status) { super(message); this.status = status; }
}

export function setUnauthorizedHandler(handler) { onUnauthorized = handler; }

export async function api(path, options = {}) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 10000);
  try {
    const response = await fetch(apiURL(path), { credentials: "include", ...options, signal: controller.signal, headers: { Accept: "application/json", ...(options.headers || {}) } });
    const type = response.headers.get("content-type") || "";
    const data = type.includes("json") ? await response.json().catch(() => { throw new ApiError("The server returned invalid JSON.", response.status); }) : {};
    if (!response.ok) {
      if (response.status === 401) onUnauthorized();
      const messages = { 401: "Please sign in again.", 403: "This action is not allowed.", 429: "Too many requests. Try again shortly.", 503: "A runtime dependency is unavailable." };
      throw new ApiError(data.error || messages[response.status] || `Request failed (${response.status}).`, response.status);
    }
    return data;
  } catch (error) {
    if (error.name === "AbortError") throw new ApiError("The request timed out. Try again.", 408);
    throw error;
  } finally { clearTimeout(timeout); }
}
