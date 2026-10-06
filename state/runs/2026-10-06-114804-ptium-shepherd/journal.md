# PR 처리기 노트 2026-10-06-114804-ptium-shepherd — ptium PR #44
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-06-105903-ptium-improve)
# 회차 노트 2026-10-06-105903-ptium-improve — ptium
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:59] base pinned — main@e81b9d9
- [러너 10:59] autonomy release — 

## 정찰 노트
- 세 회차 연속 1순위로 밀려 있던 가이드 과제를 골랐다. 열어 보니 `docs/USER_GUIDE.md:276` 한 칸이 정말 560자였고, 더 나아가 **펜스 코드 블록 경고(prose.go:375)가 가이드 어디에도 없다**(Grep: "코드 블록" 0건) — 단순 재배치가 아니라 빠진 규칙을 채우는 일이 되어 가치가 섰다.
- 코드 후보들을 제친 이유: 인라인 강조(`***중요***`)는 `markdownrule_test.go:102~105` 가 현 동작을 계약으로 박아 기존 테스트를 고쳐야 해 기각, 외곽 파이프 없는 GFM 표는 앞보기 계약이 필요해 M, setext h2 는 case 순서 설계가 선행. 차선으로 둔 `- > 인용`은 좁게 끝나지만 가치가 2다.
- 추측으로 적은 것: 한국어 제목의 GitHub 앵커 문자열(그래서 기호 없는 제목을 권했다), `USER_GUIDE.pdf` 재생성 스크립트 유무(미확인 — 건드리지 말라고 적었다), 제품 안 `/guide` 화면이 같은 표를 갖는지(미확인, 범위 밖).
- 구현자가 조심할 것: `told_test.go:16` 이 `` `.md` ``·`` `.txt` `` 문자열을 가이드 안에서 찾는다 — 표 칸을 줄이면서 그 코드 표시를 지우면 red. 그리고 코드로 확인하지 않은 규칙(`- > 인용`·setext h2·`\|`)을 "됩니다" 로 쓰면 틀린 문서가 된다.
- 프로필은 0일 전 것이라 그대로 두되, 바뀐 두 줄만 갱신해 다시 썼다(VERSION 1.69.57·base e81b9d9, 인용 규칙 landed, 가이드의 코드 블록 공백).
- [러너 11:05] scout done — `USER_GUIDE.md` 의 `.md`·`.txt` 표 한 칸에 쌓인 마크다운 규칙을 별도 절로 빼고, 빠진 규칙(펜스 코드 블록)을 

## 구현 노트
- 바꾼 것: `docs/USER_GUIDE.md` 한 파일(29+/1-). `.md`·`.txt` 표 칸을 한 문장 + 앵커 링크로 줄이고, 규칙을 새 `### 마크다운과 텍스트 파일을 읽는 규칙` 절로 옮기면서 가이드에 아예 없던 펜스 코드 블록 규칙(건너뛰고 "코드 블록 N개(M줄)" 경고)을 채웠다. 코드 0파일. 커밋 cb30ae5.
- 절에 적은 규칙은 소스를 눈으로 읽은 것이 아니라 임시 probe 테스트로 `Read` 를 실제 돌려 돌아온 덱을 보고 적었다(제목·setext `===`·`+`/`•` 글머리·파이프 표·인용·구분선·YAML 머리말·코드 블록 두 개·미닫힘 펜스). probe 는 커밋 전에 삭제했고 `git status` 로 확인했다.
- 확신 없는 곳: **한국어 제목의 GitHub 앵커가 실제로 걸리는지 렌더링으로 확인하지 못했다.** 기호 없는 제목(`마크다운과 텍스트 파일을 읽는 규칙` → `#마크다운과-텍스트-파일을-읽는-규칙`)으로 규칙을 단순하게 만들었지만 GitHub·제품 안 렌더러에서 클릭해 보지는 않았다 — 비평가가 먼저 볼 곳. 또 `` ``` `` 을 두 백틱으로 감싼 인라인 코드 span 두 곳의 렌더 결과도 눈으로 확인하지 못했다(CommonMark 로는 맞다).
- 일부러 하지 않은 것: `USER_GUIDE.pdf` 재생성(릴리즈 경로이고 범위 밖), 제품 안 `/guide` 화면(`web/src`)의 같은 표 사본 확인·수정(미확인, 웹 빌드 범위 — 커밋 메시지에 적었고 ideas.json 에 다음 회차 후보로 올렸다), 코드로 확인하지 못한 규칙(들여쓰기 코드 블록·`\|`·인라인 강조) 기술.
- 다음 역할이 조심할 것: 이 회차의 유일한 자동 가드는 `told_test.go:16` 이고 `` `.md` ``·`` `.txt` `` 코드 표시 문자열이 가이드에 남아 있어야 통과한다 — 되돌림으로 live 임을 증명했다(지우면 `told_test.go:47` FAIL). 새 절 끝 문단에 "아직 읽지 못하는 것" 세 개(setext h2·`- > 인용`·외곽 파이프 없는 표)를 적어 두었으니, 다음 회차가 그중 하나를 구현하면 **그 문장도 함께 지워야 한다**.
- [러너 11:12] brief accepted — 채택 — 재현(`:276` 한 칸의 여섯 규칙·560자, 가이드에 "코드 블록" 없음), 수용 기준 5개, 건드릴 자리 두 곳이 현재 문서�
- [러너 11:12] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인한 것: 새 절의 규칙 일곱 개와 끝 문단의 한계 세 개를 전부 코드로 대조했고 하나도 틀리지 않았다 — atxHeading:480·underlinesHeading:527·listmarker.go:19~47·isRule:657·withoutQuoteMarker:648·isThematicBreak:556·frontMatterLines:607·펜스 320~357/375~379, `.md`·`.markdown`·`.txt` 는 docs.go:62·81. `- > 인용` 은 escapeLine 이 `\>` 를 붙이지만 deck 의 unescapePayload(source.go:586)가 떼므로 "`>` 가 글자로 남는다" 는 표현이 정확하다. `go test -count=1 ./internal/docs` ok 3.012s. 승인.
- 구현자의 '확신 없는 곳' 둘 다 결함 아님: 앵커 `#마크다운과-텍스트-파일을-읽는-규칙` 은 GitHub 규칙과 정확히 일치하고(렌더러 클릭은 못 했다), `` ``` `` 두 백틱 span 은 내부 런이 3이라 CommonMark 로 안전하다.
- 못 본 것: GitHub·제품 안 렌더러에서 실제 앵커 클릭, `USER_GUIDE.pdf` 재생성(도구가 저장소에 없다 — v1.69.36 노트), DB·e2e.
- 남는 우려(릴리즈가 볼 것): v1.69.53~57 릴리즈 노트가 전부 "업로드 형식 표에 이 규칙을 한 줄 적었습니다" 라고 말하는데 그 표에는 이제 규칙이 없다 — 이번 노트는 규칙이 새 절로 옮겨졌다고 적을 것. 가이드에 아직 없는 규칙: 펜스 info string(```js {1,3} 는 펜스가 아니라 요점, prose.go:419).
- 다음 회차: 끝 문단의 한계 세 개 중 하나를 고치면 `docs/USER_GUIDE.md:306~308` 문장도 함께 지울 것. `/guide` 화면(GuidePage.tsx:131)은 한 줄 요약뿐이라 사본 모순 없음을 확인했다.
- [러너 11:17] review approved — 리뷰 승인 (risk=low)
- [러너 11:18] pr created — https://github.com/hkjang/ptium/pull/44
- [러너 11:22] ci failed — 성공이 아닌 검사: verify=failure · 실패한 검사: ? 잡: verify 

## 수리 노트
- 지적(CI verify 실패)은 **맞았지만 이 PR 탓이 아니었다**: diff 는 `docs/USER_GUIDE.md` 한 파일인데 실패 자리는 web 단계의 `npm audit --audit-level=high` 였다. 로컬 `npm ci` 후 같은 명령으로 CI 로그와 같은 4건을 재현했다 — 잠긴 의존성에 새로 붙은 권고이고 main 에서도 같이 떨어진다. 문서 쪽 지적은 하나도 없었으므로 가이드는 손대지 않았다.
- 고친 방법: `npm audit fix`(비파괴)로 postcss→source-map-js 1.2.2, 그리고 tinypool(critical)은 vitest 3 계열에 패치가 없어 `vitest ^3.2.7 → ^5.0.3`(vitest 4·5 는 tinypool 의존을 버렸다). overrides 로 tinypool 2.x 를 vitest 3 에 밀어넣는 더 작은 길은 선언 범위 밖 조합이라 버렸다. 커밋 d174036(web 3파일).
- 딸린 수정이 하나 있다: vitest 4 가 `environmentMatchGlobs` 를 없애 `.test.tsx` 42개가 `document is not defined` 로 깨졌고, 공식 대체인 `test.projects` 두 개(rules=node, components=jsdom)로 옮겼다 — 원래 설정의 node/jsdom 분리 의도와 주석을 그대로 남겼다. 바꾸기 전 271 통과 → 바꾼 뒤 271 통과로 숫자를 맞춰 확인했다.
- 검증은 CI 세 단계를 전부 로컬에서 돌렸다: go test -race+vet ok · npm ci/typecheck/build ok · `npm audit --audit-level=high` **found 0 vulnerabilities**(exit 0) · npm test 48파일 271 · docker build ok. ci.yml 과 단언은 건드리지 않았다.
- 확신 없는 곳: `vitest.config.ts` 는 `tsconfig.node.json` include 밖이라 tsc 가 검사하지 않는다(근거는 실제 실행뿐). `test:watch`·e2e 스윕 미실행. vite 8 의 "esbuild 대신 oxc" 경고는 이번 변경 전부터 나던 것으로 남겨 뒀다. 릴리즈는 vitest 메이저 상승이 노트에 적힐 값인지 판단할 것.

## 심사 노트
- 확인한 것: 새 절의 규칙·한계를 임시 probe(docs.Read → deck.ParseSource, 실행 후 삭제·git status clean)로 end-to-end 로 돌렸다 — 하나만 빼고 전부 맞았다. 수리 커밋도 CI 네 단계를 로컬에서 재현했다: audit 0건·typecheck·build ok, npm test 48파일 271, go vet + go test -race 전부 통과. vitest 노드 API 로 두 project 가 setupFiles 를 상속하는 것을 찍어 설정 주석의 서술을 확인했고, .test.ts 36 + .test.tsx 12 = 48 로 조용히 빠진 테스트 파일이 없음을 확인했다.
- 반려 사유 하나: USER_GUIDE.md:307~308 의 "양쪽 | 없이 그린 표 → 줄마다 요점" 이 프로덕션에서는 틀리다. deck/source.go:410~468 이 파이프 요점 2줄 이상을 걷어 올려 2열은 comparison, 3열 이상은 table 블록으로 만들고 `--- | ---` 는 지운다(probe 출력 확인). 두 파서를 end-to-end 로 맞춰 보라는 운영자 지시가 가리키던 자리다. 그 괄호 절만 지우거나 실제 동작으로 다시 쓰면 끝난다 — 정찰 프로필의 같은 문장도 함께.
- 못 본 것: docker build, e2e 스윕, test:watch, DB 테스트(DSN 없음), GitHub 렌더러에서의 앵커 클릭. USER_GUIDE.pdf 는 재생성 도구가 없어 이번 절을 담지 못한다(기존 드리프트).
- 권고: fix(risk=low) — 문서 한 문장과 프로필 한 줄 외에는 머지 가능한 상태다. 보안·법무 차단 소견 없음(보호 파일·개인정보 무관, 새 의존성은 모두 MIT).
