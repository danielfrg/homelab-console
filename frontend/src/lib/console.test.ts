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
