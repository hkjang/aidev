import * as fs from "node:fs";
import * as path from "node:path";
import { kstDate, kstStamp } from "../util/kst";
import { matchLine } from "../util/markdown";
import { handoffTemplate } from "./goals";

/** Latest handoff file mtime under `.vibe-code/sessions/`, 0 when none. */
function lastHandoffMs(sessions: string): number {
	if (!fs.existsSync(sessions)) return 0;
	return fs
		.readdirSync(sessions)
		.filter((n) => n.endsWith("-handoff.md"))
		.reduce((max, n) => Math.max(max, fs.statSync(path.join(sessions, n)).mtimeMs), 0);
}

/**
 * On deactivate: if the goal is active and today's audit trail has entries newer than the last
 * handoff, write one so the next session can resume without the chat history. Never throws.
 */
export function handoffOnExit(workspaceRoot: string | undefined): string | null {
	try {
		if (!workspaceRoot) return null;
		const base = path.join(workspaceRoot, ".vibe-code");
		const current = path.join(base, "goals", "current.md");
		if (!fs.existsSync(current)) return null;
		const text = fs.readFileSync(current, "utf8");
		if (matchLine(text, "상태").toLowerCase() !== "active") return null;
		const audit = path.join(base, "audit", `${kstDate()}.jsonl`);
		if (!fs.existsSync(audit)) return null;
		const sessions = path.join(base, "sessions");
		fs.mkdirSync(sessions, { recursive: true });
		const since = lastHandoffMs(sessions);
		const fresh = fs
			.readFileSync(audit, "utf8")
			.split(/\r?\n/)
			.filter(Boolean)
			.some((line) => {
				try {
					return new Date(JSON.parse(line).ts).getTime() > since;
				} catch {
					return false;
				}
			});
		if (!fresh) return null;
		const stamp = kstStamp();
		const file = path.join(sessions, `${stamp.file}-handoff.md`);
		fs.writeFileSync(file, handoffTemplate(stamp.human, text), "utf8");
		fs.appendFileSync(current, `\n- ${stamp.human} - handoff 생성: .vibe-code/sessions/${path.basename(file)} (세션 종료)\n`, "utf8");
		return file;
	} catch {
		return null;
	}
}
