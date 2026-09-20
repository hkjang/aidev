import { useQuery } from "@tanstack/react-query";
import {
  AppWindow,
  Boxes,
  ChevronDown,
  Command,
  ExternalLink,
  KeyRound,
  LogIn,
  LogOut,
  Menu,
  Moon,
  Plus,
  Search,
  Settings,
  Sun,
  X,
} from "lucide-react";
import {
  useEffect,
  useRef,
  useState,
  type FormEvent,
  type PropsWithChildren,
} from "react";
import { Link, NavLink, useLocation, useNavigate } from "react-router-dom";
import { api } from "../lib/api";
import { initials } from "../lib/utils";
import { navGroupsFor } from "../features/navigation/nav-items";
import { CommandPalette } from "../features/navigation/command-palette";
import { useAuth, useTheme } from "../app/providers";
import { Button, ButtonLink, ErrorState } from "./ui";

// Query parameters that give a path its own nav entry.
const NAV_VARIANT_PARAMS = ["mcp"];

/**
 * Marks a nav entry active from the whole URL, query string included.
 *
 * NavLink's own `isActive` only looks at the pathname, so `/apps` and
 * `/apps?mcp=true` would light up together. Every query parameter an entry
 * declares must match, and an entry that declares none must not match a URL
 * that carries one of its siblings' parameters.
 */
export function navItemActive(
  to: string,
  pathname: string,
  search: string,
  end = false,
): boolean {
  const [targetPath = "", targetSearch = ""] = to.split("?");
  const current = new URLSearchParams(search);
  const target = new URLSearchParams(targetSearch);
  const pathMatches =
    end || targetSearch
      ? pathname === targetPath
      : pathname === targetPath || pathname.startsWith(`${targetPath}/`);
  if (!pathMatches) return false;
  if (targetSearch)
    return [...target].every(([key, value]) => current.get(key) === value);
  return !NAV_VARIANT_PARAMS.some((key) => current.has(key));
}

function ProfileMenu() {
  const { session, logout } = useAuth();
  const version = useQuery({
    queryKey: ["version"],
    queryFn: ({ signal }) => api.version(signal),
    staleTime: Infinity,
  });
  const [open, setOpen] = useState(false);
  const wrapRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();

  useEffect(() => {
    if (!open) return;
    const close = (event: PointerEvent) => {
      if (!wrapRef.current?.contains(event.target as Node)) setOpen(false);
    };
    const key = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpen(false);
    };
    document.addEventListener("pointerdown", close);
    document.addEventListener("keydown", key);
    return () => {
      document.removeEventListener("pointerdown", close);
      document.removeEventListener("keydown", key);
    };
  }, [open]);

  if (!session?.authenticated || !session.user) {
    return (
      <ButtonLink to="/login" variant="secondary">
        <LogIn size={18} /> 로그인
      </ButtonLink>
    );
  }

  const user = session.user;
  const label = user.displayName || user.username;
  return (
    <div className="profile-wrap" ref={wrapRef}>
      <Button
        variant="secondary"
        className="profile-trigger"
        aria-expanded={open}
        aria-haspopup="menu"
        onClick={() => setOpen((value) => !value)}
      >
        <span className="avatar" aria-hidden="true">
          {initials(label)}
        </span>
        <span>{label}</span>
        <ChevronDown size={16} />
      </Button>
      {open && (
        <div className="profile-menu" role="menu" aria-label="프로필 메뉴">
          <div className="profile-summary">
            <strong>{label}</strong>
            <span>{user.team || user.email || "AppStore 사용자"}</span>
          </div>
          <Link
            role="menuitem"
            className="menu-item"
            to="/my/apps"
            onClick={() => setOpen(false)}
          >
            <AppWindow size={18} /> 내 앱
          </Link>
          <Link
            role="menuitem"
            className="menu-item"
            to="/my/keys"
            onClick={() => setOpen(false)}
          >
            <KeyRound size={18} /> API · MCP 키
          </Link>
          <Link
            role="menuitem"
            className="menu-item"
            to="/my/settings"
            onClick={() => setOpen(false)}
          >
            <Settings size={18} /> 개인 설정
          </Link>
          <button
            role="menuitem"
            className="menu-item"
            onClick={async () => {
              await logout();
              setOpen(false);
              navigate("/");
            }}
          >
            <LogOut size={18} /> 로그아웃
          </button>
          <div className="menu-item menu-version" role="none">
            <Boxes size={18} /> AppStore{" "}
            {version.data?.version ?? "버전 확인 중"}
          </div>
        </div>
      )}
    </div>
  );
}

function Sidebar({
  admin,
  open,
  close,
}: {
  admin: boolean;
  open: boolean;
  close: () => void;
}) {
  const { session } = useAuth();
  const location = useLocation();
  const config = useQuery({
    queryKey: ["public-config"],
    queryFn: ({ signal }) => api.publicConfig(signal),
    staleTime: 60_000,
  });
  const roles = session?.user?.roles;
  const groups = navGroupsFor({
    admin,
    authenticated: !!session?.authenticated,
    roles,
  });

  return (
    <aside
      className={`sidebar${open ? " open" : ""}`}
      aria-label={admin ? "관리자 메뉴" : "주 메뉴"}
    >
      <Link to={admin ? "/admin" : "/"} className="brand" onClick={close}>
        {config.data?.logoUrl ? (
          <img
            className="brand-logo"
            src={config.data.logoUrl}
            alt=""
            aria-hidden="true"
          />
        ) : (
          <span className="brand-mark" aria-hidden="true">
            A
          </span>
        )}
        <span>
          <span className="brand-name">
            {config.data?.siteName || "AppStore"}
          </span>
          <span className="brand-tagline">
            {admin ? "관리자 콘솔" : "사내 앱 스토어"}
          </span>
        </span>
      </Link>
      <nav className="sidebar-scroll">
        {groups.map((group) => (
          <section
            className="nav-group"
            key={group.label}
            aria-labelledby={`nav-${group.label}`}
          >
            <h2 className="nav-heading" id={`nav-${group.label}`}>
              {group.label}
            </h2>
            {group.items.map((item) => {
              const Icon = item.icon;
              const isActive = navItemActive(
                item.to,
                location.pathname,
                location.search,
                item.end,
              );
              return (
                <NavLink
                  key={item.to}
                  to={item.to}
                  end={item.end}
                  onClick={close}
                  aria-current={isActive ? "page" : undefined}
                  // The callback form keeps NavLink from appending its own
                  // pathname-only "active" class on top of ours.
                  className={() => `nav-link${isActive ? " active" : ""}`}
                >
                  <Icon aria-hidden="true" />
                  <span>{item.label}</span>
                </NavLink>
              );
            })}
          </section>
        ))}
      </nav>
      {!admin && (
        <div className="sidebar-bottom">
          <ButtonLink to="/submit" className="w-full">
            <Plus size={18} /> 앱 등록
          </ButtonLink>
          {config.data?.siteUrl && (
            <a
              className="service-url"
              href={config.data.siteUrl}
              target="_blank"
              rel="noopener noreferrer"
              title={config.data.siteUrl}
            >
              <ExternalLink size={15} /> 공식 서비스 URL
            </a>
          )}
        </div>
      )}
      {admin && (
        <div className="sidebar-bottom">
          <ButtonLink to="/" variant="secondary" className="w-full">
            스토어로 이동
          </ButtonLink>
        </div>
      )}
    </aside>
  );
}

export function AppShell({
  children,
  admin = false,
}: PropsWithChildren<{ admin?: boolean }>) {
  const [mobileOpen, setMobileOpen] = useState(false);
  const { toggle, resolved } = useTheme();
  const location = useLocation();
  const navigate = useNavigate();
  const [query, setQuery] = useState(
    new URLSearchParams(location.search).get("q") ?? "",
  );
  const [paletteOpen, setPaletteOpen] = useState(false);
  const searchRef = useRef<HTMLInputElement>(null);

  useEffect(() => setMobileOpen(false), [location.pathname]);
  useEffect(() => {
    setQuery(new URLSearchParams(location.search).get("q") ?? "");
  }, [location.search]);
  // Ctrl/Cmd+K works from any field, unlike the "/" catalogue shortcut.
  useEffect(() => {
    const openPalette = (event: KeyboardEvent) => {
      if (event.key !== "k" || !(event.metaKey || event.ctrlKey)) return;
      event.preventDefault();
      setPaletteOpen((open) => !open);
    };
    document.addEventListener("keydown", openPalette);
    return () => document.removeEventListener("keydown", openPalette);
  }, []);
  useEffect(() => {
    if (admin) return;
    const focusSearch = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      if (
        event.key !== "/" ||
        target?.matches("input, textarea, select, [contenteditable='true']")
      )
        return;
      event.preventDefault();
      searchRef.current?.focus();
    };
    document.addEventListener("keydown", focusSearch);
    return () => document.removeEventListener("keydown", focusSearch);
  }, [admin]);

  const search = (event: FormEvent) => {
    event.preventDefault();
    const params = new URLSearchParams();
    if (query.trim()) params.set("q", query.trim());
    navigate(`/apps${params.size ? `?${params}` : ""}`);
  };

  return (
    <div className="app-layout">
      <a href="#main-content" className="skip-link">
        본문으로 건너뛰기
      </a>
      {mobileOpen && (
        <button
          className="sidebar-overlay"
          aria-label="메뉴 닫기"
          onClick={() => setMobileOpen(false)}
        />
      )}
      <Sidebar
        admin={admin}
        open={mobileOpen}
        close={() => setMobileOpen(false)}
      />
      <div className="main-column">
        <header className="topbar">
          <Button
            variant="ghost"
            className="button-icon mobile-menu-button"
            aria-label={mobileOpen ? "메뉴 닫기" : "메뉴 열기"}
            aria-expanded={mobileOpen}
            onClick={() => setMobileOpen((value) => !value)}
          >
            {mobileOpen ? <X /> : <Menu />}
          </Button>
          {!admin && (
            <form className="topbar-search" role="search" onSubmit={search}>
              <Search aria-hidden="true" />
              <label className="sr-only" htmlFor="global-search">
                앱 검색
              </label>
              <input
                ref={searchRef}
                id="global-search"
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder="앱 이름, 설명, 태그 검색"
              />
              <kbd aria-hidden="true">/</kbd>
            </form>
          )}
          {admin && <strong>서비스 관리자</strong>}
          <div className="top-actions">
            <Button
              variant="secondary"
              className="palette-trigger"
              // The label and hint collapse on narrow screens, so the button
              // keeps its own name.
              aria-label="빠른 이동"
              aria-haspopup="dialog"
              aria-expanded={paletteOpen}
              onClick={() => setPaletteOpen(true)}
            >
              <Command size={17} aria-hidden="true" />
              <span>빠른 이동</span>
              <kbd aria-hidden="true">Ctrl K</kbd>
            </Button>
            <Button
              variant="secondary"
              className="button-icon"
              aria-label={`${resolved === "dark" ? "라이트" : "다크"} 모드로 전환`}
              onClick={toggle}
            >
              {resolved === "dark" ? <Sun size={19} /> : <Moon size={19} />}
            </Button>
            <ProfileMenu />
          </div>
        </header>
        <main id="main-content" tabIndex={-1}>
          {children}
        </main>
      </div>
      <CommandPalette
        open={paletteOpen}
        onClose={() => setPaletteOpen(false)}
      />
    </div>
  );
}

export function ShellConfigError({
  error,
  retry,
}: {
  error: unknown;
  retry: () => void;
}) {
  return (
    <div className="page">
      <ErrorState error={error} retry={retry} />
    </div>
  );
}
