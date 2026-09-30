# 수리 요약 (시도 2) — 커밋 3360d9a

- 지적 두 건 모두 사실이었다. 직접 확인: `BulkListController.resolveTarget`(192-198)과 `ReportService.generate`(70-71)는 `deletedAt` 을 아예 읽지 않고(두 파일 grep 0건), `ListInlineEditController`(72-74)는 읽지만 404 를 돌려준다.
- 프로덕션 주석: 거짓 선례 `BulkListController.resolveTarget` 을 실제 선례 `ListExportController.resolveExportable`(`requireLiveList` → `canView`, 329-334)로 교체하고, "exports 와 reports 가 이미 그렇게 한다" 에서 reports 를 뺐다. 미검사 두 경로와 404 한 경로를 "열린 gap, 이 순서의 선례가 아님" 으로 명시했다.
- 테스트 javadoc: "여섯 readers 전부 동일 400 / 화면이 마지막" 을 "requireLiveList 와 그 두 호출자 셋이 400" 으로 고치고, 남은 세 경로의 실제 동작(404 / 미검사)을 적어 "마지막" 완결 선언을 제거했다. 155·123 줄 인라인 주석의 같은 인용도 고쳤다.
- 프로덕션 로직·테스트 단언은 한 줄도 바꾸지 않았다(주석만). 검증: `sh ./gradlew --no-daemon test --tests 'com.nexabuilder.api.RuntimeUiSoftDeleteIntegrationTest' --tests '*ViewIntegrationTest'` → BUILD SUCCESSFUL 19s, XML 집계 9클래스/31건/실패0/에러0/skip0.
- 확신 없는 곳: 전체 `test`(약 619건)는 재실행하지 않았다 — 변경이 주석뿐이라 위 범위로 충분하다고 판단했다.
