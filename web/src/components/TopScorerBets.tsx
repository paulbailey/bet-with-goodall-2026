import type { TopScorerBet, TopScorerBetStatus } from '../types'
import { getCountry } from '../countries'
import { Flag } from './Flag'

interface TopScorerBetsProps {
  bets: TopScorerBet[]
}

const STATUS_CLASS: Record<TopScorerBetStatus, string> = {
  alive: 'bet-alive',
  won:   'bet-won',
  lost:  'bet-lost',
}

const STATUS_LABEL: Record<TopScorerBetStatus, string> = {
  alive: 'Alive',
  won:   'Won!',
  lost:  'Bust',
}

export function TopScorerBets({ bets }: TopScorerBetsProps) {
  return (
    <section className="scorer-bets-section">
      <h2 className="section-title">Top Scorer Bets</h2>
      <div className="bet-grid-scroll">
        <table className="bet-grid">
          <thead>
            <tr>
              <th className="col-ts-player">Player</th>
              <th className="col-ts-team">Team</th>
              <th className="col-stake">Stake</th>
              <th className="col-return">Return</th>
              <th className="col-status">Status</th>
            </tr>
          </thead>
          <tbody>
            {bets.map((bet, index) => {
              const { fi } = getCountry(bet.team)
              return (
                <tr key={bet.id} className={`bet-row ${STATUS_CLASS[bet.status]}`}>
                  <td className="col-ts-player">{bet.player}</td>
                  <td className="col-ts-team">
                    <Flag fi={fi} className="team-flag" />
                    {bet.team}
                  </td>
                  <td className="col-stake">
                    {bet.stake != null ? `£${bet.stake.toFixed(2)}` : '—'}
                  </td>
                  <td className={`col-return ${bet.status === 'won' ? 'return-won' : ''}`}>
                    {bet.potential_return != null ? `£${bet.potential_return.toFixed(2)}` : '—'}
                  </td>
                  <td className={`col-status status-cell ${STATUS_CLASS[bet.status]}`}>
                    {STATUS_LABEL[bet.status]}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </section>
  )
}
