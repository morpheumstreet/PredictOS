import { getWalletTrackingBackendState } from "@/server/api/wallet-tracking";

export async function GET() {
  const state = getWalletTrackingBackendState();
  return Response.json({
    ok: state.dome_configured,
    ...state,
  });
}
