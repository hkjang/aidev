# 회차 노트 2026-10-18-122857-AgentHub-improve — AgentHub
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:29] base pinned — main@42b592c
- [러너 12:29] autonomy release — 

## 정찰 노트
- 1순위 보류 아이디어("배포 base 태그가 낡았다")는 릴리즈 42b592c 가 값을 v0.27.0 으로 맞춰 전제가 반쯤 사라졌다 — 남은 본체인 "보초가 없다" 만 떼어 과제로 삼았다. 보초 둘 자리가 기존 테스트(release_version_test.go)의 checks 슬라이스 안이라 테스트 1파일·프로덕션 0파일로 끝나서 다른 후보(guide-shots 요약, dlp direction, npm 경고)를 제쳤다.
- 확신하는 것: 배포 산출물의 `agenthub-base:v*` 는 ollama-bridge.yaml:19 과 README.md:173 둘뿐(전수 grep), 카탈로그에 배포 매니페스트 필드가 없어 쉘 게이트로는 못 덮는다(runtime-images.json:26-27), 해당 테스트는 지금 `ok 0.002s` 로 통과한다(실행함).
- 추측으로 적은 것: `runtime-images.json` sourcePaths 에 `internal/buildinfo/` 가 있는지 **확인하지 않았다** — 구현자는 커밋 뒤 check-versions 를 돌려 상향이 불필요함을 직접 볼 것. 차선 후보(guide-shots 요약)의 행 번호도 재확인하지 않았다.
- 구현자가 조심할 것: 전수 스캔 정규식으로 `agenthub-base:v*` 를 훑지 말 것 — 테스트 픽스처의 가짜 태그 6파일이 거짓 실패를 낸다. 두 배포 파일의 **값은 건드리지 말 것**(지금 맞고, 값 갱신은 릴리즈 세션의 일).
- [러너 12:33] scout done — 릴리즈 일관성 검사가 배포 산출물의 `agenthub-base:v<BASE_VERSION>` 태그도 지킨다 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 무엇을 왜: 커밋 a6ecf90, 테스트 1파일만(`internal/buildinfo/release_version_test.go`, +32/-2, 프로덕션 0). `BASE_VERSION` 을 `VERSION` 과 같은 방식으로 읽고 공유 `stable` 정규식으로 형식까지 보게 한 뒤, `checks` 에 `ollama-bridge.yaml` 을 새 항목으로, README 는 기존 항목의 `want` 에 덧붙여 base 태그를 지키게 했다.
- 보초 없음을 먼저 증명했다: 두 태그를 v0.26.0 으로 되돌려도 고치기 전 테스트는 `ok … 0.002s`. 수용 기준의 변이 3종 + `-dev` 형식 1종을 각각 관측하고 전부 `git checkout --` 로 복구, `git status` 로 원상 확인.
- 확신 없는 곳: **없음에 가깝다** — 돌린 것은 전부 실제 출력으로 확인했다. 다만 과제서가 미확인으로 남긴 sourcePaths 는 이번에 확정했다(jq 로 전체 뽑아 `internal/buildinfo` 부재 확인 + 커밋 후 `check-versions` exit 0, 지목 이미지 없음).
- 검증 못 한 것: npm 계열·브라우저 e2e·live DB(AGENTHUB_TEST_DSN 없어 skip)·실제 minikube 배포는 돌리지 않았다. 이 변경은 Go 테스트 파일 하나라 그 경로에 닿지 않는다.
- 일부러 안 한 것 ①: 배포 산출물의 **값**을 건드리지 않았다(둘 다 이미 v0.27.0 으로 맞고, 매니페스트 버전 갱신은 릴리즈 세션의 일). ② `scripts/release-catalog-images.sh` 를 넓히지 않았다 — 같은 종류 검사를 두 경로에 통합하지 말라는 지시. ③ 전수 스캔 규칙을 쓰지 않았다 — 픽스처 가짜 태그가 거짓 실패를 낸다.
- 비평가가 먼저 볼 것: 루프의 `%s does not reference release %s through %q` 가 중간 `%s` 에 언제나 `version`(0.264.0)을 넣으므로 base 항목의 실패 줄은 `release 0.264.0 … "…agenthub-base:v0.27.0"` 로 찍힌다 — 파일과 빠진 문자열은 정확하지만 번호는 그 줄과 무관하다. 고치려면 `checks` 구조체에 필드를 더해 기존 7개 항목을 모두 손대게 되어(과제서가 금지한 재배열) 일부러 두고 ideas.json 에 후보로 적었다.
- 다음 역할이 조심할 것: `go test ./internal/buildinfo` 전체는 로컬에서 **30초**(9p gitdir 의 `git show`, CI 에는 없는 비용). 빠른 확인은 `-run TestReleaseVersionIsConsistentAcrossDeploymentSurfaces`. `check-versions` 는 HEAD **커밋**을 읽으므로 작업 트리로는 재현되지 않는다.
- [러너 12:39] brief accepted — 채택 — 근거·수용 기준·건드릴 파일이 현재 코드와 정확히 일치했다(`BASE_VERSION`=0.27.0, 두 배포 산출물 모두 v0.27.0 이고 
- [러너 12:40] verify passed — 검증 5개 통과 (auto)

## 비평 노트
- 판정 approve / risk low / blocking 없음. 재현을 저장소 밖(/tmp 사본)에서 직접 했다: BASE_VERSION 만 0.28.0 으로 올리면 README.md·ollama-bridge.yaml 두 줄이 red, 되돌리면 green — 원장의 변이 ③ 출력과 일치한다. 어서션은 strings.Contains 뿐이고 skip·항상 참인 조건이 없다.
- 전제가 가상이 아님을 역사로 확인했다: bb0c720 이 BASE_VERSION 0.26.0→0.27.0 을 올리며 compose·buildinfo 만 맞추고 두 배포 파일을 0.26.0 에 두고 갔으며, 릴리즈 42b592c 가 뒤늦게 따라잡았다. check_version_mirrors(release-catalog-images.sh:629-653)는 그 둘을 보지 않으므로 겹치는 검사가 아니다.
- 전수 grep 으로 agenthub-base 를 가진 19개 파일을 모두 열어 버전 태그가 박힌 배포 표면이 정말 둘뿐임을 확인했고, jq 로 sourcePaths 79개에 internal/buildinfo 가 없음도 확인(BASE_VERSION 상향 불필요). 범위 이탈·위험 변경 없음.
- 남는 우려: :72 실패 문구의 중간 %s 가 base 실패에도 version(0.264.0)을 넣어 번호가 무관하게 찍힌다 — 조치는 가능하니 비차단, 다음 회차의 별 과제. 릴리즈는 이제 BASE_VERSION 상향 시 README.md:173 과 ollama-bridge.yaml:19 을 같은 커밋에 고쳐야 하고, 0.28.0-rc1 류 프리릴리즈 base 번호는 stable 정규식에 막힌다.
- 못 본 것: npm 계열·브라우저 e2e·live DB·실제 minikube 배포·release-catalog-images.sh 전체 실행(이 변경이 닿지 않는 경로). go test ./... exit 0 은 verify.json 으로 확인, 좁힌 보초는 직접 재실행했다.
- [러너 12:43] review approved — 리뷰 승인 (risk=low)
- [러너 12:43] pr created — https://github.com/hkjang/AgentHub/pull/48
- [러너 12:46] ci passed — 검사 1개 모두 success
- [러너 12:46] merge done — a6ecf90
