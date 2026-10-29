<script lang="ts">
  import { Button, Modal } from "@kenn-io/kit-ui";
  import { fetchLogs, type LogsResult } from "$lib/console.js";

  let { pod, canPrevious, onclose }: { pod: string; canPrevious: boolean; onclose: () => void } = $props();
  let previous = $state(false);
  let logs = $state<LogsResult | null>(null);
  let loading = $state(false);
  let error = $state("");
  let request = 0;

  async function load(): Promise<void> {
    const id = ++request;
    loading = true;
    error = "";
    try {
      const result = await fetchLogs(pod, previous);
      if (id === request) logs = result;
    } catch (err) {
      if (id !== request) return;
      logs = null;
      error = err instanceof Error ? err.message : "Couldn't load logs.";
    } finally {
      if (id === request) loading = false;
    }
  }

  $effect(() => {
    void pod;
    void previous;
    void load();
  });
</script>

<Modal title={`${pod}${previous ? " · previous" : ""}`} onclose={onclose} width="min(1100px, 94vw)" maxWidth="94vw" maxHeight="88vh">
  {#if error}<p role="alert" class="directory-error">{error}</p>{/if}
  {#if loading && !logs}<p role="status" class="muted">Loading logs...</p>{/if}
  {#if logs}
    <pre class="logs-text">{logs.text || "(no output)"}</pre>
    {#if logs.truncated}<p class="muted small">Output was cut at 64 KiB.</p>{/if}
  {/if}
  {#snippet footer()}
    {#if canPrevious}<Button surface="outline" onclick={() => previous = !previous}>{previous ? "Current" : "Previous crash"}</Button>{/if}
    <Button surface="outline" disabled={loading} onclick={() => void load()}>Reload</Button>
    <Button surface="outline" onclick={onclose}>Close</Button>
  {/snippet}
</Modal>
