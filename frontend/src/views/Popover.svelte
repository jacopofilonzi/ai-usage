<script lang="ts">
  // macOS/Linux tray popover (~320×260): provider switcher, rings and per-limit details, status,
  // wake-lock quick actions and a shortcut to the settings window.
  import { SettingsService } from "../../bindings/github.com/jacopofilonzi/ai-usage/internal/app";
  import Rings from "../components/Rings.svelte";
  import WakeIcon from "../components/WakeIcon.svelte";
  import { level, levelClass, limitLabel, percent, resetText, statusText, updatedAgo, wakeText } from "../lib/format";
  import { logError } from "../lib/log";
  import { app } from "../lib/state.svelte";

  const hasProviders = $derived(app.providers.length > 0);
  const activeId = $derived(app.snapshot.provider || app.providers.find((p) => p.active)?.id || "");
  const status = $derived(statusText(app.snapshot, app.providerName, hasProviders, app.t));
  const updated = $derived(app.stale ? updatedAgo(app.snapshot, app.now, app.t) : "");
  // With a status message there is no room for the reset lines: they move to the row tooltip.
  const compact = $derived(status !== "" || updated !== "");

  function openSettings() {
    SettingsService.Open().catch((err: unknown) => logError("open settings", err));
  }
</script>

<main class="flex h-screen flex-col gap-3 overflow-hidden bg-bg p-3 text-fg">
  <header class="flex items-center gap-2">
    {#if app.providers.length > 1}
      <div class="segmented min-w-0 flex-1" role="group" aria-label={app.t("switchProvider")}>
        {#each app.providers as p (p.id)}
          <button type="button" aria-pressed={p.id === activeId} onclick={() => app.setProvider(p.id)} title={p.name}>
            {p.name}
          </button>
        {/each}
      </div>
    {:else}
      <h1 class="min-w-0 flex-1 truncate text-sm font-semibold">
        {hasProviders ? app.providerName : app.t("appName")}
      </h1>
    {/if}
    <button type="button" class="icon-btn flex-none" onclick={openSettings} title={app.t("openSettings")} aria-label={app.t("openSettings")}>
      <svg viewBox="0 0 20 20" width="16" height="16" aria-hidden="true">
        <path
          d="M8.6 2h2.8l.4 2.2 1.5.9 2.1-.8 1.4 2.4-1.7 1.5v1.6l1.7 1.5-1.4 2.4-2.1-.8-1.5.9-.4 2.2H8.6l-.4-2.2-1.5-.9-2.1.8-1.4-2.4 1.7-1.5V8.2L3.2 6.7l1.4-2.4 2.1.8 1.5-.9z"
          fill="none"
          stroke="currentColor"
          stroke-width="1.4"
          stroke-linejoin="round"
        />
        <circle cx="10" cy="10" r="2.4" fill="none" stroke="currentColor" stroke-width="1.4" />
      </svg>
    </button>
  </header>

  <section class="flex min-h-0 flex-1 items-center gap-4">
    <Rings class="flex-none" limits={app.limits} stale={app.stale} mode="full" radius={46} thickness={8} gap={3} />
    <div class="flex min-w-0 flex-1 flex-col gap-1.5">
      {#each app.limits as limit, i (limit.id + i)}
        <div class="min-w-0" title={compact ? resetText(limit, app.now, app.t) : undefined}>
          <div class="flex items-baseline justify-between gap-2">
            <span class="truncate text-xs text-muted">{limitLabel(limit, app.t)}</span>
            <span class="tabular text-sm font-bold {levelClass(level(limit.usedPct, app.stale))}">{percent(limit.usedPct)}</span>
          </div>
          {#if !compact}
            <div class="truncate text-[11px] text-muted tabular">{resetText(limit, app.now, app.t)}</div>
          {/if}
        </div>
      {/each}
      {#if status}
        <p class="line-clamp-2 text-xs {app.stale ? 'text-warn' : 'text-muted'}" title={app.snapshot.error ?? ""}>{status}</p>
      {/if}
      {#if updated}
        <p class="truncate text-[11px] text-muted">{updated}</p>
      {/if}
    </div>
  </section>

  <footer class="card flex flex-col gap-2 px-3 py-2">
    <div class="flex items-center gap-1.5 text-xs" class:text-muted={!app.wake.active}>
      <WakeIcon active={app.wake.active} />
      <span class="truncate tabular">{wakeText(app.wake, app.wakeAt, app.now, app.t)}</span>
    </div>
    <div class="flex gap-1.5">
      <button type="button" class="btn flex-1" onclick={() => app.startWake(30)}>{app.t("wake30m")}</button>
      <button type="button" class="btn flex-1" onclick={() => app.startWake(60)}>{app.t("wake1h")}</button>
      <button
        type="button"
        class="btn flex-1"
        class:btn-active={app.wake.active && app.wake.indefinite}
        onclick={() => app.startWake(0)}
        title={app.t("wakeIndefinitely")}>{app.t("wakeForever")}</button
      >
      <button type="button" class="btn btn-danger flex-1" disabled={!app.wake.active} onclick={() => app.stopWake()}>
        {app.t("wakeStop")}
      </button>
    </div>
  </footer>
</main>
