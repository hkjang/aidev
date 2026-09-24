# 회차 노트 2026-09-23-123430-visitflow-improve — visitflow
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:34] base pinned — main@7d3dccb
- [러너 12:34] autonomy release — 

## 정찰 노트
- 일정 사전 검사를 골랐다: 화면이 회사명·체크리스트·장비는 이미 막는데 날짜만 무방비라 빈 칸이면 영어 RangeError가 그대로 뜬다. 가져오기 계열 후보는 미머지 브랜치(0921 중복 헤더)와 파일이 겹쳐 제외했고, 메일·MCP OAuth 후속은 브랜치가 아직 main에 없어 또 미뤘다.
- 확신 없는 곳: 실제 Chromium에서 datetime-local을 비웠을 때 state가 ""가 되는지는 브라우저로 재현하지 않았다(소스+ECMA 계약 근거). node_modules가 없어 npm test/lint도 이번엔 미실행이다.
- 구현자 주의: 경계를 서버와 정확히 맞출 것(종료==시작 거절, 정확히 31일 통과, 31일+1분 거절). 서버 visits.go는 건드리지 말고 화면만 고친다.
- vitest는 src/**/*.test.ts만 수집하므로 순수 헬퍼를 .ts로 빼고 .tsx 테스트는 만들지 말 것. 변이 되돌리기에 git checkout -- 파일 금지.
- 프로필은 회사명 정책 머지로 낡아 profile.md를 새로 썼다.
- [러너 12:37] scout done — 방문 신청 화면에서 방문 시작·종료 시각을 제출 전에 검사 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 무엇/왜: `web/src/schedule.ts`의 `scheduleError` 하나가 빈 값·파싱 실패·종료<=시작·31일 초과를 돌려주고, `VisitFormPage`의 두 날짜 칸(error/helperText)·제출 버튼 disabled·submit 가드가 모두 그 값 하나를 읽는다. 무가드 `toISOString()`이 던지던 영어 RangeError를 없애고 서버와 같은 경계로 미리 막는다. 서버 visits.go·`localInput`·타임존 처리는 손대지 않았다(커밋 cf0bd7e).
- 확신 없는 곳: 실제 Playwright e2e(`npm run test:e2e`)는 **돌리지 못했다** — Go 서버·PostgreSQL·dist 복사가 필요해서, 대신 `npm run build` 산출물을 스텁 API(auth/config·auth/me·reference-data·visits POST) 뒤에 서빙하고 진짜 Chromium으로 확인했다. 즉 서버 응답은 가짜이고 화면·번들·브라우저만 진짜다. 서버 경계 일치는 visits.go 소스(`!EndAt.After`, `> 31*24*time.Hour`)를 읽어 맞췄을 뿐 Go 테스트로 교차 검증하지는 않았다(이번 변경에 Go 코드가 없어 Go 테스트는 build/vet만 돌렸고 DB 통합은 미실행).
- 정찰 과제서가 예로 든 `2026-02-31T10:00`은 V8이 3월 3일로 롤오버해 Invalid Date가 아니다. 그래서 테스트 전제를 실제 파서 동작에 맞추고, 화면과 submit이 같은 파서로 같은 값을 읽는다는 것을 대신 고정했다. 비평가는 이 부분을 먼저 볼 것.
- 일부러 안 한 것: 방문자 100명 상한(차선 후보, 같은 파일이라 다음 회차), 과거 시작 시각 차단(서버가 허용하므로 막으면 불일치), USER_GUIDE PDF 재생성(과제서 지시), 기준 정보 재시도 버튼(같은 파일 중복 변경 회피).
- 다음 역할 주의: 새 테스트 `web/src/schedule.test.ts`는 DB도 브라우저도 필요 없고 `cd web && npm ci && npm test`로 23개(기존 8 + 신규 15)가 돈다. 변이 되돌리기는 `/tmp` 임시 복사본으로 했고 `git checkout --`는 쓰지 않았다.
- [러너 12:43] brief accepted — 채택 — 지정한 파일·경계·문구 그대로 구현했고, 과제서가 미확인으로 남긴 "빈 datetime-local이 React state에서 ""가 되는가
- [러너 12:43] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: 화면 경계(web/src/schedule.ts:9-13)가 서버 visits.go:441-445와 문구까지 일치. 돌연변이 2종(`<=`→`<`, `>`→`>=`)으로 테스트가 실제 경계를 잡는 것을 확인(각 2개 실패). npm test 23/23·lint·build·go build/vet 통과, 트리 clean.
- 구현자가 미확인으로 남긴 타임존을 대신 시험했다 — TZ 7종(Auckland·Sydney·Santiago·Chatham 포함) 모두 15/15 통과. DST 우려 해소.
- 못 본 것: DB 통합(DSN 없음)과 Playwright e2e. 이번 diff에 Go 코드가 없고 e2e는 날짜 칸을 건드리지 않아(visit-flow.spec.ts:191-222) 영향 없다고 판단.
- 승인이어도 남는 우려: (1) 시작이 비면 종료 칸까지 빨갛게 되고 문구가 두 번 보인다 — USER_GUIDE.md:43 '해당 칸에'는 약간 앞서간 표현이니 릴리즈 노트에서 과장 금지. (2) VisitFormPage 배선(disabled·helperText·submit 가드)은 자동 테스트가 없다(vitest가 .tsx 미수집) — e2e에 '시작 칸 비우면 제출 비활성' 한 줄이 가장 싼 보강, 다음 회차 후보.
- 보안·법무 차단 없음: 클라이언트 전용 순수 함수뿐이고 인증·암호·마이그레이션·개인정보 수집 경로에 변화가 없으며 서버 검증이 권위를 유지한다.
- [러너 12:46] review approved — 리뷰 승인 (risk=low)
- [러너 12:47] pr created — https://github.com/hkjang/visitflow/pull/23
- [러너 12:51] ci passed — 검사 2개 모두 success
- [러너 12:51] merge done — cf0bd7e
- [러너 12:57] release published — v2.8.5
- [러너 12:59] assets verified — v2.8.5 자산 1개 (이전 v2.8.4: 1)
