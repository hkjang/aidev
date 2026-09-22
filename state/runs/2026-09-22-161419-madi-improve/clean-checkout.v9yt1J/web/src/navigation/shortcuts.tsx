import { useEffect, useState } from "react";
import { useApp } from "../context";
import { Button, Field, Modal } from "../ui";

export const shortcutDefinitions = [
  ["palette", "명령 팔레트", "Mod+K"],
  ["quickOpen", "문서 바로 열기", "Mod+P"],
  ["search", "전체 검색", "Mod+Shift+F"],
  ["create", "새 문서", "Mod+N"],
  ["help", "단축키 안내", "Mod+/"],
  ["save", "문서 저장", "Mod+Enter"],
  ["focus", "집중 모드", "Mod+Shift+L"],
  ["ai", "AI 도우미", "Mod+Shift+A"],
] as const;
export type ShortcutAction = (typeof shortcutDefinitions)[number][0];
export function eventChord(
  e: Pick<KeyboardEvent, "key" | "ctrlKey" | "metaKey" | "altKey" | "shiftKey">,
): string {
  let key = e.key.length === 1 ? e.key.toUpperCase() : e.key;
  if (!/^(?:[A-Z0-9/]|Enter)$/.test(key) || !(e.ctrlKey || e.metaKey))
    return "";
  return [
    "Mod",
    ...(e.altKey ? ["Alt"] : []),
    ...(e.shiftKey ? ["Shift"] : []),
    key,
  ].join("+");
}
const reserved = new Set([
  "Mod+W",
  "Mod+R",
  "Mod+T",
  "Mod+L",
  "Mod+Q",
  "Mod+B",
  "Mod+I",
  "Mod+U",
  "Mod+Z",
  "Mod+Y",
  "Mod+Shift+Z",
  "Mod+Shift+W",
  "Mod+Shift+T",
  "Mod+Alt+Q",
]);
export function validChord(value: string): boolean {
  return (
    /^Mod\+(?:Alt\+)?(?:Shift\+)?(?:[A-Z0-9/]|Enter)$/.test(value) &&
    !reserved.has(value)
  );
}
export function shortcuts(
  preferences: Record<string, any>,
): Record<ShortcutAction, string> {
  const result = Object.fromEntries(
    shortcutDefinitions.map(([id, , chord]) => [id, chord]),
  ) as Record<ShortcutAction, string>;
  const raw = preferences.keyboard_shortcuts;
  if (raw && typeof raw === "object") {
    const proposed = { ...result };
    for (const [id] of shortcutDefinitions)
      if (typeof raw[id] === "string" && validChord(raw[id]))
        proposed[id] = raw[id];
    if (new Set(Object.values(proposed)).size === shortcutDefinitions.length)
      return proposed;
  }
  return result;
}
export const displayChord = (value: string) =>
  value
    .replace("Mod", /Mac|iPhone|iPad/.test(navigator.platform) ? "⌘" : "Ctrl")
    .replaceAll("+", " + ");
export function documentCommand(action: string) {
  window.dispatchEvent(
    new CustomEvent("madi-document-command", { detail: { action } }),
  );
}
export function useAppShortcuts(
  actions: Partial<Record<ShortcutAction, () => void>>,
) {
  const { user } = useApp();
  const bindings = shortcuts(user.preferences || {});
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (
        e.isComposing ||
        e.defaultPrevented ||
        e.repeat ||
        e.getModifierState("AltGraph")
      )
        return;
      const chord = eventChord(e);
      const action = (Object.keys(bindings) as ShortcutAction[]).find(
        (id) => bindings[id] === chord,
      );
      if (!action) return;
      if (document.querySelector('[role="dialog"]') && action !== "help")
        return;
      e.preventDefault();
      if (actions[action]) actions[action]?.();
      else if (action === "help")
        window.dispatchEvent(new Event("madi-shortcut-help"));
      else documentCommand(action);
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [JSON.stringify(bindings), actions]);
}
export function ShortcutSettings({
  preferences,
  onChange,
}: {
  preferences: Record<string, any>;
  onChange: (key: string, value: any) => void;
}) {
  const bindings = shortcuts(preferences),
    [error, setError] = useState("");
  return (
    <section className="panel padded">
      <h2>나만의 키보드 단축키</h2>
      <p className="muted">
        입력란을 선택하고 Ctrl/⌘와 원하는 키를 함께 누르세요. 브라우저
        종료·새로고침 등 예약된 조합은 사용할 수 없습니다.
      </p>
      {error && (
        <p role="alert" className="error-box">
          {error}
        </p>
      )}
      <div className="form-grid">
        {shortcutDefinitions.map(([id, label]) => (
          <Field key={id} label={label}>
            <input
              aria-describedby="shortcut-description"
              readOnly
              value={displayChord(bindings[id])}
              onKeyDown={(e) => {
                if (e.key === "Tab") return;
                e.preventDefault();
                const value = eventChord(e);
                if (!validChord(value)) {
                  setError(
                    "Ctrl/⌘ + 문자·숫자·Enter·/ 조합을 사용하세요. 브라우저 예약 단축키는 사용할 수 없습니다.",
                  );
                  return;
                }
                if (
                  Object.entries(bindings).some(
                    ([other, chord]) => other !== id && value === chord,
                  )
                ) {
                  setError("다른 명령이 이미 사용 중인 단축키입니다.");
                  return;
                }
                setError("");
                onChange("keyboard_shortcuts", { ...bindings, [id]: value });
              }}
            />
          </Field>
        ))}
      </div>
      <p id="shortcut-description" className="muted">
        아래 개인 설정 저장 버튼을 눌러 적용합니다. 편집기의 기본
        굵게·기울임·실행 취소 단축키는 유지됩니다.
      </p>
      <Button
        type="button"
        onClick={() => {
          setError("");
          onChange(
            "keyboard_shortcuts",
            Object.fromEntries(
              shortcutDefinitions.map(([id, , chord]) => [id, chord]),
            ),
          );
        }}
      >
        단축키 기본값으로
      </Button>
    </section>
  );
}
export function ShortcutHelp() {
  const { user } = useApp(),
    [open, setOpen] = useState(false);
  const bindings = shortcuts(user.preferences || {});
  useEffect(() => {
    const show = () => setOpen((v) => !v);
    window.addEventListener("madi-shortcut-help", show);
    return () => window.removeEventListener("madi-shortcut-help", show);
  }, []);
  return (
    <Modal
      open={open}
      onOpenChange={setOpen}
      title="키보드로 더 빠르게"
      description="개인 설정에서 단축키를 바꿀 수 있습니다."
    >
      <dl className="shortcut-list">
        {shortcutDefinitions.map(([id, label]) => (
          <div key={id}>
            <dt>{label}</dt>
            <dd>
              <kbd>{displayChord(bindings[id])}</kbd>
            </dd>
          </div>
        ))}
      </dl>
    </Modal>
  );
}
