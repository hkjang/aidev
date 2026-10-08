- 과제: SLA·조치 우선순위 설정의 정수 필드 9개에서 소수 입력을 제한 (가치 3 / 위험 1 / 작업량 S)
- 왜: `web/src/settings.tsx:settingFields`는 서버가 정수만 받는 SLA 6개와 risk 3개에 `integer`를 선언하지 않아 `FieldForm`이 소수를 허용하고, `SettingsPage` 내부 `save`는 그 값을 그대로 PUT한다. 기존 정수 옵션을 연결하면 관리자가 저장 후에야 전체 그룹의 400 오류를 받는 입력을 화면에서 줄일 수 있다.
- 수용 기준:
  1) SLA `critical_days/high_days/medium_days/low_days/info_days/due_soon_days`와 risk `kev_boost/epss_boost/stale_after_days`가 `integer: true`를 선언하여 기존 `FieldForm`의 `allowDecimal={f.integer !== true}`로 전달된다. 앞의 다섯 SLA 필드는 map 공통 선언 한 곳에서 처리한다.
  2) `risk.epss_threshold`는 소수 허용을 유지하고 0·0.1·0.125·1이 유효하다. 모든 기존 min/max/default/label/description과 그룹 저장·초안 보존 동작은 유지한다. SLA 0은 비활성 기한이며 risk.stale_after_days의 하한은 1이다.
  3) DB 없는 서버 회귀는 매 사례 `findingOpsDefaultSettings()`에서 새 그룹을 만든 뒤 9개 각각의 범위 안 소수(예: 1.5)를 거절하고, 하한·상한 정수는 수락함을 증명한다. EPSS만 소수 허용인 예외도 검사한다. 검증기는 수정하지 않는다.
  4) 웹 기존 테스트·타입검사·빌드가 통과한다. 입력 동작은 실제 앱에서 정수 필드 1개 이상과 EPSS를 대조하고 가능하면 데스크톱/모바일 캡처를 남긴다(합성 자료임을 표시). 브라우저/DB 환경을 확보하지 못하면 미검증으로 명시하며 정적 분석이나 Go 테스트를 실제 입력 시험으로 표현하지 않는다.
- 건드릴 파일:
  - `web/src/settings.tsx:settingFields` (111~184행): 위 다섯 선언 위치에 `integer: true` 추가. 프로덕션 변경은 이 1파일뿐이다.
  - `internal/app/finding_ops_test.go:TestFindingOpsParsingAndValidation` 인접: `TestFindingOpsSettingsNumericContract`라는 순수 테이블 테스트 추가. 실제 `validateFindingOpsSettings`를 호출하고 성공/거절을 확인한다. 이미 import된 testing을 이용할 수 있으며 DB 도우미는 호출하지 않는다.
  - 읽기 전용 근거: `web/src/resources.tsx:Field`(integer?: boolean), `FieldForm`(1014~1023행); `internal/app/finding_ops.go:validateFindingOpsSettings`(34~59행); `internal/app/settings.go:validateSettings`의 sla/risk 분기; `web/src/settings.tsx:save`(471~548행)와 FieldForm 호출(689행).
- 검증 명령 (저장소 루트 기준):
  ```sh
  npm --prefix web ci
  npm --prefix web test
  npm --prefix web run typecheck
  npm --prefix web run build
  go test -run '^TestFindingOps(ParsingAndValidation|SettingsNumericContract)$' -count=1 -v ./internal/app
  git diff --check
  ```
  공식 기준은 Node 26/Go 1.26.5 이상이다. 변경한 Go 시험 파일만 gofmt한다. Go 실행 파일을 추가 검증한다면 빌드한 `web/dist/.`를 `internal/webassets/dist/`에 복사한 뒤 `go vet ./...`와 `go build ./cmd/hunter`를 실행한다. 정찰은 설치·빌드·임베드 갱신을 하지 않았다.
- 위험과 피할 것: 서버 계약·검증기·auth·DB 초기화·CI/workflows·원본 PentAGI·버전은 이 구현 과제의 범위 밖이다. inventory/agents/ai/security 그룹을 함께 고치지 않는다. `epss_threshold`만 소수 예외다. 소수를 저장 전에 임의 반올림/절삭하거나 빈 문자열을 0으로 치환하지 않는다. 설정은 일반 자원 폼과 달리 `resourceSubmitBody`를 쓰지 않으므로 그곳을 수정해도 이 문제가 해결되지 않는다. `.tsx` 직접 import를 현재 Node 순수 테스트 러너에 추가하지 말고, 선언 추출·공유 테이블·전체 폼 리팩터로 범위를 키우지 않는다. `includes('integer: true')`만 보는 테스트를 행동 검증으로 추가하지 않는다. 이미 설정된 소수값의 정리·API 클라이언트의 입력 변경은 포함하지 않는다.
- 차선 후보: 목록 CSV의 범위 밖 숫자 타임스탬프 RangeError 방어 (가치 2 / 위험 1 / S). 1순위가 이미 반영됐거나 전제가 틀린 경우에만 `web/src/list-export.ts:listCSV`와 기존 `web/tests/convenience.test.mjs`를 수정한다. `created_at=8640000000000001`에서 정찰이 실제 RangeError를 재현했다. ISO 변환은 Date.getTime()이 유한할 때만 하고, 그 외는 원래 숫자를 기존 csvCell에 전달한다. ±8640000000000000 경계/바깥값·일반 날짜·exportValue 우선·수식 방어를 검사한다. 실제 서버 자료가 이 범위 밖 숫자에 도달하는지는 미확인이므로 2순위다.

실행 순서와 점검점 (구현 전부 미착수):
1. 위 읽기 전용 경로를 재확인하고 순수 숫자 계약 테스트를 추가한다. 위 Go 명령으로 정수 범위와 EPSS 예외를 먼저 고정한다. 이 시험은 수정 전에도 통과해야 하며 UI 수정의 red/green 증명이라고 주장하지 않는다. 사람 승인 점검점 없음.
2. settings.tsx의 다섯 위치만 수정한 뒤 웹 test/typecheck/build를 실행한다. 설치한 Mantine 소스에서도 allowDecimal 옵션을 확인하고, 가능하면 실제 앱에서 타이핑/붙여넣기/blur 후 전송값을 확인한다. 점검점은 자동 검증 결과이며 사람 승인은 필요 없다. 전제가 다르면 과제서를 수정하고 이유를 남긴다.
3. 변경 파일과 diff를 검토하고 실행한 결과·실브라우저/DB 미검증을 구분해 회차 노트에 적는다. 보호 경로와 무관한 변경이 섞이면 제거한다.

대안 판단과 추정:
- 채택: 기존 필드 옵션 연결. 새 런타임 구성 없이 프로덕션 1파일로 문제에 직접 닿는다.
- 기각한 해결 방식: 서버에서 소수 허용/자동 절삭은 설정 계약을 바꾼다. 새 공통 숫자 검증 계층은 이번 9필드보다 범위가 크다. 현상 유지는 작은 비용을 아끼지만 저장 후 오류를 그대로 남긴다.
- 전제: 현재 FieldForm이 integer를 읽는 경로가 유지됨. 코드로 확인했으나 실제 브라우저의 타이핑/붙여넣기 결과는 미확인이다.
- 상향식 예상: 확인·회귀 8~10분, 선언 수정 2~3분, 설치/기존 검증 6~10분, 화면 확인·기록 6~10분 = 기본 22~33분. 알려진 설치/브라우저 변동에 예비 5~10분을 별도로 두어 총 27~43분, 정성적 중간 확신(통계적 확률 아님). 신규 DB/브라우저 인프라 구축·원격 CI/릴리즈는 제외한다. 관리 예비는 배정하지 않으며 새 범위는 후속 회차로 남긴다. 과거 회차에 실제 소요시간이 없어 유사사례 방식의 수치 교차검증은 불가하다.
- 추정 방법의 근거: 범위·작업 분해·가정·불확실성·실적 갱신을 분리한다는 [GAO Cost Estimating and Assessment Guide](https://www.gao.gov/products/gao-20-195g)의 절차를 적용했다. 위 분 단위 값은 GAO 제공치가 아니라 이 코드 범위에 대한 정찰자의 추정이다.

정찰 실측 (2026-10-08, main@9069bcf):
- 작업 트리 clean. Node v22.23.1/npm 10.9.8/Go 1.26.7. Node 26 기준과 로컬이 다르다.
- `npm --prefix web test`: 113 PASS, 0 FAIL, 0 SKIP.
- `go test -run '^TestFindingOpsParsingAndValidation$' -count=1 -v ./internal/app`: 1 PASS, 0 SKIP. 기존 시험은 소수 9필드 계약을 직접 검증하지 않는다.
- web/node_modules와 HUNTER_TEST_DSN 없음. 실제 브라우저·전체 DB 시험·typecheck/build·원격 CI·배포 미실행.
- CLAUDE.md/독립 roadmap/TODO 파일은 파일명 검색에서 미발견. README, 최근 git log -30, docs/release-v1.24.0.md·validation.md, CI/Pages/Release 구성과 관련 코드를 확인했다.
- 요청한 Skill 도구는 노출되지 않았다. 로컬 headcount의 pmo/estimating-and-contingency와 technology/implementation-planning·solution-exploration SKILL.md를 읽고 이 절차·분해·대안 판단에 적용했다.
