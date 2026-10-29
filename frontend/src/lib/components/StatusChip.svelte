<script lang="ts">
  import { Chip } from "@kenn-io/kit-ui";
  import { statusTone } from "$lib/console.js";

  let { status }: { status: string } = $props();
  const tone = $derived(statusTone(status));
</script>

<Chip size="sm" dot tone={tone === "neutral" ? "muted" : tone}>{status || "Unknown"}</Chip>
