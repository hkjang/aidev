import * as fs from "node:fs";
import * as vscode from "vscode";
import type { CoreHost } from "../core/host";
import { log } from "../log";
import { config } from "../settings";
import { matchLine, stripTask } from "../util/markdown";
import { parseGoal } from "./goals";
import { ensureWorkspacePaths, readUtf8, writeAudit } from "./workspace";

export interface NowSeen {
	item: string;
	sessions: number;
	since: string;
}

export interface GoalHealth {
	ageHours: number;
	stale: boolean;
	stuck: boolean;
	warning?: string;
}

/** Hours since `마지막 갱신:` (or `작성일:`), interpreting the KST timestamp. Infinity when unparseable. */
export function goalAgeHours(text: string, now: number = Date.now()): number {
	const stamp = matchLine(text, "마지막 갱신") || matchLine(text, "작성일");
	const m = stamp.match(/(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2})/);
	if (!m) return Number.POSITIVE_INFINITY;
	return (now - Date.UTC(+m[1], +m[2] - 1, +m[3], +m[4] - 9, +m[5])) / 36e5;
}

/** Pure health verdict from the goal text, the per-session Now tracking and thresholds. */
export function assessGoalHealth(text: string, seen: NowSeen | undefined, staleHours: number, stuckSessions = 3, now: number = Date.now()): GoalHealth {
	const goal = parseGoal(text);
	const ageHours = goalAgeHours(text, now);
	const active = goal.status.toLowerCase() === "active";
	const stale = active && Number.isFinite(ageHours) && ageHours > staleHours;
	const stuck = active && !!seen && seen.sessions >= stuckSessions;
	const parts: string[] = [];
	if (stale) parts.push(`마지막 갱신 ${Math.round(ageHours)}시간 전`);
	if (stuck && seen) parts.push(`같은 Now 항목 ${seen.sessions}세션째: ${seen.item}`);
	return { ageHours, stale, stuck, warning: parts.length ? parts.join(" · ") + " — 작업을 더 작게 쪼개거나 목표 파일을 갱신하세요." : undefined };
}

const SEEN_KEY = "vibeCode.goalNowSeen";

/** Update the "same Now item across sessions" counter once per activation. */
export async function trackNowItem(host: CoreHost): Promise<NowSeen | undefined> {
	const paths = ensureWorkspacePaths(true);
	if (!paths || !fs.existsSync(paths.current)) return undefined;
	const goal = parseGoal(readUtf8(paths.current));
	const first = goal.now.filter((l) => !/^- \[[xX]\]/.test(l)).map(stripTask)[0];
	if (!first || goal.status.toLowerCase() !== "active") {
		await host.context.workspaceState.update(SEEN_KEY, undefined);
		return undefined;
	}
	const previous = host.context.workspaceState.get<NowSeen>(SEEN_KEY);
	const next: NowSeen = previous && previous.item === first ? { ...previous, sessions: previous.sessions + 1 } : { item: first, sessions: 1, since: new Date().toISOString() };
	await host.context.workspaceState.update(SEEN_KEY, next);
	return next;
}

export function currentHealth(host: CoreHost, text: string): GoalHealth {
	return assessGoalHealth(text, host.context.workspaceState.get<NowSeen>(SEEN_KEY), config().get<number>("goalStaleHours") ?? 48);
}

/** Stall detection on activation: counts sessions with the same Now item and warns when stuck or stale. */
export async function checkGoalHealth(host: CoreHost): Promise<void> {
	const seen = await trackNowItem(host);
	const paths = ensureWorkspacePaths(true);
	if (!paths || !fs.existsSync(paths.current)) return;
	const health = currentHealth(host, readUtf8(paths.current));
	if (!health.warning) return;
	writeAudit(host, "goal", "healthWarning", { stale: health.stale, stuck: health.stuck, ageHours: Math.round(health.ageHours), sessions: seen?.sessions });
	log(host, `goal health: ${health.warning}`);
	void vscode.window.showWarningMessage(`목표 정체 감지: ${health.warning}`, "목표 열기", "최신 계획 열기").then(async (choice) => {
		if (choice === "목표 열기") await vscode.commands.executeCommand("vibe-code.openCurrentGoal");
		else if (choice === "최신 계획 열기") await vscode.commands.executeCommand("vibe-code.openLatestPlan");
	});
}
