import "./channel-settings.css";
import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import {
  Cable,
  Eye,
  History,
  Play,
  Plus,
  RefreshCw,
  Save,
  Settings2,
  ShieldCheck,
} from "lucide-react";
import { api, datetime } from "./api";
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
type Row = Record<string, any>;
const providers = [
  ["github", "GitHub 문서"],
  ["gitlab", "GitLab Wiki"],
  ["jira", "Jira 이슈"],
  ["confluence", "Confluence 문서"],
  ["drive", "Google Drive 문서"],
  ["sharepoint", "SharePoint 페이지"],
  ["rest", "REST JSON API"],
];
const defaults: Record<string, string> = {
  github: "https://api.github.com",
  gitlab: "https://gitlab.example.internal/api/v4",
  jira: "https://example.atlassian.net/rest/api/3",
  confluence: "https://example.atlassian.net/wiki/api/v2",
  drive: "https://www.googleapis.com/drive/v3",
  sharepoint: "https://graph.microsoft.com/v1.0",
  rest: "https://api.example.internal/v1",
};
const providerFields: Record<string, string[][]> = {
  github: [
    ["repository", "저장소 (소유자/저장소)", "company/knowledge"],
    ["ref", "브랜치·태그·트리 SHA", "HEAD"],
    ["path_prefix", "문서 경로 접두사", "docs/"],
  ],
  gitlab: [["project_id", "프로젝트 ID 또는 경로", "group/project"]],
  jira: [["jql", "JQL 검색 범위", "project = OPS ORDER BY updated DESC"]],
  confluence: [["remote_space", "공간 ID (Cloud) 또는 키 (Server)", ""]],
  drive: [["query", "Drive 검색 조건", "'folder-id' in parents"]],
  sharepoint: [
    [
      "site_id",
      "SharePoint 사이트 ID",
      "tenant.sharepoint.com,site-uuid,web-uuid",
    ],
  ],
  rest: [
    ["list_path", "목록 경로", "documents"],
    ["items_path", "항목 배열 JSON 경로", "data.items"],
    ["id_path", "원격 고유 ID 경로", "id"],
    ["title_path", "제목 경로", "title"],
    ["content_path", "Markdown 본문 경로", "content"],
    ["url_path", "출처 URL 경로 (선택)", "url"],
    ["next_path", "다음 페이지 URL 경로 (선택)", "links.next"],
  ],
};
const stateNames: Record<string, string> = {
  pending: "대기",
  running: "실행 중",
  succeeded: "완료",
  failed: "실패",
  cancelled: "취소",
};
function cleanDraft(row: Row, wid: string) {
  return {
    ...row,
    workspace_id: wid,
    credentials: { token: "", username: "", password: "" },
    config: {
      allow_http: false,
      insecure_tls: false,
      acknowledge_acl: false,
      auth_mode: "bearer",
      conflict_policy: "preserve_local",
      max_items: 1000,
      ...(row.config || {}),
    },
  };
}
export function ConnectorPage() {
  const { workspace, notify } = useApp();
  const [params, setParams] = useSearchParams();
  const [rows, setRows] = useState<Row[]>([]),
    [spaces, setSpaces] = useState<Row[]>([]),
    [accounts, setAccounts] = useState<Row[]>([]),
    [draft, setDraft] = useState<Row | null>(null),
    [detail, setDetail] = useState<{ runs: Row[]; records: Row[] }>({
      runs: [],
      records: [],
    }),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [loading, setLoading] = useState(true),
    [rewind, setRewind] = useState(false);
  const generation = useRef(0);
  const activeWorkspace = useRef(workspace?.id);
  activeWorkspace.current = workspace?.id;
  const manager = ["owner", "admin"].includes(workspace?.role || "");
  const selected = rows.find((x) => x.id === params.get("connector"));
  const load = useCallback(async () => {
    if (!workspace?.id || !manager) {
      setLoading(false);
      return;
    }
    const run = ++generation.current;
    try {
      const [r, s, a] = await Promise.all([
        api<Row[]>(`/connectors?workspace_id=${workspace.id}`),
        api<Row[]>(`/spaces?workspace_id=${workspace.id}`),
        api<Row[]>(`/connectors/options?workspace_id=${workspace.id}`),
      ]);
      if (run !== generation.current) return;
      setRows(r);
      setSpaces(s.filter((x) => x.can_write));
      setAccounts(a);
      setError("");
    } catch (e) {
      if (run === generation.current) setError((e as Error).message);
    } finally {
      if (run === generation.current) setLoading(false);
    }
  }, [workspace?.id, manager]);
  useEffect(() => {
    setRows([]);
    setDraft(null);
    setDetail({ runs: [], records: [] });
    setLoading(true);
    void load();
    return () => {
      generation.current++;
    };
  }, [load]);
  useEffect(() => {
    if (!selected) {
      setDetail({ runs: [], records: [] });
      return;
    }
    let active = true;
    const fetch = async () => {
      try {
        const [runs, records] = await Promise.all([
          api<Row[]>(`/connectors/${selected.id}/runs`),
          api<Row[]>(`/connectors/${selected.id}/records`),
        ]);
        if (active) setDetail({ runs, records });
      } catch (e) {
        if (active) setError((e as Error).message);
      }
    };
    void fetch();
    const timer = setInterval(() => void fetch(), 4000);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [selected?.id]);
  const start = async (id: string, preview: boolean) => {
    const scope = workspace?.id;
    setBusy(true);
    setError("");
    try {
      const result = await api<Row>(
        `/connectors/${id}/${preview ? "preview" : "run"}`,
        "POST",
        { rewind },
      );
      if (scope !== activeWorkspace.current) return;
      notify(
        preview
          ? "미리보기 작업을 등록했습니다"
          : "가져오기 작업을 등록했습니다",
      );
      setParams({ connector: id });
      setDetail({
        runs: [
          {
            job_id: result.job_id,
            status: "pending",
            kind: preview ? "connector.preview" : "connector.sync",
            report: {},
          },
        ],
        records: [],
      });
      await load();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  const config = (key: string, value: any) =>
    setDraft((old) =>
      old ? { ...old, config: { ...old.config, [key]: value } } : old,
    );
  return (
    <>
      <PageHeading
        eyebrow="READ ONLY · PERMISSION AWARE"
        title="외부 커넥터"
        description="사내 시스템의 지식을 읽어 와서 공간에 모읍니다. 원본 시스템에는 쓰기 요청을 보내지 않습니다."
        actions={
          <>
            <Link className="button" to="/app/migrations">
              파일 가져오기
            </Link>
            <Button onClick={() => void load()}>
              <RefreshCw size={17} />
              새로고침
            </Button>
            {manager && (
              <Button
                variant="primary"
                onClick={() =>
                  setDraft(
                    cleanDraft(
                      {
                        name: "",
                        kind: "github",
                        base_url: defaults.github,
                        space_id: "",
                        service_account_id: accounts[0]?.id || "",
                        enabled: false,
                        interval_minutes: 0,
                      },
                      workspace!.id,
                    ),
                  )
                }
              >
                <Plus size={17} />
                연결 추가
              </Button>
            )}
          </>
        }
      />
      <ErrorBox error={error} />
      {!manager ? (
        <Empty
          title="워크스페이스 관리자 전용"
          text="외부 연결은 워크스페이스 소유자 또는 관리자가 설정합니다."
        />
      ) : loading ? (
        <Loading />
      ) : (
        <>
          <div className="notice">
            <ShieldCheck size={18} />
            <div>
              서비스 관리자가 <Link to="/admin/connectors">외부 연결 정책</Link>
              에서 사용과 정확한 호스트를 허용해야 합니다. 전용 서비스 계정과
              실행 관리자의 현재 권한을 매 실행 시 함께 확인합니다.
            </div>
          </div>
          <div className="card">
            <div className="card-header">
              <div>
                <h2>등록된 연결</h2>
                <p>
                  미리보기는 문서를 저장하지 않습니다. 가져온 문서는 초안으로
                  생성됩니다.
                </p>
              </div>
              <Badge>{rows.length}개</Badge>
            </div>
            {!rows.length ? (
              <Empty
                title="지식이 있는 곳을 연결하세요"
                text="GitHub·GitLab·Jira·Confluence·Drive·SharePoint 또는 사내 REST API를 연결할 수 있습니다."
              />
            ) : (
              <div className="table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th>연결</th>
                      <th>대상 공간</th>
                      <th>예약</th>
                      <th>상태</th>
                      <th>관리</th>
                    </tr>
                  </thead>
                  <tbody>
                    {rows.map((row) => (
                      <tr key={row.id}>
                        <td>
                          <strong>{row.name}</strong>
                          <div className="muted">
                            {providers.find((x) => x[0] === row.kind)?.[1]}
                          </div>
                        </td>
                        <td>
                          {spaces.find((x) => x.id === row.space_id)?.name ||
                            "워크스페이스 루트"}
                        </td>
                        <td>
                          {row.interval_minutes
                            ? `${row.interval_minutes}분마다`
                            : "수동 실행"}
                        </td>
                        <td>
                          <Badge tone={row.enabled ? "success" : "neutral"}>
                            {row.enabled ? "활성" : "중지"}
                          </Badge>
                        </td>
                        <td>
                          <div className="button-row">
                            <Button
                              onClick={() =>
                                setDraft(cleanDraft(row, workspace!.id))
                              }
                            >
                              <Settings2 size={16} />
                              편집
                            </Button>
                            <Button
                              disabled={busy}
                              onClick={() => void start(row.id, true)}
                            >
                              <Eye size={16} />
                              미리보기
                            </Button>
                            <Button
                              disabled={busy || !row.enabled}
                              onClick={() => void start(row.id, false)}
                            >
                              <Play size={16} />
                              가져오기
                            </Button>
                            <Button
                              onClick={() => setParams({ connector: row.id })}
                            >
                              <History size={16} />
                              이력
                            </Button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
          <label className="check-row">
            <input
              type="checkbox"
              checked={rewind}
              onChange={(e) => setRewind(e.target.checked)}
            />
            다음 수동 가져오기는 처음부터 다시 조회 (원격 ID로 중복 방지)
          </label>
          <p className="muted">
            기본적으로 사용자가 수정한 문서는 덮어쓰지 않고 충돌로 보고합니다.
            원격에서 삭제된 자료가 있어도 madi 문서를 자동 삭제하지 않습니다.
          </p>
        </>
      )}
      {draft && (
        <Modal
          open
          title={draft.id ? "외부 연결 편집" : "외부 연결 추가"}
          onOpenChange={() => {
            if (!busy) setDraft(null);
          }}
          wide
        >
          <form
            onSubmit={async (e) => {
              e.preventDefault();
              setBusy(true);
              setError("");
              try {
                await api(
                  `/connectors${draft.id ? `/${draft.id}` : ""}`,
                  draft.id ? "PUT" : "POST",
                  draft,
                );
                notify("연결을 저장했습니다");
                setDraft(null);
                await load();
              } catch (e) {
                setError((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            <ErrorBox error={error} />
            <div className="form-grid">
              <Field label="연결 이름">
                <input
                  required
                  maxLength={120}
                  value={draft.name}
                  onChange={(e) => setDraft({ ...draft, name: e.target.value })}
                />
              </Field>
              <Field label="연결 유형">
                <select
                  disabled={!!draft.id}
                  value={draft.kind}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      kind: e.target.value,
                      base_url: defaults[e.target.value],
                      config: {
                        allow_http: false,
                        insecure_tls: false,
                        acknowledge_acl: false,
                        auth_mode: "bearer",
                        conflict_policy: "preserve_local",
                        max_items: 1000,
                      },
                    })
                  }
                >
                  {providers.map(([v, l]) => (
                    <option value={v} key={v}>
                      {l}
                    </option>
                  ))}
                </select>
              </Field>
              <Field label="API 기본 URL">
                <input
                  required
                  type="url"
                  disabled={!!draft.id}
                  value={draft.base_url}
                  onChange={(e) =>
                    setDraft({ ...draft, base_url: e.target.value })
                  }
                />
              </Field>
              <Field label="대상 공간">
                <select
                  disabled={!!draft.id}
                  value={draft.space_id || ""}
                  onChange={(e) =>
                    setDraft({ ...draft, space_id: e.target.value })
                  }
                >
                  <option value="">워크스페이스 루트</option>
                  {spaces.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.name}
                    </option>
                  ))}
                </select>
              </Field>
              <Field label="전용 서비스 계정">
                <select
                  required
                  disabled={!!draft.id}
                  value={draft.service_account_id}
                  onChange={(e) =>
                    setDraft({ ...draft, service_account_id: e.target.value })
                  }
                >
                  <option value="">선택하세요</option>
                  {accounts.map((a) => (
                    <option key={a.id} value={a.id}>
                      {a.name} ({a.email})
                    </option>
                  ))}
                </select>
              </Field>
              <Field label="예약 간격 (분, 0은 수동)">
                <input
                  type="number"
                  min={0}
                  max={525600}
                  required
                  value={draft.interval_minutes}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      interval_minutes: Number(e.target.value),
                    })
                  }
                />
              </Field>
            </div>
            <p className="muted">
              계정은 관리자 사용자 메뉴에서 서비스 계정으로 만들고
              워크스페이스에 편집자 이상으로 추가하세요. 예약은 최소 5분입니다.
              생성 후 연결 위치·계정·공간은 고정됩니다.
            </p>
            <h3>원격 자료 범위</h3>
            <div className="form-grid">
              {providerFields[draft.kind].map(([key, label, placeholder]) => (
                <Field label={label} key={key}>
                  <input
                    value={draft.config[key] || ""}
                    placeholder={placeholder}
                    onChange={(e) => config(key, e.target.value)}
                  />
                </Field>
              ))}
              {["jira", "confluence"].includes(draft.kind) && (
                <Field label="API 버전">
                  <select
                    value={draft.config.variant || "cloud"}
                    onChange={(e) => config("variant", e.target.value)}
                  >
                    <option value="cloud">
                      Cloud (Jira v3 / Confluence v2)
                    </option>
                    <option value="server">
                      Server / Data Center (기존 REST API)
                    </option>
                  </select>
                </Field>
              )}
              <Field label="한 작업의 처리 목표 (페이지 단위)">
                <input
                  required
                  type="number"
                  min={100}
                  max={10000}
                  value={draft.config.max_items}
                  onChange={(e) => config("max_items", Number(e.target.value))}
                />
              </Field>
              <Field label="로컬 변경 충돌 정책">
                <select
                  value={draft.config.conflict_policy || "preserve_local"}
                  onChange={(e) => config("conflict_policy", e.target.value)}
                >
                  <option value="preserve_local">
                    사용자 편집 보존 · 충돌 보고
                  </option>
                  <option value="replace_local">
                    원격 내용으로 교체 · 이전 버전 보존
                  </option>
                </select>
              </Field>
            </div>
            <p className="muted">
              문서는 4MB, 응답은 8MB, 한 번에 최대 100페이지로 제한합니다.
              페이지 중간을 건너뛰지 않으므로 마지막 페이지 크기만큼 처리 목표를
              넘을 수 있습니다. Drive의 비텍스트 파일은 내용 대신 출처
              메타데이터를 보존합니다.
            </p>
            <h3>인증과 연결 보안</h3>
            <div className="form-grid">
              <Field label="인증 방식">
                <select
                  value={draft.config.auth_mode || "bearer"}
                  onChange={(e) => config("auth_mode", e.target.value)}
                >
                  <option value="bearer">Bearer 토큰</option>
                  <option value="private_token">GitLab PRIVATE-TOKEN</option>
                  <option value="basic">Basic 사용자 / API 토큰</option>
                </select>
              </Field>
              {draft.config.auth_mode === "basic" ? (
                <>
                  <Field label="원격 사용자 이름">
                    <input
                      autoComplete="off"
                      value={draft.credentials.username}
                      placeholder={draft.id ? "변경할 때만 입력" : ""}
                      onChange={(e) =>
                        setDraft({
                          ...draft,
                          credentials: {
                            ...draft.credentials,
                            username: e.target.value,
                          },
                        })
                      }
                    />
                  </Field>
                  <Field label="원격 암호 / API 토큰">
                    <input
                      type="password"
                      autoComplete="new-password"
                      value={draft.credentials.password}
                      placeholder={draft.id ? "빈 값은 현재 암호 유지" : ""}
                      onChange={(e) =>
                        setDraft({
                          ...draft,
                          credentials: {
                            ...draft.credentials,
                            password: e.target.value,
                          },
                        })
                      }
                    />
                  </Field>
                </>
              ) : (
                <Field label="원격 접근 토큰">
                  <input
                    type="password"
                    autoComplete="new-password"
                    value={draft.credentials.token}
                    placeholder={
                      draft.id
                        ? "빈 값은 현재 토큰 유지"
                        : "읽기 전용 최소 권한 토큰"
                    }
                    onChange={(e) =>
                      setDraft({
                        ...draft,
                        credentials: {
                          ...draft.credentials,
                          token: e.target.value,
                        },
                      })
                    }
                  />
                </Field>
              )}
            </div>
            <Field label="사내 CA 인증서 PEM (선택)">
              <textarea
                rows={3}
                value={draft.config.ca_pem || ""}
                onChange={(e) => config("ca_pem", e.target.value)}
              />
            </Field>
            <label className="check-row">
              <input
                type="checkbox"
                checked={!!draft.config.allow_http}
                onChange={(e) => config("allow_http", e.target.checked)}
              />
              사내망 HTTP 허용 (전송 암호화 없음)
            </label>
            <label className="check-row">
              <input
                type="checkbox"
                checked={!!draft.config.insecure_tls}
                onChange={(e) => config("insecure_tls", e.target.checked)}
              />
              인증서 검증 생략 (테스트 연결에만 사용)
            </label>
            <label className="check-row">
              <input
                type="checkbox"
                required
                checked={!!draft.config.acknowledge_acl}
                onChange={(e) => config("acknowledge_acl", e.target.checked)}
              />
              원격 프로젝트가 비공개여도 대상 공간 ACL을 적용하며 원격 사용자
              권한은 복사하지 않음을 확인합니다.
            </label>
            <label className="check-row">
              <input
                type="checkbox"
                checked={!!draft.enabled}
                onChange={(e) =>
                  setDraft({ ...draft, enabled: e.target.checked })
                }
              />
              이 연결 활성화
            </label>
            <div className="modal-actions">
              <Button
                type="button"
                disabled={busy}
                onClick={() => setDraft(null)}
              >
                취소
              </Button>
              <Button type="submit" variant="primary" disabled={busy}>
                <Save size={17} />
                {busy ? "저장 중…" : "연결 저장"}
              </Button>
            </div>
          </form>
        </Modal>
      )}
      {selected && (
        <Modal
          open
          wide
          title={`${selected.name} · 가져오기 이력`}
          onOpenChange={() => setParams({})}
        >
          <p className="muted">
            작업 체크포인트와 처리 결과입니다. 실패한 작업은 작업 이력에서
            재시도할 수 있습니다.
          </p>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>유형 / 시각</th>
                  <th>상태</th>
                  <th>결과</th>
                </tr>
              </thead>
              <tbody>
                {detail.runs.map((run) => (
                  <tr key={run.job_id}>
                    <td>
                      <Link to={`/app/jobs?job=${run.job_id}`}>
                        {run.kind === "connector.preview"
                          ? "미리보기"
                          : "가져오기"}
                      </Link>
                      <div className="muted">
                        {run.created_at ? datetime(run.created_at) : "방금 전"}
                      </div>
                    </td>
                    <td>
                      <Badge>{stateNames[run.status] || run.status}</Badge>
                    </td>
                    <td>
                      {run.kind === "connector.preview"
                        ? `${run.report?.preview_count || 0}개 미리보기`
                        : `신규 ${run.report?.created || 0} · 갱신 ${run.report?.updated || 0} · 유지 ${run.report?.unchanged || 0} · 충돌 ${run.report?.conflicts || 0}`}{" "}
                      {run.report?.has_more && <Badge>다음 페이지 있음</Badge>}
                      {run.last_error && (
                        <p className="error-text">{run.last_error}</p>
                      )}
                      {Array.isArray(run.report?.samples) && (
                        <details>
                          <summary>미리보기 문서</summary>
                          <ul>
                            {run.report.samples.map((s: Row) => (
                              <li key={s.remote_id}>
                                {s.title} · {s.characters}자
                              </li>
                            ))}
                          </ul>
                        </details>
                      )}
                      {Array.isArray(run.report?.warnings) &&
                        run.report.warnings.length > 0 && (
                          <details>
                            <summary>주의 사항</summary>
                            <ul>
                              {run.report.warnings.map(
                                (s: string, i: number) => (
                                  <li key={i}>{s}</li>
                                ),
                              )}
                            </ul>
                          </details>
                        )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <h3>가져온 문서 (최근 최대 1,000개)</h3>
          {detail.records.length ? (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>문서</th>
                    <th>원격 ID</th>
                    <th>마지막 확인</th>
                  </tr>
                </thead>
                <tbody>
                  {detail.records.map((row) => (
                    <tr key={row.remote_id}>
                      <td>
                        {row.document_id ? (
                          <Link to={`/app/documents/${row.document_id}`}>
                            {row.title || "문서 보기"}
                          </Link>
                        ) : (
                          "삭제된 문서 · 자동 복원 안 함"
                        )}
                      </td>
                      <td>{row.remote_id}</td>
                      <td>{datetime(row.last_seen_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="muted">아직 저장된 문서가 없습니다.</p>
          )}
        </Modal>
      )}
    </>
  );
}

export function ConnectorPolicyPage() {
  const { notify } = useApp();
  const [enabled, setEnabled] = useState(false),
    [hosts, setHosts] = useState(""),
    [error, setError] = useState(""),
    [loading, setLoading] = useState(true),
    [busy, setBusy] = useState(false);
  useEffect(() => {
    let active = true;
    api<Row>("/admin/connectors/settings")
      .then((v) => {
        if (active) {
          setEnabled(v.enabled);
          setHosts((v.allowed_hosts || []).join("\n"));
        }
      })
      .catch((e) => {
        if (active) setError(e.message);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, []);
  return (
    <>
      <PageHeading
        eyebrow="EXPLICIT NETWORK POLICY"
        title="외부 연결 정책"
        description="서비스가 접근할 원격 시스템을 정확한 호스트 단위로 허용합니다."
      />
      <ErrorBox error={error} />
      {loading ? (
        <Loading />
      ) : (
        <form
          className="card settings-form"
          onSubmit={async (e) => {
            e.preventDefault();
            setBusy(true);
            try {
              await api("/admin/connectors/settings", "PUT", {
                enabled,
                allowed_hosts: hosts
                  .split(/[\n,]/)
                  .map((v) => v.trim())
                  .filter(Boolean),
              });
              notify("외부 연결 정책을 저장했습니다");
              setError("");
            } catch (e) {
              setError((e as Error).message);
            } finally {
              setBusy(false);
            }
          }}
        >
          <div className="card-header">
            <div>
              <h2>기본 거부 · 관리자 허용</h2>
              <p>
                폐쇄망에서는 허용한 내부 호스트만 연결하세요. 환경변수를 추가할
                필요가 없습니다.
              </p>
            </div>
          </div>
          <label className="check-row">
            <input
              type="checkbox"
              checked={enabled}
              onChange={(e) => setEnabled(e.target.checked)}
            />
            외부 커넥터 사용 허용
          </label>
          <Field label="허용 호스트 (줄마다 하나, 최대 100개)">
            <textarea
              rows={8}
              value={hosts}
              onChange={(e) => setHosts(e.target.value)}
              placeholder={
                "gitlab.example.internal\nconfluence.example.internal\napi.github.com"
              }
            />
          </Field>
          <p className="muted">
            스킴·포트·경로·와일드카드를 제외한 정확한 호스트 또는 IP만
            입력합니다. 다른 출처 리다이렉트와 페이지네이션, 클라우드
            메타데이터·링크 로컬 주소는 차단합니다. DNS는 실제 연결 시 다시
            검증합니다.
          </p>
          <div className="notice">
            <ShieldCheck size={18} />
            <span>
              원격 서비스는 읽기 전용 API 토큰을 사용하세요. 토큰은 암호화
              저장되며 조회 응답에 다시 노출하지 않습니다. 서비스 계정과 실행
              관리자의 공간 권한을 교집합으로 적용합니다.
            </span>
          </div>
          <div className="button-row">
            <Button variant="primary" type="submit" disabled={busy}>
              <Save size={17} />
              {busy ? "저장 중…" : "정책 저장"}
            </Button>
            <Link className="button" to="/app/connectors">
              워크스페이스 커넥터
            </Link>
          </div>
        </form>
      )}
    </>
  );
}
