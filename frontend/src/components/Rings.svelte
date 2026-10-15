<script lang="ts">
  // Concentric usage rings, one per limit (outermost = first limit), coloured by threshold.
  //
  // mode "full": whole circles in a (2·radius)² box, filling clockwise from 12 o'clock.
  // mode "quarter": quarter arcs centred on the box's top-left corner, in a radius² box, filling
  // from the top edge towards the left edge. The widget mirrors the whole SVG with a CSS
  // transform to anchor it at any screen corner.
  import type { Limit } from "../../bindings/github.com/jacopofilonzi/ai-usage/internal/provider";
  import { level, levelColor } from "../lib/format";

  interface Props {
    limits: Limit[];
    stale: boolean;
    mode: "full" | "quarter";
    /** Outer edge of the outermost ring. */
    radius: number;
    thickness?: number;
    gap?: number;
    class?: string;
  }

  let { limits, stale, mode, radius, thickness = 8, gap = 3, class: className = "" }: Props = $props();

  const rings = $derived(
    limits.slice(0, 3).map((l, i) => ({
      key: `${i}:${l.id}`,
      r: radius - thickness / 2 - i * (thickness + gap),
      pct: Math.min(100, Math.max(0, l.usedPct)),
      color: levelColor(level(l.usedPct, stale)),
    })),
  );
  const box = $derived(mode === "full" ? radius * 2 : radius);
  const c = $derived(mode === "full" ? radius : 0);

  function quarterArc(r: number): string {
    return `M ${r} 0 A ${r} ${r} 0 0 1 0 ${r}`;
  }
</script>

<svg
  class={className}
  width={box}
  height={box}
  viewBox="0 0 {box} {box}"
  aria-hidden="true"
  overflow="hidden"
>
  {#each rings as ring (ring.key)}
    {#if mode === "full"}
      <g transform="rotate(-90 {c} {c})">
        <circle cx={c} cy={c} r={ring.r} fill="none" style:stroke="var(--ring-track, var(--c-track))" stroke-width={thickness} />
        <circle
          class="value"
          cx={c}
          cy={c}
          r={ring.r}
          fill="none"
          style:stroke={ring.color}
          stroke-width={thickness}
          stroke-linecap="round"
          pathLength="100"
          stroke-dasharray="{ring.pct} 100"
          opacity={ring.pct > 0 ? 1 : 0}
        />
      </g>
    {:else}
      <path d={quarterArc(ring.r)} fill="none" style:stroke="var(--ring-track, var(--c-track))" stroke-width={thickness} />
      <path
        class="value"
        d={quarterArc(ring.r)}
        fill="none"
        style:stroke={ring.color}
        stroke-width={thickness}
        pathLength="100"
        stroke-dasharray="{ring.pct} 100"
        opacity={ring.pct > 0 ? 1 : 0}
      />
    {/if}
  {/each}
</svg>

<style>
  .value {
    transition:
      stroke-dasharray 500ms ease-out,
      stroke 300ms;
  }
</style>
