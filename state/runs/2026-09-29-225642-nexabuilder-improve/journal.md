# 회차 노트 2026-09-29-225642-nexabuilder-improve — nexabuilder
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 22:56] base pinned — master@a3ca143
- [러너 22:56] autonomy release — 
- [러너 23:02] scout done — PR #37 (`auto/2026-09-29-2142` @ `0421f6d`) 의 CI `test + bootJar` 실패를 원인까지 파고들어 고친다 (가치 5 / 위험 2 / 작

## 구현 노트
- 커밋 2개: ee4c7fb = PR #37 의 0421f6d 를 그대로 cherry-pick(프로덕션 1파일), f9d6436 = 신규 SqlBackedListSoftDeleteIntegrationTest 3건(프로덕션 0줄). 워크플로는 한 줄도 안 건드렸다.
- **확신 없는 곳**: CI 실패의 진짜 원인. 코드가 아니라는 것만 증명했다(clean 615건 통과, 39초 vs 성공 12회 8~10분). 어떤 환경 실패인지는 **모른다** — gh 미인증 + 비공개 저장소라 로그·annotation 본문을 못 봤다. 과제서의 후보 2(네트워크 플레이크)도 확인 못 한 가설이다. 따라서 이 PR 의 CI 가 다시 같은 39초 실패를 낸다면 원인은 여전히 미규명이며, 그때는 러너가 `gh run view --log-failed` 출력을 파일로 남기는 것이 먼저다(ideas.json 4/1/S 항목).
- 과제서가 후보 1 의 근거로 든 `annotations_count: 2` 는 성공 회차 10회의 정상값이다. 다음 역할은 이 숫자를 오류 신호로 읽지 말 것.
- **일부러 안 한 것**: `.github/workflows/ci.yml` 수정(재시도·타임아웃 포함) — 수용 기준 4 의 완화 금지, 그리고 원인 미규명 상태의 워크플로 수정은 증상 덮기다. `gradle-wrapper.properties` 의 `networkTimeout=10000` 도 후보 2 를 확인하지 못해 그대로 뒀다.
- **다음 역할 주의**: 신규 테스트는 H2 + 실제 Spring 컨텍스트가 필요하다(@SpringBootTest + MockMvc, 목 없음). 세 테스트 모두 삭제 전 200 단언을 먼저 하므로 픽스처가 깨지면 400 이 아니라 그 줄에서 빨개진다. 빨강 확인은 `requireLiveList` 의 `deletedAt` 분기를 일시 무력화해서 했고 커밋 전에 되돌렸다(`git diff` 로 확인함).
- 검증: `sh ./gradlew --no-daemon clean test` 618건 0 fail/0 error/0 skip (5분41초), `sh ./gradlew --no-daemon bootJar -x test` 성공 (6초).
- [러너 23:17] brief accepted — 채택 — 수용 기준 1(로그)은 이 세션의 권한으로 불가능해 명시적으로 미확인으로 남겼고, 과제서가 지정한 대체 경로(�
- [러너 23:17] verify passed — 검증 1개 통과 (auto)

## 비평 노트
- 확인함: ListExportController 를 master 판으로 되돌려 재실행 → 신규 6건 중 2건(csv/xlsx all-hidden)만 빨강, 되돌리기 후 19/0 + soft-delete 3/0 (워크트리 복구 확인). resolveColumns 부분 실패 경로와 columnsJson 소비자 전수 조사도 함 — 파급은 이 컨트롤러 한 곳.
- 못 본 것: 전체 618건 재실행(구현자 보고 신뢰), CI 실패 로그(gh 권한 없음), PDF 본문 렌더 결과.
- 승인이어도 남는 우려 1 — 릴리즈 노트에 "전 컬럼 hidden 목록의 내보내기는 이제 빈 파일" 을 반드시 적을 것(CSV=BOM만, XLSX=빈 시트, PDF=안내문). 사용자에게 보이는 동작 변경이다.
- 우려 2 — pdfExportOfAllHiddenColumnsIsStillAValidPdf 는 수정 전에도 통과한다(%PDF- 매직만 단언). javadoc 이 주장하는 범위보다 약하니 다음 회차에 본문 단언 추가 권장. 그리고 columnsJson 파싱 실패 시의 SELECT * fallback 은 그대로 남아 있다(원장 2/2/S).
- 우려 3 — 주과제인 PR #37 CI 39초 실패의 원인은 미규명이며 이 PR 로 해결되지 않는다. 재발 시 다음 행동은 코드 수정이 아니라 `gh run view --log-failed` 저장. 또 이 브랜치는 #37 의 0421f6d 를 cherry-pick 했으므로 #37 이 열려 있으면 머지 순서를 확인할 것.
- [러너 23:20] review approved — 리뷰 승인 (risk=low)
- [러너 23:21] pr created — https://github.com/hkjang/nexabuilder/pull/38
- [러너 23:21] ci failed — 성공이 아닌 검사: test + bootJar=failure
