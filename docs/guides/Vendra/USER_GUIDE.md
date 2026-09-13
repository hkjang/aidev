# Vendra 엔터프라이즈 사용자 가이드 (User Guide & Manual)

- **문서 버전**: v1.0.0-ENTERPRISE  
- **작성일자**: 2026년 8월 13일  
- **대상**: 일반 임직원, 구매/조달 담당자, 사외 공급업체 사용자, AI MCP 클라이언트 사용자  
- **문서 개요**: Supplier 360, RFQ/RFP 구매 수명주기, 동적 스코어카드, 사외 포털 격리, Personal Key 발급 및 Streamable MCP 연동 매뉴얼  

---

## 1. 개요 및 비즈니스 워크플로우 (Procurement Workflow)

Vendra는 공급업체 심사, 계약, 발주, 납품, 품질, 평가 및 리스크 관제를 단일 패키지로 제공하는 오프라인 공급망 인텔리전스 플랫폼입니다.

---

## 2. Supplier 360 & 구매 수명주기 관리

- **Supplier 360 뷰**: 공급업체 검색 시 기본 정보, 담당자 연락처, 계약 상태, 발주 내역, 납품 하자율, 정기 평가 점수 및 Risk 레벨을 한눈에 조회합니다.
- **구매 수명주기(RFQ/RFP/PO)**: 구매 요청 승인 ➔ 견적 요청(RFQ) ➔ 계약 체결 ➔ 발주서(PO) 발행 ➔ 입고 검수 및 Invoice 발행을 통합 처리합니다.

---

## 3. 동적 스코어카드 & 사외 포털 보안 격리

### 3.1 동적 스코어카드(Dynamic Scorecards)
- 관리자가 정의한 평가 항목 및 가중치(납기 40%, 품질 40%, 단가 20%)에 따라 정기 스코어 점수 및 Tier A/B/C 등급이 자동 계산됩니다.

### 3.2 사외 공급업체 포털 격리
- 사외 공급업체 사용자 계정은 서버 단에서 `supplier_id` 데이터 스코프가 강제 적용되어 타사 계약 및 자사 이외의 정보에 접근할 수 없습니다.

---

## 4. Personal API / MCP Key 발급 및 AI 연동

1. 프로필 메뉴 ➔ **`/me/keys` (개인 API/MCP 키)** 이동.
2. **[신규 Personal Key 발급]** 클릭 ➔ `vnd_7f9c8d11a2b3c4d5_xxxxxxxx` 형식 키 생성.
3. Claude Desktop 또는 Cursor 설정 파일에 MCP 서버를 등록하여 자연어로 공급업체 분석:

```json
{
  "mcpServers": {
    "vendra": {
      "command": "curl",
      "args": [
        "-X", "POST",
        "-H", "Authorization: Bearer vnd_7f9c8d11a2b3c4d5_xxxxxxxx",
        "https://vendra.internal/mcp"
      ]
    }
  }
}
```

### 제공되는 핵심 Read-Only MCP Tools 목록
1. `vendra_search_suppliers`: 등록 공급업체 목록 및 Tier 등급 검색
2. `vendra_compare_suppliers`: 공급업체 간 납기/품질/가격 스코어카드 비교
3. `vendra_get_supplier_risk`: 금융/납기/품질 리스크 평가 보고서 조회
4. `vendra_analyze_spend`: 카테고리별 지출 집중도 및 예산 집행률 분석
