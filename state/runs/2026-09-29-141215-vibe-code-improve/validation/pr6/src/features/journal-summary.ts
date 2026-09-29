import * as fs from "node:fs";
import * as path from "node:path";
import { kstClock, kstDate } from "../util/kst";
import { matchLine, subsection, taskLines, stripTask } from "../util/markdown";

export interface AuditEntry {
	ts: string;
	kind: string;
	action: string;
	details?: Record<string, unknown>;
}

const SUMMARY_MARKER = "<!-- vibe-code:summary-until ";

export function parseAuditLines(text: string): AuditEntry[] {
	const entries: AuditEntry[] = [];
	for (const line of text.split(/\r?\n/)) {
		if (!line.trim()) continue;
		try {
			const row = JSON.parse(line);
			if (row && typeof row.ts === "string" && typeof row.action === "string") entries.push(row);
		} catch {
			// skip malformed lines
		}
	}
	return entries;
}

/** ISO timestamp of the last summary written into a journal, or "" when none. */
export function lastSummaryTs(journal: string): string {
	const matches = [...journal.matchAll(/<!-- vibe-code:summary-until (\S+) -->/g)];
	return matches.length ? matches[matches.length - 1][1] : "";
}

const count = (entries: AuditEntry[], kind: string, action: string) => entries.filter((e) => e.kind === kind && e.action === action).length;
const str = (value: unknown) => (typeof value === "string" ? value : "");

/**
 * Build the markdown block appended to the journal at deactivate. Returns null when there is
 * nothing new to report. Pure: takes the audit entries and the current goal text.
 */
export function buildSessionSummary(entries: AuditEntry[], goalText: string, clock: string, since: string): string | null {
	const fresh = since ? entries.filter((e) => e.ts > since) : entries;
	if (fresh.length === 0) return null;
	const lines: string[] = [`## 세션 요약 (${clock})`, ""];

	const completed = fresh.filter((e) => e.kind === "plan" && (e.action === "advanceCurrentPlan" || e.action === "completePlanItem")).map((e) => str(e.details?.completed)).filter(Boolean);
	if (completed.length) lines.push(`- 완료한 계획 항목 ${completed.length}개: ${completed.join("; ")}`);

	const statusChanges = fresh.filter((e) => e.kind === "plan" && e.action === "setCurrentPlanStatus").map((e) => `${str(e.details?.status)} (${str(e.details?.planFile).replace(".vibe-code/plans/", "")})`);
	if (statusChanges.length) lines.push(`- 계획 상태 변경: ${statusChanges.join(", ")}`);

	const archived = fresh.filter((e) => e.kind === "plan" && e.action === "archiveDonePlans").reduce((n, e) => n + (Number(e.details?.count) || 0), 0);
	if (archived) lines.push(`- 보관한 완료 계획: ${archived}개`);

	const handoffs = count(fresh, "goal", "createGoalHandoff");
	if (handoffs) lines.push(`- 목표 핸드오프 생성: ${handoffs}회`);

	const proxyApplied = fresh.filter((e) => e.kind === "proxy" && e.action === "applyVibeCodersProxy").map((e) => str(e.details?.profile)).filter(Boolean);
	if (proxyApplied.length) lines.push(`- vibe-coders 프록시 적용: ${proxyApplied[proxyApplied.length - 1]}`);
	const restored = count(fresh, "proxy", "restorePreviousProviderProfile");
	if (restored) lines.push(`- 이전 프로바이더 복원: ${restored}회`);

	const commandsRun = fresh.filter((e) => e.kind === "command" && e.action === "approved").length;
	const commandsDenied = fresh.filter((e) => e.kind === "command" && e.action === "denied").length;
	const commandsFailed = fresh.filter((e) => e.kind === "command" && e.action === "exited" && Number(e.details?.exitCode) !== 0).map((e) => `${str(e.details?.command)} (exit ${e.details?.exitCode})`);
	if (commandsRun || commandsDenied) lines.push(`- 실행한 명령: ${commandsRun}개 승인, ${commandsDenied}개 거부${commandsFailed.length ? ` · 실패 ${commandsFailed.length}개: ${commandsFailed.slice(0, 3).join("; ")}` : ""}`);

	const failures = fresh.filter((e) => e.action.endsWith("Failed")).map((e) => `${e.action}: ${str(e.details?.error)}`);
	if (failures.length) lines.push(`- 실패: ${failures.join("; ")}`);

	const byKind = new Map<string, number>();
	for (const e of fresh) byKind.set(e.kind, (byKind.get(e.kind) || 0) + 1);
	lines.push(`- 감사 이벤트: ${[...byKind].map(([k, n]) => `${k} ${n}`).join(", ")}`);

	if (goalText) {
		const next = matchLine(goalText, "- 다음 행동");
		const now = taskLines(subsection(goalText, "Now")).filter((l) => !/^- \[[xX]\]/.test(l)).map(stripTask);
		if (next) lines.push(`- 다음 행동: ${next}`);
		if (now.length) lines.push(`- 목표 Now: ${now.slice(0, 3).join("; ")}${now.length > 3 ? ` 외 ${now.length - 3}개` : ""}`);
	}

	const until = fresh[fresh.length - 1].ts;
	lines.push("", `${SUMMARY_MARKER}${until} -->`, "");
	return lines.join("\n");
}

/** Append today's summary to today's journal. Safe to call from `deactivate`; never throws. */
export function appendSessionSummary(workspaceRoot: string | undefined): boolean {
	try {
		if (!workspaceRoot) return false;
		const today = kstDate();
		const auditFile = path.join(workspaceRoot, ".vibe-code", "audit", `${today}.jsonl`);
		const journalFile = path.join(workspaceRoot, ".vibe-code", "journal", `${today}.md`);
		if (!fs.existsSync(auditFile) || !fs.existsSync(journalFile)) return false;
		const journal = fs.readFileSync(journalFile, "utf8");
		const goalFile = path.join(workspaceRoot, ".vibe-code", "goals", "current.md");
		const goalText = fs.existsSync(goalFile) ? fs.readFileSync(goalFile, "utf8") : "";
		const summary = buildSessionSummary(parseAuditLines(fs.readFileSync(auditFile, "utf8")), goalText, kstClock(), lastSummaryTs(journal));
		if (!summary) return false;
		fs.appendFileSync(journalFile, (journal.endsWith("\n") ? "" : "\n") + "\n" + summary, "utf8");
		return true;
	} catch {
		return false;
	}
}
