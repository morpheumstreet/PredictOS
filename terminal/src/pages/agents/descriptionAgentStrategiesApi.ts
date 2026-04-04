import type {
  DescriptionAgentStrategy,
  DescriptionAgentStrategyTarget,
} from "@/types/description-agent-strategies";

const STRATEGIES_URL = "/api/description-agent-strategies";
const EXPAND_URL = "/api/description-agent-strategies-expand";
const STATUS_URL = "/api/description-agent-strategy-status";

export type DescriptionAgentStrategyStatusPayload = {
  strategyId: string;
  enabled: boolean;
  runnerParallelWorkers: number;
  runnerConfigWorkers: number | null;
  runStatePath: string;
  staleRunFileRemoved: boolean;
  running: boolean;
  runPid: number | null;
  runStartedAt: string | null;
  runTotalJobs: number;
  runCompletedJobs: number;
  queuedJobsThisStrategy: number;
  completedJobsThisStrategyInRun: number;
  estimatedFinishAt: string | null;
  processedRowsInDatabase: number;
  failedRowsInDatabase: number;
  lastProcessedAt: string | null;
};

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

export async function fetchStrategyStatus(
  id: string
): Promise<{ ok: true; status: DescriptionAgentStrategyStatusPayload } | Fail> {
  try {
    const res = await fetch(`${STATUS_URL}?id=${encodeURIComponent(id)}`);
    const data = (await res.json()) as {
      success?: boolean;
      error?: string;
    } & Partial<DescriptionAgentStrategyStatusPayload>;
    if (!res.ok || !data.success || !data.strategyId) {
      return { ok: false, error: data.error || `Status failed (${res.status})` };
    }
    return {
      ok: true,
      status: {
        strategyId: data.strategyId,
        enabled: Boolean(data.enabled),
        runnerParallelWorkers: Number(data.runnerParallelWorkers ?? 4),
        runnerConfigWorkers:
          data.runnerConfigWorkers === null || data.runnerConfigWorkers === undefined
            ? null
            : Number(data.runnerConfigWorkers),
        runStatePath: String(data.runStatePath ?? ""),
        staleRunFileRemoved: Boolean(data.staleRunFileRemoved),
        running: Boolean(data.running),
        runPid: data.runPid ?? null,
        runStartedAt: data.runStartedAt ?? null,
        runTotalJobs: Number(data.runTotalJobs ?? 0),
        runCompletedJobs: Number(data.runCompletedJobs ?? 0),
        queuedJobsThisStrategy: Number(data.queuedJobsThisStrategy ?? 0),
        completedJobsThisStrategyInRun: Number(data.completedJobsThisStrategyInRun ?? 0),
        estimatedFinishAt: data.estimatedFinishAt ?? null,
        processedRowsInDatabase: Number(data.processedRowsInDatabase ?? 0),
        failedRowsInDatabase: Number(data.failedRowsInDatabase ?? 0),
        lastProcessedAt: data.lastProcessedAt ?? null,
      },
    };
  } catch {
    return { ok: false, error: "Network error loading strategy status" };
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
