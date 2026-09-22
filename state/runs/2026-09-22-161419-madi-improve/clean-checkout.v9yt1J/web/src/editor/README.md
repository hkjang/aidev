# madi 고급 편집 블록

TipTap 스키마와 Go의 `collaborationRenderExtension`은 같은 Markdown 문법을 사용한다. 단일 Go 서비스는 Yjs XML을 직접 병합하고 Markdown, 블록 ID 메타데이터, 문서 이력, 자동화 outbox를 같은 PostgreSQL 트랜잭션에서 저장한다. 브라우저가 보내는 Markdown 스냅샷을 신뢰하지 않는다.

## 원본 형식

| 기능 | Markdown 원본 |
| --- | --- |
| 콜아웃 | Obsidian/GitHub 스타일 `> [!NOTE] 제목` 및 인용 본문 |
| 접기 | `:::details` 안의 `:::detailsSummary` / `:::detailsContent` |
| 2~3열 | `:::columns {count="2"}` 안의 `:::column` |
| 수식 | `$...$`, `$$` 블록; 구분자와 충돌하는 원문은 `data-madi-math` HTML |
| 다이어그램 | `mermaid` 언어 코드 블록 |
| 각주 | `[^label]`, `[^label]: 내용` |
| 북마크 | `:::bookmark {url="..." title="..." description="..."}` |
| 동기화 문서/블록 | `![[문서 UUID]]`, `![[문서 UUID#^블록 ID]]` |
| 표 | 일반 표는 GFM; 병합/열 너비/다중 문단/정렬은 HTML 표 |

Pandoc 확장은 원문에 그대로 남는다. 해당 확장을 지원하지 않는 외부 Markdown 앱에서는 원문 구문으로 보일 수 있다. Front Matter는 공동 편집 본문과 분리해 정확하게 보존한다. 알 수 없는 HTML 태그·확장·사용자 스타일·주석은 블록 편집기를 초기화하지 않고 원문 모드로 안내한다.

## 동기화 참조 보안

`GET /api/v1/documents/{id}/blocks/{blockID}`는 원본 문서의 현재 상속 ACL과 API 키 워크스페이스 제한을 적용한다. 본문이 REST/원문 편집으로 대체되면 이전 CRDT 블록 참조에는 409를 반환한다. 삭제/권한 회수/갱신 실패 시 브라우저는 캐시된 미리보기를 비운다. 갱신 주기는 2초이며 순환 참조 및 5단계 이상 중첩은 차단한다. 수정은 원본 문서 링크에서 수행한다. 임베드로 원본의 쓰기 권한을 우회하지 않는다.

## 오프라인 및 콘텐츠 안전

KaTeX와 폰트는 서비스 이미지에 포함되고 필요할 때만 로드한다. `trust:false`, 확장/크기 제한을 사용한다. Mermaid는 공용 `DiagramPreview`에서 strict 설정, 외부 URL/HTML/구성 override 차단, SVG 정화와 이미지 격리를 적용한다. 북마크는 명시적으로 입력한 URL/제목/설명만 저장하며 서버나 브라우저가 외부 본문을 수집하지 않는다.

## 검증

`MADI_TEST_POSTGRES_DSN=... go test -race ./internal/server -run 'Test.*Collaboration|TestPostgresEditorBlock|TestAdvancedCollaboration'`

프로덕션 웹 빌드 후 `MADI_BROWSER_EDITOR=1 MADI_TEST_POSTGRES_DSN=... go test ./internal/server -run '^TestBrowserAdvancedEditor$'`는 격리된 PostgreSQL 스키마에서 실제 UI 삽입, 원문 재가져오기, HTML 표 병합/너비/정렬 보존, 로컬 수식/다이어그램, 미지원 원문 보존, 동기화 참조 삭제 감지를 확인한다.
