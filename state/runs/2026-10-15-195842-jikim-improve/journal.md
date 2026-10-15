# 회차 노트 2026-10-15-195842-jikim-improve — jikim
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:58] base pinned — main@d7f6832
- [러너 19:58] autonomy release — 

## 정찰 노트
- 고른 이유: 후보 중 **DB 없이 최종 HTTP 응답까지 재현·단정할 수 있는 유일한 S** 였다. 값이 더 큰 transit 비정규 버전 토큰(가치 3)은 `Store` 에 seam 이 없어 200→400 을 보려면 PostgreSQL 통합 테스트가 필요해 한 세션 위험이 커서 차선으로 내렸다(세우기 어려우면 1순위로 돌아오라고 과제서에 적었다). secret_handlers.go 의 `version, _ := Atoi` 와 matchesWildcard·acceptedMCPMediaRange 는 각각 '반환값 무시 광맥 금지' 교훈·결함 부재로 기각하고 ideas.json 에 근거를 남겼다.
- 추측으로 적은 것: 수정 전 `page` 꼬리가 `�` **2개**로 보인다는 것은 Go JSON 인코더가 잘못된 바이트마다 하나를 낸다는 전제이고 실행해 보지 않았다 — 단정은 개수가 아니라 `utf8.ValidString` + 기대 문자열 전체 비교로 하라고 적었다. 프런트가 `page` 를 어떻게 렌더링하는지는 열지 않았다(미확인).
- 조심할 것: 512 **바이트** 상한 자체를 건드리면 무인증 엔드포인트의 메모리 예산 계약이 바뀐다. 꼬리의 *불완전한* 시퀀스만 떼야 하므로 `DecodeLastRuneInString` 의 `size <= 1` 조건이 필수다(URL 안의 진짜 U+FFFD 는 보존). `unicode/utf8` 은 violations.go 에 아직 import 되어 있지 않다.
- 회차 추세 주의: tracking 영역을 4회 연속 고른 셈이다. 같은 파일의 `MaxViolations`·`evictOldest`·`List` 정렬로 번지지 않게 과제서에 금지 목록을 명시했다.
- [러너 20:05] scout done — CSP 리포트의 `page` 512바이트 절단이 다중 바이트 문자를 쪼개 관리자 목록에 U+FFFD 를 남기는 문제 수정 (가

## 구현 노트
- `Recorder.Record` 의 `page = page[:512]` 가 다중 바이트 문자를 쪼개 잘못된 UTF-8 을 저장하고, 관리자 목록의 `json.Marshal` 이 그 바이트마다 U+FFFD 를 넣어 주소 끝이 깨져 보였다. 512**바이트** 상한은 그대로 두고 꼬리의 불완전한 시퀀스만 떼어냈다(`DecodeLastRuneInString` + `size <= 1`, 진짜 U+FFFD 는 size==3 이라 보존). 프로덕션 1파일 +13줄, 테스트 1파일. 커밋 `ce49f97`.
- 확신 없는 곳·검증 못 한 것: 프런트(`web/src`)가 `page` 를 어떻게 렌더링하는지는 열지 않았다 — 서버 JSON 까지만 확인했고 화면은 미검증(프런트 무수정 전제, 59/59 그대로 통과). 실제 브라우저가 보내는 CSP 리포트로는 확인하지 않았고 `httptest` 왕복까지다. PostgreSQL 연동 테스트는 DSN 미설정으로 skip, 이미지·Playwright E2E 는 미실행(`make release-check` 는 지시대로 돌리지 않았다).
- 일부러 하지 않은 것: 512 상한 변경·"rune 512개"로 바꾸기(무인증 엔드포인트 메모리 예산 계약), `strings.ToValidUTF8`(떼어내는 대신 치환 문자를 넣는다), `internal/tracking/tracking_test.go` 의 중복 단위 표(HTTP 왕복 5개가 같은 경계를 이미 덮고 교훈이 최종 응답 확인을 요구한다), `MaxViolations`·`evictOldest`·`List` 정렬·`Origin`/`Directive` 경로.
- 다음 역할이 조심할 것: 새 테스트는 `TestRecordedPageIsTruncatedOnACharacterBoundary`(internal/httpapi/tracking_test.go) 하나이고 DB 없이 돈다. 서브테스트 5개 중 4개는 **기존 계약**이라 수정 전부터 통과했다 — 실패하는 것은 `multi-byte_character_split_by_the_cap` 뿐이다. 수정 재현 시 루프만 되돌리면 `"unicode/utf8" imported and not used` 로 빌드가 깨지므로 import 도 함께 되돌려야 한다. 재현값(21바이트 접두사·163개 `가`·510바이트)은 접두사 길이에 민감해 접두사를 바꾸면 기대값도 다시 계산해야 한다.
- [러너 20:10] brief accepted — 채택 — 근거·배선·재현값(21바이트 접두사 + `strings.Repeat("가", 200)`, 기대값 510바이트)·권장 수정 형태·`unicode/utf8` 미impo
- [러너 20:10] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 approve(risk low, blocking 없음). 테스트 유효성을 독립 확인: `internal/tracking/violations.go` 만 main 판으로 되돌려 `tracking_test.go:451: listed page is 516 bytes ending "가가??" ... want 510 bytes ending "가가가가"` 로 1개만 실패 — ledger 의 `- 실패 재현:` 줄과 일치, 이후 워크트리 복원해 `git status` 깨끗.
- 절단 루프는 난수 혼합 문자열 50만 건으로 불변식(유효 UTF-8·512바이트 이하·3바이트 이하만 제거·원본 접두사·들어갈 수 있던 rune 보존)을 확인했다. 512바이트 메모리 예산은 줄기만 하고 늘지 않으며 무인증 report 경로·admin 게이트(server.go:151)·204 빈 본문은 그대로다.
- 구현자가 미검증으로 남긴 프런트를 대신 확인: `SettingsPage.tsx:1497` 이 `{item.page}` 를 텍스트로만 쓰고 길이 가정이 없어 프런트 무수정이 맞다. docs/README 에 `512` 언급 없어 문서 불일치도 없다.
- 승인이어도 남는 것: (a) `tracking_test.go:451` 의 `!utf8.ValidString(got)` 는 HTTP 경로에서 죽은 가지다(json.Marshal 이 먼저 유효 U+FFFD 로 바꾼다) — 실제 탐지는 RuneError 차등 + 전체 일치이니 가드로 오독하지 말 것. (b) `Record` 의 `origin` 은 길이 상한이 없다(선재·범위 밖). (c) tracking 영역 5연속이 되므로 다음 회차는 다른 영역으로.
- 못 본 것: 실제 브라우저 CSP 리포트, 이미지·Playwright E2E, PostgreSQL 연동(DSN 미설정 skip), `scripts/verify.sh` 재실행(구현자 exit 0 보고를 신뢰). 릴리즈 노트에는 "관리자 추적 목록의 긴 비ASCII 주소 끝 글자가 깨져 보이던 문제 수정"으로 적으면 된다.
- [러너 20:13] review approved — 리뷰 승인 (risk=low)
- [러너 20:14] pr created — https://github.com/hkjang/jikim/pull/61
- [러너 20:17] ci passed — 검사 2개 모두 success
- [러너 20:17] merge done — ce49f97
