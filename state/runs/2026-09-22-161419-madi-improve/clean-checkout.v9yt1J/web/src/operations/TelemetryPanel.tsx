import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import {
  Activity,
  Database,
  History,
  KeyRound,
  RefreshCw,
  Save,
  ShieldCheck,
  TestTube2,
} from "lucide-react";
import { api, datetime } from "../api";
import { useApp } from "../context";
import { Badge, Button, Empty, ErrorBox, Field, Loading } from "../ui";
import {
  leaveOperations,
  useOperationMounted,
  useOperationsGuard,
} from "./shared";
type Telemetry = {
  running: boolean;
  enabled: boolean;
  sent_spans: number;
  failed_spans: number;
  dropped_spans: number;
  dropped_errors: number;
  queued_spans?: number;
  last_sent_at?: string;
  last_result?: string;
  configuration_error?: string;
  notice: string;
  uptime_seconds?: number;
};
type Health = {
  version: string;
  checked_at: string;
  database: {
    ready: boolean;
    latency_ms: number;
    connections: number;
    idle_connections: number;
    max_connections: number;
  };
  process: { heap_bytes: number; goroutines: number };
  requests: number;
  errors: number;
  responses: Record<string, number>;
  telemetry: Telemetry;
};
type ErrorEvent = {
  request_id: string;
  route: string;
  method: string;
  status: number;
  duration_ms: number;
  kind: string;
  created_at: string;
};
const configDefaults = {
  otel_enabled: false,
  otel_endpoint: "",
  otel_allow_http: false,
  otel_ca_pem: "",
  otel_auth_token: "",
  otel_sample_rate: 0.1,
  otel_timeout_seconds: 5,
  operations_errors_enabled: true,
  operations_retention_days: 7,
};
type Key = keyof typeof configDefaults;
type Config = Record<Key, string | number | boolean> & {
  settings_revision?: string;
  otel_auth_token_configured?: boolean;
};
function metric(value: number) {
  return new Intl.NumberFormat("ko-KR").format(value);
}
export function HealthPanel() {
  const [health, setHealth] = useState<Health | null>(null),
    [events, setEvents] = useState<ErrorEvent[]>([]),
    [error, setError] = useState<unknown>(null);
  const mounted = useOperationMounted();
  async function load() {
    try {
      const [h, e] = await Promise.all([
        api<Health>("/admin/operations/health"),
        api<ErrorEvent[]>("/admin/operations/errors"),
      ]);
      if (mounted.current) {
        setHealth(h);
        setEvents(e);
        setError(null);
      }
    } catch (e) {
      if (mounted.current) setError(e);
    }
  }
  useEffect(() => {
    void load();
    const timer = setInterval(() => {
      if (document.visibilityState === "visible") void load();
    }, 10000);
    return () => clearInterval(timer);
  }, []);
  return (
    <section>
      <div className="operations-toolbar">
        <p className="muted">
          실제 프로세스·PostgreSQL 상태입니다. 10초마다 갱신하며 재시작 후 요청
          카운터는 새로 시작합니다.
        </p>
        <Button variant="secondary" onClick={() => void load()}>
          <RefreshCw size={16} />
          새로고침
        </Button>
      </div>
      <ErrorBox error={error} />
      {!health ? (
        <Loading />
      ) : (
        <>
          <div className="operations-metrics">
            <article className="panel padded">
              <Database size={22} />
              <span>PostgreSQL</span>
              <strong>
                {health.database.ready ? "정상 연결" : "연결 확인 필요"}
              </strong>
              <small>
                {health.database.latency_ms}ms · 연결{" "}
                {health.database.connections}/{health.database.max_connections}
              </small>
            </article>
            <article className="panel padded">
              <Activity size={22} />
              <span>처리한 요청</span>
              <strong>{metric(health.requests)}</strong>
              <small>오류 응답 {metric(health.errors)}건</small>
            </article>
            <article className="panel padded">
              <ShieldCheck size={22} />
              <span>프로세스 메모리</span>
              <strong>
                {(health.process.heap_bytes / 1024 / 1024).toFixed(1)} MB
              </strong>
              <small>
                Go 동시 작업 {metric(health.process.goroutines)}개 · v
                {health.version}
              </small>
            </article>
          </div>
          <div className="panel padded operations-health-status">
            <h2>응답과 추적 전송</h2>
            <div className="actions">
              {Object.entries(health.responses).map(([kind, count]) => (
                <Badge key={kind}>
                  {kind} · {metric(count)}
                </Badge>
              ))}
              <Badge tone={health.telemetry.enabled ? "green" : "amber"}>
                Trace {health.telemetry.enabled ? "전송 활성" : "전송 비활성"}
              </Badge>
            </div>
            <p className="muted">
              확인 시각 {datetime(health.checked_at)} · 오류 큐에서 저장하지
              못한 요청 {health.telemetry.dropped_errors}건
            </p>
            {health.telemetry.configuration_error && (
              <div className="notice warning">
                {health.telemetry.configuration_error}
              </div>
            )}
          </div>
          <div className="panel padded">
            <div className="operations-card-title">
              <h2>최근 오류 응답</h2>
              <Badge>최신 {events.length}건 · 최대 200건 표시</Badge>
            </div>
            <p className="muted">
              본문·사용자·실제 문서 주소를 저장하지 않습니다. 요청 ID를
              감사로그·서버 진단과 함께 확인하세요. 권한 거부 등 4xx도 포함하며
              화면에 표시된 오류가 곧 서비스 장애를 뜻하지는 않습니다.
            </p>
            {events.length === 0 ? (
              <Empty
                title="수집된 오류 응답이 없습니다"
                text="오류 수집 설정과 실제 요청 결과에 따라 기록됩니다."
              />
            ) : (
              <div className="operations-table">
                <table>
                  <thead>
                    <tr>
                      <th>시각</th>
                      <th>요청 경로 템플릿</th>
                      <th>상태</th>
                      <th>응답 시간</th>
                      <th>요청 ID</th>
                    </tr>
                  </thead>
                  <tbody>
                    {events.map((e) => (
                      <tr key={e.request_id}>
                        <td>{datetime(e.created_at)}</td>
                        <td>
                          <code>{e.route}</code>
                          {e.kind === "panic" && (
                            <Badge tone="amber">서버 예외</Badge>
                          )}
                        </td>
                        <td>
                          <Badge tone={e.status >= 500 ? "amber" : ""}>
                            {e.status}
                          </Badge>
                        </td>
                        <td>{e.duration_ms}ms</td>
                        <td>
                          <code>{e.request_id}</code>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </>
      )}
    </section>
  );
}
export function TelemetryPanel() {
  const { notify } = useApp(),
    mounted = useOperationMounted();
  const [snapshot, setSnapshot] = useState<Config | null>(null),
    [patch, setPatch] = useState<Partial<Config>>({}),
    [clearToken, setClearToken] = useState(false),
    [status, setStatus] = useState<Telemetry | null>(null),
    [error, setError] = useState<unknown>(null),
    [busy, setBusy] = useState(false),
    [diagnostic, setDiagnostic] = useState("");
  const dirty = Object.keys(patch).length > 0 || clearToken;
  useOperationsGuard(dirty);
  const value = (key: Key) =>
    patch[key] ?? snapshot?.[key] ?? configDefaults[key];
  function change(key: Key, v: string | number | boolean) {
    setPatch((old) => ({ ...old, [key]: v }));
    setDiagnostic("");
  }
  async function load() {
    try {
      const data = await api<Config>("/admin/settings");
      if (mounted.current) {
        setSnapshot({ ...data, otel_auth_token: "" });
        setPatch({});
        setClearToken(false);
        setError(null);
      }
    } catch (e) {
      if (mounted.current) setError(e);
    }
  }
  async function loadStatus() {
    try {
      const data = await api<Telemetry>("/admin/operations/telemetry");
      if (mounted.current) setStatus(data);
    } catch (e) {
      if (mounted.current) setError(e);
    }
  }
  useEffect(() => {
    void load();
    void loadStatus();
    const timer = setInterval(() => {
      if (document.visibilityState === "visible") void loadStatus();
    }, 5000);
    return () => clearInterval(timer);
  }, []);
  async function save() {
    setBusy(true);
    setError(null);
    setDiagnostic("");
    try {
      const data = await api<Config>("/admin/settings", "PUT", {
        ...patch,
        ...(clearToken ? { otel_auth_token_clear: true } : {}),
        expected_settings_revision: snapshot?.settings_revision,
      });
      if (!mounted.current) return;
      setSnapshot({ ...data, otel_auth_token: "" });
      setPatch({});
      setClearToken(false);
      notify("운영 진단 설정을 저장했습니다. 최대 1초 후 실행에 반영됩니다.");
      void loadStatus();
    } catch (e) {
      if (mounted.current) setError(e);
    } finally {
      if (mounted.current) setBusy(false);
    }
  }
  async function test() {
    setBusy(true);
    setError(null);
    setDiagnostic("");
    try {
      const data = await api<{ message: string }>(
        "/admin/operations/telemetry/test",
        "POST",
        {},
      );
      if (mounted.current) setDiagnostic(data.message);
    } catch (e) {
      if (mounted.current) setError(e);
    } finally {
      if (mounted.current) setBusy(false);
    }
  }
  return (
    <section>
      <div className="notice">
        <ShieldCheck size={22} />
        <span>
          기본적으로 외부 전송하지 않습니다. 관리자가 지정한 OTLP HTTP/protobuf
          수신 주소로 메서드·경로 템플릿·요청 ID·상태·시간·서비스 버전만
          보냅니다. 문서·질문·계정·쿠키·주소 쿼리는 수집하지 않습니다.
        </span>
      </div>
      <ErrorBox error={error} />
      {!snapshot ? (
        <Loading />
      ) : (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            void save();
          }}
          className="operations-telemetry-grid"
        >
          <div className="panel padded">
            <h2>Trace 수신 연결</h2>
            <div className="form-grid">
              <Field label="OpenTelemetry 전송">
                <select
                  value={value("otel_enabled") ? "on" : "off"}
                  disabled={busy}
                  onChange={(e) =>
                    change("otel_enabled", e.target.value === "on")
                  }
                >
                  <option value="off">비활성 · 기본값</option>
                  <option value="on">활성 · 저장된 주소로 전송</option>
                </select>
              </Field>
              <Field
                label="Trace 수집 비율"
                hint="0은 진단 버튼을 제외한 자동 Trace를 수집하지 않습니다."
              >
                <select
                  value={String(value("otel_sample_rate"))}
                  disabled={busy}
                  onChange={(e) =>
                    change("otel_sample_rate", Number(e.target.value))
                  }
                >
                  {[
                    0,
                    0.01,
                    0.05,
                    0.1,
                    0.25,
                    0.5,
                    1,
                    ...(![0, 0.01, 0.05, 0.1, 0.25, 0.5, 1].includes(
                      Number(value("otel_sample_rate")),
                    )
                      ? [Number(value("otel_sample_rate"))]
                      : []),
                  ].map((n) => (
                    <option key={n} value={n}>
                      {n * 100}%
                    </option>
                  ))}
                </select>
              </Field>
            </div>
            <Field
              label="OTLP 수신 주소"
              hint="전체 traces 주소를 입력하세요. 예: https://otel.intranet:4318/v1/traces. 쿼리·계정정보·리다이렉트는 허용하지 않습니다."
            >
              <input
                type="url"
                value={String(value("otel_endpoint"))}
                maxLength={2048}
                disabled={busy}
                onChange={(e) => change("otel_endpoint", e.target.value)}
                placeholder="https://otel.intranet:4318/v1/traces"
              />
            </Field>
            <Field label="HTTP 평문 전송 허용">
              <select
                value={value("otel_allow_http") ? "on" : "off"}
                disabled={busy}
                onChange={(e) =>
                  change("otel_allow_http", e.target.value === "on")
                }
              >
                <option value="off">허용하지 않음 · HTTPS 사용</option>
                <option value="on">내부망 HTTP 전송을 명시적으로 허용</option>
              </select>
            </Field>
            {String(value("otel_endpoint")).startsWith("http:") && (
              <div className="notice warning">
                HTTP는 Trace와 인증 토큰을 암호화하지 않습니다. 신뢰할 수 있는
                내부망에서만 사용하고 가능하면 HTTPS를 구성하세요.
              </div>
            )}
            <Field
              label="Bearer 인증 토큰"
              hint={
                snapshot.otel_auth_token_configured
                  ? "암호화된 토큰이 등록되어 있습니다. 비우면 기존 토큰을 유지합니다."
                  : "선택 사항 · 암호화해 저장하며 다시 표시하지 않습니다."
              }
            >
              <input
                type="password"
                autoComplete="new-password"
                value={String(patch.otel_auth_token || "")}
                maxLength={8192}
                disabled={busy || clearToken}
                onChange={(e) => change("otel_auth_token", e.target.value)}
                placeholder={
                  snapshot.otel_auth_token_configured
                    ? "등록된 토큰 유지"
                    : "토큰 미등록"
                }
              />
            </Field>
            {snapshot.otel_auth_token_configured && (
              <label className="operations-check">
                <input
                  type="checkbox"
                  checked={clearToken}
                  disabled={busy}
                  onChange={(e) => {
                    setClearToken(e.target.checked);
                    if (e.target.checked)
                      setPatch((old) => {
                        const next = { ...old };
                        delete next.otel_auth_token;
                        return next;
                      });
                  }}
                />
                <KeyRound size={16} />
                저장 시 기존 토큰 삭제
              </label>
            )}
            <Field
              label="사내 CA 인증서 (PEM)"
              hint="기본 시스템 CA에 추가합니다. TLS 인증서 검증은 끌 수 없습니다."
            >
              <textarea
                rows={5}
                className="code-input"
                value={String(value("otel_ca_pem"))}
                maxLength={65536}
                disabled={busy}
                onChange={(e) => change("otel_ca_pem", e.target.value)}
                placeholder="-----BEGIN CERTIFICATE-----"
              />
            </Field>
            <Field label="OTLP 요청 제한 시간 (초)">
              <input
                type="number"
                min={1}
                max={30}
                step={1}
                value={Number(value("otel_timeout_seconds"))}
                disabled={busy}
                onChange={(e) =>
                  change("otel_timeout_seconds", Number(e.target.value))
                }
              />
            </Field>
          </div>
          <aside className="operations-settings-aside">
            <section className="panel padded">
              <h2>오류 진단 기록</h2>
              <Field label="오류 응답 수집">
                <select
                  value={value("operations_errors_enabled") ? "on" : "off"}
                  disabled={busy}
                  onChange={(e) =>
                    change("operations_errors_enabled", e.target.value === "on")
                  }
                >
                  <option value="on">활성 · 제한된 요청 메타데이터</option>
                  <option value="off">비활성 · 새 기록 수집 중지</option>
                </select>
              </Field>
              <Field
                label="오류 기록 보존 기간 (일)"
                hint="1~90일 · 최대 50,000건 · 오래된 기록은 매분 정리합니다."
              >
                <input
                  type="number"
                  min={1}
                  max={90}
                  step={1}
                  value={Number(value("operations_retention_days"))}
                  disabled={busy}
                  onChange={(e) =>
                    change("operations_retention_days", Number(e.target.value))
                  }
                />
              </Field>
              <p className="muted">
                오류 기록은 운영 진단용이며 감사로그·백업 원본과 구분합니다.
                기록을 끄더라도 기존 보존 기간에 따라 정리합니다.
              </p>
            </section>
            <section className="panel padded">
              <h2>실제 전송 상태</h2>
              {status ? (
                <>
                  <Badge tone={status.enabled ? "green" : "amber"}>
                    {status.enabled ? "전송 활성" : "전송 비활성"}
                  </Badge>
                  <dl className="operations-stats">
                    <dt>전송 성공</dt>
                    <dd>{metric(status.sent_spans)}개</dd>
                    <dt>전송 실패</dt>
                    <dd>{metric(status.failed_spans)}개</dd>
                    <dt>폐기한 Trace</dt>
                    <dd>{metric(status.dropped_spans)}개</dd>
                    <dt>전송 대기</dt>
                    <dd>{metric(status.queued_spans || 0)}개</dd>
                  </dl>
                  {status.last_sent_at && (
                    <p className="muted">
                      마지막 성공 {datetime(status.last_sent_at)}
                    </p>
                  )}
                  <p>
                    {status.configuration_error ||
                      status.last_result ||
                      "아직 전송 결과가 없습니다."}
                  </p>
                </>
              ) : (
                <Loading />
              )}
              <p className="muted">
                대기 큐는 2,048개로 제한합니다. 주소·정책 변경과 종료 시 대기
                내용을 폐기합니다. 실패한 전송은 자동 재시도하지 않습니다.
              </p>
              <Button
                type="button"
                variant="secondary"
                disabled={busy || dirty || !snapshot.otel_endpoint}
                onClick={() => void test()}
              >
                <TestTube2 size={16} />
                저장된 연결 테스트
              </Button>
              {dirty && (
                <small className="muted">
                  변경사항을 먼저 저장하면 테스트할 수 있습니다.
                </small>
              )}
              {diagnostic && (
                <div className="notice" role="status">
                  {diagnostic}
                </div>
              )}
            </section>
            <Link
              className="button secondary"
              to="/admin/settings?tab=history"
              onClick={(e) => {
                if (!leaveOperations()) e.preventDefault();
              }}
            >
              <History size={16} />
              서비스 설정 이력
            </Link>
          </aside>
          <footer className="operations-save">
            <span className="muted">
              {dirty
                ? "저장되지 않은 설정이 있습니다."
                : "현재 저장된 설정입니다."}
            </span>
            <div className="actions">
              <Button
                type="button"
                variant="secondary"
                disabled={busy}
                onClick={() => {
                  if (!dirty || leaveOperations()) void load();
                }}
              >
                <RefreshCw size={16} />
                다시 불러오기
              </Button>
              <Button type="submit" disabled={busy || !dirty}>
                <Save size={17} />
                {busy ? "처리 중…" : "운영 진단 설정 저장"}
              </Button>
            </div>
          </footer>
        </form>
      )}
    </section>
  );
}
