import type { CoreHost } from "../core/host";
import { log } from "../log";
import { config } from "../settings";

/** Telemetry is opt-in: unset installs start disabled. */
export async function applyTelemetryDefault(host: CoreHost): Promise<void> {
	const current = host.context.globalState.get<string>("telemetrySetting");
	if (!current || current === "unset") {
		await host.context.globalState.update("telemetrySetting", "disabled");
		log(host, "telemetry disabled by default (privacy-first)");
	}
}

/** First run starts in the think-first `architect` mode. */
export async function applyDefaultMode(host: CoreHost): Promise<void> {
	if (!host.context.globalState.get("mode")) {
		await host.context.globalState.update("mode", "architect");
		log(host, "default mode set to architect (think-first workflow)");
	}
}

type AutoApproveKey =
	| "alwaysAllowReadOnly"
	| "alwaysAllowWrite"
	| "alwaysAllowExecute"
	| "alwaysAllowBrowser"
	| "alwaysAllowMcp"
	| "alwaysAllowModeSwitch"
	| "alwaysAllowSubtasks"
	| "alwaysAllowUpdateTodoList";

export const AUTONOMY_PRESETS: Record<"safe" | "assist" | "auto" | "yolo", Record<AutoApproveKey, boolean>> = {
	safe: {
		alwaysAllowReadOnly: false,
		alwaysAllowWrite: false,
		alwaysAllowExecute: false,
		alwaysAllowBrowser: false,
		alwaysAllowMcp: false,
		alwaysAllowModeSwitch: false,
		alwaysAllowSubtasks: false,
		alwaysAllowUpdateTodoList: false,
	},
	assist: {
		alwaysAllowReadOnly: true,
		alwaysAllowWrite: false,
		alwaysAllowExecute: false,
		alwaysAllowBrowser: false,
		alwaysAllowMcp: false,
		alwaysAllowModeSwitch: true,
		alwaysAllowSubtasks: false,
		alwaysAllowUpdateTodoList: true,
	},
	auto: {
		alwaysAllowReadOnly: true,
		alwaysAllowWrite: true,
		alwaysAllowExecute: false,
		alwaysAllowBrowser: true,
		alwaysAllowMcp: true,
		alwaysAllowModeSwitch: true,
		alwaysAllowSubtasks: true,
		alwaysAllowUpdateTodoList: true,
	},
	yolo: {
		alwaysAllowReadOnly: true,
		alwaysAllowWrite: true,
		alwaysAllowExecute: true,
		alwaysAllowBrowser: true,
		alwaysAllowMcp: true,
		alwaysAllowModeSwitch: true,
		alwaysAllowSubtasks: true,
		alwaysAllowUpdateTodoList: true,
	},
};

/**
 * Write a preset's auto-approve flags into globalState. With `overwrite` false only keys the
 * user never set are filled (first-run behaviour); the setup wizard passes true.
 */
export async function writeAutonomyPreset(host: CoreHost, autonomy: keyof typeof AUTONOMY_PRESETS, overwrite: boolean): Promise<void> {
	const preset = AUTONOMY_PRESETS[autonomy];
	for (const key of Object.keys(preset) as AutoApproveKey[]) {
		if (overwrite || host.context.globalState.get(key) === undefined) {
			await host.context.globalState.update(key, preset[key]);
		}
	}
	await host.context.globalState.update("vibeCode.autonomyApplied", autonomy);
	log(host, `autonomy preset applied: ${autonomy}`);
}

/** Apply the `vibe-code.autonomy` preset once, only filling auto-approve keys the user never set. */
export async function applyAutonomyPreset(host: CoreHost): Promise<void> {
	const autonomy = config().get<string>("autonomy");
	if (!autonomy || host.context.globalState.get("vibeCode.autonomyApplied")) return;
	if (!(autonomy in AUTONOMY_PRESETS)) return;
	await writeAutonomyPreset(host, autonomy as keyof typeof AUTONOMY_PRESETS, false);
}
