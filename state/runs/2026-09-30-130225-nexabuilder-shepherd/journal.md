# PR 처리기 노트 2026-09-30-130225-nexabuilder-shepherd — nexabuilder PR #48
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-30-122208-nexabuilder-improve)
# 회차 노트 2026-09-30-122208-nexabuilder-improve — nexabuilder
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:22] base pinned — master@a3ca143
- [러너 12:22] autonomy release — 

## 정찰 노트
- 우선 과제(CI 실패)는 이번에도 고칠 코드가 없다: `git diff --stat master...` 와 `git diff --stat master... -- .github/ build.gradle.kts settings.gradle.kts gradle/` 를 직접 실행해 둘 다 빈 출력임을 확인했다. 워크플로 완화는 금지이므로 그 자리를 실재 결함으로 채웠다.
- `TrashController.permanentDelete` 를 골랐다: 파일 전체를 읽어 `deleted_at` 게이트가 **전혀 없음**을 확인했고(추측 아님), 열린 PR #37~#47 어느 것도 이 파일을 점유하지 않는다. `/app/form`(1순위 보류 아이디어)은 PR #47 이 `RuntimeUiController` 를 점유해 제쳤고, `UiBuilderController.saveList` 의 되살림·`ListFilterViewController` 는 올바른 계약이 모호해 제쳤다.
- 추측으로 적은 것(미확인): ① 없는 id 에 Spring Data `deleteById` 가 던지는지 no-op 인지 — 존재 게이트를 넣으면 어느 쪽이든 404 로 수렴하나 "수정 전 동작" 은 빨간 테스트로 관측해 적을 것 ② 살아 있는 행을 실제로 purge 하는 것을 실행으로 재현하지 않았다(코드상 명백하나 먼저 빨간 테스트로 볼 것).
- 구현자 주의: `restore` 가 살아 있는 행에도 200 을 주는 것은 별개·범위 밖. 엔티티 하드삭제가 물리 테이블을 남기는 문제(`SchemaDdlService`)에 손대지 말 것. `templates/admin/trash.html` 은 휴지통 행만 이 URL 로 보내므로 화면을 같이 고칠 필요가 없다.
- 프로필은 1일 전 것이 지금 코드와 일치해 새로 쓰지 않았다(한 가지만 보완: `RuntimeUiController` 는 `web/controller` 패키지에 있다).
- [러너 12:27] scout done — 휴지통 하드삭제 엔드포인트가 **휴지통에 없는(살아 있는) 목록·폼·엔티티 정의를 영구 삭제**한다 — `T

## 구현 노트
- 무엇을 왜: `TrashController.permanentDelete` 의 조건 없는 `deleteById` 3분기를 타입별 `findById` → 판정 → `delete(row)` 로 바꿨다. 없음 404 `Row not found:`(restore 와 동일 문구) / 살아 있음(`deletedAt == null`) 400 `Not in trash:` / 휴지통 200 `"purged"`. 알 수 없는 타입은 기존 400 그대로. 프로덕션 1파일, 신규 테스트 1파일 8건. 커밋 ea6fec7.
- 확신 없는 곳·검증 못 한 것: (1) **PR #47 의 런 로그 본문은 미확인** — `gh` 미인증이라 열 수단이 없다. 과금 차단 판정은 앞선 회차의 인증된 증거를 인용한 것이고 이 세션이 새로 확인한 것은 `git diff --stat master...` 가 빈 출력이라는 사실뿐이다. (2) **동시성 미검증** — `findById` 와 `delete` 사이에 다른 요청이 `restore` 로 `deleted_at` 을 null 로 만드는 TOCTOU 경합은 테스트하지 않았다(트랜잭션 경계도 확인하지 않았다). 단일 관리자 화면 경로라 실해악은 작다고 판단했으나 단정하지 않는다. (3) **권한 검증은 그대로 미확인** — 이 엔드포인트에 `ScreenPermissionService` 검사가 있는지 조사하지 않았다(범위 밖). 테스트는 admin 롤로만 돈다.
- 일부러 하지 않은 것: `.github/workflows` 무간섭(과금 차단은 저장소 밖 원인이고 완화는 금지). `restore` 가 살아 있는 행에도 200 을 주는 것(범위 밖, 멱등 복원이 정당한 계약일 수 있다). 엔티티 purge 가 `t_*` 물리 테이블을 고아로 남기는 것(`SchemaDdlService` 는 위험 구역이고 DROP 여부는 데이터 유실을 뜻하는 계약 결정) — 둘 다 `ideas.json` 에 pending 으로 올렸다. `templates/admin/trash.html` 무간섭(휴지통 행만 이 URL 로 보내므로 정상 사용 동작 불변). 감사 로그 신설 안 함.
- 다음 역할이 조심할 것: `TrashPurgeIntegrationTest` 는 **H2 + 실제 리포지토리가 뜨는 `@SpringBootTest`** 라 DB 없이는 못 돈다. properties 를 주지 않아 `DataAdapterListSoftDeleteIntegrationTest` 와 컨텍스트를 공유하니 **properties 를 추가하면 컨텍스트가 하나 늘고 종료마다 ~20초를 더 쓴다**. 픽스처는 정리하지 않으므로(저장소 관례) 행이 누적된다. `"purged"`/`"restored"` 는 `$.data` 가 아니라 **`$.message`** 에 온다 — `ApiResponse.ok(String)` 이 message 오버로드로 바인딩되기 때문이며, 여기서 실수하면 프로덕션 결함으로 오진하기 쉽다.
- [러너 12:39] brief accepted — 채택 — 지정한 프로덕션 파일 1개(`TrashController.permanentDelete`)·`findById` → 판정 → `delete(row)`·판정 순서(없음 404 → 살아 �
- [러너 12:39] verify passed — 검증 1개 통과 (auto)

## 비평 노트
- 실행으로 확인: `TrashPurgeIntegrationTest` 단독 8건 0실패, **전체 스위트 617건/실패0/에러0/skip0(5분36초, exit 0)** — 둘 다 이 세션이 직접 돌렸다. 테스트는 `Not in trash: ` 문구와 MISSING 404 분기를 단언하는데 둘 다 이 diff 가 신설한 것이라 수정 전 통과가 불가능하고, 상태코드에 그치지 않고 `findById` 존재/부재로 단언한다. 원장의 빨간 출력(`expected:<400> but was:<200>`)이 고치는 증상과 일치한다.
- 승인. 인가는 `SecurityConfig.java:196` 의 `/api/v1/admin/**` → `hasRole("ADMIN")` 으로 덮이고 permitAll·DEVELOPER·CSRF-무시 목록 어디에도 없다. 변경은 파괴 범위를 좁히기만 하므로 보안·법무 차단 사유 없음. 범위 이탈 없음, `git revert ea6fec7` 로 깨끗이 복원.
- 남는 우려 ①(다음 회차 후보): `TrashController.java:100-114` 의 검사-후-삭제는 원자적이 아니다 — `@Transactional` 이 없어 `findById` 와 `delete(r)` 가 별개 트랜잭션이고 동시 `restore` 와 경합하면 살아 있는 행이 지워진다. 변경 전에는 경합 없이도 지워졌으니 퇴행은 아니다. `deleteByIdAndDeletedAtIsNotNull` 류로 원자화 가능.
- 남는 우려 ②(릴리즈 노트에 적을 것): 없는 id 의 응답이 200 `"purged"` → **404** 로 바뀌었다 — 커밋 제목보다 넓은 인접 계약 변경이다. `docs/` 에 이 엔드포인트 계약 문서가 없고 유일한 호출자 `admin/trash.html` 은 휴지통 행만 보내므로 정상 사용은 불변, 영향은 중복 purge 시 성공 토스트 대신 실패 토스트가 뜨는 것뿐.
- 못 본 것: 잔여 픽스처(물리 테이블 없는 살아 있는 `t_purge_*` 엔티티 정의 ~5행 + 리스트·폼 각 1행)가 공유 컨텍스트에 누적된다 — 전수 단언 테스트를 찾지 못하고 617건이 초록이라 실해악은 미관측이나, "정의 건강성 전수 검사" 류를 추가할 회차는 이를 염두에 둘 것. `entity` purge 가 `t_*` 테이블과 그 레코드를 남기는 것(삭제 요청 경로에서 "영구 삭제" 가 실제 데이터를 지우지 않음)은 변경 전과 동일하며 손대지 않은 게 옳다.
- [러너 12:53] review timeout — 단계 제한 시간 초과
- [러너 12:53] review approved — 리뷰 승인 (risk=low)
- [러너 12:53] pr created — https://github.com/hkjang/nexabuilder/pull/48
- [러너 12:54] ci failed — 성공이 아닌 검사: test + bootJar=failure

## 수리 노트
- **지적이 틀렸다**(코드 결함 아님): CI `test + bootJar` 실패는 계정 과금 차단이다. 잡이 3초 만에 죽었고 러너가 건넨 실패 로그 본문이 비어 있다 — 체크아웃도 못 돈 흔적. 고치지 않았고 커밋도 없다. HEAD 는 ea6fec7 그대로, 트리 clean.
- **직접 재현해 반증**: CI 의 두 명령을 그대로 로컬 실행 — `test` BUILD SUCCESSFUL 5분46초(XML 집계 156클래스/617건, 실패0·에러0·skip0, `TrashPurgeIntegrationTest` 8건 0실패), `bootJar -x test` BUILD SUCCESSFUL 6초(`nexabuilder-1.26.0.jar` 생성). 비평가의 617건 초록과 수치가 일치한다.
- **여전히 확신 없는 곳**: 이 세션도 `gh` 미인증이라 런 36666441489 의 주석을 직접 읽지 못했다. 과금 차단 판정은 ① 3초+빈 로그 ② PR #37 회차에서 인증된 세션이 읽은 잡 주석("recent account payments have failed or your spending limit needs to be increased") 두 정황의 결합이지 이 세션의 1차 증거가 아니다.
- **일부러 안 한 것**: `.github/workflows` 무간섭(원인이 저장소 밖이고 완화는 금지). 비평 노트의 남는 우려 ①(`findById`/`delete` TOCTOU, `@Transactional` 부재)과 ②(없는 id 200→404 계약 변경)는 CI 실패와 무관한 별건이라 손대지 않았다 — 다음 회차 후보로 유효.
- 다음 역할에게: 이 PR 은 코드 쪽으로 더 할 일이 없다. 소유자가 Billing & plans 를 정리한 뒤 런 재실행만 하면 초록일 것으로 본다.
