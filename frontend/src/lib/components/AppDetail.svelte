<script lang="ts">
  import { fetchApp, fetchConfig, relativeTime, type AppDetail, type Pod } from "$lib/console.js";
  import LoadState from "./LoadState.svelte";
  import LogsPanel from "./LogsPanel.svelte";
  import PodRows from "./PodRows.svelte";
  import RestartButton from "./RestartButton.svelte";
  import StatusChip from "./StatusChip.svelte";

  let { name }: { name: string } = $props();
  let data = $state<AppDetail | null>(null);
  let loading = $state(false);
  let error = $state("");
  let restartEnabled = $state(false);
  let logsPod = $state<Pod | null>(null);

  async function load(): Promise<void> {
    loading = true;
    error = "";
    try {
      [data, restartEnabled] = await Promise.all([fetchApp(name), fetchConfig().then((c) => c.restart_enabled)]);
    } catch (err) {
      error = err instanceof Error && err.message === "Not found" ? `No app or workload named ${name}.` : "Couldn't load this app.";
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    void name;
    data = null;
    logsPod = null;
    void load();
  });
</script>

<p class="breadcrumb"><a href="#/apps">Apps</a> /</p>
<LoadState {loading} {error} stale={data?.stale} hasData={data !== null} retry={() => void load()}>
  {#if data}
    <header class="app-header">
      <h2 class="mono">{data.app.name}</h2>
      <StatusChip status={data.app.sync} />
      <StatusChip status={data.app.health} />
      {#if data.matched_workload}<span class="muted">matched workload <span class="mono">{data.matched_workload}</span></span>{/if}
    </header>
    <dl class="facts">
      <div><dt>Last deploy</dt><dd>{relativeTime(data.app.deployed_at)}{#if data.app.revision} · <span class="mono">{data.app.revision.slice(0, 7)}</span>{/if}</dd></div>
      <div><dt>Last sync</dt><dd>{data.app.operation_phase || "—"}{data.app.operation_message ? ` · ${data.app.operation_message}` : ""}</dd></div>
      <div><dt>Source</dt><dd class="mono">{data.app.source || "—"}</dd></div>
      {#if data.app.links.length}
        <div><dt>Links</dt><dd class="link-list">{#each data.app.links as link (link.url)}<a href={link.url} target="_blank" rel="noreferrer">{new URL(link.url).hostname}</a>{/each}</dd></div>
      {/if}
    </dl>

    {#each data.workloads as w (`${w.namespace}/${w.name}`)}
      <section class="panel-section workload" class:highlight={w.name === data.matched_workload} aria-label={`${w.kind} ${w.name}`}>
        <header class="workload-header">
          <h3><span class="kind">{w.kind}</span> <span class="mono">{w.namespace}/{w.name}</span></h3>
          <span class="mono">{w.ready}/{w.desired} ready</span>
          <StatusChip status={w.rollout} />
          {#if w.restarted_at}<span class="muted">restarted {relativeTime(w.restarted_at)}</span>{/if}
          {#if restartEnabled}
            <RestartButton workload={`${w.namespace}/${w.name}`} label={w.name}
              disabledReason={w.restartable ? "" : `Can't restart: ${w.not_restartable_reason}`} ondone={() => void load()} />
          {/if}
        </header>
        {#if w.pods.length}
          <PodRows pods={w.pods} onlogs={(pod) => logsPod = pod} />
        {:else}
          <p class="muted">No pods.</p>
        {/if}
      </section>
    {/each}

    {#if logsPod}
      <LogsPanel pod={`${logsPod.namespace}/${logsPod.name}`} canPrevious={Boolean(logsPod.last_termination)} onclose={() => logsPod = null} />
    {/if}

    <section class="panel-section" aria-label="Warnings">
      <h3 class="section-title">Recent warnings</h3>
      {#if data.events.length === 0}
        <p class="muted">None in the last hour.</p>
      {:else}
        <ul class="row-list">
          {#each data.events as e, i (i)}
            <li><span class="mono">{e.object}</span><span class="reason">{e.reason}</span><span>{e.message}</span><span class="muted">{e.count > 1 ? `×${e.count} ` : ""}{relativeTime(e.last_seen)}</span></li>
          {/each}
        </ul>
      {/if}
    </section>
  {/if}
</LoadState>
