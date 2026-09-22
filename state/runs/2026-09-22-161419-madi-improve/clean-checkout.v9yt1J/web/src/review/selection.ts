/** Character offsets stay local. Only checked UTF-8 byte offsets go to the API. */
export type MarkdownSelection = { start: number; end: number; text: string };
export function selectedMarkdown(
  markdown: string,
  source: HTMLTextAreaElement | null,
): MarkdownSelection | null {
  if (
    source &&
    source.selectionEnd > source.selectionStart &&
    document.activeElement === source
  ) {
    const start = source.selectionStart,
      end = source.selectionEnd;
    return { start, end, text: markdown.slice(start, end) };
  }
  const selection = window.getSelection();
  if (!selection || selection.isCollapsed || !selection.rangeCount) return null;
  const range = selection.getRangeAt(0),
    root = document.querySelector(".editor-area");
  if (!root?.contains(range.commonAncestorContainer)) return null;
  const fragment = range.cloneContents();
  fragment
    .querySelectorAll(
      ".collaboration-carets__label,.collaboration-carets__caret,.collaboration-cursor__label,[contenteditable=false]",
    )
    .forEach((node) => node.remove());
  const text = fragment.textContent || "",
    start = markdown.indexOf(text);
  // Rendered formatting can make source offsets ambiguous. Never guess a range.
  if (!text.trim() || start < 0 || markdown.indexOf(text, start + 1) !== -1)
    return null;
  return { start, end: start + text.length, text };
}
export function selectionBytes(markdown: string, selection: MarkdownSelection) {
  const encoder = new TextEncoder();
  return {
    start_byte: encoder.encode(markdown.slice(0, selection.start)).length,
    end_byte: encoder.encode(markdown.slice(0, selection.end)).length,
  };
}
