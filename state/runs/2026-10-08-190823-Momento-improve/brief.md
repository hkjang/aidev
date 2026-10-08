- 과제: Custom Dimension 등록·갱신 실패를 한국어로 안내한다 (가치 3 / 위험 1 / 작업량 S)
- 왜: `DimensionsAdmin`의 저장 Alert(`web/src/pages/AdminPage.tsx:3122`)은 서버 문장을 그대로 표시하며, 이름·Property key 규칙 위반에는 영문 문장, 저장 500에는 DB 원문을 본문으로 보여 준다. 기존 관리 화면의 안내 패턴을 이 폼 하나에 적용하면 사용자가 고칠 칸과 규칙을 한국어로 이해하고, 원인을 알 수 없는 장애도 과장 없이 전달받는다.
- 수용 기준:
  1) `/admin?section=dimensions`의 「등록 또는 갱신」 실패에서 INVALID_DIMENSION은 「Dimension 이름」·「Property key」 두 라벨과 정확한 규칙(첫 글자는 영문 또는 _, 이후 영문·숫자·_·.·-, 전체 1~128자)을 한국어로 안내한다. 어떤 칸이 틀렸는지 서버 코드만으로 특정하지 않는다. INVALID_SCOPE는 「Scope」, INVALID_DATA_TYPE은 「Data type」의 기존 선택지를 가리킨다. INVALID_PAYLOAD와 UNKNOWN_SITE도 기존 관리 화면 수준으로 안내한다.
  2) DIMENSION_SAVE_FAILED는 중립적인 저장 실패 본문과 기존 AdminNoticeAlert의 detail 캡션으로 서버 원문을 분리한다. 본문에는 SQLSTATE·violates·no rows 같은 원문 표지가 없고, 이름 중복이라는 설명도 없어야 한다. 모르는 코드/일반 Error는 기존 메시지로 되돌리고, 메시지도 없으면 빈 Alert 대신 안내를 남긴다. 성공 시 폼 초기화·쿼리 무효화·편집·삭제·저장 버튼 조건은 그대로다.
  3) 순수 함수 테스트가 여섯 서버 코드, REQUEST_FAILED, 모르는 코드, 일반 Error, null/빈 메시지, 원인이 다른 두 500을 검증한다. 특히 INVALID_DIMENSION을 영문 그대로 반환하는 대조 구현에서는 안내 테스트가 실패해야 한다. 실제 앱에서도 실패 응답을 받아 저장 Alert가 새 함수의 본문·detail을 쓰는지 최소 INVALID_DIMENSION·500·unknown 세 사례로 확인한다(브라우저 검증은 구현자 수행; 정찰은 미실행).
- 건드릴 파일: `web/src/pages/adminErrors.ts:refusal 및 새 describeDimensionError` — 기존 UserErrorNotice·UNREADABLE_PAYLOAD·LOST_REQUEST·PASS_TO_ADMIN을 재사용하는 독립 mapper; `web/src/pages/AdminPage.tsx:DimensionsAdmin(2993~3190), import(80~87), AdminNoticeAlert(3394 부근)` — 저장 Alert 한 곳을 새 DimensionErrorAlert 래퍼로 연결; `web/test/adminErrors.test.mjs:apiError 및 새 차원 오류 테스트` — 기존 테스트를 유지하며 위 계약 증명. 프로덕션 2파일+테스트 1파일이며 신규 의존성은 필요 없다.
- 검증 명령:
  - 빠른 단계 검증: `(cd web && npm test)` — 정찰에서 실제 실행, 259/259 통과(exit 0). Node 22.23.1/npm 10.9.8, web/node_modules 없이도 현재 순수 테스트는 통과했다.
  - 최종 web 게이트: `(cd web && npm ci && npm audit && npm run lint && npm test && npm run build)`.
  - 이전 verify-failed 누락을 반복하지 않도록 CI의 선행 게이트도 실행: `go test -race ./cmd/... ./internal/...`, `go vet ./cmd/... ./internal/...`, `go run golang.org/x/vuln/cmd/govulncheck@latest ./cmd/... ./internal/...`, `(cd sdk && npm ci && npm audit && npm run typecheck && npm test && npm run build)`.
  - 위 명령은 현재 package.json·ci.yml에 존재한다. 정찰에서 최종 게이트·브라우저·DB·Docker는 실행하지 않았다. MOMENTO_TEST_POSTGRES_DSN이 없으면 DB 통합 테스트는 skip되므로 통과와 혼동하지 않는다. 브라우저 하네스는 저장소 기존 명령이 없어 새 npm script를 지어내지 않는다; 구현자가 사용한 실제 실행 명령과 결과를 journal에 남긴다. 마지막 `git diff --check`와 `git status --short`로 변경 범위를 확인한다.
- 위험과 피할 것: 서버·internal/auth·migrations·workflows·잠금 파일·Makefile·다른 Alert·입력 helperText/차단은 범위 밖이다. 상태 코드나 SQLSTATE로 중복을 단정하지 않는다. 서버는 `ON CONFLICT(site_id,name) DO UPDATE`이며 500 원인은 실제 DB로 재현하지 않았다. 기존 mapper 문구와 공통 레이아웃을 재정리하지 않는다. 순수 .ts의 값 import는 .ts 확장자, TSX에서는 기존 확장자 없는 관례를 따른다. prettier를 새 게이트로 추가하지 않는다. 브라우저 사례별 새 탭을 사용해 beforeunload에 막히지 않게 한다. 보안 검사 실패를 무시하거나 검사·의존성을 함께 변경하지 말고 별도 실패로 기록한다.
- 차선 후보: README 개발 명령의 cd 누적으로 make docker가 web에서 실행되는 문제를 고친다 — README.md:108~111의 SDK/web 명령을 각각 서브셸로 감싸 루트 위치를 유지한다. `make -n docker`는 루트에서 해석돼야 하고 원래 블록의 web 위치에서는 타깃이 없다; 실제 이미지 빌드는 이 문서 수정의 수용 기준이 아니다. 1순위 Alert가 이미 바뀌었거나 서버 계약이 달라졌다는 증거가 있을 때만 전환하고 이유를 기록한다.

구현 근거와 순서 (구현 전 상태: 전부 pending)

범위 밖을 먼저 고정한다: 새 필드 유효성 검사, 서버 오류 코드/DB 스키마 수정, 남은 설정·삭제 오류 일괄 번역은 하지 않는다.

1. [pending] `internal/httpapi/segments.go:223~271 saveDimension`을 확인한 뒤 mapper와 테스트를 함께 추가한다. 증명: `(cd web && npm test)`. 실패한 검증은 통과로 적지 않는다. 체크포인트: 자동 테스트 검토, 사람 승인 없음.
2. [pending] 테스트가 통과한 mapper를 DimensionsAdmin 저장 Alert에 연결한다. 증명: web 최종 게이트 및 실제 앱의 위 세 응답 확인. 체크포인트: 본문/캡션·기존 버튼 동작과 diff 범위를 검토, 사람 승인 없음.
3. [pending] Go·SDK 선행 게이트와 변경 목록을 확인하고 실행/미실행·실패를 journal에 적는다. 계획과 실제 코드가 다르면 작업을 넓히지 말고 근거와 수정된 계획을 먼저 남긴다. 체크포인트: 구현자 자체 확인 후 다음 역할에 인계, 사람 승인 없음.

서버 계약 (읽어 확인한 사실)

| 위치 | 코드 | 안내에서 지킬 점 |
|---|---|---|
| segments.go:236 | INVALID_PAYLOAD | 본문 해독 실패, 기존 공통 안내와 detail |
| segments.go:241 | UNKNOWN_SITE | 사이트를 찾지 못함, 새로 고침 안내; 화면에서 실제 발생 조건은 미확인 |
| segments.go:245 | INVALID_DIMENSION | 두 입력 중 실패한 칸을 구분할 정보가 없음 |
| segments.go:249 | INVALID_SCOPE | user/session/event/item; UI는 User/Session/Event/Item (Ecommerce) |
| segments.go:256 | INVALID_DATA_TYPE | string/number/boolean/date; 빈 값은 서버에서 string으로 기본화 |
| segments.go:266 | DIMENSION_SAVE_FAILED | DB 원문을 detail로 보존, 실패 원인은 특정하지 않음 |

정확한 이름 규칙은 서버 영문 요약보다 `internal/segment/segment.go:56 PropertyKeyPattern` (`^[A-Za-z_][A-Za-z0-9_.-]{0,127}$`)이 더 상세하다. saveDimension은 이름만 TrimSpace+ToLower하고 Property key는 그대로 검사한다. 이 과제는 입력을 새로 정규화하지 않는다. `web/src/api/client.ts:APIError/api`가 응답 code/message를 Error로 만들며, 현재 Alert는 그 message를 직접 렌더한다. 이 경로는 코드로 확인했지만 실제 브라우저에서의 변경 전 재현은 미확인이다.

선택과 작업량 근거

- 최소안(선택): 저장 Alert 하나+기존 mapper 확장. 기존 사용자·사이트·보존 정책 회차와 같은 2개 프로덕션 파일 구조라 검토 범위가 작고 현재 서버 계약도 여섯 코드로 한정된다.
- 대안: 입력 단계 helperText까지 추가하면 실패 전 안내가 가능하지만 정규화·경계값 테스트가 추가되므로 별도 pending 후보로 분리한다.
- 대안: 모든 관리 오류를 공통화하면 앞으로 중복은 줄 수 있으나 여러 서버 계약과 기존 문구가 함께 바뀌어 45분 범위를 벗어난다. 현상 유지도 가능하지만 잘못된 입력 때 영문 규칙이 계속 남는다.
- 차선 문서 수정은 더 작지만 개발자 안내만 개선한다. 선택 과제는 현재 사용자가 직접 만나는 실패 안내를 바꾸며, DataTable·CIDR DB 동작 후보보다 의도가 명확하다.
- 가장 큰 가정은 기존 AdminNoticeAlert와 브라우저 확인 방법을 재사용할 수 있다는 점이다. 새 브라우저 테스트 프레임워크를 도입하지 않는다.
- 작업 분해 추정: mapper+테스트 8~10분, UI 연결·실제 응답 확인 8~12분, 게이트·기록 10~13분 = 기본 26~35분. 알려진 변동(설치·하네스 조정)에 contingency 4~10분을 별도 두어 총 30~45분으로 본다. 이 범위의 신뢰도는 중간이며 실측 확률/보장 시간이 아니다. 관리 예비비는 0분(권한·범위 추가 배정 없음); 네트워크 지연이나 새 장애 때문에 범위를 늘리지 않는다.
- 추정법은 작업별 bottom-up이고, 과거 동일한 2파일+테스트 회차와 규모를 대조했다. 과거 실제 소요 시간이 제공되지 않아 유사 과제의 시간 통계는 주장하지 않는다. 의존성 설치 상태·첫 web 게이트 결과가 나오면 재추정한다.
- 적용 스킬: `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`, `technology/skills/implementation-planning/SKILL.md`, `technology/skills/solution-exploration/SKILL.md`(뒤 두 경로도 같은 headcount/plugins 루트). Skill 도구가 없어 로컬 정본으로 읽었고, 추정 스킬의 references/sources.md도 확인했다. 외부 문헌의 수치·견적 보증은 사용하지 않았다.

추가 실측: 차선 후보의 `make -n docker`는 저장소 루트에서 exit 0, web에서 exit 2(`No rule to make target docker`)였다. dry-run만 실행했고 이미지는 빌드하지 않았다. 최종 `git status --short`는 빈 출력이며 저장소 변경이 없다.
