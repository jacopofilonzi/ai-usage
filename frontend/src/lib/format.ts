// Pure formatting helpers shared by every view. They take the translator so they stay testable
// and free of global state; views pass `app.t` from state.svelte.ts.

import { Kind, Status, type Limit, type Snapshot } from "../../bindings/github.com/jacopofilonzi/ai-usage/internal/provider";
import type { Status as WakeStatus } from "../../bindings/github.com/jacopofilonzi/ai-usage/internal/wakelock";
import type { Translate } from "./i18n";

const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

/** Usage level used for colours: <70 ok, <90 warn, >=90 crit, stale when values are not fresh. */
export type Level = "ok" | "warn" | "crit" | "stale";

export function level(usedPct: number, stale: boolean): Level {
  if (stale) return "stale";
  if (usedPct >= 90) return "crit";
  if (usedPct >= 70) return "warn";
  return "ok";
}

// Literal class names so Tailwind's scanner emits the @utility classes from app.css.
const levelClasses: Record<Level, string> = {
  ok: "level-ok",
  warn: "level-warn",
  crit: "level-crit",
  stale: "level-stale",
};

/** Text colour class for a level. */
export function levelClass(l: Level): string {
  return levelClasses[l];
}

/** CSS colour (a token from app.css) for a level, for SVG strokes. */
export function levelColor(l: Level): string {
  return `var(--c-${l})`;
}

/** Whether the snapshot's limits are old values (drawn grey). */
export function isStale(s: Snapshot): boolean {
  return s.status === Status.StatusStale || s.status === Status.StatusError;
}

export function percent(usedPct: number): string {
  return `${Math.round(Math.max(0, usedPct))}%`;
}

function windowLabel(minutes: number, t: Translate): string {
  if (minutes <= 0) return t("limitOther");
  if (minutes % 1440 === 0) return t("windowDays", { n: minutes / 1440 });
  if (minutes % 60 === 0) return t("windowHours", { n: minutes / 60 });
  return t("windowMinutes", { n: minutes });
}

function capitalize(s: string): string {
  return s ? s[0].toUpperCase() + s.slice(1) : s;
}

export function limitLabel(l: Limit, t: Translate): string {
  switch (l.kind) {
    case Kind.KindSession:
      return t("limitSession", { h: l.windowMinutes > 0 ? Math.round(l.windowMinutes / 60) : 5 });
    case Kind.KindWeekly:
      return t("limitWeekly");
    case Kind.KindWeeklyModel:
      return l.model ? t("limitWeeklyModel", { model: capitalize(l.model) }) : t("limitWeekly");
    default:
      return windowLabel(l.windowMinutes, t);
  }
}

/** Compact duration: "3d 4h" from a day up, "2h 13m" from an hour up, else "12m". */
export function duration(ms: number, t: Translate): string {
  if (ms < MINUTE) return t("durUnderMinute");
  if (ms >= DAY) {
    return t("durDays", { d: Math.floor(ms / DAY), h: Math.floor((ms % DAY) / HOUR) });
  }
  if (ms >= HOUR) {
    return t("durHours", { h: Math.floor(ms / HOUR), m: Math.floor((ms % HOUR) / MINUTE) });
  }
  return t("durMinutes", { m: Math.floor(ms / MINUTE) });
}

function parseTime(iso: string | null | undefined): number | null {
  if (!iso) return null;
  const ms = Date.parse(iso);
  return Number.isNaN(ms) ? null : ms;
}

/** Time left until the limit resets, or null when unknown. */
export function resetRemaining(l: Limit, now: number): number | null {
  const at = parseTime(l.resetsAt);
  return at === null ? null : Math.max(0, at - now);
}

/** Short countdown for tight spaces ("2h 13m"); "" when unknown. */
export function resetShort(l: Limit, now: number, t: Translate): string {
  const left = resetRemaining(l, now);
  if (left === null) return "";
  return left < MINUTE ? t("durUnderMinute") : duration(left, t);
}

/** Full sentence: "resets in 2h 13m"; "" when unknown. */
export function resetText(l: Limit, now: number, t: Translate): string {
  const left = resetRemaining(l, now);
  if (left === null) return "";
  return left < MINUTE ? t("resetsNow") : t("resetsIn", { time: duration(left, t) });
}

/** "updated 12m ago"; "" when the snapshot was never read. */
export function updatedAgo(s: Snapshot, now: number, t: Translate): string {
  const at = parseTime(s.updatedAt);
  if (at === null) return "";
  const ago = Math.max(0, now - at);
  return ago < MINUTE ? t("updatedJustNow") : t("updatedAgo", { time: duration(ago, t) });
}

/**
 * Status line for a snapshot, or "" when there is nothing to say (fresh values).
 * `hasProviders` false means no agent is installed at all.
 */
export function statusText(s: Snapshot, providerName: string, hasProviders: boolean, t: Translate): string {
  if (!hasProviders) return t("statusNoProviders");
  switch (s.status) {
    case Status.StatusOK:
      return s.limits?.length ? "" : t("statusNoData");
    case Status.StatusStale:
      return t("statusStale", { provider: providerName });
    case Status.StatusNotLoggedIn:
      return t("statusNotLoggedIn", { provider: providerName });
    case Status.StatusError:
      return t("statusError");
    default:
      return t("statusLoading");
  }
}

/** Remaining wake-lock time in ms, or null when inactive or indefinite. */
export function wakeRemaining(w: WakeStatus, receivedAt: number, now: number): number | null {
  if (!w.active || w.indefinite) return null;
  const until = parseTime(w.until);
  const left = until !== null ? until - now : w.remainingS * 1000 - (now - receivedAt);
  return Math.max(0, left);
}

/** "Awake · 1h 20m left", "Awake indefinitely" or "Not active". */
export function wakeText(w: WakeStatus, receivedAt: number, now: number, t: Translate): string {
  if (!w.active) return t("wakeOff");
  if (w.indefinite) return t("wakeIndefinite");
  return t("wakeRemaining", { time: duration(wakeRemaining(w, receivedAt, now) ?? 0, t) });
}

/** Compact wake indicator for the widget: "1h 20m" or "∞"; "" when inactive. */
export function wakeShort(w: WakeStatus, receivedAt: number, now: number, t: Translate): string {
  if (!w.active) return "";
  if (w.indefinite) return t("wakeForever");
  return duration(wakeRemaining(w, receivedAt, now) ?? 0, t);
}
