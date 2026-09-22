export type UserPreferences = Record<string, any> & {
  density?: "comfortable" | "compact" | "relaxed";
  mobile_table_view?: "cards" | "table";
  nav_preset?: "personal" | "wiki" | "database" | "operations";
  navigation_advanced?: boolean;
  navigation_pins?: string[];
  document_panel?: "backlinks" | "properties" | "comments" | "ai" | "versions";
  document_panel_open?: boolean;
};
export const navigationPreferenceDefaults = {
  density: "comfortable",
  mobile_table_view: "cards",
  nav_preset: "wiki",
  navigation_advanced: false,
  document_panel: "backlinks",
  document_panel_open: true,
} as const;
export type User = {
  id: string;
  email: string;
  name: string;
  role: string;
  kind: string;
  disabled: boolean;
  preferences: UserPreferences;
  created_at: string;
};
export type Workspace = {
  id: string;
  name: string;
  slug: string;
  role: string;
};
export type Doc = {
  id: string;
  workspace_id: string;
  parent_id: string | null;
  space_id?: string | null;
  title: string;
  markdown: string;
  excerpt?: string;
  tags: string[];
  aliases: string[];
  icon: string;
  status: string;
  visibility: string;
  owner_id: string;
  version: number;
  created_at: string;
  updated_at: string;
  deleted_at: string | null;
  is_favorite: boolean;
  can_write?: boolean;
  can_comment?: boolean;
  can_approve?: boolean;
  block_metadata?: { blocks?: { id: string; type: string; text: string }[] };
};
export type DocSummary = Omit<Doc, "markdown" | "block_metadata"> & {
  excerpt: string;
  tags_truncated: boolean;
  aliases_truncated: boolean;
};
export type Settings = Record<string, any>;
export type Property = {
  id: string;
  name: string;
  type: string;
  options: string[];
};
export type Database = {
  id: string;
  workspace_id: string;
  name: string;
  properties: Property[];
  created_at: string;
};
export type Row = {
  id: string;
  version: number;
  database_id: string;
  values: Record<string, any>;
  created_at: string;
};
export class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}
export async function api<T = any>(
  path: string,
  method = "GET",
  data?: unknown,
  options?: { signal?: AbortSignal },
): Promise<T> {
  const res = await fetch("/api/v1" + path, {
    method,
    credentials: "same-origin",
    signal: options?.signal,
    headers: {
      "X-Madi-Request": "1",
      ...(data instanceof FormData
        ? {}
        : { "Content-Type": "application/json" }),
    },
    body:
      data === undefined
        ? undefined
        : data instanceof FormData
          ? data
          : JSON.stringify(data),
  });
  if (!res.ok) {
    const value = await res
      .json()
      .catch(() => ({ error: `요청을 처리하지 못했습니다 (${res.status})` }));
    throw new ApiError(value.error || "요청을 처리하지 못했습니다", res.status);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}
let dateTimezone = "Asia/Seoul";
let dateFormat = "ko";
export function setDateFormat(value?: string) {
  dateFormat = value && ["ko", "iso", "long"].includes(value) ? value : "ko";
}
function isoDate(value: string, withTime = false) {
  const parts = new Intl.DateTimeFormat("en-CA", {
    timeZone: dateTimezone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    ...(withTime
      ? {
          hour: "2-digit",
          minute: "2-digit",
          second: "2-digit",
          hourCycle: "h23" as const,
        }
      : {}),
  }).formatToParts(new Date(value));
  const get = (type: string) => parts.find((p) => p.type === type)?.value || "";
  return `${get("year")}-${get("month")}-${get("day")}${withTime ? ` ${get("hour")}:${get("minute")}:${get("second")}` : ""}`;
}
export function setDateTimezone(value?: string) {
  try {
    new Intl.DateTimeFormat("ko-KR", { timeZone: value || "Asia/Seoul" });
    dateTimezone = value || "Asia/Seoul";
  } catch {
    dateTimezone = "Asia/Seoul";
  }
}
export function date(value?: string) {
  if (value && dateFormat === "iso") return isoDate(value);
  return value
    ? new Date(value).toLocaleDateString("ko-KR", {
        month: "long",
        day: "numeric",
        timeZone: dateTimezone,
        ...(dateFormat === "long" ? { year: "numeric", weekday: "long" } : {}),
      })
    : "—";
}
export function datetime(value?: string) {
  if (value && dateFormat === "iso") return isoDate(value, true);
  return value
    ? new Date(value).toLocaleString("ko-KR", {
        timeZone: dateTimezone,
        ...(dateFormat === "long"
          ? { dateStyle: "full", timeStyle: "long" }
          : {}),
      })
    : "—";
}
export function bytes(value = 0) {
  if (value < 1024) return value + " B";
  if (value < 1048576) return (value / 1024).toFixed(1) + " KB";
  return (value / 1048576).toFixed(1) + " MB";
}
export function downloadText(
  name: string,
  text: string,
  type = "text/markdown",
) {
  const url = URL.createObjectURL(new Blob([text], { type }));
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  a.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
export async function download(path: string, name: string) {
  const res = await fetch("/api/v1" + path, {
    headers: { "X-Madi-Request": "1" },
  });
  if (!res.ok) {
    const data = await res.json().catch(() => ({}));
    throw new Error(data.error || "다운로드에 실패했습니다");
  }
  const url = URL.createObjectURL(await res.blob());
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  a.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
