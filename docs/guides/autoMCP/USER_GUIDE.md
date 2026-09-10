# AutoMCP 사용자 가이드

## 목차
1. [시작하기](#시작하기)
2. [DB 연결 관리](#db-연결-관리)
3. [스키마 분석](#스키마-분석)
4. [MCP 생성](#mcp-생성)
5. [배포 관리](#배포-관리)
6. [모니터링](#모니터링)
7. [고급 기능](#고급-기능)

---

## 시작하기

### 설치

```bash
# 저장소 클론
git clone <repository-url>
cd autoMCP

# 의존성 설치
pnpm install

# 환경 설정
cp apps/api/.env.example apps/api/.env.local
cp apps/web/.env.example apps/web/.env.local
```

### 데이터베이스 설정

```bash
# PostgreSQL 시작 (Docker)
docker-compose -f docker/docker-compose.yml up -d postgres

# Prisma 설정
cd apps/api
npx prisma generate
npx prisma db push
```

### 서버 시작

```bash
# 개발 모드
pnpm dev

# 또는 Docker로 전체 실행
docker-compose -f docker/docker-compose.yml up -d
```

접속 URL:
- **Web Console**: http://localhost:3000
- **API**: http://localhost:3001
- **Swagger Docs**: http://localhost:3001/api/docs

---

## DB 연결 관리

### 새 연결 추가

1. 사이드바에서 **Connections** 클릭
2. **Add Connection** 버튼 클릭
3. 정보 입력:
   - Connection Name: 식별용 이름
   - Database Type: PostgreSQL, MySQL, Oracle, MSSQL
   - Host, Port, Database, Username, Password
4. **Create Connection** 클릭

### 연결 테스트

연결 목록에서 **Test** 버튼을 클릭하여 연결 상태를 확인합니다.

- ✅ Connected: 정상 연결
- ❌ Error: 연결 실패 (설정 확인 필요)

---

## 스키마 분석

### 자동 분석 실행

1. 연결 상세 페이지로 이동
2. **Analyze Schema** 버튼 클릭
3. 분석 결과:
   - 테이블 목록
   - 컬럼 정보 (타입, 제약조건)
   - Primary Key / Foreign Key
   - 인덱스, 뷰, 프로시저

### ER 다이어그램

연결 상세 페이지에서 **View Schema** 링크를 클릭하면 ER 다이어그램을 확인할 수 있습니다.

- 드래그: 테이블 이동
- 스크롤: 확대/축소
- 클릭: 테이블 상세 보기

---

## MCP 생성

### 자동 생성

1. 스키마 분석 완료 후 **Generate MCP** 클릭
2. 자동 생성되는 항목:
   - **Tools**: 테이블별 조회/검색/집계 도구
   - **Resources**: schema.json, entities.json 등
   - **Prompts**: 시스템 프롬프트, SQL 가이드

### Tool 유형

| Tool Type | 용도 |
|-----------|------|
| query | 기본 SELECT 조회 |
| search | 조건 기반 검색 |
| aggregate | COUNT, SUM 등 집계 |
| join | FK 관계 기반 조인 |
| analysis | 날짜/금액 분석 |

### MCP 수정

MCP Configs 상세 페이지에서 탭별로 내용을 확인하고 수정할 수 있습니다:
- **Tools**: 도구 설명, SQL 템플릿
- **Resources**: 리소스 내용
- **Prompts**: 프롬프트 템플릿
- **Security**: 보안 규칙

---

## 배포 관리

### MCP 서버 배포

1. MCP Configs 목록에서 **Deploy** 클릭
2. 자동으로 포트가 할당됨 (3100~3199)
3. Health Endpoint로 상태 확인 가능

### 배포 관리

- **Stop**: 실행 중인 서버 중지
- **Restart**: 서버 재시작
- **Logs**: 실행 로그 확인

---

## 모니터링

### 대시보드

**Monitoring** 메뉴에서 실시간 시스템 상태를 확인:

- 연결 수 / 상태
- 배포된 MCP 서버 수
- 평균 응답 시간
- 최근 활동 로그

### 감사 로그

**Audit Logs** 메뉴에서 모든 API 호출 기록 확인:

- 액션 유형별 필터링
- 성공/실패 여부
- 실행 시간

---

## 고급 기능

### LLM 연동

환경 변수로 LLM 프로바이더 설정:

```bash
# Ollama (로컬)
LLM_PROVIDER=ollama
OLLAMA_BASE_URL=http://localhost:11434
OLLAMA_MODEL=llama3.2

# OpenAI
LLM_PROVIDER=openai
OPENAI_API_KEY=sk-...

# vLLM
LLM_PROVIDER=vllm
VLLM_BASE_URL=http://localhost:8000/v1
```

### 스케줄 분석

매일 새벽 2시에 자동으로 스키마 재분석이 실행됩니다.
변경 사항이 감지되면 알림을 받을 수 있습니다.

### Multi-DB 쿼리

여러 DB를 연결하여 통합 쿼리 실행:

```
POST /api/multi-db/federated-query
{
  "query": "SELECT * FROM customers",
  "connections": ["conn-1", "conn-2"],
  "mergeStrategy": "union"
}
```

### 접근 제어

테이블/컬럼 단위로 접근 권한 설정:

```
POST /api/connections/{id}/permissions
{
  "type": "DENY",
  "level": "COLUMN",
  "tableName": "users",
  "columnName": "password",
  "action": "READ"
}
```

---

## 문제 해결

### 연결 실패

1. 호스트/포트 확인
2. 방화벽 확인
3. 사용자 권한 확인

### MCP 생성 실패

1. 스키마 분석이 완료되었는지 확인
2. 테이블이 1개 이상 있는지 확인

### 배포 실패

1. 포트 사용 가능 여부 확인
2. Docker 실행 상태 확인

---

## 지원

문제가 발생하면 GitHub Issues에 보고해 주세요.
