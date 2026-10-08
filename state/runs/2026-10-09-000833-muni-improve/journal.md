# 회차 노트 2026-10-09-000833-muni-improve — muni
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:08] base pinned — main@9d781ef
- [러너 00:08] autonomy release — 

## 정찰 노트
- 임의 ZIP 파일/폴더 충돌 후보에서 고정 목록.md 충돌만 분리했다. 프로덕션 1파일의 기존 예약 로직으로 해결 가능해 Windows 문자 처리·e2e·중복 폴더 API 계약보다 위험이 낮다.
- 실제 코드의 빈 claimed·고정 목록.md·허용 폴더명을 확인했고 합성 ZIP Linux 추출 오류를 재현했다. 실제 HTTP 회귀 및 Windows/macOS 추출은 미확인이므로 구현자가 live 실패부터 확인한다.
- 선택 테스트 PASS 7/SKIP 2, placeholder 통과. 전용 DB와 자기 ID cleanup을 사용하고 map 순서·기존 문서 예약·휴지통 분리를 보존한다.
- 세 요청 스킬은 전용 Skill 도구가 없어 로컬 SKILL.md를 직접 읽어 적용했다. 10/08 HWPX 수정은 pinned base에 없지만 이미 수행된 과제라 재선택하지 않았다.
- [러너 00:14] scout done — 워크스페이스 ZIP의 루트 목록.md 폴더가 고정 안내 파일과 충돌하지 않게 하기 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- a212314: folderPaths의 claimed 초기값에 안내 파일 경로를 예약했다. 프로덕션 1파일·테스트 1파일이며 DB 폴더명과 기존 SQL 순서는 그대로다.
- 실제 API 폴더 + GET ZIP 18조합에서 수정 전 FAIL → 수정 후 PASS → 예약 한 줄 원복 시 새 테스트만 FAIL로 원인을 확인했다. 상세 출력은 test-red.log/test-green.log/test-revert.log에 있다.
- 최종 검증: 전체 Go 15패키지 ok, HTTP API PASS 248/SKIP 0/FAIL 0(Chromium 포함), vet·gofmt·placeholder·diff·Go 빌드 모두 exit 0; 프런트 타입 검사와 297테스트 통과.
- 확신 없는 곳·검증 못 한 것: 실제 Windows/macOS 압축 해제는 미검증이다. 이번에는 실제 응답 ZIP의 대소문자 접기·경로 접두사·본문·안내 대응으로 증명했다.
- 일부러 하지 않은 것: 임의 파일/폴더 충돌과 가상 휴지통 충돌은 별건으로 유지했다. make build·버전 갱신·릴리즈·원격 조작은 수행하지 않았다.
- 다음 역할 주의: live 테스트에는 전용 DB의 MUNI_TEST_DSN이 필수이며 공용 DB 사용 금지. 새 워크스페이스와 자기 ID cleanup을 등록했고 검증용 컨테이너는 종료·삭제했다.
- 결과 기록: ledger-entry.md 1개 항목과 ideas.json의 기존 12항목 유지·선택 항목 done 갱신. 빌드 바이너리는 삭제했고 저장소 산출물은 커밋하지 않았다.
- [러너 00:21] brief accepted — 채택 — claimed의 빈 초기값과 API 허용 입력을 실제 라우트 실패로 확인해 지정한 최소 수정 및 수용 기준을 충족했다.
- [러너 00:21] verify passed — 검증 7개 통과 (auto)
