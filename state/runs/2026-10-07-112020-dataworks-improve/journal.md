# 회차 노트 2026-10-07-112020-dataworks-improve — dataworks
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:20] base pinned — main@016b281
- [러너 11:20] autonomy release — 
- [러너 11:20] scout skipped — 요청된 작업이 있어 정찰을 건너뛴다 (과제는 이미 정해져 있다)

## 구현 노트
- 프로덕션 코드 0파일. 실패한 Pages 실행(37371053021)의 `build` 잡은 `runner_name: ""`·단계 0개로 15분 큐 대기 후 cancelled — 저장소 파일을 한 줄도 읽지 않았다. 성공 대조군(37260518982)은 단계 7개·21초. 같은 SHA 의 CI 는 성공. 그래서 코드는 고치지 않고 `docs/OPERATIONS.md` 에 판별 절차만 추가했다(c953e90).
- 확신 있는 것: CI 와 **같은 이미지**(`ghcr.io/actions/jekyll-build-pages:v1.0.13`)로 `INPUT_SOURCE=docs`·`./` 양쪽 빌드가 exit 0(오류·경고 0건), 음성 대조로 하네스가 exit 1 을 낼 수 있음 증명, 문서 변경 후 재빌드 exit 0 + `_site/OPERATIONS.html` 에 새 절 반영(end-to-end).
- **확신 없는 곳**: (1) 러너 미배정의 GitHub 내부 사유는 모른다 — 용량 문제인지 관리형 Pages 잡의 15분 타임아웃인지 가르지 못했다. (2) 로컬 재현은 `jekyll-github-metadata` 를 우회하려고 `GITHUB_API_URL` 을 로컬 스텁(404/최소 JSON)으로 돌렸다 — 실제 CI 의 `site.github` 값과 다르므로 metadata 값에만 의존하는 실패는 이 재현이 못 본다. (3) Go·web 검증은 돌리지 않았다(변경 파일이 `docs/OPERATIONS.md` 하나뿐).
- 일부러 하지 않은 것: `.nojekyll` 추가·독자 Pages 워크플로 추가(Pages 소스 설정을 바꿔야 하고 지금 실패와 무관), 재현에서 발견한 `Clustara_Detailed_Manual.md`/`.html` 출력 경로 충돌(과제 범위 밖 — ideas.json 에 신규 후보로 남김).
- 다음 역할이 조심할 것: 이 실패는 저장소에서 고칠 수 없다. **조치는 같은 커밋 재실행뿐**이고 그때까지 사이트는 v0.9.67 을 서빙한다(`curl -s https://hkjang.github.io/dataworks/ | grep -o 'v0\.9\.[0-9]*'`). 그리고 `docs/*.md` 는 Pages 에서 페이지로 렌더되므로 문서에 Liquid 구문(이중 중괄호 등)을 넣으면 백틱 안이라도 Pages 빌드가 깨진다 — 초안에서 실제로 밟아 지웠다.
- [러너 11:35] verify passed — 검증 8개 통과 (auto)

## 비평 노트
- 기준을 러너가 고정한 `main@016b281` 로 잡아 실제 변경이 `docs/OPERATIONS.md` 1파일 +28줄임을 확인했다(로컬 `main` ref 는 v0.9.53 이라 69파일 거짓 회귀를 보여준다 — 함정 회피). 사실 주장은 전부 검증해 맞다: `_config.yml`·`.nojekyll`·`Gemfile` 없음, 워크플로는 `ci.yml` 뿐. 추가된 28줄에 중괄호가 0개라 자기 Pages 렌더를 깨뜨릴 수 없고, 93줄 python3 한 줄은 돌려서 문서가 쓴 `build cancelled '' 0` 을 그대로 냈다.
- **거절 사유는 79줄 한 줄에 몰려 있다**: `git archive HEAD | tar -x -C /tmp/pages-ws` 를 그대로 실행하면 디렉터리가 없어 exit 2 로 죽고(실행해 확인), 디렉터리가 있으면 이전 추출물을 지우지 않아 '수정 전/후 두 번 빌드' 라는 문서 자신의 사용법에서 구 파일이 남아 CI 에 없는 content 실패를 만든다. 두 번째가 조용한 오진이라 더 해롭다. 수리는 `rm -rf /tmp/pages-ws && mkdir -p /tmp/pages-ws` 한 줄로 끝난다 — **79줄만 보면 된다.**
- 수리가 겸사겸사 볼 곳(차단 아님): 96줄의 판별 기준은 concurrency 로 취소된 선행 실행과 러너 미배정을 구별하지 못한다(둘 다 runner_name 빈 문자열·단계 0개·cancelled). 그 경우 올바른 조치는 재실행이 아니라 뒤 실행 확인이므로 한 문장 단서가 필요하다. 83줄은 토큰 범위를 안 적어 과도한 PAT 유도 가능(공개 저장소는 scope 없는 토큰으로 충분).
- 못 본 것: 실제 GitHub Pages 실행·러너 미배정의 내부 사유·실제 `site.github` 값 의존 실패는 이 환경에서도 검증 불가(구현 노트의 '확신 없는 곳' 3건 중 2건은 환경 한계로 남는다). Go·web 검증은 변경 파일이 문서 하나뿐이라 돌리지 않았다. docker 이미지로 Pages 빌드를 실제 재현하지는 않았다 — 명령의 인자 타당성만 정적으로 읽었다.
- security·legal 차단 없음(`blocking: []`): 새 엔드포인트·인가 변경·개인정보·의존성 추가가 전혀 없고 참조 이미지는 CI 가 쓰는 공식 이미지다. 릴리즈 노트용: 이 회차는 코드 동작을 바꾸지 않으며, Pages 실패 자체는 저장소에서 고칠 수 없고 재실행만이 조치다.
- [러너 11:38] review rejected — 리뷰 거절: docs/OPERATIONS.md:79 `git archive HEAD | tar -x -C /tmp/pages-ws` 는 복사-붙여넣기로 동작하지 않는다. `/tmp/pages-ws` 를 만드는 단계가 없어 새 머신에서 `ta

## 수리 노트
- 지적 2건 모두 **맞았다** — 둘 다 실행해 재현했다(디렉터리 없음 → tar exit 2, 디렉터리 재활용 → 심어둔 파일이 2회차 추출 뒤 생존). 틀린 지적은 없었다. 비차단으로 짚은 96줄(concurrency 취소 vs 러너 미배정)·83줄(토큰 스코프)도 같이 반영했다.
- 고친 방법: 블록 첫 줄에 `rm -rf /tmp/pages-ws && mkdir -p /tmp/pages-ws` + "빼지 말라"는 근거 문단. 커밋 `94ffb94`, `docs/OPERATIONS.md` 1파일 +37줄.
- **비평가가 못 본 것을 수리 중 찾았다**: 컨테이너가 `_site/` 를 root 소유로 쓰기 때문에 `rm -rf` 만 더한 수정본도 2회차에 `Permission denied` + exit 1 로 사슬이 끊긴다 — 즉 지적대로만 고치면 여전히 복사-붙여넣기로 안 된다. `docker run --user "$(id -u):$(id -g)"` 를 더해 해결했다(빌드 exit 0, `_site/` 호출자 소유 확인).
- 검증은 정적 확인이 아니라 실제 실행이다: 문서 블록을 awk 로 뽑아 2회 연속 실행(둘 다 exit 0), HEAD 에 없는 깨진 md 를 심고 재실행(지워지고 exit 0 — 유령 실패가 실제로 막힘), 음성 대조로 미종결 Liquid 태그 exit 1 확인. 중간에 `rm -rf` 종료 코드를 파이프 너머 `$?` 로 잘못 읽어 "exit 0" 이라 쓸 뻔했고, 다시 측정해 exit 1 로 고쳐 문서에 반영했다.
- 여전히 확신 없는 곳: 토큰은 로컬 API 스텁으로 대체했으므로 실제 `site.github` 값에만 의존하는 실패는 못 본다. 러너 미배정의 GitHub 내부 사유도 그대로 미해결이며, 96줄의 새 판별 단서는 API 응답 형태로만 추론한 것이고 실제 미배정 실행에 적용해 보지는 못했다.
- [러너 11:46] repair done — # fix-summary (수리 시도 2) — docs/OPERATIONS.md:79  - 비평 두 건 모두 **재현했고 맞았다**: 디렉터리 없이 `git archive HEAD | tar -x -C /tmp/pages-ws` 는 exit 2(`Cannot open:

## 비평 노트
- base 를 러너가 고정한 `main@016b281` 로 잡아 실제 변경이 `docs/OPERATIONS.md` 1파일 +35줄임을 확인했다(로컬 `main` 은 v0.9.53 — 69파일 거짓 회귀 함정 회피). 코드·테스트·워크플로·의존성 변경 0건이라 범위 이탈 없고 revert 가 쉽다.
- **정적 확인이 아니라 실제로 돌려서 검증했다.** 문서 블록을 그대로 복사해 2회 연속 실행 → 두 회차 모두 exit 0, `_site` 호출자 소유, 사슬 안 끊김. 음성 대조로 `--user` 를 빼니 `_site` 가 root:root 가 되고 다음 회차 `rm -rf` 가 Permission denied + exit 1 — 수리 노트의 주장과 정확히 일치한다. 이전 비평의 거절 사유는 해소됐다.
- 사실 주장은 라이브 API 로 전부 맞췄다: 실패 실행 37371053021(`build cancelled`/`runner_name ''`/steps 0/`deploy skipped`), 정상 대조군 37260518982(steps 7/21초), 사이트 v0.9.67. 새 concurrency 판별 단서도 실데이터(37370095013 → 37371053021)가 뒷받침한다. 토큰 지침도 gem 소스로 확인 — `Octokit::NotFound` 는 rescue 되므로(client.rb:98) 스코프 없는 토큰이면 충분하고, 404 스텁으로 돌린 빌드가 exit 0 이었다. 최소권한 방향이라 보안상 바람직하다.
- 못 본 것: 실제 스코프 없는 PAT 로는 못 돌려봤고(소스 + 404 스텁으로 대신), 따라서 실제 `site.github` 값에만 의존하는 실패와 러너 미배정의 GitHub 내부 사유는 여전히 미확인이다. security·legal 차단 없음(`blocking: []`) — 새 엔드포인트·인가 변경·개인정보·의존성이 전혀 없다.
- **승인. 다음 회차가 알아야 할 것**: 블록에 `$GITHUB_TOKEN` 가드가 없어 미설정으로 붙여넣으면 Liquid 검사 전에 BadCredentials 로 멈춘다(문서가 바로 아래에서 예고하므로 비차단) — '먼저 토큰을 설정하라' 한 줄이면 1단계가 무토큰으로 무력화되지 않는다. 범위 밖이지만 `Clustara_Detailed_Manual.md`/`.html` 출력 충돌은 재현에서 실재 확인(빌드는 exit 0).
- [러너 11:51] review approved — 리뷰 승인 (risk=low)
- [러너 11:51] pr created — https://github.com/hkjang/dataworks/pull/36
- [러너 11:54] ci passed — 검사 2개 모두 success
- [러너 11:54] merge done — 94ffb94
- [러너 12:03] release published — v0.9.69
- [러너 12:03] gh-release created — GitHub Release v0.9.69
- [러너 12:03] manifest ok — dataworks-v0.9.69.tar.gz 
- [러너 12:03] assets uploaded — 1개
- [러너 12:03] assets verified — v0.9.69 자산 1개 (이전 v0.9.67: 1)
