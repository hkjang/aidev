import type { CoreHost } from "./core/host";

export function log(host: CoreHost, message: string): void {
	host.output.appendLine(`[vibe-code] ${message}`);
}
