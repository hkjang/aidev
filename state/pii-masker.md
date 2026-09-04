# pii-masker 자율 개선 기록

## 2026-09-02
- 선택: 비동기 `/v1/jobs` 업로드 사전 검증 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `POST /v1/jobs`가 동기 경로(`/v1/mask`)와 달리 크기·MIME·페이지 수 검증을 건너뛰어, 초과/미지원 파일도 디스크에 저장하고 202를 반환한 뒤 백그라운드에서야 실패하던 문제를 고쳤습니다. `service.CreateJob`이 저장 전에 `validateAttachment`/`countPages`를 실행하고 새 `service.InvalidInputError`로 감싸 핸들러가 400 `invalid_request`를 돌려주도록 했으며, 검증 실패 시 job 디렉터리가 생기지 않는지 확인하는 통합 테스트 3개(미지원 타입/크기 초과/PDF 페이지 한도)를 추가했습니다. 부수적으로 `-race`에서 드러난 pdfcpu `ConfigPath` 전역 변수 경합을 요청마다 쓰지 않고 각 패키지 `init()`에서 한 번만 설정하도록 분리했습니다. 검증은 `go vet ./...`, `go test ./...`, `go test -race ./...` 전부 통과, 그리고 수정 코드를 임시로 되돌려 새 테스트 3개가 실제로 실패하는 것까지 확인했습니다(커밋 2개).
- 보류 아이디어: `/v1/history`의 `limit` 상한 없음(과도한 값 요청 시 전체 목록 직렬화) / job 파일 보존·정리 정책 부재로 저장소 무한 증가 / `CreateJob`의 `go runJob` 동시 실행 개수 제한 없음 / `internal/jobs`, `internal/config` 패키지 단위 테스트 전무 / 마스킹 정책 엣지케이스(빈 로컬파트 이메일, 구분자 없는 계좌번호 등) 테스트 보강
- 릴리즈: v1.0.4 (2026-09-02, 태그 사후 푸시)

## 2026-09-02 (2회차)
- 선택: 마스킹 값의 룬 정렬 보장으로 잘못된 영역 검게 칠하는 문제 수정 (가치 5 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `MaskValue`가 `TrimSpace`한 값을 마스킹해 반환하고 `maskAddress`가 `strings.Fields`+`Join(" ")`로 공백을 뭉개는 바람에, 마스킹 결과가 원본과 룬 수·위치가 어긋나 `ComputeMaskedRuneSpans`가 엉뚱한 구간을 돌려주고 문서에 잘못된 위치가 검게 칠해지던 실제 PII 노출 버그를 고쳤습니다(예: `" 800901-1234567"` → 스팬 `[7,14)`, 마지막 자리 `7`이 그대로 노출 / 이중 공백 주소는 번지 대신 도로명 일부를 가림). 마스킹은 트림된 값에 적용하되 앞뒤 공백을 복원하는 `restoreSurroundingSpace`를 추가하고, 주소는 토큰 시작 오프셋을 보존해 원본 구분자를 그대로 재조립하도록 바꿨으며, 안전망으로 `ComputeMaskedRuneSpans`가 룬 길이 불일치 시 `nil`을 반환해 호출부가 필드 전체를 덮도록 했습니다. 모든 규칙이 룬 수·공백 위치를 보존하는지 검사하는 테이블 테스트 15케이스와 회귀 케이스 4개를 추가했고, `gofmt -l`(무출력), `go vet ./...`, `go test -count=1 ./...`, `go test -race -count=1 ./...` 통과 및 수정 전 코드로 되돌려 새 테스트가 실제로 실패하는 것까지 확인했습니다.
- 보류 아이디어: `/v1/history`의 `limit` 상한 없음(과도한 값 요청 시 전체 목록 직렬화) / job 파일 보존·정리 정책 부재로 저장소 무한 증가 / `CreateJob`의 `go runJob` 동시 실행 개수 제한 없음 / `internal/jobs`, `internal/config` 패키지 단위 테스트 전무 / `maskAllVisible` 등이 `isWhitespaceRune`(ASCII 한정)만 보므로 U+00A0 같은 유니코드 공백 처리 불일치
- 릴리즈: v1.0.5 (2026-09-02)

## 2026-09-03
- 선택: 업로드 요청 본문 크기 상한 적용 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `POST /v1/mask`, `POST /v1/jobs`가 `r.Body`에 아무 상한 없이 `ParseMultipartForm(MaxFileSizeBytes + 1MB)`를 호출해, 클라이언트가 임의 크기(수 GB) 본문을 흘려보내면 서버가 51MB를 메모리에 버퍼링하고 나머지는 temp 파일로 흘린 뒤에야 `validateAttachment`에서 거절하던 메모리·디스크 DoS 경로를 막았습니다. `http.MaxBytesReader`로 본문을 `MaxFileSizeBytes + 64KB`(멀티파트 프레이밍 여유분)에서 자르고, 잘린 요청은 새 `payloadTooLargeError` → `413 payload_too_large`로 응답하며, `maxMemory`를 8MB 고정으로 낮춰 큰 파트가 RAM에 이중으로 남지 않고 temp 파일로 넘어가게 했습니다. 상한 초과 시 413을 받는 통합 테스트 2개(`/v1/mask`, `/v1/jobs` + job 미생성 확인)와 한도에 딱 맞는 업로드가 여전히 200으로 처리되는 회귀 테스트 1개를 추가했고, `gofmt -l`(무출력)·`go vet ./...`·`go test -count=1 ./...`·`go test -race -count=1 ./...` 전부 통과, `-count=5`로 새 테스트 플래키 여부 확인, 수정 전 코드로 되돌려 새 테스트 2개가 실제로 실패(본문 512KB를 끝까지 읽고 400 반환)하는 것까지 확인했습니다.
- 보류 아이디어: `/v1/history`의 `limit` 상한 없음(과도한 값 요청 시 전체 목록 직렬화) / job 파일 보존·정리 정책 부재로 저장소 무한 증가 / `CreateJob`의 `go runJob` 동시 실행 개수 제한 없음 / `internal/jobs`, `internal/config` 패키지 단위 테스트 전무 / 마스킹 결과가 원본과 동일할 때(예: 숫자 없는 주민등록번호 필드) 스팬이 비어 필드 전체를 덮는 과잉 마스킹
- 릴리즈: v1.0.6 (2026-09-03)

## 2026-09-03 (2회차)
- 선택: 이미지 압축 폭탄(선언 해상도) 디코딩 상한 적용 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: 업로드 크기 상한은 있었지만 디코딩 후 픽셀 수에는 아무 제한이 없어, 헤더에 40000x40000을 선언한 수백 KB짜리 PNG 한 장으로 `image.Decode`가 6GB 이상을 할당하게 만드는 메모리 고갈 경로가 남아 있었습니다(`masking.MaskImageFile`, `upstage.prepareUpstreamAttachment` 두 곳). 새 `document.ValidateImageDimensions`/`DecodeImage`가 `image.DecodeConfig`로 헤더만 먼저 읽어 5천만 픽셀(600dpi A4 스캔 이상) 초과를 거절하고, `service.validateAttachment`에서 이미지 MIME일 때 이를 호출해 동기·비동기 경로 모두 픽셀 버퍼 할당이나 job 저장 전에 400으로 실패하게 했으며, 두 디코딩 지점도 같은 가드를 거치도록 방어를 이중화했습니다. IHDR을 조작한 초소형 폭탄 PNG로 단위 테스트 4개와 통합 테스트 2개(`/v1/mask` 오류 메시지 확인, `/v1/jobs` 400 + job 미생성)를 추가했고, `gofmt -l`(무출력)·`go vet ./...`·`go test -count=1 ./...`·`go test -race -count=1 ./...` 전부 통과, 가드를 임시로 제거해 새 테스트 3개가 실제로 실패(각각 미거절 / 202 반환 / "not enough pixel data" 디코드 오류)하는 것까지 확인했습니다.
- 보류 아이디어: `/v1/history`의 `limit` 상한 없음(과도한 값 요청 시 전체 목록 직렬화) / job 파일 보존·정리 정책 부재로 저장소 무한 증가 / `CreateJob`의 `go runJob` 동시 실행 개수 제한 없음(고루틴마다 원본 바이트를 메모리에 유지) / `internal/jobs`, `internal/config` 패키지 단위 테스트 전무 / 마스킹 결과가 원본과 동일할 때 스팬이 비어 필드 전체를 덮는 과잉 마스킹
- 릴리즈: v1.0.7 (2026-09-03)

## 2026-09-03 (3회차)
- 선택: 업로드 파일명 제어문자 정화로 멀티파트 헤더 인젝션 차단 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `sanitizeUploadFilename`이 공백 트림과 `filepath.Base`만 하고 있어서, 클라이언트가 RFC 2231(`filename*=utf-8''sample%0D%0A...`)로 인코딩해 보낸 파일명이 Go에서 진짜 CR/LF를 품은 문자열로 디코딩된 뒤 그대로 두 곳의 헤더에 박히던 인젝션 경로를 막았습니다. `/v1/mask`가 돌려주는 `multipart/mixed` 파트의 `Content-Disposition`(멀티파트 파트 헤더는 `net/http`의 응답 헤더 정화를 거치지 않음)과 추론 서버로 나가는 요청의 document 파트 헤더가 대상이며, 실제로 수정 전 코드에서는 조작된 파일명이 업스트림 멀티파트 본문을 깨뜨려 mock이 "document part is required"로 400을 반환했습니다. 업로드 수용 시점에 제어문자·따옴표·역슬래시·경로 구분자를 제거하고 남는 게 없으면 `document`로 대체하며 120바이트(확장자 유지, 룬 경계 보존)로 자르고, 한글 등 비ASCII 이름은 그대로 둡니다. 헤더를 만드는 두 지점(`httpapi.attachmentDisposition`, `upstage.escapeMultipartValue`)에도 같은 가드를 넣어 방어를 이중화했습니다. `internal/document` 단위 테스트 16케이스(테이블 13 + 절단/`NewAttachment`/`MaskedFilename`)와 통합 테스트 2개(`/v1/mask` 응답 본문에 `\r\nX-Injected:` 없음, `/v1/jobs` 완료 후 job 디렉터리 파일명·다운로드 `Content-Disposition` 정상)를 추가했고, `gofmt -l`(무출력)·`go vet ./...`·`go build ./...`·`go test -count=1 ./...`·`go test -race -count=1 ./...` 전부 통과, `-count=5`로 플래키 여부 확인, 수정 코드를 임시로 되돌려 새 통합 테스트 2개가 실제로 실패(각각 502 업스트림 오류 / 파일명에 CRLF 잔존)하는 것까지 확인했습니다.
- 보류 아이디어: `/v1/history`의 `limit` 상한 없음(과도한 값 요청 시 전체 목록 직렬화) / job 파일 보존·정리 정책 부재로 원본 PII 파일이 무기한 잔존 / `CreateJob`의 `go runJob` 동시 실행 개수 제한 없음(고루틴마다 원본 바이트를 메모리에 유지, 디스크에 이미 저장된 입력을 재사용하면 해소 가능) / `internal/jobs`, `internal/config` 패키지 단위 테스트 전무 / `handleGetJobResult`가 결과 파일 전체를 메모리에 올린 뒤 응답(스트리밍 전환 여지)
- 릴리즈: v1.0.8 (2026-09-03)

## 2026-09-04
- 선택: 비동기 job 동시 실행 개수 제한 + 입력 바이트 디스크 재읽기 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `CreateJob`이 요청마다 `go s.runJob(jobID, input)`을 무제한으로 띄우면서 클로저가 업로드 원본(최대 50MB)을 통째로 붙들고 있어, `/v1/jobs`를 연달아 호출하면 동시에 실행되는 고루틴 수만큼 메모리가 선형으로 늘어나 OOM에 이르는 경로를 막았습니다. `Service`에 `jobSlots` 세마포어(`PII_MASKER_MAX_CONCURRENT_JOBS`, 기본 4, 0 이하이면 4로 폴백)를 두어 슬롯을 얻은 러너만 실행되게 하고, 러너에 넘기는 값을 `upstage.ParseOptions`만으로 줄인 뒤 새 `loadJobInput`이 슬롯 확보 후에야 `job.InputPath`(지금까지 기록만 하고 아무도 읽지 않던 필드)에서 원본을 다시 읽도록 바꿨습니다. 덕분에 대기 중인 작업은 고루틴 하나 값만 쓰고 `queued` 상태를 유지하며, 저장된 입력을 읽지 못하면 `storage_read_failed`로 실패 처리합니다. 검증은 업스트림 핸들러를 잡아두고 동시 요청 수를 세는 통합 테스트(제한 2, 작업 5개 → 최대 동시 2 유지, 해제 후 5개 모두 completed)와 `loadJobInput` 단위 테스트 2개를 추가했고, `gofmt -l`(무출력)·`go vet ./...`·`go build ./...`·`go test -count=1 ./...`·`go test -race -count=1 ./...` 전부 통과, `-count=5`로 플래키 여부 확인, 세마포어 획득 두 줄을 임시로 제거해 새 통합 테스트가 실제로 실패("expected at most 2 concurrent jobs, got 5")하는 것까지 확인했습니다.
- 보류 아이디어: `/v1/history`의 `limit` 상한 없음(과도한 값 요청 시 전체 목록 직렬화) / job 파일 보존·정리 정책 부재로 원본 PII 파일이 무기한 잔존(TTL 스위퍼 필요) / `handleGetJobResult`가 결과 파일 전체를 메모리에 올린 뒤 응답(`http.ServeContent` 스트리밍 전환 여지) / `internal/jobs`, `internal/config` 패키지 단위 테스트 전무 / 대기 중인 job 개수 자체는 여전히 무제한이라 디스크는 계속 증가
- 릴리즈: v1.0.9 (2026-09-04)

## 2026-09-04 (2회차)
- 선택: job 보존 기간(TTL) 정리 스위퍼 (가치 5 / 위험 3 / 작업량 M)
- 결과: 성공
- 요약: 비동기 job이 저장한 업로드 원본(=마스킹해 달라고 받은 개인정보 원문)과 마스킹 결과가 디스크에 무기한 남고 job 레코드도 메모리 맵에 영구 누적되던 문제를 고쳤습니다. 5회 연속 보류돼 온 최대 잔여 리스크입니다. `jobs.Store.DeleteExpired(cutoff)`가 `Metadata.UpdatedAt`이 cutoff 이전인 job의 디렉터리를 통째로 지우고 맵에서도 제거하되, `queued`/`running` 상태는 나이와 무관하게 보존합니다(러너가 슬롯을 얻으면 `InputPath`를 다시 읽어야 하므로). `Service`에 `PurgeExpiredJobs`/`StartRetentionSweeper`를 추가해 기동 직후 1회 + `retention/4`(1분~1시간 클램프) 주기로 정리하고, `app.New`가 컨텍스트로 스위퍼를 띄우고 새 `App.Close()`가 멈춥니다. 보존 기간은 `PII_MASKER_JOB_RETENTION_HOURS`(기본 24시간, `0`이면 정리 비활성)이며, 0을 허용해야 해서 `envNonNegativeInt`를 새로 뒀습니다(기존 `envInt`는 0 이하를 기본값으로 되돌림). 검증은 `internal/jobs` 단위 테스트 3개(만료 삭제/미완료 job 보존/재기동 후 로드된 job에도 적용 — 이 패키지 첫 테스트), `internal/service` 테스트 4개(퍼지, 보존 0이면 미삭제, 스위퍼가 컨텍스트 취소까지 반복, 주기 계산), 통합 테스트 1개(이전 실행이 남긴 만료 job 디렉터리가 기동 직후 사라지고 `GET /v1/jobs/{id}`가 404, 최근 job은 유지)를 추가했고 `gofmt -l`(무출력)·`go vet ./...`·`go build ./...`·`go test -count=1 ./...`·`go test -race -count=1 ./...` 전부 통과, `-race -count=5`로 플래키 여부 확인, 그리고 (1) `DeleteExpired` 무력화 (2) `app.New`의 스위퍼 기동 제거 (3) queued/running 가드 제거 세 가지로 되돌려 각각 대응 테스트가 실제로 실패하는 것까지 확인했습니다.
- 보류 아이디어: `/v1/history`의 `limit` 상한 없음(과도한 값 요청 시 전체 목록 직렬화) / `handleGetJobResult`가 결과 파일 전체를 메모리에 올린 뒤 응답(`http.ServeContent` 스트리밍 전환 여지) / `internal/config` 패키지 단위 테스트 전무(`normalizeAllowHosts`, `envNonNegativeInt` 등) / 대기 중인 job 개수(큐 길이) 자체는 여전히 무제한이라 폭주 시 디스크가 보존 기간 동안 계속 증가 / `cmd/pii-masker`에 graceful shutdown 없음(`App.Close`가 생겼으므로 시그널 처리와 묶을 여지)
- 릴리즈: v1.0.10 (2026-09-04)
