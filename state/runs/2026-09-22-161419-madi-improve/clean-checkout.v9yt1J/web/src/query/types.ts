export type QuerySource = "documents" | "tasks" | "relations";
export type QueryColumn = { field: string; label?: string };
export type QueryParameter = {
  type: "string" | "number" | "boolean" | "date";
  label?: string;
  default?: string | number | boolean;
  required?: boolean;
};
export type QueryDefinition = {
  version: 1;
  source: QuerySource;
  columns: QueryColumn[];
  parameters?: Record<string, QueryParameter>;
  limit: number;
};
export type QueryMetadata = {
  document_version: number;
  queries: {
    hash: string;
    source: string;
    line: number;
    definition?: QueryDefinition;
    error?: string;
  }[];
  truncated: boolean;
  notice: string;
};
export type QueryResult = {
  document_version: number;
  query_hash: string;
  columns: QueryColumn[];
  rows: {
    values: Record<string, unknown>;
    document_id: string;
    version: number;
    line?: number;
  }[];
  validation_token: string;
  executed_at: string;
  expires_at: string;
  duration_ms: number;
  diagnostics: {
    candidates: number;
    documents_scanned: number;
    scanned_bytes: number;
    oversized_documents: number;
    invalid_properties: number;
    truncated_fields: number;
    truncated: boolean;
    notice: string;
  };
};
export const queryLabels: Record<string, string> = {
  id: "문서 ID",
  title: "문서",
  status: "상태",
  tags: "태그",
  owner_id: "소유자 ID",
  owner_name: "소유자",
  created_at: "만든 시각",
  updated_at: "수정 시각",
  classification: "보안 등급",
  kind: "문서 유형",
  version: "원문 버전",
  document_id: "문서 ID",
  document_title: "문서",
  text: "할 일",
  done: "완료 여부",
  priority: "우선순위",
  assignee_id: "담당자 ID",
  assignee_name: "담당자",
  due_date: "기한",
  line: "원문 줄",
  source_id: "출발 문서 ID",
  source_title: "출발 문서",
  target_id: "대상 문서 ID",
  target_title: "대상 문서",
  relation_type: "관계 유형",
  depth: "경로 깊이",
};
export const querySourceLabels: Record<QuerySource, string> = {
  documents: "문서 속성",
  tasks: "할 일",
  relations: "문서 관계",
};
