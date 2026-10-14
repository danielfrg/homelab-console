export interface App {
  id: string;
  name: string;
  url: string;
  aliases: string[];
  description: string;
  group: string;
  icon: string;
}

export interface AppSnapshot {
  apps: App[];
  updatedAt: string;
  stale: boolean;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function parseApp(value: unknown): App {
  if (
    !isRecord(value) ||
    typeof value.id !== "string" || !value.id.trim() ||
    typeof value.name !== "string" || !value.name.trim() ||
    typeof value.url !== "string" ||
    !Array.isArray(value.aliases) ||
    !value.aliases.every((alias): alias is string => typeof alias === "string") ||
    typeof value.description !== "string" ||
    typeof value.group !== "string" ||
    typeof value.icon !== "string"
  ) {
    throw new Error("Invalid app response");
  }

  const url = new URL(value.url);
  if (url.protocol !== "https:" || !url.hostname || url.username || url.password) {
    throw new Error("Invalid app URL");
  }

  return {
    id: value.id,
    name: value.name,
    url: value.url,
    aliases: value.aliases,
    description: value.description,
    group: value.group.trim() || "Apps",
    icon: value.icon,
  };
}

export function parseSnapshot(value: unknown): AppSnapshot {
  if (
    !isRecord(value) ||
    !Array.isArray(value.apps) ||
    typeof value.updatedAt !== "string" ||
    Number.isNaN(Date.parse(value.updatedAt)) ||
    typeof value.stale !== "boolean"
  ) {
    throw new Error("Invalid app response");
  }

  const apps = value.apps.map(parseApp);
  if (new Set(apps.map((app) => app.id)).size !== apps.length) {
    throw new Error("Duplicate app IDs");
  }
  return { apps, updatedAt: value.updatedAt, stale: value.stale };
}

export async function fetchApps(refresh = false, signal?: AbortSignal): Promise<AppSnapshot> {
  const response = await fetch(refresh ? "/api/apps?refresh=true" : "/api/apps", { signal });
  if (!response.ok) throw new Error(`HTTP ${response.status}`);
  return parseSnapshot(await response.json());
}

export function appHostname(app: App): string {
  return new URL(app.url).hostname;
}

export function appHostnameParts(app: App): { prefix: string; domain: string } {
  const hostname = appHostname(app);
  const domain = ".example.net";
  return hostname.endsWith(domain)
    ? { prefix: hostname.slice(0, -domain.length), domain }
    : { prefix: hostname, domain: "" };
}

export function filterApps(apps: App[], query: string): App[] {
  const search = query.trim().toLocaleLowerCase();
  if (!search) return apps;
  return apps.filter((app) => [app.name, app.description, app.group, appHostname(app), ...app.aliases]
    .join(" ").toLocaleLowerCase().includes(search));
}

export function groupApps(apps: App[]): { name: string; apps: App[] }[] {
  const groups = new Map<string, App[]>();
  for (const app of apps) {
    const group = groups.get(app.group) ?? [];
    group.push(app);
    groups.set(app.group, group);
  }
  const rank = (name: string): number => name === "Favorites" ? 0 : name === "Apps" ? 1 : 2;
  return [...groups].sort(([left], [right]) => rank(left) - rank(right) || left.localeCompare(right))
    .map(([name, apps]) => ({ name, apps }));
}

export function appInitials(name: string): string {
  const words = name.trim().split(/\s+/);
  return (words.length > 1 ? `${words[0][0]}${words[1][0]}` : words[0].slice(0, 2)).toLocaleUpperCase();
}

export function appIconURL(icon: string): string | null {
  if (icon.length > 80 || !/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(icon)) return null;
  if (icon.startsWith("mdi-")) {
    return `https://cdn.jsdelivr.net/npm/@mdi/svg@7.4.47/svg/${icon.slice(4)}.svg`;
  }
  return `https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/svg/${icon}.svg`;
}
