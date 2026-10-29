<script lang="ts">
  import { Button } from "@kenn-io/kit-ui";
  import { fetchLogs, type LogsResult } from "$lib/console.js";

  let { pod, canPrevious, onclose }: { pod: string; canPrevious: boolean; onclose: () => void } = $props();
  let previous = $state(false);
  let logs = $state<LogsResult | null>(null);
  let loading = $state(false);
  let error = $state("");

  async function load(): Promise<void> {
    loading = true;
    error = "";
    try {
      logs = await fetchLogs(pod, previous);
    } catch (err) {
      error = err instanceof Error ? err.message : "Couldn't load logs.";
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    void pod;
    void previous;
    void load();
  });
</script>

<section class="logs-panel" aria-label={`Logs for ${pod}`}>
  <header>
    <span class="mono">{pod}{logs ? ` · ${logs.container}` : ""}{previous ? " · previous" : ""}</span>
    <span class="logs-actions">
      {#if canPrevious}<Button size="sm" surface="outline" onclick={() => previous = !previous}>{previous ? "Current" : "Previous crash"}</Button>{/if}
      <Button size="sm" surface="outline" disabled={loading} onclick={() => void load()}>Reload</Button>
      <Button size="sm" surface="outline" onclick={onclose}>Close</Button>
    </span>
  </header>
  {#if error}<p role="alert" class="directory-error">{error}</p>{/if}
  {#if loading && !logs}<p role="status" class="muted">Loading logs...</p>{/if}
  {#if logs}
    <pre class="logs-text">{logs.text || "(no output)"}</pre>
    {#if logs.truncated}<p class="muted small">Output was cut at 64 KiB.</p>{/if}
  {/if}
</section>
