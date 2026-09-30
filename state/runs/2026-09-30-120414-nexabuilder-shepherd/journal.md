# PR 처리기 노트 2026-09-30-120414-nexabuilder-shepherd — nexabuilder PR #47
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-30-111247-nexabuilder-improve)
# 회차 노트 2026-09-30-111247-nexabuilder-improve — nexabuilder
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:12] base pinned — master@a3ca143
- [러너 11:12] autonomy release — 

## 정찰 노트
- CI 는 11회차째 같은 서명이라 재조사하지 않고 종결했다: 이번 브랜치에서 `git diff --stat master...` 와 `git diff --stat master... -- .github/ build.gradle.kts settings.gradle.kts gradle/` 가 **둘 다 빈 출력**(직접 실행), 빌드·CI 설정 최신 커밋은 2026-09-28 `c446d24`. 고칠 코드가 없고 워크플로 완화는 금지라 `ideas.json` 의 CI 항목을 `done` 으로 닫았다. 잡 로그 본문은 `gh` 미인증으로 **미확인**.
- 그 자리를 `RuntimeUiController` 의 목록 soft-delete 누락으로 채웠다. 이 항목은 2026-09-29 회차에 "8군데라 M" 으로 밀렸지만, 8개 핸들러가 같은 3블록을 복제하고 있어 private 헬퍼 하나로 동시에 고쳐지고 프로덕션 파일이 1개다. 존재하지 않는 목록과 같은 예외를 던지면 템플릿을 렌더하지 않으므로 "템플릿 8개 빈 상태 기대값" 걱정도 사라진다 — 그래서 S 로 재평가했다.
- 제친 후보: 문서 표기 정렬(가치 2, 10회차 연속 차선으로 남김), `ReportService` 엔티티 deletedAt·리포트 파일명 충돌·인라인편집 권한 검사·감사 PII(전부 열린 PR 9건이 파일을 점유), 차트·피벗 fetch 실패 토스트(같은 화면 계열이라 회차 분리).
- 추측으로 적은 것: 휴지통 목록의 `/app/*` 가 지금 **200** 을 돌려준다는 것은 코드 읽기에 근거한 예상이고 실행으로 재현하지 않았다. 직전 두 회차에서 과제서의 예상 상태 코드가 실제와 달랐으므로(400 예상/실제 500), 구현자는 빨간 테스트로 현재 상태를 먼저 확인하고 관측을 따를 것. `requireLiveList:59-66`·`GlobalExceptionHandler:92-95`·`NexaListDef:134-135`·`DataAdapterListSoftDeleteIntegrationTest:151` 은 직접 읽어 확인했다.
- 조심할 것: 기존 7개 `*ViewIntegrationTest` 가 "살아 있는 목록은 200" 회귀 가드이므로 **고치지 말 것**. `/app/form` 과 `NexaUiService` 주입은 범위 밖. 프로필은 1일 전 것이 현재 코드와 어긋나지 않아 새로 쓰지 않았다.
- [러너 11:17] scout done — 수정 과제 — PR #46 의 CI 실패를 증거로 종결(코드·워크플로 원인 아님)하고, 휴지통에 넣은 목록의 `/app/*`

## 구현 노트
- 무엇을 왜: `RuntimeUiController` 의 8개 목록 화면이 복제한 `findById+orElseThrow+canView` 3블록을 private 헬퍼 `requireViewableList` 하나로 뽑고, 그 안 `orElseThrow` 직후·`canView` 보다 앞에 `deletedAt != null` 검사를 넣었다. 다른 여섯 경로와 글자 단위로 같은 `IllegalArgumentException("List not found: " + listId)` → 400. 프로덕션 파일 1개(34+/44-), 테스트 1개 신규. 커밋 8cca51d.
- 확신 없는 곳·검증 못 한 것: (1) **CI 잡 로그 본문은 미확인** — `gh` 미인증이라 이 세션에서 열 수 없다. 저장소 밖 원인(과금 차단)이라는 판정은 앞선 회차의 인증된 증거에 기댄 것이고, 이번에 내가 직접 실행해 확인한 것은 "이 브랜치의 diff 가 빈 출력" 뿐이다. (2) 템플릿 8개가 휴지통 목록에서 렌더되지 않는 것은 예외로 우회되므로 확인했지만, **살아 있는 목록의 템플릿 내부 동작은 기존 테스트가 보는 만큼만** 봤다(200 + 뷰 이름 + 모델 3속성). (3) `AccessAuditHandler` 가 403 에서 감사 행을 쓰는 것은 상태 코드로만 확인했고 **감사 테이블 행을 직접 읽지 않았다**.
- 일부러 하지 않은 것: `/app/form/{formId}` 는 `formRepository`·다른 엔티티이고 `formDefinition` 의 `includeInactive` 계약과 얽혀 증거가 흐려지므로 손대지 않았다(`ideas.json` 에 별 항목). `NexaUiService.requireLiveList` 재사용을 위한 주입도 하지 않았다 — PR #41 이 그 파일을 점유 중이라 계약이 흔들리면 두 회차가 같이 깨진다. `.github/workflows` 는 한 줄도 건드리지 않았고 재시도·완화도 넣지 않았다. 기존 7개 `*ViewIntegrationTest` 는 회귀 가드라 한 글자도 고치지 않았다.
- 다음 역할이 조심할 것: `RuntimeUiSoftDeleteIntegrationTest` 는 H2·Liquibase·MockMvc 가 올라오는 `@SpringBootTest` 라 DB 없이는 돌지 않고, 픽스처 정리를 하지 않는다(저장소 관례). 8경로는 `@ParameterizedTest` 라 실패 시 어느 뷰인지는 `viewName` 파라미터로 읽을 것. 헬퍼의 검사 **순서**(존재 → 휴지통 → 권한)가 계약이다 — 권한 검사를 앞으로 옮기면 휴지통 목록이 403 이 되어 다른 여섯 경로와 어긋난다.
- [러너 11:29] brief accepted — 채택 — 지정한 프로덕션 파일 1개·복제 3블록을 private 헬퍼 하나로 추출·`orElseThrow` 직후·`canView` 보다 앞·`!= null` 만(`isB
- [러너 11:29] verify passed — 검증 1개 통과 (auto)


## 비평 노트
- 확인한 것(실제 실행·파일 열기): diff 전체, `RuntimeUiController` 전문, 그리고 테스트 javadoc 이 선례로 인용한 여섯 경로를 하나씩 열었다. `test --tests 'RuntimeUiSoftDeleteIntegrationTest' --tests '*ViewIntegrationTest'` → BUILD SUCCESSFUL 18s, 신규 10건 0실패, 기존 뷰 8클래스 21건 0실패. `canView(null,…)` 은 `ScreenPermissionService:35` 에서 즉시 false 라 403 단언이 성립한다. 원장의 실패 재현 출력은 diff 와 앞뒤가 맞는다 — 테스트는 진짜로 바뀐 경로를 지난다.
- 거절 사유는 동작이 아니라 **주석의 사실 오류**다: `RuntimeUiController.java:30-32` 가 순서 계약의 선례로 든 `BulkListController.resolveTarget` 에는 deletedAt 검사가 없고(`BulkListController.java:192-198`), 테스트 javadoc `36-42` 의 "여섯 readers 전부 동일 400 / 화면이 마지막" 은 셋만 맞다 — `ReportService.generate` 와 `BulkListController` 는 미검사, `ListInlineEditController:72-74` 는 404 다. 수리가 가장 먼저 볼 파일은 이 두 곳의 주석이고, 프로덕션 로직과 테스트 단언은 손대지 말 것.
- 못 본 것: 전체 `clean test`(619건)는 재실행하지 않고 구현자 보고를 그대로 두었다. 감사 테이블 행, CI 잡 로그 본문, 템플릿 8개 내부 렌더도 확인하지 않았다(기존 테스트가 보는 범위까지만).
- 승인 후에도 남는 우려(릴리즈 노트·다음 회차): 휴지통 목록에 대해 bulk update/delete 와 리포트 CSV 생성이 **여전히 동작한다**. 이번 PR 이 만든 구멍이 아니고 넓히지도 않으므로 차단하지 않았지만, "휴지통 soft-delete 를 전 경로에서 닫았다" 고 릴리즈 노트에 쓰면 거짓이 된다. `ideas.json` 에 두 경로를 별 항목으로 남길 것.
- [러너 11:33] review rejected — 리뷰 거절: src/main/java/com/nexabuilder/web/controller/RuntimeUiController.java:30-32 헬퍼의 검사 순서(존재 → 휴지통 → 권한)를 '다른 readers 와 같다, 예: BulkListControll

## 수리 노트
- 지적 두 건 다 맞았다. 틀린 것은 없었다. 내가 직접 열어 확인: `BulkListController.resolveTarget:192-198` 존재→권한만(deletedAt 0건), `ReportService.generate:70-71` 미검사(파일 grep 0건), `ListInlineEditController:72-74` 검사하지만 404, 그리고 `ListExportController.resolveExportable:328-334` 는 `requireLiveList`→`canView` 라 순서 계약의 **진짜** 선례다.
- 고친 방법: 주석만 고쳤다(프로덕션 로직·테스트 단언 0줄 변경). 컨트롤러 javadoc 의 거짓 선례를 `ListExportController.resolveExportable` 로 교체하고 "reports 도 이미 그렇게 한다" 를 삭제; 미검사 2경로·404 1경로를 "열린 gap" 으로 명시. 테스트 javadoc 의 "여섯 readers 전부 400 / 화면이 마지막" 을 "셋이 400" 으로 축소하고 남은 셋의 실제 동작을 적어 완결 선언을 없앴다. 인라인 주석 123·155 줄의 같은 인용도 고쳤다. 커밋 3360d9a.
- 검증: 비평가가 돌린 것과 같은 명령 `test --tests 'RuntimeUiSoftDeleteIntegrationTest' --tests '*ViewIntegrationTest'` → BUILD SUCCESSFUL 19s, 결과 XML 집계 9클래스/31건/실패0/에러0/skip0.
- 여전히 확신 없는 곳: 전체 `test`(직전 보고 619건)는 재실행하지 않았다 — 변경이 주석뿐이라 생략했다. CI 잡 로그 본문·감사 테이블 행·템플릿 내부 렌더도 이번에 보지 않았다(앞 회차와 동일).
- 다음 회차용: 휴지통 목록에 대한 bulk update/delete 와 리포트 CSV 는 **여전히 열려 있다**. 릴리즈 노트에 "전 경로에서 닫았다" 고 쓰면 거짓이다.
- [러너 11:41] repair done — # 수리 요약 (시도 2) — 커밋 3360d9a  - 지적 두 건 모두 사실이었다. 직접 확인: `BulkListController.resolveTarget`(192-198)과 `ReportService.generate`(70-71)는 `deletedAt` 을

## 비평 노트 (중재 전 심사)
- 직접 확인: diff 전문, `RuntimeUiController` 전문(남은 `listRepository.findById` 는 헬퍼 51줄 한 곳뿐, 8경로 모두 위임), 주석이 인용한 네 파일 — `ListExportController.resolveExportable:328-334`(requireLiveList→canView, 선례 사실), `ListInlineEditController:71-73`(검사하되 404), `BulkListController.resolveTarget:193-199`(미검사), `ReportService.generate:70-71`(미검사), `NexaUiService.requireLiveList:59-66`·`GlobalExceptionHandler:92-95`(IllegalArgumentException→400+`{success:false,message}`). 주석의 사실 주장은 이제 전부 코드와 일치한다. 내가 돌린 `test --tests 'RuntimeUiSoftDeleteIntegrationTest' --tests '*ViewIntegrationTest'` → BUILD SUCCESSFUL 17s, 9클래스/31건/실패0/에러0/skip0.
- 테스트 유효성은 diff 로 확인: 수정 전 코드에는 `deletedAt` 검사가 없어 휴지통 목록도 200 을 렌더하므로 각 테스트의 live 200 → trash 400 쌍 중 뒤쪽이 반드시 실패한다. 또 `sample-list` 를 휴지통에 넣는 테스트는 저장소 전체에 없어(`grep`) `BuilderApiIntegrationTest:362` 등 기존 `/app` 테스트와 간섭하지 않는다.
- 못 본 것: 전체 `test`(619건)·CI 잡 로그·`nexa_audit_log` 행·템플릿 8개 내부 렌더. 판정: approve, risk low, 차단 없음(새 경로·권한 확대·PII·비밀값 없음, 오히려 읽기 경로를 좁힌다. revert 로 완전 복구, 마이그레이션 없음).
- 승인 후 남는 우려(릴리즈 노트): (1) `/app/form/{formId}` 는 `NexaFormDef.deletedAt:52` 를 읽지 않아 **휴지통에 넣은 폼 화면은 여전히 렌더된다** — "런타임 화면의 soft-delete 를 닫았다" 로 뭉뚱그리면 거짓이다. (2) bulk update/delete·리포트 CSV 도 여전히 열려 있다(수리 노트와 동일). (3) 휴지통 목록에 대한 **무권한 접근은 이제 권한 게이트 앞 400 이라 감사 행이 남지 않는다**(전에는 403+감사). `resolveExportable` 과 같은 순서라 일관적이지만 탐지 범위가 그만큼 줄었다 — 다음 회차가 알아야 할 것.
- [러너 11:45] review approved — 리뷰 승인 (risk=low)
- [러너 11:45] pr created — https://github.com/hkjang/nexabuilder/pull/47
- [러너 11:45] ci failed — 성공이 아닌 검사: test + bootJar=failure
