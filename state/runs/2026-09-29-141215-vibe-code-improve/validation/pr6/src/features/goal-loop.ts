import * as fs from "node:fs";
import * as vscode from "vscode";
import type { CoreHost, TaskEvent } from "../core/host";
import { log } from "../log";
import { config } from "../settings";
import { matchLine, progressLabel, stripTask } from "../util/markdown";
import { createHandoffFile, GOAL_FILE, parseGoal } from "./goals";
import { ensureWorkspacePaths, readUtf8, writeAudit } from "./workspace";

const RESUME_PROMPT = "/goal 이어서";

// --- prompt context ---------------------------------------------------------------------------

/**
 * Compact goal state that the core appends to every system prompt (see vendor/PATCHES.md).
 * Pure: takes the goal file text and returns the section, or "" when there is nothing useful.
 */
export function buildPromptContext(goalText: string): string {
	if (!goalText.trim()) return "";
	const goal = parseGoal(goalText);
	if (!["active", "draft"].includes(goal.status.toLowerCase())) return "";
	const now = goal.now.filter((l) => !/^- \[[xX]\]/.test(l)).map(stripTask).slice(0, 4);
	const next = goal.nextItems.filter((l) => !/^- \[[xX]\]/.test(l)).map(stripTask).slice(0, 2);
	const lines = [
		"====",
		"",
		"VIBE CODE GOAL STATE",
		"",
		`A persistent goal is active in ${GOAL_FILE}. Work on it unless the user asks for something else.`,
		"",
		`- 목표: ${goal.title}`,
		`- 상태: ${goal.status} / 지금 단계: ${goal.phase}`,
		`- 다음 행동: ${goal.next}`,
		`- 완료 기준 진행: ${progressLabel(goal, "체크리스트 없음")}`,
		now.length ? `- Now: ${now.join(" | ")}` : "- Now: (비어 있음 — Next에서 하나를 승격하거나 새 작업을 정의하세요)",
	];
	if (next.length) lines.push(`- Next: ${next.join(" | ")}`);
	lines.push(
		"",
		"Rules: pick the one Now item that moves the goal forward; after every meaningful change update the 작업 큐, 변경 로그 and 검증 로그 in the goal file (and the active plan under .vibe-code/plans/ if one exists); when all 완료 기준 are checked set `상태: done`; do not wait for further instructions while Now items remain.",
	);
	return "\n\n" + lines.join("\n") + "\n";
}

let promptCache: { file: string; mtimeMs: number; text: string } | undefined;

/** Called by the core for each API request; cached on the goal file's mtime. */
export function promptContext(): string {
	try {
		const ws = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
		if (!ws) return "";
		const file = `${ws}/.vibe-code/goals/current.md`;
		if (!fs.existsSync(file)) return "";
		const mtimeMs = fs.statSync(file).mtimeMs;
		if (promptCache && promptCache.file === file && promptCache.mtimeMs === mtimeMs) return promptCache.text;
		const text = buildPromptContext(readUtf8(file));
		promptCache = { file, mtimeMs, text };
		return text;
	} catch {
		return "";
	}
}

// --- task lifecycle ---------------------------------------------------------------------------

let activeHost: CoreHost | undefined;
const taskStarts = new Map<string, { startedAt: number; goalMtime: number }>();

function goalMtime(): number {
	const paths = ensureWorkspacePaths(true);
	if (!paths || !fs.existsSync(paths.current)) return 0;
	return fs.statSync(paths.current).mtimeMs;
}

function activeGoal(): { text: string; summary: ReturnType<typeof parseGoal> } | null {
	const paths = ensureWorkspacePaths(true);
	if (!paths || !fs.existsSync(paths.current)) return null;
	const text = readUtf8(paths.current);
	const summary = parseGoal(text);
	return summary.status.toLowerCase() === "active" ? { text, summary } : null;
}

async function startResumeTask(host: CoreHost, reason: string): Promise<boolean> {
	const provider = host.getProvider() as { initClineWithTask?: (task: string) => Promise<unknown> } | undefined;
	if (!provider?.initClineWithTask) {
		void vscode.window.showWarningMessage("사이드바가 아직 준비되지 않았습니다. 잠시 후 다시 시도하세요.");
		return false;
	}
	await vscode.commands.executeCommand("vibe-code.SidebarProvider.focus");
	await provider.initClineWithTask(RESUME_PROMPT);
	writeAudit(host, "goal", "resumeGoal", { reason });
	log(host, `goal resumed (${reason})`);
	return true;
}

export function handleTaskEvent(event: TaskEvent): void {
	const host = activeHost;
	if (!host) return;
	try {
		if (event.phase === "started") {
			taskStarts.set(event.taskId, { startedAt: Date.now(), goalMtime: goalMtime() });
			return;
		}
		if (event.phase === "condensed") {
			const goal = activeGoal();
			if (!goal) return;
			const paths = ensureWorkspacePaths(true);
			if (!paths) return;
			const handoff = createHandoffFile(host, paths, "context condensed");
			writeAudit(host, "goal", "autoHandoff", { trigger: "condensed", taskId: event.taskId, handoffFile: handoff });
			log(host, `auto handoff after context condense: ${handoff}`);
			return;
		}
		const start = taskStarts.get(event.taskId);
		taskStarts.delete(event.taskId);
		if (event.phase === "aborted") return;
		if (event.isSubtask) return;
		const goal = activeGoal();
		const usage = (event.tokenUsage || {}) as Record<string, unknown>;
		const cost = Number(usage.totalCost) || 0;
		const tokens = (Number(usage.totalTokensIn) || 0) + (Number(usage.totalTokensOut) || 0);
		if (!goal) {
			writeAudit(host, "goal", "taskCompleted", { taskId: event.taskId, goalActive: false, cost, tokens });
			return;
		}
		const updated = start ? goalMtime() > start.goalMtime : true;
		const openNow = goal.summary.now.filter((l) => !/^- \[[xX]\]/.test(l)).map(stripTask);
		writeAudit(host, "goal", "taskCompleted", { taskId: event.taskId, goalActive: true, goalUpdated: updated, openNow: openNow.length, cost, tokens, durationMs: start ? Date.now() - start.startedAt : undefined });
		if (!updated) {
			void vscode.window.showWarningMessage("작업이 끝났지만 목표 파일(current.md)이 갱신되지 않았습니다.", "목표 열기", "이어서 갱신 요청").then(async (choice) => {
				if (choice === "목표 열기") await vscode.commands.executeCommand("vibe-code.openCurrentGoal");
				else if (choice === "이어서 갱신 요청") await startResumeTask(host, "goal not updated after task");
			});
			return;
		}
		if (openNow.length > 0 && (config().get<string>("goalAutoResume") || "ask") !== "never") {
			void vscode.window.showInformationMessage(`목표 Now 항목이 ${openNow.length}개 남았습니다: ${openNow[0]}`, "계속 진행", "나중에").then(async (choice) => {
				if (choice === "계속 진행") await startResumeTask(host, "next Now item after task");
			});
		}
	} catch (error) {
		log(host, `task event handling skipped: ${error}`);
	}
}

// --- auto resume on activation ----------------------------------------------------------------

function goalAgeHours(text: string): number {
	const stamp = matchLine(text, "마지막 갱신") || matchLine(text, "작성일");
	const m = stamp.match(/(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2})/);
	if (!m) return Number.POSITIVE_INFINITY;
	const then = Date.UTC(+m[1], +m[2] - 1, +m[3], +m[4] - 9, +m[5]); // KST -> UTC
	return (Date.now() - then) / 36e5;
}

/** Offer (or start) `/goal 이어서` shortly after activation when an active goal still has work. */
export function scheduleAutoResume(host: CoreHost): void {
	const mode = config().get<string>("goalAutoResume") || "ask";
	if (mode === "never") return;
	const withinHours = config().get<number>("goalResumeWithinHours") ?? 72;
	const timer = setTimeout(async () => {
		try {
			const goal = activeGoal();
			if (!goal) return;
			const openNow = goal.summary.now.filter((l) => !/^- \[[xX]\]/.test(l));
			if (openNow.length === 0) return;
			if (goalAgeHours(goal.text) > withinHours) return;
			if (mode === "always") {
				await startResumeTask(host, "auto resume on activation");
				return;
			}
			const choice = await vscode.window.showInformationMessage(`활성 목표 "${goal.summary.title}"에 Now 항목 ${openNow.length}개가 남아 있습니다. 이어서 진행할까요?`, "이어서 진행", "목표 열기", "나중에");
			if (choice === "이어서 진행") await startResumeTask(host, "resume prompt on activation");
			else if (choice === "목표 열기") await vscode.commands.executeCommand("vibe-code.openCurrentGoal");
		} catch (error) {
			log(host, `auto resume skipped: ${error}`);
		}
	}, 4000);
	host.context.subscriptions.push({ dispose: () => clearTimeout(timer) });
}

export function registerGoalLoop(host: CoreHost): void {
	activeHost = host;
	host.context.subscriptions.push(
		{ dispose: () => (activeHost = undefined) },
		vscode.commands.registerCommand("vibe-code.resumeGoal", async () => {
			const goal = activeGoal();
			if (!goal) {
				void vscode.window.showWarningMessage("활성 상태(상태: active)인 목표가 없습니다. 먼저 /goal로 목표를 시작하세요.");
				return;
			}
			await startResumeTask(host, "manual");
		}),
	);
	scheduleAutoResume(host);
	log(host, "goal loop registered (prompt context, task events, auto resume)");
}
