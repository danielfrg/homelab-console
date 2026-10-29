<script lang="ts">
  import { onMount } from "svelte";
  import { SearchInput, SegmentedControl } from "@kenn-io/kit-ui";
  import { fetchApps, relativeTime, type AppList } from "$lib/console.js";
  import LoadState from "./LoadState.svelte";
  import StatusChip from "./StatusChip.svelte";

  let data = $state<AppList | null>(null);
  let loading = $state(false);
  let error = $state("");
  let status = $state<"all" | "problems">("all");
  let query = $state("");

  async function load(): Promise<void> {
    loading = true;
    error = "";
    try {
      data = await fetchApps(status);
    } catch {
      error = "Couldn't load apps.";
    } finally {
      loading = false;
    }
  }

  onMount(() => { void load(); });
  const visible = $derived((data?.apps ?? []).filter((a) => a.name.toLowerCase().includes(query.trim().toLowerCase())));
</script>

<div class="view-toolbar">
  <SegmentedControl ariaLabel="Filter apps" value={status}
    options={[{ value: "all", label: "All" }, { value: "problems", label: "Needs attention" }]}
    onchange={(value) => { status = value as "all" | "problems"; void load(); }} />
  <div class="directory-search"><SearchInput block ariaLabel="Filter apps by name" placeholder="Filter apps" value={query} oninput={(v) => query = v} /></div>
</div>

<LoadState {loading} {error} stale={data?.stale} hasData={data !== null} retry={() => void load()}>
  {#if visible.length === 0}
    <div class="directory-empty"><h2>{status === "problems" ? "Every app is Synced and Healthy" : "No apps match"}</h2></div>
  {:else}
    <div class="table-wrap">
      <table class="data-table">
        <thead><tr><th>App</th><th>Sync</th><th>Health</th><th>Workloads ready</th><th>Last deploy</th><th>Link</th></tr></thead>
        <tbody>
          {#each visible as app (app.name)}
            <tr>
              <td><a class="mono" href={`#/apps/${encodeURIComponent(app.name)}`}>{app.name}</a></td>
              <td><StatusChip status={app.sync} /></td>
              <td><StatusChip status={app.health} /></td>
              <td class="mono">{app.ready}</td>
              <td class="muted">{relativeTime(app.deployed_at)}</td>
              <td>{#if app.link}<a href={app.link} target="_blank" rel="noreferrer" class="muted">{new URL(app.link).hostname}</a>{/if}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</LoadState>
