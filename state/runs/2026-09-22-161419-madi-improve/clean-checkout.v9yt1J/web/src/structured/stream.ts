import { ApiError } from "../api";
import type { StructuredProposal } from "./types";

/** A ticket becomes actionable only after both a valid proposal and [DONE]. */
export async function streamStructuredProposal(
  path: string,
  input: unknown,
  signal: AbortSignal,
): Promise<StructuredProposal> {
  const response = await fetch(`/api/v1${path}`, {
    method: "POST",
    credentials: "same-origin",
    signal,
    headers: { "Content-Type": "application/json", "X-Madi-Request": "1" },
    body: JSON.stringify(input),
  });
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new ApiError(
      body.error || "구조화 제안을 생성하지 못했습니다.",
      response.status,
    );
  }
  if (
    !response.body ||
    !response.headers.get("Content-Type")?.includes("text/event-stream")
  )
    throw Error("AI 스트리밍 응답이 필요합니다.");
  const reader = response.body.getReader(),
    decoder = new TextDecoder();
  let buffer = "",
    completed = false,
    proposal: StructuredProposal | null = null;
  try {
    while (true) {
      const chunk = await reader.read();
      if (chunk.done) break;
      buffer = (buffer + decoder.decode(chunk.value, { stream: true })).replace(
        /\r\n/g,
        "\n",
      );
      if (buffer.length > 1 << 20)
        throw Error("AI 이벤트가 허용 크기를 초과했습니다.");
      let end: number;
      while ((end = buffer.indexOf("\n\n")) >= 0) {
        const event = buffer.slice(0, end);
        buffer = buffer.slice(end + 2);
        const raw = event
          .split("\n")
          .filter((line) => line.startsWith("data:"))
          .map((line) => line.slice(5).trimStart())
          .join("\n");
        if (!raw) continue;
        if (raw === "[DONE]") {
          if (!proposal)
            throw Error(
              "완료 확인보다 검토 가능한 제안이 먼저 도착해야 합니다.",
            );
          completed = true;
          continue;
        }
        if (completed)
          throw Error("완료 뒤에 추가된 AI 응답은 사용할 수 없습니다.");
        const value = JSON.parse(raw);
        if (value.error || value.retract)
          throw Error(value.error || "권한·원문 변경으로 제안을 회수했습니다.");
        // Progress carries no model text. Ignore even legacy raw deltas: only
        // server-validated typed fields may become visible after [DONE].
        if (value.proposal) {
          const next = value.proposal as StructuredProposal;
          if (
            typeof next.draft_ticket !== "string" ||
            !next.draft_ticket ||
            next.draft_ticket.length > 1 << 20 ||
            !Array.isArray(next.fields) ||
            next.fields.length < 1 ||
            next.fields.length > 32 ||
            next.automatic_apply !== false ||
            !Number.isInteger(next.source_version) ||
            next.source_version < 1
          )
            throw Error("완료된 제안의 형식을 확인할 수 없습니다.");
          proposal = next;
        }
      }
    }
    if (!completed || !proposal || signal.aborted)
      throw Error(
        "응답이 완료되지 않았습니다. 중간 결과는 보관·반영할 수 없습니다.",
      );
    return proposal;
  } catch (error) {
    await reader.cancel().catch(() => {});
    throw error;
  } finally {
    reader.releaseLock();
  }
}
