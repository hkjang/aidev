# 구현 검증 기록 (합성 자료)

커밋: `83d1dad` — `fix: restrict SLA and risk settings to integer inputs`.
작업 트리는 시작 시 clean; 변경은 settings.tsx 5행과 finding_ops_test.go 63행뿐이다.

## 수행한 검증

- `go test -run '^TestFindingOps(ParsingAndValidation|SettingsNumericContract)$' -count=1 -v ./internal/app`: Go 1.26.7, 상위 2 PASS, 신규 leaf 사례 51 PASS, SKIP 0. UI 수정 전에 실행했으며 Go 파일은 이후 변하지 않았다. 매 사례 새 findingOpsDefaultSettings 그룹 생성, 실제 validateFindingOpsSettings 호출. 결과: go-numeric-contract.log.
- `npm --prefix web ci --no-audit --no-fund`: 처음 Node 22.23.1, 이후 Node 26.11.1 확보 후 재설치. 최종 로그 web-ci-node26.log.
- `npm --prefix web test`: Node 26.11.1, 113 PASS/0 FAIL/0 SKIP. web-test.log.
- `npm --prefix web run typecheck`, `npm --prefix web run build`: Node 26.11.1, exit 0. 각각 web-typecheck.log/web-build.log.
- npm은 `toolchain/node_modules/node/bin`을 PATH 앞에 놓고 실행했다. `npm_config_script_shell=<이 디렉터리>/node26-shell`로 npm이 추가한 상위 .bin의 구버전 node가 하위 실행을 가로채지 않게 했다. 저장소 패키지/잠금/스크립트 변경 없음.
- `node scripts/verify-pentagi.mjs`: 변경 전후 원본 312파일 검증. `gofmt -w internal/app/finding_ops_test.go`만 수행; 최종 gofmt -l 빈 출력. `git diff --check` 성공.
- 추가 `prettier --check web/src/settings.tsx`: 실패. 포맷 결과를 HEAD와 비교해 기존 MCP/OIDC 3곳 차이만 확인(preexisting-format.diff). 이번 추가 행의 서식 차이 없음. 지정 필수 검증 명령에는 Prettier가 없으며 보호 경로의 무관한 서식 수정은 하지 않았다.

## 실제 앱 브라우저 시험

`browser-numeric-contract.py`는 저장소 Vite 앱을 그대로 실행하고 HTTP API만 Playwright로 합성한다. TSX 선언 추출·모듈 mock·소스 문자열 단언은 쓰지 않는다. `/api/auth/me`, `/api/settings`, tracking 보조 API 등에 합성 응답을 반환하고 실제 SettingsPage가 보내는 PUT JSON을 기록한다. 서버/API/DB의 통합 검증을 대체하지 않는다.

실행: web 디렉터리에서 Node 26으로 `node_modules/vite/bin/vite.js --host 127.0.0.1 --port 4179 --strictPort`, 이어 `python3 <이 디렉터리>/browser-numeric-contract.py green`. Chromium 경로 `/opt/google/chrome/chrome`. 브라우저 스크립트·캡처는 회차 산출물이며 저장소 시험 러너/CI에는 추가하지 않았다.

- 수정 전 browser-red.log: SLA critical_days에 1.5 타이핑, Tab blur, 저장 후 표시와 PUT이 1.5라 정수 단언 실패.
- browser-green.log: 1440×1000, 390×844 모두 9필드 타이핑/실제 clipboard 붙여넣기/Tab blur/PUT JSON 정수 단언, 각 필드 정수 하한/상한 값의 동일 전송 확인. EPSS 0/0.1/0.125/1을 타이핑/붙여넣기 모두 동일 전송 확인. SLA 초안을 유지한 채 risk 그룹을 저장하고 SLA 복귀 시 초안 12 유지 확인. 페이지 오류 0, 알려지지 않은 API 요청 0.
- browser-reverted.log: 프로덕션 변경 다섯 줄만 임시 제거한 뒤 원래 fractional PUT 실패 재현. try/finally로 수정본 복구 후 전체 green 재통과.
- 기존 Mantine 소스 NumberInput.mjs에서 기본 allowDecimal=true, clampBehavior=blur, allowDecimal=false → decimalScale=0 확인. 실브라우저에서 정수 필드의 1.5 타이핑은 소수점이 차단되어 15, 붙여넣기는 1이 됨. 별도 저장 전 숫자 변환 코드는 추가하지 않았으며 이 동작 차이는 후속 아이디어로 기록.
- green-risk-1440.png/green-risk-390.png: 실제 앱 화면, 합성 자료와 모의 API·DB 미검증 표시. 육안으로 데스크톱/모바일 입력·문구·배치 확인.
- 시험 하네스 초기에 tracking 보조 응답 스키마 누락과 dirty 배지로 달라지는 버튼 접근성 이름 때문에 중단됐다. 합성 응답과 시험 locator만 고쳤으며 앱 코드는 변경하지 않았다. 이런 하네스 오류를 결함 재현으로 세지 않았다.

## 한계와 제외

HUNTER_TEST_DSN이 없어 실제 서버/API/DB 저장·인증 권한·전체 race는 검증하지 않았다. Go 바이너리/vet/임베드 갱신·Docker·원격 CI·릴리즈는 범위 밖이며 실행하지 않았다. 프로덕션 변경은 옵션 배선뿐으로 이미 저장된 소수값 정리, 빈 문자열 변환, 서버 검증기 수정은 하지 않았다. 번들·캐시·캡처는 커밋하지 않았다.

요청 스킬 도구는 노출되지 않아 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/`의 completion-verification, systematic-debugging, test-driven-development SKILL.md를 읽고 적용했다.
