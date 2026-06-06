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
// We show two significant figures across the whole range so precision stays
// consistent: longshots keep their decimals ("0.42%", "0.082%") and likelier
// bets read as you'd expect ("1.3%", "7.3%", "43%"). Previously anything at 1%
// and up was rounded to a whole percent, which made the most likely bets read
// coarser than the sub-1% longshots beneath them — a 1.3% chance collapsed to
// "1%" right above a precise "0.63%". The extremes are guarded so a longshot
// doesn't collapse to "0%" nor a near certainty round up to "100%".
export function pct(p: number | null | undefined): string {
  if (p == null) return '—'
  if (p <= 0) return '0%'
  if (p >= 1) return '100%'
  if (p > 0.99) return '>99%'
  const percent = p * 100
  if (percent < 0.0001) return '<0.0001%'
  // toPrecision then Number() trims trailing zeros: 0.42 -> "0.42", 0.4 -> "0.4".
  return `${Number(percent.toPrecision(2))}%`
}
