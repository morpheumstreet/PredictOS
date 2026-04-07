import { join } from "path";

/**
 * Default: terminal/data/terminal_local.sqlite (sibling paths via import.meta.dir).
 * Override with TERMINAL_LOCAL_DB for custom deployments.
 */
export function getLocalRunLogDbPath(): string {
  const override = process.env.TERMINAL_LOCAL_DB?.trim();
  if (override) return override;
  return join(import.meta.dir, "../data/terminal_local.sqlite");
}
