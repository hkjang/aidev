# 회차 노트 2026-09-21-030408-moina-improve — moina
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:04] base pinned — main@9dba4f3
- [러너 03:04] autonomy release — 

## 정찰 노트
- 선택: 큰 multipart 업로드가 413 대신 400으로 분류되는 분기 수정. 캡처·동영상 파싱보다 작고, Makefile -race/테스트 추가만보다 직접 사용자 오류를 해결한다(3/1/S).
- 근거: social.go:uploadMedia·settings cache·OpenAPI 직접 확인; 실제 대용량 요청 재현 및 제안한 무DB 테스트 배선은 미확인이므로 수정 전 실패 확인이 필수.
- 검증: go test -race ./...·make check·make fmt 통과, DB DSN 없어 integration skip. 회사 세 스킬은 도구/로컬 검색에서 찾지 못해 고유 절차 준수 미확인.
- 주의: 본문/파일 한도와 MIME·이름·auth·migrations·workflows 유지. 과거 PATCH 403 문서 복구처럼 middleware 계약을 지우지 말 것; profile의 잘못된 014/메일 기록 상태를 main 기준으로 정정했다.
- [러너 03:08] scout done — 미디어 multipart 본문 한도 초과를 400 대신 413 media_too_large로 분류 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 1bd4408: multipart 파싱 오류 중 *http.MaxBytesError만 413 media_too_large로 분리하고 OpenAPI 설명을 보강했습니다.
- 실제 multipart/httptest 10케이스 작성: 본문 초과(길이 알려짐/-1) 2개가 수정 전 400/413 불일치로 실패, 수정 후 모두 통과했습니다.
- 지정 집중 테스트·go test -race ./...·go vet ./...·make fmt·make check 통과, diff 확인 후 3개 소스 파일만 커밋했습니다.
- 검증 못 한 것: DB DSN unset으로 PostgreSQL integration skip; 프런트·e2e는 무변경이라 미실행. 회사 세 스킬은 도구/로컬 경로에 없어 고유 절차 준수 미확인입니다.
- 한도·파싱 메모리·MIME·파일 이름·저장·인증 정책 및 기존 OpenAPI 응답은 유지했습니다. 차선 과제·릴리즈·push는 수행하지 않았습니다.
- 다음 역할 주의: 새 테스트는 DB 없이 직접 handler를 실행하며 응답은 기존 writeError 계약의 최상위 code/message입니다. 실제 세션·DB 성공 경로는 기존 PostgreSQL 테스트가 담당합니다.
- [러너 03:10] brief accepted — 채택 — 큰 본문에서 400이 반환되는 현상을 수정 전 재현했고 지정한 최소 오류 분리와 검증을 완료했습니다. error.code는 
- [러너 03:11] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- approve / low / blocking 없음: 고정 base 9dba4f3 대비 3개 파일의 오류 분류·문서·인증 경계·임시 파일 정리를 확인, 실제 결함 없음.
- 새 10케이스 race 통과; Go overlay 수정 전 코드에서 본문 초과 2케이스 실패 확인. 저장소 코드 변경 없음.
- DB·실제 세션/CSRF·프런트·e2e 미실행; 요청한 회사 세 스킬 및 Skill 도구가 없어 고유 절차 준수 미확인.
- 릴리즈 주의: 로컬 main=95255f3이라 main...HEAD는 74개 파일; 상세 승인은 회차 고정 base 이후 변경에 한정하며 이전 변경 전체 재심사는 아님.
- [러너 03:12] review approved — 리뷰 승인 (risk=low)
- [러너 03:12] pr created — https://github.com/hkjang/moina/pull/28
- [러너 03:21] ci passed — 검사 2개 모두 success
- [러너 03:21] merge done — 1bd4408
