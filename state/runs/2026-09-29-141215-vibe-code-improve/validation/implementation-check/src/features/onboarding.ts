import * as vscode from "vscode";
import type { CoreHost } from "../core/host";
import { log } from "../log";
import { config } from "../settings";
import { AUTONOMY_PRESETS, writeAutonomyPreset } from "./defaults";

interface Choice<T extends string> extends vscode.QuickPickItem {
	value: T;
}

async function pick<T extends string>(title: string, step: number, items: Choice<T>[]): Promise<T | undefined> {
	const picked = await vscode.window.showQuickPick(items, { title: `Vibe Code 시작 설정 (${step}/3) — ${title}`, ignoreFocusOut: true });
	return picked?.value;
}

type ProviderChoice = "vibe-coders" | "cloud" | "local" | "later";
type NetworkChoice = "auto" | "online" | "offline";
type AutonomyChoice = keyof typeof AUTONOMY_PRESETS;

/**
 * `Vibe Code: 시작 설정` — three QuickPick steps (provider, network, autonomy) that write the
 * relevant settings and apply the autonomy preset immediately. Escaping keeps earlier steps.
 */
export function registerOnboarding(host: CoreHost): void {
	const { context, output } = host;
	context.subscriptions.push(
		vscode.commands.registerCommand("vibe-code.setupWizard", async () => {
			const cfg = config();
			const global = vscode.ConfigurationTarget.Global;
			const done: string[] = [];

			const provider = await pick<ProviderChoice>("모델 프로바이더", 1, [
				{ label: "$(plug) vibe-coders 프록시", description: "권장", detail: `OpenAI 호환 프록시 ${cfg.get("vibeCodersBaseUrl")}로 라우팅하고 사용량을 집계합니다.`, value: "vibe-coders" },
				{ label: "$(cloud) 클라우드 API 직접 연결", detail: "Anthropic, OpenAI, Google, Bedrock 등. 사이드바 설정에서 키를 입력합니다.", value: "cloud" },
				{ label: "$(vm) 로컬 모델", detail: "Ollama, LM Studio, vLLM 등 사내/로컬 OpenAI 호환 엔드포인트.", value: "local" },
				{ label: "$(clock) 나중에", value: "later" },
			]);
			if (provider === undefined) return;
			done.push(`프로바이더: ${provider}`);

			const network = await pick<NetworkChoice>("네트워크 환경", 2, [
				{ label: "$(cloud) 인터넷 사용", description: "online", detail: "외부 API와 npx MCP 서버를 그대로 사용합니다.", value: "online" },
				{ label: "$(cloud-offline) 폐쇄망", description: "offline", detail: "네트워크 점검을 하지 않고 로컬 소스만 우선합니다.", value: "offline" },
				{ label: "$(sync) 자동 감지", description: "auto", detail: "1분마다 연결 상태를 확인해 상태바에 표시합니다.", value: "auto" },
			]);
			if (network === undefined) return finish();
			await cfg.update("offlineMode", network, global);
			done.push(`offlineMode: ${network}`);

			const autonomy = await pick<AutonomyChoice>("자율성 등급", 3, [
				{ label: "$(shield) safe", detail: "모든 도구 실행 전에 승인을 요청합니다.", value: "safe" },
				{ label: "$(eye) assist", description: "기본값", detail: "읽기와 모드 전환만 자동 승인, 쓰기/실행은 확인합니다.", value: "assist" },
				{ label: "$(rocket) auto", detail: "읽기/쓰기/브라우저/MCP 자동 승인, 명령 실행은 확인합니다.", value: "auto" },
				{ label: "$(zap) yolo", detail: "명령 실행까지 모두 자동 승인합니다. 격리된 환경에서만 권장.", value: "yolo" },
			]);
			if (autonomy === undefined) return finish();
			await cfg.update("autonomy", autonomy, global);
			await writeAutonomyPreset(host, autonomy, true);
			done.push(`autonomy: ${autonomy}`);

			await context.globalState.update("vibeCode.setupCompleted", new Date().toISOString());
			await finish();

			async function finish(): Promise<void> {
				if (done.length === 0) return;
				output.appendLine("\n=== Vibe Code 시작 설정 ===");
				for (const line of done) output.appendLine(`  ${line}`);
				output.appendLine("================================\n");
				log(host, `setup wizard: ${done.join(", ")}`);
				if (provider === "vibe-coders") {
					await vscode.commands.executeCommand("vibe-code.applyVibeCodersProxy");
				} else if (provider === "cloud" || provider === "local") {
					const open = await vscode.window.showInformationMessage(
						provider === "cloud" ? "사이드바 설정에서 프로바이더와 API 키를 입력하세요." : "사이드바 설정에서 로컬 엔드포인트(Ollama, LM Studio, OpenAI 호환)를 선택하세요.",
						"설정 열기",
					);
					if (open === "설정 열기") await vscode.commands.executeCommand("vibe-code.settingsButtonClicked");
				}
				const next = await vscode.window.showInformationMessage(`시작 설정을 저장했습니다: ${done.join(" · ")}`, "사이드바 열기", "환경 점검");
				if (next === "사이드바 열기") await vscode.commands.executeCommand("vibe-code.SidebarProvider.focus");
				else if (next === "환경 점검") await vscode.commands.executeCommand("vibe-code.healthCheck");
			}
		}),
	);
	log(host, "setup wizard command registered");
}
