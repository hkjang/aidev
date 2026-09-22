import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import {
  Building2,
  ChevronRight,
  FilePlus2,
  FolderPlus,
  FolderTree,
  LockKeyhole,
  Pencil,
  Plus,
  Settings,
  Trash2,
  Users,
} from "lucide-react";
import { api, datetime, type DocSummary } from "./api";
import { useApp } from "./context";
import {
  Badge,
  Button,
  Empty,
  ErrorBox,
  Field,
  Loading,
  Modal,
  PageHeading,
} from "./ui";
import "./spaces.css";

type Space = {
  id: string;
  name: string;
  slug: string;
  parent_id: string | null;
  visibility: string;
  classification: string;
  can_read: boolean;
  can_write: boolean;
  can_manage: boolean;
};
type Member = { id: string; name: string; email: string; role: string };
const roles: Record<string, string> = {
  owner: "소유자",
  admin: "관리자",
  editor: "작성자",
  commenter: "댓글 작성자",
  viewer: "조회자",
  member: "멤버",
};
const classes: Record<string, string> = {
  public: "공개",
  internal: "내부",
  confidential: "기밀",
  restricted: "최고 기밀",
};

export function SpacesPage() {
  const { workspace, user, createDocument, notify, reload } = useApp();
  const navigate = useNavigate();
  const [query, setQuery] = useSearchParams();
  const [spaces, setSpaces] = useState<Space[]>([]),
    [docs, setDocs] = useState<DocSummary[]>([]),
    [error, setError] = useState<unknown>(null),
    [loading, setLoading] = useState(true),
    [busy, setBusy] = useState(false);
  const [editing, setEditing] = useState<Partial<Space> | null>(null),
    [memberSpace, setMemberSpace] = useState<Space | null>(null),
    [members, setMembers] = useState<Member[]>([]),
    [workspaceMembers, setWorkspaceMembers] = useState<Member[]>([]),
    [selectedUser, setSelectedUser] = useState(""),
    [role, setRole] = useState("editor");
  const id = query.get("space") || "";
  const selected = spaces.find((s) => s.id === id);
  const manager =
    !!workspace &&
    ["owner", "admin"].includes(workspace.role) &&
    user.role !== "viewer";
  const generation = useRef(0);
  const load = useCallback(async () => {
    if (!workspace) return;
    const key = ++generation.current;
    setLoading(true);
    try {
      const v = await api<Space[]>(
        `/spaces?workspace_id=${workspace.id}${manager ? "&manage=1" : ""}`,
      );
      if (key === generation.current) {
        setSpaces(v);
        setError(null);
      }
    } catch (e) {
      if (key === generation.current) setError(e);
    } finally {
      if (key === generation.current) setLoading(false);
    }
  }, [workspace?.id, manager]);
  useEffect(() => {
    setDocs([]);
    setEditing(null);
    setMemberSpace(null);
    load();
    return () => {
      generation.current++;
    };
  }, [load]);
  useEffect(() => {
    let active = true;
    setDocs([]);
    if (selected?.can_read)
      api<DocSummary[]>(`/spaces/${id}/documents`)
        .then((v) => active && setDocs(v))
        .catch((e) => active && setError(e));
    return () => {
      active = false;
    };
  }, [id, selected?.can_read, spaces]);
  const ancestors = (space: Space) => {
    const names: string[] = [];
    let parent = space.parent_id;
    const seen = new Set<string>();
    while (parent && !seen.has(parent)) {
      seen.add(parent);
      const p = spaces.find((s) => s.id === parent);
      if (!p) break;
      names.unshift(p.name);
      parent = p.parent_id;
    }
    return names.join(" / ");
  };
  const save = async () => {
    if (!workspace || !editing) return;
    setBusy(true);
    setError(null);
    try {
      const value = await api<Space>(
        editing.id ? `/spaces/${editing.id}` : "/spaces",
        editing.id ? "PUT" : "POST",
        { ...editing, workspace_id: workspace.id },
      );
      setEditing(null);
      setQuery({ space: value.id });
      await load();
      notify("공간을 저장했습니다");
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  const openMembers = async (space: Space) => {
    setMemberSpace(space);
    setMembers([]);
    setSelectedUser("");
    setError(null);
    try {
      const [m, w] = await Promise.all([
        api<Member[]>(`/spaces/${space.id}/members`),
        api<Member[]>(`/workspaces/${workspace!.id}/members`),
      ]);
      setMembers(m);
      setWorkspaceMembers(w);
    } catch (e) {
      setError(e);
    }
  };
  const updateMember = async (uid: string, nextRole: string) => {
    if (!memberSpace) return;
    setBusy(true);
    try {
      await api(`/spaces/${memberSpace.id}/members`, "PUT", {
        user_id: uid,
        role: nextRole,
      });
      await openMembers(memberSpace);
      await load();
      notify("공간 권한을 변경했습니다");
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="page spaces-page">
      <PageHeading
        eyebrow="TEAM SPACES"
        title="공간"
        description="팀과 주제별로 지식을 모으고, 상위 공간의 권한을 안전하게 이어받습니다."
        actions={
          manager ? (
            <Button
              onClick={() =>
                setEditing({
                  name: "",
                  slug: "",
                  parent_id: id || null,
                  visibility: "workspace",
                  classification: "internal",
                })
              }
            >
              <FolderPlus size={18} />새 공간
            </Button>
          ) : undefined
        }
      />
      <ErrorBox error={error} />
      {loading ? (
        <Loading />
      ) : (
        <div className="spaces-layout">
          <section className="panel spaces-list" aria-label="공간 목록">
            {spaces.length ? (
              spaces.map((space) => (
                <button
                  key={space.id}
                  className={`space-list-item ${space.id === id ? "active" : ""}`}
                  onClick={() => setQuery({ space: space.id })}
                >
                  <FolderTree size={20} />
                  <span>
                    <strong>{space.name}</strong>
                    <small>{ancestors(space) || "워크스페이스"}</small>
                  </span>
                  {space.visibility === "restricted" && (
                    <LockKeyhole size={16} />
                  )}
                  <ChevronRight size={16} />
                </button>
              ))
            ) : (
              <Empty
                title="첫 공간을 만들어 보세요"
                text="문서를 팀별 공간으로 나누어 관리할 수 있습니다."
              />
            )}
          </section>
          <section className="panel space-content">
            {selected ? (
              <>
                <div className="space-heading">
                  <div>
                    <h2>{selected.name}</h2>
                    <p className="muted">
                      {ancestors(selected) || workspace?.name} / {selected.slug}
                    </p>
                    <div className="space-badges">
                      <Badge>
                        {selected.visibility === "restricted"
                          ? "선택 멤버 공간"
                          : "워크스페이스 공간"}
                      </Badge>
                      <Badge>{classes[selected.classification]}</Badge>
                    </div>
                  </div>
                  <div className="space-actions">
                    {selected.can_write && (
                      <Button
                        onClick={async () => {
                          const d = await createDocument("새 공간 문서", "", {
                            space_id: selected.id,
                          });
                          if (d) navigate(`/app/documents/${d.id}`);
                        }}
                      >
                        <FilePlus2 size={17} />
                        문서 작성
                      </Button>
                    )}
                    {manager && (
                      <>
                        <Button
                          variant="secondary"
                          onClick={() => setEditing({ ...selected })}
                        >
                          <Pencil size={16} />
                          공간 설정
                        </Button>
                        <Button
                          variant="secondary"
                          onClick={() => openMembers(selected)}
                        >
                          <Users size={16} />
                          멤버 권한
                        </Button>
                      </>
                    )}
                  </div>
                </div>
                {!selected.can_read ? (
                  <div className="notice">
                    관리자로서 공간 설정만 볼 수 있습니다. 내용 열람은 공간 멤버
                    권한이 필요합니다.
                  </div>
                ) : docs.length ? (
                  <div className="recent-list">
                    {docs.map((doc) => (
                      <Link
                        className="recent-item"
                        key={doc.id}
                        to={`/app/documents/${doc.id}`}
                      >
                        <FolderTree size={20} />
                        <div>
                          <strong>{doc.title}</strong>
                          <span>{doc.tags.join(" · ") || "태그 없음"}</span>
                        </div>
                        <time>{datetime(doc.updated_at)}</time>
                        <ChevronRight size={17} />
                      </Link>
                    ))}
                  </div>
                ) : (
                  <Empty
                    title="공간에 문서가 없습니다"
                    text="새 문서를 작성하거나 기존 문서를 이 공간으로 이동하세요."
                  />
                )}
                {manager && (
                  <div className="space-danger">
                    <Button
                      variant="ghost"
                      onClick={async () => {
                        if (
                          !confirm(
                            `‘${selected.name}’ 공간을 삭제할까요? 빈 공간만 삭제할 수 있습니다.`,
                          )
                        )
                          return;
                        try {
                          await api(`/spaces/${selected.id}`, "DELETE");
                          setQuery({});
                          await load();
                          await reload();
                          notify("빈 공간을 삭제했습니다");
                        } catch (e) {
                          setError(e);
                        }
                      }}
                    >
                      <Trash2 size={16} />빈 공간 삭제
                    </Button>
                  </div>
                )}
              </>
            ) : (
              <Empty
                title="공간을 선택하세요"
                text="왼쪽 공간을 선택하면 하위 공간의 문서도 함께 표시됩니다."
              />
            )}
          </section>
        </div>
      )}
      <Modal
        open={!!editing}
        onOpenChange={(v) => !v && setEditing(null)}
        title={editing?.id ? "공간 설정" : "새 공간"}
        description="제한된 상위 공간의 권한은 하위 공간에도 적용됩니다."
      >
        {editing && (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              save();
            }}
          >
            <ErrorBox error={error} />
            <Field label="공간 이름">
              <input
                required
                maxLength={200}
                value={editing.name || ""}
                onChange={(e) =>
                  setEditing({ ...editing, name: e.target.value })
                }
              />
            </Field>
            <Field
              label="주소 이름"
              hint="영문 소문자·숫자·하이픈. 비워두면 자동 생성합니다."
            >
              <input
                value={editing.slug || ""}
                pattern="[a-z0-9][a-z0-9\-]{0,79}"
                onChange={(e) =>
                  setEditing({ ...editing, slug: e.target.value })
                }
              />
            </Field>
            <Field label="상위 공간">
              <select
                value={editing.parent_id || ""}
                onChange={(e) =>
                  setEditing({ ...editing, parent_id: e.target.value || null })
                }
              >
                <option value="">워크스페이스 바로 아래</option>
                {spaces
                  .filter((s) => s.id !== editing.id && s.can_write)
                  .map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.name}
                    </option>
                  ))}
              </select>
            </Field>
            <div className="form-grid">
              <Field label="공개 범위">
                <select
                  value={editing.visibility}
                  onChange={(e) =>
                    setEditing({ ...editing, visibility: e.target.value })
                  }
                >
                  <option value="workspace">워크스페이스 멤버</option>
                  <option value="restricted">선택한 공간 멤버</option>
                </select>
              </Field>
              <Field label="문서 기본 등급">
                <select
                  value={editing.classification}
                  onChange={(e) =>
                    setEditing({ ...editing, classification: e.target.value })
                  }
                >
                  {Object.entries(classes).map(([v, n]) => (
                    <option value={v} key={v}>
                      {n}
                    </option>
                  ))}
                </select>
              </Field>
            </div>
            <div className="modal-actions">
              <Button
                type="button"
                variant="secondary"
                onClick={() => setEditing(null)}
              >
                취소
              </Button>
              <Button disabled={busy}>{busy ? "저장 중…" : "공간 저장"}</Button>
            </div>
          </form>
        )}
      </Modal>
      <Modal
        open={!!memberSpace}
        onOpenChange={(v) => !v && setMemberSpace(null)}
        title={`${memberSpace?.name || "공간"} 멤버 권한`}
        description="상위 공간과 워크스페이스 권한보다 높은 권한을 부여하지 않습니다. 조회자는 댓글과 수정이 제한됩니다."
      >
        <ErrorBox error={error} />
        <div className="form-grid">
          <Field label="워크스페이스 멤버">
            <select
              value={selectedUser}
              onChange={(e) => setSelectedUser(e.target.value)}
            >
              <option value="">사용자 선택</option>
              {workspaceMembers.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.name} ({m.email})
                </option>
              ))}
            </select>
          </Field>
          <Field label="공간 권한">
            <select value={role} onChange={(e) => setRole(e.target.value)}>
              {["admin", "editor", "commenter", "viewer"].map((v) => (
                <option key={v} value={v}>
                  {roles[v]}
                </option>
              ))}
            </select>
          </Field>
        </div>
        <Button
          disabled={!selectedUser || busy}
          onClick={() => updateMember(selectedUser, role)}
        >
          <Plus size={16} />
          권한 부여 / 변경
        </Button>
        <div className="space-member-list">
          {members.map((m) => (
            <div key={m.id}>
              <span>
                <strong>{m.name}</strong>
                <small>{m.email}</small>
              </span>
              <Badge>{roles[m.role]}</Badge>
              <Button
                variant="ghost"
                disabled={busy}
                aria-label={`${m.name} 공간 권한 제거`}
                onClick={() => {
                  if (confirm(`${m.name}님의 공간 권한을 제거할까요?`))
                    updateMember(m.id, "remove");
                }}
              >
                <Trash2 size={16} />
              </Button>
            </div>
          ))}
        </div>
      </Modal>
    </div>
  );
}

export function WorkspaceSettingsPage() {
  const { workspace, user, notify } = useApp();
  const [value, setValue] = useState<Record<string, any>>({}),
    [version, setVersion] = useState(0),
    [history, setHistory] = useState<Record<string, any>[]>([]),
    [error, setError] = useState<unknown>(null),
    [loading, setLoading] = useState(true),
    [busy, setBusy] = useState(false),
    [aiOverride, setAIOverride] = useState(false),
    [loadedScope, setLoadedScope] = useState("");
  const scope = `${user.id}:${workspace?.id || ""}`;
  const activeScope = useRef(scope),
    requestGeneration = useRef(0),
    scopeGeneration = useRef(0),
    mutationPending = useRef(false);
  activeScope.current = scope;
  const mutationGuard = () => {
    const generation = scopeGeneration.current;
    return () =>
      activeScope.current === scope && generation === scopeGeneration.current;
  };
  const allowed =
    !!workspace &&
    ["owner", "admin"].includes(workspace.role) &&
    user.role !== "viewer";
  const path = `/workspaces/${workspace?.id}/settings`;
  const load = useCallback(async () => {
    if (!allowed || activeScope.current !== scope) return false;
    const generation = ++requestGeneration.current;
    const current = () =>
      activeScope.current === scope && generation === requestGeneration.current;
    setLoading(true);
    setLoadedScope("");
    try {
      const [v, h] = await Promise.all([
        api(path),
        api<Record<string, any>[]>(path + "/history"),
      ]);
      if (!current()) return false;
      setValue(v.data);
      setVersion(v.version);
      setHistory(h);
      setAIOverride(v.data.ai_enabled !== undefined);
      setError(null);
      setLoadedScope(scope);
      return true;
    } catch (e) {
      if (current()) setError(e);
      return false;
    } finally {
      if (current()) setLoading(false);
    }
  }, [path, allowed, scope]);
  useEffect(() => {
    scopeGeneration.current++;
    setValue({});
    setHistory([]);
    setError(null);
    setBusy(false);
    setAIOverride(false);
    setLoadedScope("");
    mutationPending.current = false;
    load();
    return () => {
      scopeGeneration.current++;
      requestGeneration.current++;
    };
  }, [load]);
  const change = (key: string, v: any) =>
    setValue((old) => ({ ...old, [key]: v }));
  if (!allowed)
    return (
      <div className="page">
        <Empty
          title="워크스페이스 관리자 설정"
          text="현재 워크스페이스 소유자 또는 관리자로 접근하세요."
        />
      </div>
    );
  return (
    <div className="page">
      <PageHeading
        eyebrow="WORKSPACE SETTINGS"
        title="워크스페이스 설정"
        description="서비스 관리와 분리된 팀 설정입니다. 개별 값을 비우면 서비스 기본 설정을 사용합니다."
      />
      <ErrorBox error={error} />
      {loading ? (
        <Loading />
      ) : loadedScope !== scope ? (
        <Button onClick={() => load()}>설정 다시 불러오기</Button>
      ) : (
        <form
          onSubmit={async (e) => {
            e.preventDefault();
            if (loadedScope !== scope || mutationPending.current) return;
            const current = mutationGuard();
            mutationPending.current = true;
            setBusy(true);
            setError(null);
            try {
              const data: Record<string, any> = {
                lifecycle_enabled: !!value.lifecycle_enabled,
                review_period_days: value.review_period_days
                  ? Number(value.review_period_days)
                  : null,
              };
              for (const key of [
                "ai_enabled",
                "ai_base_url",
                "ai_api_key",
                "ai_model",
                "ai_max_tokens",
                "ai_system_prompt",
              ])
                data[key] = aiOverride
                  ? key === "ai_max_tokens"
                    ? Number(value[key] || 4096)
                    : (value[key] ?? (key === "ai_enabled" ? false : ""))
                  : null;
              await api(path, "PUT", { version, data });
              if (!current()) return;
              if ((await load()) && current())
                notify("워크스페이스 설정을 저장했습니다");
            } catch (e) {
              if (current()) setError(e);
            } finally {
              if (current()) {
                mutationPending.current = false;
                setBusy(false);
              }
            }
          }}
        >
          <fieldset
            disabled={busy}
            style={{ border: 0, padding: 0, margin: 0, minWidth: 0 }}
          >
            <div className="settings-layout">
              <aside className="panel padded settings-overview">
                <Building2 size={30} />
                <h2>{workspace?.name}</h2>
                <p>
                  서비스의 인증·보안 정책은 변경하지 않습니다. 이 워크스페이스의
                  AI 공급자와 지식 정책을 독립적으로 관리합니다.
                </p>
                <div className="notice">
                  AI는 사용자에게 허용된 문서만 참조합니다. API 키는 암호화해
                  저장하며 다시 표시하지 않습니다.
                </div>
              </aside>
              <div className="settings-form">
                <section className="panel padded">
                  <h2>지식 정책</h2>
                  <p>
                    <Link to="/app/workspace-operations?tab=branding">
                      로고·파비콘·팀 브랜딩
                    </Link>
                    과{" "}
                    <Link to="/app/workspace-operations?tab=features">
                      기능별 사용 정책
                    </Link>
                    은 워크스페이스 운영 설정에서 관리합니다.
                  </p>
                  <div className="form-grid">
                    <Field label="문서 검토 주기 (일)">
                      <input
                        type="number"
                        min={1}
                        max={3650}
                        value={value.review_period_days || ""}
                        placeholder="서비스 기본 정책"
                        onChange={(e) =>
                          change("review_period_days", e.target.value)
                        }
                      />
                    </Field>
                  </div>
                  <Field
                    label="문서 생명주기 자동화"
                    hint="켜면 매시간 게시 문서의 검토 주기를 확인하고 기한이 지난 문서를 '검토 기한 경과' 상태로 전환하여 담당자에게 알립니다. 승인 절차를 활성화하지는 않습니다."
                  >
                    <select
                      value={value.lifecycle_enabled ? "on" : "off"}
                      onChange={(e) =>
                        change("lifecycle_enabled", e.target.value === "on")
                      }
                    >
                      <option value="off">사용하지 않음 (기본)</option>
                      <option value="on">검토 주기 자동 점검</option>
                    </select>
                  </Field>
                </section>
                <section className="panel padded">
                  <h2>팀 AI 공급자</h2>
                  <label className="check-label">
                    <input
                      type="checkbox"
                      checked={aiOverride}
                      onChange={(e) => setAIOverride(e.target.checked)}
                    />
                    이 워크스페이스에서 별도 AI 설정 사용
                  </label>
                  {aiOverride && (
                    <>
                      <label className="check-label">
                        <input
                          type="checkbox"
                          checked={!!value.ai_enabled}
                          onChange={(e) =>
                            change("ai_enabled", e.target.checked)
                          }
                        />
                        AI 활성화
                      </label>
                      <Field label="OpenAI 호환 API 주소">
                        <input
                          type="url"
                          value={value.ai_base_url || ""}
                          required={!!value.ai_enabled}
                          placeholder="http://local-llm:8000/v1"
                          onChange={(e) =>
                            change("ai_base_url", e.target.value)
                          }
                        />
                      </Field>
                      <Field label="모델 이름">
                        <input
                          value={value.ai_model || ""}
                          required={!!value.ai_enabled}
                          onChange={(e) => change("ai_model", e.target.value)}
                        />
                      </Field>
                      <Field
                        label="AI API 키"
                        hint={
                          value.ai_api_key_configured
                            ? "설정된 키가 있습니다. 비워두면 기존 키를 유지합니다."
                            : "인증이 필요 없는 사내 공급자는 비워둘 수 있습니다."
                        }
                      >
                        <input
                          type="password"
                          autoComplete="new-password"
                          value={value.ai_api_key || ""}
                          onChange={(e) => change("ai_api_key", e.target.value)}
                        />
                      </Field>
                      <Field
                        label="최대 출력 토큰"
                        hint="1~262144. 실제 공급자·모델의 출력 한도 내에서 설정하세요."
                      >
                        <input
                          type="number"
                          min={1}
                          max={262144}
                          value={value.ai_max_tokens || 4096}
                          onChange={(e) =>
                            change("ai_max_tokens", e.target.value)
                          }
                        />
                      </Field>
                      <Field label="시스템 지시문">
                        <textarea
                          rows={4}
                          value={value.ai_system_prompt || ""}
                          onChange={(e) =>
                            change("ai_system_prompt", e.target.value)
                          }
                        />
                      </Field>
                    </>
                  )}
                </section>
                <div className="settings-save">
                  <span>설정 버전 {version}</span>
                  <Button disabled={busy}>
                    {busy ? "저장 중…" : "설정 저장"}
                  </Button>
                </div>
                <section className="panel padded">
                  <h2>설정 변경 이력</h2>
                  {history.length ? (
                    history.map((h) => (
                      <div className="space-history-row" key={h.id}>
                        <span>
                          버전 {h.version} · {datetime(h.created_at)}
                        </span>
                        <Button
                          type="button"
                          variant="secondary"
                          disabled={busy}
                          onClick={async () => {
                            if (
                              loadedScope !== scope ||
                              mutationPending.current
                            )
                              return;
                            if (
                              !confirm(
                                `설정을 버전 ${h.version}으로 복원할까요?`,
                              )
                            )
                              return;
                            const current = mutationGuard();
                            mutationPending.current = true;
                            setBusy(true);
                            try {
                              await api(
                                path + `/history/${h.id}/restore`,
                                "POST",
                                { version },
                              );
                              if (!current()) return;
                              if ((await load()) && current())
                                notify("설정을 복원했습니다");
                            } catch (e) {
                              if (current()) setError(e);
                            } finally {
                              if (current()) {
                                mutationPending.current = false;
                                setBusy(false);
                              }
                            }
                          }}
                        >
                          복원
                        </Button>
                      </div>
                    ))
                  ) : (
                    <p className="muted">
                      저장 후 이전 설정이 이력에 남습니다.
                    </p>
                  )}
                </section>
              </div>
            </div>
          </fieldset>
        </form>
      )}
    </div>
  );
}

export function OrganizationsPage() {
  const { workspace, notify } = useApp();
  const [items, setItems] = useState<Record<string, any>[]>([]),
    [members, setMembers] = useState<Member[]>([]),
    [query, setQuery] = useSearchParams(),
    [error, setError] = useState<unknown>(null),
    [newName, setNewName] = useState(""),
    [email, setEmail] = useState(""),
    [role, setRole] = useState("member"),
    [busy, setBusy] = useState(false);
  const id = query.get("organization") || "",
    selected = items.find((i) => i.id === id),
    manage = selected && ["owner", "admin"].includes(selected.role);
  const load = async () => {
    try {
      setItems(await api("/organizations"));
    } catch (e) {
      setError(e);
    }
  };
  useEffect(() => {
    load();
  }, []);
  useEffect(() => {
    let active = true;
    setMembers([]);
    if (manage)
      api<Member[]>(`/organizations/${id}/members`)
        .then((v) => active && setMembers(v))
        .catch((e) => active && setError(e));
    return () => {
      active = false;
    };
  }, [id, manage]);
  const update = async (uid: string, next: string) => {
    setBusy(true);
    try {
      await api(`/organizations/${id}/members`, "PUT", {
        user_id: uid,
        email,
        role: next,
      });
      setMembers(await api(`/organizations/${id}/members`));
      setEmail("");
      notify("조직 멤버를 변경했습니다");
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="page">
      <PageHeading
        eyebrow="ORGANIZATIONS"
        title="조직"
        description="조직과 워크스페이스를 연결합니다. 조직 멤버십은 문서 접근 권한을 자동으로 부여하지 않습니다."
      />
      <ErrorBox error={error} />
      <div className="panel padded">
        <form
          className="space-inline-form"
          onSubmit={async (e) => {
            e.preventDefault();
            setBusy(true);
            try {
              const o = await api("/organizations", "POST", { name: newName });
              setNewName("");
              await load();
              setQuery({ organization: o.id });
              notify("조직을 만들었습니다");
            } catch (e) {
              setError(e);
            } finally {
              setBusy(false);
            }
          }}
        >
          <Field label="새 조직 이름">
            <input
              required
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
            />
          </Field>
          <Button disabled={busy}>
            <Plus size={16} />
            조직 만들기
          </Button>
        </form>
      </div>
      <div className="spaces-layout">
        <section className="panel spaces-list">
          {items.map((o) => (
            <button
              className={`space-list-item ${id === o.id ? "active" : ""}`}
              key={o.id}
              onClick={() => setQuery({ organization: o.id })}
            >
              <Building2 size={20} />
              <span>
                <strong>{o.name}</strong>
                <small>{roles[o.role]}</small>
              </span>
              <ChevronRight size={16} />
            </button>
          ))}
        </section>
        <section className="panel padded">
          {selected ? (
            <>
              <h2>{selected.name}</h2>
              <p className="muted">조직 주소: {selected.slug}</p>
              {manage && (
                <>
                  <Button
                    variant="secondary"
                    disabled={
                      busy ||
                      !workspace ||
                      !["owner", "admin"].includes(workspace.role)
                    }
                    onClick={async () => {
                      if (
                        !workspace ||
                        !confirm(
                          `현재 워크스페이스를 ${selected.name} 조직에 연결할까요?`,
                        )
                      )
                        return;
                      try {
                        await api(
                          `/workspaces/${workspace.id}/organization`,
                          "PUT",
                          { organization_id: id },
                        );
                        notify("워크스페이스를 조직에 연결했습니다");
                      } catch (e) {
                        setError(e);
                      }
                    }}
                  >
                    <Settings size={16} />
                    현재 워크스페이스 연결
                  </Button>
                  <form
                    className="space-inline-form"
                    onSubmit={(e) => {
                      e.preventDefault();
                      update("", role);
                    }}
                  >
                    <Field label="등록된 사용자 이메일">
                      <input
                        type="email"
                        required
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                      />
                    </Field>
                    <Field label="조직 역할">
                      <select
                        value={role}
                        onChange={(e) => setRole(e.target.value)}
                      >
                        <option value="member">멤버</option>
                        <option value="admin">관리자</option>
                      </select>
                    </Field>
                    <Button disabled={busy}>
                      <Plus size={16} />
                      멤버 추가
                    </Button>
                  </form>
                  <div className="space-member-list">
                    {members.map((m) => (
                      <div key={m.id}>
                        <span>
                          <strong>{m.name}</strong>
                          <small>{m.email}</small>
                        </span>
                        <Badge>{roles[m.role]}</Badge>
                        {m.role !== "owner" && (
                          <Button
                            variant="ghost"
                            aria-label={`${m.name} 조직에서 제거`}
                            onClick={() => {
                              if (confirm(`${m.name}님을 조직에서 제거할까요?`))
                                update(m.id, "remove");
                            }}
                          >
                            <Trash2 size={16} />
                          </Button>
                        )}
                      </div>
                    ))}
                  </div>
                </>
              )}
            </>
          ) : (
            <Empty
              title="조직을 선택하세요"
              text="여러 워크스페이스를 조직 아래에서 관리할 수 있습니다."
            />
          )}
        </section>
      </div>
    </div>
  );
}
