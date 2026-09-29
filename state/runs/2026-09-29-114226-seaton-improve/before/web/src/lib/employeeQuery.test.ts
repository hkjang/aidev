import { describe, expect, it } from "vitest";
import { EMPLOYEE_QUERY_LIMIT, employeeQuery } from "./employeeQuery";

const query = (next: Parameters<typeof employeeQuery>[0] = {}): string =>
  employeeQuery(next).toString();

describe("employeeQuery", () => {
  it("아무 조건도 없으면 limit 만 붙는다", () => {
    expect(query()).toBe("limit=500");
  });

  it("빈 문자열은 키를 아예 넣지 않는다", () => {
    expect(
      query({ q: "", status: "", assignment: "", organizationId: "" }),
    ).toBe("limit=500");
  });

  it("공백만 있는 값도 없는 것과 같다", () => {
    expect(query({ q: "   ", organizationId: "  " })).toBe("limit=500");
  });

  it("조직을 고르면 organizationId 키로 실린다", () => {
    expect(query({ organizationId: "org-1" })).toBe(
      "limit=500&organizationId=org-1",
    );
  });

  it("네 조건이 모두 있으면 모두 실린다", () => {
    const params = employeeQuery({
      q: "김개발",
      status: "active",
      assignment: "assigned",
      organizationId: "org-1",
    });
    expect(params.get("q")).toBe("김개발");
    expect(params.get("status")).toBe("active");
    expect(params.get("assignment")).toBe("assigned");
    expect(params.get("organizationId")).toBe("org-1");
    expect(params.get("limit")).toBe("500");
  });

  it("검색어의 공백은 떼고 싣는다", () => {
    expect(employeeQuery({ q: "  김개발  " }).get("q")).toBe("김개발");
  });

  it("limit 은 조건과 무관하게 언제나 500 이다", () => {
    expect(employeeQuery({}).get("limit")).toBe("500");
    expect(employeeQuery({ organizationId: "org-1" }).get("limit")).toBe("500");
    expect(EMPLOYEE_QUERY_LIMIT).toBe("500");
  });

  it("호출자가 준 값을 담은 새 객체를 돌려준다 — 부르는 쪽끼리 섞이지 않는다", () => {
    const first = employeeQuery({ organizationId: "org-1" });
    const second = employeeQuery({});
    expect(first.get("organizationId")).toBe("org-1");
    expect(second.get("organizationId")).toBeNull();
  });
});
