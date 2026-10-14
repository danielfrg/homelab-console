import assert from "node:assert/strict";
import test from "node:test";
import { appHostnameParts, groupApps, type App } from "../src/lib/apps.ts";

function app(overrides: Partial<App> = {}): App {
  return {
    id: "default/sample",
    name: "Sample",
    url: "https://sample.example.net",
    aliases: [],
    description: "",
    group: "Apps",
    icon: "",
    ...overrides,
  };
}

test("splits the complete internal domain from a long app hostname", () => {
  assert.deepEqual(
    appHostnameParts(app({ url: "https://metadata-remote.example.net" })),
    { prefix: "metadata-remote", domain: ".example.net" },
  );
});

test("keeps nested subdomains together on the first hostname line", () => {
  assert.deepEqual(
    appHostnameParts(app({ url: "https://files.backup.example.net" })),
    { prefix: "files.backup", domain: ".example.net" },
  );
});

test("does not split a hostname that only resembles the internal suffix", () => {
  assert.deepEqual(
    appHostnameParts(app({ url: "https://example.net.example.com" })),
    { prefix: "example.net.example.com", domain: "" },
  );
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
