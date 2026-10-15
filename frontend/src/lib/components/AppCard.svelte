<script lang="ts">
  import { appHostname, appHostnameParts, appIconURL, appInitials, type App } from "$lib/apps.js";

  let { app }: { app: App } = $props();
  const iconURL = $derived(appIconURL(app.icon));
  const hostname = $derived(appHostname(app));
  const hostnameParts = $derived(appHostnameParts(app));
  let failedIcon = $state<string | null>(null);
</script>

<a
  href={app.url}
  target="_blank"
  rel="noopener noreferrer"
  class="app-card"
  aria-label={`${app.name}, ${hostname}, opens in a new tab`}
  title={app.description || undefined}
>
  <div class="app-poster">
    <div class="app-art" aria-hidden="true">
      {#if iconURL && failedIcon !== iconURL}
        <img
          src={iconURL}
          alt=""
          width="48"
          height="48"
          loading="lazy"
          referrerpolicy="no-referrer"
          class="app-logo"
          data-icon={app.icon}
          onerror={() => { failedIcon = iconURL; }}
        />
      {:else}
        <span class="app-initials">{appInitials(app.name)}</span>
      {/if}
    </div>
    <div class="app-details">
      <h3 class="app-name" title={app.name}>{app.name}</h3>
      <p class="app-host" title={hostname}>
        <span class="host-prefix">{hostnameParts.prefix}</span>
        {#if hostnameParts.domain}
          <span class="host-domain">{hostnameParts.domain}</span>
        {/if}
      </p>
    </div>
  </div>
</a>
