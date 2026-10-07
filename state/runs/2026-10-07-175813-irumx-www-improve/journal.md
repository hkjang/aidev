# 회차 노트 2026-10-07-175813-irumx-www-improve — irumx-www
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:58] base pinned — main@80e35fc
- [러너 17:58] autonomy release — 

## 정찰 노트
- 미선언 전이 의존성을 골랐다: `scripts/verify-build.mjs:18-19` 의 `harfbuzzjs`·`fontverter` 가 package.json 에 없고 `subset-font` 호이스팅으로만 산다 — lock 파일에서 직접 확인한 **확정 결함**이고, 파일 2개(+신규 1개)로 끝나며 검증이 `npm run build` 하나다.
- 제친 후보: 서비스 페이지 목록 2중 하드코딩(차선 — 목록을 **합치면 안 된다**, 운영자 교훈대로 감지만), handleInbound 첨부 누적 용량 버그(20MB 첨부 테스트 비용 + 운영 중 메일 경로), CI 추가(.github 보호 경로).
- 추측으로 적은 것: `npm run build` 가 `PUBLIC_TURNSTILE_SITE_KEY` 없이 통과하는지 **미확인**(실제 빌드를 돌리지 않았다) — 과제서에 테스트 키 대안을 적어 뒀다. 새 `verify-deps.mjs` 가 `.astro` 파일을 건너뛰어도 되는지도 미확인이라 범위에서 뺐다.
- 구현자가 조심할 것: `npm install harfbuzzjs fontverter` 로 최신 버전을 끌어오지 말 것 — lock 의 1.6.2 / 2.0.0 을 그대로 캐럿으로 적고 `--package-lock-only` 로 lock 만 갱신. `worker/`·`src/lib/inquiry.ts` 는 이번 과제에서 손댈 이유가 없다.
- [러너 18:03] scout done — 빌드 스크립트가 쓰는 미선언 전이 의존성(harfbuzzjs·fontverter) 명시 + 미선언 import 검사 추가 (가치 4 / 위험 

## 구현 노트
- `scripts/verify-build.mjs:18-19` 가 import 하던 `harfbuzzjs`·`fontverter` 를 lock 그대로(`^1.6.2`·`^2.0.0`) devDependencies 에 선언하고, 새 `scripts/verify-deps.mjs`(bare import 대조)를 `npm run build` 첫 단계로 끼웠다. 프로덕션 파일 2개 + 신규 1개.
- **과제서와 다르게 한 곳(의도적)**: `npm install --package-lock-only` 를 쓰지 않았다. 이 환경의 npm 10.9.8 로 돌리면 optional 패키지의 `libc:[glibc|musl]` 필드 **148줄이 함께 삭제**된다(의도한 2줄 추가 + 무관한 150줄 삭제). libc 는 musl/glibc 환경의 optional 의존성 선택에 쓰여 지우면 안 된다고 판단해, lock 을 원복하고 root `devDependencies` 두 줄만 직접 넣은 뒤 **`npm ci` 로 정합성을 확인**했다(exit 0). 최종 lock diff 는 `+2줄`뿐이다. 비평가는 여기를 먼저 보면 된다.
- 검증(모두 실제 실행): ① 고치기 전 `node scripts/verify-deps.mjs` → harfbuzzjs·fontverter 지목, exit 1 ② 선언 후 → `✓ 선언 확인 23개 파일`, exit 0 ③ 선언을 다시 지우고 **`npm run build`** → 1단계에서 멈춤(exit 1, subset-font·astro 미실행) = 빌드 배선 end-to-end 확인 ④ `npm ci` 재설치 후 `npm run build` 전체 통과, `verify-build` 도 `✓ 모두 통과`.
- 해소된 미확인: `npm run build` 는 `PUBLIC_TURNSTILE_SITE_KEY` **없이도 통과**한다(테스트 키 불필요). 설치 버전은 1.6.2·2.0.0 그대로이고 `npm ls` 로 직접 의존성이 된 것까지 확인했다.
- 확신 없는 곳: 파서가 정규식 기반이라 (a) 문자열·템플릿 리터럴 안의 `import … from '…'` 같은 꼴, (b) 들여쓰기 없이 줄머리에 오는 주석 아닌 특이 구문은 오인할 수 있다. 현 저장소 23개 파일에서는 bare 12개·상대 42개를 정확히 갈라냈고 다중 행 import(`src/scripts/contact.ts:9`)도 맞게 읽는 것을 확인했다. 동적 `import()`·`require()` 는 저장소에 없어 **처리하지 않는다**(있으면 놓친다 — 거짓 실패가 아니라 누락 방향).
- 일부러 하지 않은 것: `tests/`·`astro.config.mjs`·`playwright.config.ts`·`*.astro` 는 범위에서 뺐다(앞 3개는 넓혀도 통과하는 것을 확인했으나 과제서 범위를 지켰고, `.astro` 는 거짓 실패 위험). `verify-build.mjs` 기존 검사 로직과 `worker/`·`src/lib/inquiry.ts` 는 건드리지 않았다.
- 다음 역할이 조심할 것: 이 저장소는 `node_modules` 가 없는 상태로 체크아웃되므로 검증 전 `npm ci` 가 필요하다(빌드 ~40초). **lock 을 다시 갱신할 일이 있으면 `npm install --package-lock-only` 의 diff 를 반드시 확인할 것** — 위 libc 삭제가 조용히 섞여 들어온다. `npm test` 는 `npm run test:build` 로 dist-draft·dist-email 를 먼저 만들어야 돌고 이번에 실행하지 않았다.
- [러너 18:09] brief accepted — 채택 — 근거(미선언 import, lock 의 1.6.2·2.0.0, 호이스팅 의존)가 코드·lock 과 정확히 일치했고 수용 기준 3개를 모두 충족�
- [러너 18:09] verify failed — 실패한 검증: npm test --silent (exit 1)
