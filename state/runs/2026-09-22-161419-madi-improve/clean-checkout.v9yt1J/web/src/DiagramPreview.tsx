import { useEffect, useRef, useState } from "react";
import { AlertCircle, LoaderCircle } from "lucide-react";
import DOMPurify from "dompurify";

// Mermaid maintains global configuration and a shared rendering document. Keep
// render jobs serialized; never bind callbacks from untrusted diagram source.
let diagramQueue: Promise<unknown> = Promise.resolve();
let diagramSequence = 0;
const externalCSS = (value: string) =>
  /@import/i.test(value) ||
  [...value.matchAll(/url\s*\(([^)]*)\)/gi)].some(
    (match) =>
      !match[1]
        .trim()
        .replace(/^['"]|['"]$/g, "")
        .startsWith("#"),
  );
export function diagramSourceError(source: string): string {
  if (source.length > 50000) return "다이어그램은 50,000자 이하로 작성하세요.";
  if (
    source.split("\n").length > 1000 ||
    (source.match(/(?:-->|---|==>|\.\.>|->>)/g) || []).length > 500
  )
    return "다이어그램은 1,000줄과 연결선 500개 이하로 작성하세요.";
  if (/^\s*---/.test(source) || /%%\s*\{/.test(source))
    return "다이어그램 안에서 실행 설정을 변경할 수 없습니다.";
  for (const line of source
    .split(/[\n;]/)
    .filter((line) => /^\s*(?:classDef|linkStyle|style)\b/i.test(line))) {
    const rule = line.match(
      /^\s*(?:classDef|linkStyle|style)\s+[\p{L}\p{N}_,\-]+\s+(.+)$/iu,
    );
    if (
      !rule ||
      !rule[1]
        .split(",")
        .every((property) =>
          /^\s*(?:fill|stroke|color|background|font-size|font-weight|stroke-width|stroke-dasharray|opacity)\s*:\s*(?:#[0-9a-f]{3,8}|[a-z]{1,20}|[0-9.\s%\-]{1,30})\s*$/i.test(
            property,
          ),
        )
    )
      return "다이어그램 스타일은 안전한 색상과 숫자 속성만 사용할 수 있습니다.";
  }
  if (/Update\w*Style\s*\(/i.test(source))
    return "이 다이어그램의 사용자 스타일 함수는 사용할 수 없습니다.";
  if (
    /(?:^|[\n;])\s*(?:click|href)\s|(?:https?|javascript|data|vbscript):|@import|url\s*\(|<\/?[a-z][^>]*>|\b(?:img|image)\s*:|\\u[0-9a-f]{4}/i.test(
      source,
    )
  )
    return "외부 링크, 이미지, HTML과 실행 명령은 다이어그램에서 사용할 수 없습니다.";
  return "";
}

export default function DiagramPreview({
  source,
  className = "",
  svgBounds,
}: {
  source: string;
  className?: string;
  svgBounds?: { x: number; y: number; width: number; height: number };
}) {
  const [svg, setSVG] = useState(""),
    [error, setError] = useState(""),
    [loading, setLoading] = useState(false);
  const generation = useRef(0);
  useEffect(() => {
    const token = ++generation.current;
    setSVG("");
    setError("");
    setLoading(!!source.trim());
    if (!source.trim()) return;
    const invalid = diagramSourceError(source);
    if (invalid) {
      setError(invalid);
      setLoading(false);
      return;
    }
    const render = async () => {
      if (token !== generation.current) return;
      const id = `madi-diagram-${++diagramSequence}`;
      const host = document.createElement("div");
      host.style.cssText =
        "position:fixed;left:-100000px;top:0;width:1200px;pointer-events:none;visibility:hidden";
      host.setAttribute("aria-hidden", "true");
      host.className = "madi-diagram-render-host";
      // Reduced-motion's global transition-duration otherwise animates Mermaid's
      // SVG transforms while it measures getBBox, producing a clipped viewBox.
      const measurementStyle = document.createElement("style");
      measurementStyle.textContent =
        ".madi-diagram-render-host * { transition: none !important; animation: none !important; }";
      document.head.appendChild(measurementStyle);
      document.body.appendChild(host);
      try {
        const { default: mermaid } = await import("mermaid");
        if (token !== generation.current) return;
        await document.fonts.ready;
        mermaid.initialize({
          startOnLoad: false,
          securityLevel: "strict",
          htmlLabels: false,
          maxTextSize: 50000,
          maxEdges: 500,
          suppressErrorRendering: true,
          fontFamily: "Noto Sans KR Variable, sans-serif",
          flowchart: { htmlLabels: false },
          secure: [
            "secure",
            "securityLevel",
            "startOnLoad",
            "maxTextSize",
            "maxEdges",
            "htmlLabels",
            "flowchart",
            "fontFamily",
          ],
        });
        const result = await mermaid.render(id, source, host);
        const cleaned = DOMPurify.sanitize(result.svg, {
          USE_PROFILES: { svg: true, svgFilters: true },
          FORBID_TAGS: [
            "foreignObject",
            "script",
            "a",
            "image",
            "iframe",
            "object",
            "embed",
          ],
          FORBID_ATTR: ["onload", "onclick", "onerror"],
        });
        const documentSVG = new DOMParser().parseFromString(
          cleaned,
          "image/svg+xml",
        );
        for (const element of documentSVG.querySelectorAll("*")) {
          for (const attribute of [...element.attributes]) {
            const value = attribute.value;
            if (
              /^on/i.test(attribute.name) ||
              (/(?:href|src)$/i.test(attribute.name) &&
                !value.startsWith("#")) ||
              externalCSS(value)
            )
              element.removeAttribute(attribute.name);
          }
          if (
            element.tagName.toLowerCase() === "style" &&
            externalCSS(element.textContent || "")
          )
            element.remove();
        }
        // Mermaid emits percentage dimensions for inline SVG. Give the isolated
        // image a real intrinsic aspect ratio so small canvas cards never crop it.
        const root = documentSVG.documentElement;
        const bounds = (root.getAttribute("viewBox") || "")
          .trim()
          .split(/[\s,]+/)
          .map(Number);
        if (
          bounds.length === 4 &&
          bounds.every(Number.isFinite) &&
          bounds[2] > 0 &&
          bounds[3] > 0 &&
          bounds[2] <= 1000000 &&
          bounds[3] <= 1000000
        ) {
          root.setAttribute("width", String(bounds[2]));
          root.setAttribute("height", String(bounds[3]));
          root.removeAttribute("style");
          root.setAttribute("preserveAspectRatio", "xMidYMid meet");
        }
        if (token === generation.current)
          setSVG(
            new XMLSerializer().serializeToString(documentSVG.documentElement),
          );
      } catch {
        if (token === generation.current)
          setError(
            "다이어그램을 표시할 수 없습니다. Mermaid 문법과 노드·연결 수를 확인하세요.",
          );
      } finally {
        host.remove();
        measurementStyle.remove();
        if (token === generation.current) setLoading(false);
      }
    };
    diagramQueue = diagramQueue.catch(() => {}).then(render);
    return () => {
      generation.current++;
    };
  }, [source]);
  if (svgBounds) {
    if (svg && !error && !loading)
      return (
        <image
          {...svgBounds}
          href={`data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`}
          preserveAspectRatio="xMidYMid meet"
          role="img"
          aria-label="Mermaid 다이어그램"
        />
      );
    return (
      <text
        x={svgBounds.x + 5}
        y={svgBounds.y + 24}
        fontSize={14}
        fill={error ? "#a6403c" : "#6b806c"}
      >
        {loading
          ? "다이어그램 그리는 중…"
          : error
            ? "다이어그램 오류 · 두 번 눌러 원문 확인"
            : "Mermaid 소스를 입력하세요."}
      </text>
    );
  }
  return (
    <div
      className={`diagram-preview ${className}`}
      style={{ width: "100%", overflow: "auto" }}
    >
      {loading ? (
        <div className="muted" role="status">
          <LoaderCircle size={18} className="spin" /> 다이어그램 그리는 중…
        </div>
      ) : error ? (
        <div className="notice error" role="alert">
          <AlertCircle size={18} />
          {error}
        </div>
      ) : svg ? (
        <img
          alt="Mermaid 다이어그램"
          src={`data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`}
          style={{ maxWidth: "100%", height: "auto" }}
        />
      ) : (
        <p className="muted">Mermaid 소스를 입력하세요.</p>
      )}
    </div>
  );
}
