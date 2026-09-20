# PR 처리기 노트 2026-09-21-034357-muni-shepherd — muni PR #21
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-21-030418-muni-improve)
# 회차 노트 2026-09-21-030418-muni-improve — muni
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:04] base pinned — main@1547faa
- [러너 03:04] autonomy release — 

## 정찰 노트
- ZIP 안내 파일 목록.md가 문서 이름 예약을 우회하여 루트 동명 문서와 충돌한다. 본문 손실 위험과 작은 수정 범위 때문에 DOCX 제목 중복·보호 CI 후보보다 우선했다.
- HEAD 1547faa에는 HTML 개선 36dccc6 및 mail 캠페인이 없다. 이전 완료 기록을 현재 코드로 오인하지 말 것; 프로필도 이를 바로잡았다.
- 실제 HTTP ZIP 재현은 미확인이고 소스 경로로 판단했다. 구현자는 PostgreSQL live 테스트를 먼저 실패시킨 뒤 예약 처리·ZIP 전체 항목/본문/안내 참조를 검증해야 한다.
- go test ./...·placeholder 검사 통과, DSN 미설정으로 live 검증은 미수행. auth/migrations/workflows·가져오기·스트리밍 정책은 건드리지 않는다.
- [러너 03:09] scout done — 워크스페이스 ZIP의 목록.md와 동명 문서가 충돌하지 않게 하기 (가치 4 / 위험 1 / 작업량 S)

## 구현 노트
- 안내 경로 목록.md를 지역 상수로 정의하고 문서 순회 전 used에 예약하여 본문과 안내 파일을 모두 보존했다. 프로덕션 변경은 3줄 추가/2줄 교체뿐이다.
- TestWorkspaceExportReservesManifestName: 실제 PostgreSQL 16 + 등록 라우트 + 관리자 로그인으로 default/md/html/txt ZIP의 중복, 본문 일대일 대응, 안내 참조, 하위 폴더/HTML/TXT 이름을 검증한다. 생성한 문서/폴더만 cleanup한다.
- red: 수정 전 md가 duplicate ZIP entry: "목록.md"로 실패. green 후 수정만 되돌려 default/md 재실패·html/txt 통과 확인, 복원 후 4개 모두 PASS(0.140s).
- 전용 컨테이너에 MUNI_TEST_DSN을 설정한 go test ./... 통과(httpapi 7.222s). go vet ./...·gofmt -l 두 대상 파일·placeholder·git diff --check 모두 종료 0. brief 1~3 done, ideas 기존 항목 유지/선택 done, ledger 작성. 커밋 9ded7a4, 전용 DB 컨테이너 제거 완료.
- 확신 없는 곳·검증 못 한 것: 외부 ZIP 소비자의 고정 문서명 의존은 미확인. 브라우저/프런트 빌드는 이번 서버 내부 변경의 검증 범위에서 제외했다.
- 스트리밍·건수 제한·ACL·폴더 계산·인증·스키마·가져오기·릴리즈는 범위 밖이라 변경하지 않았다.
- 다음 역할 주의: MUNI_TEST_DSN 없으면 이 live 테스트는 SKIP한다. 전용 DB에서 실행하며 같은 제목 두 본문의 접미사 순서는 단정하지 않는다.
- [러너 03:13] brief accepted — 채택 — 코드에서 안내 파일만 이름 중복 추적을 우회했고 실제 HTTP ZIP에서도 동일 결함이 재현되어 지정한 최소 수정을 
- [러너 03:14] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- reject / security: handoff 연결 실패 로그에 원문 claim 노출 재현. 수리는 internal/httpapi/handoff.go:181 및 internal/handoff/handoff.go:275부터 확인.
- 실제 main은 1045e13으로 pinned 1547faa와 다르며, 차단은 이전 handoff 변경에 있음. ZIP 예약 및 테스트 단언 자체에서는 결함 미발견.
- go test ./...·관련 프런트 20개·diff check 통과. DSN 없어 live 테스트 SKIP; 실제 브라우저·PDF·외부 서비스 통합 미검증.
- 지정 부서 스킬/Skill 도구 미발견으로 적용 못 함. 기존 webui 변경 보존; 마이그레이션과 만료 본문은 코드 revert만으로 제거되지 않음.
- [러너 03:16] review rejected — 리뷰 거절: internal/httpapi/handoff.go:181 [security] handoff.Fetch의 연결 실패 오류를 그대로 기록해 bearer claim이 로그에 노출됩니다. internal/handoff/handoff.go:275가 cli
- [러너 03:16] review blocked — 검토 부서 차단 소견(security) — 수리·중재 없이 운영자의 위험 수용(risk-accepted) 필요
- [러너 03:16] pr created — https://github.com/hkjang/muni/pull/21

## 심사 노트
- origin/main 1547faa → HEAD 9ded7a4의 두 파일 확인: ZIP 예약·중복 방지·본문/안내 대응에 결함 미발견, 보호 파일·마이그레이션 변경 없음.
- 전용 PostgreSQL 16에서 실제 로그인/등록 HTTP 경로로 새 테스트 네 형식 및 전체 Go 테스트 PASS; vet·placeholder·gofmt·diff 검사 통과.
- 지정 세 부서 스킬/Skill 도구 미발견. 수정 전 실행·브라우저·외부 ZIP 소비자·handoff 로그 런타임 재현은 미수행.
- reject/human, blocking security: handoff.go:181 및 internal/handoff/handoff.go:268–275의 원문 claim 로그 노출 경로가 남음. 기존 차단이므로 운영자의 risk-accepted 판단 필요.
