릴리스 워크플로의 소스 검사·테스트 단계를 로컬에서 한 번에 재현하는
`make release-check`와 `make test-integration` 타깃을 더하고 실행법을
README에 문서화합니다. 빌드 절차와 문서만 바뀌고 서비스 동작·API·스키마는
그대로이므로 patch를 올립니다.

- `make release-check`: release.yml의 "소스 검사와 테스트" 단계와 같은
  명령을 같은 순서로 실행(go mod verify → vet → govulncheck → go test →
  통합 테스트 → check-screenshots → npm ci/audit high/lint/test/build),
  첫 실패에서 멈추고 마지막 줄에 `release-check OK`를 출력
- `make test-integration`: `JUPIQ_INTEGRATION_TEST_DSN`이 없으면 조용히
  skip하지 않고 PostgreSQL 준비 예시를 안내한 뒤 exit 1
- README "소스 빌드" 절에 "릴리스 전 로컬 검증" 소절 추가(postgres 컨테이너,
  DSN, govulncheck 네트워크 필요, 비loopback 인터페이스 없을 때 통합 테스트
  skip 주의)
- 워크플로 파일·Go/TS 코드·마이그레이션 변경 없음
