import * as fs from "node:fs";
import * as path from "node:path";
import * as vscode from "vscode";
import type { CoreHost } from "../core/host";
import { log } from "../log";
import { kstStamp } from "../util/kst";
import { countChecks, headingTitle, lines, matchLine, progressLabel, section, stripTask, subsection, taskLines, touchPlan } from "../util/markdown";
import { checklistItems, commandItem, makeItem, noteItems, registerTreeView, TreeNode } from "./tree";
import { ensureWorkspacePaths, openFile, readUtf8, writeAudit } from "./workspace";
import { currentHealth } from "./goal-health";

export const GOAL_FILE = ".vibe-code/goals/current.md";
const NO_GOAL_MESSAGE = "현재 목표 파일이 없습니다. 먼저 /goal을 시작하거나 현재 목표 열기 명령을 실행하세요.";

export function goalTemplate(stamp: string): string {
	return `# 목표: 새 목표

상태: draft
작성일: ${stamp}
마지막 갱신: ${stamp}
목표 파일: ${GOAL_FILE}

## 목표
/goal <원하는 결과>로 목표를 지정하거나 이 파일을 편집하세요.

## 완료 기준
- [ ] 목표를 검증 가능한 기준으로 정리
- [ ] 구현/문서/검증 범위 결정
- [ ] 최종 검증 통과

## 현재 상태
- 지금 단계: discovery
- 다음 행동: 목표 문장을 구체화
- 마지막 검증: 없음

## 작업 큐
### Now
- [ ] 목표 설명 작성

### Next
- [ ] 첫 번째 작은 작업 선택

### Done

## 결정 로그
- ${stamp} - 현재 목표 draft 생성

## 변경 로그

## 검증 로그

## 중단/재개 노트
다음 세션은 /goal 이어서로 시작하세요.
`;
}

export function handoffTemplate(stamp: string, currentGoal: string): string {
	return `# 목표 핸드오프 - ${stamp}

목표 파일: ${GOAL_FILE}
생성일: ${stamp}

## 다음 세션 시작 문장
/goal 이어서

## 바로 할 일
- current.md의 현재 상태와 작업 큐 Now 항목부터 확인
- 마지막 검증 로그 이후 변경된 파일 확인
- 가능한 가장 작은 다음 작업 하나를 구현하고 검증

## 현재 목표 스냅샷

~~~markdown
${currentGoal}
~~~
`;
}

export interface GoalSummary {
	title: string;
	status: string;
	phase: string;
	next: string;
	done: number;
	total: number;
	criteria: string[];
	now: string[];
	nextItems: string[];
	verification: string[];
}

export function parseGoal(text: string): GoalSummary {
	const checks = countChecks(text);
	return {
		title: headingTitle(text, "목표", "현재 목표"),
		status: matchLine(text, "상태", "(unknown)"),
		phase: matchLine(text, "- 지금 단계", "(unknown)"),
		next: matchLine(text, "- 다음 행동", "(unknown)"),
		done: checks.done,
		total: checks.total,
		criteria: taskLines(section(text, "완료 기준")),
		now: taskLines(subsection(text, "Now")),
		nextItems: taskLines(subsection(text, "Next")),
		verification: lines(section(text, "검증 로그")).slice(-5),
	};
}

/** Title of the current goal file, or the fallback when it is missing/unreadable. */
export function currentGoalTitle(currentFile: string, fallback = "현재 목표"): string {
	try {
		if (fs.existsSync(currentFile)) return headingTitle(readUtf8(currentFile), "목표", fallback);
	} catch {
		// unreadable goal file: use fallback
	}
	return fallback;
}

/** Write a handoff file for the current goal and note it in the goal file. Returns the workspace-relative path. */
export function createHandoffFile(host: CoreHost, paths: { current: string; sessions: string }, note = ""): string {
	const stamp = kstStamp();
	const handoff = path.join(paths.sessions, `${stamp.file}-handoff.md`);
	fs.writeFileSync(handoff, handoffTemplate(stamp.human, readUtf8(paths.current)), "utf8");
	fs.appendFileSync(paths.current, `\n- ${stamp.human} - handoff 생성: .vibe-code/sessions/${path.basename(handoff)}${note ? ` (${note})` : ""}\n`, "utf8");
	const rel = `.vibe-code/sessions/${path.basename(handoff)}`;
	log(host, `goal handoff created: ${handoff}`);
	return rel;
}

const GOAL_STATUSES = ["draft", "active", "blocked", "done"];

/** Unchecked 완료 기준 items — the gate for `상태: done`. */
export function openCriteria(text: string): string[] {
	return taskLines(section(text, "완료 기준")).filter((l) => !/^- \[[xX]\]/.test(l)).map(stripTask);
}

export function registerGoalCommands(host: CoreHost): void {
	const { context, output } = host;
	context.subscriptions.push(
		vscode.commands.registerCommand("vibe-code.setGoalStatus", async (statusArg?: unknown) => {
			const paths = ensureWorkspacePaths();
			if (!paths) return;
			if (!fs.existsSync(paths.current)) {
				void vscode.window.showWarningMessage(NO_GOAL_MESSAGE);
				return;
			}
			const text = readUtf8(paths.current);
			const explicit = typeof statusArg === "string" && GOAL_STATUSES.includes(statusArg);
			const status = explicit ? (statusArg as string) : await vscode.window.showQuickPick(GOAL_STATUSES, { placeHolder: `목표 상태 선택 (현재: ${matchLine(text, "상태", "?")})` });
			if (!status) return;
			if (status === "done" && !explicit) {
				const open = openCriteria(text);
				if (open.length > 0) {
					const go = await vscode.window.showWarningMessage(`완료 기준 ${open.length}개가 아직 체크되지 않았습니다:\n${open.slice(0, 5).join("\n")}${open.length > 5 ? "\n…" : ""}`, { modal: true }, "그래도 done으로 변경");
					if (go !== "그래도 done으로 변경") return;
					writeAudit(host, "goal", "completionGateOverridden", { open: open.length });
				}
			}
			fs.writeFileSync(paths.current, touchPlan(text, status, kstStamp().human), "utf8");
			void vscode.window.showInformationMessage("목표 상태를 변경했습니다: " + status);
			writeAudit(host, "goal", "setGoalStatus", { status, openCriteria: openCriteria(text).length });
		}),
		vscode.commands.registerCommand("vibe-code.openCurrentGoal", async () => {
			const paths = ensureWorkspacePaths();
			if (!paths) return;
			if (!fs.existsSync(paths.current)) {
				fs.writeFileSync(paths.current, goalTemplate(kstStamp().human), "utf8");
				log(host, `current goal draft created: ${paths.current}`);
			}
			await openFile(paths.current);
			void vscode.window.showInformationMessage("현재 목표 파일을 열었습니다.");
			writeAudit(host, "goal", "openCurrentGoal", { currentExists: true, goalFile: GOAL_FILE });
		}),
		vscode.commands.registerCommand("vibe-code.showGoalStatus", async () => {
			const paths = ensureWorkspacePaths();
			if (!paths) return;
			if (!fs.existsSync(paths.current)) {
				void vscode.window.showWarningMessage(NO_GOAL_MESSAGE);
				return;
			}
			const text = readUtf8(paths.current);
			const goal = parseGoal(text);
			const progress = progressLabel(goal, "체크리스트 없음");
			output.show(true);
			output.appendLine("\n=== Vibe Code 목표 상태 ===");
			output.appendLine("checklist progress: local goal file");
			output.appendLine(`목표: ${goal.title}`);
			output.appendLine(`상태: ${goal.status}`);
			output.appendLine(`진행률: ${progress}`);
			output.appendLine(`현재 단계: ${goal.phase}`);
			output.appendLine(`다음 행동: ${goal.next}`);
			output.appendLine("\n--- Now ---");
			output.appendLine(subsection(text, "Now") || "(비어 있음)");
			output.appendLine("\n--- Next ---");
			output.appendLine(subsection(text, "Next") || "(비어 있음)");
			output.appendLine("\n--- 최근 검증 로그 ---");
			if (goal.verification.length === 0) output.appendLine("(검증 로그 없음)");
			for (const line of goal.verification) output.appendLine(line);
			output.appendLine("================================\n");
			void vscode.window.showInformationMessage(`목표 상태: ${goal.title} (${progress})`);
			writeAudit(host, "goal", "showGoalStatus", {
				title: goal.title,
				status: goal.status,
				progress: goal.total > 0 ? goal.done + "/" + goal.total : "checklist-none",
				phase: goal.phase,
				next: goal.next,
			});
		}),
		vscode.commands.registerCommand("vibe-code.createGoalHandoff", async () => {
			const paths = ensureWorkspacePaths();
			if (!paths) return;
			if (!fs.existsSync(paths.current)) {
				void vscode.window.showWarningMessage(NO_GOAL_MESSAGE);
				return;
			}
			const rel = createHandoffFile(host, paths);
			await openFile(path.join(paths.ws, rel));
			void vscode.window.showInformationMessage("목표 핸드오프 파일을 생성했습니다.");
			writeAudit(host, "goal", "createGoalHandoff", { handoffFile: rel });
		}),
	);
}

/** Status bar item showing the current goal title and checklist progress; hidden without a goal file. */
export function createGoalStatusBar(host: CoreHost): void {
	const item = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 99);
	item.command = "vibe-code.openCurrentGoal";
	const refresh = () => {
		try {
			const ws = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
			const file = ws ? path.join(ws, ".vibe-code", "goals", "current.md") : undefined;
			if (!file || !fs.existsSync(file)) {
				item.hide();
				return;
			}
			const text = readUtf8(file);
			const title = headingTitle(text, "목표", "현재 목표");
			const status = matchLine(text, "상태", "active");
			const checks = countChecks(text);
			const short = title.length > 20 ? title.slice(0, 19) + "…" : title;
			const health = currentHealth(host, text);
			item.text = `${health.warning ? "$(warning)" : "$(target)"} 목표: ${short}${checks.total > 0 ? ` ${checks.done}/${checks.total}` : ""}`;
			item.backgroundColor = health.warning ? new vscode.ThemeColor("statusBarItem.warningBackground") : undefined;
			item.tooltip = `Vibe Code 현재 목표\n상태: ${status}\n진행률: ${progressLabel(checks, "체크리스트 없음")}\n파일: ${GOAL_FILE}${health.warning ? `\n⚠ ${health.warning}` : ""}\n클릭하여 목표 파일 열기`;
			item.show();
		} catch {
			item.hide();
		}
	};
	refresh();
	const interval = setInterval(refresh, 10_000);
	host.context.subscriptions.push(item, { dispose: () => clearInterval(interval) });
	log(host, "goal status bar item created");
}

const OPEN_GOAL: vscode.Command = { command: "vibe-code.openCurrentGoal", title: "현재 목표 열기" };

function goalTrackerRoots(): TreeNode[] {
	const paths = ensureWorkspacePaths(true);
	if (!paths || !fs.existsSync(paths.current)) {
		return [
			makeItem("현재 목표 없음", {
				description: "/goal로 시작",
				tooltip: "/goal 명령으로 목표를 시작하거나 현재 목표 파일을 여세요.",
				command: OPEN_GOAL,
				icon: "target",
			}),
		];
	}
	const goal = parseGoal(readUtf8(paths.current));
	const progress = progressLabel(goal, "체크 없음");
	const expanded = vscode.TreeItemCollapsibleState.Expanded;
	const collapsed = vscode.TreeItemCollapsibleState.Collapsed;
	return [
		makeItem(goal.title, {
			description: `${progress} · ${goal.status}`,
			tooltip: `상태: ${goal.status}\n현재 단계: ${goal.phase}\n다음 행동: ${goal.next}`,
			command: OPEN_GOAL,
			icon: "target",
		}),
		makeItem("완료 기준", { description: progress, tooltip: "완료 기준 체크리스트", collapsibleState: expanded, icon: "checklist", children: checklistItems(goal.criteria, "완료 기준 없음", "check") }),
		makeItem("Now", { description: goal.now.length > 0 ? `${goal.now.length}개` : "비어 있음", tooltip: "바로 할 일", collapsibleState: expanded, icon: "play", children: checklistItems(goal.now, "Now 항목 없음", "circle-outline") }),
		makeItem("Next", { description: goal.nextItems.length > 0 ? `${goal.nextItems.length}개` : "비어 있음", tooltip: "다음 작업", collapsibleState: collapsed, icon: "arrow-right", children: checklistItems(goal.nextItems, "Next 항목 없음", "circle-large-outline") }),
		makeItem("최근 검증", { description: goal.verification.length > 0 ? `${goal.verification.length}줄` : "없음", tooltip: "최근 검증 로그", collapsibleState: collapsed, icon: "beaker", children: noteItems(goal.verification, "검증 로그 없음") }),
		makeItem("바로 실행", {
			description: "명령 5개",
			tooltip: "현재 목표 관련 명령",
			collapsibleState: collapsed,
			icon: "tools",
			children: [
				commandItem("현재 목표 열기", "vibe-code.openCurrentGoal", "현재 목표 열기", "go-to-file"),
				commandItem("목표 상태 보기", "vibe-code.showGoalStatus", "목표 상태 보기", "list-tree"),
				commandItem("목표 핸드오프 생성", "vibe-code.createGoalHandoff", "목표 핸드오프 생성", "export"),
				commandItem("목표 상태 변경", "vibe-code.setGoalStatus", "목표 상태 변경", "symbol-enum"),
				commandItem("목표 이어서 진행", "vibe-code.resumeGoal", "목표 이어서 진행", "debug-continue"),
			],
		}),
	];
}

export function registerGoalTrackerView(host: CoreHost): void {
	registerTreeView(host, { viewId: "vibe-code.GoalTracker", roots: goalTrackerRoots, watch: [GOAL_FILE] });
	log(host, "goal tracker view registered");
}
