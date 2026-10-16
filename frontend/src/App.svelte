<script lang="ts">
  import { onMount } from "svelte";
  import { fetchApps, filterApps, groupApps, type AppSnapshot } from "$lib/apps.js";
  import AppCard from "$lib/components/AppCard.svelte";
  import { Button, SearchInput, ThemeToggle } from "@kenn-io/kit-ui";

  let snapshot = $state<AppSnapshot | null>(null);
  let query = $state("");
  let loading = $state(false);
  let error = $state("");
  let controller: AbortController | null = null;

  const visibleApps = $derived(filterApps(snapshot?.apps ?? [], query));
  const groups = $derived(groupApps(visibleApps));
  const total = $derived(snapshot?.apps.length ?? 0);

  function sizePosterGrid(node: HTMLDivElement) {
    // Include the shared one-pixel border in the 2:3 poster width.
    const maxCardWidth = 250 * 2 / 3 + 1;
    const observer = new ResizeObserver(([entry]) => {
      const columns = Math.max(2, Math.ceil(entry.contentRect.width / maxCardWidth));
      node.style.setProperty("--poster-columns", String(columns));
    });
    observer.observe(node);
    return { destroy: () => observer.disconnect() };
  }

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
      <ThemeToggle />
      <div class="directory-search">
        <SearchInput block ariaLabel="Search apps" placeholder="Search apps or hosts"
          value={query} oninput={(value) => query = value} />
      </div>
    </div>
  </header>

  {#if error}
    <div role="alert" class="directory-notice directory-error">
      <p>{error}</p>
      <Button surface="outline" disabled={loading} onclick={() => void loadApps(true)}>
        {loading ? "Retrying..." : "Retry"}
      </Button>
    </div>
  {:else if snapshot?.stale}
    <div role="status" class="directory-notice">
      <p>The app list may be stale. Showing the last successful discovery.</p>
      <Button surface="outline" disabled={loading} onclick={() => void loadApps(true)}>
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
            <div class="poster-grid" use:sizePosterGrid>
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
