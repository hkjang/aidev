# PR 처리기 노트 2026-09-24-065958-moyro-shepherd — moyro PR #23
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-24-053422-moyro-improve)
# 회차 노트 2026-09-24-053422-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:34] base pinned — main@4ca2ff9
- [러너 05:34] autonomy release — 

## 정찰 노트
- 북마크 교차채널 구멍을 골랐다: `bookmarks.Patch/Delete` 의 SQL 이 `WHERE id=$1` 뿐이고 같은 파일 `Reorder` 만 `channel_id=$2` 를 쓴다 — 실제 권한 경계가 깨지는 유일한 후보라 최근 5회차가 반복한 오류매핑(403/404→500) 후보들을 제쳤다. 그 오류매핑 건은 차선으로만 남겼다.
- 추측으로 적은 것: (a) `go vet`/`go build`/테스트를 샌드박스가 거부해 **한 번도 돌리지 못했다** — 근거는 전부 소스 읽기다. (b) `check-source-sizes.sh` 의 상한 여유는 프로필의 early.go 수치를 옮긴 것이라 late.go 기준은 미확인. (c) 서비스 시그니처 변경안은 호출처가 httpapi 4곳뿐임을 grep 으로 확인했으나 플러그인 SDK 쪽은 미확인.
- 구현자가 조심할 것: "다른 채널 것" 을 403 이 아니라 기존 404 `api.bookmark.not_found` 로 낼 것(id 존재 누출 방지 + 계약 불변). patch 와 delete **양쪽** 을 고칠 것 — 한쪽만 막으면 같은 공격이 다른 쪽으로 그대로 된다. IsMember 오류매핑을 같은 PR 에 끼워 넣지 말 것.
- 프로필은 2일 전 것이 지금 코드와 어긋나지 않아 새로 쓰지 않았다(base 가 960d3a1→4ca2ff9/v0.2.35 로 옮겨간 것만 드리프트).
- [러너 05:38] scout failed — 과제서 없음 — 구현자가 직접 고른다
- [러너 06:00] verify passed — 검증 2개 통과 (policy)
- [러너 06:01] pr created — https://github.com/hkjang/moyro/pull/23
- [러너 06:01] guard held — server/internal/store/migrations/000018_post_reminder_leases.up.sql 

## 심사 노트
- 실제 PostgreSQL 16(:55433)으로 돌렸다: 새 테스트 7개·서버 전체 `go test -p 1 ./...`·`go vet ./...`·check-source-sizes 모두 통과. skip 아님을 -v 로 확인.
- 역검증했다: 프로덕션 4파일을 origin/main 으로 되돌리고 000018 을 치우자 북마크 테스트가 200(교차채널 patch 성공)을 잡아 실패하고 마이그레이션 테스트도 실패했다. 구멍은 실존하고 테스트는 유효하다. 이후 트리 원상복구(clean).
- 마이그레이션 000018 은 추가 전용(ADD COLUMN/CONSTRAINT/INDEX, UPDATE 는 delivered_at=-1 행뿐)이라 사람 몫이 아니라고 판단했다. claimLeaseDuration 2분은 scheduled/service.go:24 와 정확히 같아 주석 서술도 사실이다.
- 못 본 것: 다중 인스턴스 동시 워커의 실제 경합(단일 프로세스로만 확인), 웹 빌드·e2e(Go 전용 변경이라 생략), fire() panic 시 무한 재시도 경로는 재현하지 못했다.
- 권고 merge / risk medium. 범위 혼합(북마크+리마인더 2건)과 claim_token·attempt 상한 부재는 notes 로만 남겼다 — 공격 경로나 데이터 손상이 없어 차단하지 않았다.
