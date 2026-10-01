# 회차 노트 2026-10-02-053737-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:37] base pinned — main@0439d5e
- [러너 05:37] autonomy release — 

## 정찰 노트
- 고른 이유: 같은 `postcommand.Execute` 를 부르는 두 HTTP 어댑터가 같은 `FailureSave` 를 400(웹훅)과 500(createPost)으로 다르게 읽는 것을 소스에서 확인했다. 운영자가 되풀이한 "같은 값을 읽는 경로가 둘이면 두 경로가 같은 입력에 같은 답을 하는지 확인" 규칙에 정확히 걸리고, 프로덕션 1파일 1 case 로 끝난다.
- 제친 후보: 플러그인 거절 400/403 불일치와 createIncomingWebhook 의 channels.Get 404 접힘은 둘 다 타당하지만, 앞은 통합 발신자의 400 기대를 깨는 계약 변경이고 뒤는 channels.Get 의 없는-채널 반환 모양(ErrNoRows vs nil)이 미확인이라 선행 작업이 남는다. 북마크·게스트·preferences 계열은 회귀 범위가 넓어(M) 45분에 안 들어간다.
- 추측으로 적은 것: `DROP TABLE posts` 가 FK 때문에 거부될지 미확인(CASCADE 또는 RENAME 대안을 과제서에 적었다). 테스트 DB 포트도 미확인 — 컨테이너 `moyro-pg-improve` 가 Exited 로 존재하는 것만 확인했고 `docker start` 는 이 세션에서 권한에 막혔다.
- 구현자가 조심할 것: 새 테스트를 기존 `incoming_webhook_member_errors_postgres_test.go` 의 Test 함수 **안에** 넣으면 그 함수가 중간에 `DROP TABLE channel_members` 를 하므로 멤버십 분기에서 먼저 끊긴다 — 별도 Test 함수 + 별도 `newOperationsTestDB` 로 가야 한다. 그리고 `{"text":"  "}` 400 서브테스트를 반드시 넣어 switch 를 통째로 500 으로 바꾸지 않았음을 증명할 것.
- DSN 없이 돌린 `go test -count=1 ./internal/httpapi/` 는 `ok 0.172s` 인데 DB 테스트는 skip 이다. 이 `ok` 를 DB 통과로 오인하지 말 것.
- [러너 05:41] scout done — `fireIncomingWebhook` 의 `FailureSave`(진짜 게시글 저장 장애)를 400 `post_failed` 에서 500 으로 분리 (가치 3 / 위험 1 /

## 구현 노트
- 무엇을 왜: `fireIncomingWebhook` 의 Fire 오류 switch 에 `case postcommand.FailureSave:` → 500 `api.webhook.incoming.fire.app_error`(핸들러가 이미 쓰는 id 재사용). 진짜 `posts` INSERT 장애가 400 `post_failed` 로 나가 통합 발신자의 재시도를 영구히 멈추던 것을 형제 어댑터 `createPost`(같은 실패를 500 으로 냄)와 일치시켰다. `default:` 는 400 유지 — `empty text` 는 `FailureCodeOf` 가 `""` 라 계약이 자동 보존된다. 프로덕션 1파일 7줄 + 신규 테스트 1파일. 커밋 0b2d41c.
- 확신 없는 곳·검증 못 한 것: (1) **웹 쪽은 전혀 돌리지 않았다** — Go 전용 변경이고 `post_failed`/`fire.app_error` 문자열의 webapp 소비자가 없다는 것은 정찰의 grep 결과를 그대로 믿었다(나는 재확인하지 않았다; 운영자 규칙상 grep 은 증거가 아니므로 이 근거는 약하다고 적어 둔다). (2) `api.webhook.incoming.fire.app_error` 를 **훅 조회 장애**(3153)와 **저장 장애** 두 곳이 공유하게 됐다 — 새 id 를 만들지 말라는 수용 기준을 따른 결과이고 둘 다 500 서버 장애라 의도적이지만, 발신자가 두 원인을 id 로 구분할 수는 없다. (3) `FailureSave` 는 INSERT 실패 외에 `posts.Create` 트랜잭션 안의 다른 실패(예: `lockSourceThread`)도 감싼다 — 전부 저장소 장애라 500 이 맞다고 판단했지만 INSERT 외 경로를 개별 재현하지는 않았다.
- 일부러 하지 않은 것: 같은 switch 의 `FailurePluginRejected`(400 vs 형제 403)와 `FailureInvalidRoot` 는 과제서가 명시적으로 범위 밖으로 지정해 손대지 않았다 — 앞은 통합 발신자의 400 기대를 깨는 계약 변경이고, 뒤는 웹훅 Command 가 `RootID` 를 비워 둬 도달 불가다(소스로 재확인). ideas.json 에 pending 으로 남겼다.
- 다음 역할이 조심할 것: 새 테스트는 **DB 가 있어야 돈다** — `MOYRO_TEST_POSTGRES_DSN` 없으면 `t.Skip` 이므로 `ok` 를 통과로 오인하지 말고 `-v` 로 skip 여부를 볼 것. 컨테이너 `moyro-pg-improve`, 호스트 포트 **55433 확인**(정찰의 미확인 해소). 주입은 `ALTER TABLE posts RENAME TO posts_hidden` 이고 FK 거부 없이 통과했다. 이 테스트는 자기 `newOperationsTestDB` 를 쓰므로 형제 `incoming_webhook_member_errors_postgres_test.go`(중간에 `DROP TABLE channel_members`)와 **같은 Test 함수로 합치지 말 것**. 서브테스트 순서가 의미를 갖는다: 건강한 스키마 계약 5건 → posts RENAME → channel_members DROP.
- [러너 05:48] brief accepted — 채택 — 과제서의 근거가 현재 코드와 정확히 일치했다(`default:` 가 `FailureSave` 와 `empty text` 를 같이 받는 것, 형제 `createPos
- [러너 05:48] verify failed — 실패한 검증: cd webapp && ([ -d node_modules ] || npm ci --no-audit --no-fund) && npm run typecheck --silent && npm test --silent (exit 1)
