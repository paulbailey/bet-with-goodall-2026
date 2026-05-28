// Shared display formatters so money and probabilities read the same everywhere.

export function money(n: number): string {
  const sign = n < 0 ? '-' : ''
  return `${sign}£${Math.abs(n).toFixed(2)}`
}

// pct renders a 0–1 probability as a percentage. Returns an em dash for an
// absent value (the builder omits probability when a bet can't be priced).
export function pct(p: number | null | undefined): string {
  if (p == null) return '—'
  return `${(p * 100).toFixed(1)}%`
}
