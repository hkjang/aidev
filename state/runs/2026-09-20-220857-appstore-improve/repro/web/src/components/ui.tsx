import {
  AlertCircle,
  Box,
  LoaderCircle,
  SearchX,
  ShieldAlert,
  X,
} from "lucide-react";
import {
  forwardRef,
  useEffect,
  useId,
  useRef,
  useState,
  type ButtonHTMLAttributes,
  type InputHTMLAttributes,
  type PropsWithChildren,
  type ReactNode,
  type SelectHTMLAttributes,
  type TextareaHTMLAttributes,
} from "react";
import { Link, type LinkProps } from "react-router-dom";
import { ApiError } from "../lib/api";
import { appGlyph, appTone, cn, parseList, sameList } from "../lib/utils";
import type { StoreApp } from "../types";

export const Button = forwardRef<
  HTMLButtonElement,
  ButtonHTMLAttributes<HTMLButtonElement> & {
    variant?: "primary" | "secondary" | "ghost" | "danger";
    size?: "sm";
  }
>(function Button(
  { variant = "primary", size, className, type = "button", ...props },
  ref,
) {
  return (
    <button
      ref={ref}
      type={type}
      className={cn(
        "button",
        `button-${variant}`,
        size === "sm" && "button-sm",
        className,
      )}
      {...props}
    />
  );
});

export function ButtonLink({
  variant = "primary",
  size,
  className,
  ...props
}: LinkProps & { variant?: "primary" | "secondary" | "ghost"; size?: "sm" }) {
  return (
    <Link
      className={cn(
        "button",
        `button-${variant}`,
        size === "sm" && "button-sm",
        className,
      )}
      {...props}
    />
  );
}

export function Card({
  className,
  children,
}: PropsWithChildren<{ className?: string }>) {
  return <div className={cn("card", className)}>{children}</div>;
}

export function Badge({
  children,
  tone,
}: PropsWithChildren<{
  tone?: "primary" | "positive" | "warning" | "danger";
}>) {
  return (
    <span className={cn("badge", tone && `badge-${tone}`)}>{children}</span>
  );
}

export function AppIcon({
  app,
  large = false,
}: {
  app: Pick<StoreApp, "name" | "icon" | "iconUrl" | "slug">;
  large?: boolean;
}) {
  return (
    <span
      className={cn("app-icon", large && "large")}
      data-tone={appTone(app.slug || app.name)}
      aria-hidden="true"
    >
      {app.iconUrl ? (
        <img
          alt=""
          src={app.iconUrl}
          width={large ? 80 : 52}
          height={large ? 80 : 52}
        />
      ) : (
        appGlyph(app.name, app.icon)
      )}
    </span>
  );
}

export function PageHeader({
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
    <header className="page-header">
      <div>
        {eyebrow && <p className="eyebrow">{eyebrow}</p>}
        <h1 className="page-title">{title}</h1>
        {description && <p className="page-description">{description}</p>}
      </div>
      {actions && <div className="top-actions">{actions}</div>}
    </header>
  );
}

export function Field({
  label,
  help,
  error,
  children,
  id,
}: {
  label: string;
  help?: string;
  error?: string;
  id?: string;
  children: ReactNode | ((id: string) => ReactNode);
}) {
  const generated = useId();
  const controlId = id ?? generated;
  return (
    <div className="field">
      <label htmlFor={controlId}>{label}</label>
      {typeof children === "function"
        ? (children as (id: string) => ReactNode)(controlId)
        : children}
      {help && <span className="field-help">{help}</span>}
      {error && (
        <span className="field-error" role="alert">
          {error}
        </span>
      )}
    </div>
  );
}

export function Input(props: InputHTMLAttributes<HTMLInputElement>) {
  return <input className={cn("input", props.className)} {...props} />;
}

/**
 * Text field for a comma-separated list.
 *
 * Rendering `value.join(", ")` straight from the parsed array erased the
 * separator as it was typed: the trailing empty segment was filtered out, so
 * the comma vanished on the next render and a second entry could never be
 * started. The raw text lives here and only the parsed array travels upward.
 */
export function ListInput({
  value,
  onChange,
  normalize,
  ...props
}: Omit<InputHTMLAttributes<HTMLInputElement>, "value" | "onChange"> & {
  value: readonly string[];
  onChange: (value: string[]) => void;
  normalize?: (item: string) => string;
}) {
  const parse = (raw: string) =>
    normalize ? parseList(raw).map(normalize) : parseList(raw);
  const [text, setText] = useState(() => value.join(", "));
  // Re-sync only when the incoming list stops matching what the text says, so
  // an outside change (another row, a reloaded setting) is picked up while a
  // half-typed entry is left alone.
  useEffect(() => {
    if (!sameList(parse(text), value)) setText(value.join(", "));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value]);
  return (
    <Input
      {...props}
      value={text}
      onChange={(event) => {
        setText(event.target.value);
        onChange(parse(event.target.value));
      }}
    />
  );
}

export function Select(props: SelectHTMLAttributes<HTMLSelectElement>) {
  return <select className={cn("select", props.className)} {...props} />;
}

export function Textarea(props: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea className={cn("textarea", props.className)} {...props} />;
}

export function Switch({
  checked,
  onChange,
  label,
}: {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label: string;
}) {
  return (
    <label className="switch">
      <input
        type="checkbox"
        checked={checked}
        onChange={(event) => onChange(event.target.checked)}
        aria-label={label}
      />
      <span className="switch-track" aria-hidden="true" />
    </label>
  );
}

export function LoadingState({
  label = "불러오는 중입니다",
}: {
  label?: string;
}) {
  return (
    <StatePanel
      icon={<LoaderCircle className="animate-spin" />}
      title={label}
      description="잠시만 기다려 주세요."
    />
  );
}

export function EmptyState({
  title = "표시할 항목이 없습니다",
  description = "조건을 바꾸거나 새 항목을 등록해 보세요.",
  actions,
}: {
  title?: string;
  description?: string;
  actions?: ReactNode;
}) {
  return (
    <StatePanel
      icon={<SearchX />}
      title={title}
      description={description}
      actions={actions}
    />
  );
}

export function ErrorState({
  error,
  retry,
}: {
  error: unknown;
  retry?: () => void;
}) {
  const apiError = error instanceof ApiError ? error : null;
  if (apiError?.status === 401) return <UnauthorizedState />;
  if (apiError?.status === 403) return <ForbiddenState />;
  const requestId = apiError?.requestId;
  return (
    <StatePanel
      icon={<AlertCircle />}
      title="화면을 불러오지 못했습니다"
      description={`${error instanceof Error ? error.message : "알 수 없는 오류가 발생했습니다."}${requestId ? ` · 요청 ID ${requestId}` : ""}`}
      actions={retry && <Button onClick={retry}>다시 시도</Button>}
    />
  );
}

export function UnauthorizedState({ returnTo }: { returnTo?: string }) {
  const target = returnTo ?? `${location.pathname}${location.search}`;
  return (
    <StatePanel
      icon={<ShieldAlert />}
      title="로그인이 필요합니다"
      description="회사 계정으로 로그인하면 이 화면을 계속 이용할 수 있습니다."
      actions={
        <ButtonLink to={`/login?returnTo=${encodeURIComponent(target)}`}>
          로그인
        </ButtonLink>
      }
    />
  );
}

export function ForbiddenState() {
  return (
    <StatePanel
      icon={<ShieldAlert />}
      title="접근 권한이 없습니다"
      description="필요한 역할 또는 권한이 있는지 관리자에게 확인해 주세요."
      actions={<ButtonLink to="/">스토어로 돌아가기</ButtonLink>}
    />
  );
}

export function StatePanel({
  icon,
  title,
  description,
  actions,
}: {
  icon: ReactNode;
  title: string;
  description: string;
  actions?: ReactNode;
}) {
  return (
    <div className="state-panel" role="status">
      <div>
        <div className="state-icon" aria-hidden="true">
          {icon}
        </div>
        <h2>{title}</h2>
        <p>{description}</p>
        {actions && <div className="state-actions">{actions}</div>}
      </div>
    </div>
  );
}

export function Dialog({
  open,
  title,
  description,
  onClose,
  children,
}: PropsWithChildren<{
  open: boolean;
  title: string;
  description?: string;
  onClose: () => void;
}>) {
  const closeRef = useRef<HTMLButtonElement>(null);
  const dialogRef = useRef<HTMLElement>(null);
  const titleId = useId();
  const descriptionId = useId();
  // Callers pass an inline arrow for onClose, so its identity changes on every
  // parent render. Reading it through a ref keeps the focus effect below keyed
  // on `open` alone: with onClose in the dependency list, every keystroke in a
  // dialog field re-ran the effect and threw focus onto the close button.
  const onCloseRef = useRef(onClose);
  useEffect(() => {
    onCloseRef.current = onClose;
  });
  useEffect(() => {
    if (!open) return;
    const previous = document.activeElement as HTMLElement | null;
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") onCloseRef.current();
      if (event.key !== "Tab") return;
      const focusable = dialogRef.current?.querySelectorAll<HTMLElement>(
        'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
      );
      if (!focusable?.length) return;
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last?.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first?.focus();
      }
    };
    document.addEventListener("keydown", onKey);
    const overflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    requestAnimationFrame(() => closeRef.current?.focus());
    return () => {
      document.removeEventListener("keydown", onKey);
      document.body.style.overflow = overflow;
      previous?.focus();
    };
  }, [open]);
  if (!open) return null;
  return (
    <div
      className="dialog-backdrop"
      onMouseDown={(event) => {
        if (event.currentTarget === event.target) onClose();
      }}
    >
      <section
        ref={dialogRef}
        className="dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={description ? descriptionId : undefined}
      >
        <header className="dialog-header">
          <div>
            <h2 id={titleId}>{title}</h2>
            {description && (
              <p id={descriptionId} className="page-description">
                {description}
              </p>
            )}
          </div>
          <Button
            ref={closeRef}
            variant="ghost"
            className="button-icon"
            onClick={onClose}
            aria-label="닫기"
          >
            <X />
          </Button>
        </header>
        <div className="dialog-body">{children}</div>
      </section>
    </div>
  );
}

export function SkeletonGrid({ count = 6 }: { count?: number }) {
  return (
    <div className="card-grid" aria-label="앱 목록을 불러오는 중">
      {Array.from({ length: count }, (_, index) => (
        <div className="skeleton" key={index} />
      ))}
    </div>
  );
}

export function NotFoundState() {
  return (
    <StatePanel
      icon={<Box />}
      title="페이지를 찾을 수 없습니다"
      description="주소가 변경되었거나 존재하지 않는 화면입니다."
      actions={<ButtonLink to="/">스토어 홈</ButtonLink>}
    />
  );
}
