import { useState } from "react";
import type { Editor, JSONContent } from "@tiptap/core";
import { Button, ErrorBox, Field, Modal } from "../ui";

const paragraph = (text: string): JSONContent => ({ type: "paragraph", content: text ? [{ type: "text", text }] : [] });
export function AdvancedToolbar({ editor, documentId }: { editor: Editor; documentId: string }) {
  const [embed, setEmbed] = useState(false), [reference, setReference] = useState(""), [error, setError] = useState("");
  const insert = (node: JSONContent | JSONContent[]) => {
    const selection=editor.state.selection;
    // "Add block" never replaces a selected atom/container. Add after the
    // current top-level block; inline formulas retain their text insertion point.
    if (!Array.isArray(node) && node.type === "inlineMath" && selection.$from.parent.isTextblock && selection.$from.parent.type.name!=="codeBlock") return editor.chain().focus().insertContentAt(selection.to,node).run();
    const at=selection.$to.depth>0?selection.$to.after(1):selection.to;
    if (!Array.isArray(node) && node.type === "inlineMath") node={type:"paragraph",content:[node]};
    return editor.chain().focus().insertContentAt(at,node).run();
  };
  const add = (type: string) => {
    if (!editor.isEditable || !type) return;
    if (type === "embed") { setReference(""); setError(""); setEmbed(true); return; }
    if (type === "callout") insert({ type, attrs: { type: "note", title: "참고" }, content: [paragraph("알아두면 좋은 내용을 입력하세요.")] });
    if (type === "details") insert({ type, content: [{ type: "detailsSummary", content: [{ type: "text", text: "펼쳐서 보기" }] }, { type: "detailsContent", content: [paragraph("접을 수 있는 내용")] }] });
    if (type === "columns2" || type === "columns3") insert({ type: "columns", attrs: { count: Number(type.slice(-1)) }, content: Array.from({ length: Number(type.slice(-1)) }, (_, index) => ({ type: "column", content: [paragraph(`${index + 1}열 내용`)] })) });
    if (type === "inlineMath" || type === "blockMath") insert({ type, attrs: { latex: "E=mc^2" } });
    if (type === "mermaid") insert({ type: "codeBlock", attrs: { language: "mermaid" }, content: [{ type: "text", text: "flowchart LR\n  A[아이디어] --> B[지식]" }] });
    if (type === "bookmark") insert({ type, attrs: { url: "https://example.com", title: "북마크 제목", description: "설명을 직접 입력하세요." } });
    if (type === "footnote") {
      const label = `주-${crypto.randomUUID().slice(0, 8)}`;
      const reference:JSONContent={type:"footnoteReference",attrs:{label}};
      const selection=editor.state.selection;
      if(selection.$from.parent.isTextblock&&selection.$from.parent.type.name!=="codeBlock")editor.chain().focus().insertContentAt(selection.to,reference).run();
      else insert({type:"paragraph",content:[reference]});
      editor.chain().insertContentAt(editor.state.doc.content.size, { type: "footnoteDefinition", attrs: { label }, content: [paragraph("각주 설명")] }).run();
    }
  };
  const insertReference = () => {
    let id = "", block = "";
    const raw = reference.trim();
    const wiki = /^!?\[\[([0-9a-f-]{36})(?:#\^([^\]|\s]+))?(?:\|[^\]]*)?\]\]$/i.exec(raw);
    if (wiki) { id = wiki[1]; block = wiki[2] || ""; }
    else if (/^[0-9a-f-]{36}$/i.test(raw)) id = raw;
    else try { const url = new URL(raw, window.location.origin); id = /\/app\/documents\/([0-9a-f-]{36})/i.exec(url.pathname)?.[1] || ""; block = decodeURIComponent(url.hash.slice(1)).replace(/^\^/, ""); } catch {}
    if (!/^[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}$/i.test(id) || /[\]\[|\s]/.test(block) || block.length > 128) { setError("madi 문서 URL, 블록 URL 또는 동기화 참조를 입력하세요."); return; }
    if (id === documentId && !block) { setError("현재 문서 전체를 자신 안에 넣을 수 없습니다. 특정 블록을 선택하세요."); return; }
    insert({ type: "syncedEmbed", attrs: { documentId: id, blockId: block, label: "" } }); setEmbed(false);
  };
  return <>
    <div className="advanced-insert-menu">
      <select aria-label="고급 블록 추가" value="" disabled={!editor.isEditable} onChange={(event) => add(event.target.value)}>
        <option value="" disabled>고급 블록 추가</option>{[["callout","콜아웃"],["details","접기 / 펼치기"],["columns2","2열 레이아웃"],["columns3","3열 레이아웃"],["inlineMath","인라인 수식"],["blockMath","수식 블록"],["mermaid","Mermaid 다이어그램"],["footnote","각주"],["bookmark","URL 북마크"],["embed","동기화 문서 / 블록"]].map(([value,label])=><option key={value} value={value}>{label}</option>)}
      </select>
      <select aria-label="문단 정렬" value="" disabled={!editor.isEditable} onChange={(event)=>editor.chain().focus().setTextAlign(event.target.value).run()}><option value="" disabled>문단 정렬</option>{[["left","왼쪽"],["center","가운데"],["right","오른쪽"],["justify","양쪽"]].map(([value,label])=><option key={value} value={value}>{label}</option>)}</select>
      {editor.isActive("callout") && <select aria-label="콜아웃 종류" value={editor.getAttributes("callout").type || "note"} onChange={(event)=>editor.chain().focus().updateAttributes("callout",{type:event.target.value}).run()}>{[["note","참고"],["tip","도움말"],["important","중요"],["warning","주의"],["caution","경고"]].map(([value,label])=><option key={value} value={value}>{label}</option>)}</select>}
      {editor.isActive("table") && <><Button disabled={!editor.can().mergeCells()} onClick={()=>editor.chain().focus().mergeCells().run()}>셀 병합</Button><Button disabled={!editor.can().splitCell()} onClick={()=>editor.chain().focus().splitCell().run()}>셀 나누기</Button><Button onClick={()=>editor.chain().focus().addRowAfter().run()}>행 추가</Button><Button onClick={()=>editor.chain().focus().addColumnAfter().run()}>열 추가</Button><Button onClick={()=>editor.chain().focus().deleteRow().run()}>행 삭제</Button><Button onClick={()=>editor.chain().focus().deleteColumn().run()}>열 삭제</Button><Button onClick={()=>editor.chain().focus().deleteTable().run()}>표 삭제</Button></>}
    </div>
    <Modal open={embed} onOpenChange={setEmbed} title="동기화 문서 / 블록" description="원본 변경이 자동 반영됩니다. 현재 사용자에게 원본 읽기 권한이 있어야 표시되며, 수정은 원본 문서에서 합니다.">
      <Field label="원본 문서 또는 블록 링크"><input autoFocus aria-label="동기화 원본 링크" value={reference} onChange={(event)=>setReference(event.target.value)} placeholder="문서 URL 또는 ![[문서 ID#^블록 ID]]" /></Field><ErrorBox error={error} /><Button variant="primary" onClick={insertReference}>동기화 블록 넣기</Button>
    </Modal>
  </>;
}
