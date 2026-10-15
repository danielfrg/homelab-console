import assert from "node:assert/strict";
import test from "node:test";
import { filterApps, groupApps, parseSnapshot, type App } from "../src/lib/apps.ts";

function app(overrides: Partial<App> = {}): App {
  return {
    id: "default/sample",
    name: "Sample",
    url: "https://sample.example.test",
    aliases: [],
    description: "",
    group: "Apps",
    icon: "",
    ...overrides,
  };
}

test("accepts hostnames from different domains and searches their aliases", () => {
  const snapshot = parseSnapshot({
    apps: [app({ url: "https://files.example.org", aliases: ["documents.example.test"] })],
    updatedAt: "2026-01-01T00:00:00Z",
    stale: false,
  });
  assert.equal(snapshot.apps[0].url, "https://files.example.org");
  assert.equal(filterApps(snapshot.apps, "files.example.org").length, 1);
  assert.equal(filterApps(snapshot.apps, "documents.example.test").length, 1);
  assert.deepEqual(filterApps(snapshot.apps, "missing.example.test"), []);
});

test("places Favorites before default Apps and sorts other metadata groups", () => {
  const grouped = groupApps([
    app({ id: "default/zeta", group: "Zeta" }),
    app({ id: "default/infra", group: "Infrastructure" }),
    app({ id: "default/apps", group: "Apps" }),
    app({ id: "default/favorite", group: "Favorites" }),
    app({ id: "default/alpha", group: "Alpha" }),
  ]);

  assert.deepEqual(grouped.map((group) => group.name), [
    "Favorites", "Apps", "Alpha", "Infrastructure", "Zeta",
  ]);
  assert.deepEqual(grouped[0].apps.map((app) => app.id), ["default/favorite"]);
});

test("does not create a Favorites shelf when no route belongs to it", () => {
  assert.deepEqual(
    groupApps([app()]).map((group) => group.name),
    ["Apps"],
  );
  assert.deepEqual(groupApps([]), []);
});
