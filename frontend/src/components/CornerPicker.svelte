<script lang="ts">
  // Widget corner picker: a small screen diagram with one button per corner; the selected corner
  // shows the widget's quarter disc.
  import { Corner } from "../../bindings/github.com/jacopofilonzi/ai-usage/internal/config";
  import { app } from "../lib/state.svelte";
  import type { Key } from "../lib/i18n";

  interface Props {
    value: Corner;
    onchange: (corner: Corner) => void;
  }

  let { value, onchange }: Props = $props();

  const corners: { corner: Corner; label: Key; pos: string }[] = [
    { corner: Corner.TopLeft, label: "cornerTopLeft", pos: "top-0 left-0" },
    { corner: Corner.TopRight, label: "cornerTopRight", pos: "top-0 right-0" },
    { corner: Corner.BottomLeft, label: "cornerBottomLeft", pos: "bottom-0 left-0" },
    { corner: Corner.BottomRight, label: "cornerBottomRight", pos: "bottom-0 right-0" },
  ];
</script>

<div class="flex flex-none flex-col items-center gap-1">
  <div class="screen" role="radiogroup" aria-label={app.t("corner")}>
    {#each corners as c (c.corner)}
      <button
        type="button"
        role="radio"
        aria-checked={value === c.corner}
        aria-label={app.t(c.label)}
        title={app.t(c.label)}
        class="corner absolute {c.pos}"
        data-corner={c.corner}
        onclick={() => onchange(c.corner)}
      >
        <span class="disc"></span>
      </button>
    {/each}
  </div>
  <span class="stand"></span>
</div>

<style>
  .screen {
    position: relative;
    width: 8.5rem;
    height: 5.25rem;
    border-radius: 8px;
    border: 2px solid var(--c-border);
    background: linear-gradient(135deg, var(--c-surface-2), var(--c-bg));
    overflow: hidden;
  }

  .stand {
    width: 2.25rem;
    height: 0.3rem;
    border-radius: 0 0 4px 4px;
    background: var(--c-border);
  }

  .corner {
    width: 2.75rem;
    height: 2.75rem;
  }

  .corner:focus-visible {
    outline: 2px solid var(--c-accent);
    outline-offset: -2px;
  }

  .disc {
    position: absolute;
    width: 1.25rem;
    height: 1.25rem;
    background: var(--c-muted);
    opacity: 0.25;
    transition:
      opacity 120ms,
      background-color 120ms,
      width 150ms,
      height 150ms;
  }

  .corner:hover .disc {
    opacity: 0.55;
  }

  .corner[aria-checked="true"] .disc {
    width: 1.875rem;
    height: 1.875rem;
    background: var(--c-accent);
    opacity: 1;
  }

  [data-corner="top-left"] .disc {
    top: 0;
    left: 0;
    border-bottom-right-radius: 100%;
  }

  [data-corner="top-right"] .disc {
    top: 0;
    right: 0;
    border-bottom-left-radius: 100%;
  }

  [data-corner="bottom-left"] .disc {
    bottom: 0;
    left: 0;
    border-top-right-radius: 100%;
  }

  [data-corner="bottom-right"] .disc {
    bottom: 0;
    right: 0;
    border-top-left-radius: 100%;
  }
</style>
