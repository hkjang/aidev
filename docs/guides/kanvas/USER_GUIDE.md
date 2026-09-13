# Kanvas 엔터프라이즈 사용자 가이드 (User Guide & Employee Manual)

- **문서 버전**: v0.1.0-ENTERPRISE  
- **작성일자**: 2026년 8월 9일  
- **대상**: 일반 문서 작성자, 지식 관리자, SRE 엔지니어, AI MCP 클라이언트 사용자  
- **문서 개요**: Wiki Space/Page 생성, 구조화 JSON 에디터, 버전 히스토리 복원, ACL 권한 설정, REST API 및 8개 MCP 활용 매뉴얼  

---

## 1. 개요 및 위키 워크플로우 (Wiki Workflow)

Kanvas는 사내 지식 문서를 Space와 Page 계층으로 작성·관리하고, 불변 버전 히스토리 및 세밀한 ACL 보안을 제공하는 통합 사내 위키 서비스입니다.

---

## 2. Wiki 문맥 작성 및 관리

### 2.1 Space & Page 계층 생성
- **Space 생성**: 좌측 사이드바 `+ New Space` 버튼 클릭 후 고유 Key(예: `DEV`, `HR`, `RND`)와 명칭 지정.
- **Page 계층 트리**: 원하는 Space 내에서 `+ New Page`를 눌러 상위-하위 트리 구조 문서 작성.

### 2.2 버전 히스토리 (Immutable Version History)
- 문서를 수정하고 저장할 때마다 새로운 불변 버전 번호(v1, v2, v3...)가 생성됩니다.
- 우측 상단 `Version History`를 통해 이전 버전의 작성자, 변경 일시를 대조하고 원클릭 **[Rollback]** 복원이 가능합니다.

---

## 3. 세밀한 ACL (Access Control List) 권한 설정

Page 또는 Space 설정의 `Permissions` 탭에서 사용자(User) 및 그룹(Group) 단위로 3가지 권한을 부여합니다.

- **READ**: 문서 읽기 및 검색 가능
- **WRITE**: 문서 작성, 편집 및 댓글 등록 가능
- **ADMIN**: ACL 권한 변경 및 문서 삭제/이동 가능

---

## 4. API / MCP Key 발급 및 AI 에이전트 연동

1. 우측 상단 프로필 메뉴 ➔ **[개인 API/MCP 키]** 이동.
2. **[신규 MCP 키 발급]**을 클릭하여 `knv_...` 키 생성.
3. Claude Desktop 또는 Cursor 설정 파일에 아래 8개 ACL-aware 도구를 등록하여 자연어로 사내 위키 검색/편집:

```json
{
  "mcpServers": {
    "kanvas": {
      "command": "curl",
      "args": [
        "-X", "POST",
        "-H", "Authorization: Bearer knv_7f9c8d11a2b3c4d5",
        "https://kanvas.internal/mcp"
      ]
    }
  }
}
```

### 제공되는 8대 MCP Tools
1. `kanvas_search_pages`: 권한 범위 내 위키 검색
2. `kanvas_get_page`: 특정 페이지 본문 및 메타데이터 조회
3. `kanvas_create_page`: 신규 위키 페이지 작성
4. `kanvas_update_page`: 기존 위키 페이지 수정
5. `kanvas_list_spaces`: 접근 가능한 Space 목록 조회
6. `kanvas_get_page_versions`: 페이지 버전 히스토리 조회
7. `kanvas_add_comment`: 댓글 작성
8. `kanvas_get_space_tree`: Space 내 전체 계층 구조 파싱
