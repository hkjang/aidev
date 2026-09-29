import * as fs from "node:fs";
import * as path from "node:path";
import * as vscode from "vscode";
import type { CoreHost } from "../core/host";
import { log } from "../log";
import { kstStamp } from "../util/kst";
import { headingTitle, matchLine, section, setLine, taskLines } from "../util/markdown";
import { GOAL_FILE, goalTemplate } from "./goals";
import { commandItem, makeItem, registerTreeView, TreeNode } from "./tree";
import { ensureWorkspacePaths, listMarkdown, openFile, readUtf8, writeAudit, WorkspacePaths } from "./workspace";

// --- presets ------------------------------------------------------------------------------------

export interface GoalPreset {
	key: string;
	label: string;
	detail: string;
	criteria: string[];
}

export const GOAL_PRESETS: GoalPreset[] = [
	{ key: "feature", label: "기능 개발", detail: "새 기능을 동작·테스트·문서까지 마무리", criteria: ["핵심 기능이 실제로 동작함 (수동 확인)", "자동 테스트 추가 및 통과", "문서 또는 사용법 반영", "검증 로그에 실행 결과 기록"] },
	{ key: "bugfix", label: "버그 수정", detail: "재현 → 원인 → 회귀 테스트 → 수정 확인", criteria: ["재현 절차와 원인 기록", "회귀 테스트 추가", "수정 후 재현 절차로 재확인", "변경 로그/관련 문서 갱신"] },
	{ key: "refactor", label: "리팩터링", detail: "동작은 그대로, 구조만 개선", criteria: ["기존 테스트 전부 통과 (동작 변경 없음)", "구조 개선 기준을 명시하고 달성", "성능/가독성 지표 전후 기록", "문서 갱신"] },
	{ key: "docs", label: "문서화", detail: "대상 독자 기준으로 문서 작성·검토", criteria: ["대상 독자와 범위 결정", "초안 작성", "검토 의견 반영", "링크/색인 갱신"] },
	{ key: "free", label: "자유 형식", detail: "기본 템플릿 완료 기준", criteria: ["목표를 검증 가능한 기준으로 정리", "구현/문서/검증 범위 결정", "최종 검증 통과"] },
];

/** kebab-case-ish slug that keeps Korean characters; used in goal file names. */
export function slugify(title: string): string {
	return (
		title
			.trim()
			.toLowerCase()
			.replace(/[^\p{L}\p{N}]+/gu, "-")
			.replace(/^-+|-+$/g, "")
			.slice(0, 40)
			.replace(/-+$/g, "") || "goal"
	);
}

export interface NewGoalOptions {
	title: string;
	description: string;
	criteria: string[];
	fileName: string;
	stamp: string;
}

/** Goal file text for a new goal: the standard template with title, description and preset criteria. */
export function buildGoalFile(opts: NewGoalOptions): string {
	let text = goalTemplate(opts.stamp)
		.replace("# 목표: 새 목표", `# 목표: ${opts.title}`)
		.replace("상태: draft", "상태: active")
		.replace(`목표 파일: ${GOAL_FILE}`, `목표 파일: .vibe-code/goals/${opts.fileName}`)
		.replace("/goal <원하는 결과>로 목표를 지정하거나 이 파일을 편집하세요.", opts.description || opts.title)
		.replace("- 다음 행동: 목표 문장을 구체화", "- 다음 행동: 첫 Now 항목 착수")
		.replace("- [ ] 목표 설명 작성", `- [ ] ${opts.title}: 코드 구조 파악과 첫 구현 단위 정의`)
		.replace("- 현재 목표 draft 생성", `- 목표 생성 (${opts.title})`);
	const criteria = opts.criteria.map((c) => `- [ ] ${c}`);
	text = text.replace(/## 완료 기준\n([\s\S]*?)(?=\n## )/, `## 완료 기준\n${criteria.join("\n")}\n`);
	return text;
}

// --- catalog ------------------------------------------------------------------------------------

export interface GoalEntry {
	name: string;
	title: string;
	status: string;
	updated: string;
	isCurrent: boolean;
}

/** Own file name of the goal loaded in current.md (from its `목표 파일:` line), if any. */
export function currentGoalFileName(currentText: string): string | undefined {
	const line = matchLine(currentText, "목표 파일");
	const name = line ? path.posix.basename(line.replace(/\\/g, "/")) : "";
	return name && name !== "current.md" ? name : undefined;
}

export function listGoals(paths: WorkspacePaths): GoalEntry[] {
	const currentName = fs.existsSync(paths.current) ? currentGoalFileName(readUtf8(paths.current)) : undefined;
	return listMarkdown(paths.goals)
		.filter((name) => name !== "current.md")
		.map((name) => {
			const text = readUtf8(path.join(paths.goals, name));
			return { name, title: headingTitle(text, "목표", name), status: matchLine(text, "상태", "?"), updated: matchLine(text, "마지막 갱신") || matchLine(text, "작성일"), isCurrent: name === currentName };
		})
		.sort((a, b) => (a.isCurrent === b.isCurrent ? b.updated.localeCompare(a.updated) : a.isCurrent ? -1 : 1));
}

/** Persist current.md into its own goal file so switching never loses edits. Returns the file name. */
export function saveCurrentGoal(paths: WorkspacePaths): string | undefined {
	if (!fs.existsSync(paths.current)) return undefined;
	const text = readUtf8(paths.current);
	let name = currentGoalFileName(text);
	let body = text;
	if (!name) {
		name = `${kstStamp().file}-${slugify(headingTitle(text, "목표", "goal"))}.md`;
		body = setLine(text, "목표 파일", `.vibe-code/goals/${name}`);
		fs.writeFileSync(paths.current, body, "utf8");
	}
	fs.writeFileSync(path.join(paths.goals, name), body, "utf8");
	return name;
}

export function switchToGoal(paths: WorkspacePaths, name: string): void {
	const previous = saveCurrentGoal(paths);
	if (previous === name) return;
	fs.copyFileSync(path.join(paths.goals, name), paths.current);
}

// --- plan -> criteria ----------------------------------------------------------------------------

/** Tick 완료 기준 items by 1-based index. Pure. */
export function tickCriteria(goalText: string, indexes: number[]): string {
	const body = section(goalText, "완료 기준");
	if (!body) return goalText;
	const all = goalText.split(/\r?\n/);
	let seen = 0;
	for (let i = 0; i < all.length; i++) {
		if (/^## 완료 기준/.test(all[i])) {
			for (let j = i + 1; j < all.length && !/^## /.test(all[j]); j++) {
				if (/^- \[( |x|X)\] /.test(all[j].trim())) {
					seen++;
					if (indexes.includes(seen)) all[j] = all[j].replace(/^(\s*)- \[ \] /, "$1- [x] ");
				}
			}
			break;
		}
	}
	return all.join("\n");
}

export function parseCriteriaLink(planText: string): number[] {
	return matchLine(planText, "연결 완료 기준")
		.split(/[,\s]+/)
		.map((s) => Number.parseInt(s, 10))
		.filter((n) => Number.isInteger(n) && n > 0);
}

/** When a plan reaches done, tick the goal criteria it was linked to. Returns the ticked indexes. */
export function applyPlanCriteria(paths: WorkspacePaths, planText: string): number[] {
	const indexes = parseCriteriaLink(planText);
	if (indexes.length === 0 || !fs.existsSync(paths.current)) return [];
	fs.writeFileSync(paths.current, tickCriteria(readUtf8(paths.current), indexes), "utf8");
	return indexes;
}

// --- commands + view -----------------------------------------------------------------------------

function goalListRoots(): TreeNode[] {
	const paths = ensureWorkspacePaths(true);
	if (!paths) return [makeItem("워크스페이스 없음", { icon: "circle-slash" })];
	const goals = listGoals(paths);
	const items = goals.map((g) =>
		makeItem(g.title, {
			description: `${g.isCurrent ? "현재 · " : ""}${g.status}${g.updated ? ` · ${g.updated.slice(0, 16)}` : ""}`,
			tooltip: `.vibe-code/goals/${g.name}\n상태: ${g.status}\n마지막 갱신: ${g.updated || "?"}\n클릭: 현재 목표로 전환`,
			command: { command: "vibe-code.switchGoal", title: "목표 전환", arguments: [g.name] },
			icon: g.isCurrent ? "target" : g.status === "done" ? "pass" : "circle-large-outline",
		}),
	);
	return [commandItem("새 목표 만들기", "vibe-code.newGoal", "새 목표", "add"), ...(items.length ? items : [makeItem("저장된 목표 없음", { description: "현재 목표는 전환 시 자동 저장됩니다", icon: "circle-slash" })])];
}

export function registerGoalCatalog(host: CoreHost): void {
	const { context } = host;
	context.subscriptions.push(
		vscode.commands.registerCommand("vibe-code.switchGoal", async (nameArg?: unknown) => {
			const paths = ensureWorkspacePaths();
			if (!paths) return;
			const goals = listGoals(paths).filter((g) => !g.isCurrent);
			if (goals.length === 0) {
				void vscode.window.showInformationMessage("전환할 다른 목표가 없습니다. `새 목표 만들기`로 추가하세요.");
				return;
			}
			const name =
				typeof nameArg === "string" && goals.some((g) => g.name === nameArg)
					? nameArg
					: (await vscode.window.showQuickPick(goals.map((g) => ({ label: g.title, description: `${g.status} · ${g.updated}`, detail: g.name, name: g.name })), { placeHolder: "현재 목표로 전환할 목표" }))?.name;
			if (!name) return;
			switchToGoal(paths, name);
			await openFile(paths.current);
			void vscode.window.showInformationMessage(`현재 목표를 전환했습니다: ${name}`);
			writeAudit(host, "goal", "switchGoal", { goalFile: `.vibe-code/goals/${name}` });
		}),
		vscode.commands.registerCommand("vibe-code.newGoal", async () => {
			const paths = ensureWorkspacePaths();
			if (!paths) return;
			const title = (await vscode.window.showInputBox({ prompt: "목표 한 줄 (예: 결제 모듈에 환불 API 추가)", ignoreFocusOut: true }))?.trim();
			if (!title) return;
			const preset = await vscode.window.showQuickPick(GOAL_PRESETS.map((p) => ({ label: p.label, detail: p.detail, preset: p })), { placeHolder: "목표 유형 (완료 기준 프리셋)", ignoreFocusOut: true });
			if (!preset) return;
			const description = (await vscode.window.showInputBox({ prompt: "목표 설명 (선택, 기대 결과·제약)", ignoreFocusOut: true })) ?? "";
			const stamp = kstStamp();
			const fileName = `${stamp.file}-${slugify(title)}.md`;
			saveCurrentGoal(paths);
			const body = buildGoalFile({ title, description, criteria: preset.preset.criteria, fileName, stamp: stamp.human });
			fs.writeFileSync(path.join(paths.goals, fileName), body, "utf8");
			fs.writeFileSync(paths.current, body, "utf8");
			await openFile(paths.current);
			writeAudit(host, "goal", "newGoal", { goalFile: `.vibe-code/goals/${fileName}`, preset: preset.preset.key });
			log(host, `new goal created: ${fileName}`);
			const start = await vscode.window.showInformationMessage(`새 목표를 만들었습니다: ${title}`, "바로 시작 (/goal)", "나중에");
			if (start === "바로 시작 (/goal)") {
				const provider = host.getProvider() as { initClineWithTask?: (task: string) => Promise<unknown> } | undefined;
				if (provider?.initClineWithTask) {
					await vscode.commands.executeCommand("vibe-code.SidebarProvider.focus");
					await provider.initClineWithTask(`/goal ${title}`);
				}
			}
		}),
		vscode.commands.registerCommand("vibe-code.linkPlanToCriteria", async () => {
			const paths = ensureWorkspacePaths();
			if (!paths || !fs.existsSync(paths.current)) {
				void vscode.window.showWarningMessage("현재 목표 파일이 없습니다.");
				return;
			}
			const { selectPlanName } = await import("./plans");
			const planFile = path.join(paths.plans, selectPlanName(paths));
			if (!fs.existsSync(planFile)) {
				void vscode.window.showWarningMessage("현재 계획 파일이 없습니다. 먼저 최신 계획 열기 명령을 실행하세요.");
				return;
			}
			const criteria = taskLines(section(readUtf8(paths.current), "완료 기준"));
			if (criteria.length === 0) {
				void vscode.window.showWarningMessage("목표에 완료 기준 항목이 없습니다.");
				return;
			}
			const planText = readUtf8(planFile);
			const linked = parseCriteriaLink(planText);
			const picked = await vscode.window.showQuickPick(
				criteria.map((c, i) => ({ label: `${i + 1}. ${c.replace(/^- \[( |x|X)\] /, "")}`, picked: linked.includes(i + 1), index: i + 1 })),
				{ canPickMany: true, placeHolder: "이 계획이 완료되면 체크할 완료 기준" },
			);
			if (!picked) return;
			const indexes = picked.map((p) => p.index);
			fs.writeFileSync(planFile, setLine(planText, "연결 완료 기준", indexes.join(", ") || "-"), "utf8");
			void vscode.window.showInformationMessage(`계획을 완료 기준 ${indexes.join(", ") || "(없음)"}과 연결했습니다.`);
			writeAudit(host, "plan", "linkPlanToCriteria", { planFile: `.vibe-code/plans/${path.basename(planFile)}`, criteria: indexes });
		}),
	);
	registerTreeView(host, { viewId: "vibe-code.GoalList", roots: goalListRoots, watch: [".vibe-code/goals/*.md"] });
	log(host, "goal catalog registered (list view, switch, new goal, criteria link)");
}
