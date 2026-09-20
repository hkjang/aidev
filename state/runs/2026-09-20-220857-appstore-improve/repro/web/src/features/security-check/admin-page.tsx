import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, PlugZap, Save } from "lucide-react";
import { useEffect, useState } from "react";
import { api } from "../../lib/api";
import { formatDateTime } from "../../lib/utils";
import {
  Button,
  Card,
  ErrorState,
  Field,
  Input,
  LoadingState,
  PageHeader,
  Switch,
} from "../../components/ui";

interface FormState {
  enabled: boolean;
  baseUrl: string;
  apiKey: string;
  timeoutSeconds: string;
}

export function AdminSecurityCheckPage() {
  const client = useQueryClient();
  const settings = useQuery({
    queryKey: ["admin", "security-check"],
    queryFn: ({ signal }) => api.securityCheckSettings(signal),
  });
  const [form, setForm] = useState<FormState>({
    enabled: false,
    baseUrl: "",
    apiKey: "",
    timeoutSeconds: "10",
  });
  useEffect(() => {
    if (!settings.data) return;
    setForm({
      enabled: settings.data.enabled,
      baseUrl: settings.data.baseUrl,
      // The stored key never comes back; an empty box means "keep it".
      apiKey: "",
      timeoutSeconds: String(settings.data.timeoutSeconds || 10),
    });
  }, [settings.data]);

  const save = useMutation({
    mutationFn: () =>
      api.updateSecurityCheckSettings({
        enabled: form.enabled,
        baseUrl: form.baseUrl.trim(),
        apiKey: form.apiKey.trim() || undefined,
        timeoutSeconds: Number(form.timeoutSeconds) || 10,
      }),
    onSuccess: async (saved) => {
      setForm((current) => ({ ...current, apiKey: "" }));
      client.setQueryData(["admin", "security-check"], saved);
      await client.invalidateQueries({ queryKey: ["public-config"] });
    },
  });
  const test = useMutation({ mutationFn: () => api.testSecurityCheck() });
  const set = <K extends keyof FormState>(key: K, value: FormState[K]) =>
    setForm((current) => ({ ...current, [key]: value }));

  if (settings.isPending)
    return (
      <div className="page">
        <LoadingState />
      </div>
    );
  if (settings.error)
    return (
      <div className="page">
        <ErrorState
          error={settings.error}
          retry={() => void settings.refetch()}
        />
      </div>
    );

  return (
    <div className="page">
      <PageHeader
        eyebrow="Security review"
        title="보안 심의 연동"
        description="SecCheck에서 최종 승인된 심의가 있어야 앱이 검토 대기로 넘어가도록 합니다. 등록된 API Key는 심의 조회에만 쓰입니다."
      />
      <Card className="form-section">
        <div className="switch-row">
          <div>
            <strong>보안 심의 필수</strong>
            <div className="field-help">
              켜면 새 앱 등록과 게시가 SecCheck 최종 승인 확인을 거칩니다. 이미
              게시된 앱은 내려가지 않고, 다음에 수정할 때부터 적용됩니다.
            </div>
          </div>
          <Switch
            checked={form.enabled}
            onChange={(value) => set("enabled", value)}
            label="보안 심의 필수"
          />
        </div>
        <div className="form-grid">
          <Field
            label="SecCheck 주소"
            id="security-check-url"
            help="예: https://seccheck.example.internal"
          >
            <Input
              id="security-check-url"
              type="url"
              value={form.baseUrl}
              onChange={(event) => set("baseUrl", event.target.value)}
              placeholder="https://seccheck.example.internal"
              maxLength={2048}
            />
          </Field>
          <Field
            label="API Key"
            id="security-check-key"
            help={
              settings.data?.apiKeySet
                ? "설정됨 · 비워 두면 그대로 유지합니다. SecCheck에서 AUDITOR 또는 SECURITY_REVIEWER 역할의 read 키를 발급하세요."
                : "SecCheck에서 AUDITOR 또는 SECURITY_REVIEWER 역할의 read 키를 발급해 입력하세요."
            }
          >
            <Input
              id="security-check-key"
              type="password"
              value={form.apiKey}
              autoComplete="off"
              onChange={(event) => set("apiKey", event.target.value)}
              placeholder={settings.data?.apiKeySet ? "설정됨" : ""}
              maxLength={4096}
            />
          </Field>
          <Field
            label="연결 제한 시간(초)"
            id="security-check-timeout"
            help="1~60초"
          >
            <Input
              id="security-check-timeout"
              type="number"
              min={1}
              max={60}
              value={form.timeoutSeconds}
              onChange={(event) => set("timeoutSeconds", event.target.value)}
            />
          </Field>
        </div>
        {(save.error || test.error) && (
          <div className="notice notice-danger mt-5" role="alert">
            {(save.error ?? test.error)?.message}
          </div>
        )}
        {save.isSuccess && !save.isPending && (
          <div className="notice mt-5">
            <CheckCircle2 size={19} aria-hidden="true" /> 연동 설정을
            저장했습니다.
          </div>
        )}
        {test.isSuccess && (
          <div className="notice notice-positive mt-5">
            <CheckCircle2 size={19} aria-hidden="true" /> SecCheck에
            연결했습니다 — {test.data?.displayName || test.data?.account} (
            {test.data?.roles?.join(", ")})
          </div>
        )}
        <div className="form-actions mt-6">
          <Button
            variant="secondary"
            disabled={test.isPending || !settings.data?.apiKeySet}
            onClick={() => test.mutate()}
          >
            <PlugZap size={17} /> {test.isPending ? "확인 중…" : "연결 테스트"}
          </Button>
          <Button disabled={save.isPending} onClick={() => save.mutate()}>
            <Save size={18} /> {save.isPending ? "저장 중…" : "설정 저장"}
          </Button>
        </div>
        <p className="field-help mt-5">
          마지막 수정:{" "}
          {settings.data?.updatedAt
            ? formatDateTime(settings.data.updatedAt)
            : "—"}
        </p>
      </Card>
    </div>
  );
}
