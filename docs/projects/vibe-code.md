---
title: "vibe-code — 자율 개선 이력"
description: "vibe-code: 자율 개선 회차 1회, 릴리즈 1건. 최근 릴리즈 v1.4.1 (자산 0개)."
last_modified_at: 2026-09-25 22:14:58 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "SoftwareSourceCode",
 "name": "vibe-code",
 "codeRepository": "https://github.com/hkjang/vibe-code",
 "url": "https://hkjang.github.io/aidev/projects/vibe-code/",
 "description": "vibe-code: 자율 개선 회차 1회, 릴리즈 1건. 최근 릴리즈 v1.4.1 (자산 0개).",
 "inLanguage": "ko",
 "maintainer": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-25T22:14:58+09:00",
 "version": "1.4.1"
}
</script>

# vibe-code

<p class="tldr"><strong>요약.</strong> vibe-code: 자율 개선 회차 1회, 릴리즈 1건. 최근 릴리즈 v1.4.1 (자산 0개). <span class="pill pill-merged" title="14일: 릴리즈 1, 실패 0, 경고 1, 회귀 0">건강 B</span> <span class="meta">14일: 릴리즈 1, 실패 0, 경고 1, 회귀 0</span></p>

<ul class="stats"><li><b>1</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>0</b><span>배포 준비 완료</span></li><li><b>1</b><span>릴리즈 진행 중</span></li><li><b>0</b><span>병합 완료</span></li><li><b>0</b><span>검토 대기</span></li><li><b>0</b><span>검증 실패</span></li><li><b>0</b><span>변경 없음</span></li><li><b>0</b><span>실행 오류</span></li><li><b>$7.16</b><span>비용</span></li><li><b>17분</b><span>에이전트 시간</span></li></ul>

## 현황

<dl class="kv">
<dt>저장소</dt><dd><a href="https://github.com/hkjang/vibe-code">https://github.com/hkjang/vibe-code</a></dd>
<dt>마지막 회차</dt><dd>2026-09-23 13:11 KST — <span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/vibe-code/pull/1">PR #1</a>, released <a href="https://github.com/hkjang/vibe-code/releases/tag/v1.4.1">v1.4.1</a>, <strong>ASSETS MISSING</strong></dd>
<dt>최근 릴리즈</dt><dd><a href="https://github.com/hkjang/vibe-code/releases/tag/v1.4.1">v1.4.1</a> — released · 자산 0개 (이전 v1.4.0: 2개) <span class="pill pill-failed">❌ 자산 누락</span> <a href="https://github.com/hkjang/vibe-code/releases">전체 릴리즈 →</a></dd>
<dt>사유</dt><dd>Commit and annotated tag created locally; nothing pushed. VSIX assets (vibe-code-1.4.1.vsix + .sha256) could NOT be built on this machine and must be produced on Windows before publishing: scripts/package-vsix.ps1 is PowerShell-only (no pwsh/powershell here, and it stages through C:\tmp), and it needs the dist/ runtime assets (node_modules, i18n, workers, *.wasm) which are gitignored and only obtainable from an already-published release VSIX via scripts/restore-dist-assets.mjs — no VSIX exists anywhere on this machine and gh is unavailable to download the 1.4.0 one. Building from the stale release/vibe-code-1.0.0.7z via a hand-written reimplementation was rejected as it would ship an unverifiable artifact (scripts/verify-package.ps1 also cannot run here). Release gate npm run check passed at 1.4.1: typecheck + 53 vitest tests + esbuild build. .github/workflows/ci.yml does not trigger on tags and never creates a GitHub Release, so the release and its assets are published by hand.</dd>
</dl>

## 회차 이력

<div class="table-wrap"><table class="rt"><thead><tr><th>일시</th><th class="primary">프로젝트</th><th>결과</th></tr></thead><tbody><tr data-status="released"><td data-label="일시">2026-09-23 13:11</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/vibe-code/">vibe-code</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=releasing">릴리즈 진행 중</span> merged <a href="https://github.com/hkjang/vibe-code/pull/1">PR #1</a>, released <a href="https://github.com/hkjang/vibe-code/releases/tag/v1.4.1">v1.4.1</a>, <strong>ASSETS MISSING</strong><div class="meta">5파일 <span style="color:var(--good)">+220</span>/<span style="color:var(--bad)">−15</span> · 테스트 1 — fix: checkpoint snapshots untracked files, not just tracked changes</div></td></tr></tbody></table></div>

## 비용·사용량

<div class="table-wrap"><table class="rt"><caption class="meta">최근 30세션</caption><thead><tr><th>시각</th><th class="primary">프로젝트</th><th>단계</th><th class="num">시간</th><th class="num">턴</th><th class="num">비용</th><th class="num">토큰 입력/출력</th><th>종료</th></tr></thead><tbody><tr data-status="other"><td data-label="시각">12:54</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/vibe-code/">vibe-code</a></td><td data-label="단계">릴리즈</td><td data-label="시간" class="num">6분</td><td data-label="턴" class="num">41</td><td data-label="비용" class="num">$1.82</td><td data-label="토큰 입력/출력" class="num">1.6M / 16K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">12:45</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/vibe-code/">vibe-code</a></td><td data-label="단계">비평</td><td data-label="시간" class="num">3분</td><td data-label="턴" class="num">21</td><td data-label="비용" class="num">$1.25</td><td data-label="토큰 입력/출력" class="num">735K / 15K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">12:42</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/vibe-code/">vibe-code</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">5분</td><td data-label="턴" class="num">38</td><td data-label="비용" class="num">$2.38</td><td data-label="토큰 입력/출력" class="num">2.2M / 22K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">12:38</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/vibe-code/">vibe-code</a></td><td data-label="단계">정찰</td><td data-label="시간" class="num">4분</td><td data-label="턴" class="num">36</td><td data-label="비용" class="num">$1.71</td><td data-label="토큰 입력/출력" class="num">1.3M / 18K</td><td data-label="종료">success</td></tr></tbody></table></div>

## 아이디어 백로그 — 대기 5 / 전체 7

<div class="table-wrap"><table class="rt"><caption class="meta">에이전트가 회차마다 재평가한다. 가치 높고 위험 낮은 대기 항목이 다음 회차 후보다.</caption><thead><tr><th class="primary">아이디어</th><th>가치/위험/크기</th><th>상태</th><th>메모</th><th>갱신</th></tr></thead><tbody><tr data-status="nochange"><td data-label="아이디어" class="primary">plans.ts(523줄, 테스트 0개) 순수 함수 테스트 보강</td><td data-label="가치/위험/크기">3/1/M</td><td data-label="상태">대기</td><td data-label="메모">selectPlanName(.active-plan 포인터가 없음/깨짐/존재하지 않는 파일 지목), planMeta, sortByPriority(P0~P3+미지정), linkedPlans, openPlanItems. 저장소에서 가장 큰 파일인데 단위 테스트가 하나도 없다. 이번 회차의 차선 후보였고 그대로 유효하다.</td><td data-label="갱신">2026-09-23</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">vibe-coders-proxy.ts(356줄) provider 저장/복원 경로 테스트</td><td data-label="가치/위험/크기">3/2/M</td><td data-label="상태">대기</td><td data-label="메모">사용자 provider 설정을 덮어쓰는 기능인데 테스트가 없다. 다만 vscode 설정 API 의존이 커서 스텁 확장이 필요하고, 대역 위주 테스트가 되기 쉬워(운영자 지침에 걸림) 우선순위를 낮춤.</td><td data-label="갱신">2026-09-23</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">computeGoalMetrics가 exitCode 없는 검증 항목을 통과로 집계</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">src/features/goal-metrics.ts:51 — num(undefined)=0 이라 details.exitCode 가 아예 없는 runVerification 항목이 통과로 계산되고 실패 목록에도 안 잡힌다. null 은 방어되어 있으나 undefined 는 아니다. 실제 감사 로그에 그런 줄이 나오는지 먼저 확인해야 하므로(미확인) 가치를 낮게 잡음.</td><td data-label="갱신">2026-09-23</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">createCheckpoint의 catch가 스냅샷 생성 후 실패를 &#x27;저장소 아님&#x27;으로 뭉갠다</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">update-ref 가 실패하면 commit 은 이미 할당됐는데 note 는 &#x27;git 저장소가 아니어서...&#x27; 가 된다(이번 교체 이전부터 있던 모양). 노트 본문은 `commit &amp;&amp; ref` 로 막혀 있어 사용자에게 틀린 복원 명령이 보이지는 않지만, writeAudit 의 details 에 commit 과 어긋난 note 가 함께 들어간다. catch 에서 commit/ref 를 되돌리거나 단계를 나누면 된다. 이번에는 증명된 재현이 없어 손대지 않았다.</td><td data-label="갱신">2026-09-23</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">거대한 untracked 디렉터리에서 동기 `git add -A` 지연 측정과 가드</td><td data-label="가치/위험/크기">2/2/M</td><td data-label="상태">대기</td><td data-label="메모">checkpointBeforeCommand 는 명령 승인 훅에서 동기로 돈다. 이번 변경으로 스냅샷이 untracked 파일까지 훑으므로, .gitignore 에 걸리지 않은 대형 디렉터리(빌드 산출물 등)가 있는 워크스페이스에서는 `git stash create` 때보다 느려질 수 있다. 실제 지연을 재 본 적이 없으므로(미확인) 먼저 측정하고, 필요하면 임계치나 타임아웃을 붙일 것.</td><td data-label="갱신">2026-09-23</td></tr><tr data-status="released"><td data-label="아이디어" class="primary">체크포인트가 untracked 파일을 담지 못함 — 임시 index 스냅샷으로 교체 + 첫 테스트</td><td data-label="가치/위험/크기">5/2/M</td><td data-label="상태">완료</td><td data-label="메모">createCheckpoint의 `git stash create`를 GIT_INDEX_FILE 임시 index(read-tree→add -A→write-tree→commit-tree)로 교체. 고치기 전 실제 git 2.43에서 untracked 누락을 재현했고, 복원 블록을 git diff/git checkout -- . 로 교체(스냅샷 커밋은 stash 형식이 아니라 `git stash apply`가 &#x27;is not a stash-like commit&#x27;으로 거부). 커밋 없는 저장소는 부모 없는 커밋으로 처리. tests/unit/checkpoints.test.ts 9개 신규(진짜 임시 git 저장소). npm run check 통과.</td><td data-label="갱신">2026-09-23</td></tr><tr data-status="released"><td data-label="아이디어" class="primary">체크포인트 설명 문구(package.nls.json / ko / 문서) 정정</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">완료</td><td data-label="메모">1번 과제의 수용 기준으로 흡수해 함께 처리. package.nls.json, package.nls.ko.json 의 settings.checkpointBeforeDestructive.description 과 docs/autonomous-goal-workflow.md:145 를 &#x27;추적 파일의 미커밋 변경 + 미추적 파일(.gitignore 제외분 제외)&#x27; 문구로 고쳤다. 설정 키 이름은 그대로. changelog.md 의 1.4.0 항목은 릴리즈 기록이라 건드리지 않았다.</td><td data-label="갱신">2026-09-23</td></tr></tbody></table></div>

## 원장 (에이전트가 남긴 기록)

## 2026-09-23
- 선택: 파괴적 명령 전 체크포인트가 untracked 파일까지 담도록 고치고 첫 테스트를 붙이기 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `createCheckpoint`의 `git stash create`(추적 변경만 담김)를 `os.tmpdir()` 아래 임시 index 기반 스냅샷(`read-tree`→`add -A`→`write-tree`→`commit-tree`)으로 교체해, 아직 `git add` 하지 않은 새 파일도 `rm -rf` 직전에 보호되게 했다. 스냅샷 커밋은 stash 형식이 아니므로 노트의 복원 블록을 `git diff`/`git checkout <commit> -- .` 로 바꾸고, 커밋이 없는 저장소는 부모 없는 커밋으로 처리했으며, 설정 설명(ko/en)과 `docs/autonomous-goal-workflow.md` 문구를 정정했다. 검증은 `fs.mkdtempSync`로 만든 **진짜 git 저장소**를 통과하는 신규 `tests/unit/checkpoints.test.ts` 9개(가짜 git 대역 없음 — 복원 블록은 노트에서 명령을 파싱해 실제로 실행해 파일 복구를 확인)로 했고, 고치기 전 4개가 예상대로 실패(untracked 누락·`git stash apply` 잔존·빈 저장소)하는 것을 확인한 뒤 고쳤다. `npm run check` 전체 통과(53 tests, typecheck, esbuild 빌드).
- 보류 아이디어: plans.ts(523줄, 테스트 0개) 순수 함수 테스트 보강 — selectPlanName/planMeta/sortByPriority/linkedPlans/openPlanItems (3/1/M) · computeGoalMetrics가 `details.exitCode` 없는(undefined) 검증 항목을 통과로 집계 — 실제 감사 로그에 그런 줄이 나오는지 먼저 확인 필요 (2/1/S) · vibe-coders-proxy.ts provider 저장/복원 경로 테스트 — vscode 설정 API 스텁 확장이 필요해 대역 위주가 되기 쉬움 (3/2/M) · `createCheckpoint`의 catch가 `update-ref` 실패 시에도 "git 저장소가 아님" 노트를 남겨 commit/note가 어긋남 (2/1/S) · 거대한 untracked 디렉터리(.gitignore 미등록 node_modules 등)가 있을 때 동기 `git add -A` 지연 측정과 가드 (2/2/M)
- 과제서: 채택 — 과제서의 근거(stash create가 untracked를 제외, stash apply가 비-stash 커밋 거부)를 실제 git 2.43으로 재현해 확인한 뒤 그대로 구현했다.

- 릴리즈: v1.4.1 (2026-09-23, run 2026-09-23-123422-vibe-code-improve)


[← 대시보드](https://hkjang.github.io/aidev/) · [교훈 모음](https://hkjang.github.io/aidev/lessons/)

{% endraw %}
