import { useCallback, useEffect, useRef, useState } from "react";
import "./channel-settings.css";
import { Link, useSearchParams } from "react-router-dom";
import {
  Copy,
  Inbox,
  KeyRound,
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
const states: Record<string, string> = {
  pending: "대기",
  completed: "수집 완료",
  rejected: "형식·크기 거부",
  cancelled: "취소",
  running: "수집 중",
  failed: "작업 실패",
  succeeded: "검사 완료",
};
export function InboundCapturePolicyPage() {
  const { notify } = useApp();
  const [data, setData] = useState<Row | null>(null),
    [hosts, setHosts] = useState(""),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  useEffect(() => {
    let active = true;
    api<Row>("/admin/inbound-capture-settings")
      .then((v) => {
        if (active) {
          setData(v);
          setHosts(v.allowed_hosts.join("\n"));
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
        eyebrow="PRIVATE KNOWLEDGE INTAKE"
        title="외부 수집 정책"
        description="사용자가 구성한 서명 Webhook과 읽기 전용 IMAP 연결의 사용을 허용합니다."
      />
      <ErrorBox error={error} />
      {!data ? (
        <Loading />
      ) : (
        <section className="card">
          <form
            onSubmit={async (e) => {
              e.preventDefault();
              setBusy(true);
              setError("");
              try {
                await api("/admin/inbound-capture-settings", "PUT", {
                  hooks_enabled: data.hooks_enabled,
                  imap_enabled: data.imap_enabled,
                  allowed_hosts: hosts
                    .split("\n")
                    .map((v) => v.trim())
                    .filter(Boolean),
                });
                notify("외부 수집 정책을 저장했습니다");
              } catch (e) {
                setError((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            <label className="check-row">
              <input
                type="checkbox"
                checked={!!data.hooks_enabled}
                onChange={(e) =>
                  setData({ ...data, hooks_enabled: e.target.checked })
                }
              />
              개인 HMAC Webhook 수집 허용
            </label>
            <label className="check-row">
              <input
                type="checkbox"
                checked={!!data.imap_enabled}
                onChange={(e) =>
                  setData({ ...data, imap_enabled: e.target.checked })
                }
              />
              개인 IMAP TLS 수집 허용
            </label>
            <Field label="IMAP 허용 호스트 (줄마다 하나)">
              <textarea
                rows={5}
                value={hosts}
                onChange={(e) => setHosts(e.target.value)}
                placeholder="imap.example.internal"
              />
            </Field>
            <p className="muted">
              포트·경로·와일드카드 없이 정확한 호스트를 입력하세요. 실제
              DNS/IP를 연결 시점에 다시 확인하고 메타데이터·링크 로컬 주소는
              차단합니다.
            </p>
            <Button variant="primary" type="submit" disabled={busy}>
              <Save size={17} />
              수집 정책 저장
            </Button>
          </form>
        </section>
      )}
      <div className="notice">
        <ShieldCheck size={20} />
        <span>
          메일 원문·본문·첨부는 개인 수집함의 비공개 문서로 저장합니다. IMAP은
          TLS EXAMINE과 BODY.PEEK만 사용하며 읽음 표시·이동·삭제를 수행하지
          않습니다. 별도 SMTP 수신 포트는 열지 않습니다.
        </span>
      </div>
      <section className="card">
        <h2>수집 한도와 복원</h2>
        <p>
          원문과 해제 본문·첨부 각각 10MB, 본문 4MB, 첨부 20개, MIME 계층
          10단계와 100파트를 제한합니다. 개인 대기 자료는 100MB 또는
          100건까지이며 Webhook은 채널당 분당 60건입니다.
        </p>
        <p>
          완료·취소·거부 자료의 임시 암호화 원본은 7일 뒤 정리하고 생성된 문서는
          보존합니다. 복원 후 모든 수집 정책과 채널을 끄고 대기 작업을
          취소하므로 재검증 뒤 켜세요.
        </p>
      </section>
    </>
  );
}
export function InboundCapturePage() {
  const { workspace, workspaces, notify } = useApp();
  const [params, setParams] = useSearchParams();
  const [rows, setRows] = useState<Row[]>([]),
    [policy, setPolicy] = useState<Row>({}),
    [history, setHistory] = useState<Row[]>([]),
    [jobs, setJobs] = useState<Row[]>([]),
    [loading, setLoading] = useState(true),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [draft, setDraft] = useState<Row | null>(null),
    [rotation, setRotation] = useState<Row | null>(null),
    [secret, setSecret] = useState<Row | null>(null);
  const generation = useRef(0),
    detailGeneration = useRef(0);
  const active = rows.find((c) => c.id === params.get("channel"));
  const load = useCallback(async () => {
    const run = ++generation.current;
    try {
      const [channels, flags] = await Promise.all([
        api<Row[]>("/capture-channels"),
        api<Row>("/capture-policy"),
      ]);
      if (run === generation.current) {
        setRows(channels);
        setPolicy(flags);
        setError("");
      }
    } catch (e) {
      if (run === generation.current) setError((e as Error).message);
    } finally {
      if (run === generation.current) setLoading(false);
    }
  }, []);
  const loadDetail = useCallback(async () => {
    if (!active) return;
    const run = ++detailGeneration.current;
    const [messages, attempts] = await Promise.all([
      api<Row[]>(`/capture-channels/${active.id}/history`),
      active.kind === "imap"
        ? api<Row[]>(`/capture-channels/${active.id}/jobs`)
        : Promise.resolve([]),
    ]);
    if (run === detailGeneration.current) {
      setHistory(messages);
      setJobs(attempts);
    }
  }, [active?.id, active?.kind]);
  useEffect(() => {
    void load();
    return () => {
      generation.current++;
    };
  }, [load]);
  useEffect(() => {
    setHistory([]);
    setJobs([]);
    void loadDetail().catch((e) => setError(e.message));
    const timer = setInterval(
      () => void loadDetail().catch((e) => setError(e.message)),
      5000,
    );
    return () => {
      detailGeneration.current++;
      clearInterval(timer);
    };
  }, [loadDetail]);
  const newDraft = () => ({
    workspace_id: workspace?.id || "",
    name: "",
    kind: "hmac",
    enabled: false,
    config: {},
    secrets: { username: "", password: "" },
  });
  const config = (key: string, value: any) =>
    draft && setDraft({ ...draft, config: { ...draft.config, [key]: value } });
  const endpoint = (id: string) =>
    `${location.origin}/api/v1/capture-hooks/${id}`;
  return (
    <>
      <PageHeading
        eyebrow="PERSONAL CAPTURE CHANNELS"
        title="내 외부 수집 채널"
        description="내 이메일과 자동화 자료를 비공개 수집함에 모읍니다. 공유는 나중에 직접 결정하세요."
        actions={
          <>
            <Link className="button" to="/app/inbox">
              <Inbox size={17} />
              수집함
            </Link>
            <Button
              variant="primary"
              onClick={() => {
                setDraft(newDraft());
                setError("");
              }}
            >
              <Plus size={17} />
              수집 채널 추가
            </Button>
          </>
        }
      />
      <ErrorBox error={error} />
      <div className="button-row">
        <Badge tone={policy.hooks_enabled ? "success" : ""}>
          Webhook {policy.hooks_enabled ? "관리자 허용" : "관리자 중지"}
        </Badge>
        <Badge tone={policy.imap_enabled ? "success" : ""}>
          IMAP {policy.imap_enabled ? "관리자 허용" : "관리자 중지"}
        </Badge>
        <Button disabled={busy} onClick={() => void load()}>
          <RefreshCw size={17} />
          새로고침
        </Button>
      </div>
      {loading ? (
        <Loading />
      ) : rows.length ? (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>채널</th>
                <th>수집 방식</th>
                <th>대상</th>
                <th>상태</th>
                <th>관리</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((c) => (
                <tr key={c.id}>
                  <td>
                    <strong>{c.name}</strong>
                  </td>
                  <td>{c.kind === "imap" ? "IMAP TLS" : "HMAC Webhook"}</td>
                  <td>
                    {workspaces.find((w) => w.id === c.workspace_id)?.name ||
                      "내 워크스페이스"}{" "}
                    · 비공개
                  </td>
                  <td>
                    <Badge>{c.enabled ? "활성" : "중지"}</Badge>
                  </td>
                  <td>
                    <div className="button-row">
                      <Button onClick={() => setParams({ channel: c.id })}>
                        수집 이력
                      </Button>
                      <Button
                        onClick={() => {
                          setDraft({
                            ...c,
                            config: { ...c.config },
                            secrets: { username: "", password: "" },
                          });
                          setError("");
                        }}
                      >
                        <Settings2 size={16} />
                        설정
                      </Button>
                      {c.kind === "hmac" && (
                        <Button onClick={() => setRotation(c)}>
                          <KeyRound size={16} />
                          비밀 회전
                        </Button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <Empty
          title="외부 수집 채널이 없습니다"
          text="개인 메일함이나 서명 Webhook을 연결하세요."
        />
      )}
      {active && (
        <section className="card">
          <div className="card-header">
            <div>
              <h2>{active.name} · 수집 이력</h2>
              {active.kind === "hmac" ? (
                <p>
                  <code style={{ wordBreak: "break-all" }}>
                    {endpoint(active.id)}
                  </code>
                </p>
              ) : (
                <p>
                  {active.config.host} · {active.config.mailbox} · 마지막 UID{" "}
                  {active.checkpoint.last_uid || "미검사"}
                </p>
              )}
            </div>
            <div className="button-row">
              <Button
                disabled={busy}
                onClick={() =>
                  void loadDetail().catch((e) => setError(e.message))
                }
              >
                <RefreshCw size={16} />
                이력 갱신
              </Button>
              {active.kind === "imap" && (
                <Button
                  disabled={busy || !active.enabled || !policy.imap_enabled}
                  onClick={async () => {
                    setBusy(true);
                    setError("");
                    try {
                      const result = await api<Row>(
                        `/capture-channels/${active.id}/poll`,
                        "POST",
                        {},
                      );
                      notify(
                        result.already_running
                          ? "이미 수집 검사 중입니다"
                          : "메일 수집 검사를 예약했습니다",
                      );
                      await loadDetail();
                    } catch (e) {
                      setError((e as Error).message);
                    } finally {
                      setBusy(false);
                    }
                  }}
                >
                  지금 메일 검사
                </Button>
              )}
            </div>
          </div>
          {history.length ? (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>자료</th>
                    <th>상태</th>
                    <th>시각</th>
                    <th>설명</th>
                  </tr>
                </thead>
                <tbody>
                  {history.map((m) => (
                    <tr key={m.id}>
                      <td>
                        {m.document_id ? (
                          <Link to={`/app/documents/${m.document_id}`}>
                            {m.title || "수집 문서"}
                          </Link>
                        ) : (
                          m.title || "메시지 검사"
                        )}
                      </td>
                      <td>
                        <Badge>
                          {m.status === "pending" && m.job_status === "failed"
                            ? "작업 실패"
                            : states[m.status] || m.status}
                        </Badge>
                        {m.status === "pending" && (
                          <Button
                            disabled={busy}
                            onClick={async () => {
                              setBusy(true);
                              try {
                                await api(
                                  `/capture-channels/${active.id}/messages/${m.id}/cancel`,
                                  "POST",
                                  {},
                                );
                                await loadDetail();
                                notify("수집을 취소했습니다");
                              } catch (e) {
                                setError((e as Error).message);
                              } finally {
                                setBusy(false);
                              }
                            }}
                          >
                            수집 취소
                          </Button>
                        )}
                      </td>
                      <td>{datetime(m.completed_at || m.created_at)}</td>
                      <td>{m.message || "—"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="muted">
              아직 받은 자료가 없습니다. IMAP은 처음 연결할 때 이후 메일부터
              수집하는 것이 기본입니다.
            </p>
          )}
          {jobs.length > 0 && (
            <details>
              <summary>최근 IMAP 검사 작업</summary>
              {jobs.map((j) => (
                <p key={j.id}>
                  {datetime(j.created_at)} · {states[j.status] || j.status} ·{" "}
                  {j.attempts}회 {j.last_error}
                </p>
              ))}
            </details>
          )}
        </section>
      )}
      <div className="notice">
        <ShieldCheck size={20} />
        <span>
          수집은 현재 내 문서 작성 권한으로 처리하며 항상 개인 비공개로
          시작합니다. 메일을 읽음으로 표시하거나 삭제하지 않습니다. 전송 중
          오류가 나면 같은 ID로 재시도하고, 동일 ID의 본문을 바꾸지 마세요.
        </span>
      </div>
      {draft && (
        <Modal
          open
          wide
          title={draft.id ? "수집 채널 설정" : "수집 채널 추가"}
          onOpenChange={() => {
            if (!busy) setDraft(null);
          }}
        >
          <ErrorBox error={error} />
          <form
            onSubmit={async (e) => {
              e.preventDefault();
              setBusy(true);
              setError("");
              try {
                const saved = await api<Row>(
                  "/capture-channels" + (draft.id ? "/" + draft.id : ""),
                  draft.id ? "PUT" : "POST",
                  draft,
                );
                setDraft(null);
                await load();
                setParams({ channel: saved.id });
                if (saved.signing_secret)
                  setSecret({ id: saved.id, value: saved.signing_secret });
                notify("수집 채널을 저장했습니다");
              } catch (e) {
                setError((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            <div className="form-grid">
              <Field label="수집 채널 이름">
                <input
                  required
                  maxLength={120}
                  value={draft.name}
                  onChange={(e) => setDraft({ ...draft, name: e.target.value })}
                />
              </Field>
              <Field label="수집 방식">
                <select
                  disabled={!!draft.id}
                  value={draft.kind}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      kind: e.target.value,
                      config:
                        e.target.value === "imap"
                          ? {
                              host: "",
                              port: 993,
                              mailbox: "INBOX",
                              interval_minutes: 5,
                              from_now: true,
                              ca_pem: "",
                              insecure_tls: false,
                            }
                          : {},
                    })
                  }
                >
                  <option value="hmac">서명 Webhook</option>
                  <option value="imap">IMAP TLS 이메일</option>
                </select>
              </Field>
            </div>
            <Field label="수집 대상 워크스페이스">
              <select
                required
                disabled={!!draft.id}
                value={draft.workspace_id}
                onChange={(e) =>
                  setDraft({ ...draft, workspace_id: e.target.value })
                }
              >
                {workspaces.map((w) => (
                  <option key={w.id} value={w.id}>
                    {w.name}
                  </option>
                ))}
              </select>
            </Field>
            {draft.kind === "imap" ? (
              <>
                <div className="form-grid">
                  <Field label="IMAP 호스트">
                    <input
                      required
                      disabled={!!draft.id}
                      value={draft.config.host || ""}
                      onChange={(e) => config("host", e.target.value)}
                    />
                  </Field>
                  <Field label="IMAP TLS 포트">
                    <input
                      type="number"
                      required
                      disabled={!!draft.id}
                      min={1}
                      max={65535}
                      value={draft.config.port ?? 993}
                      onChange={(e) => config("port", Number(e.target.value))}
                    />
                  </Field>
                  <Field label="메일 사용자 이름">
                    <input
                      required={!draft.id}
                      disabled={!!draft.id}
                      autoComplete="off"
                      value={draft.secrets.username}
                      onChange={(e) =>
                        setDraft({
                          ...draft,
                          secrets: {
                            ...draft.secrets,
                            username: e.target.value,
                          },
                        })
                      }
                      placeholder={draft.id ? "저장된 계정 유지" : ""}
                    />
                  </Field>
                  <Field label="메일 비밀번호 또는 앱 암호">
                    <input
                      type="password"
                      required={!draft.id}
                      autoComplete="new-password"
                      value={draft.secrets.password}
                      onChange={(e) =>
                        setDraft({
                          ...draft,
                          secrets: {
                            ...draft.secrets,
                            password: e.target.value,
                          },
                        })
                      }
                      placeholder={draft.id ? "비워 두면 기존 비밀 유지" : ""}
                    />
                  </Field>
                  <Field label="수집할 메일 폴더">
                    <input
                      required
                      disabled={!!draft.id}
                      value={draft.config.mailbox || "INBOX"}
                      onChange={(e) => config("mailbox", e.target.value)}
                    />
                  </Field>
                  <Field label="메일 검사 주기 (분)">
                    <input
                      type="number"
                      min={1}
                      max={1440}
                      required
                      value={draft.config.interval_minutes ?? 5}
                      onChange={(e) =>
                        config("interval_minutes", Number(e.target.value))
                      }
                    />
                  </Field>
                </div>
                <label className="check-row">
                  <input
                    type="checkbox"
                    disabled={!!draft.id}
                    checked={!!draft.config.from_now}
                    onChange={(e) => config("from_now", e.target.checked)}
                  />
                  처음 연결한 이후에 도착한 메일부터 수집
                </label>
                <p className="muted">
                  선택을 해제하면 기존 메일부터 검사합니다. 메시지 UID와
                  Message-ID로 중복을 구분합니다. 서버·계정·폴더를 변경하려면 새
                  채널을 만드세요.
                </p>
                <Field label="메일 서버 CA 인증서 PEM (선택)">
                  <textarea
                    rows={3}
                    value={draft.config.ca_pem || ""}
                    onChange={(e) => config("ca_pem", e.target.value)}
                  />
                </Field>
                <label className="check-row">
                  <input
                    type="checkbox"
                    checked={!!draft.config.insecure_tls}
                    onChange={(e) => config("insecure_tls", e.target.checked)}
                  />
                  메일 서버 인증서 검증 생략 (테스트 전용)
                </label>
              </>
            ) : (
              <p className="muted">
                생성 후 1회 표시되는 서명 비밀로 timestamp.ID.body를 HMAC-SHA256
                서명합니다. 비밀을 가진 발신자도 이 채널의 고정된 개인
                수집함에만 저장할 수 있습니다.
              </p>
            )}
            <label className="check-row">
              <input
                type="checkbox"
                checked={!!draft.enabled}
                onChange={(e) =>
                  setDraft({ ...draft, enabled: e.target.checked })
                }
              />
              수집 채널 활성화
            </label>
            <div className="modal-actions">
              <Button
                type="button"
                disabled={busy}
                onClick={() => setDraft(null)}
              >
                취소
              </Button>
              <Button type="submit" disabled={busy} variant="primary">
                <Save size={17} />
                수집 채널 저장
              </Button>
            </div>
          </form>
        </Modal>
      )}
      {rotation && (
        <Modal
          open
          title="수집 서명 비밀 회전"
          onOpenChange={() => {
            if (!busy) setRotation(null);
          }}
        >
          <p>
            이전 서명 비밀은 즉시 무효화됩니다. 기존 대기 작업도 채널 버전
            변경으로 취소될 수 있습니다. 새 비밀을 발신 시스템에 반영하세요.
          </p>
          <ErrorBox error={error} />
          <div className="modal-actions">
            <Button disabled={busy} onClick={() => setRotation(null)}>
              취소
            </Button>
            <Button
              variant="primary"
              disabled={busy}
              onClick={async () => {
                setBusy(true);
                try {
                  const result = await api<Row>(
                    `/capture-channels/${rotation.id}/rotate`,
                    "POST",
                    { revision: rotation.revision },
                  );
                  setSecret({ id: rotation.id, value: result.signing_secret });
                  setRotation(null);
                  await load();
                } catch (e) {
                  setError((e as Error).message);
                } finally {
                  setBusy(false);
                }
              }}
            >
              새 비밀로 회전
            </Button>
          </div>
        </Modal>
      )}
      {secret && (
        <Modal
          open
          title="수집 서명 비밀 · 한 번만 표시"
          onOpenChange={() => setSecret(null)}
        >
          <p>
            이 창을 닫으면 비밀을 다시 조회할 수 없습니다. 발신 시스템의 비밀
            저장소에 보관하세요.
          </p>
          <Field label="수집 Webhook 주소">
            <input readOnly value={endpoint(secret.id)} />
          </Field>
          <Field label="새 수집 서명 비밀">
            <input readOnly value={secret.value} autoComplete="off" />
          </Field>
          <Button
            onClick={async () => {
              try {
                await navigator.clipboard.writeText(secret.value);
                notify("비밀을 복사했습니다");
              } catch {
                notify("복사할 값을 직접 선택하세요", "error");
              }
            }}
          >
            <Copy size={17} />
            비밀 복사
          </Button>
        </Modal>
      )}
    </>
  );
}
