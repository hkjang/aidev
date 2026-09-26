// Replica of src/util/markdown.ts regexes (copied verbatim) to test CRLF behaviour.
const esc = (v) => v.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
const section = (text, name) => {
	const re = new RegExp("(?:^|\\n)## " + esc(name) + "\\n([\\s\\S]*?)(?=\\n## |$)");
	return (text.match(re)?.[1] || "").trim();
};
const subsection = (text, name) => {
	const re = new RegExp("(?:^|\\n)### " + esc(name) + "\\n([\\s\\S]*?)(?=\\n### |\\n## |$)");
	return (text.match(re)?.[1] || "").trim();
};
const writeSection = (text, name, ls) => {
	const re = new RegExp("((?:^|\\n)## " + esc(name) + "\\n)([\\s\\S]*?)(?=\\n## |$)");
	if (!re.test(text)) return text;
	return text.replace(re, (_m, head) => head + (ls.length > 0 ? ls.join("\n") + "\n" : "\n"));
};
const matchLine = (text, label, fb = "") => {
	const m = text.match(new RegExp("^" + esc(label) + ":\\s*(.+)$", "m"));
	return (m?.[1] || fb).trim();
};
const headingTitle = (t, p, fb) => {
	const m = t.match(new RegExp("^#\\s*" + esc(p) + ":\\s*(.+)$", "m"));
	return (m?.[1] || fb).trim();
};
const moveTaskToSection = (text, i, target) => {
	const all = text.split(/\r?\n/);
	const line = all[i];
	if (line === undefined || !/^- \[( |x|X)\] /.test(line.trim())) return null;
	const re = new RegExp("(?:^|\\n)## " + esc(target) + "\\n");
	if (!re.test(text)) return null;
	return "WOULD-MOVE";
};
const countChecks = (t) => [...t.matchAll(/^- \[( |x|X)\] /gm)].length;
const lines = (b) => b.split(/\r?\n/).map((l) => l.trim()).filter(Boolean);
const taskLines = (b) => lines(b).filter((l) => l.startsWith("- ["));

const LF = [
	"# 계획: 테스트",
	"",
	"상태: draft",
	"작성일: 2026-09-27",
	"마지막 갱신: 2026-09-27",
	"",
	"## 단계",
	"- [ ] 단계 하나",
	"",
	"## Now",
	"- [ ] 지금 할 일",
	"",
	"## Next",
	"- [ ] 다음 일",
	"",
	"## Done",
	"",
	"## Risks",
	"- 없음",
	"",
].join("\n");
const CRLF = LF.replace(/\n/g, "\r\n");

for (const [tag, t] of [["LF", LF], ["CRLF", CRLF]]) {
	console.log("--- " + tag + " ---");
	console.log("headingTitle:", JSON.stringify(headingTitle(t, "계획", "(없음)")));
	console.log("matchLine 상태:", JSON.stringify(matchLine(t, "상태")));
	console.log("section(단계):", JSON.stringify(section(t, "단계")));
	console.log("section(Now):", JSON.stringify(section(t, "Now")));
	console.log("taskLines(section Now).length:", taskLines(section(t, "Now")).length);
	console.log("countChecks(whole text):", countChecks(t));
	const idx = t.split(/\r?\n/).findIndex((l) => l.includes("지금 할 일"));
	console.log("moveTaskToSection(Now -> Done):", moveTaskToSection(t, idx, "Done"));
	console.log("writeSection(Now) changed?", writeSection(t, "Now", ["- [ ] 새 항목"]) !== t);
	console.log("lint sees '## Done'?", new RegExp("(?:^|\\n)## Done\\n").test(t));
}

// indented sub-task disagreement (secondary candidate)
const IND = "## Now\n- [ ] parent\n  - [ ] child\n";
console.log("--- indented ---");
console.log("taskLines:", taskLines(section(IND, "Now")).length, "countChecks:", countChecks(IND));
