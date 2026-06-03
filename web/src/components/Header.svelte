<script lang="ts">
  interface Props {
    updatedAt: string | null;
    phase: string | null;
  }

  let { updatedAt, phase }: Props = $props();

  const PHASE_LABELS: Record<string, string> = {
    pre_tournament: "Pre-Tournament",
    group_stage: "Group Stage",
    knockout: "Knockout Rounds",
    complete: "Tournament Complete",
  };

  // Opening match of the 2026 World Cup (Mexico City, 11 June 2026).
  const TOURNAMENT_START = new Date("2026-06-11T00:00:00");

  // Whole days from now until kick-off, rounded up so the morning of the opener
  // still reads "1 day to go" rather than "0". Negative once it has started.
  let daysToGo = $derived(
    Math.ceil((TOURNAMENT_START.getTime() - Date.now()) / 86_400_000),
  );

  // Only count down before kick-off: hide once the tournament is under way (or
  // the day has arrived). phase is null until the first data load, which is
  // still pre-tournament, so we treat that as "not started".
  let started = $derived(
    phase === "group_stage" || phase === "knockout" || phase === "complete",
  );
  let showCountdown = $derived(!started && daysToGo > 0);

  function formatTimestamp(iso: string): string {
    return new Date(iso).toLocaleString(undefined, {
      dateStyle: "medium",
      timeStyle: "short",
    });
  }
</script>

<header class="site-header">
  <div class="header-inner">
    <div class="header-brand">
      <div class="header-trophy">⚽</div>
      <div>
        <h1 class="header-title">Bet With Goodall</h1>
        <p class="header-subtitle">World Cup 2026</p>
      </div>
    </div>
    <div class="header-meta">
      {#if showCountdown}
        <span class="countdown-badge">
          {daysToGo} {daysToGo === 1 ? "day" : "days"} to go
        </span>
      {/if}
      {#if phase}
        <span class="phase-badge">{PHASE_LABELS[phase] ?? phase}</span>
      {/if}
      {#if updatedAt}
        <span class="updated-at">Updated {formatTimestamp(updatedAt)}</span>
      {/if}
    </div>
  </div>
</header>
