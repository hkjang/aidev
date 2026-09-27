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
