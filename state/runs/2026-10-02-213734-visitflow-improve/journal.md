# 회차 노트 2026-10-02-213734-visitflow-improve — visitflow
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:37] base pinned — main@955a02f
- [러너 21:37] autonomy release — 

## 정찰 노트
- 적재된 "릴리즈 워크플로 실패" 는 사실과 다르다 — release.yml·ci.yml 을 다 읽었고 결함이 없으며, 직전 성공 회차 stages.json 은 release published v2.8.12 → assets verified 로 끝났다. 실제 실패는 improve 단계의 벽시계 타임아웃(18:33→19:13)이다.
- 그래서 LobbyPage 경합(차선)보다 타임아웃의 반복 원인을 골랐다: 원장 5회차 연속 "실제 서버+dist+Chromium 하네스를 손으로 재구성했다" 가 그대로 적혀 있다. 워크플로를 느슨하게 하는 변경이 아니라 수작업 비용 제거다.
- 미확인(권한 때문에 실행 확인 못 함): docker·/usr/bin/google-chrome 의 현재 존재, 8080·5432 점유, 번들 chromium 부재. 셋 다 원장·프로필 출처이며 구현자가 먼저 직접 확인해야 한다.
- 조심할 것: CI 와 공유되는 파일은 web/playwright.config.ts 하나뿐이니 변경을 조건부 스프레드 한 줄로 가두고 .github/workflows 는 건드리지 말 것. webdist 스텁 복원은 trap + 임시 복사본(git checkout -- 금지).
- 범위가 커지면 「기동+실행+복원」까지만 담고 시딩·지연 주입은 다음 회차로 — 그걸 넣으면 같은 타임아웃을 반복한다.
- [러너 21:43] scout done — 실제 dist 를 임베드한 실제 서버 + 실제 브라우저 e2e 를 저장소의 한 명령으로 고정하기 (`scripts/local-e2e.sh`)

## 구현 노트
- `scripts/local-e2e.sh` 신규 + `web/playwright.config.ts` 조건부 스프레드 channel + README 한 문단. 프로덕션 Go/React 0개 파일, 커밋 b4f0781.
- 정찰의 미확인 3건을 먼저 확인했고 **18099 도 점유 상태였다**(`vf-race-app` 잔존). 그래서 포트를 박지 않고 `-p 127.0.0.1::8080` + `docker port` 로 OS 에 맡겼다. 서버가 `main.go:75` 에서 `:8080` 을 하드코딩해 앱은 컨테이너로만 띄울 수 있다.
- **확신 없는 곳**: ① 비평가가 먼저 볼 곳은 수용 기준 3 이다 — env 미설정 시 CI 동작 불변을 **실제 config 모듈 import** 로 확인했을 뿐(`channel` 키 부재, `use` 키가 Desktop Chrome 그대로) CI e2e 잡 자체는 못 돌렸다. PR 의 CI e2e 통과가 최종 증거다. ② 다른 머신에서 번들 chromium 이 **있을** 때의 분기(`node -e` launch 프로브가 성공하는 경로)는 이 환경에 번들이 없어 실행해 보지 못했다 — 로직만 검토했다. ③ `npx playwright install chromium` 최종 폴백(번들도 없고 시스템 Chrome 도 없을 때)도 미실행이다.
- **원장 5회차의 "환경 의존 실패" 는 재현되지 않았다**: `registers a site, a lobby and an organization...` 가 이 스크립트로 돌린 3회 모두 통과했다. 원인은 호스트가 아니라 손으로 재사용한 DB 의 상태 오염으로 보인다(매 실행이 새 DB + 새 서버). 스크립트 주석도 사실대로 적었다.
- 일부러 하지 않은 것: 데이터 시딩·`page.route` 지연 주입·변이 번들 생성(정찰 지시대로 다음 회차), `.github/workflows/*`(미변경), `.gitignore`(모든 산출물이 `mktemp -d` 아래라 불필요), `:8080` 하드코딩 제거(릴리즈 경로라 위험).
- 다음 역할이 조심할 것: 스크립트는 **docker 와 브라우저가 필요**하고 한 번에 약 2~3분 걸린다. `postgres:16-alpine`·`debian:12-slim` 이미지를 쓴다(alpine 에는 `Asia/Seoul` zoneinfo 가 없어 debian 을 골랐다 — `time.LoadLocation` 사용처 4곳). `go test` 의 통합 테스트는 여전히 `VISITFLOW_TEST_DSN` 이 있어야 돌고, 없으면 조용히 SKIP 된다.
- [러너 21:59] brief accepted — 채택 — 지정한 파일 3개(`.gitignore` 는 모든 산출물이 `mktemp -d` 아래라 불필요)·근거·수용 기준 5개가 지금 코드와 맞았고
- [러너 21:59] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 approve / risk low / blocking 없음. 확인한 것: `bash scripts/local-e2e.sh` 를 직접 1회 완주(10/10 통과, 호스트 포트 61760 OS 할당, `channel 'chrome'`, exit 0) → 끝난 뒤 `git status` 빈 출력·webdist/index.html md5 실행 전과 동일·`assets/` 잔존 없음·`vf-local-e2e-*` 컨테이너/네트워크 0개. 복원과 정리는 실측으로 통과했다.
- 구현자의 확신 없는 곳 ① CI 불변은 코드로 확정(playwright.config.ts:30 의 빈 객체 전개 + ci.yml 이 env 를 설정하지 않음). ② 번들 chromium 이 **있을** 때의 분기는 이 환경에서도 여전히 미검증(프로브의 '없을 때 exit 1' 만 확인) — CI e2e 통과가 그 증거다. ③ `scripts/local-e2e.sh:156` 폴백은 미실행이고 `--with-deps` 가 없어 맨 호스트에서는 설치 후 launch 실패 가능.
- 못 본 것: 실제 Excel·SMTP·Keycloak, debian:12-slim 의 ca-certificates 유무(이번 스위트는 외부 TLS 를 타지 않음), 다른 머신에서의 거동.
- 승인이어도 남는 우려(릴리즈 노트용): 원장 5회차의 "환경 의존 e2e 실패" 는 내 실행에서도 재현되지 않았으나, 스크립트 주석이 단언하는 원인("새 DB + 새 서버")은 추론이다 — 원장 항목을 '원인 확정' 으로 닫지 말고 '재현 불가' 로 남길 것.
- 다음 회차: `scripts/local-e2e.sh:63` 은 node_modules 디렉터리 존재만 보므로 빈 디렉터리면 npm ci 를 건너뛰어 build 가 죽는다. `:163` 은 스위트 실패 시 앱 컨테이너 로그를 남기지 않는다(cleanup 이 지움). web/playwright.config.ts·e2e/ 는 tsconfig include 밖이라 `npm run lint` 가 타입 검사를 하지 않는다.
- [러너 22:04] review approved — 리뷰 승인 (risk=low)
- [러너 22:04] pr created — https://github.com/hkjang/visitflow/pull/32
- [러너 22:08] ci passed — 검사 2개 모두 success
- [러너 22:08] merge done — b4f0781
- [러너 22:19] release published — v2.8.13
- [러너 22:21] assets verified — v2.8.13 자산 1개 (이전 v2.8.12: 1)
