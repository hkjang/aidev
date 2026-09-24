# PR 처리기 노트 2026-09-23-114400-seaton-shepherd — seaton PR #33
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-23-110442-seaton-improve)
# 회차 노트 2026-09-23-110442-seaton-improve — seaton
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:04] base pinned — main@00becef
- [러너 11:04] autonomy release — 

## 정찰 노트
- 배정 해제를 골랐다: 서버 핸들러(seats.go:323)와 E2E 복구 도우미(helpers.keepingSeats)가 이미 있어 프런트 한 파일 + 테스트로 끝나는데, 지금은 한 사람의 자리를 비울 방법이 화면에 전혀 없고 배정된 좌석은 삭제도 막혀 있다(SeatMapPage.tsx:2331).
- 차선인 '조직/지정 구역 구분'은 두 회차 연속 과제서로 나갔다가 스킬 부재로 구현이 멈춘 건이라 같은 것을 세 번째 1순위로 올리기보다 뒤에 두었다 — 코드에 그대로 남아 있으니 1순위가 막히면 바로 가면 된다.
- 추측으로 적은 것: 해제 후 상세 패널을 유지하는 방법(chooseMap이 setSelected(null)을 하는 것은 확인했으나, 재조회 함수를 새로 만들지 chooseMap 뒤 재선택할지는 구현자 판단). 문서 절 위치는 파일 존재만 확인했고 해당 절은 열어 보지 않았다.
- 조심할 것: 서버는 손댈 필요가 없다(트랜잭션·seat_history·audit까지 이미 함) — seats.go를 고치면 미머지 메일 브랜치와 충돌한다. E2E는 반드시 keepingSeats 안에서, openapi.go/API_AND_MCP.md는 목록 끝줄 삽입을 피할 것.
- 확인 실행: go test ./... 통과(cached). 프런트는 web/node_modules가 없어 미실행. 프로필은 2일 전 것이 지금 코드와 맞아 다시 쓰지 않았다.
- [러너 11:07] scout done — 좌석맵 상세에서 배정 해제 단추 (가치 4 / 위험 2 / 작업량 S)

## 구현 노트
- 좌석 상세에 관리자용 `배정 해제` 단추를 더했다. 서버(`unassignSeat`)는 이미 트랜잭션·seat_history·audit 까지 하고 있어 손대지 않았다. `chooseMap` 은 첫 줄에서 선택을 비우므로 쓰지 않고, 좌석만 다시 읽어 같은 id 로 선택을 갱신하는 `reloadSeats` 를 따로 두었다.
- 과제서에 없던 것 하나: 상세 패널 Paper 에 `component="section" aria-label="좌석 상세"` 를 붙였다. 도면 SVG 의 좌석 라벨과 상세의 직원 이름이 같은 글자라 E2E 로케이터가 strict mode 로 깨졌기 때문이고, 이름 붙은 region 은 화면 낭독기에도 실이득이다. 렌더 결과(글자·색·배치)는 바뀌지 않는다.
- 확신 없는 곳: (1) `reloadSeats` 는 선택한 좌석이 응답에서 사라지면 `setSelected(null)` 로 떨어뜨린다 — 해제 경로에서는 일어나지 않지만 의도한 동작인지 리뷰 볼 것. (2) 검증은 전부 admin(system_admin) 계정으로만 했다. `seat_manager` 가 단추를 보고 일반 직원이 못 보는 것은 `manager` 플래그를 그대로 쓴 코드로만 보장했고 실제 계정으로 돌려보지 않았다.
- 일부러 안 한 것: 서버 변경 없음(미머지 메일 브랜치가 seats.go 를 건드림). `삭제` 의 `disabled` 조건은 그대로 두고 해제 후 상태 갱신으로 자연히 풀리게 했다(E2E 가 이걸 확인한다). USER_GUIDE.pdf 는 굽지 않았다(미머지 브랜치와 이진 충돌) — HTML 만 다시 만들었다.
- 다음 역할이 조심할 것: `web/e2e/seat-unassign.spec.ts` 는 실서버+PostgreSQL 이 있어야 돈다. 배정을 바꾸므로 세 테스트 중 둘은 `keepingSeats` 안에 있고, 밖에서 배정을 건드리면 후속 spec 이 깨진다. red 역검증(단추만 뺀 이미지)에서 DELETE 가 나가지 않아 실패하는 것을 먼저 확인했다.
- 환경 주의: 이 회차 동안 같은 머신의 병렬 세션 vitest 워커가 RAM 28GB 를 써 docker build 의 `npm run build` 가 5초짜리에서 874초로 늘어났다. 검증이 느리면 코드가 아니라 머신 부하를 먼저 의심할 것.
- [러너 11:31] brief accepted — 채택 — 과제서의 근거(호출부 부재, `disabled={Boolean(selected.employeeId)}`, `chooseMap` 의 `setSelected(null)`, `keepingSeats` 복구)가 모�
- [러너 11:31] verify passed — 검증 7개 통과 (auto)
- [러너 11:31] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 11:31] pr created — https://github.com/hkjang/seaton/pull/33

## 심사 노트
- 확인한 것: 단추 노출 조건 manager 가 서버 requireSeatManager(server.go:105)와 일치, 새 경로·권한 확대·마이그레이션 없음, audit 은 식별자만 넘김. E2E 단언을 실제 DOM 과 한 줄씩 대조했고(Chip '배정됨'→'빈 좌석', 삭제 disabled 조건, 배치 편집 토글이 selected 를 안 비움) region 과 단추가 이 PR 에서 처음 생기므로 수정 전 코드에서는 통과할 수 없다.
- 실행한 것: go vet 무출력, go test ./... 통과, build-docs.py USER_GUIDE 재생성이 커밋된 HTML 과 바이트 일치.
- 못 본 것: web/node_modules 부재로 tsc/vitest, 실서버 필요한 Playwright 미실행 — PR CI 가 둘 다 돌린다. seat_manager 실계정 검증도 못 했다(코드 경로는 다른 관리자 기능과 동일).
- 남긴 note: reloadSeats 가 왼쪽 검색 목록·처리필요 배지를 갱신하지 않음(배정 경로도 동일, 새 결함 아님), spec 의 seats() 가 floorMapId 없이 조회(기존 seat-organization.spec 과 같은 패턴), USER_GUIDE.pdf 미갱신, seats.go:343 의 기존 커밋 오류 처리 결함.
- 권고 근거: 결함을 찾지 못했고 서버 무변경·revert 로 완전히 되돌아가므로 approve/merge. 데이터(배정)를 바꾸는 경로가 화면에 새로 열리므로 risk 는 medium.
