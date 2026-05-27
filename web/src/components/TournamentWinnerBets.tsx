import type { TournamentWinnerBet, TournamentWinnerBetStatus } from '../types'
import { getCountry } from '../countries'
import { Flag } from './Flag'

interface TournamentWinnerBetsProps {
  bets: TournamentWinnerBet[]
}

const STATUS_CLASS: Record<TournamentWinnerBetStatus, string> = {
  alive: 'bet-alive',
  won:   'bet-won',
  lost:  'bet-lost',
}

const STATUS_LABEL: Record<TournamentWinnerBetStatus, string> = {
  alive: 'Alive',
  won:   'Won!',
  lost:  'Bust',
}

export function TournamentWinnerBets({ bets }: TournamentWinnerBetsProps) {
  return (
    <section className="winner-bets-section">
      <h2 className="section-title">Tournament Winner Bets</h2>
      <div className="bet-grid-scroll">
        <table className="bet-grid">
          <thead>
            <tr>
              <th className="col-ts-team">Team</th>
              <th className="col-stake">Stake</th>
              <th className="col-return">Return</th>
              <th className="col-status">Status</th>
            </tr>
          </thead>
          <tbody>
            {bets.map((bet) => {
              const { fi } = getCountry(bet.team)
              return (
                <tr key={bet.id} className={`bet-row ${STATUS_CLASS[bet.status]}`}>
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
