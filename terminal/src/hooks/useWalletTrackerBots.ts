import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { openWalletTrackingEventSource } from "@/lib/wallet-tracking-sse";
import {
  WALLET_TRACKING_LOG_CAP,
  WALLET_TRACKING_STORAGE_KEY_V1,
  WALLET_TRACKING_STORAGE_KEY_V2,
  shortWalletLabel,
  normalizeWalletAddress,
  type WalletTrackingLogEntry,
  type WalletTrackingPersistedV2,
  type WalletTrackerBot,
} from "@/types/wallet-tracking";

function readPersisted(): WalletTrackingPersistedV2 {
  if (typeof window === "undefined") {
    return { bots: [], selected: null, running: [] };
  }
  try {
    const raw = sessionStorage.getItem(WALLET_TRACKING_STORAGE_KEY_V2);
    if (raw) {
      const p = JSON.parse(raw) as Partial<WalletTrackingPersistedV2>;
      const botsRaw = Array.isArray(p.bots) ? p.bots : [];
      const bots = [...new Set(botsRaw.map((x) => normalizeWalletAddress(String(x))).filter((x): x is string => Boolean(x)))];
      const selectedNorm = p.selected ? normalizeWalletAddress(String(p.selected)) : null;
      const selected = selectedNorm && bots.includes(selectedNorm) ? selectedNorm : bots[0] ?? null;
      const runningRaw = Array.isArray(p.running) ? p.running : [];
      const running = [
        ...new Set(
          runningRaw
            .map((x) => normalizeWalletAddress(String(x)))
            .filter((x): x is string => x !== null && bots.includes(x))
        ),
      ];
      return { bots, selected, running };
    }
    const v1 = sessionStorage.getItem(WALLET_TRACKING_STORAGE_KEY_V1);
    const k = v1 ? normalizeWalletAddress(v1) : null;
    if (k) {
      return { bots: [k], selected: k, running: [] };
    }
  } catch {
    /* ignore */
  }
  return { bots: [], selected: null, running: [] };
}

function writePersisted(p: WalletTrackingPersistedV2) {
  try {
    sessionStorage.setItem(WALLET_TRACKING_STORAGE_KEY_V2, JSON.stringify(p));
  } catch {
    /* ignore */
  }
}

export function useWalletTrackerBots(backendReady: boolean) {
  const persistedSnapshot = useRef(readPersisted());
  const [botOrder, setBotOrder] = useState<string[]>(() => persistedSnapshot.current.bots);
  const [selectedKey, setSelectedKey] = useState<string | null>(() => {
    const p = persistedSnapshot.current;
    if (p.selected && p.bots.includes(p.selected)) return p.selected;
    return p.bots[0] ?? null;
  });
  const [runningList, setRunningList] = useState<string[]>(() => {
    const p = persistedSnapshot.current;
    return p.running.filter((k) => p.bots.includes(k));
  });
  const [logsByWallet, setLogsByWallet] = useState<Record<string, WalletTrackingLogEntry[]>>(() => {
    const init: Record<string, WalletTrackingLogEntry[]> = {};
    for (const k of persistedSnapshot.current.bots) {
      init[k] = [];
    }
    return init;
  });

  const eventSourcesRef = useRef<Map<string, EventSource>>(new Map());
  const resumeStartedRef = useRef(false);

  const appendLogEntry = useCallback((key: string, entry: WalletTrackingLogEntry) => {
    setLogsByWallet((prev) => {
      const cur = prev[key] ?? [];
      const next = [...cur, entry];
      const capped =
        next.length > WALLET_TRACKING_LOG_CAP ? next.slice(-WALLET_TRACKING_LOG_CAP) : next;
      return { ...prev, [key]: capped };
    });
  }, []);

  const appendLogMessage = useCallback(
    (key: string, level: WalletTrackingLogEntry["level"], message: string, details?: Record<string, unknown>) => {
      appendLogEntry(key, {
        timestamp: new Date().toISOString(),
        level,
        message,
        details,
      });
    },
    [appendLogEntry]
  );

  const startStreaming = useCallback(
    (key: string) => {
      if (eventSourcesRef.current.has(key)) return;
      const es = openWalletTrackingEventSource(key, {
        onLog: (entry) => {
          setLogsByWallet((prev) => {
            const cur = prev[key] ?? [];
            const next = [...cur, entry];
            const capped =
              next.length > WALLET_TRACKING_LOG_CAP ? next.slice(-WALLET_TRACKING_LOG_CAP) : next;
            return { ...prev, [key]: capped };
          });
        },
        onEnded: () => {
          eventSourcesRef.current.delete(key);
          setRunningList((prev) => prev.filter((k) => k !== key));
          appendLogEntry(key, {
            timestamp: new Date().toISOString(),
            level: "ERROR",
            message: "Connection lost. Click Start to reconnect.",
          });
        },
      });
      eventSourcesRef.current.set(key, es);
    },
    [appendLogEntry]
  );

  useEffect(() => {
    writePersisted({ bots: botOrder, selected: selectedKey, running: runningList });
  }, [botOrder, selectedKey, runningList]);

  useEffect(() => {
    if (!backendReady || resumeStartedRef.current) return;
    resumeStartedRef.current = true;
    const { running, bots } = persistedSnapshot.current;
    for (const key of running) {
      if (!bots.includes(key)) continue;
      appendLogMessage(key, "INFO", `Resuming wallet tracking for ${shortWalletLabel(key)}`);
      startStreaming(key);
    }
  }, [backendReady, appendLogMessage, startStreaming]);

  useEffect(() => {
    return () => {
      for (const es of eventSourcesRef.current.values()) {
        es.close();
      }
      eventSourcesRef.current.clear();
    };
  }, []);

  const stopStreaming = useCallback(
    (key: string, logStopped: boolean) => {
      const es = eventSourcesRef.current.get(key);
      if (es) {
        es.close();
        eventSourcesRef.current.delete(key);
      }
      setRunningList((prev) => prev.filter((k) => k !== key));
      if (logStopped) {
        appendLogMessage(key, "INFO", "Wallet tracking stopped");
      }
    },
    [appendLogMessage]
  );

  const startBot = useCallback(
    (key: string): { ok: boolean; error?: string } => {
      if (!backendReady) {
        return { ok: false, error: "Wallet tracking backend is not ready." };
      }
      if (!botOrder.includes(key)) {
        return { ok: false, error: "Unknown wallet." };
      }
      if (runningList.includes(key)) {
        if (!eventSourcesRef.current.has(key)) {
          startStreaming(key);
        }
        return { ok: true };
      }
      setRunningList((prev) => (prev.includes(key) ? prev : [...prev, key]));
      appendLogMessage(
        key,
        "INFO",
        `Starting wallet tracking for ${shortWalletLabel(key)}`
      );
      startStreaming(key);
      return { ok: true };
    },
    [backendReady, botOrder, runningList, appendLogMessage, startStreaming]
  );

  const stopBot = useCallback(
    (key: string) => {
      stopStreaming(key, true);
    },
    [stopStreaming]
  );

  const addBot = useCallback((raw: string): { ok: boolean; error?: string } => {
    const key = normalizeWalletAddress(raw);
    if (!key) {
      return {
        ok: false,
        error: "Invalid wallet address format. Must be a valid Ethereum address (0x...).",
      };
    }
    let duplicate = false;
    setBotOrder((prev) => {
      if (prev.includes(key)) {
        duplicate = true;
        return prev;
      }
      return [...prev, key];
    });
    if (duplicate) {
      return { ok: false, error: "This wallet is already in the list." };
    }
    setLogsByWallet((prev) => ({ ...prev, [key]: prev[key] ?? [] }));
    setSelectedKey(key);
    return { ok: true };
  }, []);

  const removeBot = useCallback(
    (key: string) => {
      stopStreaming(key, false);
      setBotOrder((prev) => {
        const next = prev.filter((k) => k !== key);
        setSelectedKey((sel) => (sel === key ? next[0] ?? null : sel));
        return next;
      });
      setLogsByWallet((prev) => {
        const { [key]: _removed, ...rest } = prev;
        return rest;
      });
      setRunningList((prev) => prev.filter((k) => k !== key));
    },
    [stopStreaming]
  );

  const clearLogsForSelected = useCallback(() => {
    if (!selectedKey) return;
    setLogsByWallet((prev) => ({ ...prev, [selectedKey]: [] }));
  }, [selectedKey]);

  const botsList: WalletTrackerBot[] = useMemo(
    () =>
      botOrder.map((addressKey) => ({
        addressKey,
        isRunning: runningList.includes(addressKey),
        logs: logsByWallet[addressKey] ?? [],
      })),
    [botOrder, runningList, logsByWallet]
  );

  const selectedBot = useMemo(
    () => (selectedKey ? botsList.find((b) => b.addressKey === selectedKey) ?? null : null),
    [botsList, selectedKey]
  );

  const anyRunning = runningList.length > 0;

  return {
    botOrder,
    botsList,
    selectedKey,
    setSelectedKey,
    selectedBot,
    runningList,
    addBot,
    removeBot,
    startBot,
    stopBot,
    clearLogsForSelected,
    anyRunning,
  };
}
