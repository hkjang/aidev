# PR 처리기 노트 2026-10-07-170010-hunter-shepherd — hunter PR #21
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-07-161808-hunter-improve)
# 회차 노트 2026-10-07-161808-hunter-improve — hunter
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:18] base pinned — main@ba302db
- [러너 16:18] autonomy release — 

## 구현 노트
- 2026-10-06 이 남긴 두 숫자 필드를 닫았다. `findings contribution_points` 는 폼에 `max` 선언이 없어 서버 범위 `0~10000` 밖을 통과시켰고, 더 중요하게 findings 분기가 형제 검증기들과 달리 검사한 int 를 되쓰지 않아 저장된 `7.5` 가 서버 순위표·CSV(`number()`)와 `pages.tsx`(`Number()`)에서 다르게 읽혔다. `validateContributionPoints` 로 추출 + 되쓰기, 두 선언은 `resourceNumberBounds` 를 펼쳐 쓴다. 프로덕션 3파일.
- **확신 없는 곳**: 상한·정수 잠금이 실제 브라우저에서 입력을 막는 것은 Mantine 구현을 읽어 확인했을 뿐 DOM 하네스가 없어 렌더링으로 실측하지 못했다. `a.validateResource` 전체 HTTP 경로(서비스 접근·workflow 설정)는 DSN 이 없어 돌리지 않았고, 새 되쓰기가 그 경로에서 호출되는 것은 호출부 한 줄(`domain.go:519`)을 읽어 확인한 수준이다.
- **의도한 동작 변화 한 가지**: 되쓰기가 키가 없던 발견 건에도 `contribution_points: 0` 을 넣는다. 전수 grep 으로 "키 없음"과 0 을 구분하는 독자가 없음을 확인했고 `openapi.json` 은 이 필드를 integer 로 선언하지만, 비평가가 먼저 볼 지점으로 남긴다. `findingFingerprint` 는 이 필드를 쓰지 않아(imports.go:19) 중복 식별에는 영향 없음.
- 일부러 하지 않은 것: `settings.tsx` 의 `sla`·`risk` 숫자 정수 잠금(같은 결함 계약이지만 거절이 보이는 쪽이라 별 회차로 — ideas.json 에 신규 등록), `engines` 선언(릴리즈 경로 인접, 네 회차 연속 밀림), `resourceSubmitBody` 의 number 분기(이번 되쓰기로 도달 경로가 더 좁아졌다).
- 다음 역할이 조심할 것: 새 Go 테스트는 DSN 없이 돈다(`go test -run '^(TestResourceNumberBounds|TestContributionPoints)' ./internal/app`). 공유 벡터의 `reject` 열은 서버 메시지와 **완전 일치**를 요구하므로 검증기 문구를 바꾸면 벡터도 같이 고칠 것. `go test -race ./...`·원격 CI·Docker 빌드·화면 캡처는 미실행.
- [러너 16:30] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인: 구현자가 의심한 두 자리를 먼저 봤다. Mantine 기본값은 설치된 `@mantine/core` 소스에서 사실 확인(`clampBehavior: "blur"` :62, `decimalScale: allowDecimal ? decimalScale : 0` :385). `validateContributionPoints` 가 findings 분기에서 반드시 실행되는 것은 분기 전체(domain.go:463-520)에 조기 `return nil` 이 없음을 읽어 확인. 세 호출부(domain.go:342, integrations.go:377, agents_tools.go:223)와 세 독자(domain.go:738 `points>0`, :860 CSV, pages.tsx:1080-1081)를 전수 확인 — "키 없음"과 0 을 구분하는 독자는 없고 SQL·GraphQL·보고서 독자도 없다. openapi.json 9691/11322 는 integer/0/10000 로 되쓰기와 정확히 일치. 테스트는 되쓰기 없이는 실패한다(`m[d.Field] != d.Stored`, browser≠server) — 항상 참이 아니다.
- 실측: web 113통과/0실패/0skip, Go 세 테스트 전부 PASS(bounds 9행), typecheck·prettier·gofmt·vet·verify-pentagi(312)·`git diff --check` 통과.
- 못 본 것: DSN 기반 `go test -race ./...`(a.validateResource HTTP 전체), `npm run build`+자산복사+`go build ./cmd/hunter`, 문서 재생성, Docker, 브라우저 DOM 실측.
- 승인이어도 남는 우려: ① 되쓰기는 한 방향 데이터 변경이다 — revert 로 저장된 7.5·null 은 돌아오지 않는다(영향은 기여 페이지의 소수점 손실뿐). ② `docs/release-v1.23.0.md:9` 가 이 두 필드를 "그대로"라고 약속했으므로 다음 릴리즈 노트가 명시적으로 대체할 것. ③ 새 `reject` 열은 서버 메시지 완전 일치 — 네 검증기 문구를 바꾸면 벡터도 같이. ④ findings 행은 `!managerial` 덮어쓰기(domain.go:516-518)와의 상호작용을 지나지 않는다: 범위 밖 저장값은 비관리 사용자의 모든 수정을 막고 그 사용자는 풀 수 없다(이 PR 이전부터 동일, 넓히지 않음).
- [러너 16:36] review approved — 리뷰 승인 (risk=low)
- [러너 16:36] pr created — https://github.com/hkjang/hunter/pull/21
- [러너 16:55] ci timeout — 제한 시간 안에 CI 완료를 확인하지 못함

## 심사 노트
- 확인: 되쓰기(domain.go:414)를 빼면 두 Go 테스트가 `browser 7.5 != server 7`·`m[d.Field] != d.Stored` 로 실제 실패한다 — 항상 참인 단언 아님. 전수 grep 으로 contribution_points 독자 셋(domain.go:738/860, pages.tsx:1080-1081)이 '키 없음'과 0 을 구분하지 않음을, openapi.json:9691/11322 이 integer/0/10000 임을, domain.go:463-520 findings 분기에 조기 `return nil` 이 없어 호출이 무조건 도달함을 읽어 확인. 권한 순서(`!managerial` 덮어쓰기 → 범위 검사)는 보존되고 resources.tsx 의 number 필드 9개 전부가 공유 표를 spread 한다.
- 실측: go vet ./... · gofmt -l(출력 없음) · go test ./internal/app(ok, 새 테스트 24 하위케이스 PASS) · npm --prefix web ci 후 test **113 pass 0 fail 0 skip** · typecheck · **npm run build(vite 성공)** · prettier --check · verify-pentagi 312 · git diff --check · 트리 clean.
- 못 본 것: HUNTER_TEST_DSN 없어 `go test -race ./...`(a.validateResource HTTP 전체) 미실행, 자산복사+`go build ./cmd/hunter` 미실행(go vet 로 컴파일만 확인), Docker, 브라우저 DOM 실측. TS 의 "spreads the shared bounds" 는 소스 문자열 검사로 배선 증거가 아님(main 에 있던 패턴).
- 권고 근거: 결함 없음 → **approve / merge**. risk=**medium** — 되쓰기가 저장 데이터를 한 방향으로 정규화(null→0, 7.5→7, "50"→0)하고 revert 로 돌아오지 않는다. 다만 영향은 기여 점수 한 칸이고 Go 독자 둘은 변경 전에도 그 값을 잘라 읽었다. 마이그레이션·권한 확대·새 공개 경로·새 의존성 없음 → 보안·법무 차단 소견 없음.
- 후속: docs/release-v1.23.0.md:9 이 이 두 필드를 "그대로"라고 적고 있다. ba302db 관례상 릴리즈 노트는 별도 release 커밋 몫이므로 이 PR 의 누락은 아니지만 다음 릴리즈 노트가 그 문장을 대체해야 한다.
