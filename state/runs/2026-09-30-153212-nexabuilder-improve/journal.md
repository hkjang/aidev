# 회차 노트 2026-09-30-153212-nexabuilder-improve — nexabuilder
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 15:32] base pinned — master@a3ca143
- [러너 15:32] autonomy release — 

## 정찰 노트
- CI 는 15회차째 같은 서명이고 저장소 밖 원인(비공개 Actions 과금 차단)이다. 이번에도 `git merge-base`(=a3ca143)와 `.github/`·build 파일 diff 가 빈 출력임을 직접 실행해 확인했고, 열린 브랜치 13개 중 `.github/**` 를 건드린 것이 없음도 확인했다. 워크플로 완화는 금지이며 필요도 없다.
- `ListFilterViewController.save` 를 골랐다. 파일 110줄 전체와 리포지토리·엔티티·`runtime/list.html` 호출부(:479/525/561)를 읽어 성립을 확인했고, `git log master.. --all --name-only -- src/main` 으로 열린 브랜치 전부의 점유 파일을 뽑아 **이 파일 점유 0**을 직접 확인했다. 프로덕션 1파일·한 블록이라 6파일 한도에 한참 못 미친다.
- 제친 후보: `MetadataVersionService.restore*`(:334-346 을 직접 읽어 결함은 성립하지만 세 메서드 + 롤백 계약 해석 문제로 위험 3) / 문서 표기 정렬(16회차 차선, 가치 2 — 주과제가 성립해 차선으로 남김) / 폼·리포트·검색 계열은 전부 열린 브랜치 점유.
- 추측으로 적은 것(과제서에 "미확인" 으로 명시): ① 수정 전 없는/휴지통 listId POST 의 실제 응답(코드에 검증이 없다는 사실만 읽었고 실행 안 함) ② `/api/v1/lists/{listId}/views` 가 `@WithMockUser` 만으로 200 인지(세션 접근 코드가 없으므로 그럴 것으로 보이나 확인 안 함) ③ PR #50 잡 로그 본문. 이 세션은 테스트를 한 번도 돌리지 않았다.
- 구현자 주의: `list`/`delete` 에는 게이트를 걸지 말 것(고아 뷰 정리 경로). 상태 코드 선례가 400(BulkList/RuntimeUi)과 404(ListInlineEdit:73)로 섞여 있으니 400 으로 맞추고 다른 파일을 "통일" 하려 들지 말 것. `ApiResponse.ok(String)` 은 `$.message` 로 간다(직전 두 회차가 여기서 걸렸다).
- [러너 15:38] scout done — 수정 과제 — PR #50 의 CI 실패를 증거로 종결(코드·워크플로 원인 아님)하고, `ListFilterViewController.save` 가 **

## 구현 노트
- `ListFilterViewController.save` 맨 앞에 목록 생존 게이트를 넣었다 (+11/-0, 프로덕션 1파일). 없는/휴지통 listId 에 POST 하면 FK 없는 `nexa_list_filter_view` 에 다시 볼 수도 지울 수도 없는 고아 행이 생겼다 — 수정 전 버리는 프로브로 `status=200 rowsForNonexistentList=1 viewId=1` 을 직접 관측했고, 프로브는 삭제했다. 문구·상태는 다른 여섯 경로와 같은 400 `List not found: {listId}`.
- **확신 없는 곳**: `GET`(조회)과 `DELETE`(개별 삭제)는 과제서 지시대로 **일부러 게이트를 걸지 않았다** — 이미 쌓인 고아 뷰를 사용자가 지울 수 있어야 한다는 판단이고, 비평가가 "일관성" 을 이유로 다르게 볼 수 있는 지점이다. `trashedListStillListsAndDeletesTheViewsAlreadySaved` 가 이 비대칭을 고정한다.
- **검증 못 한 것**: PR #50 의 잡 로그 본문(이 세션 `gh` 미인증 — `git diff --stat master... -- .github/ …` 는 빈 출력이므로 브랜치에 CI 를 깨뜨릴 코드는 없다). `filterValuesJson` 4000자 초과·깨진 JSON 동작은 손대지 않았고 `ideas.json` 에 별 항목으로 올렸다.
- **일부러 하지 않은 것**: 리포지토리 쿼리 메서드 신설(공유 계약), 엔티티 FK/`@ManyToOne`(마이그레이션 필요), `ScreenPermissionService` 권한 검사(별개 과제·보호 경로), `AuditService`, `ListInlineEditController:73` 의 404 를 400 으로 "통일"(PR #46 점유), `templates/runtime/list.html`(살아 있는 목록에서 동작 동일).
- **다음 역할이 조심할 것**: 새 테스트는 `@SpringBootTest` + H2 가 필요하고 `properties`·`@DirtiesContext` 를 쓰지 않아 컨텍스트 캐시를 공유한다. `deleted_at` 은 프로덕션 `DELETE /api/v1/admin/lists/{id}` 로 찍는다. 픽스처 정리가 없어 뷰 행이 누적되지만 단언은 전부 랜덤 listId 스코프다. `filterValuesJson` 을 담는 요청 본문은 반드시 `ObjectMapper` 로 만들 것 — 손으로 쓰면 이스케이프 누락으로 500 이 된다(첫 실행에서 실제로 걸렸다).
- [러너 15:51] brief accepted — 채택 — 지정한 프로덕션 파일 1개(`ListFilterViewController`)·`NexaListRepository` 필드 하나만 추가(`repository` 필드명 유지)·`save` �
- [러너 15:51] verify passed — 검증 1개 통과 (auto)
- [러너 15:55] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 15:55] pr created — https://github.com/hkjang/nexabuilder/pull/51
