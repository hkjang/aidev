# pii-masker 자율 개선 기록

## 2026-09-02
- 선택: 비동기 `/v1/jobs` 업로드 사전 검증 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `POST /v1/jobs`가 동기 경로(`/v1/mask`)와 달리 크기·MIME·페이지 수 검증을 건너뛰어, 초과/미지원 파일도 디스크에 저장하고 202를 반환한 뒤 백그라운드에서야 실패하던 문제를 고쳤습니다. `service.CreateJob`이 저장 전에 `validateAttachment`/`countPages`를 실행하고 새 `service.InvalidInputError`로 감싸 핸들러가 400 `invalid_request`를 돌려주도록 했으며, 검증 실패 시 job 디렉터리가 생기지 않는지 확인하는 통합 테스트 3개(미지원 타입/크기 초과/PDF 페이지 한도)를 추가했습니다. 부수적으로 `-race`에서 드러난 pdfcpu `ConfigPath` 전역 변수 경합을 요청마다 쓰지 않고 각 패키지 `init()`에서 한 번만 설정하도록 분리했습니다. 검증은 `go vet ./...`, `go test ./...`, `go test -race ./...` 전부 통과, 그리고 수정 코드를 임시로 되돌려 새 테스트 3개가 실제로 실패하는 것까지 확인했습니다(커밋 2개).
- 보류 아이디어: `/v1/history`의 `limit` 상한 없음(과도한 값 요청 시 전체 목록 직렬화) / job 파일 보존·정리 정책 부재로 저장소 무한 증가 / `CreateJob`의 `go runJob` 동시 실행 개수 제한 없음 / `internal/jobs`, `internal/config` 패키지 단위 테스트 전무 / 마스킹 정책 엣지케이스(빈 로컬파트 이메일, 구분자 없는 계좌번호 등) 테스트 보강
