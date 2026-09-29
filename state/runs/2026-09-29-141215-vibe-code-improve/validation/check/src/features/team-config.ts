import { exec } from "node:child_process";
import * as fs from "node:fs";
import * as path from "node:path";
import * as vscode from "vscode";
import type { CoreHost } from "../core/host";
import { log } from "../log";
import { NO_WORKSPACE_MESSAGE, workspaceRoot } from "./workspace";

/** Optional runtime dependency; falls back to PowerShell when absent. */
function loadAdmZip(): any {
	// eslint-disable-next-line @typescript-eslint/no-require-imports
	return require("adm-zip");
}

const quote = (value: string): string => '\\"' + value + '\\"';

/** Export/import the team-shareable workspace files (.vibemodes, .vibeignore, slash commands, MCP recommendations). */
export function registerTeamConfig(host: CoreHost): void {
	const { context } = host;
	context.subscriptions.push(
		vscode.commands.registerCommand("vibe-code.exportTeamConfig", async () => {
			const ws = workspaceRoot();
			if (!ws) {
				void vscode.window.showWarningMessage(NO_WORKSPACE_MESSAGE);
				return;
			}
			const dest = path.join(ws, ".vibe-code", "team-config.zip");
			const candidates = [
				path.join(ws, ".vibemodes"),
				path.join(ws, ".vibeignore"),
				path.join(ws, ".vibe", "commands"),
				path.join(ws, ".vibe-code", "mcp-recommendations.json"),
			];
			const present = candidates.filter((p) => fs.existsSync(p));
			if (present.length === 0) {
				void vscode.window.showInformationMessage("내보낼 팀 설정 파일이 없습니다.");
				return;
			}
			try {
				const AdmZip = loadAdmZip();
				const zip = new AdmZip();
				for (const p of present) {
					const rel = path.relative(ws, p);
					if (fs.statSync(p).isDirectory()) zip.addLocalFolder(p, rel);
					else zip.addLocalFile(p, path.dirname(rel));
				}
				zip.writeZip(dest);
				void vscode.window.showInformationMessage(`팀 설정을 내보냈습니다: ${dest}`);
				log(host, `team config exported (${present.length} items) to ${dest}`);
			} catch {
				try {
					const command = `powershell -NoProfile -Command "Compress-Archive -Path ${present.map(quote).join(",")} -DestinationPath ${quote(dest)} -Force"`;
					exec(command, (error) => {
						if (error) void vscode.window.showErrorMessage(`팀 설정 내보내기 실패: ${error.message}`);
						else {
							void vscode.window.showInformationMessage(`팀 설정을 내보냈습니다: ${dest}`);
							log(host, `team config exported via PowerShell to ${dest}`);
						}
					});
				} catch (error) {
					void vscode.window.showErrorMessage(`팀 설정 내보내기 실패: ${error}`);
				}
			}
		}),
		vscode.commands.registerCommand("vibe-code.importTeamConfig", async () => {
			const ws = workspaceRoot();
			if (!ws) {
				void vscode.window.showWarningMessage(NO_WORKSPACE_MESSAGE);
				return;
			}
			const picked = await vscode.window.showOpenDialog({
				canSelectFiles: true,
				canSelectFolders: false,
				canSelectMany: false,
				filters: { "팀 설정": ["zip"] },
				title: "팀 설정 파일 선택",
			});
			if (!picked || picked.length === 0) return;
			const src = picked[0].fsPath;
			try {
				const AdmZip = loadAdmZip();
				new AdmZip(src).extractAllTo(ws, true);
				void vscode.window.showInformationMessage("팀 설정을 가져왔습니다. 워크스페이스를 다시 로드하세요.", "다시 로드").then((choice) => {
					if (choice === "다시 로드") void vscode.commands.executeCommand("workbench.action.reloadWindow");
				});
			} catch {
				exec(`powershell -NoProfile -Command "Expand-Archive -LiteralPath ${quote(src)} -DestinationPath ${quote(ws)} -Force"`, (error) => {
					if (error) void vscode.window.showErrorMessage(`가져오기 실패: ${error.message}`);
					else void vscode.window.showInformationMessage("팀 설정을 가져왔습니다.");
				});
			}
		}),
	);
	log(host, "team config export/import commands registered");
}
