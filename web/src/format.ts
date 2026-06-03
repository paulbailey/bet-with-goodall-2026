// Shared display formatters so money and probabilities read the same everywhere.

const MONEY_WITH_PENCE = new Intl.NumberFormat('en-GB', {
  style: 'currency',
  currency: 'GBP',
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})

const MONEY_WITHOUT_PENCE = new Intl.NumberFormat('en-GB', {
  style: 'currency',
  currency: 'GBP',
  minimumFractionDigits: 0,
  maximumFractionDigits: 0,
})

export function money(n: number): string {
  return Math.abs(n) >= 1000 ? MONEY_WITHOUT_PENCE.format(n) : MONEY_WITH_PENCE.format(n)
}

// pct renders a 0–1 probability as a readable percentage. Returns an em dash for
// an absent value (the builder omits probability when a bet can't be priced).
// Chances of 1% and up round to a whole percent. Below 1% — where a 12-leg acca
// lives, and where rounding to whole percent would flatten every bet to "0%" and
// hide the ranking — we show two significant figures (e.g. "0.42%", "0.017%") so
// near-identical longshots stay distinguishable. The extremes are guarded so a
// long shot doesn't collapse to "0%" nor a near certainty round up to "100%".
export function pct(p: number | null | undefined): string {
  if (p == null) return '—'
  if (p <= 0) return '0%'
  if (p >= 1) return '100%'
  if (p > 0.99) return '>99%'
  const percent = p * 100
  if (percent >= 1) return `${Math.round(percent)}%`
  if (percent < 0.0001) return '<0.0001%'
  // toPrecision then Number() trims trailing zeros: 0.42 -> "0.42", 0.4 -> "0.4".
  return `${Number(percent.toPrecision(2))}%`
}
