import { useState, useEffect, useRef } from 'react'
import type { TournamentState } from '../types'

const POLL_INTERVAL_MS = 60_000

export function useStateData() {
  const [data, setData] = useState<TournamentState | null>(null)
  const [error, setError] = useState<string | null>(null)
  const lastUpdatedAt = useRef<string | null>(null)

  useEffect(() => {
    async function fetchState() {
      try {
        const res = await fetch(`/data/state.json?_=${Date.now()}`)
        if (!res.ok) throw new Error(`HTTP ${res.status}`)
        const json: TournamentState = await res.json()
        if (json.updated_at !== lastUpdatedAt.current) {
          lastUpdatedAt.current = json.updated_at
          setData(json)
          setError(null)
        }
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Failed to load data')
      }
    }

    fetchState()
    const id = setInterval(fetchState, POLL_INTERVAL_MS)
    return () => clearInterval(id)
  }, [])

  return { data, error }
}
