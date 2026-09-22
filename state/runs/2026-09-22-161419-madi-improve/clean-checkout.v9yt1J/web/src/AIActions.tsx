import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { FilePlus2 } from "lucide-react";
import { api, downloadText } from "./api";
import { ChangeReview, RecoveryNotice } from "./review/ChangeReview";
import { useApp } from "./context";
import { Button, ErrorBox, Field, Modal } from "./ui";
import { MarkdownContent } from "./editor/MarkdownContent";
import type { CitationSource } from "./CitationViewer";

export type DocumentAIAction = {
  id: string;
  name: string;
  description: string;
  prompt: string;
};

export function AIActionSelect({
  value,
  disabled,
  onChange,
}: {
  value: string;
  disabled: boolean;
  onChange: (action: DocumentAIAction) => void;
}) {
  const [actions, setActions] = useState<DocumentAIAction[]>([]);
  const [error, setError] = useState("");
  const { user } = useApp();
  useEffect(() => {
    let active = true;
    api<{ actions: DocumentAIAction[] }>("/ai/actions")
      .then((v) => {
        if (active) setActions(v.actions);
      })
      .catch((e) => {
        if (active) setError(e.message);
      });
    return () => {
      active = false;
    };
  }, [user.id]);
  return (
    <div className="ai-action-select">
      <label htmlFor="madi-ai-action">AI 작업</label>
      <select
        id="madi-ai-action"
        value={value}
        disabled={disabled || !actions.length}
        onChange={(e) => {
          const selected = actions.find((a) => a.id === e.target.value);
          if (selected) onChange(selected);
        }}
      >
        {!actions.length && <option value="ask">지식에 질문</option>}
        {actions.map((action) => (
          <option key={action.id} value={action.id}>
            {action.name}
          </option>
        ))}
      </select>
      <small>
        {actions.find((action) => action.id === value)?.description ||
          "작업 목록을 불러오는 중입니다."}
      </small>
      <ErrorBox error={error} />
    </div>
  );
}

// AI suggestions never silently overwrite a live editor or publish content.
// The user reviews an exact draft and explicitly creates a new private page.
export function AISaveDraft({
  answer,
  question,
  sources,
  onSaved,
}: {
  answer: string;
  question: string;
  sources: CitationSource[];
  onSaved: () => void;
}) {
  const { workspace, user, documents, reload, notify } = useApp();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false),
    [title, setTitle] = useState(""),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const generation = useRef(0);
  const [errorStatus, setErrorStatus] = useState(0);
  const draft =
    answer +
    (sources.length
      ? "\n\n---\n\n## 참고 자료\n\n" +
        sources
          .map((source, index) => {
            const title = source.title
              .replace(/[\\[\]()`*_]/g, "\\$&")
              .replace(/[\r\n]/g, " ");
            return `- [${index + 1}] [${title}](/app/documents/${source.id}?line=${source.start_line || 1}) · 버전 ${source.version} · ${source.start_line || 1}–${source.end_line || 1}행`;
          })
          .join("\n") +
        "\n"
      : "");
  useEffect(() => {
    generation.current++;
    setOpen(false);
    setBusy(false);
    setError("");
    return () => {
      generation.current++;
    };
  }, [workspace?.id, workspace?.role, user.id, user.role, answer]);
  const save = async () => {
    if (!workspace || !title.trim() || busy) return;
    const request = generation.current;
    setBusy(true);
    setError("");
    try {
      for (const source of sources) {
        const document = await api(
          source.citation_url || `/documents/${source.id}`,
        );
        if ((document.source?.version ?? document.version) !== source.version)
          throw new Error(
            "참조 문서가 변경되었습니다. 현재 자료로 답변을 다시 생성한 뒤 저장하세요.",
          );
        if (request !== generation.current) return;
      }
      const doc = await api("/documents", "POST", {
        workspace_id: workspace.id,
        title: title.trim(),
        markdown: draft,
        visibility: "private",
      });
      if (request !== generation.current) return;
      await reload();
      if (request !== generation.current) return;
      notify("AI 제안을 새 개인 초안으로 저장했습니다.");
      setOpen(false);
      onSaved();
      navigate(`/app/documents/${doc.id}?mode=preview`);
    } catch (e) {
      if (request === generation.current) {
        setError((e as Error).message);
        setErrorStatus(Number((e as { status?: number }).status) || 0);
      }
    } finally {
      if (request === generation.current) setBusy(false);
    }
  };
  if (
    !workspace ||
    user.role === "viewer" ||
    !["owner", "admin", "editor"].includes(workspace.role)
  )
    return (
      <p className="muted">
        새 개인 문서로 저장하려면 현재 워크스페이스의 문서 작성 권한이
        필요합니다.
      </p>
    );
  return (
    <>
      <Button
        type="button"
        onClick={() => {
          setTitle(Array.from(question).slice(0, 100).join(""));
          setError("");
          setOpen(true);
        }}
      >
        <FilePlus2 size={16} /> 새 개인 문서로 저장
      </Button>
      <Modal
        open={open}
        onOpenChange={(value) => {
          if (!busy) setOpen(value);
        }}
        title="AI 초안 검토"
        wide
      >
        <ChangeReview
          title="새 개인 초안 생성"
          description="AI 제안과 참조 구간을 확인한 뒤 저장하세요."
          changes={[
            {
              label: "저장 대상",
              before: "기존 문서 유지",
              after: "현재 워크스페이스의 새 비공개 초안",
            },
            {
              label: "참조 문서",
              before: "",
              after: `${sources.length}개 · 저장 직전 현재 권한과 버전 재검사`,
            },
          ]}
          warnings={[
            "AI가 만든 제안입니다. 민감정보와 문서 저장 정책이 동일하게 적용되며, 기존 문서는 변경하거나 게시하지 않습니다.",
          ]}
          busy={busy}
          disabled={!title.trim()}
          confirmLabel="검토한 내용을 개인 초안으로 저장"
          onConfirm={() => void save()}
          onCancel={() => setOpen(false)}
        >
          <Field label="새 문서 제목">
            <input
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              required
              maxLength={150}
              disabled={busy}
            />
          </Field>
          <div
            style={{
              maxHeight: "50vh",
              overflow: "auto",
              padding: 16,
              border: "1px solid var(--border)",
              borderRadius: 12,
              margin: "16px 0",
            }}
          >
            <MarkdownContent markdown={draft} documents={documents} />
          </div>

          <RecoveryNotice
            error={error}
            status={errorStatus}
            dirty
            busy={busy}
            onCopy={() => downloadText("madi-AI-초안.md", draft)}
            onReauthenticate={() => navigate("/login")}
          />
        </ChangeReview>
      </Modal>
    </>
  );
}
