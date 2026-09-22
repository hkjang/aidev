import { protocolVersion, tasks, validateRecord } from "./model.mjs";
const $ = (id) => document.getElementById(id),
  records = [];
let start = 0,
  elapsed = null,
  help = 0,
  undo = 0,
  timer = null;
for (const task of tasks) {
  const o = document.createElement("option");
  o.value = task.id;
  o.textContent = task.title;
  $("task").append(o);
}
function describe() {
  const t = tasks.find((t) => t.id === $("task").value);
  $("task-title").textContent = t.title;
  $("instruction").textContent = t.instruction;
  $("success").textContent = "완료 기준: " + t.success;
}
$("task").addEventListener("change", describe);
describe();
function condition() {
  return {
    protocol: protocolVersion,
    measurement_type: "human_observation",
    session_id: $("session").value.trim(),
    participant_id: $("participant").value.trim(),
    cohort: $("cohort").value,
    build: $("build").value.trim(),
    task_id: $("task").value,
    attempt: Number($("attempt").value),
    outcome: $("outcome").value,
    elapsed_ms: elapsed ?? 0,
    assistance_count: help,
    undo_count: undo,
    sharing_misunderstanding: $("sharing").value,
    viewport: $("viewport").value,
    observed_at: new Date().toISOString(),
    consent: $("consent").checked,
    observer_confirmed: $("confirmed").checked,
  };
}
$("start").addEventListener("click", () => {
  try {
    validateRecord({ ...condition(), observer_confirmed: true });
    $("error").textContent = "";
    $("setup").disabled = true;
    $("start").disabled = true;
    $("finish").disabled = false;
    $("results").disabled = false;
    $("record").disabled = true;
    $("confirmed").checked = false;
    $("outcome").value = "partial";
    $("sharing").value = "not_assessed";
    elapsed = null;
    help = undo = 0;
    $("help-count").value = $("undo-count").value = "0";
    start = performance.now();
    timer = setInterval(() => {
      const s = Math.floor((performance.now() - start) / 1000);
      $("timer").value =
        `${String(Math.floor(s / 60)).padStart(2, "0")}:${String(s % 60).padStart(2, "0")}`;
    }, 250);
  } catch (e) {
    $("error").textContent = e.message;
  }
});
$("help").addEventListener("click", () => {
  $("help-count").value = String(++help);
});
$("undo").addEventListener("click", () => {
  $("undo-count").value = String(++undo);
});
$("finish").addEventListener("click", () => {
  elapsed = Math.round(performance.now() - start);
  clearInterval(timer);
  timer = null;
  $("finish").disabled = true;
  $("record").disabled = false;
});
$("record").addEventListener("click", () => {
  try {
    if (elapsed === null) throw new Error("과제 시간을 먼저 종료하세요");
    const record = validateRecord(condition());
    if (
      records.some(
        (r) =>
          [r.session_id, r.participant_id, r.task_id, r.attempt].join(":") ===
          [
            record.session_id,
            record.participant_id,
            record.task_id,
            record.attempt,
          ].join(":"),
      )
    )
      throw new Error("이미 기록한 세션·참여자·과제·시도입니다");
    records.push(record);
    $("count").textContent =
      records.length +
      "개 실제 관찰 · 아직 파일로 보관하지 않았다면 내려받으세요";
    $("download").disabled = false;
    $("results").disabled = true;
    $("setup").disabled = false;
    $("start").disabled = false;
    $("record").disabled = true;
    $("error").textContent = "";
  } catch (e) {
    $("error").textContent = e.message;
  }
});
$("download").addEventListener("click", () => {
  const url = URL.createObjectURL(
      new Blob([JSON.stringify(records, null, 2) + "\n"], {
        type: "application/json",
      }),
    ),
    a = document.createElement("a");
  a.href = url;
  a.download = "madi-usability-observations.json";
  a.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
});
$("clear").addEventListener("click", () => {
  if (
    confirm(
      "이 창의 관찰 기록과 진행 중 시간을 지울까요? 내려받은 파일은 삭제하지 않습니다.",
    )
  ) {
    records.length = 0;
    clearInterval(timer);
    timer = null;
    elapsed = null;
    $("timer").value = "00:00";
    $("count").textContent = "0개 · 관찰 미실시";
    $("download").disabled = true;
    $("setup").disabled = false;
    $("results").disabled = true;
    $("start").disabled = false;
    $("finish").disabled = true;
  }
});
window.addEventListener("beforeunload", (event) => {
  if (records.length || timer) {
    event.preventDefault();
    event.returnValue = "";
  }
});
