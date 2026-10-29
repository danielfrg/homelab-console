import assert from "node:assert/strict";
import test from "node:test";
import { parseAppDetail, parseOverview, restartWorkload, statusTone } from "./console.ts";

const overview = {
  updated_at: "2026-10-28T12:00:00Z", stale: false, argo_available: true, metrics_available: true,
  nodes: [{ name: "srv-nuc-01", role: "compute", kubelet_version: "v1", kernel_version: "6", ready: true, pressure: [], cpu_percent: 3.5, memory_percent: null }],
  app_totals: { synced: 2, healthy: 2 }, needs_attention: [], problems: [], problem_total: 0, recent_restarts: [],
};

test("parses a valid overview", () => {
  const parsed = parseOverview(overview);
  assert.equal(parsed.nodes[0].name, "srv-nuc-01");
  assert.equal(parsed.nodes[0].memory_percent, null);
});

test("rejects an overview without problems", () => {
  const { problems: _, ...broken } = overview;
  assert.throws(() => parseOverview(broken), /Invalid overview/);
});

test("rejects an app detail without workloads", () => {
  assert.throws(() => parseAppDetail({ app: { name: "x" }, events: [] }), /Invalid app/);
});

test("posts restarts as same-origin JSON", async () => {
  const calls: { url: string; init: RequestInit }[] = [];
  const original = globalThis.fetch;
  globalThis.fetch = (async (url: string, init: RequestInit) => {
    calls.push({ url, init });
    return new Response(JSON.stringify({ restarted: [{ kind: "Deployment", namespace: "default", name: "frigate", restarted_at: "2026-10-28T12:00:00Z" }], message: "Rollout started; check get_app for progress." }), { status: 200 });
  }) as typeof fetch;
  try {
    const result = await restartWorkload("default/frigate", "frozen");
    assert.equal(result.restarted[0].name, "frigate");
    assert.equal(calls[0].url, "/api/restart");
    assert.equal(calls[0].init.method, "POST");
    assert.equal((calls[0].init.headers as Record<string, string>)["Content-Type"], "application/json");
    assert.deepEqual(JSON.parse(String(calls[0].init.body)), { workload: "default/frigate", reason: "frozen" });
  } finally {
    globalThis.fetch = original;
  }
});

test("surfaces the server's refusal message", async () => {
  const original = globalThis.fetch;
  globalThis.fetch = (async () => new Response(JSON.stringify({ detail: "vault/vault cannot be restarted" }), { status: 400 })) as typeof fetch;
  try {
    await assert.rejects(restartWorkload("vault"), /cannot be restarted/);
  } finally {
    globalThis.fetch = original;
  }
});

test("maps Argo statuses to tones", () => {
  assert.equal(statusTone("Healthy"), "success");
  assert.equal(statusTone("Synced"), "success");
  assert.equal(statusTone("Degraded"), "danger");
  assert.equal(statusTone("OutOfSync"), "warning");
  assert.equal(statusTone("Progressing"), "info");
});

function summary(overrides: Partial<import("./console.ts").AppSummary> = {}): import("./console.ts").AppSummary {
  return { name: "x", sync: "Synced", health: "Healthy", ready: "1/1", link: "", deployed_at: null, links: [], ...overrides };
}

function link(url: string, overrides: Partial<import("./console.ts").Link> = {}): import("./console.ts").Link {
  return { name: "", url, aliases: [], description: "", group: "", icon: "", ...overrides };
}

test("makes one tile per link, grouped like the launcher, unlinked apps last", async () => {
  const { appTiles, groupTiles } = await import("./console.ts");
  const tiles = appTiles([
    summary({ name: "metallb" }),
    summary({ name: "home-assistant", links: [link("https://frigate.example", { name: "Frigate" }), link("https://ha.example", { name: "Home Assistant" })] }),
    summary({ name: "plex", links: [link("https://plex.example", { group: "Media" })] }),
    summary({ name: "golinks", links: [link("https://go.example", { group: "Favorites" })] }),
  ]);
  assert.deepEqual(tiles.map((t) => [t.app.name, t.title]), [
    ["metallb", "metallb"], ["home-assistant", "Frigate"], ["home-assistant", "Home Assistant"], ["plex", "plex"], ["golinks", "golinks"],
  ]);
  assert.deepEqual(groupTiles(tiles).map((g) => [g.name, g.tiles.map((t) => t.title)]), [
    ["Favorites", ["golinks"]], ["Apps", ["Frigate", "Home Assistant"]], ["Media", ["plex"]], ["Without links", ["metallb"]],
  ]);
});

test("filters tiles by app name, title, hostname, alias, and description", async () => {
  const { appTiles, filterTiles } = await import("./console.ts");
  const tiles = appTiles([
    summary({ name: "home-assistant", links: [link("https://ha.example", { name: "Home Assistant", aliases: ["home.example"], description: "Smart home" })] }),
    summary({ name: "budget" }),
  ]);
  for (const query of ["HOME-ass", "assistant", "ha.example", "home.example", "smart"]) {
    assert.deepEqual(filterTiles(tiles, query).map((t) => t.app.name), ["home-assistant"], query);
  }
  assert.equal(filterTiles(tiles, "  ").length, 2);
});

test("rejects an app list whose apps lack links", async () => {
  const { parseAppList } = await import("./console.ts");
  assert.throws(() => parseAppList({ apps: [{ name: "budget" }], updated_at: "", stale: false }), /Invalid apps/);
  assert.equal(parseAppList({ apps: [{ name: "budget", links: [] }], updated_at: "", stale: false }).apps.length, 1);
});
