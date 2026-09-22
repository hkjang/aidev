import { useCallback, useEffect, useRef, useState } from "react";
import "./channel-settings.css";
import {
  Bell,
  Mail,
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
const kinds = [
  ["smtp", "이메일 SMTP"],
  ["slack", "Slack"],
  ["teams", "Microsoft Teams"],
  ["mattermost", "Mattermost"],
  ["webhook", "일반 서명 Webhook"],
];
const states: Record<string, string> = {
  pending: "대기",
  sending: "전송 중",
  sent: "전송됨",
  failed: "실패",
  skipped: "조건 변경으로 제외",
  running: "실행 중",
  cancelled: "취소",
  succeeded: "완료",
};
function freshChannel() {
  return {
    name: "",
    kind: "smtp",
    workspace_id: "",
    enabled: false,
    config: {
      host: "",
      port: 465,
      from: "",
      tls_mode: "tls",
      ca_pem: "",
      insecure_tls: false,
      allow_http: false,
      allow_plaintext: false,
    },
    secrets: { username: "", password: "", url: "", signing_secret: "" },
  };
}
export function NotificationChannelsPage() {
  const { notify, workspaces } = useApp();
  const [rows, setRows] = useState<Row[]>([]),
    [policy, setPolicy] = useState<Row | null>(null),
    [hosts, setHosts] = useState(""),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [draft, setDraft] = useState<Row | null>(null);
  const generation = useRef(0);
  const load = useCallback(async () => {
    const run = ++generation.current;
    const [channels, settings] = await Promise.all([
      api<Row[]>("/admin/notification-channels"),
      api<Row>("/admin/notification-settings"),
    ]);
    if (run !== generation.current) return;
    setRows(channels);
    setPolicy(settings);
    setHosts(settings.allowed_hosts.join("\n"));
  }, []);
  useEffect(() => {
    void load().catch((e) => setError(e.message));
    return () => {
      generation.current++;
    };
  }, [load]);
  const changeConfig = (key: string, value: any) =>
      draft &&
      setDraft({ ...draft, config: { ...draft.config, [key]: value } }),
    changeSecret = (key: string, value: string) =>
      draft &&
      setDraft({ ...draft, secrets: { ...draft.secrets, [key]: value } });
  return (
    <>
      <PageHeading
        eyebrow="NOTIFICATION DELIVERY"
        title="외부 알림 채널"
        description="관리자가 목적지를 정하고 사용자가 개인 설정에서 받을 채널을 선택합니다."
        actions={
          <Button
            variant="primary"
            onClick={() => {
              setError("");
              setDraft(freshChannel());
            }}
          >
            <Plus size={18} />
            채널 추가
          </Button>
        }
      />
      <ErrorBox error={error} />
      {!policy ? (
        <Loading />
      ) : (
        <section className="card">
          <h2>외부 전송 정책</h2>
          <form
            onSubmit={async (e) => {
              e.preventDefault();
              setBusy(true);
              setError("");
              try {
                await api("/admin/notification-settings", "PUT", {
                  enabled: policy.enabled,
                  allowed_hosts: hosts
                    .split(/\n/)
                    .map((v) => v.trim())
                    .filter(Boolean),
                });
                await load();
                notify("외부 알림 정책을 저장했습니다");
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
                checked={!!policy.enabled}
                onChange={(e) =>
                  setPolicy({ ...policy, enabled: e.target.checked })
                }
              />
              외부 알림 전송 허용
            </label>
            <Field label="알림 서버 허용 호스트 (줄마다 하나)">
              <textarea
                rows={4}
                value={hosts}
                onChange={(e) => setHosts(e.target.value)}
                placeholder={"mail.example.internal\nhooks.slack.com"}
              />
            </Field>
            <p className="muted">
              정확한 호스트만 등록합니다. 와일드카드·리디렉션은 허용하지 않으며,
              외부 전송을 껐을 때 쌓인 알림을 나중에 일괄 발송하지 않습니다.
            </p>
            <Button disabled={busy} variant="primary" type="submit">
              <Save size={17} />
              전송 정책 저장
            </Button>
          </form>
        </section>
      )}
      <div className="notice">
        <ShieldCheck size={20} />
        <span>
          외부 채널에는 문서 제목·본문·댓글·사용자 ID를 보내지 않고 새 알림
          안내와 로그인 후 개인 알림함 링크만 보냅니다. 이메일 수신자는 사용자의
          현재 계정 주소입니다. 채널 URL과 비밀은 암호화하여 저장합니다.
        </span>
      </div>
      {rows.length ? (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>채널</th>
                <th>종류</th>
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
                    <p className="muted">{c.config.host || c.endpoint_host}</p>
                  </td>
                  <td>{kinds.find((v) => v[0] === c.kind)?.[1]}</td>
                  <td>
                    {c.workspace_id
                      ? workspaces.find((w) => w.id === c.workspace_id)?.name ||
                        "지정 워크스페이스"
                      : "서비스 공용"}
                  </td>
                  <td>
                    <Badge tone={c.enabled ? "success" : ""}>
                      {c.enabled ? "사용 중" : "중지"}
                    </Badge>
                  </td>
                  <td>
                    <Button
                      onClick={() => {
                        setError("");
                        setDraft({
                          ...c,
                          config: { ...c.config },
                          secrets: {
                            username: "",
                            password: "",
                            url: "",
                            signing_secret: "",
                          },
                        });
                      }}
                    >
                      <Settings2 size={17} />
                      편집
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <Empty
          title="등록된 외부 채널이 없습니다"
          text="사내 SMTP나 조직의 메시징 Webhook을 추가하세요."
        />
      )}
      {draft && (
        <Modal
          open
          wide
          title={draft.id ? "알림 채널 수정" : "알림 채널 추가"}
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
                await api(
                  "/admin/notification-channels" +
                    (draft.id ? "/" + draft.id : ""),
                  draft.id ? "PUT" : "POST",
                  draft,
                );
                setDraft(null);
                await load();
                notify("알림 채널을 저장했습니다");
              } catch (e) {
                setError((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            <div className="form-grid">
              <Field label="채널 이름">
                <input
                  required
                  maxLength={120}
                  value={draft.name}
                  onChange={(e) => setDraft({ ...draft, name: e.target.value })}
                />
              </Field>
              <Field label="알림 채널 종류">
                <select
                  disabled={!!draft.id}
                  value={draft.kind}
                  onChange={(e) => {
                    const next = freshChannel();
                    setDraft({
                      ...draft,
                      kind: e.target.value,
                      config:
                        e.target.value === "smtp"
                          ? next.config
                          : {
                              allow_http: false,
                              insecure_tls: false,
                              ca_pem: "",
                            },
                      secrets: next.secrets,
                    });
                  }}
                >
                  {kinds.map(([id, label]) => (
                    <option key={id} value={id}>
                      {label}
                    </option>
                  ))}
                </select>
              </Field>
            </div>
            <Field label="채널 대상 범위">
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
            <p className="muted">
              워크스페이스 전용 채널은 해당 워크스페이스 문서의 알림만 보냅니다.
              문서에 속하지 않은 일반 알림은 공용 채널로만 보냅니다.
            </p>
            {draft.kind === "smtp" ? (
              <>
                <div className="form-grid">
                  <Field label="SMTP 호스트">
                    <input
                      required
                      value={draft.config.host || ""}
                      onChange={(e) => changeConfig("host", e.target.value)}
                    />
                  </Field>
                  <Field label="SMTP 포트">
                    <input
                      type="number"
                      required
                      min={1}
                      max={65535}
                      value={draft.config.port ?? 465}
                      onChange={(e) =>
                        changeConfig("port", Number(e.target.value))
                      }
                    />
                  </Field>
                  <Field label="발신 이메일">
                    <input
                      type="email"
                      required
                      value={draft.config.from || ""}
                      onChange={(e) => changeConfig("from", e.target.value)}
                    />
                  </Field>
                  <Field label="SMTP 암호화 모드">
                    <select
                      value={draft.config.tls_mode || "tls"}
                      onChange={(e) => changeConfig("tls_mode", e.target.value)}
                    >
                      <option value="tls">TLS (일반적으로 465)</option>
                      <option value="starttls">
                        STARTTLS (일반적으로 587)
                      </option>
                      <option value="plaintext">사내 테스트 평문</option>
                    </select>
                  </Field>
                  <Field label="SMTP 사용자 이름">
                    <input
                      autoComplete="off"
                      value={draft.secrets.username}
                      onChange={(e) => changeSecret("username", e.target.value)}
                      placeholder={draft.id ? "비워 두면 기존 값 유지" : ""}
                    />
                  </Field>
                  <Field label="SMTP 비밀번호">
                    <input
                      type="password"
                      autoComplete="new-password"
                      value={draft.secrets.password}
                      onChange={(e) => changeSecret("password", e.target.value)}
                      placeholder={draft.id ? "비워 두면 기존 값 유지" : ""}
                    />
                  </Field>
                </div>
                {draft.config.tls_mode === "plaintext" && (
                  <label className="check-row">
                    <input
                      type="checkbox"
                      checked={!!draft.config.allow_plaintext}
                      onChange={(e) =>
                        changeConfig("allow_plaintext", e.target.checked)
                      }
                    />
                    사내 테스트 SMTP 평문 허용 (인증 비밀 전송 불가)
                  </label>
                )}
              </>
            ) : (
              <>
                <Field label="알림 Webhook URL">
                  <input
                    type="url"
                    required={!draft.id}
                    autoComplete="off"
                    value={draft.secrets.url}
                    onChange={(e) => changeSecret("url", e.target.value)}
                    placeholder={
                      draft.id ? "비워 두면 기존 URL 유지" : "https://…"
                    }
                  />
                </Field>
                {draft.kind === "webhook" && (
                  <Field label="Webhook 서명 비밀">
                    <input
                      type="password"
                      required={!draft.id}
                      minLength={32}
                      autoComplete="new-password"
                      value={draft.secrets.signing_secret}
                      onChange={(e) =>
                        changeSecret("signing_secret", e.target.value)
                      }
                      placeholder={
                        draft.id
                          ? "비워 두면 기존 값 유지"
                          : "32자 이상 독립적인 비밀"
                      }
                    />
                  </Field>
                )}
                {draft.kind === "teams" && (
                  <p className="muted">
                    Teams Workflow에서 Adaptive Card 요청을 받는 Webhook을
                    구성하세요. 조직 정책과 Workflow 소유자 유지 여부를
                    확인합니다.
                  </p>
                )}
                <label className="check-row">
                  <input
                    type="checkbox"
                    checked={!!draft.config.allow_http}
                    onChange={(e) =>
                      changeConfig("allow_http", e.target.checked)
                    }
                  />
                  사내망 알림 HTTP 허용
                </label>
              </>
            )}
            <Field label="알림 서버 CA 인증서 PEM (선택)">
              <textarea
                rows={3}
                value={draft.config.ca_pem || ""}
                onChange={(e) => changeConfig("ca_pem", e.target.value)}
              />
            </Field>
            <label className="check-row">
              <input
                type="checkbox"
                checked={!!draft.config.insecure_tls}
                onChange={(e) => changeConfig("insecure_tls", e.target.checked)}
              />
              서버 인증서 검증 생략 (테스트 전용)
            </label>
            <label className="check-row">
              <input
                type="checkbox"
                checked={!!draft.enabled}
                onChange={(e) =>
                  setDraft({ ...draft, enabled: e.target.checked })
                }
              />
              알림 채널 활성화
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
                {busy ? "저장 중…" : "알림 채널 저장"}
              </Button>
            </div>
          </form>
        </Modal>
      )}
    </>
  );
}
export function NotificationPreferencesPage() {
  const { notify, user } = useApp();
  const [rows, setRows] = useState<Row[]>([]),
    [history, setHistory] = useState<Row[]>([]),
    [error, setError] = useState(""),
    [loading, setLoading] = useState(true),
    [busy, setBusy] = useState("");
  const generation = useRef(0);
  const load = useCallback(async () => {
    const run = ++generation.current;
    try {
      const [channels, deliveries] = await Promise.all([
        api<Row[]>("/notification-preferences"),
        api<Row[]>("/notification-deliveries"),
      ]);
      if (run !== generation.current) return;
      setRows(channels);
      setHistory(deliveries);
      setError("");
    } catch (e) {
      if (run === generation.current) setError((e as Error).message);
    } finally {
      if (run === generation.current) setLoading(false);
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
        eyebrow="PERSONAL NOTIFICATIONS"
        title="내 외부 알림"
        description="인앱 알림과 별개로 관리자가 허용한 외부 채널을 직접 선택합니다."
        actions={
          <Button disabled={!!busy} onClick={() => void load()}>
            <RefreshCw size={17} />
            이력 새로고침
          </Button>
        }
      />
      <ErrorBox error={error} />
      <div className="notice">
        <Mail size={20} />
        <span>
          이메일은 {user.email}로 발송합니다. 외부 메시지에는 문서 본문이나
          제목이 없으며 로그인 후 개인 알림함에서 확인합니다. 다른 사람이 볼 수
          있는 팀 채널은 필요할 때만 선택하세요.
        </span>
      </div>
      {loading ? (
        <Loading />
      ) : rows.length ? (
        <section className="card">
          <h2>받을 채널</h2>
          {rows.map((c) => (
            <label className="check-row" key={c.id}>
              <input
                type="checkbox"
                disabled={!!busy || (!c.available && !c.enabled)}
                checked={!!c.enabled}
                onChange={async (e) => {
                  const enabled = e.target.checked;
                  setBusy(c.id);
                  setError("");
                  setRows((previous) =>
                    previous.map((row) =>
                      row.id === c.id ? { ...row, enabled } : row,
                    ),
                  );
                  try {
                    await api("/notification-preferences/" + c.id, "PUT", {
                      enabled,
                    });
                    await load();
                    notify(
                      enabled
                        ? "알림 채널을 선택했습니다"
                        : "외부 알림 선택을 해제했습니다",
                    );
                  } catch (e) {
                    setRows((previous) =>
                      previous.map((row) =>
                        row.id === c.id ? { ...row, enabled: c.enabled } : row,
                      ),
                    );
                    setError((e as Error).message);
                  } finally {
                    setBusy("");
                  }
                }}
              />
              <span>
                <strong>{c.name}</strong> ·{" "}
                {kinds.find((v) => v[0] === c.kind)?.[1]}{" "}
                {!c.available && <Badge>관리자 중지</Badge>}
              </span>
            </label>
          ))}
        </section>
      ) : (
        <Empty
          title="선택할 외부 알림 채널이 없습니다"
          text="서비스 관리자에게 알림 채널 구성을 요청하세요."
        />
      )}
      <section className="card">
        <h2>
          <Bell size={20} />
          최근 전송 이력
        </h2>
        {history.length ? (
          <div className="table-wrap">
            <table className="channel-history-table">
              <thead>
                <tr>
                  <th>채널</th>
                  <th>상태</th>
                  <th>시도</th>
                  <th>시간</th>
                  <th>설명</th>
                </tr>
              </thead>
              <tbody>
                {history.map((d) => (
                  <tr key={d.id}>
                    <td>{d.channel_name}</td>
                    <td>
                      <Badge>{states[d.status] || d.status}</Badge>
                      {d.status === "failed" && (
                        <small>{states[d.job_status] || d.job_status}</small>
                      )}
                    </td>
                    <td>{d.attempts || 0}</td>
                    <td>{datetime(d.sent_at || d.created_at)}</td>
                    <td>{d.message || "—"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <p className="muted">
            선택 후 발생한 새 알림의 전송 결과가 여기에 표시됩니다.
          </p>
        )}
        <p className="muted">
          일시 오류는 서버 작업 큐에서 재시도합니다. 외부 서비스가 응답 확인을
          잃으면 같은 알림이 중복 도착할 수 있습니다. 일반 Webhook은 전송 ID로
          중복을 구분할 수 있습니다.
        </p>
      </section>
    </>
  );
}
