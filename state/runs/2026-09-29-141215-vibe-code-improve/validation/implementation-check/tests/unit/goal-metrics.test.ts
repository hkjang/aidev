import { describe, expect, it } from "vitest";
import { isDestructiveCommand } from "../../src/features/checkpoints";
import { buildRetro, computeGoalMetrics } from "../../src/features/goal-metrics";
import { goalTemplate } from "../../src/features/goals";
import { parseAuditLines } from "../../src/features/journal-summary";

const AUDIT = [
	{ ts: "2026-09-20T01:00:00Z", kind: "plan", action: "advanceCurrentPlan", details: { completed: "A" } },
	{ ts: "2026-09-21T01:00:00Z", kind: "plan", action: "completePlanItem", details: { completed: "B" } },
	{ ts: "2026-09-21T02:00:00Z", kind: "plan", action: "runVerification", details: { command: "npm test", exitCode: 0 } },
	{ ts: "2026-09-21T03:00:00Z", kind: "plan", action: "runVerification", details: { command: "go test ./...", exitCode: 1 } },
	{ ts: "2026-09-21T03:10:00Z", kind: "plan", action: "runVerification", details: { command: "go test ./...", exitCode: null } },
	{ ts: "2026-09-22T01:00:00Z", kind: "command", action: "approved", details: { command: "npm test" } },
	{ ts: "2026-09-22T01:01:00Z", kind: "command", action: "exited", details: { command: "npm test", exitCode: 2 } },
	{ ts: "2026-09-22T02:00:00Z", kind: "goal", action: "taskCompleted", details: { goalActive: true, goalUpdated: false, tokens: 1500, cost: 0.25 } },
	{ ts: "2026-09-22T03:00:00Z", kind: "goal", action: "taskCompleted", details: { goalActive: true, goalUpdated: true, tokens: 500, cost: 0.05 } },
	{ ts: "2026-09-22T04:00:00Z", kind: "plan", action: "setCurrentPlanStatus", details: { status: "done" } },
	{ ts: "2026-09-22T05:00:00Z", kind: "goal", action: "autoHandoff", details: {} },
].map((e) => JSON.stringify(e)).join("\n");

describe("goal metrics", () => {
	const m = computeGoalMetrics(parseAuditLines(AUDIT), 7);
	it("aggregates completions, verifications, commands, tasks", () => {
		expect(m.completedItems).toEqual(["A", "B"]);
		expect(m.verifications).toBe(3);
		expect(m.verificationPassRate).toBeCloseTo(1 / 3);
		expect(m.failedVerifications).toEqual(["go test ./...", "go test ./..."]);
		expect(m.commandsApproved).toBe(1);
		expect(m.commandsFailed).toBe(1);
		expect(m.tasks).toBe(2);
		expect(m.tasksWithoutGoalUpdate).toBe(1);
		expect(m.tokens).toBe(2000);
		expect(m.costUsd).toBeCloseTo(0.3);
		expect(m.handoffs).toBe(1);
		expect(m.planStatusChanges).toEqual({ done: 1 });
		expect(m.activeDays).toBe(3);
	});
	it("builds a retro with bottlenecks and next-week items", () => {
		const retro = buildRetro({ from: "2026-09-16", to: "2026-09-22", metrics: m, entries: parseAuditLines(AUDIT), goalText: goalTemplate("s").replace("상태: draft", "상태: active") });
		expect(retro).toContain("# 주간 회고 — 2026-09-16 ~ 2026-09-22");
		expect(retro).toContain("- [x] A");
		expect(retro).toContain("`go test ./...` — 실패 2회");
		expect(retro).toContain("목표 파일을 갱신하지 않은 작업 1회");
		expect(retro).toContain("- [ ] Now: 목표 설명 작성");
		expect(retro).toContain("완료 기준 0/5");
		expect(buildRetro({ from: "a", to: "b", metrics: computeGoalMetrics([], 7), entries: [], goalText: "" })).toContain("활성 목표 없음");
	});
});

describe("isDestructiveCommand", () => {
	it("flags deletes, history rewrites and production-like actions", () => {
		for (const c of ["rm -rf node_modules", "git reset --hard HEAD~1", "git push origin main --force", "git clean -fdx", "DROP TABLE users;", "Remove-Item -Recurse -Force .\\dist", "del /s /q build", "npm publish", "kubectl delete deploy api", "terraform destroy"]) expect(isDestructiveCommand(c), c).toBe(true);
	});
	it("passes ordinary commands", () => {
		for (const c of ["npm test", "git status", "git push origin feature", "ls -la", "rm notes.txt", "go test ./...", "python -m pytest"]) expect(isDestructiveCommand(c), c).toBe(false);
	});
});
