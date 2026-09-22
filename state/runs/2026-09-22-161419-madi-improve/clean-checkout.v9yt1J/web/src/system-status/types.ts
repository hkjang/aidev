export type Policy = {
  revision: number;
  enabled: boolean;
  max_ttl_seconds: number;
  max_observation_age_seconds: number;
  retention_days: number;
};
export type Expected = {
  name: string;
  environment: string;
  version: string;
  deployment: string;
  note: string;
};
export type Observation = {
  version: string;
  deployment: string;
  health: "healthy" | "degraded" | "down" | "unknown";
  note: string;
};
export type Card = {
  document_id: string;
  workspace_id: string;
  revision: number;
  document_version: number;
  verification_epoch: number;
  report_revision: number;
  owner_id: string;
  reporter_ids: string[];
  ttl_seconds: number;
  expected: Expected;
  updated_at: string;
};
export type Report = {
  id: string;
  request_id: string;
  revision: number;
  card_revision: number;
  verification_epoch: number;
  document_version: number;
  actor_id: string;
  actor_kind: string;
  observed_at: string;
  received_at: string;
  observation: Observation;
  expected_at_report: Expected;
};
export type StatusState = {
  policy: Policy;
  current_document_version: number;
  card: Card | null;
  reports: Report[];
  latest_report: Report | null;
  summary: {
    state: string;
    fresh: boolean;
    drift_fields?: string[];
    expires_at?: string;
    independently_verified: false;
  };
  can_manage: boolean;
  can_report: boolean;
  reporter_options: { id: string; name: string; kind: string }[];
  reporter_options_truncated?: boolean;
  notice: string;
  context: {
    runbook: null | {
      id: string;
      phase: string;
      status: string;
      created_at: string;
      completed_at: string | null;
    };
    connector_records: {
      connector_id: string;
      kind: string;
      document_version: number;
      last_seen_at: string;
      content_checksum: string;
      enabled: boolean;
    }[];
    notice: string;
  };
};
export const stateNames: Record<string, string> = {
  unregistered: "카드 미등록",
  unobserved: "아직 관측 보고 없음",
  disabled: "보고 정책 비활성",
  document_changed: "원문 변경 · 기준 재검토",
  baseline_changed: "기준 변경 · 새 관측 필요",
  stale: "유효 시간 지난 관측",
  reported_match: "신선한 보고 · 기대값 일치",
  drift: "신선한 보고 · 기대값 불일치",
};
export const healthNames: Record<string, string> = {
  healthy: "정상이라고 보고됨",
  degraded: "저하라고 보고됨",
  down: "중단이라고 보고됨",
  unknown: "상태 확인 안 됨",
};
export const actorNames: Record<string, string> = {
  manual: "사용자 수동 보고",
  api_user: "사용자 API 보고",
  service_account: "서비스 계정 API 보고",
};
export const formatTime = (value?: string | null) =>
  value ? new Date(value).toLocaleString("ko-KR") : "기록 없음";
