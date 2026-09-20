# 회차 노트 2026-09-20-085350-Vendra-improve — Vendra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 08:53] base pinned — main@ba89c29
- [러너 08:53] autonomy release — 

## 정찰 노트
- 고른 이유: 가이드 표류 가드는 세 회차(09-11·09-12·09-14)가 매번 손으로 한 대조를 테스트로 옮기는 일이라 S·위험 1 이고 보호 경로(auth/migrations/workflows/web)를 하나도 건드리지 않는다. 이관·통화·status 는 M 에 스코프·정책 결정이 붙고, MCP OAuth·메일 후속은 그 코드가 main 에 없어(2026-09-20 ls 확인) 불가능.
- 확신 없는 곳: 현재 트리에서 그림 30장·환경 변수 4개가 실제로 양방향 일치하는지는 손으로 세지 않았다(09-14 기록만 믿음) — 구현자가 첫 실행에서 확인하고, 어긋나면 가이드를 최소로 고친다. 환경 변수 표 구간을 자르는 규칙(4열 「필수」 표)은 제안일 뿐 구현자가 더 안전한 기준을 골라도 된다.
- 조심할 것: 「효과 없는 변경」 반려를 피하려면 그림 이름 바꾸기·표 행 지우기로 테스트가 실제로 실패하는 것을 보고 그 출력을 노트에 남길 것. 테스트 문자열에 12자 이상 리터럴을 password/secret 옆에 두지 말 것(gate.py secrets). 프로필은 1일 전 것이 아직 맞아(017 이 둘인 점만 추가 — db.go 가 파일명 전체를 키로 써 무해) 다시 쓰지 않았다.
- [러너 08:56] scout done — 가이드 표류 가드 — 두 가이드가 참조하는 그림 전수와 관리자 가이드의 환경 변수 표를 코드·파일 시스

## 구현 노트
- 무엇·왜: `internal/httpapi/guide_docs_test.go` 신설(commit b676548) — 두 가이드의 그림 참조 ↔ `docs/images/guide` 양방향, ADMIN_GUIDE 환경 변수 표 ↔ `config.go` 의 `os.Getenv` 양방향 + 표 이름을 하나씩 비운 환경에서 실제 `config.Load()` 가 거부하며 이름을 말하는지. 세 회차가 손으로 하던 대조를 테스트로. 다른 파일은 손대지 않았다(현재 트리: 그림 31장·변수 4개 일치, 과제서의 30장은 admin-tracking.png 가 빠진 셈).
- 깨뜨려 보기 결과(각각 실패 확인 후 되돌림): (a) login.png→signin.png: `USER_GUIDE.md shows images/guide/login.png, which does not exist` + `holds 1 picture(s) that neither guide shows: signin.png`; (b) ENCRYPTION_KEY 행 삭제: `the server reads ENCRYPTION_KEY, but the admin guide's environment variable table does not name it`; (c) config.go 에 Getenv("VENDRA_NEW_THING") 추가: 같은 문장으로 그 이름; (d) 표에 VENDRA_STALE 행 추가: `names VENDRA_STALE, but the server never reads it` + Load 검사도 실패.
- 확신 없는 곳: 표 구간을 「필수 헤더 셀이 있는 첫 4열 표」로 자른다 — 1장 구성 요소 표는 3열이라 안 걸리지만, 앞쪽에 4열·필수 열 표가 새로 생기면 그 표를 읽어 오탐한다(그때는 이름이 대문자 코드 형식이 아니면 빈 집합 → `proves nothing` 으로 실패하므로 조용히 통과하진 않음). Getenv 스캔은 config.go 한 파일만 — 다른 비테스트 패키지에 Getenv 가 생기면 못 본다(과제서 범위대로).
- 일부러 안 한 것: 가이드·PDF 수정(표류 없어 불필요), 환경 변수 이름 리터럴(비밀정보 검사 회피 겸 스캔으로만), `docs/images/` 의 logo·favicon 대조(가이드가 참조 안 함).
- 다음 역할 주의: 두 테스트는 DSN 불필요, 0.05s. 전체 초록은 docker postgres:16-alpine 세 DSN 으로 돌려 확인했다(httpapi 22.7s). gate.py secrets clean. `t.Setenv` 를 쓰므로 이 테스트는 `t.Parallel` 과 못 섞인다 — 파일의 다른 테스트도 parallel 아님.
- [러너 09:00] brief accepted — 채택 — 과제서의 근거(문서 가드 셋이 같은 방식으로 존재, DSN 불필요, 보호 경로 무관)가 코드와 그대로 맞았고, 그림 �
- [러너 09:00] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인: diff 는 guide_docs_test.go 한 파일뿐. 두 테스트를 실행하고 직접 깨뜨림(그림 이름 변경+고아 파일, config.go 오류 문구에서 ENCRYPTION_KEY 이름 제거, 표에 선택 행 추가) — 셋 다 이름을 대며 실패. gofmt·vet 깨끗, ./internal/... ./cmd/... 초록(DSN 없이 — 통합은 구현 노트의 세 DSN 실행을 믿음), 패키지에 t.Parallel 없음.
- 못 본 것: 실제 DB 통합 테스트는 이 세션에서 안 돌림. 그림·표의 「의미」 일치(캡처가 최신 화면인지)는 이 테스트 범위 밖이고 검토하지 않음.
- 남는 우려(승인): Load 루프는 주석과 달리 필수 셀을 안 보고 표의 모든 행을 건다. 선택 변수를 표에 넣는 회차는 line 127 이 ENCRYPTION_KEY 형식 오류를 그 이름 탓으로 보고하니 루프를 필수 셀로 걸러야 한다. 채움값 x*10 은 유효 키가 아니라 「전부 있으면 기동」 은 증명 안 함.
- 릴리즈 노트: 사용자 영향 없음(테스트만). config.go 에 Getenv 를 추가하거나 docs/images/guide 를 바꾸면 이제 CI 가 잡는다는 점만 다음 회차가 알면 됨.
- [러너 09:02] review approved — 리뷰 승인 (risk=low)
- [러너 09:02] pr created — https://github.com/hkjang/Vendra/pull/127
- [러너 09:04] ci passed — 검사 2개 모두 success
- [러너 09:04] merge done — b676548
- [러너 09:07] release published — v0.7.56
- [러너 09:08] assets verified — v0.7.56 자산 1개 (이전 v0.7.55: 1)
