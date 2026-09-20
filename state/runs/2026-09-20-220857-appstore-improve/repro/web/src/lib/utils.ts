import clsx, { type ClassValue } from "clsx";

export function cn(...values: ClassValue[]): string {
  return clsx(values);
}

export function safeJsonParse<T>(value: string | null, fallback: T): T {
  if (!value) return fallback;
  try {
    return JSON.parse(value) as T;
  } catch {
    return fallback;
  }
}

export function formatDate(value?: string): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("ko-KR", { dateStyle: "medium" }).format(date);
}

export function formatDateTime(value?: string): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("ko-KR", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

export function initials(value?: string): string {
  const normalized = value?.trim();
  if (!normalized) return "AP";
  return [...normalized].slice(0, 2).join("").toUpperCase();
}

export function normalizeRoles(roles?: string[]): string[] {
  return (roles ?? []).map((role) =>
    role
      .toLowerCase()
      .replace(/^appstore[-_]/, "")
      .replace(/-/g, "_"),
  );
}

export function hasAnyRole(
  roles: string[] | undefined,
  expected: string[],
): boolean {
  const normalized = normalizeRoles(roles);
  return expected.some((role) => normalized.includes(role.toLowerCase()));
}

// Shared rules for the comma-separated fields (tags, screenshots, roles).
export function parseList(value: string): string[] {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

export function sameList(a: readonly string[], b: readonly string[]): boolean {
  return a.length === b.length && a.every((item, index) => item === b[index]);
}

export function appGlyph(name: string, icon?: string): string {
  if (icon && !icon.startsWith("http")) return icon;
  return initials(name);
}

// Number of tinted chips an app icon can land on. The portal's own app icons
// are a set of coloured illustrations, so a catalogue of flat chips reads
// closer to it than one uniform dark square.
export const APP_TONE_COUNT = 6;

// Stable per-app tint: the same app keeps its colour across pages and reloads.
export function appTone(key: string): number {
  let hash = 0;
  for (let index = 0; index < key.length; index += 1) {
    hash = (hash * 31 + key.charCodeAt(index)) % 100000;
  }
  return hash % APP_TONE_COUNT;
}

// Mirrors the backend SafeReturnTo contract. Browsers normalize a backslash to
// a slash and strip tab, CR and LF before resolving a URL, so "/\evil.test" and
// "/<TAB>/evil.test" would otherwise become the protocol-relative "//evil.test".
export function safeReturnTo(value: string | null | undefined): string {
  const trimmed = value?.trim() ?? "";
  if (!trimmed.startsWith("/") || trimmed.startsWith("//")) return "/";
  // eslint-disable-next-line no-control-regex
  if (/[\\\u0000-\u001f\u007f]/.test(trimmed)) return "/";
  return trimmed;
}

// Mirrors the branding upload limit the server enforces. Checking the picked
// file first keeps a mistaken choice off the wire instead of uploading it only
// to be told it is too large.
export const MAX_BRANDING_BYTES = 1024 * 1024;

export function brandingSizeError(size: number): string {
  return size > MAX_BRANDING_BYTES ? "이미지는 1MB 이하여야 합니다." : "";
}

export function clampToken(value: number): number {
  if (!Number.isFinite(value)) return 0;
  return Math.min(262_144, Math.max(0, Math.round(value)));
}
