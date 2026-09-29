import { describe, expect, it } from "vitest";
import {
  MAX_SNIPPET_BYTES,
  TRACKING_PROVIDERS,
  addAllowedHost,
  snippetBytes,
  trackingActive,
  trackingFields,
  usesSameOriginProxy,
} from "./tracking";

describe("TRACKING_PROVIDERS", () => {
  it("Momento 가 첫 자리다", () => {
    expect(TRACKING_PROVIDERS[0].value).toBe("momento");
  });
});

describe("trackingFields", () => {
  it("provider 마다 그 provider 의 칸만 보인다", () => {
    expect(trackingFields("momento").map((f) => f.key)).toEqual([
      "tracking.momento_url",
      "tracking.momento_site_id",
    ]);
    expect(trackingFields("ga4").map((f) => f.key)).toEqual([
      "tracking.measurement_id",
    ]);
    expect(trackingFields("gtm")[0].label).toContain("GTM");
    expect(trackingFields("matomo").map((f) => f.key)).toEqual([
      "tracking.matomo_url",
      "tracking.matomo_site_id",
    ]);
    expect(trackingFields("custom")[0].multiline).toBe(true);
    expect(trackingFields("none")).toEqual([]);
  });
});

describe("addAllowedHost", () => {
  it("빈 목록에는 출처만, 있으면 쉼표로 이어 붙인다", () => {
    expect(addAllowedHost("", "https://a.intra/")).toBe("https://a.intra");
    expect(addAllowedHost("https://a.intra", "https://b.intra")).toBe(
      "https://a.intra, https://b.intra",
    );
  });
  it("이미 있으면(대소문자 무시) 그대로 둔다", () => {
    const list = "https://a.intra, https://b.intra";
    expect(addAllowedHost(list, "HTTPS://A.intra")).toBe(list);
    expect(addAllowedHost(list, "  ")).toBe(list);
  });
  it("줄바꿈으로 나눈 목록도 중복을 알아본다", () => {
    expect(
      addAllowedHost("https://a.intra\nhttps://b.intra", "https://b.intra"),
    ).toBe("https://a.intra\nhttps://b.intra");
  });
});

describe("snippetBytes", () => {
  it("바이트로 센다 — 한글은 글자당 3바이트", () => {
    expect(snippetBytes("abc")).toBe(3);
    expect(snippetBytes("가")).toBe(3);
    expect(snippetBytes("a".repeat(MAX_SNIPPET_BYTES + 1))).toBeGreaterThan(
      MAX_SNIPPET_BYTES,
    );
  });
});

describe("trackingActive", () => {
  const momento = {
    "tracking.enabled": "true",
    "tracking.momento_url": "https://momento.intra",
    "tracking.momento_site_id": "SITE_1",
  };
  it("꺼져 있으면 무엇을 채워도 붙지 않는다", () => {
    expect(trackingActive({ ...momento, "tracking.enabled": "false" })).toBe(
      false,
    );
    expect(trackingActive({})).toBe(false);
  });
  it("provider 의 필수 칸이 비면 붙지 않는다", () => {
    expect(trackingActive(momento)).toBe(true);
    expect(
      trackingActive({ ...momento, "tracking.momento_site_id": " " }),
    ).toBe(false);
    expect(
      trackingActive({
        "tracking.enabled": "true",
        "tracking.provider": "ga4",
      }),
    ).toBe(false);
    expect(
      trackingActive({
        "tracking.enabled": "true",
        "tracking.provider": "custom",
        "tracking.custom_snippet": "<script></script>",
      }),
    ).toBe(true);
    expect(
      trackingActive({
        "tracking.enabled": "true",
        "tracking.provider": "none",
      }),
    ).toBe(false);
  });
});

describe("usesSameOriginProxy", () => {
  it("Momento 이고 프록시가 켜져 있을 때만(기본값) 참", () => {
    expect(usesSameOriginProxy({})).toBe(true);
    expect(usesSameOriginProxy({ "tracking.momento_proxy": "false" })).toBe(
      false,
    );
    expect(usesSameOriginProxy({ "tracking.provider": "matomo" })).toBe(false);
  });
});
