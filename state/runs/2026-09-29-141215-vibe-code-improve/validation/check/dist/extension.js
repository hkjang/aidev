/* vibe-code: built from src/ by scripts/build.mjs. Core runtime lives in ./extension.core.js (see vendor/PATCHES.md). */
"use strict";
var __create = Object.create;
var __defProp = Object.defineProperty;
var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
var __getOwnPropNames = Object.getOwnPropertyNames;
var __getProtoOf = Object.getPrototypeOf;
var __hasOwnProp = Object.prototype.hasOwnProperty;
var __esm = (fn, res) => function __init() {
  return fn && (res = (0, fn[__getOwnPropNames(fn)[0]])(fn = 0)), res;
};
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

// src/log.ts
function log(host, message) {
  host.output.appendLine(`[vibe-code] ${message}`);
}
var init_log = __esm({
  "src/log.ts"() {
    "use strict";
  }
});

// src/settings.ts
function config() {
  return vscode.workspace.getConfiguration(SECTION);
}
function stringSetting(key, fallback) {
  const raw = config().get(key);
  const value = typeof raw === "string" ? raw.trim() : "";
  return value || fallback;
}
var vscode, SECTION;
var init_settings = __esm({
  "src/settings.ts"() {
    "use strict";
    vscode = __toESM(require("vscode"));
    SECTION = "vibe-code";
  }
});

// src/util/kst.ts
function kstParts(date = /* @__PURE__ */ new Date()) {
  const parts = {};
  for (const part of KST_FORMAT.formatToParts(date)) parts[part.type] = part.value;
  return {
    year: parts.year ?? "",
    month: parts.month ?? "",
    day: parts.day ?? "",
    hour: parts.hour ?? "",
    minute: parts.minute ?? "",
    second: parts.second ?? ""
  };
}
function kstDate(date = /* @__PURE__ */ new Date()) {
  const p = kstParts(date);
  return `${p.year}-${p.month}-${p.day}`;
}
function kstClock(date = /* @__PURE__ */ new Date()) {
  const p = kstParts(date);
  return `${p.hour}:${p.minute} KST`;
}
function kstStamp(date = /* @__PURE__ */ new Date()) {
  const p = kstParts(date);
  return {
    file: `${p.year}-${p.month}-${p.day}-${p.hour}${p.minute}${p.second}`,
    human: `${p.year}-${p.month}-${p.day} ${p.hour}:${p.minute}:${p.second} KST`
  };
}
var KST_FORMAT;
var init_kst = __esm({
  "src/util/kst.ts"() {
    "use strict";
    KST_FORMAT = new Intl.DateTimeFormat("ko-KR", {
      timeZone: "Asia/Seoul",
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: false
    });
  }
});

// src/util/markdown.ts
function detectEol(text) {
  const first = text.indexOf("\n");
  return first > 0 && text[first - 1] === "\r" ? "\r\n" : "\n";
}
function normalizeEol(text) {
  return text.includes("\r") ? text.replace(/\r\n/g, "\n") : text;
}
function restoreEol(normalized, eol) {
  return eol === "\r\n" ? normalized.replace(/\n/g, "\r\n") : normalized;
}
function sectionHeadRe(name) {
  return new RegExp("((?:^|\\n)## " + escapeRegExp(name) + "\\n)");
}
function hasSection(text, name) {
  return sectionHeadRe(name).test(normalizeEol(text));
}
function matchLine(text, label, fallback = "") {
  const match = normalizeEol(text).match(new RegExp("^" + escapeRegExp(label) + ":([^\\r\\n]*)$", "m"));
  return match?.[1]?.trim() || fallback;
}
function headingTitle(text, prefix, fallback) {
  const match = normalizeEol(text).match(new RegExp("^#\\s*" + escapeRegExp(prefix) + ":\\s*(.+)$", "m"));
  return (match?.[1] || fallback).trim();
}
function section(text, name) {
  const re = new RegExp("(?:^|\\n)## " + escapeRegExp(name) + "\\n([\\s\\S]*?)(?=\\n## |$)");
  return (normalizeEol(text).match(re)?.[1] || "").trim();
}
function subsection(text, name) {
  const re = new RegExp("(?:^|\\n)### " + escapeRegExp(name) + "\\n([\\s\\S]*?)(?=\\n### |\\n## |$)");
  return (normalizeEol(text).match(re)?.[1] || "").trim();
}
function writeSection(text, name, lines2) {
  const eol = detectEol(text);
  const body = normalizeEol(text);
  const re = new RegExp(sectionHeadRe(name).source + "([\\s\\S]*?)(?=\\n## |$)");
  if (!re.test(body)) return text;
  return restoreEol(
    body.replace(re, (_match, head) => head + (lines2.length > 0 ? lines2.join("\n") + "\n" : "\n")),
    eol
  );
}
function sectionLines(text, name) {
  const body = section(text, name);
  return body === "" ? [] : body.split("\n");
}
function lines(block) {
  return block.split(/\r?\n/).map((line) => line.trim()).filter(Boolean);
}
function taskLines(block) {
  return lines(block).filter((line) => line.startsWith("- ["));
}
function isDoneTask(line) {
  return /^- \[[xX]\]/.test(line);
}
function stripTask(line) {
  return line.replace(/^- \[(?: |x|X)\]\s*/, "");
}
function countChecks(text) {
  const checks = [...normalizeEol(text).matchAll(/^- \[( |x|X)\] /gm)];
  return { done: checks.filter((m) => m[1].toLowerCase() === "x").length, total: checks.length };
}
function progressLabel(count2, empty) {
  return count2.total > 0 ? `${count2.done}/${count2.total}` : empty;
}
function setLine(text, label, value) {
  const eol = detectEol(text);
  const body = normalizeEol(text);
  const re = new RegExp("^" + escapeRegExp(label) + ":[^\\r\\n]*$", "m");
  const line = label + ": " + value;
  const next = re.test(body) ? body.replace(re, () => line) : /^마지막 갱신:[^\r\n]*$/m.test(body) ? body.replace(/^마지막 갱신:[^\r\n]*$/m, (m) => m + "\n" + line) : body.replace(/^작성일:[^\r\n]*$/m, (m) => m + "\n" + line);
  return restoreEol(next, eol);
}
function touchPlan(text, status, stamp) {
  const eol = detectEol(text);
  let next = normalizeEol(text);
  if (status) {
    if (/^상태:[^\r\n]*$/m.test(next)) next = next.replace(/^상태:[^\r\n]*$/m, () => "\uC0C1\uD0DC: " + status);
    else next = next.replace(/^# .+$/m, (m) => m + "\n\n\uC0C1\uD0DC: " + status);
  }
  if (/^마지막 갱신:[^\r\n]*$/m.test(next)) next = next.replace(/^마지막 갱신:[^\r\n]*$/m, () => "\uB9C8\uC9C0\uB9C9 \uAC31\uC2E0: " + stamp);
  else if (/^작성일:[^\r\n]*$/m.test(next)) next = next.replace(/^작성일:[^\r\n]*$/m, (m) => m + "\n\uB9C8\uC9C0\uB9C9 \uAC31\uC2E0: " + stamp);
  return restoreEol(next, eol);
}
function priorityRank(priority) {
  const value = String(priority || "P2").trim().toUpperCase();
  return PRIORITY_RANK[value] ?? 9;
}
function sectionOfLine(text, lineIndex) {
  const all = text.split(/\r?\n/);
  for (let i = Math.min(lineIndex, all.length - 1); i >= 0; i--) {
    const m = all[i].match(/^## (.+)$/);
    if (m) return m[1].trim();
    if (/^# /.test(all[i])) return "";
  }
  return "";
}
function isTaskLine(line) {
  return /^- \[( |x|X)\] /.test(line);
}
function toggleCheckbox(text, lineIndex) {
  const eol = detectEol(text);
  const all = normalizeEol(text).split("\n");
  const line = all[lineIndex];
  if (line === void 0 || !isTaskLine(line)) return text;
  all[lineIndex] = isDoneTask(line) ? line.replace(/^- \[[xX]\]/, "- [ ]") : line.replace(/^- \[ \]/, "- [x]");
  return restoreEol(all.join("\n"), eol);
}
function moveTaskToSection(text, lineIndex, targetSection, transform = (l) => l) {
  const eol = detectEol(text);
  const all = normalizeEol(text).split("\n");
  const line = all[lineIndex];
  if (line === void 0 || !isTaskLine(line.trim())) return null;
  if (!hasSection(all.join("\n"), targetSection)) return null;
  all.splice(lineIndex, 1);
  const without = all.join("\n");
  return restoreEol(writeSection(without, targetSection, [...sectionLines(without, targetSection), transform(line.trim())]), eol);
}
var escapeRegExp, PRIORITY_RANK;
var init_markdown = __esm({
  "src/util/markdown.ts"() {
    "use strict";
    escapeRegExp = (value) => value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    PRIORITY_RANK = { P0: 0, P1: 1, P2: 2, P3: 3 };
  }
});

// src/features/workspace.ts
function workspaceRoot() {
  return vscode3.workspace.workspaceFolders?.[0]?.uri.fsPath;
}
function ensureWorkspacePaths(silent = false) {
  const ws = workspaceRoot();
  if (!ws) {
    if (!silent) void vscode3.window.showWarningMessage(NO_WORKSPACE_MESSAGE);
    return null;
  }
  const base = path.join(ws, ".vibe-code");
  const paths = {
    ws,
    base,
    goals: path.join(base, "goals"),
    sessions: path.join(base, "sessions"),
    checkpoints: path.join(base, "checkpoints"),
    plans: path.join(base, "plans"),
    archive: path.join(base, "plans", "archive"),
    journal: path.join(base, "journal"),
    audit: path.join(base, "audit"),
    current: path.join(base, "goals", "current.md")
  };
  for (const dir of [paths.goals, paths.sessions, paths.checkpoints, paths.plans, paths.journal, paths.audit]) {
    fs2.mkdirSync(dir, { recursive: true });
  }
  return paths;
}
function writeAudit(host, kind, action, details = {}) {
  try {
    const paths = ensureWorkspacePaths(true);
    if (!paths) return null;
    const file = path.join(paths.audit, `${kstDate()}.jsonl`);
    const entry = { ts: (/* @__PURE__ */ new Date()).toISOString(), kind, action, details };
    fs2.appendFileSync(file, JSON.stringify(entry) + "\n", "utf8");
    return file;
  } catch (error) {
    log(host, `audit write skipped: ${error}`);
    return null;
  }
}
async function openFile(file) {
  const doc = await vscode3.workspace.openTextDocument(file);
  await vscode3.window.showTextDocument(doc);
}
function readUtf8(file) {
  return fs2.readFileSync(file, "utf8");
}
function listMarkdown(dir) {
  if (!fs2.existsSync(dir)) return [];
  return fs2.readdirSync(dir).filter((name) => name.endsWith(".md")).sort();
}
var fs2, path, vscode3, NO_WORKSPACE_MESSAGE;
var init_workspace = __esm({
  "src/features/workspace.ts"() {
    "use strict";
    fs2 = __toESM(require("node:fs"));
    path = __toESM(require("node:path"));
    vscode3 = __toESM(require("vscode"));
    init_log();
    init_kst();
    NO_WORKSPACE_MESSAGE = "\uC6CC\uD06C\uC2A4\uD398\uC774\uC2A4\uB97C \uBA3C\uC800 \uC5F4\uC5B4\uC8FC\uC138\uC694.";
  }
});

// src/features/tree.ts
function makeItem(label, opts = {}) {
  const item = new vscode4.TreeItem(label, opts.collapsibleState ?? vscode4.TreeItemCollapsibleState.None);
  if (opts.description !== void 0) item.description = opts.description;
  if (opts.tooltip !== void 0) item.tooltip = opts.tooltip;
  if (opts.command) item.command = opts.command;
  if (opts.icon) item.iconPath = new vscode4.ThemeIcon(opts.icon);
  if (opts.children) item.children = opts.children;
  return item;
}
function commandItem(label, command, title, icon, args) {
  return makeItem(label, { command: { command, title, arguments: args }, icon });
}
function checklistItems(lines2, emptyLabel, icon) {
  if (lines2.length === 0) return [makeItem(emptyLabel, { description: "\uC5C6\uC74C", icon: "circle-slash" })];
  return lines2.map((line) => {
    const done = isDoneTask(line);
    return makeItem(stripTask(line), { description: done ? "done" : "todo", tooltip: line, icon: done ? "pass" : icon });
  });
}
function noteItems(lines2, emptyLabel) {
  if (lines2.length === 0) return [makeItem(emptyLabel, { description: "\uC5C6\uC74C", icon: "circle-slash" })];
  return lines2.map((line) => makeItem(line, { tooltip: line, icon: "note" }));
}
function registerTreeView(host, spec) {
  const emitter = new vscode4.EventEmitter();
  const provider = {
    onDidChangeTreeData: emitter.event,
    getTreeItem: (item) => item,
    getChildren: (item) => {
      const parentId = item?.id ?? spec.viewId;
      const nodes = item ? item.children ?? [] : spec.roots();
      nodes.forEach((node, index) => {
        if (!node.id) node.id = `${parentId}/${index}:${String(node.label)}`;
      });
      return nodes;
    }
  };
  const view = vscode4.window.createTreeView(spec.viewId, { treeDataProvider: provider, showCollapseAll: false });
  const refresh = () => emitter.fire(void 0);
  const interval = setInterval(refresh, spec.refreshMs ?? 3e4);
  host.context.subscriptions.push(emitter, view, { dispose: () => clearInterval(interval) });
  const ws = workspaceRoot();
  if (!ws) return;
  for (const pattern of spec.watch) {
    try {
      const watcher = vscode4.workspace.createFileSystemWatcher(new vscode4.RelativePattern(ws, pattern));
      watcher.onDidChange(refresh);
      watcher.onDidCreate(refresh);
      watcher.onDidDelete(refresh);
      host.context.subscriptions.push(watcher);
    } catch {
    }
  }
}
var vscode4;
var init_tree = __esm({
  "src/features/tree.ts"() {
    "use strict";
    vscode4 = __toESM(require("vscode"));
    init_markdown();
    init_workspace();
  }
});

// src/features/goal-health.ts
function goalAgeHours(text, now = Date.now()) {
  const stamp = matchLine(text, "\uB9C8\uC9C0\uB9C9 \uAC31\uC2E0") || matchLine(text, "\uC791\uC131\uC77C");
  const m = stamp.match(/(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2})/);
  if (!m) return Number.POSITIVE_INFINITY;
  return (now - Date.UTC(+m[1], +m[2] - 1, +m[3], +m[4] - 9, +m[5])) / 36e5;
}
function assessGoalHealth(text, seen, staleHours, stuckSessions = 3, now = Date.now()) {
  const goal = parseGoal(text);
  const ageHours = goalAgeHours(text, now);
  const active = goal.status.toLowerCase() === "active";
  const stale = active && Number.isFinite(ageHours) && ageHours > staleHours;
  const stuck = active && !!seen && seen.sessions >= stuckSessions;
  const parts = [];
  if (stale) parts.push(`\uB9C8\uC9C0\uB9C9 \uAC31\uC2E0 ${Math.round(ageHours)}\uC2DC\uAC04 \uC804`);
  if (stuck && seen) parts.push(`\uAC19\uC740 Now \uD56D\uBAA9 ${seen.sessions}\uC138\uC158\uC9F8: ${seen.item}`);
  return { ageHours, stale, stuck, warning: parts.length ? parts.join(" \xB7 ") + " \u2014 \uC791\uC5C5\uC744 \uB354 \uC791\uAC8C \uCABC\uAC1C\uAC70\uB098 \uBAA9\uD45C \uD30C\uC77C\uC744 \uAC31\uC2E0\uD558\uC138\uC694." : void 0 };
}
async function trackNowItem(host) {
  const paths = ensureWorkspacePaths(true);
  if (!paths || !fs3.existsSync(paths.current)) return void 0;
  const goal = parseGoal(readUtf8(paths.current));
  const first = goal.now.filter((l) => !/^- \[[xX]\]/.test(l)).map(stripTask)[0];
  if (!first || goal.status.toLowerCase() !== "active") {
    await host.context.workspaceState.update(SEEN_KEY, void 0);
    return void 0;
  }
  const previous = host.context.workspaceState.get(SEEN_KEY);
  const next = previous && previous.item === first ? { ...previous, sessions: previous.sessions + 1 } : { item: first, sessions: 1, since: (/* @__PURE__ */ new Date()).toISOString() };
  await host.context.workspaceState.update(SEEN_KEY, next);
  return next;
}
function currentHealth(host, text) {
  return assessGoalHealth(text, host.context.workspaceState.get(SEEN_KEY), config().get("goalStaleHours") ?? 48);
}
async function checkGoalHealth(host) {
  const seen = await trackNowItem(host);
  const paths = ensureWorkspacePaths(true);
  if (!paths || !fs3.existsSync(paths.current)) return;
  const health = currentHealth(host, readUtf8(paths.current));
  if (!health.warning) return;
  writeAudit(host, "goal", "healthWarning", { stale: health.stale, stuck: health.stuck, ageHours: Math.round(health.ageHours), sessions: seen?.sessions });
  log(host, `goal health: ${health.warning}`);
  void vscode5.window.showWarningMessage(`\uBAA9\uD45C \uC815\uCCB4 \uAC10\uC9C0: ${health.warning}`, "\uBAA9\uD45C \uC5F4\uAE30", "\uCD5C\uC2E0 \uACC4\uD68D \uC5F4\uAE30").then(async (choice) => {
    if (choice === "\uBAA9\uD45C \uC5F4\uAE30") await vscode5.commands.executeCommand("vibe-code.openCurrentGoal");
    else if (choice === "\uCD5C\uC2E0 \uACC4\uD68D \uC5F4\uAE30") await vscode5.commands.executeCommand("vibe-code.openLatestPlan");
  });
}
var fs3, vscode5, SEEN_KEY;
var init_goal_health = __esm({
  "src/features/goal-health.ts"() {
    "use strict";
    fs3 = __toESM(require("node:fs"));
    vscode5 = __toESM(require("vscode"));
    init_log();
    init_settings();
    init_markdown();
    init_goals();
    init_workspace();
    SEEN_KEY = "vibeCode.goalNowSeen";
  }
});

// src/features/goals.ts
function goalTemplate(stamp) {
  return `# \uBAA9\uD45C: \uC0C8 \uBAA9\uD45C

\uC0C1\uD0DC: draft
\uC791\uC131\uC77C: ${stamp}
\uB9C8\uC9C0\uB9C9 \uAC31\uC2E0: ${stamp}
\uBAA9\uD45C \uD30C\uC77C: ${GOAL_FILE}

## \uBAA9\uD45C
/goal <\uC6D0\uD558\uB294 \uACB0\uACFC>\uB85C \uBAA9\uD45C\uB97C \uC9C0\uC815\uD558\uAC70\uB098 \uC774 \uD30C\uC77C\uC744 \uD3B8\uC9D1\uD558\uC138\uC694.

## \uC644\uB8CC \uAE30\uC900
- [ ] \uBAA9\uD45C\uB97C \uAC80\uC99D \uAC00\uB2A5\uD55C \uAE30\uC900\uC73C\uB85C \uC815\uB9AC
- [ ] \uAD6C\uD604/\uBB38\uC11C/\uAC80\uC99D \uBC94\uC704 \uACB0\uC815
- [ ] \uCD5C\uC885 \uAC80\uC99D \uD1B5\uACFC

## \uD604\uC7AC \uC0C1\uD0DC
- \uC9C0\uAE08 \uB2E8\uACC4: discovery
- \uB2E4\uC74C \uD589\uB3D9: \uBAA9\uD45C \uBB38\uC7A5\uC744 \uAD6C\uCCB4\uD654
- \uB9C8\uC9C0\uB9C9 \uAC80\uC99D: \uC5C6\uC74C

## \uC791\uC5C5 \uD050
### Now
- [ ] \uBAA9\uD45C \uC124\uBA85 \uC791\uC131

### Next
- [ ] \uCCAB \uBC88\uC9F8 \uC791\uC740 \uC791\uC5C5 \uC120\uD0DD

### Done

## \uACB0\uC815 \uB85C\uADF8
- ${stamp} - \uD604\uC7AC \uBAA9\uD45C draft \uC0DD\uC131

## \uBCC0\uACBD \uB85C\uADF8

## \uAC80\uC99D \uB85C\uADF8

## \uC911\uB2E8/\uC7AC\uAC1C \uB178\uD2B8
\uB2E4\uC74C \uC138\uC158\uC740 /goal \uC774\uC5B4\uC11C\uB85C \uC2DC\uC791\uD558\uC138\uC694.
`;
}
function handoffTemplate(stamp, currentGoal) {
  return `# \uBAA9\uD45C \uD578\uB4DC\uC624\uD504 - ${stamp}

\uBAA9\uD45C \uD30C\uC77C: ${GOAL_FILE}
\uC0DD\uC131\uC77C: ${stamp}

## \uB2E4\uC74C \uC138\uC158 \uC2DC\uC791 \uBB38\uC7A5
/goal \uC774\uC5B4\uC11C

## \uBC14\uB85C \uD560 \uC77C
- current.md\uC758 \uD604\uC7AC \uC0C1\uD0DC\uC640 \uC791\uC5C5 \uD050 Now \uD56D\uBAA9\uBD80\uD130 \uD655\uC778
- \uB9C8\uC9C0\uB9C9 \uAC80\uC99D \uB85C\uADF8 \uC774\uD6C4 \uBCC0\uACBD\uB41C \uD30C\uC77C \uD655\uC778
- \uAC00\uB2A5\uD55C \uAC00\uC7A5 \uC791\uC740 \uB2E4\uC74C \uC791\uC5C5 \uD558\uB098\uB97C \uAD6C\uD604\uD558\uACE0 \uAC80\uC99D

## \uD604\uC7AC \uBAA9\uD45C \uC2A4\uB0C5\uC0F7

~~~markdown
${currentGoal}
~~~
`;
}
function parseGoal(text) {
  const checks = countChecks(text);
  return {
    title: headingTitle(text, "\uBAA9\uD45C", "\uD604\uC7AC \uBAA9\uD45C"),
    status: matchLine(text, "\uC0C1\uD0DC", "(unknown)"),
    phase: matchLine(text, "- \uC9C0\uAE08 \uB2E8\uACC4", "(unknown)"),
    next: matchLine(text, "- \uB2E4\uC74C \uD589\uB3D9", "(unknown)"),
    done: checks.done,
    total: checks.total,
    criteria: taskLines(section(text, "\uC644\uB8CC \uAE30\uC900")),
    now: taskLines(subsection(text, "Now")),
    nextItems: taskLines(subsection(text, "Next")),
    verification: lines(section(text, "\uAC80\uC99D \uB85C\uADF8")).slice(-5)
  };
}
function currentGoalTitle(currentFile, fallback = "\uD604\uC7AC \uBAA9\uD45C") {
  try {
    if (fs4.existsSync(currentFile)) return headingTitle(readUtf8(currentFile), "\uBAA9\uD45C", fallback);
  } catch {
  }
  return fallback;
}
function createHandoffFile(host, paths, note = "") {
  const stamp = kstStamp();
  const handoff = path2.join(paths.sessions, `${stamp.file}-handoff.md`);
  fs4.writeFileSync(handoff, handoffTemplate(stamp.human, readUtf8(paths.current)), "utf8");
  fs4.appendFileSync(paths.current, `
- ${stamp.human} - handoff \uC0DD\uC131: .vibe-code/sessions/${path2.basename(handoff)}${note ? ` (${note})` : ""}
`, "utf8");
  const rel = `.vibe-code/sessions/${path2.basename(handoff)}`;
  log(host, `goal handoff created: ${handoff}`);
  return rel;
}
function openCriteria(text) {
  return taskLines(section(text, "\uC644\uB8CC \uAE30\uC900")).filter((l) => !/^- \[[xX]\]/.test(l)).map(stripTask);
}
function registerGoalCommands(host) {
  const { context, output } = host;
  context.subscriptions.push(
    vscode6.commands.registerCommand("vibe-code.setGoalStatus", async (statusArg) => {
      const paths = ensureWorkspacePaths();
      if (!paths) return;
      if (!fs4.existsSync(paths.current)) {
        void vscode6.window.showWarningMessage(NO_GOAL_MESSAGE);
        return;
      }
      const text = readUtf8(paths.current);
      const explicit = typeof statusArg === "string" && GOAL_STATUSES.includes(statusArg);
      const status = explicit ? statusArg : await vscode6.window.showQuickPick(GOAL_STATUSES, { placeHolder: `\uBAA9\uD45C \uC0C1\uD0DC \uC120\uD0DD (\uD604\uC7AC: ${matchLine(text, "\uC0C1\uD0DC", "?")})` });
      if (!status) return;
      if (status === "done" && !explicit) {
        const open = openCriteria(text);
        if (open.length > 0) {
          const go = await vscode6.window.showWarningMessage(`\uC644\uB8CC \uAE30\uC900 ${open.length}\uAC1C\uAC00 \uC544\uC9C1 \uCCB4\uD06C\uB418\uC9C0 \uC54A\uC558\uC2B5\uB2C8\uB2E4:
${open.slice(0, 5).join("\n")}${open.length > 5 ? "\n\u2026" : ""}`, { modal: true }, "\uADF8\uB798\uB3C4 done\uC73C\uB85C \uBCC0\uACBD");
          if (go !== "\uADF8\uB798\uB3C4 done\uC73C\uB85C \uBCC0\uACBD") return;
          writeAudit(host, "goal", "completionGateOverridden", { open: open.length });
        }
      }
      fs4.writeFileSync(paths.current, touchPlan(text, status, kstStamp().human), "utf8");
      void vscode6.window.showInformationMessage("\uBAA9\uD45C \uC0C1\uD0DC\uB97C \uBCC0\uACBD\uD588\uC2B5\uB2C8\uB2E4: " + status);
      writeAudit(host, "goal", "setGoalStatus", { status, openCriteria: openCriteria(text).length });
    }),
    vscode6.commands.registerCommand("vibe-code.openCurrentGoal", async () => {
      const paths = ensureWorkspacePaths();
      if (!paths) return;
      if (!fs4.existsSync(paths.current)) {
        fs4.writeFileSync(paths.current, goalTemplate(kstStamp().human), "utf8");
        log(host, `current goal draft created: ${paths.current}`);
      }
      await openFile(paths.current);
      void vscode6.window.showInformationMessage("\uD604\uC7AC \uBAA9\uD45C \uD30C\uC77C\uC744 \uC5F4\uC5C8\uC2B5\uB2C8\uB2E4.");
      writeAudit(host, "goal", "openCurrentGoal", { currentExists: true, goalFile: GOAL_FILE });
    }),
    vscode6.commands.registerCommand("vibe-code.showGoalStatus", async () => {
      const paths = ensureWorkspacePaths();
      if (!paths) return;
      if (!fs4.existsSync(paths.current)) {
        void vscode6.window.showWarningMessage(NO_GOAL_MESSAGE);
        return;
      }
      const text = readUtf8(paths.current);
      const goal = parseGoal(text);
      const progress = progressLabel(goal, "\uCCB4\uD06C\uB9AC\uC2A4\uD2B8 \uC5C6\uC74C");
      output.show(true);
      output.appendLine("\n=== Vibe Code \uBAA9\uD45C \uC0C1\uD0DC ===");
      output.appendLine("checklist progress: local goal file");
      output.appendLine(`\uBAA9\uD45C: ${goal.title}`);
      output.appendLine(`\uC0C1\uD0DC: ${goal.status}`);
      output.appendLine(`\uC9C4\uD589\uB960: ${progress}`);
      output.appendLine(`\uD604\uC7AC \uB2E8\uACC4: ${goal.phase}`);
      output.appendLine(`\uB2E4\uC74C \uD589\uB3D9: ${goal.next}`);
      output.appendLine("\n--- Now ---");
      output.appendLine(subsection(text, "Now") || "(\uBE44\uC5B4 \uC788\uC74C)");
      output.appendLine("\n--- Next ---");
      output.appendLine(subsection(text, "Next") || "(\uBE44\uC5B4 \uC788\uC74C)");
      output.appendLine("\n--- \uCD5C\uADFC \uAC80\uC99D \uB85C\uADF8 ---");
      if (goal.verification.length === 0) output.appendLine("(\uAC80\uC99D \uB85C\uADF8 \uC5C6\uC74C)");
      for (const line of goal.verification) output.appendLine(line);
      output.appendLine("================================\n");
      void vscode6.window.showInformationMessage(`\uBAA9\uD45C \uC0C1\uD0DC: ${goal.title} (${progress})`);
      writeAudit(host, "goal", "showGoalStatus", {
        title: goal.title,
        status: goal.status,
        progress: goal.total > 0 ? goal.done + "/" + goal.total : "checklist-none",
        phase: goal.phase,
        next: goal.next
      });
    }),
    vscode6.commands.registerCommand("vibe-code.createGoalHandoff", async () => {
      const paths = ensureWorkspacePaths();
      if (!paths) return;
      if (!fs4.existsSync(paths.current)) {
        void vscode6.window.showWarningMessage(NO_GOAL_MESSAGE);
        return;
      }
      const rel = createHandoffFile(host, paths);
      await openFile(path2.join(paths.ws, rel));
      void vscode6.window.showInformationMessage("\uBAA9\uD45C \uD578\uB4DC\uC624\uD504 \uD30C\uC77C\uC744 \uC0DD\uC131\uD588\uC2B5\uB2C8\uB2E4.");
      writeAudit(host, "goal", "createGoalHandoff", { handoffFile: rel });
    })
  );
}
function createGoalStatusBar(host) {
  const item = vscode6.window.createStatusBarItem(vscode6.StatusBarAlignment.Right, 99);
  item.command = "vibe-code.openCurrentGoal";
  const refresh = () => {
    try {
      const ws = vscode6.workspace.workspaceFolders?.[0]?.uri.fsPath;
      const file = ws ? path2.join(ws, ".vibe-code", "goals", "current.md") : void 0;
      if (!file || !fs4.existsSync(file)) {
        item.hide();
        return;
      }
      const text = readUtf8(file);
      const title = headingTitle(text, "\uBAA9\uD45C", "\uD604\uC7AC \uBAA9\uD45C");
      const status = matchLine(text, "\uC0C1\uD0DC", "active");
      const checks = countChecks(text);
      const short = title.length > 20 ? title.slice(0, 19) + "\u2026" : title;
      const health = currentHealth(host, text);
      item.text = `${health.warning ? "$(warning)" : "$(target)"} \uBAA9\uD45C: ${short}${checks.total > 0 ? ` ${checks.done}/${checks.total}` : ""}`;
      item.backgroundColor = health.warning ? new vscode6.ThemeColor("statusBarItem.warningBackground") : void 0;
      item.tooltip = `Vibe Code \uD604\uC7AC \uBAA9\uD45C
\uC0C1\uD0DC: ${status}
\uC9C4\uD589\uB960: ${progressLabel(checks, "\uCCB4\uD06C\uB9AC\uC2A4\uD2B8 \uC5C6\uC74C")}
\uD30C\uC77C: ${GOAL_FILE}${health.warning ? `
\u26A0 ${health.warning}` : ""}
\uD074\uB9AD\uD558\uC5EC \uBAA9\uD45C \uD30C\uC77C \uC5F4\uAE30`;
      item.show();
    } catch {
      item.hide();
    }
  };
  refresh();
  const interval = setInterval(refresh, 1e4);
  host.context.subscriptions.push(item, { dispose: () => clearInterval(interval) });
  log(host, "goal status bar item created");
}
function goalTrackerRoots() {
  const paths = ensureWorkspacePaths(true);
  if (!paths || !fs4.existsSync(paths.current)) {
    return [
      makeItem("\uD604\uC7AC \uBAA9\uD45C \uC5C6\uC74C", {
        description: "/goal\uB85C \uC2DC\uC791",
        tooltip: "/goal \uBA85\uB839\uC73C\uB85C \uBAA9\uD45C\uB97C \uC2DC\uC791\uD558\uAC70\uB098 \uD604\uC7AC \uBAA9\uD45C \uD30C\uC77C\uC744 \uC5EC\uC138\uC694.",
        command: OPEN_GOAL,
        icon: "target"
      })
    ];
  }
  const goal = parseGoal(readUtf8(paths.current));
  const progress = progressLabel(goal, "\uCCB4\uD06C \uC5C6\uC74C");
  const expanded = vscode6.TreeItemCollapsibleState.Expanded;
  const collapsed = vscode6.TreeItemCollapsibleState.Collapsed;
  return [
    makeItem(goal.title, {
      description: `${progress} \xB7 ${goal.status}`,
      tooltip: `\uC0C1\uD0DC: ${goal.status}
\uD604\uC7AC \uB2E8\uACC4: ${goal.phase}
\uB2E4\uC74C \uD589\uB3D9: ${goal.next}`,
      command: OPEN_GOAL,
      icon: "target"
    }),
    makeItem("\uC644\uB8CC \uAE30\uC900", { description: progress, tooltip: "\uC644\uB8CC \uAE30\uC900 \uCCB4\uD06C\uB9AC\uC2A4\uD2B8", collapsibleState: expanded, icon: "checklist", children: checklistItems(goal.criteria, "\uC644\uB8CC \uAE30\uC900 \uC5C6\uC74C", "check") }),
    makeItem("Now", { description: goal.now.length > 0 ? `${goal.now.length}\uAC1C` : "\uBE44\uC5B4 \uC788\uC74C", tooltip: "\uBC14\uB85C \uD560 \uC77C", collapsibleState: expanded, icon: "play", children: checklistItems(goal.now, "Now \uD56D\uBAA9 \uC5C6\uC74C", "circle-outline") }),
    makeItem("Next", { description: goal.nextItems.length > 0 ? `${goal.nextItems.length}\uAC1C` : "\uBE44\uC5B4 \uC788\uC74C", tooltip: "\uB2E4\uC74C \uC791\uC5C5", collapsibleState: collapsed, icon: "arrow-right", children: checklistItems(goal.nextItems, "Next \uD56D\uBAA9 \uC5C6\uC74C", "circle-large-outline") }),
    makeItem("\uCD5C\uADFC \uAC80\uC99D", { description: goal.verification.length > 0 ? `${goal.verification.length}\uC904` : "\uC5C6\uC74C", tooltip: "\uCD5C\uADFC \uAC80\uC99D \uB85C\uADF8", collapsibleState: collapsed, icon: "beaker", children: noteItems(goal.verification, "\uAC80\uC99D \uB85C\uADF8 \uC5C6\uC74C") }),
    makeItem("\uBC14\uB85C \uC2E4\uD589", {
      description: "\uBA85\uB839 5\uAC1C",
      tooltip: "\uD604\uC7AC \uBAA9\uD45C \uAD00\uB828 \uBA85\uB839",
      collapsibleState: collapsed,
      icon: "tools",
      children: [
        commandItem("\uD604\uC7AC \uBAA9\uD45C \uC5F4\uAE30", "vibe-code.openCurrentGoal", "\uD604\uC7AC \uBAA9\uD45C \uC5F4\uAE30", "go-to-file"),
        commandItem("\uBAA9\uD45C \uC0C1\uD0DC \uBCF4\uAE30", "vibe-code.showGoalStatus", "\uBAA9\uD45C \uC0C1\uD0DC \uBCF4\uAE30", "list-tree"),
        commandItem("\uBAA9\uD45C \uD578\uB4DC\uC624\uD504 \uC0DD\uC131", "vibe-code.createGoalHandoff", "\uBAA9\uD45C \uD578\uB4DC\uC624\uD504 \uC0DD\uC131", "export"),
        commandItem("\uBAA9\uD45C \uC0C1\uD0DC \uBCC0\uACBD", "vibe-code.setGoalStatus", "\uBAA9\uD45C \uC0C1\uD0DC \uBCC0\uACBD", "symbol-enum"),
        commandItem("\uBAA9\uD45C \uC774\uC5B4\uC11C \uC9C4\uD589", "vibe-code.resumeGoal", "\uBAA9\uD45C \uC774\uC5B4\uC11C \uC9C4\uD589", "debug-continue")
      ]
    })
  ];
}
function registerGoalTrackerView(host) {
  registerTreeView(host, { viewId: "vibe-code.GoalTracker", roots: goalTrackerRoots, watch: [GOAL_FILE] });
  log(host, "goal tracker view registered");
}
var fs4, path2, vscode6, GOAL_FILE, NO_GOAL_MESSAGE, GOAL_STATUSES, OPEN_GOAL;
var init_goals = __esm({
  "src/features/goals.ts"() {
    "use strict";
    fs4 = __toESM(require("node:fs"));
    path2 = __toESM(require("node:path"));
    vscode6 = __toESM(require("vscode"));
    init_log();
    init_kst();
    init_markdown();
    init_tree();
    init_workspace();
    init_goal_health();
    GOAL_FILE = ".vibe-code/goals/current.md";
    NO_GOAL_MESSAGE = "\uD604\uC7AC \uBAA9\uD45C \uD30C\uC77C\uC774 \uC5C6\uC2B5\uB2C8\uB2E4. \uBA3C\uC800 /goal\uC744 \uC2DC\uC791\uD558\uAC70\uB098 \uD604\uC7AC \uBAA9\uD45C \uC5F4\uAE30 \uBA85\uB839\uC744 \uC2E4\uD589\uD558\uC138\uC694.";
    GOAL_STATUSES = ["draft", "active", "blocked", "done"];
    OPEN_GOAL = { command: "vibe-code.openCurrentGoal", title: "\uD604\uC7AC \uBAA9\uD45C \uC5F4\uAE30" };
  }
});

// src/features/goal-catalog.ts
function slugify(title) {
  return title.trim().toLowerCase().replace(/[^\p{L}\p{N}]+/gu, "-").replace(/^-+|-+$/g, "").slice(0, 40).replace(/-+$/g, "") || "goal";
}
function buildGoalFile(opts) {
  let text = goalTemplate(opts.stamp).replace("# \uBAA9\uD45C: \uC0C8 \uBAA9\uD45C", `# \uBAA9\uD45C: ${opts.title}`).replace("\uC0C1\uD0DC: draft", "\uC0C1\uD0DC: active").replace(`\uBAA9\uD45C \uD30C\uC77C: ${GOAL_FILE}`, `\uBAA9\uD45C \uD30C\uC77C: .vibe-code/goals/${opts.fileName}`).replace("/goal <\uC6D0\uD558\uB294 \uACB0\uACFC>\uB85C \uBAA9\uD45C\uB97C \uC9C0\uC815\uD558\uAC70\uB098 \uC774 \uD30C\uC77C\uC744 \uD3B8\uC9D1\uD558\uC138\uC694.", opts.description || opts.title).replace("- \uB2E4\uC74C \uD589\uB3D9: \uBAA9\uD45C \uBB38\uC7A5\uC744 \uAD6C\uCCB4\uD654", "- \uB2E4\uC74C \uD589\uB3D9: \uCCAB Now \uD56D\uBAA9 \uCC29\uC218").replace("- [ ] \uBAA9\uD45C \uC124\uBA85 \uC791\uC131", `- [ ] ${opts.title}: \uCF54\uB4DC \uAD6C\uC870 \uD30C\uC545\uACFC \uCCAB \uAD6C\uD604 \uB2E8\uC704 \uC815\uC758`).replace("- \uD604\uC7AC \uBAA9\uD45C draft \uC0DD\uC131", `- \uBAA9\uD45C \uC0DD\uC131 (${opts.title})`);
  const criteria = opts.criteria.map((c) => `- [ ] ${c}`);
  text = text.replace(/## 완료 기준\n([\s\S]*?)(?=\n## )/, `## \uC644\uB8CC \uAE30\uC900
${criteria.join("\n")}
`);
  return text;
}
function currentGoalFileName(currentText) {
  const line = matchLine(currentText, "\uBAA9\uD45C \uD30C\uC77C");
  const name = line ? path3.posix.basename(line.replace(/\\/g, "/")) : "";
  return name && name !== "current.md" ? name : void 0;
}
function listGoals(paths) {
  const currentName = fs5.existsSync(paths.current) ? currentGoalFileName(readUtf8(paths.current)) : void 0;
  return listMarkdown(paths.goals).filter((name) => name !== "current.md").map((name) => {
    const text = readUtf8(path3.join(paths.goals, name));
    return { name, title: headingTitle(text, "\uBAA9\uD45C", name), status: matchLine(text, "\uC0C1\uD0DC", "?"), updated: matchLine(text, "\uB9C8\uC9C0\uB9C9 \uAC31\uC2E0") || matchLine(text, "\uC791\uC131\uC77C"), isCurrent: name === currentName };
  }).sort((a, b) => a.isCurrent === b.isCurrent ? b.updated.localeCompare(a.updated) : a.isCurrent ? -1 : 1);
}
function saveCurrentGoal(paths) {
  if (!fs5.existsSync(paths.current)) return void 0;
  const text = readUtf8(paths.current);
  let name = currentGoalFileName(text);
  let body = text;
  if (!name) {
    name = `${kstStamp().file}-${slugify(headingTitle(text, "\uBAA9\uD45C", "goal"))}.md`;
    body = setLine(text, "\uBAA9\uD45C \uD30C\uC77C", `.vibe-code/goals/${name}`);
    fs5.writeFileSync(paths.current, body, "utf8");
  }
  fs5.writeFileSync(path3.join(paths.goals, name), body, "utf8");
  return name;
}
function switchToGoal(paths, name) {
  const previous = saveCurrentGoal(paths);
  if (previous === name) return;
  fs5.copyFileSync(path3.join(paths.goals, name), paths.current);
}
function tickCriteria(goalText, indexes) {
  const body = section(goalText, "\uC644\uB8CC \uAE30\uC900");
  if (!body) return goalText;
  const all = goalText.split(/\r?\n/);
  let seen = 0;
  for (let i = 0; i < all.length; i++) {
    if (/^## 완료 기준/.test(all[i])) {
      for (let j = i + 1; j < all.length && !/^## /.test(all[j]); j++) {
        if (/^- \[( |x|X)\] /.test(all[j].trim())) {
          seen++;
          if (indexes.includes(seen)) all[j] = all[j].replace(/^(\s*)- \[ \] /, "$1- [x] ");
        }
      }
      break;
    }
  }
  return all.join("\n");
}
function parseCriteriaLink(planText) {
  return matchLine(planText, "\uC5F0\uACB0 \uC644\uB8CC \uAE30\uC900").split(/[,\s]+/).map((s) => Number.parseInt(s, 10)).filter((n) => Number.isInteger(n) && n > 0);
}
function applyPlanCriteria(paths, planText) {
  const indexes = parseCriteriaLink(planText);
  if (indexes.length === 0 || !fs5.existsSync(paths.current)) return [];
  fs5.writeFileSync(paths.current, tickCriteria(readUtf8(paths.current), indexes), "utf8");
  return indexes;
}
function goalListRoots() {
  const paths = ensureWorkspacePaths(true);
  if (!paths) return [makeItem("\uC6CC\uD06C\uC2A4\uD398\uC774\uC2A4 \uC5C6\uC74C", { icon: "circle-slash" })];
  const goals = listGoals(paths);
  const items = goals.map(
    (g) => makeItem(g.title, {
      description: `${g.isCurrent ? "\uD604\uC7AC \xB7 " : ""}${g.status}${g.updated ? ` \xB7 ${g.updated.slice(0, 16)}` : ""}`,
      tooltip: `.vibe-code/goals/${g.name}
\uC0C1\uD0DC: ${g.status}
\uB9C8\uC9C0\uB9C9 \uAC31\uC2E0: ${g.updated || "?"}
\uD074\uB9AD: \uD604\uC7AC \uBAA9\uD45C\uB85C \uC804\uD658`,
      command: { command: "vibe-code.switchGoal", title: "\uBAA9\uD45C \uC804\uD658", arguments: [g.name] },
      icon: g.isCurrent ? "target" : g.status === "done" ? "pass" : "circle-large-outline"
    })
  );
  return [commandItem("\uC0C8 \uBAA9\uD45C \uB9CC\uB4E4\uAE30", "vibe-code.newGoal", "\uC0C8 \uBAA9\uD45C", "add"), ...items.length ? items : [makeItem("\uC800\uC7A5\uB41C \uBAA9\uD45C \uC5C6\uC74C", { description: "\uD604\uC7AC \uBAA9\uD45C\uB294 \uC804\uD658 \uC2DC \uC790\uB3D9 \uC800\uC7A5\uB429\uB2C8\uB2E4", icon: "circle-slash" })]];
}
function registerGoalCatalog(host) {
  const { context } = host;
  context.subscriptions.push(
    vscode7.commands.registerCommand("vibe-code.switchGoal", async (nameArg) => {
      const paths = ensureWorkspacePaths();
      if (!paths) return;
      const goals = listGoals(paths).filter((g) => !g.isCurrent);
      if (goals.length === 0) {
        void vscode7.window.showInformationMessage("\uC804\uD658\uD560 \uB2E4\uB978 \uBAA9\uD45C\uAC00 \uC5C6\uC2B5\uB2C8\uB2E4. `\uC0C8 \uBAA9\uD45C \uB9CC\uB4E4\uAE30`\uB85C \uCD94\uAC00\uD558\uC138\uC694.");
        return;
      }
      const name = typeof nameArg === "string" && goals.some((g) => g.name === nameArg) ? nameArg : (await vscode7.window.showQuickPick(goals.map((g) => ({ label: g.title, description: `${g.status} \xB7 ${g.updated}`, detail: g.name, name: g.name })), { placeHolder: "\uD604\uC7AC \uBAA9\uD45C\uB85C \uC804\uD658\uD560 \uBAA9\uD45C" }))?.name;
      if (!name) return;
      switchToGoal(paths, name);
      await openFile(paths.current);
      void vscode7.window.showInformationMessage(`\uD604\uC7AC \uBAA9\uD45C\uB97C \uC804\uD658\uD588\uC2B5\uB2C8\uB2E4: ${name}`);
      writeAudit(host, "goal", "switchGoal", { goalFile: `.vibe-code/goals/${name}` });
    }),
    vscode7.commands.registerCommand("vibe-code.newGoal", async () => {
      const paths = ensureWorkspacePaths();
      if (!paths) return;
      const title = (await vscode7.window.showInputBox({ prompt: "\uBAA9\uD45C \uD55C \uC904 (\uC608: \uACB0\uC81C \uBAA8\uB4C8\uC5D0 \uD658\uBD88 API \uCD94\uAC00)", ignoreFocusOut: true }))?.trim();
      if (!title) return;
      const preset = await vscode7.window.showQuickPick(GOAL_PRESETS.map((p) => ({ label: p.label, detail: p.detail, preset: p })), { placeHolder: "\uBAA9\uD45C \uC720\uD615 (\uC644\uB8CC \uAE30\uC900 \uD504\uB9AC\uC14B)", ignoreFocusOut: true });
      if (!preset) return;
      const description = await vscode7.window.showInputBox({ prompt: "\uBAA9\uD45C \uC124\uBA85 (\uC120\uD0DD, \uAE30\uB300 \uACB0\uACFC\xB7\uC81C\uC57D)", ignoreFocusOut: true }) ?? "";
      const stamp = kstStamp();
      const fileName = `${stamp.file}-${slugify(title)}.md`;
      saveCurrentGoal(paths);
      const body = buildGoalFile({ title, description, criteria: preset.preset.criteria, fileName, stamp: stamp.human });
      fs5.writeFileSync(path3.join(paths.goals, fileName), body, "utf8");
      fs5.writeFileSync(paths.current, body, "utf8");
      await openFile(paths.current);
      writeAudit(host, "goal", "newGoal", { goalFile: `.vibe-code/goals/${fileName}`, preset: preset.preset.key });
      log(host, `new goal created: ${fileName}`);
      const start = await vscode7.window.showInformationMessage(`\uC0C8 \uBAA9\uD45C\uB97C \uB9CC\uB4E4\uC5C8\uC2B5\uB2C8\uB2E4: ${title}`, "\uBC14\uB85C \uC2DC\uC791 (/goal)", "\uB098\uC911\uC5D0");
      if (start === "\uBC14\uB85C \uC2DC\uC791 (/goal)") {
        const provider = host.getProvider();
        if (provider?.initClineWithTask) {
          await vscode7.commands.executeCommand("vibe-code.SidebarProvider.focus");
          await provider.initClineWithTask(`/goal ${title}`);
        }
      }
    }),
    vscode7.commands.registerCommand("vibe-code.linkPlanToCriteria", async () => {
      const paths = ensureWorkspacePaths();
      if (!paths || !fs5.existsSync(paths.current)) {
        void vscode7.window.showWarningMessage("\uD604\uC7AC \uBAA9\uD45C \uD30C\uC77C\uC774 \uC5C6\uC2B5\uB2C8\uB2E4.");
        return;
      }
      const { selectPlanName: selectPlanName2 } = await Promise.resolve().then(() => (init_plans(), plans_exports));
      const planFile = path3.join(paths.plans, selectPlanName2(paths));
      if (!fs5.existsSync(planFile)) {
        void vscode7.window.showWarningMessage("\uD604\uC7AC \uACC4\uD68D \uD30C\uC77C\uC774 \uC5C6\uC2B5\uB2C8\uB2E4. \uBA3C\uC800 \uCD5C\uC2E0 \uACC4\uD68D \uC5F4\uAE30 \uBA85\uB839\uC744 \uC2E4\uD589\uD558\uC138\uC694.");
        return;
      }
      const criteria = taskLines(section(readUtf8(paths.current), "\uC644\uB8CC \uAE30\uC900"));
      if (criteria.length === 0) {
        void vscode7.window.showWarningMessage("\uBAA9\uD45C\uC5D0 \uC644\uB8CC \uAE30\uC900 \uD56D\uBAA9\uC774 \uC5C6\uC2B5\uB2C8\uB2E4.");
        return;
      }
      const planText = readUtf8(planFile);
      const linked = parseCriteriaLink(planText);
      const picked = await vscode7.window.showQuickPick(
        criteria.map((c, i) => ({ label: `${i + 1}. ${c.replace(/^- \[( |x|X)\] /, "")}`, picked: linked.includes(i + 1), index: i + 1 })),
        { canPickMany: true, placeHolder: "\uC774 \uACC4\uD68D\uC774 \uC644\uB8CC\uB418\uBA74 \uCCB4\uD06C\uD560 \uC644\uB8CC \uAE30\uC900" }
      );
      if (!picked) return;
      const indexes = picked.map((p) => p.index);
      fs5.writeFileSync(planFile, setLine(planText, "\uC5F0\uACB0 \uC644\uB8CC \uAE30\uC900", indexes.join(", ") || "-"), "utf8");
      void vscode7.window.showInformationMessage(`\uACC4\uD68D\uC744 \uC644\uB8CC \uAE30\uC900 ${indexes.join(", ") || "(\uC5C6\uC74C)"}\uACFC \uC5F0\uACB0\uD588\uC2B5\uB2C8\uB2E4.`);
      writeAudit(host, "plan", "linkPlanToCriteria", { planFile: `.vibe-code/plans/${path3.basename(planFile)}`, criteria: indexes });
    })
  );
  registerTreeView(host, { viewId: "vibe-code.GoalList", roots: goalListRoots, watch: [".vibe-code/goals/*.md"] });
  log(host, "goal catalog registered (list view, switch, new goal, criteria link)");
}
var fs5, path3, vscode7, GOAL_PRESETS;
var init_goal_catalog = __esm({
  "src/features/goal-catalog.ts"() {
    "use strict";
    fs5 = __toESM(require("node:fs"));
    path3 = __toESM(require("node:path"));
    vscode7 = __toESM(require("vscode"));
    init_log();
    init_kst();
    init_markdown();
    init_goals();
    init_tree();
    init_workspace();
    GOAL_PRESETS = [
      { key: "feature", label: "\uAE30\uB2A5 \uAC1C\uBC1C", detail: "\uC0C8 \uAE30\uB2A5\uC744 \uB3D9\uC791\xB7\uD14C\uC2A4\uD2B8\xB7\uBB38\uC11C\uAE4C\uC9C0 \uB9C8\uBB34\uB9AC", criteria: ["\uD575\uC2EC \uAE30\uB2A5\uC774 \uC2E4\uC81C\uB85C \uB3D9\uC791\uD568 (\uC218\uB3D9 \uD655\uC778)", "\uC790\uB3D9 \uD14C\uC2A4\uD2B8 \uCD94\uAC00 \uBC0F \uD1B5\uACFC", "\uBB38\uC11C \uB610\uB294 \uC0AC\uC6A9\uBC95 \uBC18\uC601", "\uAC80\uC99D \uB85C\uADF8\uC5D0 \uC2E4\uD589 \uACB0\uACFC \uAE30\uB85D"] },
      { key: "bugfix", label: "\uBC84\uADF8 \uC218\uC815", detail: "\uC7AC\uD604 \u2192 \uC6D0\uC778 \u2192 \uD68C\uADC0 \uD14C\uC2A4\uD2B8 \u2192 \uC218\uC815 \uD655\uC778", criteria: ["\uC7AC\uD604 \uC808\uCC28\uC640 \uC6D0\uC778 \uAE30\uB85D", "\uD68C\uADC0 \uD14C\uC2A4\uD2B8 \uCD94\uAC00", "\uC218\uC815 \uD6C4 \uC7AC\uD604 \uC808\uCC28\uB85C \uC7AC\uD655\uC778", "\uBCC0\uACBD \uB85C\uADF8/\uAD00\uB828 \uBB38\uC11C \uAC31\uC2E0"] },
      { key: "refactor", label: "\uB9AC\uD329\uD130\uB9C1", detail: "\uB3D9\uC791\uC740 \uADF8\uB300\uB85C, \uAD6C\uC870\uB9CC \uAC1C\uC120", criteria: ["\uAE30\uC874 \uD14C\uC2A4\uD2B8 \uC804\uBD80 \uD1B5\uACFC (\uB3D9\uC791 \uBCC0\uACBD \uC5C6\uC74C)", "\uAD6C\uC870 \uAC1C\uC120 \uAE30\uC900\uC744 \uBA85\uC2DC\uD558\uACE0 \uB2EC\uC131", "\uC131\uB2A5/\uAC00\uB3C5\uC131 \uC9C0\uD45C \uC804\uD6C4 \uAE30\uB85D", "\uBB38\uC11C \uAC31\uC2E0"] },
      { key: "docs", label: "\uBB38\uC11C\uD654", detail: "\uB300\uC0C1 \uB3C5\uC790 \uAE30\uC900\uC73C\uB85C \uBB38\uC11C \uC791\uC131\xB7\uAC80\uD1A0", criteria: ["\uB300\uC0C1 \uB3C5\uC790\uC640 \uBC94\uC704 \uACB0\uC815", "\uCD08\uC548 \uC791\uC131", "\uAC80\uD1A0 \uC758\uACAC \uBC18\uC601", "\uB9C1\uD06C/\uC0C9\uC778 \uAC31\uC2E0"] },
      { key: "free", label: "\uC790\uC720 \uD615\uC2DD", detail: "\uAE30\uBCF8 \uD15C\uD50C\uB9BF \uC644\uB8CC \uAE30\uC900", criteria: ["\uBAA9\uD45C\uB97C \uAC80\uC99D \uAC00\uB2A5\uD55C \uAE30\uC900\uC73C\uB85C \uC815\uB9AC", "\uAD6C\uD604/\uBB38\uC11C/\uAC80\uC99D \uBC94\uC704 \uACB0\uC815", "\uCD5C\uC885 \uAC80\uC99D \uD1B5\uACFC"] }
    ];
  }
});

// src/features/plans.ts
var plans_exports = {};
__export(plans_exports, {
  advancePlanText: () => advancePlanText,
  archiveDonePlans: () => archiveDonePlans,
  conflictStamp: () => conflictStamp,
  freeName: () => freeName,
  linkedPlans: () => linkedPlans,
  openPlanItems: () => openPlanItems,
  planMeta: () => planMeta,
  planTemplate: () => planTemplate,
  registerPlanCommands: () => registerPlanCommands,
  registerPlanViews: () => registerPlanViews,
  selectPlanName: () => selectPlanName,
  setActivePlanName: () => setActivePlanName,
  sortByPriority: () => sortByPriority
});
function planTemplate(goalTitle, fileName, stamp) {
  return `# \uACC4\uD68D: ${goalTitle}

\uC0C1\uD0DC: draft
\uC791\uC131\uC77C: ${stamp}
\uB9C8\uC9C0\uB9C9 \uAC31\uC2E0: ${stamp}
\uC5F0\uACB0 \uBAA9\uD45C: ${GOAL_FILE}
\uACC4\uD68D \uD30C\uC77C: .vibe-code/plans/${fileName}

## \uBAA9\uD45C \uC815\uB82C
- \uBAA9\uD45C \uC81C\uBAA9: ${goalTitle}
- \uC774\uBC88 \uACC4\uD68D\uC758 \uCD08\uC810: \uAC00\uC7A5 \uC791\uC740 \uB2E4\uC74C \uAD6C\uD604 \uB2E8\uC704 \uC815\uC758

## \uB2E8\uACC4
- [ ] \uC694\uAD6C\uC0AC\uD56D/\uC81C\uC57D \uB2E4\uC2DC \uD655\uC778
- [ ] \uAC00\uC7A5 \uC791\uC740 \uAD6C\uD604 \uB2E8\uACC4 \uC120\uD0DD
- [ ] \uAC80\uC99D \uAE30\uC900 \uD655\uC815

## Now
- [ ] \uBC14\uB85C \uAD6C\uD604\uD560 \uC791\uC5C5 1\uAC1C \uC815\uC758

## Next
- [ ] \uB4A4\uC774\uC5B4 \uC2E4\uD589\uD560 \uC791\uC5C5 1\uAC1C \uC815\uC758

## Done

## Risks
- \uC5C6\uC74C

## \uAC80\uC99D \uACC4\uD68D
- [ ] \uAD00\uB828 \uC2A4\uD06C\uB9BD\uD2B8 \uB610\uB294 \uD14C\uC2A4\uD2B8 \uC2E4\uD589
`;
}
function activePointerFile(paths) {
  return path4.join(paths.plans, ".active-plan");
}
function selectPlanName(paths) {
  fs6.mkdirSync(paths.plans, { recursive: true });
  const files = listMarkdown(paths.plans);
  let active = "";
  try {
    const pointer = activePointerFile(paths);
    if (fs6.existsSync(pointer)) active = readUtf8(pointer).trim();
  } catch {
  }
  if (active && files.includes(active)) return active;
  return files.length > 0 ? files[files.length - 1] : "current-plan.md";
}
function setActivePlanName(paths, name) {
  fs6.mkdirSync(paths.plans, { recursive: true });
  fs6.writeFileSync(activePointerFile(paths), name, "utf8");
}
function latestPlan() {
  const paths = ensureWorkspacePaths(true);
  if (!paths) return null;
  const name = selectPlanName(paths);
  return { paths, file: path4.join(paths.plans, name), name };
}
function readPlan() {
  const ref = latestPlan();
  if (!ref || !fs6.existsSync(ref.file)) return null;
  return { ...ref, text: readUtf8(ref.file) };
}
function planMeta(text, name) {
  return {
    name,
    title: headingTitle(text, "\uACC4\uD68D", name),
    status: matchLine(text, "\uC0C1\uD0DC", "draft"),
    priority: matchLine(text, "\uC6B0\uC120\uC21C\uC704", "P2")
  };
}
function sortByPriority(rows) {
  return rows.sort((a, b) => priorityRank(a.priority) - priorityRank(b.priority) || a.name.localeCompare(b.name));
}
function planMetaList(dir) {
  return sortByPriority(listMarkdown(dir).map((name) => planMeta(readUtf8(path4.join(dir, name)), name)));
}
function linkedPlans(dir, bucket) {
  const rows = [];
  for (const name of listMarkdown(dir)) {
    const text = readUtf8(path4.join(dir, name));
    const linked = matchLine(text, "\uC5F0\uACB0 \uBAA9\uD45C");
    if (!linked) continue;
    rows.push({ ...planMeta(text, name), bucket, linked, linkedTitle: matchLine(text, "\uC5F0\uACB0 \uBAA9\uD45C \uC81C\uBAA9") });
  }
  return sortByPriority(rows);
}
async function pickPlanFile(files, arg, placeHolder) {
  if (typeof arg === "string" && files.includes(arg)) return arg;
  if (files.length === 1) return files[0];
  return vscode8.window.showQuickPick(files, { placeHolder });
}
function conflictStamp(date = /* @__PURE__ */ new Date()) {
  return kstStamp(date).file.replace(/[^0-9]/g, "");
}
function freeName(dir, name, stamp) {
  if (!fs6.existsSync(path4.join(dir, name))) return name;
  const parsed = path4.parse(name);
  const ext = parsed.ext || ".md";
  let candidate = `${parsed.name}-${stamp}${ext}`;
  for (let n = 2; fs6.existsSync(path4.join(dir, candidate)); n += 1) candidate = `${parsed.name}-${stamp}-${n}${ext}`;
  return candidate;
}
function archiveDonePlans(plansDir, archiveDir, stamp) {
  fs6.mkdirSync(archiveDir, { recursive: true });
  const archived = [];
  for (const name of listMarkdown(plansDir)) {
    const src = path4.join(plansDir, name);
    if (!fs6.statSync(src).isFile()) continue;
    if (matchLine(readUtf8(src), "\uC0C1\uD0DC") !== "done") continue;
    const dest = freeName(archiveDir, name, stamp);
    fs6.renameSync(src, path4.join(archiveDir, dest));
    archived.push({ name, dest });
  }
  return archived;
}
function advancePlanText(text, stamp) {
  const now = sectionLines(text, "Now");
  const nowIndex = now.findIndex(isChecklistLine);
  if (nowIndex < 0) return null;
  const next = sectionLines(text, "Next");
  const done = sectionLines(text, "Done");
  const completed = stripTask(now.splice(nowIndex, 1)[0].trim());
  done.push("- [x] " + completed);
  let promoted = "";
  const nextIndex = next.findIndex(isChecklistLine);
  if (nextIndex >= 0) {
    const line = next.splice(nextIndex, 1)[0].trim();
    promoted = stripTask(line);
    now.push(line);
  }
  let out = text;
  out = writeSection(out, "Now", now);
  out = writeSection(out, "Next", next);
  out = writeSection(out, "Done", done);
  out = touchPlan(out, "active", stamp);
  return { text: out, completed, promoted };
}
function openPlanItems(text) {
  return ["\uB2E8\uACC4", "Now", "\uAC80\uC99D \uACC4\uD68D"].flatMap((name) => taskLines(section(text, name)).filter((l) => !/^- \[[xX]\]/.test(l)).map((l) => `${name}: ${stripTask(l)}`));
}
function registerPlanCommands(host) {
  const { context, output } = host;
  const planFile = (name) => ".vibe-code/plans/" + name;
  context.subscriptions.push(
    vscode8.commands.registerCommand("vibe-code.openLatestPlan", async () => {
      const paths = ensureWorkspacePaths();
      if (!paths) return;
      const name = selectPlanName(paths);
      const target = path4.join(paths.plans, name);
      const created = !fs6.existsSync(target);
      if (created) {
        fs6.writeFileSync(target, planTemplate(currentGoalTitle(paths.current), name, kstStamp().human), "utf8");
        log(host, `current plan draft created: ${target}`);
      }
      setActivePlanName(paths, name);
      await openFile(target);
      void vscode8.window.showInformationMessage("\uCD5C\uC2E0 \uACC4\uD68D \uD30C\uC77C\uC744 \uC5F4\uC5C8\uC2B5\uB2C8\uB2E4.");
      writeAudit(host, "plan", "openLatestPlan", { planFile: planFile(name), created });
    }),
    vscode8.commands.registerCommand("vibe-code.advanceCurrentPlan", async () => {
      const plan = readPlan();
      if (!plan) {
        void vscode8.window.showWarningMessage(NO_PLAN_MESSAGE);
        return;
      }
      const advanced = advancePlanText(plan.text, kstStamp().human);
      if (!advanced) {
        void vscode8.window.showWarningMessage("\uC9C4\uD589\uD560 Now \uD56D\uBAA9\uC774 \uC5C6\uC2B5\uB2C8\uB2E4.");
        return;
      }
      fs6.writeFileSync(plan.file, advanced.text, "utf8");
      void vscode8.window.showInformationMessage("\uD604\uC7AC \uACC4\uD68D\uC744 \uD55C \uB2E8\uACC4 \uC9C4\uD589\uD588\uC2B5\uB2C8\uB2E4.");
      writeAudit(host, "plan", "advanceCurrentPlan", { planFile: planFile(plan.name), completed: advanced.completed, promoted: advanced.promoted });
    }),
    vscode8.commands.registerCommand("vibe-code.setCurrentPlanStatus", async (statusArg) => {
      const plan = readPlan();
      if (!plan) {
        void vscode8.window.showWarningMessage(NO_PLAN_MESSAGE);
        return;
      }
      const explicit = typeof statusArg === "string" && PLAN_STATUSES.includes(statusArg);
      const status = explicit ? statusArg : await vscode8.window.showQuickPick(PLAN_STATUSES, { placeHolder: "\uD604\uC7AC \uACC4\uD68D \uC0C1\uD0DC \uC120\uD0DD" });
      if (!status) return;
      if (status === "done" && !explicit) {
        const open = openPlanItems(plan.text);
        if (open.length > 0) {
          const go = await vscode8.window.showWarningMessage(`\uBBF8\uC644\uB8CC \uD56D\uBAA9\uC774 ${open.length}\uAC1C \uC788\uC2B5\uB2C8\uB2E4:
${open.slice(0, 5).join("\n")}${open.length > 5 ? "\n\u2026" : ""}`, { modal: true }, "\uADF8\uB798\uB3C4 done\uC73C\uB85C \uBCC0\uACBD");
          if (go !== "\uADF8\uB798\uB3C4 done\uC73C\uB85C \uBCC0\uACBD") return;
          writeAudit(host, "plan", "completionGateOverridden", { planFile: planFile(plan.name), open: open.length });
        }
      }
      fs6.writeFileSync(plan.file, touchPlan(plan.text, status, kstStamp().human), "utf8");
      const ticked = status === "done" ? applyPlanCriteria(plan.paths, plan.text) : [];
      void vscode8.window.showInformationMessage("\uD604\uC7AC \uACC4\uD68D \uC0C1\uD0DC\uB97C \uBCC0\uACBD\uD588\uC2B5\uB2C8\uB2E4: " + status + (ticked.length ? ` (\uC644\uB8CC \uAE30\uC900 ${ticked.join(", ")} \uCCB4\uD06C)` : ""));
      writeAudit(host, "plan", "setCurrentPlanStatus", { planFile: planFile(plan.name), status, tickedCriteria: ticked });
    }),
    vscode8.commands.registerCommand("vibe-code.setCurrentPlanPriority", async (priorityArg) => {
      const plan = readPlan();
      if (!plan) {
        void vscode8.window.showWarningMessage(NO_PLAN_MESSAGE);
        return;
      }
      const priority = typeof priorityArg === "string" && PLAN_PRIORITIES.includes(priorityArg) ? priorityArg : await vscode8.window.showQuickPick(PLAN_PRIORITIES, { placeHolder: "\uD604\uC7AC \uACC4\uD68D \uC6B0\uC120\uC21C\uC704 \uC120\uD0DD" });
      if (!priority) return;
      const text = touchPlan(setLine(plan.text, "\uC6B0\uC120\uC21C\uC704", priority), null, kstStamp().human);
      fs6.writeFileSync(plan.file, text, "utf8");
      void vscode8.window.showInformationMessage("\uD604\uC7AC \uACC4\uD68D \uC6B0\uC120\uC21C\uC704\uB97C \uBCC0\uACBD\uD588\uC2B5\uB2C8\uB2E4: " + priority);
      writeAudit(host, "plan", "setCurrentPlanPriority", { planFile: planFile(plan.name), priority });
    }),
    vscode8.commands.registerCommand("vibe-code.linkCurrentPlanToGoal", async () => {
      const plan = readPlan();
      if (!plan) {
        void vscode8.window.showWarningMessage(NO_PLAN_MESSAGE);
        return;
      }
      const paths = ensureWorkspacePaths();
      if (!paths || !fs6.existsSync(paths.current)) {
        void vscode8.window.showWarningMessage("\uD604\uC7AC \uBAA9\uD45C \uD30C\uC77C\uC774 \uC5C6\uC2B5\uB2C8\uB2E4. \uBA3C\uC800 \uD604\uC7AC \uBAA9\uD45C\uB97C \uC900\uBE44\uD558\uC138\uC694.");
        return;
      }
      const goalTitle = currentGoalTitle(paths.current);
      let text = plan.text;
      text = setLine(text, "\uC5F0\uACB0 \uBAA9\uD45C", GOAL_FILE);
      text = setLine(text, "\uC5F0\uACB0 \uBAA9\uD45C \uC81C\uBAA9", goalTitle);
      text = touchPlan(text, null, kstStamp().human);
      fs6.writeFileSync(plan.file, text, "utf8");
      void vscode8.window.showInformationMessage("\uD604\uC7AC \uACC4\uD68D\uC744 \uD604\uC7AC \uBAA9\uD45C\uC640 \uC5F0\uACB0\uD588\uC2B5\uB2C8\uB2E4.");
      writeAudit(host, "plan", "linkCurrentPlanToGoal", { planFile: planFile(plan.name), goalFile: GOAL_FILE, goalTitle });
    }),
    vscode8.commands.registerCommand("vibe-code.archiveDonePlans", async () => {
      const paths = ensureWorkspacePaths();
      if (!paths) return;
      const archived = archiveDonePlans(paths.plans, paths.archive, conflictStamp());
      if (archived.length === 0) {
        void vscode8.window.showInformationMessage("\uBCF4\uAD00\uD560 \uC644\uB8CC \uACC4\uD68D\uC774 \uC5C6\uC2B5\uB2C8\uB2E4.");
        return;
      }
      const renamed = archived.filter((row) => row.dest !== row.name);
      const renamedNote = renamed.length > 0 ? ` (\uAC19\uC740 \uC774\uB984\uC758 \uBCF4\uAD00\uBCF8\uC774 \uC788\uC5B4 ${renamed.length}\uAC1C\uB294 \uC0C8 \uC774\uB984\uC73C\uB85C \uBCF4\uAD00: ${renamed.map((row) => row.dest).join(", ")})` : "";
      void vscode8.window.showInformationMessage("\uC644\uB8CC \uACC4\uD68D\uC744 \uBCF4\uAD00\uD588\uC2B5\uB2C8\uB2E4: " + archived.length + "\uAC1C" + renamedNote);
      writeAudit(host, "plan", "archiveDonePlans", { count: archived.length, files: archived.map((row) => ".vibe-code/plans/archive/" + row.dest) });
    }),
    vscode8.commands.registerCommand("vibe-code.restoreArchivedPlan", async (nameArg) => {
      const paths = ensureWorkspacePaths();
      if (!paths) return;
      const files = listMarkdown(paths.archive);
      if (files.length === 0) {
        void vscode8.window.showWarningMessage("\uBCF5\uC6D0\uD560 \uBCF4\uAD00 \uACC4\uD68D\uC774 \uC5C6\uC2B5\uB2C8\uB2E4.");
        return;
      }
      const picked = await pickPlanFile(files, nameArg, "\uBCF5\uC6D0\uD560 \uBCF4\uAD00 \uACC4\uD68D \uC120\uD0DD");
      if (!picked) return;
      let destName = picked;
      let dest = path4.join(paths.plans, destName);
      if (fs6.existsSync(dest)) {
        const stamp = conflictStamp();
        const parsed = path4.parse(picked);
        destName = `${parsed.name}-restored-${stamp}${parsed.ext || ".md"}`;
        dest = path4.join(paths.plans, destName);
      }
      fs6.renameSync(path4.join(paths.archive, picked), dest);
      setActivePlanName(paths, destName);
      await openFile(dest);
      void vscode8.window.showInformationMessage("\uBCF4\uAD00 \uACC4\uD68D\uC744 \uBCF5\uC6D0\uD588\uC2B5\uB2C8\uB2E4: " + destName);
      writeAudit(host, "plan", "restoreArchivedPlan", { planFile: planFile(destName), source: ".vibe-code/plans/archive/" + picked });
    }),
    vscode8.commands.registerCommand("vibe-code.setActivePlan", async (nameArg) => {
      const paths = ensureWorkspacePaths(true);
      if (!paths) return;
      const files = listMarkdown(paths.plans);
      if (files.length === 0) {
        void vscode8.window.showWarningMessage(NO_PLAN_FILES_MESSAGE);
        return;
      }
      const picked = await pickPlanFile(files, nameArg, "\uD604\uC7AC \uACC4\uD68D\uC73C\uB85C \uC0AC\uC6A9\uD560 \uD30C\uC77C \uC120\uD0DD");
      if (!picked) return;
      setActivePlanName(paths, picked);
      await openFile(path4.join(paths.plans, picked));
      void vscode8.window.showInformationMessage("\uD604\uC7AC \uACC4\uD68D\uC744 \uC120\uD0DD\uD588\uC2B5\uB2C8\uB2E4: " + picked);
      writeAudit(host, "plan", "setActivePlan", { planFile: planFile(picked) });
    }),
    vscode8.commands.registerCommand("vibe-code.selectHighestPriorityPlan", async () => {
      const paths = ensureWorkspacePaths(true);
      if (!paths) return;
      const rows = planMetaList(paths.plans);
      if (rows.length === 0) {
        void vscode8.window.showWarningMessage(NO_PLAN_FILES_MESSAGE);
        return;
      }
      const picked = rows[0];
      setActivePlanName(paths, picked.name);
      await openFile(path4.join(paths.plans, picked.name));
      void vscode8.window.showInformationMessage("\uCD5C\uACE0 \uC6B0\uC120\uC21C\uC704 \uACC4\uD68D\uC744 \uD604\uC7AC \uACC4\uD68D\uC73C\uB85C \uC120\uD0DD\uD588\uC2B5\uB2C8\uB2E4: " + picked.name);
      writeAudit(host, "plan", "selectHighestPriorityPlan", { planFile: planFile(picked.name), priority: picked.priority, status: picked.status, candidateCount: rows.length });
    }),
    vscode8.commands.registerCommand("vibe-code.showPlanCatalog", async () => {
      const paths = ensureWorkspacePaths(true);
      if (!paths) return;
      const activeName = selectPlanName(paths);
      const active = planMetaList(paths.plans);
      const archived = planMetaList(paths.archive);
      const row = (r) => `${r.name} :: ${r.status} :: ${r.priority} :: ${r.title}`;
      output.show(true);
      output.appendLine("\n=== \uACC4\uD68D \uBAA9\uB85D ===");
      output.appendLine("active plan: " + activeName);
      output.appendLine("\n--- active plans ---");
      if (active.length === 0) output.appendLine("  (none)");
      for (const r of active) output.appendLine("  " + (r.name === activeName ? ">" : "-") + " " + row(r));
      output.appendLine("\n--- archived plans ---");
      if (archived.length === 0) output.appendLine("  (none)");
      for (const r of archived) output.appendLine("  - " + row(r));
      output.appendLine("===================\n");
      writeAudit(host, "plan", "showPlanCatalog", { activePlan: planFile(activeName), activeCount: active.length, archivedCount: archived.length });
    }),
    vscode8.commands.registerCommand("vibe-code.showPlanHistory", async () => {
      const paths = ensureWorkspacePaths(true);
      if (!paths) return;
      const items = [];
      if (fs6.existsSync(paths.audit)) {
        const files = fs6.readdirSync(paths.audit).filter((name) => name.endsWith(".jsonl")).sort();
        for (const name of files) {
          for (const line of readUtf8(path4.join(paths.audit, name)).split(/\r?\n/).filter(Boolean)) {
            try {
              const row = JSON.parse(line);
              if (row.kind === "plan") items.push(row);
            } catch {
            }
          }
        }
      }
      const recent = items.slice(-20);
      output.show(true);
      output.appendLine("\n=== \uACC4\uD68D \uC774\uB825 ===");
      if (recent.length === 0) output.appendLine("  (no plan audit entries)");
      for (const row of recent) {
        const file = row.details?.planFile || row.details?.source || "";
        const extra = row.details?.status || row.details?.count || "";
        output.appendLine(`  ${row.ts} :: ${row.action}${file ? ` :: ${file}` : ""}${extra ? ` :: ${extra}` : ""}`);
      }
      output.appendLine("=================\n");
      writeAudit(host, "plan", "showPlanHistory", { count: recent.length });
    }),
    vscode8.commands.registerCommand("vibe-code.showGoalPlanMap", async () => {
      const paths = ensureWorkspacePaths(true);
      if (!paths) return;
      const items = [...linkedPlans(paths.plans, "active"), ...linkedPlans(paths.archive, "archive")];
      output.show(true);
      output.appendLine("\n=== \uBAA9\uD45C-\uACC4\uD68D \uC5F0\uACB0 ===");
      output.appendLine(`goal: ${currentGoalTitle(paths.current)}`);
      output.appendLine("goal file: " + GOAL_FILE);
      if (items.length === 0) output.appendLine("\n  (linked plans \uC5C6\uC74C)");
      for (const r of items) output.appendLine(`  [${r.bucket}] ${r.name} :: ${r.status} :: ${r.priority} :: ${r.title} :: ${r.linkedTitle || r.linked}`);
      output.appendLine("======================\n");
      writeAudit(host, "plan", "showGoalPlanMap", { goalFile: GOAL_FILE, count: items.length });
    })
  );
}
function planBoardRoots() {
  const plan = readPlan();
  if (!plan) {
    return [makeItem("\uD604\uC7AC \uACC4\uD68D \uC5C6\uC74C", { description: "\uBA85\uB839\uC73C\uB85C \uC0DD\uC131", tooltip: "\uCD5C\uC2E0 \uACC4\uD68D \uC5F4\uAE30 \uBA85\uB839\uC73C\uB85C \uACC4\uD68D draft\uB97C \uB9CC\uB4DC\uC138\uC694.", command: OPEN_PLAN, icon: "project" })];
  }
  const meta = planMeta(plan.text, plan.name);
  const progress = progressLabel(countChecks(plan.text), "\uCCB4\uD06C \uC5C6\uC74C");
  const steps = taskLines(section(plan.text, "\uB2E8\uACC4"));
  const now = taskLines(section(plan.text, "Now"));
  const next = taskLines(section(plan.text, "Next"));
  const done = lines(section(plan.text, "Done"));
  const risks = lines(section(plan.text, "Risks"));
  const expanded = vscode8.TreeItemCollapsibleState.Expanded;
  const collapsed = vscode8.TreeItemCollapsibleState.Collapsed;
  return [
    makeItem(meta.title, {
      description: `${progress} \xB7 ${meta.status} \xB7 ${meta.priority}`,
      tooltip: `\uC0C1\uD0DC: ${meta.status}
\uC6B0\uC120\uC21C\uC704: ${meta.priority}
\uD30C\uC77C: .vibe-code/plans/${plan.name}`,
      command: OPEN_PLAN,
      icon: "project"
    }),
    makeItem("\uB2E8\uACC4", { description: progress, tooltip: "\uACC4\uD68D \uB2E8\uACC4 \uCCB4\uD06C\uB9AC\uC2A4\uD2B8", collapsibleState: expanded, icon: "checklist", children: checklistItems(steps, "\uB2E8\uACC4 \uC5C6\uC74C", "check") }),
    makeItem("Now", { description: now.length > 0 ? `${now.length}\uAC1C` : "\uBE44\uC5B4 \uC788\uC74C", tooltip: "\uD604\uC7AC \uC2E4\uD589 \uC791\uC5C5", collapsibleState: expanded, icon: "play", children: checklistItems(now, "Now \uD56D\uBAA9 \uC5C6\uC74C", "circle-outline") }),
    makeItem("Next", { description: next.length > 0 ? `${next.length}\uAC1C` : "\uBE44\uC5B4 \uC788\uC74C", tooltip: "\uB2E4\uC74C \uC2E4\uD589 \uC791\uC5C5", collapsibleState: collapsed, icon: "arrow-right", children: checklistItems(next, "Next \uD56D\uBAA9 \uC5C6\uC74C", "circle-large-outline") }),
    makeItem("Done", { description: done.length > 0 ? `${done.length}\uC904` : "\uC5C6\uC74C", tooltip: "\uC644\uB8CC\uB41C \uC791\uC5C5", collapsibleState: collapsed, icon: "pass", children: noteItems(done, "\uC644\uB8CC \uAE30\uB85D \uC5C6\uC74C") }),
    makeItem("Risks", { description: risks.length > 0 ? `${risks.length}\uC904` : "\uC5C6\uC74C", tooltip: "\uB9AC\uC2A4\uD06C\uC640 \uC81C\uC57D", collapsibleState: collapsed, icon: "warning", children: noteItems(risks, "\uB9AC\uC2A4\uD06C \uC5C6\uC74C") }),
    makeItem("\uBC14\uB85C \uC2E4\uD589", {
      description: "\uBA85\uB839 15\uAC1C",
      tooltip: "\uACC4\uD68D/\uBAA9\uD45C \uAD00\uB828 \uBA85\uB839",
      collapsibleState: collapsed,
      icon: "tools",
      children: [
        commandItem("\uCD5C\uC2E0 \uACC4\uD68D \uC5F4\uAE30", "vibe-code.openLatestPlan", "\uCD5C\uC2E0 \uACC4\uD68D \uC5F4\uAE30", "go-to-file"),
        commandItem("\uD604\uC7AC \uACC4\uD68D \uC9C4\uD589", "vibe-code.advanceCurrentPlan", "\uD604\uC7AC \uACC4\uD68D \uC9C4\uD589", "run"),
        commandItem("\uACC4\uD68D \uC0C1\uD0DC \uBCC0\uACBD", "vibe-code.setCurrentPlanStatus", "\uD604\uC7AC \uACC4\uD68D \uC0C1\uD0DC \uBCC0\uACBD", "symbol-enum"),
        commandItem("\uD604\uC7AC \uACC4\uD68D \uC6B0\uC120\uC21C\uC704 \uBCC0\uACBD", "vibe-code.setCurrentPlanPriority", "\uD604\uC7AC \uACC4\uD68D \uC6B0\uC120\uC21C\uC704 \uBCC0\uACBD", "arrow-up"),
        commandItem("\uD604\uC7AC \uACC4\uD68D \uBAA9\uD45C \uC5F0\uACB0", "vibe-code.linkCurrentPlanToGoal", "\uD604\uC7AC \uACC4\uD68D \uBAA9\uD45C \uC5F0\uACB0", "link"),
        commandItem("\uC644\uB8CC \uAE30\uC900 \uC5F0\uACB0", "vibe-code.linkPlanToCriteria", "\uC644\uB8CC \uAE30\uC900 \uC5F0\uACB0", "checklist"),
        commandItem("\uC644\uB8CC \uACC4\uD68D \uBCF4\uAD00", "vibe-code.archiveDonePlans", "\uC644\uB8CC \uACC4\uD68D \uBCF4\uAD00", "archive"),
        commandItem("\uBCF4\uAD00 \uACC4\uD68D \uBCF5\uC6D0", "vibe-code.restoreArchivedPlan", "\uBCF4\uAD00 \uACC4\uD68D \uBCF5\uC6D0", "history"),
        commandItem("\uD604\uC7AC \uACC4\uD68D \uC120\uD0DD", "vibe-code.setActivePlan", "\uD604\uC7AC \uACC4\uD68D \uC120\uD0DD", "go-to-file"),
        commandItem("\uCD5C\uACE0 \uC6B0\uC120\uC21C\uC704 \uACC4\uD68D \uC120\uD0DD", "vibe-code.selectHighestPriorityPlan", "\uCD5C\uACE0 \uC6B0\uC120\uC21C\uC704 \uACC4\uD68D \uC120\uD0DD", "arrow-up"),
        commandItem("\uACC4\uD68D \uBAA9\uB85D \uBCF4\uAE30", "vibe-code.showPlanCatalog", "\uACC4\uD68D \uBAA9\uB85D \uBCF4\uAE30", "list-tree"),
        commandItem("\uACC4\uD68D \uC774\uB825 \uBCF4\uAE30", "vibe-code.showPlanHistory", "\uACC4\uD68D \uC774\uB825 \uBCF4\uAE30", "history"),
        commandItem("\uBAA9\uD45C-\uACC4\uD68D \uC5F0\uACB0 \uBCF4\uAE30", "vibe-code.showGoalPlanMap", "\uBAA9\uD45C-\uACC4\uD68D \uC5F0\uACB0 \uBCF4\uAE30", "organization"),
        commandItem("\uD604\uC7AC \uBAA9\uD45C \uC5F4\uAE30", "vibe-code.openCurrentGoal", "\uD604\uC7AC \uBAA9\uD45C \uC5F4\uAE30", "target"),
        commandItem("\uBAA9\uD45C \uC0C1\uD0DC \uBCF4\uAE30", "vibe-code.showGoalStatus", "\uBAA9\uD45C \uC0C1\uD0DC \uBCF4\uAE30", "list-tree")
      ]
    })
  ];
}
function linkedPlanItems(rows, emptyLabel) {
  if (rows.length === 0) return [makeItem(emptyLabel, { description: "\uC5C6\uC74C", icon: "circle-slash" })];
  return rows.map((row) => {
    const archived = row.bucket === "archive";
    return makeItem(row.title, {
      description: `${row.bucket} \xB7 ${row.status} \xB7 ${row.priority}`,
      tooltip: `\uD30C\uC77C: .vibe-code/plans${archived ? "/archive" : ""}/${row.name}
\uC6B0\uC120\uC21C\uC704: ${row.priority}
\uC5F0\uACB0 \uBAA9\uD45C: ${row.linkedTitle || row.linked}
\uD074\uB9AD \uB3D9\uC791: ${archived ? "\uBCF5\uC6D0 \uD6C4 active plan \uC9C0\uC815" : "\uD604\uC7AC active plan\uC73C\uB85C \uC120\uD0DD"}`,
      command: archived ? { command: "vibe-code.restoreArchivedPlan", title: "\uBCF4\uAD00 \uACC4\uD68D \uBCF5\uC6D0", arguments: [row.name] } : { command: "vibe-code.setActivePlan", title: "\uD604\uC7AC \uACC4\uD68D \uC120\uD0DD", arguments: [row.name] },
      icon: archived ? "archive" : "project"
    });
  });
}
function goalPlanMapRoots() {
  const paths = ensureWorkspacePaths(true);
  if (!paths) {
    return [makeItem("\uBAA9\uD45C \uC815\uBCF4 \uC5C6\uC74C", { description: "\uD604\uC7AC \uBAA9\uD45C \uD544\uC694", tooltip: "\uD604\uC7AC \uBAA9\uD45C \uD30C\uC77C\uC744 \uB9CC\uB4E4\uAC70\uB098 \uC5EC\uC138\uC694.", command: OPEN_GOAL2, icon: "target" })];
  }
  let goalTitle = "\uD604\uC7AC \uBAA9\uD45C";
  let goalStatus = "(unknown)";
  try {
    if (fs6.existsSync(paths.current)) {
      const text = readUtf8(paths.current);
      goalTitle = headingTitle(text, "\uBAA9\uD45C", goalTitle);
      goalStatus = matchLine(text, "\uC0C1\uD0DC", goalStatus);
    }
  } catch {
  }
  const active = linkedPlans(paths.plans, "active");
  const archived = linkedPlans(paths.archive, "archive");
  const expanded = vscode8.TreeItemCollapsibleState.Expanded;
  const collapsed = vscode8.TreeItemCollapsibleState.Collapsed;
  return [
    makeItem(goalTitle, { description: `${active.length + archived.length} linked \xB7 ${goalStatus}`, tooltip: `goal file: ${GOAL_FILE}
\uC0C1\uD0DC: ${goalStatus}`, command: OPEN_GOAL2, icon: "target" }),
    makeItem("Active Linked Plans", { description: active.length > 0 ? `${active.length}\uAC1C` : "\uC5C6\uC74C", tooltip: "\uD604\uC7AC \uBAA9\uD45C\uC640 \uC5F0\uACB0\uB41C active plan", collapsibleState: expanded, icon: "project", children: linkedPlanItems(active, "linked active plan \uC5C6\uC74C") }),
    makeItem("Archived Linked Plans", { description: archived.length > 0 ? `${archived.length}\uAC1C` : "\uC5C6\uC74C", tooltip: "\uD604\uC7AC \uBAA9\uD45C\uC640 \uC5F0\uACB0\uB41C archived plan", collapsibleState: collapsed, icon: "archive", children: linkedPlanItems(archived, "linked archived plan \uC5C6\uC74C") }),
    makeItem("\uBC14\uB85C \uC2E4\uD589", {
      description: "\uBA85\uB839 3\uAC1C",
      tooltip: "\uBAA9\uD45C-\uACC4\uD68D \uC5F0\uACB0 \uAD00\uB828 \uBA85\uB839",
      collapsibleState: collapsed,
      icon: "tools",
      children: [
        commandItem("\uBAA9\uD45C-\uACC4\uD68D \uC5F0\uACB0 \uBCF4\uAE30", "vibe-code.showGoalPlanMap", "\uBAA9\uD45C-\uACC4\uD68D \uC5F0\uACB0 \uBCF4\uAE30", "organization"),
        commandItem("\uD604\uC7AC \uBAA9\uD45C \uC5F4\uAE30", "vibe-code.openCurrentGoal", "\uD604\uC7AC \uBAA9\uD45C \uC5F4\uAE30", "target"),
        commandItem("\uCD5C\uC2E0 \uACC4\uD68D \uC5F4\uAE30", "vibe-code.openLatestPlan", "\uCD5C\uC2E0 \uACC4\uD68D \uC5F4\uAE30", "go-to-file")
      ]
    })
  ];
}
function registerPlanViews(host) {
  registerTreeView(host, { viewId: "vibe-code.PlanBoard", roots: planBoardRoots, watch: [".vibe-code/plans/*.md"] });
  log(host, "plan board view registered");
  registerTreeView(host, { viewId: "vibe-code.GoalPlanMap", roots: goalPlanMapRoots, watch: [GOAL_FILE, ".vibe-code/plans/*.md", ".vibe-code/plans/archive/*.md"] });
  log(host, "goal plan map view registered");
}
var fs6, path4, vscode8, NO_PLAN_MESSAGE, NO_PLAN_FILES_MESSAGE, PLAN_STATUSES, PLAN_PRIORITIES, isChecklistLine, OPEN_PLAN, OPEN_GOAL2;
var init_plans = __esm({
  "src/features/plans.ts"() {
    "use strict";
    fs6 = __toESM(require("node:fs"));
    path4 = __toESM(require("node:path"));
    vscode8 = __toESM(require("vscode"));
    init_log();
    init_kst();
    init_markdown();
    init_goals();
    init_goal_catalog();
    init_tree();
    init_workspace();
    NO_PLAN_MESSAGE = "\uD604\uC7AC \uACC4\uD68D \uD30C\uC77C\uC774 \uC5C6\uC2B5\uB2C8\uB2E4. \uBA3C\uC800 \uCD5C\uC2E0 \uACC4\uD68D \uC5F4\uAE30 \uBA85\uB839\uC744 \uC2E4\uD589\uD558\uC138\uC694.";
    NO_PLAN_FILES_MESSAGE = "\uC120\uD0DD\uD560 \uACC4\uD68D \uD30C\uC77C\uC774 \uC5C6\uC2B5\uB2C8\uB2E4. \uBA3C\uC800 \uCD5C\uC2E0 \uACC4\uD68D \uC5F4\uAE30 \uBA85\uB839\uC744 \uC2E4\uD589\uD558\uC138\uC694.";
    PLAN_STATUSES = ["draft", "active", "blocked", "done"];
    PLAN_PRIORITIES = ["P0", "P1", "P2", "P3"];
    isChecklistLine = (line) => line.trim().startsWith("- [");
    OPEN_PLAN = { command: "vibe-code.openLatestPlan", title: "\uCD5C\uC2E0 \uACC4\uD68D \uC5F4\uAE30" };
    OPEN_GOAL2 = { command: "vibe-code.openCurrentGoal", title: "\uD604\uC7AC \uBAA9\uD45C \uC5F4\uAE30" };
  }
});

// src/extension.ts
var extension_exports = {};
__export(extension_exports, {
  activate: () => activate,
  deactivate: () => deactivate
});
module.exports = __toCommonJS(extension_exports);

// src/activation.ts
init_log();

// src/features/language.ts
init_log();
init_settings();
async function syncLanguageFromSettings(host) {
  const configured = config().get("language");
  if (configured && typeof configured === "string" && !host.context.globalState.get("language")) {
    await host.context.globalState.update("language", configured);
    log(host, `language synced from settings: ${configured}`);
  }
}
async function initializeLanguage(host) {
  const configured = config().get("language") || "ko";
  const active = host.context.globalState.get("language") ?? configured ?? "ko";
  if (!host.context.globalState.get("language")) {
    await host.context.globalState.update("language", active);
    log(host, "language default initialized: " + active);
  }
  host.changeLanguage(active);
}

// src/features/network-env.ts
var fs = __toESM(require("node:fs"));
init_log();
init_settings();
function applyNetworkEnv(host) {
  const cfg = config();
  const proxy = cfg.get("proxyUrl");
  if (proxy && typeof proxy === "string") {
    process.env.HTTPS_PROXY = process.env.HTTPS_PROXY || proxy;
    process.env.HTTP_PROXY = process.env.HTTP_PROXY || proxy;
    process.env.https_proxy = process.env.https_proxy || proxy;
    process.env.http_proxy = process.env.http_proxy || proxy;
    log(host, `proxy applied: ${proxy}`);
  }
  const ca = cfg.get("extraCaCertsPath");
  if (ca && typeof ca === "string") {
    if (fs.existsSync(ca)) {
      process.env.NODE_EXTRA_CA_CERTS = ca;
      log(host, `NODE_EXTRA_CA_CERTS applied: ${ca}`);
    } else {
      log(host, `CA path not found: ${ca}`);
    }
  }
}

// src/features/status-bar.ts
var https = __toESM(require("node:https"));
var vscode2 = __toESM(require("vscode"));
init_log();
init_settings();
var MODE_LABELS = { code: "Code", architect: "Architect", debug: "Debug", ask: "Ask", "code-reviewer": "Review" };
function checkNetwork() {
  return new Promise((resolve3) => {
    try {
      const req = https.request({ host: "1.1.1.1", port: 443, method: "HEAD", timeout: 2e3, path: "/" }, () => {
        resolve3("online");
        req.destroy();
      });
      req.on("error", () => resolve3("offline"));
      req.on("timeout", () => {
        resolve3("offline");
        req.destroy();
      });
      req.end();
    } catch {
      resolve3("offline");
    }
  });
}
function createModeStatusBar(host) {
  const { context } = host;
  const item = vscode2.window.createStatusBarItem(vscode2.StatusBarAlignment.Right, 100);
  item.command = "vibe-code.SidebarProvider.focus";
  let netState = "checking";
  const refreshText = () => {
    try {
      const mode = context.globalState.get("mode") || "code";
      const label = MODE_LABELS[mode] || mode;
      const tokens = context.globalState.get("vibeCode.sessionTokens") || 0;
      const tokenText = tokens > 1e3 ? ` \xB7 ${(tokens / 1e3).toFixed(1)}k` : tokens > 0 ? ` \xB7 ${tokens}` : "";
      const icon = netState === "online" ? "$(cloud)" : "$(cloud-offline)";
      item.text = `${icon} Vibe: ${label}${tokenText}`;
      item.tooltip = `Vibe Code v${host.pkg.version} \u2014 ${label} \uBAA8\uB4DC / \uB124\uD2B8\uC6CC\uD06C: ${netState === "online" ? "\uC628\uB77C\uC778" : "\uC624\uD504\uB77C\uC778"} / \uC790\uC728\uC131: ${context.globalState.get("vibeCode.autonomyApplied") || "(default)"}. \uD074\uB9AD\uD558\uC5EC \uC0AC\uC774\uB4DC\uBC14 \uC5F4\uAE30.`;
    } catch {
      item.text = "$(rocket) Vibe Code";
    }
  };
  const refreshNetwork = async () => {
    const mode = config().get("offlineMode") || "auto";
    const previous = netState;
    netState = mode === "offline" ? "offline" : mode === "online" ? "online" : await checkNetwork();
    if (previous !== netState) log(host, `network state: ${netState}`);
    refreshText();
  };
  void refreshNetwork();
  item.show();
  const textInterval = setInterval(refreshText, 15e3);
  const networkInterval = setInterval(() => void refreshNetwork(), 6e4);
  context.subscriptions.push(
    item,
    { dispose: () => clearInterval(textInterval) },
    { dispose: () => clearInterval(networkInterval) },
    vscode2.workspace.onDidChangeConfiguration((event) => {
      if (event.affectsConfiguration("vibe-code.offlineMode")) void refreshNetwork();
    })
  );
  log(host, "status bar item created");
}

// src/activation.ts
init_goals();
init_plans();

// src/features/seeds.ts
var fs7 = __toESM(require("node:fs"));
var path5 = __toESM(require("node:path"));
init_log();
init_workspace();
function asset(host, ...parts) {
  return path5.join(host.context.extensionPath, "assets", "demo", ...parts);
}
async function seedDemoScenarios(host) {
  if (host.context.globalState.get("vibeCode.demoSeeded")) return;
  const src = asset(host, "scenarios.md");
  if (!fs7.existsSync(src)) return;
  const ws = workspaceRoot();
  if (!ws) return;
  const destDir = path5.join(ws, ".vibe-code");
  const dest = path5.join(destDir, "demo-scenarios.md");
  if (!fs7.existsSync(dest)) {
    fs7.mkdirSync(destDir, { recursive: true });
    fs7.copyFileSync(src, dest);
    log(host, `demo scenarios seeded: ${dest}`);
  }
  await host.context.globalState.update("vibeCode.demoSeeded", true);
}
function seedTemplate(host, fileName, templateName, message) {
  const ws = workspaceRoot();
  if (!ws) return;
  const dest = path5.join(ws, fileName);
  if (fs7.existsSync(dest)) return;
  const template = asset(host, templateName);
  if (!fs7.existsSync(template)) return;
  fs7.copyFileSync(template, dest);
  log(host, message);
}
function seedVibeignore(host) {
  seedTemplate(host, ".vibeignore", ".vibeignore.template", "seeded .vibeignore at workspace root");
}
function seedVibemodes(host) {
  seedTemplate(host, ".vibemodes", ".vibemodes.template", "seeded .vibemodes with 4 Korean modes");
}
function seedSlashCommands(host) {
  const ws = workspaceRoot();
  if (!ws) return;
  const src = asset(host, ".vibe", "commands");
  if (!fs7.existsSync(src)) return;
  const dest = path5.join(ws, ".vibe", "commands");
  fs7.mkdirSync(dest, { recursive: true });
  const legacy = path5.join(dest, "\uBAA9\uD45C.md");
  const goal = path5.join(dest, "goal.md");
  if (fs7.existsSync(legacy)) {
    if (!fs7.existsSync(goal)) {
      fs7.renameSync(legacy, goal);
      log(host, `migrated legacy slash command: ${legacy} -> ${goal}`);
    } else {
      fs7.unlinkSync(legacy);
      log(host, `removed legacy slash command: ${legacy}`);
    }
  }
  let copied = 0;
  for (const name of fs7.readdirSync(src)) {
    if (!name.endsWith(".md")) continue;
    const target = path5.join(dest, name);
    if (fs7.existsSync(target)) continue;
    fs7.copyFileSync(path5.join(src, name), target);
    copied++;
  }
  if (copied > 0) log(host, `seeded ${copied} Korean slash command files at .vibe/commands/`);
}
function seedMcpRecommendations(host) {
  const ws = workspaceRoot();
  if (!ws) return;
  const dest = path5.join(ws, ".vibe-code", "mcp-recommendations.json");
  if (fs7.existsSync(dest)) return;
  const src = asset(host, "mcp-recommendations.json");
  if (!fs7.existsSync(src)) return;
  fs7.mkdirSync(path5.dirname(dest), { recursive: true });
  fs7.copyFileSync(src, dest);
  log(host, `seeded MCP recommendations: ${dest}`);
}

// src/features/defaults.ts
init_log();
init_settings();
async function applyTelemetryDefault(host) {
  const current = host.context.globalState.get("telemetrySetting");
  if (!current || current === "unset") {
    await host.context.globalState.update("telemetrySetting", "disabled");
    log(host, "telemetry disabled by default (privacy-first)");
  }
}
async function applyDefaultMode(host) {
  if (!host.context.globalState.get("mode")) {
    await host.context.globalState.update("mode", "architect");
    log(host, "default mode set to architect (think-first workflow)");
  }
}
var AUTONOMY_PRESETS = {
  safe: {
    alwaysAllowReadOnly: false,
    alwaysAllowWrite: false,
    alwaysAllowExecute: false,
    alwaysAllowBrowser: false,
    alwaysAllowMcp: false,
    alwaysAllowModeSwitch: false,
    alwaysAllowSubtasks: false,
    alwaysAllowUpdateTodoList: false
  },
  assist: {
    alwaysAllowReadOnly: true,
    alwaysAllowWrite: false,
    alwaysAllowExecute: false,
    alwaysAllowBrowser: false,
    alwaysAllowMcp: false,
    alwaysAllowModeSwitch: true,
    alwaysAllowSubtasks: false,
    alwaysAllowUpdateTodoList: true
  },
  auto: {
    alwaysAllowReadOnly: true,
    alwaysAllowWrite: true,
    alwaysAllowExecute: false,
    alwaysAllowBrowser: true,
    alwaysAllowMcp: true,
    alwaysAllowModeSwitch: true,
    alwaysAllowSubtasks: true,
    alwaysAllowUpdateTodoList: true
  },
  yolo: {
    alwaysAllowReadOnly: true,
    alwaysAllowWrite: true,
    alwaysAllowExecute: true,
    alwaysAllowBrowser: true,
    alwaysAllowMcp: true,
    alwaysAllowModeSwitch: true,
    alwaysAllowSubtasks: true,
    alwaysAllowUpdateTodoList: true
  }
};
async function writeAutonomyPreset(host, autonomy, overwrite) {
  const preset = AUTONOMY_PRESETS[autonomy];
  for (const key of Object.keys(preset)) {
    if (overwrite || host.context.globalState.get(key) === void 0) {
      await host.context.globalState.update(key, preset[key]);
    }
  }
  await host.context.globalState.update("vibeCode.autonomyApplied", autonomy);
  log(host, `autonomy preset applied: ${autonomy}`);
}
async function applyAutonomyPreset(host) {
  const autonomy = config().get("autonomy");
  if (!autonomy || host.context.globalState.get("vibeCode.autonomyApplied")) return;
  if (!(autonomy in AUTONOMY_PRESETS)) return;
  await writeAutonomyPreset(host, autonomy, false);
}

// src/features/journal.ts
var fs8 = __toESM(require("node:fs"));
var path6 = __toESM(require("node:path"));
var vscode9 = __toESM(require("vscode"));
init_log();
init_kst();
init_workspace();
function journalTemplate(ymd, clock) {
  return `# Vibe Code \uC791\uC5C5 \uC77C\uC9C0 \u2014 ${ymd}

## \uC138\uC158

- ${clock} \u2014 \uD655\uC7A5 \uD65C\uC131\uD654

## \uC791\uC5C5 \uB0B4\uC6A9

(\uC9C1\uC811 \uAE30\uB85D\uD558\uAC70\uB098 \uC5D0\uC774\uC804\uD2B8\uC5D0\uAC8C \uC694\uCCAD\uD558\uC138\uC694: "\uC624\uB298 \uC791\uC5C5 \uC77C\uC9C0\uC5D0 \uB2E4\uC74C \uB0B4\uC6A9 \uCD94\uAC00\uD574\uC918 ...")

## \uB2E4\uC74C \uC138\uC158\uC744 \uC704\uD55C \uB178\uD2B8

`;
}
function initJournal(host) {
  const ws = workspaceRoot();
  if (!ws) return;
  const journalDir = path6.join(ws, ".vibe-code", "journal");
  fs8.mkdirSync(journalDir, { recursive: true });
  const ymd = kstDate();
  const file = path6.join(journalDir, `${ymd}.md`);
  const clock = kstClock();
  if (!fs8.existsSync(file)) fs8.writeFileSync(file, journalTemplate(ymd, clock));
  else fs8.appendFileSync(file, `- ${clock} \u2014 \uD655\uC7A5 \uC7AC\uD65C\uC131\uD654
`);
  for (const dirName of ["plans", "goals", "sessions", "checkpoints"]) {
    fs8.mkdirSync(path6.join(ws, ".vibe-code", dirName), { recursive: true });
  }
  const previous = fs8.readdirSync(journalDir).filter((name) => name.endsWith(".md") && name !== `${ymd}.md`).sort().reverse();
  if (previous.length > 0) log(host, `previous journal: .vibe-code/journal/${previous[0]}`);
  log(host, `journal: .vibe-code/journal/${ymd}.md`);
}
function registerJournalCommands(host) {
  host.context.subscriptions.push(
    vscode9.commands.registerCommand("vibe-code.openJournal", async () => {
      const ws = workspaceRoot();
      if (!ws) {
        void vscode9.window.showWarningMessage(NO_WORKSPACE_MESSAGE);
        return;
      }
      const file = path6.join(ws, ".vibe-code", "journal", `${kstDate()}.md`);
      if (fs8.existsSync(file)) await openFile(file);
      else void vscode9.window.showInformationMessage(`\uC624\uB298 \uC791\uC5C5 \uC77C\uC9C0\uAC00 \uC544\uC9C1 \uC5C6\uC2B5\uB2C8\uB2E4: ${file}`);
    })
  );
}

// src/features/diagnostics.ts
var fs10 = __toESM(require("node:fs"));
var https2 = __toESM(require("node:https"));
var path8 = __toESM(require("node:path"));
var vscode10 = __toESM(require("vscode"));
init_log();
init_settings();

// src/features/i18n-overrides.ts
var fs9 = __toESM(require("node:fs"));
var os = __toESM(require("node:os"));
var path7 = __toESM(require("node:path"));
function isTree(value) {
  return !!value && typeof value === "object" && !Array.isArray(value);
}
function deepMerge(base, patch) {
  for (const key of Object.keys(patch)) {
    const next = patch[key];
    const current = base[key];
    if (isTree(next) && isTree(current)) deepMerge(current, next);
    else base[key] = next;
  }
  return base;
}
function userLocaleDir() {
  return path7.join(os.homedir(), ".vibe-code", "locales");
}
function mergeLocaleOverrides(resources) {
  const root = userLocaleDir();
  if (!fs9.existsSync(root)) return;
  const languages4 = fs9.readdirSync(root, { withFileTypes: true }).filter((entry) => entry.isDirectory());
  for (const language of languages4) {
    const lang = language.name;
    const langDir = path7.join(root, lang);
    resources[lang] = resources[lang] || {};
    for (const file of fs9.readdirSync(langDir).filter((name) => name.endsWith(".json"))) {
      try {
        const ns = file.replace(/\.json$/, "");
        const data = JSON.parse(fs9.readFileSync(path7.join(langDir, file), "utf8"));
        resources[lang][ns] = deepMerge(resources[lang][ns] || {}, data);
        console.log(`[vibe-code] override loaded: ${lang}/${ns}`);
      } catch {
      }
    }
  }
}

// src/features/diagnostics.ts
init_workspace();
function registerShowContextStats(host) {
  const { context, output } = host;
  context.subscriptions.push(
    vscode10.commands.registerCommand("vibe-code.showContextStats", async () => {
      const state = context.globalState;
      const stats = {
        mode: state.get("mode") || "code",
        language: state.get("language") || "(default)",
        autonomy: state.get("vibeCode.autonomyApplied") || "(unset)",
        telemetry: state.get("telemetrySetting") || "(unset)",
        sessionTokens: state.get("vibeCode.sessionTokens") || 0,
        autoCondenseContextPercent: state.get("autoCondenseContextPercent") || "(default 75)",
        alwaysAllowReadOnly: state.get("alwaysAllowReadOnly"),
        alwaysAllowWrite: state.get("alwaysAllowWrite"),
        alwaysAllowExecute: state.get("alwaysAllowExecute"),
        alwaysAllowBrowser: state.get("alwaysAllowBrowser"),
        alwaysAllowMcp: state.get("alwaysAllowMcp")
      };
      output.show(true);
      output.appendLine("\n=== Vibe Code \uCEE8\uD14D\uC2A4\uD2B8 \uC0AC\uC6A9\uB7C9 ===");
      for (const key of Object.keys(stats)) output.appendLine(`  ${key.padEnd(28)} ${stats[key]}`);
      output.appendLine("================================\n");
    })
  );
}
function directorySize(dir) {
  let total = 0;
  try {
    for (const entry of fs10.readdirSync(dir, { withFileTypes: true })) {
      const full = path8.join(dir, entry.name);
      total += entry.isDirectory() ? directorySize(full) : fs10.statSync(full).size;
    }
  } catch {
  }
  return total;
}
function registerDiagnostics(host) {
  const { context, output } = host;
  context.subscriptions.push(
    vscode10.commands.registerCommand("vibe-code.showIndexInfo", async () => {
      const ws = workspaceRoot();
      output.show(true);
      output.appendLine("\n=== Vibe Code \uC784\uBCA0\uB529 \uC778\uB371\uC2A4 ===");
      try {
        const globalStorage = context.globalStorageUri?.fsPath;
        output.appendLine(`globalStorage: ${globalStorage}`);
        if (globalStorage && fs10.existsSync(globalStorage)) {
          output.appendLine(`  total size: ${(directorySize(globalStorage) / 1024 / 1024).toFixed(2)} MB`);
        }
      } catch (error) {
        output.appendLine(`  (size scan failed: ${error})`);
      }
      if (ws) {
        const vectorDb = path8.join(ws, ".vibe-code", "vectordb");
        if (fs10.existsSync(vectorDb)) output.appendLine(`workspace index: ${vectorDb}`);
        else output.appendLine("workspace index: (none \u2014 codebase search not initialized)");
      }
      output.appendLine("================================\n");
    }),
    vscode10.commands.registerCommand("vibe-code.healthCheck", async () => {
      output.show(true);
      output.appendLine("\n=== Vibe Code \uD55C\uAD6D\uC5B4 \uD658\uACBD \uC810\uAC80 ===");
      const cfg = config();
      const state = context.globalState;
      const rows = [
        ["\uC5B8\uC5B4 \uC124\uC815 (settings)", cfg.get("language") || "(\uBBF8\uC124\uC815)"],
        ["\uC5B8\uC5B4 (globalState)", state.get("language") || "(\uBBF8\uC124\uC815)"],
        ["\uD1A4", cfg.get("tone") || "(\uBBF8\uC124\uC815)"],
        ["\uC790\uC728\uC131 \uB4F1\uAE09", cfg.get("autonomy") || "(\uBBF8\uC124\uC815)"],
        ["\uC801\uC6A9\uB41C \uC790\uC728\uC131", state.get("vibeCode.autonomyApplied") || "(\uBBF8\uC801\uC6A9)"],
        ["\uD154\uB808\uBA54\uD2B8\uB9AC", state.get("telemetrySetting") || "(\uBBF8\uC124\uC815)"],
        ["\uAC70\uBD80 \uBA85\uB839 \uAC1C\uC218", (cfg.get("deniedCommands") || []).length],
        ["\uD5C8\uC6A9 \uBA85\uB839 \uAC1C\uC218", (cfg.get("allowedCommands") || []).length],
        ["\uCF54\uB4DC \uC561\uC158 \uD65C\uC131\uD654", cfg.get("enableCodeActions")],
        ["\uCF54\uB4DC \uC2E4\uD589 \uD0C0\uC784\uC544\uC6C3", cfg.get("commandExecutionTimeout") || 0],
        ["VS Code \uBC84\uC804", vscode10.version],
        ["\uD655\uC7A5 \uBC84\uC804", host.pkg.version]
      ];
      for (const [key, value] of rows) output.appendLine(`  ${key.padEnd(24)} ${value}`);
      const ws = workspaceRoot();
      if (ws) {
        const checks = [
          [".vibeignore", fs10.existsSync(path8.join(ws, ".vibeignore"))],
          [".vibemodes", fs10.existsSync(path8.join(ws, ".vibemodes"))],
          [".vibe/commands \uC2AC\uB798\uC2DC", fs10.existsSync(path8.join(ws, ".vibe", "commands"))],
          [".vibe-code/journal", fs10.existsSync(path8.join(ws, ".vibe-code", "journal"))],
          [".vibe-code/demo-scenarios.md", fs10.existsSync(path8.join(ws, ".vibe-code", "demo-scenarios.md"))]
        ];
        output.appendLine("");
        output.appendLine("\uC6CC\uD06C\uC2A4\uD398\uC774\uC2A4 \uD30C\uC77C:");
        for (const [key, ok] of checks) output.appendLine(`  ${ok ? "OK " : "-- "} ${key}`);
      }
      const overrides = userLocaleDir();
      output.appendLine("");
      output.appendLine(`\uC0AC\uC6A9\uC790 \uC624\uBC84\uB77C\uC774\uB4DC: ${fs10.existsSync(overrides) ? "\uAC10\uC9C0\uB428 (" + overrides + ")" : "\uC5C6\uC74C"}`);
      output.appendLine("================================\n");
    })
  );
  log(host, "healthCheck + showIndexInfo commands registered");
}
function probeHost(hostName, port = 443, timeout = 2e3) {
  return new Promise((resolve3) => {
    try {
      const req = https2.request({ host: hostName, port, method: "HEAD", timeout, path: "/" }, (res) => {
        resolve3({ host: hostName, ok: true, status: res.statusCode });
        res.resume();
        req.destroy();
      });
      req.on("error", (error) => resolve3({ host: hostName, ok: false, err: error.code || error.message }));
      req.on("timeout", () => {
        resolve3({ host: hostName, ok: false, err: "timeout" });
        req.destroy();
      });
      req.end();
    } catch (error) {
      resolve3({ host: hostName, ok: false, err: String(error) });
    }
  });
}
var AUDIT_HOSTS = ["1.1.1.1", "api.anthropic.com", "api.openai.com", "registry.npmjs.org", "github.com", "huggingface.co"];
function registerAuditNetwork(host) {
  const { context, output } = host;
  context.subscriptions.push(
    vscode10.commands.registerCommand("vibe-code.auditNetwork", async () => {
      output.show(true);
      output.appendLine("\n=== Vibe Code \uC678\uBD80 \uD1B5\uC2E0 \uC810\uAC80 ===");
      output.appendLine(`offlineMode \uC124\uC815: ${config().get("offlineMode") || "auto"}`);
      output.appendLine(`HTTPS_PROXY: ${process.env.HTTPS_PROXY || process.env.HTTP_PROXY || "(\uC5C6\uC74C)"}`);
      output.appendLine(`NODE_EXTRA_CA_CERTS: ${process.env.NODE_EXTRA_CA_CERTS || "(\uC5C6\uC74C)"}`);
      output.appendLine("\n--- \uC678\uBD80 \uD638\uC2A4\uD2B8 \uB3C4\uB2EC\uC131 (HEAD, 2\uCD08 timeout) ---");
      for (const name of AUDIT_HOSTS) {
        const result = await probeHost(name);
        output.appendLine(`  ${result.ok ? "\u2713" : "\u2717"} ${name.padEnd(28)} ${result.ok ? "(" + result.status + ")" : result.err}`);
      }
      output.appendLine("\n--- API Provider \uD655\uC778 ---");
      const state = context.globalState;
      const apiConfiguration = state.get("apiConfiguration");
      output.appendLine(`  \uD604\uC7AC \uD504\uB85C\uBC14\uC774\uB354: ${state.get("apiProvider") || apiConfiguration?.apiProvider || "(\uBBF8\uC124\uC815)"}`);
      output.appendLine(`  Base URL (\uC788\uB2E4\uBA74): ${apiConfiguration?.openAiBaseUrl || "(N/A)"}`);
      output.appendLine("\n--- MCP \uC11C\uBC84 (\uC0AC\uC6A9\uC790 \uB4F1\uB85D\uBD84) ---");
      const servers = state.get("mcpServers") || {};
      const names = Object.keys(servers);
      if (names.length === 0) output.appendLine("  (\uB4F1\uB85D\uB41C MCP \uC11C\uBC84 \uC5C6\uC74C)");
      for (const name of names) {
        const server = servers[name];
        const usesNpx = server?.command === "npx" || (server?.args || []).some((arg) => typeof arg === "string" && arg.startsWith("@"));
        output.appendLine(`  ${usesNpx ? "\u26A0" : "\u2713"} ${name} ${usesNpx ? "(npx \u2014 \uC778\uD130\uB137 \uD544\uC694 \uAC00\uB2A5\uC131)" : ""}`);
      }
      output.appendLine("\n--- \uAD8C\uC7A5 ---");
      output.appendLine("  \xB7 \uC678\uBD80 \uD638\uC2A4\uD2B8\uAC00 \uBAA8\uB450 \u2717 \uC774\uBA74 OFFLINE-FIRST \uBAA8\uB4DC\uAC00 \uC801\uD569\uD569\uB2C8\uB2E4 (settings: vibe-code.offlineMode = offline).");
      output.appendLine("  \xB7 \uD504\uB85D\uC2DC\uAC00 \uC788\uB2E4\uBA74 vibe-code.proxyUrl \uC124\uC815.");
      output.appendLine("  \xB7 \uC0AC\uB0B4 self-signed \uC778\uC99D\uC11C\uAC00 \uC788\uB2E4\uBA74 vibe-code.extraCaCertsPath \uC124\uC815.");
      output.appendLine("================================\n");
    })
  );
  log(host, "auditNetwork command registered");
}

// src/features/team-config.ts
var import_node_child_process = require("node:child_process");
var fs11 = __toESM(require("node:fs"));
var path9 = __toESM(require("node:path"));
var vscode11 = __toESM(require("vscode"));
init_log();
init_workspace();
function loadAdmZip() {
  return require("adm-zip");
}
var quote = (value) => '\\"' + value + '\\"';
function registerTeamConfig(host) {
  const { context } = host;
  context.subscriptions.push(
    vscode11.commands.registerCommand("vibe-code.exportTeamConfig", async () => {
      const ws = workspaceRoot();
      if (!ws) {
        void vscode11.window.showWarningMessage(NO_WORKSPACE_MESSAGE);
        return;
      }
      const dest = path9.join(ws, ".vibe-code", "team-config.zip");
      const candidates = [
        path9.join(ws, ".vibemodes"),
        path9.join(ws, ".vibeignore"),
        path9.join(ws, ".vibe", "commands"),
        path9.join(ws, ".vibe-code", "mcp-recommendations.json")
      ];
      const present = candidates.filter((p) => fs11.existsSync(p));
      if (present.length === 0) {
        void vscode11.window.showInformationMessage("\uB0B4\uBCF4\uB0BC \uD300 \uC124\uC815 \uD30C\uC77C\uC774 \uC5C6\uC2B5\uB2C8\uB2E4.");
        return;
      }
      try {
        const AdmZip = loadAdmZip();
        const zip = new AdmZip();
        for (const p of present) {
          const rel = path9.relative(ws, p);
          if (fs11.statSync(p).isDirectory()) zip.addLocalFolder(p, rel);
          else zip.addLocalFile(p, path9.dirname(rel));
        }
        zip.writeZip(dest);
        void vscode11.window.showInformationMessage(`\uD300 \uC124\uC815\uC744 \uB0B4\uBCF4\uB0C8\uC2B5\uB2C8\uB2E4: ${dest}`);
        log(host, `team config exported (${present.length} items) to ${dest}`);
      } catch {
        try {
          const command = `powershell -NoProfile -Command "Compress-Archive -Path ${present.map(quote).join(",")} -DestinationPath ${quote(dest)} -Force"`;
          (0, import_node_child_process.exec)(command, (error) => {
            if (error) void vscode11.window.showErrorMessage(`\uD300 \uC124\uC815 \uB0B4\uBCF4\uB0B4\uAE30 \uC2E4\uD328: ${error.message}`);
            else {
              void vscode11.window.showInformationMessage(`\uD300 \uC124\uC815\uC744 \uB0B4\uBCF4\uB0C8\uC2B5\uB2C8\uB2E4: ${dest}`);
              log(host, `team config exported via PowerShell to ${dest}`);
            }
          });
        } catch (error) {
          void vscode11.window.showErrorMessage(`\uD300 \uC124\uC815 \uB0B4\uBCF4\uB0B4\uAE30 \uC2E4\uD328: ${error}`);
        }
      }
    }),
    vscode11.commands.registerCommand("vibe-code.importTeamConfig", async () => {
      const ws = workspaceRoot();
      if (!ws) {
        void vscode11.window.showWarningMessage(NO_WORKSPACE_MESSAGE);
        return;
      }
      const picked = await vscode11.window.showOpenDialog({
        canSelectFiles: true,
        canSelectFolders: false,
        canSelectMany: false,
        filters: { "\uD300 \uC124\uC815": ["zip"] },
        title: "\uD300 \uC124\uC815 \uD30C\uC77C \uC120\uD0DD"
      });
      if (!picked || picked.length === 0) return;
      const src = picked[0].fsPath;
      try {
        const AdmZip = loadAdmZip();
        new AdmZip(src).extractAllTo(ws, true);
        void vscode11.window.showInformationMessage("\uD300 \uC124\uC815\uC744 \uAC00\uC838\uC654\uC2B5\uB2C8\uB2E4. \uC6CC\uD06C\uC2A4\uD398\uC774\uC2A4\uB97C \uB2E4\uC2DC \uB85C\uB4DC\uD558\uC138\uC694.", "\uB2E4\uC2DC \uB85C\uB4DC").then((choice) => {
          if (choice === "\uB2E4\uC2DC \uB85C\uB4DC") void vscode11.commands.executeCommand("workbench.action.reloadWindow");
        });
      } catch {
        (0, import_node_child_process.exec)(`powershell -NoProfile -Command "Expand-Archive -LiteralPath ${quote(src)} -DestinationPath ${quote(ws)} -Force"`, (error) => {
          if (error) void vscode11.window.showErrorMessage(`\uAC00\uC838\uC624\uAE30 \uC2E4\uD328: ${error.message}`);
          else void vscode11.window.showInformationMessage("\uD300 \uC124\uC815\uC744 \uAC00\uC838\uC654\uC2B5\uB2C8\uB2E4.");
        });
      }
    })
  );
  log(host, "team config export/import commands registered");
}

// src/features/update-check.ts
var fs12 = __toESM(require("node:fs"));
var path10 = __toESM(require("node:path"));
var vscode12 = __toESM(require("vscode"));
init_log();
init_settings();

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
  const versions = fs12.readdirSync(dir).map((name) => name.match(/^vibe-code-([0-9.]+)\.vsix$/)?.[1]).filter((v) => !!v).sort(compareVersions);
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
  sources.push({ path: path10.join(context.extensionPath, "..", "..", "..", "projects", "vibe-code", "release"), label: "\uB85C\uCEEC release" });
  for (const source of sources) {
    try {
      if (!fs12.existsSync(source.path)) continue;
      const latest = latestVsixVersion(source.path);
      if (!latest || compareVersions(latest, host.pkg.version) <= 0) continue;
      const vsix = path10.join(source.path, `vibe-code-${latest}.vsix`);
      void vscode12.window.showInformationMessage(`Vibe Code \uC0C8 \uBC84\uC804 \uBC1C\uACAC (${source.label}): ${latest} \u2192 \uD604\uC7AC ${host.pkg.version}`, "\uC124\uCE58", "\uB098\uC911\uC5D0").then((choice) => {
        if (choice === "\uC124\uCE58") void vscode12.commands.executeCommand("workbench.extensions.installExtension", vscode12.Uri.file(vsix));
      });
      log(host, `update available from ${source.label}: ${latest}`);
      break;
    } catch (error) {
      log(host, `update source check failed (${source.label}): ${error}`);
    }
  }
}

// src/features/welcome.ts
var fs13 = __toESM(require("node:fs"));
var path11 = __toESM(require("node:path"));
var vscode13 = __toESM(require("vscode"));
init_log();
init_workspace();
async function showWelcome(host) {
  const { context } = host;
  if (context.globalState.get("vibeCode.welcomed")) return;
  await context.globalState.update("vibeCode.welcomed", true);
  const lang = context.globalState.get("language") || "ko";
  if (lang === "ko") {
    void vscode13.window.showInformationMessage("Vibe Code \uC5D0 \uC624\uC2E0 \uAC83\uC744 \uD658\uC601\uD569\uB2C8\uB2E4! \uD55C\uAD6D\uC5B4 \uD658\uACBD\uC73C\uB85C \uC124\uC815\uB418\uC5B4 \uC788\uC2B5\uB2C8\uB2E4.", "\uC2DC\uC791 \uC124\uC815", "\uC0AC\uC774\uB4DC\uBC14 \uC5F4\uAE30", "\uC0AC\uC6A9 \uAC00\uC774\uB4DC", "\uB098\uC911\uC5D0").then((choice) => {
      if (choice === "\uC2DC\uC791 \uC124\uC815") void vscode13.commands.executeCommand("vibe-code.setupWizard");
      else if (choice === "\uC0AC\uC774\uB4DC\uBC14 \uC5F4\uAE30") void vscode13.commands.executeCommand("vibe-code.SidebarProvider.focus");
      else if (choice === "\uC0AC\uC6A9 \uAC00\uC774\uB4DC") {
        const readme = path11.join(context.extensionPath, "readme.ko.md");
        if (fs13.existsSync(readme)) void openFile(readme);
      }
    });
  } else {
    void vscode13.window.showInformationMessage("Welcome to Vibe Code!", "Setup", "Open Sidebar", "Later").then((choice) => {
      if (choice === "Setup") void vscode13.commands.executeCommand("vibe-code.setupWizard");
      else if (choice === "Open Sidebar") void vscode13.commands.executeCommand("vibe-code.SidebarProvider.focus");
    });
  }
  log(host, `welcome shown (lang: ${lang})`);
}

// src/features/vibe-coders-proxy.ts
var fs14 = __toESM(require("node:fs"));
var http2 = __toESM(require("node:http"));
var https4 = __toESM(require("node:https"));
var path12 = __toESM(require("node:path"));
var vscode15 = __toESM(require("vscode"));
init_log();
init_settings();
init_workspace();

// src/features/usage.ts
var http = __toESM(require("node:http"));
var https3 = __toESM(require("node:https"));
var vscode14 = __toESM(require("vscode"));
init_log();
init_settings();
init_workspace();
function proxyOrigin() {
  const base = stringSetting("vibeCodersBaseUrl", "http://localhost:8080/v1");
  try {
    const url = new URL(base);
    return url.origin;
  } catch {
    return "http://localhost:8080";
  }
}
function fetchUsageReport(apiKey, window21, timeoutMs = 3e3) {
  return new Promise((resolve3) => {
    try {
      const url = new URL(`/me/report?window=${window21}`, proxyOrigin());
      const client = url.protocol === "https:" ? https3 : http;
      const req = client.request(
        { method: "GET", hostname: url.hostname, port: url.port, path: url.pathname + url.search, timeout: timeoutMs, headers: { Authorization: `Bearer ${apiKey}`, Accept: "application/json" } },
        (res) => {
          let body = "";
          res.setEncoding("utf8");
          res.on("data", (chunk) => body += chunk);
          res.on("end", () => {
            const status = res.statusCode ?? 0;
            if (status < 200 || status >= 300) return resolve3({ ok: false, status, error: body.slice(0, 200) });
            try {
              resolve3({ ok: true, status, report: JSON.parse(body) });
            } catch (error) {
              resolve3({ ok: false, status, error: `invalid JSON: ${error}` });
            }
          });
        }
      );
      req.on("error", (error) => resolve3({ ok: false, error: error.code || error.message }));
      req.on("timeout", () => {
        req.destroy();
        resolve3({ ok: false, error: "timeout" });
      });
      req.end();
    } catch (error) {
      resolve3({ ok: false, error: String(error) });
    }
  });
}
var formatKrw = (value) => `\u20A9${Math.round(value || 0).toLocaleString("ko-KR")}`;
var formatTokens = (value) => {
  const n = value || 0;
  return n >= 1e6 ? `${(n / 1e6).toFixed(2)}M` : n >= 1e3 ? `${(n / 1e3).toFixed(1)}k` : String(n);
};
var formatPercent = (ratio) => `${((ratio || 0) * 100).toFixed(1)}%`;
var modelName = (row) => String(row.model || row.name || "(unknown)");
function usageSummaryLine(report) {
  return `${formatKrw(report.cost_krw)} \xB7 ${formatTokens(report.tokens)} tok (${report.window === "monthly" ? "\uC6D4\uAC04" : "\uC8FC\uAC04"})`;
}
var cache;
var USAGE_TTL_MS = 5 * 60 * 1e3;
async function cachedWeeklyUsage(host, force = false) {
  if ((config().get("offlineMode") || "auto") === "offline") return null;
  const cp = await host.contextProxy.getInstance(host.context);
  const vals = cp.getValues();
  const expected = stringSetting("vibeCodersBaseUrl", "http://localhost:8080/v1").replace(/\/+$/g, "");
  if (vals.apiProvider !== "openai" || vals.openAiBaseUrl !== expected) return null;
  const apiKey = String(vals.openAiApiKey || "");
  if (!apiKey) return null;
  if (!force && cache && Date.now() - cache.fetchedAt < USAGE_TTL_MS) return cache.result;
  const result = await fetchUsageReport(apiKey, "weekly");
  cache = { fetchedAt: Date.now(), result };
  if (result.ok && result.report) await enforceBudget(host, result.report);
  return result;
}
async function enforceBudget(host, report) {
  const budget = Number(config().get("weeklyBudgetKrw") || 0);
  if (budget <= 0 || report.cost_krw <= budget) return false;
  const today = (/* @__PURE__ */ new Date()).toISOString().slice(0, 10);
  const state = host.context.globalState;
  if (state.get("vibeCode.budgetNotifiedDate") === today) return true;
  await state.update("vibeCode.budgetNotifiedDate", today);
  const applied = state.get("vibeCode.autonomyApplied") || "";
  const lowered = applied === "auto" || applied === "yolo";
  if (lowered) await writeAutonomyPreset(host, "assist", true);
  writeAudit(host, "proxy", "budgetExceeded", { budgetKrw: budget, costKrw: report.cost_krw, autonomyBefore: applied, lowered });
  log(host, `weekly budget exceeded: ${formatKrw(report.cost_krw)} > ${formatKrw(budget)}${lowered ? " (autonomy -> assist)" : ""}`);
  void vscode14.window.showWarningMessage(`vibe-coders \uC8FC\uAC04 \uBE44\uC6A9 ${formatKrw(report.cost_krw)}\uC774(\uAC00) \uC608\uC0B0 ${formatKrw(budget)}\uC744(\uB97C) \uB118\uC5C8\uC2B5\uB2C8\uB2E4.${lowered ? " \uC790\uC728\uC131\uC744 assist\uB85C \uB0AE\uCDC4\uC2B5\uB2C8\uB2E4." : ""}`, "\uC0AC\uC6A9\uB7C9 \uB300\uC2DC\uBCF4\uB4DC");
  return true;
}
function isOverBudget(report) {
  const budget = Number(config().get("weeklyBudgetKrw") || 0);
  return !!report && budget > 0 && report.cost_krw > budget;
}
function invalidateUsageCache() {
  cache = void 0;
}
function printReport(output, report) {
  const delta = report.cost_delta_ratio;
  output.appendLine(`--- ${report.window === "monthly" ? "\uC6D4\uAC04 (30\uC77C)" : "\uC8FC\uAC04 (7\uC77C)"} since ${report.since} ---`);
  output.appendLine(`  \uC694\uCCAD ${report.requests.toLocaleString("ko-KR")}\uD68C (\uC774\uC804 \uAD6C\uAC04 ${report.prior_requests.toLocaleString("ko-KR")}\uD68C) \xB7 \uC624\uB958 ${report.errors} \xB7 \uC131\uACF5\uB960 ${formatPercent(report.success_rate)}`);
  output.appendLine(`  \uD1A0\uD070 ${formatTokens(report.tokens)} \xB7 \uBE44\uC6A9 ${formatKrw(report.cost_krw)} (\uC774\uC804 ${formatKrw(report.prior_cost_krw)}, ${delta >= 0 ? "+" : ""}${(delta * 100).toFixed(1)}%)`);
  output.appendLine(`  \uD3C9\uADE0 \uC9C0\uC5F0 ${Math.round(report.avg_latency_ms)}ms \xB7 \uCE90\uC2DC \uC801\uC911 ${formatPercent(report.cache_rate)} \xB7 \uC704\uD5D8 \uC810\uC218 ${report.risk_score}`);
  if (report.potential_savings_krw > 0) output.appendLine(`  \uC808\uAC10 \uAC00\uB2A5 ${formatKrw(report.potential_savings_krw)} (${report.potential_savings_model}) \xB7 \uCD94\uCC9C ${report.recommendation_count}\uAC74`);
  if (Array.isArray(report.top_models) && report.top_models.length) {
    output.appendLine("  \uC0C1\uC704 \uBAA8\uB378:");
    for (const row of report.top_models) output.appendLine(`    - ${modelName(row)} :: ${formatKrw(Number(row.cost_krw))}${row.requests ? ` :: ${row.requests}\uD68C` : ""}${row.tokens ? ` :: ${formatTokens(Number(row.tokens))} tok` : ""}`);
  }
}
function registerUsageReport(host) {
  const { context, output } = host;
  context.subscriptions.push(
    vscode14.commands.registerCommand("vibe-code.showUsageReport", async () => {
      const cp = await host.contextProxy.getInstance(context);
      const apiKey = String(cp.getValues().openAiApiKey || "");
      if (!apiKey) {
        void vscode14.window.showWarningMessage("\uD604\uC7AC provider profile\uC5D0 API key\uAC00 \uC5C6\uC2B5\uB2C8\uB2E4. \uBA3C\uC800 vibe-coders \uD504\uB85D\uC2DC \uC801\uC6A9 \uBA85\uB839\uC744 \uC2E4\uD589\uD558\uC138\uC694.");
        return;
      }
      const [weekly, monthly] = await Promise.all([fetchUsageReport(apiKey, "weekly"), fetchUsageReport(apiKey, "monthly")]);
      output.show(true);
      output.appendLine(`
=== vibe-coders \uC0AC\uC6A9\uB7C9 \uB9AC\uD3EC\uD2B8 (${proxyOrigin()}) ===`);
      for (const result of [weekly, monthly]) {
        if (result.ok && result.report) printReport(output, result.report);
        else output.appendLine(`  \uC870\uD68C \uC2E4\uD328: ${result.status ? `HTTP ${result.status} ` : ""}${result.error || ""}`);
      }
      output.appendLine("================================\n");
      writeAudit(host, "proxy", "showUsageReport", { weekly: weekly.ok, monthly: monthly.ok, error: weekly.error || monthly.error || "" });
      if (!weekly.ok) void vscode14.window.showWarningMessage(`\uC0AC\uC6A9\uB7C9 \uB9AC\uD3EC\uD2B8 \uC870\uD68C \uC2E4\uD328: ${weekly.status ? `HTTP ${weekly.status}` : weekly.error}`);
    })
  );
  log(host, "usage report command registered");
}

// src/features/vibe-coders-proxy.ts
var DEFAULT_BASE_URL = "http://localhost:8080/v1";
var DEFAULT_PROFILE = "vibe-coders proxy";
var DEFAULT_MODEL = "gpt-4.1-mini";
var DEFAULT_KEY = "dev-proxy-key";
var HEADER = "X-Proxy-Provider";
var AUTO_ROUTING = "(model auto routing)";
var baseUrl = () => stringSetting("vibeCodersBaseUrl", DEFAULT_BASE_URL).replace(/\/+$/g, "") || DEFAULT_BASE_URL;
var profileName = () => stringSetting("vibeCodersProfileName", DEFAULT_PROFILE);
var defaultModel = () => stringSetting("vibeCodersDefaultModel", DEFAULT_MODEL);
var providerHeader = () => stringSetting("vibeCodersProviderHeader", "");
function proxyRootUrl(base) {
  try {
    const url = new URL(base);
    url.pathname = url.pathname.replace(/\/v1\/?$/, "/");
    url.search = "";
    url.hash = "";
    return url.origin;
  } catch {
    return "http://localhost:8080";
  }
}
function projectPath(host) {
  const raw = stringSetting("vibeCodersProjectPath", "../vibe-coders");
  const ws = workspaceRoot();
  const candidates = [];
  if (path12.isAbsolute(raw)) candidates.push(raw);
  if (ws) {
    candidates.push(path12.resolve(ws, raw));
    candidates.push(path12.resolve(ws, "..", "vibe-coders"));
    candidates.push(path12.resolve(ws, "..", "projects", "vibe-coders"));
  }
  candidates.push(path12.resolve(host.context.extensionPath, raw));
  try {
    candidates.push(path12.resolve(process.cwd(), raw));
  } catch {
  }
  const unique = Array.from(new Set(candidates.filter(Boolean)));
  return unique.find((p) => fs14.existsSync(path12.join(p, "go.mod")) || fs14.existsSync(path12.join(p, "cmd", "gateway", "main.go"))) || unique[0] || raw;
}
function probe(url) {
  return new Promise((resolve3) => {
    try {
      const target = new URL(url);
      const client = target.protocol === "https:" ? https4 : http2;
      const req = client.request({ method: "GET", hostname: target.hostname, port: target.port, path: `${target.pathname}${target.search}`, timeout: 2e3 }, (res) => {
        res.resume();
        const status = res.statusCode ?? 0;
        resolve3({ ok: status >= 200 && status < 500, status });
      });
      req.on("error", (error) => resolve3({ ok: false, error: error.code || error.message }));
      req.on("timeout", () => {
        req.destroy();
        resolve3({ ok: false, error: "timeout" });
      });
      req.end();
    } catch (error) {
      resolve3({ ok: false, error: String(error) });
    }
  });
}
function buildProfile(apiKey) {
  const header = providerHeader();
  return {
    apiProvider: "openai",
    openAiBaseUrl: baseUrl(),
    openAiApiKey: apiKey || DEFAULT_KEY,
    openAiModelId: defaultModel(),
    openAiStreamingEnabled: true,
    openAiHeaders: header ? { [HEADER]: header } : {}
  };
}
var errorText = (error) => error instanceof Error ? error.message : String(error);
function registerVibeCodersProxy(host) {
  const { context, output } = host;
  const state = context.globalState;
  const proxyStatus = async () => {
    const cp = await host.contextProxy.getInstance(context);
    const vals = cp.getValues();
    const expected = baseUrl();
    return { cp, vals, expected, routed: vals.apiProvider === "openai" && vals.openAiBaseUrl === expected };
  };
  const item = vscode15.window.createStatusBarItem(vscode15.StatusBarAlignment.Right, 98);
  item.command = "vibe-code.showVibeCodersProxyStatus";
  const refresh = async () => {
    try {
      const { vals, expected, routed } = await proxyStatus();
      const lastProfile = state.get("vibeCode.vibeCodersLastAppliedProfile") || "";
      const lastApplied = state.get("vibeCode.vibeCodersLastAppliedAt") || "";
      if (!routed && !lastProfile) {
        item.hide();
        return;
      }
      const model = vals.openAiModelId || state.get("vibeCode.vibeCodersLastAppliedModel") || "(unset)";
      const status = routed ? "ACTIVE" : "READY";
      const usage = routed ? await cachedWeeklyUsage(host) : null;
      const usageText = usage?.ok && usage.report ? ` \xB7 ${usageSummaryLine(usage.report)}` : "";
      const over = isOverBudget(usage?.report);
      item.text = `${over ? "$(warning)" : routed ? "$(plug)" : "$(debug-disconnect)"} VC: ${status}${usageText}${over ? " \xB7 \uC608\uC0B0 \uCD08\uACFC" : ""}`;
      item.backgroundColor = over ? new vscode15.ThemeColor("statusBarItem.warningBackground") : void 0;
      item.tooltip = `vibe-coders proxy
\uC0C1\uD0DC: ${status}
profile: ${vals.currentApiConfigName || "(unset)"}
baseUrl: ${vals.openAiBaseUrl || expected}
model: ${model}
lastApplied: ${lastApplied || "(never)"}` + (usage && !usage.ok ? `
\uC0AC\uC6A9\uB7C9: \uC870\uD68C \uC2E4\uD328 (${usage.status ? `HTTP ${usage.status}` : usage.error})` : "") + "\n\uD074\uB9AD\uD558\uC5EC \uC0C1\uC138 \uC0C1\uD0DC \uBCF4\uAE30";
      item.show();
    } catch {
      item.hide();
    }
  };
  const interval = setInterval(() => void refresh(), 6e4);
  context.subscriptions.push(
    item,
    { dispose: () => clearInterval(interval) },
    vscode15.workspace.onDidChangeConfiguration((event) => {
      if (event.affectsConfiguration("vibe-code")) void refresh();
    })
  );
  context.subscriptions.push(
    vscode15.commands.registerCommand("vibe-code.applyVibeCodersProxy", async () => {
      const key = await vscode15.window.showInputBox({
        prompt: "vibe-coders proxy API key\uB97C \uC785\uB825\uD558\uC138\uC694. \uBE44\uC6B0\uBA74 dev-proxy-key\uB97C \uC0AC\uC6A9\uD569\uB2C8\uB2E4.",
        placeHolder: "dev-proxy-key \uB610\uB294 vibe-coders\uC5D0\uC11C \uBC1C\uAE09\uD55C proxy key",
        password: true,
        ignoreFocusOut: true
      });
      if (key === void 0) return;
      const profile = buildProfile(key.trim() || DEFAULT_KEY);
      const name = profileName();
      try {
        const cp = await host.contextProxy.getInstance(context);
        const previous = String(cp.getValue("currentApiConfigName") || cp.getValues().currentApiConfigName || "");
        if (previous && previous !== name) {
          await state.update("vibeCode.previousProviderProfile", previous);
          log(host, `previous provider profile saved: ${previous}`);
        }
        await cp.setProviderSettings(profile);
        await cp.setValue("currentApiConfigName", name);
        const appliedAt = (/* @__PURE__ */ new Date()).toISOString();
        const header = profile.openAiHeaders[HEADER] || "";
        await state.update("vibeCode.vibeCodersLastAppliedAt", appliedAt);
        await state.update("vibeCode.vibeCodersLastAppliedProfile", name);
        await state.update("vibeCode.vibeCodersLastAppliedBaseUrl", profile.openAiBaseUrl);
        await state.update("vibeCode.vibeCodersLastAppliedModel", profile.openAiModelId);
        await state.update("vibeCode.vibeCodersLastAppliedProviderHeader", header);
        try {
          const provider = host.getProvider();
          if (provider?.providerSettingsManager) {
            await provider.providerSettingsManager.saveConfig(name, profile);
            await provider.activateProviderProfile?.({ name });
            await provider.postStateToWebview?.();
          }
        } catch (inner) {
          log(host, `provider profile UI sync skipped: ${inner}`);
        }
        output.show(true);
        output.appendLine("\n=== vibe-coders proxy \uC801\uC6A9 ===");
        output.appendLine(`profile: ${name}`);
        output.appendLine("apiProvider: openai");
        output.appendLine(`openAiBaseUrl: ${profile.openAiBaseUrl}`);
        output.appendLine(`openAiModelId: ${profile.openAiModelId}`);
        output.appendLine(`${HEADER}: ${header || AUTO_ROUTING}`);
        output.appendLine("\uC774\uD6C4 Vibe Code \uBAA8\uB378 \uD638\uCD9C\uC740 vibe-coders proxy\uB97C \uD1B5\uACFC\uD558\uBA70 \uC9D1\uACC4\uB429\uB2C8\uB2E4.");
        output.appendLine("================================\n");
        void vscode15.window.showInformationMessage(`vibe-coders proxy profile\uC744 \uC801\uC6A9\uD588\uC2B5\uB2C8\uB2E4: ${name}`);
        invalidateUsageCache();
        void refresh();
        writeAudit(host, "proxy", "applyVibeCodersProxy", { profile: name, baseUrl: profile.openAiBaseUrl, model: profile.openAiModelId, providerHeader: header, appliedAt });
      } catch (error) {
        log(host, `apply vibe-coders proxy failed: ${error}`);
        writeAudit(host, "proxy", "applyVibeCodersProxyFailed", { error: errorText(error) });
        void vscode15.window.showErrorMessage(`vibe-coders proxy \uC801\uC6A9 \uC2E4\uD328: ${errorText(error)}`);
      }
    }),
    vscode15.commands.registerCommand("vibe-code.restorePreviousProviderProfile", async () => {
      const previous = state.get("vibeCode.previousProviderProfile");
      if (!previous) {
        void vscode15.window.showWarningMessage("\uBCF5\uC6D0\uD560 \uC774\uC804 provider profile \uAE30\uB85D\uC774 \uC5C6\uC2B5\uB2C8\uB2E4. vibe-coders \uD504\uB85D\uC2DC \uC801\uC6A9 \uD6C4 \uC0AC\uC6A9\uD560 \uC218 \uC788\uC2B5\uB2C8\uB2E4.");
        return;
      }
      try {
        const provider = host.getProvider();
        let exists = true;
        try {
          if (provider?.providerSettingsManager) exists = await provider.providerSettingsManager.hasConfig(previous);
        } catch (inner) {
          log(host, `previous provider existence check skipped: ${inner}`);
        }
        if (!exists) {
          void vscode15.window.showWarningMessage(`\uC774\uC804 provider profile\uC744 \uCC3E\uC744 \uC218 \uC5C6\uC2B5\uB2C8\uB2E4: ${previous}`);
          return;
        }
        const cp = await host.contextProxy.getInstance(context);
        await cp.setValue("currentApiConfigName", previous);
        try {
          if (provider?.activateProviderProfile) {
            await provider.activateProviderProfile({ name: previous });
            await provider.postStateToWebview?.();
          }
        } catch (inner) {
          log(host, `previous provider UI sync skipped: ${inner}`);
        }
        output.show(true);
        output.appendLine("\n=== \uC774\uC804 provider profile \uBCF5\uC6D0 ===");
        output.appendLine(`profile: ${previous}`);
        output.appendLine("vibe-coders proxy \uC801\uC6A9 \uC804 profile\uB85C \uB418\uB3CC\uB838\uC2B5\uB2C8\uB2E4.");
        output.appendLine("================================\n");
        void vscode15.window.showInformationMessage(`\uC774\uC804 provider profile\uB85C \uBCF5\uC6D0\uD588\uC2B5\uB2C8\uB2E4: ${previous}`);
        void refresh();
        writeAudit(host, "proxy", "restorePreviousProviderProfile", { profile: previous });
      } catch (error) {
        log(host, `restore previous provider failed: ${error}`);
        writeAudit(host, "proxy", "restorePreviousProviderProfileFailed", { error: errorText(error) });
        void vscode15.window.showErrorMessage(`\uC774\uC804 provider \uBCF5\uC6D0 \uC2E4\uD328: ${errorText(error)}`);
      }
    }),
    vscode15.commands.registerCommand("vibe-code.showVibeCodersProxyStatus", async () => {
      const { vals, expected, routed } = await proxyStatus();
      const appliedAt = state.get("vibeCode.vibeCodersLastAppliedAt") || "";
      const appliedKst = appliedAt ? new Date(appliedAt).toLocaleString("ko-KR", { timeZone: "Asia/Seoul", hour12: false }) : "(never)";
      const header = vals.openAiHeaders?.[HEADER] || "";
      const lastHeader = state.get("vibeCode.vibeCodersLastAppliedProviderHeader") || "";
      output.show(true);
      output.appendLine("\n=== vibe-coders proxy \uC0C1\uD0DC ===");
      output.appendLine(`route: ${routed ? "ACTIVE" : "NOT ACTIVE"}`);
      output.appendLine(`expected baseUrl: ${expected}`);
      output.appendLine(`current profile: ${vals.currentApiConfigName || "(unset)"}`);
      output.appendLine(`apiProvider: ${vals.apiProvider || "(unset)"}`);
      output.appendLine(`openAiBaseUrl: ${vals.openAiBaseUrl || "(unset)"}`);
      output.appendLine(`openAiModelId: ${vals.openAiModelId || "(unset)"}`);
      output.appendLine(`${HEADER}: ${header || AUTO_ROUTING}`);
      output.appendLine(`configured ${HEADER}: ${providerHeader() || AUTO_ROUTING}`);
      output.appendLine(`previous profile: ${state.get("vibeCode.previousProviderProfile") || "(none)"}`);
      output.appendLine(`last applied: ${appliedKst}`);
      output.appendLine(`last applied profile: ${state.get("vibeCode.vibeCodersLastAppliedProfile") || "(none)"}`);
      output.appendLine(`last applied baseUrl: ${state.get("vibeCode.vibeCodersLastAppliedBaseUrl") || "(none)"}`);
      output.appendLine(`last applied model: ${state.get("vibeCode.vibeCodersLastAppliedModel") || "(none)"}`);
      output.appendLine(`last applied ${HEADER}: ${lastHeader || AUTO_ROUTING}`);
      output.appendLine("\uC774 \uBA85\uB839\uC740 \uB124\uD2B8\uC6CC\uD06C \uC694\uCCAD\uC774\uB098 \uBAA8\uB378 \uD638\uCD9C\uC744 \uB9CC\uB4E4\uC9C0 \uC54A\uC2B5\uB2C8\uB2E4.");
      output.appendLine("================================\n");
      writeAudit(host, "proxy", "showVibeCodersProxyStatus", {
        route: routed ? "ACTIVE" : "NOT ACTIVE",
        expectedBaseUrl: expected,
        currentProfile: vals.currentApiConfigName || "",
        baseUrl: vals.openAiBaseUrl || "",
        model: vals.openAiModelId || "",
        providerHeader: header
      });
      if (routed) void vscode15.window.showInformationMessage("Vibe Code \uBAA8\uB378 \uD638\uCD9C\uC774 vibe-coders proxy\uB85C \uB77C\uC6B0\uD305\uB418\uB3C4\uB85D \uC124\uC815\uB418\uC5B4 \uC788\uC2B5\uB2C8\uB2E4.");
      else void vscode15.window.showWarningMessage("\uD604\uC7AC provider profile\uC774 vibe-coders proxy\uB97C \uD5A5\uD558\uC9C0 \uC54A\uC2B5\uB2C8\uB2E4. vibe-coders \uD504\uB85D\uC2DC \uC801\uC6A9 \uBA85\uB839\uC744 \uC2E4\uD589\uD558\uC138\uC694.");
    }),
    vscode15.commands.registerCommand("vibe-code.checkVibeCodersProxy", async () => {
      const { vals, expected, routed } = await proxyStatus();
      const root = proxyRootUrl(expected);
      const health = await probe(`${root}/health`);
      const ready = await probe(`${root}/ready`);
      const project = projectPath(host);
      output.show(true);
      output.appendLine("\n=== vibe-coders proxy \uC810\uAC80 ===");
      output.appendLine(`projectPath: ${project}`);
      output.appendLine(`expected baseUrl: ${expected}`);
      output.appendLine(`current profile: ${vals.currentApiConfigName || "(unset)"}`);
      output.appendLine(`apiProvider: ${vals.apiProvider || "(unset)"}`);
      output.appendLine(`openAiBaseUrl: ${vals.openAiBaseUrl || "(unset)"}`);
      output.appendLine(`openAiModelId: ${vals.openAiModelId || "(unset)"}`);
      output.appendLine(`${HEADER}: ${vals.openAiHeaders?.[HEADER] || AUTO_ROUTING}`);
      output.appendLine(`health: ${health.ok ? "OK" : "--"} ${health.status || health.error || ""}`);
      output.appendLine(`ready: ${ready.ok ? "OK" : "--"} ${ready.status || ready.error || ""}`);
      output.appendLine("\uC8FC\uC758: /health, /ready \uC810\uAC80\uC740 \uBAA8\uB378 \uD638\uCD9C\uC774 \uC544\uB2C8\uBBC0\uB85C \uC0AC\uC6A9\uB7C9 \uC9D1\uACC4\uB97C \uC99D\uAC00\uC2DC\uD0A4\uC9C0 \uC54A\uC2B5\uB2C8\uB2E4.");
      output.appendLine("================================\n");
      writeAudit(host, "proxy", "checkVibeCodersProxy", {
        projectPath: project,
        expectedBaseUrl: expected,
        currentProfile: vals.currentApiConfigName || "",
        baseUrl: vals.openAiBaseUrl || "",
        model: vals.openAiModelId || "",
        health: health.ok ? String(health.status || "OK") : String(health.error || "error"),
        ready: ready.ok ? String(ready.status || "OK") : String(ready.error || "error")
      });
      if (routed) void vscode15.window.showInformationMessage("Vibe Code\uAC00 vibe-coders proxy\uB97C \uD5A5\uD558\uB3C4\uB85D \uC124\uC815\uB418\uC5B4 \uC788\uC2B5\uB2C8\uB2E4.");
      else void vscode15.window.showWarningMessage("\uD604\uC7AC provider profile\uC774 vibe-coders proxy\uB97C \uD5A5\uD558\uC9C0 \uC54A\uC2B5\uB2C8\uB2E4. 'vibe-coders \uD504\uB85D\uC2DC \uC801\uC6A9' \uBA85\uB839\uC744 \uC2E4\uD589\uD558\uC138\uC694.");
    }),
    vscode15.commands.registerCommand("vibe-code.writeVibeCodersProxyConfig", async () => {
      const ws = workspaceRoot();
      if (!ws) {
        void vscode15.window.showWarningMessage(NO_WORKSPACE_MESSAGE);
        return;
      }
      const dir = path12.join(ws, ".vibe-code");
      fs14.mkdirSync(dir, { recursive: true });
      const file = path12.join(dir, "vibe-coders-proxy.json");
      const body = {
        name: "vibe-coders",
        kind: "openai-compatible-proxy",
        projectPath: projectPath(host),
        baseUrl: baseUrl(),
        profileName: profileName(),
        defaultModel: defaultModel(),
        providerHeader: providerHeader() || null,
        applyCommand: "Vibe Code: vibe-coders \uD504\uB85D\uC2DC \uC801\uC6A9",
        checkCommand: "Vibe Code: vibe-coders \uD504\uB85D\uC2DC \uC810\uAC80",
        effect: "Vibe Code\uC758 OpenAI-compatible \uBAA8\uB378 \uD638\uCD9C\uC774 \uC774 base URL\uC744 \uC9C0\uB098\uAC00\uBA70 vibe-coders\uAC00 \uC790\uB3D9\uC73C\uB85C \uC0AC\uC6A9\uB7C9/\uD1A0\uD070/\uBE44\uC6A9\uC744 \uC9D1\uACC4\uD569\uB2C8\uB2E4.",
        secretHandling: "proxy API key\uB294 \uC124\uC815 \uD30C\uC77C\uC5D0 \uAE30\uB85D\uD558\uC9C0 \uC54A\uACE0 \uC801\uC6A9 \uBA85\uB839\uC5D0\uC11C Vibe Code \uB0B4\uBD80 provider \uC800\uC7A5 \uD750\uB984\uC5D0 \uB9E1\uAE41\uB2C8\uB2E4.",
        generatedAt: (/* @__PURE__ */ new Date()).toISOString()
      };
      fs14.writeFileSync(file, JSON.stringify(body, null, 2), "utf8");
      await vscode15.env.clipboard.writeText(baseUrl());
      await openFile(file);
      void vscode15.window.showInformationMessage("vibe-coders proxy \uC124\uC815 \uD30C\uC77C\uC744 \uC0DD\uC131\uD588\uACE0 base URL\uC744 \uD074\uB9BD\uBCF4\uB4DC\uC5D0 \uBCF5\uC0AC\uD588\uC2B5\uB2C8\uB2E4.");
      writeAudit(host, "proxy", "writeVibeCodersProxyConfig", { file: ".vibe-code/vibe-coders-proxy.json", baseUrl: baseUrl(), profileName: profileName(), defaultModel: defaultModel(), providerHeader: providerHeader() });
    })
  );
  void refresh();
  log(host, "proxy status bar item created");
  log(host, "vibe-coders proxy commands registered");
}

// src/features/onboarding.ts
var vscode16 = __toESM(require("vscode"));
init_log();
init_settings();
async function pick(title, step, items) {
  const picked = await vscode16.window.showQuickPick(items, { title: `Vibe Code \uC2DC\uC791 \uC124\uC815 (${step}/3) \u2014 ${title}`, ignoreFocusOut: true });
  return picked?.value;
}
function registerOnboarding(host) {
  const { context, output } = host;
  context.subscriptions.push(
    vscode16.commands.registerCommand("vibe-code.setupWizard", async () => {
      const cfg = config();
      const global = vscode16.ConfigurationTarget.Global;
      const done = [];
      const provider = await pick("\uBAA8\uB378 \uD504\uB85C\uBC14\uC774\uB354", 1, [
        { label: "$(plug) vibe-coders \uD504\uB85D\uC2DC", description: "\uAD8C\uC7A5", detail: `OpenAI \uD638\uD658 \uD504\uB85D\uC2DC ${cfg.get("vibeCodersBaseUrl")}\uB85C \uB77C\uC6B0\uD305\uD558\uACE0 \uC0AC\uC6A9\uB7C9\uC744 \uC9D1\uACC4\uD569\uB2C8\uB2E4.`, value: "vibe-coders" },
        { label: "$(cloud) \uD074\uB77C\uC6B0\uB4DC API \uC9C1\uC811 \uC5F0\uACB0", detail: "Anthropic, OpenAI, Google, Bedrock \uB4F1. \uC0AC\uC774\uB4DC\uBC14 \uC124\uC815\uC5D0\uC11C \uD0A4\uB97C \uC785\uB825\uD569\uB2C8\uB2E4.", value: "cloud" },
        { label: "$(vm) \uB85C\uCEEC \uBAA8\uB378", detail: "Ollama, LM Studio, vLLM \uB4F1 \uC0AC\uB0B4/\uB85C\uCEEC OpenAI \uD638\uD658 \uC5D4\uB4DC\uD3EC\uC778\uD2B8.", value: "local" },
        { label: "$(clock) \uB098\uC911\uC5D0", value: "later" }
      ]);
      if (provider === void 0) return;
      done.push(`\uD504\uB85C\uBC14\uC774\uB354: ${provider}`);
      const network = await pick("\uB124\uD2B8\uC6CC\uD06C \uD658\uACBD", 2, [
        { label: "$(cloud) \uC778\uD130\uB137 \uC0AC\uC6A9", description: "online", detail: "\uC678\uBD80 API\uC640 npx MCP \uC11C\uBC84\uB97C \uADF8\uB300\uB85C \uC0AC\uC6A9\uD569\uB2C8\uB2E4.", value: "online" },
        { label: "$(cloud-offline) \uD3D0\uC1C4\uB9DD", description: "offline", detail: "\uB124\uD2B8\uC6CC\uD06C \uC810\uAC80\uC744 \uD558\uC9C0 \uC54A\uACE0 \uB85C\uCEEC \uC18C\uC2A4\uB9CC \uC6B0\uC120\uD569\uB2C8\uB2E4.", value: "offline" },
        { label: "$(sync) \uC790\uB3D9 \uAC10\uC9C0", description: "auto", detail: "1\uBD84\uB9C8\uB2E4 \uC5F0\uACB0 \uC0C1\uD0DC\uB97C \uD655\uC778\uD574 \uC0C1\uD0DC\uBC14\uC5D0 \uD45C\uC2DC\uD569\uB2C8\uB2E4.", value: "auto" }
      ]);
      if (network === void 0) return finish();
      await cfg.update("offlineMode", network, global);
      done.push(`offlineMode: ${network}`);
      const autonomy = await pick("\uC790\uC728\uC131 \uB4F1\uAE09", 3, [
        { label: "$(shield) safe", detail: "\uBAA8\uB4E0 \uB3C4\uAD6C \uC2E4\uD589 \uC804\uC5D0 \uC2B9\uC778\uC744 \uC694\uCCAD\uD569\uB2C8\uB2E4.", value: "safe" },
        { label: "$(eye) assist", description: "\uAE30\uBCF8\uAC12", detail: "\uC77D\uAE30\uC640 \uBAA8\uB4DC \uC804\uD658\uB9CC \uC790\uB3D9 \uC2B9\uC778, \uC4F0\uAE30/\uC2E4\uD589\uC740 \uD655\uC778\uD569\uB2C8\uB2E4.", value: "assist" },
        { label: "$(rocket) auto", detail: "\uC77D\uAE30/\uC4F0\uAE30/\uBE0C\uB77C\uC6B0\uC800/MCP \uC790\uB3D9 \uC2B9\uC778, \uBA85\uB839 \uC2E4\uD589\uC740 \uD655\uC778\uD569\uB2C8\uB2E4.", value: "auto" },
        { label: "$(zap) yolo", detail: "\uBA85\uB839 \uC2E4\uD589\uAE4C\uC9C0 \uBAA8\uB450 \uC790\uB3D9 \uC2B9\uC778\uD569\uB2C8\uB2E4. \uACA9\uB9AC\uB41C \uD658\uACBD\uC5D0\uC11C\uB9CC \uAD8C\uC7A5.", value: "yolo" }
      ]);
      if (autonomy === void 0) return finish();
      await cfg.update("autonomy", autonomy, global);
      await writeAutonomyPreset(host, autonomy, true);
      done.push(`autonomy: ${autonomy}`);
      await context.globalState.update("vibeCode.setupCompleted", (/* @__PURE__ */ new Date()).toISOString());
      await finish();
      async function finish() {
        if (done.length === 0) return;
        output.appendLine("\n=== Vibe Code \uC2DC\uC791 \uC124\uC815 ===");
        for (const line of done) output.appendLine(`  ${line}`);
        output.appendLine("================================\n");
        log(host, `setup wizard: ${done.join(", ")}`);
        if (provider === "vibe-coders") {
          await vscode16.commands.executeCommand("vibe-code.applyVibeCodersProxy");
        } else if (provider === "cloud" || provider === "local") {
          const open = await vscode16.window.showInformationMessage(
            provider === "cloud" ? "\uC0AC\uC774\uB4DC\uBC14 \uC124\uC815\uC5D0\uC11C \uD504\uB85C\uBC14\uC774\uB354\uC640 API \uD0A4\uB97C \uC785\uB825\uD558\uC138\uC694." : "\uC0AC\uC774\uB4DC\uBC14 \uC124\uC815\uC5D0\uC11C \uB85C\uCEEC \uC5D4\uB4DC\uD3EC\uC778\uD2B8(Ollama, LM Studio, OpenAI \uD638\uD658)\uB97C \uC120\uD0DD\uD558\uC138\uC694.",
            "\uC124\uC815 \uC5F4\uAE30"
          );
          if (open === "\uC124\uC815 \uC5F4\uAE30") await vscode16.commands.executeCommand("vibe-code.settingsButtonClicked");
        }
        const next = await vscode16.window.showInformationMessage(`\uC2DC\uC791 \uC124\uC815\uC744 \uC800\uC7A5\uD588\uC2B5\uB2C8\uB2E4: ${done.join(" \xB7 ")}`, "\uC0AC\uC774\uB4DC\uBC14 \uC5F4\uAE30", "\uD658\uACBD \uC810\uAC80");
        if (next === "\uC0AC\uC774\uB4DC\uBC14 \uC5F4\uAE30") await vscode16.commands.executeCommand("vibe-code.SidebarProvider.focus");
        else if (next === "\uD658\uACBD \uC810\uAC80") await vscode16.commands.executeCommand("vibe-code.healthCheck");
      }
    })
  );
  log(host, "setup wizard command registered");
}

// src/features/plan-codelens.ts
var path13 = __toESM(require("node:path"));
var vscode17 = __toESM(require("vscode"));
init_log();
init_kst();
init_markdown();
init_plans();
init_workspace();
var PLAN_SELECTOR = { scheme: "file", pattern: "**/.vibe-code/plans/*.md" };
function lens(range, title, command, args) {
  return new vscode17.CodeLens(range, { title, command, arguments: args });
}
var PlanCodeLensProvider = class {
  provideCodeLenses(document) {
    const text = document.getText();
    const result = [];
    const args = (line) => [document.uri.toString(), line];
    for (let i = 0; i < document.lineCount; i++) {
      const line = document.lineAt(i).text;
      const range = new vscode17.Range(i, 0, i, 0);
      if (i === 0 && /^# 계획:/.test(line)) {
        result.push(lens(range, "$(target) \uD604\uC7AC \uACC4\uD68D\uC73C\uB85C \uC120\uD0DD", "vibe-code.selectPlanFile", args(i)));
        continue;
      }
      if (!isTaskLine(line)) continue;
      const sectionName = sectionOfLine(text, i);
      if (sectionName === "Now" && !isDoneTask(line)) {
        result.push(lens(range, "$(pass) \uC644\uB8CC\uB85C \uC774\uB3D9", "vibe-code.completePlanItem", args(i)));
        result.push(lens(range, "$(arrow-right) Next\uB85C \uB418\uB3CC\uB9AC\uAE30", "vibe-code.deferPlanItem", args(i)));
      } else if (sectionName === "Next" && !isDoneTask(line)) {
        result.push(lens(range, "$(arrow-up) Now\uB85C \uC2B9\uACA9", "vibe-code.promotePlanItem", args(i)));
      } else {
        result.push(lens(range, isDoneTask(line) ? "$(circle-outline) \uCCB4\uD06C \uD574\uC81C" : "$(check) \uCCB4\uD06C", "vibe-code.togglePlanCheckbox", args(i)));
      }
    }
    return result;
  }
};
async function editDocument(uriString, edit) {
  const document = await vscode17.workspace.openTextDocument(vscode17.Uri.parse(uriString));
  const before = document.getText();
  const after = edit(before);
  if (after === null || after === before) return null;
  const workspaceEdit = new vscode17.WorkspaceEdit();
  workspaceEdit.replace(document.uri, new vscode17.Range(0, 0, document.lineCount, 0), after);
  if (!await vscode17.workspace.applyEdit(workspaceEdit)) return null;
  await document.save();
  return { document, before };
}
function registerPlanCodeLens(host) {
  const { context } = host;
  const planFile = (uri) => ".vibe-code/plans/" + path13.basename(uri.fsPath);
  const lineText = (text, line) => (text.split(/\r?\n/)[line] || "").trim();
  context.subscriptions.push(
    vscode17.languages.registerCodeLensProvider(PLAN_SELECTOR, new PlanCodeLensProvider()),
    vscode17.commands.registerCommand("vibe-code.completePlanItem", async (uri, line) => {
      const result = await editDocument(uri, (text) => {
        const moved = moveTaskToSection(text, line, "Done", (l) => "- [x] " + stripTask(l));
        return moved === null ? null : touchPlan(moved, "active", kstStamp().human);
      });
      if (!result) return;
      writeAudit(host, "plan", "completePlanItem", { planFile: planFile(result.document.uri), completed: stripTask(lineText(result.before, line)) });
    }),
    vscode17.commands.registerCommand("vibe-code.deferPlanItem", async (uri, line) => {
      const result = await editDocument(uri, (text) => {
        const moved = moveTaskToSection(text, line, "Next");
        return moved === null ? null : touchPlan(moved, null, kstStamp().human);
      });
      if (!result) return;
      writeAudit(host, "plan", "deferPlanItem", { planFile: planFile(result.document.uri), item: stripTask(lineText(result.before, line)) });
    }),
    vscode17.commands.registerCommand("vibe-code.promotePlanItem", async (uri, line) => {
      const result = await editDocument(uri, (text) => {
        const moved = moveTaskToSection(text, line, "Now");
        return moved === null ? null : touchPlan(moved, "active", kstStamp().human);
      });
      if (!result) return;
      writeAudit(host, "plan", "promotePlanItem", { planFile: planFile(result.document.uri), promoted: stripTask(lineText(result.before, line)) });
    }),
    vscode17.commands.registerCommand("vibe-code.togglePlanCheckbox", async (uri, line) => {
      const result = await editDocument(uri, (text) => touchPlan(toggleCheckbox(text, line), null, kstStamp().human));
      if (!result) return;
      writeAudit(host, "plan", "togglePlanCheckbox", { planFile: planFile(result.document.uri), item: stripTask(lineText(result.before, line)), done: !isDoneTask(lineText(result.before, line)) });
    }),
    vscode17.commands.registerCommand("vibe-code.selectPlanFile", async (uri) => {
      const paths = ensureWorkspacePaths();
      if (!paths) return;
      const name = path13.basename(vscode17.Uri.parse(uri).fsPath);
      if (selectPlanName(paths) === name) {
        void vscode17.window.showInformationMessage("\uC774\uBBF8 \uD604\uC7AC \uACC4\uD68D\uC785\uB2C8\uB2E4: " + name);
        return;
      }
      setActivePlanName(paths, name);
      void vscode17.window.showInformationMessage("\uD604\uC7AC \uACC4\uD68D\uC744 \uC120\uD0DD\uD588\uC2B5\uB2C8\uB2E4: " + name);
      writeAudit(host, "plan", "setActivePlan", { planFile: ".vibe-code/plans/" + name, via: "codelens" });
    })
  );
  log(host, "plan CodeLens registered");
}

// src/features/command-audit.ts
var path15 = __toESM(require("node:path"));
var vscode19 = __toESM(require("vscode"));
init_log();
init_workspace();
var fs16 = __toESM(require("node:fs"));

// src/features/checkpoints.ts
var import_node_child_process2 = require("node:child_process");
var fs15 = __toESM(require("node:fs"));
var os2 = __toESM(require("node:os"));
var path14 = __toESM(require("node:path"));
var vscode18 = __toESM(require("vscode"));
init_log();
init_settings();
init_kst();
init_workspace();
var DESTRUCTIVE = [
  /\brm\s+(-[a-z]*r[a-z]*f?|-[a-z]*f[a-z]*r)\b/i,
  /\brm\s+-rf?\b/i,
  /\b(rmdir|rd)\s+\/s\b/i,
  /\bdel\s+\/[sq]\b/i,
  /Remove-Item\b[^\n]*-Recurse/i,
  /\bgit\s+(reset\s+--hard|clean\s+-[a-z]*f[a-z]*|checkout\s+--\s+\.|restore\s+\.|push\s+[^\n]*--force|branch\s+-D)\b/i,
  /\b(drop|truncate)\s+(table|database|schema)\b/i,
  /\bDROP\s+(TABLE|DATABASE)\b/,
  /\bmkfs\b|\bdd\s+if=/i,
  /\bnpm\s+(publish|unpublish)\b|\bpnpm\s+publish\b/i,
  /\bterraform\s+(destroy|apply)\b|\bkubectl\s+delete\b|\bdocker\s+(system\s+prune|volume\s+rm)\b/i,
  /\bchmod\s+-R\b|\bchown\s+-R\b/i
];
function isDestructiveCommand(command) {
  const text = command.trim();
  return DESTRUCTIVE.some((re) => re.test(text));
}
function hasHead(cwd) {
  try {
    git(cwd, ["rev-parse", "--verify", "--quiet", "HEAD"]);
    return true;
  } catch {
    return false;
  }
}
function git(cwd, args, indexFile) {
  const env2 = indexFile ? { ...process.env, GIT_INDEX_FILE: indexFile } : process.env;
  return (0, import_node_child_process2.execFileSync)("git", args, { cwd, env: env2, encoding: "utf8", windowsHide: true, stdio: ["ignore", "pipe", "ignore"] }).trim();
}
function snapshotTree(ws, parent) {
  const indexDir = fs15.mkdtempSync(path14.join(os2.tmpdir(), "vibe-checkpoint-"));
  const indexFile = path14.join(indexDir, "index");
  try {
    if (parent) git(ws, ["read-tree", parent], indexFile);
    git(ws, ["add", "-A"], indexFile);
    return git(ws, ["write-tree"], indexFile);
  } finally {
    fs15.rmSync(indexDir, { recursive: true, force: true });
  }
}
function createCheckpoint(ws, checkpointsDir, command, reason) {
  const stamp = kstStamp();
  const slug = command.replace(/[^\p{L}\p{N}]+/gu, "-").replace(/^-+|-+$/g, "").slice(0, 30) || "command";
  const file = path14.join(checkpointsDir, `${stamp.file}-${slug}.md`);
  let ref;
  let commit;
  let note;
  try {
    git(ws, ["rev-parse", "--is-inside-work-tree"]);
    const parent = hasHead(ws) ? git(ws, ["rev-parse", "HEAD"]) : null;
    const head = parent ? git(ws, ["rev-parse", "--short", "HEAD"]) : "\uCEE4\uBC0B \uC5C6\uC74C";
    const tree = snapshotTree(ws, parent);
    if (parent && tree === git(ws, ["rev-parse", "HEAD^{tree}"])) {
      note = `\uC791\uC5C5 \uD2B8\uB9AC\uAC00 \uAE68\uB057\uD574\uC11C \uC2A4\uB0C5\uC0F7 \uC5C6\uC74C (HEAD ${head}); \`git reflog\`\uB85C \uBCF5\uAD6C \uAC00\uB2A5`;
    } else {
      commit = git(ws, ["commit-tree", tree, ...parent ? ["-p", parent] : [], "-m", "vibe checkpoint"]);
      ref = `refs/vibe-checkpoints/${stamp.file}`;
      git(ws, ["update-ref", ref, commit]);
      note = `git \uC2A4\uB0C5\uC0F7 \uC0DD\uC131 (HEAD ${head}, \uCD94\uC801/\uBBF8\uCD94\uC801 \uD30C\uC77C \uCEE4\uBC0B ${commit.slice(0, 12)})`;
    }
  } catch {
    note = "git \uC800\uC7A5\uC18C\uAC00 \uC544\uB2C8\uC5B4\uC11C \uC2A4\uB0C5\uC0F7\uC744 \uB9CC\uB4E4\uC9C0 \uBABB\uD568 \u2014 \uD30C\uC77C \uBC31\uC5C5\uC744 \uC9C1\uC811 \uD655\uC778\uD558\uC138\uC694";
  }
  fs15.mkdirSync(checkpointsDir, { recursive: true });
  const body = [
    `# \uCCB4\uD06C\uD3EC\uC778\uD2B8 \u2014 ${stamp.human}`,
    "",
    `\uC774\uC720: ${reason}`,
    `\uBA85\uB839: \`${command}\``,
    `\uACB0\uACFC: ${note}`,
    ...commit && ref ? [
      "",
      "## \uBCF5\uC6D0",
      "",
      "```powershell",
      `git diff ${commit}        # \uC2A4\uB0C5\uC0F7\uACFC \uC9C0\uAE08 \uC791\uC5C5 \uD2B8\uB9AC\uC758 \uCC28\uC774 \uD655\uC778`,
      `git checkout ${commit} -- .        # \uC2A4\uB0C5\uC0F7 \uC2DC\uC810 \uD30C\uC77C\uB85C \uB36E\uC5B4\uC4F0\uAE30 (\uCD94\uC801/\uBBF8\uCD94\uC801 \uD30C\uC77C \uBAA8\uB450)`,
      `git update-ref -d ${ref}        # \uC815\uB9AC`,
      "```",
      "",
      "\uC2A4\uB0C5\uC0F7\uC5D0 \uC5C6\uB358 \uD30C\uC77C(\uCCB4\uD06C\uD3EC\uC778\uD2B8 \uC774\uD6C4\uC5D0 \uC0DD\uAE34 \uD30C\uC77C)\uC740 \uC774 \uBCF5\uC6D0\uC73C\uB85C \uC9C0\uC6CC\uC9C0\uC9C0 \uC54A\uC2B5\uB2C8\uB2E4."
    ] : [],
    ""
  ].join("\n");
  fs15.writeFileSync(file, body, "utf8");
  return { file, ref, commit, note };
}
function checkpointBeforeCommand(host, command) {
  if (!(config().get("checkpointBeforeDestructive") ?? true)) return;
  if (!isDestructiveCommand(command)) return;
  const ws = workspaceRoot();
  const paths = ensureWorkspacePaths(true);
  if (!ws || !paths) return;
  try {
    const cp = createCheckpoint(ws, paths.checkpoints, command, "\uD30C\uAD34\uC801 \uBA85\uB839 \uC2B9\uC778 \uC9C1\uC804 \uC790\uB3D9 \uC2A4\uB0C5\uC0F7");
    writeAudit(host, "command", "checkpoint", { command, file: `.vibe-code/checkpoints/${path14.basename(cp.file)}`, ref: cp.ref, commit: cp.commit, note: cp.note });
    log(host, `checkpoint before destructive command: ${cp.note}`);
    void vscode18.window.showInformationMessage(`\uD30C\uAD34\uC801 \uBA85\uB839 \uC804 \uCCB4\uD06C\uD3EC\uC778\uD2B8: ${cp.note}`, "\uCCB4\uD06C\uD3EC\uC778\uD2B8 \uC5F4\uAE30").then(async (choice) => {
      if (choice === "\uCCB4\uD06C\uD3EC\uC778\uD2B8 \uC5F4\uAE30") await vscode18.window.showTextDocument(await vscode18.workspace.openTextDocument(cp.file));
    });
  } catch (error) {
    log(host, `checkpoint skipped: ${error}`);
  }
}
function registerCheckpoints(host) {
  const { context, output } = host;
  context.subscriptions.push(
    vscode18.commands.registerCommand("vibe-code.createCheckpoint", async (reasonArg) => {
      const ws = workspaceRoot();
      const paths = ensureWorkspacePaths();
      if (!ws || !paths) return;
      const reason = typeof reasonArg === "string" && reasonArg ? reasonArg : await vscode18.window.showInputBox({ prompt: "\uCCB4\uD06C\uD3EC\uC778\uD2B8 \uC774\uC720 (\uC120\uD0DD)", placeHolder: "\uC608: \uB300\uADDC\uBAA8 \uB9AC\uD329\uD130\uB9C1 \uC804" }) || "\uC218\uB3D9 \uCCB4\uD06C\uD3EC\uC778\uD2B8";
      const cp = createCheckpoint(ws, paths.checkpoints, "(manual)", reason);
      writeAudit(host, "command", "checkpoint", { command: "(manual)", file: `.vibe-code/checkpoints/${path14.basename(cp.file)}`, ref: cp.ref, commit: cp.commit, note: cp.note });
      void vscode18.window.showInformationMessage(`\uCCB4\uD06C\uD3EC\uC778\uD2B8 \uC0DD\uC131: ${cp.note}`);
    }),
    vscode18.commands.registerCommand("vibe-code.listCheckpoints", async () => {
      const paths = ensureWorkspacePaths();
      if (!paths) return;
      const files = fs15.existsSync(paths.checkpoints) ? fs15.readdirSync(paths.checkpoints).filter((n) => n.endsWith(".md")).sort().slice(-20) : [];
      output.show(true);
      output.appendLine("\n=== \uCCB4\uD06C\uD3EC\uC778\uD2B8 (\uCD5C\uADFC 20\uAC1C) ===");
      if (files.length === 0) output.appendLine("  (\uC5C6\uC74C)");
      for (const name of files) {
        const text = readUtf8(path14.join(paths.checkpoints, name));
        const cmd = text.match(/^명령: `(.*)`$/m)?.[1] || "";
        const result = text.match(/^결과: (.*)$/m)?.[1] || "";
        output.appendLine(`  ${name} :: ${cmd} :: ${result}`);
      }
      output.appendLine("================================\n");
    })
  );
  log(host, "checkpoint commands registered");
}

// src/features/command-audit.ts
var activeHost;
function handleCommandEvent(event) {
  if (!activeHost) return;
  const details = { phase: event.phase, command: event.command };
  if (event.cwd) details.cwd = event.cwd;
  if (event.exitCode !== void 0) details.exitCode = event.exitCode;
  if (event.executionId) details.executionId = event.executionId;
  if (event.taskId) details.taskId = event.taskId;
  if (event.phase === "approved") checkpointBeforeCommand(activeHost, event.command);
  writeAudit(activeHost, "command", event.phase, details);
  if (event.phase === "denied") log(activeHost, `command denied: ${event.command}`);
  else if (event.phase === "exited" && event.exitCode !== 0 && event.exitCode !== void 0) log(activeHost, `command exited ${event.exitCode}: ${event.command}`);
}
function readCommandHistory(auditDir, limit = 30) {
  if (!fs16.existsSync(auditDir)) return [];
  const rows = [];
  const files = fs16.readdirSync(auditDir).filter((name) => name.endsWith(".jsonl")).sort();
  for (const name of files.slice(-7)) {
    for (const line of readUtf8(path15.join(auditDir, name)).split(/\r?\n/).filter(Boolean)) {
      try {
        const row = JSON.parse(line);
        if (row.kind !== "command") continue;
        const d = row.details || {};
        rows.push({ ts: row.ts, phase: String(d.phase || row.action), command: String(d.command || ""), cwd: d.cwd, exitCode: d.exitCode });
      } catch {
      }
    }
  }
  return rows.slice(-limit);
}
function registerCommandAudit(host) {
  activeHost = host;
  const { context, output } = host;
  context.subscriptions.push(
    { dispose: () => activeHost = void 0 },
    vscode19.commands.registerCommand("vibe-code.showCommandHistory", async () => {
      const paths = ensureWorkspacePaths(true);
      if (!paths) return;
      const rows = readCommandHistory(paths.audit);
      output.show(true);
      output.appendLine("\n=== \uBA85\uB839 \uC2E4\uD589 \uC774\uB825 (\uCD5C\uADFC 7\uC77C, \uCD5C\uB300 30\uAC74) ===");
      if (rows.length === 0) output.appendLine("  (\uAE30\uB85D\uB41C \uBA85\uB839 \uC2E4\uD589 \uC5C6\uC74C)");
      for (const row of rows) {
        const mark = row.phase === "denied" ? "\u2717 \uAC70\uBD80" : row.phase === "approved" ? "\u25B6 \uC2B9\uC778" : row.exitCode === 0 ? "\u2713 \uC885\uB8CC 0" : `! \uC885\uB8CC ${row.exitCode}`;
        output.appendLine(`  ${row.ts} :: ${mark} :: ${row.command}${row.cwd ? ` (cwd: ${row.cwd})` : ""}`);
      }
      output.appendLine("=========================================\n");
    })
  );
  log(host, "command audit hook registered");
}

// src/features/usage-dashboard.ts
var crypto = __toESM(require("node:crypto"));
var vscode20 = __toESM(require("vscode"));
init_log();
init_workspace();
var escapeHtml = (value) => String(value ?? "").replace(/[&<>"']/g, (ch) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[ch]);
function statTile(label, value, note = "") {
  return `<div class="tile"><div class="label">${escapeHtml(label)}</div><div class="value">${escapeHtml(value)}</div>${note ? `<div class="note">${escapeHtml(note)}</div>` : ""}</div>`;
}
function renderWindow(result) {
  if (!result.ok || !result.report) {
    return `<p class="error">\uC870\uD68C \uC2E4\uD328: ${escapeHtml(result.status ? `HTTP ${result.status} ` : "")}${escapeHtml(result.error || "")}</p>`;
  }
  const r = result.report;
  const delta = r.cost_delta_ratio;
  const deltaText = `${delta >= 0 ? "+" : ""}${(delta * 100).toFixed(1)}% vs \uC774\uC804 \uAD6C\uAC04 (${formatKrw(r.prior_cost_krw)})`;
  const models = Array.isArray(r.top_models) ? r.top_models : [];
  const maxCost = Math.max(1, ...models.map((m) => Number(m.cost_krw) || 0));
  const bars = models.map((m) => {
    const cost = Number(m.cost_krw) || 0;
    const pct = Math.max(2, Math.round(cost / maxCost * 100));
    return `<li title="${escapeHtml(modelName(m))}: ${escapeHtml(formatKrw(cost))}${m.requests ? `, ${escapeHtml(m.requests)}\uD68C` : ""}">
				<span class="name">${escapeHtml(modelName(m))}</span>
				<span class="track"><span class="bar" style="width:${pct}%"></span></span>
				<span class="num">${escapeHtml(formatKrw(cost))}</span>
			</li>`;
  }).join("");
  const rows = models.map((m) => `<tr><td>${escapeHtml(modelName(m))}</td><td class="num">${escapeHtml(formatKrw(Number(m.cost_krw)))}</td><td class="num">${escapeHtml(m.requests ?? "")}</td><td class="num">${escapeHtml(m.tokens ? formatTokens(Number(m.tokens)) : "")}</td></tr>`).join("");
  return `
		<div class="tiles">
			${statTile("\uBE44\uC6A9", formatKrw(r.cost_krw), deltaText)}
			${statTile("\uC694\uCCAD", r.requests.toLocaleString("ko-KR") + "\uD68C", `\uC774\uC804 \uAD6C\uAC04 ${r.prior_requests.toLocaleString("ko-KR")}\uD68C`)}
			${statTile("\uD1A0\uD070", formatTokens(r.tokens))}
			${statTile("\uC131\uACF5\uB960", formatPercent(r.success_rate), `\uC624\uB958 ${r.errors}\uAC74`)}
			${statTile("\uD3C9\uADE0 \uC9C0\uC5F0", `${Math.round(r.avg_latency_ms)} ms`, `\uCE90\uC2DC \uC801\uC911 ${formatPercent(r.cache_rate)}`)}
			${statTile("\uC808\uAC10 \uAC00\uB2A5", formatKrw(r.potential_savings_krw), r.potential_savings_model ? `${r.potential_savings_model} \xB7 \uCD94\uCC9C ${r.recommendation_count}\uAC74` : "")}
		</div>
		<h3>\uC0C1\uC704 \uBAA8\uB378 (\uBE44\uC6A9)</h3>
		${models.length ? `<ul class="bars">${bars}</ul><details><summary>\uD45C\uB85C \uBCF4\uAE30</summary><table><thead><tr><th>\uBAA8\uB378</th><th class="num">\uBE44\uC6A9</th><th class="num">\uC694\uCCAD</th><th class="num">\uD1A0\uD070</th></tr></thead><tbody>${rows}</tbody></table></details>` : `<p class="muted">\uC9D1\uACC4\uB41C \uBAA8\uB378 \uD638\uCD9C\uC774 \uC5C6\uC2B5\uB2C8\uB2E4.</p>`}
		<p class="muted">since ${escapeHtml(r.since)} \xB7 user ${escapeHtml(r.user_id)}</p>`;
}
function renderPage(nonce, origin, weekly, monthly, fetchedAt) {
  return `<!DOCTYPE html>
<html lang="ko">
<head>
<meta charset="UTF-8">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'nonce-${nonce}'; script-src 'nonce-${nonce}';">
<title>vibe-coders \uC0AC\uC6A9\uB7C9</title>
<style nonce="${nonce}">
	body { font-family: var(--vscode-font-family); color: var(--vscode-foreground); background: var(--vscode-editor-background); padding: 16px 20px; line-height: 1.4; }
	h1 { font-size: 1.3em; margin: 0 0 4px; } h2 { font-size: 1.05em; margin: 20px 0 8px; border-bottom: 1px solid var(--vscode-panel-border); padding-bottom: 4px; } h3 { font-size: 0.95em; margin: 16px 0 6px; }
	.toolbar { display: flex; gap: 8px; align-items: center; margin: 6px 0 12px; } .muted { color: var(--vscode-descriptionForeground); font-size: 0.85em; } .error { color: var(--vscode-errorForeground); }
	button { background: var(--vscode-button-background); color: var(--vscode-button-foreground); border: none; padding: 4px 12px; border-radius: 2px; cursor: pointer; } button:hover { background: var(--vscode-button-hoverBackground); }
	.tiles { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 8px; }
	.tile { background: var(--vscode-editorWidget-background); border: 1px solid var(--vscode-panel-border); border-radius: 4px; padding: 10px 12px; }
	.tile .label { color: var(--vscode-descriptionForeground); font-size: 0.8em; } .tile .value { font-size: 1.4em; font-weight: 600; margin: 2px 0; } .tile .note { color: var(--vscode-descriptionForeground); font-size: 0.78em; }
	ul.bars { list-style: none; padding: 0; margin: 0; } ul.bars li { display: grid; grid-template-columns: minmax(120px, 30%) 1fr 90px; align-items: center; gap: 8px; padding: 3px 0; }
	ul.bars li:hover { background: var(--vscode-list-hoverBackground); } .name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.track { height: 8px; background: var(--vscode-editorWidget-background); border-radius: 0 4px 4px 0; overflow: hidden; } .bar { display: block; height: 100%; background: var(--vscode-charts-blue); border-radius: 0 4px 4px 0; }
	.num { text-align: right; font-variant-numeric: tabular-nums; }
	table { border-collapse: collapse; margin-top: 6px; } th, td { padding: 3px 10px 3px 0; text-align: left; border-bottom: 1px solid var(--vscode-panel-border); } th.num, td.num { text-align: right; }
	details summary { cursor: pointer; color: var(--vscode-textLink-foreground); font-size: 0.85em; margin-top: 6px; }
</style>
</head>
<body>
	<h1>vibe-coders \uC0AC\uC6A9\uB7C9</h1>
	<div class="toolbar"><span class="muted">${escapeHtml(origin)} \xB7 \uC870\uD68C ${escapeHtml(fetchedAt)}</span><button id="refresh">\uC0C8\uB85C\uACE0\uCE68</button><button id="report">Output\uC5D0 \uB9AC\uD3EC\uD2B8</button></div>
	<h2>\uC8FC\uAC04 (7\uC77C)</h2>${renderWindow(weekly)}
	<h2>\uC6D4\uAC04 (30\uC77C)</h2>${renderWindow(monthly)}
	<script nonce="${nonce}">
		const vscode = acquireVsCodeApi();
		document.getElementById("refresh").addEventListener("click", () => vscode.postMessage({ type: "refresh" }));
		document.getElementById("report").addEventListener("click", () => vscode.postMessage({ type: "report" }));
	</script>
</body>
</html>`;
}
function registerUsageDashboard(host) {
  const { context } = host;
  let panel;
  const load = async () => {
    if (!panel) return;
    const cp = await host.contextProxy.getInstance(context);
    const vals = cp.getValues();
    const apiKey = String(vals.openAiApiKey || "");
    const origin = (() => {
      try {
        return new URL(String(vals.openAiBaseUrl || "http://localhost:8080/v1")).origin;
      } catch {
        return "http://localhost:8080";
      }
    })();
    const nonce = crypto.randomBytes(16).toString("base64");
    if (!apiKey) {
      const missing = { ok: false, error: "\uD604\uC7AC provider profile\uC5D0 API key\uAC00 \uC5C6\uC2B5\uB2C8\uB2E4. \uBA3C\uC800 vibe-coders \uD504\uB85D\uC2DC \uC801\uC6A9 \uBA85\uB839\uC744 \uC2E4\uD589\uD558\uC138\uC694." };
      panel.webview.html = renderPage(nonce, origin, missing, missing, (/* @__PURE__ */ new Date()).toLocaleTimeString("ko-KR"));
      return;
    }
    const [weekly, monthly] = await Promise.all([fetchUsageReport(apiKey, "weekly"), fetchUsageReport(apiKey, "monthly")]);
    panel.webview.html = renderPage(nonce, origin, weekly, monthly, (/* @__PURE__ */ new Date()).toLocaleTimeString("ko-KR", { hour12: false }));
    writeAudit(host, "proxy", "openUsageDashboard", { weekly: weekly.ok, monthly: monthly.ok, error: weekly.error || monthly.error || "" });
  };
  context.subscriptions.push(
    vscode20.commands.registerCommand("vibe-code.openUsageDashboard", async () => {
      if (panel) {
        panel.reveal();
        await load();
        return;
      }
      panel = vscode20.window.createWebviewPanel("vibe-code.usageDashboard", "vibe-coders \uC0AC\uC6A9\uB7C9", vscode20.ViewColumn.Beside, { enableScripts: true, localResourceRoots: [] });
      panel.iconPath = new vscode20.ThemeIcon("graph");
      panel.onDidDispose(() => panel = void 0);
      panel.webview.onDidReceiveMessage(async (message) => {
        if (message?.type === "refresh") await load();
        else if (message?.type === "report") await vscode20.commands.executeCommand("vibe-code.showUsageReport");
      });
      await load();
    })
  );
  log(host, "usage dashboard command registered");
}

// src/features/goal-loop.ts
var fs17 = __toESM(require("node:fs"));
var vscode21 = __toESM(require("vscode"));
init_log();
init_settings();
init_markdown();
init_goals();
init_workspace();
var RESUME_PROMPT = "/goal \uC774\uC5B4\uC11C";
function buildPromptContext(goalText) {
  if (!goalText.trim()) return "";
  const goal = parseGoal(goalText);
  if (!["active", "draft"].includes(goal.status.toLowerCase())) return "";
  const now = goal.now.filter((l) => !/^- \[[xX]\]/.test(l)).map(stripTask).slice(0, 4);
  const next = goal.nextItems.filter((l) => !/^- \[[xX]\]/.test(l)).map(stripTask).slice(0, 2);
  const lines2 = [
    "====",
    "",
    "VIBE CODE GOAL STATE",
    "",
    `A persistent goal is active in ${GOAL_FILE}. Work on it unless the user asks for something else.`,
    "",
    `- \uBAA9\uD45C: ${goal.title}`,
    `- \uC0C1\uD0DC: ${goal.status} / \uC9C0\uAE08 \uB2E8\uACC4: ${goal.phase}`,
    `- \uB2E4\uC74C \uD589\uB3D9: ${goal.next}`,
    `- \uC644\uB8CC \uAE30\uC900 \uC9C4\uD589: ${progressLabel(goal, "\uCCB4\uD06C\uB9AC\uC2A4\uD2B8 \uC5C6\uC74C")}`,
    now.length ? `- Now: ${now.join(" | ")}` : "- Now: (\uBE44\uC5B4 \uC788\uC74C \u2014 Next\uC5D0\uC11C \uD558\uB098\uB97C \uC2B9\uACA9\uD558\uAC70\uB098 \uC0C8 \uC791\uC5C5\uC744 \uC815\uC758\uD558\uC138\uC694)"
  ];
  if (next.length) lines2.push(`- Next: ${next.join(" | ")}`);
  lines2.push(
    "",
    "Rules: pick the one Now item that moves the goal forward; after every meaningful change update the \uC791\uC5C5 \uD050, \uBCC0\uACBD \uB85C\uADF8 and \uAC80\uC99D \uB85C\uADF8 in the goal file (and the active plan under .vibe-code/plans/ if one exists); when all \uC644\uB8CC \uAE30\uC900 are checked set `\uC0C1\uD0DC: done`; do not wait for further instructions while Now items remain."
  );
  return "\n\n" + lines2.join("\n") + "\n";
}
var promptCache;
function promptContext() {
  try {
    const ws = vscode21.workspace.workspaceFolders?.[0]?.uri.fsPath;
    if (!ws) return "";
    const file = `${ws}/.vibe-code/goals/current.md`;
    if (!fs17.existsSync(file)) return "";
    const mtimeMs = fs17.statSync(file).mtimeMs;
    if (promptCache && promptCache.file === file && promptCache.mtimeMs === mtimeMs) return promptCache.text;
    const text = buildPromptContext(readUtf8(file));
    promptCache = { file, mtimeMs, text };
    return text;
  } catch {
    return "";
  }
}
var activeHost2;
var taskStarts = /* @__PURE__ */ new Map();
function goalMtime() {
  const paths = ensureWorkspacePaths(true);
  if (!paths || !fs17.existsSync(paths.current)) return 0;
  return fs17.statSync(paths.current).mtimeMs;
}
function activeGoal() {
  const paths = ensureWorkspacePaths(true);
  if (!paths || !fs17.existsSync(paths.current)) return null;
  const text = readUtf8(paths.current);
  const summary = parseGoal(text);
  return summary.status.toLowerCase() === "active" ? { text, summary } : null;
}
async function startResumeTask(host, reason) {
  const provider = host.getProvider();
  if (!provider?.initClineWithTask) {
    void vscode21.window.showWarningMessage("\uC0AC\uC774\uB4DC\uBC14\uAC00 \uC544\uC9C1 \uC900\uBE44\uB418\uC9C0 \uC54A\uC558\uC2B5\uB2C8\uB2E4. \uC7A0\uC2DC \uD6C4 \uB2E4\uC2DC \uC2DC\uB3C4\uD558\uC138\uC694.");
    return false;
  }
  await vscode21.commands.executeCommand("vibe-code.SidebarProvider.focus");
  await provider.initClineWithTask(RESUME_PROMPT);
  writeAudit(host, "goal", "resumeGoal", { reason });
  log(host, `goal resumed (${reason})`);
  return true;
}
function handleTaskEvent(event) {
  const host = activeHost2;
  if (!host) return;
  try {
    if (event.phase === "started") {
      taskStarts.set(event.taskId, { startedAt: Date.now(), goalMtime: goalMtime() });
      return;
    }
    if (event.phase === "condensed") {
      const goal2 = activeGoal();
      if (!goal2) return;
      const paths = ensureWorkspacePaths(true);
      if (!paths) return;
      const handoff = createHandoffFile(host, paths, "context condensed");
      writeAudit(host, "goal", "autoHandoff", { trigger: "condensed", taskId: event.taskId, handoffFile: handoff });
      log(host, `auto handoff after context condense: ${handoff}`);
      return;
    }
    const start = taskStarts.get(event.taskId);
    taskStarts.delete(event.taskId);
    if (event.phase === "aborted") return;
    if (event.isSubtask) return;
    const goal = activeGoal();
    const usage = event.tokenUsage || {};
    const cost = Number(usage.totalCost) || 0;
    const tokens = (Number(usage.totalTokensIn) || 0) + (Number(usage.totalTokensOut) || 0);
    if (!goal) {
      writeAudit(host, "goal", "taskCompleted", { taskId: event.taskId, goalActive: false, cost, tokens });
      return;
    }
    const updated = start ? goalMtime() > start.goalMtime : true;
    const openNow = goal.summary.now.filter((l) => !/^- \[[xX]\]/.test(l)).map(stripTask);
    writeAudit(host, "goal", "taskCompleted", { taskId: event.taskId, goalActive: true, goalUpdated: updated, openNow: openNow.length, cost, tokens, durationMs: start ? Date.now() - start.startedAt : void 0 });
    if (!updated) {
      void vscode21.window.showWarningMessage("\uC791\uC5C5\uC774 \uB05D\uB0AC\uC9C0\uB9CC \uBAA9\uD45C \uD30C\uC77C(current.md)\uC774 \uAC31\uC2E0\uB418\uC9C0 \uC54A\uC558\uC2B5\uB2C8\uB2E4.", "\uBAA9\uD45C \uC5F4\uAE30", "\uC774\uC5B4\uC11C \uAC31\uC2E0 \uC694\uCCAD").then(async (choice) => {
        if (choice === "\uBAA9\uD45C \uC5F4\uAE30") await vscode21.commands.executeCommand("vibe-code.openCurrentGoal");
        else if (choice === "\uC774\uC5B4\uC11C \uAC31\uC2E0 \uC694\uCCAD") await startResumeTask(host, "goal not updated after task");
      });
      return;
    }
    if (openNow.length > 0 && (config().get("goalAutoResume") || "ask") !== "never") {
      void vscode21.window.showInformationMessage(`\uBAA9\uD45C Now \uD56D\uBAA9\uC774 ${openNow.length}\uAC1C \uB0A8\uC558\uC2B5\uB2C8\uB2E4: ${openNow[0]}`, "\uACC4\uC18D \uC9C4\uD589", "\uB098\uC911\uC5D0").then(async (choice) => {
        if (choice === "\uACC4\uC18D \uC9C4\uD589") await startResumeTask(host, "next Now item after task");
      });
    }
  } catch (error) {
    log(host, `task event handling skipped: ${error}`);
  }
}
function goalAgeHours2(text) {
  const stamp = matchLine(text, "\uB9C8\uC9C0\uB9C9 \uAC31\uC2E0") || matchLine(text, "\uC791\uC131\uC77C");
  const m = stamp.match(/(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2})/);
  if (!m) return Number.POSITIVE_INFINITY;
  const then = Date.UTC(+m[1], +m[2] - 1, +m[3], +m[4] - 9, +m[5]);
  return (Date.now() - then) / 36e5;
}
function scheduleAutoResume(host) {
  const mode = config().get("goalAutoResume") || "ask";
  if (mode === "never") return;
  const withinHours = config().get("goalResumeWithinHours") ?? 72;
  const timer = setTimeout(async () => {
    try {
      const goal = activeGoal();
      if (!goal) return;
      const openNow = goal.summary.now.filter((l) => !/^- \[[xX]\]/.test(l));
      if (openNow.length === 0) return;
      if (goalAgeHours2(goal.text) > withinHours) return;
      if (mode === "always") {
        await startResumeTask(host, "auto resume on activation");
        return;
      }
      const choice = await vscode21.window.showInformationMessage(`\uD65C\uC131 \uBAA9\uD45C "${goal.summary.title}"\uC5D0 Now \uD56D\uBAA9 ${openNow.length}\uAC1C\uAC00 \uB0A8\uC544 \uC788\uC2B5\uB2C8\uB2E4. \uC774\uC5B4\uC11C \uC9C4\uD589\uD560\uAE4C\uC694?`, "\uC774\uC5B4\uC11C \uC9C4\uD589", "\uBAA9\uD45C \uC5F4\uAE30", "\uB098\uC911\uC5D0");
      if (choice === "\uC774\uC5B4\uC11C \uC9C4\uD589") await startResumeTask(host, "resume prompt on activation");
      else if (choice === "\uBAA9\uD45C \uC5F4\uAE30") await vscode21.commands.executeCommand("vibe-code.openCurrentGoal");
    } catch (error) {
      log(host, `auto resume skipped: ${error}`);
    }
  }, 4e3);
  host.context.subscriptions.push({ dispose: () => clearTimeout(timer) });
}
function registerGoalLoop(host) {
  activeHost2 = host;
  host.context.subscriptions.push(
    { dispose: () => activeHost2 = void 0 },
    vscode21.commands.registerCommand("vibe-code.resumeGoal", async () => {
      const goal = activeGoal();
      if (!goal) {
        void vscode21.window.showWarningMessage("\uD65C\uC131 \uC0C1\uD0DC(\uC0C1\uD0DC: active)\uC778 \uBAA9\uD45C\uAC00 \uC5C6\uC2B5\uB2C8\uB2E4. \uBA3C\uC800 /goal\uB85C \uBAA9\uD45C\uB97C \uC2DC\uC791\uD558\uC138\uC694.");
        return;
      }
      await startResumeTask(host, "manual");
    })
  );
  scheduleAutoResume(host);
  log(host, "goal loop registered (prompt context, task events, auto resume)");
}

// src/features/verification.ts
var import_node_child_process3 = require("node:child_process");
var import_node_child_process4 = require("node:child_process");
var fs18 = __toESM(require("node:fs"));
var path16 = __toESM(require("node:path"));
var vscode22 = __toESM(require("vscode"));
init_log();
init_kst();
init_markdown();
init_workspace();
init_plans();
var RUNNER_PREFIXES = ["npm ", "npx ", "pnpm ", "yarn ", "node ", "go ", "pytest", "python ", "python3 ", "cargo ", "make", "dotnet ", "mvn ", "gradle", "./gradlew", "powershell ", "pwsh ", "bash ", "sh ", "./", "vitest", "jest", "tsc", "eslint", "ruff", "black "];
function extractCommand(line) {
  const text = stripTask(line.trim()).replace(/^-\s+/, "").trim();
  const ticked = text.match(/`([^`]+)`/);
  if (ticked) return ticked[1].trim();
  const first = text.split(/\s+/)[0] || "";
  if (RUNNER_PREFIXES.some((p) => text.startsWith(p)) || /^[\w.-]+(\.sh|\.ps1|\.cmd|\.bat)$/.test(first)) return text;
  return null;
}
function formatVerificationLine(entry) {
  const status = entry.exitCode === 0 ? "OK" : entry.exitCode === null ? "TIMEOUT" : `FAIL exit ${entry.exitCode}`;
  return `- ${entry.stamp} - \`${entry.command}\` \u2192 ${status} (${(entry.durationMs / 1e3).toFixed(1)}s)${entry.note ? ` ${entry.note}` : ""}`;
}
function appendVerificationLog(text, line) {
  if (hasSection(text, "\uAC80\uC99D \uB85C\uADF8")) {
    return writeSection(text, "\uAC80\uC99D \uB85C\uADF8", [...sectionLines(text, "\uAC80\uC99D \uB85C\uADF8"), line]);
  }
  const eol = detectEol(text);
  return restoreEol(normalizeEol(text).replace(/\s*$/, "") + `

## \uAC80\uC99D \uB85C\uADF8
${line}
`, eol);
}
function checkLine(text, lineIndex) {
  const eol = detectEol(text);
  const all = normalizeEol(text).split("\n");
  if (all[lineIndex] && /^- \[ \] /.test(all[lineIndex].trim())) all[lineIndex] = all[lineIndex].replace("- [ ] ", "- [x] ");
  return restoreEol(all.join("\n"), eol);
}
function runShellCommand(host, command, cwd, timeoutMs = 10 * 60 * 1e3) {
  return new Promise((resolve3) => {
    const started = Date.now();
    let output = "";
    host.output.show(true);
    host.output.appendLine(`
$ ${command}   (cwd: ${cwd})`);
    const child = (0, import_node_child_process3.spawn)(command, { cwd, shell: true, env: process.env, windowsHide: true });
    const onData = (chunk) => {
      const text = chunk.toString("utf8");
      output += text;
      for (const line of text.split(/\r?\n/)) if (line) host.output.appendLine(`  ${line}`);
    };
    child.stdout?.on("data", onData);
    child.stderr?.on("data", onData);
    const timer = setTimeout(() => {
      child.kill();
      resolve3({ exitCode: null, durationMs: Date.now() - started, output });
    }, timeoutMs);
    child.on("error", (error) => {
      clearTimeout(timer);
      host.output.appendLine(`  (spawn failed: ${error.message})`);
      resolve3({ exitCode: 127, durationMs: Date.now() - started, output });
    });
    child.on("close", (code) => {
      clearTimeout(timer);
      host.output.appendLine(`\u2192 exit ${code} (${((Date.now() - started) / 1e3).toFixed(1)}s)`);
      resolve3({ exitCode: code, durationMs: Date.now() - started, output });
    });
  });
}
async function runAndRecord(host, command, target) {
  const ws = workspaceRoot() || process.cwd();
  const result = await runShellCommand(host, command, ws);
  const entry = { stamp: kstStamp().human, command, exitCode: result.exitCode, durationMs: result.durationMs };
  const line = formatVerificationLine(entry);
  const touched = [];
  const record = (file, tick) => {
    if (!fs18.existsSync(file)) return;
    let text = readUtf8(file);
    text = appendVerificationLog(text, line);
    if (tick !== void 0 && result.exitCode === 0) text = checkLine(text, tick);
    text = touchPlan(text, null, entry.stamp);
    fs18.writeFileSync(file, text, "utf8");
    touched.push(path16.basename(file));
  };
  if (target) record(target.file, target.lineIndex);
  const paths = ensureWorkspacePaths(true);
  if (paths && fs18.existsSync(paths.current) && (!target || path16.resolve(target.file) !== path16.resolve(paths.current))) record(paths.current);
  writeAudit(host, "plan", "runVerification", { command, exitCode: result.exitCode, durationMs: result.durationMs, recordedIn: touched });
  const label = result.exitCode === 0 ? "\uD1B5\uACFC" : result.exitCode === null ? "\uC2DC\uAC04 \uCD08\uACFC" : `\uC2E4\uD328 (exit ${result.exitCode})`;
  void (result.exitCode === 0 ? vscode22.window.showInformationMessage : vscode22.window.showWarningMessage)(`\uAC80\uC99D ${label}: ${command}`);
  return result;
}
var TEST_FILE_RE = /(\.test\.|\.spec\.|_test\.go$|^test_.*\.py$|\/tests?\/|__tests__\/)/;
function suggestTestCommands(changedFiles, info) {
  const suggestions = [];
  const seen = /* @__PURE__ */ new Set();
  const add = (command, reason) => {
    if (seen.has(command)) return;
    seen.add(command);
    suggestions.push({ command, reason });
  };
  const jsRunner = info.devDependencies.includes("vitest") ? "npx vitest run" : info.devDependencies.includes("jest") ? "npx jest" : info.packageScripts.test ? "npm test --" : "";
  for (const raw of changedFiles) {
    const file = raw.replace(/\\/g, "/");
    const ext = path16.posix.extname(file);
    const dir = path16.posix.dirname(file);
    const base = path16.posix.basename(file, ext);
    if (TEST_FILE_RE.test(file)) {
      if (/\.(ts|tsx|js|jsx|mjs)$/.test(file) && jsRunner) add(`${jsRunner} ${file}`, `\uBCC0\uACBD\uB41C \uD14C\uC2A4\uD2B8 \uD30C\uC77C`);
      else if (file.endsWith("_test.go")) add(`go test ./${dir}/...`, `\uBCC0\uACBD\uB41C Go \uD14C\uC2A4\uD2B8`);
      else if (file.endsWith(".py")) add(`pytest ${file}`, `\uBCC0\uACBD\uB41C \uD30C\uC774\uC36C \uD14C\uC2A4\uD2B8`);
      continue;
    }
    if (/\.(ts|tsx|js|jsx|mjs)$/.test(file) && jsRunner) {
      const candidates = [`${dir}/${base}.test${ext}`, `${dir}/${base}.spec${ext}`, `${dir}/__tests__/${base}.test${ext}`, `tests/unit/${base}.test.ts`, `tests/${base}.test${ext}`, `test/${base}.test${ext}`];
      const hit = candidates.find((c) => info.existingFiles.has(c));
      if (hit) add(`${jsRunner} ${hit}`, `${file} \uC758 \uD14C\uC2A4\uD2B8`);
    } else if (file.endsWith(".go") && info.hasGoMod) {
      add(`go test ./${dir === "." ? "" : dir + "/"}...`, `${file} \uC758 \uD328\uD0A4\uC9C0`);
    } else if (file.endsWith(".py") && info.hasPytest) {
      const candidates = [`${dir}/test_${base}.py`, `tests/test_${base}.py`, `test/test_${base}.py`];
      const hit = candidates.find((c) => info.existingFiles.has(c));
      add(hit ? `pytest ${hit}` : "pytest", hit ? `${file} \uC758 \uD14C\uC2A4\uD2B8` : "\uC804\uCCB4 \uD30C\uC774\uC36C \uD14C\uC2A4\uD2B8");
    } else if (file.endsWith(".rs") && info.hasCargo) {
      add("cargo test", `${file} (Rust)`);
    }
  }
  if (changedFiles.length > 0) {
    if (info.packageScripts.check) add("npm run check", "\uD504\uB85C\uC81D\uD2B8 \uC804\uCCB4 \uAC80\uC0AC \uC2A4\uD06C\uB9BD\uD2B8");
    else if (info.packageScripts.test) add("npm test", "\uD504\uB85C\uC81D\uD2B8 \uD14C\uC2A4\uD2B8 \uC2A4\uD06C\uB9BD\uD2B8");
    if (info.hasGoMod && changedFiles.some((f) => f.endsWith(".go"))) add("go test ./...", "\uC804\uCCB4 Go \uD14C\uC2A4\uD2B8");
  }
  return suggestions;
}
function gitChangedFiles(ws) {
  try {
    const out = (0, import_node_child_process4.execFileSync)("git", ["status", "--porcelain", "--untracked-files=all"], { cwd: ws, encoding: "utf8", windowsHide: true });
    return out.split(/\r?\n/).filter(Boolean).map((l) => l.slice(3).trim().replace(/^"|"$/g, "")).map((l) => l.includes(" -> ") ? l.split(" -> ")[1] : l);
  } catch {
    return [];
  }
}
function readProjectInfo(ws, changed) {
  let packageScripts = {};
  let devDependencies = [];
  try {
    const pkg = JSON.parse(readUtf8(path16.join(ws, "package.json")));
    packageScripts = pkg.scripts || {};
    devDependencies = Object.keys({ ...pkg.devDependencies || {}, ...pkg.dependencies || {} });
  } catch {
  }
  const existing = /* @__PURE__ */ new Set();
  const probe2 = (rel) => {
    if (fs18.existsSync(path16.join(ws, rel))) existing.add(rel);
  };
  for (const raw of changed) {
    const file = raw.replace(/\\/g, "/");
    const ext = path16.posix.extname(file);
    const dir = path16.posix.dirname(file);
    const base = path16.posix.basename(file, ext);
    for (const c of [`${dir}/${base}.test${ext}`, `${dir}/${base}.spec${ext}`, `${dir}/__tests__/${base}.test${ext}`, `tests/unit/${base}.test.ts`, `tests/${base}.test${ext}`, `test/${base}.test${ext}`, `${dir}/test_${base}.py`, `tests/test_${base}.py`, `test/test_${base}.py`]) probe2(c);
  }
  return {
    packageScripts,
    devDependencies,
    hasGoMod: fs18.existsSync(path16.join(ws, "go.mod")),
    hasPytest: fs18.existsSync(path16.join(ws, "pytest.ini")) || fs18.existsSync(path16.join(ws, "pyproject.toml")) || fs18.existsSync(path16.join(ws, "setup.cfg")),
    hasCargo: fs18.existsSync(path16.join(ws, "Cargo.toml")),
    existingFiles: existing
  };
}
var SELECTOR = [
  { scheme: "file", pattern: "**/.vibe-code/plans/*.md" },
  { scheme: "file", pattern: "**/.vibe-code/goals/*.md" }
];
var VerificationCodeLensProvider = class {
  provideCodeLenses(document) {
    const text = document.getText();
    const result = [];
    for (let i = 0; i < document.lineCount; i++) {
      const line = document.lineAt(i).text;
      if (!isTaskLine(line.trim()) && !/^- /.test(line.trim())) continue;
      const sectionName = sectionOfLine(text, i);
      if (!["\uAC80\uC99D \uACC4\uD68D", "\uAC80\uC99D", "\uC644\uB8CC \uAE30\uC900"].includes(sectionName)) continue;
      const command = extractCommand(line);
      if (!command) continue;
      result.push(new vscode22.CodeLens(new vscode22.Range(i, 0, i, 0), { title: "$(play) \uAC80\uC99D \uC2E4\uD589", command: "vibe-code.runVerification", arguments: [document.uri.toString(), i] }));
    }
    return result;
  }
};
function registerVerification(host) {
  const { context } = host;
  context.subscriptions.push(
    vscode22.languages.registerCodeLensProvider(SELECTOR, new VerificationCodeLensProvider()),
    vscode22.commands.registerCommand("vibe-code.runVerification", async (uriString, lineIndex) => {
      const file = vscode22.Uri.parse(uriString).fsPath;
      if (!fs18.existsSync(file)) return;
      const line = readUtf8(file).split(/\r?\n/)[lineIndex] || "";
      const command = extractCommand(line);
      if (!command) {
        void vscode22.window.showWarningMessage("\uC774 \uC904\uC5D0\uC11C \uC2E4\uD589\uD560 \uBA85\uB839\uC744 \uCC3E\uC9C0 \uBABB\uD588\uC2B5\uB2C8\uB2E4. \uBA85\uB839\uC744 \uBC31\uD2F1(`)\uC73C\uB85C \uAC10\uC2F8\uC138\uC694.");
        return;
      }
      await runAndRecord(host, command, { file, lineIndex });
    }),
    vscode22.commands.registerCommand("vibe-code.runVerificationCommand", async (commandArg) => {
      const paths = ensureWorkspacePaths();
      if (!paths) return;
      let command = typeof commandArg === "string" ? commandArg : "";
      let target = null;
      if (!command) {
        const planFile = path16.join(paths.plans, selectPlanName(paths));
        const items = [];
        if (fs18.existsSync(planFile)) {
          const all = readUtf8(planFile).split(/\r?\n/);
          all.forEach((l, i) => {
            const c = sectionOfLine(all.join("\n"), i) === "\uAC80\uC99D \uACC4\uD68D" ? extractCommand(l) : null;
            if (c) items.push({ label: `$(play) ${c}`, description: path16.basename(planFile), command: c, lineIndex: i, file: planFile });
          });
        }
        items.push({ label: "$(edit) \uC9C1\uC811 \uC785\uB825...", command: "" });
        const picked = await vscode22.window.showQuickPick(items, { placeHolder: "\uC2E4\uD589\uD560 \uAC80\uC99D \uBA85\uB839" });
        if (!picked) return;
        command = picked.command || await vscode22.window.showInputBox({ prompt: "\uC2E4\uD589\uD560 \uBA85\uB839", placeHolder: "npm test" }) || "";
        if (!command) return;
        target = picked.file ? { file: picked.file, lineIndex: picked.lineIndex } : null;
      }
      await runAndRecord(host, command, target);
    }),
    vscode22.commands.registerCommand("vibe-code.suggestTests", async () => {
      const ws = workspaceRoot();
      if (!ws) {
        void vscode22.window.showWarningMessage("\uC6CC\uD06C\uC2A4\uD398\uC774\uC2A4\uB97C \uBA3C\uC800 \uC5F4\uC5B4\uC8FC\uC138\uC694.");
        return;
      }
      const changed = gitChangedFiles(ws);
      const suggestions = suggestTestCommands(changed, readProjectInfo(ws, changed));
      if (suggestions.length === 0) {
        void vscode22.window.showInformationMessage(changed.length ? "\uBCC0\uACBD \uD30C\uC77C\uC5D0 \uB300\uC751\uD558\uB294 \uD14C\uC2A4\uD2B8 \uBA85\uB839\uC744 \uCC3E\uC9C0 \uBABB\uD588\uC2B5\uB2C8\uB2E4." : "git \uAE30\uC900 \uBCC0\uACBD\uB41C \uD30C\uC77C\uC774 \uC5C6\uC2B5\uB2C8\uB2E4.");
        return;
      }
      const picked = await vscode22.window.showQuickPick(
        suggestions.map((s) => ({ label: `$(beaker) ${s.command}`, description: s.reason, picked: true, command: s.command })),
        { canPickMany: true, placeHolder: `\uBCC0\uACBD \uD30C\uC77C ${changed.length}\uAC1C \u2192 \uC2E4\uD589\uD560 \uAC80\uC99D \uC120\uD0DD` }
      );
      if (!picked || picked.length === 0) return;
      writeAudit(host, "plan", "suggestTests", { changed: changed.length, selected: picked.map((p) => p.command) });
      for (const item of picked) {
        const result = await runAndRecord(host, item.command, null);
        if (result.exitCode !== 0) break;
      }
    })
  );
  log(host, "verification runner registered");
}

// src/features/goal-lint.ts
var vscode23 = __toESM(require("vscode"));
init_log();
init_markdown();
var STATUSES = ["draft", "active", "blocked", "done"];
var PRIORITIES = ["P0", "P1", "P2", "P3"];
function lineOf(all, predicate, fallback = 0) {
  const idx = all.findIndex(predicate);
  return idx >= 0 ? idx : fallback;
}
function openTasks(block) {
  return taskLines(block).filter((l) => !/^- \[[xX]\]/.test(l)).length;
}
function lintGoal(text) {
  const all = text.split(/\r?\n/);
  const issues = [];
  if (!/^#\s*목표:\s*\S/m.test(text)) issues.push({ line: 0, message: "\uCCAB \uC904\uC740 `# \uBAA9\uD45C: <\uC81C\uBAA9>` \uD615\uC2DD\uC774\uC5B4\uC57C \uD569\uB2C8\uB2E4.", severity: "error" });
  for (const name of ["\uBAA9\uD45C", "\uC644\uB8CC \uAE30\uC900", "\uD604\uC7AC \uC0C1\uD0DC", "\uC791\uC5C5 \uD050", "\uAC80\uC99D \uB85C\uADF8"]) {
    if (!hasSection(text, name)) issues.push({ line: 0, message: `\`## ${name}\` \uC139\uC158\uC774 \uC5C6\uC2B5\uB2C8\uB2E4.`, severity: "warning" });
  }
  const status = matchLine(text, "\uC0C1\uD0DC");
  const statusLine = lineOf(all, (l) => /^상태:/.test(l));
  if (!status) issues.push({ line: 0, message: "`\uC0C1\uD0DC:` \uC904\uC774 \uC5C6\uC2B5\uB2C8\uB2E4 (draft | active | blocked | done).", severity: "warning" });
  else if (!STATUSES.includes(status.toLowerCase())) issues.push({ line: statusLine, message: `\uC54C \uC218 \uC5C6\uB294 \uC0C1\uD0DC "${status}". draft | active | blocked | done \uC911 \uD558\uB098\uC5EC\uC57C \uD569\uB2C8\uB2E4.`, severity: "error" });
  if (!matchLine(text, "- \uB2E4\uC74C \uD589\uB3D9")) issues.push({ line: lineOf(all, (l) => /^## 현재 상태/.test(l)), message: "`- \uB2E4\uC74C \uD589\uB3D9:` \uC904\uC774 \uBE44\uC5B4 \uC788\uC2B5\uB2C8\uB2E4. \uBC14\uB85C \uD560 \uC77C\uC744 \uD55C \uC904\uB85C \uC801\uC73C\uC138\uC694.", severity: "info" });
  const nowOpen = openTasks(subsection(text, "Now"));
  if (nowOpen > 4) issues.push({ line: lineOf(all, (l) => /^### Now/.test(l)), message: `Now\uC5D0 \uBBF8\uC644\uB8CC \uD56D\uBAA9\uC774 ${nowOpen}\uAC1C\uC785\uB2C8\uB2E4. 4\uAC1C \uC774\uD558\uB85C \uC904\uC774\uACE0 \uB098\uBA38\uC9C0\uB294 Next\uB85C \uC62E\uAE30\uC138\uC694.`, severity: "warning" });
  if (status.toLowerCase() === "done") {
    const open = openTasks(section(text, "\uC644\uB8CC \uAE30\uC900"));
    if (open > 0) issues.push({ line: statusLine, message: `\uC0C1\uD0DC\uAC00 done\uC774\uC9C0\uB9CC \uC644\uB8CC \uAE30\uC900 ${open}\uAC1C\uAC00 \uCCB4\uD06C\uB418\uC9C0 \uC54A\uC558\uC2B5\uB2C8\uB2E4.`, severity: "warning" });
  }
  return issues;
}
function lintPlan(text) {
  const all = text.split(/\r?\n/);
  const issues = [];
  if (!/^#\s*계획:\s*\S/m.test(text)) issues.push({ line: 0, message: "\uCCAB \uC904\uC740 `# \uACC4\uD68D: <\uC81C\uBAA9>` \uD615\uC2DD\uC774\uC5B4\uC57C \uD569\uB2C8\uB2E4.", severity: "error" });
  for (const name of ["\uB2E8\uACC4", "Now", "Next", "Done"]) {
    if (!hasSection(text, name)) issues.push({ line: 0, message: `\`## ${name}\` \uC139\uC158\uC774 \uC5C6\uC2B5\uB2C8\uB2E4.`, severity: "warning" });
  }
  const status = matchLine(text, "\uC0C1\uD0DC");
  const statusLine = lineOf(all, (l) => /^상태:/.test(l));
  if (!status) issues.push({ line: 0, message: "`\uC0C1\uD0DC:` \uC904\uC774 \uC5C6\uC2B5\uB2C8\uB2E4 (draft | active | blocked | done).", severity: "warning" });
  else if (!STATUSES.includes(status.toLowerCase())) issues.push({ line: statusLine, message: `\uC54C \uC218 \uC5C6\uB294 \uC0C1\uD0DC "${status}".`, severity: "error" });
  const priority = matchLine(text, "\uC6B0\uC120\uC21C\uC704");
  if (priority && !PRIORITIES.includes(priority.toUpperCase())) issues.push({ line: lineOf(all, (l) => /^우선순위:/.test(l)), message: `\uC54C \uC218 \uC5C6\uB294 \uC6B0\uC120\uC21C\uC704 "${priority}". P0 | P1 | P2 | P3`, severity: "error" });
  const nowOpen = openTasks(section(text, "Now"));
  if (nowOpen > 3) issues.push({ line: lineOf(all, (l) => /^## Now/.test(l)), message: `Now\uC5D0 \uBBF8\uC644\uB8CC \uD56D\uBAA9\uC774 ${nowOpen}\uAC1C\uC785\uB2C8\uB2E4. \uD55C \uBC88\uC5D0 1~3\uAC1C\uB9CC \uB450\uC138\uC694.`, severity: "warning" });
  if (status.toLowerCase() === "done") {
    const open = ["\uB2E8\uACC4", "Now", "\uAC80\uC99D \uACC4\uD68D"].reduce((n, name) => n + openTasks(section(text, name)), 0);
    if (open > 0) issues.push({ line: statusLine, message: `\uC0C1\uD0DC\uAC00 done\uC774\uC9C0\uB9CC \uBBF8\uC644\uB8CC \uD56D\uBAA9\uC774 ${open}\uAC1C \uC788\uC2B5\uB2C8\uB2E4.`, severity: "warning" });
  }
  if (!matchLine(text, "\uC5F0\uACB0 \uBAA9\uD45C")) issues.push({ line: lineOf(all, (l) => /^작성일:/.test(l)), message: "`\uC5F0\uACB0 \uBAA9\uD45C:` \uC904\uC774 \uC5C6\uC2B5\uB2C8\uB2E4. `\uD604\uC7AC \uACC4\uD68D \uBAA9\uD45C \uC5F0\uACB0` \uBA85\uB839\uC73C\uB85C \uBAA9\uD45C\uC640 \uC5F0\uACB0\uD558\uC138\uC694.", severity: "info" });
  return issues;
}
var SEVERITY = { error: vscode23.DiagnosticSeverity.Error, warning: vscode23.DiagnosticSeverity.Warning, info: vscode23.DiagnosticSeverity.Information };
function kindOf(document) {
  const p = document.uri.fsPath.replace(/\\/g, "/");
  if (!p.endsWith(".md") || !p.includes("/.vibe-code/")) return null;
  if (p.includes("/.vibe-code/goals/")) return "goal";
  if (p.includes("/.vibe-code/plans/") && !p.includes("/archive/")) return "plan";
  return null;
}
function registerGoalLint(host) {
  const collection = vscode23.languages.createDiagnosticCollection("vibe-code");
  const timers = /* @__PURE__ */ new Map();
  const lint = (document) => {
    const kind = kindOf(document);
    if (!kind) return;
    const issues = kind === "goal" ? lintGoal(document.getText()) : lintPlan(document.getText());
    collection.set(
      document.uri,
      issues.map((issue) => {
        const line = Math.min(issue.line, Math.max(0, document.lineCount - 1));
        const diagnostic = new vscode23.Diagnostic(new vscode23.Range(line, 0, line, document.lineAt(line).text.length), issue.message, SEVERITY[issue.severity]);
        diagnostic.source = "vibe-code";
        return diagnostic;
      })
    );
  };
  const schedule = (document) => {
    const key = document.uri.toString();
    clearTimeout(timers.get(key));
    timers.set(key, setTimeout(() => lint(document), 300));
  };
  host.context.subscriptions.push(
    collection,
    vscode23.workspace.onDidOpenTextDocument(lint),
    vscode23.workspace.onDidChangeTextDocument((e) => schedule(e.document)),
    vscode23.workspace.onDidCloseTextDocument((d) => collection.delete(d.uri))
  );
  for (const document of vscode23.workspace.textDocuments) lint(document);
  log(host, "goal/plan lint registered");
}

// src/activation.ts
init_goal_health();
init_goal_catalog();

// src/features/goal-metrics.ts
var fs20 = __toESM(require("node:fs"));
var path18 = __toESM(require("node:path"));
var vscode24 = __toESM(require("vscode"));
init_log();
init_kst();
init_markdown();
init_goals();

// src/features/journal-summary.ts
var fs19 = __toESM(require("node:fs"));
var path17 = __toESM(require("node:path"));
init_kst();
init_markdown();
var SUMMARY_MARKER = "<!-- vibe-code:summary-until ";
function parseAuditLines(text) {
  const entries = [];
  for (const line of text.split(/\r?\n/)) {
    if (!line.trim()) continue;
    try {
      const row = JSON.parse(line);
      if (row && typeof row.ts === "string" && typeof row.action === "string") entries.push(row);
    } catch {
    }
  }
  return entries;
}
function lastSummaryTs(journal) {
  const matches = [...journal.matchAll(/<!-- vibe-code:summary-until (\S+) -->/g)];
  return matches.length ? matches[matches.length - 1][1] : "";
}
var count = (entries, kind, action) => entries.filter((e) => e.kind === kind && e.action === action).length;
var str = (value) => typeof value === "string" ? value : "";
function buildSessionSummary(entries, goalText, clock, since) {
  const fresh = since ? entries.filter((e) => e.ts > since) : entries;
  if (fresh.length === 0) return null;
  const lines2 = [`## \uC138\uC158 \uC694\uC57D (${clock})`, ""];
  const completed = fresh.filter((e) => e.kind === "plan" && (e.action === "advanceCurrentPlan" || e.action === "completePlanItem")).map((e) => str(e.details?.completed)).filter(Boolean);
  if (completed.length) lines2.push(`- \uC644\uB8CC\uD55C \uACC4\uD68D \uD56D\uBAA9 ${completed.length}\uAC1C: ${completed.join("; ")}`);
  const statusChanges = fresh.filter((e) => e.kind === "plan" && e.action === "setCurrentPlanStatus").map((e) => `${str(e.details?.status)} (${str(e.details?.planFile).replace(".vibe-code/plans/", "")})`);
  if (statusChanges.length) lines2.push(`- \uACC4\uD68D \uC0C1\uD0DC \uBCC0\uACBD: ${statusChanges.join(", ")}`);
  const archived = fresh.filter((e) => e.kind === "plan" && e.action === "archiveDonePlans").reduce((n, e) => n + (Number(e.details?.count) || 0), 0);
  if (archived) lines2.push(`- \uBCF4\uAD00\uD55C \uC644\uB8CC \uACC4\uD68D: ${archived}\uAC1C`);
  const handoffs = count(fresh, "goal", "createGoalHandoff");
  if (handoffs) lines2.push(`- \uBAA9\uD45C \uD578\uB4DC\uC624\uD504 \uC0DD\uC131: ${handoffs}\uD68C`);
  const proxyApplied = fresh.filter((e) => e.kind === "proxy" && e.action === "applyVibeCodersProxy").map((e) => str(e.details?.profile)).filter(Boolean);
  if (proxyApplied.length) lines2.push(`- vibe-coders \uD504\uB85D\uC2DC \uC801\uC6A9: ${proxyApplied[proxyApplied.length - 1]}`);
  const restored = count(fresh, "proxy", "restorePreviousProviderProfile");
  if (restored) lines2.push(`- \uC774\uC804 \uD504\uB85C\uBC14\uC774\uB354 \uBCF5\uC6D0: ${restored}\uD68C`);
  const commandsRun = fresh.filter((e) => e.kind === "command" && e.action === "approved").length;
  const commandsDenied = fresh.filter((e) => e.kind === "command" && e.action === "denied").length;
  const commandsFailed = fresh.filter((e) => e.kind === "command" && e.action === "exited" && Number(e.details?.exitCode) !== 0).map((e) => `${str(e.details?.command)} (exit ${e.details?.exitCode})`);
  if (commandsRun || commandsDenied) lines2.push(`- \uC2E4\uD589\uD55C \uBA85\uB839: ${commandsRun}\uAC1C \uC2B9\uC778, ${commandsDenied}\uAC1C \uAC70\uBD80${commandsFailed.length ? ` \xB7 \uC2E4\uD328 ${commandsFailed.length}\uAC1C: ${commandsFailed.slice(0, 3).join("; ")}` : ""}`);
  const failures = fresh.filter((e) => e.action.endsWith("Failed")).map((e) => `${e.action}: ${str(e.details?.error)}`);
  if (failures.length) lines2.push(`- \uC2E4\uD328: ${failures.join("; ")}`);
  const byKind = /* @__PURE__ */ new Map();
  for (const e of fresh) byKind.set(e.kind, (byKind.get(e.kind) || 0) + 1);
  lines2.push(`- \uAC10\uC0AC \uC774\uBCA4\uD2B8: ${[...byKind].map(([k, n]) => `${k} ${n}`).join(", ")}`);
  if (goalText) {
    const next = matchLine(goalText, "- \uB2E4\uC74C \uD589\uB3D9");
    const now = taskLines(subsection(goalText, "Now")).filter((l) => !/^- \[[xX]\]/.test(l)).map(stripTask);
    if (next) lines2.push(`- \uB2E4\uC74C \uD589\uB3D9: ${next}`);
    if (now.length) lines2.push(`- \uBAA9\uD45C Now: ${now.slice(0, 3).join("; ")}${now.length > 3 ? ` \uC678 ${now.length - 3}\uAC1C` : ""}`);
  }
  const until = fresh[fresh.length - 1].ts;
  lines2.push("", `${SUMMARY_MARKER}${until} -->`, "");
  return lines2.join("\n");
}
function appendSessionSummary(workspaceRoot2) {
  try {
    if (!workspaceRoot2) return false;
    const today = kstDate();
    const auditFile = path17.join(workspaceRoot2, ".vibe-code", "audit", `${today}.jsonl`);
    const journalFile = path17.join(workspaceRoot2, ".vibe-code", "journal", `${today}.md`);
    if (!fs19.existsSync(auditFile) || !fs19.existsSync(journalFile)) return false;
    const journal = fs19.readFileSync(journalFile, "utf8");
    const goalFile = path17.join(workspaceRoot2, ".vibe-code", "goals", "current.md");
    const goalText = fs19.existsSync(goalFile) ? fs19.readFileSync(goalFile, "utf8") : "";
    const summary = buildSessionSummary(parseAuditLines(fs19.readFileSync(auditFile, "utf8")), goalText, kstClock(), lastSummaryTs(journal));
    if (!summary) return false;
    fs19.appendFileSync(journalFile, (journal.endsWith("\n") ? "" : "\n") + "\n" + summary, "utf8");
    return true;
  } catch {
    return false;
  }
}

// src/features/goal-metrics.ts
init_workspace();
var str2 = (v) => typeof v === "string" ? v : "";
var num = (v) => typeof v === "number" && Number.isFinite(v) ? v : Number(v) || 0;
function readRecentAudit(auditDir, days) {
  if (!fs20.existsSync(auditDir)) return [];
  const files = fs20.readdirSync(auditDir).filter((n) => /^\d{4}-\d{2}-\d{2}\.jsonl$/.test(n)).sort().slice(-days);
  return files.flatMap((n) => parseAuditLines(readUtf8(path18.join(auditDir, n))));
}
function computeGoalMetrics(entries, days) {
  const completedItems = entries.filter((e) => e.kind === "plan" && (e.action === "advanceCurrentPlan" || e.action === "completePlanItem")).map((e) => str2(e.details?.completed)).filter(Boolean);
  const verifications = entries.filter((e) => e.kind === "plan" && e.action === "runVerification");
  const passed = verifications.filter((e) => num(e.details?.exitCode) === 0 && e.details?.exitCode !== null).length;
  const failedVerifications = verifications.filter((e) => e.details?.exitCode === null || num(e.details?.exitCode) !== 0).map((e) => str2(e.details?.command)).filter(Boolean);
  const tasks = entries.filter((e) => e.kind === "goal" && e.action === "taskCompleted");
  const planStatusChanges = {};
  for (const e of entries.filter((e2) => e2.kind === "plan" && e2.action === "setCurrentPlanStatus")) {
    const s = str2(e.details?.status) || "?";
    planStatusChanges[s] = (planStatusChanges[s] || 0) + 1;
  }
  const activeDays = new Set(entries.map((e) => e.ts.slice(0, 10))).size;
  return {
    days,
    completedItems,
    completedPerDay: days > 0 ? completedItems.length / days : 0,
    verifications: verifications.length,
    verificationPassRate: verifications.length ? passed / verifications.length : 0,
    failedVerifications,
    commandsApproved: entries.filter((e) => e.kind === "command" && e.action === "approved").length,
    commandsDenied: entries.filter((e) => e.kind === "command" && e.action === "denied").length,
    commandsFailed: entries.filter((e) => e.kind === "command" && e.action === "exited" && num(e.details?.exitCode) !== 0).length,
    tasks: tasks.length,
    tasksWithoutGoalUpdate: tasks.filter((e) => e.details?.goalActive === true && e.details?.goalUpdated === false).length,
    tokens: tasks.reduce((n, e) => n + num(e.details?.tokens), 0),
    costUsd: tasks.reduce((n, e) => n + num(e.details?.cost), 0),
    handoffs: entries.filter((e) => e.kind === "goal" && (e.action === "createGoalHandoff" || e.action === "autoHandoff")).length,
    healthWarnings: entries.filter((e) => e.kind === "goal" && e.action === "healthWarning").length,
    planStatusChanges,
    activeDays
  };
}
function formatMetrics(m) {
  const pct = (r) => `${(r * 100).toFixed(0)}%`;
  return [
    `\uAE30\uAC04: \uCD5C\uADFC ${m.days}\uC77C (\uD65C\uB3D9\uC77C ${m.activeDays}\uC77C)`,
    `\uC644\uB8CC \uD56D\uBAA9: ${m.completedItems.length}\uAC1C (\uD558\uB8E8 ${m.completedPerDay.toFixed(1)}\uAC1C)`,
    `\uAC80\uC99D \uC2E4\uD589: ${m.verifications}\uD68C, \uD1B5\uACFC\uC728 ${pct(m.verificationPassRate)}${m.failedVerifications.length ? ` \xB7 \uC2E4\uD328: ${[...new Set(m.failedVerifications)].slice(0, 3).join("; ")}` : ""}`,
    `\uBA85\uB839 \uC2E4\uD589: \uC2B9\uC778 ${m.commandsApproved} \xB7 \uAC70\uBD80 ${m.commandsDenied} \xB7 \uC2E4\uD328 \uC885\uB8CC ${m.commandsFailed}`,
    `\uC791\uC5C5: ${m.tasks}\uD68C (\uBAA9\uD45C \uBBF8\uAC31\uC2E0 ${m.tasksWithoutGoalUpdate}\uD68C) \xB7 \uD1A0\uD070 ${m.tokens.toLocaleString("ko-KR")} \xB7 \uBE44\uC6A9 $${m.costUsd.toFixed(2)}`,
    `\uD578\uB4DC\uC624\uD504 ${m.handoffs}\uD68C \xB7 \uC815\uCCB4 \uACBD\uACE0 ${m.healthWarnings}\uD68C \xB7 \uACC4\uD68D \uC0C1\uD0DC \uBCC0\uACBD ${Object.entries(m.planStatusChanges).map(([k, v]) => `${k} ${v}`).join(", ") || "\uC5C6\uC74C"}`
  ];
}
function buildRetro(input) {
  const { metrics: m, entries, goalText } = input;
  const goal = goalText ? parseGoal(goalText) : void 0;
  const failing = /* @__PURE__ */ new Map();
  for (const c of m.failedVerifications) failing.set(c, (failing.get(c) || 0) + 1);
  for (const e of entries.filter((e2) => e2.kind === "command" && e2.action === "exited" && num(e2.details?.exitCode) !== 0)) {
    const c = str2(e.details?.command);
    if (c) failing.set(c, (failing.get(c) || 0) + 1);
  }
  const bottlenecks = [...failing].sort((a, b) => b[1] - a[1]).slice(0, 5);
  const switches = entries.filter((e) => e.kind === "goal" && (e.action === "switchGoal" || e.action === "newGoal")).length;
  const lines2 = [
    `# \uC8FC\uAC04 \uD68C\uACE0 \u2014 ${input.from} ~ ${input.to}`,
    "",
    "## \uC774\uBC88 \uC8FC \uD55C\uB208\uC5D0",
    ...formatMetrics(m).map((l) => `- ${l}`),
    ...input.usage ? [`- vibe-coders \uC0AC\uC6A9\uB7C9(\uC8FC\uAC04): ${formatKrw(input.usage.cost_krw)} \xB7 \uC694\uCCAD ${input.usage.requests}\uD68C \xB7 \uD1A0\uD070 ${input.usage.tokens.toLocaleString("ko-KR")} \xB7 \uC131\uACF5\uB960 ${(input.usage.success_rate * 100).toFixed(1)}%`] : [],
    ...switches ? [`- \uBAA9\uD45C \uC0DD\uC131/\uC804\uD658 ${switches}\uD68C`] : [],
    "",
    "## \uC644\uB8CC\uD55C \uAC83",
    ...m.completedItems.length ? m.completedItems.map((c) => `- [x] ${c}`) : ["- (\uC644\uB8CC \uD56D\uBAA9 \uC5C6\uC74C)"],
    "",
    "## \uBCD1\uBAA9\uACFC \uC2E4\uD328",
    ...bottlenecks.length ? bottlenecks.map(([c, n]) => `- \`${c}\` \u2014 \uC2E4\uD328 ${n}\uD68C`) : ["- \uBC18\uBCF5 \uC2E4\uD328 \uC5C6\uC74C"],
    ...m.tasksWithoutGoalUpdate ? [`- \uBAA9\uD45C \uD30C\uC77C\uC744 \uAC31\uC2E0\uD558\uC9C0 \uC54A\uC740 \uC791\uC5C5 ${m.tasksWithoutGoalUpdate}\uD68C \u2014 \uB8E8\uD504 \uADDC\uCE59\uC744 \uB2E4\uC2DC \uD655\uC778\uD558\uC138\uC694.`] : [],
    ...m.healthWarnings ? [`- \uC815\uCCB4 \uACBD\uACE0 ${m.healthWarnings}\uD68C`] : [],
    "",
    "## \uB2E4\uC74C \uC8FC"
  ];
  if (goal) {
    lines2.push(`- \uBAA9\uD45C: ${goal.title} (${goal.status}) \u2014 \uC644\uB8CC \uAE30\uC900 ${goal.done}/${goal.total}`);
    lines2.push(`- \uB2E4\uC74C \uD589\uB3D9: ${goal.next}`);
    const now = goal.now.filter((l) => !/^- \[[xX]\]/.test(l)).map(stripTask);
    const criteria = goal.criteria.filter((l) => !/^- \[[xX]\]/.test(l)).map(stripTask);
    for (const n of now.slice(0, 5)) lines2.push(`- [ ] Now: ${n}`);
    for (const c of criteria.slice(0, 5)) lines2.push(`- [ ] \uC644\uB8CC \uAE30\uC900: ${c}`);
  } else {
    lines2.push("- \uD65C\uC131 \uBAA9\uD45C \uC5C6\uC74C \u2014 `Vibe Code: \uC0C8 \uBAA9\uD45C`\uB85C \uC2DC\uC791\uD558\uC138\uC694.");
  }
  lines2.push("");
  return lines2.join("\n");
}
function registerGoalMetrics(host) {
  const { context, output } = host;
  context.subscriptions.push(
    vscode24.commands.registerCommand("vibe-code.showGoalMetrics", async () => {
      const paths = ensureWorkspacePaths();
      if (!paths) return;
      const metrics = computeGoalMetrics(readRecentAudit(paths.audit, 7), 7);
      output.show(true);
      output.appendLine("\n=== \uBAA9\uD45C \uC9C4\uD589 \uC9C0\uD45C (7\uC77C) ===");
      for (const line of formatMetrics(metrics)) output.appendLine(`  ${line}`);
      output.appendLine("================================\n");
      writeAudit(host, "goal", "showGoalMetrics", { completed: metrics.completedItems.length, verifications: metrics.verifications });
    }),
    vscode24.commands.registerCommand("vibe-code.createRetro", async () => {
      const paths = ensureWorkspacePaths();
      if (!paths) return;
      const entries = readRecentAudit(paths.audit, 7);
      const metrics = computeGoalMetrics(entries, 7);
      const usage = await cachedWeeklyUsage(host).catch(() => null);
      const to = kstDate();
      const from = kstDate(new Date(Date.now() - 6 * 864e5));
      const retro = buildRetro({ from, to, metrics, entries, goalText: fs20.existsSync(paths.current) ? readUtf8(paths.current) : "", usage: usage?.ok ? usage.report : void 0 });
      const dir = path18.join(paths.base, "retro");
      fs20.mkdirSync(dir, { recursive: true });
      const file = path18.join(dir, `${to}-weekly.md`);
      fs20.writeFileSync(file, retro, "utf8");
      const doc = await vscode24.workspace.openTextDocument(file);
      await vscode24.window.showTextDocument(doc);
      writeAudit(host, "goal", "createRetro", { file: `.vibe-code/retro/${path18.basename(file)}`, completed: metrics.completedItems.length });
      log(host, `retro written: ${file}`);
    })
  );
  log(host, "goal metrics + retro commands registered");
}

// src/activation.ts
async function registerGoalPlanFeatures(host) {
  registerShowContextStats(host);
  registerJournalCommands(host);
  registerGoalCommands(host);
  registerPlanCommands(host);
  registerGoalTrackerView(host);
  registerPlanViews(host);
  log(host, "aux commands registered (showContextStats, openJournal, openCurrentGoal, showGoalStatus, createGoalHandoff, openLatestPlan)");
}
var STEPS = [
  ["language sync skipped", syncLanguageFromSettings],
  ["proxy/CA setup skipped", applyNetworkEnv],
  ["status bar setup failed", createModeStatusBar],
  ["goal status bar setup failed", createGoalStatusBar],
  ["demo seed skipped", seedDemoScenarios],
  ["vibeignore seed skipped", seedVibeignore],
  ["telemetry default skipped", applyTelemetryDefault],
  ["default mode setup skipped", applyDefaultMode],
  ["vibemodes seed skipped", seedVibemodes],
  ["autonomy preset skipped", applyAutonomyPreset],
  ["journal init skipped", initJournal],
  ["aux command registration failed", registerGoalPlanFeatures],
  ["slash command seed skipped", seedSlashCommands],
  ["mcp seed skipped", seedMcpRecommendations],
  ["update check skipped", checkForUpdates],
  ["welcome skipped", showWelcome],
  ["aux2 command registration failed", registerDiagnostics],
  ["team config commands skipped", registerTeamConfig],
  ["auditNetwork registration failed", registerAuditNetwork],
  ["vibe-coders proxy registration failed", registerVibeCodersProxy],
  ["setup wizard registration failed", registerOnboarding],
  ["plan CodeLens registration failed", registerPlanCodeLens],
  ["command audit registration failed", registerCommandAudit],
  ["usage report registration failed", registerUsageReport],
  ["usage dashboard registration failed", registerUsageDashboard],
  ["goal loop registration failed", registerGoalLoop],
  ["verification runner registration failed", registerVerification],
  ["goal lint registration failed", registerGoalLint],
  ["goal catalog registration failed", registerGoalCatalog],
  ["goal metrics registration failed", registerGoalMetrics],
  ["checkpoint commands registration failed", registerCheckpoints],
  ["goal health check skipped", checkGoalHealth],
  ["language default init skipped", initializeLanguage]
];
async function runActivation(host) {
  for (const [failure, run] of STEPS) {
    try {
      await run(host);
    } catch (error) {
      log(host, `${failure}: ${error}`);
    }
  }
}

// src/core/hooks.ts
function installHooks() {
  const hooks = {
    beforeCore: runActivation,
    mergeLocaleOverrides,
    onCommand: handleCommandEvent,
    onTask: handleTaskEvent,
    promptContext
  };
  globalThis.__vibeCode = hooks;
  return hooks;
}

// src/features/handoff-on-exit.ts
var fs21 = __toESM(require("node:fs"));
var path19 = __toESM(require("node:path"));
init_kst();
init_markdown();
init_goals();
function lastHandoffMs(sessions) {
  if (!fs21.existsSync(sessions)) return 0;
  return fs21.readdirSync(sessions).filter((n) => n.endsWith("-handoff.md")).reduce((max, n) => Math.max(max, fs21.statSync(path19.join(sessions, n)).mtimeMs), 0);
}
function handoffOnExit(workspaceRoot2) {
  try {
    if (!workspaceRoot2) return null;
    const base = path19.join(workspaceRoot2, ".vibe-code");
    const current = path19.join(base, "goals", "current.md");
    if (!fs21.existsSync(current)) return null;
    const text = fs21.readFileSync(current, "utf8");
    if (matchLine(text, "\uC0C1\uD0DC").toLowerCase() !== "active") return null;
    const audit = path19.join(base, "audit", `${kstDate()}.jsonl`);
    if (!fs21.existsSync(audit)) return null;
    const sessions = path19.join(base, "sessions");
    fs21.mkdirSync(sessions, { recursive: true });
    const since = lastHandoffMs(sessions);
    const fresh = fs21.readFileSync(audit, "utf8").split(/\r?\n/).filter(Boolean).some((line) => {
      try {
        return new Date(JSON.parse(line).ts).getTime() > since;
      } catch {
        return false;
      }
    });
    if (!fresh) return null;
    const stamp = kstStamp();
    const file = path19.join(sessions, `${stamp.file}-handoff.md`);
    fs21.writeFileSync(file, handoffTemplate(stamp.human, text), "utf8");
    fs21.appendFileSync(current, `
- ${stamp.human} - handoff \uC0DD\uC131: .vibe-code/sessions/${path19.basename(file)} (\uC138\uC158 \uC885\uB8CC)
`, "utf8");
    return file;
  } catch {
    return null;
  }
}

// src/extension.ts
installHooks();
var core = require("./extension.core.js");
function activate(context) {
  return core.activate(context);
}
function deactivate() {
  const vscode25 = require("vscode");
  const workspaceRoot2 = vscode25.workspace.workspaceFolders?.[0]?.uri.fsPath;
  handoffOnExit(workspaceRoot2);
  appendSessionSummary(workspaceRoot2);
  return core.deactivate();
}
// Annotate the CommonJS export names for ESM import in node:
0 && (module.exports = {
  activate,
  deactivate
});
