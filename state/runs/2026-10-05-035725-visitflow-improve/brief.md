- 과제: 승인 메모 창에서 취소해도 방문이 승인되는 문제 수정 (가치 4 / 위험 1 / 작업량 S)
- 왜: `web/src/pages/ApprovalsPage.tsx:decide()`는 `window.prompt(...) ?? ""`로 취소 결과 null을 빈 메모로 바꿔 승인 POST까지 실행한다. 취소 의도를 보존하면 사용자가 철회한 조작으로 방문 상태가 SCHEDULED가 되고 QR·알림이 생성되는 일을 막는다.
- 수용 기준:
  1) `/approvals` 목록 행의 승인 버튼 → 네이티브 메모 prompt 취소(`dialog.dismiss()`, 사용자 Cancel/Esc에 해당) 후 해당 방문의 `/approve` POST가 0건이다. 실제 API 상세 조회와 화면 재진입에서도 PENDING_APPROVAL이고 다시 승인할 수 있다.
  2) 같은 prompt에서 빈 문자열을 확인(`dialog.accept("")`)하면 승인 POST 1건, 본문 `{reason:""}`, 응답 204, 서버 상태 SCHEDULED이며 승인 대기 행이 사라진다. null과 빈 문자열을 함께 falsy 검사로 막으면 실패다.
  3) 다른 승인 대기 방문에서 메모를 입력하고 확인하면 그 메모가 실제 POST 및 GET 상세의 `visit.approvalReason`에 보존된다.
  4) 목록 반려 prompt의 취소 및 공백 사유 확인은 기존대로 `/reject` POST 0건이다. 유효한 사유를 확인하면 204와 REJECTED, 상세 사유가 일치한다. 상세 `검토` 다이얼로그의 동작과 서버 정책은 변경하지 않는다.
  5) 영구 Playwright 회귀 테스트는 실제 로그인·서버·PostgreSQL·배포용 UI 번들을 사용한다. 수정 전에는 취소 시 요청/상태 단정이 실패하고 수정 후에는 1~4가 통과해야 한다. 성공 응답을 route.fulfill하거나 window.prompt를 가짜 함수로 바꾼 테스트는 증거로 삼지 않는다.
- 건드릴 파일:
  - `web/src/pages/ApprovalsPage.tsx:decide`(27행 부근) — prompt 반환값을 그대로 받고 `reason === null`이면 `setBusy`, `setError`, `postJSON`, `load` 전에 즉시 반환. 기존 `!approve && !reason.trim()` 검사 및 승인 빈 메모 허용은 유지. 인라인 핸들러를 재구성하거나 새 공용 추상화를 만들 필요 없다.
  - `web/e2e/visit-flow.spec.ts:login` 재사용, 승인 대기 픽스처 헬퍼와 위 취소/확인 회귀 스펙 추가. 프로덕션 변경은 1개 파일, 전체 기본 범위는 2개 파일이다.
- 검증 명령:
  - 저장소 루트에서 `bash scripts/local-e2e.sh` — npm 설치·UI 빌드·실제 서버/새 DB·브라우저 실행까지 담당하며 Playwright 종료 코드를 전달한다. 인자는 지원하지 않으므로 `--grep`를 붙이지 않는다. 수정 전 새 스펙 실패를 먼저 기록하고 수정 후 같은 명령으로 전체 스펙을 확인한다.
  - `cd web && npm run lint && npm test && npm run build` — e2e 스크립트가 설치한 의존성 사용. lint는 tsc이며 e2e 파일은 타입 검사 범위 밖이라는 한계가 있다.
  - `go test ./... -count=1`, `git diff --check`, `git status --short` — 서버 변경이 없는 회차의 기본 확인. DB 통합 PASS라고 쓰려면 별도로 유효한 VISITFLOW_TEST_DSN과 CREATE DATABASE 권한을 제공해야 한다.
- 위험과 피할 것:
  - auth/session·migrations·workflows·서버 승인 정책·전역 설정·API 헬퍼를 수정하지 않는다. QR·알림 경로는 승인 POST의 부작용을 설명하기 위해 읽은 것이며 수정 대상이 아니다.
  - `if (!reason) return`은 빈 메모 승인 계약을 깨므로 금지. 메모의 공백 제거·새 유효성 검사도 이번 범위 밖이다.
  - 이미 기각된 VisitsPage/LobbyPage 요청 티켓 접근을 섞지 않는다. 승인 중 동시 클릭 제어·상세 검토 폼 개선도 분리한다.
  - `createVisit()`는 pass URL을 필수로 단정하므로 승인 대기 픽스처에 그대로 쓰면 실패한다. 승인 목록은 requestNo를 렌더링하지 않으므로 `openVisitDetail()`의 VF 번호 행 선택자를 복사하지 않는다.
  - 호스트 포트 고정/기존 DB 재사용 금지. local-e2e의 새 DB와 자동 포트를 사용한다. webdist 복원은 스크립트의 임시 백업에 맡기고 `git checkout --`로 미커밋 작업을 잃지 않는다.
- 차선 후보: 비상 대피 명단에 최초 조회 실패를 빈 명단(0명)과 구별해 표시 — `web/src/pages/RosterPage.tsx:readCachedRoster/load/render`. 캐시도 응답도 없을 때 현재 `총 0명`·`체류 중인 방문자가 없습니다`를 출력하는 경로는 소스로 확인했고, 실브라우저 실패 재현은 미확인이다. 1순위가 실제로 이미 해결됐거나 적용 불가일 때만 선택한다.

구현에 필요한 확인된 계약
- HEAD `4d982f9` / v2.8.15. `ApprovalsPage.tsx`의 목록 `decide()`만 취소 null을 없앤다. `decideFromReview()`는 별도의 폼이며 이번 변경 대상이 아니다.
- `internal/app/visits.go:approvalAction`(1024~1124행): POST `/api/v1/visits/{id}/approve`는 빈 Reason도 허용, PENDING_APPROVAL을 SCHEDULED로 갱신하고 QR·알림을 생성한 뒤 204를 반환한다. `/reject`도 동일 함수의 분기다. 상세 API는 `visit.approvalReason`을 제공한다(검증한 기존 단정은 integration_test.go 942~950행).
- 승인 대기 픽스처는 `internal/app/integration_test.go:TestVisitTypeChecklistIsEnforced`(406행 부근)의 실제 API 사례를 그대로 따른다. 로그인 뒤 브라우저의 `page.evaluate()`에서 `/api/v1/auth/me`의 csrfToken, `/api/v1/reference-data`의 sites[0].id, `/api/v1/admin/visit-types`의 items 중 code=CONTRACTOR인 id를 읽는다. POST `/api/v1/visits`에 siteId, visitTypeId, checklist={nda:true,safetyBriefing:true}, 미래 startAt(+30분)/endAt(+2시간), 유일한 purpose, 이름·유일한 phone·company·consent=true인 visitors 한 명을 보낸다. 201 및 status=PENDING_APPROVAL을 반드시 확인하고 id를 보관한다. 원시 JSON 객체는 실제 API 요청 데이터이며 가짜 서버/전송 객체가 아니다.
- 목록에서는 `page.getByRole("row").filter({hasText: uniquePurpose})`로 정확히 1행을 고른 뒤 그 행의 exact 승인/반려 버튼을 사용한다. native dialog 리스너는 클릭 전에 `page.once("dialog", ...)`로 등록한다. `/approve`·`/reject` 요청만 추적해 조회 트래픽과 분리하고 실제 GET `/api/v1/visits/{id}`로 상태를 확인한다.
- 기존 `createVisit`, `openVisitDetail`, 알림 재발송 스펙의 인증된 fetch 및 company-policy 스펙의 실제 API 시딩이 사용 가능한 선례다. CONTRACTOR 선택은 전역 승인 설정을 변경하지 않아 다른 테스트에 정책을 남기지 않는다.

실행 순서·체크포인트 (모두 구현자 자동 확인, 사람 승인 대기 없음)
1. [pending] 위 헬퍼와 회귀 스펙 추가, 프로덕션 수정 전 `bash scripts/local-e2e.sh`로 취소가 승인 요청을 보낸다는 실패 기록 확보. 픽스처/선택자 실패라면 제품 실패라고 적지 말고 먼저 스펙을 바로잡는다.
2. [pending] `decide()`에서 null만 즉시 반환하도록 수정. `cd web && npm run lint && npm test` 통과가 다음 단계 조건이다.
3. [pending] 동일 `bash scripts/local-e2e.sh` 및 나머지 검증 명령 실행. 수용 기준과 실패 전후를 기록하고 webdist 복원·의도한 파일만 변경됐는지 확인한다. 계약이 다르면 과제서를 갱신하고 범위를 늘리지 않는다.

접근 비교와 추정
- 권고: 기존 prompt 유지 + null 조기 반환. 취소/빈 확인을 명시적으로 구별하며 프로덕션 1파일로 끝난다.
- 대안: 목록 버튼을 기존 `검토` 다이얼로그로 통일하면 native prompt를 없앨 수 있지만 빠른 승인 동선과 접근성 검증 범위가 변한다. 장래 UX 통일 과제로는 타당하나 이번 회차에는 비용이 크다.
- 보류: 변경하지 않고 안내만 추가하는 방법은 사용자의 취소 의도를 복구하지 못한다. 즉시 조치가 어려울 때 검토 다이얼로그를 쓰는 운영 우회만 가능하다.
- 추정 방식은 bottom-up: 픽스처·수정 전 재현 10~15분 + null 가드 2~3분 + 경계 스펙/전체 검증 10~17분 = 기본 22~35분. 알려진 변동(브라우저 설치·선택자 수정)에 예비 5~10분을 별도로 두어 총 27~45분, 신뢰 중간의 계획 범위이며 통계적 보장이나 실측 소요시간은 아니다. 관리 예비는 배정하지 않는다. 지난 알림 재발송 회차가 동일한 1화면+실서버 e2e 형식으로 완료된 것은 유사성 근거지만 분 단위 실측은 없어 독립적인 정량 추정으로 쓰지 않았다.
- 핵심 가정: local-e2e에 필요한 Docker/이미지/Chrome을 사용할 수 있고 CONTRACTOR 시드가 현재 Go 테스트와 일치한다. 실패하면 인프라 전면 수리로 확대하지 말고 원인과 미검증 항목을 남긴다.

정찰 검증과 한계
- 실행 완료: `go test ./... -count=1` PASS(internal/app 0.140s), `bash -n scripts/local-e2e.sh` PASS, `git diff --check` PASS, 작업 트리 clean.
- VISITFLOW_TEST_DSN 미설정이므로 DB 통합 테스트는 SKIP. web/node_modules가 없어 이번 정찰은 npm/브라우저를 실행하지 않았다. 취소→POST는 소스의 명백한 경로이며 실제 브라우저에서의 재현·새 스펙 실행 시간은 미확인이다.
- 요청한 세 스킬은 Skill 도구가 제공되지 않아 로컬 headcount의 SKILL.md를 직접 읽어 적용했다: pmo/estimating-and-contingency, technology/implementation-planning, technology/solution-exploration. PMO references/sources.md도 읽었으며 외부 문헌의 수치나 확률은 사용하지 않았다.
