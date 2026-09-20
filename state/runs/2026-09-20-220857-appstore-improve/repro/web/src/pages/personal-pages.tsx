import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Copy, Plus, RefreshCw, UserRound, XCircle } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTheme, type ThemePreference } from "../app/providers";
import { api } from "../lib/api";
import { formatDate, formatDateTime } from "../lib/utils";
import type { PersonalKey } from "../types";
import { AppCard } from "../features/apps/app-card";
import { RejectionNotice } from "../features/apps/rejection-notice";
import {
  Badge,
  Button,
  ButtonLink,
  Card,
  Dialog,
  EmptyState,
  ErrorState,
  Field,
  Input,
  LoadingState,
  PageHeader,
  Select,
  Switch,
} from "../components/ui";

export function MyDashboardPage() {
  const apps = useQuery({
    queryKey: ["my-apps"],
    queryFn: ({ signal }) => api.myApps(signal),
  });
  const keys = useQuery({
    queryKey: ["my-keys"],
    queryFn: ({ signal }) => api.myKeys(signal),
  });
  return (
    <div className="page">
      <PageHeader
        eyebrow="Personal workspace"
        title="내 AppStore"
        description="등록한 앱과 개인 키의 상태를 한눈에 확인하세요."
      />
      <div className="stats-grid">
        <Card className="stat-card">
          <span className="stat-label">내 앱</span>
          <strong className="stat-value">{apps.data?.length ?? "—"}</strong>
        </Card>
        <Card className="stat-card">
          <span className="stat-label">활성 키</span>
          <strong className="stat-value">
            {keys.data?.filter((key) => !key.revokedAt).length ?? "—"}
          </strong>
        </Card>
        <Card className="stat-card">
          <span className="stat-label">검토 대기</span>
          <strong className="stat-value">
            {apps.data?.filter((app) => app.status === "pending_review")
              .length ?? "—"}
          </strong>
        </Card>
        <Card className="stat-card">
          <span className="stat-label">게시 앱</span>
          <strong className="stat-value">
            {apps.data?.filter((app) => app.status === "published").length ??
              "—"}
          </strong>
        </Card>
      </div>
      <div className="section-header">
        <h2 className="section-title">최근 등록 앱</h2>
        <ButtonLink to="/submit" size="sm">
          <Plus size={16} /> 앱 등록
        </ButtonLink>
      </div>
      {apps.isPending && <LoadingState />}
      {apps.error && (
        <ErrorState error={apps.error} retry={() => void apps.refetch()} />
      )}
      {apps.data && !apps.data.length && (
        <EmptyState
          title="등록한 앱이 없습니다"
          actions={<ButtonLink to="/submit">첫 앱 등록</ButtonLink>}
        />
      )}
      {apps.data && (
        <div className="card-grid">
          {apps.data.slice(0, 3).map((app) => (
            <div key={app.id}>
              <AppCard app={app} showStatus />
              <RejectionNotice app={app} className="mt-2" />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export function MyAppsPage() {
  const apps = useQuery({
    queryKey: ["my-apps"],
    queryFn: ({ signal }) => api.myApps(signal),
  });
  return (
    <div className="page">
      <PageHeader
        eyebrow="My applications"
        title="내가 등록한 앱"
        description="앱 상태와 최신 변경 내용을 관리합니다."
        actions={
          <ButtonLink to="/submit">
            <Plus size={18} /> 앱 등록
          </ButtonLink>
        }
      />
      {apps.isPending && <LoadingState />}
      {apps.error && (
        <ErrorState error={apps.error} retry={() => void apps.refetch()} />
      )}
      {apps.data && !apps.data.length && (
        <EmptyState actions={<ButtonLink to="/submit">앱 등록</ButtonLink>} />
      )}
      {apps.data && (
        <div className="card-grid">
          {apps.data.map((app) => (
            <div key={app.id}>
              <AppCard app={app} showStatus />
              <RejectionNotice app={app} className="mt-2" />
              <div className="mt-2 flex justify-end gap-2">
                {/* A draft is where an app waits for its security review, so
                    that is where the owner needs the way in. */}
                {(app.status === "draft" || app.securityVerified) && (
                  <ButtonLink
                    variant="secondary"
                    size="sm"
                    to={`/my/apps/${app.id}/security`}
                  >
                    보안 심의
                  </ButtonLink>
                )}
                <ButtonLink
                  variant="secondary"
                  size="sm"
                  to={`/my/apps/${app.id}/edit`}
                >
                  수정
                </ButtonLink>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

const defaultPermissions = [
  "apps:read",
  "apps:write",
  "apps:update",
  "apps:submit",
  "favorites:read",
  "favorites:write",
  "ai:use",
  "mcp:read",
  "mcp:execute",
];

export function MyKeysPage() {
  const client = useQueryClient();
  const keys = useQuery({
    queryKey: ["my-keys"],
    queryFn: ({ signal }) => api.myKeys(signal),
  });
  const keyOptions = useQuery({
    queryKey: ["key-permission-options"],
    queryFn: ({ signal }) => api.keyPermissionOptions(signal),
  });
  const [createOpen, setCreateOpen] = useState(false);
  const [name, setName] = useState("");
  const [type, setType] = useState<"api" | "mcp">("api");
  const [permissions, setPermissions] = useState<string[]>(["apps:read"]);
  const [revealed, setRevealed] = useState<PersonalKey>();
  const availablePermissions = useMemo(
    () =>
      keyOptions.data?.permissions?.filter((permission) => permission.active) ??
      defaultPermissions.map((key) => ({
        key,
        name: key,
        description: "",
        active: true,
      })),
    [keyOptions.data],
  );
  const create = useMutation({
    mutationFn: () => api.createKey({ name, type, permissions }),
    onSuccess: async (created) => {
      setCreateOpen(false);
      setRevealed(created);
      setName("");
      await client.invalidateQueries({ queryKey: ["my-keys"] });
    },
  });
  const rotate = useMutation({
    mutationFn: (id: string) => api.rotateKey(id),
    onSuccess: async (created) => {
      setRevealed(created);
      await client.invalidateQueries({ queryKey: ["my-keys"] });
    },
  });
  const revoke = useMutation({
    mutationFn: (id: string) => api.revokeKey(id),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ["my-keys"] });
    },
  });

  return (
    <div className="page">
      <PageHeader
        eyebrow="Personal key management"
        title="API · MCP 키"
        description="키 원문은 생성 순간 한 번만 표시됩니다. 정기적으로 회전하고 필요한 권한만 부여하세요."
        actions={
          <Button onClick={() => setCreateOpen(true)}>
            <Plus size={18} /> 새 키
          </Button>
        }
      />
      {revealed?.secret && (
        <div className="secret-reveal mb-5" role="alert">
          <strong>지금 키를 복사하세요. 닫으면 다시 볼 수 없습니다.</strong>
          <code className="secret-code">{revealed.secret}</code>
          <Button
            variant="secondary"
            size="sm"
            onClick={() =>
              void navigator.clipboard.writeText(revealed.secret ?? "")
            }
          >
            <Copy size={16} /> 복사
          </Button>{" "}
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setRevealed(undefined)}
          >
            확인
          </Button>
        </div>
      )}
      {keys.isPending && <LoadingState />}
      {keys.error && (
        <ErrorState error={keys.error} retry={() => void keys.refetch()} />
      )}
      {keys.data && !keys.data.length && (
        <EmptyState
          title="발급된 키가 없습니다"
          actions={
            <Button onClick={() => setCreateOpen(true)}>첫 키 만들기</Button>
          }
        />
      )}
      {keys.data && !!keys.data.length && (
        <Card className="data-card">
          <div className="table-scroll">
            <table className="data-table">
              <thead>
                <tr>
                  <th>이름</th>
                  <th>Prefix</th>
                  <th>권한</th>
                  <th>만료</th>
                  <th>최근 사용</th>
                  <th>작업</th>
                </tr>
              </thead>
              <tbody>
                {keys.data.map((key) => (
                  <tr key={key.id}>
                    <td>
                      <strong>{key.name}</strong>
                      <div>
                        <Badge
                          tone={
                            key.revokedAt
                              ? "danger"
                              : key.rotationGraceEndsAt
                                ? "warning"
                                : "positive"
                          }
                        >
                          {key.revokedAt
                            ? "폐기됨"
                            : key.rotationGraceEndsAt
                              ? "회전 유예"
                              : (key.type ?? "api").toUpperCase()}
                        </Badge>
                      </div>
                    </td>
                    <td>
                      <code>{key.prefix}••••</code>
                    </td>
                    <td>
                      {key.permissions.slice(0, 2).join(", ")}
                      {key.permissions.length > 2
                        ? ` +${key.permissions.length - 2}`
                        : ""}
                    </td>
                    <td>{formatDate(key.expiresAt)}</td>
                    <td>{formatDateTime(key.lastUsedAt)}</td>
                    <td>
                      <div className="top-actions">
                        <Button
                          variant="secondary"
                          size="sm"
                          disabled={!!key.revokedAt || rotate.isPending}
                          onClick={() => rotate.mutate(key.id)}
                        >
                          <RefreshCw size={15} /> 회전
                        </Button>
                        <Button
                          variant="danger"
                          size="sm"
                          disabled={!!key.revokedAt || revoke.isPending}
                          onClick={() => {
                            if (confirm(`${key.name} 키를 폐기할까요?`))
                              revoke.mutate(key.id);
                          }}
                        >
                          <XCircle size={15} /> 폐기
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}
      <Dialog
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title="새 개인 키"
        description="용도에 필요한 최소 권한만 선택하세요."
      >
        <form
          onSubmit={(event) => {
            event.preventDefault();
            create.mutate();
          }}
        >
          <Field label="키 이름" id="key-name">
            <Input
              id="key-name"
              value={name}
              onChange={(event) => setName(event.target.value)}
              required
              maxLength={80}
            />
          </Field>
          <Field label="키 유형" id="key-type">
            <Select
              id="key-type"
              value={type}
              onChange={(event) => setType(event.target.value as "api" | "mcp")}
            >
              <option value="api">API Key</option>
              <option value="mcp">MCP Key</option>
            </Select>
          </Field>
          {!!keyOptions.data?.templates.length && (
            <Field label="Permission Template" id="key-template">
              <Select
                id="key-template"
                defaultValue=""
                onChange={(event) => {
                  const template = keyOptions.data.templates.find(
                    (item) => item.id === event.target.value,
                  );
                  if (template) setPermissions(template.permissions);
                }}
              >
                <option value="">직접 선택</option>
                {keyOptions.data.templates.map((template) => (
                  <option key={template.id} value={template.id}>
                    {template.name}
                  </option>
                ))}
              </Select>
            </Field>
          )}
          {keyOptions.data?.policy && (
            <div className="notice mb-4">
              최대 {keyOptions.data.policy.maxKeys}개 · 기본 만료{" "}
              {keyOptions.data.policy.defaultExpiryDays}일 · 회전 유예{" "}
              {keyOptions.data.policy.rotationGraceDays}일
            </div>
          )}
          {keyOptions.error && (
            <div className="notice notice-danger mb-4">
              최신 권한 정의를 불러오지 못해 기본 목록을 표시합니다.
            </div>
          )}
          <fieldset className="field">
            <legend className="field-label">권한</legend>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
              {availablePermissions.map((permission) => (
                <label key={permission.key} className="notice">
                  <input
                    type="checkbox"
                    checked={permissions.includes(permission.key)}
                    onChange={(event) =>
                      setPermissions((current) =>
                        event.target.checked
                          ? [...new Set([...current, permission.key])]
                          : current.filter((item) => item !== permission.key),
                      )
                    }
                  />{" "}
                  <span>
                    {permission.name}
                    {permission.description && (
                      <span className="field-help block">
                        {permission.description}
                      </span>
                    )}
                  </span>
                </label>
              ))}
            </div>
          </fieldset>
          {create.error && (
            <p className="field-error" role="alert">
              {create.error.message}
            </p>
          )}
          <div className="form-actions">
            <Button variant="secondary" onClick={() => setCreateOpen(false)}>
              취소
            </Button>
            <Button
              type="submit"
              disabled={create.isPending || !name || !permissions.length}
            >
              생성
            </Button>
          </div>
        </form>
      </Dialog>
    </div>
  );
}

export function ProfilePage() {
  const me = useQuery({
    queryKey: ["me"],
    queryFn: ({ signal }) => api.me(signal),
  });
  return (
    <div className="page">
      <PageHeader
        eyebrow="Profile"
        title="내 프로필"
        description="SSO에서 동기화된 사용자 정보와 현재 역할을 확인합니다."
      />
      {me.isPending && <LoadingState />}
      {me.error && (
        <ErrorState error={me.error} retry={() => void me.refetch()} />
      )}
      {me.data && (
        <Card className="form-section">
          <div className="flex items-center gap-4 mb-7">
            <span className="avatar !w-16 !h-16 !text-lg">
              <UserRound />
            </span>
            <div>
              <h2 className="section-title">
                {me.data.displayName || me.data.username}
              </h2>
              <p className="page-description">{me.data.email}</p>
            </div>
          </div>
          <dl className="meta-list">
            <div className="meta-row">
              <dt>사용자명</dt>
              <dd>{me.data.username}</dd>
            </div>
            <div className="meta-row">
              <dt>팀</dt>
              <dd>{me.data.team || "—"}</dd>
            </div>
            <div className="meta-row">
              <dt>역할</dt>
              <dd className="badge-row !mt-0 justify-end">
                {me.data.roles.map((role) => (
                  <Badge key={role}>{role}</Badge>
                ))}
              </dd>
            </div>
          </dl>
        </Card>
      )}
    </div>
  );
}

export function ActivityPage() {
  const activity = useQuery({
    queryKey: ["me-activity"],
    queryFn: ({ signal }) => api.myActivity(signal),
  });
  return (
    <div className="page">
      <PageHeader
        eyebrow="Activity"
        title="내 활동 내역"
        description="앱과 키에 관련된 최근 활동입니다."
      />
      {activity.isPending && <LoadingState />}
      {activity.error && (
        <ErrorState
          error={activity.error}
          retry={() => void activity.refetch()}
        />
      )}
      {activity.data && !activity.data.length && <EmptyState />}
      {activity.data && !!activity.data.length && (
        <Card className="prose-card">
          <pre>{JSON.stringify(activity.data, null, 2)}</pre>
        </Card>
      )}
    </div>
  );
}

export function PersonalSettingsPage() {
  const themeContext = useTheme();
  const settings = useQuery({
    queryKey: ["my-settings"],
    queryFn: ({ signal }) => api.mySettings(signal),
  });
  const [theme, setTheme] = useState<ThemePreference>("system");
  const [language, setLanguage] = useState("ko");
  const [reducedMotion, setReducedMotion] = useState(false);
  const [compactCards, setCompactCards] = useState(false);
  useEffect(() => {
    if (!settings.data) return;
    setTheme(settings.data.theme);
    setLanguage(settings.data.language);
    setReducedMotion(settings.data.reducedMotion);
    setCompactCards(settings.data.compactCards);
  }, [settings.data]);
  const save = useMutation({
    mutationFn: () =>
      api.updateMySettings({ theme, language, reducedMotion, compactCards }),
    onSuccess: (saved) => {
      themeContext.setPreference(saved.theme);
    },
  });
  return (
    <div className="page">
      <PageHeader
        eyebrow="Preferences"
        title="개인 설정"
        description="이 설정은 개인 화면에만 적용됩니다."
      />
      {settings.isPending && <LoadingState />}
      {settings.error && (
        <ErrorState
          error={settings.error}
          retry={() => void settings.refetch()}
        />
      )}
      {settings.data && (
        <Card className="form-section">
          <div className="form-grid">
            <Field label="Theme" id="personal-theme">
              <Select
                id="personal-theme"
                value={theme}
                onChange={(event) =>
                  setTheme(event.target.value as ThemePreference)
                }
              >
                <option value="system">System</option>
                <option value="light">Light</option>
                <option value="dark">Dark</option>
              </Select>
            </Field>
            <Field label="Language" id="personal-language">
              <Select
                id="personal-language"
                value={language}
                onChange={(event) => setLanguage(event.target.value)}
              >
                <option value="ko">한국어</option>
                <option value="en">English</option>
              </Select>
            </Field>
          </div>
          <div className="switch-row">
            <div>
              <strong>모션 줄이기</strong>
              <div className="field-help">
                애니메이션과 전환 효과를 최소화합니다.
              </div>
            </div>
            <Switch
              checked={reducedMotion}
              onChange={setReducedMotion}
              label="모션 줄이기"
            />
          </div>
          <div className="switch-row">
            <div>
              <strong>간결한 카드</strong>
              <div className="field-help">
                개인 앱 화면의 카드 간격을 줄입니다.
              </div>
            </div>
            <Switch
              checked={compactCards}
              onChange={setCompactCards}
              label="간결한 카드"
            />
          </div>
          {save.error && (
            <div className="notice notice-danger">{save.error.message}</div>
          )}
          {save.isSuccess && (
            <div className="notice">개인 설정이 저장되었습니다.</div>
          )}
          <div className="form-actions">
            <Button disabled={save.isPending} onClick={() => save.mutate()}>
              {save.isPending ? "저장 중…" : "저장"}
            </Button>
          </div>
        </Card>
      )}
    </div>
  );
}
