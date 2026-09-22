import { useEffect, useState } from "react";
import { NavLink, useLocation } from "react-router-dom";
import {
  Home,
  Search,
  FileText,
  Inbox,
  Star,
  Network,
  Database,
  CheckCircle2,
  FolderOpen,
  Boxes,
  ShieldCheck,
  History,
  Sparkles,
  Import,
  Users,
  Settings,
  HardDrive,
  Trash2,
  ChevronDown,
  Pin,
  KeyRound,
  type LucideIcon,
} from "lucide-react";
import { useApp } from "../context";
import { usePreferenceWriter } from "./preferences";
import "./workspace-navigation.css";

export type NavigationPreset = "personal" | "wiki" | "database" | "operations";
export const navigationPresets: {
  id: NavigationPreset;
  label: string;
  description: string;
}[] = [
  {
    id: "personal",
    label: "개인 노트",
    description: "수집하고, 기록하고, 연결하기",
  },
  { id: "wiki", label: "팀 위키", description: "함께 정리하고 찾는 팀의 지식" },
  {
    id: "database",
    label: "지식 데이터베이스",
    description: "속성과 일정으로 체계적으로 관리",
  },
  {
    id: "operations",
    label: "지식 운영",
    description: "지식 품질과 연결 작업 관리",
  },
];
type Item = {
  path: string;
  aliases?: string[];
  label: string;
  icon: LucideIcon;
  group: "기록과 연결" | "AI와 지식 활용" | "가져오기와 연동" | "팀 관리";
  feature?: string;
  manager?: boolean;
  approval?: boolean;
};
export const workspaceNavigation: Item[] = [
  {
    path: "/app/worksets",
    label: "작업 묶음과 참고 선반",
    icon: Boxes,
    group: "기록과 연결",
  },
  {
    path: "/app/system-status",
    label: "운영 현황 카드",
    icon: Boxes,
    group: "AI와 지식 활용",
  },
  { path: "/app", label: "홈", icon: Home, group: "기록과 연결" },
  {
    path: "/app/documents",
    label: "모든 문서",
    icon: FileText,
    group: "기록과 연결",
  },
  {
    path: "/app/my-work",
    label: "내 처리함",
    icon: Inbox,
    group: "기록과 연결",
  },
  { path: "/app/search", label: "검색", icon: Search, group: "기록과 연결" },
  { path: "/app/inbox", label: "내 수집함", icon: Inbox, group: "기록과 연결" },
  {
    path: "/app/favorites",
    label: "즐겨찾기",
    icon: Star,
    group: "기록과 연결",
  },
  {
    path: "/app/graph",
    label: "지식 그래프",
    icon: Network,
    group: "기록과 연결",
  },
  {
    path: "/app/databases",
    label: "데이터베이스",
    icon: Database,
    group: "기록과 연결",
  },
  {
    path: "/app/tasks",
    label: "할 일과 일정",
    icon: CheckCircle2,
    group: "기록과 연결",
  },
  {
    path: "/app/spaces",
    label: "공간",
    icon: FolderOpen,
    group: "기록과 연결",
  },
  {
    path: "/app/canvases",
    label: "캔버스",
    icon: Boxes,
    group: "기록과 연결",
    feature: "canvas",
  },
  {
    path: "/app/templates",
    label: "템플릿",
    icon: Boxes,
    group: "기록과 연결",
  },
  { path: "/app/trash", label: "휴지통", icon: Trash2, group: "기록과 연결" },
  {
    path: "/app/ai-history",
    label: "내 AI 대화",
    icon: History,
    group: "AI와 지식 활용",
  },
  {
    path: "/app/search-history",
    label: "내 검색 기록",
    icon: History,
    group: "AI와 지식 활용",
  },
  {
    path: "/app/evidence",
    label: "근거 보관함",
    icon: ShieldCheck,
    group: "AI와 지식 활용",
  },
  {
    path: "/app/knowledge-impact",
    label: "지식 변경 영향",
    icon: Network,
    group: "AI와 지식 활용",
  },
  {
    path: "/app/knowledge-proposals",
    label: "문서 변경 제안",
    icon: FileText,
    group: "AI와 지식 활용",
  },
  {
    path: "/app/knowledge-time",
    label: "시점 기준 지식",
    icon: History,
    group: "AI와 지식 활용",
  },
  {
    path: "/app/knowledge-distribution",
    label: "망별 지식 배포",
    icon: Boxes,
    group: "AI와 지식 활용",
  },
  {
    path: "/app/knowledge-questions",
    label: "관리 질문과 공식 답변",
    icon: FileText,
    group: "AI와 지식 활용",
  },
  {
    path: "/app/knowledge-conflicts",
    label: "문서 차이 검토",
    icon: FileText,
    group: "AI와 지식 활용",
  },
  {
    path: "/app/structured-drafts",
    label: "문서에서 데이터 정리",
    icon: Boxes,
    group: "AI와 지식 활용",
  },
  {
    path: "/app/knowledge-paths",
    label: "역할별 지식 경로",
    icon: FileText,
    group: "AI와 지식 활용",
  },
  {
    path: "/app/access-requests",
    label: "문서 접근 요청",
    icon: KeyRound,
    group: "기록과 연결",
  },
  {
    path: "/app/knowledge-packages",
    label: "지식 패키지",
    icon: Boxes,
    group: "AI와 지식 활용",
  },
  {
    path: "/app/agents",
    label: "워크스페이스 Agent",
    icon: Sparkles,
    group: "AI와 지식 활용",
    feature: "workspace-agents",
  },
  {
    path: "/app/graph-ai",
    label: "AI 그래프 제안",
    icon: Sparkles,
    group: "AI와 지식 활용",
    feature: "ai-graph",
  },
  {
    path: "/app/knowledge-health",
    label: "지식 품질 관리",
    icon: ShieldCheck,
    group: "AI와 지식 활용",
  },
  {
    path: "/app/enterprise",
    label: "기업 지식 탐색",
    icon: Network,
    group: "AI와 지식 활용",
  },
  {
    path: "/app/entities",
    label: "엔터티 사전",
    icon: Boxes,
    group: "AI와 지식 활용",
  },
  {
    path: "/app/import",
    aliases: ["/app/migrations"],
    label: "가져오기",
    icon: Import,
    group: "가져오기와 연동",
  },
  {
    path: "/app/export",
    label: "데이터 내보내기",
    icon: Import,
    group: "가져오기와 연동",
  },
  {
    path: "/app/git-sync",
    label: "Git 동기화",
    icon: History,
    group: "가져오기와 연동",
  },
  {
    path: "/app/plugins",
    label: "플러그인",
    icon: Boxes,
    group: "가져오기와 연동",
    feature: "plugins",
  },
  {
    path: "/app/connectors",
    label: "외부 지식 커넥터",
    icon: Network,
    group: "가져오기와 연동",
  },
  {
    path: "/app/data-sources",
    label: "외부 데이터 소스",
    icon: Database,
    group: "가져오기와 연동",
  },
  {
    path: "/app/automations",
    label: "자동화",
    icon: Boxes,
    group: "가져오기와 연동",
  },
  {
    path: "/app/jobs",
    label: "작업 이력",
    icon: CheckCircle2,
    group: "가져오기와 연동",
  },
  {
    path: "/app/approvals",
    label: "검토·승인함",
    icon: CheckCircle2,
    group: "팀 관리",
    approval: true,
  },
  {
    path: "/app/members",
    label: "워크스페이스 멤버",
    icon: Users,
    group: "팀 관리",
  },
  {
    path: "/app/teams",
    label: "팀과 멘션 그룹",
    icon: Users,
    group: "팀 관리",
  },
  { path: "/app/organizations", label: "조직", icon: Users, group: "팀 관리" },
  {
    path: "/app/webhooks",
    label: "Webhook",
    icon: Network,
    group: "팀 관리",
    manager: true,
  },
  {
    path: "/app/workspace-settings",
    label: "워크스페이스 설정",
    icon: Settings,
    group: "팀 관리",
    manager: true,
  },
  {
    path: "/app/workspace-audit",
    label: "워크스페이스 감사",
    icon: History,
    group: "팀 관리",
    manager: true,
  },
  {
    path: "/app/workspace-operations",
    label: "팀 운영 설정",
    icon: Settings,
    group: "팀 관리",
    manager: true,
  },
  {
    path: "/app/search-ai-settings",
    label: "AI 검색 설정",
    icon: Search,
    group: "팀 관리",
    manager: true,
  },
  {
    path: "/app/storage",
    label: "저장소 설정",
    icon: HardDrive,
    group: "팀 관리",
    manager: true,
  },
];
const essentials: Record<NavigationPreset, string[]> = {
  personal: ["favorites", "tasks", "graph", "templates"],
  wiki: ["spaces", "favorites", "graph", "templates"],
  database: ["databases", "tasks", "spaces", "templates"],
  operations: ["knowledge-health", "jobs", "approvals", "automations"],
};
export function navigationPreset(value: unknown): NavigationPreset {
  return navigationPresets.some((p) => p.id === value)
    ? (value as NavigationPreset)
    : "wiki";
}
export function WorkspaceNavigation({
  flags = {},
  children,
}: {
  flags?: Record<string, boolean>;
  children?: React.ReactNode;
}) {
  const { user, workspace, publicInfo, notify } = useApp(),
    write = usePreferenceWriter(),
    location = useLocation();
  const [preset, setPreset] = useState(
      navigationPreset(user.preferences?.nav_preset),
    ),
    [advanced, setAdvanced] = useState(
      user.preferences?.navigation_advanced === true,
    );
  const [busy, setBusy] = useState(false);
  const [toolSearch, setToolSearch] = useState("");
  const pins = Array.isArray(user.preferences?.navigation_pins)
    ? (user.preferences.navigation_pins.filter(
        (p: unknown) => typeof p === "string",
      ) as string[])
    : [];
  useEffect(() => {
    setPreset(navigationPreset(user.preferences?.nav_preset));
    setAdvanced(user.preferences?.navigation_advanced === true);
  }, [
    user.id,
    user.preferences?.nav_preset,
    user.preferences?.navigation_advanced,
  ]);
  const manager =
    !!workspace &&
    ["owner", "admin"].includes(workspace.role) &&
    user.role !== "viewer";
  const allowed = workspaceNavigation.filter(
    (i) =>
      (!i.feature || flags[i.feature] !== false) &&
      (!i.manager || manager) &&
      (!i.approval || publicInfo.approval_enabled),
  );
  const isCurrent = (item: Item) =>
    location.pathname === item.path ||
    item.aliases?.includes(location.pathname) ||
    (item.path !== "/app" && location.pathname.startsWith(item.path + "/"));
  const selected = new Set([
    "/app",
    "/app/documents",
    "/app/search",
    "/app/my-work",
    ...essentials[preset].map((p) => "/app/" + p),
    ...pins,
  ]);
  // Keeping the current permitted route visible does not turn a preset into an ACL override.
  const primary = allowed.filter(
    (i) => !i.manager && (selected.has(i.path) || (!advanced && isCurrent(i))),
  );
  const secondary = allowed.filter(
    (i) =>
      !i.manager &&
      i.label
        .toLocaleLowerCase()
        .includes(toolSearch.trim().toLocaleLowerCase()),
  );
  const management = allowed.filter((i) => i.manager);
  const save = async (patch: Record<string, unknown>) => {
    setBusy(true);
    try {
      await write(patch);
    } catch (e) {
      notify((e as Error).message, "error");
      setPreset(navigationPreset(user.preferences?.nav_preset));
      setAdvanced(user.preferences?.navigation_advanced === true);
    } finally {
      setBusy(false);
    }
  };
  const link = (item: Item) => (
    <NavLink
      key={item.path}
      end={item.path === "/app"}
      to={item.path}
      className={({ isActive }) =>
        isActive || isCurrent(item) ? "active" : ""
      }
    >
      <item.icon size={19} />
      <span>{item.label}</span>
    </NavLink>
  );
  return (
    <>
      <div className="navigation-purpose">
        <label htmlFor="navigation-preset">내 작업 방식</label>
        <select
          id="navigation-preset"
          value={preset}
          disabled={busy}
          onChange={(e) => {
            const next = navigationPreset(e.target.value);
            setPreset(next);
            void save({ nav_preset: next });
          }}
        >
          {navigationPresets.map((p) => (
            <option key={p.id} value={p.id}>
              {p.label}
            </option>
          ))}
        </select>
        <p>{navigationPresets.find((p) => p.id === preset)?.description}</p>
      </div>
      <nav aria-label="주요 메뉴">{primary.map(link)}</nav>
      {children}
      <button
        className="advanced-navigation-toggle"
        aria-expanded={advanced}
        aria-controls="advanced-navigation"
        disabled={busy}
        onClick={() => {
          setAdvanced(!advanced);
          void save({ navigation_advanced: !advanced });
        }}
      >
        <Boxes size={18} />
        <span>{advanced ? "전체 도구 접기" : "전체 도구 보기"}</span>
        <ChevronDown size={18} className={advanced ? "expanded" : ""} />
      </button>
      <div id="advanced-navigation" hidden={!advanced}>
        <label className="sr-only" htmlFor="tool-search">
          전체 도구 검색
        </label>
        <input
          id="tool-search"
          className="tool-search"
          value={toolSearch}
          onChange={(e) => setToolSearch(e.target.value)}
          placeholder="도구 이름으로 찾기"
        />
        <p className="sidebar-tool-hint">
          자주 쓰는 도구는 고정해 주요 메뉴에서 여세요.
        </p>
        {!secondary.length && (
          <p className="sidebar-tool-hint" role="status">
            일치하는 도구가 없습니다.
          </p>
        )}
        {(
          [
            "기록과 연결",
            "AI와 지식 활용",
            "가져오기와 연동",
            "팀 관리",
          ] as const
        ).map((group) => (
          <section key={group}>
            <div className="sidebar-section-label">{group}</div>
            <nav aria-label={group}>
              {secondary
                .filter((i) => i.group === group)
                .map((item) => (
                  <div className="tool-navigation-row" key={item.path}>
                    {link(item)}
                    {![
                      "/app",
                      "/app/documents",
                      "/app/search",
                      "/app/my-work",
                    ].includes(item.path) && (
                      <button
                        type="button"
                        className="tool-pin"
                        disabled={busy}
                        aria-pressed={pins.includes(item.path)}
                        aria-label={`${item.label} ${pins.includes(item.path) ? "고정 해제" : "도구 고정"}`}
                        onClick={() => {
                          if (!pins.includes(item.path) && pins.length >= 12) {
                            notify(
                              "도구는 최대 12개까지 고정할 수 있습니다.",
                              "error",
                            );
                            return;
                          }
                          void save({
                            navigation_pins: pins.includes(item.path)
                              ? pins.filter((p) => p !== item.path)
                              : [...pins, item.path],
                          });
                        }}
                      >
                        <Pin size={16} />
                      </button>
                    )}
                  </div>
                ))}
            </nav>
          </section>
        ))}
      </div>
      {management.length > 0 && (
        <details
          className="workspace-management"
          open={management.some(isCurrent) || undefined}
        >
          <summary>
            <Settings size={18} />
            관리 및 설정
          </summary>
          <nav aria-label="워크스페이스 관리">{management.map(link)}</nav>
        </details>
      )}
    </>
  );
}
