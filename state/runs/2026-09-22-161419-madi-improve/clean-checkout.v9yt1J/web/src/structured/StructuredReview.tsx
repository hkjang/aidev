import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { api, datetime } from "../api";
import { Button, Field, Loading, Modal } from "../ui";
import { ChangeReview, RecoveryNotice } from "../review/ChangeReview";
import { useModalReturnFocus } from "../review/useModalReturnFocus";
import {
  propertyLabels,
  type StructuredDraft,
  type StructuredField,
  type StructuredPreview,
  type StructuredProperty,
} from "./types";
type Edit = {
  field: StructuredField;
  included: boolean;
  value: string;
  quote: string;
  start: string;
};
function initialEdit(field: StructuredField, draft: StructuredDraft): Edit {
  return {
    field,
    included: !!draft.properties.find((p) => p.id === field.property_id),
    value:
      typeof field.value === "string"
        ? field.value
        : (JSON.stringify(field.value) ?? ""),
    quote: field.quote,
    start:
      field.start_byte >= draft.start_byte
        ? String(field.start_byte - draft.start_byte)
        : "",
  };
}
function typedValue(p: StructuredProperty | undefined, raw: string): unknown {
  if (!p) throw Error("현재 속성이 없는 항목은 제외하세요.");
  if (["number", "progress"].includes(p.type)) {
    if (!raw.trim() || !Number.isFinite(Number(raw)))
      throw Error(`${p.name}: 유한한 숫자를 입력하세요.`);
    return Number(raw);
  }
  if (p.type === "checkbox") {
    if (raw !== "true" && raw !== "false")
      throw Error(`${p.name}: 체크 값을 명시적으로 선택하세요.`);
    return raw === "true";
  }
  if (["multi_select", "multiselect"].includes(p.type)) {
    let value: unknown;
    try {
      value = JSON.parse(raw);
    } catch {
      throw Error(`${p.name}: 다중 선택 값을 확인하세요.`);
    }
    if (!Array.isArray(value) || !value.every((v) => typeof v === "string"))
      throw Error(`${p.name}: 다중 선택 값을 확인하세요.`);
    return value;
  }
  return raw;
}
export default function StructuredReview({
  id,
  workspaceId,
  onDeleted,
}: {
  id: string;
  workspaceId: string;
  onDeleted: () => void;
}) {
  const [record, setRecord] = useState<StructuredDraft | null>(null),
    [edits, setEdits] = useState<Edit[]>([]),
    [loading, setLoading] = useState(true),
    [busy, setBusy] = useState(false),
    [retry, setRetry] = useState(0),
    [error, setError] = useState<unknown>(null),
    [mutationError, setMutationError] = useState<unknown>(null);
  const [preview, setPreview] = useState<StructuredPreview | null>(null),
    [consent, setConsent] = useState(false),
    [deleting, setDeleting] = useState<number | null>(null);
  const baseline = useRef<StructuredDraft | null>(null),
    latest = useRef<StructuredDraft | null>(null),
    mounted = useRef(false),
    generation = useRef(0),
    sourceFingerprint = useRef(""),
    currentRevision = useRef<number | null>(null);
  const previewFocus = useModalReturnFocus(!!preview, `${workspaceId}:${id}`),
    deleteFocus = useModalReturnFocus(
      deleting === null ? null : String(deleting),
      `${workspaceId}:${id}`,
    );
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  useEffect(() => {
    const leave = (event: BeforeUnloadEvent) => {
      const original = baseline.current;
      if (!original || original.state !== "draft" || !record?.fresh) return;
      const changed =
        JSON.stringify(edits) !==
        JSON.stringify(
          original.fields.map((field) => initialEdit(field, original)),
        );
      if (changed) {
        event.preventDefault();
        event.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", leave);
    return () => window.removeEventListener("beforeunload", leave);
  }, [edits, record]);
  useEffect(() => {
    const abort = new AbortController();
    generation.current++;
    let pending = false,
      blocked = false;
    baseline.current = null;
    latest.current = null;
    currentRevision.current = null;
    sourceFingerprint.current = "";
    setRecord(null);
    setEdits([]);
    setPreview(null);
    setConsent(false);
    setDeleting(null);
    setError(null);
    setMutationError(null);
    setLoading(true);
    const load = async () => {
      if (pending || blocked || abort.signal.aborted) return;
      pending = true;
      try {
        const next = await api<StructuredDraft>(
          `/knowledge/structured-drafts/${id}`,
          "GET",
          undefined,
          { signal: abort.signal },
        );
        if (abort.signal.aborted) return;
        if (next.workspace_id !== workspaceId)
          throw Error("현재 워크스페이스의 개인 초안이 아닙니다.");
        const old = baseline.current;
        const previous = latest.current;
        if (!old) {
          baseline.current = next;
          setEdits(next.fields.map((f) => initialEdit(f, next)));
          sourceFingerprint.current = `${next.source_hash}:${next.current_version}:${next.fresh}`;
        } else if (
          previous &&
          (previous.revision !== next.revision ||
            previous.fresh !== next.fresh ||
            sourceFingerprint.current !==
              `${next.source_hash}:${next.current_version}:${next.fresh}`)
        ) {
          generation.current++;
          setPreview(null);
          setConsent(false);
          setDeleting(null);
        }
        currentRevision.current = next.revision;
        latest.current = next;
        sourceFingerprint.current = `${next.source_hash}:${next.current_version}:${next.fresh}`;
        setRecord(next);
        setError(null);
      } catch (e) {
        if (!abort.signal.aborted) {
          blocked = true;
          generation.current++;
          baseline.current = null;
          latest.current = null;
          currentRevision.current = null;
          setRecord(null);
          setEdits([]);
          setPreview(null);
          setConsent(false);
          setDeleting(null);
          setError(e);
        }
      } finally {
        pending = false;
        if (!abort.signal.aborted) setLoading(false);
      }
    };
    void load();
    const timer = setInterval(() => void load(), 2000);
    return () => {
      generation.current++;
      abort.abort();
      clearInterval(timer);
    };
  }, [id, workspaceId, retry]);
  const change = (index: number, patch: Partial<Edit>) => {
    setEdits((rows) =>
      rows.map((row, n) => (n === index ? { ...row, ...patch } : row)),
    );
    setPreview(null);
    setConsent(false);
  };
  const previewRow = async () => {
    if (!record || !record.fresh || busy || record.state !== "draft") return;
    const revision = record.revision;
    const epoch = generation.current;
    setBusy(true);
    setMutationError(null);
    try {
      const fields = edits
        .filter((e) => e.included)
        .map((e) => {
          const p = record.properties.find((p) => p.id === e.field.property_id);
          if (
            e.start !== "" &&
            (!/^\d+$/.test(e.start) || Number(e.start) > 32768)
          )
            throw Error(
              "인용 위치는 선택 구간 안의 0 이상 정수 바이트 또는 빈 값이어야 합니다.",
            );
          return {
            property_id: e.field.property_id,
            value: typedValue(p, e.value),
            quote: e.quote,
            ...(e.start !== "" ? { start_byte: Number(e.start) } : {}),
          };
        });
      const next = await api<StructuredPreview>(
        `/knowledge/structured-drafts/${id}/preview`,
        "POST",
        { revision, fields },
      );
      if (
        mounted.current &&
        currentRevision.current === revision &&
        generation.current === epoch
      ) {
        setPreview(next);
        setConsent(false);
      }
    } catch (e) {
      if (mounted.current && generation.current === epoch) {
        setPreview(null);
        setConsent(false);
        setMutationError(e);
      }
    } finally {
      if (mounted.current) setBusy(false);
    }
  };
  const apply = async () => {
    if (!preview || !consent || busy || !record) return;
    const ticket = preview.review_ticket,
      revision = record.revision;
    const epoch = generation.current;
    setBusy(true);
    setMutationError(null);
    try {
      await api(`/knowledge/structured-drafts/${id}/apply`, "POST", {
        review_ticket: ticket,
        consent,
      });
      if (
        mounted.current &&
        currentRevision.current === revision &&
        generation.current === epoch
      ) {
        setPreview(null);
        setConsent(false);
        setRetry((v) => v + 1);
      }
    } catch (e) {
      if (mounted.current && generation.current === epoch) {
        setPreview(null);
        setConsent(false);
        setMutationError(e);
      }
    } finally {
      if (mounted.current) setBusy(false);
    }
  };
  const remove = async () => {
    if (deleting === null || busy) return;
    setBusy(true);
    setMutationError(null);
    try {
      await api(`/knowledge/structured-drafts/${id}`, "DELETE", {
        revision: deleting,
        consent: true,
      });
      if (mounted.current) onDeleted();
    } catch (e) {
      if (mounted.current) setMutationError(e);
    } finally {
      if (mounted.current) setBusy(false);
    }
  };
  const valueInput = (edit: Edit, index: number, p?: StructuredProperty) => {
    const label = `${p?.name || edit.field.property_id} 값`,
      disabled = busy || !edit.included || !record?.fresh;
    if (p && ["select", "status"].includes(p.type)) {
      const options = p.options || [];
      return (
        <Field label={label}>
          <select
            disabled={disabled}
            value={edit.value}
            onChange={(e) => change(index, { value: e.target.value })}
          >
            <option value="">값을 선택하세요</option>
            {edit.value && !options.includes(edit.value) && (
              <option value={edit.value}>
                미등록 값: {edit.value} (수정 필요)
              </option>
            )}
            {options.map((value) => (
              <option key={value} value={value}>
                {value}
              </option>
            ))}
          </select>
        </Field>
      );
    }
    if (p?.type === "checkbox")
      return (
        <Field label={label}>
          <select
            disabled={disabled}
            value={edit.value}
            onChange={(e) => change(index, { value: e.target.value })}
          >
            {!["true", "false"].includes(edit.value) && (
              <option value={edit.value}>
                검사 필요: {edit.value || "값 없음"}
              </option>
            )}
            <option value="true">체크함</option>
            <option value="false">체크 안 함</option>
          </select>
        </Field>
      );
    if (p && ["multi_select", "multiselect"].includes(p.type)) {
      let values: string[] = [];
      try {
        const parsed = JSON.parse(edit.value);
        if (Array.isArray(parsed) && parsed.every((v) => typeof v === "string"))
          values = parsed;
      } catch {
        /* Invalid AI value stays visible until an explicit correction. */
      }
      const options = Array.from(new Set([...(p.options || []), ...values]));
      return (
        <fieldset disabled={disabled} className="structured-multiselect">
          <legend>{label}</legend>
          {!Array.isArray(
            (() => {
              try {
                return JSON.parse(edit.value);
              } catch {
                return null;
              }
            })(),
          ) && (
            <p className="notice warning">
              원래 제안: {edit.value || "값 없음"}. 옵션을 명시적으로 선택해
              수정하세요.
            </p>
          )}
          {options.map((value) => (
            <label key={value}>
              <input
                type="checkbox"
                checked={values.includes(value)}
                onChange={(e) =>
                  change(index, {
                    value: JSON.stringify(
                      e.target.checked
                        ? [...values, value]
                        : values.filter((v) => v !== value),
                    ),
                  })
                }
              />
              {value}
              {!p.options?.includes(value) ? " (미등록 · 해제 필요)" : ""}
            </label>
          ))}
        </fieldset>
      );
    }
    return (
      <Field label={label}>
        <input
          disabled={disabled}
          value={edit.value}
          inputMode={
            p && ["number", "progress"].includes(p.type) ? "decimal" : undefined
          }
          onChange={(e) => change(index, { value: e.target.value })}
          placeholder={p?.type === "date" ? "YYYY-MM-DD" : undefined}
        />
      </Field>
    );
  };
  if (loading) return <Loading />;
  return (
    <div className="structured-review">
      <RecoveryNotice
        error={error || mutationError}
        onRetry={() => setRetry((v) => v + 1)}
      />
      {record && (
        <>
          <section className="panel structured-section">
            <div className="structured-section-heading">
              <h2>
                {record.state === "committed"
                  ? "행 생성 완료 · 보관한 근거"
                  : "개인 구조화 초안 검토"}
              </h2>
              <Button
                onClick={() => setDeleting(record.revision)}
                disabled={busy}
              >
                근거 사본 삭제
              </Button>
            </div>
            <p>
              <strong>{record.title}</strong> · 원문 v{record.source_version} →{" "}
              <strong>{record.database_name}</strong>
            </p>
            <p>{record.integrity_notice}</p>
            <p className="muted">
              {record.state === "draft"
                ? `개인 검토 만료 ${datetime(record.expires_at)}`
                : "이미 생성한 행의 권한은 대상 데이터베이스 정책을 따릅니다."}{" "}
              · 초안 개정 {record.revision}
            </p>
            {!record.fresh && (
              <div className="notice warning">
                원문·속성·공유 대상이 바뀌었습니다. 아래 구간은 과거 제안의
                근거이며 새 행으로 반영할 수 없습니다. 현재 원문에서 다시
                생성하세요.
              </div>
            )}
            <div className="button-row">
              <Link
                className="button"
                to={`/app/documents/${record.document_id}?mode=preview`}
              >
                현재 원문 열기
              </Link>
              <Link
                className="button"
                to={`/app/structured-drafts?document_id=${record.document_id}&database_id=${record.database_id}`}
              >
                현재 원문에서 새 제안
              </Link>
              {record.state === "committed" && (
                <Link
                  className="button"
                  to={`/app/databases/${record.database_id}?row=${record.row_id}`}
                >
                  생성한 데이터베이스 행 확인
                </Link>
              )}
            </div>
          </section>
          {record.state === "committed" ? (
            <section className="panel structured-section">
              <h2>사람이 확인해 반영한 값과 근거</h2>
              <div className="structured-proposal-summary">
                {record.reviewed_fields.map((field) => (
                  <article key={field.property_id}>
                    <strong>
                      {record.properties.find((p) => p.id === field.property_id)
                        ?.name || field.property_id}
                    </strong>
                    <pre>{JSON.stringify(field.value)}</pre>
                    <blockquote>{field.quote}</blockquote>
                    <p>
                      {field.human_edited
                        ? "사람이 값·구간을 수정하고 확인"
                        : "AI 제안 값을 사람이 확인"}{" "}
                      · 원문 {field.start_byte}–{field.end_byte}바이트
                    </p>
                  </article>
                ))}
              </div>
            </section>
          ) : (
            <>
              <div className="structured-field-list">
                {edits.map((edit, index) => {
                  const p = record.properties.find(
                    (p) => p.id === edit.field.property_id,
                  );
                  return (
                    <section
                      className="panel structured-section"
                      key={`${edit.field.property_id}:${index}`}
                    >
                      <header>
                        <label className="checkbox-label">
                          <input
                            type="checkbox"
                            checked={edit.included}
                            disabled={busy || !record.fresh || !p}
                            onChange={(e) =>
                              change(index, { included: e.target.checked })
                            }
                          />
                          <strong>
                            {p?.name || edit.field.property_id} 반영
                          </strong>
                          <span className="badge">
                            {p
                              ? propertyLabels[p.type] || p.type
                              : "현재 속성 없음"}
                          </span>
                        </label>
                        {!edit.field.valid && (
                          <p className="notice warning">
                            AI 제안 검사: {edit.field.issue || "확인 필요"}
                          </p>
                        )}
                      </header>
                      {valueInput(edit, index, p)}
                      <Field
                        label={`${p?.name || edit.field.property_id} 근거 인용`}
                      >
                        <textarea
                          rows={3}
                          value={edit.quote}
                          disabled={busy || !edit.included || !record.fresh}
                          onChange={(e) =>
                            change(index, { quote: e.target.value })
                          }
                        />
                      </Field>
                      <Field
                        label={`${p?.name || edit.field.property_id} 인용 시작 바이트`}
                        hint="선택 구간 안의 상대 UTF-8 바이트 위치입니다. 빈 값이면 유일하게 일치하는 구간만 찾습니다."
                      >
                        <input
                          inputMode="numeric"
                          value={edit.start}
                          disabled={busy || !edit.included || !record.fresh}
                          onChange={(e) =>
                            change(index, { start: e.target.value })
                          }
                        />
                      </Field>
                      <details>
                        <summary>생성 당시 값과 근거</summary>
                        <pre>{JSON.stringify(edit.field.value)}</pre>
                        <blockquote>{edit.field.quote}</blockquote>
                        <p className="muted">
                          문서 절대 위치 {edit.field.start_byte}–
                          {edit.field.end_byte}바이트 · SHA-256{" "}
                          {edit.field.content_hash || "검증 실패"}
                        </p>
                      </details>
                    </section>
                  );
                })}
              </div>
              <section className="panel structured-section">
                <p>
                  선택한 항목만 미리 확인합니다. 잘못된 값·인용은 그대로
                  남으므로 수정하거나 항목을 제외하세요. 미리보기만으로 새 행은
                  생성되지 않습니다.
                </p>
                <p className="muted">
                  보관한 AI 제안은 검토함에 유지됩니다. 이 화면에서 직접 수정한
                  값은 새 행을 생성하기 전까지 서버에 저장되지 않으며, 화면을
                  떠나면 사라집니다.
                </p>
                <Button
                  variant="primary"
                  disabled={
                    busy || !record.fresh || !edits.some((e) => e.included)
                  }
                  onClick={previewRow}
                >
                  {busy
                    ? "현재 권한과 구간 검사 중…"
                    : "선택한 값과 공유 대상 미리보기"}
                </Button>
              </section>
            </>
          )}
        </>
      )}
      <Modal
        open={!!preview}
        onOpenChange={(v) => {
          if (!v && !busy) {
            setPreview(null);
            setConsent(false);
          }
        }}
        onCloseAutoFocus={previewFocus}
        title="대상 데이터베이스에 새 행 공유"
        wide
      >
        {preview && (
          <div className="structured-share-review">
            <ChangeReview
              title="원문과 별도로 값을 공유"
              description={preview.destination_notice}
              changes={preview.fields.map((field) => ({
                label:
                  record?.properties.find((p) => p.id === field.property_id)
                    ?.name || field.property_id,
                before: "새 행 · 기존 값 없음",
                after: (
                  <div>
                    <pre>{JSON.stringify(field.value)}</pre>
                    <blockquote>{field.quote}</blockquote>
                    <p>
                      {field.human_edited
                        ? "사람이 수정·확인한 값"
                        : "AI 제안 값을 사람이 확인"}
                    </p>
                  </div>
                ),
              }))}
              warnings={[
                `공유 대상: ${preview.database_name}`,
                "원문이 비공개여도 복사한 값에는 비공개 권한이 상속되지 않습니다.",
                "이미 공유된 사본은 원문 권한 회수로 자동 회수되지 않습니다.",
              ]}
              confirmLabel="확인한 값으로 새 행 생성"
              onConfirm={apply}
              onCancel={() => {
                setPreview(null);
                setConsent(false);
              }}
              busy={busy}
              disabled={!consent}
            >
              <label className="checkbox-label">
                <input
                  type="checkbox"
                  checked={consent}
                  disabled={busy}
                  onChange={(e) => setConsent(e.target.checked)}
                />
                근거와 값을 확인했으며 대상 데이터베이스의 현재 열람자에게
                별도로 공유합니다.
              </label>
            </ChangeReview>
          </div>
        )}
      </Modal>
      <Modal
        open={deleting !== null}
        onOpenChange={(v) => {
          if (!v && !busy) setDeleting(null);
        }}
        onCloseAutoFocus={deleteFocus}
        title="개인 근거 사본 삭제"
      >
        <RecoveryNotice error={mutationError} />
        <p>
          개인 초안·근거 사본을 삭제합니다. 원문과 이미 생성한 데이터베이스 행은
          삭제되지 않습니다. 보존 정책이 적용되면 삭제할 수 없습니다.
        </p>
        <div className="modal-footer">
          <Button onClick={() => setDeleting(null)} disabled={busy}>
            취소
          </Button>
          <Button variant="danger" onClick={remove} disabled={busy}>
            근거 사본 삭제 확인
          </Button>
        </div>
      </Modal>
    </div>
  );
}
