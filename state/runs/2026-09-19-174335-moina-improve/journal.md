# 회차 노트 2026-09-19-174335-moina-improve — moina
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:43] base pinned — main@e8a4131
- [러너 17:43] autonomy release — 

## 정찰 노트
- 우선 과제의 근거 "hold: budget"은 러너 예산 상한으로 회차가 끊긴 기록이지 워크플로 결함 신호가 아니라고 판단했고, `gh run list`가 승인 거절돼 실제 release run 결과는 미확인 — 그래서 과제서를 "실패 run 확인 → 있으면 복구(A) / 없으면 updatePost 409→500 분리(B)" 분기로 썼습니다.
- 추측으로 적은 것: 실패했다면 Runtime smoke(시각 회귀/qa-api-smoke) 단계일 가능성이 높다는 것(과거 이력 기준), mail 캠페인 branch의 main 머지 여부.
- 확인한 것: `posts.go:832`가 err와 RowsAffected()==0을 한 분기로 409 처리, 같은 파일 삭제 경로(965-971)는 이미 분리 — B의 근거는 실제 코드입니다.
- 구현자 주의: release.yml은 한 줄도 느슨하게 하지 말 것, 시각 회귀 갱신은 공식 Playwright 이미지에서만, B의 테스트는 대역 DB가 아니라 PostgreSQL integration으로 증명할 것.
- 이 세션은 make/bash/node 스크립트 실행이 모두 승인 거절돼 로컬 검증을 하나도 돌리지 못했습니다(ideas.json에 러너 허용 목록 항목으로 남김).
- [러너 17:47] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- 우선 과제의 전제(릴리즈 워크플로 2회 실패)는 GitHub API로 반증 — release.yml run v0.1.25~v0.1.32 전부 success, CI main도 success. `error`는 러너 예산 중단(`hold: budget`)이라 release.yml은 0줄 변경. 과제서 분기대로 차선 B를 구현: `updatePost`의 UPDATE에서 `err != nil`→500 `storage_error`, `RowsAffected()==0`→409 `not_editable`(코드·메시지 그대로)로 분리(9d8a6d5).
- 검증 방법에서 확신이 덜한 곳: 저장 오류는 테스트 전용 `BEFORE UPDATE` 트리거(이 테스트만의 sentinel 본문을 RAISE)로 유도 — 요청은 실제 `New()` 서버·세션·CSRF·pgx를 지나고 오류도 진짜 PostgreSQL 오류이지만, "DB 연결 끊김" 그 자체를 재현한 것은 아님. 트리거·함수 이름은 suffix로 유일하고 Cleanup에서 DROP하므로 같은 DB의 다른 integration 테스트에는 영향 없음(전체 `go test -race ./...`로 확인).
- OpenAPI PATCH 응답을 403→400·409로 바로잡음. 500은 이 문서가 어떤 경로에도 적지 않는 관례라 설명 문장에만 둠. `not_editable`을 읽는 프런트·문서·MCP 코드 없음(grep).
- 일부러 하지 않은 것: 같은 route의 DELETE 응답(403으로 적혀 있으나 실제 404) — 범위 밖이라 ideas.json에 남김. `deletePost`의 Commit/UPDATE 오류 합침도 그대로(이미 500).
- 다음 역할 주의: 새 테스트 `posts_update_postgres_integration_test.go`는 `MOINA_TEST_POSTGRES_DSN`이 있어야 돌고(없으면 skip), 트리거를 만들므로 DB에 posts 테이블 트리거 생성 권한이 필요(throwaway docker에서는 문제 없음). 프런트·e2e 무변경이라 vitest·시각 회귀는 돌리지 않았음.
- [러너 17:53] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인: 심사 대상은 단일 커밋 9d8a6d5(로컬 main은 v0.1.28로 낡음, 실제 base origin/main@e8a4131). posts.go 분리 로직은 deletePost 관례와 일치하고 올바름. 새 integration 테스트를 throwaway postgres:17에서 직접 돌려 HEAD PASS · origin/main 판 posts.go에서 FAIL(409로 감춤)을 확인 — 테스트가 수정을 실제로 판별함. go vet 통과.
- 거절 사유(수리가 먼저 볼 파일): `api/openapi.yaml:189` — PATCH /posts/{postID}는 `requirePermission("posts:write")`(server.go:171→575 `403 forbidden`)와 CSRF 검사(server.go:563 `403 invalid_csrf`)를 지나므로 403이 실제로 나옴. 형제 route POST/DELETE /posts는 403을 유지하고 있어(createPost 핸들러 자체는 403을 안 냄 = 문서의 403은 미들웨어 거부 관례) '403'을 400·409와 함께 되살리면 끝. 코드·테스트 변경 불필요.
- 구현 노트의 "DELETE도 403이 아니라 404" 판단도 같은 이유로 틀림 — ideas.json에 남긴 그 항목은 폐기 권장.
- 못 본 것: 실제 DB 연결 끊김 재현(트리거 RAISE로 대체됨, 수용 가능), 전체 `go test -race ./...`·make check(구현 세션이 통과 보고).
- 보안·법무 차단 소견 없음(인가·쿼리·개인정보 변경 없음, 500 본문은 고정 문구라 DB 오류 노출 없음).
- [러너 17:56] review rejected — 리뷰 거절: api/openapi.yaml:189 PATCH /posts/{postID} 응답에서 '403'을 '실제로 나오지 않는' 코드라며 제거했지만 실제로 나옵니다 — 이 route는 backend/internal/httpa

## 수리 노트
- 맞았던 지적: PATCH /posts/{postID}의 '403' 제거 — server.go:171 `requirePermission("posts:write")`→575 `403 forbidden`, 563 `403 invalid_csrf`를 직접 확인. 구현 노트의 "403은 실제로 안 나옴"·"DELETE도 404" 전제는 미들웨어를 빠뜨린 오판(ideas.json의 DELETE 항목도 폐기해야 함).
- 고친 방법: openapi.yaml:189 responses에 '403'을 400·409 사이에 되돌림(3b95053). 400·409·description은 그대로 두었고 코드·테스트 무변경.
- 검증: check-openapi-routes(120) · check-runtime-contract(4) · go build/vet · `go test -race ./internal/httpapi/` ok.
- 확신 없는 곳: 없음 — 단, 전체 `make check`·frontend는 yaml 한 줄 변경이라 돌리지 않았음.
- [러너 17:57] repair done — # 수리 요약 (시도 1) - 문제: 9d8a6d5가 `api/openapi.yaml:189` PATCH /posts/{postID} responses에서 '403'을 지웠으나, 이 route는 `auth.With(s.requirePermission("posts:write"))`(server.

## 비평 노트
- 확인(2차, 수리 후): 대상은 origin/main@e8a4131 대비 9d8a6d5+3b95053(로컬 main은 낡음). posts.go:832-839 분리 로직은 deletePost 관례와 일치. 새 integration 테스트를 throwaway postgres:17에서 직접 실행 — HEAD PASS, posts.go만 origin/main 판으로 되돌리면 500 케이스가 409로 FAIL → 테스트가 수정을 실제로 판별. cleanup 후 trigger/function/행 잔여 0건.
- 수리 확인: openapi.yaml:190에 '403' 복원됨 — server.go:171 requirePermission→575 forbidden, 563 invalid_csrf 경로 직접 확인. check-openapi-routes 120개·go vet 통과. 판정 approve, risk low, 차단 없음.
- 못 본 것: 실제 DB 연결 끊김 재현(트리거 RAISE 대체, 수용), 전체 `go test -race ./...`·make check·frontend(구현·수리 세션 보고에 의존).
- 다음 회차 후보: posts.go storage_error 경로 41곳 모두 err를 로그에 남기지 않음 — 이번 변경으로 status는 구분되지만 원인은 여전히 로그에 없음. ideas.json의 "DELETE도 404" 항목은 미들웨어 403을 빠뜨린 오판이니 폐기.
- 릴리즈 노트: PATCH /posts/{postID}가 저장 오류 시 409 대신 500 storage_error를 냄(클라이언트 분기 영향 없음 — not_editable 소비 코드 없음 grep 확인은 구현 노트).
- [러너 18:00] review approved — 리뷰 승인 (risk=low)
- [러너 18:00] pr created — https://github.com/hkjang/moina/pull/26
- [러너 18:09] ci passed — 검사 2개 모두 success
- [러너 18:09] merge done — 3b95053
- [러너 18:22] release published — v0.1.33
- [러너 18:28] assets verified — v0.1.33 자산 1개 (이전 v0.1.32: 1)
