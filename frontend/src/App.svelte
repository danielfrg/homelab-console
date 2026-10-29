<script lang="ts">
  import { onMount } from "svelte";
  import { SegmentedControl, ThemeToggle } from "@kenn-io/kit-ui";
  import AppDetail from "$lib/components/AppDetail.svelte";
  import AppsList from "$lib/components/AppsList.svelte";
  import PodsTable from "$lib/components/PodsTable.svelte";

  type Route = { view: "apps" | "pods" } | { view: "app"; name: string };

  function parseRoute(hash: string): Route {
    const path = hash.replace(/^#\/?/, "");
    if (path.startsWith("apps/") && path.length > 5) return { view: "app", name: decodeURIComponent(path.slice(5)) };
    if (path === "pods") return { view: "pods" };
    return { view: "apps" };
  }

  let route = $state<Route>(parseRoute(location.hash));
  const tab = $derived(route.view === "app" ? "apps" : route.view);

  onMount(() => {
    const update = () => route = parseRoute(location.hash);
    addEventListener("hashchange", update);
    return () => removeEventListener("hashchange", update);
  });
</script>

<main class="app-launcher">
  <header class="directory-header">
    <h1>Homelab Console</h1>
    <div class="directory-controls">
      <SegmentedControl ariaLabel="Views" value={tab}
        options={[{ value: "apps", label: "Apps" }, { value: "pods", label: "Pods" }]}
        onchange={(value) => location.hash = value === "apps" ? "#/" : `#/${value}`} />
      <ThemeToggle />
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
