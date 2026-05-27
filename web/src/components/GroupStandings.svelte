<script lang="ts">
  import type { GroupState } from '../types'
  import { getCountry } from '../countries'
  import Flag from './Flag.svelte'

  interface Props {
    groups: Record<string, GroupState>
  }

  let { groups }: Props = $props()
  let open = $state(true)

  let sortedGroups = $derived(
    Object.entries(groups).sort(([a], [b]) => a.localeCompare(b))
  )
</script>

<section class="standings-section">
  <button class="section-toggle" onclick={() => (open = !open)}>
    <h2 class="section-title">Group Standings</h2>
    <span class="toggle-icon">{open ? '▲' : '▼'}</span>
  </button>
  {#if open}
    <div class="standings-grid">
      {#each sortedGroups as [name, group]}
        {@const winner = group.complete && group.winner ? getCountry(group.winner) : null}
        <div class="group-table">
          <h3 class="group-name">
            Group {name}
            {#if winner}
              <span class="group-winner-badge">
                — <Flag fi={winner.fi} class="flag-inline" /> {group.winner} ★
              </span>
            {/if}
          </h3>
          <table>
            <thead>
              <tr>
                <th class="col-team">Team</th>
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
              {#each group.standings as t, i}
                <tr class={i === 0 && !group.complete ? 'row-leader' : ''}>
                  <td class="col-team">{t.team}</td>
                  <td>{t.played}</td>
                  <td>{t.won}</td>
                  <td>{t.drawn}</td>
                  <td>{t.lost}</td>
                  <td>{t.gf}</td>
                  <td>{t.ga}</td>
                  <td>{t.gd > 0 ? `+${t.gd}` : t.gd}</td>
                  <td class="col-pts">{t.points}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/each}
    </div>
  {/if}
</section>
