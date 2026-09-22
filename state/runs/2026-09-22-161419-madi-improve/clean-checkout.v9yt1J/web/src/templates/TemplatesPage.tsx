import {
  lazy,
  Suspense,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import {
  ArrowRight,
  BookOpen,
  Copy,
  Eye,
  FilePlus2,
  History,
  LockKeyhole,
  Pencil,
  Plus,
  RefreshCw,
  RotateCcw,
  Search,
  Trash2,
  X,
} from "lucide-react";
import { api, type Doc } from "../api";
import { useApp } from "../context";
import {
  Badge,
  Button,
  DocIcon,
  Empty,
  ErrorBox,
  Field,
  Loading,
  Modal,
  PageHeading,
} from "../ui";
import TemplateForm from "./TemplateForm";
import {
  blankTemplate,
  kinds,
  scopes,
  type Template,
  type TemplateSpace,
} from "./model";
import "./style.css";
const MarkdownContent = lazy(() =>
  import("../editor/MarkdownContent").then((m) => ({
    default: m.MarkdownContent,
  })),
);
type HistoryEntry = {
  version: number;
  created_at: string;
  user_name: string;
  data: Template;
};

export default function TemplatesPage() {
  const { user, workspace } = useApp();
  return <Library key={`${user.id}:${workspace?.id || ""}`} />;
}
function Library() {
  const { workspace, user, documents, reload, notify } = useApp();
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const q = params.get("q") || "",
    category = params.get("category") || "",
    scope = Object.hasOwn(scopes, params.get("scope") || "")
      ? params.get("scope")!
      : "",
    selectedID = params.get("template") || "";
  const source = ["all", "builtin", "custom", "trash"].includes(
    params.get("source") || "",
  )
    ? params.get("source")!
    : "all";
  const offset = Math.max(
    0,
    Math.min(100000, Math.floor(Number(params.get("offset")) || 0)),
  );
  const [items, setItems] = useState<Template[]>([]),
    [builtins, setBuiltins] = useState<Template[]>([]),
    [more, setMore] = useState(false),
    [spaces, setSpaces] = useState<TemplateSpace[]>([]);
  const [loading, setLoading] = useState(true),
    [selected, setSelected] = useState<Template | null>(null),
    [previewLoading, setPreviewLoading] = useState(false),
    [error, setError] = useState<unknown>(null),
    [previewError, setPreviewError] = useState<unknown>(null);
  const [form, setForm] = useState<Template | null>(null),
    [busy, setBusy] = useState(false),
    [actionError, setActionError] = useState<unknown>(null);
  const [create, setCreate] = useState(false),
    [createSource, setCreateSource] = useState<{
      id: string;
      version: number;
    } | null>(null),
    [title, setTitle] = useState(""),
    [destination, setDestination] = useState(""),
    [parent, setParent] = useState(""),
    [visibility, setVisibility] = useState("private");
  const [remove, setRemove] = useState(false),
    [history, setHistory] = useState<HistoryEntry[] | null>(null),
    [restoreVersion, setRestoreVersion] = useState<number | null>(null);
  const active = useRef(true),
    listGeneration = useRef(0),
    previewGeneration = useRef(0);
  const canCreate =
    user.role !== "viewer" &&
    !!workspace &&
    ["owner", "admin", "editor"].includes(workspace.role);
  const set = (key: string, value: string) => {
    const next = new URLSearchParams(window.location.search);
    if (value) next.set(key, value);
    else next.delete(key);
    if (["q", "category", "scope", "source"].includes(key))
      next.delete("offset");
    setParams(next, { replace: key === "q" });
  };
  const load = useCallback(
    async (quiet = false) => {
      if (!workspace) {
        setLoading(false);
        return;
      }
      const request = ++listGeneration.current;
      if (!quiet) setLoading(true);
      try {
        const search = new URLSearchParams({
          workspace_id: workspace.id,
          q,
          category,
          scope,
          trash: source === "trash" ? "1" : "0",
          limit: "30",
          offset: String(offset),
        });
        const [list, presets] = await Promise.all([
          api<{ items: Template[]; has_more: boolean }>(`/templates?${search}`),
          api<Template[]>(`/templates/builtins?workspace_id=${workspace.id}`),
        ]);
        if (!active.current || request !== listGeneration.current) return;
        setItems(list.items);
        setMore(list.has_more);
        setBuiltins((previous) =>
          JSON.stringify(previous) === JSON.stringify(presets)
            ? previous
            : presets,
        );
        setError(null);
      } catch (e) {
        if (active.current && request === listGeneration.current) {
          setItems([]);
          setBuiltins([]);
          setError(e);
        }
      } finally {
        if (active.current && request === listGeneration.current)
          setLoading(false);
      }
    },
    [workspace?.id, q, category, scope, source, offset],
  );
  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
      listGeneration.current++;
      previewGeneration.current++;
    };
  }, []);
  useEffect(() => {
    void load();
  }, [load]);
  useEffect(() => {
    if (!workspace) return;
    api<TemplateSpace[]>(`/spaces?workspace_id=${workspace.id}`)
      .then((data) => {
        if (active.current) setSpaces(data);
      })
      .catch(() => {
        if (active.current) setSpaces([]);
      });
  }, [workspace?.id]);
  const loadPreview = useCallback(
    async (quiet = false) => {
      const request = ++previewGeneration.current;
      if (!selectedID) {
        setSelected(null);
        setPreviewError(null);
        return;
      }
      const preset = builtins.find((t) => t.id === selectedID);
      if (preset) {
        setSelected(preset);
        setPreviewError(null);
        setPreviewLoading(false);
        return;
      }
      if (selectedID.startsWith("builtin-")) {
        setSelected(null);
        return;
      }
      if (!quiet) {
        setSelected(null);
        setPreviewLoading(true);
      }
      try {
        const value = await api<Template>(
          `/templates/${encodeURIComponent(selectedID)}`,
        );
        if (value.workspace_id !== workspace?.id)
          throw new Error("현재 워크스페이스의 템플릿이 아닙니다.");
        if (active.current && request === previewGeneration.current) {
          setSelected(value);
          setPreviewError(null);
        }
      } catch (e) {
        if (active.current && request === previewGeneration.current) {
          setSelected(null);
          setPreviewError(e);
        }
      } finally {
        if (active.current && request === previewGeneration.current)
          setPreviewLoading(false);
      }
    },
    [selectedID, builtins, workspace?.id],
  );
  useEffect(() => {
    setActionError(null);
    setCreate(false);
    setRemove(false);
    setHistory(null);
    void loadPreview();
  }, [loadPreview]);
  useEffect(() => {
    const check = () => {
      if (document.visibilityState === "visible" && !busy) {
        void load(true);
        void loadPreview(true);
      }
    };
    const timer = setInterval(check, 30000);
    window.addEventListener("focus", check);
    return () => {
      clearInterval(timer);
      window.removeEventListener("focus", check);
    };
  }, [load, loadPreview, busy]);
  const changed = async (value?: Template) => {
    if (value) {
      setSelected(value);
      set("template", value.id);
    }
    await load(true);
  };
  const operate = async (action: () => Promise<void>) => {
    if (busy) return;
    setBusy(true);
    setActionError(null);
    try {
      await action();
    } catch (e) {
      if (active.current) setActionError(e);
    } finally {
      if (active.current) setBusy(false);
    }
  };
  const visibleBuiltins =
    (source === "all" || source === "builtin") && !scope
      ? builtins.filter(
          (t) =>
            (!category || t.category === category) &&
            (!q ||
              `${t.name} ${t.description} ${t.tags.join(" ")}`
                .toLocaleLowerCase("ko")
                .includes(q.toLocaleLowerCase("ko"))),
        )
      : [];
  const categories = [
    ...new Set([...builtins, ...items].map((t) => t.category).filter(Boolean)),
  ].sort((a, b) => a.localeCompare(b, "ko"));
  if (category && !categories.includes(category)) categories.push(category);
  const shownItems = source === "builtin" ? [] : items;
  const date = (value?: string) =>
    value
      ? new Intl.DateTimeFormat("ko-KR", {
          dateStyle: "medium",
          timeStyle: "short",
          timeZone:
            typeof user.preferences.timezone === "string"
              ? user.preferences.timezone
              : "Asia/Seoul",
        }).format(new Date(value))
      : "";
  const card = (t: Template) => (
    <article
      className={`library-card ${selectedID === t.id ? "selected" : ""}`}
      key={t.id}
    >
      <div className="library-card-top">
        <span className="library-icon">
          <DocIcon icon={t.icon} size={26} />
        </span>
        <Badge>
          {t.builtin ? "기본 제공" : scopes[t.visibility || "private"]}
        </Badge>
      </div>
      <button
        className="library-card-name"
        onClick={() => set("template", t.id)}
        aria-label={`${t.name} 미리보기`}
      >
        <h3>{t.name}</h3>
      </button>
      <p>{t.description || "설명이 없는 사용자 템플릿입니다."}</p>
      <div className="library-tags">
        <Badge>{t.category || "미분류"}</Badge>
        {t.tags.slice(0, 3).map((tag) => (
          <span key={tag}>#{tag}</span>
        ))}
      </div>
      <footer>
        <small>
          {t.builtin
            ? kinds[t.kind]
            : `${t.owner_name || "소유자"} · v${t.version}`}
        </small>
        <Button variant="secondary" onClick={() => set("template", t.id)}>
          <Eye size={16} />
          미리보기
        </Button>
      </footer>
    </article>
  );
  return (
    <>
      <PageHeading
        eyebrow="A GOOD PLACE TO BEGIN"
        title="템플릿 라이브러리"
        description="좋은 시작을 팀의 자산으로. 기본 틀을 활용하거나, 우리만의 템플릿을 저장하고 함께 사용하세요."
        actions={
          <Button
            onClick={() => {
              setActionError(null);
              setForm({ ...blankTemplate });
            }}
            disabled={!canCreate}
          >
            <Plus size={17} />새 템플릿
          </Button>
        }
      />
      <div className="panel padded template-filters">
        <Field label="템플릿 검색">
          <div className="search-field">
            <Search size={17} />
            <input
              type="search"
              placeholder="이름, 설명, 태그 검색"
              value={q}
              onChange={(e) => set("q", e.target.value)}
            />
          </div>
        </Field>
        <Field label="템플릿 모음">
          <select
            value={source}
            onChange={(e) => set("source", e.target.value)}
          >
            <option value="all">전체 라이브러리</option>
            <option value="builtin">기본 제공</option>
            <option value="custom">사용자 템플릿</option>
            <option value="trash">템플릿 휴지통</option>
          </select>
        </Field>
        <Field label="템플릿 분류 필터">
          <select
            value={category}
            onChange={(e) => set("category", e.target.value)}
          >
            <option value="">모든 분류</option>
            {categories.map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
        </Field>
        <Field label="템플릿 공유 필터">
          <select value={scope} onChange={(e) => set("scope", e.target.value)}>
            <option value="">모든 공개 범위</option>
            {Object.entries(scopes).map(([key, label]) => (
              <option key={key} value={key}>
                {label}
              </option>
            ))}
          </select>
        </Field>
        <Button
          variant="secondary"
          onClick={() => {
            void load();
            void loadPreview();
          }}
          disabled={loading}
        >
          <RefreshCw size={17} />
          새로고침
        </Button>
      </div>
      <ErrorBox error={error} />
      {!canCreate && (
        <p className="notice">
          <LockKeyhole size={16} />
          현재 역할은 템플릿을 읽을 수 있습니다. 저장·복제·문서 생성에는
          워크스페이스 작성 권한이 필요합니다.
        </p>
      )}
      <div
        className={`template-library-layout ${selectedID ? "with-preview" : ""}`}
      >
        <div className="template-gallery">
          {loading ? (
            <Loading />
          ) : (
            <>
              {(source === "all" || source === "builtin") && (
                <section>
                  <div className="template-section-heading">
                    <h2>
                      <BookOpen size={21} />
                      기본 제공 템플릿
                    </h2>
                    <span>읽기 전용 · 내 템플릿으로 복제 가능</span>
                  </div>
                  {visibleBuiltins.length ? (
                    <div className="library-grid">
                      {visibleBuiltins.map(card)}
                    </div>
                  ) : (
                    <p className="muted">
                      선택한 조건에 맞는 기본 템플릿이 없습니다.
                    </p>
                  )}
                </section>
              )}
              {source !== "builtin" && (
                <section>
                  <div className="template-section-heading">
                    <h2>
                      {source === "trash" ? (
                        <Trash2 size={21} />
                      ) : (
                        <FilePlus2 size={21} />
                      )}{" "}
                      {source === "trash" ? "템플릿 휴지통" : "사용자 템플릿"}
                    </h2>
                    <span>현재 권한으로 열람 가능한 저장본</span>
                  </div>
                  {shownItems.length ? (
                    <div className="library-grid">{shownItems.map(card)}</div>
                  ) : (
                    <Empty
                      title={
                        source === "trash"
                          ? "휴지통이 비어 있어요"
                          : "아직 저장된 템플릿이 없어요"
                      }
                      text="새 템플릿을 작성하거나 기본 템플릿을 내 라이브러리로 복제해 보세요."
                    />
                  )}
                  {(offset > 0 || more) && (
                    <div className="template-pagination">
                      <Button
                        variant="secondary"
                        disabled={offset === 0}
                        onClick={() =>
                          set("offset", String(Math.max(0, offset - 30)))
                        }
                      >
                        이전
                      </Button>
                      <span>
                        {offset + 1}–{offset + items.length}개 표시
                      </span>
                      <Button
                        variant="secondary"
                        disabled={!more}
                        onClick={() => set("offset", String(offset + 30))}
                      >
                        다음
                      </Button>
                    </div>
                  )}
                </section>
              )}
            </>
          )}
        </div>
        {selectedID && (
          <aside className="panel template-detail" aria-label="선택한 템플릿">
            <header>
              <strong>템플릿 미리보기</strong>
              <Button
                variant="ghost"
                aria-label="템플릿 미리보기 닫기"
                onClick={() => set("template", "")}
              >
                <X size={18} />
              </Button>
            </header>
            <ErrorBox error={previewError} />
            {previewLoading ? (
              <Loading />
            ) : selected ? (
              <>
                <div className="template-detail-heading">
                  <span className="library-icon">
                    <DocIcon icon={selected.icon} size={28} />
                  </span>
                  <Badge>
                    {selected.builtin ? "기본 제공" : `v${selected.version}`}
                  </Badge>
                  <h2>{selected.name}</h2>
                  <p>{selected.description}</p>
                  <span className="muted">
                    {kinds[selected.kind]} ·{" "}
                    {selected.builtin
                      ? "madi 기본 템플릿"
                      : `${scopes[selected.visibility || "private"]} · ${selected.owner_name}`}
                  </span>
                  {selected.updated_at && (
                    <small className="muted">
                      마지막 저장 {date(selected.updated_at)}
                    </small>
                  )}
                </div>
                <div className="template-detail-actions">
                  {selected.deleted_at ? (
                    <>
                      <p className="notice">
                        휴지통의 템플릿입니다. 복원한 뒤 사용할 수 있습니다.
                      </p>
                      <Button
                        disabled={!selected.can_manage || busy}
                        onClick={() =>
                          void operate(async () => {
                            const value = await api<Template>(
                              `/templates/${selected.id}/restore`,
                              "POST",
                              { version: selected.version },
                            );
                            if (active.current) {
                              await changed(value);
                              notify("템플릿을 복원했습니다.");
                            }
                          })
                        }
                      >
                        <RotateCcw size={16} />
                        템플릿 복원
                      </Button>
                    </>
                  ) : (
                    <>
                      <Button
                        disabled={!canCreate || busy}
                        onClick={() => {
                          setTitle(selected.name);
                          setCreateSource({
                            id: selected.id,
                            version: selected.version,
                          });
                          setDestination(
                            selected.space_id &&
                              spaces.some(
                                (s) =>
                                  s.id === selected.space_id && s.can_write,
                              )
                              ? selected.space_id
                              : "",
                          );
                          setParent("");
                          setVisibility("private");
                          setActionError(null);
                          setCreate(true);
                        }}
                      >
                        <FilePlus2 size={16} />이 템플릿으로 문서 만들기
                      </Button>
                      <Button
                        variant="secondary"
                        disabled={!canCreate || busy}
                        onClick={() =>
                          void operate(async () => {
                            const value = await api<Template>(
                              `/templates/${selected.id}/duplicate`,
                              "POST",
                              {
                                workspace_id: workspace?.id,
                                expected_version: selected.version,
                              },
                            );
                            if (active.current) {
                              await changed(value);
                              notify("나만 보는 템플릿 사본을 저장했습니다.");
                            }
                          })
                        }
                      >
                        <Copy size={16} />내 템플릿으로 복제
                      </Button>
                      {selected.can_write && (
                        <Button
                          variant="secondary"
                          onClick={() => setForm({ ...selected })}
                          disabled={busy}
                        >
                          <Pencil size={16} />
                          템플릿 수정
                        </Button>
                      )}
                    </>
                  )}
                  {selected.can_manage && (
                    <>
                      <Button
                        variant="secondary"
                        disabled={busy}
                        onClick={() =>
                          void operate(async () => {
                            const versions = await api<HistoryEntry[]>(
                              `/templates/${selected.id}/versions`,
                            );
                            if (active.current) setHistory(versions);
                          })
                        }
                      >
                        <History size={16} />
                        변경 이력
                      </Button>
                      {!selected.deleted_at && (
                        <Button
                          variant="ghost"
                          disabled={busy}
                          onClick={() => setRemove(true)}
                        >
                          <Trash2 size={16} />
                          휴지통으로 이동
                        </Button>
                      )}
                    </>
                  )}
                </div>
                <ErrorBox error={actionError} />
                <div className="template-markdown-preview">
                  <Suspense fallback={<Loading />}>
                    <MarkdownContent
                      markdown={selected.markdown || ""}
                      documents={documents}
                    />
                  </Suspense>
                </div>
                <p className="muted template-hint">
                  날짜 변수는 문서를 만들 때 치환합니다. 위키 링크는 같은
                  워크스페이스의 문서와 연결됩니다.
                </p>
              </>
            ) : null}
          </aside>
        )}
      </div>
      {form && (
        <TemplateForm
          key={form.id || "new"}
          initial={form}
          spaces={spaces}
          close={() => setForm(null)}
          saved={(value) => {
            setForm(null);
            void changed(value);
          }}
        />
      )}
      <Modal
        open={create}
        onOpenChange={(open) => !busy && setCreate(open)}
        title="템플릿으로 새 문서 만들기"
        description="현재 저장된 템플릿 버전을 사용합니다. 문서는 초안으로 생성하며, 게시·승인 정책은 일반 문서와 같습니다."
      >
        <form
          onSubmit={(event) => {
            event.preventDefault();
            if (!createSource) return;
            void operate(async () => {
              const value = await api<Doc>(
                `/templates/${createSource.id}/documents`,
                "POST",
                {
                  workspace_id: workspace?.id,
                  expected_version: createSource.version,
                  title,
                  space_id: destination,
                  parent_id: parent,
                  visibility,
                },
              );
              if (active.current) {
                await reload();
                if (active.current)
                  navigate(`/app/documents/${value.id}?mode=preview`);
              }
            });
          }}
        >
          <ErrorBox error={actionError} />
          <Field label="새 문서 제목">
            <input
              required
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              disabled={busy}
            />
          </Field>
          <Field label="새 문서 공개 범위">
            <select
              value={visibility}
              onChange={(e) => setVisibility(e.target.value)}
              disabled={busy}
            >
              <option value="private">나만 보기</option>
              <option value="workspace">워크스페이스 공유</option>
            </select>
          </Field>
          <Field label="새 문서 공간">
            <select
              value={destination}
              onChange={(e) => {
                setDestination(e.target.value);
                setParent("");
              }}
              disabled={busy}
            >
              <option value="">공간 없음</option>
              {spaces
                .filter((s) => s.can_write)
                .map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.name}
                  </option>
                ))}
            </select>
          </Field>
          <Field
            label="새 문서 상위 문서"
            hint="상위 문서의 접근 권한도 함께 적용됩니다."
          >
            <select
              value={parent}
              onChange={(e) => setParent(e.target.value)}
              disabled={busy}
            >
              <option value="">최상위 문서</option>
              {documents
                .filter((d) => d.can_write !== false && !d.deleted_at)
                .map((d) => (
                  <option key={d.id} value={d.id}>
                    {d.title}
                  </option>
                ))}
            </select>
          </Field>
          {visibility !== "private" && (
            <p className="notice">
              템플릿의 내용을 워크스페이스에 공유합니다. 공간·상위 문서에 제한이
              있으면 해당 권한을 함께 적용합니다.
            </p>
          )}
          <div className="modal-actions">
            <Button
              type="button"
              variant="secondary"
              disabled={busy}
              onClick={() => setCreate(false)}
            >
              취소
            </Button>
            <Button type="submit" disabled={busy}>
              {busy ? "만드는 중…" : "문서 만들기"}
              <ArrowRight size={16} />
            </Button>
          </div>
        </form>
      </Modal>
      <Modal
        open={remove}
        onOpenChange={(open) => !busy && setRemove(open)}
        title="템플릿을 휴지통으로 옮길까요?"
        description="이 템플릿으로 이미 만든 문서는 변경하지 않습니다. 템플릿과 변경 이력은 복원할 수 있습니다."
      >
        <ErrorBox error={actionError} />
        <div className="modal-actions">
          <Button
            variant="secondary"
            disabled={busy}
            onClick={() => setRemove(false)}
          >
            취소
          </Button>
          <Button
            variant="danger"
            disabled={busy}
            onClick={() =>
              selected &&
              void operate(async () => {
                await api(
                  `/templates/${selected.id}?version=${selected.version}`,
                  "DELETE",
                );
                if (active.current) {
                  setRemove(false);
                  await loadPreview(true);
                  await load(true);
                  notify("템플릿을 휴지통으로 옮겼습니다. 복원할 수 있습니다.");
                }
              })
            }
          >
            휴지통으로 이동
          </Button>
        </div>
      </Modal>
      <Modal
        open={history !== null}
        onOpenChange={(open) => !open && setHistory(null)}
        title="템플릿 변경 이력"
        description="최근 100개 버전입니다. 복원은 당시 내용과 공개 범위를 새 버전으로 저장하며, 현재 권한과 민감정보 정책을 다시 적용합니다."
      >
        <ErrorBox error={actionError} />
        <div className="template-history">
          {history?.map((entry) => (
            <div key={entry.version}>
              <div>
                <strong>
                  v{entry.version} · {entry.data.name}
                </strong>
                <span>
                  {date(entry.created_at)} · {entry.user_name}
                </span>
                <small>
                  {scopes[entry.data.visibility || "private"]} ·{" "}
                  {entry.data.deleted_at ? "휴지통 이동" : "저장본"}
                </small>
              </div>
              <Button
                variant="secondary"
                disabled={busy || entry.version === selected?.version}
                onClick={() => setRestoreVersion(entry.version)}
              >
                이 버전 복원
              </Button>
            </div>
          ))}
        </div>
      </Modal>
      <Modal
        open={restoreVersion !== null}
        onOpenChange={(open) => !busy && !open && setRestoreVersion(null)}
        title={`v${restoreVersion || ""} 버전으로 복원할까요?`}
        description="선택한 시점의 이름·내용·태그·공개 범위·공간을 다시 적용합니다. 이후 버전은 이력에 남습니다."
      >
        <ErrorBox error={actionError} />
        <div className="modal-actions">
          <Button
            variant="secondary"
            disabled={busy}
            onClick={() => setRestoreVersion(null)}
          >
            취소
          </Button>
          <Button
            disabled={busy}
            onClick={() =>
              selected &&
              void operate(async () => {
                const value = await api<Template>(
                  `/templates/${selected.id}/versions/${restoreVersion}/restore`,
                  "POST",
                  { version: selected.version },
                );
                if (active.current) {
                  setRestoreVersion(null);
                  setHistory(null);
                  await changed(value);
                  notify("선택한 템플릿 버전을 복원했습니다.");
                }
              })
            }
          >
            선택 버전 복원
          </Button>
        </div>
      </Modal>
      <p className="muted template-footnote">
        매일 한 장씩 이어 쓰는 노트는 <Link to="/app">홈의 오늘 노트</Link> 또는
        명령 팔레트에서 사용할 수 있습니다.
      </p>
    </>
  );
}
