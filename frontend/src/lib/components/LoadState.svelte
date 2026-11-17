<script lang="ts">
  import type { Snippet } from "svelte";
  import { Button } from "@ui-sdk/svelte/button";
  import Spinner from "./Spinner.svelte";

  let {
    loading,
    error,
    stale = false,
    hasData,
    retry,
    children,
  }: {
    loading: boolean;
    error: string;
    stale?: boolean;
    hasData: boolean;
    retry: () => void;
    children: Snippet;
  } = $props();
</script>

{#if error}
  <div role="alert" class="notice error">
    <p>{hasData ? `${error} Showing the last data; it may be stale.` : error}</p>
    <Button disabled={loading} onclick={retry}>{loading ? "Retrying..." : "Retry"}</Button>
  </div>
{:else if stale}
  <div role="status" class="notice warn">
    <p>Cluster data may be stale. Showing the last successful read.</p>
    <Button disabled={loading} onclick={retry}>{loading ? "Retrying..." : "Retry"}</Button>
  </div>
{/if}
{#if loading && !hasData}
  <div class="center"><Spinner label="Loading" /></div>
{:else if hasData}
  {@render children()}
{/if}
