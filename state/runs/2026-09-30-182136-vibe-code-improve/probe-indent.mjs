// 정찰 probe — 프로덕션 markdown.ts / plans.ts 를 그대로 import 해서 들여쓴 체크리스트 판정을 비교한다.
const R = "/home/hkjang/.cache/auto-improve-wt/vibe-code/src/";
const md = await import(R + "util/markdown.ts");
const { planTemplate, advancePlanText, openPlanItems } = await import(R + "features/plans.ts");

const plan = md.writeSection(planTemplate("목표", "20260930", "P1"), "Now", [
	"- [ ] 상위 항목",
	"  - [ ] 들여쓴 하위 항목",
]);

console.log("--- Now 섹션 원문 ---");
console.log(JSON.stringify(md.section(plan, "Now")));
console.log("taskLines(Now)      =", md.taskLines(md.section(plan, "Now")));
console.log("sectionLines(Now)   =", md.sectionLines(plan, "Now"));
console.log("countChecks(plan)   =", md.countChecks(plan));
console.log("progressLabel       =", md.progressLabel(md.countChecks(plan), "체크 없음"));
console.log("openPlanItems       =", openPlanItems(plan));

const indented = "  - [ ] 들여쓴 하위 항목";
console.log("isTaskLine(indented)        =", md.isTaskLine(indented));
console.log("isTaskLine(indented.trim()) =", md.isTaskLine(indented.trim()));

const allLines = plan.split("\n");
const idx = allLines.findIndex((l) => l.includes("들여쓴 하위 항목"));
console.log("indented line index =", idx, JSON.stringify(allLines[idx]));
const toggled = md.toggleCheckbox(plan, idx);
console.log("toggleCheckbox 변화 있었나 =", toggled !== plan);
const moved = md.moveTaskToSection(plan, idx, "Done", (l) => "- [x] " + md.stripTask(l));
console.log("moveTaskToSection null 인가 =", moved === null);
if (moved) console.log("Done 섹션 =", md.sectionLines(moved, "Done"));

const adv = advancePlanText(plan, "2026-09-30 18:00");
console.log("advancePlanText completed =", adv && adv.completed);

// CodeLens 게이트와 동일한 판정 (plan-codelens.ts:29)
console.log("CodeLens 가 이 줄에 액션을 붙이나 =", md.isTaskLine(allLines[idx]));
