import { useId, useMemo, useRef, useState } from "react";
import { X, Tag } from "lucide-react";
import "./tag-chips.css";
export function TagChips({
  value,
  onChange,
  suggestions = [],
  readOnly = false,
}: {
  value: string;
  onChange: (value: string) => void;
  suggestions?: string[];
  readOnly?: boolean;
}) {
  const id = useId(),
    root = useRef<HTMLDivElement>(null),
    composing = useRef(false);
  const [draft, setDraft] = useState(""),
    [open, setOpen] = useState(false),
    [active, setActive] = useState(0),
    [highlighted, setHighlighted] = useState(false),
    [notice, setNotice] = useState("");
  const tags = value
    .split(",")
    .map((t) => t.trim())
    .filter(Boolean);
  const options = useMemo(
    () =>
      [...new Set(suggestions)]
        .filter(
          (tag) =>
            !tags.includes(tag) &&
            tag.toLocaleLowerCase().includes(draft.trim().toLocaleLowerCase()),
        )
        .slice(0, 6),
    [value, suggestions, draft],
  );
  const commit = (input = draft) => {
    if (readOnly || composing.current) return;
    const additions = input
      .split(",")
      .map((v) => v.trim().replace(/^#/, ""))
      .filter(Boolean);
    if (!additions.length) {
      setDraft("");
      return;
    }
    const next = [...new Set([...tags, ...additions])];
    if (next.length > Math.max(100, tags.length)) {
      setNotice("태그는 한 문서에 100개까지 추가할 수 있습니다.");
      return;
    }
    onChange(next.join(", "));
    setDraft("");
    setOpen(false);
    setActive(0);
    setNotice("");
  };
  return (
    <div
      className="tag-chips"
      ref={root}
      onBlur={(e) => {
        if (!root.current?.contains(e.relatedTarget as Node)) {
          commit();
          setOpen(false);
        }
      }}
    >
      <Tag size={17} aria-hidden="true" />
      <div className="tag-values">
        {tags.map((tag, index) => (
          <span className="tag-chip" key={`${tag}-${index}`}>
            <span title={tag}>{tag}</span>
            {!readOnly && (
              <button
                type="button"
                aria-label={`${tag} 태그 삭제`}
                onClick={() =>
                  onChange(tags.filter((_, i) => i !== index).join(", "))
                }
              >
                <X size={14} />
              </button>
            )}
          </span>
        ))}
        {!readOnly && (
          <div className="tag-input-wrap">
            <input
              aria-label="문서 태그"
              role="combobox"
              aria-expanded={open && options.length > 0}
              aria-controls={`${id}-options`}
              aria-autocomplete="list"
              aria-activedescendant={
                open && options[active] ? `${id}-${active}` : undefined
              }
              placeholder={tags.length ? "태그 추가" : "태그를 입력하고 Enter"}
              value={draft}
              maxLength={200}
              onFocus={() => setOpen(true)}
              onChange={(e) => {
                setDraft(e.target.value);
                setOpen(true);
                setActive(0);
                setHighlighted(false);
              }}
              onCompositionStart={() => {
                composing.current = true;
              }}
              onCompositionEnd={() => {
                composing.current = false;
              }}
              onKeyDown={(e) => {
                if (
                  e.nativeEvent.isComposing ||
                  composing.current ||
                  e.keyCode === 229
                )
                  return;
                if (e.key === "Enter" || e.key === ",") {
                  e.preventDefault();
                  commit(
                    highlighted && open && options[active]
                      ? options[active]
                      : draft,
                  );
                } else if (e.key === "ArrowDown" && options.length) {
                  e.preventDefault();
                  setOpen(true);
                  setActive(highlighted ? (active + 1) % options.length : 0);
                  setHighlighted(true);
                } else if (e.key === "ArrowUp" && options.length) {
                  e.preventDefault();
                  setActive((active - 1 + options.length) % options.length);
                  setHighlighted(true);
                } else if (
                  e.key === "Tab" &&
                  open &&
                  draft &&
                  options[active]
                ) {
                  e.preventDefault();
                  commit(options[active]);
                } else if (e.key === "Escape") {
                  setOpen(false);
                }
              }}
            />
            {open && options.length > 0 && (
              <div
                className="tag-options"
                id={`${id}-options`}
                role="listbox"
                aria-label="사용 중인 태그"
              >
                {options.map((option, index) => (
                  <button
                    type="button"
                    role="option"
                    id={`${id}-${index}`}
                    key={option}
                    aria-selected={active === index}
                    onMouseDown={(e) => e.preventDefault()}
                    onClick={() => commit(option)}
                  >
                    {option}
                  </button>
                ))}
              </div>
            )}
          </div>
        )}
        {readOnly && !tags.length && <span className="muted">태그 없음</span>}
      </div>
      {notice && (
        <span className="tag-notice" role="status">
          {notice}
        </span>
      )}
    </div>
  );
}
