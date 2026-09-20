import {
  Activity,
  AppWindow,
  Blocks,
  Bot,
  ChartNoAxesCombined,
  ClipboardCheck,
  CodeXml,
  FolderHeart,
  Gauge,
  KeyRound,
  LayoutGrid,
  ListChecks,
  Network,
  Settings,
  Shield,
  ShieldCheck,
  SlidersHorizontal,
  Sparkles,
  Tags,
  UserRound,
  Users,
  Workflow,
  type LucideIcon,
} from "lucide-react";
import { hasAnyRole } from "../../lib/utils";

export interface NavItem {
  to: string;
  label: string;
  icon: LucideIcon;
  end?: boolean;
}

export interface NavGroup {
  label: string;
  items: NavItem[];
}

export const REVIEW_ROLES = ["reviewer", "team_leader", "admin", "super_admin"];
export const ADMIN_ROLES = ["admin", "super_admin"];

export const storeGroups: NavGroup[] = [
  {
    label: "스토어",
    items: [
      { to: "/", label: "투데이", icon: Sparkles, end: true },
      { to: "/apps", label: "전체 앱", icon: LayoutGrid },
      { to: "/categories", label: "카테고리", icon: Tags },
      { to: "/apps?mcp=true", label: "MCP 앱", icon: Blocks },
      { to: "/favorites", label: "즐겨찾기", icon: FolderHeart },
    ],
  },
];

export const personalGroup: NavGroup = {
  label: "개인",
  items: [
    { to: "/my", label: "내 홈", icon: Gauge, end: true },
    { to: "/my/apps", label: "내 앱", icon: AppWindow },
    { to: "/my/keys", label: "API · MCP 키", icon: KeyRound },
    { to: "/my/profile", label: "프로필", icon: UserRound },
    { to: "/my/activity", label: "활동 내역", icon: Activity },
    { to: "/my/settings", label: "설정", icon: Settings },
  ],
};

export const reviewGroup: NavGroup = {
  label: "검토",
  items: [{ to: "/review", label: "검토 대기", icon: ListChecks }],
};

export const adminEntryGroup: NavGroup = {
  label: "관리",
  items: [{ to: "/admin", label: "관리자 콘솔", icon: ShieldCheck }],
};

export const adminGroups: NavGroup[] = [
  {
    label: "운영",
    items: [
      { to: "/admin", label: "대시보드", icon: Gauge, end: true },
      { to: "/admin/apps", label: "앱 관리", icon: AppWindow },
      { to: "/admin/categories", label: "카테고리", icon: Tags },
      { to: "/admin/users", label: "사용자", icon: Users },
      { to: "/admin/roles", label: "역할·권한", icon: ShieldCheck },
      { to: "/admin/reviews", label: "검토 관리", icon: ClipboardCheck },
      { to: "/admin/audit", label: "감사 로그", icon: Activity },
    ],
  },
  {
    label: "플랫폼",
    items: [
      { to: "/admin/workflow", label: "승인 워크플로", icon: Workflow },
      { to: "/admin/ai", label: "AI 공급자", icon: Bot },
      { to: "/admin/api", label: "REST API", icon: CodeXml },
      { to: "/admin/mcp", label: "MCP 서버", icon: Network },
      { to: "/admin/api-keys", label: "API 키", icon: KeyRound },
      { to: "/admin/authentication", label: "인증·SSO", icon: Shield },
      { to: "/admin/security", label: "보안·키 정책", icon: SlidersHorizontal },
      {
        to: "/admin/security-check",
        label: "보안 심의 연동",
        icon: ShieldCheck,
      },
      { to: "/admin/analytics", label: "방문 추적", icon: ChartNoAxesCombined },
      { to: "/admin/settings", label: "시스템 설정", icon: Settings },
    ],
  },
];

/**
 * The menu a person may use, for the sidebar and the command palette alike.
 * `admin` selects the admin console's own menu; otherwise the store menu grows
 * with whatever the roles allow.
 */
export function navGroupsFor({
  admin,
  authenticated,
  roles,
}: {
  admin: boolean;
  authenticated: boolean;
  roles?: string[];
}): NavGroup[] {
  if (admin) return adminGroups;
  return [
    ...storeGroups,
    ...(authenticated ? [personalGroup] : []),
    ...(hasAnyRole(roles, REVIEW_ROLES) ? [reviewGroup] : []),
    ...(hasAnyRole(roles, ADMIN_ROLES) ? [adminEntryGroup] : []),
  ];
}

// Everything reachable from either menu, so the palette can jump straight into
// the admin console from the store and back.
export function searchableNavGroups(options: {
  authenticated: boolean;
  roles?: string[];
}): NavGroup[] {
  const store = navGroupsFor({ ...options, admin: false }).filter(
    (group) => group !== adminEntryGroup,
  );
  return hasAnyRole(options.roles, ADMIN_ROLES)
    ? [...store, ...adminGroups]
    : store;
}
