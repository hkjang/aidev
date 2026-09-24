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
