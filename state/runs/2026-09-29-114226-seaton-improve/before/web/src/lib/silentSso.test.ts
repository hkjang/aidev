import { describe, expect, it } from "vitest";
import {
  clearSilentSsoState,
  loginPathFor,
  markSignedOut,
  returnToFrom,
  safeReturnTo,
  shouldAttemptSilentSso,
  silentSsoStartUrl,
  ssoStartUrl,
  type StorageOpener,
} from "./silentSso";

// 브라우저 sessionStorage 를 흉내 내는 메모리 저장소. 한 인스턴스가 한 탭이다.
const memoryStorage = (): StorageOpener => {
  const data = new Map<string, string>();
  const storage = {
    getItem: (key: string) => data.get(key) ?? null,
    setItem: (key: string, value: string) => void data.set(key, value),
    removeItem: (key: string) => void data.delete(key),
  };
  return () => storage;
};
// 사생활 보호 모드처럼 저장소를 여는 것 자체가 예외를 내는 경우.
const brokenStorage: StorageOpener = () => {
  throw new Error("SecurityError");
};

const on = { oidcEnabled: true, oidcAutoLogin: true };
const root = { pathname: "/", search: "" };

describe("shouldAttemptSilentSso", () => {
  it("auto_login 이 꺼져 있거나 SSO 가 꺼져 있으면 시도하지 않는다", () => {
    const open = memoryStorage();
    expect(shouldAttemptSilentSso(null, root, open)).toBe(false);
    expect(
      shouldAttemptSilentSso({ oidcEnabled: true, oidcAutoLogin: false }, root, open),
    ).toBe(false);
    expect(
      shouldAttemptSilentSso({ oidcEnabled: false, oidcAutoLogin: true }, root, open),
    ).toBe(false);
    expect(shouldAttemptSilentSso({ oidcEnabled: true }, root, open)).toBe(false);
    expect(shouldAttemptSilentSso(on, root, open)).toBe(true);
  });

  it("한 탭 세션에 한 번만 시도한다", () => {
    const open = memoryStorage();
    expect(shouldAttemptSilentSso(on, root, open)).toBe(true);
    silentSsoStartUrl("/", open);
    // 거절당한 뒤 새로고침 — 다시 시도하면 루프다.
    expect(shouldAttemptSilentSso(on, root, open)).toBe(false);
    // 새 탭은 저장소가 비어 있으므로 다시 시도한다.
    expect(shouldAttemptSilentSso(on, root, memoryStorage())).toBe(true);
  });

  it("콜백이 주소에 남긴 거절 표시가 있으면 저장소가 비어 있어도 시도하지 않는다", () => {
    const open = memoryStorage();
    expect(
      shouldAttemptSilentSso(on, { pathname: "/", search: "?sso=none" }, open),
    ).toBe(false);
    expect(
      shouldAttemptSilentSso(on, { pathname: "/", search: "?sso=error" }, open),
    ).toBe(false);
    expect(
      shouldAttemptSilentSso(on, { pathname: "/", search: "?floor=3" }, open),
    ).toBe(true);
  });

  it("스스로 로그아웃한 뒤에는 시도하지 않고, 다시 로그인하면 억제가 풀린다", () => {
    const open = memoryStorage();
    markSignedOut(open);
    expect(shouldAttemptSilentSso(on, root, open)).toBe(false);
    clearSilentSsoState(open);
    expect(shouldAttemptSilentSso(on, root, open)).toBe(true);
  });

  it("저장소를 읽지 못하면 '이미 시도했다'로 쳐서 막히는 쪽으로 실패한다", () => {
    expect(shouldAttemptSilentSso(on, root, brokenStorage)).toBe(false);
    // 표시를 쓰지 못해도 예외를 밖으로 내지 않는다.
    expect(() => markSignedOut(brokenStorage)).not.toThrow();
    expect(() => clearSilentSsoState(brokenStorage)).not.toThrow();
    expect(() => silentSsoStartUrl("/", brokenStorage)).not.toThrow();
  });

  it("콜백·로그인·API·MCP·헬스 경로에서는 시도하지 않는다", () => {
    for (const pathname of [
      "/login",
      "/login/",
      "/api/v1/auth/oidc/callback",
      "/api/v1/seats",
      "/mcp",
      "/healthz",
      "/readyz",
    ]) {
      expect(
        shouldAttemptSilentSso(on, { pathname, search: "" }, memoryStorage()),
      ).toBe(false);
    }
    for (const pathname of ["/", "/admin/maps", "/profile/keys"]) {
      expect(
        shouldAttemptSilentSso(on, { pathname, search: "" }, memoryStorage()),
      ).toBe(true);
    }
  });
});

describe("silentSsoStartUrl", () => {
  it("깊은 링크를 returnTo 로 들고 가고 prompt=none 을 붙인다", () => {
    const url = new URL(
      silentSsoStartUrl("/admin/maps?floor=3", memoryStorage()),
      "http://seaton.test",
    );
    expect(url.pathname).toBe("/api/v1/auth/oidc/start");
    expect(url.searchParams.get("prompt")).toBe("none");
    expect(url.searchParams.get("returnTo")).toBe("/admin/maps?floor=3");
  });

  it("밖으로 나가는 returnTo 는 '/' 로 바꾼다", () => {
    expect(safeReturnTo("/admin/maps")).toBe("/admin/maps");
    expect(safeReturnTo("//evil.example")).toBe("/");
    expect(safeReturnTo("/\\evil.example")).toBe("/");
    expect(safeReturnTo("https://evil.example/")).toBe("/");
    expect(safeReturnTo("")).toBe("/");
  });
});

// 주소를 짓는 쪽(App 의 Protected)과 읽는 쪽(LoginPage)이 같은 규칙을 쓰는지는
// 왕복으로만 알 수 있다. 두 곳이 각자 문자열을 지으면 한쪽만 고쳐도 테스트가
// 통과해 버린다.
describe("로그인 주소의 returnTo 왕복", () => {
  const roundTrip = (pathname: string, search = "") =>
    returnToFrom(new URL(loginPathFor(pathname, search), "http://seaton.test").search);

  it("깊은 링크는 쿼리째 보존된다", () => {
    expect(loginPathFor("/admin/maps", "")).toBe("/login?returnTo=%2Fadmin%2Fmaps");
    expect(roundTrip("/admin/maps")).toBe("/admin/maps");
    expect(roundTrip("/admin/maps", "?floor=3")).toBe("/admin/maps?floor=3");
    expect(roundTrip("/profile/keys", "?q=%EA%B0%9C%EB%B0%9C")).toBe(
      "/profile/keys?q=%EA%B0%9C%EB%B0%9C",
    );
    // 좌석맵은 기본 경로에서 map·q·edit 를 읽는다(SeatMapPage). 경로가 '/' 여도
    // 쿼리가 있으면 그것은 공유된 깊은 링크이므로 버리지 않는다.
    expect(roundTrip("/", "?map=abc&q=%EA%B9%80")).toBe("/?map=abc&q=%EA%B9%80");
  });

  it("기본 경로에서 밀려난 경우에는 returnTo 를 붙이지 않는다", () => {
    // helpers.ts 의 login() 과 '/' 진입이 지금 그대로 돌아야 한다. 로그인 뒤
    // 주소에 '/login' 이 남으면 모든 E2E 가 멎는다.
    expect(loginPathFor("/", "")).toBe("/login");
    expect(loginPathFor("/")).toBe("/login");
    expect(returnToFrom("")).toBe("/");
    expect(returnToFrom("?error=state")).toBe("/");
  });

  it("로그인 화면 자신은 returnTo 가 되지 않는다", () => {
    // returnTo=/login 이면 로그인 성공 뒤 다시 로그인 화면으로 돌아가 멎는다.
    expect(loginPathFor("/login", "?returnTo=%2Flogin")).toBe("/login");
    expect(returnToFrom("?returnTo=%2Flogin")).toBe("/");
    expect(returnToFrom("?returnTo=%2Flogin%3FreturnTo%3D%252Flogin")).toBe("/");
  });

  it("밖으로 나가는 값은 짓는 쪽과 읽는 쪽 모두에서 '/' 로 접힌다", () => {
    for (const bad of ["//evil.example", "https://evil.example", "/\\evil"]) {
      expect(loginPathFor(bad, "")).toBe("/login");
      expect(returnToFrom(`?returnTo=${encodeURIComponent(bad)}`)).toBe("/");
    }
  });
});

describe("ssoStartUrl", () => {
  it("returnTo 가 있으면 들고 가고, 없으면 파라미터 없이 출발한다", () => {
    expect(ssoStartUrl("/")).toBe("/api/v1/auth/oidc/start");
    expect(ssoStartUrl("/admin/maps?floor=3")).toBe(
      "/api/v1/auth/oidc/start?returnTo=%2Fadmin%2Fmaps%3Ffloor%3D3",
    );
    expect(ssoStartUrl("https://evil.example")).toBe("/api/v1/auth/oidc/start");
  });

  it("조용한 시도와 달리 prompt 를 붙이지 않는다", () => {
    expect(ssoStartUrl("/admin/maps")).not.toContain("prompt");
  });
});
