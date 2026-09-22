export function serverOrigin(value, allowHTTP = false) {
  let url;
  try {
    url = new URL(value.trim());
  } catch {
    throw new Error("madi 서버의 올바른 주소를 입력하세요.");
  }
  if (
    url.username ||
    url.password ||
    url.search ||
    url.hash ||
    url.pathname !== "/" ||
    !["https:", "http:"].includes(url.protocol)
  )
    throw new Error(
      "경로나 계정 정보 없이 서버 주소만 입력하세요. 예: https://madi.company.local",
    );
  if (
    url.protocol === "http:" &&
    !["localhost", "127.0.0.1", "[::1]"].includes(url.hostname) &&
    !allowHTTP
  )
    throw new Error(
      "HTTP 연결은 암호화되지 않습니다. 사내 HTTP 사용 확인이 필요합니다. HTTPS를 권장합니다.",
    );
  return url.origin;
}
export function sourceURL(value) {
  try {
    const url = new URL(value);
    if (
      !["http:", "https:"].includes(url.protocol) ||
      url.username ||
      url.password
    )
      return "";
    url.hash = "";
    return url.href;
  } catch {
    return "";
  }
}
export function captureMarkdown(capture) {
  const text = String(capture.text || "").slice(0, 900000);
  const author = String(capture.author || "")
    .replace(/[\r\n]/g, " ")
    .slice(0, 300);
  return [author ? "작성자: " + author : "", text].filter(Boolean).join("\n\n");
}
export function collectPage(mode) {
  const clean = (value) =>
    String(value || "")
      .replace(/\u0000/g, "")
      .trim();
  const selected = clean(window.getSelection()?.toString());
  const title = clean(
    document
      .querySelector('meta[property="og:title"]')
      ?.getAttribute("content") || document.title,
  ).slice(0, 250);
  const author = clean(
    document.querySelector('meta[name="author"]')?.getAttribute("content") ||
      document.querySelector('[rel="author"]')?.textContent,
  ).slice(0, 300);
  const main =
    document.querySelector("article") ||
    document.querySelector("main") ||
    document.querySelector('[role="main"]') ||
    document.body;
  let text = "";
  if (mode === "selection") text = selected;
  else if (mode === "fullpage") text = clean(document.body?.innerText);
  else if (mode === "article") text = clean(main?.innerText);
  return {
    title,
    text: text.slice(0, 900000),
    author,
    url: location.href,
    mode,
    selected: !!selected,
  };
}
