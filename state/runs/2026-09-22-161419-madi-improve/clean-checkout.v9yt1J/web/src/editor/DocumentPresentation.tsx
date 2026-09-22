import { useMemo, useRef } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { ArrowLeft, ArrowRight, Maximize, X } from "lucide-react";
import { MarkdownContent } from "./MarkdownContent";
import { Button } from "../ui";
import type { DocSummary } from "../api";
import "./organizer.css";
export function presentationSlides(source: string): string[] {
  const lines = source
    .replace(/^---\r?\n[\s\S]*?\r?\n---(?:\r?\n|$)/, "")
    .split("\n");
  const slides: string[] = [];
  let current: string[] = [],
    fence = "",
    length = 0;
  for (const line of lines) {
    const match = line.match(/^\s{0,3}(`{3,}|~{3,})/);
    if (match) {
      if (!fence) {
        fence = match[1][0];
        length = match[1].length;
      } else if (match[1][0] === fence && match[1].length >= length) fence = "";
    }
    if (!fence && /^#{1,2}\s/.test(line) && current.join("\n").trim()) {
      slides.push(current.join("\n"));
      current = [];
    }
    current.push(line);
  }
  if (current.join("\n").trim()) slides.push(current.join("\n"));
  return slides.length ? slides : ["내용이 없는 문서입니다."];
}
export default function DocumentPresentation({
  title,
  source,
  documents,
  slide,
  onSlide,
  close,
}: {
  title: string;
  source: string;
  documents: DocSummary[];
  slide: number;
  onSlide: (value: number) => void;
  close: () => void;
}) {
  const slides = useMemo(() => presentationSlides(source), [source]),
    index = Math.min(
      slides.length - 1,
      Math.max(0, Number.isFinite(slide) ? slide : 0),
    ),
    ref = useRef<HTMLDivElement>(null);
  return (
    <Dialog.Root
      open
      onOpenChange={(value) => {
        if (!value) close();
      }}
    >
      <Dialog.Portal>
        <Dialog.Content
          ref={ref}
          className="presentation-view"
          aria-describedby={undefined}
          onKeyDown={(e) => {
            if (e.key === "ArrowRight" || e.key === "PageDown") {
              e.preventDefault();
              onSlide(Math.min(slides.length - 1, index + 1));
            } else if (e.key === "ArrowLeft" || e.key === "PageUp") {
              e.preventDefault();
              onSlide(Math.max(0, index - 1));
            }
          }}
        >
          <header className="presentation-toolbar">
            <Dialog.Title asChild>
              <strong>{title}</strong>
            </Dialog.Title>
            <span className="presentation-slide-count">
              {index + 1} / {slides.length}
            </span>
            <Button
              aria-label="발표 전체 화면"
              onClick={() => {
                void ref.current?.requestFullscreen?.().catch(() => {});
              }}
            >
              <Maximize size={18} />
            </Button>
            <Dialog.Close asChild>
              <Button aria-label="발표 종료">
                <X size={18} />
              </Button>
            </Dialog.Close>
          </header>
          <article
            key={index}
            className="presentation-slide"
            aria-label={`슬라이드 ${index + 1}`}
          >
            <MarkdownContent markdown={slides[index]} documents={documents} />
          </article>
          <footer className="presentation-nav">
            <Button disabled={!index} onClick={() => onSlide(index - 1)}>
              <ArrowLeft size={18} />
              이전
            </Button>
            <span className="muted">← → 이동 · Esc 종료</span>
            <Button
              disabled={index === slides.length - 1}
              onClick={() => onSlide(index + 1)}
            >
              다음
              <ArrowRight size={18} />
            </Button>
          </footer>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
