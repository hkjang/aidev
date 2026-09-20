import type { PublicConfig } from "../../types";

/**
 * The wording AppStore ships with. An administrator can replace any of these
 * from 시스템 설정; an empty setting means "keep the default", so the banner
 * still reads correctly on a fresh install and while the config request is
 * still in flight.
 */
export const DEFAULT_HERO = {
  eyebrow: "보는 것은 자유롭게, 등록부터 안전하게",
  title: "팀의 좋은 앱을\n한곳에서 발견하세요.",
  description:
    "AppStore는 조직의 서비스를 공개적으로 탐색하고 SSO 기반으로 등록·관리하는 개발자 애플리케이션 카탈로그입니다.",
  primaryLabel: "추천 앱 보기",
  secondaryLabel: "모든 앱 탐색",
} as const;

export interface HeroCopy {
  eyebrow: string;
  /** The title split on its line breaks, so the editor controls where it wraps. */
  titleLines: string[];
  description: string;
  primaryLabel: string;
  secondaryLabel: string;
}

function pick(value: string | undefined, fallback: string) {
  return value?.trim() || fallback;
}

export function titleLines(title: string): string[] {
  const lines = title
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean);
  return lines.length ? lines : [title.trim()];
}

export function heroCopy(config?: PublicConfig): HeroCopy {
  return {
    eyebrow: pick(config?.heroEyebrow, DEFAULT_HERO.eyebrow),
    titleLines: titleLines(pick(config?.heroTitle, DEFAULT_HERO.title)),
    description: pick(config?.siteDescription, DEFAULT_HERO.description),
    primaryLabel: pick(config?.heroPrimaryLabel, DEFAULT_HERO.primaryLabel),
    secondaryLabel: pick(
      config?.heroSecondaryLabel,
      DEFAULT_HERO.secondaryLabel,
    ),
  };
}
