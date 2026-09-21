# PR 처리기 노트 2026-09-21-204926-muni-shepherd — muni PR #22
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-21-195440-muni-improve)
# 회차 노트 2026-09-21-195440-muni-improve — muni
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:54] base pinned — main@1547faa
- [러너 19:54] autonomy release — 

## 정찰 노트
- DOCX 제목 중복은 실제 writer/reader와 신규 import 분기의 근거가 연결되고 기존 제거 헬퍼로 해결 가능하여, 보호 경로 e2e·메일 캠페인 충돌·서버 캡처보다 작고 확실하다.
- HEAD 1547faa에 지난 HTML/ZIP 개선이 없어도 재수행하지 않는다. 실제 DOCX HTTP 실패 재현은 미확인; 구현자가 writer 출력으로 먼저 증명해야 한다.
- go test ./...·placeholder 통과, DSN 미설정으로 live 검증 없음. docxImport 자체·끼워 넣기·HTML 정책을 바꾸지 말고 전용 DB에서 추가 테스트 SKIP 여부 확인.
- 확대 검색으로 headcount/plugins의 요청된 세 SKILL.md를 찾아 적용했다(전용 Skill 도구는 없음). 과제서에 대안·추정 범위/예비·검증 체크포인트 보완; 기존 12개 후보와 새 후보 2개·기록 승계 2개를 기록했다.
- [러너 19:59] scout done — DOCX 신규 가져오기에서 확정 제목과 같은 첫 H1만 본문에서 제거하기 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 계획 1~4 완료, 커밋 7d406c5. markdown→titleInBody 및 DOCX 활성화로 확정 제목과 같은 첫 H1만 제거; 사용자 가이드 갱신.
- 재현: 실제 writer 파일의 첫 블록 heading·검색/최초 revision에 회의록 중복. docx-red.log → docx-green.log, 활성화만 되돌린 docx-revert.log에서도 같은 실패.
- 검증: 전용 PostgreSQL 16의 go test -count=1 -v ./internal/httpapi SKIP 0(httpapi-live.log), MUNI_TEST_DSN go test ./...(go-test-all.log), go vet ./..., gofmt -l internal/httpapi, placeholder·diff 검사 통과.
- 단위 2개·HTTP 4개 추가: JSON 본문 보존, 제목만 있는 문서, 다른 폼 제목/H2/뒤쪽·연속 H1/정규화/절단/끼워 넣기 및 최초 revision 확인; 기존 Markdown/넘겨받기·저수준 파서도 통과.
- 확신 없는 곳·검증 못 한 것: 브라우저·프런트 빌드 및 Word 외부 프로그램은 실행하지 않음(서버 가져오기 변경만 검증).
- 일부러 하지 않은 것: HTML/PDF 정책, DOCX reader/writer, auth/migrations/workflows, sealer 인프라, 릴리즈 변경은 범위 밖.
- 다음 역할: live 테스트는 전용 MUNI_TEST_DSN 필수; 사용한 일회성 DB 컨테이너는 검증 후 제거. 첫 writer/DB 성공으로 환경 예비시간 불필요, 범위·추정 확대 없음.
- [러너 20:03] brief accepted — 채택 — 현재 코드와 실제 writer의 인증 HTTP 실패가 과제서의 원인을 확인하여 지정한 최소 수정을 적용했습니다.
- [러너 20:04] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- reject / security: main...HEAD의 handoff 연결 오류에서 claim 전체 URL 로그 노출을 합성 토큰으로 재현. 수리는 internal/handoff/handoff.go:275 및 internal/httpapi/handoff.go:181부터 확인.
- 로컬 main=1045e13, 회차 base=1547faa 불일치: 차단은 누적 a8b253b 결함이며 이번 DOCX 수정 자체에서는 결함 없음.
- 실제 DOCX writer·제목 제거·검색/최초 revision·보존 경계와 red/green/revert 로그 확인; handoff/httpapi/docx 테스트 통과, live는 DSN 미설정으로 SKIP.
- DB·브라우저·프런트 빌드·Word 독립 검증 못 함. handoff 유휴 시 만료 스냅샷 보존 우려는 운영 후속 사항; 코드와 기존 webui 수정은 건드리지 않음.
- [러너 20:06] review rejected — 리뷰 거절: internal/handoff/handoff.go:275 [P1][security·차단] client.Do 실패의 *url.Error를 그대로 포함하여 claim이 붙은 전체 URL을 반환하고 internal/httpapi/handoff.go:18
- [러너 20:06] review blocked — 검토 부서 차단 소견(security) — 수리·중재 없이 운영자의 위험 수용(risk-accepted) 필요
- [러너 20:06] pr created — https://github.com/hkjang/muni/pull/22

## 심사 노트
- origin/main=1547faa..HEAD=7d406c5의 6파일과 요청한 세 스킬을 확인; DOCX 자체 추가 결함 없음.
- 전용 PostgreSQL 16에서 httpapi/handoff/docx 테스트 통과(HTTP SKIP 0); 실제 writer·검색·최초 revision·인가 거부 확인.
- 실제 인증 /handoff→기본 HTTP transport 연결 실패→WARN 로그에서 합성 claim 노출 재현; 기존 security 차단 존속으로 reject/human, 운영자 risk-accepted 판단 필요.
- 브라우저·프런트 빌드·외부 Word·배포 환경 미검증; 저장소 코드는 수정하지 않음.
