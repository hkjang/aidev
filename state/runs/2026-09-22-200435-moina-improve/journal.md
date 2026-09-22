# 회차 노트 2026-09-22-200435-moina-improve — moina
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 20:04] base pinned — main@c336a30
- [러너 20:04] autonomy release — 

## 정찰 노트
- `joinMoim`을 골랐다: 같은 파일 `leaveMoim`과 이미 릴리즈된 `updatePost`(9d8a6d5)가 저장 오류를 500 `storage_error`로 보고하는데 `joinMoim`만 404 `not_found`로 흡수하고 있고, join 경로 테스트가 httpapi에 0건이라 관례 정합·테스트 공백을 한 번에 메운다.
- 두 번 연속 no-change로 끝난 Makefile `-race`는 근거가 여전히 유효하지만 세 번째로 올리지 않았다(반복 무성과 회피). 재캡처·placeholder 정규화는 앱·DB·브라우저와 베이스라인 재승인이 필요해 45분 밖이다.
- 추측으로 적은 것: 트리거로 만든 INSERT 실패가 `ON CONFLICT DO NOTHING`과 함께일 때도 `Exec` 오류로 올라오는지는 posts 선례에서의 추론이며 직접 재현하지 않았다. DSN이 없어 integration은 이 환경에서 skip된다.
- 이번 세션에서 `go vet`·`go test`·`make` 실행이 도구 승인 거부로 불가능했다. 정찰 단계의 실행 결과는 없으므로 구현자가 검증을 처음부터 돌려야 한다.
- 조심할 것: `RowsAffected()==0`(없는 slug·비공개·재가입) 경로가 500으로 새지 않게 테스트로 못 박고, OpenAPI는 description 한 줄만 더하되 기존 응답 항목은 지우지 말 것(3b95053 교훈). 테스트 트리거는 `t.Cleanup`에서 반드시 DROP.
- [러너 20:07] scout done — `joinMoim`이 DB 저장 오류를 404 `not_found`로 감추는 것을 500 `storage_error`로 분리 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- `joinMoim`의 INSERT 실패만 500 `storage_error`("Moim에 가입할 수 없습니다")로 분리했고 `RowsAffected()==0`의 404 `not_found`와 문구는 손대지 않았습니다(c8d3628). OpenAPI는 join·members 두 경로에 `description` 한 줄만 추가, 응답 목록은 무변경.
- 정찰이 추측으로 남긴 "`BEFORE INSERT` 트리거 실패가 `ON CONFLICT DO NOTHING`과 함께여도 `Exec` 오류로 올라오는가"는 red 단계에서 실제로 확인했습니다 — 수정 전 코드에서 저장 오류 2케이스만 404로 실패하고 나머지 5케이스는 통과.
- 확신 없는 곳: frontend eslint가 이 worktree에 설치돼 있지 않아 `make lint`의 frontend 단계는 실행하지 못했습니다(Go vet·staticcheck는 통과). 프런트·e2e는 무변경이라 vitest·시각 회귀도 돌리지 않았습니다.
- 일부러 하지 않은 것: 같은 분기 안의 `_ = …Scan(&exists)`(조회 오류 무시)는 계약이 바뀌므로 범위 밖으로 두고 ideas.json에 pending으로 남겼습니다. `leaveMoim`·`updatePost`·social.go의 다른 핸들러는 건드리지 않았습니다.
- 다음 역할이 조심할 것: 새 테스트는 `MOINA_TEST_POSTGRES_DSN`이 있어야 돌고 없으면 `t.Skip`이라 저장 오류가 증명되지 않습니다. 검증에 쓴 throwaway 컨테이너는 `moina-test-pg`(127.0.0.1:55433)이며 아직 떠 있으니 필요 없으면 `docker rm -f moina-test-pg`로 지우세요.
- 트리거는 `t.Cleanup`에서 DROP 하지만 테스트가 강제 종료되면 `moim_members`에 남아 이후 가입이 전부 깨집니다 — 이름은 `moina_test_refuse_join_trg_<nano>`.
- [러너 20:10] brief accepted — 채택 — 과제서의 근거(social.go:565 joinMoim의 404 흡수, leaveMoim의 500 관례, join 경로 테스트 0건)가 현재 코드와 정확히 일치�
- [러너 20:11] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 구현 노트가 비워 둔 자리(DSN 없으면 저장 오류 미증명)를 직접 메웠습니다: throwaway postgres:16(127.0.0.1:55977)로 HEAD 7/7 PASS, c336a30 detached worktree + 테스트 파일 복사로 red 재현(저장 오류 2케이스만 404 FAIL), -race -count=2 통과. 실행 후 moina_test% 트리거·함수 0건, 관련 행 0건 — t.Cleanup이 정말 회수합니다. 같은 DSN으로 httpapi·store 전체 통과.
- 못 본 것: staticcheck 미설치로 미실행, 프런트·e2e는 이 diff에서 무변경이라 미실행(MoimsPage.tsx:207은 readableError로 상태코드 분기 없이 토스트만 띄워 404→500 전환에 깨지지 않음을 코드로만 확인).
- 승인이어도 남는 우려: 같은 분기의 `_ = …Scan(&exists)`가 그대로라 EXISTS 쿼리 실패 시 기존 멤버도 여전히 404 — openapi.yaml:470,482의 "다시 요청해도 200"에 드문 예외가 있습니다. 또 500으로 분리했을 뿐 DB 오류 본문은 로그에 남지 않아 운영자는 http_status=500만 봅니다(leaveMoim·updatePost 동일).
- 릴리즈 노트용: 동작 변경은 `POST /moims/{slug}/join`·`/members`의 저장 실패가 404 not_found → 500 storage_error 한 건뿐. 마이그레이션·외부 상태 없음 → revert는 3파일 되돌리기.
- 다음 회차 후보: storage 오류 구조적 로깅, 그리고 정찰이 남긴 Makefile `-race`(세 번째 보류 중).
- [러너 20:15] review approved — 리뷰 승인 (risk=low)
- [러너 20:15] pr created — https://github.com/hkjang/moina/pull/29
- [러너 20:24] ci passed — 검사 2개 모두 success
- [러너 20:24] merge done — c8d3628
- [러너 20:37] release published — v0.1.36
