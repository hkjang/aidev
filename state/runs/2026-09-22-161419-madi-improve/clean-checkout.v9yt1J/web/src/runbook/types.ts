export type Parameter = {
  name: string;
  label: string;
  type: "string" | "integer" | "boolean" | "enum";
  required: boolean;
  options: string[];
  pattern: string;
  min?: number;
  max?: number;
};
export type Action = {
  id: string;
  name: string;
  roles: string[];
  teams: string[];
  parameters: Parameter[];
  timeout_seconds: number;
  template_id: number;
  image: string;
  argv: string[];
  cpu_milli: number;
  memory_mi: number;
};
export type Runner = {
  id: string;
  workspace_id: string;
  name: string;
  kind: "awx" | "kubernetes";
  enabled: boolean;
  revision: number;
  config: Record<string, any>;
  actions: Action[];
};
export type Step = {
  name: string;
  runner_id: string;
  action_id: string;
  parameters: Record<string, unknown>;
};
export type Definition = {
  document_id: string;
  version: number;
  purpose: string;
  prerequisites: string;
  validation: string;
  rollback: string;
  steps: Step[];
  validation_steps: Step[];
  rollback_steps: Step[];
};
export type Runbook = {
  definition: Definition;
  title: string;
  workspace_id: string;
  owner_id: string;
  can_write: boolean;
  last_tested_at: string | null;
  document_version: number;
};
export type Execution = {
  id: string;
  owner_id: string;
  document_id: string;
  phase: string;
  version: number;
  status: string;
  approval_required: boolean;
  approval_id: string | null;
  approval?: { id: string; version: number; status: string };
  can_confirm: boolean;
  can_read_logs: boolean;
  cancel_requested: boolean;
  last_error?: string;
  created_at: string;
  completed_at: string | null;
  snapshot: {
    title: string;
    document_version: number;
    definition_version: number;
    steps: {
      name: string;
      kind: string;
      runner_id: string;
      runner_revision: number;
      action: Action;
      parameters: Record<string, unknown>;
      argv?: string[];
      remote: Record<string, any>;
    }[];
  };
  steps?: { step_index: number; state: string; external_id: string }[];
};
export const statusNames: Record<string, string> = {
  prepared: "실행 준비",
  review: "검토 중",
  approved: "승인 완료",
  rejected: "반려",
  cancelled: "취소 완료",
  queued: "작업 대기",
  running: "실행 중",
  succeeded: "완료",
  failed: "실패",
  unknown: "외부 실행 확인 필요",
  launching: "시작 요청 중",
};
export const phaseNames: Record<string, string> = {
  execute: "실행",
  validate: "검증",
  rollback: "롤백",
};
