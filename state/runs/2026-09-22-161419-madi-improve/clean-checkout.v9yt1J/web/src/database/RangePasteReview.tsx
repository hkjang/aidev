import { useEffect, useRef, useState } from "react";
import type { Property, Row } from "../api";
import { api, ApiError } from "../api";
import { useApp } from "../context";
import { Button, Modal } from "../ui";
import { ChangeReview, RecoveryNotice } from "../review/ChangeReview";
import { parseCellText } from "./cells";
import { plainValue } from "../DatabaseAdvanced";
export type RangeTarget = { row: Row; property: Property; text: string };
type Check = {
  valid: boolean;
  committed: boolean;
  errors: { row_id: string; property_id: string; message: string }[];
  changes?: unknown[];
};
export default function RangePasteReview({
  databaseId,
  schema,
  targets,
  onClose,
  onSaved,
}: {
  databaseId: string;
  schema: string;
  targets: RangeTarget[];
  onClose: () => void;
  onSaved: () => Promise<void>;
}) {
  const { user, workspace } = useApp(),
    origin = useRef(`${user.id}:${workspace?.id}:${databaseId}`),
    alive = useRef(true);
  const [texts, setTexts] = useState(targets.map((t) => t.text)),
    [check, setCheck] = useState<Check | null>(null),
    [busy, setBusy] = useState(false),
    [error, setError] = useState<unknown>(null),
    [consent, setConsent] = useState(false);
  const values = texts.map((text, i) =>
    parseCellText(targets[i].property, text),
  );
  useEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
    };
  }, []);
  const scope = `${user.id}:${workspace?.id}:${databaseId}`,
    scopeRef = useRef(scope);
  scopeRef.current = scope;
  const current = () => alive.current && scopeRef.current === origin.current;
  const request = async (commit: boolean) => {
    if (
      busy ||
      values.some((v) => v.error) ||
      (commit && (!check?.valid || !consent))
    )
      return;
    setBusy(true);
    setError(null);
    try {
      const result = await api<Check>(
        `/databases/${databaseId}/edit-cells`,
        "POST",
        {
          schema_fingerprint: schema,
          cells: targets.map((t, i) => ({
            row_id: t.row.id,
            property_id: t.property.id,
            expected_version: t.row.version,
            value: values[i].value,
          })),
          commit,
          consent: commit && consent,
        },
      );
      if (!current()) return;
      setCheck(result);
      if (result.committed) {
        await onSaved();
        if (current()) onClose();
      }
    } catch (e) {
      if (current()) setError(e);
    } finally {
      if (current()) setBusy(false);
    }
  };
  if (scope !== origin.current) return null;
  return (
    <Modal
      open
      onOpenChange={(open) => {
        if (!open && !busy) onClose();
      }}
      title="범위 붙여넣기 검토"
      description="기존 행에만 붙여넣습니다. 행·속성 기준과 모든 셀 타입을 다시 확인하고, 하나라도 오류가 있으면 전체를 저장하지 않습니다."
      wide
    >
      <div className="range-review">
        <p>
          {new Set(targets.map((t) => t.row.id)).size}행 · {targets.length}셀 ·
          입력값은 아직 저장하지 않았습니다.
        </p>
        <div className="table-scroll">
          <table className="data-table">
            <thead>
              <tr>
                <th>행 / 속성</th>
                <th>현재 값</th>
                <th>붙여넣을 값</th>
                <th>검사</th>
              </tr>
            </thead>
            <tbody>
              {targets.map((t, i) => (
                <tr key={`${t.row.id}:${t.property.id}`}>
                  <th>
                    {Math.floor(
                      i / new Set(targets.map((t) => t.property.id)).size,
                    ) + 1}
                    행 · {t.property.name}
                  </th>
                  <td>
                    {plainValue(t.row.values[t.property.id]) || "비어 있음"}
                  </td>
                  <td>
                    <input
                      aria-label={`${i + 1}번 셀 ${t.property.name}`}
                      value={texts[i]}
                      disabled={busy}
                      onChange={(e) => {
                        setTexts((v) =>
                          v.map((x, j) => (j === i ? e.target.value : x)),
                        );
                        setCheck(null);
                        setConsent(false);
                        setError(null);
                      }}
                    />
                  </td>
                  <td
                    className={
                      values[i].error ||
                      check?.errors.some(
                        (x) =>
                          x.row_id === t.row.id &&
                          (!x.property_id || x.property_id === t.property.id),
                      )
                        ? "range-cell-error"
                        : undefined
                    }
                  >
                    {values[i].error ||
                      check?.errors
                        .filter(
                          (x) =>
                            x.row_id === t.row.id &&
                            (!x.property_id || x.property_id === t.property.id),
                        )
                        .map((x) => x.message)
                        .join(" · ") ||
                      "입력 형식 확인"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {!!error && (
          <RecoveryNotice
            error={error}
            status={error instanceof ApiError ? error.status : undefined}
          />
        )}
        <Button
          disabled={busy || values.some((v) => v.error)}
          onClick={() => void request(false)}
        >
          {busy ? "확인 중…" : "현재 행과 타입 검사"}
        </Button>
        {check && (
          <ChangeReview
            title={check.valid ? "모든 셀 검사 완료" : "오류 셀을 수정하세요"}
            description="검사 후에도 다른 편집자의 변경이나 속성 변경이 있으면 충돌로 멈춥니다."
            changes={[
              {
                label: "저장할 범위",
                before: "현재 행을 유지",
                after: `${targets.length}셀을 하나의 트랜잭션으로 반영`,
              },
            ]}
            busy={busy}
            disabled={!check.valid || !consent}
            confirmLabel="확인한 셀 모두 저장"
            onConfirm={() => void request(true)}
            onCancel={onClose}
          >
            <label className="check-label">
              <input
                type="checkbox"
                checked={consent}
                disabled={busy || !check.valid}
                onChange={(e) => setConsent(e.target.checked)}
              />
              현재 값과 붙여넣을 값을 비교했으며 모든 셀을 함께 변경합니다.
            </label>
          </ChangeReview>
        )}
      </div>
    </Modal>
  );
}
