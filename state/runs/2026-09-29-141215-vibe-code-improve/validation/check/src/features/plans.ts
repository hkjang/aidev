import * as fs from "node:fs";
import * as path from "node:path";
import * as vscode from "vscode";
import type { CoreHost } from "../core/host";
import { log } from "../log";
import { kstStamp } from "../util/kst";
import { countChecks, headingTitle, lines, matchLine, priorityRank, progressLabel, section, sectionLines, setLine, stripTask, taskLines, touchPlan, writeSection } from "../util/markdown";
import { currentGoalTitle, GOAL_FILE } from "./goals";
import { applyPlanCriteria } from "./goal-catalog";
import { checklistItems, commandItem, makeItem, noteItems, registerTreeView, TreeNode } from "./tree";
import { ensureWorkspacePaths, listMarkdown, openFile, readUtf8, writeAudit, WorkspacePaths } from "./workspace";

const NO_PLAN_MESSAGE = "현재 계획 파일이 없습니다. 먼저 최신 계획 열기 명령을 실행하세요.";
const NO_PLAN_FILES_MESSAGE = "선택할 계획 파일이 없습니다. 먼저 최신 계획 열기 명령을 실행하세요.";
const PLAN_STATUSES = ["draft", "active", "blocked", "done"];
const PLAN_PRIORITIES = ["P0", "P1", "P2", "P3"];

export function planTemplate(goalTitle: string, fileName: string, stamp: string): string {
	return `# 계획: ${goalTitle}

상태: draft
작성일: ${stamp}
마지막 갱신: ${stamp}
연결 목표: ${GOAL_FILE}
계획 파일: .vibe-code/plans/${fileName}

## 목표 정렬
- 목표 제목: ${goalTitle}
- 이번 계획의 초점: 가장 작은 다음 구현 단위 정의

## 단계
- [ ] 요구사항/제약 다시 확인
- [ ] 가장 작은 구현 단계 선택
- [ ] 검증 기준 확정

## Now
- [ ] 바로 구현할 작업 1개 정의

## Next
- [ ] 뒤이어 실행할 작업 1개 정의

## Done

## Risks
- 없음

## 검증 계획
- [ ] 관련 스크립트 또는 테스트 실행
`;
}

// --- active plan pointer -------------------------------------------------------------------

function activePointerFile(paths: WorkspacePaths): string {
	return path.join(paths.plans, ".active-plan");
}

/** The plan file name the commands operate on: the `.active-plan` pointer, else the last file, else `current-plan.md`. */
export function selectPlanName(paths: WorkspacePaths): string {
	fs.mkdirSync(paths.plans, { recursive: true });
	const files = listMarkdown(paths.plans);
	let active = "";
	try {
		const pointer = activePointerFile(paths);
		if (fs.existsSync(pointer)) active = readUtf8(pointer).trim();
	} catch {
		// unreadable pointer: fall through to the newest file
	}
	if (active && files.includes(active)) return active;
	return files.length > 0 ? files[files.length - 1] : "current-plan.md";
}

export function setActivePlanName(paths: WorkspacePaths, name: string): void {
	fs.mkdirSync(paths.plans, { recursive: true });
	fs.writeFileSync(activePointerFile(paths), name, "utf8");
}

interface PlanRef {
	paths: WorkspacePaths;
	file: string;
	name: string;
}

interface LoadedPlan extends PlanRef {
	text: string;
}

function latestPlan(): PlanRef | null {
	const paths = ensureWorkspacePaths(true);
	if (!paths) return null;
	const name = selectPlanName(paths);
	return { paths, file: path.join(paths.plans, name), name };
}

function readPlan(): LoadedPlan | null {
	const ref = latestPlan();
	if (!ref || !fs.existsSync(ref.file)) return null;
	return { ...ref, text: readUtf8(ref.file) };
}

// --- plan metadata ---------------------------------------------------------------------------

export interface PlanMeta {
	name: string;
	title: string;
	status: string;
	priority: string;
}

export function planMeta(text: string, name: string): PlanMeta {
	return {
		name,
		title: headingTitle(text, "계획", name),
		status: matchLine(text, "상태", "draft"),
		priority: matchLine(text, "우선순위", "P2"),
	};
}

export function sortByPriority<T extends { name: string; priority: string }>(rows: T[]): T[] {
	return rows.sort((a, b) => priorityRank(a.priority) - priorityRank(b.priority) || a.name.localeCompare(b.name));
}

function planMetaList(dir: string): PlanMeta[] {
	return sortByPriority(listMarkdown(dir).map((name) => planMeta(readUtf8(path.join(dir, name)), name)));
}

export interface LinkedPlan extends PlanMeta {
	bucket: "active" | "archive";
	linked: string;
	linkedTitle: string;
}

/** Plans in `dir` that carry a `연결 목표:` line, sorted by priority. */
export function linkedPlans(dir: string, bucket: LinkedPlan["bucket"]): LinkedPlan[] {
	const rows: LinkedPlan[] = [];
	for (const name of listMarkdown(dir)) {
		const text = readUtf8(path.join(dir, name));
		const linked = matchLine(text, "연결 목표");
		if (!linked) continue;
		rows.push({ ...planMeta(text, name), bucket, linked, linkedTitle: matchLine(text, "연결 목표 제목") });
	}
	return sortByPriority(rows);
}

async function pickPlanFile(files: string[], arg: unknown, placeHolder: string): Promise<string | undefined> {
	if (typeof arg === "string" && files.includes(arg)) return arg;
	if (files.length === 1) return files[0];
	return vscode.window.showQuickPick(files, { placeHolder });
}

// --- archiving -------------------------------------------------------------------------------

export interface ArchivedPlan {
	/** The name the plan had in `plans/`. */
	name: string;
	/** The name it was written under in `archive/` — differs from `name` when it had to avoid a collision. */
	dest: string;
}

/** `YYYYMMDDHHMMSS` in Asia/Seoul — the suffix both archiving and restoring use to dodge a name collision. */
export function conflictStamp(date: Date = new Date()): string {
	return kstStamp(date).file.replace(/[^0-9]/g, "");
}

/** A name that is free in `dir`: `<이름>.md`, else `<이름>-<stamp>.md`, else `<이름>-<stamp>-<n>.md`. */
export function freeName(dir: string, name: string, stamp: string): string {
	if (!fs.existsSync(path.join(dir, name))) return name;
	const parsed = path.parse(name);
	const ext = parsed.ext || ".md";
	let candidate = `${parsed.name}-${stamp}${ext}`;
	for (let n = 2; fs.existsSync(path.join(dir, candidate)); n += 1) candidate = `${parsed.name}-${stamp}-${n}${ext}`;
	return candidate;
}

/**
 * Move every `상태: done` plan from `plansDir` into `archiveDir`.
 * An archived plan of the same name is never overwritten — the incoming file takes a stamped name instead.
 */
export function archiveDonePlans(plansDir: string, archiveDir: string, stamp: string): ArchivedPlan[] {
	fs.mkdirSync(archiveDir, { recursive: true });
	const archived: ArchivedPlan[] = [];
	for (const name of listMarkdown(plansDir)) {
		const src = path.join(plansDir, name);
		if (!fs.statSync(src).isFile()) continue;
		if (matchLine(readUtf8(src), "상태") !== "done") continue;
		const dest = freeName(archiveDir, name, stamp);
		fs.renameSync(src, path.join(archiveDir, dest));
		archived.push({ name, dest });
	}
	return archived;
}

/** `taskLines()` 와 같은 판정 — 들여쓴 체크리스트 항목(`  - [ ] 하위`)도 항목으로 본다. */
const isChecklistLine = (line: string): boolean => line.trim().startsWith("- [");

export interface AdvancedPlan {
	text: string;
	/** The Now item that was completed, without its checklist prefix. */
	completed: string;
	/** The Next item promoted into Now, without its checklist prefix; `""` when Next had none. */
	promoted: string;
}

/**
 * One step of the plan: the first Now checklist item moves to the end of `## Done` as `- [x] …`,
 * the first Next checklist item is promoted into Now, and 상태/마지막 갱신 are bumped.
 * Returns null — and nothing is changed — when Now has no checklist line.
 */
export function advancePlanText(text: string, stamp: string): AdvancedPlan | null {
	// 섹션을 원본 줄 그대로 읽고 항목만 splice 한다 — 걸러 읽고 통째로 덮어쓰면
	// 사용자가 Now/Next 에 적어 둔 메모 줄과 들여쓴 하위 항목이 사라진다.
	const now = sectionLines(text, "Now");
	const nowIndex = now.findIndex(isChecklistLine);
	if (nowIndex < 0) return null;
	const next = sectionLines(text, "Next");
	const done = sectionLines(text, "Done");
	const completed = stripTask((now.splice(nowIndex, 1)[0] as string).trim());
	done.push("- [x] " + completed);
	let promoted = "";
	const nextIndex = next.findIndex(isChecklistLine);
	if (nextIndex >= 0) {
		const line = (next.splice(nextIndex, 1)[0] as string).trim();
		promoted = stripTask(line);
		now.push(line);
	}
	let out = text;
	out = writeSection(out, "Now", now);
	out = writeSection(out, "Next", next);
	out = writeSection(out, "Done", done);
	out = touchPlan(out, "active", stamp);
	return { text: out, completed, promoted };
}

/** Unchecked tasks in the sections that must be finished before a plan is `done`. */
export function openPlanItems(text: string): string[] {
	return ["단계", "Now", "검증 계획"].flatMap((name) => taskLines(section(text, name)).filter((l) => !/^- \[[xX]\]/.test(l)).map((l) => `${name}: ${stripTask(l)}`));
}

// --- commands --------------------------------------------------------------------------------

export function registerPlanCommands(host: CoreHost): void {
	const { context, output } = host;
	const planFile = (name: string) => ".vibe-code/plans/" + name;

	context.subscriptions.push(
		vscode.commands.registerCommand("vibe-code.openLatestPlan", async () => {
			const paths = ensureWorkspacePaths();
			if (!paths) return;
			const name = selectPlanName(paths);
			const target = path.join(paths.plans, name);
			const created = !fs.existsSync(target);
			if (created) {
				fs.writeFileSync(target, planTemplate(currentGoalTitle(paths.current), name, kstStamp().human), "utf8");
				log(host, `current plan draft created: ${target}`);
			}
			setActivePlanName(paths, name);
			await openFile(target);
			void vscode.window.showInformationMessage("최신 계획 파일을 열었습니다.");
			writeAudit(host, "plan", "openLatestPlan", { planFile: planFile(name), created });
		}),
		vscode.commands.registerCommand("vibe-code.advanceCurrentPlan", async () => {
			const plan = readPlan();
			if (!plan) {
				void vscode.window.showWarningMessage(NO_PLAN_MESSAGE);
				return;
			}
			const advanced = advancePlanText(plan.text, kstStamp().human);
			if (!advanced) {
				void vscode.window.showWarningMessage("진행할 Now 항목이 없습니다.");
				return;
			}
			fs.writeFileSync(plan.file, advanced.text, "utf8");
			void vscode.window.showInformationMessage("현재 계획을 한 단계 진행했습니다.");
			writeAudit(host, "plan", "advanceCurrentPlan", { planFile: planFile(plan.name), completed: advanced.completed, promoted: advanced.promoted });
		}),
		vscode.commands.registerCommand("vibe-code.setCurrentPlanStatus", async (statusArg?: unknown) => {
			const plan = readPlan();
			if (!plan) {
				void vscode.window.showWarningMessage(NO_PLAN_MESSAGE);
				return;
			}
			const explicit = typeof statusArg === "string" && PLAN_STATUSES.includes(statusArg);
			const status = explicit ? (statusArg as string) : await vscode.window.showQuickPick(PLAN_STATUSES, { placeHolder: "현재 계획 상태 선택" });
			if (!status) return;
			if (status === "done" && !explicit) {
				const open = openPlanItems(plan.text);
				if (open.length > 0) {
					const go = await vscode.window.showWarningMessage(`미완료 항목이 ${open.length}개 있습니다:\n${open.slice(0, 5).join("\n")}${open.length > 5 ? "\n…" : ""}`, { modal: true }, "그래도 done으로 변경");
					if (go !== "그래도 done으로 변경") return;
					writeAudit(host, "plan", "completionGateOverridden", { planFile: planFile(plan.name), open: open.length });
				}
			}
			fs.writeFileSync(plan.file, touchPlan(plan.text, status, kstStamp().human), "utf8");
			const ticked = status === "done" ? applyPlanCriteria(plan.paths, plan.text) : [];
			void vscode.window.showInformationMessage("현재 계획 상태를 변경했습니다: " + status + (ticked.length ? ` (완료 기준 ${ticked.join(", ")} 체크)` : ""));
			writeAudit(host, "plan", "setCurrentPlanStatus", { planFile: planFile(plan.name), status, tickedCriteria: ticked });
		}),
		vscode.commands.registerCommand("vibe-code.setCurrentPlanPriority", async (priorityArg?: unknown) => {
			const plan = readPlan();
			if (!plan) {
				void vscode.window.showWarningMessage(NO_PLAN_MESSAGE);
				return;
			}
			const priority =
				typeof priorityArg === "string" && PLAN_PRIORITIES.includes(priorityArg)
					? priorityArg
					: await vscode.window.showQuickPick(PLAN_PRIORITIES, { placeHolder: "현재 계획 우선순위 선택" });
			if (!priority) return;
			const text = touchPlan(setLine(plan.text, "우선순위", priority), null, kstStamp().human);
			fs.writeFileSync(plan.file, text, "utf8");
			void vscode.window.showInformationMessage("현재 계획 우선순위를 변경했습니다: " + priority);
			writeAudit(host, "plan", "setCurrentPlanPriority", { planFile: planFile(plan.name), priority });
		}),
		vscode.commands.registerCommand("vibe-code.linkCurrentPlanToGoal", async () => {
			const plan = readPlan();
			if (!plan) {
				void vscode.window.showWarningMessage(NO_PLAN_MESSAGE);
				return;
			}
			const paths = ensureWorkspacePaths();
			if (!paths || !fs.existsSync(paths.current)) {
				void vscode.window.showWarningMessage("현재 목표 파일이 없습니다. 먼저 현재 목표를 준비하세요.");
				return;
			}
			const goalTitle = currentGoalTitle(paths.current);
			let text = plan.text;
			text = setLine(text, "연결 목표", GOAL_FILE);
			text = setLine(text, "연결 목표 제목", goalTitle);
			text = touchPlan(text, null, kstStamp().human);
			fs.writeFileSync(plan.file, text, "utf8");
			void vscode.window.showInformationMessage("현재 계획을 현재 목표와 연결했습니다.");
			writeAudit(host, "plan", "linkCurrentPlanToGoal", { planFile: planFile(plan.name), goalFile: GOAL_FILE, goalTitle });
		}),
		vscode.commands.registerCommand("vibe-code.archiveDonePlans", async () => {
			const paths = ensureWorkspacePaths();
			if (!paths) return;
			const archived = archiveDonePlans(paths.plans, paths.archive, conflictStamp());
			if (archived.length === 0) {
				void vscode.window.showInformationMessage("보관할 완료 계획이 없습니다.");
				return;
			}
			const renamed = archived.filter((row) => row.dest !== row.name);
			const renamedNote = renamed.length > 0 ? ` (같은 이름의 보관본이 있어 ${renamed.length}개는 새 이름으로 보관: ${renamed.map((row) => row.dest).join(", ")})` : "";
			void vscode.window.showInformationMessage("완료 계획을 보관했습니다: " + archived.length + "개" + renamedNote);
			writeAudit(host, "plan", "archiveDonePlans", { count: archived.length, files: archived.map((row) => ".vibe-code/plans/archive/" + row.dest) });
		}),
		vscode.commands.registerCommand("vibe-code.restoreArchivedPlan", async (nameArg?: unknown) => {
			const paths = ensureWorkspacePaths();
			if (!paths) return;
			const files = listMarkdown(paths.archive);
			if (files.length === 0) {
				void vscode.window.showWarningMessage("복원할 보관 계획이 없습니다.");
				return;
			}
			const picked = await pickPlanFile(files, nameArg, "복원할 보관 계획 선택");
			if (!picked) return;
			let destName = picked;
			let dest = path.join(paths.plans, destName);
			if (fs.existsSync(dest)) {
				const stamp = conflictStamp();
				const parsed = path.parse(picked);
				destName = `${parsed.name}-restored-${stamp}${parsed.ext || ".md"}`;
				dest = path.join(paths.plans, destName);
			}
			fs.renameSync(path.join(paths.archive, picked), dest);
			setActivePlanName(paths, destName);
			await openFile(dest);
			void vscode.window.showInformationMessage("보관 계획을 복원했습니다: " + destName);
			writeAudit(host, "plan", "restoreArchivedPlan", { planFile: planFile(destName), source: ".vibe-code/plans/archive/" + picked });
		}),
		vscode.commands.registerCommand("vibe-code.setActivePlan", async (nameArg?: unknown) => {
			const paths = ensureWorkspacePaths(true);
			if (!paths) return;
			const files = listMarkdown(paths.plans);
			if (files.length === 0) {
				void vscode.window.showWarningMessage(NO_PLAN_FILES_MESSAGE);
				return;
			}
			const picked = await pickPlanFile(files, nameArg, "현재 계획으로 사용할 파일 선택");
			if (!picked) return;
			setActivePlanName(paths, picked);
			await openFile(path.join(paths.plans, picked));
			void vscode.window.showInformationMessage("현재 계획을 선택했습니다: " + picked);
			writeAudit(host, "plan", "setActivePlan", { planFile: planFile(picked) });
		}),
		vscode.commands.registerCommand("vibe-code.selectHighestPriorityPlan", async () => {
			const paths = ensureWorkspacePaths(true);
			if (!paths) return;
			const rows = planMetaList(paths.plans);
			if (rows.length === 0) {
				void vscode.window.showWarningMessage(NO_PLAN_FILES_MESSAGE);
				return;
			}
			const picked = rows[0];
			setActivePlanName(paths, picked.name);
			await openFile(path.join(paths.plans, picked.name));
			void vscode.window.showInformationMessage("최고 우선순위 계획을 현재 계획으로 선택했습니다: " + picked.name);
			writeAudit(host, "plan", "selectHighestPriorityPlan", { planFile: planFile(picked.name), priority: picked.priority, status: picked.status, candidateCount: rows.length });
		}),
		vscode.commands.registerCommand("vibe-code.showPlanCatalog", async () => {
			const paths = ensureWorkspacePaths(true);
			if (!paths) return;
			const activeName = selectPlanName(paths);
			const active = planMetaList(paths.plans);
			const archived = planMetaList(paths.archive);
			const row = (r: PlanMeta) => `${r.name} :: ${r.status} :: ${r.priority} :: ${r.title}`;
			output.show(true);
			output.appendLine("\n=== 계획 목록 ===");
			output.appendLine("active plan: " + activeName);
			output.appendLine("\n--- active plans ---");
			if (active.length === 0) output.appendLine("  (none)");
			for (const r of active) output.appendLine("  " + (r.name === activeName ? ">" : "-") + " " + row(r));
			output.appendLine("\n--- archived plans ---");
			if (archived.length === 0) output.appendLine("  (none)");
			for (const r of archived) output.appendLine("  - " + row(r));
			output.appendLine("===================\n");
			writeAudit(host, "plan", "showPlanCatalog", { activePlan: planFile(activeName), activeCount: active.length, archivedCount: archived.length });
		}),
		vscode.commands.registerCommand("vibe-code.showPlanHistory", async () => {
			const paths = ensureWorkspacePaths(true);
			if (!paths) return;
			const items: Array<{ ts: string; action: string; details?: Record<string, unknown> }> = [];
			if (fs.existsSync(paths.audit)) {
				const files = fs
					.readdirSync(paths.audit)
					.filter((name) => name.endsWith(".jsonl"))
					.sort();
				for (const name of files) {
					for (const line of readUtf8(path.join(paths.audit, name)).split(/\r?\n/).filter(Boolean)) {
						try {
							const row = JSON.parse(line);
							if (row.kind === "plan") items.push(row);
						} catch {
							// skip malformed audit lines
						}
					}
				}
			}
			const recent = items.slice(-20);
			output.show(true);
			output.appendLine("\n=== 계획 이력 ===");
			if (recent.length === 0) output.appendLine("  (no plan audit entries)");
			for (const row of recent) {
				const file = row.details?.planFile || row.details?.source || "";
				const extra = row.details?.status || row.details?.count || "";
				output.appendLine(`  ${row.ts} :: ${row.action}${file ? ` :: ${file}` : ""}${extra ? ` :: ${extra}` : ""}`);
			}
			output.appendLine("=================\n");
			writeAudit(host, "plan", "showPlanHistory", { count: recent.length });
		}),
		vscode.commands.registerCommand("vibe-code.showGoalPlanMap", async () => {
			const paths = ensureWorkspacePaths(true);
			if (!paths) return;
			const items = [...linkedPlans(paths.plans, "active"), ...linkedPlans(paths.archive, "archive")];
			output.show(true);
			output.appendLine("\n=== 목표-계획 연결 ===");
			output.appendLine(`goal: ${currentGoalTitle(paths.current)}`);
			output.appendLine("goal file: " + GOAL_FILE);
			if (items.length === 0) output.appendLine("\n  (linked plans 없음)");
			for (const r of items) output.appendLine(`  [${r.bucket}] ${r.name} :: ${r.status} :: ${r.priority} :: ${r.title} :: ${r.linkedTitle || r.linked}`);
			output.appendLine("======================\n");
			writeAudit(host, "plan", "showGoalPlanMap", { goalFile: GOAL_FILE, count: items.length });
		}),
	);
}

// --- views -----------------------------------------------------------------------------------

const OPEN_PLAN: vscode.Command = { command: "vibe-code.openLatestPlan", title: "최신 계획 열기" };
const OPEN_GOAL: vscode.Command = { command: "vibe-code.openCurrentGoal", title: "현재 목표 열기" };

function planBoardRoots(): TreeNode[] {
	const plan = readPlan();
	if (!plan) {
		return [makeItem("현재 계획 없음", { description: "명령으로 생성", tooltip: "최신 계획 열기 명령으로 계획 draft를 만드세요.", command: OPEN_PLAN, icon: "project" })];
	}
	const meta = planMeta(plan.text, plan.name);
	const progress = progressLabel(countChecks(plan.text), "체크 없음");
	const steps = taskLines(section(plan.text, "단계"));
	const now = taskLines(section(plan.text, "Now"));
	const next = taskLines(section(plan.text, "Next"));
	const done = lines(section(plan.text, "Done"));
	const risks = lines(section(plan.text, "Risks"));
	const expanded = vscode.TreeItemCollapsibleState.Expanded;
	const collapsed = vscode.TreeItemCollapsibleState.Collapsed;
	return [
		makeItem(meta.title, {
			description: `${progress} · ${meta.status} · ${meta.priority}`,
			tooltip: `상태: ${meta.status}\n우선순위: ${meta.priority}\n파일: .vibe-code/plans/${plan.name}`,
			command: OPEN_PLAN,
			icon: "project",
		}),
		makeItem("단계", { description: progress, tooltip: "계획 단계 체크리스트", collapsibleState: expanded, icon: "checklist", children: checklistItems(steps, "단계 없음", "check") }),
		makeItem("Now", { description: now.length > 0 ? `${now.length}개` : "비어 있음", tooltip: "현재 실행 작업", collapsibleState: expanded, icon: "play", children: checklistItems(now, "Now 항목 없음", "circle-outline") }),
		makeItem("Next", { description: next.length > 0 ? `${next.length}개` : "비어 있음", tooltip: "다음 실행 작업", collapsibleState: collapsed, icon: "arrow-right", children: checklistItems(next, "Next 항목 없음", "circle-large-outline") }),
		makeItem("Done", { description: done.length > 0 ? `${done.length}줄` : "없음", tooltip: "완료된 작업", collapsibleState: collapsed, icon: "pass", children: noteItems(done, "완료 기록 없음") }),
		makeItem("Risks", { description: risks.length > 0 ? `${risks.length}줄` : "없음", tooltip: "리스크와 제약", collapsibleState: collapsed, icon: "warning", children: noteItems(risks, "리스크 없음") }),
		makeItem("바로 실행", {
			description: "명령 15개",
			tooltip: "계획/목표 관련 명령",
			collapsibleState: collapsed,
			icon: "tools",
			children: [
				commandItem("최신 계획 열기", "vibe-code.openLatestPlan", "최신 계획 열기", "go-to-file"),
				commandItem("현재 계획 진행", "vibe-code.advanceCurrentPlan", "현재 계획 진행", "run"),
				commandItem("계획 상태 변경", "vibe-code.setCurrentPlanStatus", "현재 계획 상태 변경", "symbol-enum"),
				commandItem("현재 계획 우선순위 변경", "vibe-code.setCurrentPlanPriority", "현재 계획 우선순위 변경", "arrow-up"),
				commandItem("현재 계획 목표 연결", "vibe-code.linkCurrentPlanToGoal", "현재 계획 목표 연결", "link"),
				commandItem("완료 기준 연결", "vibe-code.linkPlanToCriteria", "완료 기준 연결", "checklist"),
				commandItem("완료 계획 보관", "vibe-code.archiveDonePlans", "완료 계획 보관", "archive"),
				commandItem("보관 계획 복원", "vibe-code.restoreArchivedPlan", "보관 계획 복원", "history"),
				commandItem("현재 계획 선택", "vibe-code.setActivePlan", "현재 계획 선택", "go-to-file"),
				commandItem("최고 우선순위 계획 선택", "vibe-code.selectHighestPriorityPlan", "최고 우선순위 계획 선택", "arrow-up"),
				commandItem("계획 목록 보기", "vibe-code.showPlanCatalog", "계획 목록 보기", "list-tree"),
				commandItem("계획 이력 보기", "vibe-code.showPlanHistory", "계획 이력 보기", "history"),
				commandItem("목표-계획 연결 보기", "vibe-code.showGoalPlanMap", "목표-계획 연결 보기", "organization"),
				commandItem("현재 목표 열기", "vibe-code.openCurrentGoal", "현재 목표 열기", "target"),
				commandItem("목표 상태 보기", "vibe-code.showGoalStatus", "목표 상태 보기", "list-tree"),
			],
		}),
	];
}

function linkedPlanItems(rows: LinkedPlan[], emptyLabel: string): TreeNode[] {
	if (rows.length === 0) return [makeItem(emptyLabel, { description: "없음", icon: "circle-slash" })];
	return rows.map((row) => {
		const archived = row.bucket === "archive";
		return makeItem(row.title, {
			description: `${row.bucket} · ${row.status} · ${row.priority}`,
			tooltip:
				`파일: .vibe-code/plans${archived ? "/archive" : ""}/${row.name}\n우선순위: ${row.priority}\n연결 목표: ${row.linkedTitle || row.linked}` +
				`\n클릭 동작: ${archived ? "복원 후 active plan 지정" : "현재 active plan으로 선택"}`,
			command: archived
				? { command: "vibe-code.restoreArchivedPlan", title: "보관 계획 복원", arguments: [row.name] }
				: { command: "vibe-code.setActivePlan", title: "현재 계획 선택", arguments: [row.name] },
			icon: archived ? "archive" : "project",
		});
	});
}

function goalPlanMapRoots(): TreeNode[] {
	const paths = ensureWorkspacePaths(true);
	if (!paths) {
		return [makeItem("목표 정보 없음", { description: "현재 목표 필요", tooltip: "현재 목표 파일을 만들거나 여세요.", command: OPEN_GOAL, icon: "target" })];
	}
	let goalTitle = "현재 목표";
	let goalStatus = "(unknown)";
	try {
		if (fs.existsSync(paths.current)) {
			const text = readUtf8(paths.current);
			goalTitle = headingTitle(text, "목표", goalTitle);
			goalStatus = matchLine(text, "상태", goalStatus);
		}
	} catch {
		// unreadable goal file: keep placeholders
	}
	const active = linkedPlans(paths.plans, "active");
	const archived = linkedPlans(paths.archive, "archive");
	const expanded = vscode.TreeItemCollapsibleState.Expanded;
	const collapsed = vscode.TreeItemCollapsibleState.Collapsed;
	return [
		makeItem(goalTitle, { description: `${active.length + archived.length} linked · ${goalStatus}`, tooltip: `goal file: ${GOAL_FILE}\n상태: ${goalStatus}`, command: OPEN_GOAL, icon: "target" }),
		makeItem("Active Linked Plans", { description: active.length > 0 ? `${active.length}개` : "없음", tooltip: "현재 목표와 연결된 active plan", collapsibleState: expanded, icon: "project", children: linkedPlanItems(active, "linked active plan 없음") }),
		makeItem("Archived Linked Plans", { description: archived.length > 0 ? `${archived.length}개` : "없음", tooltip: "현재 목표와 연결된 archived plan", collapsibleState: collapsed, icon: "archive", children: linkedPlanItems(archived, "linked archived plan 없음") }),
		makeItem("바로 실행", {
			description: "명령 3개",
			tooltip: "목표-계획 연결 관련 명령",
			collapsibleState: collapsed,
			icon: "tools",
			children: [
				commandItem("목표-계획 연결 보기", "vibe-code.showGoalPlanMap", "목표-계획 연결 보기", "organization"),
				commandItem("현재 목표 열기", "vibe-code.openCurrentGoal", "현재 목표 열기", "target"),
				commandItem("최신 계획 열기", "vibe-code.openLatestPlan", "최신 계획 열기", "go-to-file"),
			],
		}),
	];
}

export function registerPlanViews(host: CoreHost): void {
	registerTreeView(host, { viewId: "vibe-code.PlanBoard", roots: planBoardRoots, watch: [".vibe-code/plans/*.md"] });
	log(host, "plan board view registered");
	registerTreeView(host, { viewId: "vibe-code.GoalPlanMap", roots: goalPlanMapRoots, watch: [GOAL_FILE, ".vibe-code/plans/*.md", ".vibe-code/plans/archive/*.md"] });
	log(host, "goal plan map view registered");
}
