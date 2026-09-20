import { ShieldCheck } from "lucide-react";
import { Badge } from "../../components/ui";
import { formatDate } from "../../lib/utils";
import type { StoreApp } from "../../types";

/**
 * The label a cleared app carries. It reads from the app itself, which the
 * server only marks verified while the approval still covers the current
 * content — so an edited app loses the label without anything sweeping for it.
 */
export function SecurityVerifiedBadge({
  app,
  showDate = false,
}: {
  app: Pick<StoreApp, "securityVerified" | "securityVerifiedAt">;
  showDate?: boolean;
}) {
  if (!app.securityVerified) return null;
  return (
    <Badge tone="positive">
      <ShieldCheck size={14} aria-hidden="true" /> 보안 심의 완료
      {showDate && app.securityVerifiedAt
        ? ` · ${formatDate(app.securityVerifiedAt)}`
        : ""}
    </Badge>
  );
}
