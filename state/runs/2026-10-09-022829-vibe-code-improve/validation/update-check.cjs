"use strict";
var __create = Object.create;
var __defProp = Object.defineProperty;
var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
var __getOwnPropNames = Object.getOwnPropertyNames;
var __getProtoOf = Object.getPrototypeOf;
var __hasOwnProp = Object.prototype.hasOwnProperty;
var __export = (target, all) => {
  for (var name in all)
    __defProp(target, name, { get: all[name], enumerable: true });
};
var __copyProps = (to, from, except, desc) => {
  if (from && typeof from === "object" || typeof from === "function") {
    for (let key of __getOwnPropNames(from))
      if (!__hasOwnProp.call(to, key) && key !== except)
        __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
  }
  return to;
};
var __toESM = (mod, isNodeMode, target) => (target = mod != null ? __create(__getProtoOf(mod)) : {}, __copyProps(
  // If the importer is in node compatibility mode or this is not an ESM
  // file that has been converted to a CommonJS file using a Babel-
  // compatible transform (i.e. "__esModule" has not been set), then set
  // "default" to the CommonJS "module.exports" for node compatibility.
  isNodeMode || !mod || !mod.__esModule ? __defProp(target, "default", { value: mod, enumerable: true }) : target,
  mod
));
var __toCommonJS = (mod) => __copyProps(__defProp({}, "__esModule", { value: true }), mod);

// src/features/update-check.ts
var update_check_exports = {};
__export(update_check_exports, {
  checkForUpdates: () => checkForUpdates,
  latestVsixVersion: () => latestVsixVersion
});
module.exports = __toCommonJS(update_check_exports);
var fs = __toESM(require("node:fs"));
var path = __toESM(require("node:path"));

// tests/unit/vscode-stub.ts
var window = {};
var workspace = { workspaceFolders: void 0 };
var commands = {};
var Uri = { file: (p) => ({ fsPath: p }) };

// src/log.ts
function log(host, message) {
  host.output.appendLine(`[vibe-code] ${message}`);
}

// src/settings.ts
var SECTION = "vibe-code";
function config() {
  return workspace.getConfiguration(SECTION);
}

// src/util/semver.ts
function compareVersions(a, b) {
  const ap = a.split(".").map(Number);
  const bp = b.split(".").map(Number);
  for (let i = 0; i < Math.max(ap.length, bp.length); i++) {
    const x = ap[i] || 0;
    const y = bp[i] || 0;
    if (x !== y) return x - y;
  }
  return 0;
}

// src/features/update-check.ts
var DAY_MS = 24 * 60 * 60 * 1e3;
function latestVsixVersion(dir) {
  const versions = fs.readdirSync(dir).map((name) => name.match(/^vibe-code-([0-9.]+)\.vsix$/)?.[1]).filter((v) => !!v).sort(compareVersions);
  return versions[versions.length - 1];
}
async function checkForUpdates(host) {
  const { context } = host;
  const lastCheck = context.globalState.get("vibeCode.lastUpdateCheck") || 0;
  if (Date.now() - lastCheck <= DAY_MS) return;
  await context.globalState.update("vibeCode.lastUpdateCheck", Date.now());
  const sources = [];
  const shared = config().get("updateSourcePath");
  if (typeof shared === "string" && shared.trim()) sources.push({ path: shared, label: "\uACF5\uC720 \uD3F4\uB354" });
  sources.push({ path: path.join(context.extensionPath, "..", "..", "..", "projects", "vibe-code", "release"), label: "\uB85C\uCEEC release" });
  for (const source of sources) {
    try {
      if (!fs.existsSync(source.path)) continue;
      const latest = latestVsixVersion(source.path);
      if (!latest || compareVersions(latest, host.pkg.version) <= 0) continue;
      const vsix = path.join(source.path, `vibe-code-${latest}.vsix`);
      void window.showInformationMessage(`Vibe Code \uC0C8 \uBC84\uC804 \uBC1C\uACAC (${source.label}): ${latest} \u2192 \uD604\uC7AC ${host.pkg.version}`, "\uC124\uCE58", "\uB098\uC911\uC5D0").then((choice) => {
        if (choice === "\uC124\uCE58") void commands.executeCommand("workbench.extensions.installExtension", Uri.file(vsix));
      });
      log(host, `update available from ${source.label}: ${latest}`);
      break;
    } catch (error) {
      log(host, `update source check failed (${source.label}): ${error}`);
    }
  }
}
// Annotate the CommonJS export names for ESM import in node:
0 && (module.exports = {
  checkForUpdates,
  latestVsixVersion
});
