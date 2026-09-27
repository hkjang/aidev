# 수리 요약 — seaton PR #37

- 지적은 맞았다. 코드로 확인: 내보내기 열에 조직코드가 없고(`employeeExport.ts`), 파서는 조직코드→`OrganizationExternalID`·조직명→`OrganizationName` 으로 읽으며(`employees.go:170`), `saveEmployee:109-120` 은 external 이 비면 `import:<조직명>` 으로 `ON CONFLICT(external_id)` 를 시도한다. `migrations.sql:14-22` 에 `organizations.name` UNIQUE 가 없으므로 같은 이름의 새 조직이 생기고, 조직명을 비우면 `organization_id=NULL` 로 소속이 지워진다(employees.go:121).
- 고친 길은 (가)다: `docs/USER_GUIDE.md:149`(와 재생성한 `USER_GUIDE.html:180`)에서 "파일을 고쳐 그대로 다시 올릴 수 있습니다"를 거두고, 이 파일은 보고·확인용이며 조직을 고칠 때는 조직코드가 있는 **직원 양식**을 쓰라고 바꿨다. 일어나는 피해(중복 조직·팀 색·구역 불일치·소속 삭제)를 그 자리에 명시했다. 프로덕션 동작은 그대로다(주석만 수정).
- (나)를 고르지 않은 이유: 내보내기에 조직코드를 더해도 화면·시드에서 만든 external_id NULL 조직은 여전히 코드가 비어 같은 중복 경로를 탄다. 진짜 안전한 왕복은 `saveEmployee` 가 이름만으로 조직을 새로 만들지 않게 바꿔야 하는데 서버 동작 변경이고 DB 테스트 하네스가 없어 이 수리 범위를 넘는다.
- 검증(로컬, 이 커밋 상태): `npm run lint` 통과, `npm test` 142 passed(10 파일, 새 11건 포함), `npm run build` 성공, `gofmt -l .` 무출력, `go vet ./...`·`go test ./...` 통과, `python3 scripts/build-docs.py USER_GUIDE` 재생성분이 커밋과 동일. E2E 는 실서버·PostgreSQL 이 없어 못 돌렸다(문서·주석만 바뀌어 spec 영향 없음).
