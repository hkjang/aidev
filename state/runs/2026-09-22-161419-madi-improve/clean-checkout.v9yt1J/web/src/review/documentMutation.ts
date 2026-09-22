import { api, ApiError, type Doc } from "../api";
/** Partial CAS mutation: omitted title/tags/visibility/aliases/space stay server-owned. */
export function applyMarkdownProposal(
  doc: Doc,
  markdown: string,
  expectedVersion: number,
) {
  if (doc.version !== expectedVersion)
    throw new ApiError(
      "문서 버전이 바뀌었습니다. 제안을 보관하고 현재 원문에서 다시 선택하세요.",
      409,
    );
  if (!doc.can_write)
    throw new ApiError("현재 문서 수정 권한이 필요합니다.", 403);
  return api<
    Doc & {
      protection?: { changed?: boolean; mode?: string; findings?: unknown[] };
    }
  >(`/documents/${doc.id}`, "PUT", { version: expectedVersion, markdown });
}
