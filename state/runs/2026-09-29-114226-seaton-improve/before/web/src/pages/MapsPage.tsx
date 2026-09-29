import { useEffect, useState, type FormEvent } from "react";
import {
  Alert,
  Box,
  Button,
  Card,
  CardActions,
  CardContent,
  Chip,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  Grid,
  MenuItem,
  Paper,
  Select,
  Skeleton,
  Stack,
  TextField,
  Tooltip,
  Typography,
} from "@mui/material";
import AddBusinessRounded from "@mui/icons-material/AddBusinessRounded";
import LayersRounded from "@mui/icons-material/LayersRounded";
import UploadFileRounded from "@mui/icons-material/UploadFileRounded";
import AutoAwesomeRounded from "@mui/icons-material/AutoAwesomeRounded";
import PublishRounded from "@mui/icons-material/PublishRounded";
import DeleteOutlineRounded from "@mui/icons-material/DeleteOutlineRounded";
import UnpublishedRounded from "@mui/icons-material/UnpublishedRounded";
import GridOnRounded from "@mui/icons-material/GridOnRounded";
import CheckCircleRounded from "@mui/icons-material/CheckCircleRounded";
import RadioButtonCheckedRounded from "@mui/icons-material/RadioButtonCheckedRounded";
import RadioButtonUncheckedRounded from "@mui/icons-material/RadioButtonUncheckedRounded";
import EditLocationAltRounded from "@mui/icons-material/EditLocationAltRounded";
import EventSeatRounded from "@mui/icons-material/EventSeatRounded";
import ArrowForwardRounded from "@mui/icons-material/ArrowForwardRounded";
import { useNavigate } from "react-router-dom";
import { api, postJSON } from "../api";
import { PageHeader } from "../components/AdminUI";
import type { AnalysisJob, Building, Floor, FloorMap } from "../types";

type DialogName = "building" | "floor" | "upload" | "grid" | null;
// 분석은 비동기 잡이다. VLM 경로는 수십 초가 걸릴 수 있어 완료까지 폴링한다.
const JOB_POLL_INTERVAL = 1500;
const JOB_POLL_LIMIT = 400;
export function MapsPage() {
  const navigate = useNavigate();
  const [buildings, setBuildings] = useState<Building[]>([]),
    [floors, setFloors] = useState<Floor[]>([]),
    [maps, setMaps] = useState<FloorMap[]>([]),
    [dialog, setDialog] = useState<DialogName>(null),
    [selectedMap, setSelectedMap] = useState(""),
    [message, setMessage] = useState(""),
    [warnings, setWarnings] = useState<string[]>([]),
    [loading, setLoading] = useState(true),
    [analyzing, setAnalyzing] = useState<Record<string, string>>({}),
    [error, setError] = useState("");
  const load = async () => {
    try {
      const [b, f, m] = await Promise.all([
        api<{ items: Building[] }>("/api/v1/buildings"),
        api<{ items: Floor[] }>("/api/v1/floors"),
        api<{ items: FloorMap[] }>("/api/v1/floor-maps"),
      ]);
      setBuildings(b.items);
      setFloors(f.items);
      setMaps(m.items);
    } catch (e) {
      setError(
        e instanceof Error ? e.message : "도면 정보를 불러오지 못했습니다",
      );
    } finally {
      setLoading(false);
    }
  };
  useEffect(() => {
    void load();
  }, []);
  // 잘못 올린 도면 버전을 지운다. 되돌릴 수 없는 조작이라 무엇을 지우는지 이름을
  // 보여주고 한 번 더 확인받는다.
  const [removing, setRemoving] = useState<FloorMap | null>(null);
  const remove = async () => {
    if (!removing) return;
    try {
      await api<void>(`/api/v1/floor-maps/${removing.id}`, {
        method: "DELETE",
      });
      setMessage(`${removing.version} 버전을 삭제했습니다`);
      setRemoving(null);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "삭제하지 못했습니다");
      setRemoving(null);
    }
  };
  const action = async (path: string, label: string) => {
    try {
      const result = await api<{ message?: string }>(path, { method: "POST" });
      setMessage(result?.message || label);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "요청에 실패했습니다");
    }
  };
  // analyze는 202를 돌려주므로 잡이 끝날 때까지 상태를 확인한 뒤 결과를 보여준다.
  const analyze = async (mapId: string) => {
    setError("");
    setWarnings([]);
    try {
      const queued = await postJSON<{ jobId: string; engine: string }>(
        `/api/v1/floor-maps/${mapId}/analyze`,
        {},
      );
      setAnalyzing((current) => ({ ...current, [mapId]: queued.engine }));
      for (let attempt = 0; attempt < JOB_POLL_LIMIT; attempt++) {
        await new Promise((resolve) => setTimeout(resolve, JOB_POLL_INTERVAL));
        const job = await api<AnalysisJob>(
          `/api/v1/analysis-jobs/${queued.jobId}`,
        );
        if (job.status === "completed") {
          setMessage(
            job.message || `좌석 후보 ${job.detected}개를 생성했습니다`,
          );
          setWarnings(job.warnings ?? []);
          await load();
          return;
        }
        if (job.status === "failed") {
          setError(job.error || "도면 분석에 실패했습니다");
          await load();
          return;
        }
      }
      setError(
        "분석이 예상보다 오래 걸립니다. 잠시 후 도면 목록을 새로 고쳐 확인하세요",
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : "분석을 시작하지 못했습니다");
    } finally {
      setAnalyzing((current) => {
        const next = { ...current };
        delete next[mapId];
        return next;
      });
    }
  };
  const setupSteps = [
    {
      label: "1. 사업장",
      done: buildings.length > 0,
      detail: `${buildings.length}개`,
    },
    {
      label: "2. 층",
      done: floors.length > 0,
      detail: `${floors.length}개`,
    },
    {
      label: "3. 도면",
      done: maps.length > 0,
      detail: `${maps.length}개 버전`,
    },
    {
      label: "4. AI 분석",
      done: maps.some((map) => map.status !== "uploaded"),
      detail: `${maps.reduce((sum, map) => sum + (map.seatCount ?? 0), 0)}석`,
    },
    {
      label: "5. 게시",
      done: maps.some((map) => map.active),
      detail: maps.some((map) => map.active) ? "서비스 중" : "대기",
    },
  ];
  return (
    <Box sx={{ p: { xs: 2, md: 3 }, maxWidth: 1400, mx: "auto" }}>
      <PageHeader
        eyebrow="OFFICE DIGITAL TWIN"
        title="도면 · 좌석"
        description="사업장 구성부터 AI 분석, 좌석 보정, 게시까지 한 흐름으로 관리합니다."
        actions={
          <Stack direction="row" spacing={1} flexWrap="wrap">
            <Button
              variant="outlined"
              startIcon={<AddBusinessRounded />}
              onClick={() => setDialog("building")}
            >
              사업장
            </Button>
            <Button
              variant="outlined"
              startIcon={<LayersRounded />}
              onClick={() => setDialog("floor")}
              disabled={!buildings.length}
            >
              층
            </Button>
            <Button
              variant="contained"
              startIcon={<UploadFileRounded />}
              onClick={() => setDialog("upload")}
              disabled={!floors.length}
            >
              도면 업로드
            </Button>
          </Stack>
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
      {warnings.length > 0 && (
        <Alert
          severity="warning"
          onClose={() => setWarnings([])}
          sx={{ mb: 2 }}
        >
          <Typography variant="subtitle2" gutterBottom>
            분석 중 확인이 필요한 사항
          </Typography>
          <Box component="ul" sx={{ m: 0, pl: 2.5 }}>
            {warnings.map((warning) => (
              <li key={warning}>{warning}</li>
            ))}
          </Box>
        </Alert>
      )}
      <Paper sx={{ p: 2, mb: 2.5 }}>
        <Stack
          direction={{ xs: "column", md: "row" }}
          divider={
            <ArrowForwardRounded
              sx={{ color: "text.disabled", alignSelf: "center" }}
            />
          }
          spacing={1}
        >
          {setupSteps.map((step, index) => {
            // 아직 하지 않은 단계에도 체크 표시가 붙어 있어, 처음 설치한 관리자가
            // 다섯 단계를 모두 끝낸 것으로 읽었다. 끝난 단계만 체크로 두고 지금
            // 할 단계를 따로 드러낸다.
            const next = setupSteps.findIndex((item) => !item.done);
            const current = index === next;
            return (
              <Stack
                key={step.label}
                direction="row"
                alignItems="center"
                spacing={1}
                aria-label={`${step.label} · ${step.done ? "완료" : current ? "다음 할 일" : "대기"}`}
                sx={{ flex: 1, minWidth: 0, p: 1 }}
              >
                {step.done ? (
                  <CheckCircleRounded color="success" />
                ) : current ? (
                  <RadioButtonCheckedRounded color="primary" />
                ) : (
                  <RadioButtonUncheckedRounded
                    sx={{ color: "text.disabled" }}
                  />
                )}
                <Box minWidth={0}>
                  <Typography
                    variant="body2"
                    fontWeight={750}
                    color={
                      step.done || current ? "text.primary" : "text.disabled"
                    }
                  >
                    {step.label}
                  </Typography>
                  <Typography
                    variant="caption"
                    color={current ? "primary.main" : "text.secondary"}
                    fontWeight={current ? 700 : 400}
                  >
                    {current ? `${step.detail} · 지금 할 차례` : step.detail}
                  </Typography>
                </Box>
              </Stack>
            );
          })}
        </Stack>
      </Paper>
      {loading ? (
        <Grid container spacing={2}>
          {[1, 2, 3].map((item) => (
            <Grid key={item} size={{ xs: 12, md: 6, lg: 4 }}>
              <Skeleton variant="rounded" height={280} />
            </Grid>
          ))}
        </Grid>
      ) : maps.length === 0 ? (
        <Paper
          sx={{
            p: 6,
            textAlign: "center",
            borderStyle: "dashed",
            boxShadow: "none",
          }}
        >
          <LayersRounded sx={{ fontSize: 50, color: "text.disabled" }} />
          <Typography variant="h6" mt={1}>
            도면을 등록해 시작하세요
          </Typography>
          <Typography color="text.secondary">
            사업장 → 층 → PNG/JPG/PDF 도면 순으로 등록합니다.
          </Typography>
        </Paper>
      ) : (
        <Grid container spacing={2}>
          {maps.map((m) => (
            <Grid key={m.id} size={{ xs: 12, md: 6, lg: 4 }}>
              {/* 도면 버전마다 주소를 붙여 화면 검증이 카드를 정확히 집을 수
                  있게 한다. 글로 카드를 찾으면 버전이 여럿일 때 엉뚱한 카드를
                  조작한다. */}
              <Card data-map-version={m.version}>
                <Box
                  sx={{
                    height: 160,
                    bgcolor: "#E9EFF2",
                    backgroundImage: m.contentType.startsWith("image/")
                      ? `url(${m.contentUrl})`
                      : "none",
                    backgroundSize: "contain",
                    backgroundRepeat: "no-repeat",
                    backgroundPosition: "center",
                    display: "grid",
                    placeItems: "center",
                  }}
                >
                  {m.contentType === "application/pdf" && (
                    <Typography color="text.secondary">
                      PDF · {m.fileName}
                    </Typography>
                  )}
                </Box>
                <CardContent>
                  <Stack direction="row" justifyContent="space-between">
                    <Box>
                      <Typography fontWeight={750}>
                        {m.buildingName} · {m.floorName}
                      </Typography>
                      <Typography variant="body2" color="text.secondary">
                        Version {m.version}
                      </Typography>
                    </Box>
                    <Chip
                      size="small"
                      color={
                        m.active
                          ? "success"
                          : m.status === "failed"
                            ? "error"
                            : m.status === "review"
                              ? "warning"
                              : "default"
                      }
                      label={
                        m.active
                          ? "게시 중"
                          : m.status === "uploaded"
                            ? "분석 전"
                            : m.status === "review"
                              ? "검토 필요"
                              : m.status === "failed"
                                ? "분석 실패"
                                : m.status
                      }
                    />
                  </Stack>
                  <Stack direction="row" spacing={2.5} sx={{ mt: 2 }}>
                    <Stack direction="row" spacing={0.7} alignItems="center">
                      <EventSeatRounded fontSize="small" color="action" />
                      <Typography variant="body2">
                        <strong>{m.seatCount ?? 0}</strong>석
                      </Typography>
                    </Stack>
                    <Typography
                      variant="body2"
                      color={
                        (m.reviewCount ?? 0) > 0
                          ? "warning.main"
                          : "text.secondary"
                      }
                    >
                      검토 {m.reviewCount ?? 0}건
                    </Typography>
                  </Stack>
                </CardContent>
                <CardActions sx={{ px: 2, pb: 2, flexWrap: "wrap" }}>
                  <Button
                    size="small"
                    startIcon={
                      analyzing[m.id] ? (
                        <CircularProgress size={16} color="inherit" />
                      ) : (
                        <AutoAwesomeRounded />
                      )
                    }
                    disabled={m.active || Boolean(analyzing[m.id])}
                    onClick={() => void analyze(m.id)}
                  >
                    {analyzing[m.id]
                      ? analyzing[m.id] === "cv"
                        ? "CV 분석 중…"
                        : "AI 판독 중…"
                      : "AI 분석"}
                  </Button>
                  <Button
                    size="small"
                    startIcon={<GridOnRounded />}
                    onClick={() => {
                      setSelectedMap(m.id);
                      setDialog("grid");
                    }}
                  >
                    좌석 일괄
                  </Button>
                  <Button
                    size="small"
                    startIcon={<EditLocationAltRounded />}
                    onClick={() => navigate(`/?map=${m.id}&edit=1`)}
                  >
                    배치 편집
                  </Button>
                  {m.active && (
                    <Tooltip title="좌석맵에서 내립니다. 좌석과 이력은 그대로 남습니다">
                      <Button
                        size="small"
                        // 설명은 툴팁으로 두되, 화면 낭독기가 읽는 이름은 조작
                        // 이름 그대로여야 한다.
                        aria-label="게시 내림"
                        startIcon={<UnpublishedRounded />}
                        onClick={() =>
                          void action(
                            `/api/v1/floor-maps/${m.id}/unpublish`,
                            "게시를 내렸습니다",
                          )
                        }
                      >
                        게시 내림
                      </Button>
                    </Tooltip>
                  )}
                  {!m.active && (
                    <>
                      <Button
                        size="small"
                        startIcon={<PublishRounded />}
                        onClick={() =>
                          void action(
                            `/api/v1/floor-maps/${m.id}/publish`,
                            "도면을 게시했습니다",
                          )
                        }
                      >
                        게시
                      </Button>
                      <Button
                        size="small"
                        color="error"
                        startIcon={<DeleteOutlineRounded />}
                        onClick={() => setRemoving(m)}
                      >
                        삭제
                      </Button>
                    </>
                  )}
                </CardActions>
              </Card>
            </Grid>
          ))}
        </Grid>
      )}
      <BuildingDialog
        open={dialog === "building"}
        onClose={() => setDialog(null)}
        done={load}
      />
      <FloorDialog
        open={dialog === "floor"}
        buildings={buildings}
        onClose={() => setDialog(null)}
        done={load}
      />
      <UploadDialog
        open={dialog === "upload"}
        floors={floors}
        onClose={() => setDialog(null)}
        done={load}
      />
      <GridDialog
        open={dialog === "grid"}
        mapId={selectedMap}
        onClose={() => setDialog(null)}
        done={() => {
          setMessage("좌석을 일괄 생성했습니다");
          return load();
        }}
      />
      <Dialog
        open={Boolean(removing)}
        onClose={() => setRemoving(null)}
        fullWidth
        maxWidth="xs"
      >
        <DialogTitle>도면 버전 삭제</DialogTitle>
        <DialogContent>
          <Typography variant="body2">
            {removing?.buildingName} · {removing?.floorName}의{" "}
            <strong>{removing?.version}</strong> 버전과 그 도면의 좌석
            {removing?.seatCount ? ` ${removing.seatCount}석` : ""}을 지웁니다.
            되돌릴 수 없습니다.
          </Typography>
          <Typography variant="caption" color="text.secondary">
            배정이나 변경 이력이 있는 도면은 이력을 지키기 위해 삭제되지
            않습니다.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRemoving(null)}>취소</Button>
          <Button
            color="error"
            variant="contained"
            onClick={() => void remove()}
          >
            삭제
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

function BuildingDialog({
  open,
  onClose,
  done,
}: {
  open: boolean;
  onClose: () => void;
  done: () => Promise<void>;
}) {
  const [name, setName] = useState(""),
    [code, setCode] = useState(""),
    [address, setAddress] = useState(""),
    [error, setError] = useState("");
  const save = async () => {
    try {
      await postJSON("/api/v1/buildings", { name, code, address });
      await done();
      onClose();
      setName("");
      setCode("");
      setAddress("");
    } catch (e) {
      setError(e instanceof Error ? e.message : "저장 실패");
    }
  };
  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="xs">
      <DialogTitle>사업장 추가</DialogTitle>
      <DialogContent>
        <Stack spacing={2} mt={1}>
          {error && <Alert severity="error">{error}</Alert>}
          <TextField
            label="사업장명"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <TextField
            label="코드"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            helperText="예: HQ"
          />
          <TextField
            label="주소 (선택)"
            value={address}
            onChange={(e) => setAddress(e.target.value)}
          />
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>취소</Button>
        <Button variant="contained" onClick={() => void save()}>
          저장
        </Button>
      </DialogActions>
    </Dialog>
  );
}
function FloorDialog({
  open,
  buildings,
  onClose,
  done,
}: {
  open: boolean;
  buildings: Building[];
  onClose: () => void;
  done: () => Promise<void>;
}) {
  const [buildingId, setBuildingId] = useState(""),
    [name, setName] = useState(""),
    [code, setCode] = useState("");
  // 고를 것이 하나뿐이면 미리 골라 둔다. 사업장이 하나인 설치에서 목록을 펼쳐
  // 유일한 항목을 고르게 하면, 저장 버튼이 왜 꺼져 있는지만 헷갈린다.
  useEffect(() => {
    if (open && !buildingId && buildings.length === 1)
      setBuildingId(buildings[0].id);
  }, [open, buildings, buildingId]);
  const save = async () => {
    await postJSON("/api/v1/floors", { buildingId, name, code });
    await done();
    onClose();
  };
  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="xs">
      <DialogTitle>층 추가</DialogTitle>
      <DialogContent>
        <Stack spacing={2} mt={1}>
          <FormControl>
            <Select
              displayEmpty
              value={buildingId}
              onChange={(e) => setBuildingId(e.target.value)}
            >
              <MenuItem value="" disabled>
                사업장 선택
              </MenuItem>
              {buildings.map((x) => (
                <MenuItem key={x.id} value={x.id}>
                  {x.name}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          <TextField
            label="층 이름"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="12층"
          />
          <TextField
            label="층 코드"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            placeholder="12F"
          />
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>취소</Button>
        <Button
          variant="contained"
          disabled={!buildingId || !name || !code}
          onClick={() => void save()}
        >
          저장
        </Button>
      </DialogActions>
    </Dialog>
  );
}
function UploadDialog({
  open,
  floors,
  onClose,
  done,
}: {
  open: boolean;
  floors: Floor[];
  onClose: () => void;
  done: () => Promise<void>;
}) {
  const [floorId, setFloorId] = useState(""),
    [version, setVersion] = useState("1"),
    [file, setFile] = useState<File | null>(null),
    [busy, setBusy] = useState(false);
  useEffect(() => {
    if (open && !floorId && floors.length === 1) setFloorId(floors[0].id);
  }, [open, floors, floorId]);
  const save = async (e: FormEvent) => {
    e.preventDefault();
    if (!file) return;
    setBusy(true);
    const form = new FormData();
    form.append("floorId", floorId);
    form.append("version", version);
    form.append("file", file);
    try {
      await api("/api/v1/floor-maps", { method: "POST", body: form });
      await done();
      onClose();
    } finally {
      setBusy(false);
    }
  };
  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
      <Box component="form" onSubmit={save}>
        <DialogTitle>도면 업로드</DialogTitle>
        <DialogContent>
          <Stack spacing={2} mt={1}>
            <FormControl>
              <Select
                displayEmpty
                value={floorId}
                onChange={(e) => setFloorId(e.target.value)}
              >
                <MenuItem value="" disabled>
                  층 선택
                </MenuItem>
                {floors.map((x) => (
                  <MenuItem key={x.id} value={x.id}>
                    {x.buildingName} · {x.name}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
            <TextField
              label="도면 버전"
              value={version}
              onChange={(e) => setVersion(e.target.value)}
              helperText="예: 2026-08 또는 1"
            />
            <Button
              component="label"
              variant="outlined"
              startIcon={<UploadFileRounded />}
            >
              {file ? file.name : "PNG, JPG, PDF 선택"}
              <input
                hidden
                type="file"
                accept="image/png,image/jpeg,application/pdf"
                onChange={(e) => setFile(e.target.files?.[0] || null)}
              />
            </Button>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose}>취소</Button>
          <Button
            type="submit"
            variant="contained"
            disabled={!floorId || !version || !file || busy}
          >
            업로드
          </Button>
        </DialogActions>
      </Box>
    </Dialog>
  );
}
function GridDialog({
  open,
  mapId,
  onClose,
  done,
}: {
  open: boolean;
  mapId: string;
  onClose: () => void;
  done: () => Promise<void>;
}) {
  const [prefix, setPrefix] = useState("A-"),
    [rows, setRows] = useState(4),
    [columns, setColumns] = useState(8);
  const save = async () => {
    await postJSON("/api/v1/seats/grid", {
      floorMapId: mapId,
      prefix,
      start: 1,
      rows,
      columns,
      x: 0.1,
      y: 0.15,
      seatWidth: 0.06,
      seatHeight: 0.07,
      gapX: 0.035,
      gapY: 0.08,
    });
    await done();
    onClose();
  };
  return (
    <Dialog open={open} onClose={onClose}>
      <DialogTitle>좌석 일괄 생성</DialogTitle>
      <DialogContent>
        <Stack spacing={2} mt={1}>
          <Alert severity="info">
            도면 좌측 상단 기준으로 생성됩니다. 생성 후 좌석맵의 배치 편집에서
            여러 좌석을 선택하고 바로 이동할 수 있습니다.
          </Alert>
          <TextField
            label="좌석 번호 접두어"
            value={prefix}
            onChange={(e) => setPrefix(e.target.value)}
          />
          <Stack direction="row" spacing={2}>
            <TextField
              label="행"
              type="number"
              value={rows}
              onChange={(e) => setRows(Number(e.target.value))}
            />
            <TextField
              label="열"
              type="number"
              value={columns}
              onChange={(e) => setColumns(Number(e.target.value))}
            />
          </Stack>
          <Typography fontWeight={700}>총 {rows * columns}석</Typography>
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>취소</Button>
        <Button
          variant="contained"
          onClick={() => void save()}
          disabled={rows * columns < 1 || rows * columns > 500}
        >
          생성
        </Button>
      </DialogActions>
    </Dialog>
  );
}
