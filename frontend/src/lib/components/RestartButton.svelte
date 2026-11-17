<script lang="ts">
  import { Button } from "@ui-sdk/svelte/button";
  import { Input } from "@ui-sdk/svelte/input";
  import {
    AlertDialog,
    AlertDialogBackdrop,
    AlertDialogClose,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogTitle,
  } from "@ui-sdk/svelte/alert-dialog";
  import { restartWorkload } from "$lib/console.js";

  let {
    workload,
    label,
    disabledReason = "",
    ondone,
  }: {
    workload: string;
    label: string;
    disabledReason?: string;
    ondone?: () => void;
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
  <Button
    disabled={Boolean(disabledReason)}
    title={disabledReason || `Restart ${label}`}
    onclick={() => {
      open = true;
      result = "";
      error = "";
    }}
  >
    Restart
  </Button>
  {#if result}<span role="status" class="restart-result">{result}</span>{/if}
</span>

{#if open}
  <!-- The parent owns the open state; a close request while busy is ignored. -->
  <AlertDialog
    open
    onOpenChange={(next, details) => {
      if (next) return;
      details.cancel();
      if (!busy) open = false;
    }}
  >
    <AlertDialogBackdrop />
    <AlertDialogContent>
      <div data-slot="stack">
        <AlertDialogTitle>Restart {label}?</AlertDialogTitle>
        <AlertDialogDescription>
          This does a rolling restart, like <code>kubectl rollout restart</code>.
        </AlertDialogDescription>
        <Input bind:value={reason} aria-label="Reason (optional)" placeholder="Reason (optional)" />
      </div>
      {#if error}<p data-slot="hint" role="alert" class="notice error">{error}</p>{/if}
      <div data-slot="row">
        <AlertDialogClose disabled={busy}>Cancel</AlertDialogClose>
        <Button data-variant="destructive" disabled={busy} onclick={() => void confirm()}>
          {busy ? "Restarting..." : "Restart"}
        </Button>
      </div>
    </AlertDialogContent>
  </AlertDialog>
{/if}
