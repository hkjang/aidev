# 회차 노트 2026-10-04-221204-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 22:12] base pinned — main@e5422a1
- [러너 22:12] autonomy release — 
- [러너 22:17] scout done — v0.2.45 태그에서 두 번 연속 같은 이유로 실패한 릴리즈 게이트의 원인을 찾아 고치기 (가치 5 / 위험 2 / 작�

## 구현 노트
- 바꾼 것: `scripts/verify-release.sh` 의 HTTP 프로브 구간만. `probe_get | grep -F` 두 줄과 `login` 두 호출이 `set -Eeuo pipefail` 아래에서 아무 출력 없이 exit 1 하던 것을, 같은 assertion·같은 exit 1 을 유지한 채 실패 이유를 stderr 로 찍게 했다. 프로덕션 코드 0파일, 1파일 +32/-6, `.github/workflows/` diff 0.
- **확신 없는 곳**: (1) 원래 과제인 "두 번 실패한 릴리즈 게이트" 의 **원인을 못 찾았다** — `gh` 미인증으로 로그를 못 읽었고, 릴리즈 전용 세 게이트까지 포함해 모든 게이트를 로컬에서 실제로 돌렸는데 전부 통과했다. 트리가 아니라 러너 쪽 원인일 수 있다. (2) 버전 assertion 의 실패 **메시지 자체**는 돌려 보지 못했다 — 제목 assertion 과 `assert_probe_contains` 를 공유하고 그 쪽을 재현했지만, 라벨과 서빙 버전의 불일치는 Dockerfile 이 하나의 `VERSION` arg 를 라벨과 ldflag 에 같이 주므로 빌드 인자로 만들 수 없었다. (3) `npm audit` 는 실시간 DB 조회라 10-04 13:18 UTC 통과가 내일을 보장하지 않는다.
- 일부러 안 한 것: 워크플로 완화 0건(금지). `verify-product-ui.sh`·`fetch-plugin-test-fixtures.sh` 의 침묵 지점은 손대지 않았다(전자는 실패 시 `docker logs` 를 찍는 cleanup trap 이 이미 있다). 버전 마커·태그·릴리즈 노트 미변경.
- 다음 역할이 조심할 것: 이 변경은 **릴리즈 잡만** 실행하는 쉘 스크립트라 Go/웹 테스트가 지나지 않는다 — 검증은 실제 docker 빌드 + 실제 PostgreSQL 컨테이너로만 가능하다. `go test -race -p 1` 은 DSN 이 있어야 돌고(없으면 0.17s 로 스킵), `pluginhost` 는 네 플러그인 아카이브 env 가 있어야 실제로 돈다. 로컬 재현에 쓴 임시 이미지·아카이브·DB 컨테이너는 전부 지웠고 `git status` 는 커밋 외 깨끗하다.
- [러너 22:34] brief fallback — 차선 — 1단계(실패 로그 확정)가 `gh` 미인증으로 구조적으로 불가능했고, 과제서가 1·2순위로 지목한 A(npm audit/govulncheck �
- [러너 22:34] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 확인한 것: diff 전체(scripts/verify-release.sh 1파일 +32/-6), 새 assert_probe_contains 의 네 분기를 격리 하네스로 실제 실행 — 구현자가 "못 돌려봤다" 고 적은 **버전 assertion 실패 메시지까지 재현**했다(메시지·바이트수·본문·exit 1 모두 정상, assertion 과 종료 코드는 변경 전과 동일). bash -n·check-source-sizes.sh 통과, 작업 트리 깨끗, .github/workflows diff 0.
- 비밀값 검토: 새로 찍히는 `docker logs --tail 50` 과 본문은 릴리즈 잡 로그로 나가지만, 자격증명은 전부 스크립트에 이미 커밋된 일회용 테스트 값(112/136/137줄)이고 `/api/v4/config/client` 는 system_handlers.go:21-50 의 고정 플래그 맵이라 비밀값이 없다. 169줄은 변경 전에도 전체 로그를 덮어썼다 — 새 노출 경로 없어 차단하지 않았다.
- 못 본 것: 실제 docker 빌드 + PostgreSQL 컨테이너로 스크립트를 끝까지 돌리지 못했고(이 세션에 이미지·DSN 없음), shellcheck 도 환경에 없어 미실행. gh 미인증이라 v0.2.45 의 실제 실패 로그는 이번에도 확인 불가.
- 남는 우려: **원래 과제(두 번 실패한 릴리즈 게이트의 원인)는 미해결**이다. 이번 것은 다음 실패 때 로그를 읽게 만드는 관측성 개선일 뿐 — 릴리즈 노트에 "원인 수정" 으로 쓰면 안 된다. 다음 회차는 gh 인증 확보 후 실패 로그 확정부터 시작할 것. `npm audit` 는 실시간 조회라 어제 통과가 오늘을 보장하지 않는다.
- 판정: approve (risk low, blocking 없음). 결함 없음 — 지적은 전부 notes 급이다.
- [러너 22:37] review approved — 리뷰 승인 (risk=low)
- [러너 22:37] pr created — https://github.com/hkjang/moyro/pull/34
- [러너 22:50] ci passed — 검사 3개 모두 success
- [러너 22:50] merge done — cd5ec92
- [러너 23:18] release published — v0.2.46
- [러너 23:37] assets verified — v0.2.46 자산 1개 (이전 v0.2.45: 1)
