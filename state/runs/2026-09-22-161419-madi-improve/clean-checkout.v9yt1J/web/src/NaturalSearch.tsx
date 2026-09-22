import { useEffect, useRef, useState } from "react";
import { Sparkles, Square } from "lucide-react";
import { useApp } from "./context";
import { Button, ErrorBox, Field, Modal } from "./ui";

type Proposal = {
  explanation: string;
  query: string;
  plan: Record<string, string | boolean>;
  date_basis: string;
};
const labels: Record<string, string> = {
  q: "검색어",
  type: "종류",
  from: "시작일",
  to: "종료일",
  tag: "태그",
  status: "문서 상태",
  has_attachment: "첨부파일",
  sort: "정렬",
};
const values: Record<string, string> = {
  document: "문서",
  block: "블록",
  code: "코드",
  task: "할 일",
  file: "파일",
  comment: "댓글",
  tag: "태그",
  database: "데이터베이스",
  row: "데이터 행",
  user: "사용자",
  ai_conversation: "내 AI 대화",
  draft: "초안",
  review: "검토 중",
  published: "게시됨",
  rejected: "반려됨",
  stale: "검토 필요",
  archived: "보관됨",
  relevance: "관련성",
  newest: "최근 수정 순",
  oldest: "오래된 순",
  title: "제목 순",
};
export default function NaturalSearch({
  initialQuery,
  onApply,
}: {
  initialQuery: string;
  onApply: (query: string) => void;
}) {
  const { workspace, user } = useApp();
  const [open, setOpen] = useState(false),
    [prompt, setPrompt] = useState(""),
    [consent, setConsent] = useState(false),
    [busy, setBusy] = useState(false),
    [raw, setRaw] = useState(""),
    [proposal, setProposal] = useState<Proposal | null>(null),
    [error, setError] = useState("");
  const controller = useRef<AbortController | null>(null);
  useEffect(() => {
    controller.current?.abort();
    controller.current = null;
    setBusy(false);
    setRaw("");
    setProposal(null);
    setError("");
    setConsent(false);
    setPrompt(initialQuery);
    return () => controller.current?.abort();
  }, [open, workspace?.id, user.id]);
  async function generate() {
    if (!workspace || busy || !consent || !prompt.trim()) return;
    controller.current?.abort();
    const request = new AbortController();
    controller.current = request;
    setBusy(true);
    setRaw("");
    setProposal(null);
    setError("");
    try {
      const response = await fetch("/api/v1/search/ai/proposal", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json", "X-Madi-Request": "1" },
        body: JSON.stringify({
          workspace_id: workspace.id,
          prompt,
          consent: true,
        }),
        signal: request.signal,
      });
      if (!response.ok) {
        const body = await response
          .json()
          .catch(() => ({ error: "검색 조건을 제안하지 못했습니다" }));
        throw new Error(body.error);
      }
      if (!response.body) throw new Error("스트리밍 응답이 없습니다");
      const reader = response.body.getReader(),
        decoder = new TextDecoder();
      let buffer = "",
        completed = false,
        proposed: Proposal | null = null;
      while (true) {
        const { value, done } = await reader.read();
        if (request.signal.aborted || controller.current !== request) return;
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() || "";
        for (const line of lines) {
          if (!line.startsWith("data:")) continue;
          const data = line.slice(5).trim();
          if (!data) continue;
          if (data === "[DONE]") {
            completed = true;
            continue;
          }
          const event = JSON.parse(data);
          if (event.retract) {
            setRaw("");
            setProposal(null);
            proposed = null;
          }
          if (event.error) throw new Error(event.error);
          if (event.text) setRaw((v) => v + event.text);
          if (event.proposal) proposed = event.proposal;
        }
      }
      if (!completed || !proposed)
        throw new Error("완료되지 않은 검색 제안은 적용할 수 없습니다");
      setProposal(proposed);
    } catch (e) {
      if (!request.signal.aborted && controller.current === request) {
        setError((e as Error).message);
        setProposal(null);
      }
    } finally {
      if (controller.current === request) {
        controller.current = null;
        setBusy(false);
      }
    }
  }
  return (
    <>
      <Button onClick={() => setOpen(true)} disabled={!workspace}>
        <Sparkles size={17} />
        자연어 검색
      </Button>
      <Modal
        open={open}
        onOpenChange={setOpen}
        title="자연어로 검색 조건 만들기"
      >
        <form
          onSubmit={(e) => {
            e.preventDefault();
            void generate();
          }}
        >
          <p>
            질문을 검색어·날짜·종류 조건으로 제안합니다. 아직 검색을 실행하지
            않으며, 문서 본문이나 목록은 AI에 전송하지 않습니다.
          </p>
          <Field label="찾고 싶은 지식">
            <textarea
              rows={3}
              maxLength={4000}
              value={prompt}
              disabled={busy}
              onChange={(e) => {
                setPrompt(e.target.value);
                setProposal(null);
              }}
              placeholder="지난달 GPU 장애 대응 문서를 최근 수정 순으로 찾아줘"
              required
            />
          </Field>
          <label className="check-label">
            <input
              type="checkbox"
              checked={consent}
              disabled={busy}
              onChange={(e) => setConsent(e.target.checked)}
            />
            입력한 질문을 관리자가 설정한 AI 공급자에 전송합니다.
          </label>
          <ErrorBox error={error} />
          <div className="modal-actions">
            {busy ? (
              <Button
                type="button"
                onClick={() => {
                  controller.current?.abort();
                  controller.current = null;
                  setBusy(false);
                  setRaw("");
                  setProposal(null);
                }}
              >
                <Square size={16} />
                생성 중단
              </Button>
            ) : (
              <Button variant="primary" disabled={!consent || !prompt.trim()}>
                <Sparkles size={17} />
                {proposal ? "다시 제안" : "검색 조건 제안"}
              </Button>
            )}
          </div>
        </form>
        {raw && !proposal && (
          <details>
            <summary>
              {busy ? "검색 조건을 스트리밍으로 생성 중" : "모델 응답 확인"}
            </summary>
            <pre
              style={{
                whiteSpace: "pre-wrap",
                overflowWrap: "anywhere",
                maxHeight: 200,
                overflow: "auto",
              }}
            >
              {raw}
            </pre>
          </details>
        )}
        {proposal && (
          <section aria-label="검색 조건 검토">
            <h3>제안된 검색 조건</h3>
            <p>{proposal.explanation}</p>
            <dl className="natural-search-plan">
              {Object.entries(proposal.plan).map(([key, value]) => (
                <div key={key}>
                  <dt>{labels[key] || key}</dt>
                  <dd>
                    {value === true
                      ? "있는 문서만"
                      : value === false
                        ? "제한 없음"
                        : value
                          ? values[String(value)] || String(value)
                          : "제한 없음"}
                  </dd>
                </div>
              ))}
            </dl>
            <p className="muted">
              날짜는 UTC 수정일 기준이며 종료일을 포함합니다. 직접 선택한
              공간·작성자 필터는 유지합니다. 적용 후 일반 검색과 동일한 현재
              접근 권한을 검사합니다.
            </p>
            <div className="modal-actions">
              <Button
                onClick={() => {
                  onApply(proposal.query);
                  setOpen(false);
                }}
                variant="primary"
              >
                조건 확인 후 검색
              </Button>
            </div>
          </section>
        )}
      </Modal>
    </>
  );
}
