// Accent colour per agent, so the widget and the popover read at a glance as "this agent's usage".
// The accent replaces the "ok" green; the warn/crit colours stay, because they carry the warning.

const accents: Record<string, string> = {
  claude: "#d97757", // Claude's terracotta orange
  codex: "#7aa2f7", // a cool blue, distinct from Claude's orange
};

/** CSS colour for the provider, or null to keep the default "ok" colour. */
export function accentFor(providerId: string | undefined): string | null {
  return (providerId && accents[providerId]) || null;
}
