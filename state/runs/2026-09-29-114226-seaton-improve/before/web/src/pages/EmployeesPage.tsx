import { useEffect, useMemo, useState, type FormEvent } from "react";
import {
  Alert,
  Avatar,
  Box,
  Button,
  Chip,
  FormControl,
  InputAdornment,
  MenuItem,
  Paper,
  Select,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from "@mui/material";
import SearchRounded from "@mui/icons-material/SearchRounded";
import UploadFileRounded from "@mui/icons-material/UploadFileRounded";
import DownloadRounded from "@mui/icons-material/DownloadRounded";
import GroupsRounded from "@mui/icons-material/GroupsRounded";
import EventSeatRounded from "@mui/icons-material/EventSeatRounded";
import PersonOffRounded from "@mui/icons-material/PersonOffRounded";
import ArrowForwardRounded from "@mui/icons-material/ArrowForwardRounded";
import { useNavigate } from "react-router-dom";
import { api } from "../api";
import { MetricCard, PageHeader, TableSkeleton } from "../components/AdminUI";
import { localDateStamp, safeFileName, toCSV } from "../lib/format";
import {
  EMPLOYEE_CSV_HEADERS,
  employeeCsvRows,
  employeeSeatLabel,
  employeeStatusLabel,
} from "../lib/employeeExport";
import { employeeQuery } from "../lib/employeeQuery";
import type { BulkFailure, Employee, Organization } from "../types";

export function EmployeesPage() {
  const navigate = useNavigate();
  const [items, setItems] = useState<Employee[]>([]),
    [organizations, setOrganizations] = useState<Organization[]>([]),
    [q, setQ] = useState(""),
    [status, setStatus] = useState(""),
    [assignment, setAssignment] = useState(""),
    [organizationId, setOrganizationId] = useState(""),
    [loading, setLoading] = useState(true),
    [message, setMessage] = useState(""),
    [error, setError] = useState("");
  const load = async (
    query = q,
    nextStatus = status,
    nextAssignment = assignment,
    nextOrganizationId = organizationId,
  ) => {
    setLoading(true);
    try {
      const params = employeeQuery({
        q: query,
        status: nextStatus,
        assignment: nextAssignment,
        organizationId: nextOrganizationId,
      });
      const data = await api<{ items: Employee[] }>(
        `/api/v1/employees?${params}`,
      );
      setItems(data.items);
    } catch (e) {
      setError(e instanceof Error ? e.message : "직원을 불러오지 못했습니다");
    } finally {
      setLoading(false);
    }
  };
  useEffect(() => {
    void load("", "", "", "");
  }, []); // eslint-disable-line react-hooks/exhaustive-deps
  // 조직 필터의 선택지. 조직을 못 읽어도 직원 목록은 그대로 보여야 하므로
  // 이 실패는 화면에 올리지 않는다 — 조직 Select 만 "전체 조직" 하나로 남는다.
  useEffect(() => {
    void api<{ items: Organization[] }>("/api/v1/organizations")
      .then((data) => setOrganizations(data.items))
      .catch(() => setOrganizations([]));
  }, []);
  const search = (event: FormEvent) => {
    event.preventDefault();
    void load();
  };
  const upload = async (file?: File) => {
    if (!file) return;
    const form = new FormData();
    form.append("file", file);
    try {
      const result = await api<{ success: number; failed: number }>(
        "/api/v1/employees/import",
        { method: "POST", body: form },
      );
      setMessage(`${result.success}명 반영, ${result.failed}건 확인 필요`);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "가져오기에 실패했습니다");
    }
  };
  // 좌석 일괄 배정. 서버는 사번과 좌석 번호 두 열만 읽고, 실패한 행은 이유와 함께
  // 돌려준다. 어느 행이 왜 걸렸는지 보여 주지 않으면 관리자가 파일을 고칠 수 없다.
  const [failures, setFailures] = useState<BulkFailure[]>([]);
  const assignFromFile = async (file?: File) => {
    if (!file) return;
    const form = new FormData();
    form.append("file", file);
    setFailures([]);
    try {
      const result = await api<{
        success: number;
        failed: number;
        failures?: BulkFailure[];
      }>("/api/v1/seat-assignments/bulk", { method: "POST", body: form });
      setMessage(
        `${result.success}건 배정, ${result.failed}건 확인 필요`.concat(
          result.failed ? " · 아래 목록에서 사유를 확인하세요" : "",
        ),
      );
      setFailures(result.failures ?? []);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "좌석 배정에 실패했습니다");
    }
  };
  const downloadSeatTemplate = () => {
    const csv = "\ufeff사번,좌석번호\n100001,HQ-3F-001\n";
    const url = URL.createObjectURL(new Blob([csv], { type: "text/csv" }));
    const link = document.createElement("a");
    link.href = url;
    link.download = "seaton-seat-assignments-template.csv";
    link.click();
    URL.revokeObjectURL(url);
  };
  const downloadTemplate = () => {
    const csv =
      "\ufeff사번,이름,이메일,조직코드,조직명,직급,직책,근무지,재직상태\n100001,홍길동,hong@example.com,DEV,개발팀,책임,팀원,본사,active\n";
    const url = URL.createObjectURL(new Blob([csv], { type: "text/csv" }));
    const link = document.createElement("a");
    link.href = url;
    link.download = "seaton-employees-template.csv";
    link.click();
    URL.revokeObjectURL(url);
  };
  // 조회 결과를 그대로 파일로 넘긴다. 이력 화면과 달리 다시 조회하지 않는다 —
  // 서버가 한 번에 500명까지만 주고 전체 건수도 돌려주지 않으므로, 다시
  // 조회해도 더 받을 것이 없고 "표에 보이는 것과 파일이 같다"는 것만 잃는다.
  const exportCsv = () => {
    const csv = toCSV(EMPLOYEE_CSV_HEADERS, employeeCsvRows(items));
    const url = URL.createObjectURL(
      new Blob([csv], { type: "text/csv;charset=utf-8;" }),
    );
    const link = document.createElement("a");
    link.href = url;
    // 파일 이름은 사용자 시간대의 날짜를 쓴다. UTC 날짜를 쓰면 하루 어긋난
    // 이름이 붙는다. 이력 화면(HistoryPage.exportCsv)과 같은 꼴이다.
    link.download = safeFileName(`직원목록_${localDateStamp()}.csv`);
    link.click();
    URL.revokeObjectURL(url);
    setMessage(`${items.length}명을 내보냈습니다.`);
  };
  const counts = useMemo(
    () => ({
      active: items.filter((item) => item.status === "active").length,
      assigned: items.filter((item) => item.seatId).length,
      unassigned: items.filter(
        (item) => item.status === "active" && !item.seatId,
      ).length,
    }),
    [items],
  );
  return (
    <Box sx={{ p: { xs: 2, md: 3 }, maxWidth: 1500, mx: "auto" }}>
      <PageHeader
        eyebrow="PEOPLE DIRECTORY"
        title="직원"
        description="인사 연동 결과와 좌석 배정 상태를 확인하고, 예외 직원만 빠르게 처리합니다."
        actions={
          <Stack direction="row" spacing={1}>
            <Button
              variant="outlined"
              startIcon={<DownloadRounded />}
              onClick={exportCsv}
              disabled={!items.length || loading}
            >
              CSV 내보내기
            </Button>
            <Button
              variant="outlined"
              startIcon={<DownloadRounded />}
              onClick={downloadTemplate}
              sx={{ display: { xs: "none", sm: "inline-flex" } }}
            >
              직원 양식
            </Button>
            <Button
              variant="outlined"
              startIcon={<DownloadRounded />}
              onClick={downloadSeatTemplate}
              sx={{ display: { xs: "none", sm: "inline-flex" } }}
            >
              배정 양식
            </Button>
            <Button
              component="label"
              variant="outlined"
              startIcon={<EventSeatRounded />}
            >
              좌석 일괄 배정
              <input
                hidden
                type="file"
                accept=".csv,.xlsx"
                onChange={(event) => {
                  void assignFromFile(event.target.files?.[0]);
                  event.target.value = "";
                }}
              />
            </Button>
            <Button
              component="label"
              variant="contained"
              startIcon={<UploadFileRounded />}
            >
              직원 가져오기
              <input
                hidden
                type="file"
                accept=".csv,.xlsx"
                onChange={(event) => {
                  void upload(event.target.files?.[0]);
                  event.target.value = "";
                }}
              />
            </Button>
          </Stack>
        }
      />
      {message && (
        <Alert severity="success" onClose={() => setMessage("")} sx={{ mb: 2 }}>
          {message}
        </Alert>
      )}
      {failures.length > 0 && (
        <Alert
          severity="warning"
          onClose={() => setFailures([])}
          sx={{ mb: 2 }}
        >
          <Typography variant="subtitle2" gutterBottom>
            반영되지 않은 {failures.length}행
          </Typography>
          <Box component="ul" sx={{ m: 0, pl: 2.5 }}>
            {failures.slice(0, 20).map((item) => (
              <li key={`${item.row}-${item.seatNo}`}>
                <Typography variant="caption">
                  {item.row}행 · {item.employeeNo || "사번 없음"} →{" "}
                  {item.seatNo || "좌석 없음"} · {item.error}
                </Typography>
              </li>
            ))}
          </Box>
          {failures.length > 20 && (
            <Typography variant="caption" color="text.secondary">
              앞의 20행만 표시했습니다.
            </Typography>
          )}
        </Alert>
      )}
      {error && (
        <Alert severity="error" onClose={() => setError("")} sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}
      <Box
        sx={{
          display: "grid",
          gridTemplateColumns: { xs: "1fr", sm: "repeat(3,1fr)" },
          gap: 2,
          mb: 2,
        }}
      >
        <MetricCard
          label="조회된 재직자"
          value={counts.active}
          helper={`현재 조건 ${items.length}명`}
          icon={<GroupsRounded />}
        />
        <MetricCard
          label="좌석 배정"
          value={counts.assigned}
          helper="현재 좌석이 있는 직원"
          tone="#3478C8"
          icon={<EventSeatRounded />}
        />
        <MetricCard
          label="미배정"
          value={counts.unassigned}
          helper="바로 처리가 필요한 재직자"
          tone="#E79418"
          icon={<PersonOffRounded />}
        />
      </Box>
      <Paper sx={{ p: 2, mb: 2 }}>
        <Stack
          component="form"
          onSubmit={search}
          direction={{ xs: "column", md: "row" }}
          spacing={1.2}
        >
          <TextField
            value={q}
            onChange={(event) => setQ(event.target.value)}
            placeholder="이름, 사번, 이메일, 조직 검색"
            sx={{ flex: 1, minWidth: 240 }}
            slotProps={{
              input: {
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchRounded />
                  </InputAdornment>
                ),
              },
            }}
          />
          <FormControl sx={{ minWidth: 140 }}>
            <Select
              value={status}
              displayEmpty
              inputProps={{ "aria-label": "재직상태 필터" }}
              onChange={(event) => {
                setStatus(event.target.value);
                void load(q, event.target.value, assignment, organizationId);
              }}
            >
              <MenuItem value="">전체 재직상태</MenuItem>
              <MenuItem value="active">재직</MenuItem>
              <MenuItem value="leave">휴직</MenuItem>
              <MenuItem value="retired">퇴직</MenuItem>
            </Select>
          </FormControl>
          <FormControl sx={{ minWidth: 140 }}>
            <Select
              value={assignment}
              displayEmpty
              inputProps={{ "aria-label": "배정상태 필터" }}
              onChange={(event) => {
                setAssignment(event.target.value);
                void load(q, status, event.target.value, organizationId);
              }}
            >
              <MenuItem value="">전체 배정상태</MenuItem>
              <MenuItem value="assigned">배정</MenuItem>
              <MenuItem value="unassigned">미배정</MenuItem>
            </Select>
          </FormControl>
          <FormControl sx={{ minWidth: 160 }}>
            <Select
              value={organizationId}
              displayEmpty
              inputProps={{ "aria-label": "조직 필터" }}
              onChange={(event) => {
                setOrganizationId(event.target.value);
                void load(q, status, assignment, event.target.value);
              }}
            >
              <MenuItem value="">전체 조직</MenuItem>
              {organizations.map((organization) => (
                <MenuItem key={organization.id} value={organization.id}>
                  {organization.name}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          <Button type="submit" variant="outlined">
            검색
          </Button>
        </Stack>
      </Paper>
      {loading ? (
        <TableSkeleton rows={8} />
      ) : (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>직원</TableCell>
                <TableCell>사번</TableCell>
                <TableCell>조직</TableCell>
                <TableCell>직책/직급</TableCell>
                <TableCell>좌석</TableCell>
                <TableCell>상태</TableCell>
                <TableCell align="right">작업</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {items.length === 0 && (
                <TableRow>
                  <TableCell colSpan={7} align="center" sx={{ py: 7 }}>
                    <Typography fontWeight={700}>
                      조건에 맞는 직원이 없습니다.
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      검색어 또는 필터를 변경해 보세요.
                    </Typography>
                  </TableCell>
                </TableRow>
              )}
              {items.map((employee) => (
                <TableRow key={employee.id} hover>
                  <TableCell>
                    <Stack direction="row" spacing={1.2} alignItems="center">
                      <Avatar
                        sx={{
                          width: 36,
                          height: 36,
                          bgcolor: employee.seatId
                            ? "primary.main"
                            : "grey.400",
                          fontSize: 13,
                        }}
                      >
                        {employee.name.slice(0, 1)}
                      </Avatar>
                      <Box>
                        <Typography variant="body2" fontWeight={700}>
                          {employee.name}
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                          {employee.email}
                        </Typography>
                      </Box>
                    </Stack>
                  </TableCell>
                  <TableCell>{employee.employeeNo}</TableCell>
                  <TableCell>{employee.organizationName || "-"}</TableCell>
                  <TableCell>
                    {[employee.position, employee.title]
                      .filter(Boolean)
                      .join(" · ") || "-"}
                  </TableCell>
                  <TableCell>
                    <Chip
                      size="small"
                      variant={employee.seatNo ? "filled" : "outlined"}
                      color={employee.seatNo ? "primary" : "warning"}
                      label={employeeSeatLabel(employee.seatNo)}
                    />
                  </TableCell>
                  <TableCell>{employeeStatusLabel(employee.status)}</TableCell>
                  <TableCell align="right">
                    <Button
                      size="small"
                      endIcon={<ArrowForwardRounded />}
                      onClick={() =>
                        navigate(
                          `/?q=${encodeURIComponent(employee.employeeNo)}`,
                        )
                      }
                    >
                      {employee.seatNo ? "지도에서 보기" : "좌석 배정"}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
      <Typography
        variant="caption"
        color="text.secondary"
        display="block"
        mt={2}
      >
        최대 500명 표시 · 대규모 데이터는 검색과 필터를 함께 사용하세요.
      </Typography>
    </Box>
  );
}
