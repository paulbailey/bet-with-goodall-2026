import { useState } from 'react'
import type { TopScorer, TopScorerBet } from '../types'
import { getCountry } from '../countries'
import { Flag } from './Flag'

interface TopScorersProps {
  scorers: TopScorer[]
  bets: TopScorerBet[]
}

export function TopScorers({ scorers, bets }: TopScorersProps) {
  const [open, setOpen] = useState(true)

  const predictedPlayers = new Set(bets.map((b) => b.player))
  const sorted = [...scorers].sort((a, b) => b.goals - a.goals)

  return (
    <section className="standings-section">
      <button className="section-toggle" onClick={() => setOpen((o) => !o)}>
        <h2 className="section-title">Top Scorers</h2>
        <span className="toggle-icon">{open ? '▲' : '▼'}</span>
      </button>
      {open && (
        <div className="top-scorers-wrap">
          <table className="top-scorers-table">
            <thead>
              <tr>
                <th className="col-rank">#</th>
                <th className="col-ts-player">Player</th>
                <th className="col-ts-team">Team</th>
                <th>Goals</th>
                <th>Games</th>
              </tr>
            </thead>
            <tbody>
              {sorted.map((s, i) => {
                const { fi } = getCountry(s.team)
                const isPredicted = predictedPlayers.has(s.player)
                return (
                  <tr
                    key={`${s.player}-${s.team}`}
                    className={[
                      s.team_eliminated ? 'scorer-eliminated' : '',
                      isPredicted       ? 'scorer-predicted'  : '',
                    ].filter(Boolean).join(' ')}
                  >
                    <td className="col-rank">{i + 1}</td>
                    <td className="col-ts-player">
                      {s.player}
                      {isPredicted && <span className="scorer-bet-marker" title="Bet placed on this player"> ★</span>}
                    </td>
                    <td className="col-ts-team">
                      <Flag fi={fi} className="team-flag" />
                      {s.team}
                    </td>
                    <td className="col-goals">{s.goals}</td>
                    <td>{s.games}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
    </section>
  )
}
