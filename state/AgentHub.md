## 2026-09-02
- 선택: 관리자 서버 로그 검색이 화면에 보이는 필드를 찾지 못하던 문제 수정 + `internal/logging` 첫 테스트 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공 — PR https://github.com/hkjang/AgentHub/pull/1 (main 머지됨)
- 요약: 운영 화면은 로그 한 줄에 message와 structured fields를 함께 출력하는데 서버의 `Ring.Entries`는 message와 source만 훑고 있어, 화면에 보이는 agent 이름·에러 문자열을 그대로 입력하면 결과가 0건이었습니다. `matches()`로 필드 키·값까지 검색하도록 고치고, 저장소에서 유일하게 테스트가 0개였던 `internal/logging`에 링 랩어라운드·limit·레벨 필터·검색·capture 핸들러·동시성 테스트 9개를 추가했습니다. 검증: go vet, go test -race ./cmd/... ./internal/..., web npm ci+lint+build 모두 통과.
- 보류 아이디어:
  - dlp.notPhone의 앞 두 조건이 죽은 코드이고 0으로 시작하는 실제 계좌번호를 전부 놓침 (3/2/S)
  - korean.EndsInConsonant가 괄호·따옴표로 끝나는 값에서 조사를 잘못 고름 (2/2/S)
  - captureHandler.WithGroup이 그룹 이름을 버려 서로 다른 그룹의 같은 키가 충돌 (2/1/S)
  - runInfoCommand가 version 외 인자를 run()으로 흘려보내 `agenthub --help`가 DB 오류로 실패 (2/1/S)
- 릴리즈: v0.226.0 (2026-09-02)

## 2026-09-02 (2차)
- 선택: DLP 계좌번호 검출기가 0으로 시작하는 계좌번호를 전부 놓치던 문제 수정 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공 — 커밋 4e240c5 (auto/2026-09-02-1730)
- 요약: `notPhone`이 계좌와 전화번호를 "0으로 시작하지 않을 것"으로 갈랐는데, 앞 두 조건(`!HasPrefix("01")`, `!HasPrefix("02")`)은 뒤의 `!HasPrefix("0")`에 이미 먹혀 죽은 코드였고 살아있던 조건은 틀린 판정이었습니다 — 기업은행·우체국·상당수 국민은행 계좌가 0으로 시작하므로 계좌번호를 '차단'으로 설정한 사이트가 그 은행들만 조용히 통과시키고 있었습니다(찾지 못한 것은 감사 로그에 아무 흔적도 남지 않는 실패). 체크섬이 없는 두 값을 실제로 가르는 것은 자릿수 묶음이라, 한국 전화번호 모양(0으로 시작하는 2~4자리 지역번호 - 3~4자리 국번 - 정확히 4자리, 세 묶음)일 때만 계좌에서 제외하도록 고쳤습니다. `internal/dlp`는 런타임 base 이미지 소스라 BASE_VERSION도 0.16.0으로 올렸습니다(5곳). 검증: 0으로 시작하는 계좌 3종·전화번호 4종·날짜 리터럴 테스트 추가, go vet, go test -race ./cmd/... ./internal/..., web npm ci+lint+build, `scripts/release-catalog-images.sh check-versions` 모두 통과.
- 보류 아이디어:
  - 사업자등록번호가 계좌번호로도 중복 집계됨(220-81-62517이 두 등급으로 보고되어 운영자가 읽는 건수가 부풀려짐) (3/2/S)
  - korean.EndsInConsonant가 괄호·따옴표로 끝나는 값에서 조사를 잘못 고름 (2/2/S)
  - captureHandler.WithGroup이 그룹 이름을 버려 서로 다른 그룹의 같은 키가 충돌 (2/1/S)
  - runInfoCommand가 version 외 인자를 run()으로 흘려보내 `agenthub --help`가 DB 오류로 실패 (2/1/S)
- 릴리즈: v0.227.0 (2026-09-02)

## 2026-09-03
- 선택: DLP에서 한 값이 두 등급으로 집계되고 잘못된 이름으로 마스킹되던 문제 수정 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공 — 커밋 493029f (auto/2026-09-03-0010)
- 요약: 검출기들이 각자 전체 페이로드를 독립적으로 매칭해 같은 값을 두 개가 가져갈 수 있었습니다. 계좌번호는 체크섬이 없고 자릿수 묶음만 보므로 카드번호(4111-1111-1111-1111)·사업자등록번호(220-81-62517)가 전부 계좌번호로도 잡혀 운영자가 읽는 "계좌번호 N건"이 부풀려졌고, 더 나쁘게는 계좌번호 검출기가 사업자등록번호보다 먼저 돌아서 회사의 등록번호가 `[계좌번호 삭제됨]`으로 나갔습니다 — 당사자가 읽는 유일한 자리에서 자기 데이터에 대해 틀린 말을 한 셈입니다. 검출기 목록 위 주석은 이미 "구체적인 것이 먼저 돌아 일반적인 것이 그 매치를 삼키지 않는다"고 적혀 있었지만 그렇게 하는 코드는 없었습니다. `Scan`이 각 검출기가 가져간 바이트 구간을 기록해 이미 claim된 후보를 건너뛰게 하고, 사업자등록번호를 계좌번호 앞으로 옮겼습니다(off인 등급은 아예 돌지 않으므로 claim도 하지 않아, 계좌번호만 켠 사이트는 그대로 탐지됩니다). 검증: 중복집계·마커 이름·공존 케이스 테스트 3개 추가, go vet ./..., go test -race ./cmd/... ./internal/..., web npm ci+lint+build, `scripts/release-catalog-images.sh check-versions` 모두 통과. internal/dlp는 런타임 base 이미지에 들어가므로 BASE_VERSION 0.17.0으로 상향(5곳).
- 보류 아이디어:
  - Scan의 redaction이 위치가 아니라 strings.ReplaceAll로 치환해, 같은 문자열이 다른 맥락에 있으면 함께 지워짐 (3/3/M)
  - korean.EndsInConsonant가 괄호·따옴표로 끝나는 값에서 조사를 잘못 고름 (2/2/S)
  - captureHandler.WithGroup이 그룹 이름을 버려 서로 다른 그룹의 같은 키가 충돌 (2/1/S)
  - runInfoCommand가 version 외 인자를 run()으로 흘려보내 `agenthub --help`가 DB 오류로 실패 (2/1/S)
- 릴리즈: v0.228.0 (2026-09-03)

## 2026-09-03 (2차)
- 선택: DLP redaction이 검출하지 않은 텍스트까지 함께 지우던 문제 수정 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공 — 커밋 14ee633 (auto/2026-09-03-0610)
- 요약: `Scan`은 값을 위치로 찾아 놓고 치환은 `strings.ReplaceAll`로 값 단위로 했습니다. 숫자 모양이 서로 포개지기 때문에 같은 편집이 아닙니다 — 계좌번호 111-1111-1111은 카드번호 4111-1111-1111-1111의 부분 문자열이라, 둘이 함께 있는 페이로드가 "카드 4[계좌번호 삭제됨]-1111 이고 사번 [계좌번호 삭제됨]"로 나갔습니다. 카드는 audit(그대로 통과)로 설정돼 있었는데도 반토막이 났고, 감사 기록은 "계좌번호 1건"이라고 맞는 말을 하는 동안 텍스트는 두 곳이 바뀌어 있었습니다 — 어느 쪽이 틀렸는지 화면에 아무 표시가 없는 불일치입니다. 이제 redact 대상 매치의 바이트 구간을 (이미 있는 claim과 함께) 기록해 한 번의 좌→우 패스로 치환합니다(검출기 실행 순서가 페이로드 순서와 달라 정렬은 필요). 검증: 중첩 케이스·다중 등급 in-place 치환 테스트 2개 추가, go vet ./..., go test -race ./cmd/... ./internal/..., web npm ci+lint+build, `scripts/release-catalog-images.sh check-versions` 모두 통과. internal/dlp는 런타임 base 이미지 소스라 BASE_VERSION 0.18.0으로 상향(5곳).
- 보류 아이디어:
  - MaxBytes 절단이 민감값 한가운데를 자르면 양쪽 다 매치되지 않아 조용히 새어나감 (3/2/S)
  - korean.EndsInConsonant가 괄호·따옴표로 끝나는 값에서 조사를 잘못 고름 (2/2/S)
  - captureHandler.WithGroup이 그룹 이름을 버려 서로 다른 그룹의 같은 키가 충돌 (2/1/S)
  - runInfoCommand가 version 외 인자를 run()으로 흘려보내 `agenthub --help`가 DB 오류로 실패 (2/1/S)
- 릴리즈: v0.229.0 (2026-09-03)

## 2026-09-03 (3차)
- 선택: DLP 감사 기록이 아무것도 지우지 않은 건까지 "redacted"로 남기던 문제 수정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공 — 커밋 4766b0d (auto/2026-09-03-1130)
- 요약: 세 boundary(모델·흐름 검사기, 컨트롤 플레인의 Pod 게이트웨이 보고 엔드포인트, 게이트웨이 자체 로그)가 모두 `outcome := "redacted"; if blocked { "blocked" }`로 시작해, 차단하지 않은 모든 findings를 '가림 처리'로 기록했습니다 — 등급이 `기록만`이라 페이로드가 손대지 않고 그대로 나간 건까지 포함해서. `기록만`은 "차단을 켜기 전에 우리 에이전트가 실제로 무엇을 다루는지 배우는 방법"으로 문서에 적힌 온보딩 경로라, 그 안내를 따른 사이트일수록 감사 기록의 모든 줄이 "플랫폼이 트래픽을 고쳤다"고 거짓말했습니다. 그 기록이 바로 등급을 `기록만`→`가리고 전송`으로 올릴지 결정할 때 읽는 자료입니다. `Result.Outcome()`이 실제로 한 일에서 단어를 뽑도록 하고(blocked / Redact 건이 하나라도 있으면 redacted / 그 외 audited), 정책 차단은 여전히 우선하게 두었습니다. 게이트웨이는 처음부터 finding마다 action을 함께 보내므로 옛 base 이미지의 Pod도 올바로 분류됩니다. 콘솔에는 두 단어의 한국어 라벨이 없어 감사 표에 영문이 그대로 찍혔으므로 라벨을 추가하고 결과 필터에 차단됨·가림 처리됨·기록만을 넣었습니다. 검증: dlp 4케이스 테이블 테스트 + 게이트웨이 통합 테스트(감사 전용 호출이 원문 그대로 나가고 로그는 audited) + guard 소스 가드, go vet ./..., go test -race ./cmd/... ./internal/..., web npm ci+lint+build, `release-catalog-images.sh validate`·`check-versions` 모두 통과. internal/dlp·cmd/runtime-proxy가 base 이미지 소스라 BASE_VERSION 0.19.0으로 상향(5곳).
- 보류 아이디어:
  - MaxBytes 절단이 민감값 한가운데를 자르면 양쪽 다 매치되지 않아 조용히 새어나감 (3/2/S)
  - korean.EndsInConsonant가 괄호·따옴표로 끝나는 값에서 조사를 잘못 고름 (2/2/S)
  - captureHandler.WithGroup이 그룹 이름을 버려 서로 다른 그룹의 같은 키가 충돌 (2/1/S)
  - runInfoCommand가 version 외 인자를 run()으로 흘려보내 `agenthub --help`가 DB 오류로 실패 (2/1/S)
- 릴리즈: v0.230.0 (2026-09-03)

## 2026-09-03 (4차)
- 선택: DLP 검사 크기 상한이 민감값을 반으로 자르던 문제 수정 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공 — 커밋 2181085 (auto/2026-09-03-1920)
- 요약: `Scan`이 `text[:limit]`으로 정확히 바이트 수에서 잘랐는데, 그 자리가 어디인지는 페이로드 길이가 정하는 우연입니다. 반토막 난 주민등록번호는 아무 패턴에도 맞지 않으므로, 주민번호를 '차단'으로 설정한 배포가 그것을 그대로 내보내면서 findings는 0건이었습니다 — 남는 흔적은 나머지가 깨끗해 보이는 감사 항목에 붙은 truncated 플래그 하나뿐입니다. 상한 뒤쪽을 검사하지 않는 것은 상한의 목적이지만, 검사 구간 안에서 시작한 값은 운영자가 봐 달라고 한 값입니다. 반대 방향으로도 잘랐습니다: 24자리 숫자는 카드번호가 아니지만 앞 16자리는 Luhn을 통과할 수 있어, 바이트 수에서 멈추는 것만으로 페이로드에 없던 후보를 만들어 냈습니다(그 등급이 차단이면 실제로 호출이 거부됨). 이제 상한 지점부터 매치가 담을 수 없는 첫 바이트까지 창을 앞으로 늘리고(어떤 모양이든 멈추도록 128바이트로 제한), Truncated는 실제로 남은 것이 있는지로 정합니다. 패턴이 기대는 `\b`도 함께 복원돼 창의 마지막 값이 접두사가 아니라 자기 자신으로 매칭됩니다. 검증: 반토막 케이스·허위생성 케이스 테스트 2개 추가(수정 전 둘 다 실패 확인), go vet ./..., go test -race ./cmd/... ./internal/..., web npm ci+lint+build, `scripts/release-catalog-images.sh check-versions`·`validate` 모두 통과. internal/dlp는 런타임 base 이미지 소스라 BASE_VERSION 0.20.0으로 상향(5곳).
- 보류 아이디어:
  - scrubDecision이 record.Agent 등 남은 자유 텍스트를 검사하지 않아 에이전트 이름으로는 무엇이든 나갈 수 있음 (2/2/S)
  - korean.EndsInConsonant가 괄호·따옴표로 끝나는 값에서 조사를 잘못 고름 (2/2/S)
  - captureHandler.WithGroup이 그룹 이름을 버려 서로 다른 그룹의 같은 키가 충돌 (2/1/S)
  - runInfoCommand가 version 외 인자를 run()으로 흘려보내 `agenthub --help`가 DB 오류로 실패 (2/1/S)
- 릴리즈: v0.231.0 (2026-09-03)

## 2026-09-04
- 선택: GPU Quota가 저장·표시되지만 실제로는 한 번도 적용되지 않던 문제 수정 (가치 5 / 위험 1 / 작업량 S)
- 결과: 성공 — 커밋 3479d73 (auto/2026-09-04-0300)
- 요약: `MaxGPUs`는 프로파일의 GPU 수가 Pod까지 도달하면서 `Limits`에 추가됐는데, 정작 중요한 루프 하나에만 들어가지 않았습니다 — `quota.Resolve`는 플랫폼·부서·개인 세 단계를 필드 단위로 병합하고, 그 목록에 GPU가 없어서 사용자 범위의 `CheckHeld`에 넘어가는 한도의 GPU는 항상 0이었습니다. 이 패키지에서 0은 '무제한'입니다. 겉으로는 아무것도 고장나지 않았습니다: 관리자가 설정 화면에 숫자를 넣으면 저장되고, store가 Platform으로 읽어 오고, CheckHeld에는 멀쩡한 검사가 기다리고 있었습니다. 다른 차원은 전부 제대로 거절했고, 하룻밤에 더 사올 수 없는 가장 희소한 자원 — 이 필드가 추가된 이유 그 자체 — 만 한 사람이 클러스터의 카드를 전부 차지할 수 있었습니다. 콘솔도 자기 목록에서 같은 식으로 한 차원 모자랐습니다: `LIMIT_FIELDS`에 행이 없어 부서·개인에 GPU 상한을 아예 설정할 수 없었고, "실제 적용되는 한도" 표와 사용자 본인의 사용량 패널에도 나오지 않았습니다(부서 총량은 Resolve를 거치지 않고 Total을 직접 읽어 강제되고 있었으므로, 요약 화면 어디에도 보이지 않으면서 강제되는 상태였습니다). 차원별로 나열해 쓴 `quotaComplaint`도 음수 GPU를 통과시켰는데, 그 값은 깨끗하게 저장된 뒤 무제한으로 읽힙니다. Resolve가 MaxGPUs를 옮기고, LIMIT_FIELDS가 GPU 행을 그리고, 검증기가 음수와 비상식적인 값을 거절하도록 고쳤습니다. 이 버그를 통과시킨 기존 테스트들은 CheckHeld를 직접 불렀으므로 새 테스트는 Resolve를 거치게 했고, reflection으로 Limits 구조체를 순회하는 sweep 두 개(모든 필드가 resolve를 살아남을 것, 모든 필드가 음수일 때 거절될 것)와 콘솔 목록이 서버보다 뒤처지면 실패하는 크로스티어 테스트를 추가했습니다. `internal/quota`·`internal/api`는 런타임 base 이미지 소스가 아니라 BASE_VERSION 상향은 불필요합니다. 검증: 수정 전 새 테스트 3개가 실패하는 것 확인, go vet ./..., go test -race ./cmd/... ./internal/..., web npm ci+lint+build, `scripts/release-catalog-images.sh check-versions`·`validate` 모두 통과.
- 보류 아이디어:
  - scrubDecision이 record.Agent 등 남은 자유 텍스트를 검사하지 않아 에이전트 이름으로는 무엇이든 나갈 수 있음 (2/2/S)
  - korean.EndsInConsonant가 괄호·따옴표로 끝나는 값에서 조사를 잘못 고름 (2/2/S)
  - captureHandler.WithGroup이 그룹 이름을 버려 서로 다른 그룹의 같은 키가 충돌 (2/1/S)
  - runInfoCommand가 version 외 인자를 run()으로 흘려보내 `agenthub --help`가 DB 오류로 실패 (2/1/S)
- 릴리즈: v0.232.0 (2026-09-04)
