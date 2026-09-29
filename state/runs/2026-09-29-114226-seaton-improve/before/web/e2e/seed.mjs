/**
 * 화면 검증용 시드 데이터.
 *
 * 빈 데이터베이스에 사업장·층·도면·좌석·직원·배정을 넣어, 좌석맵과 변경 이력이
 * 실제 운영과 비슷한 모습으로 그려지게 한다. 이미 도면이 있으면 아무것도 하지
 * 않으므로 여러 번 실행해도 안전하다.
 */
import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";

/** 검증이 기준으로 삼는 도면 버전. */
const BASE_VERSION = "v1";

const ORGS = [
  { externalId: "DEV", name: "개발팀", color: "#3478C8" },
  { externalId: "SALES", name: "영업팀", color: "#E7692F" },
  { externalId: "HR", name: "인사팀", color: "#7B5EA7" },
];
const PEOPLE = [
  ["김개발", "개발팀"],
  ["이코딩", "개발팀"],
  ["박서버", "개발팀"],
  ["최프론트", "개발팀"],
  ["정영업", "영업팀"],
  ["한세일즈", "영업팀"],
  ["오인사", "인사팀"],
  ["윤총무", "인사팀"],
  ["서기획", "개발팀"],
  ["남디자", "개발팀"],
];

/** 세션 쿠키와 CSRF 토큰을 들고 다니는 최소 클라이언트. */
const client = async (baseURL, username, password) => {
  let cookie = "";
  let csrf = "";
  const call = async (method, path, { json, form } = {}) => {
    const headers = { ...(cookie && { Cookie: cookie }) };
    if (csrf) headers["X-CSRF-Token"] = csrf;
    if (json) headers["Content-Type"] = "application/json";
    const res = await fetch(baseURL + path, {
      method,
      headers,
      body: form ?? (json ? JSON.stringify(json) : undefined),
    });
    const set = res.headers.getSetCookie?.() ?? [];
    if (set.length) cookie = set.map((c) => c.split(";")[0]).join("; ");
    if (!res.ok) {
      throw new Error(`${method} ${path} → ${res.status} ${await res.text()}`);
    }
    return res.status === 204 ? {} : res.json();
  };
  await call("POST", "/api/v1/auth/login", { json: { username, password } });
  csrf = (await call("GET", "/api/v1/auth/me")).csrfToken;
  return call;
};

const settle = async (call, jobId) => {
  for (let i = 0; i < 90; i++) {
    const job = await call("GET", `/api/v1/analysis-jobs/${jobId}`);
    if (job.status === "completed") return job;
    if (job.status === "failed") throw new Error(`분석 실패: ${job.error}`);
    await new Promise((r) => setTimeout(r, 1000));
  }
  throw new Error("분석이 끝나지 않았습니다");
};

/**
 * 이미 시드가 있는 환경을 검증 시작 상태로 되돌린다.
 *
 * 도면 게시를 바꾸는 검증이 도중에 끊기면 게시된 도면이 하나도 없는 채로 남고,
 * 그 뒤 모든 검증이 "좌석맵에 아무것도 없다"는 이유로 무너진다. 실행마다 기준
 * 버전을 다시 게시하고 남은 확인용 버전을 치운다.
 */
const repair = async (call, maps) => {
  const base = maps.find((map) => map.version === BASE_VERSION);
  if (!base) return { seeded: false, repaired: false };
  let repaired = false;
  for (const map of maps) {
    if (map.version === BASE_VERSION || !map.id) continue;
    try {
      if (map.active)
        await call("POST", `/api/v1/floor-maps/${map.id}/unpublish`);
      await call("DELETE", `/api/v1/floor-maps/${map.id}`);
      repaired = true;
    } catch {
      // 배정이나 이력이 걸린 버전은 그대로 둔다.
    }
  }
  if (!base.active) {
    await call("POST", `/api/v1/floor-maps/${base.id}/publish`);
    repaired = true;
  }
  return { seeded: false, repaired };
};

export const seed = async ({ baseURL, username, password }) => {
  const call = await client(baseURL, username, password);
  const existing = await call("GET", "/api/v1/floor-maps");
  if (existing.items?.length) return repair(call, existing.items);

  const building = await call("POST", "/api/v1/buildings", {
    json: { name: "본사", code: "HQ" },
  });
  const floor = await call("POST", "/api/v1/floors", {
    json: { buildingId: building.id, name: "3층", code: "3F" },
  });

  const form = new FormData();
  form.set("floorId", floor.id);
  form.set("version", BASE_VERSION);
  const plan = await readFile(
    fileURLToPath(new URL("./fixtures/plan.png", import.meta.url)),
  );
  form.set("file", new Blob([plan], { type: "image/png" }), "plan.png");
  await call("POST", "/api/v1/floor-maps", { form });

  const maps = await call("GET", `/api/v1/floor-maps?floorId=${floor.id}`);
  const mapId = maps.items[0].id;
  const job = await call(
    "POST",
    `/api/v1/floor-maps/${mapId}/analyze?engine=cv`,
  );
  await settle(call, job.jobId);
  await call("POST", `/api/v1/floor-maps/${mapId}/publish`);
  const seats = (await call("GET", `/api/v1/seats?floorMapId=${mapId}`)).items;
  if (seats.length < PEOPLE.length) {
    throw new Error(`좌석이 ${seats.length}개만 인식되었습니다`);
  }

  for (const org of ORGS)
    await call("POST", "/api/v1/organizations", { json: org });
  const orgs = Object.fromEntries(
    (await call("GET", "/api/v1/organizations")).items.map((o) => [
      o.name,
      o.id,
    ]),
  );

  for (const [i, [name, team]] of PEOPLE.entries()) {
    await call("POST", "/api/v1/employees", {
      json: {
        employeeNo: `E${String(i + 1).padStart(3, "0")}`,
        name,
        organizationId: orgs[team],
        title: "팀원",
        status: "active",
      },
    });
  }
  const employees = (await call("GET", "/api/v1/employees?limit=50")).items;
  const byName = Object.fromEntries(employees.map((e) => [e.name, e]));
  for (const [i, [name]] of PEOPLE.entries()) {
    await call("POST", "/api/v1/seat-assignments", {
      json: {
        employeeId: byName[name].id,
        seatId: seats[i].id,
        source: "manual",
        reason: "초기 배치",
      },
    });
  }
  // 앞줄 좌석에 개발팀 구역을 지정한다. 영업팀 직원이 앉은 자리가 하나 섞여
  // 있어 "구역 불일치" 표시도 함께 검증된다.
  for (const seat of seats.slice(0, 5)) {
    await call("PATCH", `/api/v1/seats/${seat.id}`, {
      json: { organizationId: orgs["개발팀"] },
    });
  }
  return { seeded: true, mapId, seats: seats.length };
};

if (import.meta.url === `file://${process.argv[1]}`) {
  console.log(
    await seed({
      baseURL: process.env.E2E_BASE_URL ?? "http://127.0.0.1:18781",
      username: process.env.E2E_USERNAME ?? "admin",
      password: process.env.E2E_PASSWORD ?? "e2e-verify-pass-123",
    }),
  );
}
