import * as fs from "node:fs";
import * as path from "node:path";
import * as vscode from "vscode";
import type { CoreHost } from "../core/host";
import { log } from "../log";
import { openFile } from "./workspace";

/** One-time welcome toast in the active language. */
export async function showWelcome(host: CoreHost): Promise<void> {
	const { context } = host;
	if (context.globalState.get("vibeCode.welcomed")) return;
	await context.globalState.update("vibeCode.welcomed", true);
	const lang = context.globalState.get<string>("language") || "ko";
	if (lang === "ko") {
		void vscode.window
			.showInformationMessage("Vibe Code 에 오신 것을 환영합니다! 한국어 환경으로 설정되어 있습니다.", "시작 설정", "사이드바 열기", "사용 가이드", "나중에")
			.then((choice) => {
				if (choice === "시작 설정") void vscode.commands.executeCommand("vibe-code.setupWizard");
				else if (choice === "사이드바 열기") void vscode.commands.executeCommand("vibe-code.SidebarProvider.focus");
				else if (choice === "사용 가이드") {
					const readme = path.join(context.extensionPath, "readme.ko.md");
					if (fs.existsSync(readme)) void openFile(readme);
				}
			});
	} else {
		void vscode.window.showInformationMessage("Welcome to Vibe Code!", "Setup", "Open Sidebar", "Later").then((choice) => {
			if (choice === "Setup") void vscode.commands.executeCommand("vibe-code.setupWizard");
			else if (choice === "Open Sidebar") void vscode.commands.executeCommand("vibe-code.SidebarProvider.focus");
		});
	}
	log(host, `welcome shown (lang: ${lang})`);
}
