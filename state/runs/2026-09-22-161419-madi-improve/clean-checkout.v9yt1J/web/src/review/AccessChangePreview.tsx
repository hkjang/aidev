import { useEffect, useRef, useState } from "react";
import { api, type Doc, type DocSummary } from "../api";
import { useApp } from "../context";
import { Button } from "../ui";
import { RecoveryNotice } from "./ChangeReview";
import "./access-change-preview.css";
type Placement = { parent_id: string; space_id: string; visibility: string };
type Preview = {
  document_id: string;
  version: number;
  ticket: string;
  expires_in: number;
  before: Placement;
  after: Placement;
  destination: { parent_name: string; space_name: string };
  potential_expansion: boolean;
  requires_confirmation: boolean;
  warnings: string[];
};
export function useAccessChangeReview(
  doc: Doc | DocSummary | null,
  placement: Partial<Placement>,
  enabled: boolean,
  revision = "",
) {
  const { user, workspace } = useApp(),
    [result, setResult] = useState<{ key: string; data: Preview } | null>(null),
    [error, setError] = useState<{ key: string; error: unknown } | null>(null),
    [consent, setConsent] = useState(""),
    [retry, setRetry] = useState(0);
  const values = {
      parent_id: placement.parent_id ?? doc?.parent_id ?? "",
      space_id: placement.space_id ?? doc?.space_id ?? "",
      visibility: placement.visibility ?? doc?.visibility ?? "private",
    },
    changed =
      !!doc &&
      (values.parent_id !== (doc.parent_id || "") ||
        values.space_id !== (doc.space_id || "") ||
        values.visibility !== doc.visibility);
  const key = JSON.stringify([
      user.id,
      workspace?.id,
      doc?.id,
      doc?.version,
      enabled,
      values,
      revision,
      retry,
    ]),
    current = useRef(key);
  current.current = key;
  useEffect(() => {
    setResult(null);
    setError(null);
    setConsent("");
    if (!enabled || !changed || !doc) return;
    const controller = new AbortController();
    void api<Preview>(
      `/documents/${doc.id}/access-preview`,
      "POST",
      { expected_version: doc.version, ...values },
      { signal: controller.signal },
    )
      .then((data) => {
        if (current.current === key && !controller.signal.aborted)
          setResult({ key, data });
      })
      .catch((error) => {
        if (current.current === key && !controller.signal.aborted)
          setError({ key, error });
      });
    return () => controller.abort();
  }, [key]);
  const preview = result?.key === key ? result.data : null,
    problem = error?.key === key ? error.error : null,
    required = enabled && changed;
  return {
    preview,
    error: problem,
    loading: required && !preview && !problem,
    required,
    checked: consent === key,
    setChecked: (v: boolean) => setConsent(v ? key : ""),
    refresh: () => setRetry((v) => v + 1),
    ready:
      !required ||
      (!!preview && (!preview.requires_confirmation || consent === key)),
    ticket: required ? preview?.ticket : undefined,
  };
}
const scopeName = (value: string) =>
  value === "private"
    ? "나만 보기"
    : value === "selected"
      ? "선택한 사용자"
      : "워크스페이스 멤버";
export function AccessChangePreview({
  review,
  busy = false,
}: {
  review: ReturnType<typeof useAccessChangeReview>;
  busy?: boolean;
}) {
  if (!review.required) return null;
  return (
    <section
      className="access-change-preview"
      aria-label="공유·이동 영향 미리보기"
      aria-busy={review.loading}
    >
      <h3>공유·이동 영향 확인</h3>
      {review.loading ? (
        <p role="status">현재 범위와 목적지 권한을 확인하는 중…</p>
      ) : review.error ? (
        <RecoveryNotice error={review.error} onRetry={review.refresh} />
      ) : review.preview ? (
        <>
          <dl>
            <div>
              <dt>내부 공유</dt>
              <dd>
                {scopeName(review.preview.before.visibility)} →{" "}
                {scopeName(review.preview.after.visibility)}
              </dd>
            </div>
            <div>
              <dt>이동 위치</dt>
              <dd>
                {review.preview.destination.parent_name} ·{" "}
                {review.preview.destination.space_name}
              </dd>
            </div>
            <div>
              <dt>확인 기준</dt>
              <dd>문서 v{review.preview.version} · 현재 공유·상위·공간 정책</dd>
            </div>
          </dl>
          <ul>
            {review.preview.warnings.map((w) => (
              <li key={w}>{w}</li>
            ))}
          </ul>
          {review.preview.requires_confirmation && (
            <label>
              <input
                type="checkbox"
                checked={review.checked}
                disabled={busy}
                onChange={(e) => review.setChecked(e.target.checked)}
              />
              <span>
                {review.preview.potential_expansion
                  ? "열람 범위가 넓어질 수 있음을 확인했습니다."
                  : "공유 범위와 상위 접근 제한의 변경을 확인했습니다."}
              </span>
            </label>
          )}
          <Button disabled={busy} onClick={review.refresh}>
            현재 범위 다시 확인
          </Button>
        </>
      ) : null}
    </section>
  );
}
