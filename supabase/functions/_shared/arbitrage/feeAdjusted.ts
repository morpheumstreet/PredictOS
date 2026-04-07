/**
 * Post-process AI arbitrage output with per-platform trading fees (basis points on each leg's premium).
 * Configure via Supabase secrets / env:
 *   ARBITRAGE_POLYMARKET_FEE_BPS (default 0)
 *   ARBITRAGE_KALSHI_FEE_BPS (default 0)
 *   ARBITRAGE_MIN_NET_PROFIT_USD — require net profit after fees to exceed this (default 0)
 */

import type {
  ArbitrageMarketSource,
  ArbitrageOpportunity,
} from "../../arbitrage-finder/types.ts";

export interface ArbitrageFeeConfig {
  polymarketFeeBps: number;
  kalshiFeeBps: number;
  minNetProfitUsd: number;
}

function readNonNegativeInt(name: string, defaultValue: number): number {
  const raw = Deno.env.get(name);
  if (raw === undefined || raw === "") return defaultValue;
  const n = Number(raw);
  if (!Number.isFinite(n) || n < 0) return defaultValue;
  return Math.round(n);
}

function readNonNegativeFloat(name: string, defaultValue: number): number {
  const raw = Deno.env.get(name);
  if (raw === undefined || raw === "") return defaultValue;
  const n = Number(raw);
  if (!Number.isFinite(n) || n < 0) return defaultValue;
  return n;
}

export function getArbitrageFeeConfig(): ArbitrageFeeConfig {
  return {
    polymarketFeeBps: readNonNegativeInt("ARBITRAGE_POLYMARKET_FEE_BPS", 0),
    kalshiFeeBps: readNonNegativeInt("ARBITRAGE_KALSHI_FEE_BPS", 0),
    minNetProfitUsd: readNonNegativeFloat("ARBITRAGE_MIN_NET_PROFIT_USD", 0),
  };
}

function bpsForPlatform(
  source: ArbitrageMarketSource,
  config: ArbitrageFeeConfig
): number {
  return source === "polymarket" ? config.polymarketFeeBps : config.kalshiFeeBps;
}

/**
 * Adds fee-adjusted totals to an arbitrage result. Gross fields from the model are unchanged.
 */
export function enrichArbitrageWithFees(
  arbitrage: ArbitrageOpportunity,
  config: ArbitrageFeeConfig = getArbitrageFeeConfig()
): ArbitrageOpportunity {
  const strategy = arbitrage.strategy;
  if (!strategy) {
    return { ...arbitrage };
  }

  const {
    buyYesOn,
    buyNoOn,
    buyYesPrice,
    buyNoPrice,
    totalCost,
    guaranteedPayout,
  } = strategy;

  const yesBps = bpsForPlatform(buyYesOn, config);
  const noBps = bpsForPlatform(buyNoOn, config);

  const estimatedFeeYes = buyYesPrice * (yesBps / 10000);
  const estimatedFeeNo = buyNoPrice * (noBps / 10000);
  const totalFees = estimatedFeeYes + estimatedFeeNo;

  const totalCostAfterFees = totalCost + totalFees;
  const netProfitAfterFees = guaranteedPayout - totalCostAfterFees;
  const profitPercentAfterFees =
    totalCostAfterFees > 0 ? (netProfitAfterFees / totalCostAfterFees) * 100 : null;

  const viableAfterFees = netProfitAfterFees > config.minNetProfitUsd;

  return {
    ...arbitrage,
    feeAdjusted: {
      polymarketFeeBps: config.polymarketFeeBps,
      kalshiFeeBps: config.kalshiFeeBps,
      minNetProfitUsd: config.minNetProfitUsd,
      estimatedFeeYes,
      estimatedFeeNo,
      totalFees,
      totalCostAfterFees,
      netProfitAfterFees,
      profitPercentAfterFees,
      viableAfterFees,
    },
  };
}
