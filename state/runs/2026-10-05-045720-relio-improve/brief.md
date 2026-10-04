- 과제: 수정 과제 — 이미 복구된 릴리즈 업그레이드를 실이미지로 재검증하고 오래된 실패와 에이전트 TIMEOUT을 분리 기록 (가치 5 / 위험 1 / 작업량 M)
- 왜: 자동 배정에 연결된 두 릴리즈 실패는 9월의 이전 공개 이미지 선택 오류이며, HEAD에는 이미 수정 3e85fcf와 실행 권한 보정 ef3f39a가 있고 이후 릴리즈 5건이 성공했다. 현재 HEAD도 공개 v1.14.0 이미지에서의 실제 업그레이드를 통과하므로 같은 수정을 재제출하지 않고 복구 증거를 원장에 남겨야 불필요한 변경과 오진을 피할 수 있다.
- 수용 기준: 1) 아래 run·기존 수정·실행 증거를 원장에 '수정 과제'로 기록하고 agent produced no result (TIMEOUT)을 제품/릴리즈 오류와 별개인 미확인 러너 문제로 명시한다. 2) 원본 릴리즈의 검증 강도를 유지하여 실제 공개 이미지→현재 HEAD의 업그레이드 검증이 exit 0이며, Personal Key·OIDC 암호문·데이터 키 ID 유지, 마이그레이션 버전, ENCRYPTION_KEY 제거 시 기동 거부를 기존 스크립트 그대로 확인한다. 3) 추가 재현에서 실패하지 않으면 코드 변경·새 회귀 테스트·커밋을 억지로 만들지 않는다; 새로운 실패가 확인될 경우에만 해당 경로를 수정하고 같은 명령의 수정 전 실패/수정 후 성공을 남긴다.
- 건드릴 파일: 저장소 코드 없음(정찰 재현 PASS, 이미 수정된 결함). 회차 디렉터리 ledger-entry.md·journal.md에 조사 결과를 보완한다. 추가 재현이 실패할 때만 scripts/previous-release-tag.sh:published_assets/older_tags/선택 루프 또는 scripts/run-upgrade-container-test.sh:wait_for_app/setup/verify 호출 중 실제 실패 경로 한 곳을 대상으로 과제서를 먼저 정정한다; 목표 1~2파일, 프로덕션 최대 6파일.
- 검증 명령: 아래 실제 실행 명령과 결과 참조.
- 위험과 피할 것: .github/workflows/의 skip·continue-on-error·타임아웃 연장·assertion 삭제로 통과시키지 말 것. auth·migrations·전역 npm/gh 설정을 바꾸지 말 것. 이번 TIMEOUT의 발생 단계/원인은 미확인이다. 실패 로그 본문을 읽지 못했으므로 기존 커밋의 원인 설명과 API 단계 판정을 구별할 것. 태그와 공개 릴리즈를 혼동한 옛 접근, Node 글롭 평면화, 이미 존재하는 루트 package.json 재추가 금지.
- 차선 후보: 같은 릴리즈 경로에서 새로 재현된 실패 한 곳의 최소 수정 — 재현이 없다면 차선으로 새 기능이나 별도 테스트 보강을 고르지 않고 검증 완료로 기록한다.

## 확정 근거와 미확인 범위

- 기준 HEAD: deaf151. git log -30, README.md, docs/ROADMAP_PLAN.md 및 docs/releases/v1.14.0.md, Makefile, package.json, web/package.json, Dockerfile, .dockerignore, .github/workflows/{ci,release}.yml을 읽었다. 저장소 루트 CLAUDE.md·AGENTS.md는 없었다. 범위를 한정한 TODO/FIXME 검색에는 결과가 없었다.
- 실패 run 34025552120(v1.11.18, 2026-09-06)·33927624115(v1.11.17, 2026-09-04)는 둘 다 Verify upgrade from previous release 단계 실패. 공개 API 응답을 release-evidence.json에 보존했다. 원본 job 로그 다운로드는 HTTP 403이라 오류 문장 자체는 미확인이다.
- git show 3e85fcf가 설명하는 원인: 공개되지 않은 태그를 이전 릴리즈로 골라 존재하지 않는 tar.gz를 다운로드했고, 실패 태그가 다음 릴리즈도 막았다. scripts/previous-release-tag.sh는 이제 실제 자산을 가진 이전 공개 릴리즈를 찾는다. 해당 수정은 HEAD에 있고 워크플로가 실제로 호출한다.
- 공개 API 최신 8건 중 위 실패 이후 v1.11.19·v1.11.20·v1.12.0·v1.13.0·v1.14.0의 5건은 모두 success. 최신 성공 run은 https://github.com/hkjang/relio/actions/runs/35823956251 (2026-09-23); 업그레이드 단계도 success다.
- scripts/previous-release-tag-test.sh는 기존 8개 사례가 전부 PASS. 이 테스트의 gh 대역은 선택 로직만 검사하므로 실이미지 증명으로 사용하지 않았다.
- 실제 공개 자산 https://github.com/hkjang/relio/releases/download/v1.14.0/relio-v1.14.0.tar.gz (37,585,957바이트)를 회차 디렉터리에 다운로드하고 docker load했다. 로컬에서 임의로 만든 old-image가 아니다.
- Dockerfile 그대로 현재 HEAD를 relio:scout-20261005-deaf151로 빌드했다(exit 0). Dockerfile의 프런트 빌드와 CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test ./...도 통과했다(docker-build.log).
- ./scripts/run-upgrade-container-test.sh relio:v1.14.0 relio:scout-20261005-deaf151 → exit 0. 출력: Protected credential persistence setup passed / Protected credentials survived container replacement / Relio upgrade test passed: 017_mail_notifications.sql -> 017_mail_notifications.sql (upgrade-test.log).
- 업그레이드 실행은 scripts/restart-persistence.py의 request 및 setup/verify 최상위 분기를 실제 HTTP·PostgreSQL에 연결했다. fake 앱·서비스 주입 없음. schema 017→017인 만큼 새 SQL 마이그레이션 수행 자체를 증명한 것은 아니다.
- git status --short는 빈 출력. 정찰에서 코드 편집·커밋 없음. 호스트 전체 make test·race·npm audit·전체 오프라인 신규 설치 smoke는 미실행이며 전체 릴리즈 완료라고 표현하지 않는다.

## 구현자가 실행할 순서와 확인 지점

1. **증거 확인(정찰 완료)**: release-evidence.json, docker-build.log, upgrade-test.log와 현재 git rev-parse --short HEAD를 확인한다. 위 GitHub run/기존 수정과 HEAD가 다르면 과제서를 수정하고 원인을 다시 확인한다. 확인 지점은 구현자 자체 점검이며 사람 승인 요청은 없다.
2. **동일 검증(정찰 완료, 코드/이미지 변경 시 재실행)**: 아래 명령을 저장소 루트에서 실행한다. 기존 증거와 동일 HEAD·이미지를 그대로 사용하는데 이유 없이 전체 테스트를 반복할 필요는 없다. 재현 실패가 생기면 출력과 실제 동작을 근거로 한 경로만 수정하고 이 단계가 통과한 뒤 진행한다.
3. **원장 정리(구현자)**: ledger-entry.md의 정찰 기록을 확인하고 '- 선택: 수정 과제 — 기존 릴리즈 복구 재검증', '- 결과: 기존 수정 확인 / 신규 코드 변경 없음', 실제 명령·exit code·TIMEOUT 원인 미확인을 남긴다. ideas.json의 선택 항목은 구현자 종료 시 done으로 바꾼다. 러너 상태 파일·정책·저장소 밖 소스는 이 과제의 변경 범위가 아니다.

```bash
# 저장소 루트. 정찰에서 실행해 모두 exit 0.
./scripts/previous-release-tag-test.sh
./scripts/check-env-contract.sh
./scripts/check-static-assets.sh
docker build --build-arg VERSION=scout-deaf151 --build-arg GIT_COMMIT=deaf151 -t relio:scout-20261005-deaf151 .
gunzip -c /mnt/c/Users/USER/projects/aidev/state/runs/2026-10-05-045720-relio-improve/relio-v1.14.0.tar.gz | docker load
./scripts/run-upgrade-container-test.sh relio:v1.14.0 relio:scout-20261005-deaf151
git status --short
```

## 선택과 작업량 근거

- **선택**: 기존 수정의 실제 런타임 검증과 기록 정정. 실패를 새로 가정하지 않으면서 자동 배정된 복구 과제를 다룬다.
- **기각**: 과거 태그 선택 패치 재작성 — 이미 적용되어 있고 이후 성공 이력이 있다.
- **보류**: Makefile에 릴리즈 선택 테스트 연결·업그레이드 실패 진단 보강 — 타당한 별도 아이디어지만 이번 실패의 원인으로 확인되지 않았다.
- bottom-up 예상: 증거 확인 3~5분 + 필요 시 이미지 준비/실행 10~20분 + 기록 3~5분 = 16~30분. 네트워크/이미지 캐시 편차 대비 contingency 5~10분을 별도 둬 총 21~40분(M); 정찰 캐시를 그대로 쓰면 더 짧다. 측정된 확률 신뢰수준은 없고 확신은 중간이다. 관리 예비비는 배정하지 않았으며 새 제품 결함이 나오면 범위·추정을 다시 적는다.
- 전제: Docker 및 공개 자산 접근 가능(이번에 실측). gh CLI는 미인증이나 공개 API/자산은 urllib로 접근했다. 인증 정보나 전역 설정을 만들지 않았다.
- 스킬 적용: Skill 도구 미노출로 /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/{pmo,technology}/skills/ 아래의 estimating-and-contingency, implementation-planning, solution-exploration/SKILL.md를 직접 읽었다. 작업 분해·불확실성/예비시간, 순차 실행/검증 지점, 대안 비교를 이 과제서에 반영했다.
