// 배경색의 상대 휘도로 글자색을 고른다. 조직 색을 관리자가 자유롭게 정하므로
// 색상 모드만 보고 글자색을 정하면 어두운 조직 색 위에 검은 글씨가 놓인다.
export const readableInk = (background: string) => {
  const hex = background.replace("#", "");
  const full =
    hex.length === 3
      ? hex
          .split("")
          .map((c) => c + c)
          .join("")
      : hex;
  if (full.length !== 6) return "#203846";
  const [r, g, b] = [0, 2, 4].map(
    (i) => parseInt(full.slice(i, i + 2), 16) / 255,
  );
  const channel = (v: number) =>
    v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
  const luminance =
    0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b);
  return luminance > 0.45 ? "#203846" : "#FFFFFF";
};
