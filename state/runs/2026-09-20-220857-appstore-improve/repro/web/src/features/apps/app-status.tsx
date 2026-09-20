import { Badge } from "../../components/ui";

export const APP_STATUSES = [
  { value: "draft", label: "초안", tone: undefined },
  { value: "pending_review", label: "검토 대기", tone: "warning" },
  { value: "published", label: "게시됨", tone: "positive" },
  { value: "rejected", label: "반려", tone: "danger" },
  { value: "archived", label: "보관됨", tone: undefined },
] as const;

export function AppStatusBadge({ status }: { status?: string }) {
  const entry = APP_STATUSES.find((item) => item.value === status);
  return <Badge tone={entry?.tone}>{entry?.label ?? status ?? "—"}</Badge>;
}
