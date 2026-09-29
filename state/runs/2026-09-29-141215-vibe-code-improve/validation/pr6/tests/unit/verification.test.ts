import { describe, expect, it } from "vitest";
import { appendVerificationLog, checkLine, extractCommand, formatVerificationLine, suggestTestCommands, ProjectInfo } from "../../src/features/verification";
import { openPlanItems, planTemplate } from "../../src/features/plans";
import { goalTemplate, openCriteria } from "../../src/features/goals";
import { normalizeEol, section, sectionLines, writeSection } from "../../src/util/markdown";

describe("extractCommand", () => {
	it("prefers backticked commands and recognizes runner prefixes", () => {
		expect(extractCommand("- [ ] 테스트 실행: `npm test`")).toBe("npm test");
		expect(extractCommand("- [ ] npm run check")).toBe("npm run check");
		expect(extractCommand("- go test ./...")).toBe("go test ./...");
		expect(extractCommand("- [ ] ./scripts/verify.sh")).toBe("./scripts/verify.sh");
		expect(extractCommand("- [ ] 관련 스크립트 또는 테스트 실행")).toBeNull();
		expect(extractCommand("- [x] 문서 갱신")).toBeNull();
	});
});

describe("verification log", () => {
	it("formats entries and appends to an existing or new 검증 로그 section", () => {
		const line = formatVerificationLine({ stamp: "2026-09-22 10:00:00 KST", command: "npm test", exitCode: 0, durationMs: 1234 });
		expect(line).toBe("- 2026-09-22 10:00:00 KST - `npm test` → OK (1.2s)");
		expect(formatVerificationLine({ stamp: "s", command: "x", exitCode: 2, durationMs: 0 })).toContain("FAIL exit 2");
		expect(formatVerificationLine({ stamp: "s", command: "x", exitCode: null, durationMs: 0 })).toContain("TIMEOUT");
		const goal = appendVerificationLog(goalTemplate("s"), line);
		expect(section(goal, "검증 로그")).toBe(line);
		const plan = appendVerificationLog(planTemplate("t", "p.md", "s"), line);
		expect(plan.endsWith(`## 검증 로그\n${line}\n`)).toBe(true);
		expect(section(appendVerificationLog(plan, "- second"), "검증 로그")).toBe(`${line}\n- second`);
	});

	it("ticks a checklist line", () => {
		const text = "## 검증 계획\n- [ ] `npm test`\n";
		expect(checkLine(text, 1)).toBe("## 검증 계획\n- [x] `npm test`\n");
		expect(checkLine(text, 0)).toBe(text);
	});
});

describe("verification log on CRLF files", () => {
	const line = (n: number) => `- 2026-09-28 10:0${n}:00 KST - \`npm test\` → OK (1.0s)`;
	const crlf = (text: string) => text.replace(/\n/g, "\r\n");
	const sectionCount = (text: string) => normalizeEol(text).split("\n").filter((l) => l === "## 검증 로그").length;

	it("appends into the single existing 검증 로그 section of a CRLF goal file", () => {
		let text = crlf(goalTemplate("2026-09-28 10:00:00 KST"));
		for (const n of [1, 2, 3]) text = appendVerificationLog(text, line(n));
		expect(sectionCount(text)).toBe(1);
		expect(sectionLines(text, "검증 로그")).toEqual([line(1), line(2), line(3)]);
	});

	it("creates exactly one 검증 로그 section in a CRLF plan file across repeated appends", () => {
		let text = crlf(planTemplate("t", "p.md", "2026-09-28 10:00:00 KST"));
		for (const n of [1, 2, 3]) text = appendVerificationLog(text, line(n));
		expect(sectionCount(text)).toBe(1);
		expect(sectionLines(text, "검증 로그")).toEqual([line(1), line(2), line(3)]);
	});

	it("keeps the CRLF line endings of the input", () => {
		const goal = appendVerificationLog(crlf(goalTemplate("s")), line(1));
		expect(goal.split("\n").every((l, i, all) => i === all.length - 1 || l.endsWith("\r"))).toBe(true);
		const plan = appendVerificationLog(crlf(planTemplate("t", "p.md", "s")), line(1));
		expect(plan.split("\n").every((l, i, all) => i === all.length - 1 || l.endsWith("\r"))).toBe(true);
		const ticked = checkLine(crlf("## 검증 계획\n- [ ] `npm test`\n"), 1);
		expect(ticked).toBe(crlf("## 검증 계획\n- [x] `npm test`\n"));
	});

	it("keeps indented and blank lines already in the 검증 로그 section", () => {
		const base = writeSection(goalTemplate("s"), "검증 로그", [line(1), "  출력: 78 tests passed", "", "참고: 재실행 필요"]);
		const next = appendVerificationLog(base, line(2));
		expect(sectionLines(next, "검증 로그")).toEqual([line(1), "  출력: 78 tests passed", "", "참고: 재실행 필요", line(2)]);
	});
});

describe("completion gates", () => {
	it("lists open plan items and goal criteria", () => {
		expect(openPlanItems(planTemplate("t", "p.md", "s"))).toHaveLength(5);
		expect(openPlanItems("# 계획: x\n\n## Now\n- [x] done\n\n## 단계\n- [x] a\n")).toEqual([]);
		expect(openCriteria(goalTemplate("s"))).toHaveLength(3);
	});
});

describe("suggestTestCommands", () => {
	const info: ProjectInfo = { packageScripts: { test: "vitest run", check: "npm run typecheck" }, devDependencies: ["vitest"], hasGoMod: true, hasPytest: true, hasCargo: false, existingFiles: new Set(["tests/unit/markdown.test.ts", "pkg/test_util.py"]) };

	it("maps source files to their tests and adds project-level runners", () => {
		const s = suggestTestCommands(["src/util/markdown.ts", "pkg/util.py", "internal/proxy/server.go", "tests/unit/semver.test.ts"], info);
		expect(s.map((x) => x.command)).toEqual(["npx vitest run tests/unit/markdown.test.ts", "pytest pkg/test_util.py", "go test ./internal/proxy/...", "npx vitest run tests/unit/semver.test.ts", "npm run check", "go test ./..."]);
	});

	it("returns nothing without changes", () => {
		expect(suggestTestCommands([], info)).toEqual([]);
	});
});
