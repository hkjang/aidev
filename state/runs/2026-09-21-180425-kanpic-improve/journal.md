# 회차 노트 2026-09-21-180425-kanpic-improve — kanpic
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 18:04] base pinned — main@742871c
- [러너 18:04] autonomy release — 

## 정찰 노트
- 인코딩 검증 누락은 실제 HTTPS→Resolve에서 손상 표 성공을 재현했고, 숫자 공용 픽스처·출처 API보다 직접 효과와 작은 변경 범위가 분명해 선택했다.
- #VALUE!는 기존 parseCSV 오류 코드에 맞춘 설계 판단이며 실제 브라우저/DB까지의 영향은 미확인이다. Go 전체 테스트·vet·build 통과; 웹/DB/E2E 미실행.
- ToUTF8 뒤에서만 검사하고 UTF-16 복구·캐시·WEBSERVICE·SSRF·숫자·XLSX를 건드리지 말 것. 증거 overlay는 회차 폴더에만 있으며 저장소 변경 없음.
- Skill 도구는 없어 headcount 형제 저장소의 요청한 세 SKILL.md를 직접 읽어 적용했다. brief에 대안·추정 근거·예비·검토 지점을 기록했다.
- [러너 18:10] scout done — IMPORTDATA가 업로드처럼 잘못된 UTF-8 입력을 거절한다 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 0a21ce7: parseCSV에서 ToUTF8 직후 utf8.ValidString 검사 3줄 추가. 손상 본문을 표로 내보내지 않고 고정 인코딩 #VALUE!와 0×0/빈 Values 반환.
- 회귀: 실제 HTTPS→Resolve/동일 바이트 업로드로 잘못된 UTF-8(표시 유무)·BOM UTF-32 양쪽 endian 4사례 실패→통과, 검사 제거 시 재실패 확인(encoding-mutation.log).
- 정상 인코딩 4종×내용 3종에서 한국어·이모지·유효 U+FFFD, 좌표별 타입/값/JSON 및 공백·아포스트로피 기존 차이를 검증.
- 검증: 지정 패키지 및 go test ./... 모두 ok(전체는 캐시), go vet/build ./... exit 0, gofmt 출력 없음, release docs ok: v0.251.0, 신원 검사 HEAD 커밋 전후 exit 0. 작업 트리 깨끗함.
- 확신 없는 곳·검증 못 한 것: 실제 브라우저 표시/DB 저장·웹 테스트·DB 통합·E2E는 미실행. 서버 파서만 수정하여 이번 검증 범위에서 제외.
- 일부러 하지 않은 것: 깨진 UTF-16 복구·추측·NUL·UTF-32 지원·캐시·숫자/trim·TLS 정책은 기존 계약 유지. 다음 역할도 오류 캐시 수명 변경을 끼워 넣지 말 것.
- Skill 도구가 없어 headcount/plugins/technology/skills의 completion-verification, systematic-debugging, test-driven-development 원문을 직접 읽어 적용. 테스트 작성 중 ByteOrder 인터페이스 컴파일 오류는 AppendByteOrder로 수정 후 의도한 실패를 확인했으며 mutation 출력의 UTF-8 디코딩 오류는 backslashreplace로 해결해 재검증함.
- [러너 18:14] brief accepted — 채택 — 현재 코드와 재현 결과가 과제서와 일치했으며 디코더·숫자·캐시·운영 TLS 정책을 바꾸지 않고 지정한 두 파일�
- [러너 18:14] verify passed — 검증 7개 통과 (auto)
