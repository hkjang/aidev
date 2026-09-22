export type StructuredProperty = {
  id: string;
  name: string;
  type: string;
  options?: string[];
};
export type StructuredField = {
  property_id: string;
  value: unknown;
  quote: string;
  start_byte: number;
  end_byte: number;
  content_hash: string;
  issue: string;
  valid: boolean;
  human_edited: boolean;
};
export type StructuredProvider = {
  configured: boolean;
  base_url: string;
  model: string;
  fingerprint: string;
  max_tokens: number;
};
export type StructuredContext = {
  document_id: string;
  database_id: string;
  workspace_id: string;
  version: number;
  title: string;
  markdown: string;
  database_name: string;
  space_id: string;
  schema_hash: string;
  destination_hash: string;
  properties: StructuredProperty[];
  provider: StructuredProvider;
  max_selection_bytes: number;
  destination_notice: string;
};
export type StructuredProposal = {
  draft_ticket: string;
  fields: StructuredField[];
  source_version: number;
  expires_at: string;
  automatic_apply: false;
};
export type StructuredDraft = {
  id: string;
  document_id: string;
  database_id: string;
  workspace_id: string;
  title: string;
  database_name: string;
  source_version: number;
  current_version: number;
  source_hash: string;
  start_byte: number;
  end_byte: number;
  fields: StructuredField[];
  reviewed_fields: StructuredField[];
  properties: StructuredProperty[];
  model: string;
  state: "draft" | "committed";
  revision: number;
  fresh: boolean;
  expires_at: string;
  created_at: string;
  row_id: string;
  integrity_notice: string;
};
export type StructuredListItem = Pick<
  StructuredDraft,
  | "id"
  | "document_id"
  | "title"
  | "database_id"
  | "database_name"
  | "state"
  | "fresh"
  | "revision"
  | "created_at"
>;
export type StructuredPreview = {
  review_ticket: string;
  fields: StructuredField[];
  values: Record<string, unknown>;
  database_id: string;
  database_name: string;
  source_version: number;
  expires_at: string;
  destination_notice: string;
};
export const propertyLabels: Record<string, string> = {
  text: "텍스트",
  number: "숫자",
  checkbox: "체크",
  select: "선택",
  status: "상태",
  multi_select: "다중 선택",
  multiselect: "다중 선택",
  date: "날짜",
  url: "주소",
  email: "이메일",
  phone: "전화번호",
  progress: "진행률",
};
