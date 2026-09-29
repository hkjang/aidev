import * as fs from "node:fs";
import * as https from "node:https";
import * as path from "node:path";
import * as vscode from "vscode";
import type { CoreHost } from "../core/host";
import { log } from "../log";
import { config } from "../settings";
import { userLocaleDir } from "./i18n-overrides";
import { workspaceRoot } from "./workspace";

/** `Vibe Code: 컨텍스트 사용량` — dump the runtime state that affects context size. */
export function registerShowContextStats(host: CoreHost): void {
	const { context, output } = host;
	context.subscriptions.push(
		vscode.commands.registerCommand("vibe-code.showContextStats", async () => {
			const state = context.globalState;
			const stats: Record<string, unknown> = {
				mode: state.get("mode") || "code",
				language: state.get("language") || "(default)",
				autonomy: state.get("vibeCode.autonomyApplied") || "(unset)",
				telemetry: state.get("telemetrySetting") || "(unset)",
				sessionTokens: state.get("vibeCode.sessionTokens") || 0,
				autoCondenseContextPercent: state.get("autoCondenseContextPercent") || "(default 75)",
				alwaysAllowReadOnly: state.get("alwaysAllowReadOnly"),
				alwaysAllowWrite: state.get("alwaysAllowWrite"),
				alwaysAllowExecute: state.get("alwaysAllowExecute"),
				alwaysAllowBrowser: state.get("alwaysAllowBrowser"),
				alwaysAllowMcp: state.get("alwaysAllowMcp"),
			};
			output.show(true);
			output.appendLine("\n=== Vibe Code 컨텍스트 사용량 ===");
			for (const key of Object.keys(stats)) output.appendLine(`  ${key.padEnd(28)} ${stats[key]}`);
			output.appendLine("================================\n");
		}),
	);
}

function directorySize(dir: string): number {
	let total = 0;
	try {
		for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
			const full = path.join(dir, entry.name);
			total += entry.isDirectory() ? directorySize(full) : fs.statSync(full).size;
		}
	} catch {
		// unreadable subtree: count what we could
	}
	return total;
}

/** `showIndexInfo` and `healthCheck` commands. */
export function registerDiagnostics(host: CoreHost): void {
	const { context, output } = host;
	context.subscriptions.push(
		vscode.commands.registerCommand("vibe-code.showIndexInfo", async () => {
			const ws = workspaceRoot();
			output.show(true);
			output.appendLine("\n=== Vibe Code 임베딩 인덱스 ===");
			try {
				const globalStorage = context.globalStorageUri?.fsPath;
				output.appendLine(`globalStorage: ${globalStorage}`);
				if (globalStorage && fs.existsSync(globalStorage)) {
					output.appendLine(`  total size: ${(directorySize(globalStorage) / 1024 / 1024).toFixed(2)} MB`);
				}
			} catch (error) {
				output.appendLine(`  (size scan failed: ${error})`);
			}
			if (ws) {
				const vectorDb = path.join(ws, ".vibe-code", "vectordb");
				if (fs.existsSync(vectorDb)) output.appendLine(`workspace index: ${vectorDb}`);
				else output.appendLine("workspace index: (none — codebase search not initialized)");
			}
			output.appendLine("================================\n");
		}),
		vscode.commands.registerCommand("vibe-code.healthCheck", async () => {
			output.show(true);
			output.appendLine("\n=== Vibe Code 한국어 환경 점검 ===");
			const cfg = config();
			const state = context.globalState;
			const rows: Array<[string, unknown]> = [
				["언어 설정 (settings)", cfg.get("language") || "(미설정)"],
				["언어 (globalState)", state.get("language") || "(미설정)"],
				["톤", cfg.get("tone") || "(미설정)"],
				["자율성 등급", cfg.get("autonomy") || "(미설정)"],
				["적용된 자율성", state.get("vibeCode.autonomyApplied") || "(미적용)"],
				["텔레메트리", state.get("telemetrySetting") || "(미설정)"],
				["거부 명령 개수", (cfg.get<string[]>("deniedCommands") || []).length],
				["허용 명령 개수", (cfg.get<string[]>("allowedCommands") || []).length],
				["코드 액션 활성화", cfg.get("enableCodeActions")],
				["코드 실행 타임아웃", cfg.get("commandExecutionTimeout") || 0],
				["VS Code 버전", vscode.version],
				["확장 버전", host.pkg.version],
			];
			for (const [key, value] of rows) output.appendLine(`  ${key.padEnd(24)} ${value}`);
			const ws = workspaceRoot();
			if (ws) {
				const checks: Array<[string, boolean]> = [
					[".vibeignore", fs.existsSync(path.join(ws, ".vibeignore"))],
					[".vibemodes", fs.existsSync(path.join(ws, ".vibemodes"))],
					[".vibe/commands 슬래시", fs.existsSync(path.join(ws, ".vibe", "commands"))],
					[".vibe-code/journal", fs.existsSync(path.join(ws, ".vibe-code", "journal"))],
					[".vibe-code/demo-scenarios.md", fs.existsSync(path.join(ws, ".vibe-code", "demo-scenarios.md"))],
				];
				output.appendLine("");
				output.appendLine("워크스페이스 파일:");
				for (const [key, ok] of checks) output.appendLine(`  ${ok ? "OK " : "-- "} ${key}`);
			}
			const overrides = userLocaleDir();
			output.appendLine("");
			output.appendLine(`사용자 오버라이드: ${fs.existsSync(overrides) ? "감지됨 (" + overrides + ")" : "없음"}`);
			output.appendLine("================================\n");
		}),
	);
	log(host, "healthCheck + showIndexInfo commands registered");
}

interface HostProbe {
	host: string;
	ok: boolean;
	status?: number;
	err?: string;
}

function probeHost(hostName: string, port = 443, timeout = 2000): Promise<HostProbe> {
	return new Promise((resolve) => {
		try {
			const req = https.request({ host: hostName, port, method: "HEAD", timeout, path: "/" }, (res) => {
				resolve({ host: hostName, ok: true, status: res.statusCode });
				res.resume();
				req.destroy();
			});
			req.on("error", (error: NodeJS.ErrnoException) => resolve({ host: hostName, ok: false, err: error.code || error.message }));
			req.on("timeout", () => {
				resolve({ host: hostName, ok: false, err: "timeout" });
				req.destroy();
			});
			req.end();
		} catch (error) {
			resolve({ host: hostName, ok: false, err: String(error) });
		}
	});
}

const AUDIT_HOSTS = ["1.1.1.1", "api.anthropic.com", "api.openai.com", "registry.npmjs.org", "github.com", "huggingface.co"];

interface McpServerConfig {
	command?: string;
	args?: unknown[];
}

/** `Vibe Code: 외부 통신 점검` — reachability of well-known hosts plus provider/MCP summary. */
export function registerAuditNetwork(host: CoreHost): void {
	const { context, output } = host;
	context.subscriptions.push(
		vscode.commands.registerCommand("vibe-code.auditNetwork", async () => {
			output.show(true);
			output.appendLine("\n=== Vibe Code 외부 통신 점검 ===");
			output.appendLine(`offlineMode 설정: ${config().get("offlineMode") || "auto"}`);
			output.appendLine(`HTTPS_PROXY: ${process.env.HTTPS_PROXY || process.env.HTTP_PROXY || "(없음)"}`);
			output.appendLine(`NODE_EXTRA_CA_CERTS: ${process.env.NODE_EXTRA_CA_CERTS || "(없음)"}`);
			output.appendLine("\n--- 외부 호스트 도달성 (HEAD, 2초 timeout) ---");
			for (const name of AUDIT_HOSTS) {
				const result = await probeHost(name);
				output.appendLine(`  ${result.ok ? "✓" : "✗"} ${name.padEnd(28)} ${result.ok ? "(" + result.status + ")" : result.err}`);
			}
			output.appendLine("\n--- API Provider 확인 ---");
			const state = context.globalState;
			const apiConfiguration = state.get<{ apiProvider?: string; openAiBaseUrl?: string }>("apiConfiguration");
			output.appendLine(`  현재 프로바이더: ${state.get("apiProvider") || apiConfiguration?.apiProvider || "(미설정)"}`);
			output.appendLine(`  Base URL (있다면): ${apiConfiguration?.openAiBaseUrl || "(N/A)"}`);
			output.appendLine("\n--- MCP 서버 (사용자 등록분) ---");
			const servers = state.get<Record<string, McpServerConfig>>("mcpServers") || {};
			const names = Object.keys(servers);
			if (names.length === 0) output.appendLine("  (등록된 MCP 서버 없음)");
			for (const name of names) {
				const server = servers[name];
				const usesNpx = server?.command === "npx" || (server?.args || []).some((arg) => typeof arg === "string" && arg.startsWith("@"));
				output.appendLine(`  ${usesNpx ? "⚠" : "✓"} ${name} ${usesNpx ? "(npx — 인터넷 필요 가능성)" : ""}`);
			}
			output.appendLine("\n--- 권장 ---");
			output.appendLine("  · 외부 호스트가 모두 ✗ 이면 OFFLINE-FIRST 모드가 적합합니다 (settings: vibe-code.offlineMode = offline).");
			output.appendLine("  · 프록시가 있다면 vibe-code.proxyUrl 설정.");
			output.appendLine("  · 사내 self-signed 인증서가 있다면 vibe-code.extraCaCertsPath 설정.");
			output.appendLine("================================\n");
		}),
	);
	log(host, "auditNetwork command registered");
}
