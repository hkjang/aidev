import { describe, expect, it } from "vitest";
import { assessGoalHealth, goalAgeHours } from "../../src/features/goal-health";
import { lintGoal, lintPlan } from "../../src/features/goal-lint";
import { goalTemplate } from "../../src/features/goals";
import { planTemplate } from "../../src/features/plans";

const NOW = Date.UTC(2026, 8, 24, 3, 0); // 2026-09-24 12:00 KST
const active = goalTemplate("2026-09-22 10:00:00 KST").replace("상태: draft", "상태: active");

describe("goal health", () => {
	it("computes age from the KST timestamp", () => {
		expect(Math.round(goalAgeHours(active, NOW))).toBe(50);
		expect(goalAgeHours("no stamp", NOW)).toBe(Number.POSITIVE_INFINITY);
	});
	it("flags stale and stuck goals only when active", () => {
		expect(assessGoalHealth(active, undefined, 48, 3, NOW).stale).toBe(true);
		expect(assessGoalHealth(active, undefined, 72, 3, NOW).warning).toBeUndefined();
		const stuck = assessGoalHealth(active, { item: "목표 설명 작성", sessions: 3, since: "" }, 72, 3, NOW);
		expect(stuck.stuck).toBe(true);
		expect(stuck.warning).toContain("3세션째");
		expect(assessGoalHealth(goalTemplate("2026-09-01 10:00:00 KST"), { item: "x", sessions: 9, since: "" }, 1, 3, NOW).warning).toBeUndefined();
	});
});

describe("lint", () => {
	it("accepts the templates", () => {
		expect(lintGoal(goalTemplate("s")).filter((i) => i.severity !== "info")).toEqual([]);
		expect(lintPlan(planTemplate("t", "p.md", "s")).filter((i) => i.severity !== "info")).toEqual([]);
	});
	it("reports structural problems", () => {
		const bad = "# 목표: x\n\n상태: weird\n\n## 작업 큐\n### Now\n- [ ] a\n- [ ] b\n- [ ] c\n- [ ] d\n- [ ] e\n";
		const messages = lintGoal(bad).map((i) => i.message);
		expect(messages.some((m) => m.includes('알 수 없는 상태 "weird"'))).toBe(true);
		expect(messages.some((m) => m.includes("`## 완료 기준` 섹션이 없습니다"))).toBe(true);
		expect(messages.some((m) => m.includes("Now에 미완료 항목이 5개"))).toBe(true);
		const donePlan = planTemplate("t", "p.md", "s").replace("상태: draft", "상태: done").replace("우선순위: P2", "");
		expect(lintPlan(donePlan).some((i) => i.message.includes("상태가 done이지만"))).toBe(true);
		expect(lintPlan("# 계획: x\n\n상태: active\n우선순위: P9\n\n## 단계\n\n## Now\n\n## Next\n\n## Done\n").some((i) => i.message.includes("P9"))).toBe(true);
	});
});
