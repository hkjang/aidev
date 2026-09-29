import { describe, expect, it } from "vitest";
import { buildSessionSummary, lastSummaryTs, parseAuditLines } from "../../src/features/journal-summary";
import { goalTemplate } from "../../src/features/goals";
import { planTemplate } from "../../src/features/plans";
import { moveTaskToSection, section, sectionOfLine, toggleCheckbox } from "../../src/util/markdown";

const AUDIT = [
	'{"ts":"2026-09-22T01:00:00.000Z","kind":"goal","action":"openCurrentGoal","details":{}}',
	'{"ts":"2026-09-22T01:05:00.000Z","kind":"plan","action":"advanceCurrentPlan","details":{"planFile":".vibe-code/plans/current-plan.md","completed":"바로 구현할 작업 1개 정의","promoted":"x"}}',
	'{"ts":"2026-09-22T01:06:00.000Z","kind":"plan","action":"setCurrentPlanStatus","details":{"planFile":".vibe-code/plans/current-plan.md","status":"blocked"}}',
	"not json",
	'{"ts":"2026-09-22T01:07:00.000Z","kind":"proxy","action":"applyVibeCodersProxyFailed","details":{"error":"boom"}}',
	'{"ts":"2026-09-22T01:08:00.000Z","kind":"command","action":"approved","details":{"phase":"approved","command":"npm test"}}',
	'{"ts":"2026-09-22T01:09:00.000Z","kind":"command","action":"exited","details":{"phase":"exited","command":"npm test","exitCode":1}}',
	'{"ts":"2026-09-22T01:10:00.000Z","kind":"command","action":"denied","details":{"phase":"denied","command":"rm -rf /"}}',
].join("\n");

describe("journal summary", () => {
	it("parses audit lines and skips garbage", () => {
		expect(parseAuditLines(AUDIT)).toHaveLength(7);
	});

	it("builds a summary with completed items, status changes, failures and goal next action", () => {
		const text = buildSessionSummary(parseAuditLines(AUDIT), goalTemplate("2026-09-22 10:00:00 KST"), "10:30 KST", "");
		expect(text).toContain("## 세션 요약 (10:30 KST)");
		expect(text).toContain("완료한 계획 항목 1개: 바로 구현할 작업 1개 정의");
		expect(text).toContain("계획 상태 변경: blocked (current-plan.md)");
		expect(text).toContain("실패: applyVibeCodersProxyFailed: boom");
		expect(text).toContain("실행한 명령: 1개 승인, 1개 거부 · 실패 1개: npm test (exit 1)");
		expect(text).toContain("다음 행동: 목표 문장을 구체화");
		expect(text).toContain("목표 Now: 목표 설명 작성");
		expect(text).toContain("<!-- vibe-code:summary-until 2026-09-22T01:10:00.000Z -->");
	});

	it("only reports entries after the last summary marker", () => {
		const first = buildSessionSummary(parseAuditLines(AUDIT), "", "10:30 KST", "")!;
		const since = lastSummaryTs("# journal\n" + first);
		expect(since).toBe("2026-09-22T01:10:00.000Z");
		expect(buildSessionSummary(parseAuditLines(AUDIT), "", "11:00 KST", since)).toBeNull();
		expect(lastSummaryTs("no marker")).toBe("");
	});
});

describe("line-level plan edits", () => {
	const plan = planTemplate("t", "current-plan.md", "s");
	const all = plan.split("\n");
	const nowLine = all.indexOf("- [ ] 바로 구현할 작업 1개 정의");
	const nextLine = all.indexOf("- [ ] 뒤이어 실행할 작업 1개 정의");
	const stepLine = all.indexOf("- [ ] 검증 기준 확정");

	it("finds the enclosing section of a line", () => {
		expect(sectionOfLine(plan, nowLine)).toBe("Now");
		expect(sectionOfLine(plan, nextLine)).toBe("Next");
		expect(sectionOfLine(plan, stepLine)).toBe("단계");
		expect(sectionOfLine(plan, 0)).toBe("");
	});

	it("moves a Now task to Done as checked, and a Next task to Now", () => {
		const done = moveTaskToSection(plan, nowLine, "Done", (l) => "- [x] " + l.replace(/^- \[ \] /, ""))!;
		expect(section(done, "Now")).toBe("");
		expect(section(done, "Done")).toBe("- [x] 바로 구현할 작업 1개 정의");
		const promoted = moveTaskToSection(done, done.split("\n").indexOf("- [ ] 뒤이어 실행할 작업 1개 정의"), "Now")!;
		expect(section(promoted, "Now")).toBe("- [ ] 뒤이어 실행할 작업 1개 정의");
		expect(section(promoted, "Next")).toBe("");
	});

	it("refuses non-task lines and missing sections", () => {
		expect(moveTaskToSection(plan, 0, "Done")).toBeNull();
		expect(moveTaskToSection(plan, nowLine, "없는 섹션")).toBeNull();
	});

	it("toggles checkboxes in place", () => {
		const checked = toggleCheckbox(plan, stepLine);
		expect(checked.split("\n")[stepLine]).toBe("- [x] 검증 기준 확정");
		expect(toggleCheckbox(checked, stepLine)).toBe(plan);
		expect(toggleCheckbox(plan, 0)).toBe(plan);
	});
});
