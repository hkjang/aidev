export type Template = {
  id: string;
  workspace_id?: string;
  space_id?: string | null;
  owner_id?: string;
  name: string;
  description: string;
  category: string;
  icon: string;
  kind: string;
  markdown?: string;
  tags: string[];
  visibility?: string;
  version: number;
  builtin: boolean;
  can_write?: boolean;
  can_manage?: boolean;
  owner_name?: string;
  updated_at?: string;
  deleted_at?: string | null;
  protection?: { changed: boolean; mode: string; findings: unknown[] };
};
export type TemplateSpace = { id: string; name: string; can_write: boolean };
export const kinds: Record<string, string> = {
  page: "일반 문서",
  note: "노트",
  daily: "일일 노트",
  meeting: "회의록",
  decision: "의사결정 기록",
  runbook: "운영 런북",
};
export const scopes: Record<string, string> = {
  private: "나만 보기",
  workspace: "워크스페이스 공유",
  space: "공간 공유",
};
export const blankTemplate: Template = {
  id: "",
  name: "",
  description: "",
  category: "",
  icon: "file",
  kind: "page",
  markdown: "# 새 템플릿\n\n",
  tags: [],
  visibility: "private",
  space_id: "",
  version: 0,
  builtin: false,
};
