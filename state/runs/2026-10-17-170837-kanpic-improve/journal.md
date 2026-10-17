# 회차 노트 2026-10-17-170837-kanpic-improve — kanpic
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:08] base pinned — main@b0481a9
- [러너 17:08] autonomy release — 

## 정찰 노트
- 오류 대응으로 배정된 "릴리즈 워크플로 실패" 를 저장소에서 찾아봤지만 깨진 단계가 없었다: 태그 v0.258.0~v0.265.0 과 docs/releases/v*.md 가 1:1 로 맞고 release.yml 은 태그 push 전용 4단계뿐이다. 실제 사유는 CI 거절이 아니라 PR 생성 전 중단이고, 그 증거로 지난 회차 커밋 e5b0eb4 가 auto/2026-10-17-1219 에만 남아 main 의 compareLists.ts:130 은 아직 옛 한 줄이다.
- 그래서 과제는 "사라진 그 수정을 다시 올리기" 로 잡았다. e5b0eb4^ == HEAD(b0481a9) 라 cherry-pick 이 충돌 없이 얹히고, 이미 검증된 4줄 변경이라 가치 대비 위험이 가장 낮다. 보류 후보(compareKey 0x10, ADMIN_GUIDE 오류 코드)는 계약 결정·PDF 재생성이 선행이라 제쳤다.
- 미확인: GitHub Actions 로그를 못 봤다 — 이 세션에서 `gh run list` 가 승인 거절로 막혔다. CI 가 정말 같은 단계에서 두 번 깨졌다면 이 진단이 틀린 것이니, 구현자가 `gh run list` 를 한 번만(watch 금지) 보라고 과제서에 적었다.
- 조심할 것: `web/node_modules` 가 없어 `npm ci` 가 선행이다. 같은 파일의 compareKey/looksLikeIdentifier/parseFilterRange 와 EditorPage.tsx 프리필은 별개 과제이니 묶지 말 것, Go 쪽 따옴표 파서는 이미 이중 따옴표를 풀므로 손대지 말 것.
- 가장 큰 위험은 또 PR 전에 끊기는 것이다 — 재분석 없이 cherry-pick → 검증 → 커밋 → PR 을 먼저 끝내라고 과제서 맨 앞과 위험 항목에 둘 다 적었다.
- [러너 17:12] scout done — 지난 회차가 PR 전에 끊겨 사라진 `splitSheetRange` 의 `''` 해제 수정을 main 위에 다시 올린다 (가치 3 / 위험 1 / 

## 구현 노트
- 지난 회차가 PR 전에 끊겨 main 에 못 들어온 `e5b0eb4`(splitSheetRange 의 `''` 해제)를 그대로 되살려 `2b98387` 로 커밋했다. 트리는 `git diff --cached e5b0eb4 --stat` 이 빈 출력 — `e5b0eb4` 와 한 바이트도 다르지 않다. 프로덕션 1파일(4줄)·테스트 1파일.
- 재분석하지 않고 과제서대로 가져오기 → 검증 → 커밋 순으로 끝냈다. 다만 TDD 순서를 지키려고 테스트 파일을 **먼저** 얹어 빨강(`1 failed | 20 passed`, `expected 'Bob''s Data' to be 'Bob's Data'`)을 눈으로 본 뒤 프로덕션 파일을 얹었다. 그 한 줄만 `HEAD` 판으로 되돌리면 정확히 그 테스트 1개만 같은 출력으로 다시 빨강이 된다(실행해 확인).
- **확신 없는 곳·검증 못 한 것**: ① GitHub Actions 실제 실패 로그를 끝까지 못 봤다 — 이 세션의 `gh` 는 미인증(`gh auth login` 안내)이라 `gh run list -L 10` 자체가 불가능했다. 그래서 "릴리즈 워크플로가 같은 이유로 두 번 실패" 라는 배정 전제는 CI 로그로 교차검증되지 않았고, 진단은 저장소 상태(커밋이 브랜치에만 남음)와 회차 노트의 `error: interrupted before PR` 에만 근거한다. 비평가가 로그를 볼 수 있다면 여기부터 봐 주기 바란다. ② `EditorPage.parseSheetRange` 가 실제로 시트를 찾아내는 end-to-end 경로는 단위 테스트까지만 확인했고 브라우저·E2E 로는 돌리지 않았다(Playwright 미실행).
- 일부러 하지 않은 것: `.github/workflows/*` 와 `scripts/release*.sh` 는 한 줄도 건드리지 않았다 — 이번 실패는 테스트·CI 가 거절한 것이 아니라 PR 단계 중단이므로 고칠 데가 없고, 워크플로를 느슨하게 만들어 통과시키는 것은 금지다. Go 쪽 따옴표 파서(`internal/formula/*`)는 이미 `''` 를 풀어서 그대로 뒀다(거울은 통합하지 않는 관례). 같은 파일의 `compareKey`/`looksLikeIdentifier`/`parseFilterRange`, `EditorPage.tsx` 프리필, docs/PDF 도 제외했다.
- 다음 역할이 조심할 것: 이 워크트리에는 이제 `web/node_modules` 가 있으므로 `npm ci` 를 다시 돌릴 필요 없다. 검증은 DB 없이 전부 돈다(integration 태그·Playwright 는 미실행). 릴리즈(VERSION·태그·릴리즈 노트)는 이 세션에서 손대지 않았다 — `check-release-docs.sh` 는 `v0.265.0` 기준으로 통과한다.
- [러너 17:16] brief accepted — 채택 — 과제서의 진단이 저장소 상태와 정확히 맞았고(`e5b0eb4^ == HEAD`, main 은 아직 옛 한 줄, 기존 6사례에 이중 따옴표 �
- [러너 17:16] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인: 그 한 줄을 main 판으로 되돌려 새 테스트가 `expected 'Bob''s Data' to be 'Bob's Data'` 로 실패하는 것을 직접 봤고(복구 후 워크트리 clean), `npm test` 523개·`tsc -b`·커밋 신원·릴리즈 문서 스크립트 모두 통과. 프로덕션은 web 1파일 4줄, Go·workflows·migrations 무접촉.
- Go `quotedQualifier`(structure.go:170)와 손으로 대조: `'Bob''s Data'`·`'''A'''`·`''''` 동일. 다른 두 경계(`'A'B'` 를 TS 만 받음, `''` 를 Go 만 받음)는 main 에도 있던 것으로 이번 회귀가 아니다.
- 못 본 것: GitHub Actions 로그(이 세션도 `gh` 권한 없음)와 Playwright·integration. 다만 release.yml 이 태그 push 전용이고 태그↔릴리즈 노트가 1:1 이라 "CI 거절이 아니라 PR 전 중단" 이라는 진단과 모순되는 증거는 없었다.
- 승인이어도 남는 우려: `EditorPage.tsx:986` 프리필은 아직 `'`→`''` 로 적지 않는다(쓰기 쪽 거울 미정렬, 따옴표로 감싸인 이름의 시트는 프리필이 벗겨짐) — 다음 회차 후보. 릴리즈 노트에는 "대사 범위 입력이 `'Bob''s Data'!A1:B2` 를 받는다" 한 줄이면 충분하다.
- 판정: approve / risk low / blocking 없음(security·legal 모두 공격 경로 없음 — 파싱 결과는 클라이언트 시트 목록 대조에만 쓰이고 API 경로에는 sheet.id 와 숫자 라벨만 들어간다).
- [러너 17:19] review approved — 리뷰 승인 (risk=low)
- [러너 17:19] pr created — https://github.com/hkjang/kanpic/pull/46
- [러너 17:27] ci passed — 검사 2개 모두 success
- [러너 17:27] merge done — 2b98387
- [러너 17:41] release published — v0.266.0
- [러너 17:42] assets verified — v0.266.0 자산 2개 (이전 v0.265.0: 2)
