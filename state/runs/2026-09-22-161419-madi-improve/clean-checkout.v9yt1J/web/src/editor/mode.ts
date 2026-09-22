export type EditorMode = "edit" | "source" | "preview";
export function resolveEditorMode(
  requested: string | null,
  preferred: string | undefined,
  sourceLine = 0,
): EditorMode {
  if (requested === "read") return "preview";
  if (requested && ["edit", "source", "preview"].includes(requested))
    return requested as EditorMode;
  // An invalid explicit viewing link must never silently open a write session.
  if (requested) return "preview";
  if (sourceLine > 0) return "source";
  return preferred && ["edit", "source", "preview"].includes(preferred)
    ? (preferred as EditorMode)
    : "edit";
}
