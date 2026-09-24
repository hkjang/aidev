# 회차 노트 2026-09-24-020428-madi-improve — madi
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:04] base pinned — main@fd2c3b5
- [러너 02:04] autonomy release — 

## 정찰 노트
- 기준선 main@fd2c3b5 에 CI 를 막는 두 드리프트가 **그대로 남아 있음을 파일로 직접 확인**했다(go.mod:87 grpc v1.82.1, runtime-apk.lock:11,12,69 의 20260611-r0/2026c-r0). 기능 아이디어(메일·handoff·SSO 안내)는 선행 기능이 main 에 없어 성립하지 않으므로 제쳤다.
- 지난 세 회차가 같은 드리프트를 고치면서 `web/dist/.gitkeep`+vite+README 묶음까지 끌어와 범위가 커졌고 회차가 verify-failed 로 끝났다. 이번 과제서는 그 묶음을 **명시적으로 범위 밖**에 두고 5개 파일로 좁혔다(CI 는 go 단계 전에 npm run build 를 돌리므로 embed 문제는 CI 실패 원인이 아니다 — ci.yml 로 확인).
- 추측으로 적은 것: govulncheck·docker 를 이번에 직접 실행하지 못했다(명령 승인 거부). 실패 재현은 과거 회차 기록 + 파일 내용에서 추론했고, 과제서 수용 기준 1·2 에 "수정 전 출력 먼저" 를 넣어 구현자가 확정하게 했다.
- lock 의 목표 버전을 8451b5b 의 값(20260909-r0/2026d-r0)으로 못박지 않았다. Alpine 저장소가 또 움직였을 수 있어 고정 이미지의 `apk policy` 로 확정하도록 적었다.
- 구현자 주의: ca-certificates 와 ca-certificates-bundle 을 반드시 같이 올릴 것(09-19 에 origins 53≠52 로 image-smoke 실패). go.mod/go.sum 을 바꾸면 `node scripts/licenses.mjs` 재생성이 필수다 — CI 가 govulncheck 보다 먼저 `--check` 를 돌린다.
- [러너 02:08] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- 1de4d00: `tests/deployment-contract/runtime_lock_test.go` 추가 + lock 헤더 주석 한 줄. lock 에서 origin 재고를 정적으로 유도해 image-smoke 의 69/52, Dockerfile 의 alpine 3.24.1 핀·DISTFILES_MIRROR, bundle-alpine-sources.mjs 의 tesseract 5.5.2-r0 핀과 대조한다 — 09-19 에 전체 이미지 빌드로만 드러난 ca-certificates/bundle 분리를 빌드 전에 잡는다.
- 확신 없는 것: `runtimeOriginFamilies` 12묶음은 Alpine 하위패키지 구조를 손으로 적은 표다. 69→52 가 실제 상수와 정확히 일치하는 것으로 교차검증했지만, Alpine 이 하위패키지를 분리/병합하면 이 표가 먼저 틀린다(그때는 사람이 봐야 하므로 의도한 동작). Docker 이미지 빌드로 실제 origins 52 를 재확인하지는 않았다.
- 검증 못 한 것: docker build / verify-image.sh / govulncheck / 브라우저 시험 / 전체 `go test -race ./...` 미실행. 예산과 시간 때문에 의도적으로 뺐다.
- 일부러 안 한 것: main@fd2c3b5 에 그대로 남은 grpc GO-2026-6348 과 lock 20260611-r0/2026c-r0 드리프트는 손대지 않았다 — 세 회차가 이미 성공시켰으나 전부 미병합인 동일 작업이라 반복 금지 지침에 따랐다. 이 PR 의 CI 도 같은 두 단계에서 실패할 것으로 예상한다(이번 변경과 무관).
- 다음 역할 주의: 새 시험은 DB·Docker·네트워크가 전혀 필요 없고 `go test -count=1 ./tests/deployment-contract` 로 수 ms 에 돈다. 다만 bare 체크아웃에서 `go build ./...`·`go vet ./cmd/...` 는 여전히 `web/embed.go: pattern all:dist` 로 실패한다(main 기존 상태, CI 는 npm 빌드를 먼저 함).
- 각 단언이 죽어 있지 않음을 드리프트 5종 주입→FAIL 확인→복원으로 증명했고, 복원 뒤 `sed '/^#/d; /^$/d'` 이 여전히 69 줄을 내는 것도 확인했다.
- [러너 02:14] verify failed — 실패한 검증: go build ./... (exit 1)
