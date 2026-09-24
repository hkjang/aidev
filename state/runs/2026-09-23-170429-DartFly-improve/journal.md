# 회차 노트 2026-09-23-170429-DartFly-improve — DartFly
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:04] base pinned — main@d1d3682
- [러너 17:04] autonomy release — 

## 정찰 노트
- 저장 결과 목록의 200건 벽을 골랐다: 서버(Service.List·listSavedResultsHandler)가 이미 limit/offset/total 을 주고 admin-history.js 에 베낄 pager 관례가 있어 화면만 고치면 되는데, 지금은 201번째부터 존재하지 않고 카운트도 total 이 아니라 필터된 수라 사용자가 "이게 전부"로 오인한다. 429 카운트다운·SSO 깜빡임은 가치 2, 401 헬퍼 일원화는 동작 변화가 없어(운영자 지침) 기각했다.
- 보류 항목 "드라이버 오류 원문 노출"은 코드 확인 결과 이미 해결이라 rejected 로 내렸다 — writeProblem 이 safeProblemDetail 로 한 곳에서 거른다. 구현자가 이걸 다시 고치지 않게 과제서 말미에 적어 뒀다.
- 추측으로 적은 것: pager 를 목록 밖에 두라는 권고(테스트 대역의 querySelectorAll 가정 때문)와, 페이지를 넘겼을 때 상세를 남기자는 제안은 설계 판단이지 검증된 사실이 아니다. saved.test.mjs 의 초기 stub 에 total/offset 을 넣어야 하는 점도 코드를 읽고 추론한 것이다.
- 이번 회차에 테스트는 한 번도 돌리지 못했다(node --test 실행이 승인되지 않음). 기존 11개 회귀가 지금 green 인지 구현자가 먼저 확인하고 시작할 것.
- 조심할 것: 직전 두 회차가 고친 detailRequest 순번·삭제 가드(5cd5ca0)와 tsvCell 계약(e6bd782)을 페이지 이동을 넣다가 되돌리기 쉽다. 화면 변경이므로 DF_SMOKE_REQUIRE_BROWSER=1 스모크와 실제 바이너리 클릭 확인까지 해야 과거 회차와 같은 수준의 증거가 된다(vm 대역만으로는 부족).
- [러너 17:07] scout done — 저장 결과 목록의 200건 벽 제거 — 서버가 이미 주는 total/offset 으로 페이지 이동을 붙이고, 검색 범위를 �

## 구현 노트
- 무엇을/왜: `/saved` 가 `limit=200` 한 번으로 끝나 201번째부터 화면에 없던 것을, 서버가 이미 주는 `limit/offset/total` 로 나눠 부르고 admin-history 의 pager 관례를 `#saved-pager`(목록 밖)에 붙여 고쳤습니다. 카운트는 서버 total 기준, 검색은 "이 페이지에서 N건"으로 범위를 밝힙니다. 서버·SQL 무변경, 커밋 bad8c9a.
- 확신 없는 곳: (1) PAGE 를 200→100 으로 낮췄습니다(서버 기본값·admin-history 와 일치). 한 화면에 보이던 건수가 줄어드는 사용자 체감 변화라 반대 의견이 있을 수 있습니다. (2) 한 페이지뿐이면 pager 를 숨기는 규칙(`total <= PAGE && offset === 0`)은 admin-history(항상 표시)와 다릅니다 — 좁은 목록 패널의 잡음을 줄이려는 판단입니다. (3) 삭제 직후 로컬 필터로 카드를 지울 때 `total` 을 1 줄이지만 pager 는 다시 그리지 않아, 새로고침 응답이 오기 전 수백 ms 동안 "페이지 n / m" 이 한 박자 낡습니다.
- 일부러 안 한 것: 서버 검색(`q`)은 손대지 않았습니다 — `savedSelect` 의 권한 조건(admin OR user_seq=?)을 건드리게 되어 이번 화면 전용 과제의 위험을 넘습니다. 상세 경합 로직(detailRequest·deleting 가드)과 tsvCell 계약은 그대로 뒀고, 페이지를 넘겨도 열려 있던 상세는 남기고 강조만 사라지게 했습니다(테스트로 못 박음).
- 다음 역할이 조심할 것: CSP 가 `style-src 'self'` 라 HTML 인라인 `style` 속성은 막힙니다 — pager 스타일은 반드시 JS(`style.cssText`)로 줘야 합니다. 실제로 vm 대역은 CSS 를 모르므로 "인라인 display:flex 가 hidden 을 이기는" 결함을 못 봤고 Chromium 에서만 드러났습니다(지금은 단위 테스트에도 고정). saved.test.mjs 는 node 만 있으면 돌지만(`node --test test/js/*.test.mjs`, 디렉터리 인자는 Node 22 에서 실패), 배선 증거는 `DF_SMOKE_REQUIRE_BROWSER=1 bash test/smoke/run.sh`(Docker+Chromium 필요, 메타 DB 가 TLS 시각 flake 로 첫 기동에 한 번 실패해 재실행함) 쪽입니다.
- [러너 17:23] brief accepted — 채택 — 과제서의 근거(saved.js:23 의 limit=200 고정, 서버의 limit/offset/total 지원, admin-history 관례, 하네스의 querySelectorAll 대역
- [러너 17:23] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 확인함: main 의 saved.js 로 되돌려 새 테스트 5개가 모두 fail, 새 코드에서 16/16 pass — 테스트가 변경을 실제로 못 박습니다. 서버(resultsave/service.go, resultsave_repository.go)는 limit/offset/total 을 이미 주고 COUNT 도 목록과 같은 (admin OR user_seq=?) 술어라 total 노출이 권한을 넓히지 않습니다. ORDER BY r.seq DESC 로 페이지 경계도 안정적. 보안·법무 차단 사유 없음.
- 못 봄: DF_SMOKE_REQUIRE_BROWSER=1 스모크와 실제 바이너리 클릭은 재실행하지 않았습니다(Docker·Chromium). pager 의 hidden+display:none 이중 처리는 단위 테스트 단언으로만 교차 확인했습니다.
- 승인이어도 남는 우려 (1) 마지막 페이지의 유일 항목 삭제 직후 loadList 응답 전까지 카운트가 '101-100 / 총 100건' 으로 뒤집혀 보입니다(saved.js:160-166, 구현자가 적은 (3)번 자리). 표시 전용이고 응답으로 교정됩니다.
- 승인이어도 남는 우려 (2) pager 클릭에 detailRequest 같은 순번 가드가 없어 연타 시 나중 응답이 offset 을 되돌릴 수 있습니다(saved.js:85-88). items 와 offset 이 같은 응답에서 오므로 표시는 자기모순이 없고, admin-history.js 도 같은 패턴이라 신규 결함은 아닙니다.
- 릴리즈 노트: 페이지 크기가 200→100 으로 바뀐 점과 '검색은 현재 페이지 대상' 임을 적어 주세요. 다음 회차 후보는 서버 측 q 검색(savedSelect 권한 조건을 건드리므로 별도 과제).
- [러너 17:26] review approved — 리뷰 승인 (risk=low)
- [러너 17:26] pr created — https://github.com/hkjang/DartFly/pull/12
- [러너 17:31] ci passed — 검사 3개 모두 success
- [러너 17:31] merge done — bad8c9a
- [러너 17:37] release published — v2.73.0
- [러너 17:37] gh-release created — GitHub Release v2.73.0
- [러너 17:37] manifest ok — dartfly-v2.73.0.tar.gz dartfly-v2.73.0.tar.gz.sha256 
- [러너 17:37] assets uploaded — 2개
- [러너 17:37] assets verified — v2.73.0 자산 2개 (이전 v2.72.0: 2)
