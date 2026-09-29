import { describe, expect, it } from "vitest";
import { buildGoalFile, currentGoalFileName, GOAL_PRESETS, parseCriteriaLink, slugify, tickCriteria } from "../../src/features/goal-catalog";
import { goalTemplate, parseGoal } from "../../src/features/goals";
import { matchLine, section } from "../../src/util/markdown";

describe("goal catalog", () => {
	it("slugifies titles while keeping Korean", () => {
		expect(slugify("결제 모듈에 환불 API 추가!")).toBe("결제-모듈에-환불-api-추가");
		expect(slugify("   ")).toBe("goal");
		expect(slugify("a".repeat(60)).length).toBe(40);
	});

	it("builds a goal file from a preset", () => {
		const text = buildGoalFile({ title: "환불 API", description: "부분 환불 지원", criteria: GOAL_PRESETS[1].criteria, fileName: "x.md", stamp: "s" });
		const goal = parseGoal(text);
		expect(goal.title).toBe("환불 API");
		expect(goal.status).toBe("active");
		expect(goal.criteria).toHaveLength(4);
		expect(goal.criteria[0]).toContain("재현 절차와 원인 기록");
		expect(section(text, "목표")).toBe("부분 환불 지원");
		expect(matchLine(text, "목표 파일")).toBe(".vibe-code/goals/x.md");
		expect(currentGoalFileName(text)).toBe("x.md");
		expect(currentGoalFileName(goalTemplate("s"))).toBeUndefined();
	});

	it("ticks linked criteria by index", () => {
		const goal = goalTemplate("s");
		const ticked = tickCriteria(goal, [1, 3]);
		const lines = section(ticked, "완료 기준").split("\n");
		expect(lines[0].startsWith("- [x]")).toBe(true);
		expect(lines[1].startsWith("- [ ]")).toBe(true);
		expect(lines[2].startsWith("- [x]")).toBe(true);
		expect(tickCriteria("no sections", [1])).toBe("no sections");
		expect(parseCriteriaLink("연결 완료 기준: 1, 3 5\n")).toEqual([1, 3, 5]);
		expect(parseCriteriaLink("nothing")).toEqual([]);
	});
});
