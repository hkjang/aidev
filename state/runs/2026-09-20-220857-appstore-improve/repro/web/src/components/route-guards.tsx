import { Navigate, Outlet, useLocation } from "react-router-dom";
import { hasAnyRole } from "../lib/utils";
import { useAuth } from "../app/providers";
import { AppShell } from "./app-shell";
import { ErrorState, ForbiddenState, LoadingState } from "./ui";

export function PublicLayout() {
  return (
    <AppShell>
      <Outlet />
    </AppShell>
  );
}

export function RequireAuth({
  roles,
  admin = false,
}: {
  roles?: string[];
  admin?: boolean;
}) {
  const auth = useAuth();
  const location = useLocation();
  if (auth.isPending)
    return (
      <div className="page">
        <LoadingState label="인증 상태를 확인하는 중입니다" />
      </div>
    );
  if (auth.error)
    return (
      <div className="page">
        <ErrorState error={auth.error} retry={() => void auth.refresh()} />
      </div>
    );
  if (!auth.session?.authenticated)
    return (
      <Navigate
        to={`/login?returnTo=${encodeURIComponent(location.pathname + location.search)}`}
        replace
      />
    );
  if (roles && !hasAnyRole(auth.session.user?.roles, roles))
    return (
      <div className="page">
        <ForbiddenState />
      </div>
    );
  return (
    <AppShell admin={admin}>
      <Outlet />
    </AppShell>
  );
}
