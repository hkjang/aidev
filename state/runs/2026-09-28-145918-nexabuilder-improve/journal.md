# 회차 노트 2026-09-28-145918-nexabuilder-improve — nexabuilder
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:59] base pinned — master@a36fd87
- [러너 14:59] autonomy release — 

## 정찰 노트
- 내보내기 truncation 을 골랐다: 런타임 대체 뷰 7종의 조용한 catch(3/2/M)는 템플릿 7개로 파일 규칙에 걸리고, permission 게이트는 보호 구역, listData 대문자 키는 `ListExportController.lookup` 이 이미 흡수하는 것을 소스로 확인해 기각했다. 이 과제는 프로덕션 파일 1개(`readQueryParams`)로 끝나고 관찰 가능한 데이터 손실이다.
- 소스로 확인한 것: `downloadExport`(list.html:1039)가 `filterValues()`(372, `data-filter-field` 입력만)만 담아 `_limit` 이 없다 / `readQueryParams` 가 쿼리스트링을 그대로 `listData` 로 넘긴다 / `listData:174-179` fallback 이 `page_size`→100 / `intOr:812` 가 Number·String 둘 다 처리 / `MAX_FEED_ROWS` 는 public.
- 추측으로 남긴 것: 실제 운영 목록의 `page_size` 분포(픽스처 기준만), 5000행 PDF 의 소요 시간. gradlew 는 이번 정찰에서 실행하지 않았다 — 실패 재현은 구현 단계에서 할 것.
- 구현자가 조심할 것: `listData` 의 fallback 을 바꾸면 런타임 그리드가 매 요청 전체 표를 읽는다 — 내보내기 쪽 기본값만 건드릴 것. `MAX_FEED_ROWS`(5000) 상한은 1b265b2 의 의도적 방어이므로 유지하고, 5000행 초과 잘림은 별도 과제로 ideas.json 에 갈라 뒀다.
- 같은 파일(`ListExportController`)의 OpenPDF deprecated 정리는 충돌을 피해 다음 회차로 미뤘다.
- [러너 15:05] scout done — CSV/XLSX/PDF 내보내기가 그리드 한 페이지(기본 100행)만 내려주고 조용히 잘리는 것을 고친다 (가치 4 / 위험 

## 구현 노트
- `ListExportController.readQueryParams` 에서 `_limit` 키가 없을 때만 `NexaUiService.MAX_FEED_ROWS`(5000)를 `putIfAbsent` 로 채웠다. 프로덕션 파일 1개·한 줄 + javadoc. 세 엔드포인트(CSV/XLSX/PDF)가 이 메서드 하나를 지나므로 기본값이 갈라지지 않는다.
- 확신 없는 곳: (1) 쿼리스트링에 `?_limit=`(빈 값)이 오면 `putIfAbsent` 가 발동하지 않고 `intOr("")` → 0 → `listData` 의 `page_size` fallback 으로 떨어진다. 잘못된 파라미터에 대한 **기존** 동작 그대로라 손대지 않았지만, 비평가가 계약 위반으로 볼 수도 있다. (2) 실제 운영 목록의 `page_size` 분포는 여전히 미확인(픽스처 기준만). (3) 5000행 PDF 의 실제 소요 시간 미측정 — 테스트 픽스처는 12행이다.
- 일부러 하지 않은 것: `X-Nexa-Export-Truncated` 헤더. `listData` 가 `RecordSet` 만 돌려줘 "상한에 딱 걸린 것" 과 "마침 정확히 5000행" 을 구분할 수 없고, 구분하려면 `limit+1` 을 읽거나 count 를 더 쳐야 해서 S 범위를 넘는다 — ideas.json 에 후속으로 남겼다. `listData` 의 `page_size` fallback·`MAX_FEED_ROWS` 상한·템플릿 JS·`_offset` 은 건드리지 않았다.
- `"_limit"` 는 리터럴로 썼다. `NexaUiService.PARAM_LIMIT` 은 private 이고, 같은 리터럴이 이미 `EntityCrudController`·`LookupController`·`OpenApiController` 에 흩어져 있어(기본값은 경로마다 50/20/5000) 이번 회차에 두 번째 파일을 여는 대신 ideas.json 에 판정 항목으로 남겼다.
- 다음 역할이 조심할 것: 테스트는 H2 가 붙은 `@SpringBootTest`+MockMvc 이고 목이 없다. 내보내기 테스트는 `sessionAttr("nexabuilder.user", …)` 가 필요하다. 기존 `seed()` 는 `seed(null, 1)` 로 위임하므로 기존 6건의 기대값은 변하지 않았다. 전체 605건(602+3) 통과·0 skip, `bootJar -x test` 성공. `gradlew` 는 100644 라 `sh ./gradlew` 로 실행해야 한다.
- [러너 15:20] brief accepted — 채택 — 진단(`readQueryParams` 가 `_limit` 을 넣지 않아 `listData` fallback 으로 떨어짐)과 수용 기준 4건이 실행으로 전부 확인됐�
- [러너 15:20] verify passed — 검증 1개 통과 (auto)

## 비평 노트
- 판정: approve / risk low / blocking 없음. 원장에 `- 실패 재현:` 줄이 없어 직접 재현했다 — `ListExportController.java:379` 의 putIfAbsent 한 줄만 제거하고 테스트 클래스를 돌려 csv `13→6`, xlsx `12→5` 실패(=픽스처 page_size 5)를 확인, 줄 복원 후 9/9 통과·`git status` clean. 증상이 이번 변경이 고치는 절단과 일치한다.
- `explicitQueryStringLimitWinsOverTheExportDefault` 는 수정 전에도 통과하는 가드 테스트다(검증력은 위 2건에 있다). 범위 이탈·인증/권한/마이그레이션 변경 없음: 프로덕션 diff 는 1줄 + javadoc 이고 `canView` 게이트와 `MAX_FEED_ROWS` 클램프는 그대로(클램프 테스트도 통과).
- 못 본 것: 실제 운영 목록의 `page_size` 분포와 5000행 PDF/XLSX 의 실측 소요 시간·메모리. 전체 605건은 재실행하지 않고 내보내기 클래스만 돌렸다.
- 승인이어도 남는 우려(릴리즈 노트용): ① 내보내기 기본 비용 100행→최대 5000행, PDF 는 표 전체를 메모리에 만든다 ② `?_limit=`(빈 값)은 여전히 page_size 로 절단되고 javadoc 문구와 어긋난다 ③ 5000 초과분은 아직 무신호 절단 ④ 성공한 내보내기는 감사 로그에 남지 않는데 다운로드당 개인정보 행수가 ~50배가 됐다.
- 다음 회차 후보: `PARAM_LIMIT` 를 public 으로 올려 `"_limit"` 리터럴 4개 사용처를 모으기(기본값이 50/20/5000/5000 로 갈라져 있다).
- [러너 15:28] review approved — 리뷰 승인 (risk=low)
- [러너 15:28] pr created — https://github.com/hkjang/nexabuilder/pull/35
- [러너 15:37] ci passed — 검사 2개 모두 success
- [러너 15:37] merge done — ff96bd8
- [러너 15:57] release published — v1.26.0
- [러너 15:58] assets verified — v1.26.0 자산 1개 (이전 v1.25.0: 1)
