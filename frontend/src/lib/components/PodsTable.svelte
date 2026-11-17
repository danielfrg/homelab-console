<script lang="ts">
  import { onMount } from "svelte";
  import {
    SegmentedControl,
    SegmentedControlIndicator,
    SegmentedControlItem,
  } from "@ui-sdk/svelte/segmented-control";
  import { Input, InputClear, InputGroup } from "@ui-sdk/svelte/input";
  import { SearchIcon, XIcon } from "@lucide/svelte";
  import { fetchPods, type Pod, type PodList } from "$lib/console.js";
  import EmptyState from "./EmptyState.svelte";
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

  onMount(() => {
    void load();
  });
  const visible = $derived(
    (data?.pods ?? []).filter((p) =>
      `${p.namespace}/${p.name} ${p.app} ${p.node}`
        .toLowerCase()
        .includes(query.trim().toLowerCase()),
    ),
  );
</script>

<div class="view-toolbar">
  <SegmentedControl
    aria-label="Filter pods"
    value={problemsOnly ? "problems" : "all"}
    onValueChange={(value) => {
      if (!value) return;
      problemsOnly = value === "problems";
      void load();
    }}
  >
    <SegmentedControlIndicator />
    <SegmentedControlItem value="all">All</SegmentedControlItem>
    <SegmentedControlItem value="problems">Problems</SegmentedControlItem>
  </SegmentedControl>
  <InputGroup class="search">
    <SearchIcon aria-hidden="true" />
    <Input
      type="search"
      bind:value={query}
      aria-label="Filter pods"
      placeholder="Filter by name, app, or node"
      autocomplete="off"
    />
    <InputClear aria-label="Clear filter"><XIcon aria-hidden="true" /></InputClear>
  </InputGroup>
</div>

<LoadState {loading} {error} stale={data?.stale} hasData={data !== null} retry={() => void load()}>
  {#if visible.length === 0}
    <EmptyState title={problemsOnly ? "No problem pods" : "No pods match"} />
  {:else}
    <PodRows pods={visible} showOwner onlogs={(pod) => (logsPod = pod)} />
  {/if}
  {#if logsPod}
    <LogsPanel
      pod={`${logsPod.namespace}/${logsPod.name}`}
      canPrevious={Boolean(logsPod.last_termination)}
      onclose={() => (logsPod = null)}
    />
  {/if}
</LoadState>
