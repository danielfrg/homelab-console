<script lang="ts">
  import { StatusBadge } from "../../design-system/custom/status-badge";
  import { statusTone } from "$lib/console.js";

  let { status }: { status: string } = $props();
</script>

<StatusBadge tone={statusTone(status)}>{status || "Unknown"}</StatusBadge>
