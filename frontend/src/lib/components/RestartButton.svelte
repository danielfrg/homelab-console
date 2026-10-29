<script lang="ts">
  import { Button, Modal, TextInput } from "@kenn-io/kit-ui";
  import { restartWorkload } from "$lib/console.js";

  let { workload, label, disabledReason = "", ondone }: {
    workload: string; label: string; disabledReason?: string; ondone?: () => void;
  } = $props();

  let open = $state(false);
  let reason = $state("");
  let busy = $state(false);
  let error = $state("");
  let result = $state("");

  async function confirm(): Promise<void> {
    busy = true;
    error = "";
    try {
      const out = await restartWorkload(workload, reason.trim());
      result = `${out.message}`;
      open = false;
      reason = "";
      ondone?.();
    } catch (err) {
      error = err instanceof Error ? err.message : "Restart failed";
    } finally {
      busy = false;
    }
  }
</script>

<span class="restart-control">
  <Button size="sm" surface="outline" disabled={Boolean(disabledReason)}
    title={disabledReason || `Restart ${label}`} onclick={() => { open = true; result = ""; error = ""; }}>
    Restart
  </Button>
  {#if result}<span role="status" class="restart-result">{result}</span>{/if}
</span>

{#if open}
  <Modal title={`Restart ${label}?`} tone="warning" onclose={() => open = false} maxWidth="440px">
    <p class="modal-copy">This does a rolling restart, like <code>kubectl rollout restart</code>.</p>
    <TextInput block ariaLabel="Reason (optional)" placeholder="Reason (optional)" value={reason}
      oninput={(value: string) => reason = value} />
    {#if error}<p role="alert" class="directory-error modal-copy">{error}</p>{/if}
    {#snippet footer()}
      <Button surface="outline" onclick={() => open = false}>Cancel</Button>
      <Button surface="solid" tone="danger" disabled={busy} onclick={() => void confirm()}>
        {busy ? "Restarting..." : "Restart"}
      </Button>
    {/snippet}
  </Modal>
{/if}
