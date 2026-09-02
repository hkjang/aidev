# moyro 자율 개선 기록

## 2026-09-02
- 선택: 링크 프리뷰 SSRF 가드의 DNS 리바인딩 취약점 수정 및 links 패키지 첫 테스트 추가 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `server/internal/links`의 `safeDialContext`가 호스트를 조회해 모든 응답 주소를 검사한 뒤 원래 호스트명을 `net.Dialer`에 그대로 넘겨 DNS를 두 번 해석했다. 리바인딩 응답이 검사 시점에는 공인 IP, 연결 시점에는 127.0.0.1을 반환하면 인증 사용자용 `/api/v4/link_preview_image` 프록시가 루프백·사설망으로 유도될 수 있었으므로, 검증한 주소로 직접 다이얼하도록 `dialTargets`를 분리하고 CGNAT·192.0.0.0/24·멀티캐스트·예약 대역을 차단 목록에 추가했다. 테스트가 하나도 없던 패키지에 URL 추출, OpenGraph 파싱, 주소 정책, 주소 고정, 캐시 축출 테스트 12개를 추가했고 `go vet ./...`, `go test -race ./...`(44 패키지 통과), `scripts/check-source-sizes.sh`로 검증했다. 웹 변경이 없어 webapp 빌드는 손대지 않았다.
- 보류 아이디어: (1) 로드맵의 create-post 인가·멤버십 2회 쿼리를 단일 쿼리로 병합 — PostgreSQL 통합 테스트 환경이 필요해 보류. (2) `ratelimit.Limiter.Middleware`가 JSON 본문을 `http.Error`로 써서 Content-Type이 text/plain으로 나가는 문제. (3) 테스트가 전혀 없는 `invites`/`sidebar`/`userstatus`/`postacks` 패키지의 단위 테스트 보강. (4) 로드맵의 메시지 목록 가상화(로드된 모든 행이 DOM에 남음) — 작업량 L이라 단일 세션 범위 초과.
- 릴리즈: v0.2.10 (2026-09-02)

## 2026-09-02
- 선택: @멘션 추출을 Mattermost 규칙에 맞춰 수정 (문장부호·대소문자·이메일 오탐) (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `postcommand.ExtractMentions`가 정규식 `@([a-zA-Z0-9._-]+)` 결과를 그대로 사용자명 조회에 넘겨서, 문장 끝 멘션(`cc @alice.`)은 "alice."로 조회돼 아무도 알림을 받지 못했고 `@Alice` 같은 대소문자 차이도 registration이 소문자로 저장하는 사용자명과 어긋나 실패했다. 반대로 이메일 주소(`ops@example.com`)는 "example.com"이라는 유령 후보를 만들어 매 게시마다 불필요한 조회를 유발했다. 정규식 앞에 `\B`와 영숫자 시작 조건을 추가해 단어 중간 `@`를 제외하고, 후보에 뒤쪽 `._-`를 제거한 형태와 소문자 형태를 함께 실어 보내며 후보 수를 200개로 제한했다. `mentions_test.go`에 표 기반 테스트 8건과 상한 테스트를, `service_test.go`에 종단 해석 테스트 1건을 추가했고 `go vet ./...`, `go test -race ./...`(전 패키지 통과), `scripts/check-source-sizes.sh`로 검증했다. 웹 변경이 없어 webapp 빌드는 손대지 않았다.
- 보류 아이디어: (1) `ratelimit.Limiter.Middleware`가 JSON 본문을 `http.Error`로 써서 Content-Type이 text/plain으로 나가는 문제. (2) 테스트가 전혀 없는 `invites`/`sidebar`/`userstatus`/`postacks` 패키지의 단위 테스트 보강 — PostgreSQL 통합 환경 필요. (3) 로드맵의 create-post 인가·멤버십 2회 쿼리를 단일 쿼리로 병합. (4) 로드맵의 메시지 목록 가상화 — 작업량 L이라 단일 세션 범위 초과.
- 릴리즈: v0.2.11 (2026-09-02)

## 2026-09-02
- 선택: 429 응답을 JSON Content-Type과 실제 대기 시간으로 교정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `ratelimit.Limiter.Middleware`가 JSON 문자열을 `http.Error`로 내보내 Content-Type이 text/plain으로 나갔고, 버킷 속도와 무관하게 `Retry-After: 1`을 고정 반환했다. 회원가입 버킷은 5초에 토큰 하나를 채우므로 거절된 클라이언트가 4초 일찍 재시도해 헛된 거절을 네 번 더 받는 구조였다. `take()`가 판정과 함께 버킷 상태를 돌려주도록 나눠 남은 토큰과 설정된 rate 기반 대기 시간을 계산하고, 응답에 공용 API 오류 봉투와 Mattermost 호환 `X-Ratelimit-Limit/Remaining/Reset` 헤더를 실었다. 또 버킷 정리(GC)를 판정보다 앞으로 옮겨 계속 throttle 상태인 키가 다른 버킷을 메모리에 붙잡아두지 못하게 했다. 미들웨어·봉투·재시도 지연·빈 키 우회·정리 테스트 6건을 추가했고 `go vet ./...`, `go test -race ./...`(전 패키지 통과), `scripts/check-source-sizes.sh`로 검증했다. 웹 변경이 없어 webapp 빌드는 손대지 않았다.
- 보류 아이디어: (1) 테스트가 전혀 없는 `invites`/`sidebar`/`userstatus`/`postacks` 패키지의 단위 테스트 보강 — 대부분 DB 경로라 PostgreSQL 통합 환경 필요. (2) `invites.normalizeChannelIDs` 같은 순수 함수만 골라 단위 테스트 추가. (3) 로드맵의 create-post 인가·멤버십 2회 쿼리를 단일 쿼리로 병합. (4) 로드맵의 메시지 목록 가상화 — 작업량 L이라 단일 세션 범위 초과.
- 릴리즈: v0.2.12 (2026-09-02)

## 2026-09-03
- 선택: 커스텀 이모지 검색 누락·`emoji/names` N+1·삭제된 이름 재사용 불가 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `writeEmojiSearch`가 최신 200개만 가져와 Go에서 `strings.Contains`로 걸러서, 커스텀 이모지가 한 페이지를 넘는 워크스페이스에서는 오래된 이모지가 자동완성·검색 양쪽에서 영구히 사라졌다. 매칭을 DB로 옮기고 `LIKE` 대신 `strpos`를 써서 사용자가 입력한 `%`·`_`가 와일드카드로 새지 않게 했으며, 요청당 최대 200회 순차 쿼리를 돌던 `POST /emoji/names`는 `= ANY($1::text[])` 단일 쿼리로 바꾸고 요청 순서를 유지했다. 또 v0.1 베이스라인이 `emojis.name`에 테이블 전역 UNIQUE를 걸어둔 탓에 소프트 삭제된 이모지 이름을 다시 등록하면 라이브 전용 충돌 검사는 통과하고 업로드까지 끝난 뒤 INSERT만 실패해 이름이 영구히 잠기고 매 시도마다 고아 파일이 남았으므로, 마이그레이션 000017로 제약을 `delete_at=0` 부분 유니크 인덱스로 교체했다. 테스트가 없던 `emojis` 패키지에 통합 테스트 5건과 `store`에 마이그레이션 업그레이드 테스트 1건을 추가하고 CI PostgreSQL 잡 대상에 `./internal/emojis`를 넣었다. 로컬 postgres:16 컨테이너를 띄워 `go vet ./...`, `MOYRO_TEST_POSTGRES_DSN` 설정 후 `go test -race -p 1 ./...`(전 패키지 통과), `scripts/check-source-sizes.sh`로 검증했다. 웹 변경이 없어 webapp 빌드는 손대지 않았다.
- 보류 아이디어: (1) `emojis.Create`가 크기 초과 파일을 업로드한 뒤에야 거절해 고아 file_infos 행을 남기는 문제 — files.Service에 정리 경로가 필요. (2) 테스트가 전혀 없는 `invites`/`sidebar`/`userstatus`/`postacks` 패키지의 통합 테스트 보강. (3) 로드맵의 create-post 인가·멤버십 2회 쿼리를 단일 쿼리로 병합. (4) 로드맵의 메시지 목록 가상화 — 작업량 L이라 단일 세션 범위 초과.
- 릴리즈: v0.2.13 (2026-09-03)
