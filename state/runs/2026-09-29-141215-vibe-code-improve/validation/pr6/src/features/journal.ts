import * as fs from "node:fs";
import * as path from "node:path";
import * as vscode from "vscode";
import type { CoreHost } from "../core/host";
import { log } from "../log";
import { kstClock, kstDate } from "../util/kst";
import { NO_WORKSPACE_MESSAGE, openFile, workspaceRoot } from "./workspace";

function journalTemplate(ymd: string, clock: string): string {
	return `# Vibe Code 작업 일지 — ${ymd}

## 세션

- ${clock} — 확장 활성화

## 작업 내용

(직접 기록하거나 에이전트에게 요청하세요: "오늘 작업 일지에 다음 내용 추가해줘 ...")

## 다음 세션을 위한 노트

`;
}

/** Create or append to today's KST journal and make sure the long-term state directories exist. */
export function initJournal(host: CoreHost): void {
	const ws = workspaceRoot();
	if (!ws) return;
	const journalDir = path.join(ws, ".vibe-code", "journal");
	fs.mkdirSync(journalDir, { recursive: true });
	const ymd = kstDate();
	const file = path.join(journalDir, `${ymd}.md`);
	const clock = kstClock();
	if (!fs.existsSync(file)) fs.writeFileSync(file, journalTemplate(ymd, clock));
	else fs.appendFileSync(file, `- ${clock} — 확장 재활성화\n`);

	for (const dirName of ["plans", "goals", "sessions", "checkpoints"]) {
		fs.mkdirSync(path.join(ws, ".vibe-code", dirName), { recursive: true });
	}

	const previous = fs
		.readdirSync(journalDir)
		.filter((name) => name.endsWith(".md") && name !== `${ymd}.md`)
		.sort()
		.reverse();
	if (previous.length > 0) log(host, `previous journal: .vibe-code/journal/${previous[0]}`);
	log(host, `journal: .vibe-code/journal/${ymd}.md`);
}

export function registerJournalCommands(host: CoreHost): void {
	host.context.subscriptions.push(
		vscode.commands.registerCommand("vibe-code.openJournal", async () => {
			const ws = workspaceRoot();
			if (!ws) {
				void vscode.window.showWarningMessage(NO_WORKSPACE_MESSAGE);
				return;
			}
			const file = path.join(ws, ".vibe-code", "journal", `${kstDate()}.md`);
			if (fs.existsSync(file)) await openFile(file);
			else void vscode.window.showInformationMessage(`오늘 작업 일지가 아직 없습니다: ${file}`);
		}),
	);
}
