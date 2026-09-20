import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Download, FileText, Paperclip, Trash2, Undo2 } from "lucide-react";
import { useRef, useState } from "react";
import { api } from "../../lib/api";
import { formatDate } from "../../lib/utils";
import { Button, Card } from "../../components/ui";
import type { AppDocument } from "../../types";

// These three bounds mirror what the server enforces. Checking the picked file
// first keeps a mistaken choice off the wire instead of uploading twenty
// megabytes only to be told the extension is wrong.
export const MAX_GUIDE_DOCUMENT_BYTES = 20 * 1024 * 1024;
export const MAX_GUIDE_DOCUMENTS = 10;
export const GUIDE_DOCUMENT_EXTENSIONS = [
  ".pdf",
  ".md",
  ".markdown",
  ".txt",
  ".csv",
  ".doc",
  ".docx",
  ".ppt",
  ".pptx",
  ".xls",
  ".xlsx",
  ".hwp",
  ".hwpx",
];
export const GUIDE_DOCUMENT_ACCEPT = GUIDE_DOCUMENT_EXTENSIONS.join(",");

export function formatFileSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 KB";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function guideDocumentError(
  file: File,
  alreadyAttached: number,
): string {
  const extension = file.name.slice(file.name.lastIndexOf(".")).toLowerCase();
  if (
    !file.name.includes(".") ||
    !GUIDE_DOCUMENT_EXTENSIONS.includes(extension)
  )
    return `${file.name}: PDF, Markdown, 텍스트, CSV, Word, PowerPoint, Excel, 한글 문서만 첨부할 수 있습니다.`;
  if (file.size === 0) return `${file.name}: 빈 파일은 첨부할 수 없습니다.`;
  if (file.size > MAX_GUIDE_DOCUMENT_BYTES)
    return `${file.name}: 가이드 문서는 20MB 이하여야 합니다.`;
  if (alreadyAttached >= MAX_GUIDE_DOCUMENTS)
    return `가이드 문서는 앱당 ${MAX_GUIDE_DOCUMENTS}개까지 첨부할 수 있습니다.`;
  return "";
}

export interface GuideDocumentDraft {
  /** Documents already stored for the app; empty while the app is being created. */
  stored: AppDocument[];
  storedError: Error | null;
  /** Files picked in this form and not uploaded yet. */
  pending: File[];
  /** Stored documents marked for removal on the next save. */
  removed: string[];
  pickError: string;
  applying: boolean;
  applyError: Error | null;
  add: (files: FileList | null) => void;
  dropPending: (index: number) => void;
  toggleRemoved: (id: string) => void;
  /** Applies the staged removals and uploads. Called after the app is saved. */
  apply: (appId: string) => Promise<void>;
}

/**
 * Staging area for an app form's guide attachments.
 *
 * Uploading on pick would need an app that does not exist yet on the register
 * screen, and would leave a half-attached app behind if the form is abandoned.
 * So both the additions and the removals wait here until the form is saved and
 * the app has an identifier; `apply` then replays them in one go.
 */
export function useGuideDocumentDraft(appId?: string): GuideDocumentDraft {
  const client = useQueryClient();
  const [pending, setPending] = useState<File[]>([]);
  const [removed, setRemoved] = useState<string[]>([]);
  const [pickError, setPickError] = useState("");
  const stored = useQuery({
    queryKey: ["app-documents", appId],
    queryFn: ({ signal }) => api.appDocuments(appId ?? "", signal),
    enabled: !!appId,
  });
  const documents = stored.data ?? [];

  const apply = useMutation({
    mutationFn: async (targetAppId: string) => {
      for (const id of removed) await api.deleteAppDocument(targetAppId, id);
      for (const file of pending)
        await api.uploadAppDocument(targetAppId, file);
      setPending([]);
      setRemoved([]);
      await client.invalidateQueries({ queryKey: ["app-documents"] });
    },
  });

  return {
    stored: documents,
    storedError: stored.error,
    pending,
    removed,
    pickError,
    applying: apply.isPending,
    applyError: apply.error,
    add: (files) => {
      const picked = [...(files ?? [])];
      if (!picked.length) return;
      // One name per app is what the server stores, so a name already taken
      // is refused here rather than on a save that would fail halfway.
      const taken = new Set(
        documents
          .filter((document) => !removed.includes(document.id))
          .map((document) => document.fileName.toLowerCase())
          .concat(pending.map((file) => file.name.toLowerCase())),
      );
      let attached = documents.length - removed.length + pending.length;
      const accepted: File[] = [];
      let error = "";
      for (const file of picked) {
        const failure = taken.has(file.name.toLowerCase())
          ? file.name + ": 같은 이름의 문서가 이미 있습니다."
          : guideDocumentError(file, attached);
        if (failure) {
          error = failure;
          continue;
        }
        taken.add(file.name.toLowerCase());
        accepted.push(file);
        attached += 1;
      }
      setPickError(error);
      if (accepted.length) setPending((current) => [...current, ...accepted]);
    },
    dropPending: (index) => {
      setPickError("");
      setPending((current) => current.filter((_, at) => at !== index));
    },
    toggleRemoved: (id) =>
      setRemoved((current) =>
        current.includes(id)
          ? current.filter((item) => item !== id)
          : [...current, id],
      ),
    apply: async (targetAppId: string) => {
      if (!removed.length && !pending.length) return;
      await apply.mutateAsync(targetAppId);
    },
  };
}

/** The attachment control both the owner form and the admin form render. */
export function GuideDocumentsField({
  draft,
  idPrefix = "app",
}: {
  draft: GuideDocumentDraft;
  idPrefix?: string;
}) {
  const fileRef = useRef<HTMLInputElement>(null);
  const attached =
    draft.stored.length - draft.removed.length + draft.pending.length;
  return (
    <div className="field">
      <span className="field-label">가이드 문서</span>
      <ul className="document-list">
        {draft.stored.map((document) => {
          const dropping = draft.removed.includes(document.id);
          return (
            <li
              key={document.id}
              className={dropping ? "document-row is-removed" : "document-row"}
            >
              <FileText size={18} aria-hidden="true" />
              <span className="document-name">
                <strong>{document.title || document.fileName}</strong>
                <span className="document-meta">
                  {document.fileName} · {formatFileSize(document.size)} ·{" "}
                  {formatDate(document.createdAt)}
                  {dropping ? " · 저장 시 삭제됩니다" : ""}
                </span>
              </span>
              <Button
                variant={dropping ? "secondary" : "danger"}
                size="sm"
                className="button-quiet"
                onClick={() => draft.toggleRemoved(document.id)}
              >
                {dropping ? (
                  <>
                    <Undo2 size={16} /> 삭제 취소
                  </>
                ) : (
                  <>
                    <Trash2 size={16} /> 삭제
                  </>
                )}
              </Button>
            </li>
          );
        })}
        {draft.pending.map((file, index) => (
          <li className="document-row" key={`${file.name}-${index}`}>
            <Paperclip size={18} aria-hidden="true" />
            <span className="document-name">
              <strong>{file.name}</strong>
              <span className="document-meta">
                {formatFileSize(file.size)} · 저장 시 업로드됩니다
              </span>
            </span>
            <Button
              variant="danger"
              size="sm"
              className="button-quiet"
              onClick={() => draft.dropPending(index)}
            >
              <Trash2 size={16} /> 빼기
            </Button>
          </li>
        ))}
      </ul>
      <input
        ref={fileRef}
        id={`${idPrefix}-guide-documents`}
        type="file"
        multiple
        className="sr-only"
        accept={GUIDE_DOCUMENT_ACCEPT}
        aria-label="가이드 문서 파일 선택"
        onChange={(event) => {
          draft.add(event.target.files);
          event.target.value = "";
        }}
      />
      <div className="branding-actions">
        <Button
          variant="secondary"
          size="sm"
          disabled={attached >= MAX_GUIDE_DOCUMENTS || draft.applying}
          onClick={() => fileRef.current?.click()}
        >
          <Paperclip size={16} /> 문서 첨부
        </Button>
      </div>
      <span className="field-help">
        사용자가 앱 상세 화면에서 로그인 없이 내려받습니다. PDF, Markdown,
        텍스트, CSV, Word, PowerPoint, Excel, 한글 문서 · 파일당 20MB · 앱당{" "}
        {MAX_GUIDE_DOCUMENTS}개까지.
      </span>
      {(draft.pickError || draft.applyError || draft.storedError) && (
        <span className="field-error" role="alert">
          {draft.pickError ||
            draft.applyError?.message ||
            draft.storedError?.message}
        </span>
      )}
    </div>
  );
}

/**
 * The download list on the public app detail page. Guides of a published app
 * are open to everyone, so this renders for signed-out visitors too and simply
 * stays away when an app has nothing attached.
 */
export function AppGuideDocuments({ appSlug }: { appSlug: string }) {
  const documents = useQuery({
    queryKey: ["app-documents", appSlug],
    queryFn: ({ signal }) => api.appDocuments(appSlug, signal),
    enabled: !!appSlug,
  });
  const items = documents.data ?? [];
  if (!items.length) return null;
  return (
    <Card className="prose-card">
      <h2>가이드 문서</h2>
      <ul className="document-list mt-4">
        {items.map((document) => (
          <li className="document-row" key={document.id}>
            <FileText size={18} aria-hidden="true" />
            <span className="document-name">
              <strong>{document.title || document.fileName}</strong>
              <span className="document-meta">
                {document.fileName} · {formatFileSize(document.size)} ·{" "}
                {formatDate(document.createdAt)}
              </span>
            </span>
            <a
              className="button button-secondary button-sm"
              href={document.downloadUrl ?? ""}
              download={document.fileName}
            >
              <Download size={16} /> 내려받기
            </a>
          </li>
        ))}
      </ul>
    </Card>
  );
}
