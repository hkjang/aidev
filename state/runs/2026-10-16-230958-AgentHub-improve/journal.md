# 회차 노트 2026-10-16-230958-AgentHub-improve — AgentHub
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 23:10] base pinned — main@bfe7b8a
- [러너 23:10] autonomy release — 

## 정찰 노트
- DLP 거절 로그를 골랐다 — main(bfe7b8a)에 그 한 줄이 **없음을 확인**했고(dlp.go:send 가 StatusCode 를 안 봄, 세 거울 모두 0.26.0), 지난번 이 변경을 깨뜨린 원인이 0fd2810 의 check_version_mirrors 로 이제 막히므로 같은 사고가 반복되지 않는다. 차선(deploy/README 의 박힌 태그)은 픽스처 가짜 태그 11곳 때문에 규칙 설계가 필요해 S 로 끝나기 어려워 제쳤다.
- 과제서가 추측으로 적은 것: 없음. 거울 보초의 위치는 내가 처음 짚은 경로가 틀려 실제로 열어 고쳤다 — internal/api/imagesources_test.go:403 과 internal/runtime/spawner_baseimage_test.go:82 이다(`./cmd/runtime-proxy` 만으로는 거울 누락이 안 보인다는 점이 핵심).
- 구현자가 조심할 것 ①: 세 거울(BASE_VERSION / compose.yaml:29 / buildinfo.go:12)을 한 커밋에 올리고, 올린 **뒤에** Go 스위트를 다시 돌릴 것 — 지난번 사고는 순서 때문이었다.
- 구현자가 조심할 것 ②: 6059496 을 cherry-pick 하지 말고 새로 쓸 것(운영자 지시). send 의 best-effort 계약은 손대지 말고, 거절 카운트·요약은 범위 밖이다.
- 미확인: check-versions/validate 는 이번에 돌리지 않았다(스크립트 실행 권한·시간). npm·브라우저·DB·배포도 미실행. 코드는 한 줄도 바꾸지 않았다.
- [러너 23:14] scout done — 런타임 게이트웨이가 거절당한 DLP 보고를 로그로 남긴다 — 이번에는 세 버전 거울을 함께 올려서 (가치 3

## 구현 노트
- 커밋 bb0c720. `cmd/runtime-proxy/dlp.go:send` 가 2xx 밖의 응답에 `agenthub-dlp report rejected: <status>` 한 줄을 남긴다(상태 문자열만 — 응답 본문에는 방금 보고한 이벤트가 되돌아올 수 있다). `cmd/runtime-proxy` 가 base sourcePaths 라 `BASE_VERSION`·`compose.yaml:29`·`internal/buildinfo/buildinfo.go:12` 을 **한 커밋에서** 0.27.0 으로 올렸고, 올린 **뒤에** `go test -count=1 ./...` 를 다시 돌려 두 거울 보초가 통과하는 것을 봤다(지난번 사고 지점).
- 확신 없는 곳: `deploy/kubernetes/ollama-bridge.yaml:19` 과 `README.md:173` 의 `agenthub-base:v0.26.0` 이 **이 커밋으로 낡았다**. 절대 규칙(배포 매니페스트 버전 갱신 금지)에 따라 값도 건드리지 않았으니 릴리즈 역할이 판단할 것 — 아무 테스트도 이 둘을 보지 않으므로 스위트는 조용하다.
- 검증 못 한 것: npm 계열·브라우저 e2e·live DB(AGENTHUB_TEST_DSN 없어 skip)·실제 이미지 빌드·배포. `docker compose config --quiet` 는 `.env` 가 없어 필수 환경변수 3개에서 막히는데 base(bfe7b8a)에서도 같고, 자리 표시자를 인라인으로 주면 base·HEAD 둘 다 exit 0 이다(`.env` 는 만들지 않았다). `kubectl kustomize` 는 `deploy/` 를 건드리지 않아 돌리지 않았다.
- 일부러 하지 않은 것: 거절 카운트·주기 요약(reporter 에 상태·시계를 들여야 해 best-effort 계약을 건드린다), `capturePodLog` 를 공용 파일로 빼기(사용자가 아직 한 명), 6059496 cherry-pick(운영자 지시대로 새로 썼다).
- 비평가가 알아야 할 것: 수용 기준 5 의 "올리기 전 거절" 은 작업 트리에서 재현되지 않는다 — `check_versions` 는 `${GITHUB_SHA:-HEAD}` 커밋을 읽으므로(스크립트 665-698행) 코드만 담은 임시 커밋으로 `::error::base: … BASE_VERSION is still 0.26.0`(exit 1)을 관측한 뒤 거울을 더해 amend 했고, 최종 HEAD 에서 exit 0 이다.
- 테스트 주의: `dlpreport_test.go` 는 `log.SetOutput` 으로 전역 로거를 바꾼다(t.Cleanup 으로 stderr·flags 복원). `t.Parallel` 을 붙이면 다른 테스트의 로그를 삼키니 붙이지 말 것. DB·클러스터는 필요 없다.
- [러너 23:24] brief accepted — 채택 — 결함·수용 기준·건드릴 파일·보초 위치가 현재 코드와 정확히 일치했다(`send` 가 StatusCode 를 보지 않는 것, 세 �
- [러너 23:25] verify passed — 검증 5개 통과 (auto)

## 비평 노트
- 실패 재현을 믿지 않고 직접 확인했다: main(bfe7b8a) 을 detach 워크트리로 꺼내 새 테스트 파일만 복사해 돌려 `dlpreport_test.go:89` 세 서브테스트와 `:159` 가 원장과 **똑같은 문구로** 떨어지는 것을 보고 워크트리를 지웠다. HEAD 에서는 `./cmd/runtime-proxy`·`./internal/api`·`./internal/runtime` ok, `-race -count=3` 반복도 안정, `gofmt`·`go vet` 깨끗, `validate`·`check-versions` 둘 다 exit 0(권한 없어 `bash` 로 실행).
- 로그 한 줄은 상태 문자열뿐이라 토큰·응답 본문·검출 원문이 들어갈 길이 없고(테스트가 세 가지를 모두 고정) 새 개인정보 수집·저장·전송이 없다 — 보안·법무 차단 사유 없음.
- 승인이어도 남는 우려(**릴리즈 역할이 가장 먼저 볼 것**): `README.md:173` 과 `deploy/kubernetes/ollama-bridge.yaml:19` 이 아직 `agenthub-base:v0.26.0` 이다. 직전 base 상향 커밋들(abd554b, 9f84183 …)은 다섯 파일을 한 커밋에 올렸고 이번엔 셋만 올랐다. 어떤 테스트·게이트도 이 둘을 보지 않으므로 스위트는 조용하며, 고치지 않으면 README 절차대로 만든 새 클러스터에서 ollama-bridge 가 `IfNotPresent` 로 없는 태그를 찾는다. 구현자는 절대 규칙(prompt.md:64)을 따라 일부러 두었으므로 책임은 릴리즈 쪽이다.
- 참고: `response.Status` 의 reason phrase 는 관제면이 정하는 문자열이다(Go 가 한 줄로만 읽으므로 개행 주입은 불가). 관제면이 신뢰 대상이라 차단은 아니지만, 다음에 상태 요약·카운트를 들이게 되면 그때 코드로만 쓰는 쪽을 고려할 것.
- 못 본 것: npm·브라우저 e2e·live DB(DSN 없음)·실제 이미지 빌드·클러스터 배포. `docker compose config` 도 `.env` 가 없어 돌리지 않았다.
- [러너 23:29] review approved — 리뷰 승인 (risk=medium)
- [러너 23:29] pr created — https://github.com/hkjang/AgentHub/pull/47
- [러너 23:32] ci passed — 검사 1개 모두 success
- [러너 23:32] merge done — bb0c720
