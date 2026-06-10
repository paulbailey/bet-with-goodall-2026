<script lang="ts">
  import { onMount } from "svelte";
  import type { TournamentState, DailySummaryFile } from "./types";
  import Header from "./components/Header.svelte";
  import SummaryBar from "./components/SummaryBar.svelte";
  import PushOptIn from "./components/PushOptIn.svelte";
  import DailySummary from "./components/DailySummary.svelte";
  import DailySummaryPage from "./components/DailySummaryPage.svelte";
  import TodaysMatches from "./components/TodaysMatches.svelte";
  import MaxPayoutPage from "./components/MaxPayoutPage.svelte";
  import MatchResultPage from "./components/MatchResultPage.svelte";
  import BetGrid from "./components/BetGrid.svelte";
  import MatchAccaBets from "./components/MatchAccaBets.svelte";
  import MatchResultBets from "./components/MatchResultBets.svelte";
  import FinalistBets from "./components/FinalistBets.svelte";
  import TournamentWinnerBets from "./components/TournamentWinnerBets.svelte";
  import TopScorerBets from "./components/TopScorerBets.svelte";
  import TopScorers from "./components/TopScorers.svelte";
  import GroupStandings from "./components/GroupStandings.svelte";

  const POLL_INTERVAL_MS = 60_000;
  // Routes reachable via in-app links. The match-detail page (/match/<id>) is
  // only reached by a push-notification deep link, so it isn't a navigable
  // target — it's handled separately as a view.
  type AppRoute = "/" | "/max-payout" | "/daily-summary";
  type ViewRoute = AppRoute | "/match";

  let data = $state<TournamentState | null>(null);
  let summary = $state<DailySummaryFile | null>(null);
  let error = $state<string | null>(null);
  let route = $state<ViewRoute>("/");
  let matchId = $state<string | null>(null);
  let lastUpdatedAt: string | null = null;
  let lastSummaryAt: string | null = null;

  const latestSummary = $derived(summary?.summaries?.[0] ?? null);

  function routeFromPath(pathname: string): ViewRoute {
    const cleanPath = pathname.replace(/\/+$/, "") || "/";
    if (cleanPath === "/max-payout") return "/max-payout";
    if (cleanPath === "/daily-summary") return "/daily-summary";
    if (cleanPath.startsWith("/match/")) return "/match";
    return "/";
  }

  function syncRoute() {
    const cleanPath = window.location.pathname.replace(/\/+$/, "") || "/";
    route = routeFromPath(cleanPath);
    matchId =
      route === "/match"
        ? decodeURIComponent(cleanPath.slice("/match/".length))
        : null;
  }

  function navigate(event: MouseEvent, to: AppRoute) {
    if (
      event.defaultPrevented ||
      event.button !== 0 ||
      event.altKey ||
      event.ctrlKey ||
      event.metaKey ||
      event.shiftKey
    ) {
      return;
    }

    event.preventDefault();
    if (window.location.pathname !== to) {
      history.pushState(null, "", to);
    }
    route = to;
    window.scrollTo({ top: 0 });
  }

  async function fetchState() {
    try {
      const res = await fetch(`/data/state.json?_=${Date.now()}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const json: TournamentState = await res.json();
      // state.json is an external feed; tolerate builder output that omits a
      // bet section (e.g. an older builder) so the page still renders.
      json.matches ??= [];
      json.bets ??= [];
      json.top_scorer_bets ??= [];
      json.tournament_winner_bets ??= [];
      json.match_result_bets ??= [];
      json.match_acca_bets ??= [];
      json.finalist_bets ??= [];
      json.top_scorers ??= [];
      json.max_payout ??= null;
      if (json.updated_at !== lastUpdatedAt) {
        lastUpdatedAt = json.updated_at;
        data = json;
        error = null;
      }
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load data";
    }
  }

  // The daily summary lives in its own file because it changes about once a day,
  // not every poll. A 404 (no summary generated yet) is fine — we just hide the
  // section. Only swap in new data when the file actually changed.
  async function fetchSummary() {
    try {
      const res = await fetch(`/data/daily-summary.json?_=${Date.now()}`);
      if (!res.ok) return;
      const json: DailySummaryFile = await res.json();
      if (json.updated_at !== lastSummaryAt) {
        lastSummaryAt = json.updated_at;
        summary = json;
      }
    } catch {
      // Network blip or missing file — keep whatever we last had.
    }
  }

  onMount(() => {
    syncRoute();
    window.addEventListener("popstate", syncRoute);
    fetchState();
    fetchSummary();
    const id = setInterval(() => {
      fetchState();
      fetchSummary();
    }, POLL_INTERVAL_MS);
    return () => {
      clearInterval(id);
      window.removeEventListener("popstate", syncRoute);
    };
  });
</script>

<Header
  updatedAt={data?.updated_at ?? null}
  phase={data?.tournament_phase ?? null}
/>

{#if route === "/match"}
  <!-- Self-contained: fetches its own data so a push deep-link works even if
       the dashboard's state.json is slow or unavailable. -->
  <MatchResultPage id={matchId} onNavigate={navigate} />
{:else if error}
  <p class="state-message error">Failed to load data: {error}</p>
{:else if !data}
  <p class="state-message">Loading…</p>
{:else}
  {#if route === "/max-payout"}
    <MaxPayoutPage {data} onNavigate={navigate} />
  {:else if route === "/daily-summary"}
    <DailySummaryPage file={summary} onNavigate={navigate} />
  {:else}
    <main class="app-content">
      <TodaysMatches matches={data.matches} />
      <SummaryBar {data} onNavigate={navigate} />
      <PushOptIn />
      {#if latestSummary}
        <DailySummary summary={latestSummary} onNavigate={navigate} />
      {/if}
      <BetGrid bets={data.bets} />
      <!-- The narrower bet types tile side-by-side to fill the width on desktop
           and wrap to a single column on small screens. -->
      <div class="bet-sections">
        {#if data.match_acca_bets.length > 0}
          <MatchAccaBets bets={data.match_acca_bets} />
        {/if}
        {#if data.match_result_bets.length > 0}
          <MatchResultBets bets={data.match_result_bets} />
        {/if}
        {#if data.finalist_bets.length > 0}
          <FinalistBets bets={data.finalist_bets} />
        {/if}
        {#if data.tournament_winner_bets.length > 0}
          <TournamentWinnerBets bets={data.tournament_winner_bets} />
        {/if}
        {#if data.top_scorer_bets.length > 0}
          <TopScorerBets bets={data.top_scorer_bets} />
        {/if}
        {#if data.top_scorers.length > 0}
          <TopScorers scorers={data.top_scorers} bets={data.top_scorer_bets} />
        {/if}
      </div>
      <GroupStandings groups={data.groups} />
    </main>
  {/if}
{/if}
