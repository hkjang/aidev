# 회차 노트 2026-10-05-133740-jikim-improve — jikim
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 13:37] base pinned — main@a949932
- [러너 13:37] autonomy release — 

## 정찰 노트
- 고른 이유: 이 저장소의 적중 광맥("원인 X 를 원인 Y 로 보고")에서 아직 열려 있는 사례를 찾았다 — oidc.go:121-125·252-256 이 OIDCConfig 읽기 실패를 oidc_disabled 로 접는다. 같은 파일 43-47·55-59 가 이미 ErrNotFound 와 저장소 장애를 구분하므로 "파일 자신의 관례에서 두 자리만 빠져 있다"는 근거가 코드로 선다. CanAccess 403 계열(secret_handlers.go:118·161·309, openbao.go:820)은 열어 보니 이미 분리되어 있어 기각했고, 문서 전용 차선(AI error code 표)은 관찰 가능한 동작 변화가 없어 밀었다.
- 확인한 것: base a949932 / `go test ./internal/httpapi/ -count=1` green 0.739s / store.OIDCConfig(settings.go:629-633)가 ErrNotFound 를 그대로 올림 / oidc_disabled·invalid_oidc_config 문자열이 web/src·docs·기존 테스트에 없음 / oidcCallback 252행이 네트워크 호출(261) 앞이라 oidcStateOpener seam 만으로 도달 가능.
- 추측(미확인): 차선 후보의 `ai_rate_limited` code 문자열은 ai_rate_limit.go 에서 직접 확인하지 않았다. 프로필은 1일 전 것이고 이번에 읽은 범위와 어긋나는 데가 없어 새로 쓰지 않았다.
- 구현자 주의: `errors.Is(err, store.ErrNotFound)` 를 빼면 설정 안 한 새 설치가 503 oidc_disabled 대신 404 not_found 를 받는다(errorStatus, server.go:439-453). 콜백 조건식에 IssuerURL·ClientID 검사를 새로 더하지 말 것. `New` 에서 seam 을 배선하지 말 것(nil-폴백 관례).
- [러너 13:43] scout done — OIDC 로그인 시작·콜백이 설정 읽기 실패를 "OIDC 로그인이 설정되지 않았습니다/비활성화되었습니다"로 보

## 구현 노트
- 무엇을: `oidcLogin`·`oidcCallback` 이 OIDC 설정 로더의 **모든** 오류를 `!cfg.Enabled` 와 한 조건으로 묶어 503 `oidc_disabled` 로 내보내던 것을, 같은 파일 `oidcPublicConfig`·`oidcTest` 가 이미 쓰는 `errors.Is(err, store.ErrNotFound)` 분리로 맞췄다(비-ErrNotFound → `s.storeError` → 500 `internal_error`). 커밋 `255e321`, 프로덕션 2파일 + 테스트 1파일.
- **과제서와 어긋난 것 하나 — 비평가가 먼저 볼 자리**: 과제서는 `oidcStateOpener` seam 만으로 콜백 252행에 닿는다고 했지만 244행이 그 seam 헬퍼(`s.openOIDCState`, oidc.go:366-371)를 우회해 `s.store.OpenOIDCState` 를 **직접** 부르고 있었다(저장소에서 이 헬퍼를 우회하는 유일한 자리). 헬퍼로 바꾸는 한 줄을 더했다 — nil 이면 같은 `s.store.OpenOIDCState` 로 접히므로 프로덕션 동작은 불변이지만, 과제서가 세지 않은 변경이다.
- 확신 없는 곳: 실제 PostgreSQL 장애·`oidc_client_secret` 복호화 실패를 이 환경에서 재현하지 못했다 — 500 경로는 주입한 `driverFailure`(`SQLSTATE 28P01`)로만 증명했고, `store.OIDCConfig` 가 복호화 실패 시 `ErrNotFound` 로 감싸지 **않는다**는 것은 settings.go:629-633 를 읽어 판단했을 뿐 실행으로 확인하지 않았다. 실제 Keycloak 왕복도 미검증(환경 제약).
- 일부러 하지 않은 것: `New` 에 seam 배선 안 함(nil-폴백 관례). 콜백 조건식에 `IssuerURL`·`ClientID` 검사 **안 더함**(그 뒤 `validateOIDCRuntimeConfig` 가 받는다 — 더하면 `invalid_oidc_config` 로 가던 설정이 `oidc_disabled` 로 바뀐다). `oidcExchange`(426-430)·`oidcLogout`(387-391)의 같은 결함 가족은 범위 밖으로 두고 ideas.json 에 남겼다.
- 다음 역할이 조심할 것: 새 테스트는 DB 없이 돈다(seam 주입 + `quietServer()`). 역방향 확인을 두 분기 **따로** 했으니 재확인할 때도 한 번에 둘 다 되돌리지 말 것 — 한쪽만 되돌리면 그쪽 테스트만 실패하는 것이 분리의 증거다. `bash scripts/verify.sh` exit 0(`검증 완료: jikim v0.2.28`, 프런트 59/59) 확인했고 500kB 청크 경고는 기존 상태다.
- [러너 13:49] brief accepted — 채택 — 근거가 코드와 정확히 맞았고(121-125·252-256 이 실제로 한 조건으로 묶여 있었고 43-47·55-59 는 이미 분리되어 있었�
- [러너 13:49] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 두 분기를 **따로** 되돌려 역방향 재현했다 — 로그인만 되돌리면 oidc_outage_test.go:70 만, 콜백만 되돌리면 :113 만 `status=503 (원하는 값 500)` 으로 실패했고 ErrNotFound·disabled·incomplete 5개는 양쪽 상태에서 통과했다. 원장의 `- 실패 재현:` 줄과 메시지까지 일치한다. 복원 후 트리 깨끗, `go test ./... -count=1`·`go vet`·`gofmt -l .` 전부 통과.
- 구현자가 의심한 자리를 코드로 메웠다: `GetSetting`(store/settings.go:409-411)이 `DecryptJSON` 실패를 `mapError` 없이 원본 그대로 올리므로 복호화 실패가 500 에 닿는 것이 맞다. 실제 PostgreSQL 장애·Keycloak 왕복은 나도 환경상 못 봤다(주입 오류로만 증명).
- 보안·법무 소견: 차단 없음. oidcLogin·oidcCallback 은 미인증 공개 경로지만 같은 미인증 `GET /api/v1/oidc/config` 가 이미 동일 장애를 500 `internal_error` 로 내주므로 새로 드러나는 정보가 없고, 500 본문은 `storeError` 가 덮고 테스트가 드라이버 원문 부재를 못박는다. 개인정보·새 의존성·외부 약속 문구 없음.
- 승인이어도 남는 우려: ① `loadOIDCConfig` seam 을 4개 호출처 중 2개만 쓴다(`oidcPublicConfig`:50·`oidcTest`:62 는 여전히 직접 호출). ② `mapError` 가 `ErrInvalid` 를 돌려주면 `storeError` 가 400 에 `err.Error()` 원문을 미인증 호출자에게 보낸다 — 이 경로(상수 키 SELECT)에서는 실현 불가이나 다음 1순위 `oidcExchange` 로 같은 패턴을 넓힐 때 반드시 다시 볼 것.
- 릴리즈 노트: 조용한 SSO 가 최상위 이동(silentSso.ts:89)이라 저장소 장애 시 사용자가 보던 생 JSON 503 이 생 JSON 500 으로 바뀐다(퇴행 아님, 문서화된 error code 표 없음 — api-guide.md:53-60 확인).
- [러너 13:53] review approved — 리뷰 승인 (risk=low)
- [러너 13:53] pr created — https://github.com/hkjang/jikim/pull/51
- [러너 13:56] ci passed — 검사 2개 모두 success
- [러너 13:56] merge done — 255e321
- [러너 14:03] release published — v0.2.29
- [러너 14:07] assets verified — v0.2.29 자산 2개 (이전 v0.2.28: 2)
