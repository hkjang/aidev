import { Fragment, useCallback, useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import {
  CheckCircle2,
  MessageSquare,
  Pencil,
  Quote,
  Reply,
  Send,
  Trash2,
  Users,
} from "lucide-react";
import { api, datetime, type Doc } from "./api";
import { useApp } from "./context";
import { Badge, Button, ErrorBox, Field, Modal } from "./ui";
import "./discussion.css";
type Comment = Record<string, any>;
const reactions: Record<string, string> = {
  like: "👍",
  thanks: "🙏",
  idea: "💡",
  check: "✅",
  heart: "❤️",
};
export function MentionText({ text }: { text: string }) {
  const result = [];
  let offset = 0;
  for (const m of text.matchAll(
    /@\[([^\]\n]+)\]\((user|team):([a-fA-F0-9-]{36})\)/g,
  )) {
    result.push(
      <Fragment key={`t${offset}`}>{text.slice(offset, m.index)}</Fragment>,
    );
    result.push(
      <span className="discussion-mention" key={`m${m.index}`}>
        @{m[1]}
      </span>,
    );
    offset = m.index! + m[0].length;
  }
  result.push(<Fragment key="end">{text.slice(offset)}</Fragment>);
  return <>{result}</>;
}
export default function DiscussionPanel({ document: doc }: { document: Doc }) {
  const { user, notify } = useApp();
  const [comments, setComments] = useState<Comment[]>([]),
    [members, setMembers] = useState<Comment[]>([]),
    [teams, setTeams] = useState<Comment[]>([]),
    [error, setError] = useState<unknown>(null),
    [busy, setBusy] = useState(false),
    [body, setBody] = useState(""),
    [parent, setParent] = useState(""),
    [quote, setQuote] = useState(""),
    [block, setBlock] = useState(""),
    [assigned, setAssigned] = useState(""),
    [advanced, setAdvanced] = useState(false),
    [filter, setFilter] = useState("all"),
    [editing, setEditing] = useState<Comment | null>(null),
    [editBody, setEditBody] = useState(""),
    [remove, setRemove] = useState<Comment | null>(null),
    [more, setMore] = useState(false);
  const generation = useRef(0),
    composer = useRef<HTMLTextAreaElement>(null),
    canWrite = !!doc.can_comment && !doc.deleted_at,
    base = `/documents/${doc.id}/comments`;
  const load = useCallback(
    async (append = false) => {
      const key = ++generation.current;
      try {
        const after =
          append && comments.length ? `&after=${comments.at(-1)!.id}` : "";
        const rows = await api<Comment[]>(`${base}?limit=200${after}`);
        if (key !== generation.current) return;
        setComments((old) => (append ? [...old, ...rows] : rows));
        setMore(rows.length === 200);
        setError(null);
      } catch (e) {
        if (key === generation.current) {
          setError(e);
          setComments([]);
        }
      }
    },
    [base, comments],
  );
  useEffect(() => {
    let active = true;
    setComments([]);
    setParent("");
    setBody("");
    setQuote("");
    setBlock("");
    setAssigned("");
    void api<Comment[]>(base + "?limit=200")
      .then((v) => {
        if (active) {
          setComments(v);
          setMore(v.length === 200);
        }
      })
      .catch((e) => {
        if (active) setError(e);
      });
    Promise.all([
      api<Comment[]>(`/workspaces/${doc.workspace_id}/members`),
      api<Comment[]>(`/teams?workspace_id=${doc.workspace_id}`),
    ])
      .then(([m, t]) => {
        if (active) {
          setMembers(m);
          setTeams(t);
        }
      })
      .catch((e) => {
        if (active) setError(e);
      });
    return () => {
      active = false;
      generation.current++;
    };
  }, [base, doc.workspace_id]);
  const run = async (action: () => Promise<unknown>) => {
    setBusy(true);
    setError(null);
    try {
      await action();
      await load();
      return true;
    } catch (e) {
      setError(e);
      return false;
    } finally {
      setBusy(false);
    }
  };
  const insertMention = (value: string) => {
    const [type, id] = value.split(":");
    const v = (type === "user" ? members : teams).find((m) => m.id === id);
    if (!v) return;
    const label = String(v.name).replace(/[\[\]\n()]/g, "_");
    setBody(
      (old) =>
        `${old}${old && !old.endsWith(" ") ? " " : ""}@[${label}](${type}:${id}) `,
    );
    composer.current?.focus();
  };
  const depth = (c: Comment) => {
    let n = 0,
      p = c.parent_id;
    const seen = new Set<string>();
    while (p && !seen.has(p) && n < 4) {
      seen.add(p);
      n++;
      p = comments.find((x) => x.id === p)?.parent_id;
    }
    return n;
  };
  const filtered = comments.filter(
    (c) =>
      filter === "all" ||
      (filter === "open"
        ? !c.resolved_at
        : filter === "resolved"
          ? !!c.resolved_at
          : c.assigned_to === user.id),
  );
  return (
    <section className="discussion-panel" id="discussion">
      <div className="discussion-heading">
        <h3>
          <MessageSquare size={20} />
          댓글과 토론{" "}
          <span>
            {comments.length}
            {more ? "+" : ""}
          </span>
        </h3>
        <Button type="button" onClick={() => void load()} disabled={busy}>
          새로고침
        </Button>
      </div>
      <ErrorBox error={error} />
      <div className="discussion-filters">
        {[
          ["all", "전체"],
          ["open", "미해결"],
          ["resolved", "해결됨"],
          ["mine", "내 담당"],
        ].map(([k, v]) => (
          <button
            type="button"
            key={k}
            className={filter === k ? "active" : ""}
            onClick={() => setFilter(k)}
          >
            {v}
          </button>
        ))}
      </div>
      <div className="discussion-list">
        {filtered.length === 0 ? (
          <p className="muted">
            조건에 맞는 토론이 없습니다. 첫 의견을 남겨보세요.
          </p>
        ) : (
          filtered.map((c) => (
            <article
              className={`discussion-comment ${c.resolved_at ? "resolved" : ""}`}
              key={c.id}
              id={`comment-${c.id}`}
              style={{ "--thread-depth": depth(c) } as React.CSSProperties}
            >
              <div className="discussion-avatar avatar small">
                {c.user_name?.slice(0, 1) || "M"}
              </div>
              <div className="discussion-content">
                <header>
                  <strong>{c.user_name}</strong>
                  <time>{datetime(c.created_at)}</time>
                  {c.resolved_at && <Badge tone="green">해결됨</Badge>}
                  {c.edited_at && <small>수정됨</small>}
                </header>
                {c.parent_id && (
                  <a
                    className="discussion-parent"
                    href={`#comment-${c.parent_id}`}
                  >
                    <Reply size={14} />
                    {comments.find((x) => x.id === c.parent_id)?.user_name ||
                      "이전 댓글"}
                    님에게 답글
                  </a>
                )}
                {c.quote && (
                  <blockquote>
                    <Quote size={14} />
                    {c.quote}
                  </blockquote>
                )}
                {c.block_id && (
                  <Link
                    className="discussion-parent"
                    to={`/app/documents/${doc.id}#${encodeURIComponent("^" + c.block_id)}`}
                  >
                    참조 블록으로 이동
                  </Link>
                )}
                {!c.anchor_current && (
                  <small className="discussion-anchor-warning">
                    참조 위치가 변경되었습니다. 인용은 작성 당시의 내용입니다.
                  </small>
                )}
                <p className={c.deleted_at ? "muted" : ""}>
                  {c.deleted_at ? (
                    "작성자가 삭제한 댓글입니다."
                  ) : (
                    <MentionText text={c.body} />
                  )}
                </p>
                {c.assigned_name && (
                  <div className="discussion-assignee">
                    <Users size={14} />
                    {c.assigned_name} 담당
                  </div>
                )}
                {!c.deleted_at && (
                  <div className="discussion-actions">
                    {Object.entries(reactions).map(([key, emoji]) => {
                      const r = c.reactions?.find(
                        (v: Comment) => v.reaction === key,
                      );
                      return (
                        <button
                          type="button"
                          disabled={!canWrite || busy}
                          key={key}
                          className={r?.mine ? "active" : ""}
                          aria-label={`${c.user_name} 댓글 ${key} 반응`}
                          onClick={() =>
                            void run(() =>
                              api(`${base}/${c.id}/reactions`, "POST", {
                                reaction: key,
                                remove: !!r?.mine,
                              }),
                            )
                          }
                        >
                          {emoji}
                          {r?.count || ""}
                        </button>
                      );
                    })}
                    {canWrite && (
                      <Button
                        type="button"
                        onClick={() => {
                          setParent(c.id);
                          composer.current?.focus();
                          composer.current?.scrollIntoView({
                            block: "center",
                            behavior: "smooth",
                          });
                        }}
                      >
                        <Reply size={14} />
                        답글
                      </Button>
                    )}
                    {canWrite &&
                      [c.user_id, c.assigned_to, doc.owner_id].includes(
                        user.id,
                      ) && (
                        <Button
                          type="button"
                          disabled={busy}
                          onClick={() =>
                            void run(() =>
                              api(`${base}/${c.id}`, "PATCH", {
                                resolved: !c.resolved_at,
                              }),
                            )
                          }
                        >
                          <CheckCircle2 size={14} />
                          {c.resolved_at ? "다시 열기" : "해결"}
                        </Button>
                      )}
                    {c.can_edit && canWrite && (
                      <>
                        <Button
                          type="button"
                          onClick={() => {
                            setEditing(c);
                            setEditBody(c.body);
                          }}
                        >
                          <Pencil size={14} />
                          수정
                        </Button>
                        <Button type="button" onClick={() => setRemove(c)}>
                          <Trash2 size={14} />
                          삭제
                        </Button>
                      </>
                    )}
                  </div>
                )}
              </div>
            </article>
          ))
        )}
      </div>
      {more && (
        <Button type="button" disabled={busy} onClick={() => void load(true)}>
          이전 목록에 이어 더 불러오기
        </Button>
      )}
      <form
        className="discussion-composer"
        onSubmit={async (e) => {
          e.preventDefault();
          setBusy(true);
          setError(null);
          try {
            await api(base, "POST", {
              body,
              parent_id: parent,
              quote,
              block_id: block,
              assigned_to: assigned,
              document_version: doc.version,
            });
            setBody("");
            setParent("");
            setQuote("");
            setBlock("");
            setAssigned("");
            await load();
            notify("댓글을 등록했습니다.");
          } catch (e) {
            setError(e);
          } finally {
            setBusy(false);
          }
        }}
      >
        {parent && (
          <div className="discussion-replying">
            <span>
              {comments.find((c) => c.id === parent)?.user_name}님에게 답글 작성
              중
            </span>
            <Button type="button" onClick={() => setParent("")}>
              답글 취소
            </Button>
          </div>
        )}
        <textarea
          ref={composer}
          value={body}
          disabled={!canWrite || busy}
          onChange={(e) => setBody(e.target.value)}
          placeholder="생각을 나누거나 의견을 남겨보세요."
          required
          aria-label="댓글 내용"
          rows={3}
        />
        <div className="discussion-tools">
          <Field label="사용자·팀 언급">
            <select
              value=""
              disabled={!canWrite || busy}
              onChange={(e) => insertMention(e.target.value)}
            >
              <option value="">언급할 대상 선택</option>
              <optgroup label="사용자">
                {members.map((m) => (
                  <option key={m.id} value={`user:${m.id}`}>
                    {m.name}
                  </option>
                ))}
              </optgroup>
              <optgroup label="팀">
                {teams.map((t) => (
                  <option key={t.id} value={`team:${t.id}`}>
                    {t.name}
                  </option>
                ))}
              </optgroup>
            </select>
          </Field>
          <Button
            type="button"
            onClick={() => setAdvanced((v) => !v)}
            disabled={!canWrite}
          >
            <Quote size={16} />
            인용·담당자
          </Button>
          <Button
            variant="primary"
            disabled={!canWrite || busy || !body.trim()}
          >
            <Send size={16} />
            {busy ? "등록 중…" : "댓글 등록"}
          </Button>
        </div>
        {advanced && (
          <div className="discussion-anchor-form">
            <Button
              type="button"
              disabled={!canWrite}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => {
                const selected = window.getSelection()?.toString().trim() || "";
                if (!selected) {
                  setError("문서에서 인용할 문장을 먼저 선택하세요.");
                  return;
                }
                if (!doc.markdown.includes(selected)) {
                  setError(
                    "선택한 문장은 저장된 원문과 일치하지 않습니다. 문서를 저장한 뒤 다시 선택하세요.",
                  );
                  return;
                }
                setQuote(selected);
                setError(null);
              }}
            >
              <Quote size={15} />
              선택한 문장 인용
            </Button>
            <Field
              label="인용 문장"
              hint="현재 저장된 문서 원문에서 정확히 일치하는 문장을 입력하세요."
            >
              <textarea
                value={quote}
                onChange={(e) => setQuote(e.target.value)}
                disabled={!canWrite}
                rows={2}
              />
            </Field>
            <Field label="참조 블록">
              <select
                value={block}
                onChange={(e) => setBlock(e.target.value)}
                disabled={!canWrite}
              >
                <option value="">문서 전체</option>
                {doc.block_metadata?.blocks
                  ?.filter((b) => b.id)
                  .map((b) => (
                    <option key={b.id} value={b.id}>
                      {b.type} · {b.text.slice(0, 70) || b.id.slice(0, 8)}
                    </option>
                  ))}
              </select>
            </Field>
            <Field
              label="토론 담당자"
              hint="문서 접근 권한은 따로 부여해야 합니다."
            >
              <select
                value={assigned}
                onChange={(e) => setAssigned(e.target.value)}
                disabled={!canWrite}
              >
                <option value="">지정하지 않음</option>
                {members.map((m) => (
                  <option key={m.id} value={m.id}>
                    {m.name}
                  </option>
                ))}
              </select>
            </Field>
          </div>
        )}
      </form>
      <Modal
        open={!!editing}
        onOpenChange={(v) => !v && setEditing(null)}
        title="댓글 수정"
      >
        <form
          onSubmit={async (e) => {
            e.preventDefault();
            if (!editing) return;
            if (
              await run(() =>
                api(`${base}/${editing.id}`, "PATCH", { body: editBody }),
              )
            )
              setEditing(null);
          }}
        >
          <ErrorBox error={error} />
          <Field label="수정할 댓글">
            <textarea
              value={editBody}
              onChange={(e) => setEditBody(e.target.value)}
              rows={5}
              required
            />
          </Field>
          <Button variant="primary" disabled={busy || !editBody.trim()}>
            수정 저장
          </Button>
        </form>
      </Modal>
      <Modal
        open={!!remove}
        onOpenChange={(v) => !v && setRemove(null)}
        title="댓글을 삭제할까요?"
        description="댓글 본문과 인용이 삭제됩니다. 이 댓글에 달린 답글은 남아 있습니다."
      >
        <ErrorBox error={error} />
        <Button
          variant="danger"
          disabled={busy}
          onClick={async () => {
            if (!remove) return;
            if (await run(() => api(`${base}/${remove.id}`, "DELETE")))
              setRemove(null);
          }}
        >
          댓글 삭제
        </Button>
      </Modal>
    </section>
  );
}
