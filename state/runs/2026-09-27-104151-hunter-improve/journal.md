# 회차 노트 2026-09-27-104151-hunter-improve — hunter
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:41] base pinned — main@1911952
- [러너 10:41] autonomy release — 

## 정찰 노트
- 고른 이유: 유일하게 "선행 머지·외부 환경·DB 없이 이번 세션에 끝나고 파일 3개로 닫히는" 후보였다. mail·trackingSnippetOrigins 의존 후보 2건은 main@1911952 에 선행 코드가 여전히 없어 rejected 로 내렸고, tracking/CSP 계열은 2026-09-25 두 회차(verify-failed·기각)의 반복이라 회피했다. DB 필요한 CSV 권한 회귀는 샌드박스에 PostgreSQL 준비가 없어 뒤로 미뤘다.
- 이전 원장의 전제를 하나 고쳤다: `save()` 는 "이름·8개·중복만" 검사하지 않는다 — `list-tools.tsx:144-150` 에 q/필터 각 500자 검사가 이미 있다. 그래도 8192 한도 초과 경로는 남는다.
- 추측으로 적은 것: 9,016자 계산은 손계산이다(`node -e` 가 승인 차단). 한 화면의 필터 개수 상한도 확인하지 않았다. 구현자는 첫 단계로 `savedListQuery` 실측 길이를 찍어 8192 초과를 확정하고, 아니면 필터가 많은 실제 목록 화면으로 재현할 것.
- 조심할 것: `clip(500)` 과 `savedListQuery` 시그니처는 저장·복원·주소 복사 세 소비자가 공유한다 — 건드리지 말고 판정만 추가할 것. 가짜 view 객체로 인라인 `save()` 를 증명하지 말고 `savedListQuery`→`readListPreferences` 실제 왕복으로 증명할 것.
- 프로필은 갱신했다(VERSION 1.15.0→1.16.0, 테스트 기준선 93→96, npm 차단 함정 추가).
- [러너 10:45] scout done — 저장한 목록 보기가 브라우저 저장소 한도(8192자)를 넘겨 새로고침 후 조용히 사라지는 것을 저장 시점에 

## 구현 노트
- 무엇을 왜: `readListPreferences` 의 `8192` 를 `savedListQueryLimit` 로 뽑고 순수 판정 `savedListQueryTooLong` 을 더해, `ListTools.save()` 가 이미 가진 `snapshot` 으로 같은 한도를 저장 시점에 거절하게 했다(한국어 오류, `setPreferences` 미호출). 읽기 쪽 `>` 판정·`clip(500)`·`savedListQuery`/`applySavedListQuery`/`listSharePath` 는 무변경. 프로덕션 2파일 + 테스트 1파일.
- 확신 없는 곳·검증 못 한 것: **`save()` 자체는 실행하지 못했다** — web/tests 에 DOM·React 하네스가 없어 인라인 분기의 배선(오류 문구가 실제 Mantine 오류 자리에 뜨는지, `setPreferences` 가 실제로 안 불리는지)은 코드 읽기로만 확인했다. 가짜 view 객체를 만드는 대신 판정 함수를 `savedListQuery`→`readListPreferences` 실제 왕복으로 증명했다(9016자 실측, views:[] 재현). 브라우저에서 실제 보기 저장을 눌러 본 적은 없다.
- 일부러 하지 않은 것: 보기 8개 합산 localStorage 쿼터 검사(범위 확대, ideas.json 에 별도 항목), `clip(500)` 근처 손대기(세 소비자 공유), `save()` 의 나머지 세 판정 순수 함수화(후속 항목으로만 기록).
- 다음 역할이 조심할 것: Go 무변경이라 Go 스위트·`internal/webassets/dist` 재복사는 하지 않았다(dist 는 `.gitkeep` 만 추적). 이 회차 검증은 Node 22.23.1 에서 돌았고 프로젝트 요구는 Node 26 — CI 에서 다시 확인할 것. 기준선 96 → 현재 **97 통과 / 0 실패 / 0 skip**.
- [러너 10:50] brief accepted — 채택 — 지목한 파일·행·`8192` 단일 출처·`snapshot` 재사용 지점이 현재 코드와 정확히 맞았고, 미확인으로 남긴 9,016자 손
- [러너 10:50] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인한 것: diff 3파일 전문, `save()` 배선(같은 `snapshot` 재사용·`TextInput error` 렌더·`setPreferences` 앞 return), 읽기/저장 경계 일치(둘 다 `> 8192`), `8192` 단일 출처화(다른 8192는 무관), `maxLength={60}`이 이름 쪽 동일 계열 버그를 이미 막음. 실행: web test 97/0/0, `tsc --noEmit` 통과, snapshot 9016자 실측, main에 `savedListQueryTooLong` 0건(새 테스트 main에서 통과 불가).
- 못 본 것: 브라우저 실제 저장 클릭, Node 26 환경, Go 스위트·verify-pentagi(Go 무변경이라 생략).
- 승인이어도 남는 우려 ①: 주석·커밋 제목이 8192를 "브라우저 저장소" 한도라 하지만 실제로는 `readListPreferences` 의 자체 검증값이다(localStorage는 origin당 ~5MB, 값당 8192자 제한 없음). 동작은 맞으나 릴리즈 노트는 "읽기 검증 한도"로 쓸 것.
- 우려 ②: 발동 경로가 좁다 — q 단독 최대 4502자, 실제 필터는 전부 Select(짧은 id)이므로 정상 UI로는 도달 불가하고 손으로 만든 URL에서만 발동. 정상 사용자 오차단 위험도 없어 반려 사유는 아니다.
- 우려 ③(기존 결함, 다음 회차 후속): `save()` 오류 경로가 stale `message` 를 지우지 않아 직전 성공 문구가 새 오류와 함께 남는다(`list-tools.tsx:338-342`). `save()` 인라인 분기 테스트 부재도 그대로 — 하네스 도입 없이는 계속 코드 읽기로만 확인된다.
- [러너 10:53] review approved — 리뷰 승인 (risk=low)
- [러너 10:53] pr created — https://github.com/hkjang/hunter/pull/13
- [러너 11:12] ci timeout — 제한 시간 안에 CI 완료를 확인하지 못함
