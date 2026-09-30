# 회차 노트 2026-10-01-014241-DartFly-improve — DartFly
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:42] base pinned — main@8591080
- [러너 01:42] autonomy release — 

## 정찰 노트
- 고른 이유: 보류 1순위였던 '저장된 결과 탭 100건 벽' 을 지금 코드로 다시 보니 결함이 하나 더 있었습니다 — `loadSaved`(admin-history.js:71)가 `searchEl.value` 를 아예 읽지 않아 검색창이 무반응입니다(탭 전환 시 placeholder 는 'SQL 검색...' 으로 남음). 눈에 보이는 결함 둘을 프로덕션 파일 1개로 고칠 수 있고, `/saved`(bad8c9a)·실행 감사 탭(68a51eb)의 관례와 하네스 헬퍼(`pager()`/`pagerButton()`)가 그대로 재사용됩니다.
- 제친 후보: 'history.go limit·offset 무검증' 은 서비스 계층이 이미 클램프해 결함이 아니었습니다(ideas.json 에서 done 으로 내림). '스키마·데이터 탭 200건 상한 표시' 는 서버가 total 을 안 줘 표현이 애매해 차선으로 미뤘고, 'saved 서버 검색(q)' 은 권한 조건을 건드려 L·위험 3 입니다.
- 추측으로 적은 것(미확인): 실제 브라우저에서의 동작과 baseline green 여부 — 이 세션은 샌드박스 승인 요구로 `node`/`go test` 를 한 번도 실행하지 못했습니다. `execPager` 를 일반화할지 `savedPager` 를 따로 둘지도 구현자 판단으로 남겼습니다.
- 조심할 것: `empty()` 가 `listEl` 을 통째로 덮어 빈 페이지엔 이동 바가 남지 않으므로, `/saved` 식 '한 페이지 물러나기' 가 아니라 서버 `total` 로 마지막 페이지를 계산해야 합니다(2026-09-28 회차가 실행 감사에서 겪은 차이). 이동 단추는 반드시 `load()` 를 거쳐 `listRequest` 순번을 올릴 것. HTML 인라인 style 은 CSP 가 막습니다.
- [러너 01:47] scout done — 관리자 변경 이력 화면 **'저장된 결과' 탭**의 100건 벽과 무반응 검색창 — 실행 감사 탭·`/saved` 와 같은 �

## 구현 노트
- 무엇을 왜: `loadSaved`(admin-history.js)가 `limit=100` 한 번 + `page.total` 표시라 101번째부터 닿을 수 없었고 `searchEl.value` 를 읽지 않아 검색창이 무반응이었습니다. 서버가 이미 주는 limit/offset/total 로 실행 감사 탭·`/saved` 의 이동 바·카운트·페이지내 검색 문구를 옮겼습니다(커밋 a772071, 프로덕션 파일 1개, 서버·SQL 무변경).
- 확신 없는 곳·검증 못 한 것: (1) 정찰이 제안한 대로 `execPager` 를 일반화했는데, offset 을 **그려질 때의 at** 기준으로 더하면 기존 회귀 테스트 3번이 깨집니다(응답 대기 중 `다음 →` 두 번 = 같은 페이지 재요청). 그래서 `go(±PAGE)` 로 살아 있는 offset 에서 움직입니다 — 이 부분을 먼저 봐 주세요. (2) **기존 테스트 10 은 그대로 green 일 수 없었습니다**: 저장 탭 URL `?limit=100` 과 카운트 `1건` 을 문자열로 못박고 있어, 이번 변경이 바꾸는 바로 그 값을 `?limit=100&offset=0`·`1-1 / 총 1건` 으로 고쳤습니다(단정 약화가 아니라 새 계약으로 이동). (3) 두 탭이 `savedOffset`/`execOffset` 을 따로 두므로 탭을 옮겼다 돌아오면 첫 장부터입니다 — 의도한 동작이지만 사용성 판단은 열려 있습니다.
- 일부러 하지 않은 것: 서버에 `q` 검색 추가(권한 조건 `admin OR user_seq=?` 를 손대야 해 범위 밖 — 대신 `· 이 페이지에서 N건` 으로 임시 해법임을 화면에 드러냄), 스키마·데이터 탭의 200건 상한 표시(서버가 total 을 주지 않아 성격이 다름 — ideas.json 에 pending).
- 다음 역할이 조심할 것: 실제 바이너리 검증은 Docker(MariaDB) + Chromium 이 필요하고, **이 환경은 HOME 이 임시라 `PLAYWRIGHT_BROWSERS_PATH=/home/hkjang/.cache/ms-playwright` 를 주지 않으면 run.sh 의 브라우저 단계만 트레이스백으로 실패합니다**(비브라우저 검사는 전부 통과했고, 따로 주고 돌리면 pages.py 31페이지 0개·editor.py 통과). `go test -v -run TestBrowserModuleTests` 는 Node 없으면 조용히 Skip 이므로 출력으로 PASS 를 확인하세요.
- [러너 01:59] brief accepted — 채택 — 지목한 네 자리(limit=100 고정, page.total 표시, searchEl 무시, placeholder 삼항)와 전제(서버의 limit/offset/total, `empty()` 가 
- [러너 01:59] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 확인함: main@8591080 을 별도 worktree 로 꺼내 **새 테스트 파일만** 얹어 돌려 새 테스트 7개 + 계약이 바뀐 기존 10 이 base 에서 not ok(8/18 실패), HEAD 에서는 18/18 green 임을 직접 재현했습니다(원장에 `- 실패 재현:` 줄이 없었음). gofmt·vet·`go test ./...`·`go build` 전부 green, `TestBrowserModuleTests` 는 Skip 아니라 실제 PASS.
- 구현자가 지목한 `go(±PAGE)` 는 정당합니다 — 렌더 시점 `at` 기준이면 page 2 에서 `다음 →` 후 같은 바의 `← 이전` 이 page 1 로 튑니다. `execPager`→`pagerBar` 일반화는 실행 감사 탭 동작 동일(잔여 참조 0건), 기존 테스트 3·5·8 이 그 경로를 덮습니다.
- 못 본 것: 실제 브라우저 스모크(Docker+Chromium) 미실행. 서버·라우팅·임베드 목록 무변경이라 위험 낮다고 판단했으나 릴리즈에서 한 번 돌리길 권합니다. 러너의 '**ts** 파일 변경' 메모는 오탐(.ts 변경 0건).
- 승인이어도 남는 우려(릴리스 노트에 솔직히 적을 것): 저장 탭 검색은 `input` 이 `savedOffset=0` 으로 되돌려 **사실상 1페이지 안에서만** 돌고, 검색어를 유지한 채 넘어간 페이지에 매칭이 없으면 `empty()` 가 이동 바를 지웁니다(검색창을 고쳐 1페이지로 튕기는 탈출구는 있음). 100건 넘는 저장본에서 검색은 반쪽 — 서버 `q` 를 다음 회차 후보로.
- 보안·법무 차단 없음: 새 엔드포인트·인가 변경 없고 권한은 `(? OR r.user_seq = ?)` 로 서버에서, limit/offset 은 바인드 파라미터 + `<0` 클램프. 새 개인정보 수집·전송 없고 검색어는 서버로 가지 않습니다.
- [러너 02:04] review approved — 리뷰 승인 (risk=low)
- [러너 02:04] pr created — https://github.com/hkjang/DartFly/pull/18
- [러너 02:05] ci failed — 성공이 아닌 검사: test=failure, livedb=failure, smoke=failure
