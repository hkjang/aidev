import { useQuery } from "@tanstack/react-query";
import {
  CornerDownLeft,
  History,
  Search,
  SlidersHorizontal,
} from "lucide-react";
import {
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../../app/providers";
import { api } from "../../lib/api";
import { appGlyph, appTone, hasAnyRole, safeJsonParse } from "../../lib/utils";
import { ADMIN_ROLES, searchableNavGroups } from "./nav-items";

const RECENT_KEY = "appstore.recentDestinations";
const RECENT_LIMIT = 5;

export interface Destination {
  id: string;
  to: string;
  label: string;
  hint?: string;
  group: string;
  glyph?: string;
  tone?: number;
  icon?: ReactNode;
}

// Only the plain fields are persisted: an icon is a React element and would
// come back from JSON as an object React refuses to render.
type StoredDestination = Omit<Destination, "icon" | "group">;

function readRecent(): Destination[] {
  const stored = safeJsonParse<StoredDestination[]>(
    localStorage.getItem(RECENT_KEY) ?? "",
    [],
  );
  if (!Array.isArray(stored)) return [];
  return stored
    .filter(
      (item) =>
        item && typeof item.id === "string" && typeof item.to === "string",
    )
    .slice(0, RECENT_LIMIT)
    .map((item) => ({
      ...item,
      group: "최근 이동",
      icon: <History size={17} aria-hidden="true" />,
    }));
}

function rememberRecent(destination: Destination): void {
  const entry: StoredDestination = {
    id: destination.id,
    to: destination.to,
    label: destination.label,
    hint: destination.hint,
    glyph: destination.glyph,
    tone: destination.tone,
  };
  const next = [
    entry,
    ...readRecent()
      .filter((item) => item.id !== destination.id)
      .map(({ id, to, label, hint, glyph, tone }) => ({
        id,
        to,
        label,
        hint,
        glyph,
        tone,
      })),
  ].slice(0, RECENT_LIMIT);
  try {
    localStorage.setItem(RECENT_KEY, JSON.stringify(next));
  } catch {
    // A blocked or full store only costs the convenience list.
  }
}

function matches(destination: Destination, query: string): boolean {
  if (!query) return true;
  const haystack =
    `${destination.label} ${destination.hint ?? ""} ${destination.to}`.toLowerCase();
  return query
    .toLowerCase()
    .split(/\s+/)
    .filter(Boolean)
    .every((term) => haystack.includes(term));
}

export function CommandPalette({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const navigate = useNavigate();
  const { session } = useAuth();
  const [query, setQuery] = useState("");
  const [active, setActive] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listId = useId();
  const canManage = hasAnyRole(session?.user?.roles, ADMIN_ROLES);

  // Only search the catalogue once there is something to search for.
  const apps = useQuery({
    queryKey: ["command-apps", query],
    queryFn: ({ signal }) => api.apps({ q: query, pageSize: 6 }, signal),
    enabled: open && query.trim().length > 0,
    staleTime: 30_000,
  });

  const menuDestinations = useMemo<Destination[]>(
    () =>
      searchableNavGroups({
        authenticated: !!session?.authenticated,
        roles: session?.user?.roles,
      }).flatMap((group) =>
        group.items.map((item) => ({
          id: `menu:${item.to}`,
          to: item.to,
          label: item.label,
          group: group.label,
          icon: <item.icon size={17} aria-hidden="true" />,
        })),
      ),
    [session?.authenticated, session?.user?.roles],
  );

  const appDestinations = useMemo<Destination[]>(() => {
    const items = apps.data?.items ?? [];
    return items.flatMap((app) => {
      const base: Destination = {
        id: `app:${app.slug}`,
        to: `/apps/${encodeURIComponent(app.slug)}`,
        label: app.name,
        hint: app.summary,
        group: "앱",
        glyph: appGlyph(app.name, app.icon),
        tone: appTone(app.slug || app.name),
      };
      if (!canManage) return [base];
      return [
        base,
        {
          ...base,
          id: `app-admin:${app.id}`,
          to: `/admin/apps/${encodeURIComponent(app.id)}`,
          label: `${app.name} · 관리 설정`,
          hint: undefined,
          icon: <SlidersHorizontal size={17} aria-hidden="true" />,
          glyph: undefined,
          tone: undefined,
        },
      ];
    });
  }, [apps.data, canManage]);

  const results = useMemo<Destination[]>(() => {
    const trimmed = query.trim();
    if (!trimmed) {
      const recent = readRecent();
      const recentIds = new Set(recent.map((item) => item.id));
      return [
        ...recent,
        ...menuDestinations.filter((item) => !recentIds.has(item.id)),
      ];
    }
    return [
      ...appDestinations,
      ...menuDestinations.filter((item) => matches(item, trimmed)),
    ];
  }, [appDestinations, menuDestinations, query]);

  useEffect(() => {
    if (!open) return;
    setQuery("");
    setActive(0);
    const focus = requestAnimationFrame(() => inputRef.current?.focus());
    const overflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      cancelAnimationFrame(focus);
      document.body.style.overflow = overflow;
    };
  }, [open]);

  useEffect(() => setActive(0), [query]);

  if (!open) return null;

  const go = (destination?: Destination) => {
    if (!destination) return;
    rememberRecent(destination);
    onClose();
    navigate(destination.to);
  };

  const onKeyDown = (event: React.KeyboardEvent) => {
    if (event.key === "Escape") {
      event.preventDefault();
      onClose();
      return;
    }
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      if (!results.length) return;
      const step = event.key === "ArrowDown" ? 1 : -1;
      setActive((index) => (index + step + results.length) % results.length);
      return;
    }
    if (event.key === "Enter") {
      event.preventDefault();
      go(results[active]);
    }
  };

  let renderedGroup = "";
  return (
    <div
      className="palette-backdrop"
      onMouseDown={(event) => {
        if (event.currentTarget === event.target) onClose();
      }}
    >
      <div
        className="palette"
        role="dialog"
        aria-modal="true"
        aria-label="빠른 이동"
        onKeyDown={onKeyDown}
      >
        <div className="palette-search">
          <Search size={19} aria-hidden="true" />
          <input
            ref={inputRef}
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="메뉴, 앱 이름으로 이동"
            aria-label="빠른 이동 검색"
            aria-controls={listId}
            aria-activedescendant={
              results[active] ? `${listId}-${active}` : undefined
            }
            autoComplete="off"
            role="combobox"
            aria-expanded
          />
          <kbd>esc</kbd>
        </div>
        <ul className="palette-list" id={listId} role="listbox">
          {results.map((destination, index) => {
            const heading =
              destination.group === renderedGroup ? null : destination.group;
            renderedGroup = destination.group;
            return (
              <li key={destination.id} role="none">
                {heading && <p className="palette-group">{heading}</p>}
                <button
                  type="button"
                  id={`${listId}-${index}`}
                  role="option"
                  aria-selected={index === active}
                  className={`palette-item${index === active ? " active" : ""}`}
                  onMouseMove={() => setActive(index)}
                  onClick={() => go(destination)}
                >
                  <span className="palette-icon" data-tone={destination.tone}>
                    {destination.icon ?? destination.glyph}
                  </span>
                  <span className="palette-text">
                    <strong>{destination.label}</strong>
                    {destination.hint && <span>{destination.hint}</span>}
                  </span>
                  {index === active && (
                    <CornerDownLeft size={15} aria-hidden="true" />
                  )}
                </button>
              </li>
            );
          })}
          {!results.length && (
            <li className="palette-empty" role="none">
              {apps.isFetching ? "찾는 중입니다" : "이동할 항목이 없습니다"}
            </li>
          )}
        </ul>
        <div className="palette-footer">
          <span>
            <kbd>↑</kbd>
            <kbd>↓</kbd> 선택
          </span>
          <span>
            <kbd>↵</kbd> 이동
          </span>
          <span>
            <kbd>esc</kbd> 닫기
          </span>
        </div>
      </div>
    </div>
  );
}
