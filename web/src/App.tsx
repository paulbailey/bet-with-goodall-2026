import { useStateData } from './hooks/useStateData'
import { Header } from './components/Header'
import { BetGrid } from './components/BetGrid'
import { TournamentWinnerBets } from './components/TournamentWinnerBets'
import { TopScorerBets } from './components/TopScorerBets'
import { TopScorers } from './components/TopScorers'
import { GroupStandings } from './components/GroupStandings'

export default function App() {
  const { data, error } = useStateData()

  if (error) {
    return (
      <>
        <Header updatedAt={null} phase={null} />
        <p className="state-message error">Failed to load data: {error}</p>
      </>
    )
  }

  if (!data) {
    return (
      <>
        <Header updatedAt={null} phase={null} />
        <p className="state-message">Loading…</p>
      </>
    )
  }

  return (
    <>
      <Header updatedAt={data.updated_at} phase={data.tournament_phase} />
      <main className="app-content">
        <BetGrid bets={data.bets} />
        {data.tournament_winner_bets.length > 0 && (
          <TournamentWinnerBets bets={data.tournament_winner_bets} />
        )}
        {(data.top_scorer_bets.length > 0 || data.top_scorers.length > 0) && (
          <div className="scorer-panels">
            {data.top_scorer_bets.length > 0 && (
              <TopScorerBets bets={data.top_scorer_bets} />
            )}
            {data.top_scorers.length > 0 && (
              <TopScorers scorers={data.top_scorers} bets={data.top_scorer_bets} />
            )}
          </div>
        )}
        <GroupStandings groups={data.groups} />
      </main>
    </>
  )
}
