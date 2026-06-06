// Shared formatting for the daily summary movers, used by the home-page card
// and the archive page.

import type { SummaryMover } from './types'

// formatDay renders a YYYY-MM-DD tournament day as a friendly date. The explicit
// T00:00:00 keeps it on the intended calendar day rather than shifting across
// UTC midnight in western timezones.
export function formatDay(date: string): string {
  return new Date(`${date}T00:00:00`).toLocaleDateString(undefined, {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
  })
}

// deltaLabel is the absolute move in percentage points, signed.
export function deltaLabel(m: SummaryMover): string {
  const pp = m.delta_pp * 100
  const sign = pp >= 0 ? '+' : '−'
  return `${sign}${Number(Math.abs(pp).toPrecision(2))}pp`
}

// ratioLabel is the relative move. A bet that went bust has ratio 0; an
// unmeasurable jump from zero comes through as non-finite.
export function ratioLabel(m: SummaryMover): string {
  if (!isFinite(m.ratio)) return 'new'
  if (m.ratio <= 0) return 'bust'
  return `${Number(m.ratio.toPrecision(2))}×`
}
