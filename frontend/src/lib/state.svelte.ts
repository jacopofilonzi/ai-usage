// Shared reactive state for every window. `start()` loads the initial values through the bound
// services and then keeps them current from the Go events (internal/app/contract.go):
// usage:snapshot, settings:changed and wakelock:status. Each window has its own JS context, so each
// holds its own copy; Go is the single source of truth and broadcasts every change.
//
// An event can arrive before the initial call resolves; the call's (older) result is then dropped.

import { Events } from "@wailsio/runtime";
import {
  SettingsService,
  UsageService,
  WakeLockService,
  type ProviderInfo,
  type Settings,
} from "../../bindings/github.com/jacopofilonzi/ai-usage/internal/app";
import type { Config } from "../../bindings/github.com/jacopofilonzi/ai-usage/internal/config";
import { Status, type Limit, type Snapshot } from "../../bindings/github.com/jacopofilonzi/ai-usage/internal/provider";
import type { Status as WakeStatus } from "../../bindings/github.com/jacopofilonzi/ai-usage/internal/wakelock";
import { isStale } from "./format";
import { resolveLang, translate, type Key, type Params } from "./i18n";
import { logError } from "./log";

/** How often countdowns re-render; they show minutes, so this only bounds the drift. */
const TICK_MS = 10_000;

class AppState {
  snapshot = $state<Snapshot>({ provider: "", status: Status.StatusLoading, limits: [] });
  providers = $state<ProviderInfo[]>([]);
  settings = $state<Settings | null>(null);
  wake = $state<WakeStatus>({ active: false, indefinite: false, remainingS: 0, keepUnlocked: false });
  /** When `wake` was received: `remainingS` counts down from here when `until` is missing. */
  wakeAt = $state(Date.now());
  /** Clock for countdowns, advanced every TICK_MS. */
  now = $state(Date.now());
  /** Set when the last Save failed; cleared by the next successful one. */
  saveError = $state(false);

  lang = $derived(resolveLang(this.settings?.config.language, navigator.language));
  limits = $derived<Limit[]>((this.snapshot.limits ?? []).slice(0, 3));
  stale = $derived(isStale(this.snapshot));
  providerName = $derived(
    this.providers.find((p) => p.id === this.snapshot.provider)?.name ??
      this.providers.find((p) => p.active)?.name ??
      this.snapshot.provider,
  );

  /** Translator bound to the current language; reading it inside markup makes it reactive. */
  t = (key: Key, params?: Params): string => translate(this.lang, key, params);

  #started = false;

  start(): void {
    if (this.#started) return;
    this.#started = true;

    const seen = { snapshot: false, settings: false, wake: false };
    Events.On("usage:snapshot", (ev) => {
      seen.snapshot = true;
      this.snapshot = ev.data;
    });
    Events.On("settings:changed", (ev) => {
      seen.settings = true;
      this.applySettings(ev.data);
    });
    Events.On("wakelock:status", (ev) => {
      seen.wake = true;
      this.setWake(ev.data);
    });

    UsageService.Snapshot()
      .then((s) => {
        if (!seen.snapshot) this.snapshot = s;
      })
      .catch((err: unknown) => logError("load snapshot", err));
    UsageService.Providers()
      .then((p) => {
        if (!seen.settings) this.providers = p ?? [];
      })
      .catch((err: unknown) => logError("load providers", err));
    SettingsService.Get()
      .then((s) => {
        if (!seen.settings) this.applySettings(s);
      })
      .catch((err: unknown) => logError("load settings", err));
    WakeLockService.Status()
      .then((w) => {
        if (!seen.wake) this.setWake(w);
      })
      .catch((err: unknown) => logError("load wake lock status", err));

    setInterval(() => (this.now = Date.now()), TICK_MS);

    $effect.root(() => {
      $effect(() => {
        document.documentElement.lang = this.lang;
      });
    });
  }

  private applySettings(s: Settings): void {
    this.settings = s;
    // Providers() stays authoritative until settings arrive; an empty settings list (the Go stub)
    // must not hide providers already loaded.
    if (s.providers?.length || !this.providers.length) this.providers = s.providers ?? [];
  }

  private setWake(w: WakeStatus): void {
    this.wake = w;
    this.wakeAt = Date.now();
    this.now = this.wakeAt;
  }

  /** Applies a config change immediately and persists it (Go re-broadcasts settings:changed). */
  async saveConfig(patch: Partial<Config>): Promise<void> {
    if (!this.settings) return;
    const next: Config = { ...this.settings.config, ...patch };
    this.settings.config = next;
    try {
      await SettingsService.Save(next);
      this.saveError = false;
    } catch (err) {
      logError("save settings", err);
      this.saveError = true;
      await this.reloadSettings();
    }
  }

  async setAutostart(on: boolean): Promise<void> {
    if (this.settings) this.settings.autostart = on;
    try {
      await SettingsService.SetAutostart(on);
      this.saveError = false;
    } catch (err) {
      logError("set autostart", err);
      this.saveError = true;
      await this.reloadSettings();
    }
  }

  async setKeepUnlocked(on: boolean): Promise<void> {
    this.wake.keepUnlocked = on;
    if (this.settings) this.settings.config.keep_unlocked = on;
    try {
      await WakeLockService.SetKeepUnlocked(on);
      this.saveError = false;
    } catch (err) {
      logError("set keep unlocked", err);
      this.saveError = true;
      await this.reloadSettings();
    }
  }

  async startWake(minutes: number): Promise<void> {
    try {
      await WakeLockService.Start(minutes);
    } catch (err) {
      logError("start wake lock", err);
    }
  }

  async stopWake(): Promise<void> {
    try {
      await WakeLockService.Stop();
    } catch (err) {
      logError("stop wake lock", err);
    }
  }

  async setProvider(id: string): Promise<void> {
    this.providers = this.providers.map((p) => ({ ...p, active: p.id === id }));
    try {
      await UsageService.SetProvider(id);
    } catch (err) {
      logError("set provider", err);
    }
  }

  async cycleProvider(step: number): Promise<void> {
    try {
      await UsageService.Cycle(step);
    } catch (err) {
      logError("cycle provider", err);
    }
  }

  private async reloadSettings(): Promise<void> {
    try {
      this.applySettings(await SettingsService.Get());
    } catch (err) {
      logError("reload settings", err);
    }
  }
}

export const app = new AppState();
