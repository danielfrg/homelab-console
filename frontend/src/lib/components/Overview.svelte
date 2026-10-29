<script lang="ts">
  import { onMount } from "svelte";
  import { fetchOverview, relativeTime, type Overview } from "$lib/console.js";
  import LoadState from "./LoadState.svelte";
  import StatusChip from "./StatusChip.svelte";

  let data = $state<Overview | null>(null);
  let loading = $state(false);
  let error = $state("");

  async function load(): Promise<void> {
    loading = true;
    error = "";
    try {
      data = await fetchOverview();
    } catch {
      error = "Couldn't load the cluster overview.";
    } finally {
      loading = false;
    }
  }

  onMount(() => { void load(); });

  const percent = (value: number | null): string => value === null ? "—" : `${value.toFixed(0)}%`;
  const appCount = $derived(data ? (data.app_totals.synced ?? 0) + (data.app_totals.out_of_sync ?? 0) : 0);
  const problemHref = (p: Overview["problems"][number]): string =>
    p.app ? `#/apps/${encodeURIComponent(p.app)}` : p.kind === "pod" ? "#/pods" : "#/overview";
</script>

<LoadState {loading} {error} stale={data?.stale} hasData={data !== null} retry={() => void load()}>
  {#if data}
    <section class="panel-section" aria-label="Nodes">
      <h2 class="section-title">Nodes</h2>
      <div class="node-grid">
        {#each data.nodes as node (node.name)}
          <article class="node-card">
            <header><span class="mono">{node.name}</span><StatusChip status={node.ready ? "Healthy" : "Degraded"} /></header>
            <dl>
              <div><dt>CPU</dt><dd>{percent(node.cpu_percent)}</dd></div>
              <div><dt>Memory</dt><dd>{percent(node.memory_percent)}</dd></div>
              <div><dt>Role</dt><dd>{node.role || "—"}</dd></div>
            </dl>
            <p class="muted mono small">{node.kernel_version}</p>
            {#if node.pressure.length}<p class="directory-error small">{node.pressure.join(", ")}</p>{/if}
          </article>
        {/each}
      </div>
      {#if !data.metrics_available}<p class="muted small">Usage unavailable: metrics-server did not answer.</p>{/if}
    </section>

    <section class="panel-section" aria-label="Problems">
      <h2 class="section-title">Problems</h2>
      {#if data.problems.length === 0}
        <p class="muted">No problems. {appCount} apps are tracked{data.argo_available ? "" : " (Argo CD unavailable)"}.</p>
      {:else}
        <ul class="row-list">
          {#each data.problems as p (`${p.kind}/${p.namespace}/${p.name}/${p.reason}`)}
            <li><a href={problemHref(p)}><span class="kind">{p.kind}</span><span class="mono">{p.namespace ? `${p.namespace}/` : ""}{p.name}</span><span class="reason">{p.reason}</span>{#if p.app}<span class="muted">{p.app}</span>{/if}</a></li>
          {/each}
        </ul>
        {#if data.problem_total > data.problems.length}<p class="muted small">Showing {data.problems.length} of {data.problem_total}.</p>{/if}
      {/if}
    </section>

    {#if data.needs_attention.length}
      <section class="panel-section" aria-label="Apps needing attention">
        <h2 class="section-title">Apps needing attention</h2>
        <ul class="row-list">
          {#each data.needs_attention as app (app.name)}
            <li><a href={`#/apps/${encodeURIComponent(app.name)}`}><span class="mono">{app.name}</span><StatusChip status={app.sync} /><StatusChip status={app.health} /><span class="muted">{app.ready} ready</span></a></li>
          {/each}
        </ul>
      </section>
    {/if}

    <section class="panel-section" aria-label="Recent restarts">
      <h2 class="section-title">Restarts in the last 24 hours</h2>
      {#if data.recent_restarts.length === 0}
        <p class="muted">None.</p>
      {:else}
        <ul class="row-list">
          {#each data.recent_restarts as r (`${r.namespace}/${r.name}`)}
            <li><a href={r.app ? `#/apps/${encodeURIComponent(r.app)}` : "#/pods"}><span class="mono">{r.namespace}/{r.name}</span><span class="muted">{r.app}</span><span class="muted">{relativeTime(r.restarted_at)}</span></a></li>
          {/each}
        </ul>
      {/if}
    </section>
    <p class="muted small">Updated {relativeTime(data.updated_at)}. Kubernetes status can look healthy while an app is broken; check its logs.</p>
  {/if}
</LoadState>
