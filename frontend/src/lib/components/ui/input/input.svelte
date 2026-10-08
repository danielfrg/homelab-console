<script lang="ts">
  import { cn, type WithElementRef } from "$lib/utils.js";
  import type { HTMLInputAttributes } from "svelte/elements";

  let {
    ref = $bindable(null),
    value = $bindable(""),
    class: className,
    type = "text",
    ...restProps
  }: WithElementRef<HTMLInputAttributes, HTMLInputElement> = $props();
</script>

<input
  bind:this={ref}
  bind:value
  {type}
  data-slot="input"
  class={cn("border-input placeholder:text-muted-foreground flex h-11 w-full min-w-0 rounded-md border bg-transparent px-3 py-2 text-sm outline-none transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30 disabled:pointer-events-none disabled:opacity-50 motion-reduce:transition-none", className)}
  {...restProps}
/>
