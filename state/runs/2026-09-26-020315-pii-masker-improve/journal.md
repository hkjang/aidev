# 회차 노트 2026-09-26-020315-pii-masker-improve — pii-masker
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:03] base pinned — main@1e68f03
- [러너 02:03] autonomy release — 

## 정찰 노트
- config 테스트 공백(차선)보다 mock 기본 URL 하드코딩을 골랐다: 후자는 실제 오동작(ADDR 변경 시 마스킹 전부 실패 + 8080의 다른 프로세스로 PII 오발송 가능)이고, App.Serve가 외부 리스너를 받아 config.Load→app.New→실제 리스너로 end-to-end 증명이 되기 때문이다. 대역 없이 프로덕션 배선으로 재현 가능한 쪽을 우선했다.
- 추측으로 적은 것: (1) 고치기 전 새 app 통합 테스트가 "반드시" 실패하는지는 CI 머신의 8080 점유 상태에 달려 있어(무엇이 떠 있으면 다른 실패 양상) BaseURL 포트 단언을 함께 두라고 적었다. (2) `[::]:7000` 같은 IPv6 와일드카드의 SplitHostPort 결과("::")는 코드로 확인하지 않았다 — 구현자가 확인할 것.
- 조심할 것: mock이 꺼진 경로의 기본 URL(config.go:115)과 upstage allow-host 검사는 건드리지 말 것. 포트를 닫았다 다시 여는 방식은 경합이 나므로 리스너를 열어 둔 채 ADDR에 넣을 것.
- 전체 테스트는 이번 세션에서 `go test -count=1 ./...` 한 번 돌려 전부 통과를 확인했다(코드 변경 없음). 프로필은 1일 전 것이 현재 코드와 일치해 새로 쓰지 않았다.
- [러너 02:06] scout done — 내장 mock 기본 업스트림 URL이 `PII_MASKER_ADDR`의 실제 포트를 따르게 하기 (가치 3 / 위험 2 / 작업량 S)

## 구현 노트
- `config.Load`가 mock 기본 업스트림을 `localhost:8080`으로 하드코딩해 `PII_MASKER_ADDR`를 바꾸면 자기 자신이 아닌 포트로 PII를 보내던 것을, ADDR를 Load 맨 위에서 한 번 읽어 새 헬퍼 `localMockBaseURL`로 조립하도록 고쳤다(빈 호스트·`0.0.0.0`·`::` → `localhost`, SplitHostPort 실패 시 기존 `localhost:8080`).
- 정찰이 확신 없다던 두 지점은 확인했다: `[::]:7000`의 SplitHostPort 호스트는 `::`가 맞고(테이블 테스트 통과), 통합 테스트는 8080 점유와 무관하게 BaseURL 포트 단언에서 먼저 실패한다(수정 되돌려 재확인).
- 확신 없는 곳: mock URL에 IPv6 리터럴이 들어가는 경로(`[::1]:7000`)는 config 단위 테스트로만 덮었고 실제 IPv6 리스너로 마스킹을 돌려 보지는 않았다. `net.JoinHostPort`가 대괄호를 붙이는 것은 확인했으나 upstage allow-host 검사와의 조합은 미검증. Docker/compose 경로도 실행하지 않았다(compose는 base URL을 명시하므로 이 분기를 타지 않는다).
- 일부러 하지 않은 것: mock이 꺼졌을 때의 기본값 `http://localhost:8080/inference`(외부 추론 서버 자리표시자)와 `internal/upstage`의 allow-host 검사는 과제서 지시대로 손대지 않았다. `docker-compose.yml`도 그대로다.
- 다음 역할이 조심할 것: 새 테스트 2종은 모두 `t.Setenv`를 쓰므로 `t.Parallel()`을 붙이면 안 된다. `internal/app`의 새 통합 테스트는 기존 `startServerWithUpstream` 헬퍼를 쓰지 않고 리스너를 열어 둔 채 직접 `config.Load`→`app.New`→`Serve`를 한다(헬퍼는 수정하지 않았다). 외부 네트워크나 DB는 필요 없다.
- [러너 02:10] brief accepted — 채택 — 과제서의 근거(`config.go:110-113` 하드코딩 대 `app.go:42`의 같은 mux 마운트)가 현재 코드와 정확히 일치했고, 수용 기�
- [러너 02:10] verify passed — 검증 3개 통과 (policy)

## 비평 노트
- 확인함: 새 테스트가 실제로 고정하는지(config.go:116 을 옛 하드코딩으로 되돌려 config 5케이스 + app 통합 테스트 FAIL 재현), 구현자가 미검증이라 적은 IPv6 리터럴 경로를 임시 테스트로 `[::1]:0` 실제 리스너에서 end-to-end 마스킹 200 + allow-host `[::1]` 통과까지 확인(임시 파일 삭제, 트리 clean), `0.0.0.0:port`·`:port` 도 동일하게 성공.
- 못 본 것: Docker/compose 실제 실행(둘 다 base URL 을 명시하므로 이 분기를 타지 않음), localhost 가 ::1 로만 해석되는 호스트에서의 0.0.0.0 바인드 조합.
- 승인이어도 남는 우려(릴리즈 노트용): `PII_MASKER_ADDR=:http` 같은 서비스명 포트는 net.Listen 은 받지만 이제 config.Load 가 `invalid port ":http"` 로 기동을 막는다(이전엔 기동은 됐다). `:0` 이면 base URL 포트가 0 이 된다 — 둘 다 기존에도 깨져 있던 구성이라 회귀는 아니지만 에러 양상이 바뀌었다.
- README:111 은 "host:port 가 아닌 주소 → localhost:8080 유지" 분기를 적지 않았다. 다음 회차가 이 헬퍼를 건드리면 한 문장 보태면 된다.
- 검증 재현: gofmt/vet 무출력, `go test -count=1 ./...` 전체 통과, `go test -race -count=2 ./internal/{app,config,httpapi,service}` 통과.
- [러너 02:13] review approved — 리뷰 승인 (risk=low)
- [러너 02:13] pr created — https://github.com/hkjang/pii-masker/pull/26
- [러너 02:14] ci passed — 검사 없음 — 정책으로 허용
- [러너 02:14] merge done — 89597ce
