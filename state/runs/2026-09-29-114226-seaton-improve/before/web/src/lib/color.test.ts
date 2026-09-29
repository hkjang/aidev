import { describe, expect, it } from "vitest";
import { readableInk } from "./color";

describe("readableInk", () => {
  it("밝은 배경에는 어두운 글자를 쓴다", () => {
    expect(readableInk("#FFFFFF")).toBe("#203846");
    expect(readableInk("#DFE7EB")).toBe("#203846");
    expect(readableInk("#FFB703")).toBe("#203846");
  });
  it("어두운 배경에는 흰 글자를 쓴다", () => {
    expect(readableInk("#087E8B")).toBe("#FFFFFF");
    expect(readableInk("#0E2D3E")).toBe("#FFFFFF");
    expect(readableInk("#3478C8")).toBe("#FFFFFF");
  });
  it("3자리 축약형도 다룬다", () => {
    expect(readableInk("#fff")).toBe("#203846");
    expect(readableInk("#000")).toBe("#FFFFFF");
  });
  it("해석할 수 없는 값은 어두운 글자로 물러난다", () => {
    expect(readableInk("rgb(1,2,3)")).toBe("#203846");
    expect(readableInk("")).toBe("#203846");
  });
});
