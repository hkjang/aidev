# 회차 노트 2026-09-24-113424-seaton-improve — seaton
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:34] base pinned — main@00becef
- [러너 11:34] autonomy release — 

## 구현 노트
- 상세의 '조직'이 지정 구역을 먼저 보여 도면 색·툴팁과 다른 조직으로 읽히고, 소속·근무지를 검색 결과에서 찾아 검색 없이는 비던 것을 고쳤다(e6f4f70). 지정 구역은 '지정 구역' 행으로 분리해 불일치를 함께 알린다.
- 확신 없는 곳: `seatOrgDetail` 이 "구역 불일치" 문구를 값 안에 넣는다(`영업팀 · 구역 불일치`). 도면 aria-label 과 같은 말이지만 문구가 두 곳에 각각 적혀 있어, 한쪽만 바뀌면 어긋난다.
- 검증 못 한 것: analysis·tracking·mcp-oauth spec 은 VLM·수집기·IdP 가 없어 돌리지 않았다(이번 변경과 무관한 영역). 좁은 화면(xs) 에서 '지정 구역' 행이 늘어난 상세 높이는 1440x900 에서만 봤다. USER_GUIDE.pdf 는 굽지 않았다(html 만 갱신).
- 일부러 안 한 것: `근무지` 의 `currentMap.buildingName` 대체값은 그대로 뒀다(직원에 근무지가 없을 때 쓰인다). openapi.go 는 좌석 응답 필드를 나열하지 않아 손대지 않았다.
- 다음 역할이 조심할 것: 새 `web/e2e/seat-detail.spec.ts` 는 실서버+DB 가 있어야 돌고, 좌석 구역과 김개발의 workplace 를 실제로 바꿨다가 finally 에서 되돌린다. 중간에 죽으면 김개발 좌석에 남의 팀 구역이 남을 수 있다.
- 주의: `seats.go:36` 은 `rows.Scan(...) == nil` 이라 스캔 대상을 하나라도 빠뜨리면 좌석 목록이 조용히 빈 채로 200 을 반환한다. 이번에 SELECT 열과 Scan 인자를 함께 늘렸고 E2E 로 좌석이 그대로 보이는 것을 확인했지만, 이 자리를 건드리는 다음 변경은 같은 함정을 밟기 쉽다(보류 아이디어로 남김).
- [러너 11:49] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: seats.go:23/36 의 SELECT-Scan 짝, types.go·types.ts 필드, seatOrgDetail 분기 4가지, SeatMapPage 상세 배선(selectedEmployee 제거 후에도 employees 는 검색·드래그에서 계속 쓰임), USER_GUIDE.md↔html 일치. go vet·go test·vitest 106개·tsc 모두 통과.
- 못 본 것: seat-detail.spec.ts(실서버+DB 필요). UI가 seatOrgDetail 을 실제로 쓰는지 증명하는 것은 이 스펙뿐이라, 릴리즈 전 E2E 1회는 돌려야 한다.
- 승인이어도 남는 우려: USER_GUIDE.pdf 미재생성(릴리즈 노트 반영 필요), `구역 불일치` 문구가 seats.ts:70 과 SeatMapPage.tsx:177 두 곳에 각각 적혀 있음, E2E finally 가 setZone 실패 시 saveEmployee 복구를 건너뛰어 김개발 workplace 가 남을 수 있음.
- 보안·법무: 권한 확대 없음(/seats 와 /employees 가 같은 authenticate 그룹, server.go:74-75), workplace 는 기존 수집 항목이라 차단 사유 아님.
- [러너 11:52] review approved — 리뷰 승인 (risk=low)
- [러너 11:52] pr created — https://github.com/hkjang/seaton/pull/34
- [러너 11:56] ci passed — 검사 2개 모두 success
- [러너 11:57] merge done — e6f4f70
