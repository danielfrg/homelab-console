<script lang="ts">
  import { onMount } from "svelte";
  import {
    SegmentedControl,
    SegmentedControlIndicator,
    SegmentedControlItem,
  } from "@ui-sdk/svelte/segmented-control";
  import { Input, InputClear, InputGroup } from "@ui-sdk/svelte/input";
  import { SearchIcon, XIcon } from "@lucide/svelte";
  import { appTiles, fetchApps, filterTiles, groupTiles, type AppList } from "$lib/console.js";
  import AppTile from "./AppTile.svelte";
  import EmptyState from "./EmptyState.svelte";
  import LoadState from "./LoadState.svelte";

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

  onMount(() => {
    void load();
  });
  const groups = $derived(groupTiles(filterTiles(appTiles(data?.apps ?? []), query)));
</script>

<div class="view-toolbar">
  <SegmentedControl
    aria-label="Filter apps"
    value={status}
    onValueChange={(value) => {
      if (!value) return;
      status = value as "all" | "problems";
      void load();
    }}
  >
    <SegmentedControlIndicator />
    <SegmentedControlItem value="all">All</SegmentedControlItem>
    <SegmentedControlItem value="problems">Needs attention</SegmentedControlItem>
  </SegmentedControl>
  <InputGroup class="search">
    <SearchIcon aria-hidden="true" />
    <Input
      type="search"
      bind:value={query}
      aria-label="Search apps"
      placeholder="Search apps or hosts"
      autocomplete="off"
    />
    <InputClear aria-label="Clear search"><XIcon aria-hidden="true" /></InputClear>
  </InputGroup>
</div>

<LoadState {loading} {error} stale={data?.stale} hasData={data !== null} retry={() => void load()}>
  {#if groups.length === 0}
    <EmptyState
      title={status === "problems" && !query.trim()
        ? "Every app is Synced and Healthy"
        : "No apps match your search"}
    />
  {:else}
    <div class="directory-groups">
      {#each groups as group (group.name)}
        <section aria-label={group.name}>
          {#if group.name !== "Apps"}<h2 class="directory-group-title">{group.name}</h2>{/if}
          <div class="app-grid">
            {#each group.tiles as tile (`${tile.app.name}/${tile.link?.url ?? ""}`)}<AppTile
                {tile}
              />{/each}
          </div>
        </section>
      {/each}
    </div>
  {/if}
</LoadState>
