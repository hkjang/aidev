import { SlidersHorizontal } from "lucide-react";
import { Link } from "react-router-dom";
import { useAuth } from "../../app/providers";
import { hasAnyRole } from "../../lib/utils";

const ADMIN_ROLES = ["admin", "super_admin"];

export function useCanManageApps(): boolean {
  const { session } = useAuth();
  return (
    !!session?.authenticated && hasAnyRole(session.user?.roles, ADMIN_ROLES)
  );
}

/**
 * Jumps from the storefront straight to an app's admin record. Administrators
 * browsing the catalogue would otherwise have to open the admin console and
 * search for the app again.
 */
export function AppAdminLink({
  appId,
  appName,
  compact = false,
}: {
  appId: string;
  appName: string;
  compact?: boolean;
}) {
  if (!appId) return null;
  return (
    <Link
      className={`button button-secondary${compact ? " button-sm button-icon" : ""}`}
      to={`/admin/apps/${encodeURIComponent(appId)}`}
      title={`${appName} 관리 설정 열기`}
      aria-label={`${appName} 관리 설정 열기`}
      onClick={(event) => event.stopPropagation()}
    >
      <SlidersHorizontal size={compact ? 16 : 18} aria-hidden="true" />
      {!compact && <span>관리 설정</span>}
    </Link>
  );
}
