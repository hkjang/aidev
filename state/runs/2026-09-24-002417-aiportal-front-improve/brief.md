# 과제서 — 2026-09-24-002417-aiportal-front-improve

## A. 우선 과제(릴리즈) — 이번 회차도 **무변경**으로 둔다 (16회째)

정찰이 이 워크트리(HEAD `5d948b7`)에서 직접 재조회한 근거:

- `git tag | wc -l` = **0**
- `package.json:3` `"version": "0.0.0"`, `package-lock.json:3` `"version": "0.0.0"` (lock name 은 `kcb_ai`)
- `CHANGELOG.md` / `VERSION` / `scripts/` / `Makefile` / `.github/` — **전부 없음**(`ls` 로 5건 모두 No such file or directory 확인)
- `docs/` 에는 가이드 11편 + `RELEASE.md`(근거 문서) 뿐. 릴리즈 노트 없음.
- `.gitlab-ci.yml` 전문을 읽음: 12개 job 전부 `rules: if $CI_COMMIT_BRANCH == "main"` 또는 `== "develop"`. **태그 규칙이 단 한 줄도 없다.** 배포는 전용 Runner 가 `/sw/aurda/ai-portal-pvc/node-user-build` 에서 `git fetch/reset --hard` → `npm run build:{ofc,core,int,dev,ofc_dev,dev_dev}` → `cp -rf dist/* <DEPLOY_DIR>` 하는 형태다. 테스트 단계 없음.
- `docs/RELEASE.md` 의 「미확인 항목과 한계」절이 명시: *"버전 파일이 있으므로 외부 릴리즈 절차의 '태그·버전 파일·릴리즈 노트가 모두 없음'에 따른 `skipped` 조건은 충족하지 않는다. 이 문서로 실패를 `skipped`/`released`로 바꾸지 않는다."*

**결론(구현자에게):** 이 저장소 안에서 릴리즈 게이트를 합법적으로 통과시킬 수단이 없다.
과거 릴리즈의 증가 패턴이 0건이라 "이전 패턴을 따라 다음 버전 결정"(절차 2)을 수행할 근거가 없고, `0.0.1`/`0.1.0` 중 무엇을 고르든 **새 관례를 만드는 일**이다. 동시에 버전 파일이 존재하므로 `skipped` 조건도 아니다. 실패 지점은 워크트리 밖(외부 aidev `bin/run.sh` → `release.json` → `bin/gate.py`)이고, 저장소 안에서 통과시키는 유일한 수단은 버전 파일·태그·릴리즈 노트를 새로 만드는 것인데 이는 "워크플로를 느슨하게 만들어 통과시키는 것 금지" 에 걸린다.

**따라서 하지 말 것:** `package.json`/`package-lock.json` version 수정, `git tag`, `CHANGELOG.md`/`VERSION` 신규 생성, 릴리즈 노트 작성, 릴리즈 커밋, 원격 전송, `docs/RELEASE.md` 수정, `package.json` 의 version 필드 삭제(게이트 완화라 금지). 판정을 `skipped`/`released` 로 낮추지 말 것.

**필요한 사람 입력(6건):** ①버전 릴리즈 관례 도입 여부 자체(현 배포는 GitLab CI 브랜치 푸시라 태그가 배포에 불필요) ②시작 버전(0.0.1 대 0.1.0)과 증가 단위 ③태그 형식·주석 태그 여부 ④릴리즈 커밋 메시지 양식 ⑤릴리즈 노트 위치·양식·언어 ⑥GitHub Release 사용 여부.

---

## B. 이번 회차 실행 과제

- **과제: 심플봇 *생성* 팝업의 `onCreate` 실패 경로에 'COM' 가드가 없어 세션 만료 시 전역 Alert 과 토스트가 겹쳐 안내되는 문제 수정 (가치 3 / 위험 1 / 작업량 S)**

- **왜:**
  `src/api/common/interceptors.js` 는 401 재발급 실패·재시도 초과·`BZ01` 업무 오류에서 **먼저 전역 Alert 을 띄우고**(`interceptors.js:164-168, 210-214, 222-226`, 로그인 만료 Alert 은 확인 시 `redirectToLogin()`), 그 뒤 `Promise.reject('COM')` 한다. 저장소는 이 이중 안내를 막기 위해 호출부 `catch` 첫 줄에 `if (e == 'COM') return` 을 두는 관례를 쓰며 `src` 전체에 93곳이 있다. 그런데 `PopSimpleBot.vue` 의 `onCreate`(599-601) 만 이 가드가 없다 — **같은 파일의 나머지 catch 3곳(261, 295, 365)은 모두 가드를 가지고 있고**, 직계 형제 `PopSimpleBotUpdate.vue` 의 `onCreate`(679-681)는 지난 회차에 가드를 받았다. 그 결과 세션이 만료된 상태로 '생성' 을 누르면 사용자는 "로그인이 만료 되었습니다. 재 로그인 해주세요." Alert 위에 "앱생성에 실패하였습니다." 토스트까지 겹쳐 받고, 실제 원인(세션 만료)이 앱 생성 실패로 오인된다.
  고치면 세션 만료 시 안내가 하나(전역 Alert → 로그인 이동)로 수렴하고, 형제 팝업 두 개가 같은 실패 경로를 **같게** 처리하게 된다.

- **수용 기준:**
  1. 세션 만료(인터셉터가 `'COM'` 으로 reject 하는 응답)로 `inf.app.createApp` 이 실패하면, `PopSimpleBot.vue` 의 '생성' 클릭 후 **전역 Alert 은 그대로 뜨고 `toast('앱생성에 실패하였습니다.')` 는 뜨지 않는다.**
  2. 같은 경로에서 팝업은 닫히지 않고 `emit('created', …)` 도 나가지 않는다(현재 동작 유지 — `emit`/`close` 는 이미 성공 경로 안에 있다).
  3. `BZ01` 업무 오류(HTTP 200 아님 + `code == 'BZ01'`)도 인터셉터가 Alert 을 띄우고 `'COM'` 으로 던지므로 동일하게 토스트가 중복되지 않는다.
  4. **'COM' 이 아닌 예외**(네트워크 끊김 등 인터셉터가 그대로 흘리는 오류)에서는 기존처럼 `toast('앱생성에 실패하였습니다.')` 가 **그대로 뜬다**(가드를 넓혀 실패 안내를 통째로 없애면 안 된다).
  5. `isSuccess(res)` false 인 업무 실패(HTTP 200 + 본문 code 불일치 — 인터셉터가 통과시키는 경로)에서는 기존처럼 토스트가 뜨고 팝업이 열린 채 남는다(회귀 없음).
  6. **테스트가 증명해야 하는 것:** 수정 전 1~3 이 **실제로 실패(Red)** 하고 수정 후 통과하며, `if (e == 'COM') return` 한 줄만 되돌리면 같은 케이스가 다시 실패한다(변이 실험). 4~5 는 수정 전후 모두 통과해 범위가 좁음을 고정한다.
  7. 파일 업로드 단계(`inf.app.createAppFile`)가 `'COM'` 으로 실패하는 경우도 같은 catch 로 떨어지므로 1과 같은 결과가 된다 — 케이스로 1건 넣을 것.

- **건드릴 파일:**
  - `src/components/common/popup/Global/PopSimpleBot.vue:599-601` — `onCreate` 의 `catch (e) { toast('앱생성에 실패하였습니다.') }` 를 형제 `PopSimpleBotUpdate.vue:679-682` 와 **문자 그대로 같은 형태**로 바꾼다:
    ```js
    } catch (e) {
    	if (e == 'COM') return
    	toast('앱생성에 실패하였습니다.')
    } finally {
    	isReloading.value = false
    }
    ```
    `finally { isReloading.value = false }` 는 **그대로 둘 것**(가드가 `return` 해도 `finally` 는 돈다 — 스피너/버튼 잠금이 풀린다).
  - `tests/unit/simpleBotCreateComGuard.spec.js` (신규) — 아래 검증 형태.
  - **그 외 파일은 건드리지 말 것.**

- **검증 명령:**
  ```
  npm ci            # node_modules 가 비어 있다(0개) — 반드시 선행. 수 분 걸림
  npm test          # 기준선: 29 파일 497 테스트 (직전 회차 기록). 먼저 돌려 실제 기준선을 확인할 것
  npx vitest run tests/unit/simpleBotCreateComGuard.spec.js   # 신규 스펙 단독
  npm run build:dev # 통과 확인 후 dist/ 삭제할 것
  ```
  `npm run build` · `npm run lint` 는 **존재하지 않는다**(`package.json` scripts 확인).

- **테스트를 어떻게 쓸 것인가 (이 저장소에 정착한 형태):**
  `vitest.config.js` 에 `@vitejs/plugin-vue` 가 연결돼 있어 `.vue` 를 실제로 마운트할 수 있다. 증거는 **HTTP 전송(axios adapter) 한 겹만 대역으로 바꾸고 실제 `interceptors.js` → 실제 `inf.app.*.call` → 실제 컴포넌트**를 통과시키는 형태로 만든다(`tests/unit/simpleBotUpdateClose.spec.js`, `simpleBotPromptReload.spec.js`, `shareAppCommonError.spec.js` 가 그대로 쓸 수 있는 본보기다 — 특히 `simpleBotUpdateClose.spec.js` 는 형제 컴포넌트의 같은 경로를 이미 이렇게 증명해 뒀다).
  - **`'COM'` 을 만드는 검증된 방법**(정찰이 `simpleBotUpdateClose.spec.js:202-204` 에서 확인한, 이 저장소에 이미 도는 형태): adapter 가 **`{ status: 204, data: { code: 'BZ01', message: '로그인이 만료되었습니다.' } }`** 를 돌려주게 한다. `interceptors.js:228-236` 이 `status != 200 && code == 'BZ01'` 을 보고 `openAlert(bizMsg, '')` 뒤 `Promise.reject('COM')` 한다. **`status: 400` 대를 쓰지 말 것** — 그쪽은 axios 가 reject 해 error 핸들러(`interceptors.js:315-322`)로 가고, 거기는 `!error.config.url.includes('/app/simple')` 조건이 붙어 있어 경로에 따라 'COM' 이 아닌 원본 error 가 나온다(`/app/create` 는 조건을 통과하지만 굳이 갈라질 필요가 없다). 401 재발급 경로는 루프가 있어 더 복잡하니 쓰지 말 것.
  - **주의(직전 회차가 실제로 걸린 자리):** `validateCreateApp()` 은 `name`·`description` 뿐 아니라 **LLM 모델·임베딩 모델·프롬프트·첨부 1개 이상**까지 요구한다. `getModel`(text / text-embedding 두 번) 응답을 성공으로 대역해 `selectedType`/`selectedType2` 가 채워지게 하고, 첨부는 `files.value` 에 실제 `File` 이 들어가는 경로로 만들어야 '생성' 클릭이 `onCreate` 본문까지 도달한다. 도달하지 못하면 테스트가 조용히 통과해 버리므로, **수정 전에 Red 가 실제로 나는지 반드시 먼저 확인할 것.**
  - 전역 Alert/토스트는 모듈 싱글턴(`src/utils/globalAlert.js`, 토스트)이라 케이스 간 상태가 샌다 — 각 케이스에서 정리할 것. `restoreMocks: true` 는 켜져 있다.

- **위험과 피할 것:**
  - `src/api/common/interceptors.js`, `src/api/auth.js`, `src/storage/{authStorage,userStorage}`, `src/router`, Chat 스트리밍 — **이번 과제와 무관하니 열지 말 것.**
  - `.gitlab-ci.yml` 은 배포 전용이다. 손대지 말 것.
  - `tryUpdateAppIcon`(`PopSimpleBot.vue:542`) / `tryUpdateLog`(`:563`) 의 `catch (e) {}` 빈 삼킴은 **이번 범위 밖**이다. 기대 계약(아이콘 실패가 전체 실패인가)이 미확인이고, 형제 쪽 현 동작을 `simpleBotUpdateClose.spec.js` 가 이미 고정해 뒀다 — 같이 건드리면 그 스펙과 충돌한다.
  - `PopSimpleBotUpdate.vue` 는 이미 고쳐져 있다. **양쪽을 같게 만드는 것이 목적이므로 Update 쪽은 읽기만 하고 수정하지 말 것.**
  - 교훈: 운영자는 "실제 출력·동작이 바뀌지 않는 수정" 을 반려한다. 이 과제는 토스트 1건의 출현 여부가 바뀌므로 관측 가능하다 — 다만 **그 관측을 테스트로 실제 Red→Green 으로 보여야** 무효 변경이 아니라는 증명이 된다. 소스 문자열 검사(`toContain("if (e == 'COM')")`)를 증거로 삼지 말 것.
  - 커밋 메시지는 이 저장소 관례대로 한국어 `fix:` 로.

- **차선 후보:** `src/components/layout/Sidemenu.vue:499-501` 의 `onDeleted` — `pInf.chat.deleteHistorySession` 호출 뒤 `catch (e) { toast('삭제 중 오류가 발생했습니다.') }` 에 같은 'COM' 가드가 없어 세션 만료 시 같은 이중 안내가 난다(같은 파일 `gotoShare:441` 은 클립보드 복사뿐이라 'COM' 이 나올 수 없으니 **건드리지 말 것**). 1순위가 성립하지 않을 때(예: `validateCreateApp` 을 통과시키는 마운트 경로를 45분 안에 못 만들 때)만 고를 것 — 다만 `Sidemenu.vue` 는 사이드바 전체 상태를 들고 있어 마운트 비용이 더 크다.
