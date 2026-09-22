import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { Activity, RefreshCw, Settings, Trash2 } from "lucide-react";
import { api } from "../api";
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
import { actorNames, formatTime, healthNames, stateNames } from "./types";
import type { Expected, Observation, StatusState } from "./types";
import "./system-status.css";

const emptyExpected: Expected = {
  name: "",
  environment: "",
  version: "",
  deployment: "",
  note: "",
};
const emptyObservation: Observation = {
  version: "",
  deployment: "",
  health: "unknown",
  note: "",
};
const fingerprint = (s: StatusState | null) =>
  s
    ? `${s.current_document_version}:${s.policy.revision}:${s.card?.revision}:${s.card?.verification_epoch}:${s.card?.report_revision}`
    : "";
export default function SystemStatusPage() {
  const { user, workspace, documents } = useApp();
  const [params, setParams] = useSearchParams();
  const documentId = params.get("document") || "";
  const scope = `${user?.id}:${workspace?.id}:${documentId}`;
  const scopeRef = useRef(scope);
  scopeRef.current = scope;
  const [state, setState] = useState<StatusState | null>(null),
    [loadedScope, setLoadedScope] = useState(""),
    [error, setError] = useState(""),
    [notice, setNotice] = useState(""),
    [loading, setLoading] = useState(false),
    [busy, setBusy] = useState(false);
  const [dialog, setDialog] = useState<"edit" | "report" | "delete" | null>(
      null,
    ),
    [confirmed, setConfirmed] = useState(false),
    [baseline, setBaseline] = useState("");
  const [expected, setExpected] = useState<Expected>(emptyExpected),
    [owner, setOwner] = useState(""),
    [reporters, setReporters] = useState<string[]>([]),
    [ttl, setTTL] = useState(300);
  const [observation, setObservation] = useState<Observation>(emptyObservation),
    [observedAt, setObservedAt] = useState("");
  const seq = useRef(0),
    requestID = useRef("");
  const active = loadedScope === scope ? state : null;
  const activeRef = useRef<StatusState | null>(active);
  activeRef.current = active;
  const load = useCallback(
    async (quiet = false) => {
      if (!user?.id || !workspace?.id || !documentId) return;
      const token = scope,
        current = ++seq.current;
      if (!quiet) setLoading(true);
      try {
        const value = await api<StatusState>(
          `/documents/${documentId}/system-status`,
        );
        if (token !== scopeRef.current || current !== seq.current) return;
        setState(value);
        setLoadedScope(token);
        if (!quiet) setError("");
      } catch (e) {
        if (token !== scopeRef.current || current !== seq.current) return;
        setState(null);
        setLoadedScope("");
        setDialog(null);
        setConfirmed(false);
        setError(
          e instanceof Error ? e.message : "운영 카드를 불러오지 못했습니다",
        );
      } finally {
        if (token === scopeRef.current && current === seq.current)
          setLoading(false);
      }
    },
    [user?.id, workspace?.id, documentId, scope],
  );
  useEffect(() => {
    setState(null);
    setLoadedScope("");
    setDialog(null);
    setConfirmed(false);
    setNotice("");
    setError("");
    setBusy(false);
    void load();
    return () => {
      seq.current++;
    };
  }, [load]);
  useEffect(() => {
    const timer = setInterval(() => {
      if (document.visibilityState === "visible" && !busy) void load(true);
    }, 3000);
    return () => clearInterval(timer);
  }, [load, busy]);
  useEffect(() => {
    if (dialog && baseline !== fingerprint(active)) {
      setConfirmed(false);
    }
  }, [active, dialog, baseline]);
  const open = (kind: "edit" | "report" | "delete") => {
    if (!active) return;
    setDialog(kind);
    setConfirmed(false);
    setBaseline(fingerprint(active));
    setError("");
    if (kind === "edit") {
      setExpected(
        active.card?.expected || {
          ...emptyExpected,
          name: documents.find((d) => d.id === documentId)?.title || "",
        },
      );
      setOwner(active.card?.owner_id || user?.id || "");
      setReporters(active.card?.reporter_ids || []);
      setTTL(
        active.card?.ttl_seconds ||
          Math.min(300, active.policy.max_ttl_seconds),
      );
    }
    if (kind === "report") {
      setObservation(emptyObservation);
      setObservedAt(new Date().toISOString());
      requestID.current = crypto.randomUUID();
    }
  };
  const mutate = async () => {
    const current = activeRef.current;
    if (!current || !confirmed || baseline !== fingerprint(current) || busy)
      return;
    const token = scopeRef.current;
    setBusy(true);
    setError("");
    try {
      if (dialog === "edit") {
        await api(`/documents/${documentId}/system-status`, "PUT", {
          revision: current.card?.revision || 0,
          document_version: current.current_document_version,
          owner_id: owner,
          reporter_ids: reporters,
          ttl_seconds: ttl,
          expected,
          confirm: true,
        });
      } else if (dialog === "report" && current.card) {
        await api(`/documents/${documentId}/system-status/reports`, "POST", {
          request_id: requestID.current,
          card_revision: current.card.revision,
          report_revision: current.card.report_revision,
          verification_epoch: current.card.verification_epoch,
          document_version: current.current_document_version,
          observed_at: observedAt,
          observation,
          confirm: true,
        });
      } else if (dialog === "delete" && current.card) {
        await api(`/documents/${documentId}/system-status`, "DELETE", {
          revision: current.card.revision,
          document_version: current.current_document_version,
          confirm: true,
        });
      } else return;
      if (token !== scopeRef.current) return;
      setNotice(
        dialog === "delete"
          ? "운영 카드와 관측 이력을 삭제했습니다. 원문은 유지됩니다. 백업이 없다면 카드 이력을 복구할 수 없습니다."
          : dialog === "report"
            ? "관측 보고를 저장했습니다. 서버 직접 검증으로 인증한 것이 아닙니다."
            : "기대값을 저장했습니다. 새 기준에 대한 관측 보고가 필요합니다.",
      );
      setDialog(null);
      setConfirmed(false);
      await load(true);
    } catch (e) {
      if (token !== scopeRef.current) return;
      setConfirmed(false);
      await load(true);
      if (token === scopeRef.current)
        setError(e instanceof Error ? e.message : "요청을 완료하지 못했습니다");
    } finally {
      if (token === scopeRef.current) setBusy(false);
    }
  };
  const setExpectedField = (key: keyof Expected, value: string) => {
    setExpected((old) => ({ ...old, [key]: value }));
    setConfirmed(false);
  };
  const setObservationField = (key: keyof Observation, value: string) => {
    setObservation((old) => ({ ...old, [key]: value }));
    setConfirmed(false);
  };
  const c = active?.card,
    report = active?.latest_report;
  return (
    <section className="system-status-page">
      <PageHeading
        eyebrow="KNOWLEDGE OPERATIONS"
        title="시스템 운영 현황"
        description="문서의 기대값과 명시된 보고 주체가 전달한 관측을 비교합니다."
        actions={
          <>
            <Button
              onClick={() => void load()}
              disabled={!documentId || loading || busy}
            >
              <RefreshCw size={16} />
              새로고침
            </Button>
            {user?.role === "admin" && (
              <Link className="button" to="/admin/system-status">
                <Settings size={16} />
                보고 정책
              </Link>
            )}
          </>
        }
      />
      <div className="status-authority-notice" role="note">
        <Activity size={20} />
        <p>
          madi가 대상 시스템의 배포나 정상 동작을 직접 검증하지 않습니다. 기대값
          등록과 수동/API 관측 보고, Runbook 실행, 커넥터 수집 사실은 서로
          구분합니다.
        </p>
      </div>
      <Field label="기준 문서">
        <select
          value={documentId}
          disabled={busy}
          onChange={(e) =>
            setParams(e.target.value ? { document: e.target.value } : {})
          }
        >
          <option value="">현재 열람 가능한 문서 선택</option>
          {documents.map((d) => (
            <option key={d.id} value={d.id}>
              {d.title}
            </option>
          ))}
        </select>
      </Field>
      <ErrorBox error={error} />
      {notice && (
        <p className="notice" role="status">
          {notice}
        </p>
      )}
      {!documentId ? (
        <Empty
          title="문서에 운영 맥락을 연결하세요"
          text="기준 문서를 선택하면 기대 버전·배포 식별자·담당자와 관측 보고를 관리할 수 있습니다."
        />
      ) : loading && !active ? (
        <Loading />
      ) : !active ? (
        <Button onClick={() => void load()}>현재 권한으로 다시 확인</Button>
      ) : (
        <>
          {!active.policy.enabled && (
            <p className="notice">
              관리자가 운영 보고를 활성화하지 않았습니다. 기존 이력은 과거
              자료이며 현재 상태로 확인하지 않습니다.
            </p>
          )}
          <div
            className="status-state"
            data-status-state={active.summary.state}
          >
            <Badge>
              {stateNames[active.summary.state] || active.summary.state}
            </Badge>
            <span>
              문서 저장본 v{active.current_document_version}
              {c &&
                ` · 기준 revision ${c.revision} · 보고 ${c.report_revision}`}
            </span>
            <Link to={`/app/documents/${documentId}?mode=read`}>
              문서 원문 보기
            </Link>
          </div>
          <div className="status-actions">
            {active.can_manage && (
              <Button
                variant="primary"
                disabled={!active.policy.enabled || busy}
                onClick={() => open("edit")}
              >
                {c ? "기대값·보고 주체 수정" : "운영 카드 등록"}
              </Button>
            )}
            <Button
              disabled={!active.can_report || busy}
              onClick={() => open("report")}
            >
              실제 관측 보고
            </Button>
            {c && active.can_manage && (
              <Button onClick={() => open("delete")} disabled={busy}>
                <Trash2 size={16} />
                카드 삭제
              </Button>
            )}
          </div>
          {!c ? (
            <Empty
              title="등록 기대값만으로 관측을 만들지 않습니다"
              text="기대 버전이나 배포 식별자를 먼저 등록하고, 실제 관측이 있을 때 별도 보고하세요."
            />
          ) : (
            <>
              <div className="status-comparison">
                <article>
                  <h2>등록된 기대값</h2>
                  <dl>
                    <dt>시스템</dt>
                    <dd>{c.expected.name}</dd>
                    <dt>환경</dt>
                    <dd>{c.expected.environment || "미등록"}</dd>
                    <dt>기대 버전</dt>
                    <dd>{c.expected.version || "비교 안 함"}</dd>
                    <dt>기대 배포</dt>
                    <dd>{c.expected.deployment || "비교 안 함"}</dd>
                    <dt>담당</dt>
                    <dd>
                      {active.reporter_options.find((u) => u.id === c.owner_id)
                        ?.name || c.owner_id}
                    </dd>
                    <dt>기준 원문</dt>
                    <dd>v{c.document_version}</dd>
                    <dt>유효 TTL</dt>
                    <dd>
                      {Math.min(c.ttl_seconds, active.policy.max_ttl_seconds)}초
                    </dd>
                  </dl>
                  {c.expected.note && (
                    <p className="status-note">{c.expected.note}</p>
                  )}
                </article>
                <article>
                  <h2>최근에 수신한 관측</h2>
                  {!report ? (
                    <p className="muted">실제 관측 보고가 없습니다.</p>
                  ) : (
                    <>
                      <Badge>
                        {actorNames[report.actor_kind] || report.actor_kind}
                      </Badge>
                      <dl>
                        <dt>보고한 버전</dt>
                        <dd>{report.observation.version || "미확인"}</dd>
                        <dt>보고한 배포</dt>
                        <dd>{report.observation.deployment || "미확인"}</dd>
                        <dt>보고한 상태</dt>
                        <dd>{healthNames[report.observation.health]}</dd>
                        <dt>관측한 시각</dt>
                        <dd>{formatTime(report.observed_at)}</dd>
                        <dt>서버 수신</dt>
                        <dd>{formatTime(report.received_at)}</dd>
                        <dt>관측 유효 만료</dt>
                        <dd>{formatTime(active.summary.expires_at)}</dd>
                        <dt>보고 주체 ID</dt>
                        <dd>
                          <code>{report.actor_id}</code>
                        </dd>
                      </dl>
                      {report.observation.note && (
                        <p className="status-note">{report.observation.note}</p>
                      )}
                      <p className="muted">
                        위 관측은 저장 당시 기준 revision {report.card_revision}
                        에 관한 주장입니다. 만료·기준 변경 후에는 현재 배포
                        상태로 보지 마세요.
                      </p>
                    </>
                  )}
                </article>
              </div>
              {active.summary.drift_fields?.length !== 0 &&
                active.summary.drift_fields && (
                  <p className="notice">
                    보고와 기대값이 다른 항목:{" "}
                    {active.summary.drift_fields
                      .map((f) => (f === "version" ? "버전" : "배포 식별자"))
                      .join(", ")}
                    . 자동 배포나 문서 수정은 수행하지 않습니다.
                  </p>
                )}
              <details className="status-detail">
                <summary>서로 구분되는 실행·수집 근거</summary>
                <p>{active.context.notice}</p>
                {active.context.runbook ? (
                  <p>
                    최근 Runbook 실행:{" "}
                    <Link
                      to={`/app/runbook/executions/${active.context.runbook.id}`}
                    >
                      {active.context.runbook.status}
                    </Link>{" "}
                    · 종료 {formatTime(active.context.runbook.completed_at)}
                  </p>
                ) : (
                  <p className="muted">
                    현재 열람 가능한 연결 실행 근거가 없습니다.
                  </p>
                )}
                {active.context.connector_records.map((item, i) => (
                  <p key={`${item.connector_id}:${i}`}>
                    커넥터 수집: {item.kind} · 원문 v{item.document_version} ·
                    수집 시각 {formatTime(item.last_seen_at)} · 내용 해시{" "}
                    <code>{item.content_checksum}</code>
                  </p>
                ))}
              </details>
              <details className="status-detail">
                <summary>관측 이력 · 최근 {active.reports.length}건</summary>
                <p className="muted">
                  기준이 다른 이력은 현재 기대값과 직접 비교하지 않습니다. 보존
                  정책 {active.policy.retention_days}일, 목록 최대 50건입니다.
                </p>
                {active.reports.map((item) => (
                  <article className="status-history" key={item.id}>
                    <strong>
                      보고 {item.revision} · 기준 {item.card_revision} ·{" "}
                      {actorNames[item.actor_kind]}
                    </strong>
                    <p>
                      관측 {formatTime(item.observed_at)} / 수신{" "}
                      {formatTime(item.received_at)}
                    </p>
                    <p>
                      {item.observation.version || "버전 미확인"} ·{" "}
                      {item.observation.deployment || "배포 미확인"} ·{" "}
                      {healthNames[item.observation.health]}
                    </p>
                    <p className="muted">
                      당시 기대: {item.expected_at_report.version || "없음"} /{" "}
                      {item.expected_at_report.deployment || "없음"}
                    </p>
                  </article>
                ))}
              </details>
              <details className="status-detail">
                <summary>API·MCP 보고 연결</summary>
                <p>
                  현재 문서의 작성·읽기 권한과 카드의 보고 주체 목록에 포함된
                  계정만 보고할 수 있습니다. 개인 키 페이지에서 워크스페이스
                  범위 document:read + document:write 키를 발급하세요. 서비스
                  계정은 관리자가 발급하며 로그인할 수 없습니다.
                </p>
                <p>
                  <code>GET /api/v1/documents/{documentId}/system-status</code>
                  로 최신 기준을 읽은 뒤{" "}
                  <code>
                    POST /api/v1/documents/{documentId}/system-status/reports
                  </code>
                  에 관측값을 전달합니다. MCP: <code>get_system_status</code> /{" "}
                  <code>report_system_status</code>. 문서나 배포를 변경하는
                  도구가 아닙니다.
                </p>
                <pre>
                  {JSON.stringify(
                    {
                      request_id:
                        "매 보고마다 새 UUID · 재전송은 동일 UUID/본문",
                      card_revision: c.revision,
                      report_revision: c.report_revision,
                      verification_epoch: c.verification_epoch,
                      document_version: active.current_document_version,
                      observed_at: "실제로 관측한 RFC3339 시각",
                      observation: {
                        version: "실제로 관측한 버전",
                        deployment: "실제로 관측한 배포",
                        health: "unknown",
                        note: "",
                      },
                      confirm: true,
                    },
                    null,
                    2,
                  )}
                </pre>
              </details>
            </>
          )}
        </>
      )}
      <Modal
        open={dialog !== null}
        onOpenChange={(v) => {
          if (!v && !busy) {
            setDialog(null);
            setConfirmed(false);
          }
        }}
        title={
          dialog === "edit"
            ? "운영 기대값과 보고 주체"
            : dialog === "report"
              ? "실제 관측 보고"
              : "운영 카드 삭제"
        }
        description={
          dialog === "report"
            ? "직접 관측한 사실만 보고하세요. 이 요청은 서버의 독립 검증을 뜻하지 않습니다."
            : "현재 문서의 모든 열람자가 카드와 관측 이력을 볼 수 있습니다."
        }
        wide
      >
        <ErrorBox error={error} />
        {dialog && baseline !== fingerprint(active) && (
          <p className="notice">
            원문·카드·정책 또는 최신 보고가 변경되었습니다. 창을 닫고 현재
            내용을 다시 확인하세요.
          </p>
        )}
        {dialog === "edit" && active && (
          <div className="status-form">
            <Field label="시스템 이름">
              <input
                value={expected.name}
                maxLength={200}
                onChange={(e) => setExpectedField("name", e.target.value)}
              />
            </Field>
            <Field label="운영 환경">
              <input
                value={expected.environment}
                maxLength={200}
                onChange={(e) =>
                  setExpectedField("environment", e.target.value)
                }
              />
            </Field>
            <Field label="기대 시스템 버전">
              <input
                value={expected.version}
                maxLength={500}
                onChange={(e) => setExpectedField("version", e.target.value)}
              />
            </Field>
            <Field label="기대 배포 식별자">
              <input
                value={expected.deployment}
                maxLength={500}
                onChange={(e) => setExpectedField("deployment", e.target.value)}
              />
            </Field>
            <Field label="담당 사용자">
              <select
                value={owner}
                onChange={(e) => {
                  setOwner(e.target.value);
                  setConfirmed(false);
                }}
              >
                <option value="">담당자 선택</option>
                {active.reporter_options
                  .filter((u) => u.kind === "user")
                  .map((u) => (
                    <option key={u.id} value={u.id}>
                      {u.name}
                    </option>
                  ))}
              </select>
            </Field>
            <Field label="관측 유효 TTL · 초">
              <input
                type="number"
                min={60}
                max={active.policy.max_ttl_seconds}
                value={ttl}
                onChange={(e) => {
                  setTTL(Number(e.target.value));
                  setConfirmed(false);
                }}
              />
            </Field>
            <Field label="보고 주체 · 여러 계정은 Ctrl/⌘로 선택">
              <select
                multiple
                size={Math.min(6, Math.max(2, active.reporter_options.length))}
                value={reporters}
                onChange={(e) => {
                  setReporters(
                    [...e.target.selectedOptions].map((o) => o.value),
                  );
                  setConfirmed(false);
                }}
              >
                {active.reporter_options.map((u) => (
                  <option key={u.id} value={u.id}>
                    {u.name} · {u.kind === "service" ? "서비스 계정" : "사용자"}
                  </option>
                ))}
              </select>
            </Field>
            {active.reporter_options_truncated && (
              <p className="muted">
                현재 선택 목록은 100명까지입니다. API로 현재 권한이 있는 정확한
                계정 ID를 지정할 수 있습니다.
              </p>
            )}
            <Field label="기대값 설명">
              <textarea
                rows={3}
                maxLength={4000}
                value={expected.note}
                onChange={(e) => setExpectedField("note", e.target.value)}
              />
            </Field>
          </div>
        )}
        {dialog === "report" && (
          <div className="status-form">
            <Field label="실제 관측 시각 · RFC3339">
              <input
                value={observedAt}
                onChange={(e) => {
                  setObservedAt(e.target.value);
                  setConfirmed(false);
                }}
              />
            </Field>
            <Field label="관측한 시스템 버전">
              <input
                value={observation.version}
                maxLength={500}
                onChange={(e) => setObservationField("version", e.target.value)}
              />
            </Field>
            <Field label="관측한 배포 식별자">
              <input
                value={observation.deployment}
                maxLength={500}
                onChange={(e) =>
                  setObservationField("deployment", e.target.value)
                }
              />
            </Field>
            <Field label="관측한 상태">
              <select
                value={observation.health}
                onChange={(e) => setObservationField("health", e.target.value)}
              >
                {Object.entries(healthNames).map(([key, label]) => (
                  <option key={key} value={key}>
                    {label}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="관측 근거 메모">
              <textarea
                value={observation.note}
                rows={3}
                maxLength={4000}
                onChange={(e) => setObservationField("note", e.target.value)}
              />
            </Field>
            <p className="muted">
              현재 정책은 {active?.policy.max_observation_age_seconds}초 이내의
              과거 관측과 최대 60초 미래 오차만 허용합니다. 재전송으로 수신
              시각이나 TTL을 연장하지 않습니다.
            </p>
          </div>
        )}
        {dialog === "delete" && (
          <p>
            이 카드와 관측 이력을 삭제합니다. 문서 원문과 실제 시스템은
            유지됩니다. 백업이 없다면 카드와 보고는 복구할 수 없습니다.
          </p>
        )}
        <label className="checkbox-row">
          <input
            type="checkbox"
            checked={confirmed}
            disabled={busy || baseline !== fingerprint(active)}
            onChange={(e) => setConfirmed(e.target.checked)}
          />
          {dialog === "report"
            ? "실제로 관측한 주장임을 확인하며, 현재 문서 열람자에게 이 보고를 공유합니다."
            : dialog === "delete"
              ? "카드와 관측 이력의 삭제 범위를 확인했습니다."
              : "기대값이 실제 배포 확인은 아님을 이해하며 문서 열람자에게 공유합니다."}
        </label>
        <div className="modal-actions">
          <Button
            disabled={busy}
            onClick={() => {
              setDialog(null);
              setConfirmed(false);
            }}
          >
            취소
          </Button>
          <Button
            variant="primary"
            disabled={busy || !confirmed || baseline !== fingerprint(active)}
            onClick={() => void mutate()}
          >
            {busy
              ? "저장 중…"
              : dialog === "report"
                ? "관측 보고 저장"
                : dialog === "delete"
                  ? "카드와 이력 삭제"
                  : "운영 기준 저장"}
          </Button>
        </div>
      </Modal>
    </section>
  );
}
