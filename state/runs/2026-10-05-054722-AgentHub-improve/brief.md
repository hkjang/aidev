- 과제: Momento·Matomo 제공자 주소에서 CSP로 전달하는 출처 길이를 제한한다 (가치 3 / 위험 2 / 작업량 S)
- 왜: `Settings.Validate()`는 허용 목록과 스니펫 출처를 제한하지만 `MomentoURL`·`MatomoURL`의 출처 길이는 검사하지 않아, 제공자 주소 한 개가 `PolicySources()`를 통해 CSP의 세 지시문으로 무제한 증폭된다. 이 마지막 제공자 경로에 저장 전 길이 검사를 추가하면 실수로 붙인 거대한 호스트가 콘솔 응답 헤더를 비대하게 만드는 것을 막을 수 있다.
- 수용 기준:
  1) `MaxProviderOriginRunes = 300`을 명명된 상수로 두고, 정규화된 두 URL의 `originOf()` 결과가 300룬을 넘으면 필드명(Momento/Matomo), 상한, 실제 길이가 담긴 한국어 오류로 거절한다. 오류에 과대 URL 전체를 다시 넣지 않는다. 기존 허용 출처 300룬과 같은 기준을 택한 이유를 산문 주석으로 적는다.
  2) 두 필드는 `Enabled=false`, 다른 provider 선택, MomentoProxy=true에서도 길이 검사를 한다. `!s.Enabled` 조기 반환 앞에 길이 검사만 두며, 빈 값·부분 작성·파싱 불가 입력의 기존 허용/거절 조건은 바꾸지 않는다. originOf가 빈 문자열이면 이 길이 검사에서는 통과하고 기존 provider별 유효성 검사가 책임진다.
  3) ASCII와 한국어 각각 출처 300룬 허용/301룬 거절, 두 필드 각각의 초과, 비활성·미선택·프록시 모드에서도 초과 거절을 테스트한다. URL 전체가 300자를 넘어도 출처가 짧고 `/base/` 등 긴 경로가 붙은 정상 URL은 허용한다. 입력을 절단하거나 다시 저장하지 않는다.
  4) `internal/api/tracking_test.go`에서 Matomo와 직접 Momento의 큰 출처를 `pagePolicy()`에 넣으면 크게 증폭됨을 측정하고, 같은 설정이 `Validate()`와 `server.validateSetting()` 양쪽에서 거절됨을 확인한다. 정확한 초과 입력 예시는 `https://` + 8000개의 `a` + `.corp.example`이며 site id도 넣는다. Momento는 반드시 `momentoProxy:false`를 명시한다(디코더 기본값은 true).
  5) 두 제공자 각각 최대 300룬 출처의 헤더 증가량을 같은 nonce의 provider=none 기준과 비교해 `3*(300+1)`룬임을 증명한다. 기존 기본 설정, 프록시 제공자, 스니펫/허용 목록 경계 테스트가 통과한다. 실제 DB 저장 검증을 하지 않았다면 API 검증 함수 테스트를 live 증명으로 보고하지 않는다.
- 건드릴 파일:
  - `internal/tracking/tracking.go:Settings.Validate`, 상수 블록 — 두 출처 길이 검사. 같은 파일 `MaxSnippetOriginEntries` 앞 주석의 “provider 주소는 무제한” 설명도 새 동작에 맞게 고친다. `originOf`, `Normalized`, `PolicySources`, `Snippet` 동작은 유지한다.
  - `internal/tracking/tracking_test.go:TestValidationRefusesWhatCannotWork` 인접 — 표 기반 경계·미선택 필드·긴 경로 회귀 테스트 추가.
  - `internal/api/tracking_test.go:TestTrackingSettingIsValidatedOnTheWayIn`, `carried` 주석 인접 — API 검증/실제 헤더 회귀와 더 이상 사실이 아닌 제공자 무제한 주석 수정. `carried=16*1024` 및 기존 멀티바이트 테스트는 유지한다.
  - 합계 프로덕션 파일 1개 + 테스트 파일 2개. 새 파일·프런트·문서 가이드 수정은 불필요하다.
- 검증 명령: 저장소 루트에서 `go test ./internal/tracking ./internal/api`. 정찰에서 실제 실행해 두 패키지 통과했다(tracking 0.005s, api 1.752s). 이번 구현 검증은 이 명령으로 시작하고 마친다. CI는 `go test -race ./cmd/... ./internal/...`를 실행하나 정찰에서는 전체 race/DB live를 실행하지 않았다.
- 위험과 피할 것: auth/session·migrations·workflows·runtime 이미지·BASE_VERSION·릴리즈 파일은 건드리지 않는다. 기존 URL의 경로·쿼리·포트를 무조건 자르거나 전체 URL 길이에 300을 적용하지 않는다. 이미 저장된 과대 설정은 읽을 때 재검증하지 않으므로 이 변경은 새 쓰기를 막는 것이며 DB 정리/읽기 폴백은 범위 밖이다. 기존 과대 URL을 가진 관리자가 다른 설정을 저장하거나 한 번 클릭 허용을 쓸 때 새 검증에 막힐 수 있으므로 오류가 해당 제공자 필드를 정확히 지목해야 한다. 룬 상한은 전체 CSP의 바이트 상한이 아니다: 기존 멀티바이트 허용 목록+스니펫이 16KiB를 초과하는 계약은 유지한다. 이 과제로 전체 헤더가 16KiB 안에 든다고 주장하지 않는다.
- 차선 후보: guide-shots 복원 실패에도 problems 요약을 출력한다 (2/1/S) — `web/scripts/guide-shots.mjs`의 `withGuideSettings` 뒤 요약을 오류 경로에서도 실행하되 원래 오류 전파·browser.close를 유지한다. `guide-settings-check.test.mjs`의 `boundary` 종료 앵커가 `  if (problems.length)`임을 고려해 실제 오류 경로를 테스트하고 `cd web && node --test scripts/guide-settings-check.test.mjs`로 확인한다. 1순위가 구현 시 이미 해결된 것으로 판명될 때만 선택한다.

범위와 대안 판단:
- 목표는 제공자 출처의 무제한 길이 한 경로를 닫는 것이다. 최소안(선택)은 공통 Validate에 출처 300룬 검사이며 기존 길이 정책과 같은 방식으로 끝난다.
- 전체 CSP 바이트 예산을 강제하는 안은 효과가 더 넓지만 이미 허용하는 멀티바이트 목록의 계약을 바꿔 호환성 판단이 필요하므로 후속 과제로 남긴다.
- 렌더링 때 출처를 자르는 안은 저장값과 실제 정책이 달라지므로 배제한다. 현 상태 유지/문서 경고만은 이미 소스 주석에 경고가 있어도 과대 설정이 저장되므로 선택하지 않는다.
- 핵심 가정: 300룬 출처면 정상 사내 수집기에 충분하다. 운영 데이터로는 미확인이다. 호스트 이외 URL 부분까지 제한하지 않는 것으로 불필요한 호환성 영향을 줄인다.

구현 순서와 체크포인트(모두 미착수, 사람 확인 불필요):
1. 위 3개 파일의 지점을 읽고 `go test ./internal/tracking ./internal/api`로 시작 상태를 확인한다. 계획과 실제 코드가 다르면 과제서를 먼저 수정한다.
2. 길이 검사와 테스트를 하나의 작업 단위로 넣고, 기존 경고 주석을 함께 갱신한다. `gofmt -w internal/tracking/tracking.go internal/tracking/tracking_test.go internal/api/tracking_test.go` 후 `go test ./internal/tracking ./internal/api` 통과가 다음 단계 조건이다. 새 회귀가 검사 배선 제거 시 실패하는지도 구현자가 확인하되 최종 작업 트리에는 올바른 검사를 복구한다.
3. 같은 테스트 통과와 `git diff --check`를 확인하고 프로덕션 변경이 1파일인지 검토한다. 정찰은 어떤 코드도 수정하지 않았으며 이 순서는 구현자용이다.

작업량 근거(pmo 추정): 검증 경로 확인 3~5분 + 검사/오류/주석 7~10분 + 경계·헤더/API 테스트 10~15분 = 기본 20~30분의 bottom-up 추정이다. 알려진 변동(긴 경로 fixture와 기존 주석 조정)에 예비 5~10분을 별도로 두어 총 25~40분, 신뢰도 중간의 공학적 예상이며 통계적 보장은 아니다. 유사한 09-30·10-04 회차도 프로덕션 1파일+동일 두 테스트 패키지로 끝나 크기 S를 뒷받침하지만 소요 시간 자료는 없으므로 시간의 독립 교차 검증으로 쓰지 않는다. 새 범위용 관리 예비는 0분이며 호환성 정책/전체 바이트 제한으로 커지면 다음 회차로 분리한다. 인프라·마이그레이션·브라우저·배포는 포함하지 않는다.

정찰 근거: HEAD `68d4f7c`(v0.258.0). 실제 읽은 `tracking.go:originOf/PolicySources/Validate`, `api/tracking.go:pagePolicy/cachedTrackingSettings/allowTrackingOrigin`, `api/routes.go:validateSetting`에 근거한다. 8000룬 호스트의 24339바이트 측정치는 현 소스 주석의 기록이며 정찰에서 새로 측정한 수치가 아니다. 지정한 새 회귀 테스트에서 구현자가 재측정한다.
