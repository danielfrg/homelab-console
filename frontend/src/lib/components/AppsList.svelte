<script lang="ts">
  import { onMount } from "svelte";
  import { SearchInput, SegmentedControl } from "@kenn-io/kit-ui";
  import { appTiles, fetchApps, filterTiles, groupTiles, type AppList } from "$lib/console.js";
  import AppTile from "./AppTile.svelte";
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

  onMount(() => { void load(); });
  const groups = $derived(groupTiles(filterTiles(appTiles(data?.apps ?? []), query)));
</script>

<div class="view-toolbar">
  <SegmentedControl ariaLabel="Filter apps" value={status}
    options={[{ value: "all", label: "All" }, { value: "problems", label: "Needs attention" }]}
    onchange={(value) => { status = value as "all" | "problems"; void load(); }} />
  <div class="directory-search"><SearchInput block ariaLabel="Search apps" placeholder="Search apps or hosts" value={query} oninput={(v) => query = v} /></div>
</div>

<LoadState {loading} {error} stale={data?.stale} hasData={data !== null} retry={() => void load()}>
  {#if groups.length === 0}
    <div class="directory-empty">
      <h2>{status === "problems" && !query.trim() ? "Every app is Synced and Healthy" : "No apps match your search"}</h2>
    </div>
  {:else}
    <div class="directory-groups">
      {#each groups as group (group.name)}
        <section aria-label={group.name}>
          {#if group.name !== "Apps"}<h2 class="directory-group-title">{group.name}</h2>{/if}
          <div class="app-grid">
            {#each group.tiles as tile (`${tile.app.name}/${tile.link?.url ?? ""}`)}<AppTile {tile} />{/each}
          </div>
        </section>
      {/each}
    </div>
  {/if}
</LoadState>
