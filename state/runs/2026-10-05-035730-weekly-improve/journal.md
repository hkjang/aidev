# 회차 노트 2026-10-05-035730-weekly-improve — weekly
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:57] base pinned — main@8ab3712
- [러너 03:57] autonomy release — 

## 정찰 노트
- 선택: 첫 첨부 업로드와 cleanupAttachmentFiles의 경합. 실제 DB 잠금+HTTP에서 201 뒤 404 파일 유실을 재현해, 반복 상태 카드 오류·문서 정정보다 가치가 높고 프로덕션 1파일로 좁힐 수 있다.
- 초안 작성 후 재현 결과로 brief를 덮어씀. probe-result.txt/overlay 진단 시험을 회차 폴더에 보존했고 저장소 코드·커밋은 변경하지 않았다.
- 미확인: 수정 후 성공·전체 시험, 살아 있는 보고서 고아 바이트 보존에 별도 정책이 있는지. 새 후보 동시 개수 상한·손상 이미지 수용은 아직 재현 전이다.
- 구현 주의: 부모 존재 여부 오류 시 삭제 금지; scratch DB·실제 Handler 사용, SHARE 잠금/goroutine 반드시 정리. 기존 보고서 잔여 파일 GC나 업로드 잠금으로 범위를 늘리지 않는다.
- [러너 04:04] scout done — 첨부 정리 작업이 첫 업로드의 저장 중인 이미지를 지우지 않게 한다 (가치 4 / 위험 2 / 작업량 M)

## 구현 노트
- 6dbab0b: cleanupAttachmentFiles가 weekly_reports 부모 부재를 확인한 경우만 디렉터리를 삭제하도록 수정(프로덕션 1파일, 시험 1, 운영 문서 1).
- 실제 HTTP+scratch DB의 첫 INSERT 잠금 대기 중 스윕으로 수정 전 201 뒤 GET 404 재현; 수정 후 available:true·GET 200·원본 바이트 일치. 원래 코드 재복원 시 동일 실패도 확인.
- 새 시험 4개: 첫 업로드 보존, HTTP 보고서 삭제 cascade/디렉터리 정리 및 다른 보고서 보존, 취소 context의 파일 보존, 빈 생존 디렉터리 보존.
- 지정 시험 12개·전체 Go(166.788s)·vet·build·OpenAPI 119경로·paging 10곳·guard 11개·diff·정찰 overlay 모두 통과; 출력은 이 폴더 *tests.txt 및 각 검사 로그 참조.
- 미확인: 보고서 id 운영 중 재사용/수동 복원은 범위 밖. 질의 실패 시험은 취소 context로 검증했고 실제 DB 장애 주입은 하지 않음.
- 의도적 제외: 살아 있는 보고서의 실패 업로드 잔여 파일 GC·공통 잠금·mtime 유예. 잔여 파일은 보고서 삭제까지 보존하는 비용을 운영 문서에 명시. 프런트 변경 없어 npm 미실행, 릴리즈는 후속 역할 담당.
- 다음 역할: DSN 없는 SKIP는 검증 성공 아님. run-check.py가 weekly-test-pg 자격을 자식 환경에만 넣으며 시험은 scratch DB만 잠금; 전역 파일 경로 때문에 t.Parallel 금지. 커밋 후 작업 트리 깨끗함.
- [러너 04:11] brief accepted — 채택 — 현재 코드와 실제 수정 전 실패가 과제서의 파일 선저장/행 후커밋 경합에 일치했고, 부모 부재 기준으로 좁힌 �
- [러너 04:13] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- approve / low / blocking 없음: main...6dbab0b의 3파일과 필수 스킬 3개, 업로드·삭제·유지보수·FK·권한·운영 문서를 확인했고 실제 결함은 찾지 못함.
- 원장의 수정 전 실패가 이번 증상과 일치함을 확인; 실제 scratch PostgreSQL에서 관련 12개(신규 4개 포함) 및 타 사용자 첨부 권한 5개 시험 모두 통과, 코드 변경 없음.
- 남는 우려: 생존 보고서의 실패 업로드 잔여 파일은 보고서 삭제까지 보존됨; 릴리즈 설명과 후속 보존·용량 검토에서 이 비용을 유지할 것.
- 미확인: 실제 DB 장애·ID 재사용/수동 복원·프런트; 전체 Go/build/vet는 기존 로그만 확인. 판정 상세는 review.json.
- [러너 04:16] review approved — 리뷰 승인 (risk=low)
- [러너 04:16] pr created — https://github.com/hkjang/weekly/pull/30
