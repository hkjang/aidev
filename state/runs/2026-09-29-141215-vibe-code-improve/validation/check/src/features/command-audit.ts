import * as path from "node:path";
import * as vscode from "vscode";
import type { CommandEvent, CoreHost } from "../core/host";
import { log } from "../log";
import { ensureWorkspacePaths, readUtf8, writeAudit } from "./workspace";
import * as fs from "node:fs";
import { checkpointBeforeCommand } from "./checkpoints";

let activeHost: CoreHost | undefined;

/**
 * Receives execute_command lifecycle events from the core (see vendor/PATCHES.md) and appends
 * them to `.vibe-code/audit/<date>.jsonl` as `kind: "command"`. Installed before activation, so
 * events that arrive before `registerCommandAudit` ran are dropped silently.
 */
export function handleCommandEvent(event: CommandEvent): void {
	if (!activeHost) return;
	const details: Record<string, unknown> = { phase: event.phase, command: event.command };
	if (event.cwd) details.cwd = event.cwd;
	if (event.exitCode !== undefined) details.exitCode = event.exitCode;
	if (event.executionId) details.executionId = event.executionId;
	if (event.taskId) details.taskId = event.taskId;
	if (event.phase === "approved") checkpointBeforeCommand(activeHost, event.command);
	writeAudit(activeHost, "command", event.phase, details);
	if (event.phase === "denied") log(activeHost, `command denied: ${event.command}`);
	else if (event.phase === "exited" && event.exitCode !== 0 && event.exitCode !== undefined) log(activeHost, `command exited ${event.exitCode}: ${event.command}`);
}

export interface CommandAuditRow {
	ts: string;
	phase: string;
	command: string;
	cwd?: string;
	exitCode?: number;
}

/** Recent command events across all audit files, oldest first. */
export function readCommandHistory(auditDir: string, limit = 30): CommandAuditRow[] {
	if (!fs.existsSync(auditDir)) return [];
	const rows: CommandAuditRow[] = [];
	const files = fs
		.readdirSync(auditDir)
		.filter((name) => name.endsWith(".jsonl"))
		.sort();
	for (const name of files.slice(-7)) {
		for (const line of readUtf8(path.join(auditDir, name)).split(/\r?\n/).filter(Boolean)) {
			try {
				const row = JSON.parse(line);
				if (row.kind !== "command") continue;
				const d = row.details || {};
				rows.push({ ts: row.ts, phase: String(d.phase || row.action), command: String(d.command || ""), cwd: d.cwd, exitCode: d.exitCode });
			} catch {
				// skip malformed lines
			}
		}
	}
	return rows.slice(-limit);
}

export function registerCommandAudit(host: CoreHost): void {
	activeHost = host;
	const { context, output } = host;
	context.subscriptions.push(
		{ dispose: () => (activeHost = undefined) },
		vscode.commands.registerCommand("vibe-code.showCommandHistory", async () => {
			const paths = ensureWorkspacePaths(true);
			if (!paths) return;
			const rows = readCommandHistory(paths.audit);
			output.show(true);
			output.appendLine("\n=== 명령 실행 이력 (최근 7일, 최대 30건) ===");
			if (rows.length === 0) output.appendLine("  (기록된 명령 실행 없음)");
			for (const row of rows) {
				const mark = row.phase === "denied" ? "✗ 거부" : row.phase === "approved" ? "▶ 승인" : row.exitCode === 0 ? "✓ 종료 0" : `! 종료 ${row.exitCode}`;
				output.appendLine(`  ${row.ts} :: ${mark} :: ${row.command}${row.cwd ? ` (cwd: ${row.cwd})` : ""}`);
			}
			output.appendLine("=========================================\n");
		}),
	);
	log(host, "command audit hook registered");
}
