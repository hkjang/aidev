# 회차 노트 2026-09-23-084455-postra-improve — postra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 08:45] base pinned — main@5dd0099
- [러너 08:45] autonomy release — 

## 정찰 노트
- 우선 과제("릴리즈 워크플로 2회 실패")를 추적한 결과 실패 지점은 저장소가 아니라 러너였다: 2026-09-21-042417 `release: missing`(CI 는 10개 success), 2026-09-22-234439 `scout: failed — 과제서 없음`인데 그 회차 brief.md 는 실제로 존재. `.github/workflows/release.yml` 에서는 재현 가능한 결함을 찾지 못했고 태그 v0.23.2/v0.23.3 둘 다 notes 파일을 포함한다.
- 그래서 워크플로를 손대는 대신 직전 회차가 PR 전에 끊겨 끝내 구현되지 않은 IMAP 과대 리터럴 desync 과제를 이어 받는다. 근거(client.go:162-164 가 n 바이트 미소비, sync.go:258/264 가 같은 세션 재사용)는 이번에 직접 읽어 재확인했다.
- 미확인: GitHub Actions Release 잡의 실제 실패 여부 — curl·WebFetch 가 모두 차단돼 Actions API 를 읽지 못했다. 브리프 0절에 미확인으로 명시.
- 구현자 주의: 기존 oversizeServer 는 4GiB 를 선언만 하고 보내지 않는다 — 무조건 드레인하면 그 테스트가 60초 데드라인까지 매달린다. 드레인 상한 분기 필수, io.CopyN(io.Discard) 로만. pop3/client.go 는 미병합 04b15be 와 충돌하니 금지.
- 프로필은 1일 전 작성분이 현 base(main@5dd0099, 변동 없음)와 일치해 새로 쓰지 않았다. 이번에 실행한 것은 git 읽기와 grep 뿐이다.
- [러너 08:48] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- 배정 과제("릴리즈 워크플로 2회 실패")는 기각했다: Actions API 로 Release 72회 전부 success 확인, `npm ci && npm run build` 후 spa/assets 드리프트 0 — 실패는 러너의 세션 중단이었다. 대신 끊긴 회차가 진단만 남긴 IMAP 과대 리터럴 desync 를 구현했다(77a242c, client.go + client_test.go 만).
- 확신 없는 것: 드레인 예산 32MiB 와 청크 1MiB 는 판단값이지 측정값이 아니다(느린 링크에서 60초 명령 데드라인과의 상호작용은 로컬 루프백에서만 확인). 예산 초과 시 세션을 닫으면 그 계정의 남은 메시지가 전부 Failed 로 집계되는데, 이 application 쪽 파급은 어댑터 테스트로만 덮였고 sync 통합 테스트로 재현하지는 않았다.
- 검증 못 한 것: 실제 IMAP 서버(Dovecot 등) 대상 검사, 외부 PostgreSQL, 브라우저/프런트 테스트(프런트 코드 미변경이라 생략), `Idle` 의 broken 검사는 테스트 없음(도달 경로를 실서버 없이 만들기 어려움).
- 일부러 안 한 것: `strconv.Atoi` 오버플로 가드(MaxMessageBytes=0 이면 여전히 거대 할당 가능 — ideas.json 에 신규 pending), 세션 폐기를 sync 루프에 알리는 도메인 신호, 문서 갱신(ADMIN_GUIDE 에 해당 절 없음). pop3/client.go 는 미병합 PR #22 와 충돌하므로 건드리지 않았다.
- 다음 역할 주의: 변이 2(예산 분기 제거)는 일부러 45초 타임아웃까지 매달리게 만든 것이라 재현 시 그만큼 걸린다. 새 테스트는 3MiB 를 루프백으로 흘리므로 -race 에서도 약 1초. errUnframed 는 비공개 sentinel 이라 패키지 밖에서는 errors.Is 로 볼 수 없다.
- [러너 09:00] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인: main 의 client.go 로 되돌려(errUnframed 스텁만) 새 테스트를 돌렸고 `body ... = "yyyy..."` 로 실패 — 수정 전 코드에서 통과하지 않음을 실증. gofmt/vet/gosec(0 issues)/`go test -race ./internal/adapters/... ./internal/application/...` 모두 통과, 작업 트리 복원 확인.
- 거절 사유: client.go:133 resyncDrainLimit(32MiB) < 기본 거절 임계값 51MiB(config.go:278 의 50MiB + 1MiB 여유)라 드레인 경로가 기본 설정에서 절대 실행되지 않고 항상 abandon 으로 간다. 테스트(client_test.go:186)는 1MiB 라는 비기본값을 써서 이 사실을 가린다.
- 파급: sync.go:241 루프가 같은 세션을 계속 쓰고 실패는 체크포인트를 남기지 않으므로, 기본값에서 오버사이즈 메시지 하나 뒤의 메일이 매 동기화마다 전부 실패한다(계정 스톨). 커밋 메시지의 "only the oversized message fails" 와 불일치.
- 수리가 먼저 볼 파일: internal/adapters/imap/client.go:133/224 — 예산을 maxLiteral+1MiB 에서 유도하고, 기본 50MiB 설정으로 드레인 후 재동기화되는 테스트를 추가. abandon 은 길이가 거짓인 경우의 폴백으로만 남길 것.
- 승인이어도 남는 우려(수리 후에도 유효): 거절된 리터럴이 stats.Oversize 가 아닌 stats.Failed 로 집계됨(sync.go:258); 드레인이 살아나면 1MiB 청크마다 데드라인 갱신이라 최대 32 x 명령 타임아웃(≈32분) 세션 점유; docs/releases/ 노트 미작성. 보안·법무 차단 사유는 없음(blocking 비움).
- [러너 09:08] review rejected — 리뷰 거절: internal/adapters/imap/client.go:133 resyncDrainLimit(32MiB) 이 기본 설정에서 도달 불가능한 값이다. config.go:278 의 기본 Sync.MaxMessageBytes 는 50MiB 이고 client

## 수리 노트
- 맞았던 지적: 32MiB 고정 예산 < 기본 거절 임계값 51MiB → 드레인은 기본 설정에서 절대 실행되지 않고 항상 abandon. 예산만 32MiB 로 되돌려 새 테스트가 실패하는 것으로 재현 확인. 기존 테스트가 1MiB 비기본값으로 이를 가렸다는 지적도 맞다.
- 살짝 틀린 부분: sync.go 의 remote 는 오름차순이 아니라 sync.go:220 에서 역순(최신 우선)으로 뒤집힌다. 결론(같은 자리에서 반복 실패 → 그 뒤 메일 영구 미수집)은 순서와 무관하게 성립.
- 고친 방법: `refusalThreshold()`(maxLiteral+1MiB) 와 `resyncBudget()`(임계값×4, 오버플로 포화)을 도입해 예산을 설정값에서 유도. abandon 은 4GiB 거짓 길이 같은 폴백으로만 남음. 테스트는 config.Default() 의 50MiB 로 52MiB 본문을 실제 소켓에 스트리밍(메모리 미보유, 0.04s)해 드레인·후속 수집을 단언하고, 예산>임계값 불변식을 별도 테스트로 고정.
- 확신 없는 곳: 배수 4(기본값에서 204MiB 드레인 허용)는 판단값이다. 작은 MaxMessageBytes(예: 1MiB)에서 100MiB 본문은 여전히 abandon 되는데, 이는 대역폭 낭비를 피하는 의도된 트레이드오프로 봤다.
- 미해결(범위 밖): 세션이 abandon 된 뒤에도 sync.go 루프는 같은 세션으로 계속 돌아 남은 메시지를 errUnframed 로 실패시킨다. 거짓 길이 서버에서는 어떤 클라이언트도 복구할 수 없어 세션 종료 자체는 불가피하며, 루프 조기 종료는 별도 변경으로 남겨둠.
- [러너 09:14] repair done — # fix-summary (수리 시도 2)  - 비평가 지적대로 `resyncDrainLimit = 32MiB` 는 거절 임계값(MaxMessageBytes+1MiB=51MiB, 기본값)보다 작아 드레인 경로가 출하 기본값에서

## 비평 노트
- 확인함: main 의 client.go 로 되돌려 새 테스트를 돌리니 두 동작 테스트가 실제로 실패했고(2번 메일 본문 자리에 trapHeader 의 'y'40자가 돌아옴), 수정 전 결함이 "쓰레기를 정상 메일로 저장" 이라는 데이터 무결성 문제였음이 증명됐다. 77a242c 에서 32MiB 고정 예산이 기본 임계값 51MiB 보다 작아 드레인이 죽은 코드였던 것도 확인 — 두 커밋 모두 근거 있음. build·vet·gofmt·go test -race(application + adapters 전부)·make lint(gosec 0건) 통과. 판정 approve, blocking 없음.
- 못 봄: 프런트(web/), 외부 PostgreSQL(POSTRA_TEST_PG), Chromium, 실제 IMAP 서버. 이번 diff 는 imap 어댑터 2개 파일이라 계약 -check 는 해당 없음.
- 승인이어도 남는 우려 1 (취소 응답성): client.go:260-271 이 1MiB 청크마다 마감을 갱신해, 기본값(예산 204MiB / commandTO 60s)에서 느린 서버 하나가 메시지 하나당 최대 ~3.4시간을 붙잡을 수 있다. exec 에 ctx 가 없고 sync.go:242 의 ctx 검사는 메시지 사이에서만 돌아 작업 취소·파드 종료가 이를 끊지 못한다. 청크별이 아닌 드레인 전체 벽시계 예산으로 바꾸는 게 다음 수선 1순위.
- 승인이어도 남는 우려 2 (구현자가 범위 밖으로 남긴 자리): abandon 이후 sync.go:261-264 가 같은 세션으로 계속 돌며 나머지 메일을 전부 errUnframed 로 실패시키고도 finish(JobSucceeded) 를 부른다. 발동 조건은 좁지만(RFC822.SIZE 없는 서버 + 204MiB 초과 본문) 걸리면 그 뒤 메일이 매 싱크마다 영구히 막힌다 — 다음 회차가 sync 루프에서 마무리할 값어치가 있다.
- 릴리즈 노트: docs/releases/ 항목도 버전 범프도 없다. 최근 PR(#21)은 fix + docs(release) 를 한 묶음으로 냈으니, 사용자 체감 동작 변화인 이번 재동기화는 다음 릴리즈 노트에 반드시 싣기 바람.
- [러너 09:19] review approved — 리뷰 승인 (risk=medium)
- [러너 09:19] pr created — https://github.com/hkjang/postra/pull/23
- [러너 09:25] ci passed — 검사 10개 모두 success
- [러너 09:25] merge done — bcfd415
- [러너 09:34] release published — v0.23.4
- [러너 09:36] assets verified — v0.23.4 자산 4개 (이전 v0.23.3: 5)
