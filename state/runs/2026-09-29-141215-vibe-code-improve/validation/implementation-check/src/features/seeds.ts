import * as fs from "node:fs";
import * as path from "node:path";
import type { CoreHost } from "../core/host";
import { log } from "../log";
import { workspaceRoot } from "./workspace";

function asset(host: CoreHost, ...parts: string[]): string {
	return path.join(host.context.extensionPath, "assets", "demo", ...parts);
}

/** Copy the Korean demo scenarios into the workspace once per installation. */
export async function seedDemoScenarios(host: CoreHost): Promise<void> {
	if (host.context.globalState.get("vibeCode.demoSeeded")) return;
	const src = asset(host, "scenarios.md");
	if (!fs.existsSync(src)) return;
	const ws = workspaceRoot();
	if (!ws) return;
	const destDir = path.join(ws, ".vibe-code");
	const dest = path.join(destDir, "demo-scenarios.md");
	if (!fs.existsSync(dest)) {
		fs.mkdirSync(destDir, { recursive: true });
		fs.copyFileSync(src, dest);
		log(host, `demo scenarios seeded: ${dest}`);
	}
	await host.context.globalState.update("vibeCode.demoSeeded", true);
}

function seedTemplate(host: CoreHost, fileName: string, templateName: string, message: string): void {
	const ws = workspaceRoot();
	if (!ws) return;
	const dest = path.join(ws, fileName);
	if (fs.existsSync(dest)) return;
	const template = asset(host, templateName);
	if (!fs.existsSync(template)) return;
	fs.copyFileSync(template, dest);
	log(host, message);
}

export function seedVibeignore(host: CoreHost): void {
	seedTemplate(host, ".vibeignore", ".vibeignore.template", "seeded .vibeignore at workspace root");
}

export function seedVibemodes(host: CoreHost): void {
	seedTemplate(host, ".vibemodes", ".vibemodes.template", "seeded .vibemodes with 4 Korean modes");
}

/**
 * Seed the Korean slash command templates into `.vibe/commands/` without overwriting user edits.
 * The legacy `/목표` command is migrated to `/goal` (or removed when `/goal` already exists).
 */
export function seedSlashCommands(host: CoreHost): void {
	const ws = workspaceRoot();
	if (!ws) return;
	const src = asset(host, ".vibe", "commands");
	if (!fs.existsSync(src)) return;
	const dest = path.join(ws, ".vibe", "commands");
	fs.mkdirSync(dest, { recursive: true });

	const legacy = path.join(dest, "목표.md");
	const goal = path.join(dest, "goal.md");
	if (fs.existsSync(legacy)) {
		if (!fs.existsSync(goal)) {
			fs.renameSync(legacy, goal);
			log(host, `migrated legacy slash command: ${legacy} -> ${goal}`);
		} else {
			fs.unlinkSync(legacy);
			log(host, `removed legacy slash command: ${legacy}`);
		}
	}

	let copied = 0;
	for (const name of fs.readdirSync(src)) {
		if (!name.endsWith(".md")) continue;
		const target = path.join(dest, name);
		if (fs.existsSync(target)) continue;
		fs.copyFileSync(path.join(src, name), target);
		copied++;
	}
	if (copied > 0) log(host, `seeded ${copied} Korean slash command files at .vibe/commands/`);
}

export function seedMcpRecommendations(host: CoreHost): void {
	const ws = workspaceRoot();
	if (!ws) return;
	const dest = path.join(ws, ".vibe-code", "mcp-recommendations.json");
	if (fs.existsSync(dest)) return;
	const src = asset(host, "mcp-recommendations.json");
	if (!fs.existsSync(src)) return;
	fs.mkdirSync(path.dirname(dest), { recursive: true });
	fs.copyFileSync(src, dest);
	log(host, `seeded MCP recommendations: ${dest}`);
}
