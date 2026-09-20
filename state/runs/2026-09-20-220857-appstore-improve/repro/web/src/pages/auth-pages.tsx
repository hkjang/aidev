import { useMutation, useQuery } from "@tanstack/react-query";
import {
  Boxes,
  ChevronDown,
  LockKeyhole,
  LogIn,
  ShieldCheck,
  Sparkles,
} from "lucide-react";
import { useEffect, useState } from "react";
import { Navigate, useNavigate, useSearchParams } from "react-router-dom";
import { useAuth } from "../app/providers";
import { api } from "../lib/api";
import { Button, Card, ErrorState, Field, Input } from "../components/ui";
import { safeReturnTo } from "../lib/utils";

export function LoginPage() {
  const [params] = useSearchParams();
  const returnTo = safeReturnTo(params.get("returnTo"));
  // The OIDC callback lands here with sso=none when a silent (prompt=none)
  // attempt found no identity-provider session; the marker only explains the
  // screen and never changes where the buttons go.
  const ssoRefused = params.get("sso") === "none";
  const auth = useAuth();
  const navigate = useNavigate();
  const config = useQuery({
    queryKey: ["public-config"],
    queryFn: ({ signal }) => api.publicConfig(signal),
    retry: false,
  });
  const version = useQuery({
    queryKey: ["version"],
    queryFn: ({ signal }) => api.version(signal),
    staleTime: Infinity,
    retry: false,
  });
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [localLoginOpen, setLocalLoginOpen] = useState(false);
  const bootstrap = useMutation({
    mutationFn: () => api.bootstrapLogin(username, password),
    onSuccess: async () => {
      await auth.refresh();
      navigate(returnTo, { replace: true });
    },
  });

  useEffect(() => {
    document.title = `${config.data?.siteName || "AppStore"} 로그인`;
  }, [config.data?.siteName]);

  if (auth.session?.authenticated) return <Navigate to={returnTo} replace />;
  const oidcEnabled = config.data?.oidcEnabled && config.data?.oidcConfigured;
  // The local admin remains the recovery path when the identity provider is
  // unreachable, so it stays reachable after SSO is configured. Older servers
  // that predate bootstrapAvailable fall back to the SSO-not-configured test.
  const showBootstrap =
    auth.session?.bootstrapAvailable ??
    (auth.session?.bootstrapRequired || !config.data?.oidcConfigured);
  const bootstrapIsFallback = showBootstrap && oidcEnabled;
  const bootstrapOpen = !bootstrapIsFallback || localLoginOpen;
  const oidcUrl = `/api/v1/auth/oidc/login?returnTo=${encodeURIComponent(returnTo)}`;

  return (
    <main className="login-page">
      <section className="login-visual" aria-labelledby="login-hero-title">
        <div className="login-copy">
          <span className="badge badge-primary">
            <Sparkles size={15} /> 사내 앱 스토어
          </span>
          <h1 id="login-hero-title">
            발견하고,
            <br />
            안전하게 관리하세요.
          </h1>
          <p>
            일반 탐색은 로그인 없이 자유롭게. 등록과 개인 키, 운영 설정은 명확한
            권한 아래에서 관리합니다.
          </p>
        </div>
      </section>
      <section className="login-panel" aria-labelledby="login-title">
        <Card className="login-card">
          {config.data?.logoUrl ? (
            <img
              className="brand-logo mb-5"
              src={config.data.logoUrl}
              alt=""
              aria-hidden="true"
            />
          ) : (
            <span className="brand-mark mb-5" aria-hidden="true">
              A
            </span>
          )}
          <p className="eyebrow">Secure access</p>
          <h2 className="section-title" id="login-title">
            {config.data?.siteName || "AppStore"} 로그인
          </h2>
          <p className="page-description mb-7">
            인증 후 원래 화면으로 안전하게 돌아갑니다.
          </p>
          {(config.error || auth.error) && (
            <ErrorState
              error={config.error || auth.error}
              retry={() => {
                void config.refetch();
                void auth.refresh();
              }}
            />
          )}
          {oidcEnabled && ssoRefused && (
            <div className="notice mb-5" role="status">
              <ShieldCheck size={20} />
              <span>
                회사 계정 세션이 없어 자동으로 로그인하지 않았습니다. 아래
                버튼으로 로그인하세요.
              </span>
            </div>
          )}
          {oidcEnabled && (
            <a className="button button-primary w-full" href={oidcUrl}>
              <ShieldCheck size={19} /> 회사 계정으로 SSO 로그인
            </a>
          )}
          {bootstrapIsFallback && (
            <button
              type="button"
              className="local-login-toggle"
              aria-expanded={localLoginOpen}
              aria-controls="local-login-form"
              onClick={() => setLocalLoginOpen((open) => !open)}
            >
              <LockKeyhole size={17} />
              <span>관리자 계정으로 로그인</span>
              <ChevronDown
                size={17}
                className={localLoginOpen ? "rotate-180" : ""}
              />
            </button>
          )}
          {showBootstrap && bootstrapOpen && (
            <form
              id="local-login-form"
              className={oidcEnabled ? "mt-5" : ""}
              onSubmit={(event) => {
                event.preventDefault();
                bootstrap.mutate();
              }}
            >
              <div className="notice mb-5">
                <LockKeyhole size={20} />
                <span>
                  {oidcEnabled
                    ? "SSO를 사용할 수 없을 때를 위한 복구용 관리자 계정입니다. 평소에는 회사 계정으로 로그인하세요."
                    : "최초 설치 관리자 계정입니다. SSO를 설정한 뒤에도 복구용으로 계속 사용할 수 있습니다."}
                </span>
              </div>
              <Field label="Bootstrap 관리자" id="bootstrap-user">
                <Input
                  id="bootstrap-user"
                  value={username}
                  onChange={(event) => setUsername(event.target.value)}
                  required
                  autoComplete="username"
                />
              </Field>
              <Field label="비밀번호" id="bootstrap-password">
                <Input
                  id="bootstrap-password"
                  type="password"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  required
                  autoComplete="current-password"
                />
              </Field>
              {bootstrap.error && (
                <p className="field-error mb-4" role="alert">
                  {bootstrap.error.message}
                </p>
              )}
              <Button
                type="submit"
                className="w-full"
                disabled={bootstrap.isPending}
              >
                <LogIn size={18} />{" "}
                {bootstrap.isPending ? "확인 중…" : "관리자 로그인"}
              </Button>
            </form>
          )}
          {!oidcEnabled && !showBootstrap && (
            <div className="notice notice-danger">
              사용 가능한 로그인 방식이 없습니다. Bootstrap 관리자에게
              문의하세요.
            </div>
          )}
          <div className="login-version">
            <Boxes size={15} className="inline mr-1" /> AppStore{" "}
            {version.data?.version ?? "버전 확인 중"}
            {version.data?.commit
              ? ` · ${version.data.commit.slice(0, 8)}`
              : ""}
          </div>
        </Card>
      </section>
    </main>
  );
}
