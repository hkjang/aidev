# PR 처리기 노트 2026-09-30-115107-nexabuilder-shepherd — nexabuilder PR #46
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-30-093231-nexabuilder-improve)
# 회차 노트 2026-09-30-093231-nexabuilder-improve — nexabuilder
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:32] base pinned — master@a3ca143
- [러너 09:32] autonomy release — 

## 정찰 노트
- CI 는 10회차째 같은 서명이라 재조사하지 않고 종결했다(브랜치 diff 자체가 빈 출력이고 .github/·gradle 변경 0 임을 이번 세션에서 직접 확인, 원인은 인증된 잡 주석으로 이미 증명된 Actions 과금 차단). 그 자리를 채울 결함은 **열린 PR #37~#45 가 점유한 파일을 전부 피하는 것**이 1차 기준이었다 — ReportService(차선 1순위)와 CsvImportController 는 그래서 제쳤고, 문서 정렬(9회차 연속 차선)은 실재 결함이 있는 한 뒤로 미뤘다.
- 고른 것: `ListInlineEditController.patchCell:129-133` 의 `Map.of` 가 null 값에 NPE 를 던져, 셀을 비우는 편집과 빈 셀 최초 입력에서 감사 `details` 의 old/new 가 통째로 사라진다. 기존 `ListInlineEditIntegrationTest` 가 감사 행 **개수만** 세고 내용을 전혀 단언하지 않아 공백이 그대로 남아 있었다.
- 추측으로 적은 것(미확인): NPE 발생 자체는 `Map.of` 의 JDK 계약에서 연역한 것이고 **실행으로 재현하지 않았다** — 이 세션은 Gradle 실행 권한이 거부됐다. `EntityService.update` 가 null 값을 SET NULL 로 처리한다는 것도 코드(:110-116, `params.addValue`)에서 읽은 것이지 실행 확인이 아니다. 구현자는 수정 전에 신규 테스트가 실제로 빨간지부터 확인할 것.
- 조심할 것: 감사 `details` 에 값 원문을 담는 것 자체가 운영자 지침(식별자만)과 어긋나지만 그것은 `BulkListController:157` 과 공유하는 별개 계약 문제다 — 이번에 함께 고치려 들면 M 이상으로 커지고 두 경로 중 하나만 바뀐다. 같은 파일의 권한 검사 부재·`toUpperCase()` Locale 도 명시적으로 범위 밖에 두었다.
- [러너 09:37] scout done — 수정 과제 — PR #45 의 CI 실패를 증거로 종결(코드·워크플로 원인 아님)하고, 인라인 편집이 **값을 비우거

## 구현 노트
- 무엇을 왜: `ListInlineEditController.patchCell` 의 감사 `details` 조립을 `Map.of` → `LinkedHashMap` 으로 바꿨다. `Map.of` 는 null 값에 NPE 를 던지므로, 셀을 비우는 편집(`value:null`)과 빈 셀 최초 입력(`oldValue` null)에서 바로 아래 `catch` 가 `details` 를 `{listId,column}` 축약 fallback 으로 떨어뜨려 old/new 가 통째로 사라지고 있었다. 프로덕션 1파일 한 블록, 테스트 1파일(기존 클래스에 3건 추가). 커밋 f099e93.
- 확신 없는 곳·검증 못 한 것: (1) **CI 잡 로그 본문은 미확인**이다 — `gh` 미인증 + 비공개 저장소라 열 수 없다. 과금 차단이라는 결론은 이전 회차의 인증된 잡 주석을 인용한 것이고 이 세션이 새로 확인한 것은 "브랜치 diff 와 `.github/`·gradle diff 가 빈 출력" 뿐이다. (2) 감사 `details` 에 값 원문(old/new)을 담는 것 자체가 운영자 지침(감사에는 식별자만)과 어긋난다 — 이번 수정은 그 값을 **더 많이** 남기게 한다. 과제서가 명시적으로 범위 밖에 둔 판단이지만 비평가가 이 긴장을 먼저 볼 것이다(`BulkListController:157` 과 공유하는 계약).
- 일부러 하지 않은 것: `:112` 의 `toUpperCase()` Locale.ROOT, `ScreenPermissionService` 권한 검사 추가, 내부 `catch`·fallback 문자열·`auditService.record` 인자 변경, `AuditService`/`AuditChainService`/`EntityService.update`/`isEditable` 수정, `.github/workflows` 전면(한 줄도 안 건드림 — 완화 금지).
- 다음 역할이 조심할 것: 신규 3건은 실제 H2 + Spring 컨텍스트가 필요하다(`@SpringBootTest`). 새 테스트 클래스를 만들지 않았으므로 컨텍스트 수는 그대로다. 감사 행은 공유 H2 에 누적되므로 `ORDER BY audit_id DESC` 첫 행을 읽고 id 는 UUID 8자로 유일하게 만든다 — 이 순서 가정을 깨지 말 것. `clean test` 는 약 6분이다.
- [러너 09:48] brief accepted — 채택 — 지정한 프로덕션 파일 1개·`Map.of`→`LinkedHashMap` 만을 실질 변경으로·삽입 순서 유지·`writeValueAsString`/두 `catch`/fal
- [러너 09:48] verify passed — 검증 1개 통과 (auto)

## 비평 노트
- 확인한 것: 프로덕션 파일만 master 로 되돌려 **직접 실행**했고 신규 3건 중 정확히 2건만 빨갰다(`7 tests completed, 2 failed`; 정상 경로 가드·기존 4건 초록), 복원 후 `tests="7" failures="0" errors="0"`. 원장 `실패 재현` 과 증상이 일치한다. 확인 후 워크트리 clean, HEAD=f099e93. audit_id 가 BIGINT autoIncrement(009)라 `ORDER BY audit_id DESC` 가정도 유효하고, 감사 단언은 jdbc + `readValue(Map.class)` + `containsKey` 라 fallback 축약을 실제로 구분한다. 범위 이탈·마이그레이션·워크플로 변경 없음, revert 로 완전 복구 가능.
- 못 본 것: 전체 `clean test`(원장의 612건)는 재실행하지 않고 대상 클래스만 돌렸다. 감사 로그 뷰어 UI 가 `new: null` 을 어떻게 렌더하는지도 확인하지 않았다(템플릿에 details 포맷터가 안 보였다).
- 승인이어도 남는 우려(릴리즈 노트): 이 수정은 **셀을 비우는 편집에서 지워진 값을 감사에 새로 남긴다** — 목적·보존(`retention.audit.days` 기본 365)·ADMIN 접근 통제가 다 있어 차단은 아니지만, `ErasureService.execute` 는 엔티티 행만 손대고 `nexa_audit.details`·`nexa_row_history` 에 닿지 않으므로 파기 전 인라인 편집 값이 보존 기간까지 남는다. `BulkListController:157` 과 공유하는 기존 결함이니 후속 티켓으로.
- 다음 회차 결함 후보: `/api/v1/data/list/**` 가 `SecurityConfig:198` 의 `.anyRequest().authenticated()` 로 떨어지고 `ListInlineEditController` 에 행 범위·`ScreenPermissionService` 검사가 없어, 인증된 USER 가 `listId`/`key` 만 바꿔 다른 목록·행의 editable 컬럼을 PATCH 할 수 있다. master 기존 결함이라 이번 차단 사유로 쓰지 않았다. `:141` fallback 의 listId 미이스케이프도 같은 성격(이번 수정으로 거의 도달 불가).
- [러너 09:53] review approved — 리뷰 승인 (risk=low)
- [러너 09:53] pr created — https://github.com/hkjang/nexabuilder/pull/46
- [러너 09:54] ci failed — 성공이 아닌 검사: test + bootJar=failure

## 수리 노트
- 지적이 틀렸다(코드 원인 아님): HEAD=f099e93 에서 CI 와 같은 두 명령이 로컬 통과 — `cleanTest test` 612건/실패0/에러0/skip0(155클래스, 5m46s), `bootJar -x test` 6s, 대상 클래스 7건 전부 초록. 고치지 않고 커밋 없이 끝냈다.
- 결정적 근거는 **3초**다: 실패 검사의 소요가 로컬 test 단계(5m46s)보다 두 자릿수 짧아 잡이 `./gradlew test` 전에 죽었다. 이 브랜치는 `.github/`·gradle·gradlew·build.gradle.kts 를 한 줄도 바꾸지 않았다(diff 빈 출력).
- 세운 뒤 기각한 가설: `upload-artifact@v7`/`checkout@v6`/`setup-java@v5` 가 없는 태그라 "Set up job" 에서 즉사한다 → GitHub API 로 각각 v7.0.1/v6.0.1/v7.0.1 실재 확인, 기각. 액션 핀은 문제가 아니다.
- 여전히 확신 없는 곳: 잡 로그 본문을 못 봤다(`gh` 미인증, 비공개라 API·PR 페이지 404). "과금/지출 한도 차단" 은 이전 회차의 인증된 주석을 인용한 추론이며 이번 세션이 새로 증명한 것이 아니다. 3초라는 관측과 양립하는 다른 인프라 원인(러너 미할당 등)도 배제하지 못했다.
- 중재자에게: 이 PR 의 실질 변경은 여전히 프로덕션 1파일 한 블록이고 비평가가 revert 로 빨간 것을 확인한 그대로다. CI 를 초록으로 만들려면 코드가 아니라 저장소의 Actions 권한/과금을 사람이 봐야 한다.
