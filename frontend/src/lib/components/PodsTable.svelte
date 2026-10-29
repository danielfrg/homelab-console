<script lang="ts">
  import { onMount } from "svelte";
  import { SearchInput, SegmentedControl } from "@kenn-io/kit-ui";
  import { fetchPods, type Pod, type PodList } from "$lib/console.js";
  import LoadState from "./LoadState.svelte";
  import LogsPanel from "./LogsPanel.svelte";
  import PodRows from "./PodRows.svelte";

  let data = $state<PodList | null>(null);
  let loading = $state(false);
  let error = $state("");
  let problemsOnly = $state(false);
  let query = $state("");
  let logsPod = $state<Pod | null>(null);

  async function load(): Promise<void> {
    loading = true;
    error = "";
    try {
      data = await fetchPods(problemsOnly);
    } catch {
      error = "Couldn't load pods.";
    } finally {
      loading = false;
    }
  }

  onMount(() => { void load(); });
  const visible = $derived((data?.pods ?? []).filter((p) =>
    `${p.namespace}/${p.name} ${p.app} ${p.node}`.toLowerCase().includes(query.trim().toLowerCase())));
</script>

<div class="view-toolbar">
  <SegmentedControl ariaLabel="Filter pods" value={problemsOnly ? "problems" : "all"}
    options={[{ value: "all", label: "All" }, { value: "problems", label: "Problems" }]}
    onchange={(value) => { problemsOnly = value === "problems"; void load(); }} />
  <div class="directory-search"><SearchInput block ariaLabel="Filter pods" placeholder="Filter by name, app, or node" value={query} oninput={(v) => query = v} /></div>
</div>

<LoadState {loading} {error} stale={data?.stale} hasData={data !== null} retry={() => void load()}>
  {#if visible.length === 0}
    <div class="directory-empty"><h2>{problemsOnly ? "No problem pods" : "No pods match"}</h2></div>
  {:else}
    <PodRows pods={visible} showOwner onlogs={(pod) => logsPod = pod} />
  {/if}
  {#if logsPod}
    <LogsPanel pod={`${logsPod.namespace}/${logsPod.name}`} canPrevious={Boolean(logsPod.last_termination)} onclose={() => logsPod = null} />
  {/if}
</LoadState>
