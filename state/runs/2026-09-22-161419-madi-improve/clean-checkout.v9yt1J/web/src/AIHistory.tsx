import { lazy, Suspense, useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { History, Search, Trash2 } from "lucide-react";
import { api, datetime } from "./api";
import { useApp } from "./context";
import {
  Button,
  CopyButton,
  Empty,
  ErrorBox,
  Loading,
  Modal,
  PageHeading,
} from "./ui";
import { MarkdownContent } from "./editor/MarkdownContent";
import CitationViewer, { type CitationSource } from "./CitationViewer";
import SaveEvidence from "./evidence/SaveEvidence";
const ConversationAI = lazy(() => import("./AI"));

export function SaveAIHistory({
  ticket,
  onNavigate,
}: {
  ticket: string;
  onNavigate: () => void;
}) {
  const { workspace, user } = useApp();
  const [open, setOpen] = useState(false),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [saved, setSaved] = useState("");
  const generation = useRef(0);
  useEffect(() => {
    generation.current++;
    setOpen(false);
    setBusy(false);
    setError("");
    setSaved("");
    return () => {
      generation.current++;
    };
  }, [ticket, workspace?.id, user.id]);
  const save = async () => {
    if (busy) return;
    const request = generation.current;
    setBusy(true);
    setError("");
    try {
      const data = await api("/ai/conversations", "POST", {
        ticket,
        consent: true,
      });
      if (request !== generation.current) return;
      setSaved(data.id);
      setOpen(false);
    } catch (e) {
      if (request === generation.current) setError((e as Error).message);
    } finally {
      if (request === generation.current) setBusy(false);
    }
  };
  return (
    <>
      <SaveEvidence ticket={ticket} onNavigate={onNavigate} />
      {saved ? (
        <Link
          className="button"
          to={`/app/ai-history?id=${saved}`}
          onClick={onNavigate}
        >
          <History size={16} /> 저장한 대화 보기
        </Link>
      ) : (
        <Button type="button" onClick={() => setOpen(true)}>
          <History size={16} /> 개인 대화 기록에 저장
        </Button>
      )}
      <Modal
        open={open}
        onOpenChange={(value) => {
          if (!busy) setOpen(value);
        }}
        title="개인 대화 기록 저장"
      >
        <p>
          질문, 완료된 AI 답변과 출처를 이 워크스페이스의 본인 기록으로
          저장합니다. 다른 사용자와 서비스 관리자는 볼 수 없습니다.
        </p>
        <p className="notice">
          저장된 기록은 서비스 백업에 포함됩니다. 참조 문서 권한이 회수되면
          기록도 숨겨집니다. 민감정보 저장 정책이 적용되며, 이 저장 요청은 답변
          완료 후 1시간 동안 유효합니다.
        </p>
        <ErrorBox error={error} />
        <div className="modal-actions">
          <Button type="button" disabled={busy} onClick={() => setOpen(false)}>
            취소
          </Button>
          <Button variant="primary" disabled={busy} onClick={() => void save()}>
            {busy ? "확인·저장 중…" : "동의하고 개인 기록에 저장"}
          </Button>
        </div>
      </Modal>
    </>
  );
}

type Conversation = {
  id: string;
  title: string;
  version: number;
  updated_at: string;
  messages?: {
    id: string;
    question: string;
    answer: string;
    action: string;
    model: string;
    sources: CitationSource[];
    created_at: string;
  }[];
};
export default function AIHistoryPage() {
  const { workspace, user, documents } = useApp();
  const [params, setParams] = useSearchParams();
  const id = params.get("id") || "",
    term = params.get("q") || "",
    offset = Number(params.get("offset")) || 0;
  const [query, setQuery] = useState(term),
    [items, setItems] = useState<Conversation[]>([]),
    [more, setMore] = useState(false),
    [record, setRecord] = useState<Conversation | null>(null),
    [error, setError] = useState(""),
    [loading, setLoading] = useState(true),
    [refresh, setRefresh] = useState(0),
    [remove, setRemove] = useState(false),
    [busy, setBusy] = useState(false),
    [citation, setCitation] = useState<CitationSource | null>(null),
    [continuation, setContinuation] = useState<{
      id: string;
      version: number;
    } | null>(null);
  const generation = useRef(0);
  useEffect(() => setQuery(term), [term]);
  useEffect(() => {
    const request = ++generation.current;
    setRecord(null);
    setItems([]);
    setCitation(null);
    setContinuation(null);
    setRemove(false);
    setError("");
    setLoading(true);
    setBusy(false);
    let active = true;
    let pending = false;
    const load = async (first = false) => {
      if (!workspace || pending) return;
      pending = true;
      try {
        if (id) {
          const value = await api<Conversation>(
            `/ai/conversations/${id}?workspace_id=${workspace.id}`,
          );
          if (active && request === generation.current) setRecord(value);
        } else {
          const value = await api<{ items: Conversation[]; has_more: boolean }>(
            `/ai/conversations?workspace_id=${workspace.id}&q=${encodeURIComponent(term)}&offset=${offset}`,
          );
          if (active && request === generation.current) {
            setItems(value.items);
            setMore(value.has_more);
          }
        }
        if (active && request === generation.current) setError("");
      } catch (e) {
        if (active && request === generation.current) {
          setRecord(null);
          setItems([]);
          setCitation(null);
          setContinuation(null);
          setRemove(false);
          setError((e as Error).message);
        }
      } finally {
        pending = false;
        if (first && active && request === generation.current)
          setLoading(false);
      }
    };
    void load(true);
    // Clear already-rendered private text if its source access is revoked.
    const timer = window.setInterval(() => void load(), 4000);
    return () => {
      active = false;
      generation.current++;
      clearInterval(timer);
    };
  }, [workspace?.id, user.id, id, term, offset, refresh]);
  const update = (changes: Record<string, string>) => {
    const next = new URLSearchParams(window.location.search);
    Object.entries(changes).forEach(([key, value]) => {
      if (value) next.set(key, value);
      else next.delete(key);
    });
    setParams(next);
  };
  const erase = async () => {
    if (!record || busy) return;
    const request = generation.current;
    setBusy(true);
    try {
      await api(`/ai/conversations/${record.id}`, "DELETE", {
        confirmation: "DELETE",
        version: record.version,
      });
      if (request !== generation.current) return;
      setRecord(null);
      setRemove(false);
      update({ id: "" });
      setRefresh((v) => v + 1);
    } catch (e) {
      if (request === generation.current) setError((e as Error).message);
    } finally {
      if (request === generation.current) setBusy(false);
    }
  };
  return (
    <div className="page-container">
      <PageHeading
        title="내 AI 대화 기록"
        description="명시적으로 저장한 개인 질문과 답변입니다. 현재 참조 문서 권한이 유지되는 기록만 표시됩니다."
      />
      {id && <Button onClick={() => update({ id: "" })}>목록으로</Button>}
      <ErrorBox error={error} />
      {loading ? (
        <Loading />
      ) : record ? (
        <article
          className="settings-card"
          style={{ marginTop: 20, minWidth: 0 }}
        >
          <h2 style={{ overflowWrap: "anywhere" }}>{record.title}</h2>
          <p className="notice">
            과거에 저장한 답변입니다. 문서가 수정되었을 수 있으므로 인용 원문을
            다시 확인하세요. 기록 열람만으로 AI 공급자에 내용을 재전송하지
            않습니다.
          </p>
          {record.messages?.map((message) => (
            <section
              id={message.id}
              key={message.id}
              style={{ marginBottom: 28, minWidth: 0 }}
            >
              <p className="muted">
                {datetime(message.created_at)} · {message.model}
              </p>
              <h3>질문</h3>
              <Link
                className="button"
                to={`/app/knowledge-questions?message_id=${message.id}`}
              >
                이 질문을 공식 답변 초안으로 정리
              </Link>
              <p style={{ whiteSpace: "pre-wrap", overflowWrap: "anywhere" }}>
                {message.question}
              </p>
              <h3>저장한 답변</h3>
              <MarkdownContent
                markdown={message.answer}
                documents={documents}
              />
              <div
                style={{
                  display: "flex",
                  flexWrap: "wrap",
                  gap: 10,
                  marginTop: 16,
                }}
              >
                <CopyButton value={message.question} label="질문 복사" />
                <CopyButton value={message.answer} label="답변 복사" />
              </div>
              {message.sources.map((source, index) => (
                <Button
                  key={source.citation_id || index}
                  style={{ margin: "12px 8px 0 0" }}
                  onClick={() => setCitation(source)}
                >
                  [{index + 1}] 인용 원문 · v{source.version}
                </Button>
              ))}
            </section>
          ))}
          <Button
            onClick={() =>
              setContinuation({ id: record.id, version: record.version })
            }
          >
            기록을 AI에 보내고 이어서 질문
          </Button>
          <Button onClick={() => setRemove(true)}>
            <Trash2 size={16} /> 대화 기록 삭제
          </Button>
        </article>
      ) : !id ? (
        <>
          <form
            onSubmit={(event) => {
              event.preventDefault();
              update({ q: query, offset: "" });
            }}
            style={{ display: "flex", gap: 10, margin: "24px 0" }}
          >
            <input
              aria-label="내 대화 검색"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="질문과 답변 검색"
              style={{ minWidth: 0, flex: 1 }}
            />
            <Button type="submit">
              <Search size={18} /> 검색
            </Button>
          </form>
          {items.length ? (
            <div style={{ display: "grid", gap: 12 }}>
              {items.map((item) => (
                <button
                  className="settings-card"
                  key={item.id}
                  onClick={() => update({ id: item.id })}
                  style={{
                    textAlign: "left",
                    padding: 22,
                    cursor: "pointer",
                    font: "inherit",
                    color: "inherit",
                    overflowWrap: "anywhere",
                  }}
                >
                  <strong>{item.title}</strong>
                  <p className="muted">{datetime(item.updated_at)}</p>
                </button>
              ))}
            </div>
          ) : (
            <Empty
              title="저장한 대화가 없습니다"
              text="AI 도우미에서 답변을 확인한 뒤 ‘개인 대화 기록에 저장’을 선택하세요."
            />
          )}
          <div className="modal-actions">
            <Button
              disabled={offset === 0}
              onClick={() =>
                update({ offset: String(Math.max(0, offset - 40)) })
              }
            >
              이전
            </Button>
            <Button
              disabled={!more}
              onClick={() => update({ offset: String(offset + 40) })}
            >
              다음
            </Button>
          </div>
        </>
      ) : null}
      <CitationViewer
        source={citation}
        onClose={() => setCitation(null)}
        onNavigate={() => setCitation(null)}
      />
      {continuation && (
        <Suspense fallback={<Loading />}>
          <ConversationAI
            open
            onClose={() => {
              setContinuation(null);
              setRefresh((value) => value + 1);
            }}
            conversationId={continuation.id}
            conversationVersion={continuation.version}
          />
        </Suspense>
      )}
      <Modal
        open={remove && !!record}
        onOpenChange={(value) => {
          if (!busy) setRemove(value);
        }}
        title="대화 기록을 삭제할까요?"
      >
        <p>
          이 대화의 질문·답변·출처 기록이 서비스에서 삭제됩니다. 앱에서 되돌릴
          수 없으며 기존 서버 백업 사본은 운영자의 보존 정책에 따릅니다.
        </p>
        <div className="modal-actions">
          <Button disabled={busy} onClick={() => setRemove(false)}>
            취소
          </Button>
          <Button disabled={busy} onClick={() => void erase()}>
            확인하고 기록 삭제
          </Button>
        </div>
      </Modal>
    </div>
  );
}
