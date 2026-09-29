// Pure helpers for the markdown state files under .vibe-code/ (goals, plans).

const escapeRegExp = (value: string): string => value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

/**
 * 입력의 줄바꿈 방식. 첫 줄바꿈이 `\r\n` 이면 CRLF, 그 밖에는(줄바꿈이 없을 때도) LF 다.
 * 혼합 줄바꿈 파일은 첫 줄바꿈을 따른다 — VS Code 의 `TextDocument.eol` 과 같은 규칙.
 */
export function detectEol(text: string): "\n" | "\r\n" {
	const first = text.indexOf("\n");
	return first > 0 && text[first - 1] === "\r" ? "\r\n" : "\n";
}

/** CRLF 를 LF 로 바꾼 사본. 아래 파서들은 모두 이 형태만 본다. */
export function normalizeEol(text: string): string {
	return text.includes("\r") ? text.replace(/\r\n/g, "\n") : text;
}

/** 정규화된(LF) 텍스트를 원래 줄바꿈으로 되돌린다. 정규화하지 않은 텍스트에 쓰면 `\r\r\n` 이 된다. */
export function restoreEol(normalized: string, eol: "\n" | "\r\n"): string {
	return eol === "\r\n" ? normalized.replace(/\n/g, "\r\n") : normalized;
}

function sectionHeadRe(name: string): RegExp {
	return new RegExp("((?:^|\\n)## " + escapeRegExp(name) + "\\n)");
}

/** Whether the text has a `## <name>` section. Parser and lint share this one check. */
export function hasSection(text: string, name: string): boolean {
	return sectionHeadRe(name).test(normalizeEol(text));
}

/** Value of a `label: value` line, trimmed; `fallback` when absent or empty. */
export function matchLine(text: string, label: string, fallback = ""): string {
	const match = normalizeEol(text).match(new RegExp("^" + escapeRegExp(label) + ":([^\\r\\n]*)$", "m"));
	return match?.[1]?.trim() || fallback;
}

/** Title from a `# <prefix>: <title>` heading. */
export function headingTitle(text: string, prefix: string, fallback: string): string {
	const match = normalizeEol(text).match(new RegExp("^#\\s*" + escapeRegExp(prefix) + ":\\s*(.+)$", "m"));
	return (match?.[1] || fallback).trim();
}

/** Body of a `## <name>` section up to the next `## ` heading, trimmed. */
export function section(text: string, name: string): string {
	const re = new RegExp("(?:^|\\n)## " + escapeRegExp(name) + "\\n([\\s\\S]*?)(?=\\n## |$)");
	return (normalizeEol(text).match(re)?.[1] || "").trim();
}

/** Body of a `### <name>` sub-section up to the next `### ` or `## ` heading, trimmed. */
export function subsection(text: string, name: string): string {
	const re = new RegExp("(?:^|\\n)### " + escapeRegExp(name) + "\\n([\\s\\S]*?)(?=\\n### |\\n## |$)");
	return (normalizeEol(text).match(re)?.[1] || "").trim();
}

/**
 * Replace the body of a `## <name>` section with `lines`; unchanged when the section is missing.
 * 입력의 줄바꿈을 그대로 돌려준다.
 */
export function writeSection(text: string, name: string, lines: string[]): string {
	const eol = detectEol(text);
	const body = normalizeEol(text);
	const re = new RegExp(sectionHeadRe(name).source + "([\\s\\S]*?)(?=\\n## |$)");
	if (!re.test(body)) return text;
	return restoreEol(
		body.replace(re, (_match, head: string) => head + (lines.length > 0 ? lines.join("\n") + "\n" : "\n")),
		eol,
	);
}

/**
 * Lines of a `## <name>` section exactly as written — indentation and non-checklist notes kept.
 * An empty or missing section is `[]`, never `[""]`. Pair with `writeSection` to edit a section
 * without dropping the lines the editor does not care about.
 */
export function sectionLines(text: string, name: string): string[] {
	const body = section(text, name);
	return body === "" ? [] : body.split("\n");
}

/** Trimmed, non-empty lines of a block. */
export function lines(block: string): string[] {
	return block
		.split(/\r?\n/)
		.map((line) => line.trim())
		.filter(Boolean);
}

/** Checklist lines (`- [ ]` / `- [x]`) of a block. */
export function taskLines(block: string): string[] {
	return lines(block).filter((line) => line.startsWith("- ["));
}

export function isDoneTask(line: string): boolean {
	return /^- \[[xX]\]/.test(line);
}

/** Checklist line without its `- [ ] ` prefix. */
export function stripTask(line: string): string {
	return line.replace(/^- \[(?: |x|X)\]\s*/, "");
}

export interface CheckCount {
	done: number;
	total: number;
}

/** Counts every checklist item in the text. */
export function countChecks(text: string): CheckCount {
	const checks = [...normalizeEol(text).matchAll(/^- \[( |x|X)\] /gm)];
	return { done: checks.filter((m) => m[1].toLowerCase() === "x").length, total: checks.length };
}

export function progressLabel(count: CheckCount, empty: string): string {
	return count.total > 0 ? `${count.done}/${count.total}` : empty;
}

/**
 * Set a `label: value` metadata line. Missing lines are inserted after `마지막 갱신:`,
 * or after `작성일:` when there is no `마지막 갱신:` line. 입력의 줄바꿈을 그대로 돌려준다.
 */
export function setLine(text: string, label: string, value: string): string {
	const eol = detectEol(text);
	const body = normalizeEol(text);
	const re = new RegExp("^" + escapeRegExp(label) + ":[^\\r\\n]*$", "m");
	const line = label + ": " + value;
	const next = re.test(body)
		? body.replace(re, () => line)
		: /^마지막 갱신:[^\r\n]*$/m.test(body)
			? body.replace(/^마지막 갱신:[^\r\n]*$/m, (m) => m + "\n" + line)
			: body.replace(/^작성일:[^\r\n]*$/m, (m) => m + "\n" + line);
	return restoreEol(next, eol);
}

/** Update `상태:` (when given) and bump `마지막 갱신:` to `stamp`. 입력의 줄바꿈을 그대로 돌려준다. */
export function touchPlan(text: string, status: string | null, stamp: string): string {
	const eol = detectEol(text);
	let next = normalizeEol(text);
	if (status) {
		if (/^상태:[^\r\n]*$/m.test(next)) next = next.replace(/^상태:[^\r\n]*$/m, () => "상태: " + status);
		else next = next.replace(/^# .+$/m, (m) => m + "\n\n상태: " + status);
	}
	if (/^마지막 갱신:[^\r\n]*$/m.test(next)) next = next.replace(/^마지막 갱신:[^\r\n]*$/m, () => "마지막 갱신: " + stamp);
	else if (/^작성일:[^\r\n]*$/m.test(next)) next = next.replace(/^작성일:[^\r\n]*$/m, (m) => m + "\n마지막 갱신: " + stamp);
	return restoreEol(next, eol);
}

const PRIORITY_RANK: Record<string, number> = { P0: 0, P1: 1, P2: 2, P3: 3 };

/** `P0` sorts first; unknown priorities last. */
export function priorityRank(priority: string | undefined): number {
	const value = String(priority || "P2").trim().toUpperCase();
	return PRIORITY_RANK[value] ?? 9;
}

// --- line-level editing (used by the plan CodeLens commands) ---------------------------------

/** Name of the `## ` section that contains `lineIndex`, or "" when above the first heading. */
export function sectionOfLine(text: string, lineIndex: number): string {
	const all = text.split(/\r?\n/);
	for (let i = Math.min(lineIndex, all.length - 1); i >= 0; i--) {
		const m = all[i].match(/^## (.+)$/);
		if (m) return m[1].trim();
		if (/^# /.test(all[i])) return "";
	}
	return "";
}

export function isTaskLine(line: string): boolean {
	return /^- \[( |x|X)\] /.test(line);
}

/** Flip `- [ ]` <-> `- [x]` on one line; other lines are returned unchanged. 줄바꿈은 보존한다. */
export function toggleCheckbox(text: string, lineIndex: number): string {
	const eol = detectEol(text);
	const all = normalizeEol(text).split("\n");
	const line = all[lineIndex];
	if (line === undefined || !isTaskLine(line)) return text;
	all[lineIndex] = isDoneTask(line) ? line.replace(/^- \[[xX]\]/, "- [ ]") : line.replace(/^- \[ \]/, "- [x]");
	return restoreEol(all.join("\n"), eol);
}

/**
 * Remove the task at `lineIndex` and append it (optionally transformed) to the end of
 * `## targetSection`. Returns null when the line is not a task or the section is missing.
 * 줄바꿈은 보존한다.
 */
export function moveTaskToSection(text: string, lineIndex: number, targetSection: string, transform: (line: string) => string = (l) => l): string | null {
	const eol = detectEol(text);
	const all = normalizeEol(text).split("\n");
	const line = all[lineIndex];
	if (line === undefined || !isTaskLine(line.trim())) return null;
	if (!hasSection(all.join("\n"), targetSection)) return null;
	all.splice(lineIndex, 1);
	const without = all.join("\n");
	// 대상 섹션은 원본 줄 그대로 다시 쓴다 — lines() 로 재조립하면 들여쓰기와 메모 줄이 사라진다.
	return restoreEol(writeSection(without, targetSection, [...sectionLines(without, targetSection), transform(line.trim())]), eol);
}
