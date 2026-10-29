export interface ClusterNode {
  name: string;
  role: string;
  kubelet_version: string;
  kernel_version: string;
  ready: boolean;
  pressure: string[];
  cpu_percent: number | null;
  memory_percent: number | null;
}

export interface AppSummary {
  name: string;
  sync: string;
  health: string;
  ready: string;
  link: string;
  deployed_at: string | null;
}

export interface Problem {
  kind: "node" | "workload" | "pod";
  namespace?: string;
  name: string;
  app?: string;
  reason: string;
}

export interface RecentRestart {
  app: string;
  namespace: string;
  name: string;
  restarted_at: string;
}

export interface Overview {
  updated_at: string;
  stale: boolean;
  argo_available: boolean;
  metrics_available: boolean;
  nodes: ClusterNode[];
  app_totals: Record<string, number>;
  needs_attention: AppSummary[];
  problems: Problem[];
  problem_total: number;
  recent_restarts: RecentRestart[];
}

export interface Termination {
  reason: string;
  exit_code: number;
  finished_at: string;
}

export interface Pod {
  namespace: string;
  name: string;
  app: string;
  workload: string;
  phase: string;
  node: string;
  waiting?: string;
  problem?: string;
  ready: boolean;
  restarts: number;
  last_termination?: Termination;
  started_at: string | null;
  images: string[];
  containers: string[];
}

export interface Workload {
  app: string;
  kind: string;
  namespace: string;
  name: string;
  strategy: string;
  rollout: string;
  desired: number;
  ready: number;
  updated: number;
  restarted_at: string | null;
  restartable: boolean;
  not_restartable_reason?: string;
  pods: Pod[];
}

export interface Link {
  name: string;
  url: string;
  aliases: string[];
}

export interface ClusterEvent {
  namespace: string;
  object: string;
  reason: string;
  message: string;
  count: number;
  last_seen: string;
}

export interface AppDetail {
  updated_at: string;
  stale: boolean;
  matched_workload: string;
  app: {
    name: string;
    project: string;
    namespace: string;
    sync: string;
    health: string;
    operation_phase: string;
    operation_message: string;
    revision: string;
    deployed_at: string | null;
    images: string[];
    source: string;
    links: Link[];
  };
  workloads: Workload[];
  events: ClusterEvent[];
}

export interface AppList {
  apps: AppSummary[];
  updated_at: string;
  stale: boolean;
}

export interface PodList {
  pods: Pod[];
  updated_at: string;
  stale: boolean;
}

export interface LogsResult {
  pod: string;
  container: string;
  other_pods: string[];
  text: string;
  truncated: boolean;
}

export interface RestartResult {
  restarted: { kind: string; namespace: string; name: string; restarted_at: string }[];
  skipped?: string[];
  message: string;
}

export type Tone = "success" | "warning" | "danger" | "info" | "neutral";

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function hasArrays(value: unknown, keys: string[]): value is Record<string, unknown> {
  return isRecord(value) && keys.every((key) => Array.isArray(value[key]));
}

export function parseOverview(value: unknown): Overview {
  if (!hasArrays(value, ["nodes", "needs_attention", "problems", "recent_restarts"]) || !isRecord(value.app_totals)) {
    throw new Error("Invalid overview response");
  }
  return value as unknown as Overview;
}

export function parseAppDetail(value: unknown): AppDetail {
  if (!hasArrays(value, ["workloads", "events"]) || !isRecord(value.app) || typeof value.app.name !== "string") {
    throw new Error("Invalid app response");
  }
  return value as unknown as AppDetail;
}

function parseList<T>(key: string) {
  return (value: unknown): T => {
    if (!hasArrays(value, [key])) throw new Error(`Invalid ${key} response`);
    return value as unknown as T;
  };
}

async function errorMessage(response: Response): Promise<string> {
  try {
    const body: unknown = await response.json();
    if (isRecord(body) && typeof body.detail === "string") return body.detail;
  } catch {
    // Fall through to the status code.
  }
  return `HTTP ${response.status}`;
}

async function getJSON<T>(url: string, parse: (value: unknown) => T, signal?: AbortSignal): Promise<T> {
  const response = await fetch(url, { signal });
  if (!response.ok) throw new Error(await errorMessage(response));
  return parse(await response.json());
}

export function fetchConfig(signal?: AbortSignal): Promise<{ restart_enabled: boolean }> {
  return getJSON("/api/config", (value) => {
    if (!isRecord(value) || typeof value.restart_enabled !== "boolean") throw new Error("Invalid config response");
    return { restart_enabled: value.restart_enabled };
  }, signal);
}

export function fetchOverview(signal?: AbortSignal): Promise<Overview> {
  return getJSON("/api/overview", parseOverview, signal);
}

export function fetchApps(status: "all" | "problems" = "all", signal?: AbortSignal): Promise<AppList> {
  return getJSON(`/api/apps?status=${status}`, parseList<AppList>("apps"), signal);
}

export function fetchApp(name: string, signal?: AbortSignal): Promise<AppDetail> {
  return getJSON(`/api/apps/${encodeURIComponent(name)}`, parseAppDetail, signal);
}

export function fetchPods(problemsOnly: boolean, signal?: AbortSignal): Promise<PodList> {
  return getJSON(`/api/pods?problems_only=${problemsOnly}`, parseList<PodList>("pods"), signal);
}

export function fetchLogs(pod: string, previous: boolean, lines = 200, signal?: AbortSignal): Promise<LogsResult> {
  const query = new URLSearchParams({ pod, previous: String(previous), lines: String(lines) });
  return getJSON(`/api/logs?${query}`, (value) => {
    if (!isRecord(value) || typeof value.text !== "string") throw new Error("Invalid logs response");
    return value as unknown as LogsResult;
  }, signal);
}

export async function restartWorkload(workload: string, reason = ""): Promise<RestartResult> {
  const response = await fetch("/api/restart", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ workload, reason }),
  });
  if (!response.ok) throw new Error(await errorMessage(response));
  const body: unknown = await response.json();
  if (!hasArrays(body, ["restarted"])) throw new Error("Invalid restart response");
  return body as unknown as RestartResult;
}

export function statusTone(status: string): Tone {
  switch (status) {
    case "Healthy":
    case "Synced":
    case "complete":
      return "success";
    case "Degraded":
    case "Missing":
    case "stuck":
      return "danger";
    case "OutOfSync":
    case "Suspended":
      return "warning";
    case "Progressing":
    case "progressing":
      return "info";
    default:
      return "neutral";
  }
}

export function relativeTime(value: string | null | undefined, now = Date.now()): string {
  if (!value) return "—";
  const seconds = Math.max(0, Math.round((now - Date.parse(value)) / 1000));
  if (seconds < 60) return `${seconds}s ago`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.round(seconds / 3600)}h ago`;
  return `${Math.round(seconds / 86400)}d ago`;
}
