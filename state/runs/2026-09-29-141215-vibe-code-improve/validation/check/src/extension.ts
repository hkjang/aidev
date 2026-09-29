import type * as vscode from "vscode";
import { installHooks } from "./core/hooks";
import { appendSessionSummary } from "./features/journal-summary";
import { handoffOnExit } from "./features/handoff-on-exit";

interface CoreModule {
	activate(context: vscode.ExtensionContext): Promise<unknown>;
	deactivate(): Promise<void>;
}

installHooks();

// The core bundle is staged next to this file by scripts/build.mjs and must be loaded
// after the hooks are installed, so a plain require (not a hoisted import) is used here.
// eslint-disable-next-line @typescript-eslint/no-require-imports
const core = require("./extension.core.js") as CoreModule;

export function activate(context: vscode.ExtensionContext): Promise<unknown> {
	return core.activate(context);
}

export function deactivate(): Promise<void> {
	// Summarize today's audit trail into the journal before the core tears down.
	// eslint-disable-next-line @typescript-eslint/no-require-imports
	const vscode = require("vscode") as typeof import("vscode");
	const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
	handoffOnExit(workspaceRoot);
	appendSessionSummary(workspaceRoot);
	return core.deactivate();
}
