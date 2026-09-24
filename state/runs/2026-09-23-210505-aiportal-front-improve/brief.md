# 과제서 (2026-09-23-210505-aiportal-front-improve)

- **과제**: 추천 프롬프트 생성이 실패해도 로딩 애니메이션 문구가 역할 프롬프트 입력란에 남아 그대로 저장되는 문제 수정 (가치 4 / 위험 2 / 작업량 M)

## 왜
`PopSimpleBot.vue:334 reload()` 와 `PopSimpleBotUpdate.vue:421 reload()` 는 `rolePrompt` 를 비우고 `startLoopTyping(rolePrompt, '추천 프롬프트를 가져오는중입니다.')` 로 그 입력란에 직접 타이핑 애니메이션을 그린 뒤 `inf.app.promptCreate.call()` 을 기다리는데, `if (isSuccess(res))` 에 **else 분기가 없다** — 업무 실패 응답(HTTP 200 + body `code != 200`)이나 성공했지만 `body.prompt` 가 빈 문자열인 응답에서는 `finally` 의 `typer.stop()` 이 애니메이션을 그 자리에 **얼려버린다**. 사용자에게는 아무 안내도 없이 "추천 프롬프트를 가져오는중입니" 같은 잘린 안내 문장이 자기가 쓴 역할 프롬프트인 양 남고, 그대로 저장을 누르면 `PopSimpleBot.vue:505` 의 `payload.prompt = String(rolePrompt.value||'').trim()` 가 그 문장을 앱의 실제 역할 프롬프트로 서버에 보낸다. 예외 경로(`catch`, 358-362행)는 이미 `rolePrompt.value = ""` 로 되돌리고 있으므로, 실패 처리의 **일관성이 한쪽 경로에서만 빠져 있는 것**이 결함이다.

`isSuccess` 가 false 인 상태가 실제로 여기까지 도달한다는 근거: `src/api/common/interceptors.js:227-236` 은 `status != 200 && code == 'BZ01'` 일 때만 `Promise.reject('COM')` 하고, 그 밖의 응답(HTTP 200 + body `code`/`success` 가 실패)은 **그대로 `return response`** 한다. `src/utils/common.js:55-64 isSuccess` 는 그런 응답에 false 를 돌려준다.

## 수용 기준
1. 업무 실패 응답(`{ data: { code: '500', message: '...' } }` 처럼 `isSuccess` 가 false)을 받으면, 팝업의 역할 프롬프트 입력란(`#rolePrompt` textarea)이 **빈 문자열**이 되고 "가져오는중" 문구가 남지 않는다.
2. 같은 실패 상황에서 사용자에게 실패가 **안내된다** (`toast(...)` 또는 `openAlert(...)` 중 하나 — 이 파일들이 이미 쓰는 `toast` 를 권장. `validatePromptCreate` 가 실패 안내에 `toast` 를 쓰고 있으므로 같은 수단으로 맞출 것).
3. `isSuccess` 는 true 인데 `body.prompt` 가 없거나 공백뿐인 응답에서도 입력란이 빈 문자열이 되고 안내가 뜬다(현재는 349행의 `if (typeof nextPrompt === 'string' && nextPrompt.trim())` 가 걸러 버려 애니메이션 문구가 그대로 남는다).
4. 성공 경로는 그대로다 — `body.prompt` 가 있으면 입력란이 그 값이 되고, `body.temperature` 반영(350-355행)·`selectedPromptId.value=''`·`promptDisable.value=false` 가 지금과 동일하게 동작하며 실패 안내가 뜨지 않는다.
5. `catch` 경로도 그대로다 — `'COM'` 이 던져지면 입력란이 비워지고 **추가 안내를 띄우지 않는다**(공통 인터셉터가 이미 전역 Alert 을 띄웠으므로 중복 안내가 되면 안 된다). 이 저장소의 `catch (e) { if (e == 'COM') return; ... }` 관례를 깨지 말 것.
6. 두 파일 **모두** 고친다. 같은 값을 쓰는 경로가 둘이므로 한쪽만 고치면 안 된다(운영자 지시). 테스트도 두 컴포넌트를 각각 통과시킨다.
7. 테스트가 증명해야 하는 것: 수정 **전에는** 기준 1·2·3 이 실패(Red)하고, 수정만 되돌리면 같은 케이스가 다시 실패한다. 무딘 테스트가 아님을 되돌림 실험으로 확인해 회차 노트에 적을 것.

## 건드릴 파일
- `src/components/common/popup/Global/PopSimpleBot.vue:334-369` — `reload()`. `if (isSuccess(res)) { ... }` 에 `else` 를 추가(입력란 초기화 + 실패 안내). `nextPrompt` 가 비었을 때도 같은 실패 처리로 수렴시킬 것. `finally` 의 `typer.stop()` 과 `setTimeout(1200)` 은 건드리지 말 것.
- `src/components/common/popup/Global/PopSimpleBotUpdate.vue:421-456` — 위와 **동일한** 구조. 같은 형태로 고칠 것(두 파일의 `reload()` 는 현재 문자 그대로 같다).
- `tests/unit/promptCreateFailure.spec.js` (신규) — 아래 검증 명령으로 도는 스펙.

### 구현 힌트 (권장 형태, 강제 아님)
```js
const nextPrompt = body?.prompt
if (isSuccess(res) && typeof nextPrompt === 'string' && nextPrompt.trim()) {
    rolePrompt.value = nextPrompt
    // ... 기존 temperature / selectedPromptId / promptDisable 처리 그대로
} else {
    typer.stop()                 // finally 보다 먼저 멈춰 잔여 타이핑을 막는다
    rolePrompt.value = ''
    toast('추천 프롬프트를 가져오지 못했습니다. 잠시 후 다시 시도해 주세요.')
}
```
`typer.stop()` 을 else 안에서 한 번 더 부르는 것은 안전하다 — `stop()` 은 `clearInterval` 뿐이고 `finally` 에서 중복 호출돼도 무해하다(`catch` 경로가 이미 같은 이중 호출을 하고 있다).

## 검증 명령
```
npm ci                 # node_modules 가 비어 있으면 선행 필수
npx vitest run tests/unit/promptCreateFailure.spec.js
npm test               # 기준선 27파일 477테스트 — 줄어들면 안 된다
npm run build:dev && rm -rf dist
```
`npm run build` · `npm run lint` 는 이 저장소에 **없다**. 빌드 후 `dist/` 는 반드시 지울 것.

### 테스트 작성 메모 (실제로 확인한 것)
- `vitest.config.js` 에 `@vitejs/plugin-vue` 가 연결돼 있어 `.vue` 를 **실제로 마운트**할 수 있다. 손으로 만든 대역 컴포넌트로 증명하지 말 것(운영자 지시).
- `PopSimpleBot.vue` props: `{ modelValue: Boolean, tooltip: String }`. `modelValue: true` 로 마운트한다.
- `validatePromptCreate()`(163-183행)가 먼저 통과해야 `reload()` 가 진행된다. 통과 조건: `userId` 가 있어야 하고(→ `@/storage/userStorage.js` 의 `readUser` 를 `{ user_id: 'u1' }` 로 모킹), `name` 이 비어 있지 않고 특수문자가 없어야 한다. 이름 입력은 `#rolePrompt` 와 같은 방식으로 DOM 에서 채우거나, 어려우면 `validatePromptCreate` 가 참조하는 입력 컴포넌트를 실제 DOM 으로 채울 것.
- 애니메이션은 `startLoopTyping` 의 기본 간격 **150ms** 다. 응답이 즉시 resolve 되면 한 글자도 타이핑되지 않아 결함이 드러나지 않는다 — `vi.useFakeTimers()` 로 응답 대기 중 `await vi.advanceTimersByTimeAsync(400)` 하거나, `promptCreate.call` 을 `setTimeout(resolve, 200)` 으로 지연 resolve 시켜 **최소 한 번 이상 타이핑이 일어난 뒤** 응답이 오게 만들 것. 이걸 안 하면 Red 가 안 나온다.
- `inf` 는 `@/api/interface.js` 에서 온다. 이 케이스는 인터셉터를 통과시킬 필요가 없으므로 `vi.mock('@/api/interface.js', ...)` 로 `inf.app.promptCreate.call` 만 대역화해도 된다 — 단 `isSuccess` 는 **실제 `src/utils/common.js`** 를 쓸 것(응답 판정이 이 결함의 핵심이다).
- `toast` 를 안내 수단으로 쓴다면 `@/utils/common.js` 를 부분 모킹(`importActual` 로 `isSuccess`·`startLoopTyping` 은 실물 유지)하거나, 실제 toast 가 DOM 에 남기는 흔적을 단정할 것.
- 전역 `globalAlert`/`globalLoading` 은 모듈 싱글턴이라 케이스 간 상태가 샌다. 각 케이스에서 정리할 것. `restoreMocks: true` 는 켜져 있다.

## 위험과 피할 것
- **보호 경로를 건드리지 말 것**: `src/api/common/interceptors.js`, `src/api/auth.js`, `src/storage/{authStorage,userStorage}`, `src/router`. 이번 수정은 이 중 어느 것도 필요하지 않다.
- `startLoopTyping`(`src/utils/common.js:609`) 자체를 고치지 말 것. 다른 결함(마지막 글자가 즉시 지워짐, `txt` 미지정 시 throw)이 있지만 이번 범위 밖이고, 공용 유틸을 건드리면 두 팝업 밖으로 영향이 번진다.
- `catch` 의 `if (e == 'COM') return` 앞에 안내를 추가하지 말 것 — 공통 인터셉터가 이미 전역 Alert 을 띄운다. 2026-09-23 회차(앱 공유 등록)에서 정확히 이 중복 안내가 결함이었다.
- `finally` 의 `setTimeout(..., 1200)` 으로 `isReloading`/`isActive` 를 푸는 구조를 바꾸지 말 것. 바꾸면 버튼 중복 클릭 방지 동작이 같이 흔들린다.
- 두 파일의 `reload()` 를 공용 함수로 **추출하지 말 것**. 대규모 리팩터는 금지 범위이고, 이번 회차 가치는 동작 수정에 있다.
- 운영자 지시: 관측 가능한 동작이 바뀌지 않는 수정은 넣지 말 것. 수용 기준 7 의 되돌림 실험으로 인과를 반드시 증명할 것.

## 차선 후보
**법률 문서 뷰어의 `source_seq` 엄격 파싱** — `src/views/Chat/ChatViewer/ChatData.vue:708` 이 `JSON.parse(props.doc.source_seq)` 를 `try` 블록(723행) **밖에서** 호출한다. `source_seq` 는 `src/utils/markdown.js:143-152 parseLawArgs` 가 `onclick` 의 `law('doc', [...])` 에서 브래킷 텍스트를 **문자열 그대로** 잘라낸 값이라, 작은따옴표 배열(`['1','2']`)처럼 JSON 이 아닌 형태면 SyntaxError 가 나고 `loadPage`(770행)의 catch 가 "데이터를 불러오는 중 오류가 발생했습니다." 만 띄운다. 같은 저장소의 `abc` 경로(`ChatStorageDetail.vue:591`, `Chat/Index.vue:1192`)는 같은 값을 **관대하게** 정규화(브래킷 제거 → 쉼표 분리 → Number)하고 있어 두 경로의 읽는 방식이 어긋나 있다.
**주의(미확인)**: 백엔드/LLM 이 실제로 작은따옴표 배열을 내보내는지는 이 저장소 안에서 확인하지 못했다 — 트리거 입력이 실증되지 않으면 관측 가능한 동작 변화가 없는 수정이 될 수 있다. 1순위를 먼저 시도하고, 이 후보를 고를 경우 먼저 실패 입력을 재현하는 테스트부터 세울 것.
