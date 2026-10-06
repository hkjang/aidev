# 회차 노트 2026-10-06-191820-umm-improve — umm
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:18] base pinned — main@84daebc
- [러너 19:18] autonomy release — 

## 정찰 노트
- 왜 이것인가: 지난 회차가 `spaceDisplayName` 으로 공간 이름을 한 값에 모은 직후라, 그 한 값을 읽는 네 내려받기 경로가 서로 다른 파일 이름 기준을 쓰는 것이 남은 조각이다 — 네 곳 모두 `CanvasPage.tsx:2131` 의 같은 `activeName` 을 쓰는 것을 직접 읽어 확인했고, 순수 모듈 + 실제 타입 vitest 로 증명되는 꼴이 최근 release-ready 였던 패턴과 같다. 프로덕션 2파일.
- 차선(`edge-vocabulary.ts`)을 제친 이유: 지난 회차가 "1순위 후보" 로 남겼지만 재평가하니 `migrations/010_memory_graph.sql:55-60` 의 CHECK 제약이 relation/origin 을 각각 6값으로 묶어 `?? relation` 폴백이 도달 불가였다. 사용자가 볼 결함이 없어 가치를 2로 내리고 차선으로 돌렸다(과제서에 그 사실을 정직하게 적었다).
- 추측으로 적은 것: 크롬이 `anchor.download` 의 `/`·`:` 를 `_` 로 바꾼다는 서술은 미확인이라 과제서에 그렇게 표시했고, 수용 기준은 브라우저 동작에 의존하지 않게 "네 base name 이 글자 그대로 같다" 로만 적었다. 서버가 공간 이름에 개행을 허용하는지도 미확인 — 그래서 문자 집합 확장은 범위 밖으로 두고 아이디어로 남겼다.
- 구현자가 조심할 것: 1) Go 쪽 safeFilename 등 네 함수와 통합 금지(운영자 규칙 3번). 2) `check-i18n.mjs` 가 비-테스트 `.ts` 의 `t()` 밖 한글을 거부하므로 새 모듈 본문에 한글 리터럴 금지(주석·테스트는 안전). 3) `spaceDisplayName` 이 이미 `trim()` 하므로 재적용 금지. 4) `npm ci --prefix web` 없이 돌린 exit 1 은 제품 결함이 아니다.
- 프로필: 0일 전(2026-10-05)이라 새로 쓰지 않았다. 헤더 기준만 `main@6aaf940 / 0.76.5` → 실제 `main@84daebc / 0.76.6` 으로 드리프트했고 그 사이 변경은 17e1741(space-name) 하나뿐이어서 본문은 그대로 유효하다.
- [러너 19:23] scout done — 같은 공간을 네 가지로 내려받으면 파일 이름의 기준이 서로 다르다 — 네 곳 중 PDF 만 새니타이즈한다 (�

## 구현 노트
- 무엇을 왜: `CanvasPage.tsx` 의 내려받기 네 곳이 같은 `activeName` 을 쓰면서 PDF 만 새니타이즈했다. 새 순수 모듈 `web/src/lib/download-name.ts` 의 `downloadFileName(name, extension, suffix?)` 하나로 네 곳을 모았다(프로덕션 2파일, 시험 1파일). 커밋 195c446.
- 확신 없는 곳: **런타임에 `anchor.download`/`pdf.save` 가 실제로 새 값을 받는 것은 브라우저로 확인하지 못했다.** 네 호출부는 한 식 → 한 호출의 1:1 치환이고 typecheck·build 가 통과하지만, 그 네 줄을 지나는 시험은 없다(e2e 가 필요하고 과제 범위 밖). 비평가는 `CanvasPage.tsx:1968,1992,2026,2059` 네 줄을 눈으로 먼저 보라. 과제서가 **미확인**으로 남긴 크롬의 `/`·`:` → `_` 치환도 확인하지 않았다 — 수용 기준을 브라우저에 의존하지 않게 적었으므로 결론에 영향은 없다.
- 일부러 안 한 것: ① 문자 집합 확장(개행·제어문자) — PDF 가 이미 맞게 돌던 유일한 곳이라 출력을 바꾸지 않는 것이 수용 기준 2였다. 임시 시험으로 U+0000–U+2FFF 전 코드포인트 + 개행·탭·빈/공백 이름에 대해 새 함수 == 수정 전 PDF 식임을 실제로 돌려 확인하고 그 임시 시험은 지웠다. ② Go 쪽 네 함수와 통합(계약이 다름 — `Content-Disposition` 인코딩 vs 파일 시스템). Go 0파일이라 Go 명령은 돌리지 않았다. ③ 재-trim(`spaceDisplayName` 이 이미 한다). ④ 버전·트레일러.
- 다음 역할이 조심할 것: 검증 전 **`npm ci --prefix web` 가 필수**다(없으면 `vitest is not installed …` 로 exit 1 이고 제품 결함이 아니다). 기준선 21파일/225시험 → 이제 **22파일/230시험**. `npm run lint` 는 기존 파일 경고가 많지만 새 파일 2개를 가리키는 경고는 0건(`grep -c download-name` → 0)이고 EXIT=0 이다. `dist/` 는 ignore 되어 있고 커밋에 들어가지 않았다.
- [러너 19:29] brief accepted — 채택 — 과제서의 근거가 지금 코드와 전부 맞았습니다(네 곳이 같은 `activeName` 을 쓰는 것, PDF 만 `replace` 하는 것, `:1967`·
- [러너 19:29] verify passed — 검증 17개 통과 (auto)

## 비평 노트
- 승인(위험 낮음, 차단 없음). 직접 돌린 것: `npm --prefix web test` 22파일/230시험 EXIT=0, `typecheck` 0, `lint` 0(새 파일 경고 0), `check-i18n` 1060키. 직접 읽은 것: 네 호출부(`CanvasPage.tsx:1968,1992,2026,2059`), `activeName`(`:2132` → `spaceDisplayName` 이 trim·빈이름 대체를 이미 한다 — 주석 주장 사실), `grep` 으로 web/src 에 남은 인라인 내려받기 이름 0건, Go 네 함수 실재, docs·e2e 에 파일 이름 단언 0건.
- 원장의 `실패 재현` 은 코드와 맞는다(세 곳만 날것, PDF 만 일치 — 이번 변경이 고치는 증상과 정확히 같은 출력). 다만 첫 시험의 "네 개가 같다" 는 구조상 항상 참이고(시험이 네 호출을 직접 적는다) 실제로 못 박는 것은 기대 문자열이다. 호출부 우회를 막는 방어는 시험이 아니라 grep 뿐임을 다음 회차가 알아야 한다.
- 구현자의 "네 줄을 지나는 시험이 없다" 는 절반만 맞다 — `web/e2e/export-outline.spec.ts:64`·`export-pdf.spec.ts:63,121` 이 실브라우저에서 세 줄을 지나며 확장자만 단언하므로 이번 변경에 깨지지 않는다. e2e 가 없는 것은 평범한 `.md` 백업 한 줄뿐.
- 남는 우려(차단 아님): `download-name.ts:30` 이 `suffix` 를 새니타이즈하지 않는다(현재 값은 우리 번역값뿐이라 무해하나 "결정이 여기 한 곳" 이라는 주석과 어긋남). 주석이 `safeFilename` 을 `internal/httpapi` 로 귀속했는데 실제 위치는 `internal/store/attachments.go:157`.
- 못 본 것: `run build`·`verify:pwa`, Playwright(실브라우저·격리 DB 필요), 실제 브라우저의 `anchor.download` 새니타이즈 동작(과제서·구현 모두 미확인 표시, 수용 기준이 의존하지 않음), Go 명령(Go 파일 0개). 릴리즈 노트에 쓸 사용자 변화: `.md`·차례·`.png` 이름에서 `\ / : * ? " < > |` 가 `-` 로 바뀜(PDF 는 전과 동일).
- [러너 19:33] review approved — 리뷰 승인 (risk=low)
- [러너 19:33] pr created — https://github.com/hkjang/umm/pull/167
- [러너 19:39] ci failed — 성공이 아닌 검사: verify=failure · 실패한 검사: ? 잡: verify 
