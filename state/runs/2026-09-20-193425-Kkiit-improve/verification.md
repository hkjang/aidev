# 구현 검증

- 회사 technology:completion-verification / systematic-debugging / test-driven-development 스킬 도구·파일은 찾지 못했다. 로컬 superpowers의 test-driven-development, systematic-debugging, verification-before-completion SKILL.md를 읽어 실패 재현→원인 확인→수정→실제 검증 절차를 적용했다. 회사 스킬의 고유 반환 형식은 미확인.
- RED: 변경 전 embed 번들로 실제 POST→GET→카드를 검사해 `제한 없음: missing 조건 없음 · 모든 상품에 적용; rendered 제한 없음 / talent_publish · 우선순위 100 / 비활성`으로 실패. 이후 카드 요약을 구현했다.
- 추가 RED: 기존 toggle이 정책 조회 응답 전체를 PUT하여 HTTP 400과 `요청 본문을 확인해 주세요.`를 반환했다. HEAD의 spread 전송과 decodeJSON의 DisallowUnknownFields를 확인했다. 요청에 저장 필드(resource_type/name/enabled/priority/conditions/steps)만 보내 해결했으며 서버 파일은 변경하지 않았다.
- GREEN 명령: `KKIIT_APPROVAL_SMOKE_OUTPUT=<회차 폴더>/green node scripts/approval-policies-smoke.mjs` (먼저 프런트 build 필요). 최종 출력은 browser-smoke.log, API 저장→GET/렌더 카드/편집 폼/재저장 GET 증거는 green/approval-evidence.json, 화면은 같은 폴더의 PNG 6장.
- 독립 스모크가 자신의 PostgreSQL 16 컨테이너와 CGO_ENABLED=0 Go 빌드 서버를 띄우고 임의의 루프백 포트로 Chrome을 연결했다. 호스트 8080은 이미 사용 중이어서 go run으로 기존 서버를 건드리지 않았다. 비밀번호·암호화 키는 실행 중 메모리/환경으로만 전달하고 결과에 저장하지 않았다. 컨테이너·Chrome 프로필·임시 바이너리를 정리했다.
- 조건 11건: {}, min_amount=0, min/max 양쪽 경계, 복수 유형/등급, quality_score_below=0 및 75.5, 빈 배열, max_amount=0, 알 수 없는 키, 잘못된 배열, 정상·비정상 혼합. 유효 8건의 폼 값/재저장 조건 일치(빈 배열은 기존 저장 방식대로 {}로 정규화)를 확인했다.
- UI 생성 및 편집으로 최소 금액 0→1000, 품질 75.5→0을 바꿔 GET·카드에 반영됨을 확인. 배열 조건이 활성 전환에도 보존됨을 확인. 삭제, 실제 품질 미산정 상품 2건의 공개 요청→검토 다이얼로그→승인/반려 저장 확인.
- 375px에서 모든 카드의 본문/버튼 사각형 겹침과 가로 넘침이 없음을 검사. Chrome 런타임 예외 0. desktop/edit/mobile/생성편집/승인/반려 캡처를 확인했다.
- `make check` exit 0: gofmt 검사, go test ./cmd/... ./internal/..., go vet, npm ci --ignore-scripts, lint, npm test(10개), production build. 전체 출력 make-check.log. `node --check scripts/approval-policies-smoke.mjs`, `git diff --check` 통과.
- 제한: KKIIT_TEST_DSN을 사용하는 저장소 전체 DB 통합 테스트는 실행하지 않았다. 승인 기능의 실제 DB/API/화면 경로는 위 별도 스모크로 실행했다. Chrome 이외 브라우저는 미검증.
- 산출물 지시 충돌: 과제서에는 dist 포함, 뒤의 절대 규칙에는 빌드 산출물 커밋 금지. 후자를 따라 build-diff-stat.txt에 재생성을 기록하고 dist 변경은 복원·제외했다. 후속 검증/릴리즈는 make build 또는 프런트 build 후 Go 빌드가 필요하며 현재 추적된 옛 dist만 사용하면 새 UI가 나오지 않는다.

- 로컬 커밋: 1783d3e `feat: 승인 정책 카드에 적용 조건을 표시하다` (작성자 hkjang, 트레일러 없음). 커밋 대상은 AdminPage.tsx와 재실행 가능한 스모크 스크립트 두 파일. 커밋 후 작업 트리 깨끗함. push/릴리즈 없음.
