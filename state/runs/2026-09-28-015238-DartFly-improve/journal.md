# 회차 노트 2026-09-28-015238-DartFly-improve — DartFly
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:52] base pinned — main@706c8ee
- [러너 01:52] autonomy release — 

## 정찰 노트
- 선택 이유: admin-history.js 실행 감사 탭에 saved.js 가 이미 고친 두 결함(요청 순번 없음 / 빈 페이지에서 pager 없이 early return)이 그대로 남아 있고, 고칠 패턴·테스트 하네스가 둘 다 검증돼 있어 한 세션에 맞습니다. 프로덕션 파일 1개 + 신규 테스트 1개.
- 제친 후보: 'history.go limit·offset 무검증'(보류 1순위)은 **기각** — internal/history/service.go:80,94,101-106 이 이미 클램프하고 있어 고칠 것이 없습니다(이전 정찰의 사실 오류). 401 헬퍼 일원화는 동작 변화가 없어 차선으로만 남겼습니다.
- 추측으로 적은 것: 늦은 응답 덮어쓰기는 코드를 읽고 판정했을 뿐 브라우저에서 재현하지 못했습니다(경합이라 타이밍 필요). 빈 페이지 갇힘은 loadExecutions:108 의 early return 으로 코드상 확정입니다.
- **baseline green 미확인** — 샌드박스 권한으로 node·go test 를 한 번도 돌리지 못했습니다(2026-09-26 회차와 같은 증상). 구현자가 먼저 기존 회귀를 돌려 출발점을 잡으세요.
- 조심할 것: pager 클릭 핸들러(126·128행)가 load() 를 거치지 않고 loadExecutions() 를 직접 부르므로 순번 갱신을 여기서 빠뜨리면 절반만 고쳐집니다. 가짜 Element 에 showModal/close 를 더하지 않으면 테스트가 바로 죽습니다.
- [러너 01:56] scout done — 관리자 감사 화면(실행 감사 탭)의 늦은 응답 덮어쓰기와 빈 페이지 갇힘 수정 (가치 3 / 위험 2 / 작업량 M)

## 구현 노트
- `admin-history.js` 한 파일만 고쳤습니다(커밋 68a51eb): `load()` 가 요청마다 순번을 매기고 각 로더가 응답을 그리기 전에 최신 순번인지 확인 — 늦은 응답/실패가 새 탭·새 페이지를 덮지 못합니다. pager 클릭도 `load()` 를 거칩니다. 빈 실행 감사 페이지는 서버 `total` 로 마지막 페이지를 계산해 한 번만 재요청하고, 카운트는 `items` 가 비면 0 으로 떨어집니다.
- **과제서와 다르게 한 것**: saved.js 식 '한 페이지 물러나기' 로는 부족했습니다. saved.js 는 pager 가 별도 `#saved-pager` 라 빈 페이지에도 이동 바가 남지만, admin-history 는 `empty()` 가 목록을 통째로 덮으므로 여러 페이지가 한꺼번에 사라지면 물러난 페이지도 비어 그대로 갇힙니다(브라우저에서 offset 300→200 으로 확인). 그래서 total 기반 클램프로 바꿨고 그 차이를 테스트 5번에 못박았습니다.
- **확신 없는 곳**: `execOffset = page.offset`(서버 값 채택)은 서버가 요청 offset 을 그대로 되돌려주는 현재 동작에 기댑니다. 서버가 클램프해 다른 값을 주면 화면이 그 값을 따라갑니다 — 의도한 동작이지만 실제로 그런 응답은 만들어 보지 못했습니다.
- **일부러 안 한 것**: 저장결과/스키마/데이터 탭에 pager 붙이기(범위 확대 — ideas.json 에 남김), 서버 변경, `viewSaved`/`showSource` 다이얼로그의 순번 가드(목록 경합과 무관).
- 다음 역할이 조심할 것: `node test/js/admin-history.test.mjs` 는 DB·브라우저 없이 돕니다(검색 디바운스 때문에 ~0.4초). 브라우저 재검증은 `test/smoke/run.sh --keep` 뒤 감사 기록을 직접 심어야 합니다.
- **하네스 함정 재발**: `test/smoke/run.sh` 의 `pkill -f "$BIN"` 이 내 셸 명령줄에 걸려 exit 144 로 두 번 죽었습니다. 명령줄에 `/tmp/dartfly-smoke` 문자열이 있으면(heredoc 안이든 `go build -o` 인수든) 걸립니다 — 런처를 별도 파일로 빼고, 그 파일을 만드는 호출과 실행하는 호출을 **나누어야** 합니다.
- [러너 02:34] brief accepted — 채택 — 지목한 네 결함(순번 없음, 빈 페이지 early return, 역전 카운트, pager 가 load() 를 우회)과 하네스 전제(glob 자동 배선
- [러너 02:34] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 확인: main 파일로 되돌려 새 테스트를 돌려 6개 red(원장의 `실패 재현:` 출력과 일치) → 변경 후 11개 green, `go test -run TestBrowserModuleTests -v ./internal/webui/` 에서 Skip 없이 PASS. 러너의 'ts 미검증' 경고는 .js/.mjs 언어 오분류로, 제가 직접 실행해 메웠습니다.
- 구현자가 의심한 `execOffset = page.offset` 은 서버를 읽어 안전 확인 — history.go:62 가 offset 을 그대로 넘기고 service.go:101-106 은 음수만 0 으로 깎으며, 클라이언트는 음수를 보내지 않습니다.
- 승인이어도 남는 우려(릴리즈 노트에 과장 금지): total 은 세는 쿼리와 읽는 쿼리가 별도 문장(history_repository.go:92,113)이라 'total 은 남았는데 그 페이지만 빈' 응답에서는 admin-history.js:118 의 `last < execOffset` 조건이 안 걸려 여전히 갇힐 수 있고, 그 경로의 카운트가 `0-100 / 총 150건` 으로 나옵니다. 일시 상태라 차단하지 않았습니다.
- 보안·법무: 차단 없음. 신규 엔드포인트·의존성·비밀값·암호 코드 없음, 서버 인가(historyReady→requireAdmin) 무변경, 개인정보 수집·보존 범위 변화 없음.
- 다음 회차 후보: 테스트 4번 제목('steps back one page')을 실제 동작(total 기반 클램프)에 맞게 고치기, 그리고 빈 응답에서도 이동 바를 남기는 쪽(empty() 가 pager 를 지우지 않게) 검토.
- [러너 02:40] review approved — 리뷰 승인 (risk=low)
- [러너 02:40] pr created — https://github.com/hkjang/DartFly/pull/16
- [러너 02:47] ci passed — 검사 3개 모두 success
- [러너 02:47] merge done — 68a51eb
- [러너 02:51] release published — v2.77.0
- [러너 02:51] gh-release created — GitHub Release v2.77.0
- [러너 02:51] manifest ok — dartfly-v2.77.0.tar.gz dartfly-v2.77.0.tar.gz.sha256 
- [러너 02:52] assets uploaded — 2개
- [러너 02:52] assets verified — v2.77.0 자산 2개 (이전 v2.76.0: 2)
