interface HeaderProps {
  updatedAt: string | null
  phase: string | null
}

const PHASE_LABELS: Record<string, string> = {
  pre_tournament: 'Pre-Tournament',
  group_stage: 'Group Stage',
  knockout: 'Knockout Rounds',
  complete: 'Tournament Complete',
}

function formatTimestamp(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  })
}

export function Header({ updatedAt, phase }: HeaderProps) {
  return (
    <header className="site-header">
      <div className="header-inner">
        <div className="header-brand">
          <div className="header-trophy">⚽</div>
          <div>
            <h1 className="header-title">Bet With Goodall</h1>
            <p className="header-subtitle">FIFA World Cup 2026</p>
          </div>
        </div>
        <div className="header-meta">
          {phase && (
            <span className="phase-badge">{PHASE_LABELS[phase] ?? phase}</span>
          )}
          {updatedAt && (
            <span className="updated-at">
              Updated {formatTimestamp(updatedAt)}
            </span>
          )}
        </div>
      </div>
    </header>
  )
}
