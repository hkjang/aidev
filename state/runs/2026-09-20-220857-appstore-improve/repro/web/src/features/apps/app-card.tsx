import { ExternalLink, Heart, ServerCog } from "lucide-react";
import { Link } from "react-router-dom";
import { appGlyph, formatDate } from "../../lib/utils";
import type { StoreApp } from "../../types";
import { AppIcon, Badge, Button } from "../../components/ui";
import { useFavorites } from "./favorites";
import { AppAdminLink, useCanManageApps } from "./admin-shortcut";
import { AppStatusBadge } from "./app-status";
import { SecurityVerifiedBadge } from "../security-check/verified-badge";

// The public detail route serves only published, public apps (see
// GetAppBySlug with includeAll=false), so this is what decides whether a card
// may link to /apps/{slug} at all.
function publiclyViewable(app: StoreApp) {
  return app.status === "published" && app.visibility !== "private";
}

// The public catalog only ever lists published apps, so the status is noise
// there; an owner's own list mixes drafts, pending, rejected and archived apps
// and needs it to tell them apart. Those apps have no public page either, so
// in that list the card opens the owner's edit screen instead of a 404.
export function AppCard({
  app,
  showStatus = false,
}: {
  app: StoreApp;
  showStatus?: boolean;
}) {
  const { isFavorite, toggle } = useFavorites();
  const favorite = isFavorite(app.slug);
  const canManage = useCanManageApps();
  const viewable = !showStatus || publiclyViewable(app);
  const target = viewable
    ? `/apps/${encodeURIComponent(app.slug)}`
    : `/my/apps/${encodeURIComponent(app.id)}/edit`;
  return (
    <article className="card app-card">
      <div className="app-card-body">
        <div className="app-card-head">
          <AppIcon app={app} />
          <div className="min-w-0">
            <h2 className="app-card-name">
              {/* The link stretches over the whole card (see .app-card-name a
                  in styles.css), so the summary and the badges open the app
                  too — only the footer controls stay on top of it. */}
              <Link
                to={target}
                title={viewable ? undefined : `${app.name} 수정 화면 열기`}
              >
                {app.name}
              </Link>
            </h2>
            <p className="app-card-summary">{app.summary}</p>
          </div>
        </div>
        <div className="badge-row" aria-label="앱 특성">
          {showStatus && <AppStatusBadge status={app.status} />}
          <SecurityVerifiedBadge app={app} />
          {(app.category?.name || app.categoryName) && (
            <Badge>{app.category?.name || app.categoryName}</Badge>
          )}
          {app.language && <Badge>{app.language}</Badge>}
          {app.supportsMcp && (
            <Badge tone="primary">
              <ServerCog size={14} /> MCP
            </Badge>
          )}
          {app.supportsApi && <Badge tone="positive">API</Badge>}
        </div>
      </div>
      <footer className="app-card-footer">
        <span className="text-[13px] text-[var(--text-muted)]">
          {app.updatedAt
            ? `${formatDate(app.updatedAt)} 업데이트`
            : app.version
              ? `v${app.version.replace(/^v/, "")}`
              : "앱 정보 보기"}
        </span>
        <div className="top-actions">
          {viewable && (
            <Button
              variant="ghost"
              size="sm"
              aria-label={`${app.name} ${favorite ? "즐겨찾기 해제" : "즐겨찾기 추가"}`}
              aria-pressed={favorite}
              onClick={() => toggle(app.slug)}
            >
              <Heart size={17} fill={favorite ? "currentColor" : "none"} />
            </Button>
          )}
          {canManage && (
            <AppAdminLink appId={app.id} appName={app.name} compact />
          )}
          {/* An administrator's card already carries the admin shortcut next
              to the favourite button; spelling out 자세히 as well crowds the
              footer, so their row is icons only. An unpublished app has no
              detail page to show, so its card has no 자세히 at all. */}
          {viewable && (
            <Link
              className={`button button-secondary button-sm${canManage ? " button-icon" : ""}`}
              to={target}
              title={`${app.name} 상세 보기`}
              aria-label={`${app.name} 상세 보기`}
            >
              {!canManage && <span>자세히</span>}
              <ExternalLink size={15} aria-hidden="true" />
            </Link>
          )}
        </div>
      </footer>
    </article>
  );
}

export function AppAvatar({ name, icon }: { name: string; icon?: string }) {
  return (
    <span className="app-icon" aria-hidden="true">
      {appGlyph(name, icon)}
    </span>
  );
}
