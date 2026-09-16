### 수정

- `POST /v1/transit/decrypt/{key}`가 존재하지 않는 key를 `403` `permission denied`로 보고하던 오류 수정. `authorizeTransit`이 key 부재(`TransitPermission`의 `ErrNotFound`)를 encrypt에서만 create capability 확인으로 흡수하고 decrypt에서는 그대로 돌려줘 거부로 접히던 것을, `store.ErrNotFound`를 감싼 전용 sentinel `errTransitKeyNotFound`로 구분해 OpenBao와 같은 `400` `encryption key not found`로 응답합니다. 호출자는 이미 `transit/{key}` 경로 capability를 통과한 뒤이므로 key 이름을 알려 줘도 정책이 허용하지 않은 것을 드러내지 않으며, 정책 조회의 맨 `ErrNotFound`는 기존대로 `403`, MCP `transit.decrypt`는 기존 `404` not found를 유지합니다
- `POST /api/v1/login`과 `POST /v1/auth/userpass/login/{username}`이 비밀번호가 맞는 즉시 로그인 실패 창을 초기화하던 오류 수정. 로컬 로그인이 꺼진 계정(`403`)이나 보안 설정 조회 장애(`500`)도 거부이므로 실패 횟수를 유지하고, 세션을 만들기 직전에만 초기화합니다. 설정 조회는 `securityLoader` seam을 거쳐 순서를 DB 없이 검증합니다

### 변경

- 호환성 가이드에 key 부재는 `transit/{key}` 경로의 decrypt capability를 통과한 뒤에만 알려 주며, capability가 없으면 key 존재 여부와 무관하게 `403`임을 명시했습니다

### 문서

- 문서 프로파일과 두 가이드 PDF 표지를 v0.2.14로 갱신했습니다. 화면 캡처는 실제로 찍은 `v0.2.9`를 그대로 가리킵니다

**Full Changelog**: https://github.com/hkjang/jikim/compare/v0.2.13...v0.2.14
