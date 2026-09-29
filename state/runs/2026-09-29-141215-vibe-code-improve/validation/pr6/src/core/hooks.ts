import type { VibeCodeHooks } from "./host";
import { runActivation } from "../activation";
import { mergeLocaleOverrides } from "../features/i18n-overrides";
import { handleCommandEvent } from "../features/command-audit";
import { handleTaskEvent, promptContext } from "../features/goal-loop";

/** Must run before the core bundle is required: the core reads `globalThis.__vibeCode` at load time. */
export function installHooks(): VibeCodeHooks {
	const hooks: VibeCodeHooks = {
		beforeCore: runActivation,
		mergeLocaleOverrides,
		onCommand: handleCommandEvent,
		onTask: handleTaskEvent,
		promptContext,
	};
	globalThis.__vibeCode = hooks;
	return hooks;
}
