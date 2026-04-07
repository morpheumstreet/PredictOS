import * as alphaRules from "@/server/api/alpha-rules";
import * as descriptionAgentStrategies from "@/server/api/description-agent-strategies";
import * as descriptionAgentStrategiesExpand from "@/server/api/description-agent-strategies-expand";
import * as descriptionAgentStrategyStatus from "@/server/api/description-agent-strategy-status";
import * as agentRuns from "@/server/api/agent-runs";
import * as arbitrageFinder from "@/server/api/arbitrage-finder";
import * as bookmakerAgent from "@/server/api/bookmaker-agent";
import * as eventAnalysisAgent from "@/server/api/event-analysis-agent";
import * as getEvents from "@/server/api/get-events";
import * as irysUpload from "@/server/api/irys-upload";
import * as limitOrderBot from "@/server/api/limit-order-bot";
import * as mapperAgent from "@/server/api/mapper-agent";
import * as polymarketPutOrder from "@/server/api/polymarket-put-order";
import * as polybackConfigClient from "@/server/api/polyback-config-client";
import * as polyfactualResearch from "@/server/api/polyfactual-research";
import * as positionTracker from "@/server/api/position-tracker";
import * as walletTracking from "@/server/api/wallet-tracking";
import * as x402Seller from "@/server/api/x402-seller";

export type ApiRouteHandlers = Partial<Record<string, (req: Request) => Promise<Response>>>;

const routes: Record<string, ApiRouteHandlers> = {
  "/api/alpha-rules": { GET: alphaRules.GET },
  "/api/description-agent-strategies": {
    GET: descriptionAgentStrategies.GET,
    POST: descriptionAgentStrategies.POST,
    PATCH: descriptionAgentStrategies.PATCH,
    DELETE: descriptionAgentStrategies.DELETE,
  },
  "/api/description-agent-strategies-expand": {
    POST: descriptionAgentStrategiesExpand.POST,
  },
  "/api/description-agent-strategy-status": {
    GET: descriptionAgentStrategyStatus.GET,
  },
  "/api/agent-runs": { GET: agentRuns.GET },
  "/api/arbitrage-finder": { POST: arbitrageFinder.POST },
  "/api/bookmaker-agent": { POST: bookmakerAgent.POST },
  "/api/event-analysis-agent": { POST: eventAnalysisAgent.POST },
  "/api/get-events": { POST: getEvents.POST },
  "/api/irys-upload": { GET: irysUpload.GET, POST: irysUpload.POST },
  "/api/limit-order-bot": { POST: limitOrderBot.POST },
  "/api/mapper-agent": { POST: mapperAgent.POST },
  "/api/polymarket-put-order": { POST: polymarketPutOrder.POST },
  "/api/polyfactual-research": { POST: polyfactualResearch.POST },
  "/api/polyback/config/client": { GET: polybackConfigClient.GET },
  "/api/position-tracker": { POST: positionTracker.POST },
  "/api/wallet-tracking": { GET: walletTracking.GET },
  "/api/x402-seller": { POST: x402Seller.POST, OPTIONS: x402Seller.OPTIONS },
};

export async function handleApi(req: Request): Promise<Response | null> {
  const url = new URL(req.url);
  const pathname = url.pathname.replace(/\/$/, "") || "/";
  const handlers = routes[pathname];
  if (!handlers) return null;

  const method = req.method;
  const handler = handlers[method as keyof typeof handlers];
  if (!handler) {
    return new Response(JSON.stringify({ error: "Method not allowed" }), {
      status: 405,
      headers: { "Content-Type": "application/json" },
    });
  }

  return handler(req);
}
