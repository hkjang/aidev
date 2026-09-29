import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import type { LocaleResources } from "../core/host";

type Tree = Record<string, unknown>;

function isTree(value: unknown): value is Tree {
	return !!value && typeof value === "object" && !Array.isArray(value);
}

/** Deep-merge `patch` into `base` (arrays and scalars are replaced). */
export function deepMerge(base: Tree, patch: Tree): Tree {
	for (const key of Object.keys(patch)) {
		const next = patch[key];
		const current = base[key];
		if (isTree(next) && isTree(current)) deepMerge(current, next);
		else base[key] = next;
	}
	return base;
}

export function userLocaleDir(): string {
	return path.join(os.homedir(), ".vibe-code", "locales");
}

/**
 * Merge `~/.vibe-code/locales/<lang>/<namespace>.json` files over the bundled translations.
 * Called by the core bundle while it initializes i18next; errors are swallowed per file.
 */
export function mergeLocaleOverrides(resources: LocaleResources): void {
	const root = userLocaleDir();
	if (!fs.existsSync(root)) return;
	const languages = fs.readdirSync(root, { withFileTypes: true }).filter((entry) => entry.isDirectory());
	for (const language of languages) {
		const lang = language.name;
		const langDir = path.join(root, lang);
		resources[lang] = resources[lang] || {};
		for (const file of fs.readdirSync(langDir).filter((name) => name.endsWith(".json"))) {
			try {
				const ns = file.replace(/\.json$/, "");
				const data = JSON.parse(fs.readFileSync(path.join(langDir, file), "utf8")) as Tree;
				resources[lang][ns] = deepMerge(resources[lang][ns] || {}, data) as Record<string, unknown>;
				console.log(`[vibe-code] override loaded: ${lang}/${ns}`);
			} catch {
				// a malformed override file must not break translation loading
			}
		}
	}
}
