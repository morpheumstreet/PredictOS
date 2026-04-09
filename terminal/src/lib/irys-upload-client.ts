/**
 * Browser-safe helpers for Irys upload payloads (no @irys/* or Node-only APIs).
 * Actual chain upload runs only in POST /api/irys-upload.
 */

import type {
  PmType,
  IrysAgentData,
  IrysCombinedUploadPayload,
} from "@/types/agentic";

export interface IrysUploadResult {
  success: boolean;
  transactionId?: string;
  gatewayUrl?: string;
  error?: string;
  environment?: "mainnet" | "devnet";
}

export function generateRequestId(): string {
  const timestamp = Date.now().toString(36);
  const randomPart = Math.random().toString(36).substring(2, 15);
  const secondRandom = Math.random().toString(36).substring(2, 15);
  return `pred-${timestamp}-${randomPart}${secondRandom}`;
}

export function formatCombinedAnalysisForUpload(
  agentsData: IrysAgentData[],
  metadata: {
    requestId: string;
    pmType: PmType;
    eventIdentifier: string;
    eventId?: string;
    analysisMode: "supervised" | "autonomous";
  }
): IrysCombinedUploadPayload {
  return {
    requestId: metadata.requestId,
    timestamp: new Date().toISOString(),
    pmType: metadata.pmType,
    eventIdentifier: metadata.eventIdentifier,
    eventId: metadata.eventId,
    analysisMode: metadata.analysisMode,
    agentsData,
    schemaVersion: "1.0.0",
  };
}

export function getGatewayUrl(transactionId: string): string {
  return `https://gateway.irys.xyz/${transactionId}`;
}
