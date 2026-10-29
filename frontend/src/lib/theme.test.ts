import assert from "node:assert/strict";
import test from "node:test";
import { migrateThemeKey, themeStorageKey } from "./theme.ts";

function memoryStorage(initial: Record<string, string>) {
  const data = new Map(Object.entries(initial));
  return {
    getItem: (k: string) => data.get(k) ?? null,
    setItem: (k: string, v: string) => void data.set(k, v),
    removeItem: (k: string) => void data.delete(k),
    data,
  };
}

test("carries the launcher theme choice over to the console key", () => {
  const storage = memoryStorage({ "homelab-launcher-theme": "light" });
  migrateThemeKey(storage);
  assert.equal(themeStorageKey, "homelab-console-theme");
  assert.equal(storage.getItem("homelab-console-theme"), "light");
  assert.equal(storage.getItem("homelab-launcher-theme"), null);
});

test("never overwrites an existing console choice", () => {
  const storage = memoryStorage({ "homelab-launcher-theme": "light", "homelab-console-theme": "dark" });
  migrateThemeKey(storage);
  assert.equal(storage.getItem("homelab-console-theme"), "dark");
});
