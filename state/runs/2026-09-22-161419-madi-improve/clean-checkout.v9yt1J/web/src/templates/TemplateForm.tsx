import { lazy, Suspense, useEffect, useRef, useState } from "react";
import { Eye, FileCode2, Save } from "lucide-react";
import { api } from "../api";
import { useApp } from "../context";
import { Badge, Button, ErrorBox, Field, Loading, Modal } from "../ui";
import { kinds, scopes, type Template, type TemplateSpace } from "./model";
const MarkdownContent = lazy(() =>
  import("../editor/MarkdownContent").then((m) => ({
    default: m.MarkdownContent,
  })),
);

export default function TemplateForm({
  initial,
  spaces,
  close,
  saved,
}: {
  initial: Template;
  spaces: TemplateSpace[];
  close: () => void;
  saved: (value: Template) => void;
}) {
  const { workspace, documents, notify } = useApp();
  const [draft, setDraft] = useState({ ...initial }),
    [tags, setTags] = useState(initial.tags.join(", "));
  const [busy, setBusy] = useState(false),
    [error, setError] = useState<unknown>(null),
    [preview, setPreview] = useState(false),
    [discard, setDiscard] = useState(false);
  const active = useRef(true);
  const dirty =
    JSON.stringify(draft) !== JSON.stringify(initial) ||
    tags !== initial.tags.join(", ");
  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
    };
  }, []);
  useEffect(() => {
    const block = (event: BeforeUnloadEvent) => {
      if (dirty) {
        event.preventDefault();
        event.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", block);
    return () => window.removeEventListener("beforeunload", block);
  }, [dirty]);
  const change = (field: keyof Template, value: unknown) =>
    setDraft((d) => ({ ...d, [field]: value }));
  const requestClose = () => {
    if (busy) return;
    if (dirty) setDiscard(true);
    else close();
  };
  return (
    <>
      <Modal
        open
        onOpenChange={(open) => !open && requestClose()}
        title={initial.id ? "템플릿 수정" : "새 사용자 템플릿"}
        description="Markdown 원문과 YAML Front Matter를 저장합니다. 공유해도 수정 권한은 소유자에게만 있습니다."
        wide
      >
        <form
          className="template-form"
          onSubmit={async (event) => {
            event.preventDefault();
            if (!workspace || busy) return;
            setBusy(true);
            setError(null);
            try {
              const payload = {
                workspace_id: workspace.id,
                name: draft.name,
                description: draft.description,
                category: draft.category,
                icon: draft.icon,
                kind: draft.kind,
                markdown: draft.markdown,
                tags: tags
                  .split(",")
                  .map((t) => t.trim())
                  .filter(Boolean),
                visibility: draft.visibility,
                space_id: draft.space_id || "",
                version: initial.version,
              };
              const value = await api<Template>(
                initial.id ? `/templates/${initial.id}` : "/templates",
                initial.id ? "PUT" : "POST",
                payload,
              );
              if (!active.current) return;
              notify(
                value.protection?.changed
                  ? "민감정보 보호 정책에 따라 마스킹된 템플릿을 저장했습니다."
                  : "템플릿을 저장했습니다.",
              );
              saved(value);
            } catch (e) {
              if (active.current) setError(e);
            } finally {
              if (active.current) setBusy(false);
            }
          }}
        >
          <ErrorBox error={error} />
          <div className="template-form-fields">
            <Field label="템플릿 이름">
              <input
                required
                maxLength={160}
                value={draft.name}
                onChange={(e) => change("name", e.target.value)}
                disabled={busy}
              />
            </Field>
            <Field label="템플릿 분류">
              <input
                maxLength={40}
                placeholder="예: 팀 협업, IT 운영"
                value={draft.category}
                onChange={(e) => change("category", e.target.value)}
                disabled={busy}
              />
            </Field>
            <Field label="만들 문서 종류">
              <select
                value={draft.kind}
                onChange={(e) => change("kind", e.target.value)}
                disabled={busy}
              >
                {Object.entries(kinds).map(([key, label]) => (
                  <option value={key} key={key}>
                    {label}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="템플릿 아이콘" hint="이모지 또는 file을 입력하세요.">
              <input
                maxLength={16}
                value={draft.icon}
                onChange={(e) => change("icon", e.target.value)}
                disabled={busy}
              />
            </Field>
          </div>
          <Field label="템플릿 설명">
            <textarea
              rows={2}
              maxLength={1300}
              value={draft.description}
              onChange={(e) => change("description", e.target.value)}
              disabled={busy}
            />
          </Field>
          <div className="template-form-fields">
            <Field label="템플릿 공개 범위">
              <select
                value={draft.visibility}
                onChange={(e) => change("visibility", e.target.value)}
                disabled={busy}
              >
                {Object.entries(scopes).map(([key, label]) => (
                  <option value={key} key={key}>
                    {label}
                  </option>
                ))}
              </select>
            </Field>
            <Field
              label="템플릿 공간"
              hint="공간을 지정하면 비공개·워크스페이스 공유 모두 현재 공간 권한도 적용됩니다."
            >
              <select
                value={draft.space_id || ""}
                onChange={(e) => change("space_id", e.target.value)}
                required={draft.visibility === "space"}
                disabled={busy}
              >
                <option value="">공간 없음</option>
                {draft.space_id &&
                  !spaces.some((s) => s.id === draft.space_id) && (
                    <option value={draft.space_id}>
                      접근 권한을 다시 확인할 공간
                    </option>
                  )}
                {spaces
                  .filter((s) => s.can_write)
                  .map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.name}
                    </option>
                  ))}
              </select>
            </Field>
          </div>
          {draft.visibility !== "private" && (
            <p className="notice">
              현재 공개 범위의 사용자는 원문을 열람하고 자신의 템플릿·문서로
              복제할 수 있습니다. 공유 전 내용을 확인하세요.
            </p>
          )}
          <Field
            label="템플릿 태그"
            hint="쉼표로 구분합니다. 원문의 Front Matter에 tags가 있으면 해당 값을 우선합니다."
          >
            <input
              value={tags}
              onChange={(e) => setTags(e.target.value)}
              disabled={busy}
            />
          </Field>
          <div className="template-editor-heading">
            <strong>Markdown 원문</strong>
            <Badge>저장 후 버전 관리</Badge>
            <Button
              type="button"
              variant="secondary"
              onClick={() => setPreview(!preview)}
            >
              {preview ? <FileCode2 size={16} /> : <Eye size={16} />}{" "}
              {preview ? "원문 편집" : "미리보기"}
            </Button>
          </div>
          <p className="muted template-hint">
            문서를 만들 때만 {"{{date}}"}와 {"{{datetime}}"}을 개인 시간대의
            날짜·시간으로 바꿉니다. 그 밖의 표현은 실행하지 않고 그대로
            보존합니다.
          </p>
          {preview ? (
            <div className="template-markdown-preview">
              <Suspense fallback={<Loading />}>
                <MarkdownContent
                  markdown={draft.markdown || ""}
                  documents={documents}
                />
              </Suspense>
            </div>
          ) : (
            <textarea
              className="template-source"
              aria-label="템플릿 Markdown 원문"
              value={draft.markdown || ""}
              onChange={(e) => change("markdown", e.target.value)}
              spellCheck={false}
              disabled={busy}
            />
          )}
          <div className="modal-actions">
            <Button
              type="button"
              variant="secondary"
              onClick={requestClose}
              disabled={busy}
            >
              취소
            </Button>
            <Button type="submit" disabled={busy}>
              <Save size={16} />
              {busy ? "저장 중…" : "템플릿 저장"}
            </Button>
          </div>
        </form>
      </Modal>
      <Modal
        open={discard}
        onOpenChange={setDiscard}
        title="작성 중인 변경을 버릴까요?"
        description="아직 저장하지 않은 템플릿 변경은 사라집니다. 서버에 저장된 버전은 유지됩니다."
      >
        <div className="modal-actions">
          <Button variant="secondary" onClick={() => setDiscard(false)}>
            계속 작성
          </Button>
          <Button variant="danger" onClick={close}>
            변경 버리기
          </Button>
        </div>
      </Modal>
    </>
  );
}
