import type { CoreHost } from "../core/host";
import { log } from "../log";
import { config } from "../settings";

/** First run: copy `vibe-code.language` into globalState so the core UI starts in that language. */
export async function syncLanguageFromSettings(host: CoreHost): Promise<void> {
	const configured = config().get<unknown>("language");
	if (configured && typeof configured === "string" && !host.context.globalState.get("language")) {
		await host.context.globalState.update("language", configured);
		log(host, `language synced from settings: ${configured}`);
	}
}

/** Runs last: guarantees a globalState language (default `ko`) and switches the core i18n to it. */
export async function initializeLanguage(host: CoreHost): Promise<void> {
	const configured = config().get<string>("language") || "ko";
	const active = host.context.globalState.get<string>("language") ?? configured ?? "ko";
	if (!host.context.globalState.get("language")) {
		await host.context.globalState.update("language", active);
		log(host, "language default initialized: " + active);
	}
	host.changeLanguage(active);
}
