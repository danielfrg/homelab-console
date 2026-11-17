<script lang="ts">
  import { onMount } from "svelte";
  import {
    SegmentedControl,
    SegmentedControlIndicator,
    SegmentedControlItem,
  } from "@ui-sdk/svelte/segmented-control";
  import { Preferences } from "@ui-sdk/svelte/preferences";
  import { colorScheme } from "$lib/color-scheme";
  import { ThemeToggle } from "./design-system/custom/theme-toggle";
  import AppDetail from "$lib/components/AppDetail.svelte";
  import AppsList from "$lib/components/AppsList.svelte";
  import PodsTable from "$lib/components/PodsTable.svelte";

  type Route = { view: "apps" | "pods" } | { view: "app"; name: string };

  function parseRoute(hash: string): Route {
    const path = hash.replace(/^#\/?/, "");
    if (path.startsWith("apps/") && path.length > 5)
      return { view: "app", name: decodeURIComponent(path.slice(5)) };
    if (path === "pods") return { view: "pods" };
    return { view: "apps" };
  }

  let route = $state<Route>(parseRoute(location.hash));
  const tab = $derived(route.view === "app" ? "apps" : route.view);

  onMount(() => {
    const update = () => (route = parseRoute(location.hash));
    addEventListener("hashchange", update);
    return () => removeEventListener("hashchange", update);
  });
</script>

<Preferences preferences={[colorScheme]}>
  <main class="page">
    <header class="directory-header">
      <h1>Homelab Console</h1>
      <div class="directory-controls">
        <SegmentedControl
          aria-label="Views"
          value={tab}
          onValueChange={(value) => {
            if (value) location.hash = value === "apps" ? "#/" : `#/${value}`;
          }}
        >
          <SegmentedControlIndicator />
          <SegmentedControlItem value="apps">Apps</SegmentedControlItem>
          <SegmentedControlItem value="pods">Pods</SegmentedControlItem>
        </SegmentedControl>
        <ThemeToggle {colorScheme} />
      </div>
    </header>

    {#if route.view === "apps"}
      <AppsList />
    {:else if route.view === "app"}
      <AppDetail name={route.name} />
    {:else}
      <PodsTable />
    {/if}
  </main>
</Preferences>
