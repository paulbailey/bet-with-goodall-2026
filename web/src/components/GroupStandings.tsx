import { useState } from 'react'
import type { GroupState } from '../types'
import { getCountry } from '../countries'
import { Flag } from './Flag'

interface GroupStandingsProps {
  groups: Record<string, GroupState>
}

function GroupTable({ name, group }: { name: string; group: GroupState }) {
  const winner = group.complete && group.winner ? getCountry(group.winner) : null
  return (
    <div className="group-table">
      <h3 className="group-name">
        Group {name}
        {winner && (
          <span className="group-winner-badge"> — <Flag fi={winner.fi} className="flag-inline" /> {group.winner} ★</span>
        )}
      </h3>
      <table>
        <thead>
          <tr>
            <th className="col-team">Team</th>
            <th>P</th>
            <th>W</th>
            <th>D</th>
            <th>L</th>
            <th>GF</th>
            <th>GA</th>
            <th>GD</th>
            <th>Pts</th>
          </tr>
        </thead>
        <tbody>
          {group.standings.map((t, i) => (
              <tr key={t.team} className={i === 0 && !group.complete ? 'row-leader' : ''}>
                <td className="col-team">{t.team}</td>
                <td>{t.played}</td>
                <td>{t.won}</td>
                <td>{t.drawn}</td>
                <td>{t.lost}</td>
                <td>{t.gf}</td>
                <td>{t.ga}</td>
                <td>{t.gd > 0 ? `+${t.gd}` : t.gd}</td>
                <td className="col-pts">{t.points}</td>
              </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export function GroupStandings({ groups }: GroupStandingsProps) {
  const [open, setOpen] = useState(true)
  const sortedGroups = Object.entries(groups).sort(([a], [b]) => a.localeCompare(b))

  return (
    <section className="standings-section">
      <button className="section-toggle" onClick={() => setOpen((o) => !o)}>
        <h2 className="section-title">Group Standings</h2>
        <span className="toggle-icon">{open ? '▲' : '▼'}</span>
      </button>
      {open && (
        <div className="standings-grid">
          {sortedGroups.map(([name, group]) => (
            <GroupTable key={name} name={name} group={group} />
          ))}
        </div>
      )}
    </section>
  )
}
