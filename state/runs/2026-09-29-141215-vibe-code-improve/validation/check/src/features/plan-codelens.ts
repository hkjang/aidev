import * as path from "node:path";
import * as vscode from "vscode";
import type { CoreHost } from "../core/host";
import { log } from "../log";
import { kstStamp } from "../util/kst";
import { isDoneTask, isTaskLine, moveTaskToSection, sectionOfLine, stripTask, toggleCheckbox, touchPlan } from "../util/markdown";
import { selectPlanName, setActivePlanName } from "./plans";
import { ensureWorkspacePaths, writeAudit } from "./workspace";

const PLAN_SELECTOR: vscode.DocumentSelector = { scheme: "file", pattern: "**/.vibe-code/plans/*.md" };

function lens(range: vscode.Range, title: string, command: string, args: unknown[]): vscode.CodeLens {
	return new vscode.CodeLens(range, { title, command, arguments: args });
}

/** Inline actions on plan files: complete / promote / toggle checklist lines, select as active plan. */
class PlanCodeLensProvider implements vscode.CodeLensProvider {
	provideCodeLenses(document: vscode.TextDocument): vscode.CodeLens[] {
		const text = document.getText();
		const result: vscode.CodeLens[] = [];
		const args = (line: number) => [document.uri.toString(), line];
		for (let i = 0; i < document.lineCount; i++) {
			const line = document.lineAt(i).text;
			const range = new vscode.Range(i, 0, i, 0);
			if (i === 0 && /^# 계획:/.test(line)) {
				result.push(lens(range, "$(target) 현재 계획으로 선택", "vibe-code.selectPlanFile", args(i)));
				continue;
			}
			if (!isTaskLine(line)) continue;
			const sectionName = sectionOfLine(text, i);
			if (sectionName === "Now" && !isDoneTask(line)) {
				result.push(lens(range, "$(pass) 완료로 이동", "vibe-code.completePlanItem", args(i)));
				result.push(lens(range, "$(arrow-right) Next로 되돌리기", "vibe-code.deferPlanItem", args(i)));
			} else if (sectionName === "Next" && !isDoneTask(line)) {
				result.push(lens(range, "$(arrow-up) Now로 승격", "vibe-code.promotePlanItem", args(i)));
			} else {
				result.push(lens(range, isDoneTask(line) ? "$(circle-outline) 체크 해제" : "$(check) 체크", "vibe-code.togglePlanCheckbox", args(i)));
			}
		}
		return result;
	}
}

async function editDocument(uriString: string, edit: (text: string) => string | null): Promise<{ document: vscode.TextDocument; before: string } | null> {
	const document = await vscode.workspace.openTextDocument(vscode.Uri.parse(uriString));
	const before = document.getText();
	const after = edit(before);
	if (after === null || after === before) return null;
	const workspaceEdit = new vscode.WorkspaceEdit();
	workspaceEdit.replace(document.uri, new vscode.Range(0, 0, document.lineCount, 0), after);
	if (!(await vscode.workspace.applyEdit(workspaceEdit))) return null;
	await document.save();
	return { document, before };
}

export function registerPlanCodeLens(host: CoreHost): void {
	const { context } = host;
	const planFile = (uri: vscode.Uri) => ".vibe-code/plans/" + path.basename(uri.fsPath);
	const lineText = (text: string, line: number) => (text.split(/\r?\n/)[line] || "").trim();

	context.subscriptions.push(
		vscode.languages.registerCodeLensProvider(PLAN_SELECTOR, new PlanCodeLensProvider()),
		vscode.commands.registerCommand("vibe-code.completePlanItem", async (uri: string, line: number) => {
			const result = await editDocument(uri, (text) => {
				const moved = moveTaskToSection(text, line, "Done", (l) => "- [x] " + stripTask(l));
				return moved === null ? null : touchPlan(moved, "active", kstStamp().human);
			});
			if (!result) return;
			writeAudit(host, "plan", "completePlanItem", { planFile: planFile(result.document.uri), completed: stripTask(lineText(result.before, line)) });
		}),
		vscode.commands.registerCommand("vibe-code.deferPlanItem", async (uri: string, line: number) => {
			const result = await editDocument(uri, (text) => {
				const moved = moveTaskToSection(text, line, "Next");
				return moved === null ? null : touchPlan(moved, null, kstStamp().human);
			});
			if (!result) return;
			writeAudit(host, "plan", "deferPlanItem", { planFile: planFile(result.document.uri), item: stripTask(lineText(result.before, line)) });
		}),
		vscode.commands.registerCommand("vibe-code.promotePlanItem", async (uri: string, line: number) => {
			const result = await editDocument(uri, (text) => {
				const moved = moveTaskToSection(text, line, "Now");
				return moved === null ? null : touchPlan(moved, "active", kstStamp().human);
			});
			if (!result) return;
			writeAudit(host, "plan", "promotePlanItem", { planFile: planFile(result.document.uri), promoted: stripTask(lineText(result.before, line)) });
		}),
		vscode.commands.registerCommand("vibe-code.togglePlanCheckbox", async (uri: string, line: number) => {
			const result = await editDocument(uri, (text) => touchPlan(toggleCheckbox(text, line), null, kstStamp().human));
			if (!result) return;
			writeAudit(host, "plan", "togglePlanCheckbox", { planFile: planFile(result.document.uri), item: stripTask(lineText(result.before, line)), done: !isDoneTask(lineText(result.before, line)) });
		}),
		vscode.commands.registerCommand("vibe-code.selectPlanFile", async (uri: string) => {
			const paths = ensureWorkspacePaths();
			if (!paths) return;
			const name = path.basename(vscode.Uri.parse(uri).fsPath);
			if (selectPlanName(paths) === name) {
				void vscode.window.showInformationMessage("이미 현재 계획입니다: " + name);
				return;
			}
			setActivePlanName(paths, name);
			void vscode.window.showInformationMessage("현재 계획을 선택했습니다: " + name);
			writeAudit(host, "plan", "setActivePlan", { planFile: ".vibe-code/plans/" + name, via: "codelens" });
		}),
	);
	log(host, "plan CodeLens registered");
}
