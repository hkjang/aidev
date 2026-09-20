import { describe, expect, it } from "vitest";
import { DEFAULT_HERO, heroCopy } from "./hero-copy";
import type { PublicConfig } from "../../types";

const config = (overrides: Partial<PublicConfig>): PublicConfig => ({
  siteName: "AppStore",
  publicMode: true,
  oidcEnabled: false,
  workflowEnabled: false,
  ...overrides,
});

describe("store banner copy", () => {
  it("shows the shipped wording before any configuration arrives", () => {
    const copy = heroCopy(undefined);
    expect(copy.eyebrow).toBe(DEFAULT_HERO.eyebrow);
    expect(copy.description).toBe(DEFAULT_HERO.description);
    expect(copy.titleLines).toEqual(["팀의 좋은 앱을", "한곳에서 발견하세요."]);
  });

  it("prefers what the administrator wrote and keeps their line breaks", () => {
    const copy = heroCopy(
      config({
        heroEyebrow: "사내 서비스 카탈로그",
        heroTitle: "필요한 도구를\n바로 찾으세요.",
        siteDescription: "팀이 만든 서비스를 한 곳에서 봅니다.",
        heroPrimaryLabel: "지금 둘러보기",
      }),
    );
    expect(copy.eyebrow).toBe("사내 서비스 카탈로그");
    expect(copy.titleLines).toEqual(["필요한 도구를", "바로 찾으세요."]);
    expect(copy.description).toBe("팀이 만든 서비스를 한 곳에서 봅니다.");
    expect(copy.primaryLabel).toBe("지금 둘러보기");
    // Untouched fields keep the default rather than going blank.
    expect(copy.secondaryLabel).toBe(DEFAULT_HERO.secondaryLabel);
  });

  it("treats blank wording as unset so the banner never empties out", () => {
    const copy = heroCopy(config({ heroTitle: "   ", siteDescription: "" }));
    expect(copy.titleLines).toEqual(["팀의 좋은 앱을", "한곳에서 발견하세요."]);
    expect(copy.description).toBe(DEFAULT_HERO.description);
  });
});
