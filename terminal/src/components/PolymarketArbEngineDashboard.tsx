import { useEffect, useMemo, useState, type ReactNode } from "react";
import { cn } from "@/lib/utils";

/* ─── theme (single source) ─────────────────────────────────────────── */
/** Hex tokens for inline styles / SVG; borders use literal Tailwind classes so JIT picks them up */
const T = {
  green: "#00FF41",
  cyan: "#00FFFF",
  orange: "#FFA500",
  red: "#FF0000",
} as const;

const borderCls = "border-[#1a1a1a]";

const tx = {
  g: "text-[#00FF41]",
  c: "text-[#00FFFF]",
  o: "text-[#FFA500]",
  r: "text-[#FF0000]",
} as const;

type Tone = keyof typeof tx;

const LIQ_TONE: Record<"DEEP" | "MODERATE" | "THIN" | "VERY THIN", Tone> = {
  DEEP: "g",
  MODERATE: "c",
  THIN: "o",
  "VERY THIN": "r",
};

const EXEC_TAG_TONE: Record<"ARB" | "CPY", Tone> = { ARB: "c", CPY: "o" };

/* ─── mock data ─────────────────────────────────────────────────────── */
type ScannerRow = {
  market: string;
  impl: string;
  base: string;
  ttr: string;
  ev: number;
  liq: keyof typeof LIQ_TONE;
};

const SCANNER_ROWS: ScannerRow[] = [
  { market: "RUSSIA ASSET FI", impl: "59.0%", base: "90.7%", ttr: "74d", ev: 0.318, liq: "DEEP" },
  { market: "APPLE VISION PI", impl: "42.1%", base: "38.2%", ttr: "2d", ev: -0.231, liq: "MODERATE" },
  { market: "UKRAINE CEASEF", impl: "71.5%", base: "68.0%", ttr: "45d", ev: 0.092, liq: "THIN" },
  { market: "FED RATE CUT Q2", impl: "33.0%", base: "41.2%", ttr: "120d", ev: 0.155, liq: "DEEP" },
  { market: "BTC 100K 2026", impl: "22.4%", base: "18.9%", ttr: "300d", ev: -0.044, liq: "VERY THIN" },
];

const EXEC_LINES: { tag: keyof typeof EXEC_TAG_TONE; msg: string }[] = [
  { tag: "ARB", msg: "NO OPENAI BO" },
  { tag: "CPY", msg: "YES TAIWAN S" },
  { tag: "ARB", msg: "NO US RECESS" },
  { tag: "ARB", msg: "YES ETH FLIP" },
  { tag: "CPY", msg: "NO ELECTION X" },
  { tag: "ARB", msg: "YES CPI COOL" },
];

const WALLET_LINES: { addr: string; msg: string; tone: Tone }[] = [
  { addr: "0xB81c..f044", msg: "BOOK THIN", tone: "g" },
  { addr: "0x9a2e..c901", msg: "SMART MONEY IN", tone: "o" },
  { addr: "0x71Ff..2aa0", msg: "CLUSTER DETECTED", tone: "c" },
  { addr: "0x3c44..88bd", msg: "FLOW NOMINAL", tone: "g" },
  { addr: "0xF17a..b990", msg: "LEAD WALLET ACT", tone: "o" },
];

const TOP_OPPS: { name: string; ev: number }[] = [
  { name: "RUSSIA ASSET FI", ev: 0.318 },
  { name: "FED RATE CUT Q2", ev: 0.155 },
  { name: "APPLE VISION PI", ev: -0.231 },
];

const MODEL_LINES: { line: string; tone: Tone }[] = [
  { line: "[BASE RATE] 6% vs 50%", tone: "o" },
  { line: "[TIME DECAY] NORMAL", tone: "o" },
  { line: "[LIQUIDITY] DEEP", tone: "g" },
  { line: "[EDGE] EV NEUTRAL", tone: "o" },
  { line: "[CONFIDENCE] 0.6990", tone: "g" },
  { line: "[ENTROPY] 0.3919", tone: "o" },
];

const DIAGNOSTIC_LINES = ["[API FEEDS] NOMINAL", "[EXEC LAYER] READY", "[RISK ENGINE] ACTIVE"] as const;

const SCANNER_COLUMNS = ["MARKET", "IMPL%", "BASE%", "TTR", "EV", "LIQ"] as const;

/* ─── pure helpers ──────────────────────────────────────────────────── */
function formatClock(d: Date): string {
  const pad = (n: number, w = 2) => n.toString().padStart(w, "0");
  const cs = Math.floor(d.getMilliseconds() / 10);
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}.${pad(cs)}`;
}

function formatEv(ev: number): string {
  const sign = ev >= 0 ? "+" : "";
  return `${sign}${ev.toFixed(3)}`;
}

function evTone(ev: number): Tone {
  return ev >= 0 ? "g" : "r";
}

function buildPnlSeries(seed: number, len = 48): number[] {
  const out: number[] = [];
  let v = 85 + (seed % 8);
  for (let i = 0; i < len; i++) {
    v += (Math.sin(i * 0.35 + seed) * 2.2 + (i * 0.35) / len) * 1.15;
    v += (Math.random() - 0.45) * 1.8;
    out.push(v);
  }
  return out;
}

/* ─── layout primitives ─────────────────────────────────────────────── */
const panelShell = cn("flex min-h-0 min-w-0 flex-col border p-2", borderCls);

function PanelTitle({ children }: { children: ReactNode }) {
  return (
    <div
      className={cn(
        "mb-2 shrink-0 border-b pb-1.5 text-[10px] uppercase tracking-[0.2em]",
        borderCls,
        tx.g,
      )}
    >
      {children}
    </div>
  );
}

function ArbPanel({
  title,
  className,
  children,
}: {
  title: string;
  className?: string;
  children: ReactNode;
}) {
  return (
    <section className={cn(panelShell, "overflow-hidden", className)}>
      <PanelTitle>{title}</PanelTitle>
      {children}
    </section>
  );
}

function Subheading({ tone, children }: { tone: Tone; children: ReactNode }) {
  return (
    <div className={cn("mb-1 text-[10px] uppercase tracking-widest", tx[tone])}>{children}</div>
  );
}

function FeedTimestamp({ time }: { time: string }) {
  return <span className={cn("shrink-0 tabular-nums", tx.g)}>{time}</span>;
}

/* ─── feature chunks ────────────────────────────────────────────────── */
function PnlSparkline({ series, className }: { series: number[]; className?: string }) {
  const stats = useMemo(() => {
    const min = Math.min(...series);
    const max = Math.max(...series);
    const padY = (max - min) * 0.08 || 4;
    const lo = min - padY;
    const hi = max + padY;
    const w = 400;
    const h = 110;
    const pts = series.map((y, i) => {
      const x = (i / (series.length - 1)) * w;
      const t = (y - lo) / (hi - lo || 1);
      const py = h - t * (h - 8) - 4;
      return `${x.toFixed(1)},${py.toFixed(1)}`;
    });
    const last = series[series.length - 1] ?? 0;
    const ddPct = max > 0 ? Math.max(0, ((max - last) / max) * 100) : 0;
    return { pathD: `M ${pts.join(" L ")}`, cur: last, peak: max, dd: ddPct };
  }, [series]);

  return (
    <div className={cn("flex min-h-0 flex-1 flex-col", className)}>
      <svg
        viewBox="0 0 400 110"
        className="h-[min(180px,22vh)] w-full shrink-0"
        preserveAspectRatio="none"
        aria-hidden
      >
        <path
          d={stats.pathD}
          fill="none"
          stroke={T.green}
          strokeWidth="1.5"
          vectorEffect="non-scaling-stroke"
          className="drop-shadow-[0_0_6px_rgba(0,255,65,0.45)]"
        />
      </svg>
      <div
        className={cn(
          "mt-1 flex flex-wrap gap-x-4 gap-y-1 border-t border-[#1a1a1a] pt-2 text-[11px] tabular-nums",
          tx.g,
        )}
      >
        <span>CUR: +{stats.cur.toFixed(1)}</span>
        <span>PEAK: +{stats.peak.toFixed(1)}</span>
        <span>DD: {stats.dd.toFixed(1)}%</span>
      </div>
    </div>
  );
}

function useLiveDashboardState() {
  const [now, setNow] = useState(() => new Date());
  const [tick, setTick] = useState(118);

  useEffect(() => {
    const id = window.setInterval(() => setNow(new Date()), 50);
    return () => window.clearInterval(id);
  }, []);

  useEffect(() => {
    const id = window.setInterval(() => setTick((t) => (t >= 999_999 ? 0 : t + 1)), 800);
    return () => window.clearInterval(id);
  }, []);

  const timeShort = formatClock(now).slice(0, 11);
  const tickPadded = tick.toString().padStart(6, "0");

  return { now, tick, timeShort, tickPadded };
}

/* ─── page ──────────────────────────────────────────────────────────── */
export function PolymarketArbEngineDashboard() {
  const { now, timeShort, tickPadded } = useLiveDashboardState();
  const [pnlSeries] = useState(() => buildPnlSeries(Math.floor(Date.now() / 60_000)));

  const headerStats = [
    { label: "LAT", value: "8.5ms" },
    { label: "SIGNALS", value: "13" },
    { label: "TICK", value: tickPadded },
    { label: "PNL", value: "+$191.95" },
  ] as const;

  return (
    <div
      className={cn(
        "flex h-full min-h-0 flex-col gap-2 overflow-hidden bg-black p-2 font-mono text-[11px] leading-snug sm:p-3",
        tx.g,
      )}
    >
      <header
        className={cn(
          "flex shrink-0 flex-wrap items-center gap-x-4 gap-y-1 border px-3 py-2 text-[11px] sm:text-xs",
          borderCls,
        )}
      >
        <span className={cn("font-bold tracking-wide", tx.g)}>POLYMARKET ARB ENGINE v2.4.1</span>
        <span className={cn("flex items-center gap-1.5", tx.g)}>
          STATUS:
          <span className="inline-flex gap-1" aria-hidden>
            <span className="h-2 w-2 rounded-full bg-[#00FF41] shadow-[0_0_8px_#00FF41]" />
            <span className="h-2 w-2 rounded-full bg-[#00FF41] shadow-[0_0_8px_#00FF41]" />
          </span>
        </span>
        {headerStats.map(({ label, value }) => (
          <span key={label} className={tx.g}>
            {label}: <span className="tabular-nums">{value}</span>
          </span>
        ))}
        <span className={cn("ml-auto tabular-nums", tx.g)}>{formatClock(now)}</span>
      </header>

      <div className="grid min-h-0 min-w-0 flex-1 grid-rows-[minmax(0,1.2fr)_minmax(160px,0.55fr)] gap-2">
        <div className="grid min-h-0 min-w-0 grid-cols-3 gap-2">
          <ArbPanel title="MARKET SCANNER — MISPRICING DETECTION">
            <div className="min-h-0 flex-1 overflow-auto">
              <table className="w-full border-collapse text-[10px] sm:text-[11px]">
                <thead>
                  <tr className={cn("border-b text-left", borderCls, tx.c)}>
                    {SCANNER_COLUMNS.map((col) => (
                      <th key={col} className="py-1 pr-2 font-normal last:pr-0">
                        {col}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {SCANNER_ROWS.map((row) => (
                    <tr key={row.market} className="border-b border-[#1a1a1a]/80">
                      <td className={cn("max-w-[140px] truncate py-1 pr-2", tx.g)} title={row.market}>
                        {row.market}
                      </td>
                      <td className={cn("py-1 pr-2 tabular-nums", tx.g)}>{row.impl}</td>
                      <td className={cn("py-1 pr-2 tabular-nums", tx.c)}>{row.base}</td>
                      <td className={cn("py-1 pr-2 tabular-nums", tx.c)}>{row.ttr}</td>
                      <td className={cn("py-1 pr-2 tabular-nums", tx[evTone(row.ev)])}>{formatEv(row.ev)}</td>
                      <td className={cn("whitespace-nowrap py-1", tx[LIQ_TONE[row.liq]])}>{row.liq}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </ArbPanel>

          <ArbPanel title="EXECUTION FEED">
            <ul className="min-h-0 flex-1 space-y-1 overflow-auto text-[10px] sm:text-[11px]">
              {EXEC_LINES.map((line, i) => (
                <li key={`${line.msg}-${i}`} className="flex gap-2">
                  <FeedTimestamp time={timeShort} />
                  <span className={tx[EXEC_TAG_TONE[line.tag]]}>[{line.tag}]</span>
                  <span className={tx.g}>{line.msg}</span>
                </li>
              ))}
            </ul>
          </ArbPanel>

          <ArbPanel title="MODEL OUTPUT / OPPORTUNITIES">
            <div className="space-y-1 text-[10px] sm:text-[11px] overflow-y-auto overflow-x-hidden">
              {MODEL_LINES.map(({ line, tone }) => (
                <div key={line} className={tx[tone]}>
                  {line}
                </div>
              ))}
            </div>
            <div>
              <Subheading tone="c">REGIME DETECTION</Subheading>
              <div className={tx.c}>[MODE] VOLATILITY_ARB</div>
            </div>
            <div>
              <Subheading tone="g">TOP OPPORTUNITIES</Subheading>
              <ul className="space-y-0.5 text-[10px] sm:text-[11px]">
                {TOP_OPPS.map((o) => (
                  <li key={o.name} className="flex justify-between gap-2">
                    <span className={tx.g}>{o.name}</span>
                    <span className={cn("tabular-nums", tx[evTone(o.ev)])}>{formatEv(o.ev)}</span>
                  </li>
                ))}
              </ul>
            </div>
            <div>
              <Subheading tone="g">SYSTEM DIAGNOSTICS</Subheading>
              <ul className="space-y-0.5 text-[10px] sm:text-[11px]">
                {DIAGNOSTIC_LINES.map((line) => (
                  <li key={line} className={tx.g}>
                    {line}
                  </li>
                ))}
                <li className={cn("mt-1 tabular-nums", tx.c)}>
                  LAT 8.5ms · TICK {tickPadded}
                </li>
              </ul>
            </div>
          </ArbPanel>
        </div>

        <div className="grid min-h-0 min-w-0 grid-cols-2 gap-2">
          <ArbPanel title="PNL GRAPH — CUMULATIVE EDGE EXTRACTION">
            <PnlSparkline series={pnlSeries} className="min-h-[140px]" />
          </ArbPanel>

          <ArbPanel title="WALLET FEED / SIGNAL SCANNER">
            <ul className="min-h-0 flex-1 space-y-1 overflow-auto text-[10px] sm:text-[11px]">
              {WALLET_LINES.map((w, i) => (
                <li key={`${w.addr}-${i}`} className="flex flex-wrap gap-x-2 gap-y-0">
                  <FeedTimestamp time={timeShort} />
                  <span className={tx.c}>{w.addr}</span>
                  <span className={tx[w.tone]}>{w.msg}</span>
                </li>
              ))}
              <li className={cn("mt-1 border-t border-[#1a1a1a] pt-1", tx.r)}>
                CLUSTER 4 wallets detected | LEAD: 0xF17a..b990
              </li>
            </ul>
          </ArbPanel>
        </div>
      </div>
    </div>
  );
}

export default PolymarketArbEngineDashboard;
