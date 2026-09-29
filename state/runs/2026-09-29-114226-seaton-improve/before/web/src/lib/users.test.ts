import { describe, expect, it } from "vitest";
import {
  canChangeRole,
  canToggleActive,
  emailEditable,
  normalizeEmail,
} from "./users";

describe("normalizeEmail", () => {
  it("앞뒤 공백을 걷어내고 빈 값은 지우라는 뜻으로 받는다", () => {
    expect(normalizeEmail("  admin@corp.example ")).toEqual({
      value: "admin@corp.example",
    });
    expect(normalizeEmail("   ")).toEqual({ value: "" });
  });
  it("주소 하나가 아닌 것은 거절한다", () => {
    for (const raw of [
      "admin.corp.example",
      "@corp.example",
      "admin@",
      "관리자 <admin@corp.example>",
      "a@corp.example, b@corp.example",
      "a@b@corp.example",
      "a".repeat(250) + "@corp.example",
    ]) {
      expect(normalizeEmail(raw).error, raw).toBeTruthy();
    }
  });
});

describe("emailEditable", () => {
  it("로컬 계정만 화면에서 주소를 고친다", () => {
    expect(emailEditable({ source: "local" })).toBe(true);
    expect(emailEditable({ source: "oidc" })).toBe(false);
  });
});

describe("canToggleActive", () => {
  const me = { id: "me" };
  it("자기 계정은 비활성화할 수 없다", () => {
    expect(canToggleActive({ id: "me", active: true }, me)).toBe(false);
  });
  it("다른 계정은 끌 수 있고, 꺼진 계정은 누구든 다시 켤 수 있다", () => {
    expect(canToggleActive({ id: "u2", active: true }, me)).toBe(true);
    expect(canToggleActive({ id: "u2", active: false }, me)).toBe(true);
    expect(canToggleActive({ id: "me", active: false }, me)).toBe(true);
  });
  it("내가 누군지 모르면 막지 않는다", () => {
    expect(canToggleActive({ id: "u2", active: true }, null)).toBe(true);
  });
});

describe("canChangeRole", () => {
  const me = { id: "me" };
  it("자기 계정의 권한은 바꿀 수 없다", () => {
    expect(canChangeRole({ id: "me" }, me)).toBe(false);
  });
  it("다른 계정의 권한은 바꿀 수 있다", () => {
    expect(canChangeRole({ id: "u2" }, me)).toBe(true);
  });
  it("내가 누군지 모르면 막지 않는다", () => {
    expect(canChangeRole({ id: "me" }, null)).toBe(true);
  });
});
