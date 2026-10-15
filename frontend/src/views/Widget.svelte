<script lang="ts">
  // Windows corner widget. The window is a transparent Size×Size square whose outer corner sits on
  // the screen corner; everything is drawn inside a quarter disc centred on that corner, clipped
  // with clip-path so nothing else paints. Collapsed it shows one quarter ring per limit; expanded
  // (on the Go `widget:hover` event for this window, never on DOM hover, because Go owns
  // click-through) it shows the details.
  //
  // Expanded layout: the details are stacked in slots whose height is fixed, so each slot's width
  // can be computed from the circle at its edge farthest from the screen edge. Slots nearer the
  // horizontal screen edge are wider, giving the "fan" shape; text in a slot ellipsizes or clamps.
  import { Events, Window } from "@wailsio/runtime";
  import { WidgetService, type WidgetGeometry } from "../../bindings/github.com/jacopofilonzi/ai-usage/internal/app";
  import { Corner } from "../../bindings/github.com/jacopofilonzi/ai-usage/internal/config";
  import Rings from "../components/Rings.svelte";
  import WakeIcon from "../components/WakeIcon.svelte";
  import { level, levelClass, limitLabel, percent, resetShort, statusText, updatedAgo, wakeShort } from "../lib/format";
  import { logError } from "../lib/log";
  import { app } from "../lib/state.svelte";

  // Mirrors the Go defaults (contract.go) until Geometry() answers, or if it returns zero values.
  const FALLBACK: WidgetGeometry = { corner: Corner.BottomRight, size: 200, collapsedRadius: 64, expandedRadius: 196 };
  const WHEEL_THROTTLE_MS = 350;
  const PAD = 14; // from both screen edges
  const INSET = 8; // from the disc's arc
  const GAP = 6;
  const TITLE_H = 18;
  const LIMIT_H = 17;
  const LINE_H = 13;
  const WAKE_H = 16;

  let geometry = $state<WidgetGeometry>(FALLBACK);
  let windowName = $state<string | null>(null);
  let expanded = $state(false);
  let lastWheel = 0;

  const corner = $derived<Corner>(geometry.corner || app.settings?.config.corner || Corner.BottomRight);
  const top = $derived(corner === Corner.TopLeft || corner === Corner.TopRight);
  const left = $derived(corner === Corner.TopLeft || corner === Corner.BottomLeft);
  const size = $derived(geometry.size || FALLBACK.size);
  const collapsedR = $derived(geometry.collapsedRadius || FALLBACK.collapsedRadius);
  const expandedR = $derived(geometry.expandedRadius || FALLBACK.expandedRadius);
  const radius = $derived(expanded ? expandedR : collapsedR);

  const hasProviders = $derived(app.providers.length > 0);
  const status = $derived(statusText(app.snapshot, app.providerName, hasProviders, app.t));
  const updated = $derived(app.stale ? updatedAgo(app.snapshot, app.now, app.t) : "");
  const wake = $derived(wakeShort(app.wake, app.wakeAt, app.now, app.t));
  const title = $derived(hasProviders ? app.providerName : app.t("appName"));
  const main = $derived(app.limits.length ? Math.max(...app.limits.map((l) => l.usedPct)) : null);

  // Collapsed rings: as many as fit in the collapsed radius, leaving the inner corner for the %.
  const ringRadius = $derived(collapsedR - 5);

  type SlotKind = "title" | "limits" | "status" | "wake";
  interface Slot {
    kind: SlotKind;
    near: number; // distance of the slot's edge nearest the horizontal screen edge
    width: number;
  }

  const slots = $derived.by((): Slot[] => {
    const statusLines = (status ? 2 : 0) + (updated ? 1 : 0);
    const heights: Record<SlotKind, number> = {
      title: TITLE_H,
      limits: app.limits.length * LIMIT_H,
      status: statusLines * LINE_H,
      wake: wake ? WAKE_H : 0,
    };
    // Reading order, top to bottom. The limits table needs the most width, so it goes next to the
    // horizontal screen edge: right after the title at the top, right before the wake chip at the
    // bottom.
    const order: SlotKind[] = top ? ["title", "limits", "status", "wake"] : ["title", "status", "limits", "wake"];
    const list = order.filter((k) => heights[k] > 0);

    const r = expandedR - INSET;
    const fromEdge = top ? list : [...list].reverse();
    let y = PAD;
    const out: Slot[] = [];
    for (const kind of fromEdge) {
      const far = y + heights[kind];
      const width = far < r ? Math.floor(Math.sqrt(r * r - far * far)) - PAD : 0;
      out.push({ kind, near: y, width: Math.max(0, width) });
      y = far + GAP;
    }
    return top ? out : out.reverse();
  });

  $effect(() => {
    let name: string | null = null;
    Window.Name()
      .then((n) => {
        name = n;
        windowName = n;
      })
      .catch((err: unknown) => logError("read window name", err));
    return Events.On("widget:hover", (ev) => {
      if (name !== null && ev.data.window === name) expanded = ev.data.hover;
    });
  });

  // Reload the geometry when the corner setting changes (Go may move this window in place).
  $effect(() => {
    void app.settings?.config.corner;
    WidgetService.Geometry()
      .then((g) => {
        geometry = g;
      })
      .catch((err: unknown) => logError("load widget geometry", err));
  });

  function onWheel(e: WheelEvent) {
    if (e.deltaY === 0 || e.timeStamp - lastWheel < WHEEL_THROTTLE_MS) return;
    lastWheel = e.timeStamp;
    void app.cycleProvider(e.deltaY > 0 ? 1 : -1);
  }
</script>

<div
  class="widget"
  class:expanded
  data-window={windowName}
  style:width="{size}px"
  style:height="{size}px"
  style:--r="{radius}px"
  style:--cx={left ? "0%" : "100%"}
  style:--cy={top ? "0%" : "100%"}
  onwheel={onWheel}
  role="presentation"
>
  <div class="collapsed" aria-hidden={expanded}>
    <Rings
      class="absolute {top ? 'top-0' : 'bottom-0'} {left ? 'left-0' : 'right-0'}"
      limits={app.limits}
      stale={app.stale}
      mode="quarter"
      radius={ringRadius}
      thickness={6}
      gap={2}
      --ring-transform="scale({left ? 1 : -1}, {top ? 1 : -1})"
    />
    <span
      class="main tabular {levelClass(main === null ? 'stale' : level(main, app.stale))}"
      style:top={top ? "7px" : null}
      style:bottom={top ? null : "7px"}
      style:left={left ? "7px" : null}
      style:right={left ? null : "7px"}
    >
      {main === null ? "–" : percent(main)}
    </span>
  </div>

  <div class="details" aria-hidden={!expanded}>
    {#each slots as slot (slot.kind)}
      <div
        class="slot"
        style:top={top ? `${slot.near}px` : null}
        style:bottom={top ? null : `${slot.near}px`}
        style:left={left ? `${PAD}px` : null}
        style:right={left ? null : `${PAD}px`}
        style:width="{slot.width}px"
        style:text-align={left ? "left" : "right"}
      >
        {#if slot.kind === "title"}
          <div class="title truncate">{title}</div>
        {:else if slot.kind === "wake"}
          <span class="wake tabular" title={app.t("wakeTitle")}>
            <WakeIcon active size={10} />
            {wake}
          </span>
        {:else if slot.kind === "limits"}
          <div class="limits">
            {#each app.limits as limit, i (limit.id + i)}
              <span class="label truncate" title={limitLabel(limit, app.t)}>{limitLabel(limit, app.t)}</span>
              <span class="reset tabular">{resetShort(limit, app.now, app.t)}</span>
              <span class="pct tabular {levelClass(level(limit.usedPct, app.stale))}">{percent(limit.usedPct)}</span>
            {/each}
          </div>
        {:else}
          {#if status}
            <p class="status" class:warn={app.stale} title={app.snapshot.error ?? ""}>{status}</p>
          {/if}
          {#if updated}
            <p class="updated">{updated}</p>
          {/if}
        {/if}
      </div>
    {/each}
  </div>
</div>

<style>
  /* Animatable radius: clip-path and the backdrop gradient both read it. */
  @property --r {
    syntax: "<length>";
    inherits: true;
    initial-value: 64px;
  }

  .widget {
    position: fixed;
    inset: 0;
    overflow: hidden;
    --custom-contextmenu: widget;
    --default-contextmenu: hide;
    /* The backdrop is always dark, whatever the system theme. */
    --c-ok: #3ccf7a;
    --c-warn: #f5b335;
    --c-crit: #ff5d63;
    --c-stale: #aab0ba;
    --c-track: rgb(255 255 255 / 0.13);
    color: #f2f4f7;
    clip-path: circle(var(--r) at var(--cx) var(--cy));
    background: radial-gradient(
      circle at var(--cx) var(--cy),
      rgb(18 20 26 / 0.86) calc(var(--r) - 1.5px),
      rgb(255 255 255 / 0.22) calc(var(--r) - 1px)
    );
    transition: --r 150ms ease-out;
  }

  .collapsed,
  .details {
    position: absolute;
    inset: 0;
    transition: opacity 120ms ease-out;
  }

  .collapsed :global(svg) {
    transform: var(--ring-transform);
  }

  .details {
    opacity: 0;
    pointer-events: none;
  }

  .expanded .collapsed {
    opacity: 0;
  }

  .expanded .details {
    opacity: 1;
    transition-delay: 60ms;
  }

  .main {
    position: absolute;
    font-size: 11px;
    font-weight: 700;
    line-height: 11px;
    letter-spacing: -0.02em;
  }

  .slot {
    position: absolute;
    overflow: hidden;
  }

  .title {
    height: 18px;
    font-size: 12.5px;
    font-weight: 600;
    line-height: 18px;
  }

  .wake {
    flex: none;
    display: inline-flex;
    align-items: center;
    gap: 3px;
    padding: 0 5px;
    height: 15px;
    border-radius: 8px;
    background: rgb(245 179 53 / 0.18);
    color: #f5c35f;
    font-size: 10px;
    font-weight: 600;
    line-height: 15px;
  }

  .limits {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto auto;
    column-gap: 7px;
    align-items: baseline;
  }

  .limits > span {
    height: 17px;
    line-height: 17px;
  }

  .label {
    font-size: 11px;
    color: #b4bac4;
    text-align: left;
  }

  .reset {
    font-size: 10px;
    color: #8f95a0;
  }

  .pct {
    min-width: 28px;
    font-size: 12px;
    font-weight: 700;
    text-align: right;
  }

  .status {
    margin: 0;
    font-size: 11px;
    line-height: 13px;
    color: #c3c8d0;
    display: -webkit-box;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    overflow: hidden;
  }

  .status.warn {
    color: #f5c35f;
  }

  .updated {
    margin: 0;
    font-size: 10px;
    line-height: 13px;
    color: #8f95a0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>
