// The frontend's only logger: everything else reports failures through `logError`, so the sink can
// change (e.g. forwarding to the Go log) in one place. The webview console is visible in dev tools.

export function logError(context: string, err: unknown): void {
  console.error(`[ai-usage] ${context}:`, err);
}
