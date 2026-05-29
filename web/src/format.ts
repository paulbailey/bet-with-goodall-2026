// Shared display formatters so money and probabilities read the same everywhere.

export function money(n: number): string {
  const sign = n < 0 ? '-' : ''
  return `${sign}£${Math.abs(n).toFixed(2)}`
}

// pct renders a 0–1 probability as a compact, readable percentage. Returns an
// em dash for an absent value (the builder omits probability when a bet can't
// be priced). We round to whole percent to keep the column narrow on mobile,
// but guard the extremes so a long shot doesn't collapse to "0%" and a near
// certainty doesn't round up to a misleading "100%".
export function pct(p: number | null | undefined): string {
  if (p == null) return '—'
  if (p <= 0) return '0%'
  if (p >= 1) return '100%'
  if (p < 0.01) return '<1%'
  if (p > 0.99) return '>99%'
  return `${Math.round(p * 100)}%`
}
