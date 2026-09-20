import { useQuery } from "@tanstack/react-query";
import {
  ArrowRight,
  Blocks,
  ChevronDown,
  ChevronUp,
  ExternalLink,
  Heart,
  Rocket,
  Search,
  Sparkles,
  Star,
  Tags,
} from "lucide-react";
import {
  Fragment,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type FormEvent,
} from "react";
import { Link, Navigate, useParams, useSearchParams } from "react-router-dom";
import { api } from "../lib/api";
import { AppGuideDocuments } from "../features/apps/guide-documents";
import { SecurityVerifiedBadge } from "../features/security-check/verified-badge";
import type { StoreApp } from "../types";
import { formatDate } from "../lib/utils";
import { AppCard } from "../features/apps/app-card";
import { heroCopy } from "../features/home/hero-copy";
import { useFavorites } from "../features/apps/favorites";
import {
  AppAdminLink,
  useCanManageApps,
} from "../features/apps/admin-shortcut";
import {
  AppIcon,
  Badge,
  Button,
  ButtonLink,
  Card,
  EmptyState,
  ErrorState,
  LoadingState,
  PageHeader,
  Select,
  SkeletonGrid,
} from "../components/ui";

export function HomePage() {
  const apps = useQuery({
    queryKey: ["apps", "home"],
    queryFn: ({ signal }) =>
      api.apps({ pageSize: 24, sort: "updated" }, signal),
  });
  const config = useQuery({
    queryKey: ["public-config"],
    queryFn: ({ signal }) => api.publicConfig(signal),
    staleTime: 60_000,
  });
  // The shelf is its own request so it honours the order administrators set,
  // instead of picking whatever happens to be on the first page of updates.
  const featuredApps = useQuery({
    queryKey: ["apps", "home", "featured"],
    queryFn: ({ signal }) =>
      api.apps({ pageSize: 6, featured: true, sort: "featured" }, signal),
  });
  const featured = featuredApps.data?.items ?? [];
  const trending = useMemo(
    () =>
      apps.data?.items
        .filter((app) => app.trending || (app.trendingScore ?? 0) > 0)
        .sort(
          (left, right) =>
            (right.trendingScore ?? 0) - (left.trendingScore ?? 0),
        )
        .slice(0, 6) ?? [],
    [apps.data],
  );
  const mcp = useMemo(
    () => apps.data?.items.filter((app) => app.supportsMcp).slice(0, 6) ?? [],
    [apps.data],
  );
  const primary = featured[0] ?? apps.data?.items[0];
  const hero = heroCopy(config.data);

  return (
    <div className="page">
      <section className="hero" aria-labelledby="hero-title">
        <div className="hero-content">
          <Badge tone="primary">
            <Sparkles size={15} /> {hero.eyebrow}
          </Badge>
          <h1 id="hero-title">
            {hero.titleLines.map((line, index) => (
              <Fragment key={line + index}>
                {index > 0 && <br />}
                {line}
              </Fragment>
            ))}
          </h1>
          <p>{hero.description}</p>
          <div className="hero-actions">
            <ButtonLink
              to={
                primary ? `/apps/${encodeURIComponent(primary.slug)}` : "/apps"
              }
            >
              {hero.primaryLabel} <ArrowRight size={18} />
            </ButtonLink>
            <ButtonLink to="/apps" variant="secondary">
              {hero.secondaryLabel}
            </ButtonLink>
          </div>
        </div>
      </section>

      {apps.isPending && (
        <>
          <div className="section-header">
            <h2 className="section-title">추천 앱</h2>
          </div>
          <SkeletonGrid />
        </>
      )}
      {apps.error && (
        <div className="mt-6">
          <ErrorState error={apps.error} retry={() => void apps.refetch()} />
        </div>
      )}
      {apps.data && (
        <>
          <StoreSection
            title="에디터 추천"
            icon={<Star />}
            apps={
              featured.length
                ? featured
                : featuredApps.isPending
                  ? []
                  : apps.data.items.slice(0, 6)
            }
            href="/apps?featured=true"
          />
          <StoreSection
            title="인기 급상승"
            icon={<Rocket />}
            apps={trending.length ? trending : apps.data.items.slice(6, 12)}
            href="/apps?sort=trending"
          />
          <StoreSection
            title="MCP 지원 앱"
            icon={<Blocks />}
            apps={mcp}
            href="/apps?mcp=true"
          />
        </>
      )}
    </div>
  );
}

function StoreSection({
  title,
  icon,
  apps,
  href,
}: {
  title: string;
  icon: React.ReactNode;
  apps: import("../types").StoreApp[];
  href: string;
}) {
  if (!apps.length) return null;
  // aria-labelledby takes a list of ids, so a space in the id would break the
  // section's name instead of labelling it.
  const headingId = `section-${title.replace(/\s+/g, "-")}`;
  return (
    <section aria-labelledby={headingId}>
      <div className="section-header">
        <h2 className="section-title" id={headingId}>
          <span aria-hidden="true">{icon}</span> {title}
        </h2>
        <Link className="button button-ghost button-sm" to={href}>
          모두 보기 <ArrowRight size={16} />
        </Link>
      </div>
      <div className="card-grid">
        {apps.map((app) => (
          <AppCard app={app} key={app.id || app.slug} />
        ))}
      </div>
    </section>
  );
}

export function AppsPage({
  forcedCategory,
  favoritesOnly = false,
}: { forcedCategory?: string; favoritesOnly?: boolean } = {}) {
  const [params, setParams] = useSearchParams();
  const { slugs } = useFavorites();
  const q = params.get("q") ?? "";
  const category = forcedCategory ?? params.get("category") ?? "";
  const mcp = params.get("mcp") === "true";
  const featured = params.get("featured") === "true";
  // A featured-only view is an editorial shelf, so it opens in the order
  // administrators set rather than by recency.
  const sort = params.get("sort") ?? (featured ? "featured" : "updated");
  const page = Math.max(1, Number(params.get("page") ?? 1) || 1);
  const view = params.get("view") === "list" ? "list" : "grid";
  const [draft, setDraft] = useState(q);
  useEffect(() => setDraft(q), [q]);
  const categories = useQuery({
    queryKey: ["categories"],
    queryFn: ({ signal }) => api.categories(signal),
  });
  const apps = useQuery({
    queryKey: [
      "apps",
      { q, category, sort, mcp, featured, page, favoritesOnly, slugs },
    ],
    queryFn: ({ signal }) =>
      api.apps(
        {
          q,
          category,
          sort,
          mcp: mcp || undefined,
          featured: featured || undefined,
          page,
          pageSize: favoritesOnly ? 100 : 24,
        },
        signal,
      ),
  });
  const filtered = favoritesOnly
    ? (apps.data?.items.filter((app) => slugs.includes(app.slug)) ?? [])
    : (apps.data?.items ?? []);

  const update = (key: string, value?: string) => {
    setParams((current) => {
      const next = new URLSearchParams(current);
      if (value) next.set(key, value);
      else next.delete(key);
      if (key !== "page") next.delete("page");
      return next;
    });
  };
  const submit = (event: FormEvent) => {
    event.preventDefault();
    update("q", draft.trim());
  };

  return (
    <div className="page">
      <PageHeader
        eyebrow={
          favoritesOnly
            ? "Personal collection"
            : mcp
              ? "MCP catalog"
              : "Application catalog"
        }
        title={
          favoritesOnly
            ? "즐겨찾기"
            : mcp
              ? "MCP 앱"
              : forcedCategory
                ? `${forcedCategory} 앱`
                : "모든 앱"
        }
        description="검색, 카테고리, 정렬 상태가 URL에 저장되어 새로고침하거나 공유해도 그대로 유지됩니다."
        actions={
          <ButtonLink to="/submit">
            <Rocket size={18} /> 앱 등록
          </ButtonLink>
        }
      />
      <form className="toolbar" onSubmit={submit} role="search">
        <div className="field field-grow">
          <label htmlFor="catalog-search">앱 검색</label>
          <div className="relative">
            <Search
              className="absolute left-3 top-3 text-[var(--text-muted)]"
              size={19}
              aria-hidden="true"
            />
            <input
              id="catalog-search"
              className="input !pl-10"
              value={draft}
              onChange={(event) => setDraft(event.target.value)}
              placeholder="이름, 설명, 태그"
            />
          </div>
        </div>
        {!forcedCategory && !favoritesOnly && (
          <div className="field">
            <label htmlFor="category-filter">카테고리</label>
            <Select
              id="category-filter"
              value={category}
              onChange={(event) => update("category", event.target.value)}
            >
              <option value="">전체 카테고리</option>
              {categories.data?.map((item) => (
                <option key={item.id || item.slug} value={item.slug}>
                  {item.name}
                </option>
              ))}
            </Select>
          </div>
        )}
        <div className="field">
          <label htmlFor="sort-filter">정렬</label>
          <Select
            id="sort-filter"
            value={sort}
            onChange={(event) => update("sort", event.target.value)}
          >
            {featured && <option value="featured">추천 우선순위</option>}
            <option value="updated">최근 업데이트</option>
            <option value="name">이름</option>
            <option value="created">최근 등록</option>
            <option value="trending">인기</option>
            <option value="published">최근 게시</option>
          </Select>
        </div>
        <div className="field">
          <span className="field-label">보기</span>
          <div className="tabs" role="group" aria-label="보기 방식">
            <button
              className={`tab${view === "grid" ? " active" : ""}`}
              type="button"
              aria-pressed={view === "grid"}
              onClick={() => update("view")}
            >
              카드
            </button>
            <button
              className={`tab${view === "list" ? " active" : ""}`}
              type="button"
              aria-pressed={view === "list"}
              onClick={() => update("view", "list")}
            >
              목록
            </button>
          </div>
        </div>
        <Button type="submit">검색</Button>
      </form>
      <div className="flex items-center justify-between gap-3 mb-4 text-[14px] text-[var(--text-muted)]">
        <span aria-live="polite">
          {apps.data ? `${apps.data.total}개 앱` : "앱 수 확인 중"}
        </span>
        <div className="flex gap-2">
          {featured && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => update("featured")}
            >
              추천 필터 해제
            </Button>
          )}
          {mcp && (
            <Button variant="ghost" size="sm" onClick={() => update("mcp")}>
              MCP 필터 해제
            </Button>
          )}
        </div>
      </div>
      {apps.isPending && <SkeletonGrid />}
      {apps.error && (
        <ErrorState error={apps.error} retry={() => void apps.refetch()} />
      )}
      {apps.data && !filtered.length && (
        <EmptyState
          title={
            favoritesOnly
              ? "즐겨찾기한 앱이 없습니다"
              : "조건에 맞는 앱이 없습니다"
          }
          description={
            favoritesOnly
              ? "앱 상세 또는 카드에서 하트를 눌러 보관하세요."
              : "검색어나 필터를 바꾸어 보세요."
          }
          actions={
            !favoritesOnly && (
              <Button onClick={() => setParams({})}>필터 초기화</Button>
            )
          }
        />
      )}
      {!!filtered.length && (
        <div className={view === "grid" ? "card-grid" : "grid gap-3"}>
          {filtered.map((app) => (
            <AppCard key={app.id || app.slug} app={app} />
          ))}
        </div>
      )}
      {apps.data && apps.data.total > apps.data.pageSize && (
        <nav className="state-actions mt-7" aria-label="페이지">
          <Button
            variant="secondary"
            disabled={page <= 1}
            onClick={() => update("page", String(page - 1))}
          >
            이전
          </Button>
          <Badge>{page} 페이지</Badge>
          <Button
            variant="secondary"
            disabled={page * apps.data.pageSize >= apps.data.total}
            onClick={() => update("page", String(page + 1))}
          >
            다음
          </Button>
        </nav>
      )}
    </div>
  );
}

export function AppDetailPage() {
  const { slug = "" } = useParams();
  const app = useQuery({
    queryKey: ["app", slug],
    queryFn: ({ signal }) => api.app(slug, signal),
    enabled: !!slug,
  });
  const { isFavorite, toggle } = useFavorites();
  const canManage = useCanManageApps();
  if (app.isPending)
    return (
      <div className="page">
        <LoadingState label="앱 정보를 불러오는 중입니다" />
      </div>
    );
  if (app.error)
    return (
      <div className="page">
        <ErrorState error={app.error} retry={() => void app.refetch()} />
      </div>
    );
  if (!app.data) return null;
  const item = app.data;
  const favorite = isFavorite(item.slug);
  return (
    <div className="page">
      <Card>
        <header className="detail-head">
          <AppIcon app={item} large />
          <div className="detail-head-content">
            <div className="badge-row !mt-0">
              {item.featured && (
                <Badge tone="primary">
                  <Star size={14} /> 추천
                </Badge>
              )}
              {item.status && (
                <Badge
                  tone={item.status === "published" ? "positive" : "warning"}
                >
                  {item.status}
                </Badge>
              )}
              <SecurityVerifiedBadge app={item} showDate />
            </div>
            <h1 className="detail-title">{item.name}</h1>
            <p className="detail-summary">{item.summary}</p>
            <div className="hero-actions">
              {item.serviceUrl ? (
                <a
                  className="button button-primary"
                  href={item.serviceUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <ExternalLink size={18} /> 서비스 열기
                </a>
              ) : (
                <Button disabled>서비스 URL 준비 중</Button>
              )}
              <Button
                variant="secondary"
                aria-pressed={favorite}
                onClick={() => toggle(item.slug)}
              >
                <Heart size={18} fill={favorite ? "currentColor" : "none"} />{" "}
                {favorite ? "보관됨" : "즐겨찾기"}
              </Button>
              {canManage && (
                <AppAdminLink appId={item.id} appName={item.name} />
              )}
            </div>
          </div>
        </header>
      </Card>
      <div className="detail-body">
        {/* The guide files sit in the same column as the introduction rather
            than in a third grid cell: a long 정보 card used to decide where
            row two started, which pushed the downloads off the first screen. */}
        <div className="detail-main">
          <AppIntroCard app={item} />
          <AppGuideDocuments appSlug={item.slug} />
        </div>
        <Card className="prose-card">
          <h2>정보</h2>
          <dl className="meta-list">
            <Meta
              label="버전"
              value={item.version ? `v${item.version.replace(/^v/, "")}` : "—"}
            />
            <Meta
              label="카테고리"
              value={item.category?.name || item.categoryName || "—"}
            />
            <Meta label="언어" value={item.language || "—"} />
            <Meta label="Framework" value={item.framework || "—"} />
            <Meta label="MCP" value={item.supportsMcp ? "지원" : "미지원"} />
            <Meta label="API" value={item.supportsApi ? "지원" : "미지원"} />
            <Meta label="담당팀" value={item.team || "—"} />
            <Meta label="업데이트" value={formatDate(item.updatedAt)} />
          </dl>
        </Card>
      </div>
    </div>
  );
}

/**
 * 앱 소개 whose long description is folded away.
 *
 * The description is free text an owner types, and a thorough one ran long
 * enough to push everything below it — the guide downloads included — past the
 * fold. It is clamped to a few lines, and the toggle only appears when there is
 * something left to show.
 */
function AppIntroCard({ app }: { app: StoreApp }) {
  const description = app.description || app.summary;
  const [expanded, setExpanded] = useState(false);
  const [overflows, setOverflows] = useState(false);
  const intro = useRef<HTMLParagraphElement>(null);
  useLayoutEffect(() => {
    const element = intro.current;
    if (!element) return;
    // Measuring while expanded reports no overflow, so the answer from the
    // clamped render is the one that decides whether the toggle belongs.
    // A narrower window fits fewer lines, so the same text can start needing
    // the toggle without the app ever re-rendering.
    const measure = () => {
      if (!expanded) {
        setOverflows(element.scrollHeight - element.clientHeight > 4);
      }
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(element);
    return () => observer.disconnect();
  }, [description, expanded]);
  return (
    <Card className="prose-card">
      <h2>앱 소개</h2>
      <p
        ref={intro}
        className={expanded ? "app-intro" : "app-intro is-clamped"}
      >
        {description}
      </p>
      {overflows && (
        <Button
          variant="ghost"
          size="sm"
          className="app-intro-toggle"
          aria-expanded={expanded}
          onClick={() => setExpanded((current) => !current)}
        >
          {expanded ? (
            <>
              접기 <ChevronUp size={16} aria-hidden="true" />
            </>
          ) : (
            <>
              더 보기 <ChevronDown size={16} aria-hidden="true" />
            </>
          )}
        </Button>
      )}
      {app.tags?.length ? (
        <>
          <h2 className="!mt-8">태그</h2>
          <div className="badge-row">
            {app.tags.map((tag) => (
              <Badge key={tag}>{tag}</Badge>
            ))}
          </div>
        </>
      ) : null}
    </Card>
  );
}

function Meta({ label, value }: { label: string; value: string }) {
  return (
    <div className="meta-row">
      <dt>{label}</dt>
      <dd>{value}</dd>
    </div>
  );
}

export function CategoriesPage() {
  const categories = useQuery({
    queryKey: ["categories"],
    queryFn: ({ signal }) => api.categories(signal),
  });
  return (
    <div className="page">
      <PageHeader
        eyebrow="Browse by category"
        title="카테고리"
        description="업무 목적에 맞는 앱 모음을 빠르게 찾아보세요."
      />
      {categories.isPending && <SkeletonGrid count={4} />}
      {categories.error && (
        <ErrorState
          error={categories.error}
          retry={() => void categories.refetch()}
        />
      )}
      {categories.data && !categories.data.length && <EmptyState />}
      {categories.data && (
        <div className="card-grid">
          {categories.data.map((category) => (
            <Link
              className="card app-card !min-h-48"
              key={category.id || category.slug}
              to={`/categories/${encodeURIComponent(category.slug)}`}
            >
              <div className="state-icon !m-0">
                <Tags />
              </div>
              <h2 className="app-card-name !mt-5">{category.name}</h2>
              <p className="app-card-summary">
                {category.description || `${category.name} 앱을 둘러보세요.`}
              </p>
              <div className="app-card-footer">
                <span>{category.appCount ?? 0}개 앱</span>
                <ArrowRight size={18} />
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}

export function CategoryPage() {
  const { category = "" } = useParams();
  return <AppsPage forcedCategory={category} />;
}

export function FavoritesPage() {
  return <AppsPage favoritesOnly />;
}

export function SearchCompatibilityPage() {
  const [params] = useSearchParams();
  return <Navigate to={`/apps?${params.toString()}`} replace />;
}
