import type { Property } from "../api";
import { derivedProperty } from "../DatabaseAdvanced";

export function parseCellText(
  property: Property,
  text: string,
): { value: unknown; error?: string } {
  if (derivedProperty(property) || ["ai", "relation"].includes(property.type))
    return {
      value: null,
      error:
        "이 속성은 셀 붙여넣기로 변경하지 않습니다. 항목 상세에서 확인하세요.",
    };
  const value = text.trim();
  if (!value) return { value: null };
  if (["number", "progress"].includes(property.type)) {
    if (
      !/^[+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:e[+-]?\d+)?$/i.test(value) ||
      !Number.isFinite(Number(value))
    )
      return { value: text, error: "숫자 형식을 확인하세요." };
    const number = Number(value);
    if (property.type === "progress" && (number < 0 || number > 100))
      return { value: text, error: "진행률은0~100 사이입니다." };
    return { value: number };
  }
  if (property.type === "checkbox") {
    if (["true", "예", "완료", "1", "✓"].includes(value.toLowerCase()))
      return { value: true };
    if (["false", "아니요", "미완료", "0"].includes(value.toLowerCase()))
      return { value: false };
    return { value: text, error: "예/아니요 또는 true/false를 입력하세요." };
  }
  if (["select", "status"].includes(property.type))
    return (property.options || []).includes(value)
      ? { value }
      : { value, error: "등록된 선택 옵션이 아닙니다." };
  if (["multi_select", "multiselect"].includes(property.type)) {
    const list = [
      ...new Set(
        value
          .split(",")
          .map((v) => v.trim())
          .filter(Boolean),
      ),
    ];
    return list.every((v) => (property.options || []).includes(v))
      ? { value: list }
      : {
          value: list,
          error: "쉼표로 구분한 값 중 등록되지 않은 옵션이 있습니다.",
        };
  }
  if (property.type === "date") {
    if (
      !/^\d{4}-\d{2}-\d{2}$/.test(value) ||
      !Number.isFinite(Date.parse(value)) ||
      new Date(value).toISOString().slice(0, 10) !== value
    )
      return {
        value: text,
        error: "날짜는 실제 존재하는 YYYY-MM-DD 형식이어야 합니다.",
      };
    return { value };
  }
  return { value: text };
}

/** Quoted TSV accepts embedded tabs/newlines without guessing cell boundaries. */
export function parseTabularPaste(text: string): string[][] {
  if (new TextEncoder().encode(text).length > 1024 * 1024)
    throw Error("범위 붙여넣기는1MiB까지 지원합니다.");
  const rows: string[][] = [];
  let row: string[] = [],
    value = "",
    quoted = false,
    closed = false;
  const pushCell = () => {
    row.push(value);
    value = "";
    closed = false;
  };
  const pushRow = () => {
    pushCell();
    rows.push(row);
    row = [];
    if (rows.length > 100)
      throw Error("한 번에100행까지 붙여넣을 수 있습니다.");
  };
  for (let i = 0; i < text.length; i++) {
    const c = text[i];
    if (quoted) {
      if (c === '"') {
        if (text[i + 1] === '"') {
          value += '"';
          i++;
        } else {
          quoted = false;
          closed = true;
        }
      } else value += c;
      continue;
    }
    if (c === '"' && value === "" && !closed) {
      quoted = true;
      continue;
    }
    if (c === "\t") {
      pushCell();
      continue;
    }
    if (c === "\n" || c === "\r") {
      if (c === "\r" && text[i + 1] === "\n") i++;
      pushRow();
      continue;
    }
    if (closed) throw Error("인용부호 뒤의 셀 구분을 확인하세요.");
    value += c;
  }
  if (quoted) throw Error("닫히지 않은 인용부호가 있습니다.");
  if (value || row.length || !rows.length) pushRow();
  const columns = rows[0]?.length || 0;
  if (!columns || rows.some((r) => r.length !== columns))
    throw Error("행마다 열 개수가 다릅니다. 같은 직사각형 범위로 복사하세요.");
  if (rows.reduce((n, r) => n + r.length, 0) > 500)
    throw Error("한 번에500셀까지 붙여넣을 수 있습니다.");
  return rows;
}
