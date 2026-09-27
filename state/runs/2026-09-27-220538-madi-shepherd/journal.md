# PR 처리기 노트 2026-09-27-220538-madi-shepherd — madi PR #13
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-27-192907-madi-improve)
# 회차 노트 2026-09-27-192907-madi-improve — madi
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:29] base pinned — main@6575ecd
- [러너 19:29] autonomy release — 
- [러너 19:29] scout skipped — 요청된 작업이 있어 정찰을 건너뛴다 (과제는 이미 정해져 있다)

## 구현 노트
- `deploy/runtime-apk.lock` 의 `libexpat=2.8.4-r0` → `2.8.5-r0` 한 줄만 바꿨다(커밋 ad5084a). 저장소에 2.8.5-r0 만 있고 같은 락의 fontconfig=2.17.1-r1 이 `so:libexpat.so.1` 로 그것을 끌어와 world 제약과 충돌했다.
- 락 갱신 근거는 **저장소 인덱스 조회**(`apk search -x -v`)로 구했다. 무고정 `apk add` 로 버전을 수집하면 베이스 이미지에 이미 설치된 버전을 그대로 보고해(libcrypto3/libssl3 3.5.7-r0, ca-certificates-bundle 20260611-r0) 09-19 와 같은 하위패키지 어긋남을 되살린다 — 다음 회차도 이 방법을 쓰지 말 것.
- 확신 없는 곳: 락의 alpine-release(3.24.1-r0), apk-tools·libapk(3.0.6-r0)는 저장소 인덱스(3.24.2-r0 / 3.0.8-r0)보다 뒤처져 있다. 지금은 베이스 이미지 설치본이 제약을 충족해 통과하지만 Dockerfile 다이제스트를 올리면 같이 깨질 수 있다. 이번엔 Dockerfile 금지 범위라 의도적으로 두었다.
- 검증 못 한 것: 원격 CI/릴리즈, `go test -race ./...`, 브라우저 시험, govulncheck. 로컬 이미지 경로만 확인했다.
- 일부러 하지 않은 것: 71줄 전체 재생성(불필요한 4줄 변경까지 끌어들임), `scripts/`·`tests/` 추가(과제 제약이 락 단일 파일), Dockerfile 다이제스트 상승.
- 다음 역할이 조심할 것: `bash scripts/verify-image.sh` 는 `/tmp/verify-image.txt` 고정 경로를 쓰므로 실행 전에 지워야 이전 회차 출력을 PASS 로 오독하지 않는다(이번에 삭제 후 실행). `runtime_lock_test.go` 는 이 브랜치에 없다.
- [러너 19:42] verify passed — 검증 5개 통과 (auto)

## 비평 노트
- 판정 approve(risk low, blocking 없음). 재검증: v3.24 APKINDEX 직접 파싱 → libexpat은 2.8.5-r0 하나뿐이고 `so:libexpat.so.1` 유일 공급자라 2.8.4-r0 고정은 fontconfig=2.17.1-r1 과 실제로 충돌; 내가 직접 `docker build --target runtime-base --no-cache` 를 돌려 exit 0 / `OK: 92.0 MiB in 69 packages` 를 받았고 이미지 installed DB 69줄과 락 69줄이 완전 일치했다. 파급 없음(저장소에서 libexpat 참조는 이 락뿐, distfiles에 expat-2.8.5.tar.xz 200, extraction.go 의 69/52/2 기대치 불변, licenses.mjs 는 npm+Go 범위).
- 구현자의 "확신 없는 곳" 은 실재한다(확인): alpine-release 3.24.1-r0, apk-tools·libapk 3.0.6-r0 은 저장소 인덱스에 없고(3.24.2-r0 / 3.0.8-r0) digest 고정 베이스 이미지가 정확히 그 버전을 설치해 둔 덕에만 해석된다 — 베이스 이미지에서 직접 읽어 확인했다. 다이제스트 상승 시 이번과 똑같이 깨진다. main 기존 문제이나 다음 회차가 반드시 알아야 한다.
- 못 본 것: 원격 CI/릴리즈, `go test -race ./...`, 브라우저, govulncheck 실행(단 go.mod:87 은 이미 grpc v1.83.1 로 GO-2026-6348 수정본).
- 승인이어도 남는 우려 — 릴리즈: RELEASE_NOTES.md:13 이 재정렬 대상을 ca-certificates*·tzdata 로만 열거하고 libexpat 이 빠졌으니 추가할 것(같은 줄의 "설치 집합은 잠금 파일과 정확히 일치" 는 main 에선 거짓이었고 이번 커밋으로 참이 되었다).
- 다음 회차: 락 드리프트는 여전히 전체 이미지 빌드로만 잡힌다. 보류 아이디어의 `runtime_lock_test.go`(정적 lock↔인덱스 계약 시험) 통합이 올바른 후속이다.
- [러너 19:47] review approved — 리뷰 승인 (risk=low)
- [러너 19:47] pr created — https://github.com/hkjang/madi/pull/13
- [러너 20:05] ci timeout — 제한 시간 안에 CI 완료를 확인하지 못함

## 수리 노트
- 지적은 전부 맞았다: RELEASE_NOTES.md:13 이 재정렬 대상에서 libexpat 을 빠뜨렸고, release.yml:132 이 이 파일을 게시 본문으로 그대로 복사하므로 공지가 실제 이미지와 어긋난 채 나갈 참이었다. 틀린 지적은 없었다.
- 고친 방법: 그 한 문장에 `libexpat` 2.8.5-r0(CVE-2026-93990 수정) 을 열거(커밋 cb624b5). deploy/runtime-apk.lock 은 지시대로 손대지 않았다.
- CVE 번호는 공개 본문에 들어가므로 노트를 믿지 않고 직접 확인했다: madi:ci 이미지의 aports expat APKBUILD secfixes 에 `2.8.5-r0: CVE-2026-93990`, 설치 DB `V:2.8.5-r0` 이 락 31줄과 일치.
- 재검증: deployment-contract ok, node 2/2 pass, git diff --check 깨끗. *.md 전체 grep 으로 락 버전 열거는 여전히 이 한 줄뿐임을 확인했다.
- 확신 없는 곳: 원격 CI/릴리즈, go test -race ./..., 이미지 재빌드는 돌리지 않았다(문서 1줄 변경이라 영향 없다고 판단). 앞 회차가 남긴 alpine-release·apk-tools 락 드리프트 우려는 이번 범위 밖이라 그대로 둔다.

## 심사 노트
- 확인한 것: 직전 거절 사유 해소 — v0.2.0 이후 락 변경은 정확히 ca-certificates(-bundle) 20260909-r0·tzdata 2026d-r0·libexpat 2.8.5-r0 네 줄이고 RELEASE_NOTES.md:13 이 이제 빠짐없이 열거한다(누락도 과잉도 없음).
- 노트를 믿지 않고 실물로 재확인: 이미지 aports APKBUILD secfixes 의 `2.8.5-r0: CVE-2026-93990`, 이미지 설치 69줄 == 락 비주석 69줄(diff 없음, libexpat=2.8.5-r0), origin 52개, 묶인 expat-2.8.5.tar.xz, go.mod:87 grpc v1.83.1, release.yml:132 이 이 파일을 게시 본문으로 복사.
- 로컬 통과: deployment-contract, node 2/2, go build ./..., git diff --check. 보안·법무 차단 소견 없음(인가 경로·비밀값·개인정보 무관, 서명 APK 패치 상승).
- 못 본 것: 원격 CI/release 실행, go test -race 전체, 브라우저, govulncheck 실행, --no-cache 이미지 재빌드(앞 회차 exit 0 신뢰).
- 권고 merge(risk low). 다음 회차용: 락에 alpine-release 3.24.1-r0·apk-tools/libapk 3.0.6-r0 드리프트가 남아 다이제스트 상승 시 같은 방식으로 깨지고, deployment-contract 는 락을 읽지 않아 드리프트는 전체 이미지 빌드로만 잡힌다 — runtime_lock_test.go 가 후속.
