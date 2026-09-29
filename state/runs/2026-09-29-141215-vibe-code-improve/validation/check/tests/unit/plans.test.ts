import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import { advancePlanText, archiveDonePlans, conflictStamp, openPlanItems, planMeta, planTemplate, selectPlanName } from "../../src/features/plans";
import type { WorkspacePaths } from "../../src/features/workspace";
import { matchLine, section, writeSection } from "../../src/util/markdown";

const temps: string[] = [];

function tempDir(): string {
	const dir = fs.mkdtempSync(path.join(os.tmpdir(), "vibe-plans-test-"));
	temps.push(dir);
	return fs.realpathSync(dir);
}

/** Real `.vibe-code/plans` + `plans/archive` directories — archiving is a filesystem contract, so nothing here is faked. */
function planDirs(): { plans: string; archive: string } {
	const plans = path.join(tempDir(), ".vibe-code", "plans");
	const archive = path.join(plans, "archive");
	fs.mkdirSync(archive, { recursive: true });
	return { plans, archive };
}

function plan(dir: string, name: string, status: string, body = ""): string {
	const file = path.join(dir, name);
	fs.writeFileSync(file, `# 계획: ${name}\n\n상태: ${status}\n${body}`, "utf8");
	return file;
}

function names(dir: string): string[] {
	return fs.readdirSync(dir).sort();
}

afterEach(() => {
	while (temps.length > 0) fs.rmSync(temps.pop() as string, { recursive: true, force: true });
});

describe("archiveDonePlans", () => {
	it("moves only done plans and leaves the rest of plans/ alone", () => {
		const { plans, archive } = planDirs();
		plan(plans, "done-plan.md", "done");
		plan(plans, "active-plan.md", "active");
		fs.writeFileSync(path.join(plans, ".active-plan"), "active-plan.md", "utf8");

		const archived = archiveDonePlans(plans, archive, "20260926070059");

		expect(archived).toEqual([{ name: "done-plan.md", dest: "done-plan.md" }]);
		expect(names(plans)).toEqual([".active-plan", "active-plan.md", "archive"]);
		expect(names(archive)).toEqual(["done-plan.md"]);
	});

	it("keeps the existing archived copy when a done plan reuses its name", () => {
		const { plans, archive } = planDirs();
		plan(archive, "current-plan.md", "done", "## 단계\n- [x] 지난 회차 작업\n");
		plan(plans, "current-plan.md", "done", "## 단계\n- [x] 이번 회차 작업\n");

		const archived = archiveDonePlans(plans, archive, "20260926070059");

		expect(fs.readFileSync(path.join(archive, "current-plan.md"), "utf8")).toContain("지난 회차 작업");
		expect(names(archive)).toEqual(["current-plan-20260926070059.md", "current-plan.md"]);
		expect(fs.readFileSync(path.join(archive, "current-plan-20260926070059.md"), "utf8")).toContain("이번 회차 작업");
		expect(archived).toEqual([{ name: "current-plan.md", dest: "current-plan-20260926070059.md" }]);
	});

	it("keeps both archived copies when the same name is archived twice within one stamp", () => {
		const { plans, archive } = planDirs();
		plan(archive, "current-plan.md", "done", "첫 번째\n");
		plan(archive, "current-plan-20260926070059.md", "done", "두 번째\n");
		plan(plans, "current-plan.md", "done", "세 번째\n");

		const archived = archiveDonePlans(plans, archive, "20260926070059");

		expect(archived[0].dest).toBe("current-plan-20260926070059-2.md");
		expect(names(archive)).toEqual(["current-plan-20260926070059-2.md", "current-plan-20260926070059.md", "current-plan.md"]);
		expect(fs.readFileSync(path.join(archive, "current-plan.md"), "utf8")).toContain("첫 번째");
		expect(fs.readFileSync(path.join(archive, "current-plan-20260926070059.md"), "utf8")).toContain("두 번째");
	});

	it("does not descend into the archive directory or touch non-markdown files", () => {
		const { plans, archive } = planDirs();
		plan(archive, "old.md", "done");
		fs.writeFileSync(path.join(plans, "notes.txt"), "상태: done\n", "utf8");

		expect(archiveDonePlans(plans, archive, "20260926070059")).toEqual([]);
		expect(names(archive)).toEqual(["old.md"]);
		expect(names(plans)).toEqual(["archive", "notes.txt"]);
	});

	it("creates the archive directory when it does not exist yet", () => {
		const plans = path.join(tempDir(), ".vibe-code", "plans");
		fs.mkdirSync(plans, { recursive: true });
		const archive = path.join(plans, "archive");
		plan(plans, "done-plan.md", "done");

		expect(archiveDonePlans(plans, archive, "20260926070059")).toEqual([{ name: "done-plan.md", dest: "done-plan.md" }]);
		expect(names(archive)).toEqual(["done-plan.md"]);
	});
});

describe("conflictStamp", () => {
	it("is 14 digits of Asia/Seoul time — the one format archive and restore both suffix with", () => {
		// 2026-09-25 22:01:02 UTC is 2026-09-26 07:01:02 KST.
		expect(conflictStamp(new Date("2026-09-25T22:01:02Z"))).toBe("20260926070102");
	});
});

describe("selectPlanName", () => {
	function paths(plans: string): WorkspacePaths {
		const base = path.dirname(plans);
		return {
			ws: path.dirname(base),
			base,
			goals: path.join(base, "goals"),
			sessions: path.join(base, "sessions"),
			checkpoints: path.join(base, "checkpoints"),
			plans,
			archive: path.join(plans, "archive"),
			journal: path.join(base, "journal"),
			audit: path.join(base, "audit"),
			current: path.join(base, "goals", "current.md"),
		};
	}

	it("follows the .active-plan pointer when it names an existing file", () => {
		const { plans } = planDirs();
		plan(plans, "a-plan.md", "active");
		plan(plans, "z-plan.md", "active");
		fs.writeFileSync(path.join(plans, ".active-plan"), "a-plan.md\n", "utf8");

		expect(selectPlanName(paths(plans))).toBe("a-plan.md");
	});

	it("falls back to the last file when the pointer names a file that is gone", () => {
		const { plans } = planDirs();
		plan(plans, "a-plan.md", "active");
		plan(plans, "z-plan.md", "active");
		fs.writeFileSync(path.join(plans, ".active-plan"), "archived-away.md", "utf8");

		expect(selectPlanName(paths(plans))).toBe("z-plan.md");
	});

	it("falls back to the default name for an empty plans directory", () => {
		const { plans } = planDirs();

		expect(selectPlanName(paths(plans))).toBe("current-plan.md");
	});
});

describe("openPlanItems", () => {
	it("lists unchecked items of 단계/Now/검증 계획 only", () => {
		const text = [
			"# 계획: 예시",
			"",
			"## 단계",
			"- [x] 끝난 단계",
			"- [ ] 남은 단계",
			"",
			"## Now",
			"- [ ] 지금 할 일",
			"",
			"## Next",
			"- [ ] 다음 할 일",
			"",
			"## 검증 계획",
			"- [ ] 테스트 실행",
			"",
		].join("\n");

		expect(openPlanItems(text)).toEqual(["단계: 남은 단계", "Now: 지금 할 일", "검증 계획: 테스트 실행"]);
	});
});

describe("advancePlanText", () => {
	// 프로덕션 템플릿 그대로 — 대역 없이 실제 계획 문자열을 그대로 넣는다.
	const STAMP = "2026-09-22 10:00:00 KST";
	const NEXT_STAMP = "2026-09-28 11:22:33 KST";
	const template = planTemplate("테스트 목표", "current-plan.md", STAMP);
	const nowTask = "- [ ] 바로 구현할 작업 1개 정의";
	const nextTask = "- [ ] 뒤이어 실행할 작업 1개 정의";
	/** 사용자가 손으로 Now/Next 에 메모와 들여쓴 하위 항목을 적어 둔 계획. */
	const annotated = writeSection(writeSection(template, "Now", ["참고: Now 메모", nowTask, "  - [ ] 하위 항목", "  참고: 들여쓴 메모"]), "Next", [
		"- 전제: Next 메모",
		nextTask,
		"  - [ ] 다음 하위",
	]);

	it("Now/Next 의 체크리스트가 아닌 줄과 들여쓴 줄을 그대로 남긴다", () => {
		const result = advancePlanText(annotated, NEXT_STAMP);

		expect(result).not.toBeNull();
		expect(section(result!.text, "Now")).toBe(["참고: Now 메모", "  - [ ] 하위 항목", "  참고: 들여쓴 메모", nextTask].join("\n"));
		expect(section(result!.text, "Next")).toBe(["- 전제: Next 메모", "  - [ ] 다음 하위"].join("\n"));
	});

	it("첫 Now 항목을 Done 끝에 붙이고 Next 첫 항목을 승격하며 상태·갱신 시각을 올린다", () => {
		const result = advancePlanText(template, NEXT_STAMP);

		expect(result).not.toBeNull();
		expect(result!.completed).toBe("바로 구현할 작업 1개 정의");
		expect(result!.promoted).toBe("뒤이어 실행할 작업 1개 정의");
		expect(section(result!.text, "Done")).toBe("- [x] 바로 구현할 작업 1개 정의");
		expect(section(result!.text, "Now")).toBe(nextTask);
		expect(section(result!.text, "Next")).toBe("");
		expect(matchLine(result!.text, "상태")).toBe("active");
		expect(matchLine(result!.text, "마지막 갱신")).toBe(NEXT_STAMP);
	});

	it("이미 있던 Done 기록의 들여쓰기를 유지한 채 뒤에 붙인다", () => {
		const withDone = writeSection(annotated, "Done", ["- [x] 지난 작업", "  참고: 들여쓴 완료 메모"]);

		const result = advancePlanText(withDone, NEXT_STAMP);

		expect(section(result!.text, "Done")).toBe(["- [x] 지난 작업", "  참고: 들여쓴 완료 메모", "- [x] 바로 구현할 작업 1개 정의"].join("\n"));
	});

	it("Now 에 체크리스트 줄이 없으면 null 을 돌려준다", () => {
		expect(advancePlanText(writeSection(template, "Now", ["참고: 메모만 있다"]), NEXT_STAMP)).toBeNull();
		expect(advancePlanText(writeSection(template, "Now", []), NEXT_STAMP)).toBeNull();
	});

	it("Next 가 비어 있으면 승격 없이 Now 만 비운다", () => {
		const result = advancePlanText(writeSection(template, "Next", []), NEXT_STAMP);

		expect(result!.promoted).toBe("");
		expect(section(result!.text, "Now")).toBe("");
		expect(section(result!.text, "Next")).toBe("");
	});

	it("CRLF 계획에서도 같은 결과를 내고 줄바꿈을 보존한다", () => {
		const crlf = (text: string) => text.replace(/\n/g, "\r\n");
		const result = advancePlanText(crlf(annotated), NEXT_STAMP);

		expect(result!.text).toBe(crlf(advancePlanText(annotated, NEXT_STAMP)!.text));
		expect(result!.text).not.toMatch(/[^\r]\n/);
		expect(result!.text).not.toMatch(/\r\r/);
	});
});


describe.each(["\n", "\r\n"])("완료 보관의 상태 줄 경계 (%j)", (eol) => {
	it("빈 상태 다음 done은 원본 바이트를 보존하고 같은 줄의 done만 보관한다", () => {
		const { plans, archive } = planDirs();
		const template = planTemplate("보관 경계", "current-plan.md", "2026-09-29 12:00:00 KST");
		const inputs = ["", " \t", "\ndone", " \t\ndone", " done", " \tdone \t", " DONE"].map((value, index) => ({
			name: `plan-${index}.md`,
			text: template.replace("상태: draft", `상태:${value}`).replace(/\n/g, eol),
			done: value.trim() === "done" && !value.includes("\n"),
		}));
		for (const input of inputs) fs.writeFileSync(path.join(plans, input.name), input.text, "utf8");

		const archived = archiveDonePlans(plans, archive, "20260929120000");

		expect(archived.map((row) => row.name).sort()).toEqual(inputs.filter((input) => input.done).map((input) => input.name));
		for (const input of inputs) {
			expect(planMeta(input.text, input.name).status === "done").toBe(input.done);
			const destination = path.join(input.done ? archive : plans, input.name);
			expect(fs.readFileSync(destination)).toEqual(Buffer.from(input.text, "utf8"));
			expect(fs.existsSync(path.join(input.done ? plans : archive, input.name))).toBe(false);
		}
	});
});
