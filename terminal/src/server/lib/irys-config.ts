/**
 * Terminal-side check for Irys upload proxy (Go service URL).
 * Solana keys and RPC live only on the irys-upload process — see mm/irys-upload/README.md.
 */

function normalizeBaseURL(raw: string): string {
  return raw.replace(/\/+$/, "");
}

export function getIrysUploadServiceBaseURL(): string | undefined {
  const u = process.env.IRYS_UPLOAD_SERVICE_URL?.trim();
  if (!u) return undefined;
  try {
    const parsed = new URL(u);
    if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
      return undefined;
    }
    return normalizeBaseURL(parsed.toString());
  } catch {
    return undefined;
  }
}

export function validateIrysUploadProxyConfig(): { valid: boolean; error?: string } {
  const base = getIrysUploadServiceBaseURL();
  if (!base) {
    return {
      valid: false,
      error:
        "IRYS_UPLOAD_SERVICE_URL is not set or invalid (expected http(s) URL of the Go irys-upload service)",
    };
  }
  return { valid: true };
}
