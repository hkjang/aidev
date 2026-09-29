# 회차 노트 2026-09-29-214228-orbit-improve — orbit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:42] base pinned — main@851820f
- [러너 21:42] autonomy release — 

## 정찰 노트
- rows.Err() 누락을 골랐다: 다섯 자리를 실제로 열어 확인했고(data.go:114~163, personal.go 셋, settings.go:207~215) 같은 파일에 올바른 관용구가 이미 있어 수정 자체는 5줄·프로덕션 3파일이다. mcp.go 검색 루프와 auth.go:userByAPIKey 는 구조·보호경로 이유로 일부러 뺐다(파일 수와 판단을 함께 늘리지 않으려고).
- 기각한 후보: mcp.go 의 PersonID uuid 가드 — mcp.go:220~228 오류 렌더를 직접 읽어 500 도 원문 유출도 없고 메시지 문구 차이뿐임을 확정했다(더는 pending 으로 끌고 다니지 말 것). decodeJSON 강화는 공유 헬퍼라 전 엔드포인트 수용 범위를 한꺼번에 좁혀 위험 3 으로 내렸다.
- **확신 없는 곳(가장 중요)**: 행 스트림을 도중에 깨뜨리는 뷰 기법(`users` → `users_probe` + `username::int` CASE)을 정찰은 **실행해 보지 못했다** — 이 환경에서 docker run 이 승인되지 않았다. 프로토콜 추론일 뿐이니 구현자는 이것부터 확인하고, 성립하지 않으면 과제서의 대안 경로(수용 기준 1·2 포기 후 5줄 수정만, 또는 차선 후보로 전환)를 따를 것.
- 구현자가 조심할 것: 뷰 DDL 은 반드시 t.Cleanup 으로 원복하고 t.Parallel 금지(다른 DB 시험과 users 를 공유한다). 커밋되는 것은 테스트 코드뿐이고 migrations 에는 아무것도 더하지 않는다. 손으로 만든 대역으로 증명하지 말 것 — s.store.DB 는 구체 타입 *pgxpool.Pool 이다.
- 프로필은 0일 전이라 새로 쓰지 않았다. 다만 헤더의 `main@c8bceef, VERSION 0.7.4` 는 지금 `main@851820f, VERSION 0.7.5` 로 낡았다 — 그 줄만 무시하고 읽을 것.
- [러너 21:47] scout done — 목록 조회 다섯 곳이 `rows.Err()` 를 검사하지 않아 중간에 끊긴 결과를 200 으로 돌려주는 것을 고치기 (가치

## 구현 노트
- `rows.Err()` 를 빠뜨린 목록 핸들러 다섯 곳(settings.go:listUsers, data.go:listPeople, personal.go 의 listKeys/listAPIKeys/listKeyPermissions)에 같은 파일의 기존 관용구를 채웠다. pgx 는 스트림이 깨져도 Next() 가 조용히 false 라 잘린 목록이 200 으로 나갔다. 프로덕션 3파일 22줄, SQL·시그니처 무변경.
- 정찰이 "미확인" 으로 남긴 뷰 주입법은 **성립한다** — 격리 postgres(포트 55603)에서 재현·수정·단일 줄 되돌림 재실패까지 확인했다. 다만 정찰 예측(200+1행)과 달리 잘린 결과는 **0행**이다: `ORDER BY created_at` 의 Sort 가 블로킹이라 DataRow 가 하나도 나가기 전에 ErrorResponse 가 온다. 결함의 성질(오류가 rows.Err() 에만 남음)은 그대로다.
- **확신 없는 곳**: 깨진 스트림을 실제로 주입해 증명한 것은 `listUsers` 하나뿐이다. 나머지 네 곳은 (a) 코드가 기계적으로 동일하고 (b) 정상 경로 200 회귀를 실제 DB 로 따로 고정했을 뿐, 각각의 깨진 스트림은 주입하지 않았다. 비평가는 여기를 먼저 보면 된다.
- 일부러 하지 않은 것: auth.go:userByAPIKey(401 fail-closed·보호 경로)와 mcp.go:151(구조가 달라 JSON-RPC 오류 매핑 확인 필요)은 과제서대로 제외. migrations 에는 아무것도 더하지 않았다 — 뷰 DDL 은 테스트 안의 일회용이다.
- 다음 역할이 조심할 것: `internal/server/settings_db_test.go` 는 DSN(ORBIT_TEST_DATABASE_URL)이 있어야 돌고 없으면 SKIP 한다(CI 는 SKIP). 이 시험은 `users` 를 전역 DDL 로 잠깐 뒤바꾸므로 `t.Parallel()` 을 넣지 말 것 — 원복은 t.Cleanup 이 하고, 시험 뒤 pg_class 로 users 가 일반 테이블로 복구되고 잔여 행 0 임을 확인했다.
- [러너 21:53] brief accepted — 채택 — 다섯 자리 모두 지금 코드와 정확히 일치했고 수용 기준 1~4 를 그대로 구현했다. 정찰이 실행하지 못해 "미확인" 
- [러너 21:53] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 구현 노트가 의심한 자리를 직접 시험했다: `git archive main` 사본에 새 테스트 파일만 얹어 격리 postgres(55617, 리뷰 뒤 컨테이너 삭제)에서 돌렸더니 `settings_db_test.go:106: status = 200(0행), want 500` 로 **실패**하고 이 브랜치에서는 PASS 한다 — 원장의 `실패 재현:` 줄과 출력이 일치하고 시험이 정말 바뀐 경로를 지난다. DSN 있는 `go test -race ./...` 전부 ok, `gofmt -l internal/server` 무출력, `go vet` 조용. 뷰 컬럼 목록도 001_init.sql 의 users 정의와 정확히 일치했다.
- 못 본 것: 깨진 스트림을 실제로 주입한 것은 여전히 `listUsers` 하나뿐이다. 나머지 네 곳(listPeople/listKeys/listAPIKeys/listKeyPermissions)은 happy-path 200 만 고정되어 있고 그 시험은 **고치기 전에도 PASS** 한다(설계상 회귀 기준). 코드가 기계적으로 동일해 결함으로 보지 않았으나, 다음 회차가 그중 하나의 질의를 손대면 가드가 깨져도 시험은 초록일 수 있다.
- 승인이어도 남는 우려(릴리즈 노트감): `TestListUsersBrokenRowStream` 은 시험 동안 `users` 를 전역으로 `users_probe` 로 개명한다. t.Cleanup 이 원복하고 실패한 실행 뒤에도 pg_class 로 `users` relkind='r'·`users_probe` 없음·잔여 행 0 을 확인했지만, 그 창 안에서 `-timeout` 패닉이나 SIGKILL 이 나면 스키마가 깨진 채 남고 다음 실행은 RENAME 에서 죽는다. 복구 절차가 어디에도 적혀 있지 않다 — DSN 을 오래 쓰는 개발 DB 로 돌리지 말 것.
- 보안·법무 차단 없음: 새 경로·식별자·권한·의존성·마이그레이션 없고 라우팅 무변경, 클라이언트 본문은 기존 generic `internal_error` 뿐이다. 다만 pgx 오류 원문이 이제 slog 에 닿으므로 컬럼 값이 로그에 남을 수 있다(내 실행에서 username `boom-53b04bd0`) — 같은 핸들러의 기존 Scan 오류와 동일한 성질이라 notes 로만 남긴다.
- 되돌리기는 단일 커밋 e16e59d, 코드만. 범위 이탈 없음. 판정 approve / risk low / blocking 없음.
- [러너 21:58] review approved — 리뷰 승인 (risk=low)
- [러너 21:58] pr created — https://github.com/hkjang/orbit/pull/16
- [러너 22:00] ci passed — 검사 1개 모두 success
- [러너 22:00] merge done — e16e59d

## 릴리즈 노트
- v0.7.6 을 이전과 똑같은 방식으로 만들었다: `VERSION` 을 `0.7.5` → `0.7.6` (한 줄, 개행 포함, `v` 접두사 없음), 커밋 `chore(release): v0.7.6` (3aceddd, VERSION 1파일 1줄만), 주석 태그 `v0.7.6` 주석 `Orbit v0.7.6`. 최근 다섯 릴리즈(0.7.1~0.7.5)가 모두 패치 증가였고 이번 변경은 버그 수정 한 건이라 패치로 올렸다. 작성자는 환경 설정 그대로 hkjang, 트레일러 없음, detached HEAD 에서 작업해 브랜치는 만들지도 옮기지도 않았다.
- 버전이 적힌 곳은 `VERSION` **하나뿐**임을 확인했다(`0.7.5`/`0.7.4` 전체 검색 결과 VERSION:1 단 한 줄). `web/package.json` 은 `0.1.0` 으로 릴리즈와 무관하게 고정되어 있어 건드리지 않았다 — 과거 릴리즈 커밋 셋(v0.7.5/v0.7.4/v0.7.3) 모두 `VERSION | 2 +-` 한 파일뿐이었다.
- CHANGELOG·docs/RELEASE* 는 이 저장소에 **없다**. 릴리즈 노트 본문은 `release.yml` 이 `gh release create --generate-notes --title "Orbit $TAG"` 로 GitHub 에서 자동 생성한다(기존 릴리즈 본문의 "What's Changed / by @hkjang in PR" 양식이 그 결과물). 그래서 손으로 쓴 노트 파일은 만들지 않았고 `notes_file` 은 빈 문자열이다.
- **자산은 빈 배열이다** — 이전 릴리즈의 `orbit-vX.Y.Z.tar.gz` 는 사람이 만든 것이 아니라 `release.yml` 이 태그 푸시에 반응해 `docker build` → `docker save | gzip -9` → `gh release create` 로 직접 만들어 붙인다. 같은 이유로 `github_release` 도 false 다(워크플로가 자동 생성하므로 사람이 만들면 `--verify-tag` 단계와 충돌한다). Makefile 의 `release-image` 타깃은 같은 산출물의 로컬 버전이지만 릴리즈 경로가 아니라 실행하지 않았다.
- 검증은 이전 릴리즈가 밟던 CI 게이트(ci.yml)를 이 기계에서 그대로 재현해 전부 초록이다: `gofmt -l .`(무출력) · `go vet ./...`(조용) · `go build ./...` · `go test -race -count=1 ./...`(config·secure·server·scripts ok, DSN 없어 새 DB 시험은 SKIP — 비평 노트의 경고대로 개발 DB 를 붙이지 않았다) · `npm ci` · `npm run test -- --run`(16파일 121시험 통과) · `npm run build`. 여기에 `release.yml` 이 태그 푸시에서 할 `docker build --platform linux/amd64 --build-arg VERSION=v0.7.6` 까지 미리 돌려 성공을 확인했다(확인용 이미지는 바로 삭제). 버전 일치 검사 스크립트는 저장소에 없다.
- 원격에는 아무것도 보내지 않았다 — 푸시·태그 푸시·릴리즈 생성 없음. 커밋 3aceddd 와 주석 태그 v0.7.6 이 로컬에 있고, 푸시되면 `release.yml` 이 태그를 검증하고 이미지·자산·GitHub Release 를 만든다.
- 다음 회차에 넘기는 것: 비평 노트가 "릴리즈 노트감" 으로 남긴 우려(`TestListUsersBrokenRowStream` 이 시험 중 `users` 를 전역 개명하므로 그 창에서 SIGKILL 이 나면 스키마가 깨진 채 남고 복구 절차가 어디에도 없다)는 **이번 릴리즈 본문에 담기지 않았다** — 본문이 자동 생성이라 끼워 넣을 자리가 없고, 문서 추가는 릴리즈 세션의 범위를 넘는다. 개발자용 주의사항이므로 docs 나 테스트 파일 주석에 적는 일을 다음 회차 후보로 남긴다.
