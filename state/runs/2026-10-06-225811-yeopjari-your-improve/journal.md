# 회차 노트 2026-10-06-225811-yeopjari-your-improve — yeopjari-your
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 22:58] base pinned — main@f4a924c
- [러너 22:58] autonomy release — 

## 정찰 노트
- 고른 이유: 브라우저 오류 중복 묶음 서명 결함은 **읽어서 확인한** 결함이다(client-errors.ts 가 new Error → pipeline.ts:164 code='internal' → 서명 한 개, 0002_security.sql:458 이 창 안에서 send=false). 테스트 공백 메우기(차선)보다 가치가 높고, 단위 테스트가 두 개뿐인 이 저장소에서 순수 함수 하나를 꺼내면 테스트도 같이 생긴다. 죽은 코드(isSafeLink)·v.instant 는 호출부가 없거나 미확인이라 제쳤다.
- 추측으로 적은 것: 지문 방식(FNV-1a 8자 hex)은 내 권고이고 코드에 없다. launch.ts 의 published_at NULL 순번 가설(ideas 6번)은 제약을 확인하지 않았으니 이번 과제로 쓰지 말 것.
- 구현자가 조심할 것: (1) 추가 인자 기본값이 '' 일 때 서명이 **바이트 단위로** 지금과 같아야 한다 — 다르면 운영 DB 에 억제돼 있던 오류가 전부 메일로 쏟아진다. (2) 지문에 시각·requestId·랜덤을 섞으면 반대 방향(오류 1건=메일 1통)으로 깨진다. (3) scripts/e2e.mjs:800 이 `signature LIKE 'CLIENT /chat/:id%'` 로 문자열 모양에 의존하므로 조각은 **끝에** 붙인다.
- 배선 증명은 단위 테스트가 못 한다. e2e "브라우저 오류 수집" 절(798~805행)에 같은 경로·다른 메시지 두 건 → DISTINCT signature 2 를 한 줄 더하는 것을 과제서에 선택 항목으로 넣었다. 가능하면 꼭 할 것.
- 프로필이 없어 profile.md 를 새로 썼다(CLAUDE.md 는 이 저장소에 없다).
- [러너 23:04] scout done — 브라우저 오류 알림이 한 경로에서 서로 다른 오류를 삼킨다 — 중복 묶음 서명에 오류 종류·내용을 넣는

## 구현 노트
- 무엇·왜: `alertSignature(method, path, code, extra='')` 를 `pipeline.ts` 에서 export(서명 조립의 유일한 지점), `alertOperator` 에 기본값 `''` 인 마지막 인자, `client-errors.ts` 는 스크럽된 메시지+kind 로 FNV-1a 8자 hex 지문(`clientErrorFingerprint`)을 그 인자로 넘긴다. 폴딩 경로는 `folded` 변수 하나로 모아 DB `path` 칼럼·메일이 서명과 어긋날 수 없게 했다. 프로덕션 파일 2개 + 테스트 2개.
- 확신 없는 곳·검증 못 한 것: (1) **서버 자신의 실패 경로는 e2e 로 검증하지 못했다** — 완주 후 `error_alerts` 에 CLIENT 행 4개만 있고 서버 500 행이 하나도 안 생긴다. 수용 기준 3)(서명 바이트 동일)은 단위 테스트 + "pipeline.ts:122 가 extra 를 안 넘김 → `extra ? ... : signature` 가 옛 문자열 그대로"라는 코드 논증으로만 지킨다. 비평가는 여기부터 볼 것. (2) 160/0 통과한 e2e 실행은 `scripts/e2e.mjs` 의 가짜 서버 listen 포트만 18998→18871·18997→18872 로 임시 바꾼 상태였다(다른 저장소 세션이 그 포트를 쥐고 있었다). 커밋 전에 원래 값으로 되돌렸고 diff 에 포트 변경은 없다 — 포트는 테스트 픽스처 주소일 뿐 바꾼 코드와 무관하지만, 되돌린 그 상태로의 완주는 못 봤다.
- 일부러 하지 않은 것: `.env.dev` 를 만들지 않고(세션 규칙) 환경변수를 인라인으로 줘 서버를 띄웠다. `scripts/e2e.mjs` 의 하드코딩 포트를 환경변수화하지 않았다(범위 밖 — ideas.json 에 적음). `IGNORED` 필터와 `logFailure`(서명을 만들지 않는 다른 계약)는 손대지 않았다.
- 다음 역할이 조심할 것: 새 e2e 검사는 **DB 를 리셋한 뒤** 돌려야 통과한다(`occurrences === 2` 를 세므로 같은 DB 에 두 번 돌리면 4가 되어 실패한다 — 실제로 한 번 겪었다). `bash scripts/dev-reset.sh` 또는 CI 처럼 새 DB 로 시작할 것. 단위 테스트는 DB 가 필요 없다.
- [러너 23:13] brief accepted — 채택 — 근거(code 가 늘 'internal', 서명 PK upsert 가 창 안에서 send=false)가 지금 코드와 정확히 맞고 e2e 로 재현됐다.
- [러너 23:13] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인함: `alertSignature` 의 `extra=''` 경로가 옛 서명과 바이트 동일(pipeline.ts:151)이고 서버 유일 호출부 pipeline.ts:122 는 extra 를 넘기지 않음 — 구현자가 "검증 못 했다"고 남긴 수용 기준 3)은 코드로 성립. `folded` 가 path 칼럼·메일에만 쓰이고 지문이 섞이지 않음도 확인. 원장의 실패 재현(`occurrences: 4` 한 행)이 증상과 일치하고, 새 e2e 검사는 수정 전 코드에서 통과할 수 없다. `npm run check` 39건 통과.
- 못 본 것: e2e 전체(Docker PostgreSQL 16)는 이 세션에서 돌리지 않았다. 서버 자신의 500 경로가 실제 DB 행으로 옛 서명을 유지하는지도 실행으로는 확인 못 했다(코드 논증만).
- 승인이어도 남는 우려(릴리즈 노트에 넣을 것): 배포 직후 1회성으로 운영 DB 의 기존 `CLIENT <경로> internal` 행이 고아가 되고 창에 억제돼 있던 브라우저 오류가 전부 한 번씩 메일로 나간다(고아 행은 internal.ts:253 의 30일 정리로 소멸, 서버측 서명은 무영향).
- 다음 회차가 알 것: CLIENT 행이 (경로, kind, 메시지)당 1개로 늘어나 운영 메일량이 오른다 — `ops.error_alert_window_minutes` 가 충분한지 볼 것. FNV 32비트는 ~7.7만 메시지에서 충돌 가능(그룹화 용도로 수용). 새 e2e 검사는 `occurrences === 2` 를 세므로 새 DB 필수.
- security·legal 차단 사유 없음: 신규 수집·전송·인가 변경·비밀값·보안용 손제작 암호 없고, 지문은 scrub() 된 텍스트(이미 last_message 에 저장되던 것)에서만 만들어진다.
- [러너 23:16] review approved — 리뷰 승인 (risk=low)
- [러너 23:17] pr created — https://github.com/hkjang/yeopjari-your/pull/1
- [러너 23:22] ci passed — 검사 1개 모두 success
- [러너 23:22] merge done — 50005e3
- [러너 23:28] release published — 0.1.1
