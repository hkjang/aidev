import { useEffect, useRef, useState } from "react";
import { ShortcutSettings } from "./navigation/shortcuts";
import {
  navigationPresets,
  navigationPreset,
} from "./navigation/WorkspaceNavigation";
import { useSearchParams } from "react-router-dom";
import {
  Check,
  Code2,
  Copy,
  KeyRound,
  LockKeyhole,
  Plus,
  RefreshCw,
  Settings,
  ShieldCheck,
  Trash2,
  UserRound,
} from "lucide-react";
import { api, date, datetime, type User } from "./api";
import { useApp } from "./context";
import {
  Badge,
  Button,
  CopyButton,
  Empty,
  ErrorBox,
  Field,
  Modal,
  PageHeading,
  Toggle,
  roleNames,
} from "./ui";

export function ProfilePage() {
  const { user, setUser, notify } = useApp();
  const changedPreferences = useRef(new Set<string>());
  const mounted = useRef(true),
    currentUser = useRef(user.id);
  currentUser.current = user.id;
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  const [name, setName] = useState(user.name),
    [preferences, setPreferences] = useState({ ...user.preferences }),
    [currentPassword, setCurrentPassword] = useState(""),
    [password, setPassword] = useState(""),
    [confirm, setConfirm] = useState(""),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const update = (key: string, v: any) => {
    changedPreferences.current.add(key);
    setPreferences((current) => ({ ...current, [key]: v }));
  };
  useEffect(
    () =>
      setPreferences((current) => ({
        ...user.preferences,
        ...Object.fromEntries(
          [...changedPreferences.current].map((key) => [key, current[key]]),
        ),
      })),
    [user.preferences],
  );
  return (
    <>
      <PageHeading
        eyebrow="MAKE IT YOURS"
        title="개인 설정"
        description="나에게 편한 방식으로 madi를 사용하세요."
      />
      <div className="settings-layout">
        <aside className="settings-overview">
          <span className="profile-large-avatar">{user.name?.slice(0, 1)}</span>
          <h2>{user.name}</h2>
          <p>{user.email}</p>
          <Badge tone="green">{roleNames[user.role] || user.role}</Badge>
          <div className="notice subtle">
            <ShieldCheck size={19} />
            <span>개인 설정은 계정에 저장되어 다른 기기에서도 이어집니다.</span>
          </div>
        </aside>
        <form
          className="settings-form"
          onSubmit={async (e) => {
            e.preventDefault();
            if (busy) return;
            const actor = user.id;
            setBusy(true);
            setError("");
            try {
              if (password && password !== confirm)
                throw new Error("새 비밀번호가 일치하지 않습니다.");
              const updated = await api<User>("/profile", "PUT", {
                expected_user_id: user.id,
                name,
                preferences: Object.fromEntries(
                  [...changedPreferences.current].map((key) => [
                    key,
                    preferences[key],
                  ]),
                ),
                ...(password
                  ? { password, current_password: currentPassword }
                  : {}),
              });
              if (!mounted.current || currentUser.current !== updated.id)
                return;
              setUser(updated);
              changedPreferences.current.clear();
              setPassword("");
              setCurrentPassword("");
              setConfirm("");
              notify("개인 설정을 저장했습니다.");
            } catch (e) {
              if (mounted.current && currentUser.current === actor)
                setError((e as Error).message);
            } finally {
              if (mounted.current && currentUser.current === actor)
                setBusy(false);
            }
          }}
        >
          <fieldset className="profile-fields" disabled={busy}>
            <section className="panel padded">
              <h2>
                <UserRound size={20} /> 프로필
              </h2>
              <Field label="이름">
                <input
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  required
                />
              </Field>
              <Field label="이메일">
                <input value={user.email} readOnly />
                <small>이메일 변경은 관리자에게 문의하세요.</small>
              </Field>
            </section>
            <section className="panel padded">
              <h2>
                <Settings size={20} /> 화면과 편집
              </h2>
              <div className="form-grid">
                <Field label="화면 밀도">
                  <select
                    value={preferences.density || "comfortable"}
                    onChange={(e) => update("density", e.target.value)}
                  >
                    <option value="comfortable">기본 · 편안한 간격</option>
                    <option value="compact">
                      촘촘하게 · 글자와 누름 영역 유지
                    </option>
                    <option value="relaxed">여유롭게 · 간격 넓히기</option>
                  </select>
                  <small>
                    글자는 16px 이상, 주요 조작 영역은 44px 이상을 유지합니다.
                  </small>
                </Field>
                <Field label="모바일 데이터베이스 기본 보기">
                  <select
                    value={preferences.mobile_table_view || "cards"}
                    onChange={(e) =>
                      update("mobile_table_view", e.target.value)
                    }
                  >
                    <option value="cards">카드 · 행별로 읽기</option>
                    <option value="table">표 · 가로 비교와 셀 편집</option>
                  </select>
                </Field>
                <Field label="테마">
                  <select
                    value={preferences.theme || "light"}
                    onChange={(e) => update("theme", e.target.value)}
                  >
                    <option value="light">라이트 · 밝고 편안하게</option>
                    <option value="dark">다크 · 눈부심을 줄이게</option>
                  </select>
                </Field>
                <Field label="글자 크기">
                  <select
                    value={preferences.font_size || 16}
                    onChange={(e) =>
                      update("font_size", Number(e.target.value))
                    }
                  >
                    {[16, 17, 18, 19, 20, 21, 22, 23, 24].map((n) => (
                      <option key={n} value={n}>
                        {n}px {n === 16 ? "· 기본" : n === 18 ? "· 크게" : ""}
                      </option>
                    ))}
                  </select>
                </Field>
                <Field label="사이드바 너비">
                  <select
                    value={preferences.sidebar_width || 268}
                    onChange={(e) =>
                      update("sidebar_width", Number(e.target.value))
                    }
                  >
                    <option value={240}>240px · 좁게</option>
                    <option value={268}>268px · 기본</option>
                    <option value={300}>300px · 넓게</option>
                    <option value={340}>340px · 더 넓게</option>
                    {preferences.sidebar_width &&
                      ![240, 268, 300, 340].includes(
                        Number(preferences.sidebar_width),
                      ) && (
                        <option value={preferences.sidebar_width}>
                          {preferences.sidebar_width}px · 직접 조정
                        </option>
                      )}
                  </select>
                </Field>
                <Field label="글꼴">
                  <select
                    value={preferences.font_family || "sans"}
                    onChange={(e) => update("font_family", e.target.value)}
                  >
                    <option value="sans">고딕 · 내장 Noto Sans KR</option>
                    <option value="system">기기 기본 고딕</option>
                    <option value="serif">명조 · 기기에 설치된 글꼴</option>
                  </select>
                  <small>
                    글꼴을 외부 서버에서 내려받지 않습니다. 명조 글꼴이 없으면
                    기기 기본 글꼴로 표시합니다.
                  </small>
                </Field>
                <Field label="문서 본문 너비">
                  <select
                    value={preferences.page_width || "standard"}
                    onChange={(e) => update("page_width", e.target.value)}
                  >
                    <option value="standard">표준 · 최대 880px</option>
                    <option value="wide">넓게 · 최대 1,200px</option>
                    <option value="full">전체 · 현재 화면에 맞춤</option>
                  </select>
                </Field>
                <Field label="코드 색상">
                  <select
                    value={preferences.code_theme || "auto"}
                    onChange={(e) => update("code_theme", e.target.value)}
                  >
                    <option value="auto">화면 테마 따르기</option>
                    <option value="light">밝은 코드 배경</option>
                    <option value="dark">어두운 코드 배경</option>
                  </select>
                </Field>
                <Field label="맞춤법 검사">
                  <select
                    value={preferences.spell_check === false ? "off" : "on"}
                    onChange={(e) =>
                      update("spell_check", e.target.value === "on")
                    }
                  >
                    <option value="on">기기·브라우저 맞춤법 검사 사용</option>
                    <option value="off">사용하지 않음</option>
                  </select>
                  <small>
                    madi는 검사 서버를 호출하지 않습니다. 브라우저의 향상된
                    맞춤법 검사는 브라우저 설정과 조직 정책을 확인하세요.
                  </small>
                </Field>
                <Field label="기본 편집 모드">
                  <select
                    value={preferences.editor_mode || "edit"}
                    onChange={(e) => update("editor_mode", e.target.value)}
                  >
                    <option value="edit">블록 편집기</option>
                    <option value="source">Markdown 원문</option>
                    <option value="preview">읽기 모드</option>
                  </select>
                </Field>
                <Field label="시간대">
                  <select
                    value={preferences.timezone || "Asia/Seoul"}
                    onChange={(e) => update("timezone", e.target.value)}
                  >
                    <option value="Asia/Seoul">서울 · Asia/Seoul</option>
                    <option value="UTC">UTC</option>
                    <option value="Asia/Tokyo">도쿄 · Asia/Tokyo</option>
                    <option value="America/New_York">
                      뉴욕 · America/New_York
                    </option>
                  </select>
                </Field>
                <Field label="언어">
                  <select value="ko" onChange={() => {}}>
                    <option value="ko">한국어</option>
                  </select>
                </Field>
                <Field label="날짜 표시">
                  <select
                    value={preferences.date_format || "ko"}
                    onChange={(e) => update("date_format", e.target.value)}
                  >
                    <option value="ko">한국어 기본 · 9월 8일</option>
                    <option value="iso">숫자형 · 2026-09-08</option>
                    <option value="long">자세히 · 연도와 요일 포함</option>
                  </select>
                  <small>
                    문서·이력·감사 기록의 날짜와 시각에 저장한 시간대를 함께
                    적용합니다.
                  </small>
                </Field>
              </div>
            </section>
            <section className="panel padded">
              <h2>
                <Settings size={20} />
                탐색과 문서 패널
              </h2>
              <p className="muted">
                메뉴 구성만 바뀝니다. 접근 권한과 관리자가 설정한 기능 정책은
                그대로 적용됩니다.
              </p>
              <div className="form-grid">
                <Field label="기본 작업 방식">
                  <select
                    value={navigationPreset(preferences.nav_preset)}
                    onChange={(e) => update("nav_preset", e.target.value)}
                  >
                    {navigationPresets.map((p) => (
                      <option key={p.id} value={p.id}>
                        {p.label}
                      </option>
                    ))}
                  </select>
                </Field>
                <Field label="고급 도구 메뉴">
                  <select
                    value={
                      preferences.navigation_advanced === true ? "on" : "off"
                    }
                    onChange={(e) =>
                      update("navigation_advanced", e.target.value === "on")
                    }
                  >
                    <option value="off">필요할 때 전체 도구 펼치기</option>
                    <option value="on">전체 도구 항상 표시</option>
                  </select>
                </Field>
                <Field label="기본 문서 패널">
                  <select
                    value={preferences.document_panel || "backlinks"}
                    onChange={(e) => update("document_panel", e.target.value)}
                  >
                    <option value="backlinks">연결된 문서와 목차</option>
                    <option value="properties">문서 속성과 도구</option>
                    <option value="comments">댓글</option>
                    <option value="ai">AI 지식 도우미</option>
                    <option value="versions">변경 이력</option>
                  </select>
                </Field>
                <Field label="문서 패널 표시">
                  <select
                    value={
                      preferences.document_panel_open !== false ? "on" : "off"
                    }
                    onChange={(e) =>
                      update("document_panel_open", e.target.value === "on")
                    }
                  >
                    <option value="on">펼쳐서 시작</option>
                    <option value="off">접어서 시작</option>
                  </select>
                </Field>
              </div>
            </section>
            <ShortcutSettings preferences={preferences} onChange={update} />
            <section className="panel padded">
              <h2>
                <LockKeyhole size={20} /> 비밀번호 변경
              </h2>
              <Field label="현재 비밀번호">
                <input
                  type="password"
                  autoComplete="current-password"
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                  required={!!password}
                />
              </Field>
              <div className="form-grid">
                <Field label="새 비밀번호">
                  <input
                    type="password"
                    autoComplete="new-password"
                    minLength={12}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="12자 이상"
                  />
                </Field>
                <Field label="새 비밀번호 확인">
                  <input
                    type="password"
                    autoComplete="new-password"
                    value={confirm}
                    onChange={(e) => setConfirm(e.target.value)}
                    required={!!password}
                  />
                </Field>
              </div>
            </section>
            <ErrorBox error={error} />
            <div className="settings-save">
              <span>변경한 설정을 저장하면 적용됩니다.</span>
              <Button variant="primary" disabled={busy}>
                {busy ? "저장 중…" : "변경사항 저장"}
                <Check size={17} />
              </Button>
            </div>
          </fieldset>
        </form>
      </div>
    </>
  );
}

export const scopeNames: Record<string, string> = {
  "document:read": "문서 읽기",
  "document:write": "문서 작성·수정",
  "database:read": "데이터베이스 읽기",
  "database:write": "데이터베이스 수정",
  "search:read": "문서 검색",
  "ai:execute": "AI 실행",
};
export function KeysPage() {
  const { workspaces, workspace, user, notify } = useApp();
  const [params] = useSearchParams();
  const owner = user.role === "admin" ? params.get("user_id") || "" : "";
  const [items, setItems] = useState<any[]>([]),
    [open, setOpen] = useState(false),
    [editing, setEditing] = useState<any>(null),
    [name, setName] = useState(""),
    [ws, setWs] = useState(workspace?.id || ""),
    [scopes, setScopes] = useState<string[]>(["document:read", "search:read"]),
    [days, setDays] = useState(90),
    [ips, setIps] = useState(""),
    [rate, setRate] = useState(120),
    [token, setToken] = useState(""),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  const load = () =>
    api<any[]>("/keys" + (owner ? "?user_id=" + owner : ""))
      .then(setItems)
      .catch((e) => setError(e.message));
  useEffect(() => {
    load();
  }, [owner]);
  const edit = (k?: any) => {
    setEditing(k || null);
    setName(k?.name || "");
    setWs(k?.workspace_id || workspace?.id || "");
    setScopes(k?.scopes || ["document:read", "search:read"]);
    setDays(90);
    setIps((k?.ip_allowlist || []).join(", "));
    setRate(k?.rate_limit || 120);
    setOpen(true);
  };
  const endpoint = window.location.origin + "/api/v1/mcp";
  return (
    <>
      <PageHeading
        eyebrow="CONNECTED, WITH CONTROL"
        title={owner ? "서비스 계정 API 키" : "내 API 키"}
        description="외부 도구와 AI 에이전트에 필요한 권한만 연결하세요."
        actions={
          <Button variant="primary" onClick={() => edit()}>
            <Plus size={18} /> 새 API 키
          </Button>
        }
      />
      <div className="notice">
        <ShieldCheck size={20} />
        <span>
          키마다 워크스페이스와 권한을 지정합니다. 키는 소유자의 실제 접근 권한
          안에서만 동작하며, 회전하면 이전 키는 즉시 폐기됩니다.
        </span>
      </div>
      <ErrorBox error={error} />
      <div className="panel key-list">
        {items.length ? (
          items.map((k) => (
            <article
              className={`key-item ${k.revoked_at ? "revoked" : ""}`}
              key={k.id}
            >
              <span className="key-icon">
                <KeyRound size={23} />
              </span>
              <div className="key-details">
                <div>
                  <h3>{k.name}</h3>
                  <Badge
                    tone={
                      k.revoked_at
                        ? ""
                        : new Date(k.expires_at) < new Date()
                          ? "amber"
                          : "green"
                    }
                  >
                    {k.revoked_at
                      ? "폐기됨"
                      : new Date(k.expires_at) < new Date()
                        ? "만료됨"
                        : "사용 중"}
                  </Badge>
                </div>
                <code>{k.prefix}••••••••</code>
                <p>
                  {workspaces.find((w) => w.id === k.workspace_id)?.name ||
                    "워크스페이스"}{" "}
                  · 만료 {date(k.expires_at)} · 최근 사용{" "}
                  {k.last_used_at ? date(k.last_used_at) : "없음"}
                </p>
                <div className="tags">
                  {k.scopes.map((s: string) => (
                    <Badge key={s}>{scopeNames[s] || s}</Badge>
                  ))}
                </div>
              </div>
              {!k.revoked_at && (
                <div className="key-actions">
                  <Button onClick={() => edit(k)}>권한 변경</Button>
                  <Button
                    onClick={async () => {
                      if (
                        !window.confirm(
                          "키를 회전하면 기존 키는 즉시 사용할 수 없습니다. 새 키를 발급할까요?",
                        )
                      )
                        return;
                      try {
                        const result = await api(
                          `/keys/${k.id}/rotate`,
                          "POST",
                        );
                        setToken(result.token);
                        load();
                        notify(
                          "키를 회전했습니다. 새 키를 안전하게 보관하세요.",
                        );
                      } catch (e) {
                        notify((e as Error).message, "error");
                      }
                    }}
                  >
                    <RefreshCw size={16} /> 회전
                  </Button>
                  <button
                    className="icon-button danger-text"
                    aria-label={`${k.name} 키 폐기`}
                    onClick={async () => {
                      if (
                        !window.confirm(
                          `'${k.name}' 키를 폐기할까요? 이 키의 접근이 즉시 차단됩니다.`,
                        )
                      )
                        return;
                      try {
                        await api("/keys/" + k.id, "DELETE");
                        load();
                        notify("API 키를 폐기했습니다.");
                      } catch (e) {
                        notify((e as Error).message, "error");
                      }
                    }}
                  >
                    <Trash2 size={18} />
                  </button>
                </div>
              )}
            </article>
          ))
        ) : (
          <Empty
            title="아직 API 키가 없어요"
            text="MCP 클라이언트와 외부 서비스에 연결할 키를 만들어 보세요."
          />
        )}
      </div>
      <div className="two-columns integration-examples">
        <section className="panel padded">
          <h2>
            <Code2 size={20} /> REST API
          </h2>
          <p className="muted">개인 키를 Bearer 인증 헤더에 넣어 사용하세요.</p>
          <pre className="code-example">{`curl '${window.location.origin}/api/v1/documents?workspace_id=${workspace?.id || "WORKSPACE_ID"}' \\\n  -H 'Authorization: Bearer YOUR_API_KEY'`}</pre>
          <CopyButton
            value={`curl '${window.location.origin}/api/v1/documents?workspace_id=${workspace?.id || "WORKSPACE_ID"}' -H 'Authorization: Bearer YOUR_API_KEY'`}
          />
        </section>
        <section className="panel padded">
          <h2>
            <KeyRound size={20} /> MCP 연결
          </h2>
          <p className="muted">
            HTTP MCP 클라이언트에서 아래 주소와 인증 헤더를 설정하세요.
          </p>
          <Field label="MCP 서버 주소">
            <div className="input-with-button">
              <input readOnly value={endpoint} />
              <CopyButton value={endpoint} />
            </div>
          </Field>
          <code>Authorization: Bearer YOUR_API_KEY</code>
        </section>
      </div>
      <Modal
        open={open}
        onOpenChange={setOpen}
        title={editing ? "API 키 권한 변경" : "새 API 키 만들기"}
        description="최소한의 권한과 적절한 유효기간을 선택하세요."
      >
        <form
          onSubmit={async (e) => {
            e.preventDefault();
            setBusy(true);
            try {
              const result = await api(
                `/keys${editing ? "/" + editing.id : ""}`,
                editing ? "PUT" : "POST",
                {
                  name,
                  workspace_id: ws,
                  scopes,
                  expires_in_days: days,
                  ip_allowlist: ips
                    .split(",")
                    .map((s) => s.trim())
                    .filter(Boolean),
                  rate_limit: rate,
                  ...(owner ? { user_id: owner } : {}),
                },
              );
              setOpen(false);
              if (result.token) setToken(result.token);
              load();
              notify(
                editing ? "키 권한을 변경했습니다." : "API 키를 만들었습니다.",
              );
            } catch (e) {
              notify((e as Error).message, "error");
            } finally {
              setBusy(false);
            }
          }}
        >
          <Field label="키 이름">
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="예: 사내 AI 에이전트"
              required
              maxLength={120}
            />
          </Field>
          <Field label="워크스페이스">
            <select value={ws} onChange={(e) => setWs(e.target.value)} required>
              <option value="" disabled>
                워크스페이스를 선택하세요
              </option>
              {workspaces.map((w) => (
                <option key={w.id} value={w.id}>
                  {w.name}
                </option>
              ))}
            </select>
          </Field>
          <fieldset className="scope-fieldset">
            <legend>허용 권한</legend>
            <div className="scope-grid">
              {Object.entries(scopeNames).map(([v, l]) => (
                <label key={v}>
                  <input
                    type="checkbox"
                    checked={scopes.includes(v)}
                    onChange={(e) =>
                      setScopes(
                        e.target.checked
                          ? [...scopes, v]
                          : scopes.filter((s) => s !== v),
                      )
                    }
                  />
                  <span>
                    {l}
                    <small>{v}</small>
                  </span>
                </label>
              ))}
            </div>
          </fieldset>
          <div className="form-grid">
            <Field label="유효기간 (일)">
              <input
                type="number"
                min={1}
                max={3650}
                value={days}
                onChange={(e) => setDays(Number(e.target.value))}
                required
              />
            </Field>
            <Field label="분당 최대 호출">
              <input
                type="number"
                min={1}
                max={10000}
                value={rate}
                onChange={(e) => setRate(Number(e.target.value))}
                required
              />
            </Field>
          </div>
          <Field
            label="허용 IP / CIDR"
            hint="쉼표로 구분합니다. 비워 두면 IP 제한을 적용하지 않습니다."
          >
            <input
              value={ips}
              onChange={(e) => setIps(e.target.value)}
              placeholder="10.0.0.0/8, 192.168.1.20"
            />
          </Field>
          <div className="modal-actions">
            <Button type="button" onClick={() => setOpen(false)}>
              취소
            </Button>
            <Button variant="primary" disabled={busy || !scopes.length}>
              저장
            </Button>
          </div>
        </form>
      </Modal>
      <Modal
        open={!!token}
        onOpenChange={(v) => {
          if (!v) setToken("");
        }}
        title="새 API 키를 보관하세요"
        description="이 비밀 키는 지금 한 번만 표시됩니다. 안전한 곳에 복사한 후 닫으세요."
      >
        <div className="secret-token">
          <code>{token}</code>
          <CopyButton value={token} />
        </div>
        <div className="modal-actions">
          <Button variant="primary" onClick={() => setToken("")}>
            확인, 안전하게 보관했어요
          </Button>
        </div>
      </Modal>
    </>
  );
}
