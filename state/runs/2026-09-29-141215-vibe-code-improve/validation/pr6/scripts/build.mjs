// Builds dist/extension.js from src/ and stages the vendored core bundle next to it.
//
//   dist/extension.js       <- esbuild bundle of src/extension.ts (this repo's own code)
//   dist/extension.core.js  <- copy of vendor/extension.core.js (upstream Roo-Code core + hooks)
//
// dist/ is never wiped: dist/node_modules, dist/i18n, *.wasm and dist/workers are runtime
// assets that ship with the VSIX and are not produced by this build.
import { build, context } from "esbuild";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const watch = process.argv.includes("--watch");
const distDir = path.join(root, "dist");
const vendorCore = path.join(root, "vendor", "extension.core.js");
const distCore = path.join(distDir, "extension.core.js");

fs.mkdirSync(distDir, { recursive: true });

if (!fs.existsSync(vendorCore)) {
	console.error(`vendor core bundle not found: ${vendorCore}`);
	process.exit(1);
}

function stageCore() {
	const src = fs.statSync(vendorCore);
	const same = fs.existsSync(distCore) && fs.statSync(distCore).size === src.size && fs.statSync(distCore).mtimeMs >= src.mtimeMs;
	if (!same) {
		fs.copyFileSync(vendorCore, distCore);
		console.log(`staged ${path.relative(root, distCore)} (${(src.size / 1024 / 1024).toFixed(1)} MB)`);
	}
}

function warnMissingRuntimeAssets() {
	const required = ["node_modules", "i18n", "workers", "tree-sitter.wasm", "tiktoken_bg.wasm"];
	const missing = required.filter((name) => !fs.existsSync(path.join(distDir, name)));
	if (missing.length > 0) {
		console.warn(`warning: dist/ is missing runtime assets: ${missing.join(", ")}`);
		console.warn("         copy them from an existing release VSIX (extension/dist/) before packaging.");
	}
}

const options = {
	entryPoints: [path.join(root, "src", "extension.ts")],
	outfile: path.join(distDir, "extension.js"),
	bundle: true,
	platform: "node",
	target: "node20",
	format: "cjs",
	// vscode is provided by the extension host; adm-zip is optional at runtime;
	// the core bundle is loaded from the same directory at runtime.
	external: ["vscode", "adm-zip", "./extension.core.js"],
	sourcemap: watch ? "inline" : false,
	legalComments: "none",
	logLevel: "info",
	banner: {
		js: "/* vibe-code: built from src/ by scripts/build.mjs. Core runtime lives in ./extension.core.js (see vendor/PATCHES.md). */",
	},
};

stageCore();
warnMissingRuntimeAssets();

if (watch) {
	const ctx = await context(options);
	await ctx.watch();
	console.log("watching src/ ...");
} else {
	await build(options);
}
