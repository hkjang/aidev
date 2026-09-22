import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import {
  ArrowRight,
  Camera,
  Inbox,
  Link2,
  Lock,
  Paperclip,
  Plus,
  RefreshCw,
  X,
} from "lucide-react";
import { api, date, type Doc } from "./api";
import { useApp } from "./context";
import {
  Badge,
  Button,
  Empty,
  ErrorBox,
  Field,
  Modal,
  PageHeading,
} from "./ui";
import "./inbox.css";
type Row = Record<string, any>;
const kinds: Record<string, string> = {
  page: "문서",
  note: "노트",
  daily: "일일 노트",
  meeting: "회의록",
  decision: "결정 기록",
  runbook: "운영 절차",
  template: "템플릿",
  entity: "엔티티",
};
export default function InboxPage() {
  const captureRequest = useRef({ payload: "", id: "" }),
    fileRequests = useRef(new WeakMap<File, string>());
  const { workspace, user, documents, reload, notify } = useApp(),
    [query] = useSearchParams();
  const [items, setItems] = useState<Row[]>([]),
    [spaces, setSpaces] = useState<Row[]>([]),
    [title, setTitle] = useState(query.get("title") || ""),
    [text, setText] = useState(query.get("text") || ""),
    [url, setURL] = useState(query.get("url") || ""),
    [tags, setTags] = useState(""),
    [files, setFiles] = useState<File[]>([]),
    [pending, setPending] = useState<Doc | null>(null),
    [busy, setBusy] = useState(false),
    [error, setError] = useState<unknown>(null),
    [more, setMore] = useState(false),
    [loading, setLoading] = useState(true),
    [selected, setSelected] = useState<Doc | null>(null),
    [kind, setKind] = useState("note"),
    [space, setSpace] = useState(""),
    [parent, setParent] = useState(""),
    [visibility, setVisibility] = useState("private"),
    [confirmed, setConfirmed] = useState(false);
  const writable =
    !!workspace &&
    ["owner", "admin", "editor"].includes(workspace.role) &&
    user.role !== "viewer";
  const load = useCallback(
    async (after = "") => {
      if (!workspace) return;
      setLoading(true);
      try {
        const rows = await api<Row[]>(
          `/captures?workspace_id=${workspace.id}${after ? `&after=${after}` : ""}`,
        );
        setItems((prev) =>
          after
            ? [...prev, ...rows.filter((r) => !prev.some((p) => p.id === r.id))]
            : rows,
        );
        setMore(rows.length === 200);
      } catch (e) {
        setError(e);
      } finally {
        setLoading(false);
      }
    },
    [workspace?.id],
  );
  useEffect(() => {
    void load();
  }, [load]);
  function addFiles(next: FileList | null) {
    if (!next) return;
    const incoming = Array.from(next);
    if (
      incoming.some((f) => f.size > 50 * 1024 * 1024) ||
      files.length + incoming.length > 10
    ) {
      setError(
        new Error(
          "파일은 각각 50MB 이하, 한 번에 10개까지 수집할 수 있습니다.",
        ),
      );
      return;
    }
    setFiles((prev) => [...prev, ...incoming]);
    setError(null);
  }
  async function capture() {
    if (!workspace) return;
    setBusy(true);
    setError(null);
    let saved = pending;
    try {
      if (!saved) {
        const payload = {
          workspace_id: workspace.id,
          title,
          text,
          url,
          tags: tags
            .split(",")
            .map((t) => t.trim())
            .filter(Boolean),
        };
        const signature = JSON.stringify(payload);
        if (captureRequest.current.payload !== signature)
          captureRequest.current = {
            payload: signature,
            id: crypto.randomUUID(),
          };
        saved = await api<Doc>("/captures", "POST", {
          ...payload,
          client_request_id: captureRequest.current.id,
        });
        setPending(saved);
      }
      let remaining = [...files];
      for (const file of files) {
        const data = new FormData();
        data.append("file", file);
        if (!fileRequests.current.has(file))
          fileRequests.current.set(file, crypto.randomUUID());
        await api(
          `/attachments?document_id=${saved.id}&client_request_id=${fileRequests.current.get(file)}`,
          "POST",
          data,
        );
        remaining = remaining.slice(1);
        setFiles(remaining);
      }
      setPending(null);
      captureRequest.current = { payload: "", id: "" };
      setTitle("");
      setText("");
      setURL("");
      setTags("");
      await load();
      await reload();
      notify("내 수집함에 비공개로 저장했습니다");
    } catch (e) {
      setError(
        saved
          ? new Error(
              `메모는 저장되었습니다. 남은 파일을 다시 첨부할 수 있습니다. ${e instanceof Error ? e.message : ""}`,
            )
          : e,
      );
      if (saved) await load();
    } finally {
      setBusy(false);
    }
  }
  async function openClassify(item: Row) {
    setBusy(true);
    setError(null);
    try {
      const [doc, list] = await Promise.all([
        api<Doc>(`/documents/${item.id}`),
        api<Row[]>(`/spaces?workspace_id=${workspace?.id}`),
      ]);
      setSelected(doc);
      setSpaces(list);
      setKind("note");
      setSpace("");
      setParent("");
      setVisibility("private");
      setConfirmed(false);
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  }
  async function classify() {
    if (!selected) return;
    setBusy(true);
    setError(null);
    try {
      await api(`/captures/${selected.id}/classify`, "POST", {
        version: selected.version,
        kind,
        space_id: space,
        parent_id: parent,
        visibility,
      });
      setSelected(null);
      await load();
      await reload();
      notify("문서를 분류했습니다. 수집함에서만 제외되며 원문은 유지됩니다.");
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="page inbox-page">
      <PageHeading
        eyebrow="QUICK CAPTURE"
        title="내 수집함"
        description="생각이 떠오를 때 먼저 담아두세요. 나중에 문서·노트·회의록으로 분류할 수 있습니다."
        actions={
          <Button
            disabled={loading || busy}
            onClick={() => {
              setError(null);
              void load();
            }}
          >
            <RefreshCw size={16} />
            새로 불러오기
          </Button>
        }
      />
      <ErrorBox error={selected ? null : error} />
      <div className="inbox-layout">
        <section className="panel padded capture-panel">
          <div className="capture-intro">
            <Inbox size={23} />
            <div>
              <h2>빠르게 수집</h2>
              <p>
                <Lock size={14} /> 저장한 내용은 나에게만 보입니다
              </p>
            </div>
          </div>
          {!writable ? (
            <p className="muted">
              수집하려면 워크스페이스 작성 권한이 필요합니다.
            </p>
          ) : (
            <form
              onSubmit={(e) => {
                e.preventDefault();
                void capture();
              }}
              onDragOver={(e) => e.preventDefault()}
              onDrop={(e) => {
                e.preventDefault();
                if (!busy && !pending) addFiles(e.dataTransfer.files);
              }}
            >
              <fieldset disabled={busy || !!pending}>
                <Field
                  label="수집 제목"
                  hint="비워두면 내용의 첫 줄 또는 URL에서 정합니다."
                >
                  <input
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                    maxLength={150}
                    placeholder="아이디어, 참고자료, 빠른 메모…"
                  />
                </Field>
                <Field label="수집할 내용">
                  <textarea
                    rows={8}
                    value={text}
                    onChange={(e) => setText(e.target.value)}
                    placeholder="Markdown으로 자유롭게 기록하세요."
                  />
                </Field>
                <Field
                  label="출처 URL"
                  hint="링크만 저장합니다. 서버가 해당 사이트에 접속하지 않습니다."
                >
                  <input
                    type="url"
                    value={url}
                    onChange={(e) => setURL(e.target.value)}
                    placeholder="https://"
                    maxLength={8192}
                  />
                </Field>
                <Field label="수집 태그">
                  <input
                    value={tags}
                    onChange={(e) => setTags(e.target.value)}
                    placeholder="쉼표로 구분: 아이디어, 참고"
                  />
                </Field>
                <div className="capture-file-actions">
                  <label className="button">
                    <Paperclip size={16} />
                    파일 첨부
                    <input
                      aria-label="수집 파일 첨부"
                      type="file"
                      multiple
                      onChange={(e) => {
                        addFiles(e.target.files);
                        e.target.value = "";
                      }}
                    />
                  </label>
                  <label className="button">
                    <Camera size={16} />
                    사진 촬영
                    <input
                      aria-label="수집 사진 촬영"
                      type="file"
                      accept="image/*"
                      capture="environment"
                      onChange={(e) => {
                        addFiles(e.target.files);
                        e.target.value = "";
                      }}
                    />
                  </label>
                </div>
              </fieldset>
              {files.length > 0 && (
                <ul className="capture-files">
                  {files.map((f, i) => (
                    <li key={`${f.name}-${i}`}>
                      <Paperclip size={14} />
                      <span>{f.name}</span>
                      <button
                        type="button"
                        className="icon-button"
                        disabled={busy}
                        aria-label={`${f.name} 첨부에서 제외`}
                        onClick={() =>
                          setFiles(files.filter((_, n) => n !== i))
                        }
                      >
                        <X size={14} />
                      </button>
                    </li>
                  ))}
                </ul>
              )}
              {pending && (
                <div className="capture-saved">
                  <p>
                    메모가 이미 저장되어 있습니다. 재시도해도 메모가 중복
                    생성되지 않습니다.
                  </p>
                  <Link to={`/app/documents/${pending.id}`}>
                    저장한 문서 확인
                  </Link>
                  <Button
                    type="button"
                    disabled={busy}
                    onClick={() => {
                      setPending(null);
                      captureRequest.current = { payload: "", id: "" };
                      setFiles([]);
                      setTitle("");
                      setText("");
                      setURL("");
                      setTags("");
                      setError(null);
                      void reload();
                    }}
                  >
                    남은 첨부 없이 마치기
                  </Button>
                </div>
              )}
              <Button
                variant="primary"
                disabled={
                  busy ||
                  (!pending &&
                    !text.trim() &&
                    !url.trim() &&
                    !title.trim() &&
                    !files.length)
                }
              >
                <Plus size={16} />
                {busy
                  ? "저장 중…"
                  : pending
                    ? "남은 첨부 다시 저장"
                    : "내 수집함에 저장"}
              </Button>
              <p className="muted capture-hint">
                파일을 끌어 놓아도 됩니다. 파일당 50MB, 최대 10개.
              </p>
            </form>
          )}
        </section>
        <section className="inbox-items">
          <div className="inbox-list-heading">
            <h2>아직 분류하지 않은 기록</h2>
            <Badge>
              {items.length}
              {more ? "+" : ""}개
            </Badge>
          </div>
          {loading && !items.length ? (
            <p className="muted">수집함을 불러오는 중입니다…</p>
          ) : items.length === 0 ? (
            <div className="panel padded">
              <Empty
                title="수집함이 비어 있습니다"
                text="빠른 수집에 메모나 URL을 입력하세요. 파일과 사진도 함께 담을 수 있습니다."
              />
            </div>
          ) : (
            items.map((item) => (
              <article className="panel capture-card" key={item.id}>
                <div className="capture-card-heading">
                  <Link to={`/app/documents/${item.id}`}>{item.title}</Link>
                  <Badge>
                    <Lock size={12} />
                    {item.visibility === "private" ? "개인" : "분류 대기"}
                  </Badge>
                </div>
                <p className="capture-preview">
                  {item.preview || "첨부파일 또는 제목만 저장한 기록입니다."}
                </p>
                {item.source_url && (
                  <a
                    className="capture-source"
                    href={item.source_url}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    <Link2 size={14} />
                    {item.source_url}
                  </a>
                )}
                <div className="capture-tags">
                  {(item.tags || []).map((tag: string, i: number) => (
                    <Badge key={`${tag}-${i}`}>#{tag}</Badge>
                  ))}
                </div>
                <footer>
                  <span>{date(item.created_at)} 저장</span>
                  <Button
                    disabled={busy || !writable}
                    onClick={() => void openClassify(item)}
                  >
                    문서로 분류
                    <ArrowRight size={15} />
                  </Button>
                </footer>
              </article>
            ))
          )}
          {more && (
            <Button
              disabled={loading}
              onClick={() => void load(items.at(-1)?.id)}
            >
              이전 수집 기록 더 보기
            </Button>
          )}
        </section>
      </div>
      {selected && (
        <Modal
          open
          title="수집 기록 분류"
          onOpenChange={(open) => {
            if (!open && !busy) setSelected(null);
          }}
        >
          <form
            onSubmit={(e) => {
              e.preventDefault();
              void classify();
            }}
          >
            <p className="muted">
              {selected.title} · v{selected.version}
            </p>
            <ErrorBox error={error} />
            <Field label="분류할 문서 종류">
              <select value={kind} onChange={(e) => setKind(e.target.value)}>
                {Object.entries(kinds).map(([k, label]) => (
                  <option key={k} value={k}>
                    {label}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="대상 공간">
              <select value={space} onChange={(e) => setSpace(e.target.value)}>
                <option value="">워크스페이스 기본 공간</option>
                {spaces
                  .filter((s) => s.can_write)
                  .map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.name}
                    </option>
                  ))}
              </select>
            </Field>
            <Field label="상위 문서">
              <select
                value={parent}
                onChange={(e) => setParent(e.target.value)}
              >
                <option value="">최상위 문서</option>
                {documents
                  .filter(
                    (d) => d.id !== selected.id && d.can_write && !d.deleted_at,
                  )
                  .map((d) => (
                    <option key={d.id} value={d.id}>
                      {d.title}
                    </option>
                  ))}
              </select>
            </Field>
            <Field label="분류 후 공유 범위">
              <select
                value={visibility}
                onChange={(e) => {
                  setVisibility(e.target.value);
                  setConfirmed(false);
                }}
              >
                <option value="private">개인 · 나만 보기</option>
                <option value="workspace">
                  워크스페이스 · 공간 권한 내 공유
                </option>
                <option value="selected">
                  선택한 사용자 · 공유 대상은 문서에서 지정
                </option>
              </select>
            </Field>
            {visibility !== "private" && (
              <label className="capture-confirm">
                <input
                  type="checkbox"
                  checked={confirmed}
                  onChange={(e) => setConfirmed(e.target.checked)}
                />
                <span>이 문서의 공유 범위가 변경되는 것을 확인했습니다.</span>
              </label>
            )}
            <p className="muted">
              분류하면 수집함에서 제외됩니다. 원문·첨부파일·문서 URL은 그대로
              유지됩니다.
            </p>
            <div className="modal-actions">
              <Button
                type="button"
                disabled={busy}
                onClick={() => setSelected(null)}
              >
                취소
              </Button>
              <Button
                variant="primary"
                disabled={busy || (visibility !== "private" && !confirmed)}
              >
                분류 저장
              </Button>
            </div>
          </form>
        </Modal>
      )}
    </div>
  );
}
