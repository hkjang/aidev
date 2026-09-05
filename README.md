# 자율 개선 에이전트 (aidev) 운영 문서

저장소: https://github.com/hkjang/aidev — 러너·프롬프트·원장·로그를 여기서 관리한다.
작성일: 2026-09-02 · 대상: `/mnt/c/Users/USER/projects` 하위 저장소 · 담당: hkjang

## 1. 목적

최근 한 달 안에 손댄 저장소를 대상으로, Claude Code 에이전트가 **스스로 개선 아이디어를 내고 → 하나를 골라 구현하고 → 테스트로 검증하고 → PR을 열어 main에 머지**하는 무인 사이클을 돌린다. 사람은 결과(원장·로그·머지 커밋)만 확인한다.

## 2. 저장소 구성

| 경로 | 역할 | git |
|---|---|---|
| `bin/run.sh` | 러너. 후보 선정 → 워크트리 생성 → `claude -p` 실행 → 푸시/PR/머지 → 원장 커밋·푸시 | ✔ |
| `bin/daily.sh` | 스케줄러용 래퍼. PATH 고정, `git pull`로 최신 러너 반영, `flock` 겹침 방지, `logs/cron.log` 기록 | ✔ |
| `prompt.md` | 에이전트에게 주는 지시문 (절차 8단계 + 절대 규칙) | ✔ |
| `state/<프로젝트>.md` | **원장**: 회차별 선택·결과·보류 아이디어. 다음 회차 프롬프트에 주입되어 중복 방지 | ✔ (러너가 자동 커밋) |
| `state/.cursor` | 라운드로빈 커서 | ✔ |
| `state/<프로젝트>.env` | 프로젝트별 환경변수 (예: 테스트 DB DSN). 에이전트 실행 시 주입 | ✘ 비밀값 — .gitignore |
| `logs/<날짜>.log` | 러너 요약 로그 (후보·선택·PR·머지) | ✔ (러너가 자동 커밋) |
| `logs/<날짜>-<프로젝트>.txt` | 에이전트 최종 출력 | ✘ 크기 |
| `logs/cron.log` | 스케줄 실행 누적 로그 | ✘ |

로컬 체크아웃: `/mnt/c/Users/USER/projects/aidev` (Windows `C:\Users\USER\projects\aidev`). 잠금 파일은 `~/.auto-improve/run.lock`.

> 이 저장소는 **public**이다. `state/*.env`는 절대 커밋하지 말 것. 원장에는 대상 저장소의 버그 설명이 들어가므로 비공개가 필요하면 `gh repo edit hkjang/aidev --visibility private`.

## 3. 한 회차의 흐름

```
bin/daily.sh (Windows 작업 스케줄러 → wsl.exe)
 ├─ git pull --ff-only          이 저장소의 최신 러너·프롬프트·원장을 반영
 └─ bin/run.sh
     1. 후보 선정   최근 30일 커밋 有 + 작업트리 깨끗 + origin 원격 有
                    (제외 목록 EXCLUDE_RE: aidev 자신, Naviq, sqlpad, _tmp*, 임시 폴더)
     2. 선택        라운드로빈 커서로 --count 개
     3. 워크트리    ~/.cache/auto-improve-wt/<프로젝트> 에 auto/<날짜>-<시각> 브랜치
     4. 에이전트    claude -p (prompt.md + 원장) --permission-mode acceptEdits
                    --allowedTools Bash,Read,Edit,Write,Glob,Grep,WebFetch,WebSearch
                    --max-budget-usd <예산>
     5. 결과 확인   base..branch 커밋 수
     6. 커밋 있음   push → gh pr create → 워크트리 제거 → gh pr merge --merge --delete-branch
                    → 로컬 main pull --ff-only
        커밋 없음   브랜치 삭제
     7. 원장        에이전트가 못 남겼으면 러너가 커밋 제목으로 대체 기록
     8. 릴리즈      (머지된 경우) release-prompt.md 로 릴리즈 에이전트를 한 번 더 실행 — 아래 참조
     9. 동기화      state/ + logs/*.log 를 aidev 에 커밋·푸시  ("run(날짜): 프로젝트 — 결과")
```

### 릴리즈 단계 (`release-prompt.md`, `release_project()`)
머지가 성공한 프로젝트는 **그 저장소가 지금까지 해 온 방식 그대로** 릴리즈한다.
1. 러너가 `git fetch --tags` 후 `origin/<base>`에서 detached 워크트리를 만든다 (로컬엔 옛 태그만 있는 저장소가 많다)
2. 릴리즈 에이전트가 실제로 확인: 최근 태그 3개와 그 커밋, 릴리즈 커밋 메시지 양식, 버전 파일(여러 곳이면 전부), CHANGELOG/릴리즈 노트 양식·언어, 태그에 반응하는 워크플로가 무엇을 자동으로 하는지, `gh release list` 양식, 릴리즈 전 검사 스크립트
3. 최근 릴리즈들의 증가 패턴대로 다음 버전을 정하고(패치↔마이너), 같은 양식으로 버전 갱신·노트·릴리즈 커밋·태그를 만든다. 이전 릴리즈가 밟던 검증을 똑같이 통과해야 한다
4. 결과를 `state/<프로젝트>.release.json`에 남긴다: `status`(released/skipped/failed), `version`, `tag`, `title`, `notes_file`, `github_release`, `reason`
5. 러너가 `HEAD:<base>`와 태그를 푸시하고, `github_release=true`(이전에 GitHub Release를 썼고 워크플로가 자동 생성하지 않는 경우)면 `gh release create`
6. 원장에 `- 릴리즈: vX.Y.Z (날짜)` 추가

릴리즈 이력이 전혀 없는 저장소는 **skipped** — 관례를 새로 정하는 건 사람의 일이다. 에이전트는 원격에 아무것도 보내지 않으며(push/tag push/gh release/npm publish 금지), 패키지 배포는 CI 또는 사람이 한다. 옵션: `--no-release`, `--release-budget USD(기본 6)`.

### 릴리즈 자산 (도커 이미지 tar.gz 등)
여러 저장소(Clustara, dataworks, ptium, pii-masker, Invenqor …)는 워크플로가 아니라 **사람이 로컬 스크립트**(`scripts/release.sh`, `scripts/build-offline.sh`, `docker save | gzip`)로 이미지 압축본·sha256·compose/k8s 파일·로드 스크립트를 만들어 GitHub Release 에 올려 왔다. 그래서 릴리즈 에이전트는
1. `gh release view <이전 태그> --json assets` 로 **이전 릴리즈 자산 목록**을 확인하고
2. 자산이 있는데 워크플로가 만들지 않으면 저장소의 스크립트로 **같은 이름 규칙의 파일을 이 기계에서 빌드**해(업로드·푸시 단계는 제외) `release.json` 의 `assets` 배열에 절대경로로 적는다.
3. 러너 `publish_release()` 가 Release 존재를 보장하고(직접 생성 또는 워크플로 생성을 최대 15분 대기) `gh release upload --clobber` 로 올린 뒤, **이전 릴리즈엔 자산이 있는데 이번엔 없으면** `ASSETS MISSING` 을 로그·결과에 남긴다.

이미 나간 릴리즈에 자산이 빠졌으면 `bin/run.sh --assets-only <프로젝트>[:태그]` — 최신(또는 지정) 태그를 체크아웃해 자산만 만들어 올린다(버전·커밋·태그 변경 없음). 도커 빌드가 있으므로 릴리즈 예산 기본값은 $10 이다.

### CI 게이트 (`wait_for_checks()`)
- **PR 머지 전**: PR 커밋의 check-runs 가 모두 끝날 때까지(최대 20분) 기다리고, 실패가 하나라도 있으면 머지하지 않고 PR 을 열어 둔다(`CI failed, PR open`). CI 가 없는 저장소는 5분 뒤 통과.
- **태그 푸시 전**: 릴리즈 커밋을 먼저 밀고 그 커밋의 CI 성공을 기다린 뒤 태그를 민다. moina 처럼 릴리즈 워크플로가 "정확히 그 커밋의 CI 성공"을 요구하는 저장소에서 커밋과 태그를 동시에 밀면 매번 실패했다(v0.1.16~20 이 그렇게 빠졌고 `gh run rerun` 으로 복구). CI 가 실패한 커밋에는 태그를 밀지 않는다.
- 릴리즈 에이전트에게 빌드는 **포그라운드에서 끝내라**고 지시한다 — 세션이 끝나면 백그라운드 빌드는 버려져 `assets missing` 이 됐다(dataworks/ptium/Invenqor 1차 시도).

### 에이전트 지시문 요약 (`prompt.md`)
1. 파악 — CLAUDE.md/README/docs/로드맵/TODO/`git log -30`/테스트·CI
2. 과거 기록 확인 — 원장에 있는 시도·실패 반복 금지
3. 아이디어 5개 — `가치(1-5)/위험(1-5)/작업량(S,M,L)` 채점
4. 1개 선택 — 고가치·저위험·S/M. 대규모 리팩터·아키텍처·메이저 업그레이드 금지
5. 구현 — 기존 스타일 준수, 최소 변경, 테스트 추가
6. 검증 — 테스트/린트/빌드 실제 실행. 실패 시 되돌리고 "실패" 기록
7. 커밋 — 검증 통과 시에만. 저장소의 커밋 메시지 스타일 따름
8. 원장 기록 — 선택/결과/요약/보류 아이디어 4개

**절대 규칙**: push·force-push·브랜치 전환·원격 조작 금지(러너가 함), 비밀값·.env 금지, 코드 외부 반출 금지.

## 4. 안전장치

- **격리**: 임시 워크트리 + 새 브랜치. 사용자의 작업 체크아웃은 건드리지 않음
- **더러운 작업트리 제외**: 미커밋 변경이 있는 저장소는 후보에서 자동 제외
- **원격 권한 분리**: 에이전트에는 push 권한 없음. push/PR/merge는 러너가 수행
- **자기 수정 금지**: `aidev`는 후보에서 제외 — 러너·프롬프트는 사람만 고친다
- **검증 게이트**: 테스트·린트·빌드 통과 시에만 커밋
- **예산 상한**: `--max-budget-usd`
- **겹침 방지**: `flock -n`
- **머지 실패 시**: PR을 열어둔 채 로그에 `merge FAILED` 기록
- **커밋 명의**: 러너가 `GIT_AUTHOR_*`/`GIT_COMMITTER_*`를 `hkjang <gagagiga@naver.com>`으로 강제하고, 에이전트 세션에 `--settings '{"attribution":{"commit":"","pr":""}}'`를 줘 Claude 공동 작성자 트레일러를 붙이지 않는다. 전역 git 설정도 hkjang. GitHub 머지 커밋은 gh 로그인 계정(hkjang) 명의

주의: `--allowedTools`에 `Bash`가 통째로 열려 있다(테스트·빌드에 필요). 워크트리 격리와 규칙으로 막지만 완전한 샌드박스는 아니다.

## 5. 스케줄 (현재 설정)

Windows 작업 스케줄러 `AutoImprove`
```
트리거: 10분마다 (flock 으로 실질 연속 실행)
명령:   C:\WINDOWS\system32\wsl.exe -d Ubuntu-24.04 -u hkjang --
        /mnt/c/Users/USER/projects/aidev/bin/daily.sh --count 1 --budget 20
```
WSL이 꺼져 있어도 `wsl.exe`가 띄운다. PC가 꺼져 있으면 그 트리거는 건너뛴다.

변경 명령 (`/Change`는 암호를 요구하므로 `/Create /F`로 재생성):
```
# 간격 30분
schtasks.exe /Create /F /SC MINUTE /MO 30 /ST 00:00 /TN AutoImprove /TR "<위 명령>"
# 매시간 / 4시간마다
schtasks.exe /Create /F /SC HOURLY /MO 1 ...   |  /SC HOURLY /MO 4 ...
# 중지 / 재개 / 삭제
schtasks.exe /Change /TN AutoImprove /DISABLE
schtasks.exe /Change /TN AutoImprove /ENABLE
schtasks.exe /Delete /TN AutoImprove /F
```

## 6. 수동 실행

```bash
cd /mnt/c/Users/USER/projects/aidev
bin/run.sh --dry-run --no-sync           # 후보·선택만 확인 (커서는 전진함)
bin/run.sh --project weekly --budget 5   # 특정 프로젝트 1회
bin/run.sh --count 2 --no-merge          # 2개, PR만 열고 머지는 사람이
```
옵션: `--dry-run` `--count N` `--project NAME` `--days N(기본 30)` `--budget USD(기본 8)` `--no-merge` `--no-sync`

## 7. 프로젝트별 환경 (`state/<프로젝트>.env`)

DB가 필요한 테스트는 DSN이 없으면 **조용히 SKIP**된다. 자동 머지가 켜져 있으므로 반드시 등록할 것. 이 파일은 git에 올라가지 않으므로 로컬에서만 관리한다.
```
# state/weekly.env  — CI(ci.yaml services.postgres)와 같은 이미지·계정
WEEKLY_TEST_POSTGRES_DSN=postgres://postgres:weekly@localhost:15434/weeklytest?sslmode=disable
```
로컬 테스트용 Postgres 컨테이너: `weekly-test-pg`(pgvector/pgvector:pg16, 15434, `--restart unless-stopped`), `umm-test-pg`(15433), `kanpic-test-pg`(55444).
DSN은 **그 프로젝트 CI가 쓰는 것과 같은 이미지**를 가리켜야 한다 — weekly를 pgvector 없는 umm DB에 붙였더니 테스트가 무더기로 실패해 릴리즈 에이전트가 직접 컨테이너를 띄워야 했다.

## 8. 지금까지의 결과

| 회차 | 프로젝트 | 결과 |
|---|---|---|
| 2026-09-02 #1 | weekly | [PR #1](https://github.com/hkjang/weekly/pull/1) 머지. 주 시작 요일 변경 후 이미 제출한 팀원에게 작성 권고 메일이 가던 버그. `weekCoveringDays()`로 기간 겹침 판정 통일, 회귀 테스트 2개. 실제 DB로 테스트 8개 통과 확인 |
| 2026-09-02 #2 | AgentHub | [PR #1](https://github.com/hkjang/AgentHub/pull/1) 머지. 관리자 로그 검색이 structured field를 못 찾던 버그(`Ring.Entries`→`matches()`), `internal/logging` 첫 테스트 9개. go vet/test -race/web lint+build 통과. 보류 아이디어 4개 원장에 기록 |
| 2026-09-02 #3 | Clustara | [PR #1](https://github.com/hkjang/clustara/pull/1) 머지. 스캔/SBOM 정규화기가 버리던 필드 4가지(Grype CVSS·EPSS, SBOM generated_at, CycloneDX 1.4 generator) 복구 + 첫 단위 테스트 |
| 2026-09-02 #4 | ReSSO | [PR #2](https://github.com/hkjang/ReSSO/pull/2) 머지. JWKS `Cache-Control`이 `writeJSON`의 `no-store`에 덮어써지던 문제. max-age를 `SigningKeyTTL`에 맞추고 `Vary: Origin` 상시 부착. 테스트 2개 추가, go/lint/govulncheck/web 전부 통과 |
| 2026-09-02 릴리즈 | weekly | **v0.281.0** — `chore: v0.281.0 을 냅니다` + `.github/release-notes/v0.281.0.md`, 이전 릴리즈와 동일 양식. 워크플로가 GitHub Release·Offline Docker Release 생성(성공). 릴리즈 에이전트가 CI와 같은 pgvector 컨테이너로 121개 테스트·guard-check 276개 통과, 도중에 발견한 테스트 경쟁 조건(`awaitReminderQueueEmpty`)도 수정 |
| 2026-09-02 릴리즈 | AgentHub | **v0.226.0** — `chore(release): 0.226.0`, 이전 릴리즈 커밋과 동일한 10개 파일 14줄(VERSION, Helm, kustomize, offline compose, docs). release-catalog-images validate/check-versions/kustomize/compose config 통과. Offline Image Release 워크플로가 Release 생성 |

이후 회차는 `state/*.md`(원장)와 `logs/<날짜>.log`, 그리고 이 저장소의 커밋 이력(`run(날짜): 프로젝트 — 결과`)으로 추적한다.

## 9. 운영 중 겪은 문제와 수정 이력

| 문제 | 원인 | 조치 |
|---|---|---|
| 원장 미기록 | 에이전트가 `.claude/` 아래 쓰기 거부 | 원장을 `.claude/` 밖(이 저장소 `state/`)으로 이동, 러너 대체 기록 추가 |
| `merge FAILED` (원격은 머지됨) | 워크트리 안에서 `gh pr merge --delete-branch` → main이 다른 워크트리에 체크아웃 | 워크트리 제거 후 main 체크아웃에서 머지 |
| 1회차 예산 $3 소진 | Go 저장소 676파일 분석에 부족 | 기본 $8, 스케줄은 $20 |
| DB 테스트 SKIP | DSN 없음 | `state/<프로젝트>.env` 주입 |
| `schtasks /Change` 암호 요구 | 비대화 세션 | `/Create /F` 재생성으로 대체 |
| 관리 지점 분산 | 러너는 `.claude/`, 원장은 `~/.auto-improve/` | 전부 `hkjang/aidev` 저장소로 통합, 러너가 회차마다 자동 커밋·푸시 |
| `aidev sync FAILED: empty ident name` | 스케줄러 셸에 git user.name/email 없음 | aidev 체크아웃에 `git config user.name hkjang`, `user.email` 설정 (저장소 로컬 설정) |
| 로컬 태그가 오래돼 "이전 릴리즈"를 잘못 봄 | 로컬 체크아웃은 태그를 안 받아옴 (kanpic 로컬 v0.194 vs 실제 v0.221) | 릴리즈 워크트리 생성 전 `git fetch --tags` |
| 머지는 됐는데 릴리즈 안 됨 (ReSSO, Clustara) | 릴리즈 단계 도입 전 회차 | `bin/run.sh --release-only <프로젝트>`로 사후 릴리즈 |

## 10. 일일 보고 (GitHub Pages)

**https://hkjang.github.io/aidev/** — 회차가 끝날 때마다 러너가 다시 만드는 대시보드.

| 페이지 | 내용 |
|---|---|
| `/` (`docs/index.md`) | 오늘 요약(회차·릴리즈·머지·변경 없음·실패), 최근 14일 표, 프로젝트별 마지막 회차와 최근 릴리즈 |
| `/reports/<날짜>` (`docs/reports/<날짜>.md`) | 그날 회차 표 + 각 프로젝트 원장에서 "무엇을 왜 바꿨나" 발췌 |
| `docs/data/runs.jsonl` | 회차 원본 기록 한 줄씩 `{ts, date, project, result}` — 다른 도구에서 읽기 좋음 |

동작: `run.sh` → 회차 끝 `record_run()` 이 `runs.jsonl` 에 한 줄 추가 → `bin/report.py` 가 `docs/` 전체 재생성 → `sync_repo()` 가 `state logs docs` 를 커밋·푸시 → GitHub Pages(Jekyll)가 1~2분 안에 반영.

SEO / AEO / 모바일:
- `docs/_layouts/default.html` + `docs/assets/style.css` 자체 레이아웃 — viewport, `lang=ko`, 시스템 글꼴, 다크 모드, 44px 터치 타깃, 표는 좁은 화면에서 가로 스크롤, 스킵 링크·빵부스러기·`<main>` 시맨틱
- `jekyll-seo-tag`(title·description·canonical·OG·Twitter 카드·WebPage JSON-LD) + `jekyll-sitemap`(`/sitemap.xml`) + `robots.txt` + `assets/og.png` 공유 이미지
- 페이지마다 front matter `title/description/date/last_modified_at` 과 본문 첫 문장 요약(TL;DR) — 검색·답변 엔진이 그대로 인용할 수 있는 한 문장
- 구조화 데이터: 대시보드에 `WebSite`·`FAQPage`(FAQ 6문항)·`ItemList`(일일 보고 목록)·`Dataset`(runs.jsonl), 일일 보고에 `Report`·`BreadcrumbList`
- 검증: `curl -s https://hkjang.github.io/aidev/ | grep -c 'application/ld+json'` (5), Google Rich Results Test 에 URL 입력

대시보드 구성 요소 (모두 `bin/report.py` 가 생성, JS 없이도 동작):
| 요소 | 내용 |
|---|---|
| ⚠️ 주의 필요 | 최근 2일 회차의 `CI failed`·`merge failed`·`release failed/missing`·`ASSETS MISSING`·자산 빌드 실패와, 최신 릴리즈 자산이 0개인데 이전엔 있던 프로젝트. `ASSETS MISSING` 은 생성 시점에 GitHub 에서 다시 확인해 워크플로가 늦게 붙인 건 빼고, 실패한 워크플로 이름을 함께 적는다. 없으면 초록 "주의 필요 없음" |
| 상태 알약 | 🚀 릴리즈 / ✅ 머지 / ➖ 변경 없음 / ❌ 실패 — 색·아이콘·글자를 함께 써 색만으로 뜻을 나르지 않음 |
| 최근 14일 차트 | 인라인 SVG 누적 막대(상태 팔레트 good/warning/neutral/critical, dataviz 검증). 세그먼트마다 `<title>` 툴팁, 아래 표가 테이블 뷰 |
| 표 | HTML `table.rt` — 셀마다 `data-label`, 행마다 `data-status`. **640px 이하에서는 카드로 접힘**, JS 가 있으면 검색창 + 상태 칩 필터 |
| 프로젝트 페이지 | `/projects/<이름>/` — 현황(마지막 회차, 최근 릴리즈·자산 수·누락 표시), 회차 이력, 원장 전문. `SoftwareSourceCode` JSON-LD |
| Atom 피드 | `/feed.xml` — 일일 보고 30건. RSS 리더·Slack 등에서 구독 |
| summary.json | `/data/summary.json` — 오늘·누적·일별·프로젝트별·주의 필요를 한 번에. 외부 도구용 |
| PWA | `manifest.webmanifest` + 192/512 아이콘 — 모바일 홈 화면에 추가 가능, safe-area 여백 |
| 기타 | 맨 위로 버튼(스크롤 600px 뒤), 갱신 시각 상대 표시("n분 전"), 인쇄 스타일, `prefers-reduced-motion`, 44px 터치 타깃 |
매일 아침 대시보드의 "일일 보고" 표에서 어제 날짜를 누르면 된다. 수동 재생성은 `python3 bin/report.py`.

## 10-1. 실패 복구 · 알림 · 주간/월간 · 비용

**릴리즈 워크플로 실패 복구** (`retry_release_workflow()`)
1. 태그 워크플로가 실패하면 `gh run rerun --failed` 로 한 번 재실행하고 최대 20분 기다린다 → 성공하면 `recovered on rerun`
2. 또 실패하면 실패 단계·실행 URL·로그 요지를 `state/fix-queue.tsv` 에 넣는다 (프로젝트당 1건)
3. 다음 회차에서 러너는 라운드로빈보다 **큐의 프로젝트를 먼저** 집고, `prompt.md` 의 `$FIX_NOTE` 로 "새 아이디어 대신 이 실패를 고쳐라 — 검증을 느슨하게 만드는 건 금지" 를 지시한다. 회차 결과는 `fix-round:` 로 표시되고 큐에서 빠진다
4. 큐에 있는 동안 대시보드 '주의 필요' 와 프로젝트 페이지에 '수정 과제' 로 보인다

**알림** (`bin/notify.sh`, 보고 생성 직후 실행) — `summary.json` 의 alerts 중 처음 보는 것만:
| 채널 | 설정 | 비고 |
|---|---|---|
| GitHub Issue | 없음 | 라벨 `alert`, 열린 이슈가 있으면 코멘트. 추적용 — 본인 활동은 GitHub 이 이메일로 알리지 않음 |
| Windows 토스트 | 없음 | `bin/toast.ps1` (PowerShell AppId 사용) |
| Slack | `AIDEV_SLACK_WEBHOOK` | incoming webhook |
| 이메일 | `AIDEV_SMTP_URL/USER/PASS`, `AIDEV_MAIL_TO` | curl SMTP. 네이버는 `smtps://smtp.naver.com:465` + 앱 비밀번호 |
설정 파일: `state/notify.env.example` → `~/.auto-improve/notify.env`. 해소된 경고는 seen 목록에서 빠져 재발 시 다시 알린다.

**주간·월간 보고** — `/weekly/YYYY-Www/`, `/monthly/YYYY-MM/`: 기간 요약(활동일·실패율·비용), 날짜별 차트·표, 프로젝트별 회차·릴리즈 태그, 실패·경고 회차 목록. 대시보드 '주간·월간 보고' 절과 `summary.json` 의 `weeks`/`months` 에도 있다.

**비용·사용량** — `claude -p --output-format json` 의 `total_cost_usd`·`duration_ms`·`num_turns`·`usage` 를 `record_usage()` 가 `docs/data/usage.jsonl` 에 남긴다(단계: improve/release/assets). 사람이 읽을 답변은 종전처럼 `logs/<날짜>-<프로젝트>.txt`. 대시보드 '비용·사용량', 일일 보고, 프로젝트 페이지에 표시. **정액제에서는 참고값**이다.

## 10-0. 러너 v2 — 검증되지 않은 변경 차단 (1단계)

원칙: **에이전트는 제안하고, 러너가 검증하며, 게시는 러너만 한다. 확인하지 못한 것은 성공이 아니다.**

| 항목 | 구현 | 완료 기준 |
|---|---|---|
| CI 판정 | `bin/gate.py ci` — check-runs 를 `--paginate` 로 전부 받아 API 오류·파싱 오류·빈 응답·미완료·취소·시간 초과·다른 커밋을 성공과 구분. 정책 `required_checks` 로 필수 검사 지정, 재실행은 최신 시도만. 두 번 연속 통과 필요. CI 없는 저장소는 정책 `allow_merge_without_ci: true` 일 때만 | 필수 검사 성공을 확인하지 못하면 PR 유지, 머지 안 함 (`CI <상태>, PR open`) |
| 판정 회귀 테스트 | `tests/test_gate.py` 26건 — 정상/실패/취소/시간초과/진행 중/빈 응답/API 오류/페이지네이션/재실행/커밋 불일치/리뷰 판정값/릴리즈 스키마·자산 경로/아이디어 스키마/비밀정보 | `python3 tests/test_gate.py` 통과 |
| 리뷰 명시 승인 | `gate.py review` — 결과 파일 존재·JSON 유효·`verdict ∈ {approve, reject}` 를 각각 검증. 누락·알 수 없는 값은 보류 | 유효한 `approve` 일 때만 CI 단계로 |
| 러너 직접 검증 | `run_verify()` — 정책 `verify` 명령(없으면 go/npm/pytest/cargo/make 자동 감지)을 러너가 실행, 종료 코드·소요 시간을 `state/runs/<run_id>/verify.json` 에 기록. 실패하면 PR 을 열지 않음 | 머지되는 모든 변경에 러너가 수집한 검증 결과가 run_id 로 연결 |
| 에이전트 권한 격리 | `run_agent()` — `env -i` + **임시 HOME**(gh 미인증, git 자격증명 없음, 홈의 비밀 파일 없음), 토큰 환경변수 없음, `CLAUDE_CONFIG_DIR` 만 실제 경로, 워크트리 push URL `DISABLED`, `--add-dir` 는 실행별 출력 디렉터리만. GitHub 정보는 러너가 미리 뽑아 프롬프트에 넣음(`release_context()`) | 에이전트가 원격 머지·릴리즈 게시·정책 변경을 직접 할 수 없음. **잔여 위험**: 도커 소켓은 자산 빌드를 위해 아직 노출 |
| 정책·기록 보호 | 에이전트는 `$OUT`(실행별 디렉터리)에만 씀. 원장 항목(`ledger-entry.md`)·아이디어(`ideas.json`)·릴리즈(`release.json`)·리뷰(`review.json`)를 러너가 `gate.py` 로 검사한 뒤 영구 기록에 반영(`merge_outputs()`). `default.policy.json`·`caps.env`·`*.guard`·다른 프로젝트 원장은 에이전트가 보지도 고치지도 못함 | 작업 중 정책 완화·타 프로젝트 기록 변경 불가 |
| 기준 커밋 고정 | 정책 `base_branch`, `origin/<base>` 를 fetch 한 SHA 에서 시작(`BASE_SHA`), 리뷰·검증·CI 는 `HEAD_SHA` 에 연결. 머지 직전 base 가 움직였으면 리베이스 → 재검증 → force-with-lease → CI 재확인. 머지는 `gh pr merge --match-head-commit` | 검증하지 않은 커밋·의도하지 않은 브랜치로 머지되지 않음 |
| 결과 파일 검증 | `gate.py release/ideas` 스키마, 자산은 `$OUT/assets/` 안의 존재하는 비어 있지 않은 파일만, 태그 형식 검사. 커밋·태그 존재는 러너가 git 으로 재확인 | 잘못된 판정값·외부 경로·없는 자산으로 후속 작업 없음 |
| 비밀정보·공개 분리 | `gate.py secrets` — PR/릴리즈 diff 에 토큰·키·자격증명 URL 이 있으면 푸시 안 함. 원장 항목은 사설 IP·내부 호스트까지 검사해 걸리면 `state/private/`(git 제외)로. 러너 로그는 커밋 전 `redact_log()` | 토큰·내부 주소가 공개 보고에 들어가지 않음 |

실행 기록: `state/runs/<run_id>/` 에 `run.json`·`stages.json`(단계별 상태/사유/시각)·`verify.json`·`review.json`·`release.json`·`ci-*.json`. `runs.jsonl` 에는 `run_id`·`base_sha`·`head_sha`·`pr`·`outcome`(`no-change | review-pending | verify-failed | merged | releasing | release-ready | error`)·`stages` 가 구조화 필드로 들어간다.

## 10-1a. 러너 v2 — 실행·릴리즈 신뢰성 (2단계)

| 항목 | 구현 |
|---|---|
| 실행 ID | `state/runs/<날짜-시각-프로젝트-종류>/` 에 실행별 증거(`run.json` 시작/완료/outcome, `stages.json`, verify/review/release/ci 결과). 같은 날 같은 프로젝트를 여러 번 돌려도 덮어쓰지 않음 |
| 구조화 상태 | `runs.jsonl` 의 `outcome`(변경 없음·검토 대기·검증 실패·병합 완료·릴리즈 진행 중·배포 준비 완료·실행 오류)과 `stages`(단계별 state/reason/at). 대시보드·통계는 outcome 기준, 옛 기록은 문장으로 추정 |
| 중단 재개 | `resume_runs()` — 시작 시 6시간 안의 미완료 improve 실행을 찾아 원격 상태로 이어감: PR 없음→실행 오류로 닫음(에이전트 단계는 재실행 안 함), PR 머지됨→릴리즈만, PR 열림→유효한 리뷰 결과가 있으면 CI→머지→릴리즈. PR·태그·Release 는 존재 확인 후 재생성하지 않음 |
| 실패 유형별 재시도 | `with_retry` — stderr 로 분류: 인증/권한 → 즉시 중단, 충돌/거절 → 재시도 없음, 네트워크·5xx·rate limit → 10/30/90초 간격 3회. 시작 시 `gh auth status` 실패면 회차를 열지 않음(exit 3) |
| 잠금·종료 | `~/.auto-improve/run.owner` 에 pid·시작 시각·스크립트 기록(종료 시 제거). 단계별 제한 시간 `timeout -k 30`(개선 45분·리뷰 15분·릴리즈/자산 60분) — 초과 시 그 단계 프로세스만 종료하고 `timeout` 으로 기록. 다른 회차의 작업 디렉터리는 건드리지 않음 |
| 전체 회차 예산 | 개선+리뷰+릴리즈 예산 합이 오늘 남은 상한 안에 있어야 시작(`hold: budget`). 단계 시작 전 `budget_ok` 재확인. `--project` 수동 실행에도 일일 상한 적용. 비용 없는 세션은 `cost_usd: null`(미확인)로 기록해 0 과 구분 |
| 릴리즈 완료 조건 분리 | `stages`: `release`(태그 게시) / `gh-release`(Release 생성) / `manifest` / `assets`(업로드·검증). `release-ready` 는 자산 검증까지 끝났을 때만 |
| 자산 매니페스트 | `state/<프로젝트>.assets.json` (`required` 이름 패턴 `{version}`/`{tag}`, `min_bytes`) — 없으면 **이전 릴리즈 자산 이름에서 버전만 치환한 목록**을 필수로 삼음. 누락·너무 작은 파일·`.sha256` 불일치면 업로드하지 않고 `manifest failed` |
| 자산 불변성 | 같은 이름의 게시 자산은 체크섬 비교 → 같으면 건너뜀, 다르면 `asset conflict` 로 사람 확인 (덮어쓰기 없음) |
| 코드/배포 복구 분리 | 롤백 PR(코드) 과 별도로 라벨 `deploy-recovery` 이슈를 열어 이전 정상 릴리즈·자산 링크를 안내. 머지 커밋에 `migrations/` 가 있으면 자동 복구 대상이 아님을 명시하고 사람 계획·승인 요구 |
| 품질 지표 | 대시보드 '품질 지표(최근 14일)': 검증된 개선 완료율(24h 관찰 후 회귀 없음), 완전한 릴리즈 비율, 사람의 재작업률, 변경 후 회귀율, 유효 개선당 비용(비용 확인 세션만), 예외 처리 소요 시간 중앙값(`state/alerts-history.jsonl` open→close), 실행 오류 수 |

## 10-2. 품질·안전 게이트

| 단계 | 파일 | 동작 |
|---|---|---|
| **회귀 감시** | `bin/regress.sh` (회차마다) | 머지 2~48시간 뒤 머지 커밋의 CI 실패·되돌림 커밋을 확인해 `state/lessons.jsonl` 에 교훈 기록 → `prompt.md` `$LESSONS` 로 그 프로젝트 다음 회차에 주입, `/lessons/` 페이지 |
| **리뷰 게이트** | `review-prompt.md`, `review_pr()` | PR 생성 후 CI 대기 전에 별도 세션이 diff 만 읽고 거절 사유 탐색(검증 안 하는 테스트·논리 오류·설명 불일치·위험 변경). reject → PR 코멘트 + 열어 둠(`review rejected`). `--no-review`, 예산 `RVBUDGET`(기본 $4) |
| **보호 파일** | `state/default.guard`, `state/<프로젝트>.guard` | 정규식에 걸리는 파일(workflows, migrations, auth, payment, Dockerfile/compose, deploy, secret, LICENSE)을 건드리면 자동 머지 안 함 + PR 코멘트(`guarded files`) |
| **자동 롤백** | `rollback_project()` | 수정 회차(fix-round)까지 머지에 실패하면 큐에 저장된 원래 머지 커밋을 `git revert` 한 PR 을 열고 교훈 기록(`rollback PR`). 머지는 사람 |
| **헬스체크** | `bin/health.sh`, 작업 스케줄러 `AutoImproveHealth`(30분) | 마지막 회차 3시간 초과·gh 인증·docker·디스크 점검, 3시간 넘게 매달린 러너 종료·잠금 해제, `docs/data/health.json` 갱신·푸시, 문제 시 토스트/Slack |
| **건강 등급** | `report.py health_grade()` | 프로젝트별 최근 14일 실패·경고·회귀로 A~D. 프로젝트 표와 페이지에 알약으로 표시 |

## 10-3. 운영 편의

| 기능 | 파일 | 동작 |
|---|---|---|
| **아이디어 백로그** | `state/<프로젝트>.ideas.json`, `prompt.md` 3·9단계 | 에이전트가 회차마다 `pending` 항목을 재평가(done/rejected)하고 새 아이디어 2개 이상을 더해 저장. 다음 회차는 "새로 5개" 대신 백로그 최고점부터. 프로젝트 페이지 '아이디어 백로그' 표 |
| **휴면** | `caps.env` `DORMANT_AFTER/DAYS` | "변경 없음"이 3회 연속이면 7일간 후보 제외(수동·수정 큐는 예외) |
| **일일 상한** | `state/caps.env` | 비용·회차·릴리즈가 상한에 닿으면 그날은 새 회차를 시작하지 않고 토스트 + 회차 기록(`(runner)` 행) + 대시보드 '오늘 상한' 게이지 |
| **수동 트리거** | `bin/inbox.sh` | aidev 저장소에 라벨 `run`, 제목 `run: <프로젝트>` 이슈를 만들면 다음 회차에 우선 실행 → 끝나면 이슈에 결과 코멘트 후 닫음. 후보가 아니면 안내 코멘트 |
| **변경 미리보기** | `RUN_META` → `runs.jsonl` | 파일 수·+/−·테스트 파일 수·커밋 제목을 회차 행 아래 한 줄로(테스트가 없으면 *테스트 없음* 강조) |
| **주간 다이제스트** | `bin/digest.sh` (월요일 첫 회차) | 지난 7일의 사람 판단 PR(리뷰 보류·보호 파일·CI 실패), 롤백 PR, 릴리즈 관례 없는 프로젝트, 수정 큐, 새 교훈을 라벨 `digest` 이슈 하나로. `--force` 로 즉시 |
| 스케줄러 상태 | `health.sh` | `health.json` 에 작업 스케줄러 상태·다음 실행 시각(한글/영문 Windows 모두) |

## 11. 점검 루틴

- **매일**: `tail -50 logs/cron.log` — `merge FAILED`, `claude exited non-zero`(사용량 한도 가능성), `aidev sync FAILED` 확인
- **주간**: `state/*.md` — "변경없음"·"실패" 비율이 오르면 간격을 늦추거나(`/MO 30`) 후보 기준(`--days`) 조정
- **머지 커밋 검토**: 각 저장소 `git log --merges --since=1.week` 로 자동 머지된 PR 훑기
- **GitHub에서**: https://github.com/hkjang/aidev/commits/main — 회차별 커밋 한 줄이 곧 실행 이력

## 11. 정액제 관련

- 비용은 요금제에 포함되지만 **5시간 사용량 창**에 걸리면 회차가 실패로 끝난다. 러너는 다음 트리거에 자동 재시도한다.
- `--max-budget-usd`는 추정 비용 기준 안전장치일 뿐이며, 정액제에서도 그 값을 넘으면 세션이 끊긴다. 작업 도중 끊기지 않도록 넉넉히($20) 잡았다.
