import assert from "node:assert/strict";
import test from "node:test";
import { formatRelativeTime, formatTimestamp } from "./format.ts";

const now = Date.parse("2026-03-10T12:00:00Z");
const ago = (seconds: number) => new Date(now - seconds * 1000).toISOString();

test("formatRelativeTime counts seconds, minutes, hours and days", () => {
  assert.equal(formatRelativeTime(ago(5), now), "5s ago");
  assert.equal(formatRelativeTime(ago(5 * 60), now), "5m ago");
  assert.equal(formatRelativeTime(ago(3 * 3600), now), "3h ago");
  assert.equal(formatRelativeTime(ago(2 * 86400), now), "2d ago");
});

test("formatRelativeTime never goes negative and shows a dash when empty", () => {
  assert.equal(formatRelativeTime(ago(-30), now), "0s ago");
  assert.equal(formatRelativeTime("", now), "—");
  assert.equal(formatRelativeTime(null, now), "—");
});

test("formatTimestamp handles empty values", () => {
  assert.equal(formatTimestamp(undefined), "—");
  assert.match(formatTimestamp("2026-03-10T12:00:00Z"), /\d/);
});
