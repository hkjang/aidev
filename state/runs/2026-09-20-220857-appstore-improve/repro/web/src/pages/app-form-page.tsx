import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, Send } from "lucide-react";
import { useEffect, useState, type FormEvent } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { api } from "../lib/api";
import { parseList } from "../lib/utils";
import { RejectionNotice } from "../features/apps/rejection-notice";
import {
  GuideDocumentsField,
  useGuideDocumentDraft,
} from "../features/apps/guide-documents";
import {
  Button,
  Card,
  ErrorState,
  Field,
  Input,
  LoadingState,
  PageHeader,
  Select,
  Switch,
  Textarea,
} from "../components/ui";

interface FormState {
  name: string;
  slug: string;
  summary: string;
  description: string;
  icon: string;
  serviceUrl: string;
  categoryId: string;
  tags: string;
  screenshots: string;
  team: string;
  language: string;
  framework: string;
  version: string;
  supportsMcp: boolean;
  supportsApi: boolean;
  visibility: "public" | "private";
}

const empty: FormState = {
  name: "",
  slug: "",
  summary: "",
  description: "",
  icon: "📦",
  serviceUrl: "",
  categoryId: "",
  tags: "",
  screenshots: "",
  team: "",
  language: "",
  framework: "",
  version: "",
  supportsMcp: false,
  supportsApi: false,
  visibility: "public",
};

export function AppFormPage({ edit = false }: { edit?: boolean }) {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const client = useQueryClient();
  const categories = useQuery({
    queryKey: ["categories"],
    queryFn: ({ signal }) => api.categories(signal),
  });
  const existing = useQuery({
    queryKey: ["my-app", id],
    queryFn: async ({ signal }) =>
      (await api.myApps(signal)).find((app) => app.id === id),
    enabled: edit,
  });
  const [form, setForm] = useState<FormState>(empty);
  const [complete, setComplete] = useState(false);
  const documents = useGuideDocumentDraft(edit ? id : undefined);
  useEffect(() => {
    if (!existing.data) return;
    setForm({
      name: existing.data.name,
      slug: existing.data.slug,
      summary: existing.data.summary,
      description: existing.data.description ?? "",
      icon: existing.data.icon ?? "📦",
      serviceUrl: existing.data.serviceUrl ?? "",
      categoryId: existing.data.category?.id ?? "",
      tags: existing.data.tags?.join(", ") ?? "",
      screenshots: existing.data.screenshots?.join(", ") ?? "",
      team: existing.data.team ?? "",
      language: existing.data.language ?? "",
      framework: existing.data.framework ?? "",
      version: existing.data.version ?? "",
      supportsMcp: !!existing.data.supportsMcp,
      supportsApi: !!existing.data.supportsApi,
      visibility: existing.data.visibility ?? "public",
    });
  }, [existing.data]);
  const save = useMutation({
    mutationFn: async () => {
      const payload = {
        ...form,
        tags: parseList(form.tags),
        screenshots: parseList(form.screenshots),
      };
      const saved = edit
        ? await api.updateApp(id, payload)
        : await api.createApp(payload);
      // Attachments need the app to exist, so they are applied once it does.
      // A failure here surfaces on the form with the app already saved.
      await documents.apply(saved?.id ?? id);
      return saved;
    },
    onSuccess: async () => {
      setComplete(true);
      await client.invalidateQueries({ queryKey: ["my-apps"] });
    },
  });
  const update = <K extends keyof FormState>(key: K, value: FormState[K]) =>
    setForm((current) => ({ ...current, [key]: value }));
  const submit = (event: FormEvent) => {
    event.preventDefault();
    save.mutate();
  };

  if (edit && existing.isPending)
    return (
      <div className="page">
        <LoadingState />
      </div>
    );
  if (edit && existing.error)
    return (
      <div className="page">
        <ErrorState
          error={existing.error}
          retry={() => void existing.refetch()}
        />
      </div>
    );
  if (complete)
    return (
      <div className="page">
        <div className="state-panel">
          <div>
            <div className="state-icon">
              <CheckCircle2 />
            </div>
            <h2>{edit ? "앱이 수정되었습니다" : "앱이 등록되었습니다"}</h2>
            <p>
              승인 Workflow 설정에 따라 즉시 게시되거나 검토 대기 상태가 됩니다.
            </p>
            <div className="state-actions">
              <Button onClick={() => navigate("/my/apps")}>
                내 앱으로 이동
              </Button>
            </div>
          </div>
        </div>
      </div>
    );

  return (
    <div className="page">
      <PageHeader
        eyebrow={edit ? "Edit application" : "Self-service publishing"}
        title={edit ? "앱 수정" : "앱 등록"}
        description="사용자가 실제로 접속할 서비스 URL과 앱 정보를 입력하세요. Git 저장소 정보는 수집하거나 노출하지 않습니다."
      />
      {edit && existing.data && (
        <RejectionNotice app={existing.data} className="mb-5" />
      )}
      <Card className="form-section">
        <form onSubmit={submit}>
          <div className="form-grid">
            <Field label="앱 이름" id="app-name">
              <Input
                id="app-name"
                value={form.name}
                onChange={(event) => update("name", event.target.value)}
                required
                maxLength={100}
              />
            </Field>
            <Field
              label="Slug"
              id="app-slug"
              help="URL에 사용되는 영문 소문자 식별자"
            >
              <Input
                id="app-slug"
                value={form.slug}
                onChange={(event) =>
                  update(
                    "slug",
                    event.target.value
                      .toLowerCase()
                      .replace(/[^a-z0-9-]/g, "-"),
                  )
                }
                required
                pattern="[a-z0-9]+(?:-[a-z0-9]+)*"
              />
            </Field>
            <Field label="한 줄 설명" id="app-summary">
              <Input
                id="app-summary"
                value={form.summary}
                onChange={(event) => update("summary", event.target.value)}
                required
                maxLength={180}
              />
            </Field>
            <Field
              label="앱 아이콘"
              id="app-icon"
              help="Emoji 또는 짧은 문자(예: 🚀, AI)"
            >
              <Input
                id="app-icon"
                value={form.icon}
                onChange={(event) => update("icon", event.target.value)}
                required
                maxLength={16}
              />
            </Field>
            <Field
              label="서비스 URL"
              id="app-service-url"
              help="사용자에게 앱 실행 링크로 노출됩니다."
            >
              <Input
                id="app-service-url"
                type="url"
                value={form.serviceUrl}
                onChange={(event) => update("serviceUrl", event.target.value)}
                required
                maxLength={2048}
                placeholder="https://service.example.internal"
              />
            </Field>
            <Field label="카테고리" id="app-category">
              <Select
                id="app-category"
                value={form.categoryId}
                onChange={(event) => update("categoryId", event.target.value)}
                required
              >
                <option value="">선택</option>
                {categories.data?.map((category) => (
                  <option key={category.id} value={category.id}>
                    {category.name}
                  </option>
                ))}
              </Select>
            </Field>
            <Field label="공개 범위" id="app-visibility">
              <Select
                id="app-visibility"
                value={form.visibility}
                onChange={(event) =>
                  update(
                    "visibility",
                    event.target.value as "public" | "private",
                  )
                }
              >
                <option value="public">Public</option>
                <option value="private">Private</option>
              </Select>
            </Field>
            <Field label="개발 언어" id="app-language">
              <Input
                id="app-language"
                value={form.language}
                onChange={(event) => update("language", event.target.value)}
                maxLength={60}
              />
            </Field>
            <Field label="Framework" id="app-framework">
              <Input
                id="app-framework"
                value={form.framework}
                onChange={(event) => update("framework", event.target.value)}
                maxLength={60}
              />
            </Field>
            <Field label="앱 버전" id="app-version">
              <Input
                id="app-version"
                value={form.version}
                onChange={(event) => update("version", event.target.value)}
                maxLength={40}
                placeholder="1.0.0"
              />
            </Field>
            <Field
              label="담당팀"
              id="app-team"
              help="비워두면 SSO 팀 정보를 사용합니다."
            >
              <Input
                id="app-team"
                value={form.team}
                onChange={(event) => update("team", event.target.value)}
                maxLength={120}
              />
            </Field>
            <Field label="태그" id="app-tags" help="쉼표로 구분">
              <Input
                id="app-tags"
                value={form.tags}
                onChange={(event) => update("tags", event.target.value)}
              />
            </Field>
            <Field
              label="Screenshot URL"
              id="app-screenshots"
              help="내부망에서 접근 가능한 URL을 쉼표로 구분"
            >
              <Input
                id="app-screenshots"
                value={form.screenshots}
                onChange={(event) => update("screenshots", event.target.value)}
              />
            </Field>
          </div>
          <Field label="상세 설명" id="app-description">
            <Textarea
              id="app-description"
              value={form.description}
              onChange={(event) => update("description", event.target.value)}
              required
            />
          </Field>
          <GuideDocumentsField draft={documents} idPrefix="app" />
          <div className="switch-row">
            <div>
              <strong>MCP 지원</strong>
              <div className="field-help">
                MCP client에서 사용할 수 있는 앱입니다.
              </div>
            </div>
            <Switch
              checked={form.supportsMcp}
              onChange={(value) => update("supportsMcp", value)}
              label="MCP 지원"
            />
          </div>
          <div className="switch-row">
            <div>
              <strong>API 지원</strong>
              <div className="field-help">REST API를 제공하는 앱입니다.</div>
            </div>
            <Switch
              checked={form.supportsApi}
              onChange={(value) => update("supportsApi", value)}
              label="API 지원"
            />
          </div>
          {save.error && (
            <div className="notice notice-danger mt-5" role="alert">
              {save.error.message}
            </div>
          )}
          <div className="form-actions mt-6">
            <Button variant="secondary" onClick={() => navigate(-1)}>
              취소
            </Button>
            <Button type="submit" disabled={save.isPending}>
              <Send size={18} />{" "}
              {save.isPending ? "저장 중…" : edit ? "변경 저장" : "등록"}
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
}
