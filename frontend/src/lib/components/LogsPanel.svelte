<script lang="ts">
  import { Button } from "@ui-sdk/svelte/button";
  import { fetchLogs, type LogsResult } from "$lib/console.js";
  import Modal from "./Modal.svelte";
  import Spinner from "./Spinner.svelte";

  let { pod, canPrevious, onclose }: { pod: string; canPrevious: boolean; onclose: () => void } =
    $props();
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

<Modal title={`${pod}${previous ? " · previous" : ""}`} size="xl" {onclose}>
  {#if error}<p role="alert" class="notice error">{error}</p>{/if}
  {#if loading && !logs}<div class="center"><Spinner label="Loading logs" /></div>{/if}
  {#if logs}
    <pre class="logs-text">{logs.text || "(no output)"}</pre>
    {#if logs.truncated}<p class="muted small">Output was cut at 64 KiB.</p>{/if}
  {/if}
  {#snippet footer()}
    {#if canPrevious}<Button onclick={() => (previous = !previous)}
        >{previous ? "Current" : "Previous crash"}</Button
      >{/if}
    <Button disabled={loading} onclick={() => void load()}>Reload</Button>
    <Button onclick={onclose}>Close</Button>
  {/snippet}
</Modal>
