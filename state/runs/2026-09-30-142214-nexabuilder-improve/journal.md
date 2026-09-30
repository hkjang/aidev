# 회차 노트 2026-09-30-142214-nexabuilder-improve — nexabuilder
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:22] base pinned — master@a3ca143
- [러너 14:22] autonomy release — 

## 정찰 노트
- 이번 브랜치는 master@a3ca143 위의 빈 브랜치라 CI 를 깨뜨릴 코드가 없다(`git diff --stat master...` 빈 출력, `.github/`·`build.gradle.kts`·`settings.gradle.kts`·`gradle/` 도 빈 출력). 14회차째 같은 서명이므로 재추정에 시간을 쓰지 않고 실재 결함으로 넘어갔다 — PR #49 의 잡 로그 본문은 gh 미인증으로 **미확인**이다. 워크플로는 한 줄도 건드리지 말 것.
- 폼 되살리기(#41·#49 점유)·리포트 엔티티 게이트(#44·#45)·`/app/form`(#47)·`TrashController.restore`(#48)는 전부 열린 PR 이 파일을 점유해 제외했다. `git log --all --not master --name-only -- src/main` 으로 미머지 커밋의 프로덕션 파일을 전부 열거해 `core/search/**` 히트 0 을 확인하고 `GlobalSearchService` 를 골랐다. 14회차 연속 차선인 문서 정렬보다 가치가 높고, 전역 검색 테스트가 저장소에 0건이라 테스트 공백도 같이 메운다.
- 추측으로 적은 것: ① 휴지통 행이 실제로 `/api/v1/search` 응답에 나오는지는 **실행으로 확인하지 않았다**(근거는 코드 경로 — `findByActiveTrue` 는 `active` 만 보고 삭제 경로는 `deletedAt` 만 찍는다). 구현자는 수정 전 빨간 테스트로 먼저 관측할 것 ② `/api/v1/search` 가 세션 `User` 없이 200 인지 미확인(401/403 이면 `sessionAttr("nexabuilder.user", User)` 추가).
- 조심할 것: `findByActiveTrue()` 를 다른 finder 로 갈아 끼우거나 리포지토리에 메서드를 추가하지 말 것(공유 계약). 응답 봉투 7개 키·`cap` 상한은 `static/js/nexa-search.js` 가 읽으므로 불변. `workflows`/`codes`/`sqls`/`users` 는 `deletedAt` 필드가 없어 범위 밖.
- 함정: 휴지통 DELETE 의 `"trashed"` 는 `$.data` 가 아니라 `$.message` 에 온다(직전 회차가 밟았다). 엔티티 휴지통은 `DELETE /api/v1/admin/entities/{id}`(`AdminMetadataController:154-171`)이고 `system=true` 면 400 이다.
- [러너 14:28] scout done — 수정 과제 — PR #49 의 CI 실패를 증거로 종결(코드·워크플로 원인 아님)하고, 전역 검색(Ctrl+K)이 **휴지통�

## 구현 노트
- 무엇을 왜: `GlobalSearchService.search` 의 세 mapper 람다(entities/forms/lists)가 `deletedAt != null` 인 후보에서 `hit(...)` 대신 `null` 을 돌려주게 했다. `filter` 가 이미 null 후보를 `continue` 로 건너뛰므로(지금까지 발동하지 않던 가지) 시그니처·`MatchableHit`·`cap`·응답 봉투·나머지 네 타입이 불변이고, `continue` 가 `out.size()` 를 늘리지 않아 휴지통 행이 `limit` 슬롯도 먹지 않는다. 클래스 javadoc 6줄만 새 계약에 맞춰 보탰다(동작 변화 없음).
- 확신 없는 곳·검증 못 한 것: (1) **PR #49 의 CI 실패 사유는 미확인이다** — 이 세션의 `gh auth status` 가 미인증이라 잡 로그 본문을 열지 못했다. 과금 차단이라는 진단은 앞선 셰퍼드 회차의 인증된 증거를 인용한 것이고 내가 재확인한 것은 "이 브랜치에 CI 를 깨뜨릴 코드가 없다"(`git diff master...` 빈 출력) + "CI 의 두 명령이 로컬에서 통과한다" 뿐이다. (2) cap 테스트는 `findByActiveTrue()` 의 순서가 H2 의 PK 인덱스 순이라는 관측에 의존한다(그래서 휴지통 id 를 `gsa1/2/3`, 살아 있는 id 를 `gsz` 로 두어 앞에 오게 했다) — `ORDER BY` 가 없으므로 다른 DB·다른 플랜에서는 순서가 달라져 이 테스트가 결함을 못 잡을 수 있다. 수정 후 통과는 순서와 무관하다. (3) `totalCount` 를 정확한 값(2 → 1)으로 단언한 것은 UUID 8자 접미사가 다른 행과 겹치지 않는다는 전제에 의존한다.
- 일부러 하지 않은 것: 리포지토리에 `findByActiveTrueAndDeletedAtIsNull()` 추가 — `findByActiveTrue()` 는 다른 호출자와 공유하는 계약이다. 전역 검색에 권한 검사 추가 — 컨트롤러 javadoc 이 "역할 무관" 을 의도로 명시하므로 계약 결정이 선행이고 `ideas.json` 에 신규 pending 으로 올렸다. `UiBuilderController`/`NexaUiService` 수정 — PR #49·#41 점유(테스트에서 HTTP 로만 호출했다). `.github/**`·`build.gradle.kts`·`gradle/**` 는 한 줄도 건드리지 않았고 커밋 후 `git diff --stat HEAD~1 --` 로 빈 출력을 확인했다.
- 다음 역할이 조심할 것: 신규 테스트는 Spring 컨텍스트 + H2 가 필요하다(`@SpringBootTest`, `properties` 없음 — 컨텍스트 캐시 재사용을 깨지 말 것). `deleted_at` 은 프로덕션 DELETE 엔드포인트로 찍으므로 `UiBuilderController.deleteList/deleteForm` 이나 `AdminMetadataController.deleteEntity` 의 응답이 바뀌면 이 테스트가 같이 빨개진다. 봉투 주의: `$.total` 이 아니라 **`$.totalCount`** 다(`ApiResponse:17`) — 과제서가 `$.total` 로 적어 첫 실행이 잘못된 이유로 빨갰다.
- [러너 14:42] brief accepted — 채택 — 지정한 프로덕션 파일 1개(`GlobalSearchService` 의 세 람다만)·`mapper` 가 `null` 을 돌려주는 형태·`filter`/`MatchableHit`/`hi
- [러너 14:42] verify passed — 검증 1개 통과 (auto)

## 비평 노트
- 확인한 것: `GlobalSearchService.java` 만 master 판으로 되돌려 신규 테스트를 돌려 `6 tests completed, 4 failed`(:124/:149/:174/:205)을 직접 관측 — 원장의 실패 재현과 증상 일치. 복구 후 `sh ./gradlew --no-daemon test` 전체 **156클래스 615건 / failures 0 errors 0 skipped 0, 5m46s**. `!= null` 이 기존 20여 판정과 같은 형태이고 세 타입 모두 `String` + 스탬프가 `now()` 임을, 나머지 네 타입에 `deletedAt` 히트 0 임을, 소비자가 `nexa-search.js` 하나뿐이고 7키가 불변임을 파일로 확인했다. **판정 approve / risk low / blocking 없음**.
- 못 본 것: PR #49 의 잡 로그(이 세션도 `gh` 미인증), 그리고 `run.json`·`agent-*.txt` 본문.
- 남는 우려 ①: `verify.json` 의 자동 검증은 `--offline test` exit 0 / **1초** — UP-TO-DATE 무효 신호다. 게이트의 "verified" 를 증거로 쓰지 말 것(위 두 실행으로 대체했다).
- 남는 우려 ②: 신규 테스트 두 곳이 순서 의존이다 — cap 테스트는 `findByActiveTrue()` 의 ORDER BY 없는 H2 PK 순(gsa1/2/3 < gsz), `typesWithoutASoftDeleteColumnKeepAnswering` 은 `limit=10` 안에 시드 `admin` 이 들어온다는 전제(haystack 에 `roleId` 가 있어 `ADMIN` 사용자를 10명 넘게 심는 미래 테스트에 흔들릴 수 있다). 수정 자체는 순서와 무관.
- 남는 우려 ③(선재·차단 아님): `/api/v1/search` 는 역할 검사가 없어(`GlobalSearchController:40-43` 이 의도로 명시) 로그인한 아무나 전 메타데이터를 열거하고 email 부분문자열 오라클을 얻는다 — 이번 diff 는 노출을 좁히기만 하므로 차단하지 않았으나 계약 결정이 필요하다. 릴리즈 노트: 남은 휴지통 누수는 `MetadataVersionService.restoreList/Form/Entity(:334-346)` 와 `ListFilterViewController`.
- [러너 14:51] review approved — 리뷰 승인 (risk=low)
- [러너 14:51] pr created — https://github.com/hkjang/nexabuilder/pull/50
- [러너 14:52] ci failed — 성공이 아닌 검사: test + bootJar=failure
