import assert from "node:assert/strict";
import test from "node:test";
import { appHostnameParts, filterApps, groupApps, parseSnapshot, type App } from "./apps.ts";

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

test("splits hostnames after the first label without assuming a domain", () => {
  assert.deepEqual(appHostnameParts(app()), { prefix: "sample", domain: ".example.test" });
  assert.deepEqual(
    appHostnameParts(app({ url: "https://files.backup.another.example" })),
    { prefix: "files", domain: ".backup.another.example" },
  );
});

test("keeps single-label hosts and IP addresses intact", () => {
  for (const hostname of ["localhost", "192.0.2.1", "[2001:db8::1]"]) {
    assert.deepEqual(appHostnameParts(app({ url: `https://${hostname}` })), { prefix: hostname, domain: "" });
  }
});

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
