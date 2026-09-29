import type * as vscode from "vscode";

/** Package constants exposed by the core bundle (`Ls` in the minified code). */
export interface PackageInfo {
	publisher: string;
	name: string;
	version: string;
	outputChannel: string;
	sha?: string;
}

/** Subset of the core ContextProxy API used by this extension's own features. */
export interface ProviderValues {
	currentApiConfigName?: string;
	apiProvider?: string;
	openAiBaseUrl?: string;
	openAiModelId?: string;
	openAiHeaders?: Record<string, string>;
	[key: string]: unknown;
}

export interface ContextProxyLike {
	getValue(key: string): unknown;
	getValues(): ProviderValues;
	setValue(key: string, value: unknown): Promise<void>;
	setProviderSettings(values: Record<string, unknown>): Promise<void>;
}

export interface ContextProxyStatic {
	getInstance(context: vscode.ExtensionContext): Promise<ContextProxyLike>;
}

/** Subset of the core sidebar provider (ClineProvider) used for profile switching. */
export interface ProviderLike {
	providerSettingsManager?: {
		saveConfig(name: string, config: Record<string, unknown>): Promise<unknown>;
		hasConfig(name: string): Promise<boolean>;
	};
	activateProviderProfile?(args: { name: string }): Promise<unknown>;
	postStateToWebview?(): Promise<unknown>;
}

/**
 * What the core bundle hands to `beforeCore` at the start of its `activate`.
 * See vendor/PATCHES.md for the exact call site.
 */
export interface CoreHost {
	context: vscode.ExtensionContext;
	output: vscode.OutputChannel;
	pkg: PackageInfo;
	contextProxy: ContextProxyStatic;
	changeLanguage(language: string): void;
	/** Resolves to the sidebar provider once the core has created it; undefined before that. */
	getProvider(): ProviderLike | undefined;
}

/** i18next resource tree: language -> namespace -> keys. */
export type LocaleResources = Record<string, Record<string, Record<string, unknown>>>;

/** execute_command lifecycle as reported by the core (approval decision, then shell exit). */
export interface CommandEvent {
	phase: "approved" | "denied" | "exited";
	command: string;
	cwd?: string;
	exitCode?: number;
	executionId?: string;
	taskId?: string;
}

/** Task lifecycle as reported by the core (Task.initiateTaskLoop / ClineProvider listeners / condense). */
export interface TaskEvent {
	phase: "started" | "completed" | "aborted" | "condensed";
	taskId: string;
	tokenUsage?: Record<string, unknown>;
	toolUsage?: Record<string, unknown>;
	isSubtask?: boolean;
	cost?: number;
}

export interface VibeCodeHooks {
	/** Runs inside the core `activate`, right after the output channel exists and before the provider is created. */
	beforeCore(host: CoreHost): Promise<void>;
	/** Runs while the core loads its bundled translations; may mutate `resources` in place. */
	mergeLocaleOverrides(resources: LocaleResources): void;
	/** Fired by the execute_command tool: once on approval/denial and once when the shell exits. */
	onCommand(event: CommandEvent): void;
	/** Fired when a task starts, completes, aborts, or has its context condensed. */
	onTask(event: TaskEvent): void;
	/** Extra system-prompt text appended after the OBJECTIVE section on every request ("" for none). */
	promptContext(): string;
}

declare global {
	// eslint-disable-next-line no-var
	var __vibeCode: VibeCodeHooks | undefined;
}
