import { useId, type ReactNode } from "react";
import { AlertTriangle, ArrowRight, ShieldCheck } from "lucide-react";
import { Button } from "../ui";
import "./style.css";

export type ReviewChange = {
  label: string;
  before?: ReactNode;
  after?: ReactNode;
  tone?: "normal" | "warning" | "danger";
};
/** Display-only review: callers retain all consent, authorization and CAS checks. */
export function ChangeReview({
  title,
  description,
  changes,
  warnings = [],
  confirmLabel = "변경 적용",
  busy = false,
  disabled = false,
  onConfirm,
  onCancel,
  children,
}: {
  title: string;
  description?: string;
  changes: ReviewChange[];
  warnings?: string[];
  confirmLabel?: string;
  busy?: boolean;
  disabled?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
  children?: ReactNode;
}) {
  const id = useId();
  return (
    <section className="change-review" aria-labelledby={id} aria-busy={busy}>
      <header>
        <ShieldCheck size={22} />
        <div>
          <h3 id={id}>{title}</h3>
          {description && <p>{description}</p>}
        </div>
      </header>
      <ol className="review-changes">
        {changes.map((change, index) => (
          <li
            key={index}
            className={`review-change ${change.tone || "normal"}`}
          >
            <strong>{change.label}</strong>
            <div className="review-comparison">
              <div>
                <span className="review-label">변경 전</span>
                <div>{change.before ?? "없음"}</div>
              </div>
              <ArrowRight size={18} aria-hidden="true" />
              <div>
                <span className="review-label">변경 후</span>
                <div>{change.after ?? "없음"}</div>
              </div>
            </div>
          </li>
        ))}
      </ol>
      {warnings.length > 0 && (
        <div className="review-warnings" role="note">
          <AlertTriangle size={20} />
          <ul>
            {warnings.map((warning, i) => (
              <li key={i}>{warning}</li>
            ))}
          </ul>
        </div>
      )}
      {children}
      <footer>
        <Button type="button" disabled={busy} onClick={onCancel}>
          취소
        </Button>
        <Button
          type="button"
          variant="primary"
          disabled={busy || disabled}
          onClick={onConfirm}
        >
          {busy ? "처리 중…" : confirmLabel}
        </Button>
      </footer>
    </section>
  );
}

export function RecoveryNotice({
  error,
  status,
  dirty = false,
  busy = false,
  onRetry,
  onReview,
  onCopy,
  onReauthenticate,
}: {
  error: unknown;
  status?: number;
  dirty?: boolean;
  busy?: boolean;
  onRetry?: () => void;
  onReview?: () => void;
  onCopy?: () => void;
  onReauthenticate?: () => void;
}) {
  if (!error) return null;
  const code =
    status ??
    (typeof error === "object" && error && "status" in error
      ? Number(error.status)
      : 0);
  const message = error instanceof Error ? error.message : String(error);
  const info =
    code === 409
      ? [
          "다른 변경과 충돌했습니다",
          "서버의 최신 내용과 내 변경을 비교한 뒤 다시 적용하세요. 자동으로 덮어쓰지 않습니다.",
        ]
      : code === 401
        ? [
            "로그인이 필요합니다",
            "세션이 만료되었을 수 있습니다. 변경 내용을 보관한 뒤 다시 로그인하세요.",
          ]
        : code === 403
          ? [
              "현재 접근 권한을 확인하세요",
              "권한이 변경되었을 수 있습니다. 복사본을 보관하고 소유자에게 접근 권한을 요청하세요.",
            ]
          : code === 404 || code === 410
            ? [
                "대상을 더 이상 열 수 없습니다",
                "삭제되었거나 현재 접근할 수 없는 대상입니다. 다른 대상에 자동 적용하지 않습니다.",
              ]
            : [
                "요청을 완료하지 못했습니다",
                "연결 상태와 오류 내용을 확인하세요. 다시 시도하기 전 현재 변경을 보관할 수 있습니다.",
              ];
  return (
    <section className="recovery-notice" role="alert">
      <AlertTriangle size={22} />
      <div>
        <h3>{info[0]}</h3>
        <p>{info[1]}</p>
        <p className="recovery-detail">{message}</p>
        {dirty && <p>아직 서버에 확정되지 않은 변경이 있습니다.</p>}
        <div className="button-row">
          {onCopy && (
            <Button type="button" disabled={busy} onClick={onCopy}>
              내 변경 보관
            </Button>
          )}
          {onReview && (
            <Button type="button" disabled={busy} onClick={onReview}>
              서버 내용과 비교
            </Button>
          )}
          {onReauthenticate && code === 401 && (
            <Button type="button" disabled={busy} onClick={onReauthenticate}>
              다시 로그인
            </Button>
          )}
          {onRetry && ![401, 403, 404, 409, 410].includes(code) && (
            <Button type="button" disabled={busy} onClick={onRetry}>
              다시 시도
            </Button>
          )}
        </div>
      </div>
    </section>
  );
}
