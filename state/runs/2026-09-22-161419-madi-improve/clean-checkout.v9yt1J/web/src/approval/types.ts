export type Gate = { name: string; kind: string; id?: string; role?: string };
export type Stage = { name: string; mode: "any" | "all"; gates: Gate[] };
export type Policy = {
  id: string;
  workspace_id: string;
  space_id: string;
  resource_kind: string;
  name: string;
  enabled: boolean;
  version: number;
  stages: Stage[];
};
export type Request = {
  id: string;
  resource_kind: string;
  resource_id: string;
  resource_version: number;
  requester_id: string;
  owner_id: string;
  policy: Policy;
  status: string;
  current_stage: number;
  version: number;
  comment: string;
  snapshot?: Record<string, any>;
  review_context?: Record<string, any>;
  stale?: boolean;
  reason?: string;
  eligible_gates?: number[];
  assignments?: Record<string, any>[];
  decisions?: Record<string, any>[];
  title?: string;
  document_id?: string;
};
export type ApprovalStatus = {
  enabled: boolean;
  policy: Policy;
  resource_title: string;
  resource_version: number;
  request: Request | null;
  history: Request[];
  eligible_gates: number[];
  stale: boolean;
  reason?: string;
  assignments?: Record<string, any>[];
  decisions?: Record<string, any>[];
};
export const statusNames: Record<string, string> = {
  pending: "검토 중",
  approved: "승인 완료",
  rejected: "반려",
  cancelled: "취소",
  superseded: "이전 요청",
  consumed: "실행에 사용됨",
};
export const kindNames: Record<string, string> = {
  document: "문서 게시",
  runbook: "격리 실행",
  sql_query_plan: "SQL 계획",
  impact_exception: "변경 영향 예외",
  knowledge_distribution: "망간 지식 배포",
  learning_step: "지식 경로 실습 검토",
};
