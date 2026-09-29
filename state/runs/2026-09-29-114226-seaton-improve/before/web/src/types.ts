export type Role =
  "employee" | "department_manager" | "seat_manager" | "system_admin";
export interface User {
  id: string;
  username: string;
  displayName: string;
  email?: string;
  employeeId?: string;
  role: Role;
  source: "local" | "oidc";
  lastLoginAt?: string;
  active: boolean;
}
export interface VersionInfo {
  version: string;
  commit: string;
  builtAt?: string;
}
export interface AuthConfig {
  serviceName: string;
  companyName: string;
  localEnabled: boolean;
  oidcEnabled: boolean;
  /** Keycloak 세션이 살아 있으면 로그인 화면 없이 조용히 로그인할지(prompt=none). */
  oidcAutoLogin?: boolean;
  version: VersionInfo;
}
export interface Building {
  id: string;
  name: string;
  code: string;
  address: string;
}
export interface Floor {
  id: string;
  buildingId: string;
  buildingName: string;
  name: string;
  code: string;
  sortOrder: number;
}
// 도면별 좌석 격자 보정값. 모든 값은 도면 대비 비율 좌표다.
export interface SeatGrid {
  originX: number;
  originY: number;
  pitchX: number;
  pitchY: number;
  /** manual은 관리자 보정값이라 재분석이 덮어쓰지 않는다. cv는 자동 추론값. */
  source?: "manual" | "cv";
}
export interface FloorMap {
  id: string;
  floorId: string;
  version: string;
  fileName: string;
  contentType: string;
  /** 좌석 좌표의 기준이 되는 래스터 픽셀 크기. PDF는 미리보기 크기다. */
  width?: number;
  height?: number;
  status: string;
  active: boolean;
  createdAt: string;
  floorName: string;
  buildingName: string;
  seatCount?: number;
  reviewCount?: number;
  contentUrl: string;
  /** 좌석 오버레이 배경용 래스터. PDF도 이 URL로 이미지를 받는다. */
  previewUrl: string;
  overlayReady: boolean;
  grid?: SeatGrid | null;
}
export type AnalysisEngine = "cv" | "vlm" | "hybrid";
// 도면 분석은 비동기 잡이다. VLM 호출이 수십 초 걸릴 수 있어 상태를 폴링한다.
export interface AnalysisJob {
  jobId: string;
  floorMapId: string;
  status: "queued" | "running" | "completed" | "failed";
  engine: AnalysisEngine;
  detected: number;
  needsReview: number;
  warnings: string[];
  message?: string;
  error?: string;
  details?: Record<string, unknown>;
  createdAt: string;
  completedAt?: string | null;
}
export interface Seat {
  id: string;
  floorMapId: string;
  seatNo: string;
  type: string;
  status: string;
  x: number;
  y: number;
  width: number;
  height: number;
  rotation: number;
  confidence?: number;
  organizationId?: string;
  organizationName?: string;
  employeeId?: string;
  employeeNo?: string;
  employeeName?: string;
  /** 좌석에 지정된 구역(organizationId)과 달리, 실제로 앉은 직원의 소속이다. */
  employeeOrganizationId?: string;
  employeeOrganizationName?: string;
  /** 앉은 직원의 근무지. 좌석 상세를 직원 검색 결과 없이 채우기 위한 값이다. */
  employeeWorkplace?: string;
}
export interface Employee {
  id: string;
  employeeNo: string;
  name: string;
  email?: string;
  organizationId?: string;
  organizationName?: string;
  title?: string;
  position?: string;
  workplace?: string;
  status: string;
  seatId?: string;
  seatNo?: string;
}
export interface Organization {
  id: string;
  externalId?: string;
  name: string;
  parentId?: string;
  color: string;
}

/** 좌석 일괄 배정에서 반영되지 않은 행. */
export type BulkFailure = {
  row: number;
  employeeNo: string;
  seatNo: string;
  error: string;
};
