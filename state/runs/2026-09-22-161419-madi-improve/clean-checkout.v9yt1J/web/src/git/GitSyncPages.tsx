import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { GitBranch, Plus, RefreshCw, Save } from "lucide-react";
import { api, datetime } from "../api";
import { useApp } from "../context";
import {
  Badge,
  Button,
  Empty,
  ErrorBox,
  Field,
  Loading,
  Modal,
  PageHeading,
} from "../ui";
import "./style.css";
type Row = Record<string, any>;
const stateNames: Record<string, string> = {
  preparing: "미리보기 준비",
  preview: "동의 대기",
  queued: "실행 대기",
  running: "실행 중",
  succeeded: "완료",
  conflict: "충돌",
  failed: "실패",
  cancelled: "취소",
  unknown: "결과 확인 필요",
};
const actionNames: Record<string, string> = {
  available: "선택 가능",
  create: "새 파일",
  update: "변경",
  unchanged: "동일",
  conflict: "충돌",
};
const emptyConfig = {
  url: "",
  branch: "main",
  prefix: "madi",
  username: "",
  password: "",
  private_key: "",
  private_key_password: "",
  host_key: "",
  ca_pem: "",
};

export function GitSyncPage() {
  const { workspace, user, documents, notify } = useApp();
  const [params, setParams] = useSearchParams();
  const [connections, setConnections] = useState<Row[]>([]),
    [runs, setRuns] = useState<Row[]>([]),
    [run, setRun] = useState<Row | null>(null),
    [draft, setDraft] = useState<Row | null>(null),
    [spaces, setSpaces] = useState<Row[]>([]),
    [members, setMembers] = useState<Row[]>([]),
    [selectedDocs, setSelectedDocs] = useState<string[]>([]),
    [paths, setPaths] = useState<string[]>([]),
    [direction, setDirection] = useState("push"),
    [readConsent, setReadConsent] = useState(false),
    [confirm, setConfirm] = useState(false),
    [ack, setAck] = useState(false),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [loading, setLoading] = useState(true);
  const generation = useRef(0);
  const operation = useRef(0);
  const wid = workspace?.id;
  const currentScope = useRef(wid);
  currentScope.current = wid;
  const activeConnection = useRef(params.get("connection"));
  activeConnection.current = params.get("connection");
  const activeRun = useRef(params.get("run"));
  activeRun.current = params.get("run");
  const manager = ["owner", "admin"].includes(workspace?.role || "");
  const selected = connections.find((x) => x.id === params.get("connection"));
  const load = useCallback(async () => {
    if (!wid || !manager) {
      setLoading(false);
      return;
    }
    const g = ++generation.current;
    try {
      const rows = await api<Row[]>(
        `/git-sync/connections?workspace_id=${wid}`,
      );
      if (g !== generation.current) return;
      setConnections(rows);
      setError("");
    } catch (e) {
      if (g === generation.current) setError((e as Error).message);
    } finally {
      if (g === generation.current) setLoading(false);
    }
  }, [wid, manager]);
  useEffect(() => {
    setConnections([]);
    setRuns([]);
    setRun(null);
    setDraft(null);
    setPaths([]);
    setSelectedDocs([]);
    setConfirm(false);
    setReadConsent(false);
    operation.current++;
    setBusy(false);
    setLoading(true);
    void load();
    return () => {
      generation.current++;
    };
  }, [load]);
  useEffect(() => {
    setRun(null);
    setPaths([]);
    setRuns([]);
    setReadConsent(false);
    setSelectedDocs([]);
    operation.current++;
    setBusy(false);
    setConfirm(false);
    if (!selected) return;
    let active = true;
    const reload = async () => {
      try {
        const rows = await api<Row[]>(
          `/git-sync/connections/${selected.id}/runs`,
        );
        if (active) setRuns(rows);
      } catch (e) {
        if (active) setError((e as Error).message);
      }
    };
    void reload();
    const timer = setInterval(() => void reload(), 3500);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [selected?.id]);
  const runID = params.get("run");
  useEffect(() => {
    setRun(null);
    setAck(false);
    setConfirm(false);
    operation.current++;
    setBusy(false);
    if (!runID || !selected) return;
    let active = true;
    const reload = async () => {
      try {
        const row = await api<Row>(`/git-sync/runs/${runID}`);
        if (active) {
          if (row.connection_id !== selected.id) {
            setError("선택한 연결의 작업이 아닙니다");
            setRun(null);
          } else setRun(row);
        }
      } catch (e) {
        if (active) {
          setError((e as Error).message);
          setRun(null);
        }
      }
    };
    void reload();
    const timer = setInterval(() => void reload(), 2500);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [runID, selected?.id, wid]);
  const execute = async (fn: (alive: () => boolean) => Promise<void>) => {
    const scope = wid;
    const connectionID = activeConnection.current;
    const expectedRun = activeRun.current;
    const op = ++operation.current;
    const alive = () =>
      op === operation.current &&
      scope === currentScope.current &&
      connectionID === activeConnection.current &&
      expectedRun === activeRun.current;
    setBusy(true);
    setError("");
    try {
      await fn(alive);
    } catch (e) {
      if (alive()) setError((e as Error).message);
    } finally {
      if (alive()) setBusy(false);
    }
  };
  const openDraft = async (row?: Row) =>
    execute(async (alive) => {
      const scope = wid;
      const [s, m] = await Promise.all([
        api<Row[]>(`/spaces?workspace_id=${wid}`),
        api<Row[]>(`/workspaces/${wid}/members`),
      ]);
      if (scope !== currentScope.current || !alive()) return;
      setSpaces(s.filter((x) => x.can_write));
      setMembers(m);
      setDraft({
        ...row,
        workspace_id: wid,
        owner_id: row?.owner_id || user.id,
        space_id: row?.space_id || "",
        name: row?.name || "",
        enabled: row?.enabled || false,
        config: {
          ...emptyConfig,
          ...row?.config,
          username: "",
          password: "",
          private_key: "",
          private_key_password: "",
        },
      });
    });
  const preview = () =>
    execute(async (alive) => {
      if (!selected) return;
      const scope = wid;
      const result = await api<Row>(
        `/git-sync/connections/${selected.id}/preview`,
        "POST",
        {
          direction,
          document_ids: direction === "push" ? selectedDocs : [],
          paths: direction === "pull" ? paths : [],
          read_consent: readConsent,
        },
      );
      if (scope !== currentScope.current || !alive()) return;
      setParams({ connection: selected.id, run: result.id });
      notify("Git 미리보기 작업을 등록했습니다");
    });
  if (!manager)
    return (
      <Empty
        title="워크스페이스 관리자 전용"
        text="Git 원격 연결과 문서 전송은 워크스페이스 소유자·관리자만 진행할 수 있습니다."
      />
    );
  return (
    <>
      <PageHeading
        eyebrow="GIT KNOWLEDGE SYNC"
        title="Git 동기화"
        description="선택한 Markdown과 첨부파일을 명시적으로 비교하고, 동의한 변경만 고정 브랜치에 반영합니다."
        actions={
          <div className="button-row">
            <Button variant="ghost" onClick={() => void load()}>
              <RefreshCw size={18} />
              새로고침
            </Button>
            {user.role === "admin" && (
              <Button onClick={() => void openDraft()}>
                <Plus size={18} />
                연결 추가
              </Button>
            )}
          </div>
        }
      />
      <ErrorBox error={error} />
      <div className="notice git-intro">
        원격 Git 저장소의 권한은 madi와 독립적입니다. 개인 문서도 전송하면 Git
        열람자가 볼 수 있습니다. 강제 푸시·원격 삭제 자동 반영·훅 실행은 하지
        않습니다.{" "}
        {user.role === "admin" && (
          <Link to="/admin/git-sync">서비스 Git 정책</Link>
        )}
      </div>
      {loading ? (
        <Loading />
      ) : (
        <div className="git-layout">
          <section className="card git-connections">
            <h2>저장된 연결</h2>
            {connections.length === 0 ? (
              <p className="muted">
                서비스 관리자에게 원격 연결 등록을 요청하세요.
              </p>
            ) : (
              connections.map((c) => (
                <button
                  key={c.id}
                  className={`git-connection ${selected?.id === c.id ? "active" : ""}`}
                  onClick={() => setParams({ connection: c.id })}
                >
                  <GitBranch size={20} />
                  <span>
                    <strong>{c.name}</strong>
                    <small>
                      {c.config.branch} · {c.config.prefix}
                    </small>
                  </span>
                  <Badge>{c.enabled ? "사용" : "중지"}</Badge>
                </button>
              ))
            )}
          </section>
          <div className="git-main">
            {!selected ? (
              <Empty
                title="Git 연결 선택"
                text="먼저 관리자가 저장한 고정 원격·브랜치를 선택하세요."
              />
            ) : (
              <>
                <section className="card">
                  <div className="section-heading">
                    <h2>{selected.name}</h2>
                    {user.role === "admin" && (
                      <Button
                        variant="ghost"
                        onClick={() => void openDraft(selected)}
                      >
                        연결 설정
                      </Button>
                    )}
                  </div>
                  <p className="git-url">{selected.config.url}</p>
                  <div className="button-row">
                    <Badge>{selected.config.branch}</Badge>
                    <Badge>{selected.config.prefix}</Badge>
                    <Button
                      variant="secondary"
                      disabled={busy || !selected.enabled}
                      onClick={() =>
                        void execute(async (alive) => {
                          const value = await api<Row>(
                            `/git-sync/connections/${selected.id}/test`,
                            "POST",
                            {},
                          );
                          if (alive()) notify(value.message);
                        })
                      }
                    >
                      읽기 연결 진단
                    </Button>
                  </div>
                  <div className="form-grid">
                    <Field label="동기화 방향">
                      <select
                        value={direction}
                        onChange={(e) => {
                          setDirection(e.target.value);
                          setPaths([]);
                          setSelectedDocs([]);
                        }}
                      >
                        <option value="push">선택 문서를 Git으로 전송</option>
                        <option value="pull">Git에서 개인 초안으로 수신</option>
                      </select>
                    </Field>
                    <Field label="읽기 동의">
                      <label className="git-checkbox">
                        <input
                          type="checkbox"
                          checked={readConsent}
                          onChange={(e) => setReadConsent(e.target.checked)}
                        />
                        이 고정 원격 저장소의 브랜치·파일을 읽겠습니다
                      </label>
                    </Field>
                  </div>
                  {direction === "push" ? (
                    <fieldset className="git-selection">
                      <legend>전송할 문서 · {selectedDocs.length}개</legend>
                      {documents
                        .filter((d) => d.workspace_id === wid && !d.deleted_at)
                        .map((d) => (
                          <label key={d.id} className="git-checkbox">
                            <input
                              type="checkbox"
                              checked={selectedDocs.includes(d.id)}
                              onChange={(e) =>
                                setSelectedDocs((old) =>
                                  e.target.checked
                                    ? [...old, d.id]
                                    : old.filter((id) => id !== d.id),
                                )
                              }
                            />
                            <span>
                              {d.title}
                              <small>
                                {d.visibility === "private"
                                  ? "개인 문서"
                                  : d.visibility === "selected"
                                    ? "선택 공유"
                                    : "워크스페이스"}{" "}
                                · v{d.version}
                              </small>
                            </span>
                          </label>
                        ))}
                    </fieldset>
                  ) : (
                    <p className="muted">
                      먼저 원격 목록을 미리 본 뒤 수신할 파일을 선택하여
                      미리보기를 다시 만드세요. 수신 연결의 실행 계정은 현재
                      사용자여야 하며 새 문서는 개인 초안으로 저장합니다.
                    </p>
                  )}
                  <Button
                    disabled={
                      busy ||
                      !selected.enabled ||
                      !readConsent ||
                      (direction === "push" && selectedDocs.length === 0)
                    }
                    onClick={() => void preview()}
                  >
                    {busy
                      ? "처리 중…"
                      : direction === "pull" && paths.length
                        ? `선택한 ${paths.length}개 다시 비교`
                        : "원격 읽기·미리보기"}
                  </Button>
                </section>
                {run && (
                  <section className="card">
                    <div className="section-heading">
                      <h2>변경 미리보기</h2>
                      <Badge>{stateNames[run.status] || run.status}</Badge>
                    </div>
                    <p className="muted">
                      {datetime(run.created_at)} · 동의 만료{" "}
                      {datetime(run.expires_at)}
                    </p>
                    <p className="git-url">
                      원격 commit: {run.remote_commit || "확인 중"}
                    </p>
                    {run.candidate_commit && (
                      <p className="git-url">
                        적용 후보 commit: {run.candidate_commit}
                      </p>
                    )}
                    <ErrorBox error={run.report?.error || ""} />
                    {run.report?.remote_visibility_warning && (
                      <div className="notice">
                        {run.report.remote_visibility_warning}
                      </div>
                    )}
                    {Array.isArray(run.report?.files) && (
                      <div className="table-scroll">
                        <table>
                          <thead>
                            <tr>
                              {run.direction === "pull" && <th>수신 선택</th>}
                              <th>파일</th>
                              <th>변경</th>
                              <th>크기</th>
                            </tr>
                          </thead>
                          <tbody>
                            {run.report.files.map((f: Row) => (
                              <tr key={f.path}>
                                {run.direction === "pull" && (
                                  <td>
                                    <input
                                      aria-label={`${f.path} 수신 선택`}
                                      type="checkbox"
                                      checked={paths.includes(f.path)}
                                      onChange={(e) =>
                                        setPaths((old) =>
                                          e.target.checked
                                            ? [...old, f.path]
                                            : old.filter((x) => x !== f.path),
                                        )
                                      }
                                    />
                                  </td>
                                )}
                                <td className="git-path">
                                  {f.path}
                                  {f.reason && (
                                    <small className="error-text">
                                      {f.reason}
                                    </small>
                                  )}
                                </td>
                                <td>{actionNames[f.action] || f.action}</td>
                                <td>{Math.ceil(f.size / 1024)} KB</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    )}
                    <div className="button-row">
                      {run.status === "preview" &&
                        !run.report?.has_conflicts &&
                        !run.report?.selection_required && (
                          <Button
                            disabled={busy}
                            onClick={() => {
                              setAck(false);
                              setConfirm(true);
                            }}
                          >
                            목록 확인 후 실행 동의
                          </Button>
                        )}
                      {[
                        "preparing",
                        "preview",
                        "queued",
                        "running",
                        "unknown",
                      ].includes(run.status) && (
                        <Button
                          variant="secondary"
                          disabled={busy}
                          onClick={() =>
                            void execute(async (alive) => {
                              await api(
                                `/git-sync/runs/${run.id}/cancel`,
                                "POST",
                                {},
                              );
                              if (alive())
                                notify(
                                  "취소를 요청했습니다. 이미 시작된 원격 쓰기는 되돌리지 않습니다",
                                );
                            })
                          }
                        >
                          작업 취소
                        </Button>
                      )}
                      {run.status === "unknown" && (
                        <Button
                          variant="secondary"
                          disabled={busy}
                          onClick={() =>
                            void execute(async (alive) => {
                              const value = await api<Row>(
                                `/git-sync/runs/${run.id}/inspect`,
                                "POST",
                                {},
                              );
                              if (alive()) notify(value.message);
                            })
                          }
                        >
                          원격 commit 결과 확인
                        </Button>
                      )}
                    </div>
                  </section>
                )}
                <section className="card">
                  <h2>내 실행 이력</h2>
                  {runs.length ? (
                    runs.map((row) => (
                      <button
                        key={row.id}
                        className="git-history"
                        onClick={() =>
                          setParams({ connection: selected.id, run: row.id })
                        }
                      >
                        <span>
                          {row.direction === "push" ? "Git 전송" : "Git 수신"} ·{" "}
                          {datetime(row.created_at)}
                        </span>
                        <Badge>{stateNames[row.status] || row.status}</Badge>
                      </button>
                    ))
                  ) : (
                    <p className="muted">아직 실행 이력이 없습니다.</p>
                  )}
                </section>
              </>
            )}
          </div>
        </div>
      )}
      {confirm && run && (
        <Modal
          title="Git 변경 실행 동의"
          open
          onOpenChange={(open) => !busy && setConfirm(open)}
        >
          <p>
            미리보기에 표시된 원격·파일 목록을 확인하세요. Git 저장소 열람자는
            madi의 개인 문서 권한과 무관하게 전송 원문을 볼 수 있습니다. 실행 후
            외부 Git 이력에서 내용을 완전히 제거할 수 있다는 보장은 없습니다.
          </p>
          <label className="git-checkbox">
            <input
              type="checkbox"
              checked={ack}
              onChange={(e) => setAck(e.target.checked)}
            />
            전송 목록·독립된 Git 권한·변경 결과를 확인하고 동의합니다
          </label>
          <div className="button-row">
            <Button
              variant="secondary"
              disabled={busy}
              onClick={() => setConfirm(false)}
            >
              돌아가기
            </Button>
            <Button
              disabled={busy || !ack}
              onClick={() =>
                void execute(async (alive) => {
                  await api(`/git-sync/runs/${run.id}/confirm`, "POST", {
                    revision: run.revision,
                    remote_commit: run.remote_commit,
                    config_fingerprint: run.config_fingerprint,
                    confirm_external_visibility: true,
                  });
                  if (!alive()) return;
                  setConfirm(false);
                  notify("동의한 Git 작업을 등록했습니다");
                })
              }
            >
              동의한 변경 실행
            </Button>
          </div>
        </Modal>
      )}
      {draft && (
        <Modal
          title={draft.id ? "Git 연결 설정" : "Git 연결 추가"}
          open
          onOpenChange={(open) => !busy && !open && setDraft(null)}
        >
          <form
            onSubmit={(e) => {
              e.preventDefault();
              void execute(async (alive) => {
                const value = await api<Row>(
                  draft.id
                    ? `/git-sync/connections/${draft.id}`
                    : "/git-sync/connections",
                  draft.id ? "PUT" : "POST",
                  {
                    ...draft,
                    config: Object.fromEntries(
                      Object.keys(emptyConfig).map((key) => [
                        key,
                        draft.config[key] || "",
                      ]),
                    ),
                  },
                );
                if (!alive()) return;
                setDraft(null);
                await load();
                if (!alive()) return;
                setParams({ connection: value.id });
                notify("암호화된 Git 연결을 저장했습니다");
              });
            }}
          >
            <fieldset className="git-form-fields" disabled={busy}>
              <div className="form-grid">
                <Field label="연결 이름">
                  <input
                    required
                    value={draft.name}
                    onChange={(e) =>
                      setDraft({ ...draft, name: e.target.value })
                    }
                  />
                </Field>
                <Field label="실행 계정">
                  <select
                    disabled={!!draft.id}
                    value={draft.owner_id}
                    onChange={(e) =>
                      setDraft({ ...draft, owner_id: e.target.value })
                    }
                  >
                    <option value={user.id}>{user.name} (나)</option>
                    {members
                      .filter((m) => (m.user_id || m.id) !== user.id)
                      .map((m) => (
                        <option
                          key={m.user_id || m.id}
                          value={m.user_id || m.id}
                        >
                          {m.name || m.email} ·{" "}
                          {m.kind === "service" ? "서비스" : "사용자"}
                        </option>
                      ))}
                  </select>
                </Field>
                <Field label="수신 공간">
                  <select
                    disabled={!!draft.id}
                    value={draft.space_id}
                    onChange={(e) =>
                      setDraft({ ...draft, space_id: e.target.value })
                    }
                  >
                    <option value="">워크스페이스 루트</option>
                    {spaces.map((s) => (
                      <option value={s.id} key={s.id}>
                        {s.name}
                      </option>
                    ))}
                  </select>
                </Field>
                <Field label="연결 상태">
                  <select
                    value={String(draft.enabled)}
                    onChange={(e) =>
                      setDraft({ ...draft, enabled: e.target.value === "true" })
                    }
                  >
                    <option value="false">중지</option>
                    <option value="true">사용</option>
                  </select>
                </Field>
              </div>
              {[
                ["url", "원격 URL (HTTPS 또는 SSH)"],
                ["branch", "고정 브랜치"],
                ["prefix", "Git 전용 폴더"],
              ].map(([key, label]) => (
                <Field key={key} label={label}>
                  <input
                    required
                    disabled={!!draft.id}
                    value={draft.config[key]}
                    onChange={(e) =>
                      setDraft({
                        ...draft,
                        config: { ...draft.config, [key]: e.target.value },
                      })
                    }
                  />
                </Field>
              ))}
              <p className="muted">
                원격·브랜치·폴더·실행 계정은 저장 후 변경할 수 없습니다. 새
                연결을 만드세요. 비밀 값은 비워 두면 유지되며 저장 후 다시
                표시되지 않습니다.
              </p>
              {[
                ["username", "HTTPS 사용자 이름"],
                ["password", "HTTPS 암호·개인 토큰"],
                ["private_key_password", "SSH 개인 키 암호"],
              ].map(([key, label]) => (
                <Field key={key} label={label}>
                  <input
                    type="password"
                    autoComplete="new-password"
                    value={draft.config[key]}
                    placeholder={
                      draft.config[`${key}_configured`]
                        ? "저장됨 · 변경할 때만 입력"
                        : ""
                    }
                    onChange={(e) =>
                      setDraft({
                        ...draft,
                        config: { ...draft.config, [key]: e.target.value },
                      })
                    }
                  />
                </Field>
              ))}
              {[
                ["private_key", "SSH 개인 키 (PEM/OpenSSH)"],
                ["host_key", "SSH 서버 고정 공개 키 (필수)"],
                ["ca_pem", "HTTPS 내부 CA 인증서 (선택)"],
              ].map(([key, label]) => (
                <Field key={key} label={label}>
                  <textarea
                    rows={3}
                    spellCheck={false}
                    value={draft.config[key]}
                    onChange={(e) =>
                      setDraft({
                        ...draft,
                        config: { ...draft.config, [key]: e.target.value },
                      })
                    }
                  />
                </Field>
              ))}
              <Button type="submit" disabled={busy}>
                <Save size={18} />
                연결 저장
              </Button>
            </fieldset>
          </form>
        </Modal>
      )}
    </>
  );
}

export function GitSyncPolicyPage() {
  const { notify } = useApp();
  const [value, setValue] = useState<Row | null>(null),
    [hosts, setHosts] = useState(""),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  useEffect(() => {
    let active = true;
    api<Row>("/admin/git-sync/settings")
      .then((row) => {
        if (active) {
          setValue(row);
          setHosts((row.settings.allowed_hosts || []).join("\n"));
        }
      })
      .catch((e) => active && setError(e.message));
    return () => {
      active = false;
    };
  }, []);
  return (
    <>
      <PageHeading
        eyebrow="GIT POLICY"
        title="Git 연동 정책"
        description="서비스 관리자만 허용 원격 호스트와 네트워크 범위를 정합니다. 기본값은 비활성화입니다."
      />
      <ErrorBox error={error} />
      {!value ? (
        <Loading />
      ) : (
        <form
          className="card git-policy"
          onSubmit={(e) => {
            e.preventDefault();
            setBusy(true);
            setError("");
            api<Row>("/admin/git-sync/settings", "PUT", {
              revision: value.revision,
              settings: {
                ...value.settings,
                allowed_hosts: hosts
                  .split(/\n|,/)
                  .map((x) => x.trim())
                  .filter(Boolean),
              },
            })
              .then((row) => {
                setValue(row);
                notify(
                  "Git 정책을 저장했습니다. 이전 미리보기 동의는 무효화됩니다",
                );
              })
              .catch((e) => setError(e.message))
              .finally(() => setBusy(false));
          }}
        >
          <fieldset disabled={busy}>
            <Field label="Git 기능">
              <select
                value={String(value.settings.enabled)}
                onChange={(e) =>
                  setValue({
                    ...value,
                    settings: {
                      ...value.settings,
                      enabled: e.target.value === "true",
                    },
                  })
                }
              >
                <option value="false">사용 안 함 (기본)</option>
                <option value="true">사용</option>
              </select>
            </Field>
            <Field label="정확한 허용 호스트 (한 줄에 하나)">
              <textarea
                rows={5}
                value={hosts}
                placeholder="git.example.internal"
                onChange={(e) => setHosts(e.target.value)}
              />
            </Field>
            <Field label="사내망 IP 연결">
              <select
                value={String(value.settings.allow_private_networks)}
                onChange={(e) =>
                  setValue({
                    ...value,
                    settings: {
                      ...value.settings,
                      allow_private_networks: e.target.value === "true",
                    },
                  })
                }
              >
                <option value="false">사설·루프백 IP 거부</option>
                <option value="true">명시한 호스트의 사내망 IP 허용</option>
              </select>
            </Field>
            <Field label="미리보기 최대 크기 (MB, 1~100)">
              <input
                required
                type="number"
                min={1}
                max={100}
                step={1}
                value={value.settings.max_snapshot_mb}
                onChange={(e) =>
                  setValue({
                    ...value,
                    settings: {
                      ...value.settings,
                      max_snapshot_mb: Number(e.target.value),
                    },
                  })
                }
              />
            </Field>
            <div className="notice">
              HTTPS 인증서는 항상 검증합니다. SSH는 저장된 서버 공개 키와
              일치해야 합니다. HTTP·파일 경로·Git 데몬·리디렉션·ambient SSH
              agent/proxy 설정을 사용하지 않습니다. 원격 pack은 별도 64MB 한도로
              제한됩니다.
            </div>
            <Button type="submit" disabled={busy}>
              <Save size={18} />
              정책 저장
            </Button>
          </fieldset>
        </form>
      )}
    </>
  );
}
