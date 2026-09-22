import { useState } from "react";
import { NodeViewContent, NodeViewWrapper, ReactNodeViewRenderer, type NodeViewProps } from "@tiptap/react";
import CodeBlock from "@tiptap/extension-code-block";
import DiagramPreview from "../DiagramPreview";
import PluginBlockPreview from "../PluginBlockPreview";
import MathPreview from "./MathPreview";
import { SyncedContent, BookmarkCard } from "./SyncedContent";
import { InlineMath, BlockMath, SyncedEmbed, Bookmark } from "./nodes";
import { MarkdownContent } from "./MarkdownContent";

function CodeView({ node, updateAttributes, editor }: NodeViewProps) {
  const languages = [...new Set(["text", "mermaid", "go", "javascript", "typescript", "python", "sql", "yaml", "json", "shell", "html", "css", node.attrs.language || "text"])];
  return <NodeViewWrapper className="advanced-code-block">
    <div contentEditable={false} className="code-language"><label>코드 언어 <select aria-label="코드 블록 언어" value={node.attrs.language || "text"} disabled={!editor.isEditable} onChange={(event) => updateAttributes({ language: event.target.value })}>{languages.map((language) => <option value={language} key={language}>{language === "text" ? "일반 텍스트" : language}</option>)}</select></label></div>
    <pre><NodeViewContent<"code"> as="code" /></pre>
    {node.attrs.language === "mermaid" && <div contentEditable={false}><DiagramPreview source={node.textContent} /></div>}
    {node.attrs.language === "madi-plugin" && <div contentEditable={false}><PluginBlockPreview source={node.textContent} /></div>}
  </NodeViewWrapper>;
}
function MathView({ node, updateAttributes, editor }: NodeViewProps) {
  const [editing, setEditing] = useState(false);
  const block = node.type.name === "blockMath";
  return <NodeViewWrapper as={block ? "div" : "span"} className="editor-math" contentEditable={false}>
    <span onDoubleClick={() => editor.isEditable && setEditing(true)}><MathPreview source={node.attrs.latex} display={block} /></span>
    {editor.isEditable && <button aria-label="수식 편집" className="text-button" onClick={() => setEditing(!editing)}>{editing ? "완료" : "수식"}</button>}
    {editing && <textarea aria-label="LaTeX 수식 원문" value={node.attrs.latex} onChange={(event) => updateAttributes({ latex: event.target.value })} spellCheck={false} />}
  </NodeViewWrapper>;
}
function EmbedView({ node }: NodeViewProps) {
  return <NodeViewWrapper contentEditable={false}><SyncedContent documentId={node.attrs.documentId} blockId={node.attrs.blockId} label={node.attrs.label} renderMarkdown={(markdown) => <MarkdownContent markdown={markdown} />} /></NodeViewWrapper>;
}
function BookmarkView({ node, updateAttributes, editor }: NodeViewProps) {
  const [editing, setEditing] = useState(!node.attrs.url);
  const [draft,setDraft] = useState<Record<string,string>>({url:node.attrs.url,title:node.attrs.title,description:node.attrs.description});
  const [error,setError] = useState("");
  const toggle = () => {
    if(!editing){setDraft({url:node.attrs.url,title:node.attrs.title,description:node.attrs.description});setEditing(true);return;}
    try { if(!["https:","http:"].includes(new URL(draft.url).protocol))throw new Error(); } catch {setError("http 또는 https URL을 입력하세요.");return;}
    updateAttributes(draft);setError("");setEditing(false);
  };
  return <NodeViewWrapper contentEditable={false}><BookmarkCard url={node.attrs.url} title={node.attrs.title} description={node.attrs.description} />
    {editor.isEditable && <button className="text-button" onClick={toggle}>{editing ? "북마크 편집 완료" : "북마크 편집"}</button>}
    {editing && editor.isEditable && <div className="bookmark-fields">{[["url", "북마크 URL"], ["title", "북마크 제목"], ["description", "북마크 설명"]].map(([key, label]) => <label key={key}>{label}<input aria-label={label} value={draft[key]} maxLength={8000} onChange={(event) => setDraft({...draft,[key]:event.target.value})} /></label>)}
    {error&&<p role="alert">{error}</p>}
    <small>입력한 주소와 설명만 저장합니다. 외부 사이트를 자동으로 요청하지 않습니다.</small></div>}
  </NodeViewWrapper>;
}

export const VisualCodeBlock = CodeBlock.extend({ addNodeView() { return ReactNodeViewRenderer(CodeView); } });
export const VisualInlineMath = InlineMath.extend({ addNodeView() { return ReactNodeViewRenderer(MathView); } });
export const VisualBlockMath = BlockMath.extend({ addNodeView() { return ReactNodeViewRenderer(MathView); } });
export const VisualSyncedEmbed = SyncedEmbed.extend({ addNodeView() { return ReactNodeViewRenderer(EmbedView); } });
export const VisualBookmark = Bookmark.extend({ addNodeView() { return ReactNodeViewRenderer(BookmarkView); } });
