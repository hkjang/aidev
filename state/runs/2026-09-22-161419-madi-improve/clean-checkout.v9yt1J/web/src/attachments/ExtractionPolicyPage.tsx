import { useEffect, useRef, useState } from "react";
import { api } from "../api";
import { useApp } from "../context";
import { Button, Field, Loading, Modal, PageHeading, Toggle } from "../ui";
import { ChangeReview, RecoveryNotice } from "../review/ChangeReview";
import type { Policy } from "./types";
import "./style.css";
export default function ExtractionPolicyPage() {
  const { user } = useApp();
  const actor = useRef(user.id);
  actor.current = user.id;
  const [saved, setSaved] = useState<Policy | null>(null),
    [draft, setDraft] = useState<Policy | null>(null),
    [error, setError] = useState<unknown>(null),
    [busy, setBusy] = useState(false),
    [review, setReview] = useState(false),
    [reload, setReload] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    setSaved(null);
    setDraft(null);
    setReview(false);
    setBusy(false);
    void api<Policy>(
      "/admin/attachment-extraction/settings",
      "GET",
      undefined,
      { signal: controller.signal },
    )
      .then((v) => {
        if (!controller.signal.aborted) {
          setSaved(v);
          setDraft(structuredClone(v));
          setError(null);
        }
      })
      .catch((e) => {
        if (!controller.signal.aborted) setError(e);
      });
    return () => controller.abort();
  }, [user.id, reload]);
  const save = async () => {
    if (!draft || busy) return;
    const id = user.id;
    setBusy(true);
    try {
      const v = await api<Policy>(
        "/admin/attachment-extraction/settings",
        "PUT",
        draft,
      );
      if (actor.current !== id) return;
      setSaved(v);
      setDraft(structuredClone(v));
      setReview(false);
      setError(null);
    } catch (e) {
      if (actor.current === id) setError(e);
    } finally {
      if (actor.current === id) setBusy(false);
    }
  };
  return (
    <main className="page extract-page">
      <PageHeading
        eyebrow="관리자 정책"
        title="첨부 본문 추출"
        description="인터넷 없이 동작하는 PDF·Office 추출과 선택 OCR 정책입니다. 기본값은 꺼짐이며 설정 변경은 현재 추출 결과를 무효화합니다."
      />
      <RecoveryNotice
        error={error}
        dirty={JSON.stringify(draft) !== JSON.stringify(saved)}
        onReview={() => {
          setReview(false);
          setReload((v) => v + 1);
        }}
        onRetry={() => setReload((v) => v + 1)}
      />
      {!draft ? (
        <Loading />
      ) : (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            setReview(true);
          }}
          className="card"
        >
          <fieldset disabled={busy}>
            <Toggle
              label="첨부 본문 추출 사용"
              checked={draft.data.enabled}
              onChange={(v) =>
                setDraft({ ...draft, data: { ...draft.data, enabled: v } })
              }
              description="쓰기 권한 사용자가 요청한 원본만 PostgreSQL 작업으로 처리합니다."
            />
            <Toggle
              label="선택 OCR 허용"
              checked={draft.data.ocr_enabled}
              onChange={(v) =>
                setDraft({ ...draft, data: { ...draft.data, ocr_enabled: v } })
              }
              description="텍스트 없는 PDF 쪽을 사용자가 지정해야 실행합니다. 한 작업 최대 20쪽·영어 및 한국어."
            />
            <div className="extract-policy-grid">
              {(
                [
                  ["max_pages", "최대 PDF 쪽 수", 1, 500],
                  ["max_fragments", "최대 본문 조각 수", 100, 20000],
                  ["timeout_seconds", "전체 작업 제한시간(초)", 30, 600],
                ] as const
              ).map(([key, label, min, max]) => (
                <Field key={key} label={label}>
                  <input
                    type="number"
                    min={min}
                    max={max}
                    required
                    value={draft.data[key]}
                    onChange={(e) =>
                      setDraft({
                        ...draft,
                        data: { ...draft.data, [key]: Number(e.target.value) },
                      })
                    }
                  />
                </Field>
              ))}
              <Field label="최대 원본 크기(MiB)">
                <input
                  type="number"
                  min={1}
                  max={50}
                  required
                  step={1}
                  value={draft.data.max_file_bytes / 1048576}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      data: {
                        ...draft.data,
                        max_file_bytes: Number(e.target.value) * 1048576,
                      },
                    })
                  }
                />
              </Field>
              <Field label="최대 추출 텍스트(KiB)">
                <input
                  type="number"
                  min={64}
                  max={8192}
                  required
                  step={1}
                  value={draft.data.max_text_bytes / 1024}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      data: {
                        ...draft.data,
                        max_text_bytes: Number(e.target.value) * 1024,
                      },
                    })
                  }
                />
              </Field>
            </div>
            <p className="notice">
              PDF는 Poppler, OCR은 Tesseract를 고정 인자로 실행합니다. Linux
              Landlock·seccomp가 지원되지 않으면 PDF/OCR은 실행을 거부합니다.
              환경변수 비밀·네트워크·다른 파일 경로는 추출기에 전달하지
              않습니다.
            </p>
            <p>
              Office는 DOCX·PPTX·XLSX의 ZIP/XML을 제한된 크기로 읽습니다. 레거시
              DOC·PPT·XLS, 암호화 문서, 이미지 속 Office 글자, 매크로 실행은
              지원하지 않습니다. 원본을 열어 품질을 대조하세요.
            </p>
            <Button type="submit" variant="primary">
              정책 변경 검토
            </Button>
          </fieldset>
        </form>
      )}
      <Modal
        open={review}
        onOpenChange={(v) => {
          if (!busy) setReview(v);
        }}
        title="첨부 추출 정책 변경"
        wide
      >
        {draft && saved && (
          <ChangeReview
            title="현재 추출 결과와 실행 작업에 영향"
            changes={[
              {
                label: "추출 사용",
                before: saved.data.enabled ? "사용" : "꺼짐",
                after: draft.data.enabled ? "사용" : "꺼짐",
              },
              {
                label: "선택 OCR",
                before: saved.data.ocr_enabled ? "허용" : "차단",
                after: draft.data.ocr_enabled ? "허용" : "차단",
              },
              {
                label: "정책 버전",
                before: saved.revision,
                after: saved.revision + 1,
              },
            ]}
            warnings={[
              "정책의 어떤 값이라도 바꾸면 기존 추출 결과는 검색·인용에서 제외되고 실행 중 작업은 중단됩니다. 필요한 첨부를 새 정책으로 다시 추출하세요.",
              "추출은 완전한 문서 변환이나 개인정보 탐지를 보증하지 않습니다. 원본과 결과를 비교해야 합니다.",
            ]}
            onConfirm={() => void save()}
            onCancel={() => setReview(false)}
            busy={busy}
          />
        )}
      </Modal>
    </main>
  );
}
