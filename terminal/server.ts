import { handleApi } from "@/server/handle-api";

// HTTP API includes read-only SQLite at GET /api/alpha-rules (strat/alpha-rules/data/alpha_rules.sqlite).
// Optional: ALPHA_RULES_DB=/absolute/path/to/alpha_rules.sqlite
// Agent run log (writable): terminal/data/terminal_local.sqlite — optional TERMINAL_LOCAL_DB override; GET /api/agent-runs
// Polyback-mm: POLYBACK_BOOTSTRAP_URL (default http://127.0.0.1:8080) → GET /api/polyback/config/client, GET /api/polyback/relay

const isProd = process.env.BUN_ENV === "production";
const rootDir = isProd ? "dist" : "public";
const port = Number(process.env.PORT) || 3000;

function spaResponse(): Response {
  const file = Bun.file(`${rootDir}/index.html`);
  return new Response(file, {
    headers: { "Content-Type": "text/html;charset=utf-8" },
  });
}

const server = Bun.serve({
  port,
  async fetch(req) {
    const url = new URL(req.url);
    const pathname = url.pathname;

    const apiRes = await handleApi(req);
    if (apiRes) return apiRes;

    if (pathname !== "/" && !pathname.includes("..")) {
      const file = Bun.file(`${rootDir}${pathname}`);
      if (await file.exists()) {
        return new Response(file);
      }
    }

    if (req.method === "GET" && !pathname.match(/\.[a-zA-Z0-9]+$/)) {
      return spaResponse();
    }

    return new Response("Not Found", { status: 404 });
  },
});

console.log(`PredictOS terminal → http://localhost:${server.port} (${isProd ? "production" : "development"})`);
