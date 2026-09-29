import * as fs from "node:fs";
import * as http from "node:http";
import * as https from "node:https";
import * as path from "node:path";
import * as vscode from "vscode";
import type { CoreHost } from "../core/host";
import { log } from "../log";
import { stringSetting } from "../settings";
import { NO_WORKSPACE_MESSAGE, openFile, workspaceRoot, writeAudit } from "./workspace";
import { cachedWeeklyUsage, invalidateUsageCache, isOverBudget, usageSummaryLine } from "./usage";

const DEFAULT_BASE_URL = "http://localhost:8080/v1";
const DEFAULT_PROFILE = "vibe-coders proxy";
const DEFAULT_MODEL = "gpt-4.1-mini";
const DEFAULT_KEY = "dev-proxy-key";
const HEADER = "X-Proxy-Provider";
const AUTO_ROUTING = "(model auto routing)";

const baseUrl = (): string => stringSetting("vibeCodersBaseUrl", DEFAULT_BASE_URL).replace(/\/+$/g, "") || DEFAULT_BASE_URL;
const profileName = (): string => stringSetting("vibeCodersProfileName", DEFAULT_PROFILE);
const defaultModel = (): string => stringSetting("vibeCodersDefaultModel", DEFAULT_MODEL);
const providerHeader = (): string => stringSetting("vibeCodersProviderHeader", "");

/** Origin of the proxy (base URL without `/v1`), used for /health and /ready probes. */
export function proxyRootUrl(base: string): string {
	try {
		const url = new URL(base);
		url.pathname = url.pathname.replace(/\/v1\/?$/, "/");
		url.search = "";
		url.hash = "";
		return url.origin;
	} catch {
		return "http://localhost:8080";
	}
}

/** Best guess for the local vibe-coders checkout, honoring `vibe-code.vibeCodersProjectPath`. */
function projectPath(host: CoreHost): string {
	const raw = stringSetting("vibeCodersProjectPath", "../vibe-coders");
	const ws = workspaceRoot();
	const candidates: string[] = [];
	if (path.isAbsolute(raw)) candidates.push(raw);
	if (ws) {
		candidates.push(path.resolve(ws, raw));
		candidates.push(path.resolve(ws, "..", "vibe-coders"));
		candidates.push(path.resolve(ws, "..", "projects", "vibe-coders"));
	}
	candidates.push(path.resolve(host.context.extensionPath, raw));
	try {
		candidates.push(path.resolve(process.cwd(), raw));
	} catch {
		// cwd may be unavailable in the extension host
	}
	const unique = Array.from(new Set(candidates.filter(Boolean)));
	return unique.find((p) => fs.existsSync(path.join(p, "go.mod")) || fs.existsSync(path.join(p, "cmd", "gateway", "main.go"))) || unique[0] || raw;
}

interface ProbeResult {
	ok: boolean;
	status?: number;
	error?: string;
}

function probe(url: string): Promise<ProbeResult> {
	return new Promise((resolve) => {
		try {
			const target = new URL(url);
			const client = target.protocol === "https:" ? https : http;
			const req = client.request({ method: "GET", hostname: target.hostname, port: target.port, path: `${target.pathname}${target.search}`, timeout: 2000 }, (res) => {
				res.resume();
				const status = res.statusCode ?? 0;
				resolve({ ok: status >= 200 && status < 500, status });
			});
			req.on("error", (error: NodeJS.ErrnoException) => resolve({ ok: false, error: error.code || error.message }));
			req.on("timeout", () => {
				req.destroy();
				resolve({ ok: false, error: "timeout" });
			});
			req.end();
		} catch (error) {
			resolve({ ok: false, error: String(error) });
		}
	});
}

type ProxyProfile = {
	apiProvider: "openai";
	openAiBaseUrl: string;
	openAiApiKey: string;
	openAiModelId: string;
	openAiStreamingEnabled: true;
	openAiHeaders: Record<string, string>;
};

function buildProfile(apiKey: string): ProxyProfile {
	const header = providerHeader();
	return {
		apiProvider: "openai",
		openAiBaseUrl: baseUrl(),
		openAiApiKey: apiKey || DEFAULT_KEY,
		openAiModelId: defaultModel(),
		openAiStreamingEnabled: true,
		openAiHeaders: header ? { [HEADER]: header } : {},
	};
}

const errorText = (error: unknown): string => (error instanceof Error ? error.message : String(error));

/** Commands and status bar for routing model calls through the OpenAI-compatible vibe-coders proxy. */
export function registerVibeCodersProxy(host: CoreHost): void {
	const { context, output } = host;
	const state = context.globalState;
	const proxyStatus = async () => {
		const cp = await host.contextProxy.getInstance(context);
		const vals = cp.getValues();
		const expected = baseUrl();
		return { cp, vals, expected, routed: vals.apiProvider === "openai" && vals.openAiBaseUrl === expected };
	};

	const item = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 98);
	item.command = "vibe-code.showVibeCodersProxyStatus";
	const refresh = async () => {
		try {
			const { vals, expected, routed } = await proxyStatus();
			const lastProfile = state.get<string>("vibeCode.vibeCodersLastAppliedProfile") || "";
			const lastApplied = state.get<string>("vibeCode.vibeCodersLastAppliedAt") || "";
			if (!routed && !lastProfile) {
				item.hide();
				return;
			}
			const model = vals.openAiModelId || state.get<string>("vibeCode.vibeCodersLastAppliedModel") || "(unset)";
			const status = routed ? "ACTIVE" : "READY";
			const usage = routed ? await cachedWeeklyUsage(host) : null;
			const usageText = usage?.ok && usage.report ? ` · ${usageSummaryLine(usage.report)}` : "";
			const over = isOverBudget(usage?.report);
			item.text = `${over ? "$(warning)" : routed ? "$(plug)" : "$(debug-disconnect)"} VC: ${status}${usageText}${over ? " · 예산 초과" : ""}`;
			item.backgroundColor = over ? new vscode.ThemeColor("statusBarItem.warningBackground") : undefined;
			item.tooltip =
				`vibe-coders proxy\n상태: ${status}\nprofile: ${vals.currentApiConfigName || "(unset)"}\nbaseUrl: ${vals.openAiBaseUrl || expected}\nmodel: ${model}\nlastApplied: ${lastApplied || "(never)"}` +
				(usage && !usage.ok ? `\n사용량: 조회 실패 (${usage.status ? `HTTP ${usage.status}` : usage.error})` : "") +
				"\n클릭하여 상세 상태 보기";
			item.show();
		} catch {
			item.hide();
		}
	};
	// Refreshed when settings change and after the apply/restore commands; the timer only
	// catches profile switches made inside the core webview.
	const interval = setInterval(() => void refresh(), 60_000);
	context.subscriptions.push(
		item,
		{ dispose: () => clearInterval(interval) },
		vscode.workspace.onDidChangeConfiguration((event) => {
			if (event.affectsConfiguration("vibe-code")) void refresh();
		}),
	);

	context.subscriptions.push(
		vscode.commands.registerCommand("vibe-code.applyVibeCodersProxy", async () => {
			const key = await vscode.window.showInputBox({
				prompt: "vibe-coders proxy API key를 입력하세요. 비우면 dev-proxy-key를 사용합니다.",
				placeHolder: "dev-proxy-key 또는 vibe-coders에서 발급한 proxy key",
				password: true,
				ignoreFocusOut: true,
			});
			if (key === undefined) return;
			const profile = buildProfile(key.trim() || DEFAULT_KEY);
			const name = profileName();
			try {
				const cp = await host.contextProxy.getInstance(context);
				const previous = String(cp.getValue("currentApiConfigName") || cp.getValues().currentApiConfigName || "");
				if (previous && previous !== name) {
					await state.update("vibeCode.previousProviderProfile", previous);
					log(host, `previous provider profile saved: ${previous}`);
				}
				await cp.setProviderSettings(profile);
				await cp.setValue("currentApiConfigName", name);
				const appliedAt = new Date().toISOString();
				const header = profile.openAiHeaders[HEADER] || "";
				await state.update("vibeCode.vibeCodersLastAppliedAt", appliedAt);
				await state.update("vibeCode.vibeCodersLastAppliedProfile", name);
				await state.update("vibeCode.vibeCodersLastAppliedBaseUrl", profile.openAiBaseUrl);
				await state.update("vibeCode.vibeCodersLastAppliedModel", profile.openAiModelId);
				await state.update("vibeCode.vibeCodersLastAppliedProviderHeader", header);
				try {
					const provider = host.getProvider();
					if (provider?.providerSettingsManager) {
						await provider.providerSettingsManager.saveConfig(name, profile);
						await provider.activateProviderProfile?.({ name });
						await provider.postStateToWebview?.();
					}
				} catch (inner) {
					log(host, `provider profile UI sync skipped: ${inner}`);
				}
				output.show(true);
				output.appendLine("\n=== vibe-coders proxy 적용 ===");
				output.appendLine(`profile: ${name}`);
				output.appendLine("apiProvider: openai");
				output.appendLine(`openAiBaseUrl: ${profile.openAiBaseUrl}`);
				output.appendLine(`openAiModelId: ${profile.openAiModelId}`);
				output.appendLine(`${HEADER}: ${header || AUTO_ROUTING}`);
				output.appendLine("이후 Vibe Code 모델 호출은 vibe-coders proxy를 통과하며 집계됩니다.");
				output.appendLine("================================\n");
				void vscode.window.showInformationMessage(`vibe-coders proxy profile을 적용했습니다: ${name}`);
				invalidateUsageCache();
				void refresh();
				writeAudit(host, "proxy", "applyVibeCodersProxy", { profile: name, baseUrl: profile.openAiBaseUrl, model: profile.openAiModelId, providerHeader: header, appliedAt });
			} catch (error) {
				log(host, `apply vibe-coders proxy failed: ${error}`);
				writeAudit(host, "proxy", "applyVibeCodersProxyFailed", { error: errorText(error) });
				void vscode.window.showErrorMessage(`vibe-coders proxy 적용 실패: ${errorText(error)}`);
			}
		}),
		vscode.commands.registerCommand("vibe-code.restorePreviousProviderProfile", async () => {
			const previous = state.get<string>("vibeCode.previousProviderProfile");
			if (!previous) {
				void vscode.window.showWarningMessage("복원할 이전 provider profile 기록이 없습니다. vibe-coders 프록시 적용 후 사용할 수 있습니다.");
				return;
			}
			try {
				const provider = host.getProvider();
				let exists = true;
				try {
					if (provider?.providerSettingsManager) exists = await provider.providerSettingsManager.hasConfig(previous);
				} catch (inner) {
					log(host, `previous provider existence check skipped: ${inner}`);
				}
				if (!exists) {
					void vscode.window.showWarningMessage(`이전 provider profile을 찾을 수 없습니다: ${previous}`);
					return;
				}
				const cp = await host.contextProxy.getInstance(context);
				await cp.setValue("currentApiConfigName", previous);
				try {
					if (provider?.activateProviderProfile) {
						await provider.activateProviderProfile({ name: previous });
						await provider.postStateToWebview?.();
					}
				} catch (inner) {
					log(host, `previous provider UI sync skipped: ${inner}`);
				}
				output.show(true);
				output.appendLine("\n=== 이전 provider profile 복원 ===");
				output.appendLine(`profile: ${previous}`);
				output.appendLine("vibe-coders proxy 적용 전 profile로 되돌렸습니다.");
				output.appendLine("================================\n");
				void vscode.window.showInformationMessage(`이전 provider profile로 복원했습니다: ${previous}`);
				void refresh();
				writeAudit(host, "proxy", "restorePreviousProviderProfile", { profile: previous });
			} catch (error) {
				log(host, `restore previous provider failed: ${error}`);
				writeAudit(host, "proxy", "restorePreviousProviderProfileFailed", { error: errorText(error) });
				void vscode.window.showErrorMessage(`이전 provider 복원 실패: ${errorText(error)}`);
			}
		}),
		vscode.commands.registerCommand("vibe-code.showVibeCodersProxyStatus", async () => {
			const { vals, expected, routed } = await proxyStatus();
			const appliedAt = state.get<string>("vibeCode.vibeCodersLastAppliedAt") || "";
			const appliedKst = appliedAt ? new Date(appliedAt).toLocaleString("ko-KR", { timeZone: "Asia/Seoul", hour12: false }) : "(never)";
			const header = vals.openAiHeaders?.[HEADER] || "";
			const lastHeader = state.get<string>("vibeCode.vibeCodersLastAppliedProviderHeader") || "";
			output.show(true);
			output.appendLine("\n=== vibe-coders proxy 상태 ===");
			output.appendLine(`route: ${routed ? "ACTIVE" : "NOT ACTIVE"}`);
			output.appendLine(`expected baseUrl: ${expected}`);
			output.appendLine(`current profile: ${vals.currentApiConfigName || "(unset)"}`);
			output.appendLine(`apiProvider: ${vals.apiProvider || "(unset)"}`);
			output.appendLine(`openAiBaseUrl: ${vals.openAiBaseUrl || "(unset)"}`);
			output.appendLine(`openAiModelId: ${vals.openAiModelId || "(unset)"}`);
			output.appendLine(`${HEADER}: ${header || AUTO_ROUTING}`);
			output.appendLine(`configured ${HEADER}: ${providerHeader() || AUTO_ROUTING}`);
			output.appendLine(`previous profile: ${state.get<string>("vibeCode.previousProviderProfile") || "(none)"}`);
			output.appendLine(`last applied: ${appliedKst}`);
			output.appendLine(`last applied profile: ${state.get<string>("vibeCode.vibeCodersLastAppliedProfile") || "(none)"}`);
			output.appendLine(`last applied baseUrl: ${state.get<string>("vibeCode.vibeCodersLastAppliedBaseUrl") || "(none)"}`);
			output.appendLine(`last applied model: ${state.get<string>("vibeCode.vibeCodersLastAppliedModel") || "(none)"}`);
			output.appendLine(`last applied ${HEADER}: ${lastHeader || AUTO_ROUTING}`);
			output.appendLine("이 명령은 네트워크 요청이나 모델 호출을 만들지 않습니다.");
			output.appendLine("================================\n");
			writeAudit(host, "proxy", "showVibeCodersProxyStatus", {
				route: routed ? "ACTIVE" : "NOT ACTIVE",
				expectedBaseUrl: expected,
				currentProfile: vals.currentApiConfigName || "",
				baseUrl: vals.openAiBaseUrl || "",
				model: vals.openAiModelId || "",
				providerHeader: header,
			});
			if (routed) void vscode.window.showInformationMessage("Vibe Code 모델 호출이 vibe-coders proxy로 라우팅되도록 설정되어 있습니다.");
			else void vscode.window.showWarningMessage("현재 provider profile이 vibe-coders proxy를 향하지 않습니다. vibe-coders 프록시 적용 명령을 실행하세요.");
		}),
		vscode.commands.registerCommand("vibe-code.checkVibeCodersProxy", async () => {
			const { vals, expected, routed } = await proxyStatus();
			const root = proxyRootUrl(expected);
			const health = await probe(`${root}/health`);
			const ready = await probe(`${root}/ready`);
			const project = projectPath(host);
			output.show(true);
			output.appendLine("\n=== vibe-coders proxy 점검 ===");
			output.appendLine(`projectPath: ${project}`);
			output.appendLine(`expected baseUrl: ${expected}`);
			output.appendLine(`current profile: ${vals.currentApiConfigName || "(unset)"}`);
			output.appendLine(`apiProvider: ${vals.apiProvider || "(unset)"}`);
			output.appendLine(`openAiBaseUrl: ${vals.openAiBaseUrl || "(unset)"}`);
			output.appendLine(`openAiModelId: ${vals.openAiModelId || "(unset)"}`);
			output.appendLine(`${HEADER}: ${vals.openAiHeaders?.[HEADER] || AUTO_ROUTING}`);
			output.appendLine(`health: ${health.ok ? "OK" : "--"} ${health.status || health.error || ""}`);
			output.appendLine(`ready: ${ready.ok ? "OK" : "--"} ${ready.status || ready.error || ""}`);
			output.appendLine("주의: /health, /ready 점검은 모델 호출이 아니므로 사용량 집계를 증가시키지 않습니다.");
			output.appendLine("================================\n");
			writeAudit(host, "proxy", "checkVibeCodersProxy", {
				projectPath: project,
				expectedBaseUrl: expected,
				currentProfile: vals.currentApiConfigName || "",
				baseUrl: vals.openAiBaseUrl || "",
				model: vals.openAiModelId || "",
				health: health.ok ? String(health.status || "OK") : String(health.error || "error"),
				ready: ready.ok ? String(ready.status || "OK") : String(ready.error || "error"),
			});
			if (routed) void vscode.window.showInformationMessage("Vibe Code가 vibe-coders proxy를 향하도록 설정되어 있습니다.");
			else void vscode.window.showWarningMessage("현재 provider profile이 vibe-coders proxy를 향하지 않습니다. 'vibe-coders 프록시 적용' 명령을 실행하세요.");
		}),
		vscode.commands.registerCommand("vibe-code.writeVibeCodersProxyConfig", async () => {
			const ws = workspaceRoot();
			if (!ws) {
				void vscode.window.showWarningMessage(NO_WORKSPACE_MESSAGE);
				return;
			}
			const dir = path.join(ws, ".vibe-code");
			fs.mkdirSync(dir, { recursive: true });
			const file = path.join(dir, "vibe-coders-proxy.json");
			const body = {
				name: "vibe-coders",
				kind: "openai-compatible-proxy",
				projectPath: projectPath(host),
				baseUrl: baseUrl(),
				profileName: profileName(),
				defaultModel: defaultModel(),
				providerHeader: providerHeader() || null,
				applyCommand: "Vibe Code: vibe-coders 프록시 적용",
				checkCommand: "Vibe Code: vibe-coders 프록시 점검",
				effect: "Vibe Code의 OpenAI-compatible 모델 호출이 이 base URL을 지나가며 vibe-coders가 자동으로 사용량/토큰/비용을 집계합니다.",
				secretHandling: "proxy API key는 설정 파일에 기록하지 않고 적용 명령에서 Vibe Code 내부 provider 저장 흐름에 맡깁니다.",
				generatedAt: new Date().toISOString(),
			};
			fs.writeFileSync(file, JSON.stringify(body, null, 2), "utf8");
			await vscode.env.clipboard.writeText(baseUrl());
			await openFile(file);
			void vscode.window.showInformationMessage("vibe-coders proxy 설정 파일을 생성했고 base URL을 클립보드에 복사했습니다.");
			writeAudit(host, "proxy", "writeVibeCodersProxyConfig", { file: ".vibe-code/vibe-coders-proxy.json", baseUrl: baseUrl(), profileName: profileName(), defaultModel: defaultModel(), providerHeader: providerHeader() });
		}),
	);

	void refresh();
	log(host, "proxy status bar item created");
	log(host, "vibe-coders proxy commands registered");
}
