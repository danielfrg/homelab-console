<script lang="ts">
  import { Button } from "@ui-sdk/svelte/button";
  import { formatRelativeTime, formatTimestamp } from "$lib/format.js";
  import type { Pod } from "$lib/console.js";
  import {
    sortRows,
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
    type TableSort,
  } from "../../design-system/custom/table";
  import StatusChip from "./StatusChip.svelte";

  let {
    pods,
    showOwner = false,
    onlogs,
  }: { pods: Pod[]; showOwner?: boolean; onlogs: (pod: Pod) => void } = $props();

  let sort = $state<TableSort | null>(null);

  const status = (pod: Pod): string =>
    pod.problem ? "Degraded" : pod.ready ? "Healthy" : pod.phase;
  const rows = $derived(
    sortRows(pods, sort, (pod, key) => {
      switch (key) {
        case "name":
          return `${pod.namespace}/${pod.name}`;
        case "app":
          return pod.app || null;
        case "status":
          return status(pod);
        case "restarts":
          return pod.restarts;
        case "age":
          return pod.started_at ? Date.parse(pod.started_at) : null;
        default:
          return null;
      }
    }),
  );
</script>

<div class="panel">
  <Table bind:sort>
    <TableHeader>
      <TableRow>
        <TableHead sortKey="name">Pod</TableHead>
        {#if showOwner}<TableHead sortKey="app">App</TableHead>{/if}
        <TableHead sortKey="status">Status</TableHead>
        <TableHead sortKey="restarts" firstDirection="desc" align="end">Restarts</TableHead>
        <!-- Newest first: the largest start time is the youngest pod. -->
        <TableHead sortKey="age" firstDirection="desc">Age</TableHead>
        <TableHead><span class="sr-only">Logs</span></TableHead>
      </TableRow>
    </TableHeader>
    <TableBody>
      {#each rows as pod (`${pod.namespace}/${pod.name}`)}
        <TableRow>
          <TableCell class="mono">{showOwner ? `${pod.namespace}/` : ""}{pod.name}</TableCell>
          {#if showOwner}
            <TableCell>
              {#if pod.app}<a class="mono" href={`#/apps/${encodeURIComponent(pod.app)}`}
                  >{pod.app}</a
                >{:else}<span class="muted">—</span>{/if}
            </TableCell>
          {/if}
          <TableCell>
            {#if pod.problem}<StatusChip status="Degraded" />
              <span class="reason">{pod.problem}</span>
            {:else}<StatusChip status={pod.ready ? "Healthy" : pod.phase} />{/if}
          </TableCell>
          <TableCell
            align="end"
            class="mono"
            title={pod.last_termination
              ? `Last: ${pod.last_termination.reason} (exit ${pod.last_termination.exit_code}) ${formatRelativeTime(pod.last_termination.finished_at)}`
              : undefined}>{pod.restarts}</TableCell
          >
          <TableCell class="muted" title={formatTimestamp(pod.started_at)}
            >{formatRelativeTime(pod.started_at)}</TableCell
          >
          <TableCell align="end">
            <Button data-variant="link" onclick={() => onlogs(pod)}>Logs</Button>
          </TableCell>
        </TableRow>
      {/each}
    </TableBody>
  </Table>
</div>
