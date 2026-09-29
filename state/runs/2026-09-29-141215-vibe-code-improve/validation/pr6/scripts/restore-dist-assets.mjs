// Restores the runtime assets that ship inside a release VSIX (extension/dist/**) into dist/.
//
//   node scripts/restore-dist-assets.mjs                 # newest release/vibe-code-*.vsix
//   node scripts/restore-dist-assets.mjs --from x.vsix   # explicit VSIX
//
// dist/extension.js and dist/extension.core.js are build outputs and are never taken from
// the VSIX; everything else under extension/dist/ (node_modules, i18n, workers, *.wasm) is.
// Zero dependencies: a VSIX is a plain zip (store/deflate), parsed here directly.
import fs from "node:fs";
import path from "node:path";
import zlib from "node:zlib";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const distDir = path.join(root, "dist");
const BUILD_OUTPUTS = new Set(["extension/dist/extension.js", "extension/dist/extension.core.js"]);

function newestReleaseVsix() {
	const dir = path.join(root, "release");
	if (!fs.existsSync(dir)) return undefined;
	const files = fs
		.readdirSync(dir)
		.filter((name) => /^vibe-code-\d+\.\d+\.\d+\.vsix$/.test(name))
		.sort((a, b) => {
			const va = a.match(/(\d+)\.(\d+)\.(\d+)/).slice(1).map(Number);
			const vb = b.match(/(\d+)\.(\d+)\.(\d+)/).slice(1).map(Number);
			for (let i = 0; i < 3; i++) if (va[i] !== vb[i]) return va[i] - vb[i];
			return 0;
		});
	return files.length ? path.join(dir, files[files.length - 1]) : undefined;
}

function readCentralDirectory(buf) {
	let eocd = -1;
	for (let i = buf.length - 22; i >= Math.max(0, buf.length - 65557); i--) {
		if (buf.readUInt32LE(i) === 0x06054b50) {
			eocd = i;
			break;
		}
	}
	if (eocd < 0) throw new Error("not a zip archive (no end-of-central-directory record)");
	const count = buf.readUInt16LE(eocd + 10);
	const cdOffset = buf.readUInt32LE(eocd + 16);
	if (count === 0xffff || cdOffset === 0xffffffff) throw new Error("zip64 archives are not supported");
	const entries = [];
	let p = cdOffset;
	for (let n = 0; n < count; n++) {
		if (buf.readUInt32LE(p) !== 0x02014b50) throw new Error(`bad central directory entry at ${p}`);
		const method = buf.readUInt16LE(p + 10);
		const compSize = buf.readUInt32LE(p + 20);
		const size = buf.readUInt32LE(p + 24);
		const nameLen = buf.readUInt16LE(p + 28);
		const extraLen = buf.readUInt16LE(p + 30);
		const commentLen = buf.readUInt16LE(p + 32);
		const localOffset = buf.readUInt32LE(p + 42);
		const name = buf.toString("utf8", p + 46, p + 46 + nameLen).replace(/\\/g, "/");
		entries.push({ name, method, compSize, size, localOffset });
		p += 46 + nameLen + extraLen + commentLen;
	}
	return entries;
}

function entryData(buf, entry) {
	const lh = entry.localOffset;
	if (buf.readUInt32LE(lh) !== 0x04034b50) throw new Error(`bad local header for ${entry.name}`);
	const nameLen = buf.readUInt16LE(lh + 26);
	const extraLen = buf.readUInt16LE(lh + 28);
	const start = lh + 30 + nameLen + extraLen;
	const raw = buf.subarray(start, start + entry.compSize);
	if (entry.method === 0) return raw;
	if (entry.method === 8) return zlib.inflateRawSync(raw);
	throw new Error(`${entry.name}: unsupported compression method ${entry.method}`);
}

const args = process.argv.slice(2);
const fromIndex = args.indexOf("--from");
const vsix = fromIndex >= 0 ? path.resolve(args[fromIndex + 1] || "") : newestReleaseVsix();
if (!vsix || !fs.existsSync(vsix)) {
	console.error("VSIX not found. Pass --from <path-to-vsix> or place a release VSIX under release/.");
	process.exit(1);
}

console.log(`reading ${path.relative(root, vsix) || vsix}`);
const buf = fs.readFileSync(vsix);
const entries = readCentralDirectory(buf);
let written = 0;
let bytes = 0;
for (const entry of entries) {
	if (!entry.name.startsWith("extension/dist/") || entry.name.endsWith("/")) continue;
	if (BUILD_OUTPUTS.has(entry.name)) continue;
	const rel = entry.name.slice("extension/dist/".length);
	if (rel.split("/").includes("..")) throw new Error(`refusing to extract ${entry.name}`);
	const target = path.join(distDir, rel);
	fs.mkdirSync(path.dirname(target), { recursive: true });
	const data = entryData(buf, entry);
	fs.writeFileSync(target, data);
	written++;
	bytes += data.length;
}
console.log(`restored ${written} files (${(bytes / 1024 / 1024).toFixed(1)} MB) into ${path.relative(root, distDir)}/`);
console.log("next: npm run build");
