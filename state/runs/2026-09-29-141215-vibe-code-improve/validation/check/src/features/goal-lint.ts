import * as vscode from "vscode";
import type { CoreHost } from "../core/host";
import { log } from "../log";
import { hasSection, matchLine, section, subsection, taskLines } from "../util/markdown";

export interface LintIssue {
	line: number;
	message: string;
	severity: "error" | "warning" | "info";
}

const STATUSES = ["draft", "active", "blocked", "done"];
const PRIORITIES = ["P0", "P1", "P2", "P3"];

function lineOf(all: string[], predicate: (l: string) => boolean, fallback = 0): number {
	const idx = all.findIndex(predicate);
	return idx >= 0 ? idx : fallback;
}

function openTasks(block: string): number {
	return taskLines(block).filter((l) => !/^- \[[xX]\]/.test(l)).length;
}

/** Structural checks for `.vibe-code/goals/*.md`. Pure. */
export function lintGoal(text: string): LintIssue[] {
	const all = text.split(/\r?\n/);
	const issues: LintIssue[] = [];
	if (!/^#\s*목표:\s*\S/m.test(text)) issues.push({ line: 0, message: "첫 줄은 `# 목표: <제목>` 형식이어야 합니다.", severity: "error" });
	for (const name of ["목표", "완료 기준", "현재 상태", "작업 큐", "검증 로그"]) {
		if (!hasSection(text, name)) issues.push({ line: 0, message: `\`## ${name}\` 섹션이 없습니다.`, severity: "warning" });
	}
	const status = matchLine(text, "상태");
	const statusLine = lineOf(all, (l) => /^상태:/.test(l));
	if (!status) issues.push({ line: 0, message: "`상태:` 줄이 없습니다 (draft | active | blocked | done).", severity: "warning" });
	else if (!STATUSES.includes(status.toLowerCase())) issues.push({ line: statusLine, message: `알 수 없는 상태 "${status}". draft | active | blocked | done 중 하나여야 합니다.`, severity: "error" });
	if (!matchLine(text, "- 다음 행동")) issues.push({ line: lineOf(all, (l) => /^## 현재 상태/.test(l)), message: "`- 다음 행동:` 줄이 비어 있습니다. 바로 할 일을 한 줄로 적으세요.", severity: "info" });
	const nowOpen = openTasks(subsection(text, "Now"));
	if (nowOpen > 4) issues.push({ line: lineOf(all, (l) => /^### Now/.test(l)), message: `Now에 미완료 항목이 ${nowOpen}개입니다. 4개 이하로 줄이고 나머지는 Next로 옮기세요.`, severity: "warning" });
	if (status.toLowerCase() === "done") {
		const open = openTasks(section(text, "완료 기준"));
		if (open > 0) issues.push({ line: statusLine, message: `상태가 done이지만 완료 기준 ${open}개가 체크되지 않았습니다.`, severity: "warning" });
	}
	return issues;
}

/** Structural checks for `.vibe-code/plans/*.md`. Pure. */
export function lintPlan(text: string): LintIssue[] {
	const all = text.split(/\r?\n/);
	const issues: LintIssue[] = [];
	if (!/^#\s*계획:\s*\S/m.test(text)) issues.push({ line: 0, message: "첫 줄은 `# 계획: <제목>` 형식이어야 합니다.", severity: "error" });
	for (const name of ["단계", "Now", "Next", "Done"]) {
		if (!hasSection(text, name)) issues.push({ line: 0, message: `\`## ${name}\` 섹션이 없습니다.`, severity: "warning" });
	}
	const status = matchLine(text, "상태");
	const statusLine = lineOf(all, (l) => /^상태:/.test(l));
	if (!status) issues.push({ line: 0, message: "`상태:` 줄이 없습니다 (draft | active | blocked | done).", severity: "warning" });
	else if (!STATUSES.includes(status.toLowerCase())) issues.push({ line: statusLine, message: `알 수 없는 상태 "${status}".`, severity: "error" });
	const priority = matchLine(text, "우선순위");
	if (priority && !PRIORITIES.includes(priority.toUpperCase())) issues.push({ line: lineOf(all, (l) => /^우선순위:/.test(l)), message: `알 수 없는 우선순위 "${priority}". P0 | P1 | P2 | P3`, severity: "error" });
	const nowOpen = openTasks(section(text, "Now"));
	if (nowOpen > 3) issues.push({ line: lineOf(all, (l) => /^## Now/.test(l)), message: `Now에 미완료 항목이 ${nowOpen}개입니다. 한 번에 1~3개만 두세요.`, severity: "warning" });
	if (status.toLowerCase() === "done") {
		const open = ["단계", "Now", "검증 계획"].reduce((n, name) => n + openTasks(section(text, name)), 0);
		if (open > 0) issues.push({ line: statusLine, message: `상태가 done이지만 미완료 항목이 ${open}개 있습니다.`, severity: "warning" });
	}
	if (!matchLine(text, "연결 목표")) issues.push({ line: lineOf(all, (l) => /^작성일:/.test(l)), message: "`연결 목표:` 줄이 없습니다. `현재 계획 목표 연결` 명령으로 목표와 연결하세요.", severity: "info" });
	return issues;
}

const SEVERITY = { error: vscode.DiagnosticSeverity.Error, warning: vscode.DiagnosticSeverity.Warning, info: vscode.DiagnosticSeverity.Information };

function kindOf(document: vscode.TextDocument): "goal" | "plan" | null {
	const p = document.uri.fsPath.replace(/\\/g, "/");
	if (!p.endsWith(".md") || !p.includes("/.vibe-code/")) return null;
	if (p.includes("/.vibe-code/goals/")) return "goal";
	if (p.includes("/.vibe-code/plans/") && !p.includes("/archive/")) return "plan";
	return null;
}

/** Diagnostics (squiggles + Problems panel) for goal and plan files while they are open. */
export function registerGoalLint(host: CoreHost): void {
	const collection = vscode.languages.createDiagnosticCollection("vibe-code");
	const timers = new Map<string, NodeJS.Timeout>();
	const lint = (document: vscode.TextDocument) => {
		const kind = kindOf(document);
		if (!kind) return;
		const issues = kind === "goal" ? lintGoal(document.getText()) : lintPlan(document.getText());
		collection.set(
			document.uri,
			issues.map((issue) => {
				const line = Math.min(issue.line, Math.max(0, document.lineCount - 1));
				const diagnostic = new vscode.Diagnostic(new vscode.Range(line, 0, line, document.lineAt(line).text.length), issue.message, SEVERITY[issue.severity]);
				diagnostic.source = "vibe-code";
				return diagnostic;
			}),
		);
	};
	const schedule = (document: vscode.TextDocument) => {
		const key = document.uri.toString();
		clearTimeout(timers.get(key));
		timers.set(key, setTimeout(() => lint(document), 300));
	};
	host.context.subscriptions.push(
		collection,
		vscode.workspace.onDidOpenTextDocument(lint),
		vscode.workspace.onDidChangeTextDocument((e) => schedule(e.document)),
		vscode.workspace.onDidCloseTextDocument((d) => collection.delete(d.uri)),
	);
	for (const document of vscode.workspace.textDocuments) lint(document);
	log(host, "goal/plan lint registered");
}
