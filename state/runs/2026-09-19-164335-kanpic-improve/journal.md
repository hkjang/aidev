# 회차 노트 2026-09-19-164335-kanpic-improve — kanpic
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:43] base pinned — main@5f4ec38
- [러너 16:43] autonomy release — 

## 구현 노트
- 배정된 '릴리즈 워크플로 두 번 실패' 는 저장소 실패가 아니다: 09-16·09-18 회차의 evidence.json 이 모두 `hold: budget`(러너 하루 예산 상한, base_sha 비어 있음)이고, 09-17 은 v0.249.0 을 정상 릴리즈했다. release.yml 의 docker build → save|gzip → release-notes.sh 를 로컬에서 재현해 모두 통과 — 워크플로·스크립트는 손대지 않았다.
- 남은 예산으로 보류 목록의 가장 작은 것 하나: 일괄 등록 파일 고르기 accept 에 .tsv·.txt 를 더했다(e98ce33, UI 한 속성 + 테스트 1개).
- 확신 없는 곳: 없음. 다만 새 테스트는 jsdom File 에 arrayBuffer 를 손으로 채운다(플랫폼 대역, 우리 코드는 대역 없이 통과) — 실제 브라우저 픽커가 .txt 를 보여 주는 것은 브라우저의 accept 처리에 맡긴다.
- 일부러 하지 않은 것: parseUserCSV LazyQuotes(워크북 가져오기도 엄격해 명단만 느슨하게 하면 새 어긋남), IMPORTDATA 파싱 실패 캐시(LazyQuotes 인 encoding/csv 는 오류를 내지 않아 닿을 수 없는 갈래 — 둘 다 ideas.json 에 rejected 사유), 관리자 가이드(파일 형식을 말한 적 없어 PDF 재생성 불필요).
- 다음 역할이 조심할 것: Go 코드 변경 없음(go test 안 돌림, go build 만). 러너 쪽에 hold: budget 을 error 로 분류하는 문제가 있어 같은 수정 과제가 또 적재될 수 있다 — 저장소에서 고칠 것이 아니다.
- [러너 16:49] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인: diff 는 accept 속성 한 줄 + 테스트 1개뿐. 새 테스트를 main 의 UserImportDialog.tsx 에 대고 돌려 실패(`['.csv','text/csv']`)를 봤고, HEAD 에서 통과·tsc·web 전체 499개 통과를 확인했다. 서버 parseUserCSV 가 delimited.Delimiter 로 탭·세미콜론을 실제로 고르는 것도 읽었다 — 주석과 동작이 맞는다.
- 못 본 것: 실제 브라우저 픽커의 accept 처리(구현자도 밝힌 대로 브라우저 몫). Go 변경 없어 go test 는 안 돌렸다.
- 남는 우려(차단 아님): `.txt`·`text/plain` 이 열리면 고정폭·공백 구분 명단도 골라지는데, 서버는 "머리글 줄에 user_id 열이 있어야 합니다" 로 거절하므로 데이터 손상은 없다. 관리자 가이드 215행은 형식을 말하지 않아 갱신 불필요.
- 릴리즈 노트: 사용자 눈에 보이는 변화는 "일괄 등록 파일 고르기에서 .tsv·.txt 가 보인다" 한 줄. 보안·개인정보 영향 없음(수집 항목·전송 경로·인가 변화 없음).
- [러너 16:50] review approved — 리뷰 승인 (risk=low)
- [러너 16:50] pr created — https://github.com/hkjang/kanpic/pull/26
- [러너 16:58] ci passed — 검사 2개 모두 success
- [러너 16:59] merge done — e98ce33
- [러너 17:10] release published — v0.250.0
- [러너 17:11] assets verified — v0.250.0 자산 2개 (이전 v0.249.0: 2)
