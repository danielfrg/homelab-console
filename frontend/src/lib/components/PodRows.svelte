<script lang="ts">
  import { relativeTime, type Pod } from "$lib/console.js";
  import StatusChip from "./StatusChip.svelte";

  let { pods, showOwner = false, onlogs }: { pods: Pod[]; showOwner?: boolean; onlogs: (pod: Pod) => void } = $props();
</script>

<div class="table-wrap">
  <table class="data-table">
    <thead>
      <tr><th>Pod</th>{#if showOwner}<th>App</th>{/if}<th>Status</th><th>Restarts</th><th>Age</th><th><span class="sr-only">Logs</span></th></tr>
    </thead>
    <tbody>
      {#each pods as pod (`${pod.namespace}/${pod.name}`)}
        <tr>
          <td class="mono">{showOwner ? `${pod.namespace}/` : ""}{pod.name}</td>
          {#if showOwner}<td>{#if pod.app}<a class="mono" href={`#/apps/${encodeURIComponent(pod.app)}`}>{pod.app}</a>{:else}<span class="muted">—</span>{/if}</td>{/if}
          <td>
            {#if pod.problem}<StatusChip status="Degraded" /> <span class="reason">{pod.problem}</span>
            {:else}<StatusChip status={pod.ready ? "Healthy" : pod.phase} />{/if}
          </td>
          <td class="mono" title={pod.last_termination ? `Last: ${pod.last_termination.reason} (exit ${pod.last_termination.exit_code}) ${relativeTime(pod.last_termination.finished_at)}` : undefined}>{pod.restarts}</td>
          <td class="muted">{relativeTime(pod.started_at)}</td>
          <td><button class="link-button" type="button" onclick={() => onlogs(pod)}>Logs</button></td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>
