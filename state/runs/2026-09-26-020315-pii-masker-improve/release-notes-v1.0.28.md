## 내장 mock 기본 업스트림 URL이 `PII_MASKER_ADDR`의 포트를 따르지 않던 문제 수정

`config.Load`는 mock 모드의 기본 업스트림을 `http://localhost:8080/internal/mock/upstage/inference`로 하드코딩했습니다. 그런데 mock 핸들러는 `app.New`에서 API와 같은 mux에 마운트되므로 그 URL은 "자기 자신"을 가리켜야 합니다. `PII_MASKER_ADDR`를 8080 이외의 포트로 두면 서비스는 자기가 듣고 있지 않은 포트로 마스킹 요청을 보내 모든 마스킹이 실패했고, 그 포트에 다른 프로세스가 떠 있으면 업로드한 PII 원문과 인증 토큰이 그쪽으로 전송될 수 있었습니다(기본 allow list는 base URL의 호스트 하나뿐이고 `hostAllowed`가 bare-host 항목의 포트를 무시하므로 차단되지 않습니다).

- `PII_MASKER_ADDR`를 `Load` 맨 위에서 한 번만 읽어 서버 설정과 mock 기본 URL이 같은 값을 보게 했습니다. 새 헬퍼 `localMockBaseURL`이 `net.SplitHostPort`로 주소를 쪼개 빈 호스트·`0.0.0.0`·`::`는 `localhost`로 대체해 URL을 조립하며, `SplitHostPort`가 실패하면 기존 `localhost:8080`을 유지합니다.
- mock이 꺼졌을 때의 외부 기본값 `http://localhost:8080/inference`와, 명시한 `PII_MASKER_UPSTAGE_BASE_URL`이 언제나 우선한다는 규칙은 그대로입니다. base URL을 명시하는 `docker-compose.yml`도 영향이 없습니다.
- `internal/config` 테이블 테스트 3개(호스트·포트 조합 7종, 명시 URL이 mock on/off 양쪽에서 우선, mock 꺼짐 기본값 유지)와 프로덕션 배선을 끝까지 지나는 통합 테스트 `TestEmbeddedMockMasksOnANonDefaultListenAddress`로 검증합니다. 리스너를 열어 둔 채 그 주소를 `PII_MASKER_ADDR`에 넣고 `config.Load` → `app.New` → `Serve`를 거쳐 같은 주소로 PNG를 `POST /v1/mask` 하여 첫 파트의 `status=completed`를 확인하며, `cfg.Upstage.BaseURL`이 리스너 포트를 담는지도 함께 단언해 CI 머신의 8080 점유 상태와 무관하게 실패 원인이 드러나도록 했습니다.
- README 환경변수 문단에 이 동작을 설명하는 한 문장을 추가했습니다.
