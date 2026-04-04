import type {
  DescriptionAgentStrategy,
  DescriptionAgentStrategyTarget,
} from "@/types/description-agent-strategies";

const STRATEGIES_URL = "/api/description-agent-strategies";
const EXPAND_URL = "/api/description-agent-strategies-expand";

type Fail = { ok: false; error: string };

export async function fetchStrategiesList(): Promise<
  { ok: true; strategies: DescriptionAgentStrategy[] } | Fail
> {
  try {
    const res = await fetch(STRATEGIES_URL);
    const data = (await res.json()) as {
      success?: boolean;
      strategies?: DescriptionAgentStrategy[];
      error?: string;
    };
    if (!res.ok || !data.success || !Array.isArray(data.strategies)) {
      return { ok: false, error: data.error || `Request failed (${res.status})` };
    }
    return { ok: true, strategies: data.strategies };
  } catch {
    return { ok: false, error: "Network error loading strategies" };
  }
}

export type ExpandGeneratedPayload = {
  id: string;
  display_name: string;
  system_prompt: string;
  user_prompt_template: string;
};

export async function postExpandStrategy(input: {
  intent: string;
  targets: DescriptionAgentStrategyTarget[];
  existingId?: string;
}): Promise<{ ok: true; generated: ExpandGeneratedPayload } | Fail> {
  try {
    const res = await fetch(EXPAND_URL, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        intent: input.intent,
        targets: input.targets,
        existing_id: input.existingId,
      }),
    });
    const data = (await res.json()) as {
      success?: boolean;
      error?: string;
      generated?: ExpandGeneratedPayload;
    };
    if (!res.ok || !data.success || !data.generated) {
      return { ok: false, error: data.error || `Generate failed (${res.status})` };
    }
    return { ok: true, generated: data.generated };
  } catch {
    return { ok: false, error: "Network error while generating prompts" };
  }
}

export async function postStrategy(body: Record<string, unknown>): Promise<{ ok: true } | Fail> {
  try {
    const res = await fetch(STRATEGIES_URL, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    const data = (await res.json()) as { success?: boolean; error?: string };
    if (!res.ok || !data.success) {
      return { ok: false, error: data.error || `Create failed (${res.status})` };
    }
    return { ok: true };
  } catch {
    return { ok: false, error: "Network error" };
  }
}

export async function patchStrategy(
  id: string,
  body: Record<string, unknown>
): Promise<{ ok: true } | Fail> {
  try {
    const res = await fetch(`${STRATEGIES_URL}?id=${encodeURIComponent(id)}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    const data = (await res.json()) as { success?: boolean; error?: string };
    if (!res.ok || !data.success) {
      return { ok: false, error: data.error || `Update failed (${res.status})` };
    }
    return { ok: true };
  } catch {
    return { ok: false, error: "Network error" };
  }
}

export async function deleteStrategy(id: string): Promise<{ ok: true } | Fail> {
  try {
    const res = await fetch(`${STRATEGIES_URL}?id=${encodeURIComponent(id)}`, {
      method: "DELETE",
    });
    const data = (await res.json()) as { success?: boolean; error?: string };
    if (!res.ok || !data.success) {
      return { ok: false, error: data.error || `Delete failed (${res.status})` };
    }
    return { ok: true };
  } catch {
    return { ok: false, error: "Network error" };
  }
}
