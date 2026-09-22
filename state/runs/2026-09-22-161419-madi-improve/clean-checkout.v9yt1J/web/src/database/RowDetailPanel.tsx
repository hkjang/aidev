import * as Dialog from "@radix-ui/react-dialog";
import { useRef, type ReactNode } from "react";
import { X } from "lucide-react";
import "./editing.css";
export default function RowDetailPanel({
  open,
  onOpenChange,
  title,
  children,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  children: ReactNode;
}) {
  const opener = useRef<HTMLElement | null>(null),
    previous = useRef(false),
    rowID = useRef("");
  if (open && !previous.current) {
    opener.current = document.activeElement as HTMLElement;
    rowID.current = opener.current?.dataset.rowDetail || "";
  }
  previous.current = open;
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="database-row-overlay" />
        <Dialog.Content
          className="database-row-panel"
          aria-describedby={undefined}
          onCloseAutoFocus={(event) => {
            event.preventDefault();
            const element = opener.current;
            const replacement = rowID.current
              ? document.querySelector<HTMLElement>(
                  `[data-row-detail="${CSS.escape(rowID.current)}"]`,
                )
              : null;
            if (replacement) replacement.focus();
            else if (element?.isConnected) element.focus();
            else
              document
                .querySelector<HTMLElement>(".database-edit-grid")
                ?.focus();
          }}
        >
          <header>
            <Dialog.Title>{title}</Dialog.Title>
            <Dialog.Close asChild>
              <button className="icon-button" aria-label="항목 상세 닫기">
                <X size={20} />
              </button>
            </Dialog.Close>
          </header>
          {children}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
