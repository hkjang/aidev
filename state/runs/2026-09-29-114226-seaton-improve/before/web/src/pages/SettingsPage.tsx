import { useEffect, useMemo, useState } from "react";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  Divider,
  FormControl,
  FormControlLabel,
  Grid,
  Skeleton,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  Switch,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Tabs,
  TextField,
  Typography,
} from "@mui/material";
import SaveRounded from "@mui/icons-material/SaveRounded";
import LinkRounded from "@mui/icons-material/LinkRounded";
import CheckCircleRounded from "@mui/icons-material/CheckCircleRounded";
import WarningAmberRounded from "@mui/icons-material/WarningAmberRounded";
import RefreshRounded from "@mui/icons-material/RefreshRounded";
import ContentCopyRounded from "@mui/icons-material/ContentCopyRounded";
import { api, postJSON, putJSON } from "../api";
import { PageHeader } from "../components/AdminUI";
import {
  MAX_SNIPPET_BYTES,
  TRACKING_PROVIDERS,
  addAllowedHost,
  snippetBytes,
  trackingActive,
  trackingFields,
  usesSameOriginProxy,
} from "../lib/tracking";
import {
  mcpMetadataURL,
  mcpOAuthActive,
  mcpOAuthProblem,
  mcpResource,
} from "../lib/mcpOAuth";

type Setting = {
  key: string;
  value: string;
  secret: boolean;
  configured: boolean;
};
type Violation = {
  origin: string;
  directive: string;
  page: string;
  count: number;
  lastSeen: string;
  allowed: boolean;
};
const fields: Record<
  string,
  {
    key: string;
    label: string;
    help?: string;
    secret?: boolean;
    type?: string;
    multiline?: boolean;
  }[]
> = {
  general: [
    { key: "general.service_name", label: "서비스 이름" },
    { key: "general.company_name", label: "회사/조직명" },
    { key: "general.default_locale", label: "기본 언어" },
  ],
  oidc: [
    {
      key: "oidc.issuer_url",
      label: "Keycloak Issuer URL",
      help: "예: https://keycloak.intra/realms/company",
    },
    { key: "oidc.client_id", label: "Client ID" },
    { key: "oidc.client_secret", label: "Client Secret", secret: true },
    { key: "oidc.scopes", label: "Scopes" },
    { key: "oidc.admin_group", label: "시스템 관리자 그룹" },
    { key: "oidc.seat_manager_group", label: "좌석 관리자 그룹" },
  ],
  security: [
    {
      key: "security.session_hours",
      label: "세션 유효시간 (시간)",
      type: "number",
    },
    {
      key: "security.api_key_days",
      label: "개인 키 기본 유효기간 (일)",
      type: "number",
    },
    {
      key: "security.rotation_grace_hours",
      label: "키 회전 유예시간 (시간)",
      type: "number",
    },
  ],
  ai: [
    {
      key: "ai.confidence_threshold",
      label: "좌석 후보 최소 신뢰도",
      type: "number",
    },
    {
      key: "ai.auto_approve_threshold",
      label: "자동 승인 신뢰도",
      type: "number",
    },
    {
      key: "ai.vlm_base_url",
      label: "VLM 엔드포인트 (OpenAI 호환)",
      help: "예: http://vllm.intra:8000/v1 · 사내 주소만 사용하세요",
    },
    {
      key: "ai.vlm_model",
      label: "VLM 모델 이름",
      help: "예: qwen2.5-vl-7b-instruct",
    },
    { key: "ai.vlm_api_key", label: "VLM API 키", secret: true },
    {
      key: "ai.vlm_timeout_seconds",
      label: "VLM 응답 제한 (초)",
      type: "number",
      help: "10~600 · 큰 도면은 넉넉하게",
    },
    {
      key: "ai.vlm_max_image_side",
      label: "VLM 전송 이미지 최대 변 (px)",
      type: "number",
      help: "512~4096",
    },
    {
      key: "ai.vlm_max_seats",
      label: "VLM 좌석 상한",
      type: "number",
      help: "1~500",
    },
    {
      key: "ai.vlm_tiles",
      label: "VLM 타일 분할 (한 변)",
      type: "number",
      help: "1 권장 · Qwen2.5-VL 7B 실측에서는 분할이 오히려 정확도를 떨어뜨렸습니다",
    },
    {
      key: "ai.vlm_json_mode",
      label: "VLM JSON 강제 모드 (true/false)",
      help: "true 권장 · 모델에 따라 끄는 편이 인식률이 높을 수 있습니다",
    },
    {
      key: "ai.fusion_iou",
      label: "하이브리드 일치 판정 IoU",
      type: "number",
      help: "0.1~0.9 · 낮추면 더 관대하게 교차 검증합니다",
    },
  ],
  hr: [
    { key: "hr.api_url", label: "인사 시스템 API URL" },
    { key: "hr.api_token", label: "API Bearer Token", secret: true },
    {
      key: "hr.schedule",
      label: "동기화 일정 (Cron)",
      help: "기본: 매일 02:00",
    },
  ],
  // 방문 추적 칸은 provider 에 따라 달라서 trackingFields() 가 정한다.
  tracking: [],
};
export function SettingsPage() {
  const [items, setItems] = useState<Setting[]>([]),
    [values, setValues] = useState<Record<string, string>>({}),
    [tab, setTab] = useState("general"),
    [message, setMessage] = useState(""),
    [error, setError] = useState(""),
    [saving, setSaving] = useState(false),
    [currentPassword, setCurrentPassword] = useState(""),
    [testingVLM, setTestingVLM] = useState(false),
    [loading, setLoading] = useState(true),
    [violations, setViolations] = useState<Violation[]>([]),
    [violationsError, setViolationsError] = useState(""),
    [newPassword, setNewPassword] = useState("");

  const load = async () => {
    try {
      const data = await api<{ items: Setting[] }>("/api/v1/settings");
      setItems(data.items);
      setValues(Object.fromEntries(data.items.map((x) => [x.key, x.value])));
    } catch (e) {
      setError(e instanceof Error ? e.message : "설정을 불러오지 못했습니다");
    } finally {
      setLoading(false);
    }
  };
  useEffect(() => {
    void load();
  }, []);
  const configured = useMemo(
    () => Object.fromEntries(items.map((x) => [x.key, x.configured])),
    [items],
  );
  const engine = values["ai.engine"] || "cv";
  const trackingProvider = values["tracking.provider"] || "momento";
  const trackingOn = trackingActive(values);
  const mcpOn = mcpOAuthActive(values);
  const mcpResourceValue = mcpResource(
    values["mcp.oauth.resource"] ?? "",
    window.location.origin,
  );
  const mcpProblem = mcpOAuthProblem(values);
  const tabFields =
    tab === "tracking" ? trackingFields(trackingProvider) : fields[tab];
  const dirty = useMemo(
    () => items.some((item) => (values[item.key] ?? "") !== item.value),
    [items, values],
  );
  useEffect(() => {
    const warn = (event: BeforeUnloadEvent) => {
      if (!dirty) return;
      event.preventDefault();
    };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [dirty]);
  const save = async () => {
    setSaving(true);
    try {
      await putJSON("/api/v1/settings", { settings: values });
      setMessage("설정을 저장했습니다. 새 요청부터 즉시 적용됩니다.");
      await load();
      return true;
    } catch (e) {
      setError(e instanceof Error ? e.message : "저장하지 못했습니다");
      return false;
    } finally {
      setSaving(false);
    }
  };
  const test = async () => {
    try {
      if (!(await save())) return;
      const result = await api<{ issuer: string; redirectUri: string }>(
        "/api/v1/settings/oidc/test",
        { method: "POST" },
      );
      setMessage(
        `Keycloak Discovery 연결 성공 · Callback ${result.redirectUri}`,
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : "연결하지 못했습니다");
    }
  };
  // VLM 연결 시험은 합성 도면 한 장을 왕복시켜 주소·인증·응답 형식을 한 번에 검증한다.
  const testVLMConnection = async () => {
    setTestingVLM(true);
    try {
      if (!(await save())) return;
      const result = await api<{
        ok: boolean;
        message: string;
        elapsedMs?: number;
      }>("/api/v1/settings/ai/vlm/test", { method: "POST" });
      const elapsed = result.elapsedMs
        ? ` · 응답 ${(result.elapsedMs / 1000).toFixed(1)}초`
        : "";
      if (result.ok) setMessage(result.message + elapsed);
      else setError(result.message + elapsed);
    } catch (e) {
      setError(e instanceof Error ? e.message : "VLM에 연결하지 못했습니다");
    } finally {
      setTestingVLM(false);
    }
  };
  const syncNow = async () => {
    try {
      if (!(await save())) return;
      const result = await api<{ employees: number; organizations: number }>(
        "/api/v1/settings/hr/sync",
        { method: "POST" },
      );
      setMessage(
        `인사 동기화 완료 · 직원 ${result.employees}명, 조직 ${result.organizations}개`,
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : "동기화하지 못했습니다");
    }
  };
  // 브라우저가 CSP 로 막은 출처. 추적이 켜진 동안 서버가 신고를 모아 둔다.
  const loadViolations = async () => {
    try {
      const data = await api<{ items: Violation[] }>(
        "/api/v1/settings/tracking/violations",
      );
      setViolations(data.items);
      setViolationsError("");
    } catch (e) {
      setViolationsError(
        e instanceof Error ? e.message : "차단 기록을 불러오지 못했습니다",
      );
    }
  };
  useEffect(() => {
    if (tab === "tracking") void loadViolations();
  }, [tab]);
  const forgetViolations = async () => {
    try {
      await api<void>("/api/v1/settings/tracking/violations", {
        method: "DELETE",
      });
      await loadViolations();
      setMessage(
        "차단 기록을 비웠습니다. 화면을 새로 고치면 아직 막히는 것만 다시 쌓입니다.",
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : "기록을 비우지 못했습니다");
    }
  };
  // 막힌 출처를 한 번 눌러 허용 목록에 넣고 바로 저장한다. 저장하지 않은 다른
  // 변경도 함께 저장되므로 값을 먼저 합쳐 한 요청으로 보낸다.
  const allowOrigin = async (origin: string) => {
    const next = {
      ...values,
      "tracking.allowed_hosts": addAllowedHost(
        values["tracking.allowed_hosts"] ?? "",
        origin,
      ),
    };
    setValues(next);
    setSaving(true);
    try {
      await putJSON("/api/v1/settings", { settings: next });
      setMessage(
        `${origin} 을 허용 목록에 더했습니다. 새로 고친 화면부터 정책에 반영됩니다.`,
      );
      await load();
      await loadViolations();
    } catch (e) {
      setError(e instanceof Error ? e.message : "저장하지 못했습니다");
    } finally {
      setSaving(false);
    }
  };
  const changePassword = async () => {
    try {
      await postJSON("/api/v1/auth/password", { currentPassword, newPassword });
      setCurrentPassword("");
      setNewPassword("");
      setMessage(
        "관리자 비밀번호를 변경했습니다. 다른 세션은 로그아웃되었습니다.",
      );
    } catch (e) {
      setError(
        e instanceof Error ? e.message : "비밀번호를 변경하지 못했습니다",
      );
    }
  };
  const switchValue = (key: string) => (
    <FormControlLabel
      control={
        <Switch
          checked={values[key] === "true"}
          onChange={(e) =>
            setValues((v) => ({ ...v, [key]: String(e.target.checked) }))
          }
        />
      }
      label={
        key === "oidc.enabled"
          ? "Keycloak SSO 사용"
          : key === "auth.local_enabled"
            ? "로컬 관리자 로그인 허용"
            : key === "oidc.auto_provision"
              ? "SSO 사용자 자동 생성"
              : key === "oidc.auto_login"
                ? "Keycloak 세션이 있으면 자동 로그인"
                : key === "mcp.oauth.enabled"
                  ? "MCP 를 Keycloak 액세스 토큰으로도 열기"
                  : key === "tracking.enabled"
                    ? "방문 추적 사용"
                    : key === "tracking.include_admin"
                      ? "관리 화면(/admin, /profile)도 추적"
                      : key === "tracking.momento_proxy"
                        ? "같은 오리진 프록시(/momento) 사용"
                        : "자동 동기화 사용"
      }
    />
  );
  return (
    <Box sx={{ p: { xs: 2, md: 3 }, maxWidth: 1100, mx: "auto" }}>
      <PageHeader
        eyebrow="SYSTEM CONTROL"
        title="시스템 설정"
        description="최초 실행 환경변수는 3개뿐이며, 이후 SSO·인사·AI·보안 정책은 이 화면에서 운영합니다."
        actions={
          dirty ? (
            <Chip
              icon={<WarningAmberRounded />}
              color="warning"
              label="저장하지 않은 변경"
            />
          ) : (
            <Chip
              icon={<CheckCircleRounded />}
              color="success"
              variant="outlined"
              label="모든 변경 저장됨"
            />
          )
        }
      />
      {message && (
        <Alert severity="success" onClose={() => setMessage("")} sx={{ mb: 2 }}>
          {message}
        </Alert>
      )}
      {error && (
        <Alert severity="error" onClose={() => setError("")} sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}
      {loading ? (
        // 불러오기 전 빈 폼을 보여주면, 그 사이 입력한 값이 응답이 도착하는
        // 순간 덮어써진다. 값이 준비될 때까지 자리만 잡아 둔다.
        <Card>
          <CardContent sx={{ p: { xs: 2, md: 4 } }}>
            <Skeleton height={48} sx={{ mb: 2 }} />
            <Grid container spacing={2.5}>
              {[1, 2, 3, 4, 5, 6].map((item) => (
                <Grid key={item} size={{ xs: 12, md: 6 }}>
                  <Skeleton variant="rounded" height={56} />
                </Grid>
              ))}
            </Grid>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <Stack
            direction={{ xs: "column", sm: "row" }}
            spacing={1}
            sx={{ px: 2.5, pt: 2 }}
          >
            <Chip
              size="small"
              color={values["oidc.enabled"] === "true" ? "success" : "default"}
              variant="outlined"
              label={`Keycloak ${values["oidc.enabled"] === "true" ? "활성" : "비활성"}`}
            />
            <Chip
              size="small"
              color={
                values["hr.sync_enabled"] === "true" ? "success" : "default"
              }
              variant="outlined"
              label={`인사 동기화 ${values["hr.sync_enabled"] === "true" ? "활성" : "비활성"}`}
            />
            <Chip
              size="small"
              color={engine === "cv" ? "default" : "info"}
              variant="outlined"
              label={
                engine === "cv"
                  ? "오프라인 CV 엔진"
                  : engine === "vlm"
                    ? "VLM 엔진"
                    : "CV + VLM 하이브리드"
              }
            />
            <Chip
              size="small"
              color={trackingOn ? "success" : "default"}
              variant="outlined"
              label={`방문 추적 ${trackingOn ? "활성" : "비활성"}`}
            />
          </Stack>
          <Tabs
            value={tab}
            onChange={(_, v) => setTab(v)}
            variant="scrollable"
            sx={{ px: 2, borderBottom: "1px solid #E3EAEE" }}
          >
            <Tab value="general" label="일반" />
            <Tab value="oidc" label="Keycloak SSO" />
            <Tab value="security" label="보안 · 키" />
            <Tab value="ai" label="AI 분석" />
            <Tab value="hr" label="인사 연동" />
            <Tab value="tracking" label="방문 추적" />
          </Tabs>
          <CardContent sx={{ p: { xs: 2, md: 4 } }}>
            {tab === "oidc" && (
              <>
                <Stack
                  direction={{ xs: "column", sm: "row" }}
                  spacing={2}
                  mb={3}
                >
                  {switchValue("oidc.enabled")}
                  {switchValue("auth.local_enabled")}
                  {switchValue("oidc.auto_provision")}
                  {switchValue("oidc.auto_login")}
                </Stack>
                <Alert severity="info" sx={{ mb: 3 }}>
                  Keycloak Client의 Valid Redirect URI에{" "}
                  <strong>
                    {window.location.origin}/api/v1/auth/oidc/callback
                  </strong>{" "}
                  을 등록하세요. Client authentication은 ON으로 설정합니다.
                </Alert>
              </>
            )}
            {tab === "hr" && <Box mb={3}>{switchValue("hr.sync_enabled")}</Box>}
            {tab === "ai" && (
              <Box mb={3}>
                <FormControl sx={{ minWidth: 280 }}>
                  <InputLabel id="ai-engine-label">좌석 인식 엔진</InputLabel>
                  <Select
                    labelId="ai-engine-label"
                    label="좌석 인식 엔진"
                    value={engine}
                    onChange={(e) =>
                      setValues((v) => ({ ...v, "ai.engine": e.target.value }))
                    }
                  >
                    <MenuItem value="cv">
                      오프라인 CV 전용 (외부 통신 없음)
                    </MenuItem>
                    <MenuItem value="vlm">비전 모델(VLM) 전용</MenuItem>
                    <MenuItem value="hybrid">
                      하이브리드 · CV + VLM 교차 검증
                    </MenuItem>
                  </Select>
                </FormControl>
              </Box>
            )}
            {tab === "tracking" && (
              <Box mb={3}>
                <Stack
                  direction={{ xs: "column", sm: "row" }}
                  spacing={2}
                  mb={2}
                  flexWrap="wrap"
                >
                  {switchValue("tracking.enabled")}
                  {switchValue("tracking.include_admin")}
                  {trackingProvider === "momento" &&
                    switchValue("tracking.momento_proxy")}
                </Stack>
                <Stack direction={{ xs: "column", sm: "row" }} spacing={2}>
                  <FormControl sx={{ minWidth: 280 }}>
                    <InputLabel id="tracking-provider-label">
                      추적 도구
                    </InputLabel>
                    <Select
                      labelId="tracking-provider-label"
                      label="추적 도구"
                      value={trackingProvider}
                      onChange={(e) =>
                        setValues((v) => ({
                          ...v,
                          "tracking.provider": e.target.value,
                        }))
                      }
                    >
                      {TRACKING_PROVIDERS.map((p) => (
                        <MenuItem key={p.value} value={p.value}>
                          {p.label}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                  <FormControl sx={{ minWidth: 200 }}>
                    <InputLabel id="tracking-placement-label">
                      넣는 자리
                    </InputLabel>
                    <Select
                      labelId="tracking-placement-label"
                      label="넣는 자리"
                      value={values["tracking.placement"] || "head"}
                      onChange={(e) =>
                        setValues((v) => ({
                          ...v,
                          "tracking.placement": e.target.value,
                        }))
                      }
                    >
                      <MenuItem value="head">&lt;head&gt; 끝</MenuItem>
                      <MenuItem value="body">&lt;body&gt; 끝</MenuItem>
                    </Select>
                  </FormControl>
                </Stack>
              </Box>
            )}
            <Grid container spacing={2.5}>
              {tabFields.map((f) => (
                <Grid key={f.key} size={{ xs: 12, md: f.multiline ? 12 : 6 }}>
                  <TextField
                    fullWidth
                    label={f.label}
                    type={f.secret ? "password" : f.type || "text"}
                    multiline={f.multiline}
                    minRows={f.multiline ? 6 : undefined}
                    value={values[f.key] ?? ""}
                    onChange={(e) =>
                      setValues((v) => ({ ...v, [f.key]: e.target.value }))
                    }
                    error={
                      f.key === "tracking.custom_snippet" &&
                      snippetBytes(values[f.key] ?? "") > MAX_SNIPPET_BYTES
                    }
                    helperText={
                      f.key === "tracking.custom_snippet"
                        ? `${f.help} · 지금 ${snippetBytes(values[f.key] ?? "").toLocaleString()}바이트`
                        : f.help ||
                          (f.secret && configured[f.key]
                            ? "암호화되어 저장되어 있습니다. 변경할 때만 새 값을 입력하세요."
                            : " ")
                    }
                    slotProps={
                      f.multiline
                        ? { htmlInput: { style: { fontFamily: "monospace" } } }
                        : undefined
                    }
                  />
                </Grid>
              ))}
              {tab === "tracking" && trackingProvider !== "none" && (
                <Grid size={{ xs: 12 }}>
                  <TextField
                    fullWidth
                    label="추가로 허용할 출처"
                    value={values["tracking.allowed_hosts"] ?? ""}
                    onChange={(e) =>
                      setValues((v) => ({
                        ...v,
                        "tracking.allowed_hosts": e.target.value,
                      }))
                    }
                    helperText="쉼표로 구분한 https://host[:port] 목록. 스니펫에서 자동으로 읽지 못한 출처를 여기에 더합니다. 아래 차단 기록의 '허용에 추가' 를 누르면 자동으로 채워집니다."
                  />
                </Grid>
              )}
            </Grid>
            {tab === "oidc" && (
              <Box sx={{ mt: 4 }} data-testid="mcp-oauth-card">
                <Divider sx={{ mb: 3 }} />
                <Typography variant="h6">MCP SSO(OAuth) 인증</Typography>
                <Typography
                  variant="body2"
                  color="text.secondary"
                  sx={{ mb: 2 }}
                >
                  켜면 MCP 클라이언트(Claude, Cursor 등)에 개인 키 없이 MCP 주소
                  하나만 주면 됩니다. 클라이언트가 아래 메타데이터를 읽어
                  Keycloak 에서 스스로 로그인하고, 이 서버는 받은 액세스 토큰의
                  서명·발급자·만료·대상을 요청마다 검사합니다. 계정은 만들지
                  않으며 웹으로 한 번 로그인한 활성 SSO 사용자만 통과합니다.
                  REST API 는 지금처럼 키와 세션만 받습니다.
                </Typography>
                <Box mb={2}>{switchValue("mcp.oauth.enabled")}</Box>
                <Grid container spacing={2.5}>
                  <Grid size={{ xs: 12, md: 6 }}>
                    <TextField
                      fullWidth
                      label="리소스 식별자 (resource)"
                      placeholder={`${window.location.origin}/mcp`}
                      value={values["mcp.oauth.resource"] ?? ""}
                      onChange={(e) =>
                        setValues((v) => ({
                          ...v,
                          "mcp.oauth.resource": e.target.value,
                        }))
                      }
                      helperText="클라이언트가 실제로 접속하는 공개 주소 + /mcp. 켜려면 필수입니다 — 요청 주소로 대신 만들지 않습니다(그러면 Host 헤더가 허용 대상을 정하게 됩니다). Keycloak Audience 매퍼에 넣는 값이기도 합니다."
                    />
                  </Grid>
                  <Grid size={{ xs: 12, md: 6 }}>
                    <TextField
                      fullWidth
                      label="허용 대상 (aud 또는 azp)"
                      value={values["mcp.oauth.audience"] ?? ""}
                      onChange={(e) =>
                        setValues((v) => ({
                          ...v,
                          "mcp.oauth.audience": e.target.value,
                        }))
                      }
                      helperText="공백으로 구분한 Keycloak 클라이언트 ID 목록. Audience 매퍼 없이 쓰는 호환 경로 — Keycloak 26 은 클라이언트 ID 를 aud 가 아니라 azp 에 담습니다. 거절 메시지가 적을 값을 알려 줍니다."
                    />
                  </Grid>
                  <Grid size={{ xs: 12, md: 6 }}>
                    <TextField
                      fullWidth
                      label="SSO 토큰에 주는 범위"
                      value={values["mcp.oauth.scopes"] ?? ""}
                      onChange={(e) =>
                        setValues((v) => ({
                          ...v,
                          "mcp.oauth.scopes": e.target.value,
                        }))
                      }
                      error={mcpProblem !== ""}
                      helperText={
                        mcpProblem ||
                        "read, write, mcp 중에서 공백으로 구분. 토큰의 scope 가 아니라 이 값이 천장입니다. 배정 도구(assign_seat)까지 열려면 write 를 더합니다."
                      }
                    />
                  </Grid>
                  <Grid size={{ xs: 12, md: 6 }}>
                    <Stack spacing={1}>
                      <Stack direction="row" spacing={1} alignItems="center">
                        <TextField
                          fullWidth
                          size="small"
                          label="MCP 주소 (클라이언트에 줄 값)"
                          value={mcpResourceValue}
                          slotProps={{ input: { readOnly: true } }}
                        />
                        <Button
                          size="small"
                          startIcon={<ContentCopyRounded />}
                          onClick={() =>
                            void navigator.clipboard.writeText(mcpResourceValue)
                          }
                        >
                          복사
                        </Button>
                      </Stack>
                      <Stack direction="row" spacing={1} alignItems="center">
                        <TextField
                          fullWidth
                          size="small"
                          label="메타데이터 주소"
                          value={mcpMetadataURL(mcpResourceValue)}
                          slotProps={{ input: { readOnly: true } }}
                        />
                        <Button
                          size="small"
                          startIcon={<ContentCopyRounded />}
                          onClick={() =>
                            void navigator.clipboard.writeText(
                              mcpMetadataURL(mcpResourceValue),
                            )
                          }
                        >
                          복사
                        </Button>
                      </Stack>
                    </Stack>
                  </Grid>
                </Grid>
                <Alert severity={mcpOn ? "success" : "info"} sx={{ mt: 2 }}>
                  {mcpOn
                    ? "저장하면 /mcp 의 401 에 메타데이터 주소가 붙고 Keycloak 액세스 토큰을 받습니다. Keycloak 에는 웹 로그인과 다른 공개(public) 클라이언트를 만들고 PKCE S256, Standard Flow 만 켭니다 — 관리자 가이드 §3.3 을 따르세요."
                    : values["mcp.oauth.enabled"] === "true"
                      ? "Keycloak Issuer URL 또는 리소스 식별자가 비어 있어 켜도 꺼진 것처럼 동작합니다."
                      : "꺼져 있습니다. MCP 는 개인 API 키로만 열리고 메타데이터 주소는 404 입니다."}
                </Alert>
              </Box>
            )}
            {tab === "tracking" && (
              <>
                <Alert
                  severity={trackingOn ? "success" : "info"}
                  sx={{ mt: 2 }}
                >
                  {trackingProvider === "none"
                    ? "추적 도구가 없으면 켜도 아무것도 붙지 않습니다."
                    : usesSameOriginProxy(values)
                      ? "Momento 로더와 수집 요청이 이 서비스의 /momento 경로를 거쳐 수집기로 갑니다. 외부 출처가 정책(CSP)에 등장하지 않으므로 정책을 바꿀 수 없는 설치에서도 동작합니다. Momento 사이트의 허용 도메인에 이 서비스 주소를 등록하세요."
                      : "브라우저가 수집기에 직접 연결합니다. 정책(CSP)의 script-src·connect-src·img-src 에 수집기 출처가 자동으로 더해지고, 스니펫의 모든 <script> 에는 요청마다 다른 nonce 가 붙습니다. 'unsafe-inline' 은 쓰지 않습니다."}
                  {trackingOn
                    ? " 저장하면 다음에 여는 화면부터 붙습니다."
                    : values["tracking.enabled"] === "true"
                      ? " 필수 칸이 비어 있어 아직 붙지 않습니다."
                      : ""}
                </Alert>
                <Box sx={{ mt: 3 }}>
                  <Stack
                    direction="row"
                    justifyContent="space-between"
                    alignItems="center"
                    mb={1}
                  >
                    <Box>
                      <Typography variant="h6">정책이 차단한 출처</Typography>
                      <Typography variant="body2" color="text.secondary">
                        추적이 켜진 동안 브라우저가 신고한 것입니다. 스니펫이
                        조용히 멎었다면 여기에 이유가 있습니다. 재시작하면
                        비워집니다.
                      </Typography>
                    </Box>
                    <Stack direction="row" spacing={1}>
                      <Button
                        size="small"
                        startIcon={<RefreshRounded />}
                        onClick={() => void loadViolations()}
                      >
                        새로 고침
                      </Button>
                      <Button
                        size="small"
                        disabled={violations.length === 0}
                        onClick={() => void forgetViolations()}
                      >
                        기록 비우기
                      </Button>
                    </Stack>
                  </Stack>
                  {violationsError && (
                    <Alert severity="warning" sx={{ mb: 1 }}>
                      {violationsError}
                    </Alert>
                  )}
                  {violations.length === 0 ? (
                    <Typography variant="body2" color="text.secondary">
                      차단된 출처가 없습니다.
                    </Typography>
                  ) : (
                    <Table size="small" aria-label="정책이 차단한 출처">
                      <TableHead>
                        <TableRow>
                          <TableCell>출처</TableCell>
                          <TableCell>지시어</TableCell>
                          <TableCell>화면</TableCell>
                          <TableCell align="right">횟수</TableCell>
                          <TableCell>마지막</TableCell>
                          <TableCell />
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {violations.map((v) => (
                          <TableRow key={`${v.directive} ${v.origin}`}>
                            <TableCell sx={{ fontFamily: "monospace" }}>
                              {v.origin}
                            </TableCell>
                            <TableCell>{v.directive}</TableCell>
                            <TableCell>{v.page}</TableCell>
                            <TableCell align="right">{v.count}</TableCell>
                            <TableCell>
                              {new Date(v.lastSeen).toLocaleString("ko-KR")}
                            </TableCell>
                            <TableCell align="right">
                              {v.allowed ? (
                                <Chip
                                  size="small"
                                  color="success"
                                  variant="outlined"
                                  label="허용됨"
                                />
                              ) : (
                                <Button
                                  size="small"
                                  variant="outlined"
                                  disabled={saving}
                                  onClick={() => void allowOrigin(v.origin)}
                                >
                                  허용에 추가
                                </Button>
                              )}
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  )}
                </Box>
              </>
            )}
            {tab === "ai" && (
              <Alert
                severity={engine === "cv" ? "info" : "warning"}
                sx={{ mt: 2 }}
              >
                {engine === "cv"
                  ? "오프라인 CV 엔진만 사용합니다. 외부 통신이 전혀 없고 결과가 항상 재현됩니다."
                  : engine === "vlm"
                    ? "비전 모델 판독 결과는 보정되지 않은 값이므로 신뢰도가 자동 승인선을 넘지 않도록 제한되며, 모든 좌석이 검토 대상으로 남습니다. VLM 호출이 실패하면 CV 결과로 자동 대체됩니다."
                    : "CV와 VLM 결과를 IoU로 교차 검증합니다. 두 엔진이 합의한 좌석만 자동 승인 구간으로 올라가고, 한쪽만 찾은 좌석은 검토 대상이 됩니다. VLM이 실패해도 CV 결과로 분석은 완료됩니다. 도면 종류에 따라 CV 단독보다 나쁠 수 있으니 대표 도면으로 두 엔진을 비교한 뒤 선택하세요."}
              </Alert>
            )}
            {tab === "security" && (
              <Box sx={{ mt: 3 }}>
                <Divider sx={{ mb: 3 }} />
                <Typography variant="h6">내 로컬 관리자 비밀번호</Typography>
                <Typography
                  variant="body2"
                  color="text.secondary"
                  sx={{ mb: 2 }}
                >
                  SSO 계정은 Keycloak에서 변경합니다. 변경 시 현재 세션을 제외한
                  다른 세션이 종료됩니다.
                </Typography>
                <Stack direction={{ xs: "column", sm: "row" }} spacing={2}>
                  <TextField
                    type="password"
                    label="현재 비밀번호"
                    value={currentPassword}
                    onChange={(e) => setCurrentPassword(e.target.value)}
                  />
                  <TextField
                    type="password"
                    label="새 비밀번호 (12자 이상)"
                    value={newPassword}
                    onChange={(e) => setNewPassword(e.target.value)}
                  />
                  <Button
                    variant="outlined"
                    disabled={!currentPassword || newPassword.length < 12}
                    onClick={() => void changePassword()}
                  >
                    비밀번호 변경
                  </Button>
                </Stack>
              </Box>
            )}
            <Divider sx={{ my: 3 }} />
            <Stack
              direction={{ xs: "column-reverse", sm: "row" }}
              justifyContent="flex-end"
              spacing={1}
              sx={{
                position: "sticky",
                bottom: 0,
                bgcolor: "background.paper",
                py: 1,
                zIndex: 1,
              }}
            >
              {tab === "hr" && (
                <Button
                  startIcon={<LinkRounded />}
                  onClick={() => void syncNow()}
                >
                  저장 후 지금 동기화
                </Button>
              )}
              {tab === "oidc" && (
                <Button startIcon={<LinkRounded />} onClick={() => void test()}>
                  저장 후 연결 테스트
                </Button>
              )}
              {tab === "ai" && engine !== "cv" && (
                <Button
                  startIcon={<LinkRounded />}
                  disabled={testingVLM}
                  onClick={() => void testVLMConnection()}
                >
                  {testingVLM ? "시험 도면 판독 중…" : "저장 후 VLM 연결 시험"}
                </Button>
              )}
              <Button
                variant="contained"
                startIcon={<SaveRounded />}
                disabled={!dirty || saving}
                onClick={() => void save()}
              >
                {saving ? "저장 중…" : "설정 저장"}
              </Button>
            </Stack>
          </CardContent>
        </Card>
      )}
      <Stack direction="row" spacing={1} mt={2}>
        <Chip label="비밀값 AES-256-GCM 암호화" variant="outlined" />
        <Chip label="설정 변경 감사로그" variant="outlined" />
      </Stack>
    </Box>
  );
}
