# Jamail 사용자 가이드 📖

Jamail 사용자 가이드에 오신 것을 환영합니다. 이 문서는 Jamail 이메일 템플릿 관리 시스템의 모든 강력한 기능을 이해하고 활용하는 데 도움을 드립니다.

## 목차
1. [시작하기](#시작하기)
2. [템플릿 관리](#템플릿-관리)
3. [고급 기능](#고급-기능)
   - [다국어 지원](#다국어-지원)
   - [A/B 테스팅](#ab-테스팅)
   - [Multi-SMTP 라우팅](#multi-smtp-라우팅)
   - [웹훅](#웹훅)
4. [설정 및 관리](#설정-및-관리)

---

## 시작하기

`http://localhost:3000`에서 Jamail 대시보드에 접속할 수 있습니다. 상단 네비게이션 바의 언어 전환기를 사용하여 한국어와 영어를 전환할 수 있습니다.

**대시보드**는 다음 정보를 제공합니다:
- 전체 템플릿 수
- 오늘 발송된 이메일
- 최근 활동 로그
- 시스템 상태

---

## 템플릿 관리

### 템플릿 생성하기
1. **Templates** 페이지로 이동합니다.
2. **새 템플릿** 버튼을 클릭합니다.
3. 이름, 고유 키(슬러그), 설명을 입력합니다.
4. 카테고리를 선택합니다 (마케팅, 트랜잭션, 알림 등).

### 편집 및 미리보기
- **에디터**: 분할 화면 에디터를 사용하여 왼쪽에서 HTML을 작성하고 오른쪽에서 미리보기를 확인합니다.
- **변수**: 동적 콘텐츠를 위해 Handlebars 구문 `{{변수명}}`을 사용합니다.
- **미리보기 데이터**: "미리보기" 버튼을 클릭하고 JSON 데이터를 입력하여 변수 치환을 테스트합니다.

### 버전 관리
- 템플릿을 저장할 때마다 새로운 **버전**이 생성됩니다.
- 버전 히스토리를 확인하고, 변경 사항을 비교하며, 언제든지 이전 버전으로 롤백할 수 있습니다.

---

## 고급 기능

### 🌐 다국어 지원
수신자의 언어에 자동으로 적응하는 템플릿을 만듭니다.

**사용 방법:**
1. 템플릿을 열고 **번역** 탭으로 이동합니다.
2. **언어 추가**를 클릭하고 로케일을 선택합니다 (예: 한국어, 일본어).
3. 번역된 제목과 내용을 입력합니다.
4. **발송 시**: 이메일을 보낼 때 `locale` 매개변수를 제공합니다. Jamail이 자동으로 일치하는 번역을 선택합니다.
   - **Fallback 로직**: 요청된 로케일 → 템플릿 기본 언어 → 시스템 기본 언어

**예시:**
```json
{
  "to": "user@example.com",
  "locale": "ko",
  "variables": {
    "name": "홍길동"
  }
}
```

---

### ⚖️ A/B 테스팅
다양한 변형을 테스트하여 이메일을 최적화합니다.

**사용 방법:**
1. 템플릿 에디터에서 **A/B 테스팅** 탭으로 이동합니다.
2. **변형 생성** (예: "제목 라인 B")을 클릭합니다.
3. **트래픽 분할**을 설정합니다 (예: 50% / 50%).
4. Jamail이 설정한 가중치에 따라 자동으로 트래픽을 분배합니다.
5. 대시보드에서 **열람률**과 **클릭률**을 모니터링합니다.
6. **승자 선언**을 클릭하여 가장 성과가 좋은 변형을 기본값으로 설정합니다.

**메트릭:**
- 총 발송 수
- 열람 수 / 열람률
- 클릭 수 / 클릭률
- 전환 수 / 전환율

---

### 🔀 Multi-SMTP 라우팅
여러 제공업체를 통해 이메일을 라우팅하여 높은 전달률을 보장합니다.

**사용 방법:**
1. **설정 > SMTP**로 이동합니다.
2. 여러 SMTP 구성을 추가합니다 (예: 마케팅용 AWS SES, 트랜잭션용 SendGrid).
3. 각 SMTP 서버에 **카테고리**를 할당합니다.
   - 예: "마케팅" 이메일 → AWS SES
   - 예: "알림" → 내부 SMTP
4. 시스템이 템플릿의 카테고리에 따라 자동으로 이메일을 라우팅합니다.

**SMTP 설정 예시:**
```json
{
  "name": "AWS SES",
  "host": "email-smtp.ap-northeast-2.amazonaws.com",
  "port": 587,
  "username": "AKIAIOSFODNN7EXAMPLE",
  "password": "your-password",
  "categories": ["MARKETING", "NOTIFICATION"]
}
```

---

### 🔔 웹훅
외부 시스템과 통합합니다.

**사용 방법:**
1. **설정 > 웹훅**으로 이동합니다.
2. **웹훅 추가**를 클릭합니다.
3. 엔드포인트 URL을 입력하고 이벤트를 선택합니다 (`email.sent`, `email.opened`, `email.failed`).
4. **서명 시크릿**을 복사하여 서버에서 페이로드를 검증합니다.
5. 이벤트가 발생할 때마다 Jamail이 JSON 페이로드와 함께 POST 요청을 보냅니다.

**이벤트 타입:**
- `email.sent`: 이메일 전송 성공
- `email.delivered`: 수신자에게 전달됨
- `email.opened`: 이메일 열람
- `email.clicked`: 링크 클릭
- `email.failed`: 전송 실패

**페이로드 예시:**
```json
{
  "event": "email.sent",
  "timestamp": "2024-12-01T08:00:00Z",
  "data": {
    "templateId": "uuid",
    "recipient": "user@example.com",
    "subject": "환영합니다!",
    "sentAt": "2024-12-01T08:00:00Z"
  }
}
```

**서명 검증:**
```javascript
const crypto = require('crypto');

function verifyWebhook(payload, signature, secret) {
  const hmac = crypto.createHmac('sha256', secret);
  const expectedSignature = hmac.update(JSON.stringify(payload)).digest('hex');
  return signature === expectedSignature;
}
```

---

## 설정 및 관리

### API 키
- 백엔드 서비스가 프로그래밍 방식으로 이메일을 트리거할 수 있도록 API 키를 생성합니다.
- 보안을 위해 범위를 할당합니다 (예: `send:email`, `read:templates`).

**사용 예시:**
```bash
curl -X POST http://localhost:3001/templates/send \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "templateId": "welcome-email",
    "to": "user@example.com",
    "variables": {
      "name": "홍길동"
    }
  }'
```

### 사용자 관리 (RBAC)
- **Admin**: 모든 설정 및 데이터에 대한 전체 액세스
- **Operator**: 템플릿 생성/편집 및 이메일 발송 가능, 시스템 설정 변경 불가
- **Viewer**: 읽기 전용 액세스

### 감사 로그
- 보안 감사를 위해 누가 언제 무엇을 했는지에 대한 자세한 기록을 확인합니다.
- 모든 중요한 작업(템플릿 생성, 수정, 삭제, 설정 변경)이 기록됩니다.

---

## 문제 해결

### 이메일이 발송되지 않는 경우
1. SMTP 설정이 올바른지 확인합니다.
2. Redis와 PostgreSQL이 실행 중인지 확인합니다.
3. 백엔드 로그에서 오류 메시지를 확인합니다.

### 템플릿 미리보기가 작동하지 않는 경우
1. Handlebars 구문이 올바른지 확인합니다.
2. JSON 변수 데이터가 유효한지 확인합니다.

### 추가 지원
GitHub Issues에서 문제를 보고하거나 커뮤니티 포럼에 질문을 게시하세요.

---

**즐거운 이메일 관리 되세요! 📧**
