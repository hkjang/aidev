import { useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { api, datetime, type Doc } from "../api";
import { useApp } from "../context";
import {
  Button,
  Empty,
  ErrorBox,
  Field,
  Loading,
  Modal,
  PageHeading,
} from "../ui";
import { MarkdownContent } from "../editor/MarkdownContent";
import { ChangeReview } from "../review/ChangeReview";
import "./evidence.css";
import "./questions.css";

type Ref = {
  id: string;
  version: number;
  hash?: string;
  attachment?: {
    attachment_id: string;
    extraction_id: string;
    fragment_id: string;
    [key: string]: unknown;
  } | null;
  title?: string;
  current_version?: number;
  url?: string;
};
type Question = {
  id: string;
  workspace_id: string;
  document_id: string;
  document_title: string;
  owner_id: string;
  question: string;
  reason: string;
  revision: number;
  state: string;
  origin: string;
  review_due: string;
  answer_version: number;
  current_version: number;
  fresh: boolean;
  overdue: boolean;
  owner_valid: boolean;
  published_current: boolean;
  approval_enabled: boolean;
  official: boolean;
  can_manage: boolean;
  can_confirm: boolean;
  markdown: string;
  sources: Ref[];
  notice: string;
  events: {
    revision: number;
    state: string;
    note: string;
    created_at: string;
  }[];
};
type Origin = {
  preview_hash: string;
  workspace_id: string;
  question: string;
  answer: string;
  model: string;
  sources: Ref[];
  notice: string;
};
type Input = {
  document_id: string;
  version: number;
  revision?: number;
  owner_id: string;
  question: string;
  reason: string;
  review_due: string;
  sources: Ref[];
  consent: boolean;
  message_id?: string;
};
const stateName: Record<string, string> = {
  proposed: "정리 중",
  confirmed: "담당자 확인 이력 있음",
  archived: "보관됨",
};
const refKey = (ref: Ref) =>
  ref.attachment
    ? `${ref.id}:${ref.attachment.attachment_id}:${ref.attachment.fragment_id}`
    : ref.id;
const futureDay = () => {
  const d = new Date();
  d.setUTCDate(d.getUTCDate() + 30);
  return d.toISOString().slice(0, 10);
};

export default function KnowledgeQuestionsPage() {
  const { workspace, user, documents, notify, reload } = useApp();
  const [params, setParams] = useSearchParams();
  const id = params.get("id") || "",
    messageID = params.get("message_id") || "",
    after = params.get("after") || "",
    creating = params.get("new") === "1";
  const [items, setItems] = useState<Question[]>([]),
    [next, setNext] = useState("");
  const [record, setRecord] = useState<Question | null>(null),
    [origin, setOrigin] = useState<Origin | null>(null);
  const [members, setMembers] = useState<{ id: string; name: string }[]>([]);
  const [editing, setEditing] = useState(false),
    [answer, setAnswer] = useState(""),
    [question, setQuestion] = useState(""),
    [reason, setReason] = useState(""),
    [owner, setOwner] = useState(user.id),
    [due, setDue] = useState(futureDay);
  const [selected, setSelected] = useState<string[]>([]),
    [attachmentRefs, setAttachmentRefs] = useState<Ref[]>([]);
  const [pending, setPending] = useState<{
      input: Input;
      answer: Doc | null;
      names: string[];
    } | null>(null),
    [decision, setDecision] = useState(""),
    [note, setNote] = useState(""),
    [consent, setConsent] = useState(false);
  const [orphanDraft, setOrphanDraft] = useState<Doc | null>(null);
  const [loading, setLoading] = useState(true),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [refresh, setRefresh] = useState(0);
  const generation = useRef(0),
    latest = useRef<Question | null>(null),
    fingerprint = useRef("");
  const opener = useRef<{ element: HTMLElement; generation: number } | null>(
    null,
  );
  const rememberOpener = () => {
    if (document.activeElement instanceof HTMLElement)
      opener.current = {
        element: document.activeElement,
        generation: generation.current,
      };
  };
  const restoreFocus = (event: Event) => {
    event.preventDefault();
    const saved = opener.current;
    queueMicrotask(() => {
      if (
        saved &&
        saved.generation === generation.current &&
        saved.element.isConnected &&
        !document.querySelector('[role="dialog"][data-state="open"]')
      )
        saved.element.focus({ preventScroll: true });
    });
  };
  const hasChanges = useRef(false);
  hasChanges.current = editing || !!question || !!reason;
  useEffect(() => {
    const leave = (e: BeforeUnloadEvent) => {
      if (hasChanges.current) {
        e.preventDefault();
        e.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", leave);
    return () => window.removeEventListener("beforeunload", leave);
  }, []);
  const navigate = (values: Record<string, string>) => {
    if (
      hasChanges.current &&
      !window.confirm("저장하지 않은 질문 입력을 버리고 이동할까요?")
    )
      return;
    setParams(values);
  };
  useEffect(() => {
    const token = ++generation.current,
      abort = new AbortController();
    let active = true,
      timer: ReturnType<typeof setTimeout>;
    setRecord(null);
    latest.current = null;
    setOrigin(null);
    setItems([]);
    setNext("");
    setEditing(false);
    setAnswer("");
    setQuestion("");
    setReason("");
    setOwner(user.id);
    setDue(futureDay());
    setSelected([]);
    setAttachmentRefs([]);
    setPending(null);
    setDecision("");
    setConsent(false);
    setOrphanDraft(null);
    setBusy(false);
    setLoading(true);
    setError("");
    fingerprint.current = "";
    if (!workspace) {
      setLoading(false);
      return () => {
        active = false;
        abort.abort();
      };
    }
    const wid = workspace.id;
    void api<{ id: string; name: string }[]>(
      `/workspaces/${wid}/members`,
      "GET",
      undefined,
      { signal: abort.signal },
    )
      .then((v) => {
        if (active && token === generation.current) setMembers(v);
      })
      .catch(() => {
        if (active) setMembers([{ id: user.id, name: user.name }]);
      });
    const load = async () => {
      try {
        if (id) {
          const v = await api<Question>(
            `/knowledge/questions/${id}`,
            "GET",
            undefined,
            { signal: abort.signal },
          );
          if (v.workspace_id !== wid)
            throw Error("현재 워크스페이스의 질문만 확인하세요.");
          if (active && token === generation.current) {
            const fp = JSON.stringify([
              v.revision,
              v.current_version,
              v.fresh,
              v.owner_valid,
              v.can_manage,
              v.can_confirm,
              v.approval_enabled,
              v.published_current,
              v.overdue,
              v.official,
              v.markdown,
              v.question,
              v.reason,
              v.sources,
            ]);
            if (fingerprint.current && fingerprint.current !== fp) {
              setPending(null);
              setDecision("");
              setConsent(false);
              setEditing(false);
              setQuestion("");
              setReason("");
              setSelected([]);
              setAttachmentRefs([]);
            }
            fingerprint.current = fp;
            latest.current = v;
            setRecord(v);
          }
        } else if (messageID) {
          const v = await api<Origin>(
            `/ai/messages/${messageID}/question-preview`,
            "GET",
            undefined,
            { signal: abort.signal },
          );
          if (v.workspace_id !== wid)
            throw Error("개인 질문이 저장된 워크스페이스로 전환하세요.");
          if (active && token === generation.current) {
            const fp = JSON.stringify(v);
            if (fingerprint.current && fingerprint.current !== fp) {
              setPending(null);
              setConsent(false);
              setQuestion("");
              setReason("");
            }
            fingerprint.current = fp;
            setOrigin(v);
            setQuestion(v.question);
          }
        } else {
          const v = await api<{ items: Question[]; next_after: string }>(
            `/knowledge/questions?workspace_id=${wid}&after=${after}`,
            "GET",
            undefined,
            { signal: abort.signal },
          );
          if (active && token === generation.current) {
            setItems(v.items);
            setNext(v.next_after);
          }
        }
      } catch (e) {
        if (active && token === generation.current) {
          latest.current = null;
          setRecord(null);
          setOrigin(null);
          setItems([]);
          setPending(null);
          setDecision("");
          setConsent(false);
          setQuestion("");
          setReason("");
          setEditing(false);
          setSelected([]);
          setAttachmentRefs([]);
          setError((e as Error).message);
        }
      } finally {
        if (active && token === generation.current) {
          setLoading(false);
          timer = setTimeout(() => void load(), 2000);
        }
      }
    };
    void load();
    return () => {
      active = false;
      generation.current++;
      abort.abort();
      clearTimeout(timer);
    };
  }, [id, messageID, after, creating, user.id, workspace?.id, refresh]);
  const beginEdit = () => {
    if (!record?.can_manage) return;
    setEditing(true);
    setAnswer(record.document_id);
    setQuestion(record.question);
    setReason("");
    setOwner(record.owner_id);
    setDue(record.review_due);
    setSelected(record.sources.filter((x) => !x.attachment).map((x) => x.id));
    setAttachmentRefs(record.sources.filter((x) => x.attachment));
    setConsent(false);
  };
  const prepare = async () => {
    if (busy || !workspace) return;
    rememberOpener();
    const token = generation.current;
    setBusy(true);
    setError("");
    setConsent(false);
    try {
      if (
        !question.trim() ||
        new TextEncoder().encode(question).length > 2000 ||
        !reason.trim() ||
        !due ||
        !owner
      )
        throw Error(
          "질문·정리 이유·담당자·검토일을 입력하세요. 질문은 UTF-8 2000바이트 이하입니다.",
        );
      let doc: Doc | null = null,
        refs: Ref[] = [],
        names: string[] = [];
      if (origin) {
        refs = origin.sources;
        names = refs.map(
          (x) =>
            documents.find((d) => d.id === x.id)?.title ||
            "현재 접근 가능한 근거",
        );
      } else {
        if (
          !answer ||
          selected.length + attachmentRefs.length < 1 ||
          selected.length + attachmentRefs.length > 32
        )
          throw Error("답변 문서와 1~32개 근거를 선택하세요.");
        const all = await Promise.all(
          [answer, ...selected].map((d) => api<Doc>(`/documents/${d}`)),
        );
        if (all.some((d) => d.workspace_id !== workspace.id))
          throw Error("같은 워크스페이스의 자료만 선택하세요.");
        doc = all[0];
        refs = all.slice(1).map((d) => ({ id: d.id, version: d.version }));
        refs.push(...attachmentRefs);
        names = all.slice(1).map((d) => d.title);
        names.push(...attachmentRefs.map((x) => x.title || "첨부 근거"));
      }
      if (token !== generation.current) return;
      setPending({
        input: {
          document_id: doc?.id || "",
          version: doc?.version || 0,
          ...(editing && record ? { revision: record.revision } : {}),
          owner_id: owner,
          question,
          reason,
          review_due: due,
          sources: refs,
          consent: true,
          ...(origin ? { message_id: messageID } : {}),
        },
        answer: doc,
        names,
      });
    } catch (e) {
      if (token === generation.current) setError((e as Error).message);
    } finally {
      if (token === generation.current) setBusy(false);
    }
  };
  const save = async () => {
    if (!pending || busy || !consent || !workspace) return;
    const token = generation.current;
    const snapshot = pending;
    setBusy(true);
    setError("");
    try {
      const input = { ...snapshot.input };
      if (origin) {
        // Two explicit normal API operations. If registration fails, the created
        // private document is preserved and linked instead of deleted or duplicated.
        let d = orphanDraft;
        if (!d) {
          d = await api<Doc>(
            `/ai/messages/${messageID}/question-draft`,
            "POST",
            {
              preview_hash: origin.preview_hash,
              consent: true,
            },
          );
          if (token !== generation.current) return;
          setOrphanDraft(d);
        }
        input.document_id = d.id;
        input.version = d.version;
      }
      const out = await api<{ id: string }>(
        editing && record
          ? `/knowledge/questions/${record.id}`
          : "/knowledge/questions",
        editing ? "PUT" : "POST",
        input,
      );
      if (token !== generation.current) return;
      hasChanges.current = false;
      setPending(null);
      setQuestion("");
      setReason("");
      setEditing(false);
      notify(
        "관리 질문을 정리 중 상태로 저장했습니다. 게시와 담당자 공식 확인은 별도입니다.",
      );
      setParams({ id: out.id });
      setRefresh((n) => n + 1);
      void reload();
    } catch (e) {
      if (token === generation.current) setError((e as Error).message);
    } finally {
      if (token === generation.current) setBusy(false);
    }
  };
  const decide = async () => {
    if (!record || !decision || !consent || !note.trim() || busy) return;
    const token = generation.current;
    setBusy(true);
    setError("");
    try {
      await api(`/knowledge/questions/${record.id}/decision`, "POST", {
        revision: record.revision,
        state: decision,
        note,
        consent: true,
      });
      if (token === generation.current) {
        setDecision("");
        setNote("");
        setConsent(false);
        setRefresh((n) => n + 1);
      }
    } catch (e) {
      if (token === generation.current) setError((e as Error).message);
    } finally {
      if (token === generation.current) setBusy(false);
    }
  };
  const formVisible =
    !!origin || editing || (!id && !messageID && params.get("new") === "1");
  return (
    <div className="evidence-page question-page">
      <PageHeading
        title="관리 질문 · 공식 답변"
        description="반복 질문을 담당자와 검토 기한, 현재 근거가 연결된 답변으로 정리합니다."
      />
      <div className="question-actions">
        <Button onClick={() => navigate({})}>질문 목록</Button>
        <Button onClick={() => navigate({ new: "1" })}>새 관리 질문</Button>
        <Button onClick={() => setRefresh((n) => n + 1)} disabled={busy}>
          현재 권한으로 새로고침
        </Button>
      </div>
      <ErrorBox error={error} />
      {loading && <Loading />}
      {orphanDraft && (
        <p className="notice">
          비공개 답변 초안이 생성되었습니다. 등록 실패 시 이 문서는 유지됩니다.{" "}
          <Link to={`/app/documents/${orphanDraft.id}`}>보관된 초안 확인</Link>
        </p>
      )}
      {record && !editing && (
        <section className="card">
          <h2>{record.question}</h2>
          <p>
            <strong>
              {record.official
                ? "공식 답변 · 현재 조건 충족"
                : "현재 공식 답변으로 표시하지 않음"}
            </strong>{" "}
            · {stateName[record.state]}
          </p>
          <div className="evidence-status" aria-label="공식 답변 조건">
            <span>
              답변·근거:{" "}
              {record.fresh ? "현재 버전 일치" : "변경됨 · 재확인 필요"}
            </span>
            <span>
              담당자 권한: {record.owner_valid ? "유효" : "재지정 필요"}
            </span>
            <span>
              검토일: {record.review_due} (UTC){" "}
              {record.overdue ? "· 기한 경과" : "· 기한 내"}
            </span>
            <span>
              게시:{" "}
              {record.published_current
                ? "현재 게시 조건 충족"
                : "게시 또는 현재 승인 필요"}
            </span>
            <span>
              승인 절차:{" "}
              {record.approval_enabled
                ? "관리자 설정 적용"
                : "미설정 · 별도 검토/승인 절차 없음"}
            </span>
          </div>
          <p>
            담당자:{" "}
            {members.find((m) => m.id === record.owner_id)?.name ||
              "등록된 구성원"}{" "}
            · 답변 기준 v{record.answer_version} / 현재 v
            {record.current_version}
          </p>
          <p>
            {record.origin === "personal_ai_question"
              ? "본인의 개인 AI 질문에서 명시적으로 정리한 질문입니다. 대화 전체를 공유한 것이 아닙니다."
              : "사람이 등록한 관리 질문입니다."}
          </p>
          <p className="notice">{record.notice}</p>
          <div className="question-actions">
            <Link
              className="button"
              to={`/app/documents/${record.document_id}`}
            >
              답변 문서 · 공유/게시 관리
            </Link>
            {record.can_manage && (
              <>
                <Button onClick={beginEdit}>근거·담당자·검토일 정리</Button>
                <Button
                  disabled={
                    !record.can_confirm ||
                    !record.fresh ||
                    record.overdue ||
                    !record.published_current
                  }
                  onClick={() => {
                    rememberOpener();
                    setDecision("confirmed");
                    setNote("");
                    setConsent(false);
                  }}
                >
                  담당자로 공식 답변 확인
                </Button>
                <Button
                  disabled={!record.can_confirm}
                  onClick={() => {
                    rememberOpener();
                    setDecision("archived");
                    setNote("");
                    setConsent(false);
                  }}
                >
                  질문 보관
                </Button>
              </>
            )}
          </div>
          <h3>현재 정본 답변</h3>
          <MarkdownContent markdown={record.markdown} documents={documents} />
          <h3>연결된 근거</h3>
          <ul>
            {record.sources.map((ref) => (
              <li key={refKey(ref)}>
                <Link to={ref.url || `/app/documents/${ref.id}`}>
                  {ref.title} · 기준 v{ref.version} / 현재 v
                  {ref.current_version}
                  {ref.attachment ? " · 첨부 위치" : ""}
                </Link>
              </li>
            ))}
          </ul>
          <h3>정리·확인 이력</h3>
          {record.events.length ? (
            record.events.map((e) => (
              <p key={e.revision}>
                r{e.revision} · {stateName[e.state]} · {datetime(e.created_at)}
                <br />
                {e.note}
              </p>
            ))
          ) : (
            <p>아직 담당자 확인 이력이 없습니다.</p>
          )}
        </section>
      )}
      {formVisible && (
        <section className="card">
          <h2>
            {origin
              ? "개인 질문을 답변 초안으로 정리"
              : editing
                ? "질문·근거 재정리"
                : "새 관리 질문 등록"}
          </h2>
          {origin && (
            <>
              <p className="notice">{origin.notice}</p>
              <details>
                <summary>복사할 개인 답변과 근거 확인</summary>
                <MarkdownContent
                  markdown={origin.answer}
                  documents={documents}
                />
                <ul>
                  {origin.sources.map((ref) => (
                    <li key={refKey(ref)}>
                      {" "}
                      {documents.find((d) => d.id === ref.id)?.title ||
                        "현재 근거"}{" "}
                      · v{ref.version}
                      {ref.attachment ? " · 첨부 조각 포함" : ""}
                    </li>
                  ))}
                </ul>
              </details>
            </>
          )}
          {!origin && (
            <Field label="정본 답변 문서">
              <select
                aria-label="정본 답변 문서"
                value={answer}
                disabled={editing || busy}
                onChange={(e) => setAnswer(e.target.value)}
              >
                <option value="">문서 선택</option>
                {documents.map((d) => (
                  <option key={d.id} value={d.id}>
                    {d.title}
                  </option>
                ))}
              </select>
            </Field>
          )}
          <Field label="관리할 질문">
            <textarea
              aria-label="관리할 질문"
              value={question}
              readOnly={!!origin}
              onChange={(e) => setQuestion(e.target.value)}
              rows={3}
            />
          </Field>
          <div className="evidence-compare">
            <Field label="답변 담당자">
              <select
                aria-label="답변 담당자"
                value={owner}
                onChange={(e) => setOwner(e.target.value)}
              >
                {!members.some((m) => m.id === owner) && (
                  <option value={owner}>현재 담당자</option>
                )}
                {members.map((m) => (
                  <option key={m.id} value={m.id}>
                    {m.name}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="다음 검토일 (UTC 날짜)">
              <input
                aria-label="다음 검토일"
                type="date"
                value={due}
                onChange={(e) => setDue(e.target.value)}
              />
            </Field>
          </div>
          <Field label="정리 이유">
            <textarea
              aria-label="정리 이유"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              rows={2}
            />
          </Field>
          {!origin && (
            <>
              <h3>답변을 뒷받침하는 근거 · 합계 1~32개</h3>
              <div className="package-document-list">
                {documents.map((d) => (
                  <label className="question-choice" key={d.id}>
                    <input
                      type="checkbox"
                      checked={selected.includes(d.id)}
                      onChange={(e) =>
                        setSelected((v) =>
                          e.target.checked
                            ? [...v, d.id]
                            : v.filter((x) => x !== d.id),
                        )
                      }
                    />
                    {d.title}
                  </label>
                ))}
                {attachmentRefs.map((ref) => (
                  <div className="question-choice" key={refKey(ref)}>
                    <span>{ref.title || "첨부 근거"} · 현재 선택 조각</span>
                    <Button
                      onClick={() =>
                        setAttachmentRefs((v) =>
                          v.filter((x) => refKey(x) !== refKey(ref)),
                        )
                      }
                    >
                      이 첨부 근거 제외
                    </Button>
                  </div>
                ))}
              </div>
            </>
          )}
          <p className="notice">
            질문은 답변 문서와 모든 근거의 현재 접근 범위에서만 표시됩니다.
            담당자는 답변을 수정하고 모든 근거를 읽을 수 있어야 합니다. 저장하면
            이전 공식 확인이 해제되며 문서 자체의 공유·게시 상태는 바뀌지
            않습니다.
          </p>
          <Button
            variant="primary"
            disabled={busy}
            onClick={() => void prepare()}
          >
            질문 등록 내용 비교
          </Button>
          {editing && (
            <Button
              onClick={() => {
                if (window.confirm("저장하지 않은 정리 입력을 버릴까요?")) {
                  setEditing(false);
                  setQuestion("");
                  setReason("");
                }
              }}
            >
              정리 취소
            </Button>
          )}
        </section>
      )}
      {!id && !messageID && !formVisible && !loading && (
        <section>
          {items.length ? (
            <div className="evidence-list">
              {items.map((q) => (
                <Link className="card" key={q.id} to={`?id=${q.id}`}>
                  <strong>{q.question}</strong>
                  <span>
                    {q.official
                      ? "공식 답변 · 현재 조건 충족"
                      : "정리·재확인 대상"}{" "}
                    · {q.document_title}
                  </span>
                  <span>
                    검토일 {q.review_due} ·{" "}
                    {q.fresh ? "현재 버전" : "근거 또는 답변 변경"}
                  </span>
                </Link>
              ))}
            </div>
          ) : (
            <Empty
              title="표시할 관리 질문이 없습니다"
              text="반복 질문과 근거 문서를 선택해 담당자가 확인할 답변으로 정리하세요."
            />
          )}
          {next && (
            <Button onClick={() => navigate({ after: next })}>다음 질문</Button>
          )}
          {after && <Button onClick={() => navigate({})}>처음 질문</Button>}
        </section>
      )}
      <Modal
        open={!!pending}
        onCloseAutoFocus={restoreFocus}
        onOpenChange={(v) => {
          if (!v && !busy) {
            setPending(null);
            setConsent(false);
          }
        }}
        title="관리 질문 공유 내용 확인"
      >
        {pending && (
          <ChangeReview
            title="정리할 내용과 접근 범위"
            changes={[
              {
                label: "질문",
                before: record?.question,
                after: pending.input.question,
              },
              {
                label: "답변",
                before: record?.document_title,
                after: pending.answer
                  ? `${pending.answer.title} · v${pending.answer.version} · ${pending.answer.visibility === "private" ? "나만 보기" : pending.answer.visibility === "workspace" ? "워크스페이스 공유" : "선택 사용자 공유"}`
                  : "새 비공개 초안 · 팀 공유/게시 안 함",
              },
              {
                label: "근거",
                before: record?.sources.map((x) => x.title).join(", "),
                after: pending.names.join(", "),
              },
              {
                label: "검토일",
                before: record?.review_due,
                after: `${pending.input.review_due} (UTC)`,
              },
            ]}
            warnings={[
              "담당자와 근거 접근 권한을 서버에서 다시 확인합니다.",
              "개인 AI 대화 전체를 공유하지 않습니다. 이미 전달한 사본을 이후 권한 변경으로 회수할 수는 없습니다.",
              "저장·공식 답변 확인·문서 게시 승인은 별개입니다.",
            ]}
            confirmLabel="동의하고 관리 질문 저장"
            busy={busy}
            disabled={!consent}
            onConfirm={() => void save()}
            onCancel={() => setPending(null)}
          >
            <label className="question-choice">
              <input
                type="checkbox"
                checked={consent}
                onChange={(e) => setConsent(e.target.checked)}
              />
              선택한 질문·답변·근거와 현재 공유 범위를 확인했습니다.
            </label>
            <ErrorBox error={error} />
          </ChangeReview>
        )}
      </Modal>
      <Modal
        open={!!decision}
        onCloseAutoFocus={restoreFocus}
        onOpenChange={(v) => {
          if (!v && !busy) setDecision("");
        }}
        title={
          decision === "confirmed" ? "담당자 공식 답변 확인" : "관리 질문 보관"
        }
      >
        <p>
          {decision === "confirmed"
            ? "현재 답변·근거와 검토 기한을 확인한 담당자의 기록입니다. 문서의 게시 승인을 대신하지 않으며 원문·정책이 바뀌면 공식 표시가 해제됩니다."
            : "질문을 보관 상태로 바꾸며 답변 문서를 삭제하지 않습니다."}
        </p>
        <Field label="확인 이유">
          <textarea
            aria-label="확인 이유"
            value={note}
            onChange={(e) => setNote(e.target.value)}
          />
        </Field>
        <label className="question-choice">
          <input
            type="checkbox"
            checked={consent}
            onChange={(e) => setConsent(e.target.checked)}
          />
          현재 조건과 처리 영향을 확인했습니다.
        </label>
        <ErrorBox error={error} />
        <Button
          variant="primary"
          disabled={busy || !consent || !note.trim()}
          onClick={() => void decide()}
        >
          {decision === "confirmed" ? "현재 답변 확인 기록" : "보관 기록"}
        </Button>
      </Modal>
    </div>
  );
}
