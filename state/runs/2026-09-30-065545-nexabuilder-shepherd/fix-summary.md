# fix-summary — nexabuilder PR #43 (커밋 없음)

- **고칠 결함이 없다.** 브랜치 HEAD 에서 CI 두 명령을 그대로 돌려 둘 다 초록: `sh ./gradlew --no-daemon test` → BUILD SUCCESSFUL 5m53s, 156클래스 **615건 / 실패0 / 에러0 / skip0**; `sh ./gradlew --no-daemon bootJar -x test` → BUILD SUCCESSFUL. 신규 `BulkListSoftDeleteIntegrationTest` 6/6 통과.
- **테스트가 대상을 실제로 잡는다**(되돌림 검증): 컨트롤러만 `origin/master` 판으로 되돌리면 `6 tests completed, 4 failed`(휴지통2+SQL2, 회귀가드2는 초록), 복원하면 6/6 — 작업 트리 clean.
- **CI 실패는 이 변경 탓이 아니라 인프라 차단이다.** 실패 잡은 **3초**(21:42:33→36Z)만에 끝났다 — 같은 저장소의 마지막 정상 실행은 **8분27초**(ee7f786). checkout+setup-java 조차 못 도는 시간이라 테스트가 실행된 적이 없고, 체크의 `output.title/summary/text` 는 전부 null 이다. 브랜치는 `.github/`·`build.gradle.kts`·`gradle*` 를 **0줄** 건드린다(전체 diff 2파일 284줄, 프로덕션 1 + 테스트 1).
- **이번에 새로 잡은 증거 — 저장소 공개 여부와 완벽 상관**: 같은 계정·같은 시간대에 PUBLIC(`releasedock`/`ptium`/`muni`/`orbit`/`moyro`/`jupiq`, API 200)은 전부 분 단위로 success(18:52Z·18:56Z·19:05Z), PRIVATE(`nexabuilder`·`vibe-code`, API 404)만 3~9초 failure. Actions 는 공개 저장소는 무료·비공개는 과금이므로 **계정의 비공개 저장소 Actions 과금이 소진/차단**된 것이다(앞 회차의 "계정 전체 차단" 추정을 한 단계 좁힌 것 — 무료 분은 살아 있다).
- 따라서 워크플로도 코드도 손대지 않고 **커밋 없이 종료**한다. 결제가 풀리면 재실행만으로 초록이 될 것이다. 못 본 것: CI 잡의 annotation 2건 본문 — `gh` 미인증 + 비공개 저장소라 이 세션에서도 열 수 없다.
