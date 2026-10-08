<script lang="ts">
  import { appHostname, appIconURL, appInitials, type App } from "$lib/apps.js";
  import * as Card from "$lib/components/ui/card/index.js";

  let { app }: { app: App } = $props();
  const iconURL = $derived(appIconURL(app.icon));
  let failedIcon = $state<string | null>(null);
</script>

<a
  href={app.url}
  target="_blank"
  rel="noopener noreferrer"
  class="group block h-full min-w-0 rounded-lg outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
>
  <Card.Root size="sm" class="h-full gap-3 px-4 py-4 transition-colors group-hover:bg-accent/50 motion-reduce:transition-none">
    <div class="flex min-w-0 items-start gap-3">
      <div class="flex size-9 shrink-0 items-center justify-center" aria-hidden="true">
        {#if iconURL && failedIcon !== iconURL}
          <img
            src={iconURL}
            alt=""
            width="32"
            height="32"
            loading="lazy"
            referrerpolicy="no-referrer"
            class="size-8 object-contain"
            onerror={() => { failedIcon = iconURL; }}
          />
        {:else}
          <span class="text-sm font-semibold text-muted-foreground">{appInitials(app.name)}</span>
        {/if}
      </div>
      <div class="min-w-0 flex-1">
        <h3 class="text-sm font-semibold leading-5 [overflow-wrap:anywhere]">{app.name}</h3>
        <p class="mt-1 text-xs leading-5 text-muted-foreground [overflow-wrap:anywhere]">{appHostname(app)}</p>
      </div>
    </div>
    {#if app.description}
      <p class="text-sm leading-5 text-muted-foreground [overflow-wrap:anywhere]">{app.description}</p>
    {/if}
    <span class="sr-only">Opens in a new tab</span>
  </Card.Root>
</a>
