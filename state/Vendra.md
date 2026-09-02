# Vendra 자율 개선 기록

## 2026-09-02
- 선택: 요청 본문의 날짜를 검증 없이 `$n::date`로 캐스팅하던 네 개의 쓰기 경로 수정 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공 (commit e5cef01)
- 요약: `validDateFields`는 business object의 세 날짜에만 적용돼 있었고, 같은 결함이 견적 유효일(포털 입찰), 예정일(포털 납품/인보이스/문의), 거래 시작일(공급업체 등록), 거래일(구매 원장) 네 곳에 남아 있었다. 잘못된 날짜가 PostgreSQL 캐스트까지 도달해 "…저장하지 못했습니다"라는 필드를 지목하지 않는 오류로 돌아왔고, 특히 포털 입찰은 마감 직전에 견적 전체(금액·품목·조건)가 저장되지 않으면서 공급업체에게 아무 단서도 주지 못했다. 철자가 아니라 연산(요청 파라미터의 date 캐스트)을 기준으로 훑어 찾았고, 같은 기준으로 패키지를 파싱해 검증 누락을 잡는 `TestEveryDateCastIsGuarded`와 네 엔드포인트를 실제로 호출하는 통합 테스트를 추가했다. 검증: docker postgres:16-alpine을 띄워 CI와 동일한 세 DSN(VENDRA_TEST_DSN / MIGRATE / UPGRADE)으로 `go test ./internal/... ./cmd/...` 전체 통과, `gofmt -l`·`go vet` 무결. 가드 테스트는 수정 하나를 되돌리면 실패하는 것까지 확인했다.
- 보류 아이디어:
  - CI의 go job이 `go test ./internal/...`만 돌려 `./cmd/...`를 빼놓음 — Makefile/README와 불일치 (가치 2 / 위험 1 / S)
  - `web` 프론트엔드 테스트 커버리지가 9개 파일뿐 — Sourcing/Objects/Admin 페이지에 테스트 없음 (가치 3 / 위험 1 / L)
  - 공급업체 수정(PATCH /suppliers/{id})은 tradingSince·annualSpend를 아예 갱신하지 않음 — 의도인지 누락인지 확인 필요 (가치 3 / 위험 2 / S)
  - `POST /spend/transactions`의 quantity·unitPrice·amount에 상한이 없음 — 폼에만 존재하는 제약 (가치 3 / 위험 2 / M)
  - `Makefile`의 VERSION(0.6.21)이 README의 릴리스 예시(0.7.26)와 어긋남 — 릴리스 문서 정합성 (가치 2 / 위험 1 / S)
