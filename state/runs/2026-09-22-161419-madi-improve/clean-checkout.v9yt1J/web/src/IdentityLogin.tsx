import { useState } from "react";
import { ShieldCheck } from "lucide-react";
import { api, type User } from "./api";
import { Button, ErrorBox, Field } from "./ui";

export default function IdentityLogin({
  info,
  onLogin,
}: {
  info: Record<string, any>;
  onLogin: (user: User) => void;
}) {
  const [open, setOpen] = useState(false),
    [username, setUsername] = useState(""),
    [password, setPassword] = useState(""),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  if (!info.ldap_enabled && !info.saml_enabled) return null;
  return (
    <div className="identity-login">
      {!info.oidc_enabled && (
        <div className="divider-label">또는 회사 계정</div>
      )}
      {info.saml_enabled && (
        <a className="button full" href="/api/v1/auth/saml/start">
          <ShieldCheck size={18} /> SAML 회사 계정으로 로그인
        </a>
      )}
      {info.ldap_enabled && (
        <>
          <Button
            type="button"
            className="full"
            aria-expanded={open}
            onClick={() => {
              setOpen(!open);
              setError("");
              setPassword("");
            }}
          >
            <ShieldCheck size={18} /> 디렉터리 계정으로 로그인
          </Button>
          {open && (
            <form
              onSubmit={async (e) => {
                e.preventDefault();
                setBusy(true);
                setError("");
                try {
                  onLogin(
                    await api<User>("/auth/ldap/login", "POST", {
                      username,
                      password,
                    }),
                  );
                  setPassword("");
                } catch (e) {
                  setError((e as Error).message);
                } finally {
                  setBusy(false);
                }
              }}
            >
              <ErrorBox error={error} />
              <Field label="디렉터리 사용자명">
                <input
                  required
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  autoComplete="username"
                  maxLength={254}
                />
              </Field>
              <Field label="디렉터리 비밀번호">
                <input
                  required
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="current-password"
                  maxLength={4096}
                />
              </Field>
              <Button className="full" variant="primary" disabled={busy}>
                {busy ? "인증 중…" : "디렉터리 로그인"}
              </Button>
            </form>
          )}
        </>
      )}
    </div>
  );
}
