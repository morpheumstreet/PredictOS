import { AlertTriangle } from "lucide-react";

type Props = { message: string | null };

export function AgentsErrorBanner({ message }: Props) {
  if (!message) return null;
  return (
    <div className="flex items-start gap-2 p-3 rounded-lg border border-destructive/50 bg-destructive/10 text-sm text-destructive">
      <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" />
      <span>{message}</span>
    </div>
  );
}
