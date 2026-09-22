import { useCallback, useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import {
  Archive,
  CheckCircle2,
  Cloud,
  Database,
  Download,
  HardDrive,
  Plus,
  RefreshCw,
  ShieldCheck,
} from "lucide-react";
import { api, bytes, datetime } from "./api";
import { useApp } from "./context";
import "./storage.css";
import {
  Badge,
  Button,
  Empty,
  ErrorBox,
  Field,
  Loading,
  Modal,
  PageHeading,
  Toggle,
} from "./ui";
type Row = Record<string, any>;
const defaultS3 = () => ({
  endpoint: "",
  bucket: "",
  prefix: "madi",
  region: "us-east-1",
  access_key: "",
  secret_key: "",
  session_token: "",
  allow_http: false,
  insecure_tls: false,
  ca_pem: "",
});

export function StoragePage({ admin = false }: { admin?: boolean }) {
  const { workspace, workspaces, user, notify } = useApp();
  const scope = admin ? "" : workspace?.id || "";
  const manager = admin
    ? user.role === "admin"
    : ["owner", "admin"].includes(workspace?.role || "");
  const [providers, setProviders] = useState<Row[]>([]),
    [assignment, setAssignment] = useState(""),
    [draft, setDraft] = useState<Row | null>(null),
    [loading, setLoading] = useState(true),
    [error, setError] = useState(""),
    [formError, setFormError] = useState(""),
    [busy, setBusy] = useState(false),
    [diagnosing, setDiagnosing] = useState("");
  const generation = useRef(0);
  const load = useCallback(async () => {
    const run = ++generation.current;
    setError("");
    if (!manager) {
      setLoading(false);
      return;
    }
    try {
      const [p, a] = await Promise.all([
        api<Row[]>(`/storage/providers?workspace_id=${scope}`),
        api<Row>(`/storage/assignment?workspace_id=${scope}`),
      ]);
      if (run === generation.current) {
        setProviders(p);
        setAssignment(a.provider_id || "");
      }
    } catch (e) {
      if (run === generation.current) setError((e as Error).message);
    } finally {
      if (run === generation.current) setLoading(false);
    }
  }, [scope, manager]);
  useEffect(() => {
    setProviders([]);
    setDraft(null);
    setLoading(true);
    void load();
    return () => {
      generation.current++;
    };
  }, [load]);
  const edit = (p?: Row) => {
    setFormError("");
    setDraft(
      p
        ? {
            ...p,
            config: {
              ...p.config,
              access_key: "",
              secret_key: "",
              session_token: "",
            },
          }
        : {
            workspace_id: scope,
            name: "",
            kind: "s3",
            enabled: true,
            config: defaultS3(),
          },
    );
  };
  const field = (key: string, value: any) =>
    setDraft((d) => (d ? { ...d, config: { ...d.config, [key]: value } } : d));
  return (
    <>
      <PageHeading
        eyebrow="LOCAL · S3 · MINIO"
        title={admin ? "서비스 저장소" : "워크스페이스 저장소"}
        description="오프라인 기본 저장소와 사내 S3·MinIO 연결을 관리합니다. 기존 파일의 위치는 유지됩니다."
        actions={
          <>
            {admin && (
              <Link to="/admin/backup-schedule" className="button">
                <Archive size={17} />
                예약 백업
              </Link>
            )}
            <Button onClick={() => void load()}>
              <RefreshCw size={17} />
              새로고침
            </Button>
            {manager && (
              <Button variant="primary" onClick={() => edit()}>
                <Plus size={17} />
                저장소 연결
              </Button>
            )}
          </>
        }
      />
      <ErrorBox error={error} />
      {!manager ? (
        <Empty
          title="저장소 관리 권한이 필요합니다"
          text="워크스페이스 소유자 또는 관리자에게 문의하세요."
        />
      ) : loading ? (
        <Loading />
      ) : (
        <>
          <form
            className="panel padded"
            onSubmit={async (e) => {
              e.preventDefault();
              setBusy(true);
              setError("");
              try {
                await api("/storage/assignment", "PUT", {
                  workspace_id: scope,
                  provider_id: assignment,
                });
                notify(
                  "새 파일의 기본 저장소를 변경했습니다. 기존 첨부파일은 원래 저장소에서 읽습니다.",
                );
              } catch (e) {
                setError((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            <div className="section-heading">
              <h2>
                <Database size={20} />새 파일을 저장할 위치
              </h2>
            </div>
            <Field
              label={admin ? "서비스 기본 저장소" : "이 워크스페이스의 저장소"}
            >
              <select
                value={assignment}
                onChange={(e) => setAssignment(e.target.value)}
              >
                <option value="">
                  {admin
                    ? "기본 로컬 폴더 (관리자 시스템 설정)"
                    : "서비스 기본 저장소 상속"}
                </option>
                {providers
                  .filter(
                    (p) =>
                      (!admin || !p.workspace_id) &&
                      (p.enabled || p.id === assignment),
                  )
                  .map((p) => (
                    <option key={p.id} value={p.id} disabled={!p.enabled}>
                      {p.name} · {p.kind === "local" ? "로컬" : "S3 / MinIO"}
                      {!p.enabled ? " (중지됨)" : ""}
                    </option>
                  ))}
              </select>
            </Field>
            <Button disabled={busy} variant="primary">
              기본 저장소 저장
            </Button>
          </form>
          <div style={{ display: "grid", gap: 18, marginTop: 24 }}>
            {providers.length ? (
              providers.map((p) => (
                <article className="panel padded" key={p.id}>
                  <div className="section-heading">
                    <h2>
                      {p.kind === "local" ? (
                        <HardDrive size={21} />
                      ) : (
                        <Cloud size={21} />
                      )}{" "}
                      {p.name}
                    </h2>
                    <Badge tone={p.enabled ? "green" : ""}>
                      {p.enabled ? "새 파일 저장 가능" : "새 파일 저장 중지"}
                    </Badge>
                  </div>
                  <p className="muted">
                    {p.workspace_id ? "워크스페이스 전용" : "서비스 공용"} ·{" "}
                    {p.kind === "local" ? "로컬 파일" : "S3 / MinIO"}
                  </p>
                  {p.config.endpoint && (
                    <p style={{ overflowWrap: "anywhere" }}>
                      {p.config.endpoint} / {p.config.bucket} /{" "}
                      {p.config.prefix || ""}
                    </p>
                  )}
                  {p.config.root && (
                    <p style={{ overflowWrap: "anywhere" }}>{p.config.root}</p>
                  )}
                  <p className="muted">
                    파일마다 SHA-256 검증 · 연결 정보 암호화 · 기존 객체 위치
                    보존
                  </p>
                  {p.can_manage && (
                    <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                      <Button onClick={() => edit(p)}>
                        설정 · 자격 증명 회전
                      </Button>
                      <Button
                        disabled={!!diagnosing}
                        onClick={async () => {
                          setDiagnosing(p.id);
                          try {
                            await api(
                              `/storage/providers/${p.id}/test`,
                              "POST",
                              {},
                            );
                            notify(
                              "쓰기 · 읽기 · 체크섬 · 삭제 진단을 모두 통과했습니다.",
                            );
                          } catch (e) {
                            notify((e as Error).message, "error");
                          } finally {
                            setDiagnosing("");
                          }
                        }}
                      >
                        <ShieldCheck size={17} />
                        {diagnosing === p.id ? "연결 진단 중…" : "연결 진단"}
                      </Button>
                    </div>
                  )}
                </article>
              ))
            ) : (
              <Empty
                title="기본 로컬 저장소를 사용 중입니다"
                text="별도 연결 없이 동작합니다. S3 또는 MinIO가 필요하면 저장소 연결을 추가하세요."
              />
            )}
          </div>
        </>
      )}
      <section className="notice subtle" style={{ marginTop: 24 }}>
        <ShieldCheck size={22} />
        <span>
          비활성화는 새 파일 저장을 막습니다. 기존 파일 읽기는 계속 가능하며,
          연결 위치는 생성 후 변경하지 않습니다. 사내 HTTPS 인증서는 CA PEM으로
          등록하고, HTTP나 인증서 검증 생략은 관리자가 위험을 확인한 경우에만
          사용하세요.
        </span>
      </section>
      <Modal
        open={!!draft}
        onOpenChange={(v) => {
          if (!v && !busy) setDraft(null);
        }}
        title={draft?.id ? "저장소 설정과 자격 증명" : "저장소 연결"}
        description="키는 암호화하여 보관하고 저장 후 다시 표시하지 않습니다. 수정 시 빈 키는 기존 값을 유지합니다."
        wide
      >
        {draft && (
          <form
            onSubmit={async (e) => {
              e.preventDefault();
              setBusy(true);
              setFormError("");
              try {
                await api(
                  `/storage/providers${draft.id ? `/${draft.id}` : ""}`,
                  draft.id ? "PUT" : "POST",
                  draft,
                );
                setDraft(null);
                await load();
                notify(
                  "저장소 연결을 저장했습니다. 연결 진단을 실행해 주세요.",
                );
              } catch (e) {
                setFormError((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            <ErrorBox error={formError} />
            <Field label="연결 이름">
              <input
                required
                maxLength={120}
                value={draft.name}
                onChange={(e) => setDraft({ ...draft, name: e.target.value })}
              />
            </Field>
            <div className="form-grid">
              <Field label="저장소 유형">
                <select
                  disabled={!!draft.id}
                  value={draft.kind}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      kind: e.target.value,
                      config:
                        e.target.value === "s3"
                          ? defaultS3()
                          : { root: "/var/lib/madi/attachments" },
                    })
                  }
                >
                  <option value="s3">S3 / MinIO</option>
                  {user.role === "admin" && (
                    <option value="local">로컬 파일</option>
                  )}
                </select>
              </Field>
              {admin && (
                <Field label="사용 범위">
                  <select
                    disabled={!!draft.id}
                    value={draft.workspace_id || ""}
                    onChange={(e) =>
                      setDraft({ ...draft, workspace_id: e.target.value })
                    }
                  >
                    <option value="">서비스 공용</option>
                    {workspaces.map((w) => (
                      <option key={w.id} value={w.id}>
                        {w.name}
                      </option>
                    ))}
                  </select>
                </Field>
              )}
            </div>
            {draft.kind === "local" ? (
              <Field
                label="로컬 절대 경로"
                hint="컨테이너 안의 경로입니다. 호스트 디렉터리를 볼륨으로 마운트하세요."
              >
                <input
                  required
                  readOnly={!!draft.id}
                  value={draft.config.root || ""}
                  onChange={(e) => field("root", e.target.value)}
                />
              </Field>
            ) : (
              <>
                <Field
                  label="S3 엔드포인트"
                  hint="버킷을 제외한 내부 HTTP(S) 주소. 예: https://minio.internal:9000"
                >
                  <input
                    type="url"
                    required
                    readOnly={!!draft.id}
                    value={draft.config.endpoint || ""}
                    onChange={(e) => field("endpoint", e.target.value)}
                  />
                </Field>
                <div className="form-grid">
                  <Field label="버킷">
                    <input
                      required
                      readOnly={!!draft.id}
                      value={draft.config.bucket || ""}
                      onChange={(e) => field("bucket", e.target.value)}
                    />
                  </Field>
                  <Field label="리전">
                    <input
                      readOnly={!!draft.id}
                      value={draft.config.region || "us-east-1"}
                      onChange={(e) => field("region", e.target.value)}
                    />
                  </Field>
                </div>
                <Field
                  label="객체 접두사"
                  hint="앞뒤 슬래시 없이 입력합니다. 파일은 이 경로 아래 UUID 키로 저장됩니다."
                >
                  <input
                    readOnly={!!draft.id}
                    value={draft.config.prefix || ""}
                    onChange={(e) => field("prefix", e.target.value)}
                  />
                </Field>
                <div className="form-grid">
                  <Field label={draft.id ? "액세스 키 교체" : "액세스 키"}>
                    <input
                      type="password"
                      required={!draft.id}
                      autoComplete="new-password"
                      value={draft.config.access_key || ""}
                      onChange={(e) => field("access_key", e.target.value)}
                    />
                  </Field>
                  <Field label={draft.id ? "비밀 키 교체" : "비밀 키"}>
                    <input
                      type="password"
                      required={!draft.id}
                      autoComplete="new-password"
                      value={draft.config.secret_key || ""}
                      onChange={(e) => field("secret_key", e.target.value)}
                    />
                  </Field>
                </div>
                <Field label="세션 토큰 (선택)">
                  <input
                    type="password"
                    autoComplete="new-password"
                    value={draft.config.session_token || ""}
                    onChange={(e) => field("session_token", e.target.value)}
                  />
                </Field>
                {draft.id && (
                  <Toggle
                    label="저장된 세션 토큰 삭제"
                    description="임시 자격 증명에서 일반 액세스 키로 전환할 때 사용합니다."
                    checked={!!draft.config.clear_session_token}
                    onChange={(v) => field("clear_session_token", v)}
                  />
                )}
                <Field label="사내 CA 인증서 (PEM, 선택)">
                  <textarea
                    rows={4}
                    value={draft.config.ca_pem || ""}
                    onChange={(e) => field("ca_pem", e.target.value)}
                    placeholder="-----BEGIN CERTIFICATE-----"
                  />
                </Field>
                <Toggle
                  label="사내 HTTP 사용 허용"
                  description="암호화되지 않은 네트워크 전송입니다. 신뢰할 수 있는 내부망에서만 사용하세요."
                  checked={!!draft.config.allow_http}
                  onChange={(v) => field("allow_http", v)}
                />
                <Toggle
                  label="TLS 인증서 검증 생략 (권장하지 않음)"
                  description="기본은 인증서를 검증합니다. 가능하면 사내 CA 인증서를 등록하세요."
                  checked={!!draft.config.insecure_tls}
                  onChange={(v) => {
                    if (
                      !v ||
                      confirm(
                        "인증서 검증을 생략하면 서버 위조를 탐지하지 못합니다. 위험을 확인하고 사용하시겠습니까?",
                      )
                    )
                      field("insecure_tls", v);
                  }}
                />
              </>
            )}
            <Toggle
              label="새 파일 저장 허용"
              description="기존 파일은 비활성화 후에도 해당 저장소에서 읽습니다."
              checked={draft.enabled}
              onChange={(v) => setDraft({ ...draft, enabled: v })}
            />
            <Button variant="primary" disabled={busy}>
              {busy ? "저장 중…" : "연결 저장"}
            </Button>
          </form>
        )}
      </Modal>
    </>
  );
}

export function BackupSchedulePage() {
  const { notify } = useApp();
  const [policy, setPolicy] = useState<Row | null>(null),
    [providers, setProviders] = useState<Row[]>([]),
    [artifacts, setArtifacts] = useState<Row[]>([]),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  const generation = useRef(0);
  const load = useCallback(async () => {
    const run = ++generation.current;
    try {
      const [p, s, a] = await Promise.all([
        api<Row>("/admin/backups/policy"),
        api<Row[]>("/storage/providers"),
        api<Row[]>("/admin/backups"),
      ]);
      if (run === generation.current) {
        setPolicy(p);
        setProviders(s.filter((x) => !x.workspace_id));
        setArtifacts(a);
        setError("");
      }
    } catch (e) {
      if (run === generation.current) setError((e as Error).message);
    }
  }, []);
  useEffect(() => {
    void load();
    return () => {
      generation.current++;
    };
  }, [load]);
  return (
    <>
      <PageHeading
        eyebrow="RECOVER WITH CONFIDENCE"
        title="예약 백업"
        description="PostgreSQL 논리 데이터와 모든 저장소의 첨부파일을 작업 큐에서 안전하게 백업합니다."
        actions={
          <>
            <Link className="button" to="/admin/backup">
              내보내기 · 복원
            </Link>
            <Link className="button" to="/admin/storage">
              저장소 연결
            </Link>
            <Button onClick={() => void load()}>
              <RefreshCw size={17} />
              새로고침
            </Button>
          </>
        }
      />
      <ErrorBox error={error} />
      <div className="notice subtle">
        <ShieldCheck size={21} />
        <span>
          ENCRYPTION_KEY는 백업에 포함하지 않습니다. 별도로 안전하게 보관하세요.
          첨부파일과 암호화된 연결 설정은 포함됩니다. 시점 복구(PITR)는
          PostgreSQL 운영 백업과 함께 구성해야 합니다.
        </span>
      </div>
      {policy ? (
        <form
          className="panel padded"
          onSubmit={async (e) => {
            e.preventDefault();
            setBusy(true);
            setError("");
            try {
              await api("/admin/backups/policy", "PUT", {
                ...policy,
                provider_id: policy.provider_id || "",
              });
              notify("예약 백업 정책을 저장했습니다.");
              await load();
            } catch (e) {
              setError((e as Error).message);
            } finally {
              setBusy(false);
            }
          }}
        >
          <h2>
            <Archive size={21} />
            백업 정책
          </h2>
          <Toggle
            label="예약 백업 사용"
            description="작업 처리 설정이 일시 중지되면 예약 백업도 대기합니다."
            checked={policy.enabled}
            onChange={(v) => setPolicy({ ...policy, enabled: v })}
          />
          <Field label="백업 저장소">
            <select
              value={policy.provider_id || ""}
              onChange={(e) =>
                setPolicy({ ...policy, provider_id: e.target.value })
              }
            >
              <option value="">별도 로컬 백업 폴더</option>
              {providers
                .filter((p) => p.enabled || p.id === policy.provider_id)
                .map((p) => (
                  <option key={p.id} value={p.id} disabled={!p.enabled}>
                    {p.name} · {p.kind === "s3" ? "S3 / MinIO" : "로컬"}
                  </option>
                ))}
            </select>
          </Field>
          <Field
            label="백업 전용 로컬 폴더"
            hint="로컬 폴더를 선택할 때 사용합니다. 다른 파일은 삭제하지 않고 서비스가 기록한 백업만 정리합니다."
          >
            <input
              required
              value={policy.local_path}
              onChange={(e) =>
                setPolicy({ ...policy, local_path: e.target.value })
              }
            />
          </Field>
          <div className="form-grid">
            <Field label="예약 간격 (분)">
              <input
                type="number"
                min={5}
                max={525600}
                required
                value={policy.interval_minutes}
                onChange={(e) =>
                  setPolicy({
                    ...policy,
                    interval_minutes: Number(e.target.value),
                  })
                }
              />
            </Field>
            <Field label="최근 백업 보존 개수">
              <input
                type="number"
                min={1}
                max={1000}
                required
                value={policy.retention_count}
                onChange={(e) =>
                  setPolicy({
                    ...policy,
                    retention_count: Number(e.target.value),
                  })
                }
              />
            </Field>
          </div>
          <p className="muted">
            다음 예정: {datetime(policy.next_run)} · 저장할 때 다음 실행 시각이
            다시 계산됩니다.
          </p>
          <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
            <Button variant="primary" disabled={busy}>
              {busy ? "처리 중…" : "백업 정책 저장"}
            </Button>
            <Button
              type="button"
              disabled={busy}
              onClick={async () => {
                setBusy(true);
                try {
                  await api("/admin/backups/run", "POST", {});
                  notify(
                    "현재 저장된 정책으로 백업 작업을 예약했습니다. 작업 이력에서 완료를 확인하세요.",
                  );
                } catch (e) {
                  notify((e as Error).message, "error");
                } finally {
                  setBusy(false);
                }
              }}
            >
              지금 백업
            </Button>
            <Link className="button" to="/app/jobs">
              작업 이력 확인
            </Link>
          </div>
        </form>
      ) : (
        !error && <Loading />
      )}
      <section style={{ marginTop: 28 }}>
        <div className="section-heading">
          <h2>보관된 백업</h2>
          <Badge>{artifacts.length}개</Badge>
        </div>
        <p className="muted">
          백업과 웹 복원은 압축 파일 256MB · 해제 크기 1GB · 10,000개 항목까지
          지원합니다. 한도를 초과하면 백업 작업이 실패하며, 대규모 운영 복구는
          PostgreSQL 네이티브 백업과 스토리지 백업을 사용하세요.
        </p>
        {artifacts.length ? (
          <div className="panel table-scroll backup-artifact-table">
            <table className="data-table">
              <thead>
                <tr>
                  <th>생성 시각</th>
                  <th>크기</th>
                  <th>무결성</th>
                  <th>파일</th>
                </tr>
              </thead>
              <tbody>
                {artifacts.map((a) => (
                  <tr key={a.id}>
                    <td>
                      {datetime(a.created_at)}
                      {a.managed === false && (
                        <p className="muted">
                          이전 보관소 이력 · 자동 정리 제외
                        </p>
                      )}
                    </td>
                    <td>{bytes(a.size)}</td>
                    <td>
                      <CheckCircle2 size={16} /> SHA-256
                      <details>
                        <summary>체크섬 보기</summary>
                        <code style={{ overflowWrap: "anywhere" }}>
                          {a.checksum_sha256}
                        </code>
                      </details>
                      {a.last_error && <ErrorBox error={a.last_error} />}
                    </td>
                    <td>
                      <a
                        className="button"
                        href={`/api/v1/admin/backups/${a.id}/download`}
                      >
                        <Download size={17} />
                        다운로드
                      </a>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <Empty
            title="아직 보관된 백업이 없습니다"
            text="정책을 저장하고 지금 백업을 실행해 복원 준비 상태를 확인하세요."
          />
        )}
      </section>
    </>
  );
}
