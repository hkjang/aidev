import {
  memo,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type DragEvent,
  type FormEvent,
  type MouseEvent as ReactMouseEvent,
  type KeyboardEvent as ReactKeyboardEvent,
  type PointerEvent as ReactPointerEvent,
} from "react";
import {
  Alert,
  Avatar,
  Box,
  Button,
  Chip,
  CircularProgress,
  Divider,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  IconButton,
  InputAdornment,
  MenuItem,
  Paper,
  Select,
  Skeleton,
  Stack,
  TextField,
  Tooltip,
  Typography,
} from "@mui/material";
import SearchRounded from "@mui/icons-material/SearchRounded";
import ZoomInRounded from "@mui/icons-material/ZoomInRounded";
import ZoomOutRounded from "@mui/icons-material/ZoomOutRounded";
import CenterFocusStrongRounded from "@mui/icons-material/CenterFocusStrongRounded";
import PersonPinCircleRounded from "@mui/icons-material/PersonPinCircleRounded";
import ApartmentRounded from "@mui/icons-material/ApartmentRounded";
import AddRounded from "@mui/icons-material/AddRounded";
import EditRounded from "@mui/icons-material/EditRounded";
import DeleteOutlineRounded from "@mui/icons-material/DeleteOutlineRounded";
import PersonRemoveRounded from "@mui/icons-material/PersonRemoveRounded";
import OpenWithRounded from "@mui/icons-material/OpenWithRounded";
import GridOnRounded from "@mui/icons-material/GridOnRounded";
import GridOffRounded from "@mui/icons-material/GridOffRounded";
import StraightenRounded from "@mui/icons-material/StraightenRounded";
import AutoFixHighRounded from "@mui/icons-material/AutoFixHighRounded";
import UndoRounded from "@mui/icons-material/UndoRounded";
import RedoRounded from "@mui/icons-material/RedoRounded";
import DoneRounded from "@mui/icons-material/DoneRounded";
import VerticalAlignTopRounded from "@mui/icons-material/VerticalAlignTopRounded";
import AlignHorizontalLeftRounded from "@mui/icons-material/AlignHorizontalLeftRounded";
import RotateRightRounded from "@mui/icons-material/RotateRightRounded";
import { useNavigate, useSearchParams } from "react-router-dom";
import { api, patchJSON, postJSON, putJSON } from "../api";
import {
  centerOn,
  FIT_VIEW,
  fitScaleFor,
  focusOn,
  type MapView,
  toViewBox,
  viewRectFor,
  visibleSize,
  zoomAround,
} from "../lib/mapView";
import { readableInk } from "../lib/color";
import {
  readSeatMapParams,
  seatSearchKey,
  writeSeatMapParams,
} from "../lib/seatMapLink";
import {
  type ColorMode,
  commonSeatPrefix,
  deriveGrid,
  needsReviewSeat,
  SEAT_FILTERS,
  seatColor,
  type SeatFilter,
  nextSeatInDirection,
  type SeatDirection,
  seatHighlighted,
  seatLabelLayout,
  seatOrgDetail,
  seatOrgId,
  seatsInReadingOrder,
  shortSeatNo,
  zoneMismatched,
  zoomTier,
} from "../lib/seats";
import { useAuth } from "../auth";
import type {
  Building,
  Employee,
  Floor,
  FloorMap,
  Organization,
  Seat,
  SeatGrid,
} from "../types";

type SeatPosition = Pick<Seat, "id" | "x" | "y" | "rotation">;
type MoveOperation = { before: SeatPosition[]; after: SeatPosition[] };
type ActiveDrag = {
  pointerId: number;
  startX: number;
  startY: number;
  before: SeatPosition[];
  after: SeatPosition[];
};
// 화면을 끌어 옮기는 동안의 상태. 좌석 드래그와 배타적으로 동작한다.
type ActivePan = {
  pointerId: number;
  startX: number;
  startY: number;
  startCx: number;
  startCy: number;
  moved: boolean;
};
const fallbackCanvas = { width: 1000, height: 700 };

// 좌석은 도면 대비 비율 좌표로 저장되므로, viewBox를 도면 원본 비율과
// 동일하게 잡아야 비율 좌표가 도면 픽셀에 1:1로 대응한다.
const canvasFor = (map?: FloorMap) => {
  const width = map?.width ?? 0,
    height = map?.height ?? 0;
  if (width <= 0 || height <= 0) return fallbackCanvas;
  const scale = 1000 / Math.max(width, height);
  return {
    width: Math.round(width * scale),
    height: Math.round(height * scale),
  };
};

const clamp = (value: number, maximum: number) =>
  Math.max(0, Math.min(maximum, value));

// 좌석 하나를 그리는 단위. 화면을 끌거나 확대할 때는 viewBox만 바뀌므로,
// 좌석 속성이 그대로면 다시 그리지 않도록 memo로 감싼다. 500석 도면에서
// 팬 한 프레임마다 전체를 리렌더하던 비용을 없애는 것이 목적이다.
type SeatShapeProps = {
  seat: Seat;
  canvasWidth: number;
  canvasHeight: number;
  /** 도면 좌석들이 공유하는 번호 접두사. 좁은 좌석 안에서는 떼고 보여준다. */
  numberPrefix: string;
  /** 확대 단계. 라벨을 이 값으로 나눠 화면상 크기를 일정하게 지킨다. */
  tier: number;
  /**
   * 탭 순서에 들어가는 좌석인지. 좌석 수백 개를 모두 탭으로 훑게 하면 도면을
   * 지나 다음 조작으로 가는 데만 수백 번을 눌러야 하므로, 한 좌석만 탭으로
   * 들어오고 그 안에서는 방향키로 옮겨 다닌다.
   */
  focusable: boolean;
  onKeyDown: (event: ReactKeyboardEvent<SVGGElement>, seat: Seat) => void;
  onFocus: (seat: Seat) => void;
  active: boolean;
  focused: boolean;
  dimmed: boolean;
  mismatch: boolean;
  needsReview: boolean;
  fill: string;
  darkLabel: boolean;
  editMode: boolean;
  manager: boolean;
  onPointerDown: (event: ReactPointerEvent<SVGGElement>, seat: Seat) => void;
  onSelect: (seat: Seat) => void;
  onEdit: (seat: Seat) => void;
  onDropEmployee: (event: DragEvent, seat: Seat) => void;
};

/** 화면 낭독기와 툴팁이 함께 쓰는 좌석 설명. */
const seatSpeech = (seat: Seat, mismatch: boolean, needsReview: boolean) =>
  `${seat.seatNo}${seat.employeeName ? ` · ${seat.employeeName}` : " · 빈 좌석"}${
    seat.employeeOrganizationName ? ` · ${seat.employeeOrganizationName}` : ""
  }${needsReview ? " · 검토 필요" : ""}${
    mismatch ? ` · 구역 불일치(지정 ${seat.organizationName})` : ""
  }`;

const SeatShape = memo(function SeatShape({
  seat,
  canvasWidth,
  canvasHeight,
  numberPrefix,
  tier,
  focusable,
  onKeyDown,
  onFocus,
  active,
  focused,
  dimmed,
  mismatch,
  needsReview,
  fill,
  darkLabel,
  editMode,
  manager,
  onPointerDown,
  onSelect,
  onEdit,
  onDropEmployee,
}: SeatShapeProps) {
  const left = seat.x * canvasWidth,
    top = seat.y * canvasHeight,
    width = seat.width * canvasWidth,
    height = seat.height * canvasHeight;
  const label = seat.employeeName || shortSeatNo(seat.seatNo, numberPrefix);
  const {
    show: showLabel,
    fontSize,
    text,
  } = seatLabelLayout(label, width, height, tier);
  return (
    <g
      opacity={dimmed ? 0.14 : 1}
      transform={`rotate(${seat.rotation} ${left + width / 2} ${top + height / 2})`}
      role="button"
      data-seat-id={seat.id}
      tabIndex={focusable ? 0 : -1}
      aria-label={seatSpeech(seat, mismatch, needsReview)}
      onKeyDown={(event) => onKeyDown(event, seat)}
      onFocus={() => onFocus(seat)}
      onPointerDown={(event) => onPointerDown(event, seat)}
      onClick={() => onSelect(seat)}
      onDoubleClick={(event) => {
        event.stopPropagation();
        onEdit(seat);
      }}
      onDragOver={(event) => manager && event.preventDefault()}
      onDrop={(event) => onDropEmployee(event, seat)}
      style={{
        cursor: editMode ? "move" : "pointer",
        // 조회 모드에서도 좌석 위에서 끌면 화면이 움직여야 하므로 기본 제스처를 끈다.
        touchAction: "none",
        // 초점 테두리는 좌석 밖으로 밀어 좌석 색과 섞이지 않게 한다.
        outlineOffset: 3,
      }}
    >
      {active && (
        <rect
          x={left - 4}
          y={top - 4}
          width={width + 8}
          height={height + 8}
          rx="9"
          fill="none"
          stroke="#FFB703"
          strokeWidth="2"
          opacity=".55"
        >
          {/* 검색으로 찾아온 좌석 하나만 맥동시켜 눈이 바로 가게 한다. */}
          {focused && (
            <animate
              attributeName="opacity"
              values="0.9;0.2;0.9"
              dur="1.4s"
              repeatCount="6"
            />
          )}
        </rect>
      )}
      <rect
        x={left}
        y={top}
        width={width}
        height={height}
        rx={Math.min(6, Math.min(width, height) * 0.22)}
        fill={fill}
        fillOpacity={seat.employeeId ? 0.95 : 0.85}
        stroke={
          active
            ? "#FFB703"
            : mismatch
              ? "#C1436D"
              : needsReview
                ? "#E79418"
                : "#263E4D"
        }
        strokeWidth={active ? 3 : mismatch || needsReview ? 2 : 1.4}
        strokeDasharray={
          !active && (needsReview || mismatch) ? "5 3" : undefined
        }
      />
      {showLabel && (
        <text
          x={left + width / 2}
          y={top + height / 2}
          textAnchor="middle"
          dominantBaseline="central"
          fontSize={fontSize}
          fontWeight="700"
          fill={darkLabel ? "#203846" : "white"}
          style={{ pointerEvents: "none" }}
        >
          {text}
        </text>
      )}
      <title>{seatSpeech(seat, mismatch, needsReview)}</title>
    </g>
  );
});

// 전역 키 처리는 도면 위에서만 동작해야 한다. 다이얼로그나 목록에 포커스가
// 있을 때 방향키를 가로채면 그 화면이 조작 불가가 된다.
const keyboardTargetsMap = (event: KeyboardEvent) => {
  const target = event.target as HTMLElement | null;
  if (!target) return true;
  if (["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName)) return false;
  if (target.isContentEditable) return false;
  // 열려 있는 모달·메뉴 안이면 그쪽이 우선이다.
  if (target.closest('[role="dialog"], [role="listbox"], [role="menu"]'))
    return false;
  // 좌석에 초점이 있으면 방향키는 좌석 사이를 옮기는 조작이다. 여기서 걸러내지
  // 않으면 좌석 초점이 옮겨 가는 동시에 화면까지 밀려 두 번 움직인다.
  if (target.closest("[data-seat-id]")) return false;
  if (document.querySelector('[role="dialog"]')) return false;
  return true;
};

export function SeatMapPage() {
  const { user } = useAuth(),
    navigate = useNavigate(),
    [searchParams, setSearchParams] = useSearchParams();
  const manager =
    user?.role === "seat_manager" || user?.role === "system_admin";
  const [buildings, setBuildings] = useState<Building[]>([]),
    [floors, setFloors] = useState<Floor[]>([]),
    [maps, setMaps] = useState<FloorMap[]>([]),
    [seats, setSeats] = useState<Seat[]>([]),
    [employees, setEmployees] = useState<Employee[]>([]);
  const [buildingId, setBuildingId] = useState(""),
    [floorId, setFloorId] = useState(""),
    [mapId, setMapId] = useState(""),
    [query, setQuery] = useState(""),
    [selected, setSelected] = useState<Seat | null>(null),
    [loading, setLoading] = useState(true),
    [view, setView] = useState<MapView>(FIT_VIEW),
    [viewport, setViewport] = useState({ width: 0, height: 0 }),
    [error, setError] = useState(""),
    [notice, setNotice] = useState(""),
    [editor, setEditor] = useState<Partial<Seat> | null>(null);
  const [organizations, setOrganizations] = useState<Organization[]>([]),
    [colorMode, setColorMode] = useState<ColorMode>("status"),
    [filters, setFilters] = useState<Set<SeatFilter>>(new Set()),
    [showZones, setShowZones] = useState(false),
    // 범례에서 조직을 누르면 그 조직 좌석만 도드라진다.
    [activeOrg, setActiveOrg] = useState<string | null>(null);
  const [editMode, setEditMode] = useState(
      Boolean(manager && readSeatMapParams(searchParams).edit),
    ),
    [snapEnabled, setSnapEnabled] = useState(true),
    [selectedIds, setSelectedIds] = useState<Set<string>>(new Set()),
    [undoStack, setUndoStack] = useState<MoveOperation[]>([]),
    [redoStack, setRedoStack] = useState<MoveOperation[]>([]),
    [moving, setMoving] = useState(false);
  const dragRef = useRef<ActiveDrag | null>(null);
  const panRef = useRef<ActivePan | null>(null);
  // 화면을 끌고 놓은 직후의 click은 좌석 선택으로 오해되므로 한 번 삼킨다.
  const suppressClickRef = useRef(false);
  // 커서 모양은 렌더에 반영되어야 하므로 ref가 아니라 상태로 둔다.
  const [panning, setPanning] = useState(false);
  // 컨테이너 크기 측정 전에 들어온 이동 요청은 잡아 두었다가 측정 후 적용한다.
  // URL의 ?q= 로 들어와 첫 렌더에서 검색이 실행되는 경우가 여기 해당한다.
  const pendingFocusRef = useRef<Seat | null>(null);
  const stageRef = useRef<HTMLDivElement | null>(null);
  const svgRef = useRef<SVGSVGElement | null>(null);
  const observerRef = useRef<ResizeObserver | null>(null);
  const lastSearchRef = useRef("");
  const loadBase = async () => {
    setLoading(true);
    try {
      const [b, f, m, o] = await Promise.all([
        api<{ items: Building[] }>("/api/v1/buildings"),
        api<{ items: Floor[] }>("/api/v1/floors"),
        api<{ items: FloorMap[] }>("/api/v1/floor-maps"),
        api<{ items: Organization[] }>("/api/v1/organizations"),
      ]);
      setBuildings(b.items);
      setFloors(f.items);
      setMaps(m.items);
      setOrganizations(o.items);
      const requestedMapId = readSeatMapParams(searchParams).mapId;
      const requestedMap = m.items.find((item) => item.id === requestedMapId);
      const requestedFloor = f.items.find(
        (item) => item.id === requestedMap?.floorId,
      );
      const bid =
        requestedFloor?.buildingId || buildingId || b.items[0]?.id || "";
      setBuildingId(bid);
      const fid =
        requestedMap?.floorId ||
        floorId ||
        f.items.find((x) => x.buildingId === bid)?.id ||
        "";
      setFloorId(fid);
      const mid =
        requestedMap?.id ||
        m.items.find((x) => x.floorId === fid && x.active)?.id ||
        m.items.find((x) => x.floorId === fid)?.id ||
        "";
      setMapId(mid);
      if (mid) {
        const data = await api<{ items: Seat[] }>(
          `/api/v1/seats?floorMapId=${mid}`,
        );
        setSeats(data.items);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "데이터를 불러오지 못했습니다");
    } finally {
      setLoading(false);
    }
  };
  useEffect(() => {
    void loadBase();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps
  const buildingFloors = useMemo(
    () => floors.filter((f) => f.buildingId === buildingId),
    [floors, buildingId],
  );
  const floorMaps = useMemo(
    () => maps.filter((m) => m.floorId === floorId),
    [maps, floorId],
  );
  const currentMap = maps.find((m) => m.id === mapId);
  const canvas = useMemo(() => canvasFor(currentMap), [currentMap]);
  // 비율 좌표는 축마다 기준 길이가 달라서, 같은 화면 거리를 만들려면 y값에 이 비율을 곱한다.
  const aspect = canvas.width / canvas.height;
  // 도면에 보정된 격자가 있으면 스냅 간격을 실제 책상 열 간격에 맞춘다.
  const grid = currentMap?.grid ?? null;
  const snapPoint = (x: number, y: number): [number, number] => {
    // 자유 이동에서도 좌표에 부동소수 잡음이 남지 않도록 최소 단위는 유지한다.
    if (!snapEnabled)
      return [Math.round(x / 0.001) * 0.001, Math.round(y / 0.001) * 0.001];
    if (grid)
      return [
        grid.originX +
          Math.round((x - grid.originX) / grid.pitchX) * grid.pitchX,
        grid.originY +
          Math.round((y - grid.originY) / grid.pitchY) * grid.pitchY,
      ];
    const step = 0.005;
    return [
      Math.round(x / step) * step,
      Math.round(y / (step * aspect)) * (step * aspect),
    ];
  };
  // 화면에 꼭 맞는 배율을 1로 두고, 그 위에 view.zoom을 곱해 실제 배율을 만든다.
  // viewBox 종횡비를 컨테이너와 같게 유지하므로 레터박스가 생기지 않는다.
  const fitScale = useMemo(
    () => fitScaleFor(viewport, canvas),
    [viewport, canvas],
  );
  const visible = useMemo(
    () => visibleSize(viewport, canvas, view.zoom),
    [viewport, canvas, view.zoom],
  );
  const viewRect = useMemo(
    () => viewRectFor(view, viewport, canvas),
    [view, viewport, canvas],
  );
  const viewBox = toViewBox(viewRect);
  // SVG 요소 자체를 관측한다. 컨테이너를 재면 안쪽 여백이 함께 잡혀
  // viewBox 종횡비가 어긋나고 화면 이동이 커서를 못 따라간다.
  // effect가 아니라 콜백 ref를 쓰는 이유는, 로딩 중 스켈레톤을 먼저 그리는
  // 구조라 effect 시점에는 노드가 아직 없고 이후 의존성도 바뀌지 않기 때문이다.
  useEffect(() => () => observerRef.current?.disconnect(), []);
  const attachStage = useCallback((node: SVGSVGElement | null) => {
    svgRef.current = node;
    observerRef.current?.disconnect();
    if (!node) {
      setViewport({ width: 0, height: 0 });
      return;
    }
    const measure = () => {
      const box = node.getBoundingClientRect();
      setViewport({ width: box.width, height: box.height });
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(node);
    observerRef.current = observer;
  }, []);
  const applyZoom = (next: number, pivot?: { x: number; y: number }) =>
    setView((current) => zoomAround(current, next, viewport, canvas, pivot));
  const moveCenter = (cx: number, cy: number) =>
    setView((current) => centerOn(current, cx, cy, viewport, canvas));
  // 좌석을 화면 중앙으로 가져오고 최소 배율까지 확대한다. 검색 결과 이동에 쓴다.
  const focusSeat = (seat: Seat) => {
    if (!viewport.width || !viewport.height) {
      pendingFocusRef.current = seat;
      return;
    }
    setView((current) =>
      focusOn(
        current,
        { x: seat.x, y: seat.y, width: seat.width, height: seat.height },
        viewport,
        canvas,
      ),
    );
  };
  useEffect(() => {
    const seat = pendingFocusRef.current;
    if (!seat || !viewport.width || !viewport.height) return;
    pendingFocusRef.current = null;
    setView((current) =>
      focusOn(
        current,
        { x: seat.x, y: seat.y, width: seat.width, height: seat.height },
        viewport,
        canvas,
      ),
    );
  }, [viewport, canvas]);
  const orgColor = useMemo(
    () => new Map(organizations.map((o) => [o.id, o.color])),
    [organizations],
  );
  // 조직 모드에서는 실제로 앉은 직원의 소속 색을 쓰고, 비었으면 좌석 구역 색을 쓴다.
  const fillFor = (seat: Seat) => {
    if (colorMode !== "organization") return seatColor(seat);
    const key = seat.employeeOrganizationId ?? seat.organizationId;
    return (key && orgColor.get(key)) || "#DFE7EB";
  };
  // 좌석에 구역이 지정된 조직마다 경계 상자를 만들어 배경에 깔아준다.
  const zones = useMemo(() => {
    if (!showZones) return [];
    const boxes = new Map<
      string,
      { minX: number; minY: number; maxX: number; maxY: number; count: number }
    >();
    for (const seat of seats) {
      if (!seat.organizationId) continue;
      const box = boxes.get(seat.organizationId);
      const right = seat.x + seat.width,
        bottom = seat.y + seat.height;
      if (!box)
        boxes.set(seat.organizationId, {
          minX: seat.x,
          minY: seat.y,
          maxX: right,
          maxY: bottom,
          count: 1,
        });
      else {
        box.minX = Math.min(box.minX, seat.x);
        box.minY = Math.min(box.minY, seat.y);
        box.maxX = Math.max(box.maxX, right);
        box.maxY = Math.max(box.maxY, bottom);
        box.count += 1;
      }
    }
    return [...boxes.entries()].map(([id, box]) => ({
      id,
      name: organizations.find((o) => o.id === id)?.name ?? "미지정 구역",
      color: orgColor.get(id) || "#8796A1",
      ...box,
    }));
  }, [showZones, seats, organizations, orgColor]);
  // 이 도면에 실제로 좌석이 있는 조직만 좌석 수와 함께 모은다. 조직 색상 모드의
  // 범례로 쓰이며, 상태 기준 범례와 달리 도면마다 내용이 달라진다.
  const mapOrganizations = useMemo(() => {
    const counts = new Map<string, number>();
    for (const seat of seats) {
      const id = seatOrgId(seat);
      if (id) counts.set(id, (counts.get(id) ?? 0) + 1);
    }
    return [...counts.entries()]
      .map(([id, count]) => ({
        id,
        count,
        name: organizations.find((o) => o.id === id)?.name ?? "알 수 없는 조직",
        color: orgColor.get(id) || "#8796A1",
      }))
      .sort((a, b) => b.count - a.count);
  }, [seats, organizations, orgColor]);
  // 실제로 도드라지는 좌석 수. 필터와 조직 강조를 함께 반영해야 화면과 맞는다.
  const highlightedCount = useMemo(
    () =>
      seats.filter((seat) => seatHighlighted(seat, filters, activeOrg)).length,
    [seats, filters, activeOrg],
  );
  const highlighting = filters.size > 0 || activeOrg !== null;
  const toggleFilter = (key: SeatFilter) =>
    setFilters((current) => {
      const next = new Set(current);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  // 휠 확대/축소. React의 onWheel은 passive로 붙어 페이지 스크롤을 막을 수
  // 없으므로 직접 등록한다. 커서 아래 지점을 고정해 원하는 곳을 바로 파고든다.
  useEffect(() => {
    const svg = svgRef.current;
    if (!svg) return;
    const onWheel = (event: WheelEvent) => {
      event.preventDefault();
      const point = toMapPoint(svg, event.clientX, event.clientY);
      applyZoom(
        view.zoom * Math.exp(-event.deltaY * 0.0015),
        point ?? undefined,
      );
    };
    svg.addEventListener("wheel", onWheel, { passive: false });
    return () => svg.removeEventListener("wheel", onWheel);
  });
  const toMapPoint = (
    svg: SVGSVGElement,
    clientX: number,
    clientY: number,
  ): { x: number; y: number } | null => {
    const ctm = svg.getScreenCTM();
    if (!ctm) return null;
    const point = svg.createSVGPoint();
    point.x = clientX;
    point.y = clientY;
    const local = point.matrixTransform(ctm.inverse());
    return { x: local.x / canvas.width, y: local.y / canvas.height };
  };
  const chooseBuilding = (id: string) => {
    setBuildingId(id);
    const fid = floors.find((f) => f.buildingId === id)?.id || "";
    chooseFloor(fid);
  };
  const chooseFloor = (id: string) => {
    setFloorId(id);
    const mid =
      maps.find((m) => m.floorId === id && m.active)?.id ||
      maps.find((m) => m.floorId === id)?.id ||
      "";
    void chooseMap(mid);
  };
  /**
   * 주소를 화면 상태에 맞춘다. 히스토리는 더럽히지 않고(replace), 값이 그대로면
   * 아예 쓰지 않는다 — chooseMap 은 배정·저장 뒤 새로고침 용도로 같은 도면 id 로
   * 다시 불리므로, 그때마다 주소를 쓰면 의미 없는 이동이 쌓인다.
   */
  const syncParams = (next: { mapId?: string; query?: string }) => {
    const params = writeSeatMapParams(searchParams, next);
    if (params.toString() === searchParams.toString()) return;
    setSearchParams(params, { replace: true });
  };
  const chooseMap = async (id: string) => {
    setMapId(id);
    syncParams({ mapId: id });
    setView(FIT_VIEW);
    setActiveOrg(null);
    setSelected(null);
    setSelectedIds(new Set());
    setUndoStack([]);
    setRedoStack([]);
    if (!id) {
      setSeats([]);
      return;
    }
    try {
      const data = await api<{ items: Seat[] }>(
        `/api/v1/seats?floorMapId=${id}`,
      );
      setSeats(data.items);
    } catch (e) {
      setError(e instanceof Error ? e.message : "좌석을 불러오지 못했습니다");
    }
  };
  const runSearch = async (term: string) => {
    if (!term.trim()) {
      setEmployees([]);
      return;
    }
    try {
      const data = await api<{ items: Employee[] }>(
        `/api/v1/employees?q=${encodeURIComponent(term)}&limit=30`,
      );
      setEmployees(data.items);
      const first = data.items.find((x) => x.seatId);
      if (first?.seatId) {
        const found = seats.find((s) => s.id === first.seatId);
        if (found) {
          setSelected(found);
          focusSeat(found);
        }
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "검색하지 못했습니다");
    }
  };
  const search = (event?: FormEvent) => {
    event?.preventDefault();
    // 주소에 q 를 쓰면 아래 effect 가 깨어나 같은 검색을 한 번 더 보낸다.
    // 같은 키를 먼저 세워 두어 그 effect 가 조용히 지나가게 한다.
    lastSearchRef.current = seatSearchKey(mapId, query);
    syncParams({ query });
    void runSearch(query);
  };
  useEffect(() => {
    const term = readSeatMapParams(searchParams).query;
    const key = seatSearchKey(mapId, term);
    // 좌석이 도착하기 전에 검색하면 결과 좌석을 찾지 못해 이동이 조용히 무산된다.
    // mapId 는 좌석 조회보다 먼저 정해지므로 좌석이 실릴 때까지 기다린다.
    if (!term || !mapId || !seats.length || key === lastSearchRef.current)
      return;
    lastSearchRef.current = key;
    setQuery(term);
    void runSearch(term);
  }, [mapId, seats, searchParams]); // eslint-disable-line react-hooks/exhaustive-deps
  const drop = async (event: DragEvent, seat: Seat) => {
    event.preventDefault();
    if (!manager) return;
    const employeeId = event.dataTransfer.getData(
      "application/seaton-employee",
    );
    if (!employeeId) return;
    try {
      await postJSON("/api/v1/seat-assignments", {
        employeeId,
        seatId: seat.id,
        reason: "좌석맵 Drag & Drop",
        source: "manual",
      });
      await chooseMap(mapId);
    } catch (e) {
      setError(e instanceof Error ? e.message : "좌석을 배정하지 못했습니다");
    }
  };
  const openNewSeat = (x = 0.45, y = 0.45) => {
    // 도면 비율과 무관하게 화면에서 정사각형으로 보이도록 높이 비율을 보정한다.
    const width = 0.04,
      height = Math.min(0.5, width * aspect);
    setEditor({
      floorMapId: mapId,
      seatNo: `NEW-${String(seats.length + 1).padStart(3, "0")}`,
      type: "fixed",
      status: "available",
      x: clamp(x - width / 2, 1 - width),
      y: clamp(y - height / 2, 1 - height),
      width,
      height,
      rotation: 0,
    });
  };
  const mapDoubleClick = (event: ReactMouseEvent<SVGSVGElement>) => {
    if (!manager || !editMode) return;
    const point = toMapPoint(event.currentTarget, event.clientX, event.clientY);
    if (!point) return;
    openNewSeat(point.x, point.y);
  };
  const saveSeat = async () => {
    if (!editor) return;
    try {
      const body = {
        seatNo: editor.seatNo,
        // 생성은 null로 미지정, 수정은 빈 문자열로 기존 구역을 해제한다.
        organizationId: editor.id
          ? (editor.organizationId ?? "")
          : editor.organizationId || null,
        type: editor.type,
        status: editor.status,
        x: Number(editor.x),
        y: Number(editor.y),
        width: Number(editor.width),
        height: Number(editor.height),
        rotation: Number(editor.rotation),
      };
      if (editor.id) await patchJSON(`/api/v1/seats/${editor.id}`, body);
      else await postJSON("/api/v1/seats", { ...body, floorMapId: mapId });
      setEditor(null);
      await chooseMap(mapId);
    } catch (e) {
      setError(e instanceof Error ? e.message : "좌석을 저장하지 못했습니다");
    }
  };
  const removeSeat = async () => {
    if (!selected || selected.employeeId) return;
    if (!confirm(`${selected.seatNo} 좌석을 삭제할까요?`)) return;
    try {
      await api(`/api/v1/seats/${selected.id}`, { method: "DELETE" });
      setSelected(null);
      await chooseMap(mapId);
    } catch (e) {
      setError(e instanceof Error ? e.message : "좌석을 삭제하지 못했습니다");
    }
  };
  // 좌석만 다시 읽고 보고 있던 좌석은 그대로 둔다. chooseMap 은 도면을 바꿀 때
  // 쓰는 것이라 첫 줄에서 선택을 비우므로, 배정만 달라진 자리에는 쓸 수 없다.
  const reloadSeats = async () => {
    if (!mapId) return;
    const data = await api<{ items: Seat[] }>(
      `/api/v1/seats?floorMapId=${mapId}`,
    );
    setSeats(data.items);
    setSelected((current) =>
      current ? (data.items.find((s) => s.id === current.id) ?? null) : current,
    );
  };
  const unassignSeat = async () => {
    if (!selected?.employeeId) return;
    if (
      !confirm(
        `${selected.seatNo} 좌석에서 ${selected.employeeName ?? "직원"} 님의 배정을 해제할까요?`,
      )
    )
      return;
    try {
      await api(`/api/v1/seat-assignments/${selected.id}`, {
        method: "DELETE",
      });
      await reloadSeats();
    } catch (e) {
      setError(e instanceof Error ? e.message : "배정을 해제하지 못했습니다");
    }
  };
  const updateLocalPositions = (positions: SeatPosition[]) => {
    const byID = new Map(positions.map((position) => [position.id, position]));
    setSeats((current) =>
      current.map((seat) => {
        const position = byID.get(seat.id);
        return position ? { ...seat, ...position } : seat;
      }),
    );
    setSelected((current) => {
      if (!current) return current;
      const position = byID.get(current.id);
      return position ? { ...current, ...position } : current;
    });
  };
  const persistPositions = async (positions: SeatPosition[]) =>
    patchJSON<{ updated: number }>("/api/v1/seats/bulk", {
      updates: positions,
    });
  const commitOperation = async (operation: MoveOperation) => {
    if (
      operation.before.every((position, index) => {
        const after = operation.after[index];
        return (
          after &&
          position.x === after.x &&
          position.y === after.y &&
          position.rotation === after.rotation
        );
      })
    )
      return;
    setMoving(true);
    try {
      await persistPositions(operation.after);
      setUndoStack((current) => [...current.slice(-39), operation]);
      setRedoStack([]);
    } catch (e) {
      updateLocalPositions(operation.before);
      setError(
        e instanceof Error ? e.message : "좌석 위치를 저장하지 못했습니다",
      );
    } finally {
      setMoving(false);
    }
  };
  const beginSeatMove = (event: ReactPointerEvent<SVGGElement>, seat: Seat) => {
    if (!manager || !editMode || event.button !== 0) return;
    event.preventDefault();
    event.stopPropagation();
    setSelected(seat);
    if (event.shiftKey) {
      setSelectedIds((current) => {
        const next = new Set(current);
        if (next.has(seat.id)) next.delete(seat.id);
        else next.add(seat.id);
        return next;
      });
      return;
    }
    const ids = selectedIds.has(seat.id)
      ? selectedIds
      : new Set<string>([seat.id]);
    setSelectedIds(new Set(ids));
    const before = seats
      .filter((item) => ids.has(item.id))
      .map(({ id, x, y, rotation }) => ({ id, x, y, rotation }));
    const svg = event.currentTarget.ownerSVGElement;
    if (!svg) return;
    const start = toMapPoint(svg, event.clientX, event.clientY);
    if (!start) return;
    event.currentTarget.setPointerCapture(event.pointerId);
    dragRef.current = {
      pointerId: event.pointerId,
      startX: start.x,
      startY: start.y,
      before,
      after: before,
    };
  };
  const moveSeats = (event: ReactPointerEvent<SVGSVGElement>) => {
    const drag = dragRef.current;
    if (!drag || drag.pointerId !== event.pointerId) return;
    const point = toMapPoint(event.currentTarget, event.clientX, event.clientY);
    if (!point) return;
    const dx = point.x - drag.startX;
    const dy = point.y - drag.startY;
    const after = drag.before.map((position) => {
      const seat = seats.find((item) => item.id === position.id);
      const [sx, sy] = snapPoint(position.x + dx, position.y + dy);
      return {
        ...position,
        x: clamp(sx, 1 - (seat?.width ?? 0)),
        y: clamp(sy, 1 - (seat?.height ?? 0)),
      };
    });
    drag.after = after;
    updateLocalPositions(after);
  };
  const finishSeatMove = (event: ReactPointerEvent<SVGSVGElement>) => {
    const drag = dragRef.current;
    if (!drag || drag.pointerId !== event.pointerId) return;
    dragRef.current = null;
    void commitOperation({ before: drag.before, after: drag.after });
  };
  // 좌석이 아닌 빈 영역에서 시작한 드래그는 화면 이동으로 처리한다.
  const beginPan = (event: ReactPointerEvent<SVGSVGElement>) => {
    // 이미 끌고 있는 포인터가 있으면 무시한다. 두 번째 손가락이 팬을 가로채면
    // 화면이 튀고, 그 손가락을 떼는 순간 첫 손가락의 이동이 갈 곳을 잃는다.
    if (dragRef.current || panRef.current) return;
    panRef.current = {
      pointerId: event.pointerId,
      startX: event.clientX,
      startY: event.clientY,
      startCx: view.cx,
      startCy: view.cy,
      moved: false,
    };
    // 여기서 포인터를 캡처하면 뒤따르는 click이 SVG로 재지정되어 좌석 선택이
    // 죽는다. 실제로 임계값을 넘어 움직인 뒤에 캡처한다.
  };
  const movePan = (event: ReactPointerEvent<SVGSVGElement>) => {
    const pan = panRef.current;
    if (!pan || pan.pointerId !== event.pointerId) return false;
    const scale = fitScale * view.zoom;
    if (scale <= 0) return true;
    const dx = (event.clientX - pan.startX) / scale / canvas.width;
    const dy = (event.clientY - pan.startY) / scale / canvas.height;
    if (
      !pan.moved &&
      (Math.abs(event.clientX - pan.startX) > 3 ||
        Math.abs(event.clientY - pan.startY) > 3)
    ) {
      pan.moved = true;
      event.currentTarget.setPointerCapture(event.pointerId);
      setPanning(true);
    }
    if (!pan.moved) return true;
    moveCenter(pan.startCx - dx, pan.startCy - dy);
    return true;
  };
  const endPan = (event: ReactPointerEvent<SVGSVGElement>) => {
    const pan = panRef.current;
    if (!pan || pan.pointerId !== event.pointerId) return false;
    panRef.current = null;
    suppressClickRef.current = pan.moved;
    if (pan.moved) {
      if (event.currentTarget.hasPointerCapture(event.pointerId))
        event.currentTarget.releasePointerCapture(event.pointerId);
      setPanning(false);
    }
    return pan.moved;
  };
  const canvasPointerDown = (event: ReactPointerEvent<SVGSVGElement>) => {
    // 좌석 위에서 시작한 편집 드래그는 좌석 쪽에서 이미 전파를 멈춘다.
    if (event.button !== 0 && event.button !== 1) return;
    suppressClickRef.current = false;
    if (event.target === event.currentTarget) {
      setSelected(null);
      setSelectedIds(new Set());
    }
    beginPan(event);
  };
  const canvasPointerMove = (event: ReactPointerEvent<SVGSVGElement>) => {
    if (movePan(event)) return;
    moveSeats(event);
  };
  const canvasPointerUp = (event: ReactPointerEvent<SVGSVGElement>) => {
    endPan(event);
    finishSeatMove(event);
  };
  // memo된 좌석이 매 렌더 무효화되지 않도록 콜백을 고정한다. 바뀌는 값은
  // ref로 읽어 콜백 신원(identity)을 유지한다.
  // 초점이 옮겨 간 좌석이 화면 밖이면 그 좌석이 보이도록 화면을 옮긴다.
  // 이미 보이는 좌석까지 매번 가운데로 끌어오면 방향키를 누를 때마다 도면이
  // 요동쳐 어디를 보고 있는지 잃는다.
  const revealSeat = (seat: Seat) => {
    if (!viewport.width || !viewport.height) return;
    const rect = viewRectFor(view, viewport, canvas);
    const left = seat.x * canvas.width,
      top = seat.y * canvas.height,
      right = left + seat.width * canvas.width,
      bottom = top + seat.height * canvas.height;
    const inside =
      left >= rect.x &&
      right <= rect.x + rect.width &&
      top >= rect.y &&
      bottom <= rect.y + rect.height;
    if (inside) return;
    moveCenter(seat.x + seat.width / 2, seat.y + seat.height / 2);
  };
  const seatActionsRef = useRef({
    beginSeatMove,
    drop,
    setEditor,
    setSelected,
    revealSeat,
    editMode,
    manager,
  });
  seatActionsRef.current = {
    beginSeatMove,
    drop,
    setEditor,
    setSelected,
    revealSeat,
    editMode,
    manager,
  };
  const onSeatPointerDown = useCallback(
    (event: ReactPointerEvent<SVGGElement>, seat: Seat) =>
      seatActionsRef.current.beginSeatMove(event, seat),
    [],
  );
  const onSeatSelect = useCallback((seat: Seat) => {
    if (suppressClickRef.current) return;
    if (!seatActionsRef.current.editMode)
      seatActionsRef.current.setSelected(seat);
  }, []);
  const onSeatEdit = useCallback((seat: Seat) => {
    const actions = seatActionsRef.current;
    if (actions.manager && actions.editMode) actions.setEditor(seat);
  }, []);
  const onSeatDrop = useCallback(
    (event: DragEvent, seat: Seat) =>
      void seatActionsRef.current.drop(event, seat),
    [],
  );
  // 탭으로 들어오는 좌석 하나. 그 안에서는 방향키로 옮겨 다닌다.
  const [rovingId, setRovingId] = useState("");
  // 방향키로 옮길 때는 초점을 실제로 그 좌석에 준다. 초점이 따라가야 화면
  // 낭독기가 어디로 갔는지 읽어 준다.
  const focusSeatNode = (seatId: string) =>
    stageRef.current
      ?.querySelector<SVGGElement>(`[data-seat-id="${seatId}"]`)
      ?.focus();
  const orderedSeats = useMemo(() => seatsInReadingOrder(seats), [seats]);
  const rovingSeatId =
    orderedSeats.find((seat) => seat.id === rovingId)?.id ??
    orderedSeats[0]?.id ??
    "";
  const seatKeyDown = useCallback(
    (event: ReactKeyboardEvent<SVGGElement>, seat: Seat) => {
      const move: Record<string, SeatDirection> = {
        ArrowLeft: "left",
        ArrowRight: "right",
        ArrowUp: "up",
        ArrowDown: "down",
      };
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        if (!seatActionsRef.current.editMode)
          seatActionsRef.current.setSelected(seat);
        return;
      }
      // 편집 모드의 방향키는 좌석을 옮기는 조작이므로 그쪽에 넘긴다.
      if (editMode || !move[event.key]) {
        if (event.key === "Home" || event.key === "End") {
          const target =
            event.key === "Home"
              ? orderedSeats[0]
              : orderedSeats[orderedSeats.length - 1];
          if (target) {
            event.preventDefault();
            focusSeatNode(target.id);
          }
        }
        return;
      }
      const next = nextSeatInDirection(orderedSeats, seat.id, move[event.key]);
      if (!next) return;
      event.preventDefault();
      event.stopPropagation();
      focusSeatNode(next.id);
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [orderedSeats, editMode],
  );
  const seatFocused = useCallback((seat: Seat) => {
    setRovingId(seat.id);
    seatActionsRef.current.revealSeat(seat);
  }, []);
  const seatNumberPrefix = useMemo(
    () => commonSeatPrefix(seats.map((seat) => seat.seatNo)),
    [seats],
  );
  // 라벨 크기는 확대 배율에 반비례해야 화면상 크기가 유지되지만, 배율을 그대로
  // 쓰면 휠을 굴릴 때마다 좌석 전체를 다시 그린다. 단계로 뭉쳐 다시 그리는 횟수를
  // 전체 배율 구간에서 네 번으로 묶는다.
  const labelTier = zoomTier(view.zoom);
  // 좌석 레이어는 좌석 데이터와 표시 기준이 바뀔 때만 다시 만든다.
  // 화면 이동은 viewBox만 바꾸므로 이 목록을 건드리지 않는다.
  const seatLayer = useMemo(
    () =>
      seats.map((seat) => (
        <SeatShape
          key={seat.id}
          seat={seat}
          canvasWidth={canvas.width}
          canvasHeight={canvas.height}
          numberPrefix={seatNumberPrefix}
          tier={labelTier}
          focusable={seat.id === rovingSeatId}
          onKeyDown={seatKeyDown}
          onFocus={seatFocused}
          active={selectedIds.has(seat.id) || selected?.id === seat.id}
          focused={selected?.id === seat.id}
          dimmed={!seatHighlighted(seat, filters, activeOrg)}
          mismatch={zoneMismatched(seat)}
          needsReview={needsReviewSeat(seat)}
          fill={fillFor(seat)}
          darkLabel={readableInk(fillFor(seat)) === "#203846"}
          editMode={editMode}
          manager={Boolean(manager)}
          onPointerDown={onSeatPointerDown}
          onSelect={onSeatSelect}
          onEdit={onSeatEdit}
          onDropEmployee={onSeatDrop}
        />
      )),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [
      seats,
      canvas,
      seatNumberPrefix,
      labelTier,
      rovingSeatId,
      seatKeyDown,
      seatFocused,
      selectedIds,
      selected?.id,
      filters,
      activeOrg,
      colorMode,
      orgColor,
      editMode,
      manager,
    ],
  );
  // 미니맵의 좌석 점도 화면 이동마다 다시 그릴 이유가 없다. 팬 중에는
  // 현재 영역 사각형만 움직이면 되므로 좌석 목록을 따로 묶어 둔다.
  const minimapSeats = useMemo(
    () =>
      seats.map((seat) => (
        <rect
          key={seat.id}
          x={seat.x * canvas.width}
          y={seat.y * canvas.height}
          width={Math.max(3, seat.width * canvas.width)}
          height={Math.max(3, seat.height * canvas.height)}
          fill={fillFor(seat)}
          opacity={seatHighlighted(seat, filters, activeOrg) ? 0.9 : 0.15}
        />
      )),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [seats, canvas, filters, activeOrg, colorMode, orgColor],
  );
  // 미니맵의 한 점을 화면 중심으로 삼는다. 클릭과 드래그가 같은 경로를 쓴다.
  const moveViewToMinimap = (event: ReactPointerEvent<SVGSVGElement>) => {
    const point = toMapPoint(event.currentTarget, event.clientX, event.clientY);
    if (!point) return;
    moveCenter(point.x, point.y);
  };
  const applyTransform = (
    kind: "left" | "top" | "rotate" | "nudge",
    dx = 0,
    dy = 0,
  ) => {
    const chosen = seats.filter((seat) => selectedIds.has(seat.id));
    if (!chosen.length) return;
    const before = chosen.map(({ id, x, y, rotation }) => ({
      id,
      x,
      y,
      rotation,
    }));
    const left = Math.min(...chosen.map((seat) => seat.x));
    const top = Math.min(...chosen.map((seat) => seat.y));
    const after = chosen.map((seat) => ({
      id: seat.id,
      x: kind === "left" ? left : clamp(seat.x + dx, 1 - seat.width),
      y: kind === "top" ? top : clamp(seat.y + dy, 1 - seat.height),
      rotation: kind === "rotate" ? (seat.rotation + 90) % 360 : seat.rotation,
    }));
    updateLocalPositions(after);
    void commitOperation({ before, after });
  };
  const undo = async () => {
    const operation = undoStack.at(-1);
    if (!operation || moving) return;
    setMoving(true);
    updateLocalPositions(operation.before);
    try {
      await persistPositions(operation.before);
      setUndoStack((current) => current.slice(0, -1));
      setRedoStack((current) => [...current, operation]);
    } catch (e) {
      updateLocalPositions(operation.after);
      setError(
        e instanceof Error ? e.message : "실행 취소를 저장하지 못했습니다",
      );
    } finally {
      setMoving(false);
    }
  };
  const redo = async () => {
    const operation = redoStack.at(-1);
    if (!operation || moving) return;
    setMoving(true);
    updateLocalPositions(operation.after);
    try {
      await persistPositions(operation.after);
      setRedoStack((current) => current.slice(0, -1));
      setUndoStack((current) => [...current, operation]);
    } catch (e) {
      updateLocalPositions(operation.before);
      setError(
        e instanceof Error ? e.message : "다시 실행을 저장하지 못했습니다",
      );
    } finally {
      setMoving(false);
    }
  };
  const applyGridToMap = (next: SeatGrid | null) =>
    setMaps((current) =>
      current.map((item) =>
        item.id === mapId ? { ...item, grid: next } : item,
      ),
    );
  const calibrateGrid = async () => {
    const selection = seats.filter((seat) => selectedIds.has(seat.id));
    const next = deriveGrid(selection);
    if (!next) {
      setError(
        "격자를 계산할 수 없습니다. 가로·세로로 떨어진 좌석을 2개 이상 선택하세요",
      );
      return;
    }
    try {
      const saved = await putJSON<SeatGrid>(
        `/api/v1/floor-maps/${mapId}/grid`,
        next,
      );
      const applied = saved ?? next;
      applyGridToMap(applied);
      setNotice(
        `격자를 보정했습니다 · 가로 ${(applied.pitchX * 100).toFixed(1)}% · 세로 ${(applied.pitchY * 100).toFixed(1)}%`,
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : "격자를 저장하지 못했습니다");
    }
  };
  const clearGrid = async () => {
    try {
      await putJSON(`/api/v1/floor-maps/${mapId}/grid`, {});
      applyGridToMap(null);
      setNotice("격자 보정을 해제했습니다");
    } catch (e) {
      setError(e instanceof Error ? e.message : "격자를 해제하지 못했습니다");
    }
  };
  const alignToGrid = async () => {
    const ids = [...selectedIds];
    try {
      const result = await postJSON<{
        aligned: number;
        maxShift: number;
        warning?: string;
      }>(`/api/v1/floor-maps/${mapId}/seats/align`, { seatIds: ids });
      setUndoStack([]);
      setRedoStack([]);
      await chooseMap(mapId);
      const scope = ids.length ? "" : " (도면 전체)";
      if (result.warning) setError(result.warning);
      setNotice(
        `${result.aligned}개 좌석을 격자에 정렬했습니다${scope} · 최대 이동 ${(result.maxShift * 100).toFixed(1)}%`,
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : "좌석을 정렬하지 못했습니다");
    }
  };
  useEffect(() => {
    if (!editMode) return;
    const keyboard = (event: KeyboardEvent) => {
      if (!keyboardTargetsMap(event)) return;
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "z") {
        event.preventDefault();
        if (event.shiftKey) void redo();
        else void undo();
        return;
      }
      if (event.key === "Escape") {
        setSelectedIds(new Set());
        setSelected(null);
        return;
      }
      const distance = event.shiftKey ? 0.02 : 0.005;
      const vertical = distance * aspect;
      const delta: Record<string, [number, number]> = {
        ArrowLeft: [-distance, 0],
        ArrowRight: [distance, 0],
        ArrowUp: [0, -vertical],
        ArrowDown: [0, vertical],
      };
      if (delta[event.key]) {
        event.preventDefault();
        applyTransform("nudge", ...delta[event.key]);
      }
    };
    window.addEventListener("keydown", keyboard);
    return () => window.removeEventListener("keydown", keyboard);
  });
  // 조회 모드에서는 방향키가 화면을 옮기고, +/-로 확대, 0으로 전체 보기를 한다.
  // 편집 모드의 방향키는 좌석 미세 이동이라 그쪽 핸들러가 먼저 가져간다.
  useEffect(() => {
    if (editMode) return;
    const keyboard = (event: KeyboardEvent) => {
      if (!keyboardTargetsMap(event)) return;
      const stepX = (visible.width / canvas.width) * 0.25;
      const stepY = (visible.height / canvas.height) * 0.25;
      const pan: Record<string, [number, number]> = {
        ArrowLeft: [-stepX, 0],
        ArrowRight: [stepX, 0],
        ArrowUp: [0, -stepY],
        ArrowDown: [0, stepY],
      };
      if (pan[event.key]) {
        event.preventDefault();
        const [dx, dy] = pan[event.key];
        moveCenter(view.cx + dx, view.cy + dy);
        return;
      }
      if (event.key === "+" || event.key === "=") {
        event.preventDefault();
        applyZoom(view.zoom * 1.35);
      } else if (event.key === "-" || event.key === "_") {
        event.preventDefault();
        applyZoom(view.zoom / 1.35);
      } else if (event.key === "0") {
        event.preventDefault();
        setView(FIT_VIEW);
      }
    };
    window.addEventListener("keydown", keyboard);
    return () => window.removeEventListener("keydown", keyboard);
  });
  if (loading)
    return (
      <Box sx={{ p: { xs: 2, md: 3 } }}>
        <Skeleton height={60} />
        <Skeleton variant="rounded" height="70vh" />
      </Box>
    );
  return (
    <Box
      sx={{
        p: { xs: 2, md: 3 },
        height: { md: "calc(100vh - 72px)" },
        display: "flex",
        flexDirection: "column",
        gap: 2,
      }}
    >
      {error && (
        <Alert severity="error" onClose={() => setError("")}>
          {error}
        </Alert>
      )}
      {notice && (
        <Alert severity="success" onClose={() => setNotice("")}>
          {notice}
        </Alert>
      )}
      <Box
        sx={{
          display: "flex",
          alignItems: { xs: "stretch", md: "center" },
          gap: 1.5,
          flexDirection: { xs: "column", md: "row" },
        }}
      >
        <Box>
          <Typography variant="h5">좌석맵</Typography>
          <Typography variant="body2" color="text.secondary">
            사람과 조직의 현재 위치를 한눈에 확인하세요.
          </Typography>
        </Box>
        <Box sx={{ flex: 1 }} />
        {manager && currentMap && (
          <Stack direction="row" spacing={1}>
            {editMode && (
              <Button
                variant="outlined"
                startIcon={<AddRounded />}
                onClick={() => openNewSeat()}
              >
                좌석 추가
              </Button>
            )}
            <Button
              variant={editMode ? "contained" : "outlined"}
              startIcon={editMode ? <DoneRounded /> : <OpenWithRounded />}
              onClick={() => {
                setEditMode((value) => !value);
                setSelectedIds(new Set());
              }}
            >
              {editMode ? "편집 완료" : "배치 편집"}
            </Button>
          </Stack>
        )}
        <Stack direction="row" spacing={1} sx={{ overflowX: "auto" }}>
          <FormControl sx={{ minWidth: 145 }}>
            <Select
              value={buildingId}
              displayEmpty
              onChange={(e) => chooseBuilding(e.target.value)}
            >
              {buildings.length ? (
                buildings.map((x) => (
                  <MenuItem key={x.id} value={x.id}>
                    {x.name}
                  </MenuItem>
                ))
              ) : (
                <MenuItem value="">사업장 없음</MenuItem>
              )}
            </Select>
          </FormControl>
          <FormControl sx={{ minWidth: 120 }}>
            <Select
              value={floorId}
              displayEmpty
              onChange={(e) => chooseFloor(e.target.value)}
            >
              {buildingFloors.map((x) => (
                <MenuItem key={x.id} value={x.id}>
                  {x.name}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          {floorMaps.length > 1 && (
            <FormControl sx={{ minWidth: 110 }}>
              <Select
                value={mapId}
                onChange={(e) => void chooseMap(e.target.value)}
              >
                {floorMaps.map((x) => (
                  <MenuItem key={x.id} value={x.id}>
                    V{x.version}
                    {x.active ? " · 게시" : ""}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          )}
        </Stack>
      </Box>
      {!currentMap ? (
        <Paper
          sx={{
            flex: 1,
            minHeight: 420,
            display: "grid",
            placeItems: "center",
            borderStyle: "dashed",
            boxShadow: "none",
          }}
        >
          <Stack alignItems="center" spacing={2}>
            <Box
              sx={{
                width: 64,
                height: 64,
                borderRadius: "50%",
                display: "grid",
                placeItems: "center",
                bgcolor: "rgba(8,126,139,.09)",
                color: "primary.main",
              }}
            >
              <ApartmentRounded fontSize="large" />
            </Box>
            <Box textAlign="center">
              <Typography variant="h6">표시할 좌석맵이 없습니다</Typography>
              <Typography color="text.secondary">
                관리자가 사업장과 도면을 등록하면 이곳에서 바로 찾을 수 있어요.
              </Typography>
            </Box>
            {manager && (
              <Button
                variant="contained"
                startIcon={<AddRounded />}
                onClick={() => navigate("/admin/maps")}
              >
                첫 도면 등록
              </Button>
            )}
          </Stack>
        </Paper>
      ) : (
        <Box
          sx={{
            display: "grid",
            // 세 칸의 최소 폭을 합치면 1280px 노트북 화면을 넘겨 상세 패널이
            // 잘려 나갔다. 도면 칸이 줄어들 수 있게 두고, 넓은 화면에서만 양옆을
            // 넉넉히 준다.
            gridTemplateColumns: {
              xs: "1fr",
              lg: "220px minmax(0, 1fr) 240px",
              xl: "260px minmax(0, 1fr) 280px",
            },
            gap: 2,
            flex: 1,
            minHeight: 0,
          }}
        >
          <Paper
            sx={{
              p: 2,
              display: "flex",
              flexDirection: "column",
              minHeight: { xs: 240, lg: 0 },
              overflow: "hidden",
            }}
          >
            <Box component="form" onSubmit={search}>
              <TextField
                fullWidth
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="이름, 사번, 조직 검색"
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
            </Box>
            <Typography
              variant="caption"
              color="text.secondary"
              sx={{ mt: 1.5, mb: 1 }}
            >
              {employees.length
                ? `${employees.length}명 검색됨`
                : "사람 또는 조직을 검색하세요"}
            </Typography>
            <Stack spacing={0.75} sx={{ overflowY: "auto" }}>
              {employees.map((e) => {
                const show = () => {
                  const seat = seats.find((s) => s.id === e.seatId);
                  if (!seat) return;
                  setSelected(seat);
                  focusSeat(seat);
                };
                return (
                  <Box
                    key={e.id}
                    draggable={manager}
                    onDragStart={(event) =>
                      event.dataTransfer.setData(
                        "application/seaton-employee",
                        e.id,
                      )
                    }
                    // 검색 결과는 마우스로만 고를 수 있었다. 좌석은 도면 위 그림이라
                    // 탭으로 닿지 않으므로, 키보드 사용자에게는 이 목록이 좌석을
                    // 고르는 유일한 길이다.
                    role={e.seatId ? "button" : undefined}
                    tabIndex={e.seatId ? 0 : undefined}
                    aria-label={
                      e.seatId
                        ? `${e.name} · ${e.organizationName || "소속 없음"} · ${e.seatNo} 좌석 보기`
                        : undefined
                    }
                    onKeyDown={(event) => {
                      if (event.key === "Enter" || event.key === " ") {
                        event.preventDefault();
                        show();
                      }
                    }}
                    onClick={show}
                    sx={{
                      display: "flex",
                      gap: 1.2,
                      p: 1,
                      borderRadius: 2,
                      cursor: e.seatId ? "pointer" : "default",
                      "&:hover": { bgcolor: "#F1F6F7" },
                      "&:focus-visible": {
                        outline: "2px solid",
                        outlineColor: "primary.main",
                        outlineOffset: 2,
                      },
                    }}
                  >
                    <Avatar
                      sx={{
                        width: 34,
                        height: 34,
                        fontSize: 13,
                        bgcolor: e.seatId ? "primary.main" : "grey.400",
                      }}
                    >
                      {e.name.slice(0, 1)}
                    </Avatar>
                    <Box minWidth={0}>
                      <Typography variant="body2" fontWeight={700} noWrap>
                        {e.name}{" "}
                        <Typography
                          component="span"
                          variant="caption"
                          color="text.secondary"
                        >
                          {e.employeeNo}
                        </Typography>
                      </Typography>
                      <Typography
                        variant="caption"
                        color="text.secondary"
                        noWrap
                        display="block"
                      >
                        {e.organizationName || "소속 없음"} ·{" "}
                        {e.seatNo || "미배정"}
                      </Typography>
                    </Box>
                  </Box>
                );
              })}
            </Stack>
          </Paper>
          <Paper
            sx={{
              display: "flex",
              flexDirection: "column",
              overflow: "hidden",
              minHeight: { xs: 430, lg: 0 },
              bgcolor: "#E9EFF2",
            }}
          >
            {/* 색상 기준과 좌석 필터. 도면을 가리지 않도록 캔버스 위쪽에 따로 놓는다. */}
            <Box
              sx={{
                display: "flex",
                flexWrap: "wrap",
                alignItems: "center",
                rowGap: 0.75,
                columnGap: 1,
                px: 1.5,
                py: 1,
                bgcolor: "#fff",
                borderBottom: "1px solid rgba(14,45,62,.1)",
              }}
            >
              <Stack direction="row" spacing={0.5} alignItems="center">
                <Chip
                  size="small"
                  label="상태 색"
                  variant={colorMode === "status" ? "filled" : "outlined"}
                  color={colorMode === "status" ? "primary" : "default"}
                  onClick={() => setColorMode("status")}
                />
                <Chip
                  size="small"
                  label="조직 색"
                  variant={colorMode === "organization" ? "filled" : "outlined"}
                  color={colorMode === "organization" ? "primary" : "default"}
                  onClick={() => setColorMode("organization")}
                />
                <Tooltip title="좌석에 지정된 조직 구역을 배경으로 표시">
                  <Chip
                    size="small"
                    label="구역"
                    variant={showZones ? "filled" : "outlined"}
                    color={showZones ? "secondary" : "default"}
                    onClick={() => setShowZones((v) => !v)}
                  />
                </Tooltip>
              </Stack>
              <Divider orientation="vertical" flexItem sx={{ my: 0.25 }} />
              <Box
                sx={{ display: "flex", flexWrap: "wrap", gap: 0.5 }}
                role="group"
                aria-label="좌석 필터"
              >
                {SEAT_FILTERS.map((filter) => (
                  <Chip
                    key={filter.key}
                    size="small"
                    label={filter.label}
                    variant={filters.has(filter.key) ? "filled" : "outlined"}
                    color={filters.has(filter.key) ? "primary" : "default"}
                    onClick={() => toggleFilter(filter.key)}
                  />
                ))}
              </Box>
              {activeOrg !== null && (
                <Chip
                  size="small"
                  color="secondary"
                  label={`${
                    mapOrganizations.find((o) => o.id === activeOrg)?.name ??
                    "조직"
                  } 강조 해제`}
                  onDelete={() => setActiveOrg(null)}
                  onClick={() => setActiveOrg(null)}
                />
              )}
              {editMode && (
                <Box
                  sx={{
                    display: "flex",
                    alignItems: "center",
                    gap: 0.5,
                    p: 0.5,
                    bgcolor: "rgba(7,26,43,.92)",
                    color: "white",
                    borderRadius: 2,
                    boxShadow: 3,
                  }}
                >
                  <Tooltip
                    title={
                      grid
                        ? `보정된 격자에 스냅 · 가로 ${(grid.pitchX * 100).toFixed(1)}% 세로 ${(grid.pitchY * 100).toFixed(1)}%`
                        : "고정 간격에 스냅 · 도면 격자를 보정하면 실제 책상 간격을 따릅니다"
                    }
                  >
                    <Chip
                      size="small"
                      icon={<GridOnRounded />}
                      label={
                        snapEnabled
                          ? grid
                            ? "도면 격자"
                            : "격자 스냅"
                          : "자유 이동"
                      }
                      onClick={() => setSnapEnabled((value) => !value)}
                      sx={{
                        bgcolor: !snapEnabled
                          ? "rgba(255,255,255,.12)"
                          : grid
                            ? "rgba(8,126,139,.4)"
                            : "rgba(255,183,3,.22)",
                        color: "white",
                        fontWeight: 600,
                        "& .MuiChip-icon": { color: "inherit" },
                      }}
                    />
                  </Tooltip>
                  <Tooltip
                    title={
                      selectedIds.size > 1
                        ? "선택한 좌석의 간격으로 도면 격자를 보정합니다"
                        : "가로·세로로 떨어진 좌석을 2개 이상 선택하세요"
                    }
                  >
                    <span>
                      <IconButton
                        size="small"
                        disabled={selectedIds.size < 2 || moving}
                        onClick={() => void calibrateGrid()}
                        sx={{ color: "white" }}
                        aria-label="선택 좌석으로 격자 보정"
                      >
                        <StraightenRounded />
                      </IconButton>
                    </span>
                  </Tooltip>
                  <Tooltip
                    title={
                      grid
                        ? selectedIds.size
                          ? `선택한 ${selectedIds.size}개 좌석을 격자에 정렬`
                          : "도면 전체 좌석을 격자에 정렬"
                        : "먼저 도면 격자를 보정하세요"
                    }
                  >
                    <span>
                      <IconButton
                        size="small"
                        disabled={!grid || moving}
                        onClick={() => void alignToGrid()}
                        sx={{ color: "white" }}
                        aria-label="격자에 정렬"
                      >
                        <AutoFixHighRounded />
                      </IconButton>
                    </span>
                  </Tooltip>
                  {grid && (
                    <Tooltip title="격자 보정 해제">
                      <span>
                        <IconButton
                          size="small"
                          disabled={moving}
                          onClick={() => void clearGrid()}
                          sx={{ color: "white" }}
                          aria-label="격자 보정 해제"
                        >
                          <GridOffRounded />
                        </IconButton>
                      </span>
                    </Tooltip>
                  )}
                  <Tooltip title="실행 취소 · Ctrl/⌘ Z">
                    <span>
                      <IconButton
                        size="small"
                        disabled={!undoStack.length || moving}
                        onClick={() => void undo()}
                        sx={{ color: "white" }}
                      >
                        <UndoRounded />
                      </IconButton>
                    </span>
                  </Tooltip>
                  <Tooltip title="다시 실행 · Ctrl/⌘ Shift Z">
                    <span>
                      <IconButton
                        size="small"
                        disabled={!redoStack.length || moving}
                        onClick={() => void redo()}
                        sx={{ color: "white" }}
                      >
                        <RedoRounded />
                      </IconButton>
                    </span>
                  </Tooltip>
                  <Typography
                    variant="caption"
                    sx={{ px: 0.7, whiteSpace: "nowrap" }}
                  >
                    {moving
                      ? "저장 중…"
                      : selectedIds.size
                        ? `${selectedIds.size}개 선택`
                        : "Shift로 다중 선택"}
                  </Typography>
                </Box>
              )}
              <Typography
                variant="caption"
                color="text.secondary"
                sx={{ ml: "auto", whiteSpace: "nowrap" }}
              >
                {highlighting
                  ? `${highlightedCount} / ${seats.length}석 강조`
                  : `${seats.length}석 전체`}
              </Typography>
            </Box>
            <Box
              sx={{
                position: "relative",
                flex: 1,
                minHeight: 0,
                overflow: "hidden",
              }}
            >
              <Box
                sx={{
                  position: "absolute",
                  top: 12,
                  left: 12,
                  zIndex: 2,
                  display: "flex",
                  gap: 0.5,
                  p: 0.5,
                  alignItems: "center",
                  bgcolor: "rgba(255,255,255,.92)",
                  backdropFilter: "blur(6px)",
                  borderRadius: 2,
                  boxShadow: 2,
                }}
              >
                <Tooltip title="축소">
                  <IconButton
                    size="small"
                    onClick={() => applyZoom(view.zoom / 1.35)}
                  >
                    <ZoomOutRounded />
                  </IconButton>
                </Tooltip>
                <Tooltip title="전체 보기 (도면 맞춤)">
                  <IconButton size="small" onClick={() => setView(FIT_VIEW)}>
                    <CenterFocusStrongRounded />
                  </IconButton>
                </Tooltip>
                <Tooltip title="확대">
                  <IconButton
                    size="small"
                    onClick={() => applyZoom(view.zoom * 1.35)}
                  >
                    <ZoomInRounded />
                  </IconButton>
                </Tooltip>
                <Typography
                  variant="caption"
                  fontWeight={700}
                  sx={{
                    px: 0.75,
                    minWidth: 42,
                    textAlign: "center",
                    color: "text.secondary",
                    fontVariantNumeric: "tabular-nums",
                  }}
                >
                  {Math.round(view.zoom * 100)}%
                </Typography>
              </Box>
              {!currentMap.overlayReady && (
                <Chip
                  size="small"
                  color="warning"
                  label={
                    currentMap.contentType === "application/pdf"
                      ? "PDF 미리보기 없음 · 오버레이를 표시할 수 없습니다"
                      : "도면 크기 정보 없음 · 좌석 정렬이 어긋날 수 있습니다"
                  }
                  sx={{
                    position: "absolute",
                    top: 12,
                    right: 12,
                    zIndex: 2,
                    fontWeight: 600,
                  }}
                />
              )}
              <Box
                ref={stageRef}
                sx={{
                  position: "absolute",
                  inset: 0,
                  overflow: "hidden",
                  p: 2.5,
                }}
              >
                {/* 오버레이 기준 래스터가 없을 때만 원본 뷰어로 물러난다. */}
                {!currentMap.overlayReady ? (
                  <Box
                    sx={{ width: "100%", height: "100%", position: "relative" }}
                  >
                    <object
                      data={currentMap.contentUrl}
                      type={currentMap.contentType}
                      width="100%"
                      height="100%"
                      aria-label="원본 도면"
                    />
                    <Typography
                      variant="caption"
                      sx={{
                        position: "absolute",
                        bottom: 10,
                        left: 10,
                        bgcolor: "rgba(255,255,255,.94)",
                        border: "1px solid rgba(14,45,62,.1)",
                        borderRadius: 1.5,
                        boxShadow: 1,
                        px: 1.25,
                        py: 0.5,
                      }}
                    >
                      좌석 오버레이 기준 이미지를 준비하지 못했습니다. 도면을
                      다시 업로드하거나 AI 분석을 실행해 주세요.
                    </Typography>
                  </Box>
                ) : (
                  <svg
                    ref={attachStage}
                    // 편집 모드에서는 좌석이 조작 대상이므로 단일 이미지로 묶지 않는다.
                    role={editMode ? "group" : "img"}
                    aria-label={`${currentMap.floorName} 좌석 배치도`}
                    viewBox={viewBox}
                    onDoubleClick={mapDoubleClick}
                    onPointerMove={canvasPointerMove}
                    onPointerUp={canvasPointerUp}
                    onPointerCancel={canvasPointerUp}
                    onPointerDown={canvasPointerDown}
                    style={{
                      display: "block",
                      width: "100%",
                      height: "100%",
                      background: "#fff",
                      borderRadius: 14,
                      boxShadow: "0 10px 30px rgba(14,45,62,.14)",
                      // 화면 이동과 확대를 직접 다루므로 브라우저 기본 제스처를 끈다.
                      touchAction: "none",
                      cursor: editMode
                        ? "default"
                        : panning
                          ? "grabbing"
                          : "grab",
                    }}
                  >
                    <defs>
                      {/* 보정된 격자가 있으면 그 간격과 원점을 그대로 그린다. */}
                      <pattern
                        id="seat-snap-grid"
                        x={grid ? grid.originX * canvas.width : 0}
                        y={grid ? grid.originY * canvas.height : 0}
                        width={(grid ? grid.pitchX : 0.05) * canvas.width}
                        height={
                          grid
                            ? grid.pitchY * canvas.height
                            : 0.05 * canvas.width
                        }
                        patternUnits="userSpaceOnUse"
                      >
                        <path
                          d={`M ${(grid ? grid.pitchX : 0.05) * canvas.width} 0 L 0 0 0 ${grid ? grid.pitchY * canvas.height : 0.05 * canvas.width}`}
                          fill="none"
                          stroke={grid ? "#087E8B" : "#0E2D3E"}
                          strokeWidth={grid ? 1.4 : 1}
                          opacity={grid ? 0.34 : 0.16}
                        />
                      </pattern>
                    </defs>
                    {/* 초점 표시는 좌석마다 인라인으로 두면 좌석 수만큼 늘어난다.
                        도면 안에 규칙 하나로 둔다. */}
                    <style>{`
                      [data-seat-id]:focus { outline: none; }
                      [data-seat-id]:focus-visible {
                        outline: 3px solid #0B5FFF;
                        outline-offset: 3px;
                      }
                    `}</style>
                    <image
                      href={currentMap.previewUrl}
                      x="0"
                      y="0"
                      width={canvas.width}
                      height={canvas.height}
                      preserveAspectRatio="none"
                      opacity=".92"
                    />
                    {editMode && snapEnabled && (
                      <rect
                        width={canvas.width}
                        height={canvas.height}
                        fill="url(#seat-snap-grid)"
                        style={{ pointerEvents: "none" }}
                      />
                    )}
                    {/* 조직 구역: 좌석에 지정된 구역의 경계 상자를 배경에 깐다. */}
                    {zones.map((zone) => {
                      const x = zone.minX * canvas.width - 6,
                        y = zone.minY * canvas.height - 6,
                        w = (zone.maxX - zone.minX) * canvas.width + 12,
                        h = (zone.maxY - zone.minY) * canvas.height + 12;
                      return (
                        <g key={zone.id} style={{ pointerEvents: "none" }}>
                          <rect
                            x={x}
                            y={y}
                            width={w}
                            height={h}
                            rx="12"
                            fill={zone.color}
                            fillOpacity="0.08"
                            stroke={zone.color}
                            strokeWidth="1.6"
                            strokeDasharray="8 5"
                            strokeOpacity="0.55"
                          />
                          <text
                            x={x + 8}
                            y={y + 16}
                            fontSize="12"
                            fontWeight="700"
                            fill={zone.color}
                            opacity="0.85"
                          >
                            {`${zone.name} · ${zone.count}석`}
                          </text>
                        </g>
                      );
                    })}
                    {seatLayer}
                  </svg>
                )}
              </Box>
              {/* 미니맵: 확대했을 때만 나타나 현재 보는 영역을 알려준다. */}
              {(view.zoom > 1.05 || filters.size > 0 || activeOrg !== null) && (
                <Box
                  sx={{
                    position: "absolute",
                    left: 12,
                    bottom: 12,
                    zIndex: 2,
                    width: 168,
                    p: 0.75,
                    bgcolor: "rgba(255,255,255,.94)",
                    backdropFilter: "blur(6px)",
                    border: "1px solid rgba(14,45,62,.1)",
                    borderRadius: 2,
                    boxShadow: 2,
                  }}
                >
                  <svg
                    viewBox={`0 0 ${canvas.width} ${canvas.height}`}
                    role="img"
                    aria-label="도면 전체 미니맵"
                    style={{
                      display: "block",
                      width: "100%",
                      background: "#F1F6F7",
                      borderRadius: 6,
                      cursor: "pointer",
                    }}
                    onPointerDown={(event) => {
                      event.currentTarget.setPointerCapture(event.pointerId);
                      moveViewToMinimap(event);
                    }}
                    onPointerMove={(event) => {
                      // 버튼을 누른 채 끌면 화면이 따라온다.
                      if (event.buttons === 1) moveViewToMinimap(event);
                    }}
                  >
                    {minimapSeats}
                    {/* 현재 화면 영역 */}
                    <rect
                      x={viewRect.x}
                      y={viewRect.y}
                      width={viewRect.width}
                      height={viewRect.height}
                      fill="none"
                      stroke="#FFB703"
                      strokeWidth={Math.max(4, canvas.width * 0.006)}
                    />
                  </svg>
                </Box>
              )}
              {/* 범례는 색상 기준을 따라간다. 조직 모드에서 상태 범례를 보여주면
                  화면의 색과 설명이 어긋나기 때문이다. */}
              <Box
                sx={{
                  position: "absolute",
                  right: 12,
                  bottom: 12,
                  display: "flex",
                  flexWrap: "wrap",
                  columnGap: 1.25,
                  rowGap: 0.5,
                  maxWidth: "calc(100% - 200px)",
                  maxHeight: 96,
                  overflowY: "auto",
                  bgcolor: "rgba(255,255,255,.94)",
                  backdropFilter: "blur(6px)",
                  border: "1px solid rgba(14,45,62,.1)",
                  borderRadius: 2,
                  boxShadow: 1,
                  px: 1.25,
                  py: 0.85,
                }}
              >
                {colorMode === "organization" ? (
                  mapOrganizations.length ? (
                    <>
                      {mapOrganizations.map((org) => (
                        <Stack
                          key={org.id}
                          direction="row"
                          spacing={0.6}
                          alignItems="center"
                          onClick={() =>
                            setActiveOrg((current) =>
                              current === org.id ? null : org.id,
                            )
                          }
                          sx={{
                            cursor: "pointer",
                            opacity:
                              activeOrg === null || activeOrg === org.id
                                ? 1
                                : 0.4,
                          }}
                        >
                          <Box
                            sx={{
                              width: 11,
                              height: 11,
                              borderRadius: 0.5,
                              bgcolor: org.color,
                              border: "1px solid rgba(14,45,62,.25)",
                            }}
                          />
                          <Typography
                            variant="caption"
                            sx={{ whiteSpace: "nowrap" }}
                            fontWeight={activeOrg === org.id ? 700 : 400}
                          >
                            {`${org.name} ${org.count}`}
                          </Typography>
                        </Stack>
                      ))}
                      {activeOrg !== null && (
                        <Chip
                          size="small"
                          label="강조 해제"
                          variant="outlined"
                          onClick={() => setActiveOrg(null)}
                        />
                      )}
                    </>
                  ) : (
                    <Typography variant="caption" color="text.secondary">
                      조직이 지정된 좌석이 없습니다
                    </Typography>
                  )
                ) : (
                  [
                    { color: "#087E8B", label: "배정", dashed: false },
                    { color: "#FFFFFF", label: "빈 좌석", dashed: false },
                    { color: "#3478C8", label: "공용", dashed: false },
                    { color: "#8796A1", label: "사용불가", dashed: false },
                    { color: "#FFFFFF", label: "검토 필요", dashed: true },
                    {
                      color: "#FFFFFF",
                      label: "구역 불일치",
                      dashed: true,
                      tone: "#C1436D",
                    },
                  ].map(({ color, label, dashed, tone }) => (
                    <Stack
                      key={label}
                      direction="row"
                      spacing={0.6}
                      alignItems="center"
                    >
                      <Box
                        sx={{
                          width: 11,
                          height: 11,
                          borderRadius: 0.5,
                          bgcolor: color,
                          border: dashed
                            ? `1.5px dashed ${tone ?? "#E79418"}`
                            : "1px solid #8796A1",
                        }}
                      />
                      <Typography
                        variant="caption"
                        sx={{ whiteSpace: "nowrap" }}
                      >
                        {label}
                      </Typography>
                    </Stack>
                  ))
                )}
              </Box>
            </Box>
          </Paper>
          <Paper
            component="section"
            aria-label="좌석 상세"
            sx={{ p: 2.5, minHeight: { xs: 220, lg: 0 } }}
          >
            {editMode && selectedIds.size > 1 ? (
              <Stack spacing={2.2}>
                <Box>
                  <Chip size="small" color="primary" label="다중 선택" />
                  <Typography variant="h5" sx={{ mt: 1.5 }}>
                    좌석 {selectedIds.size}개
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    선택한 좌석을 한 번에 정렬하거나 회전할 수 있습니다.
                  </Typography>
                </Box>
                <Divider />
                <Button
                  variant="outlined"
                  startIcon={<AlignHorizontalLeftRounded />}
                  onClick={() => applyTransform("left")}
                >
                  왼쪽 맞춤
                </Button>
                <Button
                  variant="outlined"
                  startIcon={<VerticalAlignTopRounded />}
                  onClick={() => applyTransform("top")}
                >
                  위쪽 맞춤
                </Button>
                <Button
                  variant="outlined"
                  startIcon={<RotateRightRounded />}
                  onClick={() => applyTransform("rotate")}
                >
                  90° 회전
                </Button>
                <Alert severity="info" icon={<OpenWithRounded />}>
                  Shift+클릭으로 선택을 추가하고, 방향키로 5px씩 이동합니다.
                  Shift+방향키는 20px 이동입니다.
                </Alert>
                <Button
                  color="inherit"
                  onClick={() => {
                    setSelectedIds(new Set());
                    setSelected(null);
                  }}
                >
                  선택 해제
                </Button>
              </Stack>
            ) : selected ? (
              <>
                <Stack
                  direction="row"
                  justifyContent="space-between"
                  alignItems="center"
                >
                  <Chip
                    label={
                      selected.type === "shared"
                        ? "공용 좌석"
                        : selected.status === "unavailable"
                          ? "사용 불가"
                          : selected.employeeId
                            ? "배정됨"
                            : "빈 좌석"
                    }
                    size="small"
                    color={selected.employeeId ? "primary" : "default"}
                  />
                  {selected.confidence != null && (
                    <Typography
                      variant="caption"
                      color={
                        selected.confidence < 0.95
                          ? "warning.main"
                          : "text.secondary"
                      }
                    >
                      AI {Math.round(selected.confidence * 100)}%
                    </Typography>
                  )}
                </Stack>
                <Typography variant="h5" sx={{ mt: 2 }}>
                  {selected.seatNo}
                </Typography>
                <Divider sx={{ my: 2 }} />
                {manager && editMode && (
                  <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
                    <Button
                      size="small"
                      variant="outlined"
                      startIcon={<EditRounded />}
                      onClick={() => setEditor(selected)}
                    >
                      편집
                    </Button>
                    <Button
                      size="small"
                      color="error"
                      startIcon={<DeleteOutlineRounded />}
                      disabled={Boolean(selected.employeeId)}
                      onClick={() => void removeSeat()}
                    >
                      삭제
                    </Button>
                  </Stack>
                )}
                {seatOrgDetail(selected).zone && (
                  <Box sx={{ mb: 2 }}>
                    <Info
                      label="지정 구역"
                      value={seatOrgDetail(selected).zone}
                    />
                  </Box>
                )}
                {selected.employeeId ? (
                  <Stack spacing={1.4}>
                    <Stack direction="row" spacing={1.2} alignItems="center">
                      <Avatar sx={{ bgcolor: "primary.main" }}>
                        {selected.employeeName?.slice(0, 1) ?? "?"}
                      </Avatar>
                      <Box>
                        <Typography fontWeight={750}>
                          {selected.employeeName}
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                          {selected.employeeNo}
                        </Typography>
                      </Box>
                    </Stack>
                    <Info
                      label="조직"
                      value={seatOrgDetail(selected).employee}
                    />
                    <Info
                      label="근무지"
                      value={
                        selected.employeeWorkplace || currentMap.buildingName
                      }
                    />
                    {manager && (
                      <Button
                        size="small"
                        color="error"
                        variant="outlined"
                        startIcon={<PersonRemoveRounded />}
                        onClick={() => void unassignSeat()}
                      >
                        배정 해제
                      </Button>
                    )}
                  </Stack>
                ) : (
                  <Stack alignItems="center" spacing={1.2} sx={{ pt: 2 }}>
                    <PersonPinCircleRounded
                      sx={{ fontSize: 42, color: "text.disabled" }}
                    />
                    <Typography color="text.secondary">
                      배정된 직원이 없습니다.
                    </Typography>
                    {manager && (
                      <Typography variant="caption" textAlign="center">
                        왼쪽 검색 결과의 직원을
                        <br />이 좌석으로 끌어 놓으세요.
                      </Typography>
                    )}
                  </Stack>
                )}
              </>
            ) : (
              <Stack
                sx={{ height: "100%" }}
                alignItems="center"
                justifyContent="center"
                spacing={1}
              >
                <CenterFocusStrongRounded
                  sx={{ fontSize: 40, color: "text.disabled" }}
                />
                <Typography color="text.secondary" textAlign="center">
                  좌석을 선택하면
                  <br />
                  상세 정보가 표시됩니다.
                </Typography>
              </Stack>
            )}
          </Paper>
        </Box>
      )}
      <SeatEditor
        value={editor}
        organizations={organizations}
        onChange={setEditor}
        onClose={() => setEditor(null)}
        onSave={() => void saveSeat()}
      />
    </Box>
  );
}
function SeatEditor({
  value,
  organizations,
  onChange,
  onClose,
  onSave,
}: {
  value: Partial<Seat> | null;
  organizations: Organization[];
  onChange: (value: Partial<Seat> | null) => void;
  onClose: () => void;
  onSave: () => void;
}) {
  const number = (key: keyof Seat, raw: string) =>
    value && onChange({ ...value, [key]: Number(raw) });
  return (
    <Dialog open={Boolean(value)} onClose={onClose} fullWidth maxWidth="sm">
      <DialogTitle>{value?.id ? "좌석 보정" : "좌석 추가"}</DialogTitle>
      <DialogContent>
        {value && (
          <Stack spacing={2} sx={{ mt: 1 }}>
            <Stack direction={{ xs: "column", sm: "row" }} spacing={2}>
              <TextField
                fullWidth
                label="좌석 번호"
                value={value.seatNo ?? ""}
                onChange={(event) =>
                  onChange({ ...value, seatNo: event.target.value })
                }
              />
              <FormControl fullWidth>
                <Select
                  value={value.type ?? "fixed"}
                  onChange={(event) =>
                    onChange({ ...value, type: event.target.value })
                  }
                >
                  <MenuItem value="fixed">고정 좌석</MenuItem>
                  <MenuItem value="shared">공용 좌석</MenuItem>
                  <MenuItem value="unavailable">사용 불가</MenuItem>
                  <MenuItem value="meeting_room">회의실</MenuItem>
                  <MenuItem value="executive">임원실</MenuItem>
                  <MenuItem value="utility">기타 공간</MenuItem>
                </Select>
              </FormControl>
            </Stack>
            <TextField
              select
              fullWidth
              id="seat-organization"
              label="조직 구역"
              value={value.organizationId ?? ""}
              onChange={(event) =>
                onChange({ ...value, organizationId: event.target.value })
              }
              helperText="좌석에 지정할 구역이며 직원 소속은 바뀌지 않습니다"
              slotProps={{
                select: {
                  displayEmpty: true,
                  SelectDisplayProps: {
                    "aria-labelledby": "seat-organization-label",
                  },
                },
                inputLabel: { shrink: true },
              }}
            >
              <MenuItem value="">미지정</MenuItem>
              {organizations.map((organization) => (
                <MenuItem key={organization.id} value={organization.id}>
                  {organization.name}
                </MenuItem>
              ))}
            </TextField>
            <Alert severity="info">
              좌표와 크기는 도면 대비 0~1 비율입니다. 도면을 더블 클릭하면 해당
              위치로 새 좌석이 만들어집니다.
            </Alert>
            <Stack direction={{ xs: "column", sm: "row" }} spacing={2}>
              {(["x", "y", "width", "height"] as const).map((key) => (
                <TextField
                  key={key}
                  label={key}
                  type="number"
                  value={value[key] ?? 0}
                  onChange={(event) => number(key, event.target.value)}
                  slotProps={{ htmlInput: { min: 0, max: 1, step: 0.005 } }}
                />
              ))}
            </Stack>
            <TextField
              label="회전 각도"
              type="number"
              value={value.rotation ?? 0}
              onChange={(event) => number("rotation", event.target.value)}
              slotProps={{ htmlInput: { min: -360, max: 360, step: 5 } }}
            />
          </Stack>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>취소</Button>
        <Button
          variant="contained"
          disabled={!value?.seatNo || !value.width || !value.height}
          onClick={onSave}
        >
          저장
        </Button>
      </DialogActions>
    </Dialog>
  );
}
function Info({ label, value }: { label: string; value: string }) {
  return (
    <Box>
      <Typography variant="caption" color="text.secondary">
        {label}
      </Typography>
      {/* data-info 은 좌석 상세의 값만 집어 읽기 위한 것이다. 화면에는 보이지 않는다. */}
      <Typography variant="body2" fontWeight={650} data-info={label}>
        {value}
      </Typography>
    </Box>
  );
}
