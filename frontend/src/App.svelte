<script lang="ts">
  import { onMount } from "svelte";
  import { Button } from "$lib/components/ui/button/index.js";
  import * as Card from "$lib/components/ui/card/index.js";

  let message = $state("");
  let loading = $state(false);
  let error = $state("");

  async function loadMessage(): Promise<void> {
    if (loading) return;
    loading = true;
    error = "";

    try {
      const response = await fetch("/api/hello");
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      const data: unknown = await response.json();
      if (
        typeof data !== "object" ||
        data === null ||
        !("message" in data) ||
        typeof data.message !== "string"
      ) {
        throw new Error("Invalid greeting response");
      }
      message = data.message;
    } catch {
      error = "Couldn't load the message. Try again.";
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    void loadMessage();
  });
</script>

<main class="mx-auto flex min-h-svh w-full max-w-md flex-col justify-center px-6 py-12">
  <h1 class="mb-6 text-3xl font-semibold tracking-tight">Homelab</h1>

  <Card.Root>
    <Card.Header>
      <Card.Title class="text-base">Welcome</Card.Title>
    </Card.Header>
    <Card.Content>
      <p
        role="status"
        aria-live="polite"
        aria-atomic="true"
        class="min-h-7 text-base leading-relaxed"
        class:text-destructive={Boolean(error)}
      >
        {#if loading}
          Loading message...
        {:else if error}
          {error}
        {:else}
          {message}
        {/if}
      </p>
    </Card.Content>
    <Card.Footer>
      <Button
        class="h-10 px-4 text-sm"
        disabled={loading}
        onclick={() => void loadMessage()}
      >
        Refresh message
      </Button>
    </Card.Footer>
  </Card.Root>
</main>
