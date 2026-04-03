/**
 * POST /api/description-agent-strategies-expand
 * Uses OpenAI-compatible Chat Completions to turn plain-language intent into
 * id, display_name, system_prompt, user_prompt_template for description_agent.py.
 */

const ID_RE = /^[a-zA-Z0-9][a-zA-Z0-9._-]{0,79}$/;

const DEFAULT_BASE = "https://api.openai.com/v1";

function json(data: unknown, status = 200) {
  return Response.json(data, { status });
}

function openaiChatUrl(): string {
  const base = (process.env.OPENAI_BASE_URL || DEFAULT_BASE).trim().replace(/\/$/, "");
  return `${base}/chat/completions`;
}

function fallbackId(intent: string): string {
  const base = intent
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "")
    .slice(0, 48);
  const core = base.match(/^[a-z]/) ? base : `s_${base}`;
  const id = core || `strategy_${Date.now().toString(36)}`;
  return id.slice(0, 79);
}

function sanitizeId(raw: unknown, intent: string, existingId?: string): string {
  if (existingId && ID_RE.test(existingId)) return existingId;
  if (typeof raw === "string" && ID_RE.test(raw.trim())) return raw.trim();
  return fallbackId(intent);
}

function targetsLabel(targets: string[]): string {
  if (!targets.length) return "both events and markets";
  if (targets.length === 2) return "both events and markets";
  return targets.includes("events") ? "events only" : "markets only";
}

const RUNTIME_APPEND =
  'The Python runtime will append this exact block to system_prompt; do NOT repeat it in system_prompt: ' +
  '"Reply with a single JSON object only, no markdown, no extra text. ' +
  'Keys: \\"answer\\" (string, exactly \\"yes\\" or \\"no\\") and ' +
  '\\"supporting_description\\" (string, concise evidence from the rules text)."';

export async function POST(request: Request): Promise<Response> {
  const apiKey = process.env.OPENAI_API_KEY?.trim();
  if (!apiKey) {
    return json(
      {
        success: false,
        error: "OPENAI_API_KEY is not set (terminal .env or environment).",
      },
      503
    );
  }

  let body: { intent?: unknown; targets?: unknown; existing_id?: unknown };
  try {
    body = (await request.json()) as typeof body;
  } catch {
    return json({ success: false, error: "Invalid JSON body" }, 400);
  }

  const intent =
    typeof body.intent === "string" ? body.intent.trim() : "";
  if (!intent || intent.length < 8) {
    return json(
      { success: false, error: "intent must be at least 8 characters" },
      400
    );
  }
  if (intent.length > 8000) {
    return json({ success: false, error: "intent is too long (max 8000 chars)" }, 400);
  }

  const targetsRaw = Array.isArray(body.targets) ? body.targets : [];
  const targets = targetsRaw.filter((x) => x === "events" || x === "markets") as (
    | "events"
    | "markets"
  )[];

  const existingId =
    typeof body.existing_id === "string" && ID_RE.test(body.existing_id.trim())
      ? body.existing_id.trim()
      : undefined;

  const model =
    process.env.DESCRIPTION_AGENT_EXPAND_MODEL?.trim() || "gpt-4.1-mini";
  const tlabel = targetsLabel(targets);

  const userPrompt = existingId
    ? `The strategy id MUST remain exactly: "${existingId}" (do not change it).

Targets: ${tlabel}.

Scientist's intent (update prompts to match this):
---
${intent}
---

Return a JSON object with keys: id, display_name, system_prompt, user_prompt_template
- id: must equal "${existingId}"
- display_name: short human-readable title (under 80 chars)
- system_prompt: role and evaluation criteria only — no JSON output instructions (${RUNTIME_APPEND})
- user_prompt_template: must include {description} for the rules text. Include context using placeholders:
  Always useful: {event_title}, {event_slug}, {resolution_source}, {description}
  For markets also: {market_question}, {market_slug}, {market_id}, {event_id}
  End with a clear yes/no question aligned with the intent.`
    : `Targets: ${tlabel}.

Scientist's strategy (plain language):
---
${intent}
---

Return a JSON object with keys: id, display_name, system_prompt, user_prompt_template
- id: unique snake_case, 3-60 chars, [a-z0-9_], start with a letter
- display_name: short human-readable title (under 80 chars)
- system_prompt: role and evaluation criteria only — no JSON output instructions (${RUNTIME_APPEND})
- user_prompt_template: must include {description} for the rules text. Include context using placeholders:
  Events: {event_title}, {event_slug}, {resolution_source}, {description}
  Markets: add {market_question}, {market_slug}, {market_id}, {event_id} where relevant
  End with a clear yes/no question aligned with the strategy.`;

  const payload = {
    model,
    temperature: 0.2,
    response_format: { type: "json_object" },
    messages: [
      {
        role: "system" as const,
        content:
          "You write prompts for a prediction-market rule-analysis agent. " +
          "Output a single JSON object only, no markdown. " +
          "Keys required: id, display_name, system_prompt, user_prompt_template.",
      },
      { role: "user" as const, content: userPrompt },
    ],
  };

  const ctrl = new AbortController();
  const timer = setTimeout(() => ctrl.abort(), 120_000);

  let completionText: string;
  try {
    const res = await fetch(openaiChatUrl(), {
      method: "POST",
      headers: {
        Authorization: `Bearer ${apiKey}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
      signal: ctrl.signal,
    });
    const raw = (await res.json()) as {
      choices?: { message?: { content?: string } }[];
      error?: { message?: string };
    };
    if (!res.ok) {
      return json(
        {
          success: false,
          error: raw.error?.message || `OpenAI error HTTP ${res.status}`,
        },
        502
      );
    }
    const content = raw.choices?.[0]?.message?.content;
    if (typeof content !== "string" || !content.trim()) {
      return json({ success: false, error: "Empty model response" }, 502);
    }
    completionText = content.trim();
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    return json(
      { success: false, error: msg.includes("abort") ? "Request timed out" : msg },
      502
    );
  } finally {
    clearTimeout(timer);
  }

  let parsed: Record<string, unknown>;
  try {
    parsed = JSON.parse(completionText) as Record<string, unknown>;
  } catch {
    return json({ success: false, error: "Model returned invalid JSON" }, 502);
  }

  const system =
    typeof parsed.system_prompt === "string" ? parsed.system_prompt.trim() : "";
  const userT =
    typeof parsed.user_prompt_template === "string"
      ? parsed.user_prompt_template.trim()
      : "";
  const displayName =
    typeof parsed.display_name === "string" ? parsed.display_name.trim() : "";

  if (!system || !userT) {
    return json(
      { success: false, error: "Model omitted system_prompt or user_prompt_template" },
      502
    );
  }
  if (!userT.includes("{description}")) {
    return json(
      {
        success: false,
        error: 'user_prompt_template must contain the placeholder {description}',
      },
      502
    );
  }

  const id = sanitizeId(parsed.id, intent, existingId);
  if (!ID_RE.test(id)) {
    return json({ success: false, error: "Could not produce a valid strategy id" }, 502);
  }

  return json({
    success: true,
    generated: {
      id,
      display_name: displayName || id.replace(/_/g, " "),
      system_prompt: system,
      user_prompt_template: userT,
    },
  });
}
