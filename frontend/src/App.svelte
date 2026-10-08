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
  let refreshFailed = $state(false);
  let controller: AbortController | null = null;

  const visibleApps = $derived(filterApps(snapshot?.apps ?? [], query));
  const groups = $derived(groupApps(visibleApps));
  const total = $derived(snapshot?.apps.length ?? 0);
  const stale = $derived(Boolean(snapshot?.stale) || refreshFailed);
  const updatedLabel = $derived(snapshot ? new Date(snapshot.updatedAt).toLocaleString(undefined, {
    month: "short", day: "numeric", hour: "numeric", minute: "2-digit",
  }) : "");

  async function loadApps(refresh = false): Promise<void> {
    if (loading) return;
    loading = true;
    error = "";
    const request = new AbortController();
    controller = request;

    try {
      snapshot = await fetchApps(refresh, request.signal);
      refreshFailed = false;
    } catch {
      if (request.signal.aborted) return;
      refreshFailed = snapshot !== null;
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

<main class="mx-auto w-full max-w-7xl px-4 py-8 sm:px-6 sm:py-10">
  <header>
    <h1 class="text-3xl font-semibold tracking-tight">Homelab</h1>
    <p class="mt-2 text-sm text-muted-foreground" role="status" aria-live="polite">
      {#if snapshot}
        {#if query.trim()}
          {visibleApps.length} of {total} {total === 1 ? "app" : "apps"}
        {:else}
          {total} {total === 1 ? "app" : "apps"}
        {/if}
      {:else}
        Your app directory
      {/if}
    </p>

    <div class="mt-6 flex items-center gap-3">
      <label for="app-search" class="sr-only">Search apps</label>
      <Input
        id="app-search"
        type="search"
        bind:value={query}
        placeholder="Search apps, hosts, or groups"
        autocomplete="off"
        aria-controls="app-directory"
        class="sm:max-w-md"
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

    {#if snapshot}
      <p class="mt-3 text-xs leading-5 text-muted-foreground">
        {#if stale}Last successful update{:else}Updated{/if}
        <time datetime={snapshot.updatedAt}>{updatedLabel}</time>
      </p>
    {/if}
  </header>

  {#if error}
    <p role="alert" class="mt-5 rounded-lg border border-destructive/25 px-4 py-3 text-sm text-destructive">{error}</p>
  {:else if snapshot?.stale}
    <p role="status" class="mt-5 rounded-lg border px-4 py-3 text-sm text-muted-foreground">
      The app list may be stale. Showing the last successful discovery; Refresh to try again.
    </p>
  {/if}

  <div id="app-directory" class="mt-7" aria-busy={loading}>
    {#if loading && !snapshot}
      <p role="status" class="rounded-lg border px-6 py-12 text-center text-sm text-muted-foreground">Loading apps...</p>
    {:else if snapshot && total === 0}
      <div class="rounded-lg border px-6 py-12 text-center">
        <h2 class="text-base font-medium">No apps found</h2>
        <p class="mt-2 text-sm text-muted-foreground">Refresh to check for web apps.</p>
      </div>
    {:else if snapshot && visibleApps.length === 0}
      <div class="rounded-lg border px-6 py-12 text-center">
        <h2 class="text-base font-medium">No apps match your search</h2>
        <p class="mt-2 text-sm text-muted-foreground">Try a name, hostname, description or group.</p>
      </div>
    {:else}
      <div class="space-y-7">
        {#each groups as group (group.name)}
          <section aria-label={group.name}>
            <h2 class="mb-3 text-xs font-semibold tracking-wider text-muted-foreground uppercase">{group.name}</h2>
            <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
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
