import { lazy, Suspense, useEffect, useRef, useState } from "react";
import { Link, NavLink, useNavigate, useParams } from "react-router-dom";
import {
  ArrowDown,
  ArrowUp,
  BookOpen,
  Eye,
  Plus,
  RefreshCw,
  Trash2,
} from "lucide-react";
import { api, ApiError, datetime } from "../api";
import { useApp } from "../context";
import {
  Badge,
  Button,
  Empty,
  Field,
  Loading,
  Modal,
  PageHeading,
} from "../ui";
import { ChangeReview, RecoveryNotice } from "../review/ChangeReview";
import { useModalReturnFocus } from "../review/useModalReturnFocus";
import { restoreWorksetPosition } from "../navigation/NavigationMemory";
import DocumentPassport from "./DocumentPassport";
import type { Workset, WorksetItem, WorksetData, Resolved } from "./types";
import "./style.css";
const DocumentPreview = lazy(() => import("../review/DocumentPreview"));
export default function WorksetsPage() {
  const { user, workspace, documents, notify } = useApp(),
    navigate = useNavigate(),
    id = useParams()["*"] || "";
  const [list, setList] = useState<any[]>([]),
    [data, setData] = useState<WorksetData | null>(null),
    [error, setError] = useState<unknown>(null),
    [mutationError, setMutationError] = useState<unknown>(null),
    [busy, setBusy] = useState(false),
    [refresh, setRefresh] = useState(0),
    [editing, setEditing] = useState<Workset | null>(null),
    [deleting, setDeleting] = useState<{
      name: string;
      version: number;
    } | null>(null),
    [passport, setPassport] = useState<string | null>(null),
    [preview, setPreview] = useState<string | null>(null),
    [compare, setCompare] = useState<[string, string]>(["", ""]);
  const scope = `${user.id}:${workspace?.id}:${id}`,
    current = useRef(scope),
    loaded = useRef(""),
    alive = useRef(true);
  current.current = scope;
  const deleteReturnFocus = useModalReturnFocus(
    !!deleting,
    `${user.id}:${workspace?.id}`,
  );
  useEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
    };
  }, []);
  useEffect(() => {
    setData(null);
    setList([]);
    setError(null);
    setMutationError(null);
    setBusy(false);
    setEditing(null);
    setDeleting(null);
    setPassport(null);
    setPreview(null);
    setCompare(["", ""]);
    loaded.current = "";
    if (!workspace) return;
    const ctl = new AbortController();
    let pending = false;
    const load = async () => {
      if (pending) return;
      pending = true;
      try {
        const [items, item] = await Promise.all([
          api(`/worksets?workspace_id=${workspace.id}`, "GET", undefined, {
            signal: ctl.signal,
          }),
          id
            ? api<WorksetData>(`/worksets/${id}`, "GET", undefined, {
                signal: ctl.signal,
              })
            : Promise.resolve(null),
        ]);
        if (!ctl.signal.aborted && current.current === scope) {
          if (item && item.workset.workspace_id !== workspace.id)
            throw new Error("현재 워크스페이스의 작업 묶음이 아닙니다.");
          setList(items.items);
          setData(item);
          loaded.current = scope;
          setError(null);
          setCompare(
            (old) =>
              old.map((value) =>
                item?.resolved.some(
                  (r) =>
                    r.available &&
                    r.kind === "document" &&
                    r.resource_id === value,
                )
                  ? value
                  : "",
              ) as [string, string],
          );
        }
      } catch (e) {
        if (!ctl.signal.aborted && current.current === scope) {
          setData(null);
          setList([]);
          setCompare(["", ""]);
          setPassport(null);
          setPreview(null);
          setError(e);
          if (e instanceof ApiError && [401, 403, 404].includes(e.status))
            setEditing(null);
        }
      } finally {
        pending = false;
      }
    };
    void load();
    const timer = setInterval(() => void load(), 2000);
    return () => {
      ctl.abort();
      clearInterval(timer);
    };
  }, [scope, refresh]);
  const visible = loaded.current === scope ? data : null;
  const openEditor = () =>
    setEditing(
      visible
        ? structuredClone(visible.workset)
        : {
            id: "",
            workspace_id: workspace?.id || "",
            owner_id: user.id,
            name: "새 작업 묶음",
            kind: "workset",
            version: 0,
            items: [],
          },
    );
  const docs = (visible?.resolved || []).filter(
    (r) => r.available && r.kind === "document",
  );
  const remove = async () => {
    if (!visible || !deleting || busy) return;
    setBusy(true);
    try {
      await api(`/worksets/${id}`, "DELETE", {
        expected_version: deleting.version,
      });
      if (alive.current && current.current === scope) {
        setDeleting(null);
        navigate("/app/worksets");
        notify("개인 작업 묶음만 삭제했습니다. 원문은 삭제하지 않았습니다.");
      }
    } catch (e) {
      if (alive.current && current.current === scope) setMutationError(e);
    } finally {
      if (alive.current && current.current === scope) setBusy(false);
    }
  };
  return (
    <div className="worksets-page">
      <PageHeading
        eyebrow="나의 업무 맥락"
        title="작업 묶음과 참고 선반"
        description="필요한 문서·보기·할 일을 모으고 현재 권한으로 다시 이어갑니다."
        actions={
          <Button
            variant="primary"
            onClick={() =>
              setEditing({
                id: "",
                workspace_id: workspace?.id || "",
                owner_id: user.id,
                name: "새 작업 묶음",
                kind: "workset",
                version: 0,
                items: [],
              })
            }
          >
            <Plus size={18} />새 개인 묶음
          </Button>
        }
      />
      <RecoveryNotice error={error} />
      <div className="worksets-layout">
        <nav className="worksets-list" aria-label="내 작업 묶음">
          {(loaded.current === scope ? list : []).map((item) => (
            <NavLink key={item.id} to={`/app/worksets/${item.id}`}>
              <strong>{item.name}</strong>
              <span>
                {item.kind === "reference" ? "참고 선반" : "작업 묶음"} ·{" "}
                {item.item_count}개
              </span>
              <small>{datetime(item.updated_at)}</small>
            </NavLink>
          ))}
        </nav>
        <div>
          {visible ? (
            <>
              <div className="page-heading">
                <div>
                  <Badge>나만 보기</Badge>
                  <h2>{visible.workset.name}</h2>
                </div>
                <div className="workset-actions">
                  <Button onClick={openEditor}>묶음 편집</Button>
                  <Button onClick={() => setRefresh((n) => n + 1)}>
                    <RefreshCw size={16} />
                    다시 확인
                  </Button>
                  <Button
                    variant="danger"
                    onClick={() => {
                      setMutationError(null);
                      setDeleting({
                        name: visible.workset.name,
                        version: visible.workset.version,
                      });
                    }}
                  >
                    <Trash2 size={16} />
                    묶음 삭제
                  </Button>
                </div>
              </div>
              <p className="notice subtle">{visible.notice}</p>
              <div className="workset-items">
                {visible.resolved.map((item, i) => (
                  <article
                    className="panel workset-item"
                    key={`${i}:${item.resource_id}`}
                  >
                    <Badge>
                      {
                        {
                          document: "문서",
                          database: "데이터베이스",
                          task: "할 일",
                        }[item.kind]
                      }
                    </Badge>
                    <h3>
                      {item.available
                        ? item.task_text || item.title
                        : "현재 접근 불가"}
                    </h3>
                    {!item.available ? (
                      <p className="muted">
                        본인이 이전에 보관한 참조입니다. 현재 원문·제목은
                        표시하지 않습니다.
                      </p>
                    ) : (
                      <>
                        {item.context_changed && (
                          <p className="notice">{item.context_notice}</p>
                        )}
                        {item.stale && (
                          <p className="muted">
                            마지막 변경 후 90일 이상 지났습니다. 내용 오류라는
                            뜻은 아닙니다.
                          </p>
                        )}
                        {item.kind === "document" && !item.tags?.length && (
                          <p className="muted">
                            등록된 태그가 없습니다. 문서 여권에서 본문 기반
                            후보를 확인할 수 있습니다.
                          </p>
                        )}
                        <div className="workset-actions">
                          <Link
                            className="button primary"
                            to={item.url!}
                            onClick={() => {
                              if (item.restore_context && workspace)
                                restoreWorksetPosition(
                                  user.id,
                                  workspace.id,
                                  item.url!,
                                  item.restore_context,
                                );
                            }}
                          >
                            저장한 맥락으로 열기
                          </Link>
                          {item.kind === "document" && (
                            <>
                              <Button
                                onClick={() => setPreview(item.resource_id)}
                              >
                                <Eye size={16} />
                                미리보기
                              </Button>
                              <Button
                                onClick={() => setPassport(item.resource_id)}
                              >
                                <BookOpen size={16} />
                                문서 여권
                              </Button>
                            </>
                          )}
                        </div>
                      </>
                    )}
                  </article>
                ))}
              </div>
              {!visible.resolved.length && (
                <Empty
                  title="참조를 추가해 보세요"
                  text="묶음 편집에서 선택하거나 문서·데이터베이스 화면의 보관 버튼을 이용하세요."
                />
              )}
              {docs.length > 0 && (
                <section>
                  <h3>참고 선반 · 선택한 문서 비교</h3>
                  <p className="muted">
                    원문은 선택할 때 현재 권한으로 읽으며 브라우저 저장소에
                    보관하지 않습니다.
                  </p>
                  <div className="workset-comparison">
                    {([0, 1] as const).map((side) => (
                      <div className="workset-compare-slot" key={side}>
                        <Field label={`${side + 1}번째 비교 문서`}>
                          <select
                            value={compare[side]}
                            onChange={(e) =>
                              setCompare((old) =>
                                side === 0
                                  ? [e.target.value, old[1]]
                                  : [old[0], e.target.value],
                              )
                            }
                          >
                            <option value="">선택하지 않음</option>
                            {docs.map((d, i) => (
                              <option value={d.resource_id} key={i}>
                                {d.title}
                              </option>
                            ))}
                          </select>
                        </Field>
                        {compare[side] && (
                          <Suspense fallback={<Loading />}>
                            <DocumentPreview
                              documentId={compare[side]}
                              expectedVersion={
                                docs.find(
                                  (d) => d.resource_id === compare[side],
                                )?.version
                              }
                              variant="panel"
                              onClose={() =>
                                setCompare((old) =>
                                  side === 0 ? ["", old[1]] : [old[0], ""],
                                )
                              }
                            />
                          </Suspense>
                        )}
                      </div>
                    ))}
                  </div>
                </section>
              )}
            </>
          ) : (
            !error && (
              <Empty
                title="업무 맥락을 하나로 모으세요"
                text="개인 작업 묶음은 최대20개 참조, 참고 선반은 문서6개를 저장합니다. 원문 권한과 공개 범위는 바뀌지 않습니다."
              />
            )
          )}
        </div>
      </div>
      {loaded.current === scope && editing && (
        <WorksetEditor
          key={editing.id || "new"}
          value={editing}
          onClose={() => setEditing(null)}
          onSaved={(saved) => {
            setEditing(null);
            if (saved.id === id) setRefresh((n) => n + 1);
            else navigate(`/app/worksets/${saved.id}`);
          }}
        />
      )}
      <Modal
        open={!!deleting && !!visible}
        onOpenChange={(v) => {
          if (!v && !busy) setDeleting(null);
        }}
        title="개인 작업 묶음 삭제"
        onCloseAutoFocus={deleteReturnFocus}
      >
        <RecoveryNotice error={mutationError || error} />
        <ChangeReview
          title="참조 묶음만 삭제합니다"
          changes={[
            {
              label: "대상",
              before: deleting?.name,
              after: "묶음과 저장한 위치 제거",
            },
          ]}
          warnings={["실제 문서·데이터베이스·할 일은 삭제하지 않습니다."]}
          busy={busy}
          onCancel={() => setDeleting(null)}
          onConfirm={() => void remove()}
          confirmLabel="묶음만 삭제"
        />
      </Modal>
      <DocumentPassport
        documentID={passport}
        onClose={() => setPassport(null)}
        onUpdated={() => setRefresh((n) => n + 1)}
      />
      <Suspense fallback={<Loading />}>
        <DocumentPreview
          documentId={preview}
          expectedVersion={docs.find((d) => d.resource_id === preview)?.version}
          onClose={() => setPreview(null)}
        />
      </Suspense>
    </div>
  );
}
function WorksetEditor({
  value,
  onClose,
  onSaved,
}: {
  value: Workset;
  onClose: () => void;
  onSaved: (v: { id: string; version: number }) => void;
}) {
  const { user, workspace, documents } = useApp(),
    [draft, setDraft] = useState(value),
    [busy, setBusy] = useState(false),
    [error, setError] = useState<unknown>(null),
    [kind, setKind] = useState<WorksetItem["kind"]>("document"),
    [resource, setResource] = useState(""),
    [taskID, setTaskID] = useState(""),
    [sources, setSources] = useState<any[]>([]);
  const current = useRef(`${user.id}:${workspace?.id}`),
    scope = `${user.id}:${workspace?.id}`,
    alive = useRef(true);
  current.current = scope;
  useEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
    };
  }, []);
  useEffect(() => {
    setResource("");
    setTaskID("");
    setSources([]);
    if (!workspace || kind === "document") return;
    const ctl = new AbortController();
    void api(
      kind === "database"
        ? `/databases?workspace_id=${workspace.id}`
        : `/tasks/board?workspace_id=${workspace.id}`,
      "GET",
      undefined,
      { signal: ctl.signal },
    )
      .then((v) => {
        if (!ctl.signal.aborted && current.current === scope)
          setSources(
            kind === "database"
              ? v
              : v.items.filter((t: any) =>
                  /^[a-f0-9-]{36}$/i.test(t.task_id || ""),
                ),
          );
      })
      .catch((e) => {
        if (!ctl.signal.aborted) setError(e);
      });
    return () => ctl.abort();
  }, [kind, scope]);
  const selected =
    kind === "document"
      ? documents.find((d) => d.id === resource)
      : kind === "task"
        ? sources.find((t) => t.task_id === taskID)
        : sources.find((d) => d.id === resource);
  const returnFocus = useModalReturnFocus(true, scope);
  async function save() {
    if (busy) return;
    setBusy(true);
    setError(null);
    try {
      const result = await api(
        value.id ? `/worksets/${value.id}` : "/worksets",
        value.id ? "PUT" : "POST",
        {
          workspace_id: workspace?.id,
          name: draft.name,
          kind: draft.kind,
          expected_version: value.version,
          items: draft.items,
        },
      );
      if (alive.current && current.current === scope) onSaved(result);
    } catch (e) {
      if (alive.current && current.current === scope) setError(e);
    } finally {
      if (alive.current && current.current === scope) setBusy(false);
    }
  }
  return (
    <Modal
      open
      onOpenChange={(v) => {
        if (!v && !busy) onClose();
      }}
      title="내 작업 묶음 편집"
      onCloseAutoFocus={returnFocus}
      description="문서 원문이 아닌 참조 ID·보기·위치만 저장합니다."
      wide
    >
      <RecoveryNotice error={error} dirty />
      <Field label="묶음 이름">
        <input
          value={draft.name}
          disabled={busy}
          maxLength={200}
          onChange={(e) => setDraft({ ...draft, name: e.target.value })}
        />
      </Field>
      <Field label="묶음 종류">
        <select
          disabled={busy}
          value={draft.kind}
          onChange={(e) =>
            setDraft({ ...draft, kind: e.target.value as Workset["kind"] })
          }
        >
          <option value="workset">작업 묶음 · 최대20개</option>
          <option value="reference">참고 선반 · 문서 최대6개</option>
        </select>
      </Field>
      <ol className="workset-items">
        {draft.items.map((item, i) => (
          <li className="panel workset-item" key={i}>
            <span>
              {
                { document: "문서", database: "데이터베이스", task: "할 일" }[
                  item.kind
                ]
              }{" "}
              · {item.resource_id.slice(0, 8)}
            </span>
            <div className="workset-actions">
              <Button
                disabled={busy || i === 0}
                aria-label={`${i + 1}번 참조 위로`}
                onClick={() => {
                  const items = [...draft.items];
                  [items[i - 1], items[i]] = [items[i], items[i - 1]];
                  setDraft({ ...draft, items });
                }}
              >
                <ArrowUp size={16} />
              </Button>
              <Button
                disabled={busy || i === draft.items.length - 1}
                aria-label={`${i + 1}번 참조 아래로`}
                onClick={() => {
                  const items = [...draft.items];
                  [items[i + 1], items[i]] = [items[i], items[i + 1]];
                  setDraft({ ...draft, items });
                }}
              >
                <ArrowDown size={16} />
              </Button>
              <Button
                disabled={busy}
                aria-label={`${i + 1}번 참조 제거`}
                onClick={() =>
                  setDraft({
                    ...draft,
                    items: draft.items.filter((_, j) => i !== j),
                  })
                }
              >
                참조 제거
              </Button>
            </div>
          </li>
        ))}
      </ol>
      <div className="form-grid">
        <Field label="추가할 참조 종류">
          <select
            disabled={busy}
            value={kind}
            onChange={(e) => setKind(e.target.value as WorksetItem["kind"])}
          >
            <option value="document">문서</option>
            <option value="database">데이터베이스</option>
            <option value="task">할 일</option>
          </select>
        </Field>
        <Field label="현재 접근 가능한 참조">
          <select
            disabled={busy}
            value={kind === "task" ? taskID : resource}
            onChange={(e) => {
              if (kind === "task") {
                setTaskID(e.target.value);
                setResource(
                  sources.find((t) => t.task_id === e.target.value)
                    ?.document_id || "",
                );
              } else setResource(e.target.value);
            }}
          >
            <option value="">참조 선택</option>
            {(kind === "document"
              ? documents.filter(
                  (d) => d.workspace_id === workspace?.id && !d.deleted_at,
                )
              : sources
            ).map((item: any, i) => (
              <option value={kind === "task" ? item.task_id : item.id} key={i}>
                {item.title || item.name}
                {item.text ? ` · ${item.text.slice(0, 100)}` : ""}
              </option>
            ))}
          </select>
        </Field>
      </div>
      <Button
        disabled={busy || !resource}
        onClick={() => {
          setDraft({
            ...draft,
            items: [
              ...draft.items,
              {
                kind,
                resource_id: resource,
                context:
                  kind === "document"
                    ? { version: selected?.version || 1, mode: "read" }
                    : kind === "task"
                      ? { task_id: taskID, version: selected?.version || 1 }
                      : {},
              },
            ],
          });
          setResource("");
          setTaskID("");
        }}
      >
        참조 추가
      </Button>
      <p className="muted">
        특정 문서 위치나 DB 필터·보기는 해당 화면의 보관 버튼으로 추가할 수
        있습니다.
      </p>
      <div className="modal-actions">
        <Button disabled={busy} onClick={onClose}>
          취소
        </Button>
        <Button
          variant="primary"
          disabled={busy || !draft.name.trim()}
          onClick={() => void save()}
        >
          개인 묶음 저장
        </Button>
      </div>
    </Modal>
  );
}
