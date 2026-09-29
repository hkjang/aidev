import { describe, expect, it } from "vitest";
import {
	countChecks,
	detectEol,
	hasSection,
	headingTitle,
	lines,
	matchLine,
	moveTaskToSection,
	normalizeEol,
	priorityRank,
	section,
	sectionLines,
	setLine,
	subsection,
	taskLines,
	toggleCheckbox,
	touchPlan,
	writeSection,
} from "../../src/util/markdown";
import { openPlanItems, planMeta, planTemplate } from "../../src/features/plans";
import { goalTemplate, parseGoal } from "../../src/features/goals";
import { lintGoal, lintPlan } from "../../src/features/goal-lint";

const STAMP = "2026-09-22 10:00:00 KST";

describe("section parsing", () => {
	const plan = planTemplate("테스트 목표", "current-plan.md", STAMP);

	it("reads whole multi-line sections, not just the first line", () => {
		expect(taskLines(section(plan, "단계"))).toHaveLength(3);
		expect(section(plan, "Now")).toBe("- [ ] 바로 구현할 작업 1개 정의");
		expect(section(plan, "Done")).toBe("");
		expect(lines(section(plan, "Risks"))).toEqual(["- 없음"]);
		expect(section(plan, "검증 계획")).toBe("- [ ] 관련 스크립트 또는 테스트 실행");
	});

	it("does not confuse ### sub-sections with ## sections", () => {
		const goal = goalTemplate(STAMP);
		expect(section(goal, "Now")).toBe("");
		expect(subsection(goal, "Now")).toBe("- [ ] 목표 설명 작성");
		expect(subsection(goal, "Next")).toBe("- [ ] 첫 번째 작은 작업 선택");
		expect(subsection(goal, "Done")).toBe("");
	});

	it("returns empty strings for unknown sections", () => {
		expect(section(plan, "없는 섹션")).toBe("");
		expect(subsection(plan, "없는 섹션")).toBe("");
	});
});

describe("writeSection", () => {
	const plan = planTemplate("테스트 목표", "current-plan.md", STAMP);

	it("replaces a section body and keeps the rest intact", () => {
		const next = writeSection(plan, "Now", ["- [ ] A", "- [ ] B"]);
		expect(section(next, "Now")).toBe("- [ ] A\n- [ ] B");
		expect(section(next, "Next")).toBe(section(plan, "Next"));
		expect(headingTitle(next, "계획", "")).toBe("테스트 목표");
	});

	it("fills an empty section and can empty a filled one", () => {
		const filled = writeSection(plan, "Done", ["- [x] 완료"]);
		expect(section(filled, "Done")).toBe("- [x] 완료");
		expect(section(writeSection(filled, "Done", []), "Done")).toBe("");
	});

	it("leaves text unchanged when the section is missing", () => {
		expect(writeSection(plan, "없음", ["x"])).toBe(plan);
	});

	it("supports the advance flow end to end", () => {
		const now = taskLines(section(plan, "Now"));
		const next = taskLines(section(plan, "Next"));
		const done = lines(section(plan, "Done"));
		done.push("- [x] " + now.shift()!.replace(/^- \[ \] /, ""));
		now.push(next.shift()!);
		let text = writeSection(plan, "Now", now);
		text = writeSection(text, "Next", next);
		text = writeSection(text, "Done", done);
		expect(section(text, "Now")).toBe("- [ ] 뒤이어 실행할 작업 1개 정의");
		expect(section(text, "Next")).toBe("");
		expect(section(text, "Done")).toBe("- [x] 바로 구현할 작업 1개 정의");
	});
});

describe("sectionLines", () => {
	const plan = planTemplate("테스트 목표", "current-plan.md", STAMP);

	it("원본 줄을 들여쓰기까지 그대로 돌려주고 빈 섹션은 빈 배열이다", () => {
		const filled = writeSection(plan, "Now", ["- [ ] 상위", "  - [ ] 하위", "  참고: 들여쓴 메모"]);
		expect(sectionLines(filled, "Now")).toEqual(["- [ ] 상위", "  - [ ] 하위", "  참고: 들여쓴 메모"]);
		expect(sectionLines(plan, "Done")).toEqual([]);
		expect(sectionLines(plan, "없는 섹션")).toEqual([]);
	});
});

describe("moveTaskToSection", () => {
	const plan = planTemplate("테스트 목표", "current-plan.md", STAMP);
	const complete = (line: string) => line.replace(/^- \[ \]/, "- [x]");

	it("대상 섹션에 이미 있던 들여쓴 줄과 메모 줄을 보존한다", () => {
		const withDone = writeSection(plan, "Done", ["- [x] 지난 작업", "  - [x] 하위 완료", "  참고: 들여쓴 메모"]);
		const nowIndex = withDone.split("\n").indexOf("- [ ] 바로 구현할 작업 1개 정의");

		const moved = moveTaskToSection(withDone, nowIndex, "Done", complete);

		expect(moved).not.toBeNull();
		expect(section(moved!, "Done")).toBe(["- [x] 지난 작업", "  - [x] 하위 완료", "  참고: 들여쓴 메모", "- [x] 바로 구현할 작업 1개 정의"].join("\n"));
	});
});

describe("metadata lines", () => {
	const plan = planTemplate("테스트 목표", "current-plan.md", STAMP);

	it("reads and rewrites label lines", () => {
		expect(matchLine(plan, "상태")).toBe("draft");
		expect(matchLine(plan, "우선순위", "P2")).toBe("P2");
		const next = setLine(plan, "우선순위", "P0");
		expect(matchLine(next, "우선순위")).toBe("P0");
		expect(matchLine(setLine(next, "우선순위", "P1"), "우선순위")).toBe("P1");
		expect(next.indexOf("우선순위: P0")).toBeGreaterThan(next.indexOf("마지막 갱신:"));
	});

	it("touches status and timestamp", () => {
		const next = touchPlan(plan, "blocked", "2026-09-23 09:00:00 KST");
		expect(matchLine(next, "상태")).toBe("blocked");
		expect(matchLine(next, "마지막 갱신")).toBe("2026-09-23 09:00:00 KST");
		expect(matchLine(touchPlan(plan, null, "x"), "상태")).toBe("draft");
	});

	it("inserts a status line after the title when missing", () => {
		const text = "# 계획: x\n\n작성일: a\n\n## Now\n";
		const next = touchPlan(text, "active", "b");
		expect(next.startsWith("# 계획: x\n\n상태: active\n")).toBe(true);
		expect(matchLine(next, "마지막 갱신")).toBe("b");
	});

	it("escapes regex characters in labels", () => {
		expect(matchLine("a.b: 1\naxb: 2", "a.b")).toBe("1");
	});
});

describe("goal parsing", () => {
	it("summarizes the goal template", () => {
		const goal = parseGoal(goalTemplate(STAMP));
		expect(goal.title).toBe("새 목표");
		expect(goal.status).toBe("draft");
		expect(goal.phase).toBe("discovery");
		expect(goal.criteria).toHaveLength(3);
		expect(goal.now).toEqual(["- [ ] 목표 설명 작성"]);
		expect(goal.nextItems).toEqual(["- [ ] 첫 번째 작은 작업 선택"]);
		expect(goal.verification).toEqual([]);
		expect(countChecks(goalTemplate(STAMP))).toEqual({ done: 0, total: 5 });
	});
});

describe("CRLF 줄바꿈", () => {
	// Windows 체크아웃(core.autocrlf)이나 zip 왕복으로 `.vibe-code/*.md` 가 CRLF 가 되는 경로.
	const crlf = (text: string) => text.replace(/\n/g, "\r\n");
	const planLf = planTemplate("테스트 목표", "current-plan.md", STAMP);
	const planCrlf = crlf(planLf);
	const goalLf = goalTemplate(STAMP);
	const goalCrlf = crlf(goalLf);
	const nowTask = "- [ ] 바로 구현할 작업 1개 정의";
	const nowIndex = planLf.split("\n").indexOf(nowTask);
	const complete = (line: string) => line.replace(/^- \[ \]/, "- [x]");

	it("줄바꿈 방식을 첫 줄바꿈 기준으로 판정한다", () => {
		expect(detectEol(planLf)).toBe("\n");
		expect(detectEol(planCrlf)).toBe("\r\n");
		expect(detectEol("줄바꿈 없음")).toBe("\n");
		expect(detectEol("a\r\nb\nc")).toBe("\r\n");
		expect(detectEol("a\nb\r\nc")).toBe("\n");
		expect(normalizeEol(planCrlf)).toBe(planLf);
		expect(normalizeEol(planLf)).toBe(planLf);
	});

	it("읽기 함수가 CRLF 에서 LF 와 같은 값을 돌려준다", () => {
		for (const name of ["목표 정렬", "단계", "Now", "Next", "Done", "Risks", "검증 계획"]) {
			expect(section(planCrlf, name)).toBe(section(planLf, name));
			expect(section(planCrlf, name)).not.toMatch(/\r/);
			expect(hasSection(planCrlf, name)).toBe(true);
		}
		expect(section(planCrlf, "Now")).toBe(nowTask);
		expect(taskLines(section(planCrlf, "단계"))).toHaveLength(3);
		expect(hasSection(planCrlf, "없는 섹션")).toBe(false);
		for (const name of ["Now", "Next", "Done"]) expect(subsection(goalCrlf, name)).toBe(subsection(goalLf, name));
		expect(subsection(goalCrlf, "Now")).toBe("- [ ] 목표 설명 작성");
		expect(headingTitle(planCrlf, "계획", "")).toBe("테스트 목표");
		expect(matchLine(planCrlf, "상태")).toBe("draft");
		expect(countChecks(planCrlf)).toEqual(countChecks(planLf));
	});

	it("writeSection 이 CRLF 에서 같은 편집 결과를 내고 줄바꿈을 보존한다", () => {
		const next = writeSection(planCrlf, "Now", ["- [ ] A", "- [ ] B"]);
		expect(section(next, "Now")).toBe("- [ ] A\n- [ ] B");
		expect(next).toBe(crlf(writeSection(planLf, "Now", ["- [ ] A", "- [ ] B"])));
		expect(next).not.toMatch(/[^\r]\n/);
		expect(next).not.toMatch(/\r\r/);
	});

	it("moveTaskToSection 이 CRLF 에서 null 대신 옮긴 텍스트를 돌려준다", () => {
		const moved = moveTaskToSection(planCrlf, nowIndex, "Done", complete);
		expect(moved).not.toBeNull();
		expect(section(moved!, "Done")).toBe("- [x] 바로 구현할 작업 1개 정의");
		expect(section(moved!, "Now")).toBe("");
		expect(moved).toBe(crlf(moveTaskToSection(planLf, nowIndex, "Done", complete)!));
	});

	it("toggleCheckbox·touchPlan·setLine 이 입력의 줄바꿈을 보존한다", () => {
		expect(toggleCheckbox(planCrlf, nowIndex)).toBe(crlf(toggleCheckbox(planLf, nowIndex)));
		expect(touchPlan(planCrlf, "active", STAMP)).toBe(crlf(touchPlan(planLf, "active", STAMP)));
		expect(setLine(planCrlf, "우선순위", "P0")).toBe(crlf(setLine(planLf, "우선순위", "P0")));
		expect(touchPlan(planCrlf, "active", STAMP)).not.toMatch(/[^\r]\n/);
		expect(setLine(planCrlf, "우선순위", "P0")).not.toMatch(/[^\r]\n/);
	});

	it("계획·목표 소비자(parseGoal·openPlanItems)도 CRLF 에서 같은 값을 읽는다", () => {
		expect(parseGoal(goalCrlf)).toEqual(parseGoal(goalLf));
		expect(openPlanItems(planCrlf)).toEqual(openPlanItems(planLf));
		expect(openPlanItems(planCrlf)).toHaveLength(5);
	});

	it("린트가 CRLF 계획·목표에 섹션 누락 경고를 내지 않는다", () => {
		expect(lintPlan(planCrlf)).toEqual(lintPlan(planLf));
		expect(lintGoal(goalCrlf)).toEqual(lintGoal(goalLf));
		const missing = (issues: { message: string }[]) => issues.map((i) => i.message).filter((m) => m.includes("섹션이 없습니다"));
		expect(missing(lintPlan(planCrlf))).toEqual([]);
		expect(missing(lintGoal(goalCrlf))).toEqual([]);
	});
});

describe("priorityRank", () => {
	it("orders P0..P3 and pushes unknowns last", () => {
		expect(["P3", "p1", "P0", "zzz"].sort((a, b) => priorityRank(a) - priorityRank(b))).toEqual(["P0", "p1", "P3", "zzz"]);
		expect(priorityRank(undefined)).toBe(2);
		expect(priorityRank("")).toBe(2);
	});
});


describe.each(["\n", "\r\n"])("빈 메타데이터 줄 경계 (%j)", (eol) => {
	const encode = (text: string) => text.replace(/\n/g, eol);
	for (const blank of ["", " ", "\t", " \t "]) {
		it(`빈 값 ${JSON.stringify(blank)}은 다음 줄이나 EOF 대신 fallback을 반환한다`, () => {
			for (const suffix of ["", "\n작성일: original", "\n\n본문", "\n## Now", "\ndone"]) {
				expect(matchLine(encode(`상태:${blank}${suffix}`), "상태", "fallback")).toBe("fallback");
			}
			expect(matchLine("작성일: original", "상태", "fallback")).toBe("fallback");
			expect(matchLine(encode("상태: \t active \t\n작성일: original"), "상태")).toBe("active");
		});

		for (const kind of ["plan", "goal"] as const) {
			const template = kind === "plan" ? planTemplate("테스트 목표", "current-plan.md", STAMP) : goalTemplate(STAMP);
			const readStatus = (text: string) => kind === "plan" ? planMeta(text, "current-plan.md").status : parseGoal(text).status;
			const lint = kind === "plan" ? lintPlan : lintGoal;
			const input = encode(template.replace("상태: draft", `상태:${blank}`).replace(`마지막 갱신: ${STAMP}`, `마지막 갱신:${blank}`));
			it(`${kind}: 빈 상태 ${JSON.stringify(blank)}를 소비자별 기본값으로 읽는다`, () => {
				expect(readStatus(input)).toBe(kind === "plan" ? "draft" : "(unknown)");
				expect(matchLine(input, "상태")).toBe("");
				expect(lint(input).filter((issue) => issue.message.includes("상태"))).toEqual([
					{ line: 0, message: "`상태:` 줄이 없습니다 (draft | active | blocked | done).", severity: "warning" },
				]);
			});
			it(`${kind}: setLine은 빈 상태 ${JSON.stringify(blank)}만 제자리 교체한다`, () => {
				const edited = setLine(input, "상태", "active");
				expect(edited).toBe(input.replace(`상태:${blank}${eol}`, `상태: active${eol}`));
				expect(readStatus(edited)).toBe("active");
				expect(lint(edited)).toEqual(lint(template));
			});
			it(`${kind}: touchPlan은 빈 상태·갱신 ${JSON.stringify(blank)}만 변경한다`, () => {
				const edited = touchPlan(input, "active", "new");
				expect(edited).toBe(input.replace(`상태:${blank}${eol}`, `상태: active${eol}`).replace(`마지막 갱신:${blank}${eol}`, `마지막 갱신: new${eol}`));
				expect(readStatus(edited)).toBe("active");
				expect(lint(edited)).toEqual(lint(template));
				expect(touchPlan(input, null, "new")).toBe(input.replace(`마지막 갱신:${blank}${eol}`, `마지막 갱신: new${eol}`));
			});
		}
	}

	it("setLine은 값이 빈 앵커 바로 뒤에 삽입하고 마지막 갱신을 우선한다", () => {
		for (const anchor of ["작성일", "마지막 갱신"]) {
			for (const blank of ["", " \t"]) {
				for (const suffix of ["", "\n\n## Now\n본문"]) {
					const prefix = anchor === "마지막 갱신" ? "# 계획: 테스트\n작성일: original\n" : "# 계획: 테스트\n";
					const input = encode(`${prefix}${anchor}:${blank}${suffix}`);
					expect(setLine(input, "우선순위", "P0")).toBe(encode(`${prefix}${anchor}:${blank}\n우선순위: P0${suffix}`));
				}
			}
		}
		expect(setLine("본문", "상태", "active")).toBe("본문");
	});

	it("파일 끝의 빈 라벨도 제자리 교체한다", () => {
		expect(setLine(encode("작성일: original\na.b:\t"), "a.b", "value")).toBe(encode("작성일: original\na.b: value"));
		expect(touchPlan(encode("작성일: original\n상태:"), "active", "new")).toBe(encode("작성일: original\n마지막 갱신: new\n상태: active"));
		expect(touchPlan(encode("상태: active\n마지막 갱신:"), null, "new")).toBe(encode("상태: active\n마지막 갱신: new"));
	});

	it("touchPlan은 빈 작성일 바로 뒤에 누락 갱신을 삽입한다", () => {
		for (const suffix of ["", "\n\n## Now\n본문"]) {
			const input = encode(`상태:\n작성일: \t${suffix}`);
			expect(touchPlan(input, null, "new")).toBe(encode(`상태:\n작성일: \t\n마지막 갱신: new${suffix}`));
		}
	});
});
