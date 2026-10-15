<script lang="ts">
  // Wake-lock glyph (a power symbol); amber when the lock is active.
  interface Props {
    active: boolean;
    size?: number;
  }

  let { active, size = 14 }: Props = $props();
</script>

<svg viewBox="0 0 16 16" width={size} height={size} aria-hidden="true" class="flex-none {active ? 'text-warn' : 'text-muted'}">
  <path d="M8 1.5v6M4.4 3.6a5.5 5.5 0 1 0 7.2 0" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
</svg>
