# 회차 노트 2026-09-21-115410-aiportal-py-improve — aiportal-py
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:54] base pinned — main@383d3d0
- [러너 11:54] autonomy release — 

## 정찰 노트
- 선택: 짧은 버퍼 뒤 긴 줄이 source 15,000자 제한을 우회함을 실제 XML→source→RAG로 재현; 외부 서비스·응답 계약 변경 없이 S 범위로 해결 가능해 토큰/종료/재색인 후보보다 우선.
- 전체 테스트 1052 passed(2.46s). 기존 12개 후보 유지 재평가, 문서 후보 유지, 신규 2개 추가; 빈 부칙 id 수정은 운영 적재가 uuid4로 덮어써 효과 미입증으로 rejected.
- 미확인: 운영 XML 발생 빈도·실제 Milvus 실패·뷰어 표시 영향. 요청된 세 스킬/Skill 도구는 찾지 못해 절차를 적용했다고 간주하지 않음.
- 주의: 별표·부칙 두 경로의 source_seq까지 실제 체인으로 검증; byte 제한·markdown 재설계·RAG UUID/청킹·보호 경로·운영 재색인으로 확대하지 말 것.
- [러너 11:58] scout done — 법령 별표·부칙의 긴 한 줄이 source 청크 길이 제한을 우회하는 오류 수정 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 긴 줄 앞의 버퍼를 내보내고 초기화한 뒤 강제 분할하도록 _split_long_record만 최소 수정했다.
- 실제 XML→source→RAG 회귀 15건: Red 8 실패/7 통과 → Green 15 통과; 본문 순서·반복 문자, 별표/부칙 경계와 모든 source_seq 및 짧은 출력 계약 확인.
- 전체 검사 1회 1071 passed(2.74s), 변경 Python pyflakes·git diff --check 통과.
- 확신 없는 곳·검증 못 한 것: 운영 XML 발생 빈도, 실제 Milvus 적재 실패 여부와 뷰어 표시 영향은 미확인.
- 일부러 하지 않은 것: XML 문법·RAG 청킹/UUID·markdown 재설계·조항 분할 확대·기존 색인 재색인·릴리즈는 범위 밖.
- 다음 역할 주의: 제한은 Python 문자 수이며 byte/markdown 크기 보장이 아니다. 테스트는 외부 DB·네트워크 없이 실제 변환 모듈을 사용한다.
- 요청된 technology 스킬 3종과 Skill 도구는 로컬·도구 목록에서 찾지 못해 원문 절차/반환 형식은 미확인이다. 후보 재선정 없이 정찰의 아이디어 목록·판정을 유지했다.
- [러너 12:01] brief accepted — 채택 — 현재 코드에서 별표·부칙 모두 지정 오류가 재현되어 분할 분기만 최소 수정하고 실제 변환 체인으로 연결을 검
- [러너 12:01] verify passed — 검증 1개 통과 (auto)

## 비평 노트
- approve / low, security·legal 차단 없음. 변경 3파일·커밋·변환/적재 배선·범위·되돌리기 검토, 실제 도입 결함 미발견.
- 전체 1071 passed 및 diff --check 통과; 수정 전 builder 메모리 대조에서 신규 8 실패/7 통과로 회귀 검출 확인.
- 운영 XML 빈도·Milvus 적재·뷰어 렌더링 미검증. 문자 제한은 byte/markdown 보장 아님; 긴 청크의 개행·markdown 전체 계약은 테스트 범위 밖. 기존 색인 자동 변경 없음.
- 요청된 세 부서 스킬과 Skill 도구는 검색에서 찾지 못해 원문 절차 미적용; 프롬프트 기준으로 검토 완료.
- [러너 12:03] review approved — 리뷰 승인 (risk=low)
- [러너 12:03] pr created — https://github.com/hkjang/aiportal-py/pull/22
- [러너 12:03] ci passed — 검사 없음 — 정책으로 허용
- [러너 12:03] merge done — 4af5b2f
- [러너 12:04] release missing — 릴리즈 결과 없음/손상: missing
