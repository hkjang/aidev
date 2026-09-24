# 회차 노트 2026-09-23-210505-aiportal-front-improve — aiportal-front
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:05] base pinned — main@108cc47
- [러너 21:05] autonomy release — 

## 정찰 노트
- 보류 목록의 남은 항목이 전부 "관측 가능한 동작 변화 없음"(alerts.js 제거, Confirm import 제거) 또는 "기대 계약 미확정 선행"(loading.vue 타이머, showHeader, globalLoading 참조 카운트)이라 새로 찾았다. 릴리즈는 14회째 진입 조건 미충족 — 저장소 안에 합법적 수단이 없어 그대로 pending.
- 고른 것: PopSimpleBot/PopSimpleBotUpdate 의 `reload()` — `if(isSuccess(res))` 에 else 가 없어 실패 시 타이핑 애니메이션 문구가 입력란에 얼어붙고 `payload.prompt` 로 서버까지 간다. 소스로 끝까지 추적했다(interceptors.js:227-236 이 BZ01 아닌 200 응답을 통과시킴 → common.js:55-64 isSuccess false → PopSimpleBot.vue:344 else 없음 → :505 전송).
- 확신 없는 곳: 서버가 실제로 그런 실패 응답을 얼마나 자주 내는지는 백엔드가 없어 미확인이다. 또 `validatePromptCreate` 통과를 위해 테스트에서 name 입력을 실제 DOM 으로 채우는 구체적 셀렉터는 확인하지 못했다 — 구현자가 마운트해 보고 정할 것.
- 구현자가 조심할 것: 응답을 즉시 resolve 시키면 150ms 인터벌이 한 번도 안 돌아 Red 가 안 나온다(fake timer 또는 지연 resolve 필수). `catch` 의 'COM' 경로에 안내를 더하면 인터셉터 Alert 과 중복된다. 공용 `startLoopTyping` 은 건드리지 말 것.
- 차선(ChatData.vue:708 source_seq JSON.parse)은 트리거 입력이 실증되지 않아 무효 변경 위험이 있다 — 재현 테스트부터 세우지 않으면 고르지 말 것.
- [러너 21:11] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- PopSimpleBot/PopSimpleBotUpdate 의 `reload()` 에 실패 else 를 넣었다. 기존엔 `isSuccess(res)` false 일 때 아무 것도 하지 않아 타이핑 안내 문구가 입력란에 얼어붙고 그대로 `payload.prompt` 로 전송됐다. 'COM' 이 아닌 예외에도 같은 토스트를 붙였다(COM early return 뒤라 인터셉터 Alert 과 겹치지 않는다 — 변이로 확인).
- 확신 없는 곳: ①서버가 HTTP 200 + 본문 code 불일치 응답을 실제로 얼마나 내는지는 백엔드가 없어 미확인이다(경로가 도달 가능함은 interceptors.js:228-237 로 증명). ②토스트 문구 '추천 프롬프트를 가져오지 못했습니다.' 는 이 파일의 기존 관례(`toast('앱생성에 실패하였습니다.')`)를 따라 내가 정한 것이지 디자인 확정본이 아니다. ③실패 시 `promptDisable`/`selectedPromptId` 는 손대지 않았다 — 기대 계약 미확정(ideas.json 에 항목으로 남김).
- 일부러 하지 않은 것: 두 컴포넌트의 중복된 `reload()` 통합(대규모 리팩터 금지), `startLoopTyping` 수정(공용 유틸), 차선 후보 ChatData.vue source_seq(트리거 입력 미실증).
- 다음 역할이 조심할 것: 새 스펙은 `startLoopTyping` 의 150ms 실제 setInterval 을 돌려야 Red 가 나므로 응답을 deferred 로 잡고 `await new Promise(r=>setTimeout(r,500))` 로 실제 시간을 흘려보낸다 — 케이스당 ~0.5s 가 정상이다(느린 게 아니라 의도). 즉시 resolve 하는 대역으로 바꾸면 무딘 테스트가 된다. DB·네트워크는 필요 없다(axios adapter 만 대역).
- [러너 21:21] verify passed — 검증 2개 통과 (auto)

## 비평 노트
- 확인함: Red 재현(두 .vue 만 main 으로 되돌려 10개 중 6개 실패) → 수정본 10/10 통과 → 전체 28파일 487테스트 통과(기준선 477+10, 회귀 없음). 테스트는 실제 인터셉터·실제 API·실제 마운트를 통과시키고 settle 전에 타이핑이 돌았음을 먼저 단언하므로 무딘 테스트가 아니다.
- 구현자의 3가지 의심 모두 확인: COM 중복 안내 없음(spec 3번째 케이스가 Alert 상태로 단언), 토스트 문구는 파일 관례와 일관, `promptDisable`/`selectedPromptId` 미변경은 데이터 파괴로 이어지지 않음 — 두 파일 제출 검증의 `if (!p) return false`(PopSimpleBot.vue:227 / Update:315)가 빈 prompt 전송을 막는다.
- 못 본 것: 백엔드가 없어 서버가 HTTP 200 + code 불일치 응답을 실제로 내는 빈도는 여전히 미확인(경로 도달 가능성만 소스로 증명). 브라우저 실행·수동 확인은 하지 않았다.
- 승인이어도 남는 우려(다음 회차 1순위): 같은 계열 구멍이 **성공 분기**에 남아 있다 — PopSimpleBot.vue:344-349 / PopSimpleBotUpdate.vue:431-436 은 `isSuccess` true 인데 `body.prompt` 가 비면 rolePrompt 에 아무 것도 대입하지 않아 타이핑 문구가 그대로 남고 payload 로 간다. 구현자가 그 경우를 `typeof ... && trim()` 으로 예상해 놓고 처리를 비운 자리다.
- 잔여 UX(계약 미확정): Update 는 promptDisable=true(:196) 상태라 실패 후 '비어 있고 입력 불가'가 되고 제출은 토스트 없이 조용히 막힌다 — 회복은 '프롬프트 초기화'뿐. 악화는 아니지만 릴리즈 노트에 남길 것. 테스트 위생 하나: afterEach 의 `delete axios.defaults.adapter` 는 내장 기본 adapter 를 지운다(파일 격리 덕에 현재는 안전).
- [러너 21:24] review approved — 리뷰 승인 (risk=low)
- [러너 21:24] pr created — https://github.com/hkjang/aiportal-front/pull/28
- [러너 21:24] ci passed — 검사 없음 — 정책으로 허용
- [러너 21:24] merge done — c0ff704

## 릴리즈 노트
- 판정: **failed** (15회째 동일 원인). 추측 없이 이번 회차에 전부 재조회했다: `git tag` 0개(`--is-shallow-repository`=false), package.json/package-lock.json `version`=0.0.0 이며 최초 커밋 `ab700ba` 부터 한 번도 증가한 적 없음(lock 은 최초에 version 필드 자체가 부재), CHANGELOG/VERSION/릴리즈 노트 파일 없음, `scripts/`·`Makefile`·`.github` 없음, 러너가 미리 가져온 GitHub Release 목록·워크플로 파일 모두 비어 있음.
- `.gitlab-ci.yml` 은 릴리즈 워크플로가 아니다 — main/develop **브랜치 조건**으로 `npm run build:*` 후 `dist` 를 서버 디렉터리에 `cp` 하는 배포만 정의한다. 파일 안의 `tags:` 는 전용 Runner 태그이지 git 태그가 아니며, 태그 규칙·버전 증가·릴리즈 생성·산출물 업로드가 전혀 없다. 따라서 이전 릴리즈 자산도 없고 `assets` 는 빈 배열이다.
- 막힌 지점은 절차 2(다음 버전 결정)다. 과거 릴리즈가 0건이라 따라갈 증가 패턴 자체가 존재하지 않고, 0.0.1/0.1.0 중 무엇을 골라도 새 관례를 발명하는 것이 된다 — AGENTS.md 와 docs/RELEASE.md:83 이 명시적으로 금지한다. 절차 5 의 `skipped`(태그·버전 파일·릴리즈 노트가 모두 없음)는 **버전 파일이 존재하므로** 충족하지 않는다(docs/RELEASE.md:84 와 같은 판정).
- README.md:434 / docs/01-시작하기.md:356 의 과거 `0.0.1 / 2025-01-01 / 초기 릴리스` 기재는 현재 docs/RELEASE.md 로 연결되어 있고 실제 릴리즈 사실이 미확인이라 증가 기준으로 쓸 수 없다.
- 이번 회차에 만들지 않은 것: 버전 파일 변경, 태그, CHANGELOG, 릴리즈 노트, 릴리즈 커밋. 원격에 아무것도 보내지 않았다. 작업 트리는 그대로 깨끗하다(`git status --porcelain` 무출력, HEAD=38cc2c3).
- 다음 회차로 넘기는 것: 실패 지점이 워크트리 밖(외부 aidev `bin/run.sh` → `release.json` → `bin/gate.py`)의 **버전 결정 입력**이라 저장소 안에 합법적 수단이 없다. 사람이 증가 정책(첫 릴리즈를 0.0.1 로 볼지 0.1.0 으로 볼지, 태그 형식, 릴리즈 노트 위치)을 한 번 정해 주면 이후는 자동 진행 가능하다. 비평 노트가 남긴 성공 분기 구멍(PopSimpleBot.vue:344-349 / Update:431-436 의 빈 `body.prompt`)과 Update 의 promptDisable 잔여 UX 는 릴리즈가 아닌 다음 개선 회차의 몫이다.
- [러너 21:26] release failed — 릴리즈 안 함: 다음 버전을 정할 근거가 저장소 안에 없다. 이번 회차 재확인: git tag 0개(비-shallow), package.json/package-lock.json version=0.0.0 이며 최초 커밋 ab70
