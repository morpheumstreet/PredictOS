import { handleApi } from "@/server/handle-api";

const isProd = process.env.NODE_ENV === "production";
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
