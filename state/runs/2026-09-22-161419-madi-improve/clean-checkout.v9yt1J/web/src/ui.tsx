import {
  Children,
  cloneElement,
  isValidElement,
  useId,
  useState,
  type ReactNode,
  type ReactElement,
} from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { copyText } from "./navigation/clipboard";
import {
  AlertCircle,
  Check,
  LoaderCircle,
  X,
  FilePlus2,
  FileText,
} from "lucide-react";
export function DocIcon({ icon, size = 20 }: { icon?: string; size?: number }) {
  return !icon || /^[a-z_-]+$/i.test(icon) ? (
    <FileText size={size} />
  ) : (
    <>{icon}</>
  );
}
export function Button({
  children,
  variant = "",
  className = "",
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & { variant?: string }) {
  return (
    <button className={`button ${variant} ${className}`} {...props}>
      {children}
    </button>
  );
}
export function Modal({
  open,
  onOpenChange,
  title,
  description,
  children,
  wide = false,
  onCloseAutoFocus,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  title: string;
  description?: string;
  children: ReactNode;
  wide?: boolean;
  onCloseAutoFocus?: (event: Event) => void;
}) {
  const descriptionId = useId();
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="modal-overlay" />
        <Dialog.Content
          className={`modal ${wide ? "wide" : ""}`}
          aria-describedby={description ? descriptionId : undefined}
          onCloseAutoFocus={onCloseAutoFocus}
        >
          <div className="modal-heading">
            <Dialog.Title>{title}</Dialog.Title>
            <Dialog.Close className="icon-button" aria-label="닫기">
              <X size={20} />
            </Dialog.Close>
          </div>
          {description && (
            <Dialog.Description id={descriptionId} className="muted">
              {description}
            </Dialog.Description>
          )}
          {children}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
export function Field({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: string;
  children: ReactNode;
}) {
  const id = useId();
  let assigned = false;
  const labelControl = (nodes: ReactNode): ReactNode =>
    Children.map(nodes, (node) => {
      if (!isValidElement(node)) return node;
      const child = node as ReactElement<Record<string, any>>;
      if (
        !assigned &&
        ["input", "select", "textarea"].includes(String(child.type))
      ) {
        assigned = true;
        return cloneElement(child, {
          id,
          "aria-describedby": hint ? id + "-hint" : undefined,
        });
      }
      if (child.props.children)
        return cloneElement(child, {
          children: labelControl(child.props.children),
        });
      return child;
    });
  const grouped =
    isValidElement(children) &&
    String((children.props as any).className || "").includes(
      "checkbox-options",
    );
  if (grouped)
    return (
      <fieldset className="field">
        <legend>{label}</legend>
        {children}
        {hint && <small>{hint}</small>}
      </fieldset>
    );
  const controls = labelControl(children);
  return (
    <div className="field">
      <label htmlFor={id}>{label}</label>
      {controls}
      {hint && <small id={id + "-hint"}>{hint}</small>}
    </div>
  );
}
export function ErrorBox({ error }: { error: unknown }) {
  return error ? (
    <div role="alert" className="notice error">
      <AlertCircle size={18} />
      <span>{error instanceof Error ? error.message : String(error)}</span>
    </div>
  ) : null;
}
export function Empty({
  title = "아직 문서가 없어요",
  text = "첫 문서를 만들고 팀의 지식을 연결해 보세요.",
  action,
}: {
  title?: string;
  text?: string;
  action?: ReactNode;
}) {
  return (
    <div className="empty-state">
      <span className="empty-icon">
        <FilePlus2 size={30} />
      </span>
      <h3>{title}</h3>
      <p>{text}</p>
      {action}
    </div>
  );
}
export function Loading() {
  return (
    <div className="loading">
      <LoaderCircle className="spin" size={24} /> 불러오는 중…
    </div>
  );
}
export function Badge({
  children,
  tone = "",
}: {
  children: ReactNode;
  tone?: string;
}) {
  return <span className={`badge ${tone}`}>{children}</span>;
}
export function PageHeading({
  eyebrow,
  title,
  description,
  actions,
}: {
  eyebrow?: string;
  title: string;
  description?: string;
  actions?: ReactNode;
}) {
  return (
    <div className="page-heading">
      <div>
        {eyebrow && <div className="eyebrow">{eyebrow}</div>}
        <h1>{title}</h1>
        {description && <p>{description}</p>}
      </div>
      {actions && <div className="heading-actions">{actions}</div>}
    </div>
  );
}
export function Toggle({
  checked,
  onChange,
  label,
  description,
}: {
  checked: boolean;
  onChange: (v: boolean) => void;
  label: string;
  description?: string;
}) {
  return (
    <label className="toggle-row">
      <span>
        <strong>{label}</strong>
        {description && <small>{description}</small>}
      </span>
      <input
        type="checkbox"
        className="switch"
        checked={!!checked}
        onChange={(e) => onChange(e.target.checked)}
      />
    </label>
  );
}
export function CopyButton({
  value,
  label = "복사",
}: {
  value: string;
  label?: string;
}) {
  const [copied, setCopied] = useState(false);
  return (
    <Button
      type="button"
      onClick={async () => {
        try {
          await copyText(value);
          setCopied(true);
          setTimeout(() => setCopied(false), 1800);
        } catch {
          window.prompt("아래 내용을 복사하세요.", value);
        }
      }}
    >
      {copied ? (
        <>
          <Check size={16} />
          복사됨
        </>
      ) : (
        label
      )}
    </Button>
  );
}
export const statusNames: Record<string, string> = {
  draft: "초안",
  published: "게시됨",
  review: "검토 중",
  rejected: "반려",
  archived: "보관됨",
  stale: "검토 필요",
};
export const roleNames: Record<string, string> = {
  admin: "관리자",
  owner: "소유자",
  editor: "편집자",
  commenter: "댓글 작성자",
  viewer: "뷰어",
};
