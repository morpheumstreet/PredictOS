/** Mirrors server relay allowlist — browser direct mode must not widen it. */

const ALLOWED_PATH_PREFIXES = [
  "/api/polymarket",
  "/api/strategy",
  "/api/ingestor",
  "/api/analytics",
  "/api/infrastructure",
  "/actuator",
] as const;

export function isAllowedPolybackRelayPath(path: string): boolean {
  if (!path.startsWith("/") || path.includes("..")) {
    return false;
  }
  return ALLOWED_PATH_PREFIXES.some((p) => path === p || path.startsWith(`${p}/`));
}
