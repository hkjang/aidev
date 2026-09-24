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

## 릴리즈 노트
- v1.4.6 (패치 — 최근 5회가 모두 패치였고 이번 변경은 결함 하나를 닫는 것). 태그 `v1.4.6` 을 a964d46 에 경량 태그로 달았다(이전 5개 태그 모두 경량, 주석 없음). 푸시는 하지 않았다.
- 릴리즈 커밋 a964d46 `docs: v1.4.6 기준으로 문서 정비` — v1.4.3~v1.4.5 와 같은 제목·같은 두 항목 구성. 바뀐 파일도 v1.4.5 회차와 같은 10개(README.md, ADMIN_GUIDE.md/html/pdf, USER_GUIDE.md/html/pdf, ROADMAP_PLAN.md/html/pdf). 버전 파일은 없고 버전 문자열은 문서 4곳(README, ADMIN_GUIDE, USER_GUIDE, ROADMAP_PLAN)에만 있다 — 애플리케이션 버전은 Dockerfile `--build-arg VERSION` 으로 주입된다.
- 비평 노트가 남긴 두 숙제를 릴리즈에서 닫았다: (1) 미뤄 둔 USER_GUIDE.pdf 를 ADMIN_GUIDE.pdf 와 함께 md2pdf 로 다시 구웠다(2259KB/1726KB, %%EOF 확인, 26/28 페이지). (2) 릴리즈 전 E2E 1회 — 실제 릴리즈 이미지 `seaton:v1.4.6` + PostgreSQL 16 으로 좌석·로그인·관리 11개 파일 47건 전부 통과했고, 새 `seat-detail.spec.ts` 2건도 여기 포함된다.
- 검증: go vet/go test(app 11.3s) 무출력·통과, gofmt -l 무출력, `npm test` vitest 106건, `npm run build`(tsc -b + vite build) 통과, `bash scripts/release-image.sh 1.4.6` 로 이미지 빌드 + `gzip -t` 통과(README 의 로컬 릴리즈 검증 절차 그대로). 버전 일치는 문서가 아니라 실행물로 확인했다 — 띄운 컨테이너의 `/api/v1/version` 이 `{"version":"1.4.6","commit":"694debfda5e4"}` 를 돌려줬다.
- 자산은 만들지 않았다(`assets: []`). `.github/workflows/release.yml` 이 `v*.*.*` 태그 푸시에 반응해 이미지를 빌드하고 `SeatOn-v1.4.6.tar.gz` 를 붙이며 `gh release create --generate-notes --title "SeatOn v1.4.6"` 까지 스스로 한다. 그래서 `github_release` 도 false 다 — 사람이나 러너가 따로 Release 를 만들면 워크플로와 충돌한다. 로컬 검증용으로 만든 tar.gz 는 지웠다(gitignore 대상이기도 함).
- 다음 회차가 알아야 할 것: `scripts/build-docs.py` 를 인자 없이 돌리면 md 가 안 바뀐 EXECUTIVE_REPORT·USER_GROUPS_ANALYSIS 의 html/pdf 까지 갱신된다(템플릿에 figure CSS 가 늘어난 뒤로 두 파일이 뒤처져 있다). 이번엔 이전 릴리즈들과 diff 모양을 맞추려고 되돌렸지만, 언젠가 한 번은 따로 커밋해 정리하는 편이 낫다.
- 남은 우려(닫지 않음): `구역 불일치` 문구가 seats.ts:70 과 SeatMapPage.tsx:177 두 곳에 적혀 있다. 기능 변경이라 릴리즈에서 건드리지 않았다.
