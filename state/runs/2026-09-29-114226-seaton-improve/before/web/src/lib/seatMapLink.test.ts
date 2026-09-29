import { describe, expect, it } from "vitest";
import {
  readSeatMapParams,
  seatSearchKey,
  writeSeatMapParams,
} from "./seatMapLink";

const read = (search: string) => readSeatMapParams(new URLSearchParams(search));
const write = (
  search: string,
  next: { mapId?: string; query?: string },
): string => writeSeatMapParams(new URLSearchParams(search), next).toString();

describe("readSeatMapParams", () => {
  it("비어 있는 주소에서는 도면도 검색어도 없고 편집 모드가 아니다", () => {
    expect(read("")).toEqual({ mapId: "", query: "", edit: false });
  });

  it("map·q·edit 을 읽는다", () => {
    expect(read("?map=m1&q=김개발&edit=1")).toEqual({
      mapId: "m1",
      query: "김개발",
      edit: true,
    });
  });

  it("공백만 있는 q 는 없는 것과 같다", () => {
    expect(read("?q=%20%20").query).toBe("");
  });

  it("edit 은 1 일 때만 참이다", () => {
    expect(read("?edit=0").edit).toBe(false);
    expect(read("?edit=true").edit).toBe(false);
  });
});

describe("writeSeatMapParams", () => {
  it("검색어를 q 로 쓴다", () => {
    expect(write("", { query: "김개발" })).toBe(
      "q=%EA%B9%80%EA%B0%9C%EB%B0%9C",
    );
  });

  it("빈 검색어는 q 를 지운다", () => {
    expect(write("?q=김개발", { query: "" })).toBe("");
    expect(write("?q=김개발", { query: "   " })).toBe("");
  });

  it("edit 과 모르는 키는 그대로 보존한다", () => {
    const out = new URLSearchParams(
      write("?edit=1&floor=3&q=옛값", { query: "새값" }),
    );
    expect(out.get("edit")).toBe("1");
    expect(out.get("floor")).toBe("3");
    expect(out.get("q")).toBe("새값");
  });

  it("도면을 갱신해도 검색어는 건드리지 않는다", () => {
    const out = new URLSearchParams(write("?q=김개발&map=m1", { mapId: "m2" }));
    expect(out.get("map")).toBe("m2");
    expect(out.get("q")).toBe("김개발");
  });

  it("주지 않은 키는 손대지 않는다", () => {
    expect(write("?map=m1&q=김개발", {})).toBe(
      new URLSearchParams("?map=m1&q=김개발").toString(),
    );
  });

  it("빈 도면 id 는 map 을 지운다", () => {
    expect(write("?map=m1", { mapId: "" })).toBe("");
  });

  it("쓴 값을 그대로 다시 읽는다(왕복)", () => {
    const written = writeSeatMapParams(new URLSearchParams("?edit=1"), {
      mapId: "m9",
      query: " 김개발 ",
    });
    expect(readSeatMapParams(written)).toEqual({
      mapId: "m9",
      query: "김개발",
      edit: true,
    });
  });
});

describe("seatSearchKey", () => {
  it("도면과 검색어가 같으면 같은 키다", () => {
    expect(seatSearchKey("m1", " 김개발 ")).toBe(seatSearchKey("m1", "김개발"));
  });

  it("도면이 다르면 다른 키다", () => {
    expect(seatSearchKey("m1", "김개발")).not.toBe(
      seatSearchKey("m2", "김개발"),
    );
  });

  it("주소에 쓴 뒤 읽은 검색어가 제출 때와 같은 키를 낸다", () => {
    // 제출 경로와 ?q= 감시 경로가 다른 키를 내면 같은 검색이 두 번 나간다.
    const submitted = " 김개발 ";
    const params = writeSeatMapParams(new URLSearchParams(""), {
      query: submitted,
    });
    expect(seatSearchKey("m1", readSeatMapParams(params).query)).toBe(
      seatSearchKey("m1", submitted),
    );
  });
});
