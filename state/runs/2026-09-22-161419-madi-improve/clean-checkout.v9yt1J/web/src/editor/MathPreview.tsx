import { useEffect, useRef, useState } from "react";

export default function MathPreview({ source, display = false }: { source: string; display?: boolean }) {
  const target = useRef<HTMLSpanElement>(null);
  const [error, setError] = useState("");
  useEffect(() => {
    let active = true;
    setError("");
    if (source.length > 16000) { setError("수식은 16,000자 이하로 입력하세요."); return; }
    Promise.all([import("katex"), import("katex/dist/katex.min.css")]).then(([module]) => {
      if (!active || !target.current) return;
      try {
        module.default.render(source, target.current, { displayMode: display, trust: false, strict: "error", throwOnError: true, maxExpand: 500, maxSize: 20, output: "htmlAndMathml", macros: {} });
      } catch { setError("수식 문법을 확인하세요. 원문은 그대로 보관됩니다."); }
    }).catch(() => { if (active) setError("수식 표시 모듈을 불러오지 못했습니다."); });
    return () => { active = false; };
  }, [source, display]);
  return <span className={display ? "math-preview display" : "math-preview"}><span ref={target} hidden={!!error} />{error && <span role="status" title={error}><code>{source}</code><small>{error}</small></span>}</span>;
}
