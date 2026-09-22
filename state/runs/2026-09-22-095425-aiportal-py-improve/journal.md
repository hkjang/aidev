# 회차 노트 2026-09-22-095425-aiportal-py-improve — aiportal-py
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:54] base pinned — main@04daffd
- [러너 09:54] autonomy release — 

## 정찰 노트
- 긴 별표·부칙 Markdown 들여쓰기 누락을 실제 XML→source→RAG 체인으로 재현해 선택; 외부 계약 미확인인 토큰·Milvus 후보보다 위험이 낮고 문서 정정보다 출력 개선이 직접적이다.
- 직전 길이 제한 수정은 완료로 유지하고 Markdown 생성 두 분기만 대상으로 한다. content/분할/seq/RAG와 auth·migrations·CI는 변경 금지.
- 운영 화면·발생 빈도·Milvus 적재는 미확인; 전체 1071개 및 법령 23개 통과. 요청한 세 스킬/Skill 도구는 검색에서 발견 못해 원문 준수 여부 미확인.
- 첫/후속 청크 둘 다 검증하고 짧은 출력 보존; AST·대역 대신 실제 변환 객체 사용. 30분 구현·검증 + 15분 여유로 산정.
- [러너 09:59] scout done — 긴 법령 별표·부칙의 분할 Markdown에도 들여쓰기 변환 적용 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 8a11ce7: 긴 별표·부칙의 첫/후속 Markdown에 기존 들여쓰기 변환 재사용. content와 분할 알고리즘은 유지했다.
- 실제 XML→source→RAG 회귀: 수정 전 들여쓰기 2 실패/25 통과, 수정 후 27 통과; 전체 1075 passed(2.56s), 지정 pyflakes·diff 검사 통과.
- 2/3칸·탭, 일반 줄·인라인 공백·기존 entity, 짧은 출력, 정확한 분할 경계·메타데이터·모든 RAG 연결을 검사했다. assets/indentation-baseline.json과 비교하여 Markdown 외 source/seq_map/RAG 전체 동일 확인.
- 확신 없는 곳·검증 못 한 것: 운영 뷰어·Milvus 적재·운영 발생 빈도. technology 스킬 3종/Skill 도구를 찾지 못해 원문 절차·반환 형식 준수는 미확인(다른 네임스페이스 유사 스킬은 존재).
- 일부러 하지 않은 것: 제목 중복·길이 제한 재정의·일반 조문 확장·UUID·외부 I/O 수정은 별도 계약이므로 제외.
- 다음 역할 주의: 테스트는 외부 DB 없이 실제 순수 변환 모듈을 실행한다. 기존 색인은 자동 갱신되지 않으며 push·릴리즈는 수행하지 않았다.
- [러너 10:02] brief accepted — 채택 — 현 코드에서 긴 별표·부칙의 들여쓰기 누락이 재현되어 지정 Markdown 생성 지점만 수정했다.
- [러너 10:02] verify passed — 검증 1개 통과 (auto)

## 비평 노트
- approve / low / blocking 없음: 8a11ce7의 세 변경 파일·별표/부칙 호출 경로·범위·되돌리기 가능성을 확인했고 실제 결함을 찾지 못했다.
- 전체 1075 통과, 지정 pyflakes·diff 검사 통과; main builder를 메모리에 로드한 역검증은 신규 들여쓰기 2 실패/25 통과로 결함 검출 확인.
- 운영 뷰어·Milvus 적재·발생 빈도는 미검증; 기존 색인에는 별도 재색인 필요. 인증·개인정보·의존성·외부 상태 변경 없음.
- 요청된 세 부서 스킬/Skill 도구를 찾지 못해 원문 준수 미확인; 사용자 지정 기준으로 검토. 저장소 코드 수정 없음.
- [러너 10:04] review approved — 리뷰 승인 (risk=low)
- [러너 10:04] pr created — https://github.com/hkjang/aiportal-py/pull/23
- [러너 10:04] ci passed — 검사 없음 — 정책으로 허용
- [러너 10:04] merge done — 8a11ce7
- [러너 10:04] release missing — 릴리즈 결과 없음/손상: missing
