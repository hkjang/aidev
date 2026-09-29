# 회차 노트 2026-09-29-191711-jupiq-improve — jupiq
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:17] base pinned — main@2db2da2
- [러너 19:17] autonomy release — 

## 구현 노트
- 배정된 우선 과제는 이미 해결된 상태였다: PR #29의 'React 검사' 실패를 cdb9d16 + 깨끗한 npm ci로 재현(TS2307 node:http)했고, 고친 커밋 7791175이 base에 있다. 릴리즈 워크플로 단계를 로컬에서 전부 돌려 통과시켰으므로 '두 번 실패'는 재현되지 않았다 — 워크플로 파일은 한 줄도 손대지 않았다.
- 대신 보류 아이디어를 구현했다: client.ts parseResponse가 깨진/빈 JSON 본문에서 SyntaxError를 흘려 ApiError의 status·code를 잃던 것을 막았다(프로덕션 파일 1개).
- 확신 없는 곳: (1) 성공 응답의 깨진 본문을 ApiError로 **던지는** 것이 이 앱의 모든 화면에서 최선인지는 판단이다 — 예전엔 SyntaxError였으니 회귀는 없지만, ok+빈 JSON 본문을 null로 돌려주는 선택지도 있었다. (2) 200 + 빈 JSON 본문이 서버에서 나오는 경로는 helpers.go:29 writeJSON이 Encode 오류를 버리는 것을 소스에서 읽어 추론했고, NaN/Inf를 실제로 흘려 재현하지는 않았다(그 서버 쪽 수정은 ideas.json에 보류로 남겼다). (3) 프록시가 자른 응답은 테스트에서 끊긴 본문으로 모사했을 뿐 실제 프록시로는 검증하지 않았다.
- 일부러 하지 않은 것: 서버 writeJSON 수정(같은 회차에 두 경로를 함께 바꾸지 않기 위해), 화면 코드의 오류 표시 변경, tsconfig 분리(@types/node 전역 오염 — 빌드 경로라 별도 회차).
- 다음 역할이 조심할 것: 신규 테스트는 실제 node:http 서버를 127.0.0.1에 띄우고 네이티브 fetch를 쓴다(DB·브라우저는 필요 없다). client.ts는 모듈 로드 시 VITE_API_BASE_URL을 읽으므로 vi.stubEnv → vi.resetModules → dynamic import 순서를 지켜야 한다. Go 코드는 무변경이지만 검증으로 PG14 통합까지 돌렸다.
- [러너 19:29] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: 변경 전 client.ts로 신규 테스트를 실제 실행해 17/21 실패(SyntaxError)를 재현했고 변경 후 21/21 통과 — 테스트가 고친 경로를 지난다. 프런트 전체 144개 통과, tsc -b --force·vite build 통과, 트리 깨끗.
- 구현자가 의심한 (1)은 결함 아님: 구 코드는 json content-type이면 무조건 response.json()이라 깨진/빈 본문에서 이미 항상 throw했다. 오류 응답은 throw→null(개선), 성공 응답은 예외 종류만 SyntaxError→ApiError. 회귀 경로 없음.
- 못 본 것: 실제 프록시 절단·서버 NaN 유입 실측(모사만), Go 통합/릴리즈 전체 경로는 재실행하지 않음(Go 무변경이라 생략).
- 승인이어도 남는 우려: status가 200인 ApiError가 새로 생긴다 — 현재 유일한 status 분기(AuthContext.tsx:71, 401/403)는 무해하지만 릴리즈 노트에 남길 것.
- 다음 회차: helpers.go:29 writeJSON의 Encode 오류 폐기(빈 본문의 실제 발생원)와, /api에 text/html이 오면 request()가 HTML 문자열을 성공 값으로 돌려주는 기존 구멍.
- [러너 19:32] review approved — 리뷰 승인 (risk=low)
- [러너 19:32] pr created — https://github.com/hkjang/jupiq/pull/30
- [러너 19:36] ci passed — 검사 3개 모두 success
- [러너 19:36] merge done — 84ebd9d
- [러너 19:43] release published — v1.8.7
- [러너 19:47] assets verified — v1.8.7 자산 1개 (이전 v1.8.6: 1)
