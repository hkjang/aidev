import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { Link } from "react-router-dom";
import { api } from "../api";

const EmbedPath = createContext<string[]>([]);
export function SyncedContent({ documentId, blockId = "", label, renderMarkdown }: { documentId: string; blockId?: string; label?: string; renderMarkdown: (markdown: string) => ReactNode }) {
  const ancestors = useContext(EmbedPath);
  const key = `${documentId}#${blockId}`;
  const circular = ancestors.includes(key) || ancestors.length >= 5;
  const [value, setValue] = useState<{ title: string; markdown: string; version: number } | null>(null);
  const [error, setError] = useState("");
  useEffect(() => {
    let active = true;
    let timer: ReturnType<typeof setTimeout>;
    setValue(null); setError("");
    if (circular) return;
    const refresh = async () => {
      try {
        const path = blockId ? `/documents/${encodeURIComponent(documentId)}/blocks/${encodeURIComponent(blockId)}` : `/documents/${encodeURIComponent(documentId)}`;
        const result = await api(path);
        if (!active) return;
        if (result.deleted_at) throw new Error("원본 문서가 삭제되었습니다.");
        if (result.markdown.length > 100000) { setValue(null); setError("큰 문서는 원본 문서에서 확인하세요 (미리보기 100,000자 한도)."); }
        else { setValue(result); setError(""); }
      } catch (error) {
        // Drop a cached preview on every failed authorization/refresh. Showing a
        // previously allowed body after revocation is not a successful sync.
        if (active) { setValue(null); setError((error as Error).message); }
      } finally { if (active) timer = setTimeout(refresh, 2000); }
    };
    refresh();
    return () => { active = false; clearTimeout(timer); };
  }, [documentId, blockId, circular]);
  if (circular) return <aside className="synced-embed error">순환 참조 또는 최대 5단계 중첩입니다. <Link to={`/app/documents/${documentId}`}>원본 열기</Link></aside>;
  return <aside className="synced-embed">
    <header><Link to={`/app/documents/${documentId}${blockId ? `#^${encodeURIComponent(blockId)}` : ""}`}>{label || value?.title || "동기화된 원본"} ↗</Link><small>원본과 동기화{value ? ` · v${value.version}` : ""}</small></header>
    {error ? <p role="status">{error}</p> : value ? <EmbedPath.Provider value={[...ancestors, key]}>{renderMarkdown(value.markdown)}</EmbedPath.Provider> : <p>권한과 원본을 확인하고 있습니다…</p>}
  </aside>;
}

export function BookmarkCard({ url, title, description }: { url: string; title?: string; description?: string }) {
  let safe = false;
  try { safe = ["https:", "http:"].includes(new URL(url).protocol); } catch {}
  return <aside className="bookmark-card"><strong>{safe ? <a href={url} target="_blank" rel="noopener noreferrer">{title || url} ↗</a> : title || "북마크 주소를 확인하세요"}</strong>{description && <p>{description}</p>}<small>{url}</small></aside>;
}
