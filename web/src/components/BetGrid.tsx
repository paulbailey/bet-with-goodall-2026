import type { Bet, LegStatus, BetStatus } from '../types'
import { getCountry } from '../countries'
import { Flag } from './Flag'

interface BetGridProps {
  bets: Bet[]
}

const STATUS_CLASS: Record<LegStatus, string> = {
  pending: 'leg-pending',
  alive:   'leg-alive',
  won:     'leg-won',
  lost:    'leg-lost',
}

const STATUS_LABEL: Record<LegStatus, string> = {
  pending: '–',
  alive:   '✓',
  won:     '★',
  lost:    '✗',
}

const BET_STATUS_CLASS: Record<BetStatus, string> = {
  pending: 'bet-pending',
  alive:   'bet-alive',
  won:     'bet-won',
  lost:    'bet-lost',
}

const BET_STATUS_LABEL: Record<BetStatus, string> = {
  pending: 'Pending',
  alive:   'Alive',
  won:     'Won!',
  lost:    'Bust',
}

export function BetGrid({ bets }: BetGridProps) {
  const allGroups = Array.from(
    new Set(bets.flatMap((b) => b.legs.map((l) => l.group)))
  ).sort()

  return (
    <section className="bet-grid-section">
      <h2 className="section-title">Accumulators</h2>

      {/* Desktop table */}
      <div className="bet-grid-scroll">
        <table className="bet-grid">
          <thead>
            <tr>
              {allGroups.map((g) => (
                <th key={g} className="col-group">Grp {g}</th>
              ))}
              <th className="col-stake">Stake</th>
              <th className="col-return">Return</th>
              <th className="col-status">Status</th>
            </tr>
          </thead>
          <tbody>
            {bets.map((bet) => {
              const legByGroup = Object.fromEntries(
                bet.legs.map((l) => [l.group, l])
              )
              return (
                <tr key={bet.id} className={`bet-row ${BET_STATUS_CLASS[bet.status]}`}>
                  {allGroups.map((g) => {
                    const leg = legByGroup[g]
                    if (!leg) {
                      return <td key={g} className="leg-cell leg-empty"><span className="leg-na">—</span></td>
                    }
                    const { fi, code } = getCountry(leg.team)
                    return (
                      <td key={g} className={`leg-cell ${STATUS_CLASS[leg.status]}`} title={leg.team}>
                        <span className="leg-full">{leg.team}</span>
                        <span className="leg-short">
                          <Flag fi={fi} className="leg-flag" />
                          <span className="leg-code">{code}</span>
                        </span>
                        <span className="leg-icon">{STATUS_LABEL[leg.status]}</span>
                      </td>
                    )
                  })}
                  <td className="col-stake">
                    {bet.stake != null ? `£${bet.stake.toFixed(2)}` : '—'}
                  </td>
                  <td className={`col-return ${bet.status === 'won' ? 'return-won' : ''}`}>
                    {bet.potential_return != null ? `£${bet.potential_return.toFixed(2)}` : '—'}
                  </td>
                  <td className={`col-status status-cell ${BET_STATUS_CLASS[bet.status]}`}>
                    {BET_STATUS_LABEL[bet.status]}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      {/* Mobile cards */}
      <div className="bet-cards">
        {bets.map((bet) => (
          <div key={bet.id} className={`bet-card ${BET_STATUS_CLASS[bet.status]}`}>
            <div className="bet-card-legs">
              {bet.legs.map((leg) => {
                const { fi, code } = getCountry(leg.team)
                return (
                  <div key={leg.group} className={`bet-card-leg ${STATUS_CLASS[leg.status]}`} title={leg.team}>
                    <span className="bet-card-leg-group">Grp {leg.group}</span>
                    <Flag fi={fi} className="bet-card-leg-flag" />
                    <span className="bet-card-leg-code">{code}</span>
                    <span className="bet-card-leg-icon">{STATUS_LABEL[leg.status]}</span>
                  </div>
                )
              })}
            </div>
            <div className="bet-card-footer">
              {bet.stake != null && (
                <span className="bet-card-stake">£{bet.stake.toFixed(2)}</span>
              )}
              {bet.potential_return != null && (
                <span className="bet-card-return">→ £{bet.potential_return.toFixed(2)}</span>
              )}
              <span className={`bet-card-status ${BET_STATUS_CLASS[bet.status]}`}>
                {BET_STATUS_LABEL[bet.status]}
              </span>
            </div>
          </div>
        ))}
      </div>
    </section>
  )
}
