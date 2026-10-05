# 회차 노트 2026-10-06-011806-Momento-improve — Momento
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:18] base pinned — main@0fec64d
- [러너 01:18] autonomy release — 

## 정찰 노트
- CIDR 화면 검증을 골랐다: 서버 문구 한국어화 계열(2956 보존 정책 등)은 네 회차 연속 통과한 안전한 길이지만 같은 패턴 다섯 번째여서, 같은 폼의 다음 단계인 '왕복 없애기' 를 집었다 — 프로덕션 3파일(+테스트 1)로 끝나고 PASSWORD_RULE 선례가 폼·Alert 문장 공유 방식까지 이미 정해 준다.
- 더 값있어 보이는 후보(createNetwork 가 Postgres `cidr` 의 거절을 400 대신 500 으로 흘린다 — adminErrors.ts:292-295 가 적어 둔 틈)는 제쳤다: 이 환경에 Postgres 가 있는지 확인하지 못했고 DSN 없으면 통합 테스트가 조용히 skip 되어 end-to-end 증명이 안 된다. ideas.json 에 신규 pending 으로 남겼다.
- 추측으로 적은 것: Postgres `cidr` 컬럼이 호스트 비트 켜진 값을 거절하는지(미확인 — 그래서 그 값은 **차단하지 말고** 비오류 안내만 하게 했다), 테스트 기준선 211건(직전 회차 기록, 이 환경은 web/node_modules 가 없어 재실행 안 함).
- 구현자가 조심할 것: 화면이 `net.ParseCIDR` 보다 좁아지면 안 된다(IPv6 본문·호스트 비트는 통과), `adminErrors.test.mjs:619` 의 문장 단언을 글자 하나도 고치지 말 것, 설정 화면의 「신뢰할 Reverse Proxy CIDR」(AdminPage.tsx:2376, 다른 엔드포인트·다른 서버 검사 admin.go:720-731)을 함께 묶지 말 것.
- 프로필은 2026-10-01 판(5일 전)을 그대로 둔다 — 이번에 읽은 범위(web/src/pages 순수 모듈 목록, web/test 38개 파일, node_modules 없음, admin.go·adminErrors.ts 구조)가 전부 일치했다.
- [러너 01:23] scout done — 「망 대역 추가」 폼이 CIDR 표기를 보내기 전에 화면에서 거른다 — 순수 모듈 `cidrRule.ts` + helperText, 그리�

## 구현 노트
- 「망 대역 추가」 폼의 CIDR 칸이 보내기 전에 거르게 했다(`cidrRule.ts` 신규 + `adminErrors.ts` 가 `CIDR_RULE` 공유 + `AdminPage.tsx` 의 error/helperText/disabled). 왕복 한 번과 영문 왕복 안내를 없애고 안내 문장의 정본을 하나로 모은 것이 전부다. 서버(`internal/`)는 열지 않았다.
- **확신 없는 곳·검증 못 한 것**: ① Postgres `cidr` 컬럼이 호스트 비트 켜진 `10.20.30.1/24` 를 받는지 **DB 로 재현하지 못했다**(이 환경에 Postgres 없음). 그래서 그 값을 차단하지 않고 안내도 아무 주장을 하지 않는다 — 만약 DB 가 거절한다면 사용자는 여전히 500 을 보게 된다(그 문구는 중립이라 거짓은 아니다). ② `judgeCIDR` 은 매 글자 호출된다 — 한 줄 입력이고 정규식 둘뿐이라 쟀을 필요를 못 느꼈으나 프로파일링은 안 했다. ③ IPv6 본문·IPv4 선행 0 은 **일부러** 통과시켜 왕복이 남는다(코퍼스 888개 중 36건) — 축소가 아니라 '화면이 서버보다 좁아지지 않는다' 를 지킨 결과다.
- **일부러 하지 않은 것**: 설정 화면의 「신뢰할 Reverse Proxy CIDR」(AdminPage.tsx:2376)에 `judgeCIDR` 을 쓰지 않았다 — 다른 엔드포인트(settings PUT)·다른 서버 검사(admin.go:720-731, 쉼표로 여러 개)라 같은 계약이 아니다. 과제서가 묶지 말라고 한 자리이기도 하다. 입력 trim 도 하지 않았다(공백 포함 값은 서버도 거절하므로 막는 쪽이 서버와 일치).
- **다음 역할이 조심할 것**: `adminErrors.test.mjs` 는 **한 글자도 고치지 않았다** — `CIDR_RULE` 문구를 "개선" 하면 그 619행 단언과 `cidrRule.test.mjs` 의 마지막 단언이 함께 깨진다. 그게 문구가 안 바뀌었다는 증거이니 의도된 것이다. 브라우저 하네스는 `/tmp/cidrharness` 에만 있고 커밋에 없다(`web/dist` 도 지웠다, 4파일 확인); 다시 돌리려면 `npm run build` 로 dist 를 만들고 `/api/v1/me` 의 role 이 `organization_admin` 이어야 한다 — networks 섹션이 `orgOnly` 라 `owner` 로는 폼이 아예 렌더되지 않는다. `go test` 는 불필요(Go 미변경).
- [러너 01:36] brief accepted — 채택 — 인용한 행 번호(AdminPage.tsx:3235-3239 CIDR TextField·3250 `disabled`·3192 `NetworksAdmin`·76 `./passwordRule`·3563 `helperText={PASSWORD_R
- [러너 01:37] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 **approve** (risk low, blocking 없음). 핵심 계약을 손으로 재현했다: PARSE_CIDR 표 28줄을 go1.26.7 로 실측해 전부 일치, 주소30×마스크32+무슬래시30 = **1023개 차분 퍼즈에서 narrowing 0건**(화면이 막는데 서버가 받는 값 없음, 왕복 남는 값 123건). `judgeCIDR = () => ({})` 스텁으로는 12건 중 8건이 실패해 테스트가 실제로 새 경로를 지난다 — ledger-entry.md:5 기록과 테스트 이름까지 일치.
- 게이트: `tsc -b` rc=0 · `eslint` rc=0 · `node --test` **223/223**(211+12) · `vite build` 성공. 확인 뒤 web/dist 지우고 `git status` 빈 상태 복원. adminErrors.ts:291 템플릿이 기존 문장과 바이트 동일해 adminErrors.test.mjs:619 전문 단언이 무수정 통과. 권한·세션·마이그레이션·워크플로 무변경, revert 는 단일 커밋.
- **남는 우려(차단 아님)**: 구현자 미확인 ① 은 저장소가 이미 답한다 — `cidr cidr NOT NULL`(001_initial.sql:67) + adminErrors.test.mjs:594-595 의 `invalid cidr value: "10.0.0.5/24" (22P02)` 픽스처. Postgres 는 호스트 비트를 **거절한다**. 따라서 `10.20.30.1/24` 는 hint 만 뜨고 버튼이 열려 여전히 500(「잠시 후 다시 시도」 — 원인이 입력인데 일시 장애처럼 읽힌다)으로 간다. **PR 전과 동일 동작 + 고칠 거리 추가**라 회귀가 아니므로 통과시켰다.
- 다음 회차/릴리즈 노트: ideas.json 의 「createNetwork 가 Postgres cidr 거절을 400 대신 500 으로 흘린다」를 '미확인' → **'확인됨(픽스처가 증거, DB 없이도 착수 가능)'** 으로 올려 잡을 것. cidrRule.ts:~105 와 테스트 주석의 「확인하지 못했다」 문구도 그때 사실에 맞게 고치면 된다.
- 못 본 것: 실제 Postgres end-to-end(DSN 없음 — 통합 테스트 조용히 skip)와 브라우저 DOM 재확인(구현자의 `/tmp/cidrharness` 기록만 읽음). 변경이 순수 모듈 + TextField prop 3개로 닫혀 있어 판정을 바꿀 자리는 아니다. 참고: 이 워크트리에는 **web/node_modules 가 이미 있다**(프로필의 '없음' 과 다름) — 다음 회차는 npm ci 수 분을 아낄 수 있다.
- [러너 01:42] review approved — 리뷰 승인 (risk=low)
- [러너 01:42] pr created — https://github.com/hkjang/Momento/pull/26
- [러너 01:48] ci passed — 검사 1개 모두 success
- [러너 01:48] merge done — 842d827
- [러너 02:02] release published — v0.34.58
- [러너 02:04] assets verified — v0.34.58 자산 2개 (이전 v0.34.57: 2)
