import path from "node:path";
import { defineConfig } from "vitest/config";

export default defineConfig({
	resolve: {
		// `vscode` only exists inside the extension host; unit tests use a small stub.
		alias: { vscode: path.resolve(__dirname, "tests/unit/vscode-stub.ts") },
	},
	test: {
		include: ["tests/unit/**/*.test.ts"],
	},
});
