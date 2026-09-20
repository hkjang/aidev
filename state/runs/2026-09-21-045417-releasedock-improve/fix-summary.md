NBSP DSN에서 make test 경고 0회·종료 0이나 DB fixture는 SKIP하는 정확성 결함을 재현했다.
Makefile의 Bash 공백 판정을 Go 보조 명령의 strings.TrimSpace로 교체해 fixture와 일치시켰다.
Go 회귀 테스트 8개 사례와 verify_make.py NBSP 실제 make test 검증을 추가했다.
NBSP/미설정/빈 값/ASCII 공백 make test 성공·WARN 1회·웹 94건 통과; 잘못된 DSN은 WARN 0회·종료 2. 원복 시 NBSP 검증 실패도 확인했다.
유효 DB 연결은 이번에 재검증하지 않았으며, 공백 DSN 실행의 DB 통합 테스트는 의도대로 생략된다.
