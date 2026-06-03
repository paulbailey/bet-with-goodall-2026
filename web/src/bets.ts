// Shared logic for the group-winner accumulators, used by both the desktop
// table (BetGrid) and the mobile collapsed list (BetCardList / BetCardRow).

import type { Bet, BetLeg, LegStatus, BetStatus } from './types'

export const STATUS_CLASS: Record<LegStatus, string> = {
  alive: 'leg-alive',
  won:   'leg-won',
  lost:  'leg-lost',
}

export const STATUS_LABEL: Record<LegStatus, string> = {
  alive: '✓',
  won:   '★',
  lost:  '✗',
}

export const BET_STATUS_CLASS: Record<BetStatus, string> = {
  alive: 'bet-alive',
  won:   'bet-won',
  lost:  'bet-lost',
}

export const BET_STATUS_LABEL: Record<BetStatus, string> = {
  alive: 'Alive',
  won:   'Won!',
  lost:  'Bust',
}

// Every accumulator is a variation on the same favourites: the pick chosen by
// the most bets in each group is the "favourite", and the leg(s) where a bet
// departs from it are what make that bet unique. We surface those as a row
// label and highlight the cell so 40-odd near-identical rows can be told apart.
export function computeFavByGroup(bets: Bet[]): Record<string, string> {
  const counts: Record<string, Record<string, number>> = {}
  for (const b of bets) {
    for (const l of b.legs) {
      counts[l.group] ??= {}
      counts[l.group][l.team] = (counts[l.group][l.team] ?? 0) + 1
    }
  }
  const fav: Record<string, string> = {}
  for (const [g, teams] of Object.entries(counts)) {
    fav[g] = Object.entries(teams).sort((a, b) => b[1] - a[1])[0][0]
  }
  return fav
}

export function isPick(leg: BetLeg, favByGroup: Record<string, string>): boolean {
  return favByGroup[leg.group] != null && leg.team !== favByGroup[leg.group]
}

export function betLabel(bet: Bet, favByGroup: Record<string, string>): string {
  const dev = deviations(bet, favByGroup)
  if (dev.length === 0) return 'Favourites'
  return dev.map((l) => `${l.group}: ${l.team}`).join(', ')
}

// The legs where this bet swaps away from the favourites — its distinguishing picks.
export function deviations(bet: Bet, favByGroup: Record<string, string>): BetLeg[] {
  return bet.legs.filter((l) => isPick(l, favByGroup))
}

// The legs that have killed this bet. The single most useful thing to surface
// for a bust bet — its label may just be "Favourites", which says nothing about
// which leg actually died.
export function lostLegs(bet: Bet): BetLeg[] {
  return bet.legs.filter((l) => l.status === 'lost')
}

const STATUS_RANK: Record<BetStatus, number> = { alive: 0, won: 1, lost: 2 }

// Display order shared by the desktop table and the mobile list: live bets
// first (most likely leading), then won, then bust. Within the alive band we
// sort by descending probability; an unpriced live bet (no probability) sinks
// to the back of that band. Won/bust bands keep their original order — their
// chances are all ~1 or 0. Returns a new array; the input is left untouched.
export function sortBetsForDisplay(bets: Bet[]): Bet[] {
  return [...bets].sort((a, b) => {
    if (STATUS_RANK[a.status] !== STATUS_RANK[b.status]) {
      return STATUS_RANK[a.status] - STATUS_RANK[b.status]
    }
    if (a.status === 'alive') return (b.probability ?? -1) - (a.probability ?? -1)
    return 0
  })
}
