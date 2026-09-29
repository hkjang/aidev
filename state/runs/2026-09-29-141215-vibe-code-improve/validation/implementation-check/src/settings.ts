import * as vscode from "vscode";

export const SECTION = "vibe-code";

export function config(): vscode.WorkspaceConfiguration {
	return vscode.workspace.getConfiguration(SECTION);
}

/** Trimmed string setting with a fallback for unset/blank values. */
export function stringSetting(key: string, fallback: string): string {
	const raw = config().get<unknown>(key);
	const value = typeof raw === "string" ? raw.trim() : "";
	return value || fallback;
}
