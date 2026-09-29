import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  MenuItem,
  Paper,
  Skeleton,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Tooltip,
  Typography,
} from "@mui/material";
import ArrowForwardRounded from "@mui/icons-material/ArrowForwardRounded";
import SearchRounded from "@mui/icons-material/SearchRounded";
import DownloadRounded from "@mui/icons-material/DownloadRounded";
import HistoryToggleOffRounded from "@mui/icons-material/HistoryToggleOffRounded";
import RestartAltRounded from "@mui/icons-material/RestartAltRounded";
import { api } from "../api";
import { PageHeader } from "../components/AdminUI";
import {
  absoluteTime,
  dayRangeToISO,
  localDateStamp,
  relativeTime,
  safeFileName,
  SOURCE_LABELS,
  sourceLabel,
  toCSV,
} from "../lib/format";

type Item = {
  id: string;
  changedAt: string;
  employeeNo: string;
  employeeName: string;
  previousSeat: string;
  newSeat: string;
  actor: string;
  reason: string;
  source: string;
};

const PAGE_SIZE = 100;
// 서버가 한 번에 돌려주는 최대치. 내보내기는 이 값까지 받아 온다.
const EXPORT_LIMIT = 500;
const EMPTY_FILTERS = { q: "", source: "", from: "", to: "" };

export function HistoryPage() {
  const [items, setItems] = useState<Item[]>([]),
    [total, setTotal] = useState(0),
    [limit, setLimit] = useState(PAGE_SIZE),
    [filters, setFilters] = useState(EMPTY_FILTERS),
    // 입력 중에 매 글자 요청하지 않도록 확정된 조건만 따로 둔다.
    [applied, setApplied] = useState(EMPTY_FILTERS),
    [totalCapped, setTotalCapped] = useState(false),
    // 조회 버튼은 조건이 그대로여도 다시 불러와야 한다. 값만 비교하면 React가
    // 상태 변경을 건너뛰어 아무 일도 일어나지 않는다.
    [refreshKey, setRefreshKey] = useState(0),
    [loading, setLoading] = useState(true),
    [exporting, setExporting] = useState(false),
    [notice, setNotice] = useState(""),
    [error, setError] = useState("");
  // 늦게 도착한 이전 응답이 최신 결과를 덮어쓰지 않도록 순번을 센다.
  const requestRef = useRef(0);

  const load = useCallback(async () => {
    const sequence = ++requestRef.current;
    setLoading(true);
    setError("");
    try {
      const params = new URLSearchParams({ limit: String(limit) });
      if (applied.q) params.set("q", applied.q);
      if (applied.source) params.set("source", applied.source);
      // 날짜 경계는 사용자 시간대를 아는 브라우저가 시각으로 바꿔 넘긴다.
      const range = dayRangeToISO(applied.from, applied.to);
      if (range.from) params.set("from", range.from);
      if (range.to) params.set("to", range.to);
      const data = await api<{
        items: Item[];
        total: number;
        totalCapped?: boolean;
      }>(`/api/v1/seat-history?${params}`);
      if (sequence !== requestRef.current) return;
      setItems(data.items);
      setTotal(data.total ?? data.items.length);
      setTotalCapped(Boolean(data.totalCapped));
    } catch (e) {
      if (sequence !== requestRef.current) return;
      setError(e instanceof Error ? e.message : "이력을 불러오지 못했습니다");
    } finally {
      if (sequence === requestRef.current) setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [applied, limit, refreshKey]);

  useEffect(() => {
    void load();
  }, [load]);

  const filtering = useMemo(
    () => Object.values(applied).some(Boolean),
    [applied],
  );
  const submit = (event?: React.FormEvent) => {
    event?.preventDefault();
    setLimit(PAGE_SIZE);
    setApplied({ ...filters });
    setRefreshKey((n) => n + 1);
  };
  const reset = () => {
    setFilters(EMPTY_FILTERS);
    setApplied({ ...EMPTY_FILTERS });
    setLimit(PAGE_SIZE);
    setRefreshKey((n) => n + 1);
  };

  // 감사 자료라 화면에 보이는 만큼이 아니라 조건에 맞는 전체를 내보내야 한다.
  // 화면은 100건씩 끊어 보여주므로, 내보낼 때는 서버 상한까지 다시 조회한다.
  const exportCsv = async () => {
    setExporting(true);
    setError("");
    try {
      const params = new URLSearchParams({ limit: String(EXPORT_LIMIT) });
      if (applied.q) params.set("q", applied.q);
      if (applied.source) params.set("source", applied.source);
      const range = dayRangeToISO(applied.from, applied.to);
      if (range.from) params.set("from", range.from);
      if (range.to) params.set("to", range.to);
      const data = await api<{ items: Item[]; total: number }>(
        `/api/v1/seat-history?${params}`,
      );
      const csv = toCSV(
        [
          "변경일시",
          "사번",
          "직원",
          "이전 좌석",
          "새 좌석",
          "처리자",
          "사유",
          "방식",
        ],
        data.items.map((x) => [
          absoluteTime(x.changedAt),
          x.employeeNo,
          x.employeeName,
          x.previousSeat || "미배정",
          x.newSeat || "해제",
          x.actor,
          x.reason,
          sourceLabel(x.source),
        ]),
      );
      const url = URL.createObjectURL(
        new Blob([csv], { type: "text/csv;charset=utf-8;" }),
      );
      const link = document.createElement("a");
      link.href = url;
      // 파일 이름도 사용자 시간대의 날짜를 쓴다. UTC 날짜를 쓰면 방금 고른
      // 기간과 하루 어긋난 이름이 붙는다.
      link.download = safeFileName(`좌석변경이력_${localDateStamp()}.csv`);
      link.click();
      URL.revokeObjectURL(url);
      if (data.items.length < data.total)
        setNotice(
          `조건에 맞는 ${data.total}건 중 ${data.items.length}건을 내보냈습니다. 한 번에 최대 ${EXPORT_LIMIT}건까지 받을 수 있으니 기간을 나눠 주세요.`,
        );
      else setNotice(`${data.items.length}건을 내보냈습니다.`);
    } catch (e) {
      setError(e instanceof Error ? e.message : "내보내지 못했습니다");
    } finally {
      setExporting(false);
    }
  };

  return (
    <Box sx={{ p: { xs: 2, md: 3 }, maxWidth: 1400, mx: "auto" }}>
      <PageHeader
        eyebrow="AUDIT TRAIL"
        title="변경 이력"
        description="좌석 이동은 별도 입력 없이 처리 주체와 사유까지 자동 기록됩니다."
        actions={
          <Button
            variant="outlined"
            startIcon={<DownloadRounded />}
            onClick={() => void exportCsv()}
            disabled={!items.length || exporting}
          >
            {exporting ? "내보내는 중…" : "CSV 내보내기"}
          </Button>
        }
      />
      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError("")}>
          {error}
        </Alert>
      )}
      {notice && (
        <Alert severity="info" sx={{ mb: 2 }} onClose={() => setNotice("")}>
          {notice}
        </Alert>
      )}
      <Paper component="form" onSubmit={submit} sx={{ p: 2, mb: 2.5 }}>
        <Stack
          direction={{ xs: "column", md: "row" }}
          spacing={1.5}
          alignItems={{ md: "center" }}
        >
          <TextField
            size="small"
            fullWidth
            value={filters.q}
            onChange={(e) => setFilters((f) => ({ ...f, q: e.target.value }))}
            placeholder="이름, 사번, 좌석 번호"
            slotProps={{
              input: {
                startAdornment: (
                  <SearchRounded sx={{ mr: 1, color: "text.disabled" }} />
                ),
              },
            }}
          />
          <TextField
            select
            size="small"
            label="방식"
            value={filters.source}
            onChange={(e) =>
              setFilters((f) => ({ ...f, source: e.target.value }))
            }
            sx={{ minWidth: 150 }}
          >
            <MenuItem value="">전체</MenuItem>
            {Object.entries(SOURCE_LABELS).map(([value, label]) => (
              <MenuItem key={value} value={value}>
                {label}
              </MenuItem>
            ))}
          </TextField>
          <TextField
            size="small"
            type="date"
            label="시작일"
            value={filters.from}
            onChange={(e) =>
              setFilters((f) => ({ ...f, from: e.target.value }))
            }
            slotProps={{ inputLabel: { shrink: true } }}
          />
          <TextField
            size="small"
            type="date"
            label="종료일"
            value={filters.to}
            onChange={(e) => setFilters((f) => ({ ...f, to: e.target.value }))}
            slotProps={{ inputLabel: { shrink: true } }}
          />
          <Stack direction="row" spacing={1}>
            <Button type="submit" variant="contained">
              조회
            </Button>
            <Tooltip title="조건 초기화">
              <span>
                <Button
                  onClick={reset}
                  disabled={!filtering && !Object.values(filters).some(Boolean)}
                >
                  <RestartAltRounded />
                </Button>
              </span>
            </Tooltip>
          </Stack>
        </Stack>
      </Paper>

      <Stack
        direction="row"
        alignItems="center"
        spacing={1}
        sx={{ mb: 1.5, minHeight: 28 }}
      >
        {loading ? (
          <CircularProgress size={16} />
        ) : (
          <Typography variant="body2" color="text.secondary">
            {`${filtering ? "조건에 맞는" : "전체"} ${total}${
              totalCapped ? "+" : ""
            }건 중 ${items.length}건`}
          </Typography>
        )}
      </Stack>

      {loading ? (
        <Paper sx={{ p: 2 }}>
          {Array.from({ length: 6 }, (_, i) => (
            <Skeleton key={i} height={44} />
          ))}
        </Paper>
      ) : items.length === 0 ? (
        <Paper sx={{ p: 6, textAlign: "center" }}>
          <HistoryToggleOffRounded
            sx={{ fontSize: 44, color: "text.disabled", mb: 1 }}
          />
          <Typography color="text.secondary">
            {filtering
              ? "조건에 맞는 변경 이력이 없습니다."
              : "아직 좌석 변경 이력이 없습니다."}
          </Typography>
          {filtering && (
            <Button onClick={reset} sx={{ mt: 1.5 }}>
              조건 초기화
            </Button>
          )}
        </Paper>
      ) : (
        <>
          <TableContainer component={Paper} sx={{ overflowX: "auto" }}>
            <Table size="small" aria-label="좌석 변경 이력">
              <TableHead>
                <TableRow>
                  <TableCell>변경일시</TableCell>
                  <TableCell>직원</TableCell>
                  <TableCell>좌석 변경</TableCell>
                  <TableCell>처리자</TableCell>
                  <TableCell>사유</TableCell>
                  <TableCell>방식</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {items.map((x) => (
                  <TableRow key={x.id} hover>
                    <TableCell sx={{ whiteSpace: "nowrap" }}>
                      <Tooltip title={absoluteTime(x.changedAt)}>
                        <span>{relativeTime(x.changedAt)}</span>
                      </Tooltip>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" fontWeight={700}>
                        {x.employeeName || "-"}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        {x.employeeNo}
                      </Typography>
                    </TableCell>
                    <TableCell sx={{ whiteSpace: "nowrap" }}>
                      <Chip
                        size="small"
                        variant="outlined"
                        label={x.previousSeat || "미배정"}
                      />
                      <ArrowForwardRounded
                        sx={{ fontSize: 16, verticalAlign: "middle", mx: 0.5 }}
                      />
                      <Chip
                        size="small"
                        color={x.newSeat ? "primary" : "default"}
                        label={x.newSeat || "해제"}
                      />
                    </TableCell>
                    <TableCell>{x.actor}</TableCell>
                    <TableCell>{x.reason || "-"}</TableCell>
                    <TableCell>
                      <Chip
                        size="small"
                        variant="outlined"
                        label={sourceLabel(x.source)}
                      />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
          {items.length < total && (
            <Stack alignItems="center" sx={{ mt: 2 }}>
              <Button
                onClick={() => setLimit((n) => Math.min(500, n + PAGE_SIZE))}
                disabled={loading || limit >= 500}
              >
                {limit >= 500
                  ? "한 번에 최대 500건까지 볼 수 있습니다"
                  : `더 보기 (남은 ${total - items.length}건)`}
              </Button>
            </Stack>
          )}
        </>
      )}
    </Box>
  );
}
