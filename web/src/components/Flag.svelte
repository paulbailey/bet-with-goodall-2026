<script lang="ts">
  interface Props {
    fi: string
    class?: string
  }

  let { fi, class: className = '' }: Props = $props()
</script>

{#if fi}
  <span class="fi fi-{fi}{className ? ` ${className}` : ''}"></span>
{/if}
