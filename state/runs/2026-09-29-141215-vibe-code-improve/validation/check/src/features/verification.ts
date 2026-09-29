import { spawn } from "node:child_process";
import { execFileSync } from "node:child_process";
import * as fs from "node:fs";
import * as path from "node:path";
import * as vscode from "vscode";
import type { CoreHost } from "../core/host";
import { log } from "../log";
import { kstStamp } from "../util/kst";
import { detectEol, hasSection, isTaskLine, normalizeEol, restoreEol, sectionLines, sectionOfLine, stripTask, touchPlan, writeSection } from "../util/markdown";
import { ensureWorkspacePaths, readUtf8, workspaceRoot, writeAudit } from "./workspace";
import { selectPlanName } from "./plans";

const RUNNER_PREFIXES = ["npm ", "npx ", "pnpm ", "yarn ", "node ", "go ", "pytest", "python ", "python3 ", "cargo ", "make", "dotnet ", "mvn ", "gradle", "./gradlew", "powershell ", "pwsh ", "bash ", "sh ", "./", "vitest", "jest", "tsc", "eslint", "ruff", "black "];

/** The shell command embedded in a checklist/verification line, or null when it is prose. */
export function extractCommand(line: string): string | null {
	const text = stripTask(line.trim()).replace(/^-\s+/, "").trim();
	const ticked = text.match(/`([^`]+)`/);
	if (ticked) return ticked[1].trim();
	const first = text.split(/\s+/)[0] || "";
	if (RUNNER_PREFIXES.some((p) => text.startsWith(p)) || /^[\w.-]+(\.sh|\.ps1|\.cmd|\.bat)$/.test(first)) return text;
	return null;
}

export interface VerificationEntry {
	stamp: string;
	command: string;
	exitCode: number | null;
	durationMs: number;
	note?: string;
}

export function formatVerificationLine(entry: VerificationEntry): string {
	const status = entry.exitCode === 0 ? "OK" : entry.exitCode === null ? "TIMEOUT" : `FAIL exit ${entry.exitCode}`;
	return `- ${entry.stamp} - \`${entry.command}\` → ${status} (${(entry.durationMs / 1000).toFixed(1)}s)${entry.note ? ` ${entry.note}` : ""}`;
}

/** Append a line to `## 검증 로그`, creating the section at the end when missing. 줄바꿈은 보존한다. */
export function appendVerificationLog(text: string, line: string): string {
	// 섹션 판정은 hasSection 하나로 — 자체 정규식은 CRLF 파일의 `## 검증 로그\r\n` 을 놓쳐 섹션을 중복 생성했다.
	if (hasSection(text, "검증 로그")) {
		// 기존 줄은 원본 그대로 다시 쓴다 — lines() 로 재조립하면 들여쓰기와 빈 줄이 사라진다.
		return writeSection(text, "검증 로그", [...sectionLines(text, "검증 로그"), line]);
	}
	const eol = detectEol(text);
	return restoreEol(normalizeEol(text).replace(/\s*$/, "") + `\n\n## 검증 로그\n${line}\n`, eol);
}

/** Tick the checklist line at `lineIndex` when it is unchecked. 줄바꿈은 보존한다. */
export function checkLine(text: string, lineIndex: number): string {
	const eol = detectEol(text);
	const all = normalizeEol(text).split("\n");
	if (all[lineIndex] && /^- \[ \] /.test(all[lineIndex].trim())) all[lineIndex] = all[lineIndex].replace("- [ ] ", "- [x] ");
	return restoreEol(all.join("\n"), eol);
}

export interface RunResult {
	exitCode: number | null;
	durationMs: number;
	output: string;
}

/** Run a shell command in the workspace root, streaming output to the Vibe Code channel. */
export function runShellCommand(host: CoreHost, command: string, cwd: string, timeoutMs = 10 * 60 * 1000): Promise<RunResult> {
	return new Promise((resolve) => {
		const started = Date.now();
		let output = "";
		host.output.show(true);
		host.output.appendLine(`\n$ ${command}   (cwd: ${cwd})`);
		const child = spawn(command, { cwd, shell: true, env: process.env, windowsHide: true });
		const onData = (chunk: Buffer) => {
			const text = chunk.toString("utf8");
			output += text;
			for (const line of text.split(/\r?\n/)) if (line) host.output.appendLine(`  ${line}`);
		};
		child.stdout?.on("data", onData);
		child.stderr?.on("data", onData);
		const timer = setTimeout(() => {
			child.kill();
			resolve({ exitCode: null, durationMs: Date.now() - started, output });
		}, timeoutMs);
		child.on("error", (error) => {
			clearTimeout(timer);
			host.output.appendLine(`  (spawn failed: ${error.message})`);
			resolve({ exitCode: 127, durationMs: Date.now() - started, output });
		});
		child.on("close", (code) => {
			clearTimeout(timer);
			host.output.appendLine(`→ exit ${code} (${((Date.now() - started) / 1000).toFixed(1)}s)`);
			resolve({ exitCode: code, durationMs: Date.now() - started, output });
		});
	});
}

interface VerificationTarget {
	file: string;
	lineIndex?: number;
}

/** Run one command and record it in the target plan/goal file (and the goal file when a plan was targeted). */
async function runAndRecord(host: CoreHost, command: string, target: VerificationTarget | null): Promise<RunResult> {
	const ws = workspaceRoot() || process.cwd();
	const result = await runShellCommand(host, command, ws);
	const entry: VerificationEntry = { stamp: kstStamp().human, command, exitCode: result.exitCode, durationMs: result.durationMs };
	const line = formatVerificationLine(entry);
	const touched: string[] = [];
	const record = (file: string, tick?: number) => {
		if (!fs.existsSync(file)) return;
		let text = readUtf8(file);
		text = appendVerificationLog(text, line);
		if (tick !== undefined && result.exitCode === 0) text = checkLine(text, tick);
		text = touchPlan(text, null, entry.stamp);
		fs.writeFileSync(file, text, "utf8");
		touched.push(path.basename(file));
	};
	if (target) record(target.file, target.lineIndex);
	const paths = ensureWorkspacePaths(true);
	if (paths && fs.existsSync(paths.current) && (!target || path.resolve(target.file) !== path.resolve(paths.current))) record(paths.current);
	writeAudit(host, "plan", "runVerification", { command, exitCode: result.exitCode, durationMs: result.durationMs, recordedIn: touched });
	const label = result.exitCode === 0 ? "통과" : result.exitCode === null ? "시간 초과" : `실패 (exit ${result.exitCode})`;
	void (result.exitCode === 0 ? vscode.window.showInformationMessage : vscode.window.showWarningMessage)(`검증 ${label}: ${command}`);
	return result;
}

// --- test suggestion ---------------------------------------------------------------------------

export interface ProjectInfo {
	packageScripts: Record<string, string>;
	devDependencies: string[];
	hasGoMod: boolean;
	hasPytest: boolean;
	hasCargo: boolean;
	existingFiles: Set<string>;
}

export interface TestSuggestion {
	command: string;
	reason: string;
}

const TEST_FILE_RE = /(\.test\.|\.spec\.|_test\.go$|^test_.*\.py$|\/tests?\/|__tests__\/)/;

/** Map changed source files to test commands. Pure; `info.existingFiles` holds workspace-relative paths. */
export function suggestTestCommands(changedFiles: string[], info: ProjectInfo): TestSuggestion[] {
	const suggestions: TestSuggestion[] = [];
	const seen = new Set<string>();
	const add = (command: string, reason: string) => {
		if (seen.has(command)) return;
		seen.add(command);
		suggestions.push({ command, reason });
	};
	const jsRunner = info.devDependencies.includes("vitest") ? "npx vitest run" : info.devDependencies.includes("jest") ? "npx jest" : info.packageScripts.test ? "npm test --" : "";
	for (const raw of changedFiles) {
		const file = raw.replace(/\\/g, "/");
		const ext = path.posix.extname(file);
		const dir = path.posix.dirname(file);
		const base = path.posix.basename(file, ext);
		if (TEST_FILE_RE.test(file)) {
			if (/\.(ts|tsx|js|jsx|mjs)$/.test(file) && jsRunner) add(`${jsRunner} ${file}`, `변경된 테스트 파일`);
			else if (file.endsWith("_test.go")) add(`go test ./${dir}/...`, `변경된 Go 테스트`);
			else if (file.endsWith(".py")) add(`pytest ${file}`, `변경된 파이썬 테스트`);
			continue;
		}
		if (/\.(ts|tsx|js|jsx|mjs)$/.test(file) && jsRunner) {
			const candidates = [`${dir}/${base}.test${ext}`, `${dir}/${base}.spec${ext}`, `${dir}/__tests__/${base}.test${ext}`, `tests/unit/${base}.test.ts`, `tests/${base}.test${ext}`, `test/${base}.test${ext}`];
			const hit = candidates.find((c) => info.existingFiles.has(c));
			if (hit) add(`${jsRunner} ${hit}`, `${file} 의 테스트`);
		} else if (file.endsWith(".go") && info.hasGoMod) {
			add(`go test ./${dir === "." ? "" : dir + "/"}...`, `${file} 의 패키지`);
		} else if (file.endsWith(".py") && info.hasPytest) {
			const candidates = [`${dir}/test_${base}.py`, `tests/test_${base}.py`, `test/test_${base}.py`];
			const hit = candidates.find((c) => info.existingFiles.has(c));
			add(hit ? `pytest ${hit}` : "pytest", hit ? `${file} 의 테스트` : "전체 파이썬 테스트");
		} else if (file.endsWith(".rs") && info.hasCargo) {
			add("cargo test", `${file} (Rust)`);
		}
	}
	if (changedFiles.length > 0) {
		if (info.packageScripts.check) add("npm run check", "프로젝트 전체 검사 스크립트");
		else if (info.packageScripts.test) add("npm test", "프로젝트 테스트 스크립트");
		if (info.hasGoMod && changedFiles.some((f) => f.endsWith(".go"))) add("go test ./...", "전체 Go 테스트");
	}
	return suggestions;
}

function gitChangedFiles(ws: string): string[] {
	try {
		const out = execFileSync("git", ["status", "--porcelain", "--untracked-files=all"], { cwd: ws, encoding: "utf8", windowsHide: true });
		return out
			.split(/\r?\n/)
			.filter(Boolean)
			.map((l) => l.slice(3).trim().replace(/^"|"$/g, ""))
			.map((l) => (l.includes(" -> ") ? l.split(" -> ")[1] : l));
	} catch {
		return [];
	}
}

function readProjectInfo(ws: string, changed: string[]): ProjectInfo {
	let packageScripts: Record<string, string> = {};
	let devDependencies: string[] = [];
	try {
		const pkg = JSON.parse(readUtf8(path.join(ws, "package.json")));
		packageScripts = pkg.scripts || {};
		devDependencies = Object.keys({ ...(pkg.devDependencies || {}), ...(pkg.dependencies || {}) });
	} catch {
		// not a node project
	}
	const existing = new Set<string>();
	const probe = (rel: string) => {
		if (fs.existsSync(path.join(ws, rel))) existing.add(rel);
	};
	for (const raw of changed) {
		const file = raw.replace(/\\/g, "/");
		const ext = path.posix.extname(file);
		const dir = path.posix.dirname(file);
		const base = path.posix.basename(file, ext);
		for (const c of [`${dir}/${base}.test${ext}`, `${dir}/${base}.spec${ext}`, `${dir}/__tests__/${base}.test${ext}`, `tests/unit/${base}.test.ts`, `tests/${base}.test${ext}`, `test/${base}.test${ext}`, `${dir}/test_${base}.py`, `tests/test_${base}.py`, `test/test_${base}.py`]) probe(c);
	}
	return {
		packageScripts,
		devDependencies,
		hasGoMod: fs.existsSync(path.join(ws, "go.mod")),
		hasPytest: fs.existsSync(path.join(ws, "pytest.ini")) || fs.existsSync(path.join(ws, "pyproject.toml")) || fs.existsSync(path.join(ws, "setup.cfg")),
		hasCargo: fs.existsSync(path.join(ws, "Cargo.toml")),
		existingFiles: existing,
	};
}

// --- CodeLens + commands ---------------------------------------------------------------------------

const SELECTOR: vscode.DocumentSelector = [
	{ scheme: "file", pattern: "**/.vibe-code/plans/*.md" },
	{ scheme: "file", pattern: "**/.vibe-code/goals/*.md" },
];

class VerificationCodeLensProvider implements vscode.CodeLensProvider {
	provideCodeLenses(document: vscode.TextDocument): vscode.CodeLens[] {
		const text = document.getText();
		const result: vscode.CodeLens[] = [];
		for (let i = 0; i < document.lineCount; i++) {
			const line = document.lineAt(i).text;
			if (!isTaskLine(line.trim()) && !/^- /.test(line.trim())) continue;
			const sectionName = sectionOfLine(text, i);
			if (!["검증 계획", "검증", "완료 기준"].includes(sectionName)) continue;
			const command = extractCommand(line);
			if (!command) continue;
			result.push(new vscode.CodeLens(new vscode.Range(i, 0, i, 0), { title: "$(play) 검증 실행", command: "vibe-code.runVerification", arguments: [document.uri.toString(), i] }));
		}
		return result;
	}
}

export function registerVerification(host: CoreHost): void {
	const { context } = host;
	context.subscriptions.push(
		vscode.languages.registerCodeLensProvider(SELECTOR, new VerificationCodeLensProvider()),
		vscode.commands.registerCommand("vibe-code.runVerification", async (uriString: string, lineIndex: number) => {
			const file = vscode.Uri.parse(uriString).fsPath;
			if (!fs.existsSync(file)) return;
			const line = readUtf8(file).split(/\r?\n/)[lineIndex] || "";
			const command = extractCommand(line);
			if (!command) {
				void vscode.window.showWarningMessage("이 줄에서 실행할 명령을 찾지 못했습니다. 명령을 백틱(`)으로 감싸세요.");
				return;
			}
			await runAndRecord(host, command, { file, lineIndex });
		}),
		vscode.commands.registerCommand("vibe-code.runVerificationCommand", async (commandArg?: unknown) => {
			const paths = ensureWorkspacePaths();
			if (!paths) return;
			let command = typeof commandArg === "string" ? commandArg : "";
			let target: VerificationTarget | null = null;
			if (!command) {
				const planFile = path.join(paths.plans, selectPlanName(paths));
				const items: Array<vscode.QuickPickItem & { command: string; lineIndex?: number; file?: string }> = [];
				if (fs.existsSync(planFile)) {
					const all = readUtf8(planFile).split(/\r?\n/);
					all.forEach((l, i) => {
						const c = sectionOfLine(all.join("\n"), i) === "검증 계획" ? extractCommand(l) : null;
						if (c) items.push({ label: `$(play) ${c}`, description: path.basename(planFile), command: c, lineIndex: i, file: planFile });
					});
				}
				items.push({ label: "$(edit) 직접 입력...", command: "" });
				const picked = await vscode.window.showQuickPick(items, { placeHolder: "실행할 검증 명령" });
				if (!picked) return;
				command = picked.command || (await vscode.window.showInputBox({ prompt: "실행할 명령", placeHolder: "npm test" })) || "";
				if (!command) return;
				target = picked.file ? { file: picked.file, lineIndex: picked.lineIndex } : null;
			}
			await runAndRecord(host, command, target);
		}),
		vscode.commands.registerCommand("vibe-code.suggestTests", async () => {
			const ws = workspaceRoot();
			if (!ws) {
				void vscode.window.showWarningMessage("워크스페이스를 먼저 열어주세요.");
				return;
			}
			const changed = gitChangedFiles(ws);
			const suggestions = suggestTestCommands(changed, readProjectInfo(ws, changed));
			if (suggestions.length === 0) {
				void vscode.window.showInformationMessage(changed.length ? "변경 파일에 대응하는 테스트 명령을 찾지 못했습니다." : "git 기준 변경된 파일이 없습니다.");
				return;
			}
			const picked = await vscode.window.showQuickPick(
				suggestions.map((s) => ({ label: `$(beaker) ${s.command}`, description: s.reason, picked: true, command: s.command })),
				{ canPickMany: true, placeHolder: `변경 파일 ${changed.length}개 → 실행할 검증 선택` },
			);
			if (!picked || picked.length === 0) return;
			writeAudit(host, "plan", "suggestTests", { changed: changed.length, selected: picked.map((p) => p.command) });
			for (const item of picked) {
				const result = await runAndRecord(host, item.command, null);
				if (result.exitCode !== 0) break;
			}
		}),
	);
	log(host, "verification runner registered");
}
