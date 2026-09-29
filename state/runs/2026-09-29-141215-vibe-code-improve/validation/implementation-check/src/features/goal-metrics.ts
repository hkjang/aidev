import * as fs from "node:fs";
import * as path from "node:path";
import * as vscode from "vscode";
import type { CoreHost } from "../core/host";
import { log } from "../log";
import { kstDate } from "../util/kst";
import { stripTask } from "../util/markdown";
import { parseGoal } from "./goals";
import { AuditEntry, parseAuditLines } from "./journal-summary";
import { cachedWeeklyUsage, formatKrw, UsageReport } from "./usage";
import { ensureWorkspacePaths, readUtf8, writeAudit } from "./workspace";

const str = (v: unknown) => (typeof v === "string" ? v : "");
const num = (v: unknown) => (typeof v === "number" && Number.isFinite(v) ? v : Number(v) || 0);

/** Audit entries from the last `days` daily files (oldest first). */
export function readRecentAudit(auditDir: string, days: number): AuditEntry[] {
	if (!fs.existsSync(auditDir)) return [];
	const files = fs
		.readdirSync(auditDir)
		.filter((n) => /^\d{4}-\d{2}-\d{2}\.jsonl$/.test(n))
		.sort()
		.slice(-days);
	return files.flatMap((n) => parseAuditLines(readUtf8(path.join(auditDir, n))));
}

export interface GoalMetrics {
	days: number;
	completedItems: string[];
	completedPerDay: number;
	verifications: number;
	verificationPassRate: number;
	failedVerifications: string[];
	commandsApproved: number;
	commandsDenied: number;
	commandsFailed: number;
	tasks: number;
	tasksWithoutGoalUpdate: number;
	tokens: number;
	costUsd: number;
	handoffs: number;
	healthWarnings: number;
	planStatusChanges: Record<string, number>;
	activeDays: number;
}

/** Pure aggregation over audit entries. */
export function computeGoalMetrics(entries: AuditEntry[], days: number): GoalMetrics {
	const completedItems = entries.filter((e) => e.kind === "plan" && (e.action === "advanceCurrentPlan" || e.action === "completePlanItem")).map((e) => str(e.details?.completed)).filter(Boolean);
	const verifications = entries.filter((e) => e.kind === "plan" && e.action === "runVerification");
	const passed = verifications.filter((e) => num(e.details?.exitCode) === 0 && e.details?.exitCode !== null).length;
	const failedVerifications = verifications.filter((e) => e.details?.exitCode === null || num(e.details?.exitCode) !== 0).map((e) => str(e.details?.command)).filter(Boolean);
	const tasks = entries.filter((e) => e.kind === "goal" && e.action === "taskCompleted");
	const planStatusChanges: Record<string, number> = {};
	for (const e of entries.filter((e) => e.kind === "plan" && e.action === "setCurrentPlanStatus")) {
		const s = str(e.details?.status) || "?";
		planStatusChanges[s] = (planStatusChanges[s] || 0) + 1;
	}
	const activeDays = new Set(entries.map((e) => e.ts.slice(0, 10))).size;
	return {
		days,
		completedItems,
		completedPerDay: days > 0 ? completedItems.length / days : 0,
		verifications: verifications.length,
		verificationPassRate: verifications.length ? passed / verifications.length : 0,
		failedVerifications,
		commandsApproved: entries.filter((e) => e.kind === "command" && e.action === "approved").length,
		commandsDenied: entries.filter((e) => e.kind === "command" && e.action === "denied").length,
		commandsFailed: entries.filter((e) => e.kind === "command" && e.action === "exited" && num(e.details?.exitCode) !== 0).length,
		tasks: tasks.length,
		tasksWithoutGoalUpdate: tasks.filter((e) => e.details?.goalActive === true && e.details?.goalUpdated === false).length,
		tokens: tasks.reduce((n, e) => n + num(e.details?.tokens), 0),
		costUsd: tasks.reduce((n, e) => n + num(e.details?.cost), 0),
		handoffs: entries.filter((e) => e.kind === "goal" && (e.action === "createGoalHandoff" || e.action === "autoHandoff")).length,
		healthWarnings: entries.filter((e) => e.kind === "goal" && e.action === "healthWarning").length,
		planStatusChanges,
		activeDays,
	};
}

export function formatMetrics(m: GoalMetrics): string[] {
	const pct = (r: number) => `${(r * 100).toFixed(0)}%`;
	return [
		`기간: 최근 ${m.days}일 (활동일 ${m.activeDays}일)`,
		`완료 항목: ${m.completedItems.length}개 (하루 ${m.completedPerDay.toFixed(1)}개)`,
		`검증 실행: ${m.verifications}회, 통과율 ${pct(m.verificationPassRate)}${m.failedVerifications.length ? ` · 실패: ${[...new Set(m.failedVerifications)].slice(0, 3).join("; ")}` : ""}`,
		`명령 실행: 승인 ${m.commandsApproved} · 거부 ${m.commandsDenied} · 실패 종료 ${m.commandsFailed}`,
		`작업: ${m.tasks}회 (목표 미갱신 ${m.tasksWithoutGoalUpdate}회) · 토큰 ${m.tokens.toLocaleString("ko-KR")} · 비용 $${m.costUsd.toFixed(2)}`,
		`핸드오프 ${m.handoffs}회 · 정체 경고 ${m.healthWarnings}회 · 계획 상태 변경 ${Object.entries(m.planStatusChanges).map(([k, v]) => `${k} ${v}`).join(", ") || "없음"}`,
	];
}

// --- retro ---------------------------------------------------------------------------------------

export interface RetroInput {
	from: string;
	to: string;
	metrics: GoalMetrics;
	entries: AuditEntry[];
	goalText: string;
	usage?: UsageReport;
}

/** Weekly retrospective markdown. Pure. */
export function buildRetro(input: RetroInput): string {
	const { metrics: m, entries, goalText } = input;
	const goal = goalText ? parseGoal(goalText) : undefined;
	const failing = new Map<string, number>();
	for (const c of m.failedVerifications) failing.set(c, (failing.get(c) || 0) + 1);
	for (const e of entries.filter((e) => e.kind === "command" && e.action === "exited" && num(e.details?.exitCode) !== 0)) {
		const c = str(e.details?.command);
		if (c) failing.set(c, (failing.get(c) || 0) + 1);
	}
	const bottlenecks = [...failing].sort((a, b) => b[1] - a[1]).slice(0, 5);
	const switches = entries.filter((e) => e.kind === "goal" && (e.action === "switchGoal" || e.action === "newGoal")).length;
	const lines: string[] = [
		`# 주간 회고 — ${input.from} ~ ${input.to}`,
		"",
		"## 이번 주 한눈에",
		...formatMetrics(m).map((l) => `- ${l}`),
		...(input.usage ? [`- vibe-coders 사용량(주간): ${formatKrw(input.usage.cost_krw)} · 요청 ${input.usage.requests}회 · 토큰 ${input.usage.tokens.toLocaleString("ko-KR")} · 성공률 ${(input.usage.success_rate * 100).toFixed(1)}%`] : []),
		...(switches ? [`- 목표 생성/전환 ${switches}회`] : []),
		"",
		"## 완료한 것",
		...(m.completedItems.length ? m.completedItems.map((c) => `- [x] ${c}`) : ["- (완료 항목 없음)"]),
		"",
		"## 병목과 실패",
		...(bottlenecks.length ? bottlenecks.map(([c, n]) => `- \`${c}\` — 실패 ${n}회`) : ["- 반복 실패 없음"]),
		...(m.tasksWithoutGoalUpdate ? [`- 목표 파일을 갱신하지 않은 작업 ${m.tasksWithoutGoalUpdate}회 — 루프 규칙을 다시 확인하세요.`] : []),
		...(m.healthWarnings ? [`- 정체 경고 ${m.healthWarnings}회`] : []),
		"",
		"## 다음 주",
	];
	if (goal) {
		lines.push(`- 목표: ${goal.title} (${goal.status}) — 완료 기준 ${goal.done}/${goal.total}`);
		lines.push(`- 다음 행동: ${goal.next}`);
		const now = goal.now.filter((l) => !/^- \[[xX]\]/.test(l)).map(stripTask);
		const criteria = goal.criteria.filter((l) => !/^- \[[xX]\]/.test(l)).map(stripTask);
		for (const n of now.slice(0, 5)) lines.push(`- [ ] Now: ${n}`);
		for (const c of criteria.slice(0, 5)) lines.push(`- [ ] 완료 기준: ${c}`);
	} else {
		lines.push("- 활성 목표 없음 — `Vibe Code: 새 목표`로 시작하세요.");
	}
	lines.push("");
	return lines.join("\n");
}

export function registerGoalMetrics(host: CoreHost): void {
	const { context, output } = host;
	context.subscriptions.push(
		vscode.commands.registerCommand("vibe-code.showGoalMetrics", async () => {
			const paths = ensureWorkspacePaths();
			if (!paths) return;
			const metrics = computeGoalMetrics(readRecentAudit(paths.audit, 7), 7);
			output.show(true);
			output.appendLine("\n=== 목표 진행 지표 (7일) ===");
			for (const line of formatMetrics(metrics)) output.appendLine(`  ${line}`);
			output.appendLine("================================\n");
			writeAudit(host, "goal", "showGoalMetrics", { completed: metrics.completedItems.length, verifications: metrics.verifications });
		}),
		vscode.commands.registerCommand("vibe-code.createRetro", async () => {
			const paths = ensureWorkspacePaths();
			if (!paths) return;
			const entries = readRecentAudit(paths.audit, 7);
			const metrics = computeGoalMetrics(entries, 7);
			const usage = await cachedWeeklyUsage(host).catch(() => null);
			const to = kstDate();
			const from = kstDate(new Date(Date.now() - 6 * 864e5));
			const retro = buildRetro({ from, to, metrics, entries, goalText: fs.existsSync(paths.current) ? readUtf8(paths.current) : "", usage: usage?.ok ? usage.report : undefined });
			const dir = path.join(paths.base, "retro");
			fs.mkdirSync(dir, { recursive: true });
			const file = path.join(dir, `${to}-weekly.md`);
			fs.writeFileSync(file, retro, "utf8");
			const doc = await vscode.workspace.openTextDocument(file);
			await vscode.window.showTextDocument(doc);
			writeAudit(host, "goal", "createRetro", { file: `.vibe-code/retro/${path.basename(file)}`, completed: metrics.completedItems.length });
			log(host, `retro written: ${file}`);
		}),
	);
	log(host, "goal metrics + retro commands registered");
}
