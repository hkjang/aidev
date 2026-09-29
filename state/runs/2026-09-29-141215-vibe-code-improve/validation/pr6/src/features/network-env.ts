import * as fs from "node:fs";
import type { CoreHost } from "../core/host";
import { log } from "../log";
import { config } from "../settings";

/** Apply `vibe-code.proxyUrl` and `vibe-code.extraCaCertsPath` to the extension host process. */
export function applyNetworkEnv(host: CoreHost): void {
	const cfg = config();
	const proxy = cfg.get<unknown>("proxyUrl");
	if (proxy && typeof proxy === "string") {
		process.env.HTTPS_PROXY = process.env.HTTPS_PROXY || proxy;
		process.env.HTTP_PROXY = process.env.HTTP_PROXY || proxy;
		process.env.https_proxy = process.env.https_proxy || proxy;
		process.env.http_proxy = process.env.http_proxy || proxy;
		log(host, `proxy applied: ${proxy}`);
	}
	const ca = cfg.get<unknown>("extraCaCertsPath");
	if (ca && typeof ca === "string") {
		if (fs.existsSync(ca)) {
			process.env.NODE_EXTRA_CA_CERTS = ca;
			log(host, `NODE_EXTRA_CA_CERTS applied: ${ca}`);
		} else {
			log(host, `CA path not found: ${ca}`);
		}
	}
}
