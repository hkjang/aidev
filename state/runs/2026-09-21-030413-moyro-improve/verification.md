# 구현 검증

- 요청된 technology:completion-verification, technology:systematic-debugging, technology:test-driven-development는 callable Skill 도구/카탈로그와 로컬 검색에서 찾지 못했다. 원문 절차·반환 형식 적용은 미확인이다.
- 일회용 postgres:16-alpine, localhost 임의 포트(49428), 전용 컨테이너 moyro-order-20260921. 다른 프로젝트 DB/55432는 사용하지 않았다.
- 변경 전: `go test -count=1 ./internal/httpapi -run '^TestSidebarOrder'` 실제 DB 실행 exit 1. 혼합 입력 PUT `[second first second ghost foreign-user foreign-team ""]` vs GET `[fav second channels first dm]` 불일치, 빈 최초 PUT `[]` vs GET 기본 3개 불일치. 빈 입력 되읽기 장애도 기존 코드는 성공 배열과 성공 이벤트를 발행해 실패.
- 변경 후: `go test -race -p 1 -count=1 ./internal/sidebar ./internal/httpapi` 실제 DB 실행 exit 0 (sidebar 9.609s, httpapi 27.337s).
- `go vet ./...` exit 0; `bash scripts/check-source-sizes.sh` exit 0 (early.go 54038/58000); `git diff --check` exit 0.
- 신규 통합 테스트는 실제 sidebar.Service와 PUT/GET 핸들러, 실제 ws.Hub.Run 및 두 대상 Client.Send와 다른 사용자 Client.Send를 사용한다. resolver 대역/이벤트 조립 대역/소스 문자열 검사는 없다.
- 한계: DROP TABLE로 비어 있지 않은 요청의 UPDATE 오류와 빈 요청의 Order 오류를 분리했다. 행 갱신 커밋 직후 SELECT만 실패하도록 트리거/권한을 바꾸는 시나리오는 미실행. 되읽기 실패가 이미 커밋된 UPDATE를 롤백한다고 주장하지 않는다.
- 웹은 변경하지 않아 웹 빌드/브라우저 검증은 미실행. 실제 플러그인 archive가 없는 환경의 archive 통합 테스트 범위는 별도 공백이다.
- 최종 전체 게이트: 같은 실제 PG16 DSN으로 `go test -race -p 1 ./...` exit 0. 일부 DB 비의존 패키지는 Go 캐시 사용; sidebar/httpapi는 실제 실행(sidebar 9.705s, httpapi 26.779s).
- 커밋: bddb463 `fix: return persisted sidebar category order`; 작업 트리 clean. 전용 PostgreSQL 컨테이너 정리 완료.
