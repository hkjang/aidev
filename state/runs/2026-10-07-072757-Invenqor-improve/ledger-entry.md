## 2026-10-07
- 선택: 문서 사이트 배포를 깨뜨린 Liquid 태그 여는 괄호 수정과 CI 가드 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 두 번 연속 실패한 검사는 ci.yml·publish-container.yml 이 아니라 GitHub Pages 의
  `pages-build-deployment` 잡이었다. 실패 로그가 파일·행·원인을 그대로 지목한다 —
  `docs/RELEASE_NOTES_v0.2.45.md:150` 의 `` `WHERE resource_id LIKE '{%'` ``. Jekyll 은 docs/ 의
  Markdown·HTML 본문 전체에 Liquid 를 먼저 돌리므로 코드 스팬 안이라도 `{%` 는 태그 여는 괄호이고,
  닫히지 않아 문서 사이트 배포가 통째로 섰다. 이 배포는 별도 워크플로라 CI 가 게이트하지 않아 같은
  커밋의 CI 는 전부 초록이었다 — 그래서 (1) 진단 질의를 뜻을 바꾸지 않고
  `substr(resource_id, 1, 1) = '{'` (두 방언 공통) 로 고치고 (2) `scripts/check-docs-liquid.sh` 가
  발행 대상 문서의 `{%` 를 잡아 실패하게 하고 ci.yml 의 server 잡에서 기존 PDF 검사 옆에 돌렸다.
  워크플로를 느슨하게 한 곳은 없고 검사만 하나 늘었다. 검증: 가드를 고치기 전에 돌려 실패를 확인하고,
  고친 뒤 통과(exit=0), 다른 파일에 `{%` 를 임시로 넣어 그것도 잡는 것까지 확인. `sh -n` 통과,
  ci.yml 을 PyYAML 로 파싱해 단계가 server 잡에 들어간 것 확인, `gofmt -l .` 빈 출력,
  `go vet ./...`·`go test ./...` 전부 ok(httpapi 28.350s, storage 21.634s).
  이 워크트리는 `core.fileMode=false` 라 `chmod +x` 가 인덱스에 반영되지 않았다 —
  `git update-index --chmod=+x` 로 100755 를 넣었다. 빠뜨렸으면 CI 가 Permission denied 로 떨어졌다.
- 실패 재현: 고치기 전 가드 실행 — `A published document contains a Liquid tag opener, which fails the
  GitHub Pages build:` / `docs/RELEASE_NOTES_v0.2.45.md:150:  `WHERE resource_id LIKE '{%'` 와 ...`
  (exit=1). 이것은 Pages 로그의 `Liquid syntax error (line 150): Tag '{%' was not properly terminated
  ... in RELEASE_NOTES_v0.2.45.md` 와 같은 파일·같은 행이다. Jekyll 자체는 재현하지 못했다 —
  이 환경에 ruby/gem 이 없고 apt 는 암호 없는 sudo 가 불가해 liquid 를 설치할 수 없었다.
- 보류 아이디어:
  - docs/SERVER_INSTALLATION.md:98-99 의 `{{.Id}}` 가 발행 사이트에서 빈 문자열로 렌더된다 (가치 3 / 위험 2 / S): 빌드는 세우지 않아 이번 가드 대상에서 제외. 고치면 PDF 재생성이 따라붙어 구현 회차에 섞지 않는다.
  - docs/index.html 의 `href="USER_GUIDE.md"` 류 링크가 Jekyll 산출물에 없다 (가치 3 / 위험 2 / M): Jekyll 이 .md 를 .html 로 바꾸고 jekyll-relative-links 는 HTML 페이지를 손대지 않아 발행 사이트에서 404 일 가능성 — 먼저 실제 URL 로 확인해야 한다.
  - 관계 삭제를 경로의 부모 assetID 에 속한 관계로 제한 (가치 4 / 위험 2 / M): assets.go:665 가 relationID 만으로 UPDATE. incoming/outgoing 부모 정책 확정이 선행. 네 회차 연속 차선.
  - canonicalUUID 미적용 나머지 id 경로 점검 (mcp.go·users.go·api_keys.go·agents.go) (가치 3 / 위험 2 / M): 먼저 어느 핸들러가 UUID 컬럼에 날것을 넘기는지 세어 파일 한둘로 쪼개야 한다.
  - 트랜잭션 안 internalError 의 SQLite 교착 (가치 4 / 위험 3 / M): 프로덕션 배선으로 오류를 터뜨릴 재현 수단이 여전히 없다. 여덟 회차 연속 미선택.
- 과제서: 채택 — "워크플로 파일과 실패한 단계" 지시를 따라 읽었더니 실패 단계가 저장소 워크플로가
  아닌 Pages 배포였고, 그래서 CI 가 못 잡은 구멍 자체가 고칠 대상이었다. 정찰 프로필은 docs/ 가
  Pages 발행 소스라는 사실을 적고 있지 않다 — 다음 정찰이 보탤 것.
