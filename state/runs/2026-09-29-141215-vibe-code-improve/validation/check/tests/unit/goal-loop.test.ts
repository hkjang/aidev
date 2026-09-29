import { describe, expect, it } from "vitest";
import { buildPromptContext } from "../../src/features/goal-loop";
import { goalTemplate } from "../../src/features/goals";

describe("buildPromptContext", () => {
	const goal = goalTemplate("2026-09-22 10:00:00 KST").replace("상태: draft", "상태: active");

	it("summarizes an active goal for the system prompt", () => {
		const text = buildPromptContext(goal);
		expect(text).toContain("VIBE CODE GOAL STATE");
		expect(text).toContain("- 목표: 새 목표");
		expect(text).toContain("지금 단계: discovery");
		expect(text).toContain("- Now: 목표 설명 작성");
		expect(text).toContain("- Next: 첫 번째 작은 작업 선택");
		expect(text).toContain("완료 기준 진행: 0/5");
		expect(text.startsWith("\n\n====\n\nVIBE CODE GOAL STATE")).toBe(true);
	});

	it("returns nothing for done goals or empty text", () => {
		expect(buildPromptContext("")).toBe("");
		expect(buildPromptContext(goal.replace("상태: active", "상태: done"))).toBe("");
	});

	it("tells the model when Now is empty", () => {
		const empty = goal.replace("- [ ] 목표 설명 작성", "- [x] 목표 설명 작성");
		expect(buildPromptContext(empty)).toContain("Now: (비어 있음");
	});
});
