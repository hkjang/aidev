/**
 * 방문 추적 설정 화면의 순수 규칙. 서버(internal/tracking)와 같은 어휘를 쓴다.
 *
 * provider 목록은 Momento 가 첫 자리다 — 사내 자체 호스팅 수집기라 데이터가
 * 밖으로 나가지 않는 유일한 선택지다.
 */
export const TRACKING_PROVIDERS = [
  { value: "momento", label: "Momento (사내 수집기)" },
  { value: "ga4", label: "Google Analytics 4" },
  { value: "gtm", label: "Google Tag Manager" },
  { value: "matomo", label: "Matomo" },
  { value: "custom", label: "직접 붙여넣기" },
  { value: "none", label: "없음" },
] as const;

export type TrackingProvider = (typeof TRACKING_PROVIDERS)[number]["value"];

export const MAX_SNIPPET_BYTES = 8 * 1024;

export type TrackingField = {
  key: string;
  label: string;
  help?: string;
  secret?: boolean;
  type?: string;
  multiline?: boolean;
};

/** provider 가 필요로 하는 입력 칸. 다른 provider 의 값은 건드리지 않고 숨긴다. */
export function trackingFields(provider: string): TrackingField[] {
  switch (provider) {
    case "momento":
      return [
        {
          key: "tracking.momento_url",
          label: "Momento 수집기 주소",
          help: "예: https://momento.intra · tracker.js 와 /collect/v1/events 를 내주는 곳",
        },
        {
          key: "tracking.momento_site_id",
          label: "사이트 ID",
          help: "Momento 관리 센터 → 사이트에서 만든 SITE_… 키",
        },
      ];
    case "ga4":
    case "gtm":
      return [
        {
          key: "tracking.measurement_id",
          label: provider === "ga4" ? "측정 ID (G-…)" : "컨테이너 ID (GTM-…)",
        },
      ];
    case "matomo":
      return [
        {
          key: "tracking.matomo_url",
          label: "Matomo 주소",
          help: "예: https://matomo.intra",
        },
        { key: "tracking.matomo_site_id", label: "사이트 ID" },
      ];
    case "custom":
      return [
        {
          key: "tracking.custom_snippet",
          label: "추적 스니펫 (HTML)",
          help: "<script> 태그를 그대로 붙여 넣습니다. 8KB 까지. 안에 적힌 http(s) 주소는 자동으로 정책에 더해집니다",
          multiline: true,
        },
      ];
    default:
      return [];
  }
}

/** UTF-8 바이트 수. 서버는 바이트로 제한하므로 글자 수로 세면 한글에서 어긋난다. */
export function snippetBytes(snippet: string): number {
  return new TextEncoder().encode(snippet).length;
}

/**
 * 쉼표로 구분한 허용 목록에 출처 하나를 더한다. 이미 있으면 그대로 두고 기존
 * 항목의 순서는 바꾸지 않는다. 서버의 tracking.AddAllowedHost 와 같은 규칙.
 */
export function addAllowedHost(existing: string, origin: string): string {
  const cleaned = origin.trim().replace(/\/$/, "");
  if (!cleaned) return existing;
  const hosts = existing
    .split(/[\s,]+/)
    .map((host) => host.trim())
    .filter(Boolean);
  if (hosts.some((host) => host.toLowerCase() === cleaned.toLowerCase()))
    return existing;
  return existing.trim() ? `${existing.trim()}, ${cleaned}` : cleaned;
}

/**
 * 설정값 묶음이 켜졌을 때 실제로 스니펫이 붙는지. 화면의 상태 칩과 안내가
 * "켜짐" 이라고 말해 놓고 아무것도 안 붙는 상황을 막는다.
 */
export function trackingActive(values: Record<string, string>): boolean {
  if (values["tracking.enabled"] !== "true") return false;
  const provider = values["tracking.provider"] || "momento";
  const has = (key: string) => (values[key] ?? "").trim() !== "";
  switch (provider) {
    case "momento":
      return has("tracking.momento_url") && has("tracking.momento_site_id");
    case "ga4":
    case "gtm":
      return has("tracking.measurement_id");
    case "matomo":
      return has("tracking.matomo_url") && has("tracking.matomo_site_id");
    case "custom":
      return has("tracking.custom_snippet");
    default:
      return false;
  }
}

/** 프록시를 쓰면 정책에 더할 외부 출처가 없다. 안내문이 이 사실을 말한다. */
export function usesSameOriginProxy(values: Record<string, string>): boolean {
  return (
    (values["tracking.provider"] || "momento") === "momento" &&
    (values["tracking.momento_proxy"] ?? "true") === "true"
  );
}
