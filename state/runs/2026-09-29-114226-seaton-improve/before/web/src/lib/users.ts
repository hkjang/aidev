import type { User } from "../types";

/**
 * 사용자 권한 화면의 편집 규칙. 서버(validateUserPatch)와 같은 판단을 화면에서
 * 먼저 해 왕복 없이 안내한다.
 */

/** 메일 주소 입력을 다듬는다. 빈 값은 "주소 없음"으로 지우라는 뜻이다. */
export function normalizeEmail(raw: string): { value: string; error?: string } {
  const value = raw.trim();
  if (value === "") return { value };
  // 서버와 같은 기준: 주소 하나만, 표시 이름·여러 주소·공백은 받지 않는다.
  const at = value.indexOf("@");
  const local = value.slice(0, at);
  const domain = value.slice(at + 1);
  if (
    value.length > 254 ||
    /[\s,;<>]/.test(value) ||
    at <= 0 ||
    domain === "" ||
    domain.includes("@") ||
    local === ""
  ) {
    return { value, error: "메일 주소 형식이 올바르지 않습니다" };
  }
  return { value };
}

/**
 * 주소를 화면에서 고칠 수 있는 계정. SSO 사용자의 주소는 로그인할 때마다
 * Keycloak 프로필로 덮어쓰므로 여기서 고쳐도 다음 로그인에 사라진다.
 */
export function emailEditable(user: Pick<User, "source">): boolean {
  return user.source === "local";
}

/**
 * 비활성화 스위치를 만질 수 있는 계정. 자기 계정을 잠그면 마지막 관리자가
 * 화면으로는 되돌릴 길이 없으므로 막는다. 다시 켜는 것은 언제나 된다.
 */
export function canToggleActive(
  user: Pick<User, "id" | "active">,
  me: Pick<User, "id"> | null,
): boolean {
  if (!user.active) return true;
  return me === null || user.id !== me.id;
}

/**
 * 권한 선택을 만질 수 있는 계정. 자기 권한을 낮추면 이 화면을 더는 열 수 없어
 * 되돌릴 길이 없으므로 서버(self_demotion)와 같이 자기 행은 막는다.
 */
export function canChangeRole(
  user: Pick<User, "id">,
  me: Pick<User, "id"> | null,
): boolean {
  return me === null || user.id !== me.id;
}
