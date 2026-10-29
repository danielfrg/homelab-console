<script lang="ts">
  import { appIconURL, appInitials } from "$lib/apps.js";
  import { statusTone, type AppTile } from "$lib/console.js";

  let { tile }: { tile: AppTile } = $props();
  const iconURL = $derived(appIconURL(tile.link?.icon ?? ""));
  let failedIcon = $state<string | null>(null);
</script>

<article class="app-card app-tile" title={tile.link?.description || tile.title}>
  <div class="app-art" aria-hidden="true">
    {#if iconURL && failedIcon !== iconURL}
      <img src={iconURL} alt="" width="32" height="32" loading="lazy" referrerpolicy="no-referrer"
        class="app-logo" onerror={() => { failedIcon = iconURL; }} />
    {:else}
      <span class="app-initials">{appInitials(tile.title)}</span>
    {/if}
  </div>
  <div class="app-details">
    <h3 class="app-name">
      <a class="app-tile-link" href={`#/apps/${encodeURIComponent(tile.app.name)}`}>{tile.title}</a>
    </h3>
    {#if tile.link}
      <a class="app-host app-host-link" href={tile.link.url} target="_blank" rel="noopener noreferrer"
        aria-label={`Open ${tile.host} in a new tab`}>{tile.host}</a>
    {:else}
      <p class="app-host">{tile.app.name}</p>
    {/if}
  </div>
  <span class="tile-status" aria-label={`${tile.app.sync}, ${tile.app.health}`}>
    <span class={`status-dot tone-${statusTone(tile.app.sync)}`} title={tile.app.sync}></span>
    <span class={`status-dot tone-${statusTone(tile.app.health)}`} title={tile.app.health}></span>
  </span>
</article>
