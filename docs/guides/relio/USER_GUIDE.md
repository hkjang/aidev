# Relio 영업대표 및 팀장 사용자 가이드 (User & Sales Guide)

- **문서 버전**: v1.6.0  
- **작성일자**: 2026년 8월 11일  
- **대상**: 영업대표(Sales Rep), 영업팀장(Sales Manager), Presales, 영업운영팀  
- **문서 개요**: Customer 360, Relationship Map, Strategic Account Plan, Sales Playbook & Exit Criteria, Deal Intelligence, Forecast Waterfall, Dynamic Currency 및 AI Agent MCP 연동 사용 가이드  

---

## 1. 개요 및 워크스페이스 구조 (Workspace Overview)

Relio는 B2B 영업 조직의 성공적인 딜 체결과 매출 성장을 돕기 위해 직관적인 3대 워크스페이스 구조를 제공합니다.

| 영역 | 경로 | 주요 기능 |
|---|---|---|
| **CRM 메인** | `/app/*` | Dashboard, Customer 360, Opportunity, Pipeline, Deal Health, Forecast, 계약 관리 |
| **개인화** | `/me/*` | 개인 Dashboard, 목표/실적, 일일 후속 작업, Personal API/MCP Key, 활동 이력 |
| **관리자 콘솔** | `/admin/*` | Operations Center, Data Quality Center, OIDC, 사용자/조직, 영업 정책, Audit Log |

---

## 2. Customer 360 & Relationship Intelligence (`/app/customers/*`)

### 2.1 Customer 360 View
고객 단일 화면에서 기본 정보, 주요 담당자 목록, 진행 중인 Opportunity, 과거 활동 이력, 계약 및 누적 매출 현황을 통합하여 제공합니다.

```
+-----------------------------------------------------------------------------------+
|                        Customer 360: (주)한국테크놀로지                             |
|                                                                                   |
|  [ 업종: IT/소프트웨어 ]    [ 담당자: 홍길동 팀장 ]    [ 누적매출: ₩ 450,000,000 ]  |
|                                                                                   |
|  +-----------------------------------------------------------------------------+  |
|  |  Relationship Map (고객사 의사결정 관계망)                                  |  |
|  |                                                                             |  |
|  |     [ 김철수 대표이사 ] --- (의사결정자 / Champion)                          |  |
|  |             |                                                               |  |
|  |             +--- [ 이영희 CISO ] --- (기술 검토자 / Supporter)               |  |
|  |             |                                                               |  |
|  |             +--- [ 박민수 구매팀장 ] --- (재무 검토자 / Neutral)            |  |
|  +-----------------------------------------------------------------------------+  |
|                                                                                   |
|  +-----------------------------------------------------------------------------+  |
|  |  Strategic Account Plan (전략 고객 실행 계획)                               |  |
|  |   - 고객 2026 사업 목표: 사내 AI 도입 및 보안 에어갭 인프라 구축              |  |
|  |   - White Space Cross-sell 기회: [CRM: 도입완료], [MCP Server: 탐색중]        |  |
|  +-----------------------------------------------------------------------------+  |
+-----------------------------------------------------------------------------------+
```

### 2.2 Relationship Map (의사결정 관계망)
- **역할 및 지지 성향 시각화**: Decision Maker, Champion, Technical Evaluator, Financial Evaluator 등 역할과 지지 성향(Champion, Supporter, Neutral, Block)을 그래프로 한눈에 파악합니다.
- **영향력 매핑**: 의사결정권자와 내부 영향력 구조를 시각화하여 딜 리스크를 사전에 예방합니다.

### 2.3 Strategic Account Plan & White Space
- **연간 매출 목표 및 전략 수립**: 고객의 연간 목표, 당사 영업 전략, 경쟁사 위협 요소를 수립합니다.
- **White Space Matrix**: 당사 솔루션 제품군 중 고객이 미도입했거나 검토 중인 영역을 탐색하여 Cross-sell/Up-sell 기회를 발굴합니다.

---

## 3. Opportunity Management & Sales Playbook (`/app/opportunities/*`)

### 3.1 Opportunity Team (협업 구성원 관리)
- Owner 외에 Presales, Consultant, Manager, Legal 등 영업에 참여하는 팀원을 등록하고 역할을 할당합니다. (팀원 추가 시 사용자의 기본 Data Scope는 확장되지 않고 안전하게 유지됩니다.)

### 3.2 Sales Playbook & Exit Criteria
- **Stage별 실행 가이드**: 각 영업 Stage(Prospect, Qualification, Proposal, Negotiation)별 가이드라인을 제공합니다.
- **Exit Criteria 단계 제어**:
  - `OFF`: 가이드만 제공.
  - `WARNING`: 필수 제출 서류나 확인 사항 미흡 시 경고 표시.
  - `BLOCK`: Exit Criteria 검증을 통과해야만 다음 Stage로 이동 가능.

### 3.3 Dynamic Currency (다중 통화)
- 해외 거래 건에 대해 **거래 통화 원금**과 **생성 시점 고정 환율**을 보존합니다.
- 파이프라인 및 승인 가이드 집계 시 **KRW 기준 금액**으로 일관되게 변환하여 관리합니다.

---

## 4. Deal Intelligence & Health Score (`/app/deal-intelligence`)

### 4.1 Deal Health Score (0~100점)
영업건의 위험도를 활동 빈도, Stage 체류 일수, Exit Criteria 이행률 등을 바탕으로 0~100점 점수로 산출합니다.

### 4.2 Deal Inspection & Sales Coaching
- **Deal Inspection**: 고위험(Health Score 40점 이하) 딜에 대해 위험 요인(예: 30일 이상 미접촉, Decision Maker 미지정)을 분석해 줍니다.
- **팀장 Sales Coaching**: 영업팀장은 팀원별 고위험 딜 목록과 실행 공백을 확인하여 1:1 영업 코칭 인사이트를 얻을 수 있습니다.

---

## 5. Forecast & Revenue Schedule (`/app/forecast`)

### 5.1 Forecast Snapshot & Waterfall 차트
- **Daily Snapshot**: 매일 자정 자동 저장되는 Snapshot을 비교하여 Commit, Best Case, Pipeline 금액의 변동 원인(신규 딜, 금액 증감, Close Lost, Slippage)을 Waterfall 차트로 분석합니다.
- **Manager Override**: 영업 담당자의 추정치 외에 팀장의 독립적인 판단 금액과 사유를 분리하여 기록합니다.

### 5.2 Revenue Schedule (매출 인식 일정)
- 계약 활성화 시 일시, 월, 분기, 연 단위로 매출 인식 일정이 자동 분할 생성되어 차기 갱신(Renewal) 및 매출 인식을 추적합니다.

---

## 6. 팀장 승인 요청 절차 (Approval Workflow)

- **활성 승인 정책 연동**: 관리자가 승인 정책(Policy)을 활성화한 Entity(Opportunity, Quotation, Contract 등)에 대해서만 승인 요청 버튼 및 메뉴가 노출됩니다.
- **팀장 검토**: 승인 요청 시 사유를 입력하며, 팀장은 승인 또는 반려(반려 사유 입력) 처리를 진행합니다.

---

## 7. AI Agent 연동 및 Personal MCP Key 활용 (`/me/keys`)

### 7.1 Personal API/MCP Key 발급
1. `/me/keys` 메뉴 이동 후 `새 키 발급` 클릭.
2. Scope 선택 (예: `mcp:use`, `opportunity:read` 등).
3. 발급된 `relio_4f30d2a1b7c9_xxxxxxxx` 키를 안전한 곳에 보관.
4. 발급 후에는 목록의 `권한` 버튼에서 Scope와 REST/MCP 채널을 변경할 수 있습니다. Secret은 바뀌지 않고 다음 요청부터 바로 적용됩니다.

### 7.2 Qwen Code 연동

```bash
qwen mcp add --scope user --transport http relio https://relio.company.internal/mcp \
  --header "Authorization: Bearer relio_4f30d2a1b7c9_xxxxxxxx"
```

```json
{
  "mcpServers": {
    "relio-crm": {
      "httpUrl": "https://relio.company.internal/mcp",
      "headers": {
        "Authorization": "Bearer relio_4f30d2a1b7c9_xxxxxxxx"
      }
    }
  }
}
```

Qwen에서 `url`은 구형 SSE 전송을 뜻하므로 반드시 `httpUrl`을 사용합니다.

### 7.3 OpenCode 연동

```bash
opencode mcp add relio --url https://relio.company.internal/mcp \
  --header "Authorization=Bearer relio_4f30d2a1b7c9_xxxxxxxx"
```

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "relio": {
      "type": "remote",
      "url": "https://relio.company.internal/mcp",
      "enabled": true,
      "oauth": false,
      "headers": {
        "Authorization": "Bearer relio_4f30d2a1b7c9_xxxxxxxx"
      }
    }
  }
}
```

Qwen의 `mcpServers` 설정을 OpenCode에 그대로 붙여 넣지 않습니다. OpenCode CLI의 Header는 `KEY=VALUE` 형식입니다.

- AI Agent는 권한에 따라 `explain_deal_risk`, `recommend_next_actions`, `get_account_brief` 등의 MCP 도구를 통해 실시간 영업 분석을 지원합니다.
