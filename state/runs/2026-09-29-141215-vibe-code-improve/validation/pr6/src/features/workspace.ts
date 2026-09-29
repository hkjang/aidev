import * as fs from "node:fs";
import * as path from "node:path";
import * as vscode from "vscode";
import type { CoreHost } from "../core/host";
import { log } from "../log";
import { kstDate } from "../util/kst";

export const NO_WORKSPACE_MESSAGE = "워크스페이스를 먼저 열어주세요.";

export interface WorkspacePaths {
	ws: string;
	base: string;
	goals: string;
	sessions: string;
	checkpoints: string;
	plans: string;
	archive: string;
	journal: string;
	audit: string;
	/** `.vibe-code/goals/current.md` */
	current: string;
}

export function workspaceRoot(): string | undefined {
	return vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
}

/**
 * Resolve the `.vibe-code/` state directories of the first workspace folder and make sure
 * they exist. Returns null (and warns unless `silent`) when no workspace is open.
 */
export function ensureWorkspacePaths(silent = false): WorkspacePaths | null {
	const ws = workspaceRoot();
	if (!ws) {
		if (!silent) void vscode.window.showWarningMessage(NO_WORKSPACE_MESSAGE);
		return null;
	}
	const base = path.join(ws, ".vibe-code");
	const paths: WorkspacePaths = {
		ws,
		base,
		goals: path.join(base, "goals"),
		sessions: path.join(base, "sessions"),
		checkpoints: path.join(base, "checkpoints"),
		plans: path.join(base, "plans"),
		archive: path.join(base, "plans", "archive"),
		journal: path.join(base, "journal"),
		audit: path.join(base, "audit"),
		current: path.join(base, "goals", "current.md"),
	};
	for (const dir of [paths.goals, paths.sessions, paths.checkpoints, paths.plans, paths.journal, paths.audit]) {
		fs.mkdirSync(dir, { recursive: true });
	}
	return paths;
}

export type AuditKind = "goal" | "plan" | "proxy" | "command";

/** Append one JSON line to `.vibe-code/audit/<KST date>.jsonl`. Returns the file path or null. */
export function writeAudit(host: CoreHost, kind: AuditKind, action: string, details: Record<string, unknown> = {}): string | null {
	try {
		const paths = ensureWorkspacePaths(true);
		if (!paths) return null;
		const file = path.join(paths.audit, `${kstDate()}.jsonl`);
		const entry = { ts: new Date().toISOString(), kind, action, details };
		fs.appendFileSync(file, JSON.stringify(entry) + "\n", "utf8");
		return file;
	} catch (error) {
		log(host, `audit write skipped: ${error}`);
		return null;
	}
}

export async function openFile(file: string): Promise<void> {
	const doc = await vscode.workspace.openTextDocument(file);
	await vscode.window.showTextDocument(doc);
}

export function readUtf8(file: string): string {
	return fs.readFileSync(file, "utf8");
}

export function listMarkdown(dir: string): string[] {
	if (!fs.existsSync(dir)) return [];
	return fs
		.readdirSync(dir)
		.filter((name) => name.endsWith(".md"))
		.sort();
}
