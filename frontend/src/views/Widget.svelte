<script lang="ts">
  // Windows corner widget. The window is a transparent Size×Size square whose outer corner sits on
  // the screen corner; the widget is the quarter disc of `radius` centred on that corner, clipped
  // with clip-path so nothing else paints. It shows two quarter rings, the 5-hour window outermost
  // and bold, the weekly one thinner inside, and the 5-hour percentage in the inner corner; every
  // limit (per-model ones too) and the reset times are in the tooltip.
  //
  // Unless "always show" is on, the widget is drawn only while Go reports it revealed (the cursor
  // was pushed into the screen corner, `widget:reveal` for this window). Go also owns
  // click-through, so DOM hover is never used.
  import { Events, Window } from "@wailsio/runtime";
  import { WidgetService, type WidgetGeometry } from "../../bindings/github.com/jacopofilonzi/ai-usage/internal/app";
  import { Corner } from "../../bindings/github.com/jacopofilonzi/ai-usage/internal/config";
  import { Kind, type Limit } from "../../bindings/github.com/jacopofilonzi/ai-usage/internal/provider";
  import Rings from "../components/Rings.svelte";
  import { accentFor } from "../lib/brand";
  import { level, levelClass, limitLabel, percent, resetText, statusText, updatedAgo, wakeShort } from "../lib/format";
  import { logError } from "../lib/log";
  import { app } from "../lib/state.svelte";

  // Mirrors the Go defaults (contract.go) until Geometry() answers, or if it returns zero values.
  const FALLBACK: WidgetGeometry = { corner: Corner.BottomRight, size: 72, radius: 64 };
  const WHEEL_THROTTLE_MS = 350;

  let geometry = $state<WidgetGeometry>(FALLBACK);
  let revealed = $state(false);
  let lastWheel = 0;

  const corner = $derived<Corner>(geometry.corner || app.settings?.config.corner || Corner.BottomRight);
  const top = $derived(corner === Corner.TopLeft || corner === Corner.TopRight);
  const left = $derived(corner === Corner.TopLeft || corner === Corner.BottomLeft);
  const size = $derived(geometry.size || FALLBACK.size);
  const radius = $derived(geometry.radius || FALLBACK.radius);
  const visible = $derived(Boolean(app.settings?.config.always_show) || revealed);

  // The 5-hour window leads: outermost ring and the percentage shown.
  const ordered = $derived.by((): Limit[] => {
    const session = app.limits.filter((l) => l.kind === Kind.KindSession);
    return [...session, ...app.limits.filter((l) => l.kind !== Kind.KindSession)];
  });
  const lead = $derived<Limit | undefined>(ordered[0]);
  const ringed = $derived(ordered.slice(0, 2));

  const tooltip = $derived.by(() => {
    const hasProviders = app.providers.length > 0;
    const lines = [hasProviders ? app.providerName : app.t("appName")];
    for (const l of ordered) lines.push(`${limitLabel(l, app.t)}  ${percent(l.usedPct)}  ·  ${resetText(l, app.now, app.t)}`);
    const status = statusText(app.snapshot, app.providerName, hasProviders, app.t);
    if (status) lines.push(status);
    if (app.stale) lines.push(updatedAgo(app.snapshot, app.now, app.t));
    const wake = wakeShort(app.wake, app.wakeAt, app.now, app.t);
    if (wake) lines.push(`${app.t("wakeTitle")}: ${wake}`);
    return lines.join("\n");
  });

  $effect(() => {
    let name: string | null = null;
    Window.Name()
      .then((n) => {
        name = n;
      })
      .catch((err: unknown) => logError("read window name", err));
    return Events.On("widget:reveal", (ev) => {
      if (name !== null && ev.data.window === name) revealed = ev.data.revealed;
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
  class:visible
  style:width="{size}px"
  style:height="{size}px"
  style:--r="{radius}px"
  style:--cx={left ? "0%" : "100%"}
  style:--cy={top ? "0%" : "100%"}
  style:--c-ok={accentFor(app.snapshot.provider)}
  title={tooltip}
  onwheel={onWheel}
  role="presentation"
>
  <Rings
    class="absolute {top ? 'top-0' : 'bottom-0'} {left ? 'left-0' : 'right-0'}"
    limits={ringed}
    stale={app.stale}
    mode="quarter"
    radius={radius - 4}
    thickness={7}
    minorThickness={3.5}
    gap={3}
    look="glow"
    --ring-transform="scale({left ? 1 : -1}, {top ? 1 : -1})"
  />
  <div
    class="lead"
    style:top={top ? "6px" : null}
    style:bottom={top ? null : "5px"}
    style:left={left ? "6px" : null}
    style:right={left ? null : "6px"}
    style:align-items={left ? "flex-start" : "flex-end"}
  >
    {#if top && lead}<span class="caption">{limitLabel(lead, app.t)}</span>{/if}
    <span class="pct tabular {levelClass(lead ? level(lead.usedPct, app.stale) : 'stale')}">
      {lead ? percent(lead.usedPct) : "–"}
    </span>
    {#if !top && lead}<span class="caption">{limitLabel(lead, app.t)}</span>{/if}
  </div>
  {#if app.wake.active}
    <span
      class="wake-dot"
      style:top={top ? "36px" : null}
      style:bottom={top ? null : "36px"}
      style:left={left ? "8px" : null}
      style:right={left ? null : "8px"}
    ></span>
  {/if}
</div>

<style>
  .widget {
    position: fixed;
    inset: 0;
    overflow: hidden;
    --custom-contextmenu: widget;
    --default-contextmenu: hide;
    /* Always dark, whatever the system theme: a cool translucent slate with a light rim. */
    --c-ok: #4ade80; /* replaced by the agent's accent (lib/brand.ts) */
    --c-warn: #fbbf24;
    --c-crit: #fb7185;
    --c-stale: #aab0ba;
    --c-track: rgb(255 255 255 / 0.1);
    color: #f2f4f7;
    font-family: "Segoe UI Variable Text", "Segoe UI", system-ui, sans-serif;
    background:
      radial-gradient(circle at var(--cx) var(--cy), transparent calc(var(--r) - 1.5px), rgb(255 255 255 / 0.35) calc(var(--r) - 1px)),
      radial-gradient(circle at var(--cx) var(--cy), rgb(30 41 59 / 0.8) 0, rgb(15 23 42 / 0.92) calc(var(--r) - 2px));
    clip-path: circle(var(--r) at var(--cx) var(--cy));
    transform-origin: var(--cx) var(--cy);
    opacity: 0;
    transform: scale(0.6);
    transition:
      opacity 140ms ease-out,
      transform 160ms ease-out;
  }

  .widget.visible {
    opacity: 1;
    transform: none;
  }

  .widget :global(svg) {
    transform: var(--ring-transform);
  }

  .lead {
    position: absolute;
    display: flex;
    flex-direction: column;
    line-height: 1;
  }

  .pct {
    font-size: 12px;
    font-weight: 700;
    letter-spacing: -0.02em;
  }

  .caption {
    font-size: 8.5px;
    font-weight: 600;
    color: rgb(255 255 255 / 0.55);
    margin: 1px 0;
  }

  .wake-dot {
    position: absolute;
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: #f5c35f;
    box-shadow: 0 0 4px #f5c35f;
  }
</style>
