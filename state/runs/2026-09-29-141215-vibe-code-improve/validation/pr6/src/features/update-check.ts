import * as fs from "node:fs";
import * as path from "node:path";
import * as vscode from "vscode";
import type { CoreHost } from "../core/host";
import { log } from "../log";
import { config } from "../settings";
import { compareVersions } from "../util/semver";

const DAY_MS = 24 * 60 * 60 * 1000;

interface UpdateSource {
	path: string;
	label: string;
}

/** Newest `vibe-code-<version>.vsix` version in a directory, or undefined. */
export function latestVsixVersion(dir: string): string | undefined {
	const versions = fs
		.readdirSync(dir)
		.map((name) => name.match(/^vibe-code-([0-9.]+)\.vsix$/)?.[1])
		.filter((v): v is string => !!v)
		.sort(compareVersions);
	return versions[versions.length - 1];
}

/** Once a day, look for a newer VSIX in the shared folder setting or the local release folder. */
export async function checkForUpdates(host: CoreHost): Promise<void> {
	const { context } = host;
	const lastCheck = context.globalState.get<number>("vibeCode.lastUpdateCheck") || 0;
	if (Date.now() - lastCheck <= DAY_MS) return;
	await context.globalState.update("vibeCode.lastUpdateCheck", Date.now());

	const sources: UpdateSource[] = [];
	const shared = config().get<unknown>("updateSourcePath");
	if (typeof shared === "string" && shared.trim()) sources.push({ path: shared, label: "공유 폴더" });
	sources.push({ path: path.join(context.extensionPath, "..", "..", "..", "projects", "vibe-code", "release"), label: "로컬 release" });

	for (const source of sources) {
		try {
			if (!fs.existsSync(source.path)) continue;
			const latest = latestVsixVersion(source.path);
			if (!latest || compareVersions(latest, host.pkg.version) <= 0) continue;
			const vsix = path.join(source.path, `vibe-code-${latest}.vsix`);
			void vscode.window
				.showInformationMessage(`Vibe Code 새 버전 발견 (${source.label}): ${latest} → 현재 ${host.pkg.version}`, "설치", "나중에")
				.then((choice) => {
					if (choice === "설치") void vscode.commands.executeCommand("workbench.extensions.installExtension", vscode.Uri.file(vsix));
				});
			log(host, `update available from ${source.label}: ${latest}`);
			break;
		} catch (error) {
			log(host, `update source check failed (${source.label}): ${error}`);
		}
	}
}
