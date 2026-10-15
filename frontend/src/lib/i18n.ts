// UI strings for the windows (the tray menu has its own Go dictionary in internal/i18n).
// `it` is typed from `en`, so a missing or extra key fails the type check. Placeholders are
// `{name}` and are filled by `translate`.

const en = {
  appName: "AI Usage",

  // Limit labels
  limitSession: "{h}h",
  limitWeekly: "Weekly",
  limitWeeklyModel: "Weekly · {model}",
  limitOther: "Limit",

  // Durations
  durDays: "{d}d {h}h",
  durHours: "{h}h {m}m",
  durMinutes: "{m}m",
  durUnderMinute: "<1m",
  windowDays: "{n}d",
  windowHours: "{n}h",
  windowMinutes: "{n}m",

  // Reset and freshness
  resetsIn: "resets in {time}",
  resetsNow: "resets now",
  updatedAgo: "updated {time} ago",
  updatedJustNow: "updated just now",

  // Status
  statusLoading: "Loading…",
  statusStale: "Token expired: open {provider} to refresh",
  statusNotLoggedIn: "Not signed in: log in to {provider}",
  statusError: "Couldn't read usage",
  statusNoData: "No usage data",
  statusNoProviders: "No supported agent detected",

  // Wake lock
  wakeOff: "Not active",
  wakeRemaining: "Awake · {time} left",
  wakeIndefinite: "Awake indefinitely",
  wake30m: "30m",
  wake1h: "1h",
  wakeForever: "∞",
  wakeStop: "Stop",
  wakeTitle: "Keep awake",

  // Popover
  openSettings: "Settings",
  switchProvider: "Provider",

  // Settings
  settingsTitle: "Settings",
  sectionUsage: "Usage",
  sectionWakeLock: "Wake lock",
  sectionGeneral: "General",
  provider: "Agent",
  providerHint: "The agent whose limits are shown",
  noProvidersHint: "Install and sign in to Claude Code or Codex, then reopen this window.",
  corner: "Widget corner",
  cornerHint: "Where the widget sits on the screen",
  alwaysShow: "Always show",
  alwaysShowHint: "Otherwise the widget appears when you push the cursor into the corner",
  cornerTopLeft: "Top left",
  cornerTopRight: "Top right",
  cornerBottomLeft: "Bottom left",
  cornerBottomRight: "Bottom right",
  monitor: "Monitor",
  monitorHint: "Screens that show the widget",
  monitorPrimary: "Primary",
  monitorAll: "All monitors",
  screenLabel: "{name} ({w}×{h})",
  refresh: "Refresh interval",
  refreshHint: "How often usage is read",
  refreshDefault: "Agent default",
  refreshMinutes: "Every {n} min",
  wakeStatus: "Status",
  wakeDuration: "Keep awake for",
  wakeMinutes: "{n} min",
  wakeHours: "{n} h",
  wakeIndefinitely: "Indefinitely",
  keepUnlocked: "Keep display on",
  keepUnlockedHint: "Also prevent the screen from turning off and locking",
  autostart: "Start at login",
  autostartHint: "Open AI Usage when you sign in",
  language: "Language",
  languageSystem: "System",
  languageEnglish: "English",
  languageItalian: "Italiano",
  version: "Version",
  quit: "Quit AI Usage",
  saveFailed: "Couldn't save the change",
} as const;

type Dict = { [K in keyof typeof en]: string };

const it: Dict = {
  appName: "AI Usage",

  limitSession: "{h}h",
  limitWeekly: "Settimanale",
  limitWeeklyModel: "Settimanale · {model}",
  limitOther: "Limite",

  durDays: "{d}g {h}h",
  durHours: "{h}h {m}m",
  durMinutes: "{m}m",
  durUnderMinute: "<1m",
  windowDays: "{n}g",
  windowHours: "{n}h",
  windowMinutes: "{n}m",

  resetsIn: "si azzera tra {time}",
  resetsNow: "si azzera ora",
  updatedAgo: "aggiornato {time} fa",
  updatedJustNow: "aggiornato ora",

  statusLoading: "Caricamento…",
  statusStale: "Token scaduto: apri {provider} per aggiornarlo",
  statusNotLoggedIn: "Accesso non effettuato: accedi a {provider}",
  statusError: "Impossibile leggere l'utilizzo",
  statusNoData: "Nessun dato di utilizzo",
  statusNoProviders: "Nessun agente supportato rilevato",

  wakeOff: "Non attivo",
  wakeRemaining: "Attivo · ancora {time}",
  wakeIndefinite: "Attivo senza limite",
  wake30m: "30m",
  wake1h: "1h",
  wakeForever: "∞",
  wakeStop: "Ferma",
  wakeTitle: "Tieni sveglio",

  openSettings: "Impostazioni",
  switchProvider: "Agente",

  settingsTitle: "Impostazioni",
  sectionUsage: "Utilizzo",
  sectionWakeLock: "Blocco sospensione",
  sectionGeneral: "Generale",
  provider: "Agente",
  providerHint: "L'agente di cui mostrare i limiti",
  noProvidersHint: "Installa Claude Code o Codex ed esegui l'accesso, poi riapri questa finestra.",
  corner: "Angolo del widget",
  cornerHint: "Dove si trova il widget sullo schermo",
  alwaysShow: "Mostra sempre",
  alwaysShowHint: "Altrimenti il widget compare quando porti il cursore nell'angolo",
  cornerTopLeft: "In alto a sinistra",
  cornerTopRight: "In alto a destra",
  cornerBottomLeft: "In basso a sinistra",
  cornerBottomRight: "In basso a destra",
  monitor: "Monitor",
  monitorHint: "Schermi su cui mostrare il widget",
  monitorPrimary: "Principale",
  monitorAll: "Tutti i monitor",
  screenLabel: "{name} ({w}×{h})",
  refresh: "Intervallo di aggiornamento",
  refreshHint: "Ogni quanto leggere l'utilizzo",
  refreshDefault: "Predefinito dell'agente",
  refreshMinutes: "Ogni {n} min",
  wakeStatus: "Stato",
  wakeDuration: "Tieni sveglio per",
  wakeMinutes: "{n} min",
  wakeHours: "{n} h",
  wakeIndefinitely: "Senza limite",
  keepUnlocked: "Tieni acceso lo schermo",
  keepUnlockedHint: "Impedisci anche lo spegnimento e il blocco dello schermo",
  autostart: "Avvia all'accesso",
  autostartHint: "Apri AI Usage quando accedi",
  language: "Lingua",
  languageSystem: "Sistema",
  languageEnglish: "English",
  languageItalian: "Italiano",
  version: "Versione",
  quit: "Esci da AI Usage",
  saveFailed: "Impossibile salvare la modifica",
};

export const dictionaries = { en, it } satisfies Record<string, Dict>;
export type Lang = keyof typeof dictionaries;
export type Key = keyof Dict;
export type Params = Record<string, string | number>;
export type Translate = (key: Key, params?: Params) => string;

export function translate(lang: Lang, key: Key, params?: Params): string {
  const text: string = dictionaries[lang][key];
  if (!params) return text;
  return text.replace(/\{(\w+)\}/g, (match, name: string) =>
    name in params ? String(params[name]) : match,
  );
}

/** Resolves the configured language ("system", "en", "it") to a dictionary. */
export function resolveLang(configured: string | undefined, systemLanguage: string): Lang {
  const code = !configured || configured === "system" ? systemLanguage : configured;
  return code.toLowerCase().startsWith("it") ? "it" : "en";
}
