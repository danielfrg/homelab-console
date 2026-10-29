<script lang="ts">
  import type { Snippet } from "svelte";
  import { Button } from "@kenn-io/kit-ui";

  let { loading, error, stale = false, hasData, retry, children }: {
    loading: boolean; error: string; stale?: boolean; hasData: boolean; retry: () => void; children: Snippet;
  } = $props();
</script>

{#if error}
  <div role="alert" class="directory-notice directory-error">
    <p>{hasData ? `${error} Showing the last data; it may be stale.` : error}</p>
    <Button surface="outline" disabled={loading} onclick={retry}>{loading ? "Retrying..." : "Retry"}</Button>
  </div>
{:else if stale}
  <div role="status" class="directory-notice">
    <p>Cluster data may be stale. Showing the last successful read.</p>
    <Button surface="outline" disabled={loading} onclick={retry}>{loading ? "Retrying..." : "Retry"}</Button>
  </div>
{/if}
{#if loading && !hasData}
  <p role="status" class="directory-empty">Loading...</p>
{:else if hasData}
  {@render children()}
{/if}
