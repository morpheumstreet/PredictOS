/** Base URL for Polyback Intelligence (Go :8085). Override with INTELLIGENCE_BASE_URL. */
export function getIntelligenceBaseUrl(): string {
  return (process.env.INTELLIGENCE_BASE_URL?.trim() || "http://127.0.0.1:8085").replace(
    /\/$/,
    "",
  );
}

/** POST target for /api/intelligence/<segment> (segment without leading slash). */
export function intelligenceApiUrl(segment: string): string {
  const s = segment.replace(/^\//, "");
  return `${getIntelligenceBaseUrl()}/api/intelligence/${s}`;
}
