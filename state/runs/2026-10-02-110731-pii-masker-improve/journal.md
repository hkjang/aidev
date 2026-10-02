# 회차 노트 2026-10-02-110731-pii-masker-improve — pii-masker
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:07] base pinned — main@b183632
- [러너 11:07] autonomy release — 

## 정찰 노트
- 선택 이유: 001f6f7이 시간 설정 여섯 개만 닫고 남겨 둔 같은 결함의 마지막 한 곳(`config.go:148`의 MB 곱셈)이고, 고칠 자리와 고칠 방법(`envDuration`의 상한 검사)이 같은 파일에 이미 선례로 있어 프로덕션 파일 1개로 끝난다. 차선인 정규화 테이블 테스트는 프로덕션 변경이 0이라 가치가 더 낮고, download_url 헬퍼·405 Allow는 관찰 가능한 개선이 없어 제쳤다.
- 추측으로 적은 것: Go `http.MaxBytesReader`가 음수 n을 어떻게 다루는지 소스 확인을 못 했다(샌드박스가 `go env GOROOT`/`go run`을 막음). 그래서 수정 전 상태코드를 413으로 단정하지 않고 "413 또는 400, 실행해서 기록" 으로 적었다. 음수로 감싸는 산술(8796093022208×1048576 = 2^63)과 세 지점이 모두 그 음수를 읽는다는 것은 코드로 확인했다.
- 구현자 주의: `envInt` 함수 자체를 고치면 `MAX_CONCURRENT_JOBS`/`MAX_CONCURRENT_SYNC`까지 파급된다 — 2026-09-27처럼 호출 지점만 바꿔라. HTTP 회귀는 `app_test.go:509`의 `MAX_PAGES=0` 테스트를 본뜨면 되고, `t.Setenv`는 `t.Parallel`과 함께 쓸 수 없다.
- 확인한 것: `go test -count=1 ./...` 전부 ok(이번 정찰에서 실행), `math`는 config.go에 이미 import됨, `docker-compose.yml`·`Dockerfile`은 `MAX_FILE_SIZE_MB`를 설정하지 않음.
- 프로필은 3일 전(main@6badf77 기준)이고 이번에 읽은 코드와 어긋나지 않아 새로 쓰지 않았다.
- [러너 11:11] scout done — `PII_MASKER_MAX_FILE_SIZE_MB`의 MiB 곱셈 오버플로 방어 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- `config.go:148`의 무검사 MB→바이트 곱셈을 새 헬퍼 `envMebibytes`(ParseInt + 곱셈 전 `math.MaxInt64/mebibyte` 상한)로 교체. 수정 전에는 `MAX_FILE_SIZE_MB=8796093022208`이 `MaxFileSizeBytes`를 `MinInt64`로 감싸 비어 있지 않은 모든 업로드가 **413 payload_too_large**(`...maximum size of -9223372036854710272 bytes`)로 거절됐다 — 정찰이 미확인으로 남긴 "셋 중 어디서 먼저 끊기나"는 `http.MaxBytesReader`가 음수 n을 즉시 초과로 처리해 거기서 끊기는 것으로 실행 확인했다.
- 확신 없는 곳: 상한을 `MaxInt64/1MiB`로 둔 뒤 `server.go`의 `+64KiB` 덧셈이 오버플로하지 않는 것은 산술로만 확인했고(여유 1MiB-64KiB), 그 불변식을 고정하는 테스트는 **추가하지 않았다**(ideas.json에 후보로 적음). 또 `/v1/config/public`이 노출하는 값은 config가 양수를 보장하니 함께 고쳐지지만 그 응답을 직접 단언하는 테스트는 없다.
- 일부러 하지 않은 것: `envInt` 함수 자체 수정(파급 차단 — `MAX_CONCURRENT_JOBS`/`MAX_CONCURRENT_SYNC`는 곱셈이 없어 범위 밖), `uploadBodyHeadroomBytes` 변경, `docker-compose.yml`·`Dockerfile`(이 변수를 설정하지 않음 — grep 확인).
- 다음 역할 주의: 새 HTTP 회귀 `TestOverflowingMaxFileSizeStillAcceptsAnUpload`는 `t.Setenv`를 쓰므로 `t.Parallel` 금지이고 `127.0.0.1:0` 리스너를 연다(외부 네트워크·DB 불필요, 내장 mock 업스트림 사용). 검증: `go test -count=1 ./...` rc=0, `go vet`·`go build`·`gofmt -l ./cmd ./internal`(무출력)·`git diff --check`·`-race ./internal/config ./internal/app`·`GOARCH=386 go test ./internal/config` 전부 통과. 프로덕션 수정만 되돌리면 두 테스트가 다시 실패하는 것을 확인 후 복원했다.
- [러너 11:16] brief accepted — 채택 — `config.go:148`의 무검사 곱셈이 현재 HEAD에 그대로 남아 있었고, 수용 기준 세 개(경계 테이블·프로덕션 배선 HTTP �
- [러너 11:16] verify passed — 검증 3개 통과 (policy)

## 비평 노트
- 구현자가 "산술로만 확인" 이라 적은 자리를 재계산해 참임을 확인했다: 허용 최대 8796093022207 MiB = 9223372036853727232 바이트, MaxInt64 까지 여유 1048575 바이트 → `server.go:315` 의 +64KiB 와 `:330` 의 +1 은 오버플로 불가. 네 소비 지점 모두 int64 로만 다뤄 32비트 절단도 없다(`GOARCH=386` 테스트·빌드 통과).
- 프로덕션 수정만 main 으로 되돌려 두 신규 테스트가 실제로 실패함을 직접 확인했다(config 서브테스트 3개 + `app_test.go:598`). 테스트가 바뀐 경로를 지난다. 검증: `go test ./...`·`-race`·`vet`·`gofmt`·`diff --check` 전부 통과. 보안·법무 차단 사유 없음(신규 경로·인가·비밀값·개인정보 처리 변경 0, 변경 방향은 상한을 넓히지 않고 좁힌다).
- 승인이어도 남는 우려 ①: `uploadBodyHeadroomBytes`(server.go:29)를 1MiB 이상으로 올리면 `envMebibytes` 상한이 더는 `bodyLimit` 덧셈을 보호하지 못한다 — 현재 결함은 아니지만 그 1MiB 여유를 고정하는 테스트가 없다. 다음 회차 후보.
- 우려 ②: `TestLoadMaxFileSizeBounds` 가 형제 `TestLoadDurationBounds` 와 달리 `UPSTAGE_BASE_URL`/`ENABLE_EMBEDDED_UPSTAGE_MOCK` 를 비우지 않아, 주변 환경이 오염되면 `Load()` 에러로 플레이크될 수 있다(거짓 통과는 아니다). 또 `app_test.go` 신규 테스트의 HTTP 200 단언은 수정 전 config 단언에서 먼저 멈추므로 413 회귀의 핀이 아니라 배선 문서화 역할이다.
- 못 본 것: Docker 이미지 빌드(Dockerfile 1.25.0 vs 로컬 1.26.7), 실제 Upstage 업스트림 경로, UI 의 `max_file_size_bytes` 표시.
- [러너 11:19] review approved — 리뷰 승인 (risk=low)
- [러너 11:19] pr created — https://github.com/hkjang/pii-masker/pull/30
- [러너 11:19] ci passed — 검사 없음 — 정책으로 허용
- [러너 11:19] merge done — 28b6bcc
- [러너 11:22] release published — v1.0.32
- [러너 11:22] gh-release created — GitHub Release v1.0.32
- [러너 11:22] manifest ok — pii-masker-image.tar.gz 
- [러너 11:23] assets uploaded — 1개
- [러너 11:23] assets verified — v1.0.32 자산 1개 (이전 v1.0.31: 1)
