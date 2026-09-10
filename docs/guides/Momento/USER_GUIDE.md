# Momento 엔터프라이즈 사용자 가이드 (User Guide & Developer Manual)

- **문서 버전**: v0.34.37
- **대상**: 웹/앱 개발자, 데이터 분석가, 서비스 기획자(PO), BI 엔지니어  
- **문서 개요**: Momento JavaScript SDK 연동, 이벤트 트래킹 규칙, 방문자 인사이트·추적·이상 감지·기여도, 집단 비교 분석, 정기 배달과 BI 내보내기 실전 매뉴얼  

---

## 목차

1. [플랫폼 아키텍처 개요](#1-플랫폼-아키텍처-개요)
2. [JavaScript SDK 연동 및 설정 가이드](#2-javascript-sdk-연동-및-설정-가이드) — 설치, 자동 감지 신호와 사이트 검색, CSP 허용, 동의 모드, 식별, 커스텀 이벤트, SPA, 오프라인 큐
3. [고급 분석 기능 사용법](#3-고급-분석-기능-사용법)
   - 무엇을 볼지: 3.1 첫 화면, 3.2 방문자 인사이트, 3.3 방문자 추적, 3.4 이상 감지, 3.5 전환 기여도
   - 누가 다른지: 3.6 행동 기반 Segment, 3.7 Funnel 비교, 3.8 Retention 비교, 3.9 경험 비교
   - 직접 파보기: 3.10 쿼리 빌더 ~ 3.19 Experiment와 Goal
   - 이어서: 3.20 정기 배달 연결, 3.21 분석 쿼리 제한
4. [BI 연동 & 데이터 내보내기](#4-bi-연동--데이터-내보내기-export)
5. [Query Mode와 비용 보호](#5-query-mode와-비용-보호)
6. [개인정보 요청 Workflow](#6-개인정보-요청-workflow)
7. [Console 탐색과 표 활용](#7-console-탐색과-표-활용)

---

## 1. 플랫폼 아키텍처 개요

Momento는 사내 애플리케이션 및 인트라넷 환경에서 발생하는 모든 행동 이벤트를 외부 SaaS로 전송하지 않고 사내 DB에 원시 이벤트(Raw Event) 수준으로 직접 저장·분석하는 온프레미스 플랫폼입니다.

---

## 2. JavaScript SDK 연동 및 설정 가이드

### 2.1 스크립트 비동기 설치

웹 애플리케이션의 `<head>` 영역에 아래 스크립트를 비동기로 삽입합니다.

```html
<!-- Momento JavaScript Tracker SDK -->
<script 
  async 
  src="https://momento.internal/tracker.js" 
  data-site-id="SITE_CORPORATE_001"
  data-environment="prd"
  data-contract-version="1"
  data-mode="full"
  data-debug="false"
></script>
```

| 속성 (Attribute) | 타입 | 필수 여부 | 설명 |
| :--- | :--- | :--- | :--- |
| `data-site-id` | String | **필수** | 관리자 콘솔에서 생성한 사이트 고유 식별 키 |
| `data-environment` | String | 선택 | `dev`, `stg`, `prd` 등 관리자에게 등록한 환경. 기본 `prd` |
| `data-contract-version` | Number | 선택 | 전송 Event Contract version. 기본 `1` |
| `data-mode` | String | 선택 | `full`, `consent-required`, `cookieless`, `disabled` 중 선택 |
| `data-debug` | Boolean | 선택 | `true`이면 브라우저 콘솔에 SDK 진단 로그 출력 |
| `data-collect-element-text` | Boolean | 선택 | 버튼 문구 수집. 개인정보 최소화를 위해 기본값은 `false` |
| `data-auto-rum` | Boolean | 선택 | Core Web Vitals와 Resource Error 자동 수집. 기본 `true` |
| `data-session-timeout` | Number | 선택 | Session 구분 기준(분). 관리 콘솔의 사이트 Session Timeout 설정이 이 속성으로 전달됩니다. 기본 `30` |
| `data-frustration-signals` | Boolean | 선택 | Rage Click, Dead Click, Rapid Back, Form Retry, Error After Click, Slow Interaction 자동 감지. 기본 `true` |
| `data-search-tracking` | Boolean | 선택 | 결과 페이지의 질의 문자열로 사이트 검색 자동 인식. 기본 `true` |
| `data-collect-search-terms` | Boolean | 선택 | 검색어 원문 수집. 개인정보 최소화를 위해 기본값은 `false` |
| `data-search-params` | String | 선택 | 검색어 질의 문자열 이름 추가 지정(쉼표 구분). 기본값은 `q,query,search,searchword,keyword,kwd,term,s` |
| `data-release-version` | String | 선택 | Release Impact 비교용 애플리케이션 릴리스 |
| `data-git-sha` | String | 선택 | 배포 소스 revision |
| `data-endpoint` | String | 선택 | Collector 주소 override. 절대 URL 또는 같은 Origin의 프록시 경로(`/momento`) |

Collector endpoint는 `tracker.js`를 제공한 Origin의 `/collect/v1/events`로 자동 설정됩니다. Page View, SPA History 변경, 클릭, 스크롤, Form, Download, Outbound Link, Error, Heartbeat, LCP/INP/CLS/FCP/TTFB와 Resource Error는 기본 자동 수집됩니다.

### 2.1.1 자동 감지되는 Frustration 신호와 사이트 검색

별도 계측 없이 다음을 감지합니다. 감지 기준은 화면의 신호 설명과 동일합니다.

| Event | 감지 기준 |
| :--- | :--- |
| `rage_click` | 같은 요소를 1초 안에 3번 이상 클릭. 실제 클릭 수를 `clicks`로 전달 |
| `dead_click` | 클릭 가능해 보이는 요소를 눌렀지만 1.2초 동안 DOM 변화, 이동, 스크롤, 포커스 이동, 텍스트 선택이 모두 없음 |
| `rapid_back` | 도착 후 3초 안에 뒤로 이동. 머문 시간을 `dwell_ms`로 전달 |
| `form_retry` | 같은 Form 재제출(`reason=resubmit`) 또는 입력 검증 실패(`reason=validation`) |
| `repeated_search` | 같은 검색어를 2분 안에 재검색 |
| `error_after_click` | 클릭 후 2초 안에 오류 발생. 원인 요소를 함께 전달 |
| `slow_interaction` | 입력 응답이 500ms 초과(INP `poor` 구간) |
| `search` | 결과 페이지 질의 문자열에서 검색 인식. `query_length`, `query_words`, `result_count` 전달 |
| `search_click` | 검색 결과 링크 클릭. 순위를 `position`으로 전달 |
| `search_refine` | 검색어를 좁히거나 넓혀 재검색. 방향을 `direction`으로 전달 |
| `collection_dropped` | 오프라인 큐가 저장 한도(200건)를 넘겨 오래된 이벤트를 버렸습니다. 잃어버린 건수를 `events_dropped`로 전달 |

`collection_dropped`가 보이면 **해당 구간의 수치는 실제보다 낮습니다.** 이전에는 같은 상황에서 아무 흔적도 남지 않아, 숫자가 낮은 이유를 알 수 없었습니다.

브라우저가 오프라인일 때 페이지를 벗어나면 `sendBeacon`은 페이로드를 접수하지만 전달을 보장하지 않으므로, 그 배치는 전송하지 않고 큐에 남겨 다음 페이지 로드에서 보냅니다. 온라인 상태에서 브라우저가 강제 종료된 경우의 손실은 페이지가 관측할 수 없어 막지 못합니다.

한 페이지에서 보고하는 신호 수는 20건으로 제한되어, 렌더 루프에 빠진 화면이 수천 건의 Event로 번지지 않습니다.

정확도를 높이는 선택 계측은 세 가지입니다. 없어도 검색 횟수와 Frustration 신호는 수집되지만, 결과 0건 비율과 클릭된 결과 순위는 페이지가 알려줘야만 알 수 있습니다.

```html
<div data-momento-search-results="12">
  <a href="/doc/1" data-momento-search-position="1">첫 번째 결과</a>
</div>

<!-- 정상 동작하는 위젯이 Dead Click으로 잡히면 제외 -->
<div data-momento-ignore-dead-click>...</div>
```

URL이 바뀌지 않는 검색은 직접 알려줍니다.

```js
analytics.trackSearch(query, results.length);
```

검색어 원문은 사람이 입력한 자유 텍스트이므로 기본적으로 보내지 않습니다. `data-collect-search-terms="true"`를 켜면 공백 정리와 소문자 정규화 후 100자까지 전송하며, 이메일·주민등록번호·휴대전화 번호는 브라우저에서 먼저 제거하고 서버의 PII 정책이 한 번 더 검사합니다.

### 2.1.2 Content-Security-Policy 허용

측정 대상 애플리케이션이 CSP를 사용하면 `tracker.js` 로드와 수집 요청을 명시적으로 허용해야 합니다. 예를 들어 `connect-src 'self' ws: wss:`만 허용된 페이지에서는 브라우저가 `/collect/v1/events` 요청을 차단하고 콘솔에 `Refused to connect ... violates the document's Content Security Policy`를 남깁니다.

```
Content-Security-Policy: script-src 'self' https://momento.internal; connect-src 'self' https://momento.internal
```

CSP를 변경할 수 없는 애플리케이션은 Collector를 같은 Origin으로 프록시하고 `data-endpoint`로 그 경로를 지정하면 `connect-src 'self'`만으로 동작합니다.

```nginx
location /momento/ {
  proxy_pass https://momento.internal/;
  proxy_set_header Host $host;
  proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
  proxy_set_header X-Forwarded-Proto $scheme;
}
```

```html
<script async src="https://momento.internal/tracker.js"
  data-site-id="SITE_CORPORATE_001" data-endpoint="/momento"></script>
```

관리 → 사이트 → SDK 설치 화면의 **CSP 허용** 항목에서 위 정책과 프록시 설정을 복사할 수 있고, **설치 진단** 탭에서 수집 수신 여부와 허용 도메인, 환경 일치, 적재 파이프라인 상태를 서버 기준으로 확인할 수 있습니다. SDK도 CSP 위반을 감지하면 브라우저 콘솔에 필요한 정책을 안내합니다.

각 이벤트에는 `track()` 호출 순간의 URL, 제목, Referrer, Device와 최초 UTM Context가 snapshot으로 저장됩니다. 따라서 1초 배치 전송을 기다리는 동안 SPA Route가 바뀌어도 이전 페이지의 Click이 새 페이지로 잘못 분류되지 않습니다. SDK가 자동 수집하는 URL에서는 Query String과 Fragment를 제거합니다.

### 2.2 Consent와 Cookieless

| 모드 | 동의 전 | Visitor 저장 | 용도 |
| :--- | :--- | :--- | :--- |
| `disabled` | Event 없음 | 없음 | 추적 중지 |
| `consent-required` | Event 없음 | 동의 후에만 저장 | 분석 동의가 필수인 환경 |
| `cookieless` | Event 수집 | 영속 저장 없음 | 익명 분석 |
| `full` | Event 수집 | 영속 Visitor/Session | 허용된 내부 분석 |

`consent-required`는 `localStorage`가 차단된 브라우저에서도 안전하게 수집을 중지합니다. `analytics.consent.grant()`는 저장소가 없어도 현재 Page에서 유효하며, 동의를 기다리는 동안에도 최초 UTM은 유지합니다. `deny()`와 `revoke()`는 대기열·영속 식별자·Offline Queue를 정리합니다.

### 2.3 사용자 및 부서 식별 (`analytics.identify`)

로그인한 사용자의 사내 식별자와 부서/조직 체계 정보를 수집기에 전달합니다.

```javascript
// 사용자 로그인 시 호출
analytics.identify("EMP_2026_9012", {
  department: "Digital Platform Team",
  organization: "R&D Center",
  role: "Senior Architect",
  location: "HQ_Seoul"
});
```

> ⚠️ **보안 수칙 (Security Policy)**:  
> 이메일 주소, 전화번호, 주민등록번호, 카드번호 등 개인식별정보(PII)는 `user_id` 또는 속성으로 전달하지 마십시오. 수집기는 관리자가 지정한 Property key 제거와 URL 정책에 더해 값 기반 PII 탐지·마스킹을 Inbox 저장 전에 수행하지만, 애플리케이션의 최소 수집 책임을 대신하지는 않습니다.

---

### 2.4 커스텀 이벤트 트래킹 (`analytics.track`)

사용자의 특정 비즈니스 액션을 세밀하게 기록합니다.

```javascript
// 서식 제출 이벤트
analytics.track("document_submitted", {
  document_id: "DOC_2026_0808",
  category: "Approval",
  amount: 1500000,
  approval_step: "Final"
});

// 파일 다운로드 이벤트
analytics.track("file_downloaded", {
  file_name: "Q2_Financial_Report.pdf",
  file_size_mb: 14.2,
  download_source: "Intranet_Notice"
});
```

세션 동안 유지할 로그인 상태나 업무 흐름은 Event Property와 구분해 설정할 수 있습니다. 값은 각 Event 발생 시점에 snapshot되므로 배치 전송이나 Raw Event 재집계 후에도 Session Scope Dimension이 동일합니다.

```javascript
analytics.setSessionProperties({
  login_status: "authenticated",
  workflow: "approval"
});
```

---

### 2.5 SPA (Single Page Application) 라우트 변경 추적

React, Vue, Next.js 등 SPA 라우터 전환 시 페이지뷰를 수동 또는 자동 수집합니다.

```javascript
// History API 또는 라우터 변경 감지 시
router.on('routeChangeComplete', () => {
  analytics.track("page_view");
});
```

---

### 2.6 오프라인 큐 & Beacon 재전송 메커니즘

- **Beacon API**: 브라우저 닫힘 또는 페이지 이동 시 이벤트 손실 방지를 위해 `navigator.sendBeacon`을 이용합니다.
- **Offline Queue**: 사내 Wi-Fi/네트워크 유실 시 최근 이벤트를 `localStorage`에 보관하며, 네트워크 복구 시 자동으로 배치 재전송합니다.

---

## 3. 고급 분석 기능 사용법

Overview의 `conversion_rate`는 호환성을 위해 User Conversion Rate를 의미합니다. API는 `conversion_users`, `conversion_sessions`, `user_conversion_rate`, `session_conversion_rate`를 모두 제공합니다. 날짜는 관리자에게 설정된 Site Timezone 기준이며, 저장 Timestamp 자체는 UTC입니다.

### 3.1 첫 화면: 지금 봐야 할 것

`개요` 화면 상단은 총계보다 먼저 **오늘 확인할 것**을 보여줍니다.

- 이상 감지 결과(심각·주의·긍정 변화)와 **미달 전망 Goal**을 심각도 순으로 통합합니다. 같은 심각도면 방금 발생한 변화인 이상을 표준 목표인 Goal보다 앞에 둡니다.
- 각 항목은 근거와 다음 행동을 함께 제시하고 `확인` 버튼으로 상세 화면(방문자 인사이트 / Goal)으로 이동합니다.
- 알림 상태가 있으면 `신규`, `지속 N일`을 제목에 표시합니다.
- **달성했거나 순항하는 Goal, 정상·데이터 부족 판정 이상은 넣지 않습니다.** 모든 것을 넣은 목록은 아무 것도 알려주지 않기 때문입니다. 이상이 없으면 그 사실을 명확히 표시합니다.
- 이 화면은 일별 Rollup 기반 이상 감지와 Metric Registry만 조회하므로 첫 화면 응답 속도에 영향을 주지 않습니다.

### 3.2 방문자 인사이트 (Visitor Insights)


좌측 `모니터링 → 방문자 인사이트`는 여러 화면을 순회하지 않고 방문자 상황과 다음 행동을 한 화면에서 확인하는 요약 보고서입니다. 모든 지표는 **이전 동일 기간과 자동 비교**됩니다.

- **핵심 인사이트**: 영향이 큰 순서로 정렬되며 각 항목이 `근거 → 원인 후보 → 다음 행동`을 함께 제시합니다. 방문자 급증·급감(주도 채널 지목), 신규·재방문 전환율 격차, 참여율 하락, 이탈률 높은 진입 페이지, 기기 간 전환율 격차, 채널 전환율 편차, 반복 방문 미전환, 휴면 전환을 감지합니다.
- **지표**: 방문자, 신규·재방문, 신규 비중, 세션, 1인당 방문 횟수, 세션당 페이지뷰, 참여율, 평균 체류 시간, 사용자 전환율. 신규 비중처럼 방향이 모호한 지표는 증감을 성과 색으로 표시하지 않습니다.
- **유입 채널**: Source·Medium을 Direct, Organic Search, Paid Search, Email, Social, Referral, Internal Portal, Internal Notice, Internal Message, Display, Other로 분류합니다. 사내망에서 유입 정보가 없는 방문은 `Direct (사내망)`으로 구분합니다.
- **진입 페이지**: 세션 비중과 이탈률, 참여율, 전환율, 평균 체류를 함께 제공해 비중이 큰데 이탈률이 높은 페이지를 먼저 찾습니다.
- **방문 빈도·최근 활동**: 1회 / 2~3회 / 4~9회 / 10회 이상, 그리고 최근 1일 / 2~7일 / 8~30일 / 31일 이상 미활동으로 나눠 충성도와 휴면 위험을 봅니다.
- **실행 대상**: `3회 이상 방문했지만 미전환`, `한 번만 방문한 신규`, `이전 기간에만 활동(휴면)`, `휴면 후 복귀` 인원과 권장 조치를 제시합니다. 같은 조건으로 Segment를 만들어 Action으로 연결할 수 있습니다.

**바로 가져가기**: 우측 상단 `요약 복사`는 결론·근거·표를 포함한 Markdown을 클립보드에 넣고, `Markdown`은 같은 내용을 파일로 내려받습니다. 각 표는 CSV로 내보낼 수 있습니다. 관리자 → Action의 Scheduled Report에서 `visitor_insight` 종류를 선택하면 같은 보고서를 Webhook·Mail·Confluence·사내 메시지·AI Agent로 정기 배달하고, MCP 도구 `get_visitor_insights`로 AI Agent가 직접 가져갈 수 있습니다.

채널·기기별 사용자 합계는 한 사용자가 여러 채널로 방문하면 중복될 수 있고, 신규 판정은 선택한 환경의 전체 수집 이력을 기준으로 합니다.

### 3.3 방문자 추적 (Visitor Timeline)


`탐색 → User Explorer`는 실제 방문자 한 사람을 추적합니다. 개인정보 설정에서 Visitor Profile을 비활성화하면 화면과 API가 모두 차단되고, 조회 사실은 Audit Log에 기록됩니다.

- **찾기**: User ID, Visitor ID 일부, 부서, 조직, 페이지 URL, 이벤트 이름, 기능 이름으로 검색합니다. 각 결과는 무엇으로 일치했는지와 세션·이벤트·전환·최근 활동을 함께 보여줍니다. 세션·사용자 리포트의 `추적` 버튼으로 바로 들어올 수도 있습니다.
- **사람 단위 vs 단일 Visitor**: 기본은 사람 단위입니다. SSO User ID로 연결된 데스크톱·모바일 등 모든 Visitor의 활동을 하나의 시간순 기록으로 합칩니다. 아직 identify되지 않은 방문자는 기기 단위로만 추적됩니다.
- **세션 그룹 타임라인**: 방문(세션)별로 시작 시각, 체류 시간, 참여 여부, 전환, 기기·브라우저·OS, 유입 채널, 사내망 이름, 진입 → 종료 페이지를 헤더로 보여주고, 그 안의 이벤트를 시간순으로 나열합니다. 각 이벤트에는 **이전 이벤트와의 간격**이 표시되어 어디서 멈췄는지 보입니다.
- **식별 시점**: 익명 활동이 특정 사용자로 연결된 이벤트에 `사용자 식별` 표시가 붙습니다. `식별 연결` 카드는 각 기기가 언제 이 사람으로 연결됐는지 보여줍니다. fingerprint는 사용하지 않습니다.
- **과거 탐색**: `이전 기록 더 보기`가 커서로 이전 구간을 이어서 불러옵니다. 한 페이지에 일부만 로드된 세션은 `부분 로드`로 표시합니다.
- **교차 서비스**: 같은 SSO User가 Workspace의 다른 서비스에서 활동했다면 함께 표시합니다.
- **가져가기**: `추적 기록 복사`는 사람 단위 맥락과 세션 흐름을 Markdown으로 복사해 장애 티켓이나 개인정보 요청 답변에 바로 붙일 수 있습니다.

### 3.4 이상 감지 (Anomaly Detection)


`모니터링 → 방문자 인사이트` 상단에서 직전 완료된 하루를 **같은 요일 최근 8주의 중위수**와 비교합니다. 사내 서비스는 요일 주기가 강해 단순 전주 대비나 7일 평균은 오탐이 많고, 부분 집계된 오늘을 평가하면 매일 아침 급감으로 보입니다. 그래서 완료된 하루만, 같은 요일끼리 비교합니다.

- 중위수와 MAD(중위 절대편차)를 사용해 하루의 장애가 기준선을 끌고 가지 않습니다.
- 편차 2.5σ 이상은 경고, 3.5σ 이상은 심각, 좋은 방향의 큰 변화는 긍정 변화로 구분합니다.
- 같은 요일 표본이 3개 미만이면 최근 28일로 대체하고, 그것도 부족하면 `데이터 부족`으로 판정을 보류합니다. 대체 기준선이 쓰이면 **근거 문장이 그렇게 말합니다** — `최근 N일 기준선(같은 요일 표본 부족, 요일 혼합)`. 요일이 섞인 비교는 이 감지기가 피하려는 바로 그 비교이므로, 판정 자체는 같아 보여도 읽는 사람은 무엇과 비교했는지 알아야 합니다. v0.34.37 이전에는 어느 쪽이든 `같은 요일 기준선`이라고 적었고, 배포 3주 이내의 사이트는 **항상** 대체 기준선을 쓰므로 그 기간의 모든 알림이 하지 않은 비교로 설명됐습니다.
- 감시 지표: 방문자, 세션, 이벤트, 전환, 오류(오류는 증가가 나쁨).
- 감지 결과는 **알림 상태**로 관리됩니다. 처음 감지되면 `신규`, 이전에 이미 알린 이상이 계속되면 `지속 N일`, 기준선 범위로 돌아오면 `회복`입니다.
- 관리자 → Action에서 `anomaly` 종류의 Scheduled Report를 만들면 기본적으로 **`신규`와 `회복`만** Webhook·Mail·사내 메시지·AI Agent로 전송합니다. 매시간 실행해도 같은 이상을 반복 통보하지 않고, 문제가 해소되면 회복을 한 번 알립니다.
- 보낼 상태가 없으면 전송 이력에 `skipped`로 남습니다. 정의에 `"notify_on": ["new","ongoing","recovered"]`를 넣으면 지속 상태도 하루 한 번 보내고, `"always_send": true`는 매번 보냅니다.
- 화면 조회는 알림 이력을 바꾸지 않습니다. 상태 저장은 배달 경로에서만 일어나므로 대시보드를 새로 고쳐도 "이미 알렸다"는 기록이 바뀌지 않습니다.

### 3.5 전환 기여도 (Attribution)


SDK는 세션 단위로 유입 정보를 기록하므로 방문(세션)이 Touchpoint입니다. `방문자 인사이트 → 전환 기여도`에서 모델을 바꿔 비교합니다.

| 모델 | 배분 기준 | 종류 |
| :--- | :--- | :--- |
| `last_non_direct` | 전환 직전, 채널 정보가 있는 마지막 방문 (기본값) | 단일 |
| `first_touch` | Lookback 안의 첫 방문 | 단일 |
| `last_touch` | 전환 직전 방문 그대로 | 단일 |
| `linear` | 경로의 모든 방문에 같은 비중 | 다중 |
| `time_decay` | 전환에 가까운 방문에 더 많이. 반감기 1·3·7·14·30일 선택 | 다중 |
| `position_based` | 첫 방문 40%, 마지막 방문 40%, 중간 방문들이 20%를 균등 분할 | 다중 |

다중 터치 모델은 하나의 전환을 여러 방문에 나눠 배분하므로 채널별 `배분 전환`이 소수로 표시됩니다. 어떤 모델이든 한 전환의 가중치 합은 정확히 1입니다.

`배분 전환`과 함께 `관여 전환`(경로에 등장한 전환 수), `관여 비중`, `관여만`(이 모델에서 배분받지 못한 전환)과 `평균 경로 방문 수`를 제공해 모델 간 차이를 확인할 수 있습니다. Lookback(기본 30일) 안에 방문 기록이 없는 전환은 `미배분`으로 분리 표기합니다.

### 3.5.1 교차 서비스 기여도


`전환 기여도`의 **배분 범위**를 `전사 서비스`로 바꾸면 같은 Workspace의 다른 서비스 방문도 Touchpoint로 인정합니다. 인사 시스템 공지를 보고 포털에서 신청한 흐름처럼, 서비스 경계를 넘는 기여를 측정합니다.

- **SSO로 식별된 사용자에게만** 적용됩니다. 익명 방문자는 설계상 서비스별로 격리되므로 교차 배분 대상이 아닙니다.
- 서비스별 배분 전환과 비중을 함께 제공하고, 전환이 발생한 서비스에는 `전환 발생` 표시가 붙습니다.
- 범위는 **조회자가 이미 접근할 수 있는 서비스**로만 확장됩니다. 권한이 없는 서비스는 포함되지 않습니다.
- 다른 서비스의 기여가 없으면 그 사실을 명시합니다.

### 3.5.2 봇과 모니터링 트래픽은 리포트에 포함됩니다

Collector는 User-Agent로 모든 Event를 분류해 저장합니다.

| 분류 | 판정 기준 |
| :--- | :--- |
| `known_bot` | `bot`, `crawler`, `spider`, `slurp`, `bingpreview`, `headlesschrome` |
| `monitoring` | `uptime`, `pingdom`, `healthcheck`, `monitoring`, `prometheus` |
| `suspicious` | User-Agent가 비어 있음 |
| `normal` | 위에 해당하지 않음 |

`traffic.class`는 **어떤 클라이언트가 보냈는지**만 말합니다. **어느 네트워크에서 왔는지**는 별도 필드 `traffic.internal`이 답합니다. 사내망에서 도는 크롤러는 `known_bot`이면서 `traffic.internal = true`입니다 — 두 사실이 서로를 지우지 않습니다.

> v0.34.33 이전에는 사내망에서 발생한 Event의 분류를 `internal_traffic`이 덮어썼습니다. 그래서 아래 권장 조건이 **사내 사용자 전체를 함께 제외**했고, 사내망 크롤러는 어떤 `traffic.class` 필터로도 찾을 수 없었습니다.
>
> 그 이전에 수집된 Event도 `traffic.class`로는 **`normal`로 읽힙니다** — `internal_traffic`으로 저장됐다는 것은 그 클라이언트가 다른 무엇으로도 분류되지 않았다는 뜻이기 때문입니다. 덕분에 릴리스 날짜를 걸치는 기간을 조회해도 **저장한 Segment의 의미가 도중에 바뀌지 않습니다.** Tracking Debugger는 실제로 저장된 값을 그대로 보여 줍니다.

**중요: 리포트는 이 분류로 걸러내지 않습니다.** 크롤러와 모니터링 요청도 사용자·세션·페이지뷰에 함께 집계됩니다. 1분마다 페이지를 확인하는 Uptime 감시는 하루 1,440건의 페이지뷰를 더합니다.

제외하려면 Segment 조건 `traffic.class = normal`을 사용하세요 — 사내 사용자는 그대로 남습니다. 사내망 트래픽만 제외하려면 `traffic.internal = false`를 사용합니다. 저장한 Segment는 Query·Funnel·Retention·경험 비교에 그대로 넣을 수 있습니다.

분류 결과는 사용자 탐색기의 타임라인과 관리자 → 수집 디버거에서 Event별로 확인할 수 있습니다.

### 3.5.3 같은 이름의 숫자가 뜻하는 것

- **사용자**는 사람 단위입니다. 같은 SSO User로 연결된 여러 브라우저는 한 명으로 셉니다. 첫 화면, 방문자 인사이트, Workspace Roll-Up, 쿼리 빌더의 `users`가 모두 같은 정의를 씁니다.
- **Visitor**는 브라우저 단위입니다. `방문자 상세` 목록은 Visitor 단위이므로, 한 사람이 두 기기를 쓰면 두 줄로 나타납니다. 사용자 수보다 줄 수가 많은 것은 오류가 아닙니다.
- **매출**은 `purchase` Event의 `value` 또는 `revenue` Property에서 읽습니다. 두 이름 모두 동일하게 인식하며, 첫 화면·Ecommerce·쿼리 빌더·MCP가 같은 정의를 사용합니다. v0.29.1 이전에는 쿼리 빌더만 `value`만 읽어 `revenue`를 쓰는 사이트에서 0으로 표시됐습니다.
- **Session**은 sessions 테이블에서 **시작 시각 기준**으로 셉니다. 자정을 넘긴 Session은 시작한 날에만 속하므로 연속된 기간의 합이 맞습니다. 참여 Session, 평균 Session 시간, Session 전환율도 같은 출처를 사용하므로 첫 화면과 방문자 인사이트가 같은 값을 보여줍니다. 파생 데이터가 아직 생성되지 않은 기간에는 이벤트에서 계산합니다.
- 쿼리 빌더에는 Session 지표가 두 개 있습니다. `sessions`는 **이 기간에 활동이 있었던** Session이고(페이지·채널·기능별 분해에 필요한 값), `sessions_started`는 **이 기간에 시작된** Session입니다(첫 화면과 같은 값). 자정에 열려 있던 Session 때문에 두 값은 다릅니다. 지표 선택 목록에 각 정의를 함께 표시합니다.
- **전환율**은 사용자 기준과 Session 기준을 구분해 제공합니다. 이름이 `전환율`인 값은 사용자 기준이며, Session 기준은 `Session 전환율`로 표기합니다.

### 3.6 행동 기반 Segment


Segment 조건에 사람의 전체 이력을 기준으로 하는 필드를 사용할 수 있습니다.

- `entity.sessions`, `entity.events`, `entity.conversions`
- `entity.days_since_last_seen`, `entity.days_since_first_seen`
- `entity.frustration_signals`, `entity.frustration_sessions` — tracker가 자동 감지한 막힘 신호(2.1.1 참고)를 겪은 횟수와 방문 수
- `entity.searches`, `entity.zero_result_searches`, `entity.search_clicks` — 검색 횟수, 결과 0건 검색, 결과 클릭 수
- `traffic.class`, `traffic.internal` — 봇·모니터링·사내망 트래픽 포함 여부(3.5.0 참고)

이 필드들은 사람의 **전체 이력**을 한 번 집계한 뒤 대상자를 추려내는 방식으로 실행됩니다. 조회 기간이 넓거나 이벤트가 많아도 집계는 한 번만 수행됩니다.

숫자 비교(`>=`, `<=`, `=` 등)만 지원합니다. 예를 들어 `entity.sessions >= 3` AND `entity.conversions = 0`은 "세 번 이상 방문했지만 전환하지 않은 사람"이고, `entity.zero_result_searches >= 1` AND `entity.search_clicks = 0`은 "검색했지만 아무것도 찾지 못한 사람"입니다. 막힘 신호 목록은 서버가 고정하므로 Segment 조건에 Event 이름을 직접 넣을 필요가 없습니다.

방문자 인사이트의 `실행 대상`, Frustration과 검색 분석 화면의 `실행 대상`에서 `Segment 만들기`를 누르면 서버가 세어 준 정의가 그대로 저장되어 Query·Funnel·Action에서 재사용됩니다. 기간 기준과 전체 이력 기준의 차이 때문에 인원이 다를 수 있는 경우에는 안내 문구를 함께 표시합니다.

정기 배달에서도 같은 정의를 씁니다. 관리자 → Action의 Scheduled Report에서 `Segment 집계`를 고르면 **저장된 Segment**를 선택할 수 있고, 화면이 쓰는 것과 같은 컴파일러로 평가하므로 중첩 조건과 행동 기반 조건이 그대로 적용됩니다. Event 이름·Feature·부서 입력란은 그 위에 겹쳐 더 좁히는 용도이며, Segment를 고르지 않으면 그 세 조건만으로 집계합니다.

### 3.7 Segment 비교 Funnel


`퍼널` 화면의 **비교 Segment**에서 최대 3개를 선택하면 전체와 나란히 같은 퍼널을 평가합니다. 단계·모드·최대 전환 시간이 동일하게 적용되므로 열 사이 비교가 성립합니다.

- 비교 시 차트는 사용자 수 대신 **완주율(%)** 로 그립니다. 규모가 다른 조직을 나란히 놓으면 절대 수치가 형태를 가리기 때문입니다.
- 각 Segment마다 전체 대비 완주율 격차(pp)와 상대 격차(%), 그리고 **격차가 가장 크게 벌어지는 단계**를 함께 제시합니다. 그 단계가 먼저 확인할 지점입니다.
- 전체보다 뒤처지는 단계가 없으면 없는 단계를 만들어 표시하지 않습니다.
- 진입 20명 미만 Segment는 `표본 부족`으로 표시하고 우열을 판정하지 않습니다.

### 3.8 Retention Segment 비교


`코호트` 화면의 **비교 Segment**에서 최대 3개를 선택하면 전체와 Retention 곡선을 비교합니다.

- 곡선은 **Cohort 크기로 가중한 평균**입니다. 비율의 단순 평균은 3명 Cohort가 1000명 Cohort와 같은 무게를 갖게 만듭니다.
- **아직 해당 주차에 도달하지 못한 Cohort는 분모에서 제외**합니다. 지난주에 시작한 Cohort를 4주차에서 0%로 세면 곡선이 실제보다 낮게 보입니다.
- Segment마다 첫 재방문(1주차) 격차와 **격차가 가장 큰 주차**를 제시하며, granularity에 따라 일·주·개월 단위로 표기합니다.
- Cohort 인원 20명 미만은 `표본 부족`으로 표시하고 우열을 판정하지 않습니다.

### 3.9 집단별 경험 비교


`경험` 화면의 **비교 Segment**에서 최대 3개를 선택하면 같은 측정을 집단별로 나눠 봅니다. 사이트 전체 p75는 빠른 환경과 느린 환경을 평균해 둘 다 가립니다.

- 집단별 **Core Web Vitals p75**와 **오류 경험 사용자 비율**을 제공합니다.
- 전체보다 **30% 이상 느린** 지표만 보고합니다. 조금 느린 것은 조치할 대상이 아닙니다.
- 전체는 권장 기준 안에 있는데 해당 집단만 초과하면 `심각`으로 구분합니다. 단순히 느린 것과 기준을 넘긴 것은 다른 문제입니다. 권장 기준은 LCP 2500ms, INP 200ms, CLS 0.1, FCP 1800ms, TTFB 800ms입니다.
- 오류 경험 비율이 전체보다 5pp 이상 높으면 보고하고, 15pp 이상은 `심각`입니다.
- 표본 20건 미만은 판정하지 않습니다. 뚜렷한 격차가 없으면 그 사실을 표시합니다.

### 3.10 쿼리 빌더 (Query Builder)

1. **필터링 조건**: 날짜 범위, 부서, 특정 이벤트명, 커스텀 속성(Key-Value) 조건 설정.
2. **그룹핑 (GroupBy)**: `department`별 또는 `browser`별 시계열 집계 그래프 생성.

### 3.11 2~10단계 퍼널 분석 (Funnel Analysis)

- 서비스 진입 ➔ 주요 기능 탐색 ➔ 최종 결재/제출까지 단계별 전환율 및 이탈율(Drop-off) 시각화.

### 3.12 사용자 경로 분석 (Path Analysis)

- 동일 Session에서 연속으로 발생한 Page·Event 이동을 시작/도착 두 계층 Sankey 다이어그램으로 분석합니다. 왕복 이동도 안전하게 표시하며 현재 Environment의 최근 30일 전환 수와 이동 횟수를 함께 제공합니다.

### 3.13 Segment와 저장된 Exploration

- Segment 화면에서 최대 5단계의 중첩 `AND`/`OR` 조건과 14개 연산자를 조합합니다.
- `event.has = purchase`를 사용하면 조회 행의 Event 종류와 무관하게 구매 경험이 있는 사용자를 선택합니다.
- 저장된 Segment는 Query Builder와 Funnel에서 재사용할 수 있으며 Query 조합 자체도 Exploration으로 저장합니다.

### 3.14 Ecommerce와 User Explorer

- Ecommerce는 `view_item`, `add_to_cart`, `begin_checkout`, `purchase`, `refund`와 `items` 배열을 기준으로 매출·거래·상품 성과를 계산합니다.
- User Explorer는 Visitor별 Event Timeline과 Deterministic Identity Graph를 제공합니다. 같은 `user_id`로 식별된 브라우저·기기의 Visitor ID는 하나의 canonical user로 연결되며, 로그인 전 익명 Event도 해당 사용자의 Funnel·Segment·전환 및 부서/조직 분석에 포함됩니다.
- Momento는 fingerprint나 확률 기반 결합을 사용하지 않습니다. `analytics.identify()` 또는 SSO에서 받은 내부 pseudonymous ID만 신뢰하며, 관리자가 Visitor Profile을 비활성화하면 Identity Graph API와 화면도 함께 차단됩니다.

### 3.15 Cohort, Business Journey와 Feature Adoption


- Cohort는 최초 Event/가입/구매 등 Cohort Event와 Return Event를 분리해 Day/Week/Month Retention을 계산합니다.
- Business Journey는 2~12개의 Event·Service·Feature 조건을 실제 도달 순서와 Conversion Window로 연결합니다.
- Feature Adoption은 Canonical User의 Organization/Department와 Event의 `feature`를 연결하며 대상자, 사용률, 재사용률, 최근 활성 및 비활성 사용자를 제공합니다.

### 3.16 Experience와 Release Impact


SDK의 자동 RUM은 LCP, INP, CLS, FCP, TTFB, Load와 Resource Error를 `web_vital`, `resource_error` Event로 전송합니다. Release 비교가 필요하면 초기화 시 `releaseVersion`, `gitSha`, `deploymentId`를 지정하십시오. Experience 화면은 오류가 발생한 사용자와 정상 사용자의 전환율을 비교합니다.

### 3.17 AI / Agent / MCP Event 표준


`ai_prompt`, `ai_response`, `ai_model_call`, `ai_tool_call`, `ai_agent_run`, `ai_mcp_call`을 사용하고 `model`, `provider`, `agent`, `mcp_server`, `tool`, `success`, `latency_ms`, `input_tokens`, `output_tokens`, `cost`, `fallback_model`을 Property로 전달합니다. 실제 Prompt/Response 원문은 개인정보와 기밀정보 위험 때문에 기본 분석 규격에 포함하지 않는 것을 권장합니다.

### 3.18 Workspace, Feature, Search와 Frustration


- Workspace Roll-Up은 같은 Workspace의 Site를 합쳐 서비스별 사용자·Event·Session·Service Score를 비교합니다. 같은 SSO `user_id`는 Site를 넘어 한 명으로 계산하고 익명 Visitor는 Site별로 격리합니다.
- Feature Intelligence는 `feature` Property를 기준으로 Adoption, Repeat, Conversion, Error, 기간 추세와 Dead Feature 후보를 계산합니다.
- Search Analytics는 `search`, `search_result`, `search_click`, `search_no_result`, `search_refine`, `search_exit`, `search_success` 표준 Event를 사용합니다. `search`, `search_click`, `search_refine`은 tracker가 자동 전송하므로 별도 계측 없이 검색 횟수, CTR, 재검색을 볼 수 있습니다(2.1.1 참고).
- Frustration Analytics는 Replay를 저장하지 않고 `rage_click`, `dead_click`, `rapid_back`, `form_retry`, `repeated_search`, `error_after_click`, `slow_interaction`과 오류 Event만으로 막힘을 추정합니다. 아홉 신호 모두 tracker가 자동 감지합니다.
- 두 화면은 표가 비어 있을 때 "문제가 없다"와 "측정되지 않는다"를 구분해 안내합니다. Session은 있는데 신호가 0건이면 스니펫 버전과 `data-frustration-signals` 설정을 확인하라고 표시하고, 검색 횟수는 있는데 검색어가 모두 `(not set)`이면 `data-collect-search-terms` 안내를 표시합니다. 결과 0건이 한 건도 없으면 `data-momento-search-results` 계측이 빠졌을 가능성을 함께 알립니다.
- Frustration 표는 각 신호의 뜻과 확인할 것을 함께 보여주므로 신호 이름만 보고 해석을 추측하지 않아도 됩니다. `겪은 사람 찾기`를 누르면 사용자 탐색기가 그 신호로 검색된 상태로 열려, 신호에서 실제 방문자 추적까지 한 번에 이어집니다.
- Frustration 화면은 **신호별 전환 영향**을 함께 계산합니다. 각 신호를 겪은 사람과 겪지 않은 사람의 전환율을 비교해 `전환 손실`, `차이 없음`, `전환과 함께 발생`, `판단 보류`로 판정하고, 차이가 전환 몇 건에 해당하는지 추정합니다. 목록은 **차이의 크기가 아니라 전환 손실 추정치 순**으로 정렬됩니다. 많은 사람이 겪는 작은 차이가 소수만 겪는 큰 차이보다 먼저 고칠 대상일 수 있기 때문입니다.
- 양쪽 집단이 각각 20명 미만이면 판단을 보류합니다. 거의 모든 사람이 겪는 신호는 겪지 않은 쪽이 기준선이 될 수 없으므로 같은 규칙으로 보류합니다.
- 이 비교는 **연관성이며 인과가 아닙니다**. 신호를 겪은 집단이 원래 더 어려운 업무를 하고 있었을 수 있으므로, 화면에 그 사실을 항상 함께 표시합니다. 인과를 확인하려면 해당 화면을 고친 뒤 같은 집단을 다시 비교하세요.
- `form_retry`처럼 전환 과정에서 자연히 발생하는 신호는 `전환과 함께 발생`으로 구분해, 고칠 필요가 없는 것을 고치러 가지 않게 합니다.
- 두 화면은 `실행 대상`으로 바로 저장할 수 있는 집단을 함께 제시합니다. Frustration은 `막힘을 겪고 전환하지 않은 사람`, `두 번 이상의 방문에서 막힌 사람`, 검색은 `결과를 못 찾은 사람`, `여러 번 검색했지만 아무것도 열지 않은 사람`입니다. 저장한 Segment는 Funnel·Retention·경험 비교에 그대로 넣어 "막힌 집단이 실제로 덜 전환하는지" 확인할 수 있습니다.

### 3.18.1 사내 사용 현황

내부 서비스에서 "누가 쓰고 있는가"는 조직 단위로 묻는 질문입니다. 사내 사용 현황은 같은 기간을 여섯 가지로 나눠 보여줍니다.

| 기준 | 어디서 오는가 |
| :--- | :--- |
| 망 | Collector가 IP를 관리자 → 네트워크 망 설정과 대조해 분류합니다. 어디에도 맞지 않으면 `External / Unclassified`입니다. |
| 부서 · 조직 | `analytics.identify`가 보낸 사용자 속성입니다(2.3 참고). 보내지 않으면 `(미지정)`으로 모입니다. |
| 서비스 | Event의 `service` Property이며, 없으면 Site의 Service Name을 씁니다. |
| 기능 | Event의 `feature` Property입니다. |
| 버튼 | `click` Event의 `button`·`element_text`·`element_id` 중 먼저 있는 것입니다. |

각 기준은 Event 수·사용자 수·Session 수를 함께 보여주므로, "많이 눌렀다"와 "많은 사람이 눌렀다"를 구분할 수 있습니다. 상위 20개까지 표시합니다.

`(미지정)`이 대부분이라면 데이터가 없는 것이 아니라 그 속성을 보내지 않고 있는 것입니다. 부서·조직은 `analytics.identify`로, 기능은 Event Property로 채웁니다.

### 3.19 Experiment와 Goal


Metric Goal 평가에는 기간 진행률과 **착지 예상치**가 함께 표시됩니다. 누적 지표는 현재 속도를 기간 끝까지 연장하고, 비율 지표는 누적되지 않으므로 현재 관측값을 그대로 사용합니다. 누적 지표에는 목표까지 남은 양과 `필요 일일 속도`를 제공하며, 기간 진행률이 10% 미만이면 추정을 보류합니다.

- Experiment Event에는 `experiment_id`와 `variant` Property를 함께 전송합니다. 등록한 Semantic Metric을 Variant별로 계산하고 첫 Variant를 Control로 사용해 Lift와 Confidence를 제공합니다.
- Goal 화면은 일/주/월/분기 Metric 목표의 현재값, 진행률과 달성 여부를 Site Timezone 기준으로 표시합니다.
- Change Calendar는 배포, Release, 장애, 캠페인, 교육, Feature Flag와 조직 변경을 분석 기간에 함께 표시합니다.

---

### 3.20 발견을 정기 배달로 연결


분석 화면에서 찾은 내용을 반복해서 받으려면 `관리 → Action`에서 Scheduled Report를 만듭니다. v0.19부터 정의 문서를 직접 작성하지 않습니다.

- **보낼 내용**을 고르면(개요 요약, 인사이트 요약, 방문자 인사이트 전체, 이상 감지 알림, 기능 도입 현황, 경험·오류, AI 사용량, Segment 집계) 그 종류가 쓰는 입력만 표시됩니다.
- 이상 감지 알림은 기간 대신 **전송할 상태**(신규·지속·회복)와 `이상이 없어도 매번 전송`을 고릅니다. 나머지 종류는 집계 기간을 고릅니다.
- 저장 전에 "매주 방문자 인사이트 전체 · PRD · 최근 30일" 같은 한 줄 요약으로 무엇이 언제 전송되는지 확인할 수 있습니다.
- 방문자 인사이트 화면의 `정기 배달`과 이상 감지 카드의 `알림 설정` 버튼은 종류가 미리 선택된 상태로 이 화면을 엽니다. Delivery Channel은 먼저 등록되어 있어야 합니다.
- Delivery Channel의 인증 Header는 이름과 값을 행 단위로 입력합니다. 이름 중복, Host 재정의, 줄바꿈, 값 누락은 저장 전에 표시됩니다. 값은 `MOMENTO_ENCRYPTION_KEY`로 암호화 저장되고 목록에는 이름만 나타납니다.
- 특수한 정의가 필요하면 `고급: Definition JSON 직접 입력`에 값을 넣으면 그 JSON을 그대로 사용합니다.
- v0.21.3 이전 버전에서는 `방문자 인사이트 전체`와 `이상 감지 알림` 종류가 데이터베이스 제약에 걸려 저장되지 않았습니다. 두 종류를 사용하려면 v0.21.3 이상으로 업그레이드하십시오.

### 3.20.1 정기 배달의 조회 기간

정기 배달은 화면과 **같은 기간**을 사용합니다. `days: 7`이면 사이트 시간대의 최근 7일이며 로컬 자정에 끝납니다. 배달이 실제로 나간 시각과 무관하므로, 배달된 숫자와 화면의 숫자가 일치합니다.

payload에 `from`과 `to`가 함께 담기므로 어떤 기간을 측정한 것인지 확인할 수 있습니다.

### 3.21 분석 쿼리 제한

조회가 25초 제한을 넘으면 화면이 원인과 **바로 누를 수 있는 대안**을 함께 제시합니다: 기간 줄이기(기간 선택이 있는 화면), Segment로 대상 좁히기, 쿼리 빌더에서 Fast 모드로 실행, 정기 배달로 받기. 기간을 바꿀 수 없는 화면에서는 기간 줄이기를 제안하지 않습니다 — 누를 수 없는 버튼은 안내가 아니라 막다른 길이기 때문입니다.

조회가 8초를 넘기면 대기 화면이 상황을 알리고, 20초를 넘기면 제한에 가까워지고 있음을 알려 완료 전에 대응할 수 있게 합니다.

분석 기간 선택지는 사이트의 **최대 정확 조회 기간** 정책을 반영합니다. 정책이 허용하지 않는 기간은 목록에 나타나지 않고, 넘는 기간을 요청하면 `RANGE_EXCEEDS_POLICY`로 거절하면서 현재 한도를 함께 알려줍니다. 이 한도는 관리자 → 분석 쿼리 보호에서 조정합니다.

경험 비교, Retention, Adoption, Frustration, 검색 분석, Feature Intelligence, Workspace Roll-Up, AI 분석, Ecommerce 화면에서 **분석 기간을 직접 선택**할 수 있습니다. Retention은 코호트 특성상 90·180·365일, Feature Intelligence는 30·60·90일, 나머지는 7·30·90일을 제공합니다. 인사이트와 데이터 품질 화면은 최근 7일 고정입니다 — 최근 상태를 보는 운영 화면이기 때문입니다.

권한 부족처럼 사용자가 화면에서 해결할 수 없는 문제에는 버튼을 만들지 않고 무엇을 요청해야 하는지 알려줍니다.


대화형 분석 조회는 25초를 넘기면 중단되고 `분석 쿼리가 25초 제한을 초과했습니다`라는 안내와 함께 대안을 제시합니다. 기간을 좁히거나 Segment로 범위를 줄이고, 반복적으로 필요한 넓은 범위의 집계는 Scheduled Report로 정기 배달받으십시오.

## 4. BI 연동 & 데이터 내보내기 (Export)

Raw CSV / NDJSON 형식으로 원클릭 내보내기가 가능합니다.

```python
# Python Pandas 연동 예시
import pandas as pd

# Momento Export NDJSON 읽기
df = pd.read_json('momento_events_20260808.ndjson', lines=True)

# 부서별 이벤트 건수 분석
dept_stats = df.groupby('department')['event_name'].value_counts()
print(dept_stats)
```

### 4.1 개인 API Key로 접근할 때

BI 도구나 스크립트는 개인 API Key(`mom_key_`)를 `Authorization: Bearer`로 보냅니다. Key는 **소유자의 권한을 그대로 따르되, 두 가지는 항상 거부됩니다.**

- **관리자 기능**: 사용자·설정·감사 로그 등은 소유자가 super_admin이어도 Key로 접근할 수 없습니다. 스크립트에 오래 남는 자격 증명이 배포 설정을 바꿀 수 있어서는 안 됩니다
- **대화형 쓰기**: Segment 저장처럼 콘솔에서 수행하는 작업은 `SESSION_REQUIRED`로 거부됩니다

Key를 폐기하거나 만료시키면, 그리고 소유자 계정을 비활성화하면 즉시 401을 받습니다.

Key는 조회 범위를 따로 제한하지 않습니다. 특정 사이트만 허용하려면 소유자의 Workspace 권한을 조정하세요 — 조회 가능한 사이트는 사람과 Key가 동일합니다.

## 5. Query Mode와 비용 보호

- `Exact`는 Raw Event 100%를 계산합니다. 관리자가 정한 최대 기간과 Complexity를 넘으면 실행 전에 거부됩니다.
- `Fast`와 `Preview`는 Event ID의 결정적 Hash로 관리자 설정 비율을 표본화합니다. 같은 입력은 같은 표본을 사용합니다.
- 결과에는 Query Mode, Complexity Score, Sample Percent, 실행 분류와 보수적인 예상 오차 안내가 포함됩니다.

## 6. 개인정보 요청 Workflow

관리자는 Privacy Requests에서 삭제 또는 Export 요청을 먼저 생성하고 별도의 승인 동작으로 실행합니다. User ID 삭제는 Identity Graph에 연결된 Visitor까지 포함하며, 기간 삭제는 Site Timezone 경계를 사용합니다. 요청자·승인자·결과 건수·상태는 Audit와 요청 이력에 남습니다. 승인 완료된 Export는 이벤트·사용자·세션 속성을 포함한 전체 NDJSON으로 내려받을 수 있습니다.

## 7. Console 탐색과 표 활용

- 좌측 메뉴는 모니터링, 웹 분석, 제품 분석, 탐색·실험, 경험·AI로 묶여 있으며 현재 화면이 속한 그룹만 자동으로 펼쳐집니다.
- `Ctrl+K` 또는 macOS의 `Cmd+K`를 누르면 화면, 분석 기능, 관리자 설정을 이름으로 검색해 바로 이동할 수 있습니다. 일반 사용자는 권한이 없는 관리자 명령을 볼 수 없습니다.
- 상단 Breadcrumb와 Site·Environment Context를 확인하면 현재 분석 범위를 놓치지 않을 수 있습니다.
- 데이터 표는 행 검색, 페이지당 행 수 변경과 CSV 내보내기를 공통으로 지원합니다. CSV는 현재 검색 결과를 UTF-8 BOM 형식으로 저장합니다.
- 모바일에서는 Sidebar 대신 Drawer와 축약된 Site Selector를 사용하며, 넓은 표는 가로 스크롤로 열 손실 없이 확인합니다.
