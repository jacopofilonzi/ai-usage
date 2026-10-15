<script lang="ts">
  // Settings window (~520×680). Every control applies immediately: config fields go through
  // SettingsService.Save with the full updated config (app.saveConfig); autostart and keep-unlocked
  // have their own service calls.
  import { SettingsService } from "../../bindings/github.com/jacopofilonzi/ai-usage/internal/app";
  import { Corner } from "../../bindings/github.com/jacopofilonzi/ai-usage/internal/config";
  import CornerPicker from "../components/CornerPicker.svelte";
  import Toggle from "../components/Toggle.svelte";
  import WakeIcon from "../components/WakeIcon.svelte";
  import { wakeText } from "../lib/format";
  import type { Key } from "../lib/i18n";
  import { logError } from "../lib/log";
  import { app } from "../lib/state.svelte";

  const REFRESH_SECONDS = [0, 60, 120, 300, 600];
  const WAKE_MINUTES = [5, 10, 30, 60, 120];
  const LANGUAGES: { value: string; label: Key }[] = [
    { value: "system", label: "languageSystem" },
    { value: "en", label: "languageEnglish" },
    { value: "it", label: "languageItalian" },
  ];

  const config = $derived(app.settings?.config);
  const isWindows = $derived(app.settings?.platform === "windows");
  const providerValue = $derived(config?.provider || app.providers.find((p) => p.active)?.id || "");
  const refreshOptions = $derived(
    config && !REFRESH_SECONDS.includes(config.poll_seconds) ? [...REFRESH_SECONDS, config.poll_seconds] : REFRESH_SECONDS,
  );

  $effect(() => {
    document.title = app.t("settingsTitle");
  });

  function refreshLabel(seconds: number): string {
    if (seconds === 0) return app.t("refreshDefault");
    return app.t("refreshMinutes", { n: Math.round((seconds / 60) * 10) / 10 });
  }

  function wakeLabel(minutes: number): string {
    return minutes % 60 === 0 ? app.t("wakeHours", { n: minutes / 60 }) : app.t("wakeMinutes", { n: minutes });
  }

  function selectValue(e: Event): string {
    return (e.currentTarget as HTMLSelectElement).value;
  }

  function onProvider(e: Event) {
    const id = selectValue(e);
    // SetProvider persists the choice on the Go side; mirror it locally so the select doesn't jump.
    if (app.settings) app.settings.config.provider = id;
    void app.setProvider(id);
  }

  function quit() {
    SettingsService.Quit().catch((err: unknown) => logError("quit", err));
  }
</script>

<main class="min-h-screen overflow-x-hidden bg-bg px-5 pt-5 pb-6 text-fg">
  <h1 class="mb-4 text-lg font-semibold">{app.t("settingsTitle")}</h1>

  {#if app.saveError}
    <p class="card mb-4 border-danger! px-4 py-2 text-sm text-danger" role="alert">{app.t("saveFailed")}</p>
  {/if}

  <section class="mb-5">
    <h2 class="section-title mb-2 px-1">{app.t("sectionUsage")}</h2>
    <div class="card">
      <div class="setting-row">
        <div class="min-w-0">
          <div class="text-sm font-medium">{app.t("provider")}</div>
          <div class="text-xs text-muted">{app.providers.length ? app.t("providerHint") : app.t("statusNoProviders")}</div>
        </div>
        {#if app.providers.length}
          <select class="field w-44" value={providerValue} onchange={onProvider} aria-label={app.t("provider")}>
            {#each app.providers as p (p.id)}
              <option value={p.id}>{p.name}</option>
            {/each}
          </select>
        {/if}
      </div>
      {#if !app.providers.length}
        <p class="px-4 pb-3 text-xs text-muted">{app.t("noProvidersHint")}</p>
      {/if}

      {#if isWindows && config}
        <div class="setting-row">
          <div class="min-w-0">
            <div class="text-sm font-medium">{app.t("corner")}</div>
            <div class="text-xs text-muted">{app.t("cornerHint")}</div>
          </div>
          <CornerPicker value={config.corner || Corner.BottomRight} onchange={(corner) => app.saveConfig({ corner })} />
        </div>

        <div class="setting-row">
          <div class="min-w-0">
            <div class="text-sm font-medium">{app.t("alwaysShow")}</div>
            <div class="text-xs text-muted">{app.t("alwaysShowHint")}</div>
          </div>
          <Toggle
            checked={config.always_show}
            label={app.t("alwaysShow")}
            onchange={(always_show) => app.saveConfig({ always_show })}
          />
        </div>

        <div class="setting-row">
          <div class="min-w-0">
            <div class="text-sm font-medium">{app.t("monitor")}</div>
            <div class="text-xs text-muted">{app.t("monitorHint")}</div>
          </div>
          <select
            class="field w-52"
            value={config.monitor || "primary"}
            onchange={(e) => app.saveConfig({ monitor: selectValue(e) })}
            aria-label={app.t("monitor")}
          >
            <option value="primary">{app.t("monitorPrimary")}</option>
            <option value="all">{app.t("monitorAll")}</option>
            {#each app.settings?.screens ?? [] as s (s.id)}
              <option value={s.id}>{app.t("screenLabel", { name: s.name, w: s.width, h: s.height })}</option>
            {/each}
          </select>
        </div>
      {/if}

      {#if config}
        <div class="setting-row">
          <div class="min-w-0">
            <div class="text-sm font-medium">{app.t("refresh")}</div>
            <div class="text-xs text-muted">{app.t("refreshHint")}</div>
          </div>
          <select
            class="field w-44"
            value={String(config.poll_seconds)}
            onchange={(e) => app.saveConfig({ poll_seconds: Number(selectValue(e)) })}
            aria-label={app.t("refresh")}
          >
            {#each refreshOptions as s (s)}
              <option value={String(s)}>{refreshLabel(s)}</option>
            {/each}
          </select>
        </div>
      {/if}
    </div>
  </section>

  <section class="mb-5">
    <h2 class="section-title mb-2 px-1">{app.t("sectionWakeLock")}</h2>
    <div class="card">
      <div class="setting-row">
        <div class="flex min-w-0 items-center gap-2">
          <WakeIcon active={app.wake.active} size={16} />
          <div class="min-w-0">
            <div class="text-sm font-medium">{app.t("wakeStatus")}</div>
            <div class="truncate text-xs text-muted tabular">{wakeText(app.wake, app.wakeAt, app.now, app.t)}</div>
          </div>
        </div>
        <button type="button" class="btn btn-danger" disabled={!app.wake.active} onclick={() => app.stopWake()}>
          {app.t("wakeStop")}
        </button>
      </div>

      <div class="setting-row flex-col items-stretch! gap-2!">
        <div class="text-sm font-medium">{app.t("wakeDuration")}</div>
        <div class="flex flex-wrap gap-1.5">
          {#each WAKE_MINUTES as m (m)}
            <button type="button" class="btn" onclick={() => app.startWake(m)}>{wakeLabel(m)}</button>
          {/each}
          <button
            type="button"
            class="btn"
            class:btn-active={app.wake.active && app.wake.indefinite}
            onclick={() => app.startWake(0)}>{app.t("wakeIndefinitely")}</button
          >
        </div>
      </div>

      <div class="setting-row">
        <div class="min-w-0">
          <div class="text-sm font-medium">{app.t("keepUnlocked")}</div>
          <div class="text-xs text-muted">{app.t("keepUnlockedHint")}</div>
        </div>
        <Toggle
          checked={config?.keep_unlocked ?? app.wake.keepUnlocked}
          label={app.t("keepUnlocked")}
          onchange={(on) => app.setKeepUnlocked(on)}
        />
      </div>
    </div>
  </section>

  <section>
    <h2 class="section-title mb-2 px-1">{app.t("sectionGeneral")}</h2>
    <div class="card">
      <div class="setting-row">
        <div class="min-w-0">
          <div class="text-sm font-medium">{app.t("autostart")}</div>
          <div class="text-xs text-muted">{app.t("autostartHint")}</div>
        </div>
        <Toggle
          checked={app.settings?.autostart ?? false}
          disabled={!app.settings}
          label={app.t("autostart")}
          onchange={(on) => app.setAutostart(on)}
        />
      </div>

      <div class="setting-row">
        <div class="text-sm font-medium">{app.t("language")}</div>
        <select
          class="field w-44"
          value={config?.language || "system"}
          disabled={!config}
          onchange={(e) => app.saveConfig({ language: selectValue(e) })}
          aria-label={app.t("language")}
        >
          {#each LANGUAGES as l (l.value)}
            <option value={l.value}>{app.t(l.label)}</option>
          {/each}
        </select>
      </div>

      <div class="setting-row">
        <div class="text-sm font-medium">{app.t("version")}</div>
        <span class="text-sm text-muted tabular">{app.settings?.version || "–"}</span>
      </div>
    </div>
    <div class="mt-4 flex justify-end">
      <button type="button" class="btn btn-danger" onclick={quit}>{app.t("quit")}</button>
    </div>
  </section>
</main>
