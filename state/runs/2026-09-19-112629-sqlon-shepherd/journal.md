# PR 처리기 노트 2026-09-19-112629-sqlon-shepherd — sqlon PR #7
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## 수리 노트
- 두 지적 모두 맞았음: diff 에서 `SetCacheTTL` SettingDef 가 -만 있고 + 가 없었고, PUT 핸들러는 map 순회 중 키별 저장→실패 시 400 이라 부분 저장이 실제로 남음(재현 테스트로 확인).
- 고침: `SetCacheTTL` 항목을 `SettingDefs` 끝에 복원; `meta.ValidateSetting` 을 추가해 PUT 이 본문 전체를 검증한 뒤에만 저장(OpenAPI 문구 "아무것도 저장하지 않습니다" 와 일치). 커밋 750a3d8, 테스트 `TestSettingsPutValidatesBeforeStoringAndKeepsCacheTTL` 추가, `go test ./...` 통과.
- 확신 없는 곳: 저장 단계(SetSetting)의 실패는 이제 500 으로 응답하는데, 이 경우엔 여전히 부분 저장이 가능함(스토어 트랜잭션 미도입 — PR 범위 밖으로 판단). `null` 삭제는 이전과 같이 검증 없이 처리.

## 심사 노트
- 확인: 이전 거절 2건(cache_ttl_seconds 복원, PUT 전체 검증 후 저장)은 코드·테스트로 제대로 고쳐짐. 기본 꺼짐·공개 경로(RFC 9728 두 개뿐)·Host 미사용·aud/iss/exp/nbf/typ/cnf/HS256/미등록/비활성 거부·빈 scope 교집합 거부·계정 미생성·시크릿 미노출을 파일과 테스트에서 확인, `go test ./...`·`go vet` 통과.
- 결함(reject/fix): mcpoauth.go jwksCache.key 가 뮤텍스를 잡은 채 discovery+JWKS 네트워크 호출 → 임시 테스트로 정상 캐시 토큰이 2.8초(운영 최대 ~20초) 블록됨을 재현; keys==nil/TTL 만료 뒤 fetch 실패 시 스로틀 우회로 IdP 장애 때 요청마다 재시도(5회 호출 재현). 둘 다 캠페인 규칙 위반.
- 부족: refuseOAuth 로그에 request_id 없음(저장소에 request-id 미들웨어 자체가 없음 → 최소 훅 추가 필요), 거부 로그 캡처 테스트 없음; README/CHANGELOG/keys.html/settings.html 전체 CRLF→LF 변환(범위 밖, 충돌 유발).
- 못 본 것: 실제 Keycloak 과의 e2e 는 안 함(가짜 IdP 로 운영 경로만 검증); 웹 UI(settings.html/keys.html) 는 브라우저로 열어보지 않음; PDF 산출물 내용은 확인 안 함.
- 권고 근거: 결함이 인증 경로 가용성(IdP 지연이 정상 토큰까지 막음)에 있고 고치는 방법이 명확해 fix. 위험도 medium(기본 꺼짐이라 미설정 설치엔 영향 없음).
