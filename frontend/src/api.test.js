import { describe, expect, it, vi } from "vitest";
import { api } from "./api";

describe("api client", () => {
  it("sends cross-origin requests with credentials", async () => {
    const fetch = vi.fn(async () => ({ ok: true, status: 200, headers: new Headers({ "content-type": "application/json" }), json: async () => ({ status: "ok" }) }));
    vi.stubGlobal("fetch", fetch);
    await api("/healthz");
    expect(fetch).toHaveBeenCalledWith("/healthz", expect.objectContaining({ credentials: "include" }));
  });
});
