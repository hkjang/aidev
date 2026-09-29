"use strict";

const fs = require("fs");
const os = require("os");
const path = require("path");

async function main() {
	let runTests;
	try {
		({ runTests } = require("@vscode/test-electron"));
	} catch (error) {
		console.error("@vscode/test-electron is not installed.");
		console.error("Run the PowerShell wrapper for the installed-VS-Code fallback:");
		console.error("  powershell -ExecutionPolicy Bypass -File scripts/test-extension-host.ps1");
		process.exit(1);
	}

	const root = path.resolve(__dirname, "..");
	const extensionDevelopmentPath = root;
	const extensionTestsPath = path.join(root, "tests", "extension-host", "index.js");
	const tempRoot = process.env.VIBE_CODE_EXTENSION_HOST_TEMP || path.join(os.tmpdir(), "vibe-code-extension-host-test-electron");
	const workspacePath = path.join(tempRoot, "workspace");

	fs.rmSync(tempRoot, { recursive: true, force: true });
	fs.mkdirSync(workspacePath, { recursive: true });

	await runTests({
		extensionDevelopmentPath,
		extensionTestsPath,
		launchArgs: [workspacePath, "--disable-extensions", "--disable-gpu", "--log", "error"],
		extensionTestsEnv: {
			VIBE_CODE_EXTENSION_HOST_TEST: "1",
		},
	});
}

main().catch((error) => {
	console.error(error);
	process.exit(1);
});
