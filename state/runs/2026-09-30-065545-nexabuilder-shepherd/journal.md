# PR 처리기 노트 2026-09-30-065545-nexabuilder-shepherd — nexabuilder PR #43
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-30-062227-nexabuilder-improve)
# 회차 노트 2026-09-30-062227-nexabuilder-improve — nexabuilder
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 06:22] base pinned — master@a3ca143
- [러너 06:22] autonomy release — 

## 정찰 노트
- CI 는 7회차째 같은 서명(PR #42 `a7d58cb0` 의 `test + bootJar` 가 **9초** 만에 failure, `analyze (java)` skipped)이고 브랜치의 빌드·CI 설정 변경은 0줄(`git diff --stat master... -- .github/ build.gradle.kts settings.gradle.kts gradle/` 빈 출력, 이번에 실제 실행)이라 고칠 코드가 없다 — 과금 차단으로 종결하고 워크플로는 손대지 말 것. 6회차 재추정은 낭비였다.
- 그 자리를 채울 과제로 `BulkListController.resolveTarget` 의 soft-delete 누락을 골랐다: 목록 판독 경로 다섯 곳이 전부 `deletedAt` 을 거부하는데 **가장 파괴적인 두 엔드포인트만** 통과시키고, 이 컨트롤러는 테스트가 **0건**이며(`src/test` 에서 `bulk/delete`·`BulkListController` 검색 결과 없음), 열린 PR 6건(#37~#42)이 점유한 여섯 파일을 모두 비껴간다.
- 문서 정렬(차선)은 4회차 연속 밀렸지만 가치 2 라 또 제쳤다. PII 마스킹·권한 게이트는 계약 설계 선행 또는 보호 경로라 자율 회차에 부적합.
- 추측으로 적은 것: (1) SQL 기반 목록(`entityId` null)에서 현재 어떤 상태 코드가 나오는지 **미확인** — 빨간 테스트로 먼저 확인하고 이미 400 이면 그 항목은 버릴 것. (2) CI 로그 본문은 이 세션에서도 열지 못했다(`gh` 미인증 + 비공개 저장소).
- 구현자가 조심할 것: `@WithMockUser` 만으로는 부족하다 — `hasExplicitOrAdmin` 이 `session.getAttribute("nexabuilder.user")` 를 읽으므로 `sessionAttr` 로 실제 `User` 를 넣어야 하고, 넣지 않으면 403 이 나서 결함이 아닌 이유로 빨개진다. 회귀 가드(살아 있는 목록 200 + `affected:1`)를 수정 **전에** 초록으로 확인하고 시작할 것.
- [러너 06:27] scout done — 수정 과제 — PR #42 의 CI 실패를 증거로 종결(코드·워크플로 원인 아님)하고, **휴지통에 넣은 목록으로도 

## 구현 노트
- 무엇을 왜: `BulkListController.resolveTarget` 이 목록의 `deletedAt` 을 안 봐서 휴지통 목록의 `/bulk/delete`·`/bulk/update` 가 밑의 테이블 행을 계속 지우고 고쳤다. `orElseThrow` 직후·권한 검사보다 앞에서 확인해 다른 다섯 판독 경로와 동일한 400 `"List not found: {listId}"` 를 던진다. 같은 메서드에서 SQL 기반 목록(`entityId` null)이 `findById(null)` 로 내려가 불투명한 500 이 되던 것도 400 으로 접었다. 프로덕션 파일 1개, 테스트 1개(신규 6건). 커밋 eaf9078.
- 확신 없는 곳·검증 못 한 것: (1) **CI 로그 본문은 못 봤다** — `gh` 미인증이라 이 세션에서 여는 것이 불가능하다. 종결 근거는 `ci-a7d58cb02962.json`(9초 failure, output 전부 null, analyze skipped)과 앞선 셰퍼드 회차의 인증된 과금 차단 증거다. 재현이 아니라 **인용**이다. (2) SQL 기반 목록의 400 문구는 `ListInlineEditController` 를 본떠 새로 지은 것이라 기존 계약을 고정한 게 아니다 — 프런트엔드가 이 문구에 의존하지는 않는다고 보았지만 확인하지 않았다. (3) 권한이 **없는** 사용자가 휴지통 목록을 부르면 이제 403 이 아니라 400 이 난다(soft-delete 검사를 앞에 둔 결과). 의도한 것이지만 테스트로 고정하지 않았다.
- 일부러 하지 않은 것: `.github/workflows` 무수정(재시도·`continue-on-error`·타임아웃 완화 없음). `NexaListRepository` 에 쿼리 메서드 추가 안 함 — 컨트롤러에서 `getDeletedAt()` 만 본다. `hasExplicitOrAdmin`·`ScreenPermissionService`·`EntityService`·`AuditService` 무수정. 감사 `values` 원문 문제(`:157`)는 PII 계약 설계가 선행이라 뺐다(`ideas.json` 에 pending).
- 다음 역할이 조심할 것: 새 테스트는 **세션 `User` 와 `@WithMockUser` 둘 다** 필요하다 — `@WithMockUser` 만으로는 `AccessDeniedException` 으로 403 이 난다. H2 통합 테스트라 DB 가 떠야 돌고, 정리(cleanup)·`@DirtiesContext` 없음. 열린 PR 6건과 파일이 겹치지 않는다.
- [러너 06:39] brief accepted — 채택 — 지정한 프로덕션 파일 1개·`orElseThrow` 직후·권한 검사보다 앞·`IllegalArgumentException` + 동일 문구·`NexaListRepository` �
- [러너 06:39] verify passed — 검증 1개 통과 (auto)

## 비평 노트
- 판정 approve(risk low, blocking 없음). 테스트를 신뢰하지 않고 직접 재현했다: 컨트롤러만 master 판으로 되돌리면 `6 tests completed, 4 failed`(휴지통 2 + SQL 2), 회귀 가드 2건은 그대로 초록, 복원 후 6/0/0/0. 원장의 실패 재현과 증상이 맞는다.
- 코드는 `requireLiveList:62` 와 글자 단위로 같고 `GlobalExceptionHandler:92-95` 가 400 으로 접는 것까지 확인했다. 마이그레이션·외부 상태 없음 → revert 로 완전히 돌아온다.
- 남는 우려(차단 아님): ① soft-delete 를 권한 앞에 둬 권한 없는 사용자가 휴지통 여부를 403/400 으로 구분한다(데이터는 못 얻음, 테스트로 고정 안 됨). ② 테스트 javadoc 의 "다른 다섯 경로 전부 400" 은 부정확 — `ListInlineEditController:73` 은 404 다. 릴리즈 노트에 400 이라 단정하지 말 것. ③ 커밋 메시지가 SQL 기반 목록 500→400 을 빠뜨렸으니 릴리즈 노트에는 두 변경 다 적을 것.
- 못 본 것: 전체 스위트·bootJar 재실행(원장 인용만), CI 과금 차단 근거(이 세션도 `gh` 미인증).
- 다음 회차 우선 후보: `BulkListController:157` 의 감사 details 에 들어가는 `values` 원문(PII, 이번 회차가 들여온 것은 아님).
- [러너 06:42] review approved — 리뷰 승인 (risk=low)
- [러너 06:42] pr created — https://github.com/hkjang/nexabuilder/pull/43
- [러너 06:43] ci failed — 성공이 아닌 검사: test + bootJar=failure

## 수리 노트
- 지적("CI test + bootJar 실패")은 **틀렸다 — 이 변경의 결함이 아니다**. 브랜치 HEAD 에서 CI 두 명령을 그대로 재현: `test` 초록(615건/실패0/에러0/skip0, 5m53s), `bootJar -x test` 초록. 고친 것 없음, **커밋 없음**.
- 되돌림 검증으로 테스트가 대상을 실제로 실행함을 확인: 컨트롤러만 master 판으로 되돌리면 4/6 FAILED(회귀가드 2는 초록), 복원 후 6/6. 비평가의 재현과 일치한다.
- 실패 원인은 인프라: 잡이 **3초**(마지막 정상 실행 8분27초)만에 끝나고 output 이 전부 null. **새 증거** — 같은 계정 같은 시간대에 PUBLIC 저장소(releasedock/ptium/muni/orbit/moyro/jupiq)는 전부 분 단위 success, PRIVATE(nexabuilder·vibe-code, API 404)만 3~9초 failure. 공개=무료·비공개=과금이므로 **비공개 저장소 Actions 과금 차단**이다. 앞 회차의 "계정 전체 차단"보다 좁고, 인용이 아니라 이 세션에서 직접 측정했다.
- 여전히 확신 없는 곳: 실패 체크의 annotation 2건 본문(`gh` 미인증+비공개라 못 봄) — 차단 문구를 직접 읽지는 못했고 타이밍·가시성 상관으로 추론했다. 2026-09-29T13:01Z 첫 실패만 39초로 이례적이라(이후 3~9초) 그 회차가 별개 원인이었을 가능성은 배제 못 했다.
- 중재자에게: 결제가 풀리면 재실행만으로 초록이 될 것으로 본다. 워크플로·검증 명령은 규칙대로 무수정. 비평가가 남긴 우려 3건(권한 앞 soft-delete, javadoc 의 "다섯 경로 전부 400" 부정확 — `ListInlineEditController:73` 은 404, 커밋 메시지가 SQL목록 500→400 누락)은 차단이 아니라 이번에 손대지 않았다.
