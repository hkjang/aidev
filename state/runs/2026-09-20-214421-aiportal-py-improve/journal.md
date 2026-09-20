# 회차 노트 2026-09-20-214421-aiportal-py-improve — aiportal-py
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:44] base pinned — main@8ee250c
- [러너 21:44] autonomy release — 

## 정찰 노트
- 실제 XML→source→RAG에서 행정규칙 가지조문 표기 불일치를 재현해 선택; token·shutdown 후보보다 위험/검증 비용이 작고 문서 정정보다 동작 가치가 큼.
- 1040 tests 통과(2.55s); 실제 변환 모듈 import 성공. 운영 발생 빈도·UI 영향·전체 API import는 미확인. 요청한 세 pmo/technology 스킬은 도구/로컬 검색에서 없었음.
- 파서 표시만 고치고 article_number·UUID·source_seq를 보존; 두 소비자 실제 출력으로 검증하고 운영 재색인·auth·CI·숫자 문법 확대는 피할 것.
- 기존 보류 12개 유지·재평가, 신규 2개 추가 및 A-116 done 기록. 이전 프로필의 테스트 수·삭제 실패 설명이 낡아 갱신.
- [러너 21:48] scout done — 행정규칙 가지조문 계층 경로를 제54조의3 형식으로 바로잡기 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- A-117: 행정규칙 가지조문 계층 표시만 제54조의3으로 수정; 번호 추출·소비자는 유지.
- 합성 XML 실제 factory→source→RAG 8건: Red 4 failed/4 passed → Green 8 passed. UUID·seq·메타데이터·본문 및 장/절 전환 확인.
- 전체 pytest 1052 passed(2.59s), 변경 Python 파일 pyflakes·git diff --check 통과.
- 확신 없는 곳·검증 못 한 것: 운영 발생 빈도·UI 영향·외부 API/DB/Milvus 통합은 미확인.
- 요청한 technology 세 스킬/Skill 도구를 찾지 못함; 다른 네임스페이스 유사 파일은 있지만 지정 스킬 원문 절차·반환 형식을 사용했다고 간주하지 않음.
- 기존 색인 재작성·삭제, 숫자 문법 확대, auth·CI·배포는 범위 밖이라 수행하지 않음.
- 다음 역할: 테스트는 외부 서비스 없이 실행 가능하며 수정은 신규 변환에만 적용됨.
- [러너 21:51] brief accepted — 채택 — 현재 코드에서 동일 오류가 재현되어 지정된 표시 지점만 수정하고 실제 소비자 체인으로 검증했다.
- [러너 21:51] verify passed — 검증 1개 통과 (auto)

## 비평 노트
- 판정: approve, risk low, security/legal 차단 없음. 변경 3개 파일과 파서·소비자·fetch_and_parse 연결을 확인했고 실제 결함을 찾지 못함.
- 전체 1052 tests 통과; main 파서 메모리 대조에서 신규 4건 실패/4건 통과로 회귀 탐지 확인. 변경 Python pyflakes·diff --check 통과.
- 번호·UUID·source_seq 보존과 표시 변경 범위 확인. 저장소 코드 수정 및 외부 상태 변경 없음.
- 남는 제한: 기존 색인 자동 교정 없음. 운영 빈도·UI 영향·외부 API/DB/Milvus 통합·전체 API startup 미검증.
- [러너 21:52] review approved — 리뷰 승인 (risk=low)
- [러너 21:52] pr created — https://github.com/hkjang/aiportal-py/pull/21
