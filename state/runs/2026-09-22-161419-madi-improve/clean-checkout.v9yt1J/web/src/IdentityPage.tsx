import { useCallback, useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { KeyRound, Plus, RefreshCw, ShieldCheck, Trash2 } from "lucide-react";
import { api, datetime, type Settings, type User } from "./api";
import { useApp } from "./context";
import {
  Button,
  CopyButton,
  Empty,
  ErrorBox,
  Field,
  Loading,
  Modal,
  PageHeading,
  Toggle,
} from "./ui";
import { scopeNames } from "./PersonalPages";
import "./identity.css";

type Row = Record<string, any>;
const sections = [
  ["policy", "계정 연결·그룹 권한"],
  ["ldap", "LDAP · Active Directory"],
  ["saml", "SAML SSO"],
  ["scim", "SCIM 동기화"],
];
const identityScopes: Record<string, string> = {
  ...scopeNames,
  "identity:provision": "SCIM 사용자·그룹 관리",
};

export default function IdentityPage() {
  const { workspaces, notify, refreshPublic } = useApp();
  const [params, setParams] = useSearchParams(),
    section = sections.some(([v]) => v === params.get("tab"))
      ? params.get("tab")!
      : "policy";
  const [settings, setSettings] = useState<Settings | null>(null),
    [changes, setChanges] = useState<Settings>({}),
    [users, setUsers] = useState<User[]>([]),
    [links, setLinks] = useState<Row[]>([]),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  const [link, setLink] = useState<Row | null>(null),
    [owner, setOwner] = useState(""),
    [keys, setKeys] = useState<Row[]>([]),
    [keyDraft, setKeyDraft] = useState<Row | null>(null),
    [token, setToken] = useState("");
  const load = useCallback(async () => {
    try {
      const [cfg, accounts, bindings] = await Promise.all([
        api<Settings>("/admin/settings"),
        api<User[]>("/admin/users"),
        api<Row[]>("/admin/identity/links"),
      ]);
      setSettings(cfg);
      setUsers(accounts);
      setLinks(bindings);
      setChanges({});
      setError("");
    } catch (e) {
      setError((e as Error).message);
    }
  }, []);
  useEffect(() => {
    void load();
  }, [load]);
  const loadKeys = useCallback(async () => {
    if (!owner) {
      setKeys([]);
      return;
    }
    try {
      setKeys(await api<Row[]>("/keys?user_id=" + owner));
    } catch (e) {
      setError((e as Error).message);
    }
  }, [owner]);
  useEffect(() => {
    let active = true;
    setKeys([]);
    if (owner)
      api<Row[]>("/keys?user_id=" + owner)
        .then((rows) => {
          if (active) setKeys(rows);
        })
        .catch((e) => {
          if (active) setError(e.message);
        });
    return () => {
      active = false;
    };
  }, [owner]);
  const cfg = { ...settings, ...changes },
    set = (key: string, value: any) =>
      setChanges((old) => ({ ...old, [key]: value }));
  const run = async (operation: () => Promise<void>) => {
    setBusy(true);
    setError("");
    try {
      await operation();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  const text = (key: string, label: string, help = "", password = false) => (
    <Field label={label} key={key}>
      <input
        type={password ? "password" : "text"}
        autoComplete={password ? "new-password" : "off"}
        value={String(cfg[key] || "")}
        onChange={(e) => set(key, e.target.value)}
        placeholder={
          password && settings?.[key + "_configured"]
            ? "저장된 비밀 유지 (변경할 때만 입력)"
            : ""
        }
        maxLength={password ? 4096 : 4000}
      />
      {help && <small className="muted">{help}</small>}
    </Field>
  );
  const area = (key: string, label: string, help = "") => (
    <Field label={label}>
      <textarea
        rows={6}
        value={String(cfg[key] || "")}
        onChange={(e) => set(key, e.target.value)}
        maxLength={65536}
      />
      {help && <small className="muted">{help}</small>}
    </Field>
  );
  const toggle = (key: string, label: string) => (
    <Toggle
      checked={!!cfg[key]}
      onChange={(value) => set(key, value)}
      label={label}
    />
  );
  const save = () =>
    run(async () => {
      await api("/admin/settings", "PUT", changes);
      await refreshPublic();
      await load();
      notify("인증 설정과 그룹 권한을 저장했습니다.");
    });
  if (!settings && !error) return <Loading />;
  return (
    <>
      <PageHeading
        eyebrow="IDENTITY & TRUST"
        title="인증·디렉터리 관리"
        description="회사 계정 연결부터 조직 권한과 자동 프로비저닝까지, 관리자가 통제합니다."
        actions={
          <Button onClick={() => void load()} disabled={busy}>
            <RefreshCw size={17} /> 새로고침
          </Button>
        }
      />
      <ErrorBox error={error} />
      <nav className="identity-tabs" aria-label="인증 설정 영역">
        {sections.map(([value, label]) => (
          <Button
            key={value}
            variant={section === value ? "primary" : ""}
            onClick={() => setParams({ tab: value })}
          >
            {label}
          </Button>
        ))}
      </nav>
      <div className="notice">
        <ShieldCheck size={20} />
        <span>
          Bootstrap 관리자 로컬 로그인은 유지됩니다. 외부 계정은 이메일이 같다는
          이유만으로 관리자 계정과 자동 연결되지 않습니다.
        </span>
      </div>
      {section === "policy" && (
        <>
          <section className="panel padded">
            <h2>외부 계정 연결 정책</h2>
            <Field label="기존 계정 연결">
              <select
                value={cfg.identity_link_policy || "manual"}
                onChange={(e) => set("identity_link_policy", e.target.value)}
              >
                <option value="manual">
                  관리자가 고유 ID로 직접 연결 (기본)
                </option>
                <option value="verified_email_non_admin">
                  검증된 이메일로 일반 사용자만 자동 연결
                </option>
              </select>
            </Field>
            <p className="muted">
              자동 연결을 허용해도 관리자·서비스 계정·다른 외부 ID에 연결된
              계정은 제외됩니다. LDAP와 SAML 이메일은 관리자가 신뢰한 제공자의
              주장으로 취급합니다.
            </p>
            {text(
              "oidc_groups_claim",
              "OIDC 그룹 claim 경로",
              "Keycloak 기본 예: groups. 중첩 JSON은 realm_access.roles처럼 점으로 구분합니다.",
            )}
          </section>
          <section className="panel padded">
            <div className="identity-section-head">
              <h2>외부 그룹 → 워크스페이스 역할</h2>
              <Button
                onClick={() =>
                  set("identity_group_mappings", [
                    ...(cfg.identity_group_mappings || []),
                    {
                      provider: "oidc",
                      group: "",
                      workspace_id: "",
                      role: "viewer",
                    },
                  ])
                }
              >
                <Plus size={17} /> 그룹 매핑
              </Button>
            </div>
            <p className="muted">
              여러 그룹에서는 가장 높은 역할을 적용합니다. 관리자가 직접 지정한
              멤버 역할은 우선 유지됩니다. SCIM은 키의 워크스페이스를 벗어난
              매핑을 적용하지 않습니다.
            </p>
            <div className="identity-table-wrap">
              <table className="identity-table">
                <thead>
                  <tr>
                    <th>제공자</th>
                    <th>외부 그룹 이름</th>
                    <th>워크스페이스</th>
                    <th>역할</th>
                    <th>삭제</th>
                  </tr>
                </thead>
                <tbody>
                  {(cfg.identity_group_mappings || []).map(
                    (mapping: Row, index: number) => {
                      const edit = (key: string, value: string) =>
                        set(
                          "identity_group_mappings",
                          cfg.identity_group_mappings.map(
                            (m: Row, i: number) =>
                              i === index ? { ...m, [key]: value } : m,
                          ),
                        );
                      return (
                        <tr key={index}>
                          <td>
                            <select
                              aria-label={`매핑 ${index + 1} 제공자`}
                              value={mapping.provider}
                              onChange={(e) => edit("provider", e.target.value)}
                            >
                              {["oidc", "ldap", "saml", "scim"].map((value) => (
                                <option key={value} value={value}>
                                  {value.toUpperCase()}
                                </option>
                              ))}
                            </select>
                          </td>
                          <td>
                            <input
                              aria-label={`매핑 ${index + 1} 그룹`}
                              value={mapping.group}
                              onChange={(e) => edit("group", e.target.value)}
                              maxLength={2000}
                            />
                          </td>
                          <td>
                            <select
                              aria-label={`매핑 ${index + 1} 워크스페이스`}
                              value={mapping.workspace_id}
                              onChange={(e) =>
                                edit("workspace_id", e.target.value)
                              }
                            >
                              <option value="">워크스페이스 선택</option>
                              {workspaces.map((ws) => (
                                <option key={ws.id} value={ws.id}>
                                  {ws.name}
                                </option>
                              ))}
                            </select>
                          </td>
                          <td>
                            <select
                              aria-label={`매핑 ${index + 1} 역할`}
                              value={mapping.role}
                              onChange={(e) => edit("role", e.target.value)}
                            >
                              <option value="viewer">조회자</option>
                              <option value="commenter">댓글 작성자</option>
                              <option value="editor">편집자</option>
                              <option value="admin">워크스페이스 관리자</option>
                            </select>
                          </td>
                          <td>
                            <Button
                              aria-label={`매핑 ${index + 1} 삭제`}
                              onClick={() =>
                                set(
                                  "identity_group_mappings",
                                  cfg.identity_group_mappings.filter(
                                    (_: Row, i: number) => i !== index,
                                  ),
                                )
                              }
                            >
                              <Trash2 size={17} />
                            </Button>
                          </td>
                        </tr>
                      );
                    },
                  )}
                </tbody>
              </table>
            </div>
          </section>
          <section className="panel padded">
            <div className="identity-section-head">
              <h2>명시적으로 연결된 계정</h2>
              <Button
                onClick={() =>
                  setLink({
                    user_id: "",
                    provider: "oidc",
                    issuer: cfg.oidc_issuer || "",
                    subject: "",
                  })
                }
              >
                <Plus size={17} /> 계정 연결
              </Button>
            </div>
            <div className="identity-table-wrap">
              <table className="identity-table">
                <thead>
                  <tr>
                    <th>사용자</th>
                    <th>제공자</th>
                    <th>발급자 / 고유 ID</th>
                    <th>연결 해제</th>
                  </tr>
                </thead>
                <tbody>
                  {links.map((row) => (
                    <tr key={row.id}>
                      <td>
                        {row.name}
                        <small>{row.email}</small>
                      </td>
                      <td>{row.provider.toUpperCase()}</td>
                      <td className="identity-break">
                        {row.issuer}
                        <small>{row.subject}</small>
                      </td>
                      <td>
                        {row.provider !== "scim" && (
                          <Button
                            disabled={busy}
                            onClick={() => {
                              if (
                                window.confirm(
                                  "연결을 해제하고 이 사용자의 모든 로그인 세션을 종료할까요?",
                                )
                              )
                                void run(async () => {
                                  await api(
                                    "/admin/identity/links/" + row.id,
                                    "DELETE",
                                  );
                                  await load();
                                });
                            }}
                          >
                            해제
                          </Button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {!links.length && (
              <Empty
                title="아직 연결된 외부 계정이 없습니다"
                text="관리자가 직접 연결하거나 허용된 제공자로 로그인하면 여기에 표시됩니다."
              />
            )}
          </section>
        </>
      )}
      {section === "ldap" && (
        <section className="panel padded">
          <h2>LDAP · Active Directory</h2>
          {toggle("ldap_enabled", "디렉터리 로그인 사용")}
          {toggle(
            "ldap_auto_register",
            "검증된 신규 디렉터리 사용자 자동 생성",
          )}
          <div className="two-columns">
            {text(
              "ldap_url",
              "디렉터리 URL",
              "ldaps://directory.company:636 또는 StartTLS를 사용하는 ldap:// URL",
            )}
            {text("ldap_base_dn", "검색 Base DN", "예: DC=company,DC=local")}
          </div>
          {toggle("ldap_starttls", "ldap:// 연결에서 StartTLS 사용 (필수)")}
          {area(
            "ldap_ca_pem",
            "사내 CA 인증서 (PEM)",
            "TLS 서버 이름과 인증서 체인을 검증합니다. 비워 두면 시스템 신뢰 저장소를 사용합니다.",
          )}
          <div className="two-columns">
            {text("ldap_bind_dn", "검색용 Bind DN")}
            {text(
              "ldap_bind_password",
              "검색 계정 비밀번호",
              "사용자 비밀번호는 저장하지 않습니다.",
              true,
            )}
          </div>
          {text(
            "ldap_user_filter",
            "사용자 검색 필터",
            "{username} 한 개가 필요하며 입력값은 LDAP 필터 규칙으로 escape됩니다. AD 예: (&(objectClass=user)(sAMAccountName={username}))",
          )}
          <div className="two-columns">
            {text(
              "ldap_subject_attribute",
              "불변 고유 ID 속성",
              "OpenLDAP: entryUUID / Active Directory: objectGUID",
            )}
            {text("ldap_email_attribute", "이메일 속성")}
            {text("ldap_name_attribute", "표시 이름 속성")}
            {text(
              "ldap_groups_attribute",
              "그룹 속성",
              "AD 기본 memberOf: 그룹 DN 전체를 매핑 값으로 사용합니다.",
            )}
          </div>
        </section>
      )}
      {section === "saml" && (
        <section className="panel padded">
          <h2>SAML 서비스 제공자</h2>
          <p className="muted">
            HTTPS 서비스 URL이 필요합니다. 서명된 HTTP-Redirect 인증 요청과
            서명된 HTTP-POST assertion을 지원합니다. IdP 시작 로그인, Artifact
            및 암호화 assertion은 활성화하지 않습니다.
          </p>
          {toggle("saml_enabled", "SAML 로그인 사용")}
          {toggle("saml_auto_register", "검증된 신규 SAML 사용자 자동 생성")}
          <Field label="SP Entity ID / Metadata URL">
            <input
              readOnly
              value={
                String(cfg.site_url || "").replace(/\/$/, "") +
                "/api/v1/auth/saml/metadata"
              }
            />
          </Field>
          <Field label="응답 수신 주소 (ACS)">
            <input
              readOnly
              value={
                String(cfg.site_url || "").replace(/\/$/, "") +
                "/api/v1/auth/saml/acs"
              }
            />
          </Field>
          <div className="identity-actions">
            <Button
              disabled={busy}
              onClick={() => {
                const rotate = !!cfg.saml_sp_private_key_configured;
                if (
                  rotate &&
                  !window.confirm(
                    "SAML 서명 인증서를 회전하면 기존 인증 요청이 무효화됩니다. 새 Metadata를 IdP에 즉시 등록할 준비가 되었나요?",
                  )
                )
                  return;
                void run(async () => {
                  await api("/admin/identity/saml/certificate", "POST", {
                    confirm: rotate,
                  });
                  await load();
                  notify("인증서를 생성했습니다. IdP에 Metadata를 등록하세요.");
                });
              }}
            >
              <RefreshCw size={17} />{" "}
              {cfg.saml_sp_private_key_configured
                ? "서명 인증서 회전"
                : "서명 인증서 생성"}
            </Button>
            {cfg.saml_sp_private_key_configured && (
              <a
                className="button"
                href="/api/v1/auth/saml/metadata"
                download="madi-saml-metadata.xml"
              >
                SP Metadata 다운로드
              </a>
            )}
          </div>
          {area(
            "saml_idp_metadata",
            "IdP Metadata XML",
            "Keycloak SAML 클라이언트 설정에서 서명 알고리즘 RSA-SHA256, 응답 바인딩 POST, assertion 암호화 끄기를 선택하세요. IdP 서명 인증서 만료 전 Metadata를 갱신하세요.",
          )}
          <div className="two-columns">
            {text("saml_email_attribute", "이메일 attribute")}
            {text("saml_name_attribute", "표시 이름 attribute")}
            {text("saml_groups_attribute", "그룹 attribute")}
          </div>
          <details>
            <summary>현재 SP 공개 인증서</summary>
            <pre className="code-example">
              {cfg.saml_sp_certificate || "아직 생성되지 않았습니다."}
            </pre>
          </details>
        </section>
      )}
      {section === "scim" && (
        <>
          <section className="panel padded">
            <h2>SCIM 2.0 프로비저닝</h2>
            {toggle("scim_enabled", "SCIM 사용자·그룹 동기화 사용")}
            <Toggle
              checked={(cfg.allowed_key_scopes || []).includes(
                "identity:provision",
              )}
              onChange={(value) =>
                set(
                  "allowed_key_scopes",
                  value
                    ? [
                        ...new Set([
                          ...(cfg.allowed_key_scopes || []),
                          "identity:provision",
                        ]),
                      ]
                    : cfg.allowed_key_scopes.filter(
                        (v: string) => v !== "identity:provision",
                      ),
                )
              }
              label="서비스 계정 키에 SCIM 관리 권한 발급 허용"
            />
            <Field label="SCIM 기본 URL">
              <input
                readOnly
                value={
                  String(cfg.site_url || "").replace(/\/$/, "") +
                  "/api/v1/scim/v2"
                }
              />
            </Field>
            <p className="muted">
              Users · Groups, 추가/조회/변경/비활성화/삭제, PATCH 및 ETag를
              지원합니다. 키마다 워크스페이스가 고정되며 기존 로컬 계정은 자동
              편입하지 않습니다. 삭제는 관리 사용자 비활성화와 SCIM 목록 제외로
              처리해 문서 소유권을 보존합니다.
            </p>
            <p className="muted">
              필터는 userName/displayName/externalId/id eq "값"을 지원하며 한
              페이지 최대 200개입니다. Bulk, 암호 변경, 중첩 그룹, Enterprise
              schema 확장 및 서버 정렬은 지원하지 않습니다.
            </p>
          </section>
          <section className="panel padded">
            <div className="identity-section-head">
              <h2>프로비저닝 서비스 계정 키</h2>
              <Button
                disabled={!owner}
                onClick={() =>
                  setKeyDraft({
                    name: "디렉터리 프로비저닝",
                    workspace_id: "",
                    scopes: ["identity:provision"],
                    expires_in_days: 90,
                    rate_limit: 120,
                    ips: "",
                  })
                }
              >
                <Plus size={17} /> 키 발급
              </Button>
            </div>
            <Field label="서비스 계정">
              <select
                value={owner}
                onChange={(e) => {
                  setOwner(e.target.value);
                  setKeyDraft(null);
                }}
              >
                <option value="">서비스 계정 선택</option>
                {users
                  .filter((user) => user.kind === "service" && !user.disabled)
                  .map((user) => (
                    <option key={user.id} value={user.id}>
                      {user.name} · {user.email}
                    </option>
                  ))}
              </select>
            </Field>
            <p className="muted">
              서비스 계정은 관리자 사용자 메뉴에서 생성한 뒤 대상 워크스페이스의
              멤버로 추가하세요. 설정 변경은 아래 저장 버튼으로 먼저 적용합니다.
            </p>
            <div className="identity-table-wrap">
              <table className="identity-table">
                <thead>
                  <tr>
                    <th>키</th>
                    <th>권한 / 만료</th>
                    <th>관리</th>
                  </tr>
                </thead>
                <tbody>
                  {keys.map((key) => (
                    <tr key={key.id}>
                      <td>
                        {key.name}
                        <small>{key.prefix}…</small>
                      </td>
                      <td>
                        {key.scopes
                          .map((v: string) => identityScopes[v] || v)
                          .join(", ")}
                        <small>
                          {datetime(key.expires_at)}
                          {key.revoked_at ? " · 폐기됨" : ""}
                        </small>
                      </td>
                      <td>
                        {!key.revoked_at && (
                          <div className="identity-actions">
                            <Button
                              onClick={() =>
                                setKeyDraft({
                                  ...key,
                                  expires_in_days: Math.max(
                                    1,
                                    Math.ceil(
                                      (Date.parse(key.expires_at) -
                                        Date.now()) /
                                        86400000,
                                    ),
                                  ),
                                  ips: key.ip_allowlist.join(", "),
                                })
                              }
                            >
                              권한 변경
                            </Button>
                            <Button
                              disabled={busy}
                              onClick={() => {
                                if (
                                  window.confirm(
                                    "키를 회전하고 이전 키를 즉시 폐기할까요?",
                                  )
                                )
                                  void run(async () => {
                                    const result = await api(
                                      "/keys/" + key.id + "/rotate",
                                      "POST",
                                      {},
                                    );
                                    setToken(result.token);
                                    await loadKeys();
                                  });
                              }}
                            >
                              회전
                            </Button>
                            <Button
                              disabled={busy}
                              onClick={() => {
                                if (
                                  window.confirm(
                                    "이 키의 접근을 즉시 차단할까요?",
                                  )
                                )
                                  void run(async () => {
                                    await api("/keys/" + key.id, "DELETE");
                                    await loadKeys();
                                  });
                              }}
                            >
                              폐기
                            </Button>
                          </div>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        </>
      )}
      <div className="identity-save">
        <span className="muted">
          {Object.keys(changes).length
            ? "아직 저장하지 않은 설정이 있습니다."
            : "현재 저장된 설정입니다."}
        </span>
        <Button
          variant="primary"
          disabled={busy || !Object.keys(changes).length}
          onClick={() => void save()}
        >
          {busy ? "처리 중…" : "인증 설정 저장"}
        </Button>
      </div>
      <Modal
        open={!!link}
        onOpenChange={(open) => !open && setLink(null)}
        title="외부 계정 직접 연결"
        description="신뢰할 수 있는 제공자에서 확인한 불변 고유 ID를 입력하세요."
      >
        {link && (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              void run(async () => {
                await api("/admin/identity/links", "POST", link);
                setLink(null);
                await load();
              });
            }}
          >
            <Field label="사용자">
              <select
                required
                value={link.user_id}
                onChange={(e) => setLink({ ...link, user_id: e.target.value })}
              >
                <option value="">사용자 선택</option>
                {users
                  .filter((user) => user.kind === "user" && !user.disabled)
                  .map((user) => (
                    <option key={user.id} value={user.id}>
                      {user.name} · {user.email}
                    </option>
                  ))}
              </select>
            </Field>
            <Field label="제공자">
              <select
                value={link.provider}
                onChange={(e) =>
                  setLink({
                    ...link,
                    provider: e.target.value,
                    issuer:
                      e.target.value === "oidc"
                        ? cfg.oidc_issuer || ""
                        : e.target.value === "ldap"
                          ? `${cfg.ldap_url || ""}|${cfg.ldap_base_dn || ""}`
                          : "",
                  })
                }
              >
                <option value="oidc">OIDC / Keycloak</option>
                <option value="ldap">LDAP / AD</option>
                <option value="saml">SAML</option>
              </select>
            </Field>
            <Field label="Issuer / 디렉터리 식별자">
              <input
                required
                value={link.issuer}
                onChange={(e) => setLink({ ...link, issuer: e.target.value })}
              />
              <small className="muted">
                LDAP은 URL|BaseDN, SAML은 IdP Entity ID를 입력하세요.
              </small>
            </Field>
            <Field label="고유 ID (OIDC sub / LDAP entryUUID·objectGUID / SAML NameID)">
              <input
                required
                value={link.subject}
                onChange={(e) => setLink({ ...link, subject: e.target.value })}
              />
              <small className="muted">
                AD objectGUID는 원시 16바이트를 URL-safe Base64(padding 없음)로
                변환한 값입니다.
              </small>
            </Field>
            <Button variant="primary" disabled={busy}>
              계정 연결
            </Button>
          </form>
        )}
      </Modal>
      <Modal
        open={!!keyDraft}
        onOpenChange={(open) => !open && setKeyDraft(null)}
        title={keyDraft?.id ? "서비스 키 권한 변경" : "프로비저닝 키 발급"}
      >
        {keyDraft && (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              void run(async () => {
                const result = await api(
                  "/keys" + (keyDraft.id ? "/" + keyDraft.id : ""),
                  keyDraft.id ? "PUT" : "POST",
                  {
                    ...keyDraft,
                    user_id: owner,
                    ip_allowlist: keyDraft.ips
                      .split(",")
                      .map((v: string) => v.trim())
                      .filter(Boolean),
                  },
                );
                setKeyDraft(null);
                if (result.token) setToken(result.token);
                await loadKeys();
              });
            }}
          >
            <Field label="키 이름">
              <input
                required
                value={keyDraft.name}
                onChange={(e) =>
                  setKeyDraft({ ...keyDraft, name: e.target.value })
                }
              />
            </Field>
            <Field label="키 워크스페이스">
              <select
                required
                disabled={!!keyDraft.id}
                value={keyDraft.workspace_id}
                onChange={(e) =>
                  setKeyDraft({ ...keyDraft, workspace_id: e.target.value })
                }
              >
                <option value="">워크스페이스 선택</option>
                {workspaces.map((ws) => (
                  <option key={ws.id} value={ws.id}>
                    {ws.name}
                  </option>
                ))}
              </select>
            </Field>
            <fieldset className="scope-fieldset">
              <legend>키 권한</legend>
              {Object.entries(identityScopes).map(([scope, label]) => (
                <label key={scope}>
                  <input
                    type="checkbox"
                    checked={keyDraft.scopes.includes(scope)}
                    onChange={(e) =>
                      setKeyDraft({
                        ...keyDraft,
                        scopes: e.target.checked
                          ? [...keyDraft.scopes, scope]
                          : keyDraft.scopes.filter((v: string) => v !== scope),
                      })
                    }
                  />
                  {label}
                </label>
              ))}
            </fieldset>
            <div className="two-columns">
              <Field label="유효기간 (일)">
                <input
                  required
                  type="number"
                  min={1}
                  max={3650}
                  value={keyDraft.expires_in_days}
                  onChange={(e) =>
                    setKeyDraft({
                      ...keyDraft,
                      expires_in_days: Number(e.target.value),
                    })
                  }
                />
              </Field>
              <Field label="분당 요청 제한">
                <input
                  required
                  type="number"
                  min={1}
                  max={10000}
                  value={keyDraft.rate_limit}
                  onChange={(e) =>
                    setKeyDraft({
                      ...keyDraft,
                      rate_limit: Number(e.target.value),
                    })
                  }
                />
              </Field>
            </div>
            <Field label="허용 IP / CIDR (쉼표 구분)">
              <input
                value={keyDraft.ips}
                onChange={(e) =>
                  setKeyDraft({ ...keyDraft, ips: e.target.value })
                }
              />
            </Field>
            <Button variant="primary" disabled={busy}>
              저장
            </Button>
          </form>
        )}
      </Modal>
      <Modal
        open={!!token}
        onOpenChange={(open) => !open && setToken("")}
        title="발급된 키를 안전하게 보관하세요"
        description="비밀 키 원문은 이번에만 표시됩니다. 서버에는 해시만 저장합니다."
      >
        <Field label="API 비밀 키">
          <input readOnly value={token} />
        </Field>
        <CopyButton value={token} />
        <p className="muted">Authorization: Bearer 헤더에 사용하세요.</p>
        <Button onClick={() => setToken("")}>
          <KeyRound size={17} /> 안전하게 보관했어요
        </Button>
      </Modal>
    </>
  );
}
