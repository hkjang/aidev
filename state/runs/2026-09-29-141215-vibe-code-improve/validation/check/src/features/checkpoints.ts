import { execFileSync } from "node:child_process";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import * as vscode from "vscode";
import type { CoreHost } from "../core/host";
import { log } from "../log";
import { config } from "../settings";
import { kstStamp } from "../util/kst";
import { ensureWorkspacePaths, readUtf8, workspaceRoot, writeAudit } from "./workspace";

const DESTRUCTIVE: RegExp[] = [
	/\brm\s+(-[a-z]*r[a-z]*f?|-[a-z]*f[a-z]*r)\b/i,
	/\brm\s+-rf?\b/i,
	/\b(rmdir|rd)\s+\/s\b/i,
	/\bdel\s+\/[sq]\b/i,
	/Remove-Item\b[^\n]*-Recurse/i,
	/\bgit\s+(reset\s+--hard|clean\s+-[a-z]*f[a-z]*|checkout\s+--\s+\.|restore\s+\.|push\s+[^\n]*--force|branch\s+-D)\b/i,
	/\b(drop|truncate)\s+(table|database|schema)\b/i,
	/\bDROP\s+(TABLE|DATABASE)\b/,
	/\bmkfs\b|\bdd\s+if=/i,
	/\bnpm\s+(publish|unpublish)\b|\bpnpm\s+publish\b/i,
	/\bterraform\s+(destroy|apply)\b|\bkubectl\s+delete\b|\bdocker\s+(system\s+prune|volume\s+rm)\b/i,
	/\bchmod\s+-R\b|\bchown\s+-R\b/i,
];

/** True for commands that delete, rewrite history, or hit production-like targets. Pure. */
export function isDestructiveCommand(command: string): boolean {
	const text = command.trim();
	return DESTRUCTIVE.some((re) => re.test(text));
}

export interface Checkpoint {
	file: string;
	ref?: string;
	commit?: string;
	note: string;
}

/** False in a repository with no commits yet, where `HEAD` cannot be resolved. */
function hasHead(cwd: string): boolean {
	try {
		git(cwd, ["rev-parse", "--verify", "--quiet", "HEAD"]);
		return true;
	} catch {
		return false;
	}
}

function git(cwd: string, args: string[], indexFile?: string): string {
	const env = indexFile ? { ...process.env, GIT_INDEX_FILE: indexFile } : process.env;
	return execFileSync("git", args, { cwd, env, encoding: "utf8", windowsHide: true, stdio: ["ignore", "pipe", "ignore"] }).trim();
}

/**
 * Write a tree holding uncommitted tracked changes *and* untracked files (ignored files excluded,
 * as `git add -A` honours .gitignore). A throwaway index under the OS temp directory keeps the
 * user's own index and working tree untouched — unlike `git stash create`, which snapshots tracked
 * changes only, and unlike `git stash push`, which would move the user's files out from under them.
 */
function snapshotTree(ws: string, parent: string | null): string {
	const indexDir = fs.mkdtempSync(path.join(os.tmpdir(), "vibe-checkpoint-"));
	const indexFile = path.join(indexDir, "index");
	try {
		if (parent) git(ws, ["read-tree", parent], indexFile);
		git(ws, ["add", "-A"], indexFile);
		return git(ws, ["write-tree"], indexFile);
	} finally {
		fs.rmSync(indexDir, { recursive: true, force: true });
	}
}

/**
 * Snapshot the working tree before a destructive command: a commit of tracked changes plus
 * untracked files, built through a temporary index so nothing the user sees moves. It is pinned
 * under refs/vibe-checkpoints/. A note is always written under `.vibe-code/checkpoints/`, even
 * outside a git repository.
 */
export function createCheckpoint(ws: string, checkpointsDir: string, command: string, reason: string): Checkpoint {
	const stamp = kstStamp();
	const slug = command.replace(/[^\p{L}\p{N}]+/gu, "-").replace(/^-+|-+$/g, "").slice(0, 30) || "command";
	const file = path.join(checkpointsDir, `${stamp.file}-${slug}.md`);
	let ref: string | undefined;
	let commit: string | undefined;
	let note: string;
	try {
		git(ws, ["rev-parse", "--is-inside-work-tree"]);
		// A repository with no commits yet has no HEAD: snapshot it as a parentless commit.
		const parent = hasHead(ws) ? git(ws, ["rev-parse", "HEAD"]) : null;
		const head = parent ? git(ws, ["rev-parse", "--short", "HEAD"]) : "커밋 없음";
		const tree = snapshotTree(ws, parent);
		if (parent && tree === git(ws, ["rev-parse", "HEAD^{tree}"])) {
			note = `작업 트리가 깨끗해서 스냅샷 없음 (HEAD ${head}); \`git reflog\`로 복구 가능`;
		} else {
			commit = git(ws, ["commit-tree", tree, ...(parent ? ["-p", parent] : []), "-m", "vibe checkpoint"]);
			ref = `refs/vibe-checkpoints/${stamp.file}`;
			git(ws, ["update-ref", ref, commit]);
			note = `git 스냅샷 생성 (HEAD ${head}, 추적/미추적 파일 커밋 ${commit.slice(0, 12)})`;
		}
	} catch {
		note = "git 저장소가 아니어서 스냅샷을 만들지 못함 — 파일 백업을 직접 확인하세요";
	}
	fs.mkdirSync(checkpointsDir, { recursive: true });
	const body = [
		`# 체크포인트 — ${stamp.human}`,
		"",
		`이유: ${reason}`,
		`명령: \`${command}\``,
		`결과: ${note}`,
		...(commit && ref
			? [
					"",
					"## 복원",
					"",
					"```powershell",
					`git diff ${commit}        # 스냅샷과 지금 작업 트리의 차이 확인`,
					`git checkout ${commit} -- .        # 스냅샷 시점 파일로 덮어쓰기 (추적/미추적 파일 모두)`,
					`git update-ref -d ${ref}        # 정리`,
					"```",
					"",
					"스냅샷에 없던 파일(체크포인트 이후에 생긴 파일)은 이 복원으로 지워지지 않습니다.",
				]
			: []),
		"",
	].join("\n");
	fs.writeFileSync(file, body, "utf8");
	return { file, ref, commit, note };
}

/** Called from the command audit hook before an approved command runs. Synchronous by design. */
export function checkpointBeforeCommand(host: CoreHost, command: string): void {
	if (!(config().get<boolean>("checkpointBeforeDestructive") ?? true)) return;
	if (!isDestructiveCommand(command)) return;
	const ws = workspaceRoot();
	const paths = ensureWorkspacePaths(true);
	if (!ws || !paths) return;
	try {
		const cp = createCheckpoint(ws, paths.checkpoints, command, "파괴적 명령 승인 직전 자동 스냅샷");
		writeAudit(host, "command", "checkpoint", { command, file: `.vibe-code/checkpoints/${path.basename(cp.file)}`, ref: cp.ref, commit: cp.commit, note: cp.note });
		log(host, `checkpoint before destructive command: ${cp.note}`);
		void vscode.window.showInformationMessage(`파괴적 명령 전 체크포인트: ${cp.note}`, "체크포인트 열기").then(async (choice) => {
			if (choice === "체크포인트 열기") await vscode.window.showTextDocument(await vscode.workspace.openTextDocument(cp.file));
		});
	} catch (error) {
		log(host, `checkpoint skipped: ${error}`);
	}
}

export function registerCheckpoints(host: CoreHost): void {
	const { context, output } = host;
	context.subscriptions.push(
		vscode.commands.registerCommand("vibe-code.createCheckpoint", async (reasonArg?: unknown) => {
			const ws = workspaceRoot();
			const paths = ensureWorkspacePaths();
			if (!ws || !paths) return;
			const reason = typeof reasonArg === "string" && reasonArg ? reasonArg : (await vscode.window.showInputBox({ prompt: "체크포인트 이유 (선택)", placeHolder: "예: 대규모 리팩터링 전" })) || "수동 체크포인트";
			const cp = createCheckpoint(ws, paths.checkpoints, "(manual)", reason);
			writeAudit(host, "command", "checkpoint", { command: "(manual)", file: `.vibe-code/checkpoints/${path.basename(cp.file)}`, ref: cp.ref, commit: cp.commit, note: cp.note });
			void vscode.window.showInformationMessage(`체크포인트 생성: ${cp.note}`);
		}),
		vscode.commands.registerCommand("vibe-code.listCheckpoints", async () => {
			const paths = ensureWorkspacePaths();
			if (!paths) return;
			const files = fs.existsSync(paths.checkpoints) ? fs.readdirSync(paths.checkpoints).filter((n) => n.endsWith(".md")).sort().slice(-20) : [];
			output.show(true);
			output.appendLine("\n=== 체크포인트 (최근 20개) ===");
			if (files.length === 0) output.appendLine("  (없음)");
			for (const name of files) {
				const text = readUtf8(path.join(paths.checkpoints, name));
				const cmd = text.match(/^명령: `(.*)`$/m)?.[1] || "";
				const result = text.match(/^결과: (.*)$/m)?.[1] || "";
				output.appendLine(`  ${name} :: ${cmd} :: ${result}`);
			}
			output.appendLine("================================\n");
		}),
	);
	log(host, "checkpoint commands registered");
}
