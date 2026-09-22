import { useEffect, useRef, useState } from "react";
import {
  Link,
  useLocation,
  useNavigate,
  useParams,
  useSearchParams,
} from "react-router-dom";
import {
  ArrowLeft,
  Blocks,
  Code2,
  Copy,
  Download,
  FileInput,
  LoaderCircle,
  LockKeyhole,
  Play,
  Plus,
  Puzzle,
  Settings2,
  ShieldCheck,
  Upload,
} from "lucide-react";
import { api, datetime } from "./api";
import { useApp } from "./context";
import { Button, ErrorBox, Field, Modal } from "./ui";
import {
  PluginFrame,
  type PluginFrameHandle,
  type PluginInfo,
  type Contribution,
} from "./PluginFrame";

export const pluginCapabilityLabels: Record<string, string> = {
  "document:read": "문서 조회",
  "document:write": "문서 작성·수정·삭제",
  "database:read": "데이터베이스 조회",
  "database:write": "데이터베이스 행 변경",
  "ai:execute": "관리자 설정 AI 사용",
  "storage:personal": "개인별 플러그인 설정 저장",
  "ui:notify": "서비스 알림 표시",
  "ui:navigate": "접근 가능한 내부 화면 이동",
  "file:import": "사용자가 선택한 문서 가져오기",
  "file:export": "사용자 확인 후 문서 내보내기",
};
const kinds: Record<string, string> = {
  blocks: "문서 블록",
  commands: "명령",
  sidebars: "사이드바",
  menus: "메뉴",
  importers: "가져오기",
  exporters: "내보내기",
  ai_providers: "AI 공급자 연결",
};
type AdminPlugin = PluginInfo & {
  enabled: boolean;
  created_at: string;
  updated_at: string;
  workspace_count: number;
};
function Empty({
  title,
  description,
}: {
  icon?: unknown;
  title: string;
  description: string;
}) {
  return (
    <div className="empty-state">
      <Puzzle size={32} />
      <h2>{title}</h2>
      <p>{description}</p>
    </div>
  );
}

export default function PluginsPage() {
  const location = useLocation();
  return (
    <>
      <PluginStyles />
      {location.pathname.startsWith("/admin") ? (
        <AdminPlugins />
      ) : (
        <WorkspacePlugins />
      )}
    </>
  );
}
function AdminPlugins() {
  const { notify } = useApp();
  const [plugins, setPlugins] = useState<AdminPlugin[]>([]),
    [error, setError] = useState(""),
    [loading, setLoading] = useState(true),
    [install, setInstall] = useState(false),
    [file, setFile] = useState<File | null>(null),
    [replacement, setReplacement] = useState(""),
    [busy, setBusy] = useState(false),
    [toggle, setToggle] = useState<AdminPlugin | null>(null);
  const load = async () => {
    try {
      const result = await api<{ plugins: AdminPlugin[] }>("/admin/plugins");
      setPlugins(result.plugins);
      setError("");
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  };
  useEffect(() => {
    void load();
  }, []);
  async function upload() {
    if (!file) return;
    setBusy(true);
    setError("");
    try {
      const form = new FormData();
      form.append("file", file);
      form.append("replace", replacement);
      const result = await api("/admin/plugins", "POST", form);
      setInstall(false);
      setFile(null);
      setReplacement("");
      notify(
        result.permissions_reset
          ? "업데이트되었습니다. 각 워크스페이스에서 권한을 다시 승인하세요."
          : "플러그인이 설치되었습니다. 워크스페이스에서 사용 권한을 승인하세요.",
      );
      await load();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="page">
      <div className="page-heading">
        <div>
          <span className="eyebrow">SERVICE EXTENSIONS</span>
          <h1>플러그인 관리</h1>
          <p>오프라인 확장을 설치하고 서비스 전체의 사용 여부를 관리합니다.</p>
        </div>
        <Button
          onClick={() => {
            setError("");
            setInstall(true);
          }}
        >
          <Plus size={18} />
          ZIP 설치
        </Button>
      </div>
      <div className="notice">
        <ShieldCheck size={20} />
        <div>
          플러그인 코드는 DOM 접근이 없는 격리 Worker에서 실행됩니다. 설치만으로
          문서 권한이 부여되지 않으며, 워크스페이스 관리자의 별도 승인이
          필요합니다.
        </div>
      </div>
      <ErrorBox error={!install && !toggle ? error : ""} />
      {loading ? (
        <div className="empty-state">
          <LoaderCircle className="spin" />
        </div>
      ) : !plugins.length ? (
        <Empty
          icon={Puzzle}
          title="설치된 플러그인이 없습니다"
          description="검토한 플러그인 ZIP을 업로드하세요. 인터넷 연결 없이 설치할 수 있습니다."
        />
      ) : (
        <div className="plugin-grid">
          {plugins.map((plugin) => (
            <article className="card plugin-card" key={plugin.id}>
              <div className="plugin-card-top">
                <div className="plugin-symbol">
                  <Puzzle />
                </div>
                <span className={`badge ${plugin.enabled ? "green" : ""}`}>
                  {plugin.enabled ? "서비스 활성" : "서비스 비활성"}
                </span>
              </div>
              <h2>{plugin.manifest.name}</h2>
              <p>
                {plugin.manifest.description || "설명이 없는 플러그인입니다."}
              </p>
              <div className="plugin-meta">
                v{plugin.manifest.version} ·{" "}
                {plugin.manifest.author || "제작자 미지정"}
              </div>
              <div className="plugin-meta">
                활성 워크스페이스 {plugin.workspace_count}개 ·{" "}
                {datetime(plugin.updated_at)}
              </div>
              <div className="plugin-capabilities">
                {plugin.manifest.capabilities.map((cap) => (
                  <span key={cap}>{pluginCapabilityLabels[cap] || cap}</span>
                ))}
              </div>
              <Button
                variant="secondary"
                onClick={() => {
                  setError("");
                  setToggle(plugin);
                }}
              >
                {plugin.enabled ? "전체 사용 중지" : "서비스에서 허용"}
              </Button>
            </article>
          ))}
        </div>
      )}
      <section className="card plugin-sdk-note">
        <Code2 size={24} />
        <div>
          <h2>madi Plugin SDK v1</h2>
          <p>
            manifest.json · main.js · 선택적 style.css와 로컬 에셋을 ZIP 루트에
            담습니다. 소스와 예제는 저장소의 sdk/plugins에서 제공합니다.
          </p>
          <p className="muted">
            registerBlock · registerCommand · registerSidebar · registerMenu ·
            registerImporter · registerExporter · registerAIProvider
          </p>
        </div>
      </section>
      <Modal
        open={install}
        onOpenChange={(v) => {
          if (!busy) setInstall(v);
        }}
        title="오프라인 플러그인 설치"
        description="설치 전 제작자와 코드를 검토하세요. 최대 ZIP 10MB / 압축 해제 20MB / 200개 파일을 지원합니다."
      >
        <ErrorBox error={error} />
        <Field label="플러그인 ZIP">
          <input
            type="file"
            accept=".zip,application/zip"
            onChange={(e) => setFile(e.target.files?.[0] || null)}
          />
        </Field>
        <Field
          label="기존 플러그인 업데이트 확인"
          hint="같은 ID를 교체하려면 REPLACE를 입력하세요. 모든 워크스페이스 승인 권한이 해제되며 재승인이 필요합니다."
        >
          <input
            value={replacement}
            onChange={(e) => setReplacement(e.target.value)}
            placeholder="신규 설치는 비워두세요"
          />
        </Field>
        <div className="modal-actions">
          <Button
            variant="secondary"
            disabled={busy}
            onClick={() => setInstall(false)}
          >
            취소
          </Button>
          <Button disabled={!file || busy} onClick={upload}>
            {busy ? (
              <LoaderCircle className="spin" size={17} />
            ) : (
              <Upload size={17} />
            )}
            검토한 ZIP 설치
          </Button>
        </div>
      </Modal>
      <Modal
        open={!!toggle}
        onOpenChange={(v) => {
          if (!v && !busy) setToggle(null);
        }}
        title={
          toggle?.enabled ? "서비스 전체 사용 중지" : "서비스에서 플러그인 허용"
        }
        description={
          toggle?.enabled
            ? "모든 워크스페이스의 새 요청이 즉시 거절되고 실행 중인 AI도 중단됩니다. 기존 플러그인 데이터는 보존됩니다."
            : "워크스페이스에서 승인한 사용자만 실행할 수 있습니다."
        }
      >
        <ErrorBox error={error} />
        <p>{toggle?.manifest.name}</p>
        <div className="modal-actions">
          <Button
            variant="secondary"
            disabled={busy}
            onClick={() => setToggle(null)}
          >
            취소
          </Button>
          <Button
            disabled={busy}
            onClick={async () => {
              if (!toggle) return;
              setBusy(true);
              try {
                await api(`/admin/plugins/${toggle.id}`, "PUT", {
                  enabled: !toggle.enabled,
                });
                setToggle(null);
                await load();
                notify("플러그인 사용 정책을 변경했습니다.");
              } catch (e: any) {
                setError(e.message);
              } finally {
                setBusy(false);
              }
            }}
          >
            확인
          </Button>
        </div>
      </Modal>
    </div>
  );
}
function WorkspacePlugins() {
  const { workspace } = useApp(),
    [params, setParams] = useSearchParams(),
    path = useParams()["*"] || "";
  const id = path.split("/")[0];
  if (!workspace)
    return (
      <div className="page">
        <Empty
          icon={Puzzle}
          title="워크스페이스를 선택하세요"
          description="플러그인 권한은 워크스페이스마다 분리됩니다."
        />
      </div>
    );
  if (id)
    return (
      <PluginDetail
        key={`${workspace.id}:${id}`}
        id={id}
        workspaceId={workspace.id}
      />
    );
  return (
    <PluginList
      key={workspace.id}
      workspaceId={workspace.id}
      canManage={["owner", "admin"].includes(workspace.role)}
      manage={params.get("manage") === "1"}
      onManage={(value) => setParams(value ? { manage: "1" } : {})}
    />
  );
}
function PluginList({
  workspaceId,
  canManage,
  manage,
  onManage,
}: {
  workspaceId: string;
  canManage: boolean;
  manage: boolean;
  onManage: (v: boolean) => void;
}) {
  const { notify } = useApp();
  const [plugins, setPlugins] = useState<PluginInfo[]>([]),
    [loading, setLoading] = useState(true),
    [error, setError] = useState(""),
    [grant, setGrant] = useState<PluginInfo | null>(null),
    [caps, setCaps] = useState<string[]>([]),
    [enabled, setEnabled] = useState(false),
    [busy, setBusy] = useState(false);
  const managing = canManage && manage;
  const loadGeneration = useRef(0),
    loadScope = useRef("");
  loadScope.current = `${workspaceId}:${managing}`;
  const load = async () => {
    const scope = `${workspaceId}:${managing}`;
    if (scope !== loadScope.current) return;
    const generation = ++loadGeneration.current;
    try {
      const list = await api<PluginInfo[]>(
        managing
          ? `/workspaces/${workspaceId}/plugins`
          : `/plugins?workspace_id=${workspaceId}`,
      );
      if (generation !== loadGeneration.current || scope !== loadScope.current)
        return;
      setPlugins(list);
      setError("");
    } catch (e: any) {
      if (generation === loadGeneration.current && scope === loadScope.current)
        setError(e.message);
    } finally {
      if (generation === loadGeneration.current && scope === loadScope.current)
        setLoading(false);
    }
  };
  useEffect(() => {
    setLoading(true);
    void load();
    return () => {
      loadGeneration.current++;
    };
  }, [workspaceId, managing]);
  return (
    <div className="page">
      <div className="page-heading">
        <div>
          <span className="eyebrow">YOUR WORKSPACE, EXTENDED</span>
          <h1>플러그인</h1>
          <p>
            {managing
              ? "이 워크스페이스에서 필요한 기능과 권한만 명시적으로 허용합니다."
              : "승인된 확장으로 문서와 지식 워크플로를 넓혀보세요."}
          </p>
        </div>
        {canManage && (
          <Button variant="secondary" onClick={() => onManage(!managing)}>
            <Settings2 size={18} />
            {managing ? "사용 가능한 확장" : "워크스페이스 권한"}
          </Button>
        )}
      </div>
      <ErrorBox error={!grant ? error : ""} />
      {managing && (
        <div className="notice">
          <LockKeyhole size={20} />
          <div>
            문서 내용은 현재 사용자가 접근할 수 있는 범위에서만 제공됩니다.
            관리자도 개인 문서 권한을 우회할 수 없습니다. 권한 취소는 다음 API
            요청부터 반영됩니다.
          </div>
        </div>
      )}
      {loading ? (
        <div className="empty-state">
          <LoaderCircle className="spin" />
        </div>
      ) : plugins.length === 0 ? (
        <Empty
          icon={Puzzle}
          title={
            managing
              ? "서비스에 설치된 확장이 없습니다"
              : "아직 사용 가능한 플러그인이 없습니다"
          }
          description={
            managing
              ? "서비스 관리자에게 오프라인 ZIP 설치를 요청하세요."
              : "워크스페이스 관리자가 설치된 플러그인의 권한을 승인하면 여기에 표시됩니다."
          }
        />
      ) : (
        <div className="plugin-grid">
          {plugins.map((plugin) => (
            <article className="card plugin-card" key={plugin.id}>
              <div className="plugin-card-top">
                <div className="plugin-symbol">
                  <Puzzle />
                </div>
                <span className="badge">v{plugin.manifest.version}</span>
              </div>
              <h2>{plugin.manifest.name}</h2>
              <p>{plugin.manifest.description}</p>
              <div className="plugin-meta">
                {plugin.manifest.author || "제작자 미지정"}
              </div>
              <div className="plugin-capabilities">
                {plugin.capabilities.map((cap) => (
                  <span key={cap}>{pluginCapabilityLabels[cap] || cap}</span>
                ))}
                {!plugin.capabilities.length && (
                  <span>데이터 접근 권한 없음</span>
                )}
              </div>
              {managing ? (
                <>
                  <div className="plugin-meta">
                    {plugin.global_enabled === false
                      ? "서비스 관리자가 사용을 중지했습니다"
                      : plugin.enabled
                        ? "워크스페이스 활성"
                        : "아직 승인하지 않음"}
                  </div>
                  <Button
                    variant="secondary"
                    onClick={() => {
                      setError("");
                      setGrant(plugin);
                      setCaps([...plugin.capabilities]);
                      setEnabled(!!plugin.enabled);
                    }}
                  >
                    <ShieldCheck size={17} />
                    권한 검토
                  </Button>
                </>
              ) : (
                <Link className="button" to={`/app/plugins/${plugin.id}`}>
                  <Play size={17} />
                  플러그인 열기
                </Link>
              )}
            </article>
          ))}
        </div>
      )}
      <Modal
        open={!!grant}
        onOpenChange={(v) => {
          if (!v && !busy) setGrant(null);
        }}
        title="워크스페이스 플러그인 권한"
        description="권한을 모두 선택할 필요는 없습니다. 실제 사용할 기능에 필요한 범위만 허용하세요."
      >
        <ErrorBox error={error} />
        <h3>{grant?.manifest.name}</h3>
        <Field label="사용 상태">
          <select
            value={enabled ? "enabled" : "disabled"}
            onChange={(e) => setEnabled(e.target.value === "enabled")}
          >
            <option value="disabled">사용 중지</option>
            <option value="enabled">이 워크스페이스에서 사용</option>
          </select>
        </Field>
        <fieldset className="field">
          <legend>승인할 권한</legend>
          <div className="plugin-grant-options">
            {grant?.manifest.capabilities.map((cap) => (
              <label key={cap}>
                <input
                  type="checkbox"
                  checked={caps.includes(cap)}
                  onChange={(e) =>
                    setCaps((old) =>
                      e.target.checked
                        ? [...old, cap]
                        : old.filter((v) => v !== cap),
                    )
                  }
                />
                <span>
                  {pluginCapabilityLabels[cap] || cap}
                  <small>{cap}</small>
                </span>
              </label>
            ))}
          </div>
        </fieldset>
        <div className="modal-actions">
          <Button
            variant="secondary"
            disabled={busy}
            onClick={() => setGrant(null)}
          >
            취소
          </Button>
          <Button
            disabled={busy || (enabled && grant?.global_enabled === false)}
            onClick={async () => {
              if (!grant) return;
              setBusy(true);
              try {
                await api(
                  `/workspaces/${workspaceId}/plugins/${grant.id}`,
                  "PUT",
                  { enabled, capabilities: caps },
                );
                setGrant(null);
                await load();
                notify("플러그인 권한을 저장했습니다.");
              } catch (e: any) {
                setError(e.message);
              } finally {
                setBusy(false);
              }
            }}
          >
            권한 저장
          </Button>
        </div>
      </Modal>
    </div>
  );
}
function PluginDetail({
  id,
  workspaceId,
}: {
  id: string;
  workspaceId: string;
}) {
  const { notify } = useApp(),
    navigate = useNavigate();
  const frame = useRef<PluginFrameHandle>(null),
    file = useRef<HTMLInputElement>(null);
  const [plugin, setPlugin] = useState<PluginInfo | null>(null),
    [error, setError] = useState(""),
    [registered, setRegistered] = useState<string[]>([]),
    [busy, setBusy] = useState(""),
    [result, setResult] = useState<any>(undefined),
    [importer, setImporter] = useState<Contribution | null>(null),
    [importFile, setImportFile] = useState<{
      name: string;
      content: string;
    } | null>(null),
    [exportFile, setExportFile] = useState<{
      name: string;
      content: string;
      id: string;
    } | null>(null),
    [block, setBlock] = useState<Contribution | null>(null),
    [ai, setAI] = useState<Contribution | null>(null),
    [prompt, setPrompt] = useState("");
  useEffect(() => {
    let active = true;
    api<PluginInfo[]>(`/plugins?workspace_id=${workspaceId}`)
      .then((items) => {
        if (active) {
          const item = items.find((p) => p.id === id);
          if (item) setPlugin(item);
          else
            setError("이 플러그인의 사용 권한이 없거나 사용이 중지되었습니다.");
        }
      })
      .catch((e) => {
        if (active) setError(e.message);
      });
    return () => {
      active = false;
    };
  }, [id, workspaceId]);
  async function invoke(kind: string, item: Contribution, input?: unknown) {
    setBusy(`${kind}:${item.id}`);
    setError("");
    try {
      if (kind === "importers") {
        const selected = input as { name?: string } | undefined;
        await api(`/plugins/${id}/bridge`, "POST", {
          workspace_id: workspaceId,
          operation: "file.import",
          args: { contribution_id: item.id, name: selected?.name },
        });
      }
      const value = await frame.current!.invoke(kind, item.id, input);
      if (kind === "exporters") {
        if (
          !value ||
          typeof value.name !== "string" ||
          typeof value.content !== "string" ||
          value.content.length > 1 << 20
        )
          throw new Error(
            "내보내기는 1MB 이하 {name,content} 텍스트를 반환해야 합니다",
          );
        await api(`/plugins/${id}/bridge`, "POST", {
          workspace_id: workspaceId,
          operation: "file.export",
          args: { contribution_id: item.id, name: value.name },
        });
        setExportFile({ ...value, id: item.id });
      } else if (value !== null && value !== undefined) setResult(value);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setBusy("");
    }
  }
  async function download() {
    if (!exportFile) return;
    try {
      await api(`/plugins/${id}/bridge`, "POST", {
        workspace_id: workspaceId,
        operation: "file.export",
        args: { contribution_id: exportFile.id, name: exportFile.name },
      });
      const url = URL.createObjectURL(
        new Blob([exportFile.content], { type: "text/plain;charset=utf-8" }),
      );
      const a = document.createElement("a");
      a.href = url;
      a.download = exportFile.name;
      a.click();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
      setExportFile(null);
    } catch (e: any) {
      setError(e.message);
    }
  }
  return (
    <div className="page">
      <Button variant="ghost" onClick={() => navigate("/app/plugins")}>
        <ArrowLeft size={17} />
        플러그인 목록
      </Button>
      <div className="page-heading">
        <div>
          <span className="eyebrow">SANDBOXED EXTENSION</span>
          <h1>{plugin?.manifest.name || "플러그인"}</h1>
          <p>{plugin?.manifest.description}</p>
        </div>
        {plugin && <span className="badge">v{plugin.manifest.version}</span>}
      </div>
      <ErrorBox error={error} />
      {plugin && (
        <>
          <div className="plugin-detail-grid">
            <section>
              <PluginFrame
                ref={frame}
                key={`${id}:${workspaceId}`}
                pluginId={id}
                workspaceId={workspaceId}
                onRegistered={(kind, id) =>
                  setRegistered((old) =>
                    old.includes(`${kind}:${id}`)
                      ? old
                      : [...old, `${kind}:${id}`],
                  )
                }
              />
            </section>
            <aside className="card plugin-contributions">
              <h2>확장 기능</h2>
              {Object.entries(plugin.manifest.contributions).map(
                ([kind, items]) =>
                  items.length > 0 && (
                    <div key={kind}>
                      <h3>{kinds[kind] || kind}</h3>
                      {items.map((item) => (
                        <button
                          key={item.id}
                          disabled={
                            !!busy || !registered.includes(`${kind}:${item.id}`)
                          }
                          title={item.description}
                          onClick={() => {
                            if (kind === "importers") {
                              setImporter(item);
                              file.current?.click();
                            } else if (kind === "blocks") setBlock(item);
                            else if (kind === "ai_providers") {
                              setAI(item);
                              setPrompt("");
                            } else void invoke(kind, item);
                          }}
                        >
                          {busy === `${kind}:${item.id}` ? (
                            <LoaderCircle size={16} className="spin" />
                          ) : kind === "blocks" ? (
                            <Blocks size={16} />
                          ) : kind === "importers" ? (
                            <FileInput size={16} />
                          ) : kind === "exporters" ? (
                            <Download size={16} />
                          ) : (
                            <Play size={16} />
                          )}
                          <span>{item.title}</span>
                        </button>
                      ))}
                    </div>
                  ),
              )}
              <p className="muted">
                확장 기능은 플러그인이 등록을 완료하면 활성화됩니다.
              </p>
            </aside>
          </div>
          <div className="plugin-capabilities plugin-detail-caps">
            <ShieldCheck size={17} />
            {plugin.capabilities.map((cap) => (
              <span key={cap}>{pluginCapabilityLabels[cap] || cap}</span>
            ))}
          </div>
        </>
      )}
      <input
        hidden
        ref={file}
        type="file"
        accept={importer?.extensions?.join(",") || ".md,.txt,.json,.csv"}
        onChange={async (e) => {
          const selected = e.target.files?.[0];
          e.target.value = "";
          if (!selected || !importer) return;
          try {
            if (selected.size > 1 << 20)
              throw new Error("가져올 문서는 1MB 이하여야 합니다");
            await api(`/plugins/${id}/bridge`, "POST", {
              workspace_id: workspaceId,
              operation: "file.import",
              args: { contribution_id: importer.id, name: selected.name },
            });
            setImportFile({
              name: selected.name,
              content: await selected.text(),
            });
          } catch (e: any) {
            setError(e.message);
          }
        }}
      />
      <Modal
        open={!!importFile}
        onOpenChange={(v) => {
          if (!v) setImportFile(null);
        }}
        title="플러그인으로 문서 가져오기"
        description="선택한 파일 내용이 이 플러그인에 제공됩니다. 승인된 문서 작성 권한이 있으면 현재 워크스페이스에 문서를 만들 수 있습니다."
      >
        <p>{importFile?.name}</p>
        <pre className="plugin-output">
          {importFile?.content.slice(0, 4000)}
        </pre>
        <div className="modal-actions">
          <Button variant="secondary" onClick={() => setImportFile(null)}>
            취소
          </Button>
          <Button
            onClick={async () => {
              const input = importFile;
              setImportFile(null);
              if (importer) await invoke("importers", importer, input);
            }}
          >
            가져오기 실행
          </Button>
        </div>
      </Modal>
      <Modal
        open={!!exportFile}
        onOpenChange={(v) => {
          if (!v) setExportFile(null);
        }}
        title="내보내기 결과 확인"
        description="내용을 검토한 뒤 다운로드하세요. 플러그인은 확인 없이 파일을 저장할 수 없습니다."
      >
        <ErrorBox error={error} />
        <p>{exportFile?.name}</p>
        <pre className="plugin-output">
          {exportFile?.content.slice(0, 4000)}
        </pre>
        <div className="modal-actions">
          <Button variant="secondary" onClick={() => setExportFile(null)}>
            취소
          </Button>
          <Button onClick={download}>
            <Download size={17} />
            다운로드
          </Button>
        </div>
      </Modal>
      <Modal
        open={!!block}
        onOpenChange={(v) => {
          if (!v) setBlock(null);
        }}
        title="플러그인 블록 삽입"
        description="아래 Markdown을 문서 소스 모드에 붙여넣으세요. 플러그인을 사용할 수 없을 때도 원본 데이터는 보존됩니다."
      >
        <pre className="plugin-output">{blockSource(id, block?.id || "")}</pre>
        <div className="modal-actions">
          <Button variant="secondary" onClick={() => setBlock(null)}>
            닫기
          </Button>
          <Button
            onClick={async () => {
              try {
                await navigator.clipboard.writeText(
                  blockSource(id, block?.id || ""),
                );
                notify("플러그인 블록 Markdown을 복사했습니다.");
              } catch {
                setError(
                  "클립보드 접근이 제한되었습니다. 코드를 직접 선택해 복사하세요.",
                );
              }
            }}
          >
            <Copy size={17} />
            Markdown 복사
          </Button>
        </div>
      </Modal>
      <Modal
        open={!!ai}
        onOpenChange={(v) => {
          if (!v && !busy) setAI(null);
        }}
        title="워크스페이스 AI 연결"
        description="관리자가 설정한 AI 공급자를 사용합니다. 플러그인에는 공급자 API 키가 전달되지 않습니다."
      >
        <Field label="질문">
          <textarea
            rows={5}
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            maxLength={32000}
          />
        </Field>
        <div className="modal-actions">
          <Button
            variant="secondary"
            disabled={!!busy}
            onClick={() => setAI(null)}
          >
            취소
          </Button>
          <Button
            disabled={!prompt.trim() || !!busy}
            onClick={async () => {
              if (!ai) return;
              const item = ai;
              setAI(null);
              await invoke("ai_providers", item, { prompt });
            }}
          >
            AI에 질문
          </Button>
        </div>
      </Modal>
      <Modal
        open={result !== undefined}
        onOpenChange={(v) => {
          if (!v) setResult(undefined);
        }}
        title="확장 기능 실행 결과"
      >
        <pre className="plugin-output">
          {typeof result === "string"
            ? result
            : JSON.stringify(result, null, 2)}
        </pre>
        <div className="modal-actions">
          <Button onClick={() => setResult(undefined)}>닫기</Button>
        </div>
      </Modal>
    </div>
  );
}
function blockSource(pluginID: string, blockID: string) {
  return (
    "```madi-plugin\n" +
    JSON.stringify(
      { plugin_id: pluginID, block_id: blockID, data: {} },
      null,
      2,
    ) +
    "\n```"
  );
}
function PluginStyles() {
  return (
    <style>{`.plugin-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:22px;margin:24px 0}.plugin-card{padding:26px;display:flex;flex-direction:column;align-items:flex-start;gap:12px}.plugin-card-top{display:flex;align-items:center;justify-content:space-between;width:100%;gap:12px}.plugin-symbol{width:48px;height:48px;border:1px solid #cee1d3;border-radius:12px;background:#eef5e9;color:#487c57;display:grid;place-items:center}.plugin-card h2{font-size:20px;margin:4px 0 0}.plugin-card>p{font-size:15px;line-height:1.8;margin:0;color:var(--muted)}.plugin-meta{font-size:13px;color:var(--muted)}.plugin-capabilities{display:flex;flex-wrap:wrap;gap:7px;align-items:center}.plugin-capabilities>span{font-size:12px;background:var(--soft);padding:4px 8px;border-radius:5px;color:var(--muted)}.plugin-card>.button{margin-top:auto}.plugin-sdk-note{display:flex;gap:18px;margin-top:24px;padding:25px}.plugin-sdk-note>svg{flex-shrink:0;color:var(--primary)}.plugin-sdk-note h2{font-size:19px;margin:0 0 8px}.plugin-sdk-note p{font-size:14px;line-height:1.8;margin:5px 0}.plugin-grant-options{display:flex;flex-direction:column;gap:13px;max-height:340px;overflow:auto}.plugin-grant-options>label{display:flex;align-items:flex-start;gap:10px;font-size:15px}.plugin-grant-options input{width:17px;height:17px;flex-shrink:0}.plugin-grant-options small{display:block;color:var(--muted);font-size:12px;margin-top:3px}.plugin-detail-grid{display:grid;grid-template-columns:minmax(0,1fr) 250px;gap:22px}.plugin-detail-grid>section{min-width:0}.plugin-contributions{padding:21px;align-self:start}.plugin-contributions h2{font-size:18px;margin:0 0 20px}.plugin-contributions h3{font-size:13px;color:var(--muted);margin:18px 0 9px}.plugin-contributions button{display:flex;align-items:center;gap:10px;width:100%;padding:11px 9px;text-align:left;border:0;background:transparent;border-radius:6px;color:var(--text);font-size:15px;cursor:pointer}.plugin-contributions button:hover:not(:disabled){background:var(--soft)}.plugin-contributions button:disabled{opacity:.4;cursor:default}.plugin-contributions p{font-size:12px;line-height:1.8;margin-top:20px}.plugin-detail-caps{margin:20px 0}.plugin-output{max-height:400px;overflow:auto;white-space:pre-wrap;overflow-wrap:anywhere;font-size:14px;padding:18px;background:var(--soft);border-radius:8px;line-height:1.7}@media(max-width:1150px){.plugin-grid{grid-template-columns:repeat(2,minmax(0,1fr))}.plugin-detail-grid{grid-template-columns:minmax(0,1fr) 220px}}@media(max-width:700px){.plugin-grid{grid-template-columns:1fr}.plugin-detail-grid{grid-template-columns:1fr}.plugin-contributions{display:flex;flex-wrap:wrap;gap:15px}.plugin-contributions>h2{width:100%;margin:0}.plugin-contributions>div{flex:1;min-width:120px}.plugin-card{padding:22px}.plugin-sdk-note{padding:20px}}`}</style>
  );
}
