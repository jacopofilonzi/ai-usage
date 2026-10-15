// UI strings for the windows (the tray menu has its own Go dictionary in internal/i18n).
// `it` is typed from `en`, so a missing or extra key fails the type check.

const en = {
  appName: "AI Usage",
} as const;

type Dict = { [K in keyof typeof en]: string };

const it: Dict = {
  appName: "AI Usage",
};

export const dictionaries = { en, it } satisfies Record<string, Dict>;
export type Lang = keyof typeof dictionaries;
export type Key = keyof Dict;
