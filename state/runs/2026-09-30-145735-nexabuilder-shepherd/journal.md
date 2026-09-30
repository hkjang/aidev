# PR 처리기 노트 2026-09-30-145735-nexabuilder-shepherd — nexabuilder PR #49
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-30-133217-nexabuilder-improve)
# 회차 노트 2026-09-30-133217-nexabuilder-improve — nexabuilder
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 13:32] base pinned — master@a3ca143
- [러너 13:32] autonomy release — 

## 정찰 노트
- CI 는 13회차째 같은 서명이고 이번 브랜치에서 `git diff --stat master...` 와 `... -- .github/ build.gradle.kts settings.gradle.kts gradle/` 가 **둘 다 빈 출력**임을 직접 실행해 확인했다 — 고칠 코드가 없으므로 그 자리를 실재 결함으로 채웠다. PR #48 의 잡 로그 본문은 `gh` 미인증으로 **미확인**이다.
- 고른 것: `UiBuilderController.saveList` 의 `deleted_at` 덮어쓰기. `ListFilterViewController`(계약 선행)·문서 정렬(가치 2)·`scaffold`(overwrite 가 명시적 의사표시라 계약이 다름)을 제쳤다. 결정적 이유는 **목록은 저장 경로가 하나뿐**이라 "한쪽만 고치지 말 것" 지침을 위반하지 않고 프로덕션 1파일로 완결된다는 점이다.
- 같은 파일의 `saveForm` 과 `NexaUiService.saveFull` 이 **똑같은 세 번째 결함**임을 읽어 확인했지만, 폼은 두 경로를 동시에 고쳐야 하고 `NexaUiService` 가 PR #41 점유 중이라 일부러 범위 밖으로 뺐다 — 구현자가 눈에 보이는 `saveForm` 을 반쯤 고치는 것이 이번 회차의 최대 위험이다.
- 추측으로 적은 것: 수용 기준 2(거부 시 버전 스냅샷이 남지 않음)의 `GET /api/v1/admin/versions` 응답 봉투와 `TYPE_LIST` 의 실제 문자열 값은 확인하지 않았다 — 성립하지 않으면 단언을 버리고 게이트 위치만 지키라고 과제서에 적었다. 열린 PR 12건의 점유 파일은 회차 기록으로만 판단했고 각 브랜치 diff 를 새로 열지는 않았다.
- [러너 13:38] scout done — 수정 과제 — PR #48 의 CI 실패를 증거로 종결(코드·워크플로 원인 아님)하고, `UiBuilderController.saveList` 가 휴

## 구현 노트
- `UiBuilderController.saveList` 맨 앞(`recordSnapshot` 보다 앞)에 기존 행의 `deletedAt != null` 게이트를 넣어 휴지통 목록이 POST 한 번으로 되살아나는 것을 막았다. 프로덕션 1파일 +11/-0, 신규 테스트 5건. 커밋 8fe2b2e.
- 확신 없는 곳·검증 못 한 것: **PR #48 의 잡 로그 본문은 못 봤다**(`gh` 미인증) — CI 원인은 앞선 셰퍼드 회차들의 증거를 인용한 것이고 이 세션이 새로 증명한 것은 "이 브랜치에 빌드·CI diff 가 없다" 까지다. 그 외에는 전부 실행으로 확인했다.
- 근본 원인은 상태 코드가 아니라 버리는 프로브로 직접 관측했다: `afterDelete deletedAt=20260930134053` → `afterSave deletedAt=null listName=revived`. 프로브 파일은 커밋 전에 삭제했다.
- 일부러 하지 않은 것: `saveForm`(:74-87)과 `NexaUiService.saveFull`(:488-494)은 **똑같은 결함**이지만 폼은 저장 경로가 둘이라 동시에 고쳐야 하고 `NexaUiService` 가 PR #41 점유 중이라 손대지 않았다 — 반쯤 고치면 운영자 지침 위반이다. `scaffold`·`MetadataVersionService` 는 계약이 달라 범위 밖. `.github/workflows` 는 한 줄도 건드리지 않았다.
- 다음 역할이 조심할 것: 새 테스트는 H2 + 전체 Spring 컨텍스트가 필요하다(properties 없음 = 기존 컨텍스트 캐시 재사용, `@DirtiesContext` 없음 — 추가하면 회차마다 ~20초가 더 붙는다). 정리를 하지 않으므로 `nexa_list` 에 `list_ui_*` 행이 누적된다(저장소 관례).
- 검증: `clean test` → BUILD SUCCESSFUL 5m 48s, 614건/실패0/에러0/skip0(156클래스). `bootJar -x test` → `nexabuilder-1.26.0.jar`. 수정 전 타깃 실행은 `15 tests completed, 2 failed`.
- [러너 13:50] brief accepted — 채택 — 지정한 프로덕션 파일 1개(`UiBuilderController.saveList`)·메서드 맨 앞(`recordSnapshot` 보다 앞) 게이트·`listRepository.findByI
- [러너 13:50] verify passed — 검증 1개 통과 (auto)

## 비평 노트
- 실행으로 확인: 신규 테스트 5건 초록(XML `tests="5" failures="0"`)·기존 `EntityBindings`/`SqlBackedListDefinition` 초록, `GlobalExceptionHandler:92-95` 로 IAE→400+message 확정, 문구가 여섯 경로와 동일, `TrashController` 에 restore+purge 가 있어 ID 영구 잠김 없음. 원장의 `실패 재현`(400 expected/200 was)은 수정 전 saveList 에 400 경로가 없다는 코드 독해와 일치한다.
- 못 본 것: 전체 `clean test`(614건)와 `bootJar` 는 이 세션에서 재실행하지 않고 원장 기록을 받아들였다. PR #48 의 CI 잡 로그 본문도 `gh` 미인증으로 미확인 — 구현자가 스스로 표시한 그 한 곳은 이 세션도 뚫지 못했다.
- 승인이어도 남는 우려(릴리즈 노트): `deleted_at` 을 null 로 되돌리는 경로가 아직 둘 더 있다 — `NexaUiService:372 scaffold(overwrite=true, DEVELOPER 도 호출 가능)`, `MetadataVersionService:334-338 restoreList`. "trash/restore 가 유일한 복원 경로" 로 과대 서술하지 말 것.
- 다음 회차: `saveForm`/`saveFull` 의 동일 결함은 PR #41 머지 후 **두 경로 동시에**. 신규 테스트가 정리를 안 해 `list_ui_*` 행이 누적되지만 목록 개수를 절대값으로 박는 기존 테스트는 없음을 확인했다.
- 보안·법무 차단 없음: 새 입력·비밀값·개인정보 없음이고, DELETE 는 ADMIN 전용인데 POST 는 DEVELOPER 가능했던 비대칭(=DEVELOPER 가 ADMIN 삭제를 POST 로 되돌리던 구멍)을 오히려 좁힌다.
- [러너 13:54] review approved — 리뷰 승인 (risk=low)
- [러너 13:54] pr created — https://github.com/hkjang/nexabuilder/pull/49
- [러너 13:55] ci failed — 성공이 아닌 검사: test + bootJar=failure

## 수리 노트
- **지적(CI red)은 이 변경 탓이 아니다 — 커밋 없이 끝냈다.** HEAD `8fe2b2e` 그대로 CI 두 명령 재현: `test` → BUILD SUCCESSFUL 5m44s, 156클래스/**614건**/실패0/에러0/skip0; `bootJar -x test` → BUILD SUCCESSFUL 6s, jar 생성. 잡은 3초에 죽었고 로그 본문이 비어 있다(로컬 test 만 5m44s).
- **맞았던 것**: 구현 노트의 "이 브랜치에 빌드·CI diff 가 없다"와 비평 노트의 신규 5건 초록은 이 세션이 다시 실행해 확인했다. 되돌림 검증도 재현됨 — 프로덕션 파일만 origin/master 로 되돌리면 `5 tests completed, 2 failed`(휴지통 되살림 + 스냅샷 게이트), 복원하면 5/5. 워크트리 clean.
- **새로 더한 증거**: 같은 시각 **다른 브랜치**(PR #50 `685ea05`, GlobalSearchService, 전혀 다른 diff)의 체크 원본이 동일 서명 — 4초 failure·output 전부 null·ann=2·짝 `analyze (java)` 는 completed_at < started_at 인 음수 지속시간 skipped. 코드 무관성이 브랜치 간 대조로 확정됐다.
- **기각**: 액션 핀 부재 가설. checkout@v6 / setup-java@v5 / upload-artifact@v7 태그 모두 GitHub API 로 실재 확인(최신 v7.0.1/v6.0.1/v7.0.1).
- **여전히 확신 없는 곳**: 잡 로그 본문과 annotation 2건은 이 세션도 못 읽었다(`gh` 미인증 + private → 익명 API 404). "과금 차단"이라는 **문구**는 앞 회차의 인증 기록 인용이고 이 세션이 직접 본 것은 타임스탬프·null output 뿐이다. 남은 별개 과제는 `saveForm`/`NexaUiService.saveFull` 동일 결함(PR #41 머지 후 두 경로 동시에)과 `scaffold(overwrite)`·`MetadataVersionService.restoreList` 의 deleted_at 되돌림 경로다.
