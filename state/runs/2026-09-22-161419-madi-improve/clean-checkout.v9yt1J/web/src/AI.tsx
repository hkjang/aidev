import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import {
  ArrowUp,
  BookOpen,
  Check,
  FileText,
  LoaderCircle,
  RefreshCw,
  Send,
  Sparkles,
  Square,
  X,
} from "lucide-react";
import { useApp } from "./context";
import { Button, CopyButton, ErrorBox } from "./ui";
import { MarkdownContent as MarkdownView } from "./editor/MarkdownContent";
import CitationViewer, { type CitationSource } from "./CitationViewer";
import { AIActionSelect, AISaveDraft } from "./AIActions";
import { SaveAIHistory } from "./AIHistory";
import "./ai-readability.css";

export default function AI({
  open,
  onClose,
  documentId,
  conversationId,
  conversationVersion,
  selectedDocuments,
  embedded = false,
}: {
  open: boolean;
  onClose: () => void;
  documentId?: string;
  conversationId?: string;
  conversationVersion?: number;
  selectedDocuments?: { id: string; version: number }[];
  embedded?: boolean;
}) {
  const { workspace, documents, user } = useApp();
  const [prompt, setPrompt] = useState(""),
    [answer, setAnswer] = useState(""),
    [question, setQuestion] = useState(""),
    [sources, setSources] = useState<CitationSource[]>([]),
    [retrieval, setRetrieval] = useState<{
      mode: string;
      backend: string;
      scanned: number;
      truncated: boolean;
      reranked: boolean;
      warnings: string[];
    } | null>(null),
    [citation, setCitation] = useState<CitationSource | null>(null),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [action, setAction] = useState("ask"),
    [historyTicket, setHistoryTicket] = useState(""),
    [useDoc, setUseDoc] = useState(true);
  const controller = useRef<AbortController | null>(null),
    body = useRef<HTMLDivElement>(null);
  const selectedKey = JSON.stringify(selectedDocuments || []);
  useEffect(() => () => controller.current?.abort(), []);
  useEffect(() => {
    controller.current?.abort();
    controller.current = null;
    setPrompt(
      selectedDocuments?.length
        ? "선택한 문서의 근거 조각을 바탕으로 핵심 내용과 차이, 확인할 점을 요약해 주세요."
        : "",
    );
    setQuestion("");
    setAnswer("");
    setSources([]);
    setRetrieval(null);
    setCitation(null);
    setError("");
    setBusy(false);
    setAction(selectedDocuments?.length ? "summarize" : "ask");
    setHistoryTicket("");
  }, [
    workspace?.id,
    documentId,
    user.id,
    conversationId,
    conversationVersion,
    selectedKey,
  ]);
  useEffect(() => {
    if (body.current) body.current.scrollTop = body.current.scrollHeight;
  }, [answer]);
  const send = async (text = prompt) => {
    if (!text.trim() || busy) return;
    setQuestion(text);
    setPrompt("");
    setAnswer("");
    setSources([]);
    setRetrieval(null);
    setCitation(null);
    setError("");
    setBusy(true);
    setHistoryTicket("");
    const requestController = new AbortController();
    controller.current = requestController;
    try {
      const res = await fetch("/api/v1/ai/chat", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json", "X-Madi-Request": "1" },
        body: JSON.stringify({
          prompt: text,
          action,
          ...(conversationId
            ? {
                conversation_id: conversationId,
                conversation_version: conversationVersion,
              }
            : {}),
          workspace_id: workspace?.id,
          ...(selectedDocuments?.length
            ? { selected_documents: selectedDocuments }
            : {}),
          ...(documentId && useDoc ? { document_id: documentId } : {}),
        }),
        signal: requestController.signal,
      });
      if (!res.ok) {
        const e = await res
          .json()
          .catch(() => ({ error: "AI 요청을 처리할 수 없습니다." }));
        throw new Error(e.error);
      }
      if (!res.body) throw new Error("스트리밍 응답을 받을 수 없습니다.");
      const reader = res.body.getReader(),
        decoder = new TextDecoder();
      let buffer = "";
      while (true) {
        const { value, done } = await reader.read();
        if (
          requestController.signal.aborted ||
          controller.current !== requestController
        )
          return;
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() || "";
        for (const line of lines) {
          if (!line.startsWith("data:")) continue;
          const raw = line.slice(5).trim();
          if (!raw || raw === "[DONE]") continue;
          const data = JSON.parse(raw);
          if (data.retract) {
            setHistoryTicket("");
            setAnswer("");
            setSources([]);
            setRetrieval(null);
            setCitation(null);
          }
          if (data.error) throw new Error(data.error);
          if (data.text) setAnswer((v) => v + data.text);
          if (data.sources) setSources(data.sources);
          if (data.retrieval) setRetrieval(data.retrieval);
          if (data.history_ticket) setHistoryTicket(data.history_ticket);
          if (data.history_notice)
            setRetrieval((value) =>
              value
                ? {
                    ...value,
                    warnings: [...value.warnings, data.history_notice],
                  }
                : value,
            );
        }
      }
    } catch (e) {
      if (
        controller.current === requestController &&
        (e as Error).name !== "AbortError"
      )
        setError((e as Error).message);
    } finally {
      if (controller.current === requestController) setBusy(false);
    }
  };
  if (!open) return null;
  return (
    <>
      {!embedded && <div className="ai-backdrop" onClick={onClose} />}
      <aside
        className={`ai-panel ${embedded ? "ai-embedded" : ""}`}
        aria-label="AI 지식 도우미"
      >
        <header>
          <span className="ai-icon">
            <Sparkles size={23} />
          </span>
          <div>
            <h2>madi AI</h2>
            <small>우리 팀의 지식을 더 가깝게</small>
          </div>
          <button
            className="icon-button"
            aria-label="AI 대화 초기화"
            onClick={() => {
              controller.current?.abort();
              controller.current = null;
              setBusy(false);
              setQuestion("");
              setAnswer("");
              setError("");
              setSources([]);
              setRetrieval(null);
              setCitation(null);
              setHistoryTicket("");
            }}
          >
            <RefreshCw size={18} />
          </button>
          <button
            className="icon-button"
            aria-label="AI 도우미 닫기"
            onClick={onClose}
          >
            <X size={21} />
          </button>
        </header>
        <div className="ai-body" ref={body}>
          {conversationId && (
            <p className="notice">
              선택한 개인 기록을 같은 AI 공급자에게 전송하여 이어서 질문합니다.
              과거 참조 버전이나 공급자가 바뀌었다면 새 대화를 시작해야 합니다.
            </p>
          )}
          {!question ? (
            <div className="ai-welcome">
              <span className="ai-welcome-orb">
                <Sparkles size={38} />
              </span>
              <h3>어떤 생각을 함께할까요?</h3>
              <p>
                워크스페이스의 지식에서 답을 찾고,
                <br />
                문서 작업에 새로운 영감을 더해 보세요.
              </p>
              {[
                "이 문서의 핵심 내용을 요약해 줘",
                "주요 의사결정과 다음 할 일을 정리해 줘",
                "관련 문서를 찾아 연결할 내용을 추천해 줘",
              ].map((q) => (
                <button key={q} onClick={() => send(q)}>
                  <Sparkles size={17} />
                  {q}
                  <ArrowUp size={16} />
                </button>
              ))}
            </div>
          ) : (
            <>
              <div className="ai-question">
                <span className="avatar small">{user.name?.slice(0, 1)}</span>
                <p>{question}</p>
              </div>
              <div className="ai-answer">
                <div className="ai-answer-label">
                  <Sparkles size={18} />
                  <strong>madi AI</strong>
                  {busy && <span className="streaming-dot" />}
                </div>
                {answer ? (
                  <MarkdownView markdown={answer} documents={documents} />
                ) : busy ? (
                  <div className="thinking">
                    <LoaderCircle className="spin" size={18} /> 지식에서 답을
                    찾고 있어요…
                  </div>
                ) : null}
                {sources.length > 0 && (
                  <div className="ai-sources">
                    <strong>참고한 문서</strong>
                    {sources.map((s, i) =>
                      s.citation_url ? (
                        <button
                          key={s.citation_id || `${s.id}:${i}`}
                          className="text-button"
                          style={{
                            display: "flex",
                            gap: 7,
                            textAlign: "left",
                            alignItems: "center",
                            padding: "9px 0",
                          }}
                          onClick={() => setCitation(s)}
                        >
                          <FileText size={16} />
                          <span>
                            [{i + 1}] {s.title} · {s.start_line}–{s.end_line}행
                          </span>
                        </button>
                      ) : (
                        <Link
                          key={`${s.id}:${i}`}
                          to={s.url || `/app/documents/${s.id}`}
                          onClick={onClose}
                        >
                          <FileText size={16} />
                          {s.title}
                        </Link>
                      ),
                    )}
                  </div>
                )}
                {retrieval && (
                  <div
                    className="notice"
                    style={{
                      fontSize: "0.9375rem",
                      marginTop: 14,
                      overflowWrap: "anywhere",
                    }}
                  >
                    <strong>
                      {(
                        {
                          keyword: "키워드 검색",
                          hybrid: "통합 검색",
                          semantic: "의미 검색",
                          selected: "선택 문서 요약",
                        } as Record<string, string>
                      )[retrieval.mode] || retrieval.mode}
                    </strong>
                    {retrieval.backend !== "none" && (
                      <p>
                        {retrieval.backend === "pgvector"
                          ? "pgvector 코사인 검색"
                          : "배열 코사인 검색"}{" "}
                        · {retrieval.scanned.toLocaleString()}개 조각 검사
                        {retrieval.reranked ? " · 재정렬 적용" : ""}
                        {retrieval.truncated
                          ? " · 설정된 검사 범위에 도달"
                          : ""}
                      </p>
                    )}
                    {(retrieval.warnings || []).map((warning, i) => (
                      <p key={i}>{warning}</p>
                    ))}
                  </div>
                )}
                <CitationViewer
                  source={citation}
                  onClose={() => setCitation(null)}
                  onNavigate={onClose}
                />
                {answer && !busy && (
                  <CopyButton value={answer} label="답변 복사" />
                )}
                {answer && !busy && !error && (
                  <AISaveDraft
                    answer={answer}
                    question={question}
                    sources={sources}
                    onSaved={onClose}
                  />
                )}
                {historyTicket && !busy && !error && (
                  <SaveAIHistory ticket={historyTicket} onNavigate={onClose} />
                )}
              </div>
            </>
          )}
          <ErrorBox error={error} />
          {error && user.role === "admin" && (
            <Link
              to="/admin/settings?tab=ai"
              onClick={onClose}
              className="text-button"
            >
              관리자 AI 설정 확인 →
            </Link>
          )}
        </div>
        <form
          className="ai-composer"
          onSubmit={(e) => {
            e.preventDefault();
            send();
          }}
        >
          <AIActionSelect
            value={action}
            disabled={busy}
            onChange={(selected) => {
              setAction(selected.id);
              setPrompt(selected.prompt);
            }}
          />
          {documentId && (
            <label className="ai-context">
              <input
                type="checkbox"
                checked={useDoc}
                onChange={(e) => setUseDoc(e.target.checked)}
              />
              <FileText size={14} /> 현재 문서에서 답변 찾기
            </label>
          )}
          {!!selectedDocuments?.length && (
            <details className="ai-selection-notice">
              <summary>
                선택 문서 {selectedDocuments.length}개만 참조 · 보내기 전 확인
              </summary>
              <p>
                선택한 문서의 관련 원문 조각만 AI에 전송합니다. 아래 보내기를
                눌러 요약을 요청하세요. 다른 문서로 검색 범위를 확장하지
                않습니다.
              </p>
            </details>
          )}
          <div>
            <textarea
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="지식에 관해 무엇이든 물어보세요…"
              aria-label="AI 질문"
              onKeyDown={(e) => {
                if (
                  e.key === "Enter" &&
                  !e.shiftKey &&
                  !e.nativeEvent.isComposing
                ) {
                  e.preventDefault();
                  send();
                }
              }}
            />
            {busy ? (
              <Button
                type="button"
                onClick={() => controller.current?.abort()}
                aria-label="생성 중지"
              >
                <Square size={17} />
              </Button>
            ) : (
              <Button
                variant="primary"
                disabled={!prompt.trim()}
                aria-label="AI 질문 보내기"
              >
                <ArrowUp size={21} />
              </Button>
            )}
          </div>
          <small>AI의 답변은 참고 자료와 함께 확인해 주세요.</small>
        </form>
      </aside>
    </>
  );
}
