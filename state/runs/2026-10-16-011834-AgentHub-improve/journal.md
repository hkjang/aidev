# 회차 노트 2026-10-16-011834-AgentHub-improve — AgentHub
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:18] base pinned — main@938fb1d
- [러너 01:18] autonomy release — 

## 구현 노트
- 실패 원인: 6059496(미머지)이 BASE_VERSION 만 0.27.0 으로 올려 compose.yaml 빌드 인자·internal/buildinfo 기본값이 0.26.0 에 남았다. 그 두 거울은 Go 스위트만 지키고, 작성자가 올린 직후 보는 check-versions 는 통과를 돌려줘 릴리즈가 두 번 깨졌다. 고친 것은 그 게이트다(0fd2810, 프로덕션 파일 scripts/release-catalog-images.sh 하나). 느슨하게 한 곳 없음, 추가만.
- 확신 없는 곳: ① 새 회귀가 git commit-tree 로 참조 없는 커밋을 만든다 — 커밋/브랜치/작업 트리는 건드리지 않지만 느슨한 객체를 저장소에 쓴다(gc 대상). CI 의 얕지 않은 체크아웃(fetch-depth: 0)에서 돌려 보지는 못했다. ② 이 회귀가 git 플럼빙을 많이 돌려 로컬 internal/buildinfo 가 17s→41s 가 됐다 — 원인은 알려진 9p gitdir 이고 CI(로컬 디스크)에서는 훨씬 작을 것이나 **측정하지 못했다**. ③ commit-tree 에는 GIT_AUTHOR/COMMITTER 를 명시로 넘겼지만 git config 가 전혀 없는 러너에서 확인하지 못했다.
- 일부러 하지 않은 것: 6059496 의 dlp.go 재상정(수리 PR 에 독립 기능을 끼우지 않음 — ideas.json 에 올리는 방법까지 적어 두었다), 그리고 거울이 빌드 인자·심볼을 '아예 적지 않은' 경우의 검사(옛 커밋 재생에서 거짓 실패를 만든다 — Go 스위트가 그쪽을 본다). deploy/kubernetes/ollama-bridge.yaml:19·README.md:173 의 agenthub-base:v0.26.0 은 아무 검사도 없는 진짜 공백이지만 이번 범위에서 빼고 후보로 기록했다.
- 다음 역할이 조심할 것: 새 테스트는 git 과 v0.262.0 태그가 보이는 체크아웃이 필요하고(얕은 클론에서는 check-versions 가 비교 대상 없음으로 조용히 통과한다), 느린 패키지는 internal/buildinfo 다. check-versions 는 작업 트리가 아니라 HEAD(또는 GITHUB_SHA)를 읽으니 커밋 전에 돌리면 수정 전 상태를 채점한다. 돌린 검증: go test ./... exit 0, go test -race -p 1 ./cmd/... ./internal/... exit 0, validate·check-versions exit 0, kustomize·compose config 통과. npm 계열·브라우저·DB live 는 돌리지 않았다(web 변경 없음).
- [러너 01:38] verify passed — 검증 5개 통과 (auto)

## 비평 노트
- 판정 approve/low/차단없음. 원장에 `- 실패 재현:` 이 없어 직접 재현했다: main 의 스크립트를 꺼내 BASE_VERSION 만 올린 commit-tree 커밋을 먹이니 구버전 exit 0, 신버전 exit 1 로 compose·buildinfo 둘을 지목 — 새 테스트 첫 단언이 수정 전 반드시 실패한다. 테스트 2개 PASS, validate·check-versions exit 0, 작업 트리 무변경.
- 구현자의 불확신 셋 중 ③은 해소(HOME 비우고 git config 전부 무효화해도 PASS), ②는 측정했다(게이트 8.57s→12.02s, 테스트 +19.5s, 9p 기준 — CI 값은 여전히 미측정이니 릴리즈 노트에 그대로 적을 것). ①은 참조 없는 느슨한 객체뿐이라 무해.
- 잠식 검사: 기존 replay 테스트의 9a35494/b742060 을 구·신 양쪽으로 돌려 출력 동일 — 새 die 가 앞서서 그 테스트가 다른 이유로 통과하는 일은 없다. 주석이 '아예 안 적힌 경우는 Go 스위트' 라 주장한 가드(imagesources_test.go:402, spawner_baseimage_test.go:117)도 열어 실존 확인.
- 승인이어도 남는 우려: ①sh:648,653 이 카탈로그 값을 sed 식에 날것으로 보간하고 check_versions 는 validate_catalog 를 안 부른다(권한 상승 없어 차단 아님, 한 줄로 굳힘 가능). ②README.md:173·ollama-bridge.yaml:19 는 여전히 무검사 — 다만 '이미 퍼블리시된 옛 태그' 라 위험이 낮다. ③compose 에 따옴표를 넣으면 거짓 실패, 버전 파일이 양쪽에 없으면 die 대신 셸 오류.
- 다음 회차: internal/buildinfo 가 사실상 두 배 느려졌으니 CI 실측 한 번 남길 것. 못 본 것 — npm 계열·브라우저·DB live·배포(web 변경 없어 생략), 전체 -race 스위트는 재실행하지 않고 구현 노트의 exit 0 을 믿었다.
- [러너 01:49] review approved — 리뷰 승인 (risk=low)
- [러너 01:49] pr created — https://github.com/hkjang/AgentHub/pull/46
- [러너 01:52] ci passed — 검사 1개 모두 success
- [러너 01:52] merge done — 0fd2810
- [러너 01:59] release published — v0.263.0
- [러너 02:05] assets verified — v0.263.0 자산 8개 (이전 v0.262.0: 8)
