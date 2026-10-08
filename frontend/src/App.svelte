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
        ? "Couldn't refresh apps. Showing the last successful list; it may be stale."
        : "Couldn't load apps. Use Refresh to try again.";
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    void loadApps();
    return () => controller?.abort();
  });
</script>

<main class="compact-launcher">
  <header class="directory-header">
    <h1>Homelab</h1>
    <div class="directory-controls">
      <label for="app-search" class="sr-only">Search apps</label>
      <Input
        id="app-search"
        type="search"
        bind:value={query}
        placeholder="Search apps, hosts, or groups"
        autocomplete="off"
        aria-controls="app-directory"
      />
      <Button
        variant="outline"
        class="h-11 px-4 text-sm motion-reduce:transition-none"
        disabled={loading}
        onclick={() => void loadApps(true)}
      >
        {loading && snapshot ? "Refreshing..." : "Refresh"}
      </Button>
    </div>
  </header>

  {#if error}
    <p role="alert" class="directory-notice text-destructive border-destructive/25">{error}</p>
  {:else if snapshot?.stale}
    <p role="status" class="directory-notice text-muted-foreground">
      The app list may be stale. Showing the last successful discovery; Refresh to try again.
    </p>
  {/if}

  <div id="app-directory" class="directory-content" aria-busy={loading}>
    {#if loading && !snapshot}
      <p role="status" class="directory-empty">Loading apps...</p>
    {:else if snapshot && total === 0}
      <div class="directory-empty">
        <h2>No apps found</h2>
        <p>Refresh to check for web apps.</p>
      </div>
    {:else if snapshot && visibleApps.length === 0}
      <div class="directory-empty">
        <h2>No apps match your search</h2>
        <p>Try a name, hostname, description or group.</p>
      </div>
    {:else}
      <div class="directory-groups">
        {#each groups as group (group.name)}
          <section aria-label={group.name}>
            <h2 class="directory-group-title">{group.name}</h2>
            <div class="compact-grid">
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
