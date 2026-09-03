상태 확인 endpoint가 HEAD 요청을 거부하던 문제를 바로잡은 패치 릴리스입니다.

- 로드 밸런서와 가동 감시 도구가 흔히 사용하는 HEAD 요청이 `/health/live`, `/health/ready`, `/api/v1/health/live`, `/api/v1/health/ready`, `/api/v1/meta`, `/api/v1/openapi.json`에서 405를 반환하던 문제 해결. 라우터가 GET에서 HEAD를 유도하지 않아 실제로 존재하는 경로가 없는 것처럼 응답했습니다
- 여섯 경로 모두 GET과 동일한 상태 코드·header를 반환하며, 본문은 HTTP 규격대로 비어 있습니다. 부작용이 없는 공개 상태·메타 endpoint에만 적용하고 나머지 경로의 메서드 계약은 그대로 유지
- 위 경로의 `Allow` header가 `GET, HEAD`를 함께 나열하도록 정정되고, OpenAPI 문서에 `head` operation을 추가
- HEAD 응답 상태·Content-Type과 `Allow` header를 라우터 단위 테스트로 검증

**Full Changelog**: https://github.com/hkjang/ai-admin/compare/v1.2.5...v1.2.6
