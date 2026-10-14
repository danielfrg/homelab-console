<script lang="ts">
  import { onMount } from "svelte";
  import { fetchApps, filterApps, groupApps, type AppSnapshot } from "$lib/apps.js";
  import AppCard from "$lib/components/AppCard.svelte";
  import { Button } from "$lib/components/ui/button/index.js";
  import { Input } from "$lib/components/ui/input/index.js";

  let snapshot = $state<AppSnapshot | null>(null);
  let query = $state("");
  let loading = $state(false);
  let error = $state("");
  let controller: AbortController | null = null;

  const visibleApps = $derived(filterApps(snapshot?.apps ?? [], query));
  const groups = $derived(groupApps(visibleApps));
  const total = $derived(snapshot?.apps.length ?? 0);

  async function loadApps(refresh = false): Promise<void> {
    if (loading) return;
    loading = true;
    error = "";
    const request = new AbortController();
    controller = request;

    try {
      snapshot = await fetchApps(refresh, request.signal);
    } catch {
      if (request.signal.aborted) return;
      error = snapshot
        ? "Couldn't update apps. Showing the last successful list; it may be stale."
        : "Couldn't load apps. Try again.";
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    void loadApps();
    return () => controller?.abort();
  });
</script>

<main class="poster-launcher">
  <header class="directory-header">
    <h1>Homelab</h1>
    <div class="directory-controls">
      <label for="app-search" class="sr-only">Search apps</label>
      <div class="directory-search">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
          <circle cx="11" cy="11" r="8" />
          <path d="m21 21-4.3-4.3" />
        </svg>
        <Input
          id="app-search"
          type="search"
          bind:value={query}
          placeholder="Search apps or hosts"
          autocomplete="off"
          aria-controls="app-directory"
        />
      </div>
    </div>
  </header>

  {#if error}
    <div role="alert" class="directory-notice directory-error">
      <p>{error}</p>
      <Button variant="outline" disabled={loading} onclick={() => void loadApps(true)}>
        {loading ? "Retrying..." : "Retry"}
      </Button>
    </div>
  {:else if snapshot?.stale}
    <div role="status" class="directory-notice">
      <p>The app list may be stale. Showing the last successful discovery.</p>
      <Button variant="outline" disabled={loading} onclick={() => void loadApps(true)}>
        {loading ? "Retrying..." : "Retry"}
      </Button>
    </div>
  {/if}

  <div id="app-directory" class="directory-content" aria-busy={loading}>
    {#if loading && !snapshot}
      <p role="status" class="directory-empty">Loading apps...</p>
    {:else if snapshot && total === 0}
      <div class="directory-empty">
        <h2>No apps found</h2>
        <p>No web apps are available yet.</p>
      </div>
    {:else if snapshot && visibleApps.length === 0}
      <div class="directory-empty">
        <h2>No apps match your search</h2>
        <p>Try a name, hostname, description or group.</p>
      </div>
    {:else if snapshot}
      <div class="directory-groups">
        {#each groups as group (group.name)}
          <section aria-label={group.name}>
            {#if group.name !== "Apps"}
              <h2 class="directory-group-title">{group.name}</h2>
            {/if}
            <div class="poster-grid">
              {#each group.apps as app (app.id)}
                <AppCard {app} />
              {/each}
            </div>
          </section>
        {/each}
      </div>
    {/if}
  </div>
</main>
