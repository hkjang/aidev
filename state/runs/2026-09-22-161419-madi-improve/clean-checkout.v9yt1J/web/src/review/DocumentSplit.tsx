import { useEffect, useRef, useState } from "react";
import { api, ApiError, type Doc } from "../api";
import { useApp } from "../context";
import { Button, Field, Modal } from "../ui";
import { ChangeReview, RecoveryNotice } from "./ChangeReview";
import { selectionBytes, type MarkdownSelection } from "./selection";
import "./selection-ai.css";

export type SplitResult = {
  source: Doc;
  child: Doc;
  replayed: boolean;
  receipt: {
    request_id: string;
    source_version: number;
    child_version: number;
  };
};
type Preview = {
  ticket: string;
  child_id: string;
  child_title: string;
  source_version: number;
  selected_markdown: string;
  replacement_markdown: string;
  notice: string;
  expires_in: number;
};
export default function DocumentSplit({
  doc,
  selection,
  stale,
  onClose,
  onCommit,
}: {
  doc: Doc;
  selection: MarkdownSelection;
  stale: boolean;
  onClose: () => void;
  onCommit: (
    ticket: string,
    requestID: string,
  ) => Promise<SplitResult | undefined>;
}) {
  const { user, workspace } = useApp();
  const [title, setTitle] = useState(
      selection.text
        .trim()
        .split("\n")[0]
        .replace(/^#{1,6}\s+/, "")
        .slice(0, 120) || "분리한 문서",
    ),
    [preview, setPreview] = useState<Preview | null>(null),
    [error, setError] = useState<unknown>(null),
    [consent, setConsent] = useState(false),
    [busy, setBusy] = useState(false),
    [applying, setApplying] = useState(false);
  const initialScope = useRef(`${user.id}:${workspace?.id}:${doc.id}`),
    mounted = useRef(true),
    generation = useRef(0),
    controller = useRef<AbortController | null>(null),
    requestID = useRef("");
  const scope = `${user.id}:${workspace?.id}:${doc.id}`;
  const scopeRef = useRef(scope);
  scopeRef.current = scope;
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      generation.current++;
      controller.current?.abort();
    };
  }, []);
  const invalidate = () => {
    generation.current++;
    controller.current?.abort();
    setPreview(null);
    setConsent(false);
    setBusy(false);
    requestID.current = "";
  };
  useEffect(() => {
    if (stale) {
      invalidate();
      setError(
        new ApiError(
          "문서나 권한이 변경되었습니다. 저장된 원문에서 다시 선택하세요.",
          409,
        ),
      );
    }
  }, [stale]);
  const alive = (sequence: number) =>
    mounted.current &&
    generation.current === sequence &&
    scopeRef.current === initialScope.current;
  const check = async () => {
    if (stale || busy || applying || !title.trim()) return;
    invalidate();
    const sequence = generation.current,
      abort = new AbortController();
    controller.current = abort;
    setBusy(true);
    setError(null);
    try {
      const result = await api<Preview>(
        `/documents/${doc.id}/split-preview`,
        "POST",
        {
          expected_version: doc.version,
          ...selectionBytes(doc.markdown, selection),
          selected_text: selection.text,
          title: title.trim(),
        },
        { signal: abort.signal },
      );
      if (!alive(sequence)) return;
      requestID.current = crypto.randomUUID();
      setPreview(result);
    } catch (e) {
      if (alive(sequence) && !abort.signal.aborted) setError(e);
    } finally {
      if (alive(sequence)) setBusy(false);
    }
  };
  const commit = async () => {
    if (!preview || stale || !consent || applying || !requestID.current) return;
    const sequence = generation.current;
    setApplying(true);
    setError(null);
    try {
      const result = await onCommit(preview.ticket, requestID.current);
      if (result && alive(sequence)) onClose();
    } catch (e) {
      if (alive(sequence)) setError(e);
    } finally {
      if (alive(sequence)) setApplying(false);
    }
  };
  if (scope !== initialScope.current) return null;
  return (
    <Modal
      open
      onOpenChange={(open) => {
        if (!open && !applying) onClose();
      }}
      title="선택 블록을 새 문서로 분리"
      description="저장된 완전한 Markdown 블록만 분리합니다. 원본의 참조 변경과 새 하위 문서는 하나의 작업으로 저장되고, 오류가 나면 둘 다 변경하지 않습니다."
      wide
    >
      <Field label="분리할 문서 제목">
        <input
          value={title}
          maxLength={160}
          disabled={applying || stale}
          onChange={(e) => {
            invalidate();
            setTitle(e.target.value);
            setError(null);
          }}
        />
      </Field>
      <p className="muted">
        원문 최대 1MiB · 선택 최대 256KiB. 문단·목록·표·코드 블록 전체를
        선택하세요. Front Matter와 HTML 확장 블록, 블록 중간은 원문 구조 보호를
        위해 분리하지 않습니다.
      </p>
      <Button
        disabled={stale || busy || applying || !title.trim()}
        onClick={() => void check()}
      >
        {busy
          ? "구조와 권한 확인 중…"
          : preview
            ? "분리 영향 다시 확인"
            : "분리 영향 확인"}
      </Button>
      {!!error && (
        <RecoveryNotice
          error={error}
          status={error instanceof ApiError ? error.status : undefined}
        />
      )}
      {preview && (
        <ChangeReview
          title="두 문서의 변경 확인"
          description={`원본 기준 v${preview.source_version} · 새 문서는 초안으로 생성하며 게시 승인을 대신하지 않습니다.`}
          changes={[
            {
              label: "원본의 선택 블록",
              before: <pre>{preview.selected_markdown}</pre>,
              after: <pre>{preview.replacement_markdown}</pre>,
            },
            {
              label: "새 하위 문서",
              before: "없음",
              after: (
                <>
                  <strong>{preview.child_title}</strong>
                  <pre>{preview.selected_markdown}</pre>
                </>
              ),
            },
          ]}
          warnings={[
            preview.notice,
            "실행 중 연결이 끊기면 같은 확인 화면에서 재시도하세요. 요청 ID를 보존해 중복 문서를 만들지 않습니다.",
          ]}
          busy={applying}
          disabled={!consent || stale}
          confirmLabel="확인한 블록 분리"
          onConfirm={() => void commit()}
          onCancel={onClose}
        >
          <label className="check-label">
            <input
              type="checkbox"
              checked={consent}
              disabled={applying || stale}
              onChange={(e) => setConsent(e.target.checked)}
            />
            원본에 참조를 남기고, 원본 권한을 상속하는 새 초안으로 분리하는 것을
            확인했습니다.
          </label>
        </ChangeReview>
      )}
    </Modal>
  );
}
