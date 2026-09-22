import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Download, Square, Sparkles } from "lucide-react";
import { api, ApiError, type Doc } from "../api";
import { useApp } from "../context";
import { Button, Field, Loading, Modal } from "../ui";
import { ChangeReview, RecoveryNotice } from "./ChangeReview";
import { selectionBytes, type MarkdownSelection } from "./selection";
import "./selection-ai.css";
type Metadata = {
  version: number;
  provider: {
    configured: boolean;
    base_url: string;
    model: string;
    fingerprint: string;
    max_tokens: number;
  };
};
type Proposal = {
  document_id: string;
  expected_version: number;
  start_byte: number;
  end_byte: number;
  markdown: string;
  draft_ticket: string;
};
export default function SelectionAI({
  doc,
  selection,
  stale,
  onClose,
  onApply,
}: {
  doc: Doc;
  selection: MarkdownSelection;
  stale: boolean;
  onClose: () => void;
  onApply: (markdown: string, expectedVersion: number) => Promise<void>;
}) {
  const { user, workspace, notify, reload } = useApp(),
    navigate = useNavigate();
  const [metadata, setMetadata] = useState<Metadata | null>(null),
    [error, setError] = useState<unknown>(null),
    [busy, setBusy] = useState(false),
    [saving, setSaving] = useState(false),
    [text, setText] = useState(""),
    [proposal, setProposal] = useState<Proposal | null>(null),
    [action, setAction] = useState("rewrite"),
    [prompt, setPrompt] = useState(""),
    [consent, setConsent] = useState(false),
    [applyMode, setApplyMode] = useState("replace");
  const controller = useRef<AbortController | null>(null),
    current = useRef("");
  const scope = `${user.id}:${workspace?.id}:${doc.id}:${doc.version}`;
  const originScope = useRef(scope),
    mounted = useRef(true);
  current.current = scope;
  const bytes = selectionBytes(doc.markdown, selection);
  useEffect(() => {
    mounted.current = true;
    let active = true;
    api<Metadata>(`/documents/${doc.id}/ai-selection`)
      .then((value) => {
        if (active && current.current === scope) setMetadata(value);
      })
      .catch((e) => {
        if (active && current.current === scope) setError(e);
      });
    return () => {
      active = false;
      mounted.current = false;
      controller.current?.abort();
    };
  }, [scope]);
  useEffect(() => {
    if (stale) {
      controller.current?.abort();
      setProposal(null);
      setText("");
      setConsent(false);
      setError(
        new ApiError(
          "문서 또는 권한이 변경되었습니다. 현재 원문에서 다시 선택하세요.",
          409,
        ),
      );
    }
  }, [stale]);
  const download = () => {
    const url = URL.createObjectURL(
      new Blob([text], { type: "text/markdown;charset=utf-8" }),
    );
    const a = document.createElement("a");
    a.href = url;
    a.download = "madi-ai-selection-draft.md";
    a.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  };
  const start = async () => {
    if (!metadata || busy || !consent || stale) return;
    const abort = new AbortController();
    controller.current = abort;
    const alive = () => !abort.signal.aborted && current.current === scope;
    setBusy(true);
    setError(null);
    setText("");
    setProposal(null);
    try {
      const response = await fetch(`/api/v1/documents/${doc.id}/ai-selection`, {
        method: "POST",
        credentials: "same-origin",
        signal: abort.signal,
        headers: { "Content-Type": "application/json", "X-Madi-Request": "1" },
        body: JSON.stringify({
          ...bytes,
          expected_version: doc.version,
          selected_text: selection.text,
          provider_fingerprint: metadata.provider.fingerprint,
          consent: true,
          action,
          prompt,
        }),
      });
      if (!response.ok) {
        const body = await response.json().catch(() => ({}));
        throw new ApiError(
          body.error || "AI 요청을 처리하지 못했습니다.",
          response.status,
        );
      }
      if (
        !response.body ||
        !response.headers.get("Content-Type")?.includes("text/event-stream")
      )
        throw new Error("AI 스트리밍 응답이 필요합니다.");
      const reader = response.body.getReader(),
        decoder = new TextDecoder();
      let buffer = "",
        output = "",
        completed = false,
        accepted = false;
      while (true) {
        const chunk = await reader.read();
        if (chunk.done) break;
        buffer += decoder.decode(chunk.value, { stream: true });
        if (buffer.length > 300000)
          throw new Error("AI 이벤트가 허용 크기를 초과했습니다.");
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
            completed = true;
            continue;
          }
          const value = JSON.parse(raw);
          if (value.error || value.retract) {
            if (alive()) {
              setText("");
              setProposal(null);
            }
            throw new Error(
              value.error || "권한 또는 원문 변경으로 제안을 회수했습니다.",
            );
          }
          if (typeof value.text === "string") {
            output += value.text;
            if (new TextEncoder().encode(output).length > 65536)
              throw new Error("AI 제안은 64KiB 이하여야 합니다.");
            if (alive()) setText(output);
          }
          if (value.proposal) {
            const result = value.proposal as Proposal;
            if (
              result.document_id !== doc.id ||
              result.expected_version !== doc.version ||
              result.start_byte !== bytes.start_byte ||
              result.end_byte !== bytes.end_byte ||
              typeof result.markdown !== "string" ||
              result.markdown !== output
            )
              throw new Error("AI 결과가 선택 원문과 일치하지 않습니다.");
            accepted = true;
            if (alive()) setProposal(result);
          }
        }
      }
      if (!completed || !accepted)
        throw new Error(
          "AI 응답이 완료되지 않았습니다. 중간 결과는 적용할 수 없습니다.",
        );
    } catch (e) {
      if (alive()) {
        setProposal(null);
        setError(e);
      }
    } finally {
      if (current.current === scope) setBusy(false);
    }
  };
  const apply = async () => {
    if (!proposal || stale || saving) return;
    setSaving(true);
    setError(null);
    const alive = () => mounted.current && current.current === scope;
    try {
      if (applyMode === "private") {
        const created = await api<Doc>("/ai/selection-drafts", "POST", {
          ticket: proposal.draft_ticket,
          title: `${doc.title.slice(0, 110)} · AI 선택 초안`,
          markdown: proposal.markdown,
        });
        if (!alive()) return;
        notify("AI 제안을 새 개인 문서로 저장했습니다.");
        onClose();
        await reload();
        navigate(`/app/documents/${created.id}?mode=source`);
      } else {
        const next =
          applyMode === "replace"
            ? doc.markdown.slice(0, selection.start) +
              proposal.markdown +
              doc.markdown.slice(selection.end)
            : doc.markdown.slice(0, selection.end) +
              "\n\n" +
              proposal.markdown +
              doc.markdown.slice(selection.end);
        await onApply(next, proposal.expected_version);
        if (alive()) onClose();
      }
    } catch (e) {
      if (alive()) setError(e);
    } finally {
      if (alive()) setSaving(false);
    }
  };
  if (originScope.current !== scope) return null;
  return (
    <Modal
      open
      onOpenChange={(value) => {
        if (!value && !saving) {
          controller.current?.abort();
          onClose();
        }
      }}
      title="선택 영역 AI"
      description="선택한 원문만 전달하고, 완료된 제안을 비교한 뒤 적용합니다."
    >
      <div className="selection-ai">
        {!metadata && !error ? <Loading /> : null}
        {metadata && (
          <>
            <div className="notice subtle">
              <strong>
                전송 대상: {metadata.provider.model || "설정되지 않음"}
              </strong>
              <span>
                {metadata.provider.base_url || "관리자 AI 설정이 필요합니다."}
              </span>
              <span>
                선택 원문{" "}
                {new TextEncoder()
                  .encode(selection.text)
                  .length.toLocaleString()}
                바이트 · v{doc.version} · 최대 출력{" "}
                {metadata.provider.max_tokens.toLocaleString()} 토큰
              </span>
              <span>
                주소의 인증 쿼리는 표시하지 않습니다. 제목·태그·나머지 본문·검색
                결과는 보내지 않습니다.
              </span>
              {metadata.provider.base_url.startsWith("http:") && (
                <strong>HTTP 공급자: 전송 구간이 암호화되지 않습니다.</strong>
              )}
            </div>
            <fieldset disabled={busy || saving || stale}>
              <Field label="선택 영역 작업">
                <select
                  value={action}
                  onChange={(e) => {
                    setAction(e.target.value);
                    setProposal(null);
                    setConsent(false);
                  }}
                >
                  <option value="rewrite">문장 개선</option>
                  <option value="summarize">핵심 요약</option>
                  <option value="translate">번역</option>
                  <option value="write">이어 쓰기</option>
                </select>
              </Field>
              <Field label="추가 지시">
                <textarea
                  maxLength={1000}
                  value={prompt}
                  placeholder="예: 영어로 번역해 주세요."
                  onChange={(e) => {
                    setPrompt(e.target.value);
                    setProposal(null);
                    setConsent(false);
                  }}
                />
              </Field>
              <details>
                <summary>전송할 선택 원문 확인</summary>
                <pre>{selection.text}</pre>
              </details>
              <label className="check-label">
                <input
                  type="checkbox"
                  checked={consent}
                  onChange={(e) => setConsent(e.target.checked)}
                />
                표시한 공급자에 선택 원문과 추가 지시를 전송하는 데 동의합니다.
              </label>
            </fieldset>
            <div className="button-row">
              <Button
                variant="primary"
                disabled={
                  busy ||
                  saving ||
                  stale ||
                  !consent ||
                  !metadata.provider.configured ||
                  metadata.version !== doc.version
                }
                onClick={() => void start()}
              >
                <Sparkles size={17} />
                {busy ? "AI 제안 생성 중…" : "선택 원문 전송"}
              </Button>
              {busy && (
                <Button
                  onClick={() => {
                    controller.current?.abort();
                    setProposal(null);
                    setBusy(false);
                    setError(
                      new Error(
                        "생성을 중단했습니다. 중간 결과는 적용하지 않습니다.",
                      ),
                    );
                  }}
                >
                  <Square size={16} />
                  생성 중단
                </Button>
              )}
              {text && (
                <Button onClick={download}>
                  <Download size={16} />
                  제안 내려받기
                </Button>
              )}
            </div>
          </>
        )}
        {error ? (
          <RecoveryNotice error={error} onCopy={text ? download : undefined} />
        ) : null}
        {text && !proposal && (
          <section aria-live="polite">
            <h3>생성 중인 제안 · 아직 적용할 수 없음</h3>
            <pre>{text}</pre>
          </section>
        )}
        {proposal && (
          <ChangeReview
            title="선택 원문과 제안 비교"
            description="제목·태그·공유 범위는 바꾸지 않습니다. 원문 적용은 버전 검사와 현재 승인·정보보호 정책을 거칩니다."
            changes={[
              {
                label:
                  applyMode === "replace"
                    ? "선택 범위 교체"
                    : applyMode === "append"
                      ? "선택 범위 뒤에 추가"
                      : "새 개인 문서",
                before: selection.text,
                after: proposal.markdown,
              },
            ]}
            warnings={
              applyMode === "private"
                ? [
                    "개인 문서 생성 권한이 필요하며, 원본 문서는 바뀌지 않습니다.",
                  ]
                : [
                    "Markdown 원문 변경으로 공동 편집의 기준 버전이 바뀝니다. 다른 사용자의 미확정 초안은 자동 덮어쓰지 않습니다.",
                  ]
            }
            confirmLabel={
              applyMode === "private"
                ? "새 개인 문서로 저장"
                : "비교한 변경 적용"
            }
            busy={saving}
            disabled={stale || (applyMode !== "private" && !doc.can_write)}
            onCancel={onClose}
            onConfirm={() => void apply()}
          >
            <Field label="제안 적용 위치">
              <select
                value={applyMode}
                onChange={(e) => setApplyMode(e.target.value)}
                disabled={saving}
              >
                <option value="replace" disabled={!doc.can_write}>
                  선택한 범위만 교체
                </option>
                <option value="append" disabled={!doc.can_write}>
                  선택한 범위 뒤에 추가
                </option>
                <option value="private">새 개인 문서로 저장</option>
              </select>
            </Field>
          </ChangeReview>
        )}
      </div>
    </Modal>
  );
}
