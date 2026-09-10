# Unlog Admin Dashboard - 관리자 가이드

## 개요

Unlog Admin Dashboard는 사주/운세 서비스의 운영을 위한 종합 관리 시스템입니다.

## 접속 정보

| 항목          | 값                                   |
| ------------- | ------------------------------------ |
| URL           | `http://localhost:3000/admin`        |
| 기본 관리자   | `admin@unlog.com` / `password123`    |
| 운영팀 계정   | `operator@unlog.com` / `password123` |
| 콘텐츠팀 계정 | `content@unlog.com` / `password123`  |

---

## 주요 기능

### 1. 대시보드 (`/admin`)

- **핵심 지표**: 총 사용자 수, 총 매출, 활성 상담사 수
- **최근 활동**: 신규 가입자, 최근 결제 내역

### 2. 사용자 관리 (`/admin/users`)

- 사용자 목록 조회 (검색/필터링)
- 사용자 상세 정보 및 히스토리
- 계정 상태 변경 (활성/정지)

### 3. 콘텐츠 관리 (`/admin/content`)

- 운세 템플릿 CRUD
- 카테고리: 일반, 재물, 연애, 건강
- 타입: 일일, 월간, 신년 운세

### 4. AI 엔진 설정 (`/admin/engine`)

- AI 프롬프트 관리
- 모델 선택 (GPT-4, Claude 등)
- 버전 관리 및 활성화 토글

### 5. 상품 관리 (`/admin/products`)

- 상품 CRUD (리포트, 구독, 상담)
- 가격 및 상태 관리

### 6. 결제 내역 (`/admin/payments`)

- 결제 트랜잭션 조회
- 상태: 결제완료, 대기중, 환불

### 7. 시스템 로그 (`/admin/logs`)

- 관리자 활동 로그
- 사용자 활동 로그

### 8. 관리자 설정 (`/admin/settings/admins`)

- 관리자 계정 생성
- 역할 할당 (SUPER_ADMIN, OPERATOR, CONTENT_MANAGER)

---

## 시작하기

```bash
# 1. Docker로 PostgreSQL 시작
docker compose up -d

# 2. 데이터베이스 초기화
npx prisma db push

# 3. 시드 데이터 삽입
npx tsx prisma/seed.ts

# 4. 개발 서버 시작
npm run dev
```

## 역할별 권한

| 역할              | 권한                  |
| ----------------- | --------------------- |
| SUPER_ADMIN       | 모든 기능 접근 가능   |
| OPERATOR          | 사용자/결제/로그 관리 |
| CONTENT_MANAGER   | 콘텐츠/AI 엔진 관리   |
| COUNSELOR_MANAGER | 상담사/상담 관리      |
