# 회차 노트 2026-10-18-111743-umm-improve — umm
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:17] base pinned — main@c00a54b
- [러너 11:17] autonomy low-risk — 회귀(reverted) 2026-10-06T22:38:39+09:00

## 정찰 노트
- 골랐다: toolchain↔Dockerfile 핀 드리프트 검사. 결정적 사실은 `ci.yml:49` 가 이미 `./scripts/check-version.sh` 를 돌린다는 것 — 이전 회차가 "배선이 보호 경로라 제외" 하며 보류했던 이유가 사라졌고, 파일 1개로 끝난다. Node 하한 검사는 `engines` 가 두 밴드 범위라 파서가 필요해 같이 담지 않았고, `make test-go` 와 outline 파서는 운영자 규칙 1·3번에 걸려 제쳤다.
- 추측으로 적은 것: `check-version.sh` 를 **직접 돌려 보지 못했다**(샌드박스가 스크립트 실행을 막음). 릴리즈 노트·web/package.json 블록은 글롭·읽기로 확인했으나 package-lock·compose.yaml 블록은 미확인 — 구현자는 기준선부터 찍을 것. 제안한 `sed` 패턴도 추출 결과를 먼저 echo 로 찍어 확인할 것.
- 구현자가 조심할 것: `Dockerfile`·`go.mod` 는 **읽기만** — 드릴로 임시 변경했으면 반드시 원복하고 `git status --porcelain` 으로 확인. 숫자(`1.26.9`)를 스크립트에 하드코딩하지 말고 두 파일을 서로 비교할 것. 빈 추출값을 통과로 처리하면 검사가 죽은 채 초록이 된다.
- 아이디어 두 건의 상태를 되돌렸다: `edge-vocabulary` 시험(a5959a5)과 그 망에 의존하는 `isInferred` 건은 a5959a5 가 이 베이스(c00a54b)에 **없어** done 이 아니라 pending 이다.
- [러너 11:23] scout done — Go 툴체인 핀 드리프트를 기존 CI 게이트(`scripts/check-version.sh`)에서 닫기 (가치 4 / 위험 2 / 작업량 S)

## 구현 노트
- 무엇을 왜: `scripts/check-version.sh` 한 파일에 Go 핀 블록을 더해 `go.mod` 의 `toolchain` 과 `Dockerfile` go-build 스테이지의 `golang:<patch>-alpine` 이 같은 Go 패치를 가리키는지 검사한다(커밋 0857189). 두 값을 각 파일에서 읽어 **서로** 비교하므로 숫자 하드코딩이 없고, 한쪽 추출이 비면 실패한다. `ci.yml:49` 가 이미 이 스크립트를 돌려 추가 배선 0건.
- 확신 없는 곳·검증 못 한 것: ① 빌더 패턴을 과제서의 `-alpine .*` 에서 `-alpine.*` 로 한 글자 느슨하게 했다 — `AS go-build` 가 없는 줄에서도 태그를 읽게 한 것이고 태그 비교 자체는 그대로 엄격하지만, 과제서 문구와 다른 유일한 지점이니 비평가가 먼저 볼 자리다. ② 이 검사는 **go-build 스테이지의 첫 `golang:` 줄 하나만** 본다(`head -1`). 지금 `golang:` 은 `Dockerfile:9` 에 한 번만 나오지만 둘째 Go 스테이지가 생기면 검사가 그것을 보지 않는다 — 넓히지 않았다. ③ `make test-go` 는 EXIT 0 이지만 `POSTGRES_DSN` 이 unset 이어서 store 통합은 내부 SKIP 이고 대부분 `(cached)` 였다 — Go 소스 0줄 변경을 확인하는 값싼 게이트로만 썼고 통합 통과로 읽지 말 것.
- 일부러 하지 않은 것: `make test-web`·`docker build`·DB 통합·E2E 를 **돌리지 않았다**(쉘 한 파일 변경, 웹·Go 소스 0줄, 이 체크아웃에 `web/node_modules` 없음). Node 하한 드리프트(`engines`·`ci.yml:41`·`Dockerfile:2`)는 과제서 지시대로 담지 않았다 — 실제 불일치가 없고 `engines` 두 밴드 파서가 필요하다. 성공 메시지는 과거 로그가 통과 신호로 읽는 `version <X> is consistent` 조각을 지키려 한 글자도 바꾸지 않았다.
- 다음 역할이 조심할 것: 드릴을 재현하려면 `Dockerfile`/`go.mod` 를 임시로 바꾸게 되는데 **반드시 `git checkout -- Dockerfile go.mod` 로 원복**할 것(네 드릴 모두 원복·`git status --porcelain` 확인했고 최종 트리는 `scripts/check-version.sh` 한 줄만 변경). 이 검사는 네트워크·DB·도커가 필요 없어 `./scripts/check-version.sh` 한 번으로 재확인된다.
- [러너 11:27] brief accepted — 채택 — 과제서의 근거가 지금 코드와 전부 맞았고(`go.mod:3` `go 1.26.0` / `:5-7` 주석 / `:8` `toolchain go1.26.9`, `Dockerfile` 의 이미�
- [러너 11:28] verify passed — 검증 17개 통과 (auto)

## 비평 노트
- 판정 approve / risk low / blocking 없음. 원장에 `- 실패 재현:` 줄이 없어 게이트를 사본 트리에서 직접 재현했다: go.mod만 1.26.10 → exit 1 + 정확한 불일치 메시지(2026-10-16 회귀 그대로 잡힘), 둘 다 올림 → exit 0, toolchain 줄 삭제 → exit 1(missing), `FROM --platform=... golang:` → exit 1(missing, 조용한 통과 아님), VERSION 부정합 시 기존 5개 검사도 전부 보고(status 누적). 현재 트리 exit 0, `git status --porcelain` 깨끗, `bash -n` 통과.
- 구현자가 의심한 ①②를 봤다: `-alpine.*` + `head -1` 은 `AS go-build` 를 통짜지 않으므로 :51 메시지와 :39 주석의 "go-build 스테이지" 는 한 칸 느슨한 표현이다. golang 줄이 Dockerfile:9 하나뿐이어서 지금은 결과가 맞고 잘못된 쪽은 요란하게 실패한다 — 결함 아님.
- 남는 우려(다음 회차): Dockerfile:9 **앞에** 둘째 Go 스테이지가 생기면 검사가 그것을 읽고, 그 태그가 우연히 같으면 진짜 바이더가 드리프트해도 초록이 된다. 그때 `AS go-build` 를 박거나 매치 개수 1을 단언할 것. go.mod 의 `toolchain` 줄을 지우는 정당한 변경도 이 블록을 같이 고쳐야 한다.
- 못 본 것: `make test-web`·`docker build`·DB 통합·E2E 미실행(쉘 1파일, web·Go 소스 0줄 변경이라 불필요하다고 판단). shellcheck 은 이 환경에 없어 돌리지 못했다.
- 범위·되돌리기: scripts/check-version.sh 18줄 추가뿐, 마이그레이션·외부 상태·의존성·인증 변경 0. 보안·법무 소진 없음. VERSION 0.76.8 미변경이 관례와 맞아 신규 릴리즈 노트 불필요.
- [러너 11:31] review approved — 리뷰 승인 (risk=low)
- [러너 11:32] pr created — https://github.com/hkjang/umm/pull/175
