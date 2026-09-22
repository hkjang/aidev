export const protocolVersion = "madi-usability-1";
export const tasks = Object.freeze([
  {
    id: "capture",
    title: "개인 메모 만들기",
    instruction:
      "홈에서 개인 메모를 만들고 두 문장을 작성한 뒤 서버에 저장됐는지 확인하세요.",
    success: "문서가 나만 보기이며 입력한 내용의 서버 저장을 확인",
  },
  {
    id: "find",
    title: "찾고 맥락으로 돌아오기",
    instruction:
      "지정된 점검 문서를 검색하고 미리보기에서 필요한 내용을 확인한 뒤 검색 결과로 돌아오세요.",
    success: "올바른 문서를 찾고 검색 조건·선택이 유지된 결과로 복귀",
  },
  {
    id: "share",
    title: "공유와 게시 구분하기",
    instruction:
      "개인 메모를 지정한 동료에게만 읽기 공유하세요. 저장·내부 공유·게시·외부 링크의 현재 상태를 설명하세요.",
    success:
      "지정 동료만 실제 열람 가능하고 외부 링크를 만들지 않음. 네 상태를 올바르게 설명",
  },
  {
    id: "ai",
    title: "선택 AI 변경 검토",
    instruction:
      "지정한 문단만 AI로 다듬고 변경 전후를 확인하세요. 제안 하나는 취소하고 다시 요청한 제안은 선택 범위에만 적용하세요.",
    success: "전송 범위를 확인, 취소는 원문 불변, 적용은 선택 범위만 변경",
  },
  {
    id: "database",
    title: "표의 오류 수정과 개인 보기",
    instruction:
      "지정한 세 셀을 편집하고 잘못된 선택값을 수정한 뒤 개인 보기를 저장하세요. 새로고침해 다시 확인하세요.",
    success:
      "타입이 맞는 값만 저장하고 다른 사용자 팀 보기를 바꾸지 않으며 개인 보기 유지",
  },
  {
    id: "resume",
    title: "업무 맥락 다시 이어가기",
    instruction:
      "문서와 필터가 적용된 DB 보기를 작업 묶음에 보관하세요. 홈에서 묶음을 다시 열어 작업을 이어가세요.",
    success: "현재 권한으로 문서 위치와 DB 보기·필터를 복원",
  },
]);
const validToken = /^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$/;
const fields = new Set([
  "protocol",
  "measurement_type",
  "session_id",
  "participant_id",
  "cohort",
  "build",
  "task_id",
  "attempt",
  "outcome",
  "elapsed_ms",
  "assistance_count",
  "undo_count",
  "sharing_misunderstanding",
  "viewport",
  "observed_at",
  "consent",
  "observer_confirmed",
]);
export function validateRecord(value, index = 0) {
  const fail = (msg) => {
    throw new Error(`${index + 1}번째 기록: ${msg}`);
  };
  if (!value || typeof value !== "object" || Array.isArray(value))
    fail("객체가 필요합니다");
  for (const key of Object.keys(value))
    if (!fields.has(key))
      fail(
        `허용하지 않은 필드 ${key} (원문·이름·자유 메모를 수집하지 않습니다)`,
      );
  for (const key of fields) if (!(key in value)) fail(`${key} 필드 누락`);
  if (value.protocol !== protocolVersion) fail("프로토콜 버전 불일치");
  if (
    !["human_observation", "automated_probe"].includes(value.measurement_type)
  )
    fail("실제 관찰과 자동시험 유형을 명시하세요");
  for (const key of ["session_id", "participant_id", "build"])
    if (typeof value[key] !== "string" || !validToken.test(value[key]))
      fail(`${key}는 이름·메일·URL이 아닌 64자 이내 익명 식별자여야 합니다`);
  if (!["new", "existing"].includes(value.cohort))
    fail("new/existing 코호트가 필요합니다");
  if (!tasks.some((t) => t.id === value.task_id))
    fail("정의하지 않은 과제입니다");
  if (!["completed", "partial", "abandoned"].includes(value.outcome))
    fail("완료/부분/중단 결과가 필요합니다");
  for (const key of ["attempt", "assistance_count", "undo_count"])
    if (
      !Number.isSafeInteger(value[key]) ||
      value[key] < (key === "attempt" ? 1 : 0) ||
      value[key] > 10000
    )
      fail(`${key} 범위 오류`);
  if (
    typeof value.elapsed_ms !== "number" ||
    !Number.isFinite(value.elapsed_ms) ||
    value.elapsed_ms < 0 ||
    value.elapsed_ms > 14400000
  )
    fail("과제 시간은 0~4시간의 밀리초여야 합니다");
  if (!["yes", "no", "not_assessed"].includes(value.sharing_misunderstanding))
    fail("공유 오해 평가값 오류");
  if (!["desktop", "mobile"].includes(value.viewport))
    fail("desktop/mobile 구분 필요");
  if (
    typeof value.observed_at !== "string" ||
    !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$/.test(value.observed_at) ||
    !Number.isFinite(Date.parse(value.observed_at)) ||
    new Date(value.observed_at).toISOString() !== value.observed_at
  )
    fail("UTC ISO 관찰 시각이 필요합니다");
  if (
    typeof value.consent !== "boolean" ||
    typeof value.observer_confirmed !== "boolean"
  )
    fail("동의·관찰 확인은 boolean이어야 합니다");
  if (
    value.measurement_type === "human_observation" &&
    (!value.consent || !value.observer_confirmed)
  )
    fail("실제 참가자 동의와 관찰자 확인 없이 사람 관찰로 기록할 수 없습니다");
  if (
    value.measurement_type === "automated_probe" &&
    (value.consent || value.observer_confirmed)
  )
    fail("자동시험을 참가자 동의·관찰로 표시할 수 없습니다");
  return value;
}
function median(values) {
  if (!values.length) return null;
  const a = [...values].sort((a, b) => a - b),
    i = Math.floor(a.length / 2);
  return a.length % 2 ? a[i] : (a[i - 1] + a[i]) / 2;
}
export function summarize(records) {
  if (!Array.isArray(records) || records.length > 10000)
    throw new Error("최대10000개 기록 배열이 필요합니다");
  const seen = new Set(),
    groups = new Map();
  records.forEach((input, i) => {
    const r = validateRecord(input, i),
      key = [
        r.measurement_type,
        r.session_id,
        r.participant_id,
        r.build,
        r.task_id,
        r.attempt,
      ].join(":");
    if (seen.has(key))
      throw new Error("같은 참여자·세션·과제·시도의 중복 기록입니다");
    seen.add(key);
    const groupKey = [
      r.measurement_type,
      r.build,
      r.cohort,
      r.viewport,
      r.task_id,
    ].join(":");
    if (!groups.has(groupKey)) groups.set(groupKey, []);
    groups.get(groupKey).push(r);
  });
  const results = [...groups.values()].map((rows) => {
    const first = rows[0],
      completed = rows.filter((r) => r.outcome === "completed"),
      assessed = rows.filter(
        (r) => r.sharing_misunderstanding !== "not_assessed",
      );
    return {
      measurement_type: first.measurement_type,
      build: first.build,
      cohort: first.cohort,
      viewport: first.viewport,
      task_id: first.task_id,
      participants:
        first.measurement_type === "human_observation"
          ? new Set(rows.map((r) => r.participant_id)).size
          : 0,
      probe_scenarios:
        first.measurement_type === "automated_probe"
          ? new Set(rows.map((r) => r.participant_id)).size
          : 0,
      attempts: rows.length,
      completed: completed.length,
      partial: rows.filter((r) => r.outcome === "partial").length,
      abandoned: rows.filter((r) => r.outcome === "abandoned").length,
      completion_rate: completed.length / rows.length,
      median_all_elapsed_ms: median(rows.map((r) => r.elapsed_ms)),
      median_completed_elapsed_ms: median(completed.map((r) => r.elapsed_ms)),
      assistance_total: rows.reduce((n, r) => n + r.assistance_count, 0),
      undo_total: rows.reduce((n, r) => n + r.undo_count, 0),
      sharing_assessed: assessed.length,
      sharing_misunderstanding_rate: assessed.length
        ? assessed.filter((r) => r.sharing_misunderstanding === "yes").length /
          assessed.length
        : null,
    };
  });
  return {
    protocol: protocolVersion,
    human_observation_records: records.filter(
      (r) => r.measurement_type === "human_observation",
    ).length,
    automated_probe_records: records.filter(
      (r) => r.measurement_type === "automated_probe",
    ).length,
    notice:
      "자동시험을 사람의 완료시간·사용성 개선으로 해석하지 않습니다. 집단/빌드/화면/과제를 합치지 않으며 표본이 없으면 관찰 미실시입니다. 결과는 인과관계나 통계적 유의성을 주장하지 않습니다.",
    groups: results,
  };
}
