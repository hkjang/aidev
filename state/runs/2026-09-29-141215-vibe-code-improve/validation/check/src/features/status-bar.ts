import * as https from "node:https";
import * as vscode from "vscode";
import type { CoreHost } from "../core/host";
import { log } from "../log";
import { config } from "../settings";

const MODE_LABELS: Record<string, string> = { code: "Code", architect: "Architect", debug: "Debug", ask: "Ask", "code-reviewer": "Review" };

type NetState = "checking" | "online" | "offline";

function checkNetwork(): Promise<NetState> {
	return new Promise((resolve) => {
		try {
			const req = https.request({ host: "1.1.1.1", port: 443, method: "HEAD", timeout: 2000, path: "/" }, () => {
				resolve("online");
				req.destroy();
			});
			req.on("error", () => resolve("offline"));
			req.on("timeout", () => {
				resolve("offline");
				req.destroy();
			});
			req.end();
		} catch {
			resolve("offline");
		}
	});
}

/** Right-side status bar item: current mode, session tokens, and online/offline state. */
export function createModeStatusBar(host: CoreHost): void {
	const { context } = host;
	const item = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
	item.command = "vibe-code.SidebarProvider.focus";
	let netState: NetState = "checking";

	const refreshText = () => {
		try {
			const mode = context.globalState.get<string>("mode") || "code";
			const label = MODE_LABELS[mode] || mode;
			const tokens = context.globalState.get<number>("vibeCode.sessionTokens") || 0;
			const tokenText = tokens > 1000 ? ` · ${(tokens / 1000).toFixed(1)}k` : tokens > 0 ? ` · ${tokens}` : "";
			const icon = netState === "online" ? "$(cloud)" : "$(cloud-offline)";
			item.text = `${icon} Vibe: ${label}${tokenText}`;
			item.tooltip =
				`Vibe Code v${host.pkg.version} — ${label} 모드 / 네트워크: ${netState === "online" ? "온라인" : "오프라인"}` +
				` / 자율성: ${context.globalState.get<string>("vibeCode.autonomyApplied") || "(default)"}. 클릭하여 사이드바 열기.`;
		} catch {
			item.text = "$(rocket) Vibe Code";
		}
	};

	// `offlineMode` decides whether we probe the network at all: "offline"/"online" are
	// asserted from settings, "auto" probes once a minute (not every text refresh).
	const refreshNetwork = async () => {
		const mode = config().get<string>("offlineMode") || "auto";
		const previous = netState;
		netState = mode === "offline" ? "offline" : mode === "online" ? "online" : await checkNetwork();
		if (previous !== netState) log(host, `network state: ${netState}`);
		refreshText();
	};

	void refreshNetwork();
	item.show();
	const textInterval = setInterval(refreshText, 15_000);
	const networkInterval = setInterval(() => void refreshNetwork(), 60_000);
	context.subscriptions.push(
		item,
		{ dispose: () => clearInterval(textInterval) },
		{ dispose: () => clearInterval(networkInterval) },
		vscode.workspace.onDidChangeConfiguration((event) => {
			if (event.affectsConfiguration("vibe-code.offlineMode")) void refreshNetwork();
		}),
	);
	log(host, "status bar item created");
}
