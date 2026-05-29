<script lang="ts">
  import { onMount } from "svelte";
  import type { TournamentState } from "./types";
  import Header from "./components/Header.svelte";
  import SummaryBar from "./components/SummaryBar.svelte";
  import MaxPayoutPage from "./components/MaxPayoutPage.svelte";
  import BetGrid from "./components/BetGrid.svelte";
  import MatchAccaBets from "./components/MatchAccaBets.svelte";
  import MatchResultBets from "./components/MatchResultBets.svelte";
  import FinalistBets from "./components/FinalistBets.svelte";
  import TournamentWinnerBets from "./components/TournamentWinnerBets.svelte";
  import TopScorerBets from "./components/TopScorerBets.svelte";
  import TopScorers from "./components/TopScorers.svelte";
  import GroupStandings from "./components/GroupStandings.svelte";

  const POLL_INTERVAL_MS = 60_000;
  type AppRoute = "/" | "/max-payout";

  let data = $state<TournamentState | null>(null);
  let error = $state<string | null>(null);
  let route = $state<AppRoute>("/");
  let lastUpdatedAt: string | null = null;

  function routeFromPath(pathname: string): AppRoute {
    const cleanPath = pathname.replace(/\/+$/, "") || "/";
    return cleanPath === "/max-payout" ? "/max-payout" : "/";
  }

  function syncRoute() {
    route = routeFromPath(window.location.pathname);
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

  onMount(() => {
    syncRoute();
    window.addEventListener("popstate", syncRoute);
    fetchState();
    const id = setInterval(fetchState, POLL_INTERVAL_MS);
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

{#if error}
  <p class="state-message error">Failed to load data: {error}</p>
{:else if !data}
  <p class="state-message">Loading…</p>
{:else}
  {#if route === "/max-payout"}
    <MaxPayoutPage {data} onNavigate={navigate} />
  {:else}
    <main class="app-content">
      <SummaryBar {data} onNavigate={navigate} />
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
