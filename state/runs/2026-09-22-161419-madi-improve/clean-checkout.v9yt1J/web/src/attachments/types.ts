export type Position = {
  page?: number;
  slide?: number;
  sheet?: string;
  cell?: string;
  paragraph?: number;
  table?: number;
  row?: number;
  column?: number;
  bounds?: number[];
  page_size?: number[];
  ocr?: boolean;
};
export type Fragment = {
  id: string;
  extraction_id: string;
  ordinal: number;
  text: string;
  content_hash: string;
  position: Position;
};
export type Run = {
  id: string;
  status: string;
  revision: number;
  format: string;
  checksum: string;
  actor_id: string;
  job_id: string;
  created_at: string;
  error: string;
  result: {
    warnings?: string[];
    empty_pages?: number[];
    pages?: number;
    fragments?: number;
    masked?: boolean;
  };
};
export type Policy = {
  revision: number;
  data: {
    enabled: boolean;
    ocr_enabled: boolean;
    max_file_bytes: number;
    max_pages: number;
    max_text_bytes: number;
    max_fragments: number;
    timeout_seconds: number;
  };
};
export type Context = {
  id: string;
  document_id: string;
  workspace_id: string;
  document_title: string;
  document_version: number;
  checksum: string;
  name: string;
  size: number;
  format: string;
  can_write: boolean;
  policy: Policy;
  runs: Run[];
};
export const statusName: Record<string, string> = {
  queued: "대기",
  running: "추출 중",
  ready: "완료",
  failed: "실패",
  cancelled: "취소",
  obsolete: "이전 결과",
};
export function positionName(p: Position) {
  return (
    [
      p.page && `${p.page}쪽`,
      p.slide && `슬라이드 ${p.slide}`,
      p.sheet && `시트 ${p.sheet}`,
      p.cell && `셀 ${p.cell}`,
      p.table && `표 ${p.table}`,
      p.row && `${p.row}행`,
      p.column && `${p.column}열`,
      p.paragraph && `문단 ${p.paragraph}`,
      p.ocr && "OCR",
    ]
      .filter(Boolean)
      .join(" · ") || "본문"
  );
}
