import type { ReactNode } from "react";
import {
  Box,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  LinearProgress,
  Paper,
  Skeleton,
  Stack,
  Typography,
} from "@mui/material";

export function PageHeader({
  eyebrow,
  title,
  description,
  actions,
}: {
  eyebrow?: string;
  title: string;
  description: string;
  actions?: ReactNode;
}) {
  return (
    <Stack
      direction={{ xs: "column", md: "row" }}
      justifyContent="space-between"
      alignItems={{ md: "center" }}
      spacing={2}
      sx={{ mb: 3 }}
    >
      <Box>
        {eyebrow && (
          <Typography
            variant="overline"
            sx={{ color: "primary.main", fontWeight: 800, letterSpacing: 1.2 }}
          >
            {eyebrow}
          </Typography>
        )}
        <Typography variant="h5">{title}</Typography>
        <Typography color="text.secondary" sx={{ mt: 0.4 }}>
          {description}
        </Typography>
      </Box>
      {actions && (
        <Stack direction="row" spacing={1}>
          {actions}
        </Stack>
      )}
    </Stack>
  );
}

export function MetricCard({
  label,
  value,
  helper,
  icon,
  tone = "#087E8B",
  progress,
}: {
  label: string;
  value: string | number;
  helper: string;
  icon: ReactNode;
  tone?: string;
  progress?: number;
}) {
  return (
    <Card sx={{ height: "100%" }}>
      <CardContent>
        <Stack direction="row" justifyContent="space-between" spacing={2}>
          <Box>
            <Typography variant="body2" color="text.secondary">
              {label}
            </Typography>
            <Typography variant="h4" sx={{ mt: 0.6, mb: 0.4 }}>
              {value}
            </Typography>
          </Box>
          <Box
            sx={{
              width: 44,
              height: 44,
              borderRadius: 3,
              display: "grid",
              placeItems: "center",
              bgcolor: `${tone}16`,
              color: tone,
              flexShrink: 0,
            }}
          >
            {icon}
          </Box>
        </Stack>
        {progress != null && (
          <LinearProgress
            variant="determinate"
            value={Math.max(0, Math.min(100, progress))}
            sx={{
              my: 1,
              height: 6,
              borderRadius: 9,
              bgcolor: `${tone}16`,
              "& .MuiLinearProgress-bar": { bgcolor: tone },
            }}
          />
        )}
        <Typography variant="caption" color="text.secondary">
          {helper}
        </Typography>
      </CardContent>
    </Card>
  );
}

export function SeverityChip({ severity }: { severity: string }) {
  const config: Record<
    string,
    { label: string; color: "error" | "warning" | "info" | "default" }
  > = {
    critical: { label: "긴급", color: "error" },
    warning: { label: "확인", color: "warning" },
    info: { label: "검토", color: "info" },
  };
  const item = config[severity] ?? {
    label: severity,
    color: "default" as const,
  };
  return (
    <Chip
      size="small"
      color={item.color}
      label={item.label}
      sx={{ fontWeight: 750 }}
    />
  );
}

export function LoadingCards() {
  return (
    <Stack spacing={2}>
      <Skeleton variant="rounded" height={150} />
      <Stack direction={{ xs: "column", md: "row" }} spacing={2}>
        {[1, 2, 3, 4].map((item) => (
          <Skeleton
            key={item}
            variant="rounded"
            height={140}
            sx={{ flex: 1 }}
          />
        ))}
      </Stack>
      <Skeleton variant="rounded" height={320} />
    </Stack>
  );
}

export function InlineBusy({ label = "처리 중" }: { label?: string }) {
  return (
    <Stack direction="row" spacing={1} alignItems="center">
      <CircularProgress size={15} />
      <Typography variant="caption">{label}</Typography>
    </Stack>
  );
}

/**
 * 표를 불러오는 동안 자리를 잡아 준다. 빈 화면이 스쳤다가 채워지면 화면이
 * 흔들려 보이고, "데이터가 없다"와 "아직 못 불러왔다"를 구분할 수 없다.
 */
export function TableSkeleton({
  rows = 6,
  height = 44,
}: {
  rows?: number;
  height?: number;
}) {
  return (
    <Paper sx={{ p: 2 }}>
      {Array.from({ length: rows }, (_, index) => (
        <Skeleton key={index} height={height} />
      ))}
    </Paper>
  );
}

/**
 * 내용이 없을 때 무엇을 하면 되는지까지 알려 주는 자리. 화면마다 다른 모양으로
 * 흩어져 있던 빈 상태를 한 곳으로 모은다.
 */
export function EmptyState({
  icon,
  title,
  description,
  action,
}: {
  icon?: ReactNode;
  title: string;
  description?: string;
  action?: ReactNode;
}) {
  return (
    <Paper sx={{ p: 6, textAlign: "center" }}>
      {icon && (
        <Box sx={{ color: "text.disabled", mb: 1, "& svg": { fontSize: 44 } }}>
          {icon}
        </Box>
      )}
      <Typography color="text.secondary">{title}</Typography>
      {description && (
        <Typography
          variant="caption"
          color="text.disabled"
          display="block"
          sx={{ mt: 0.5 }}
        >
          {description}
        </Typography>
      )}
      {action && <Box sx={{ mt: 2 }}>{action}</Box>}
    </Paper>
  );
}
