- 과제: 매칭된 PII 규칙이 원문을 한 글자도 가리지 않고 `masked_value`로 되돌려주는 경우 막기 (가치 3 / 위험 2 / 작업량 S)

- 왜: `internal/masking/policy.go`의 자리별 마스킹 헬퍼 중 숫자만 가리는 것들(`maskDigitsAfter`·`maskLastDigits`·`maskKeepTrailingDigits`·`maskDigitsRange`)과 `maskEveryEvenRune`은 입력이 짧거나 숫자가 없으면 **입력을 글자 그대로 반환**한다. 이번 회차에 임시 probe 테스트(`internal/masking/zz_probe_test.go`, 실행 후 삭제)로 11개 입력을 실제 `MaskValue`에 통과시켜 8건이 `masked == original`임을 확인했다 — 특히 `MaskValue("주민등록번호","900101")` → `"900101"`(주민번호 앞 6자리 = 생년월일, 규칙명은 `resident_registration_number`), `MaskValue("이름","이")` → `"이"`, `MaskValue("firstname","J")` → `"J"`.
  이 값은 `masking.FieldEntry.MaskedValue`(engine.go:41) → `service.summarizeFields`(service.go:662) → `core.PIISummaryItem.MaskedValue`(core/types.go:16, `json:"masked_value"`)를 거쳐 **HTTP 응답 본문의 `pii_summary[].masked_value`와 디스크의 `job.json`에 평문으로 저장·반환된다**. 즉 "마스킹했다"고 규칙명까지 붙여 놓고 원문 PII를 그대로 내보내는 상태다. 고치면 규칙이 매칭된 값은 어떤 입력에서도 최소 한 글자 이상 가려지고, 응답·저장본·이력에 원문이 남지 않는다.
  (도면 위 가림 자체는 이미 안전하다: `ComputeMaskedRuneSpans`가 빈 스팬을 돌려주면 `buildSubRegionsFromBBox`가 nil을 반환해 `collectMaskRegionsRecursive`가 bbox 전체를 덮는다. 그래서 이 과제는 **응답·저장 데이터의 누출**을 고치는 것이고 이미지/PDF 출력 픽셀은 바뀌지 않아야 한다.)

- 수용 기준:
  1) `MaskValue(key, value)`가 `empty` 규칙이 아닌 규칙을 적용했고 `value`에 공백 아닌 룬이 하나라도 있으면, 반환 `MaskedValue`는 **절대 `value`와 같지 않다**. 최소한 다음 8개 입력이 모두 바뀐다: `("이름","이")`, `("firstname","J")`, `("주민등록번호","900101")`, `("주민등록번호","확인불가")`, `("신용카드번호","카드없음")`, `("계좌번호","미상")`, `("전화번호","연락처없음")`, `("운전면허번호","없음")`.
  2) 룬 정렬 불변식이 유지된다 — 모든 신규/기존 케이스에서 `utf8.RuneCountInString(masked) == utf8.RuneCountInString(value)`이고 `ComputeMaskedRuneSpans(value, masked)`가 nil이 아니다(기존 `TestMaskValueKeepsRuneAlignment`가 policy_test.go:63에 있으니 거기에 케이스를 더하는 쪽이 자연스럽다). 공백 룬은 계속 공백으로 남는다.
  3) 기존 동작 불변: 이미 한 글자 이상 가려지던 입력의 출력이 **한 바이트도 바뀌지 않는다**. 최소한 `TestMaskValueExamples`(policy_test.go:5)와 integration_test.go:64의 `PIISummary[0].MaskedValue != "홍*동"` 단언이 무수정 통과해야 한다.
  4) 프로덕션 배선을 끝까지 지나는 회귀 1건: `startAppServerWithUpstream`(integration_test.go:1637)에 `주민등록번호`/`900101` 필드와 `boundingBoxes`를 돌려주는 직접 만든 업스트림 핸들러를 물리고 `POST /v1/mask`(또는 `/v1/jobs`) 응답의 `pii_summary[0].masked_value`가 `"900101"`이 아님을 단언한다. 손으로 만든 대역이 아니라 `app.New` → 실제 `httptest` 리스너를 지나야 한다(운영자 지시).
  5) 프로덕션 수정만 임시로 되돌리면 1)·4)의 실패가 그대로 재현되는 것을 확인하고 복원한다.

- 건드릴 파일 (프로덕션 1개):
  - `internal/masking/policy.go:29 MaskValue` — `maskTrimmedValue`가 돌려준 결과가 `trimmedValue`와 같고 규칙이 `empty`가 아니면 `maskAllVisible(trimmedValue)`로 폴백(규칙명·표시명은 그대로 유지해 어떤 규칙이 걸렸는지는 계속 보고). `maskAllVisible`(policy.go 하단)은 공백 아닌 룬만 `*`로 바꾸므로 룬 수가 보존되고 기준 2)가 자동으로 성립한다. 헬퍼 각각을 고치지 말고 **이 한 지점에서만** 막을 것 — 헬퍼를 개별로 손대면 숫자 자리 보존 계약(`maskDigitsAfter`의 "앞 6자리는 보인다")이 흔들린다. 왜 폴백이 필요한지 한 단락 주석을 남길 것.
  - `internal/masking/policy_test.go` — 위 8개 입력 테이블 + 불변식(1·2·3) 단언.
  - `internal/httpapi/integration_test.go` — 기준 4)의 회귀 1건.
  - README에 한 문장(마스킹 정책 문단)만 선택적으로 추가.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test -count=1 ./internal/masking`
  - `go test -count=1 ./internal/httpapi`
  - `go test -count=1 ./...`  (8패키지, 1초 내)
  - `go vet ./...` / `go build ./...` / `gofmt -l ./cmd ./internal`(무출력이어야 함) / `git diff --check`
  - `go test -race -count=1 ./internal/masking ./internal/httpapi`

- 위험과 피할 것:
  - `ComputeMaskedRuneSpans`의 룬 정렬 계약(policy.go:88 주석)을 깨면 PDF/이미지에 가림 상자가 **엉뚱한 글자** 위에 그려진다. `maskAllVisible` 외의 방법(글자 추가·`[MASKED]` 같은 치환)을 쓰지 말 것.
  - `restoreSurroundingSpace`가 폴백 **뒤에** 적용되는 현재 순서를 유지할 것(`MaskValue`는 trimmed 값으로 마스킹한 뒤 공백을 되붙인다). 폴백을 `maskTrimmedValue` 바깥의 `MaskValue` 안, `restoreSurroundingSpace` 호출 **전에** 끼워야 한다.
  - `empty` 규칙(trimmedValue가 빈 문자열)은 지금처럼 원본을 그대로 돌려주고 폴백을 타지 않아야 한다 — 가릴 글자가 없다.
  - 이미지/PDF 마스킹 픽셀 출력은 바뀌지 않아야 한다. `collectMaskRegionsRecursive`(engine.go:116)·`buildSubRegionsFromBBox`·`MaskImageFile`은 손대지 말 것. (스팬이 빈 상태에서 "전체 bbox 덮기"였던 것이, 폴백 후 "전 구간 스팬 1개"가 되어도 덮는 영역은 사실상 같다. 신규 테스트가 PDF/이미지 바이트를 단언할 필요는 없다.)
  - 보호 경로(auth·migrations·.github)는 건드리지 않는다. 이 저장소에는 마이그레이션도 CI 워크플로도 없다.
  - 과거 재선정 금지 항목: UTF-8 치환(9/21), created_at(9/22), download_url(9/24), mock 주소·HTTPS 강등(9/26), MAX_PAGES=0(9/27), duration 오버플로(9/29), MAX_FILE_SIZE_MB 오버플로(10/02), 선언 MIME 인코딩(10/03), 업스트림 PNG 전송(10/04).

- 차선 후보: `/v1/config/public`이 노출하는 제한값의 양수 불변식 + `uploadBodyHeadroomBytes` 덧셈 경계 테스트 (가치 2 / 위험 1 / 작업량 S — server.go:29,116,315, 프로덕션 변경 0). 1순위가 이미 고쳐져 있거나 기존 테스트와 정면 충돌하면 이것을 할 것.
