# ai-admin v1.2.37 릴리즈 준비

- 분류: marketing:product-launch의 Tier 3 개선. 전달 수단은 기존 한국어 CHANGELOG와 GitHub 자동 생성 릴리즈 노트입니다.
- 대상: /v1/chat/completions 호출자와 운영자. 최상위 null은 기본 공급자 설정 여부와 무관하게 400 invalid_json이며 upstream 호출은 0회입니다. 지원 답변은 docs/api.md의 단일 JSON 객체 계약을 따릅니다. 정상 객체 처리 계약은 유지됩니다.
- 기준일: 2026-10-08. 날짜보다 버전 일치, 전체 race 테스트, 실제 DB/Keycloak 경로, 웹 테스트·빌드, 문서 참조 검사를 릴리즈 기준으로 우선합니다.
- 범위: e12916a가 포함된 merge 81cdcf0에 릴리즈 메타데이터만 추가합니다. 스키마 변경·추가 플래그·가격 변경은 없습니다.
- 실행 경계: 이 세션은 로컬 커밋·주석 태그와 러너 인계까지만 수행합니다. 원격 푸시·게시·운영 배포·외부 공지는 수행하지 않습니다.
- 빌드/게시: 태그를 받은 기존 release.yml이 package-offline.sh → verify-offline.sh → SHA256SUMS 생성 → GitHub Release 게시를 수행합니다. 외부 러너에 중복 게시나 로컬 자산 업로드를 요청하지 않습니다.
- 배포 후 확인 제안(미실행): 운영 담당 hkjang이 첫 배포 환경에서 null/공백 null의 400·upstream 0회, 정상 객체의 200·upstream 1회를 확인한 후 적용을 확대합니다. 어느 조건이든 1건 실패하거나 nil map panic이 1건 발생하면 확대를 중단하고 직전 v1.2.36 이미지와 기존 운영 롤백 절차를 사용합니다.
- 후속 확인 제안(미실행): 배포 후 24시간 동안 해당 입력으로 인한 500/503·panic 0건 및 정상 요청 회귀 0건을 확인하고 최초 관련 지원 문의를 검토합니다. 실제 사용량·성공률은 이 로컬 세션에서 측정하지 않았습니다.
- 제한: 팀 외 사용자에 의한 최초 사용 점검과 운영 canary는 수행하지 않았습니다. 테스트 환경의 실제 DB/HTTP 경로와 실제 Keycloak 검증을 운영 사용 검증으로 표현하지 않습니다.
- 스킬 호출 도구는 제공되지 않아 로컬 SKILL.md를 직접 읽었습니다. 두 스킬 모두 고정 JSON 반환 형식은 지정하지 않아 사용자가 요구한 release.json 형식을 따릅니다.

적용 스킬:
- /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/marketing/skills/product-launch/SKILL.md
- /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/release-and-deployment/SKILL.md
