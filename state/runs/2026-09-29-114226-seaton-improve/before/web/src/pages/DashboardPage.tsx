import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  Divider,
  Grid,
  IconButton,
  LinearProgress,
  Paper,
  Stack,
  Tab,
  Tabs,
  Tooltip,
  Typography,
} from "@mui/material";
import CheckCircleRounded from "@mui/icons-material/CheckCircleRounded";
import PersonOffRounded from "@mui/icons-material/PersonOffRounded";
import EventSeatRounded from "@mui/icons-material/EventSeatRounded";
import AutoAwesomeRounded from "@mui/icons-material/AutoAwesomeRounded";
import GroupsRounded from "@mui/icons-material/GroupsRounded";
import DoneAllRounded from "@mui/icons-material/DoneAllRounded";
import ArrowForwardRounded from "@mui/icons-material/ArrowForwardRounded";
import RefreshRounded from "@mui/icons-material/RefreshRounded";
import ApartmentRounded from "@mui/icons-material/ApartmentRounded";
import CloudSyncRounded from "@mui/icons-material/CloudSyncRounded";
import MapRounded from "@mui/icons-material/MapRounded";
import OpenInNewRounded from "@mui/icons-material/OpenInNewRounded";
import { useNavigate } from "react-router-dom";
import { api } from "../api";
import {
  LoadingCards,
  MetricCard,
  PageHeader,
  SeverityChip,
} from "../components/AdminUI";

type Counts = {
  unassignedEmployees: number;
  unusedSeats: number;
  retiredAssignments: number;
  organizationMismatch: number;
  lowConfidenceSeats: number;
  actionRequired: number;
  totalEmployees: number;
  assignedEmployees: number;
  totalSeats: number;
  assignedSeats: number;
  mapsInReview: number;
};
type Dashboard = {
  counts: Counts;
  readiness: Record<string, boolean>;
  integration: { lastSyncStatus?: string; lastSyncAt?: string };
};
type Issue = {
  id: string;
  kind: string;
  severity: string;
  title: string;
  description: string;
  employeeNo?: string;
  employeeName?: string;
  organizationName?: string;
  seatNo?: string;
  floorName?: string;
  confidence?: number;
  occurredAt?: string;
  action: string;
};
type History = {
  id: string;
  changedAt: string;
  employeeName: string;
  previousSeat: string;
  newSeat: string;
  reason: string;
};

const issueTabs = [
  { value: "", label: "전체" },
  { value: "unassigned_employee", label: "미배정" },
  { value: "retired_assignment", label: "퇴직자" },
  { value: "organization_mismatch", label: "조직 불일치" },
  { value: "low_confidence", label: "AI 확인" },
];

export function DashboardPage() {
  const navigate = useNavigate();
  const [data, setData] = useState<Dashboard | null>(null);
  const [issues, setIssues] = useState<Issue[]>([]);
  const [history, setHistory] = useState<History[]>([]);
  const [tab, setTab] = useState("");
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  const load = useCallback(async () => {
    setError("");
    try {
      const suffix = tab ? `?kind=${tab}` : "";
      const [dashboard, issueData, historyData] = await Promise.all([
        api<Dashboard>("/api/v1/dashboard"),
        api<{ items: Issue[] }>(`/api/v1/dashboard/issues${suffix}`),
        api<{ items: History[] }>("/api/v1/seat-history?limit=6"),
      ]);
      setData(dashboard);
      setIssues(issueData.items);
      setHistory(historyData.items);
    } catch (e) {
      setError(
        e instanceof Error ? e.message : "운영 현황을 불러오지 못했습니다",
      );
    }
  }, [tab]);
  useEffect(() => {
    void load();
  }, [load]);

  const assignmentRate = data?.counts.totalEmployees
    ? Math.round(
        (data.counts.assignedEmployees / data.counts.totalEmployees) * 100,
      )
    : 0;
  const occupancyRate = data?.counts.totalSeats
    ? Math.round((data.counts.assignedSeats / data.counts.totalSeats) * 100)
    : 0;
  const readinessItems = useMemo(
    () =>
      data
        ? [
            { key: "hasBuilding", label: "사업장 등록", route: "/admin/maps" },
            { key: "hasFloor", label: "층 구성", route: "/admin/maps" },
            {
              key: "hasPublishedMap",
              label: "좌석맵 게시",
              route: "/admin/maps",
            },
            {
              key: "hasEmployees",
              label: "직원 데이터",
              route: "/admin/employees",
            },
            {
              key: "oidcEnabled",
              label: "Keycloak SSO",
              route: "/admin/settings",
            },
            {
              key: "hrEnabled",
              label: "인사 자동 동기화",
              route: "/admin/settings",
            },
          ]
        : [],
    [data],
  );
  const readinessDone = readinessItems.filter(
    (item) => data?.readiness[item.key],
  ).length;

  const resolve = async (issue: Issue) => {
    if (issue.action === "assign") {
      navigate(
        `/?q=${encodeURIComponent(issue.employeeNo ?? issue.employeeName ?? "")}`,
      );
      return;
    }
    const kind = issue.kind.replaceAll("_", "-");
    setBusy(issue.id);
    try {
      await api(`/api/v1/dashboard/issues/${kind}/${issue.id}/resolve`, {
        method: "POST",
      });
      setMessage(`${issue.title} 항목을 처리했습니다.`);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "항목을 처리하지 못했습니다");
    } finally {
      setBusy("");
    }
  };
  const resolveRetired = async () => {
    setBusy("all-retired");
    try {
      const result = await api<{ resolved: number }>(
        "/api/v1/dashboard/issues/retired-assignment/resolve-all",
        { method: "POST" },
      );
      setMessage(`퇴직자 좌석 ${result.resolved}건을 해제했습니다.`);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "일괄 처리하지 못했습니다");
    } finally {
      setBusy("");
    }
  };

  if (!data && !error)
    return (
      <Box sx={{ p: { xs: 2, md: 3 }, maxWidth: 1500, mx: "auto" }}>
        <LoadingCards />
      </Box>
    );
  return (
    <Box sx={{ p: { xs: 2, md: 3 }, maxWidth: 1500, mx: "auto" }}>
      <PageHeader
        eyebrow="OPERATIONS"
        title="처리필요"
        description="예외를 발견하고 판단하고 처리하는 과정을 한 화면에서 끝냅니다."
        actions={
          <Tooltip title="새로고침">
            <IconButton onClick={() => void load()}>
              <RefreshRounded />
            </IconButton>
          </Tooltip>
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
      {data && (
        <>
          <Card
            sx={{
              mb: 2.5,
              overflow: "hidden",
              color: "white",
              background: data.counts.actionRequired
                ? "linear-gradient(125deg,#102E3F 0%,#075D69 65%,#087E8B 100%)"
                : "linear-gradient(125deg,#143D33,#2E9D67)",
            }}
          >
            <CardContent
              sx={{
                p: { xs: 3, md: 4 },
                "&:last-child": { pb: { xs: 3, md: 4 } },
              }}
            >
              <Stack
                direction={{ xs: "column", md: "row" }}
                justifyContent="space-between"
                alignItems={{ md: "center" }}
                spacing={3}
              >
                <Box>
                  <Typography
                    variant="overline"
                    sx={{ color: "rgba(255,255,255,.65)", letterSpacing: 1.5 }}
                  >
                    TODAY&apos;S FOCUS
                  </Typography>
                  <Stack direction="row" alignItems="baseline" spacing={1}>
                    <Typography
                      sx={{
                        fontSize: { xs: 48, md: 64 },
                        lineHeight: 1,
                        fontWeight: 850,
                        letterSpacing: "-.06em",
                      }}
                    >
                      {data.counts.actionRequired}
                    </Typography>
                    <Typography variant="h6">건의 확인이 필요합니다</Typography>
                  </Stack>
                  <Typography sx={{ mt: 1.2, color: "rgba(255,255,255,.7)" }}>
                    {data.counts.actionRequired
                      ? "긴급 항목부터 처리하면 직원 검색과 좌석 데이터가 즉시 정상화됩니다."
                      : "모든 좌석과 직원 데이터가 정상 상태입니다."}
                  </Typography>
                </Box>
                {data.counts.retiredAssignments > 0 ? (
                  <Button
                    variant="contained"
                    color="secondary"
                    startIcon={<DoneAllRounded />}
                    disabled={busy === "all-retired"}
                    onClick={() => void resolveRetired()}
                  >
                    퇴직자 {data.counts.retiredAssignments}건 일괄 해제
                  </Button>
                ) : (
                  <CheckCircleRounded
                    sx={{ fontSize: 64, color: "rgba(255,255,255,.8)" }}
                  />
                )}
              </Stack>
            </CardContent>
          </Card>
          <Grid container spacing={2} sx={{ mb: 2.5 }}>
            <Grid size={{ xs: 12, sm: 6, lg: 3 }}>
              <MetricCard
                label="직원 배정률"
                value={`${assignmentRate}%`}
                helper={`${data.counts.assignedEmployees} / ${data.counts.totalEmployees}명 배정`}
                progress={assignmentRate}
                icon={<GroupsRounded />}
              />
            </Grid>
            <Grid size={{ xs: 12, sm: 6, lg: 3 }}>
              <MetricCard
                label="좌석 점유율"
                value={`${occupancyRate}%`}
                helper={`${data.counts.unusedSeats}석 사용 가능`}
                progress={occupancyRate}
                tone="#3478C8"
                icon={<EventSeatRounded />}
              />
            </Grid>
            <Grid size={{ xs: 12, sm: 6, lg: 3 }}>
              <MetricCard
                label="미배정 직원"
                value={data.counts.unassignedEmployees}
                helper="좌석 배정이 필요한 재직자"
                tone="#E79418"
                icon={<PersonOffRounded />}
              />
            </Grid>
            <Grid size={{ xs: 12, sm: 6, lg: 3 }}>
              <MetricCard
                label="AI 검토 대기"
                value={data.counts.lowConfidenceSeats}
                helper={`${data.counts.mapsInReview}개 도면이 검토 단계`}
                tone="#8E5FC7"
                icon={<AutoAwesomeRounded />}
              />
            </Grid>
          </Grid>
          <Grid container spacing={2.5}>
            <Grid size={{ xs: 12, lg: 8 }}>
              <Paper sx={{ overflow: "hidden" }}>
                <Box sx={{ px: 2.5, pt: 2.2 }}>
                  <Stack
                    direction="row"
                    justifyContent="space-between"
                    alignItems="center"
                  >
                    <Box>
                      <Typography variant="h6">작업 큐</Typography>
                      <Typography variant="body2" color="text.secondary">
                        {issues.length}개 항목
                      </Typography>
                    </Box>
                    <Chip
                      size="small"
                      label="자동 새로고침 대신 명시적 처리"
                      variant="outlined"
                    />
                  </Stack>
                </Box>
                <Tabs
                  value={tab}
                  onChange={(_, value) => setTab(value)}
                  variant="scrollable"
                  sx={{
                    px: 1.5,
                    borderBottom: "1px solid",
                    borderColor: "divider",
                  }}
                >
                  {issueTabs.map((item) => (
                    <Tab
                      key={item.value}
                      value={item.value}
                      label={item.label}
                    />
                  ))}
                </Tabs>
                {issues.length === 0 ? (
                  <Stack alignItems="center" spacing={1} sx={{ py: 7 }}>
                    <CheckCircleRounded color="success" sx={{ fontSize: 45 }} />
                    <Typography fontWeight={750}>
                      선택한 유형의 처리 항목이 없습니다.
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      정상 데이터는 작업 큐에 표시하지 않습니다.
                    </Typography>
                  </Stack>
                ) : (
                  issues.map((issue, index) => (
                    <Box key={`${issue.kind}-${issue.id}`}>
                      <Stack
                        direction={{ xs: "column", sm: "row" }}
                        alignItems={{ sm: "center" }}
                        spacing={2}
                        sx={{ px: 2.5, py: 2 }}
                      >
                        <SeverityChip severity={issue.severity} />
                        <Box sx={{ flex: 1, minWidth: 0 }}>
                          <Stack
                            direction="row"
                            spacing={1}
                            alignItems="center"
                          >
                            <Typography fontWeight={750}>
                              {issue.title}
                            </Typography>
                            {issue.floorName && (
                              <Chip
                                size="small"
                                variant="outlined"
                                label={issue.floorName}
                              />
                            )}
                          </Stack>
                          <Typography
                            variant="body2"
                            color="text.secondary"
                            sx={{ mt: 0.35 }}
                          >
                            {issue.description}
                          </Typography>
                          <Typography variant="caption" color="text.disabled">
                            {issue.occurredAt
                              ? new Date(issue.occurredAt).toLocaleString(
                                  "ko-KR",
                                )
                              : ""}
                            {issue.confidence != null
                              ? ` · AI 신뢰도 ${Math.round(issue.confidence * 100)}%`
                              : ""}
                          </Typography>
                        </Box>
                        <Button
                          size="small"
                          variant={
                            issue.severity === "critical"
                              ? "contained"
                              : "outlined"
                          }
                          color={
                            issue.severity === "critical" ? "error" : "primary"
                          }
                          disabled={busy === issue.id}
                          endIcon={<ArrowForwardRounded />}
                          onClick={() => void resolve(issue)}
                        >
                          {issue.action === "assign"
                            ? "좌석 배정"
                            : issue.action === "release"
                              ? "좌석 해제"
                              : issue.action === "approve"
                                ? "확인 완료"
                                : "영역 맞춤"}
                        </Button>
                      </Stack>
                      {index < issues.length - 1 && <Divider />}
                    </Box>
                  ))
                )}
              </Paper>
            </Grid>
            <Grid size={{ xs: 12, lg: 4 }}>
              <Stack spacing={2.5}>
                <Paper sx={{ p: 2.5 }}>
                  <Stack
                    direction="row"
                    justifyContent="space-between"
                    alignItems="center"
                  >
                    <Box>
                      <Typography variant="h6">운영 준비도</Typography>
                      <Typography variant="body2" color="text.secondary">
                        {readinessDone} / {readinessItems.length} 완료
                      </Typography>
                    </Box>
                    <Typography variant="h5" color="primary.main">
                      {Math.round(
                        (readinessDone / readinessItems.length) * 100,
                      )}
                      %
                    </Typography>
                  </Stack>
                  <LinearProgress
                    variant="determinate"
                    value={(readinessDone / readinessItems.length) * 100}
                    sx={{ my: 2, height: 7, borderRadius: 8 }}
                  />
                  <Stack spacing={0.5}>
                    {readinessItems.map((item) => (
                      <Stack
                        key={item.key}
                        direction="row"
                        alignItems="center"
                        spacing={1}
                        onClick={() => navigate(item.route)}
                        sx={{
                          p: 1,
                          mx: -1,
                          borderRadius: 2,
                          cursor: "pointer",
                          "&:hover": { bgcolor: "action.hover" },
                        }}
                      >
                        <CheckCircleRounded
                          fontSize="small"
                          color={
                            data.readiness[item.key] ? "success" : "disabled"
                          }
                        />
                        <Typography variant="body2" sx={{ flex: 1 }}>
                          {item.label}
                        </Typography>
                        {!data.readiness[item.key] && (
                          <OpenInNewRounded
                            sx={{ fontSize: 15, color: "text.disabled" }}
                          />
                        )}
                      </Stack>
                    ))}
                  </Stack>
                </Paper>
                <Paper sx={{ p: 2.5 }}>
                  <Typography variant="h6">연동 상태</Typography>
                  <Stack spacing={1.5} mt={2}>
                    <IntegrationRow
                      icon={<CloudSyncRounded />}
                      label="인사 동기화"
                      value={
                        data.readiness.hrEnabled
                          ? data.integration.lastSyncStatus || "대기 중"
                          : "사용 안 함"
                      }
                      ok={data.integration.lastSyncStatus === "completed"}
                    />
                    <IntegrationRow
                      icon={<ApartmentRounded />}
                      label="Keycloak SSO"
                      value={data.readiness.oidcEnabled ? "활성" : "설정 필요"}
                      ok={data.readiness.oidcEnabled}
                    />
                    <IntegrationRow
                      icon={<MapRounded />}
                      label="게시 좌석맵"
                      value={
                        data.readiness.hasPublishedMap ? "정상" : "설정 필요"
                      }
                      ok={data.readiness.hasPublishedMap}
                    />
                  </Stack>
                  {data.integration.lastSyncAt && (
                    <Typography
                      variant="caption"
                      color="text.secondary"
                      display="block"
                      mt={1.5}
                    >
                      최근 동기화{" "}
                      {new Date(data.integration.lastSyncAt).toLocaleString(
                        "ko-KR",
                      )}
                    </Typography>
                  )}
                </Paper>
                <Paper sx={{ p: 2.5 }}>
                  <Typography variant="h6">최근 좌석 변경</Typography>
                  <Stack spacing={1.5} mt={2}>
                    {history.length ? (
                      history.map((item) => (
                        <Stack key={item.id} direction="row" spacing={1.2}>
                          <Box
                            sx={{
                              width: 7,
                              height: 7,
                              borderRadius: "50%",
                              bgcolor: "primary.main",
                              mt: 0.7,
                              flexShrink: 0,
                            }}
                          />
                          <Box>
                            <Typography variant="body2" fontWeight={700}>
                              {item.employeeName || "직원"} ·{" "}
                              {item.newSeat || "해제"}
                            </Typography>
                            <Typography
                              variant="caption"
                              color="text.secondary"
                            >
                              {item.reason || "좌석 변경"} ·{" "}
                              {new Date(item.changedAt).toLocaleDateString(
                                "ko-KR",
                              )}
                            </Typography>
                          </Box>
                        </Stack>
                      ))
                    ) : (
                      <Typography variant="body2" color="text.secondary">
                        최근 변경이 없습니다.
                      </Typography>
                    )}
                  </Stack>
                </Paper>
              </Stack>
            </Grid>
          </Grid>
        </>
      )}
    </Box>
  );
}

function IntegrationRow({
  icon,
  label,
  value,
  ok,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
  ok: boolean;
}) {
  return (
    <Stack direction="row" spacing={1.2} alignItems="center">
      <Box
        sx={{ color: ok ? "success.main" : "text.disabled", display: "flex" }}
      >
        {icon}
      </Box>
      <Typography variant="body2" sx={{ flex: 1 }}>
        {label}
      </Typography>
      <Chip
        size="small"
        color={ok ? "success" : "default"}
        variant={ok ? "filled" : "outlined"}
        label={value}
      />
    </Stack>
  );
}
