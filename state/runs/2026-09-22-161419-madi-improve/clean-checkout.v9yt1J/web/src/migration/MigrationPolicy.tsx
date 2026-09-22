import { useEffect, useRef, useState } from "react";
import { api } from "../api";
import { useApp } from "../context";
import { Button, Field } from "../ui";
import { ChangeReview, RecoveryNotice } from "../review/ChangeReview";

export default function MigrationPolicy() {
  const { user, notify } = useApp();
  const generation = useRef(0);
  const [value, setValue] = useState<Record<string, number> | null>(null),
    [original, setOriginal] = useState<Record<string, number> | null>(null),
    [review, setReview] = useState(false),
    [busy, setBusy] = useState(false),
    [error, setError] = useState<unknown>(null);
  useEffect(() => {
    let live = true;
    generation.current++;
    setValue(null);
    setOriginal(null);
    setReview(false);
    setBusy(false);
    setError(null);
    if (user?.role === "admin")
      api<Record<string, number>>("/admin/migration/settings")
        .then((v) => {
          if (live) {
            setValue(v);
            setOriginal(v);
          }
        })
        .catch((e) => {
          if (live) setError(e);
        });
    return () => {
      live = false;
      generation.current++;
    };
  }, [user?.id]);
  if (user?.role !== "admin") return null;
  return (
    <details className="migration-legacy">
      <summary>관리자 · 재개 이관 보관 한도</summary>
      <section className="panel padded">
        <RecoveryNotice error={error} />
        {value && original && (
          <>
            <p>
              실행 중인 서비스의 환경변수를 추가하지 않고 보관 한도를
              설정합니다. 원본은 암호화되고, 항목별 문서 4MiB/첨부 50MiB 한도는
              유지합니다.
            </p>
            <fieldset
              disabled={busy}
              style={{ border: 0, padding: 0, minWidth: 0 }}
            >
              <div className="form-grid">
                {[
                  ["max_session_bytes", "세션 원본 한도 (MiB)", 1048576],
                  ["max_user_bytes", "개인 보관 한도 (MiB)", 1048576],
                  ["max_items", "세션 항목 수", 1],
                  ["retention_days", "비교 데이터 보관일", 1],
                ].map(([key, label, unit]) => (
                  <Field key={key} label={String(label)}>
                    <input
                      type="number"
                      min={1}
                      value={value[String(key)] / Number(unit)}
                      onChange={(e) =>
                        setValue((v) =>
                          v
                            ? {
                                ...v,
                                [String(key)]:
                                  Number(e.target.value) * Number(unit),
                              }
                            : v,
                        )
                      }
                    />
                  </Field>
                ))}
              </div>
              <Button onClick={() => setReview(true)}>설정 변경 검토</Button>
            </fieldset>
            {review && (
              <ChangeReview
                title="이관 보관 정책 변경"
                changes={Object.keys(value)
                  .filter((k) => k !== "revision" && value[k] !== original[k])
                  .map((k) => ({
                    label:
                      (
                        {
                          max_session_bytes: "세션 원본·첨부 복제 한도",
                          max_user_bytes: "개인 원본 보관 한도",
                          max_items: "세션 항목 수",
                          retention_days: "비교 데이터 보관일",
                        } as Record<string, string>
                      )[k] || k,
                    before: original[k],
                    after: value[k],
                  }))}
                busy={busy}
                onCancel={() => setReview(false)}
                onConfirm={() => {
                  const currentGeneration = generation.current;
                  setBusy(true);
                  setError(null);
                  api("/admin/migration/settings", "PUT", value)
                    .then(() =>
                      api<Record<string, number>>("/admin/migration/settings"),
                    )
                    .then((v) => {
                      if (currentGeneration !== generation.current) return;
                      setValue(v);
                      setOriginal(v);
                      setReview(false);
                      notify("이관 보관 정책을 저장했습니다.");
                    })
                    .catch((e) => {
                      if (currentGeneration === generation.current) setError(e);
                    })
                    .finally(() => {
                      if (currentGeneration === generation.current)
                        setBusy(false);
                    });
                }}
              />
            )}
          </>
        )}
      </section>
    </details>
  );
}
