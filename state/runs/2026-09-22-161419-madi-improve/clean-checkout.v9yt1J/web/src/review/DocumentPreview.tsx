import { lazy, Suspense, useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { ArrowUpRight, FileText, X } from "lucide-react";
import { ApiError, type Doc } from "../api";
import { useApp } from "../context";
import { Loading, Modal } from "../ui";
import { RecoveryNotice } from "./ChangeReview";
import "./document-preview.css";
const MarkdownContent = lazy(() =>
  import("../editor/MarkdownContent").then((m) => ({
    default: m.MarkdownContent,
  })),
);
export type DocumentPreviewProps = {
  documentId: string | null;
  onClose: () => void;
  expectedVersion?: number;
  sourceLine?: number;
  onOpen?: () => void;
  variant?: "modal" | "panel";
};
export default function DocumentPreview({
  documentId,
  onClose,
  expectedVersion,
  sourceLine,
  onOpen,
  variant = "modal",
}: DocumentPreviewProps) {
  const { user, workspace } = useApp(),
    [data, setData] = useState<Doc | null>(null),
    [error, setError] = useState<unknown>(null),
    [retry, setRetry] = useState(0);
  const scope = `${user.id}:${workspace?.id}:${documentId}`,
    current = useRef(scope),
    loadedScope = useRef("");
  current.current = scope;
  const opener = useRef<HTMLElement | null>(null),
    previousID = useRef<string | null>(null);
  if (documentId !== previousID.current) {
    if (documentId && document.activeElement instanceof HTMLElement)
      opener.current = document.activeElement;
    previousID.current = documentId;
  }
  const close = () => {
    onClose();
    requestAnimationFrame(() => {
      if (opener.current?.isConnected)
        opener.current.focus({ preventScroll: true });
    });
  };
  useEffect(() => {
    setData(null);
    setError(null);
    if (!documentId) return;
    const controller = new AbortController();
    let blocked = false,
      readSequence = 0;
    const fresh = () =>
      !blocked && !controller.signal.aborted && current.current === scope;
    const invalidate = (failure: unknown) => {
      if (!fresh()) return;
      blocked = true;
      readSequence++;
      setData(null);
      setError(failure);
    };
    const load = async () => {
      if (!fresh()) return;
      const sequence = ++readSequence;
      try {
        const response = await fetch(
          `/api/v1/documents/${encodeURIComponent(documentId)}`,
          {
            credentials: "same-origin",
            headers: { "X-Madi-Request": "1" },
            signal: controller.signal,
          },
        );
        if (!response.ok) {
          const body = await response.json().catch(() => ({}));
          throw new ApiError(
            body.error || "현재 문서를 미리 볼 수 없습니다.",
            response.status,
          );
        }
        const doc = (await response.json()) as Doc;
        if (fresh() && sequence === readSequence) {
          loadedScope.current = scope;
          setData(doc);
          setError(null);
        }
      } catch (e) {
        if (
          sequence === readSequence ||
          (e instanceof ApiError && [401, 403, 404].includes(e.status))
        )
          invalidate(e);
      }
    };
    void load();
    // Recheck current ACL with a small response, not a repeated 5MiB body load.
    let checking = false;
    const check = async () => {
      if (checking || !fresh()) return;
      checking = true;
      try {
        const r = await fetch(
          `/api/v1/documents/${encodeURIComponent(documentId)}/protection`,
          {
            credentials: "same-origin",
            headers: { "X-Madi-Request": "1" },
            signal: controller.signal,
          },
        );
        if (!r.ok)
          invalidate(
            new ApiError("현재 문서를 미리 볼 수 없습니다.", r.status),
          );
      } catch (e) {
        invalidate(e);
      } finally {
        checking = false;
      }
    };
    const timer = setInterval(() => void check(), 2000);
    window.addEventListener("focus", load);
    return () => {
      controller.abort();
      clearInterval(timer);
      window.removeEventListener("focus", load);
    };
  }, [scope, retry]);
  // Render gating prevents one paint of the previous source during an ID/scope change.
  const doc =
    loadedScope.current === scope &&
    data?.id === documentId &&
    data.workspace_id === workspace?.id
      ? data
      : null;
  if (!documentId) return null;
  const line =
    sourceLine && Number.isInteger(sourceLine) && sourceLine > 0
      ? sourceLine
      : undefined;
  const link = `/app/documents/${encodeURIComponent(documentId)}?${line ? `mode=source&line=${line}` : "mode=preview"}`;
  let previewStart = 0;
  if (doc && line) {
    for (let i = 1; i < Math.max(1, line - 5); i++) {
      const next = doc.markdown.indexOf("\n", previewStart);
      if (next < 0) break;
      previewStart = next + 1;
    }
  }
  const previewMarkdown =
    doc?.markdown.slice(previewStart, previewStart + 30000) || "";
  const body = (
    <div className="document-preview-content">
      <header>
        <div>
          <FileText size={18} />
          <h3>{doc?.title || "문서 미리보기"}</h3>
        </div>
        {variant === "panel" && (
          <button
            className="icon-button"
            type="button"
            aria-label="문서 미리보기 닫기"
            onClick={close}
          >
            <X size={20} />
          </button>
        )}
      </header>
      {error ? (
        <RecoveryNotice error={error} onRetry={() => setRetry((v) => v + 1)} />
      ) : !doc ? (
        <Loading />
      ) : (
        <>
          <p className="muted">
            불러온 버전 {doc.version}
            {line ? ` · 선택한 출처 ${line}행` : ""}
          </p>
          {expectedVersion !== undefined && expectedVersion !== doc.version && (
            <p className="notice">
              검색 당시 버전 {expectedVersion}에서 문서가 변경되었습니다. 아래는
              미리보기를 열 때 확인한 내용입니다.
            </p>
          )}
          <Suspense fallback={<Loading />}>
            <MarkdownContent markdown={previewMarkdown} />
          </Suspense>
          {doc.markdown.length > 30000 && (
            <p className="notice subtle">
              미리보기는 {previewStart ? "선택 위치 부근 최대" : "앞"}{" "}
              30,000자입니다. 전체 문서를 열어 나머지를 확인하세요.
            </p>
          )}
          <Link
            className="button primary"
            to={link}
            onClick={() => {
              onOpen?.();
              onClose();
            }}
          >
            <ArrowUpRight size={18} />
            문서 전체 열기
          </Link>
        </>
      )}
    </div>
  );
  return variant === "panel" ? (
    <aside className="document-preview-panel" aria-label="문서 미리보기">
      {body}
    </aside>
  ) : (
    <Modal
      open={!!documentId}
      onOpenChange={(open) => {
        if (!open) close();
      }}
      title="문서 미리보기"
      description="현재 접근 권한으로 문서를 다시 확인합니다. 미리보기는 편집하거나 저장하지 않습니다."
      wide
    >
      {body}
    </Modal>
  );
}
