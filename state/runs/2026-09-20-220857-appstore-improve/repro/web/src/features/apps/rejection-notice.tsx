import { TriangleAlert } from "lucide-react";
import { cn, formatDateTime } from "../../lib/utils";
import type { StoreApp } from "../../types";

// Owners cannot open the review queue, so a rejected app would otherwise say
// only "반려됨". /me/apps carries the review that rejected it; this shows the
// reason wherever the owner is standing next to the edit button.
export function RejectionNotice({
  app,
  className,
}: {
  app: StoreApp;
  className?: string;
}) {
  const review = app.review;
  if (app.status !== "rejected" || review?.status !== "rejected") return null;
  const meta = [
    review.decidedAt ? formatDateTime(review.decidedAt) : undefined,
    review.reviewerName || undefined,
    review.level ? `${review.level}단계 검토` : undefined,
  ].filter(Boolean);
  return (
    <div className={cn("notice notice-danger", className)} role="alert">
      <TriangleAlert size={19} aria-hidden="true" />
      <span>
        <strong>반려 사유</strong>
        <br />
        <span className="whitespace-pre-line">
          {review.reason ||
            "사유가 기록되지 않았습니다. 검토자에게 문의하세요."}
        </span>
        {!!meta.length && (
          <>
            <br />
            <span className="text-[13px] opacity-80">{meta.join(" · ")}</span>
          </>
        )}
      </span>
    </div>
  );
}
