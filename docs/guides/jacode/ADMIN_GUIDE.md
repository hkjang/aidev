# Jacode 관리자 가이드

이 문서는 Jacode 시스템 관리자를 위한 가이드입니다. 시스템 설치, AI 모델 설정, 사용자 관리 및 문제 해결 방법을 다룹니다.

## 1. 시스템 요구사항 및 설치

### 필수 구성 요소

- **OS**: Linux (Ubuntu 20.04+ 권장), macOS, Windows (WSL2)
- **Runtime**: Node.js v18 이상 (LTS 권장)
- **Database**: PostgreSQL 16
- **Cache**: Redis 7
- **AI Engine**: Ollama (또는 vLLM)
- **Container**: Docker & Docker Compose (선택 사항)

### 설치 절차 (Docker Compose 권장)

1. 저장소를 클론하고 프로젝트 루트로 이동합니다.
2. 환경 변수 파일을 준비합니다.
   ```bash
   cp .env.example .env
   # .env 파일 내의 POSTGRES_*, REDIS_*, JWT_SECRET 등을 설정합니다.
   ```
3. Docker Compose로 실행합니다.
   ```bash
   docker-compose up -d
   ```
4. 초기 데이터베이스 마이그레이션 및 시딩을 수행합니다.
   ```bash
   docker-compose exec backend npm run prisma:migrate
   docker-compose exec backend npm run db:seed
   ```

---

## 2. AI Provider 설정

Jacode는 다양한 로컬 LLM 서버(Provider)를 지원합니다.

### 지원 Provider

- **Ollama**: 설치가 간편하고 개인용/소규모 팀에 적합합니다.
- **vLLM**: 고성능 추론이 필요하거나 엔터프라이즈 환경에 적합합니다.

### Provider 연결 설정

1. 관리자 대시보드 > **Settings** 페이지로 이동합니다.
2. **Global AI Settings** 섹션에서 AI API URL을 설정합니다.
   - 예: `http://localhost:11434` (Ollama 기본값)
3. **API Key**: 필요한 경우 API 키를 입력합니다 (Ollama는 불필요).

---

## 3. 모델 및 라우팅 정책 관리 (Model & Routing)

### 모델 관리

- 시스템은 연결된 Provider에서 사용 가능한 모델 목록을 자동으로 가져옵니다.
- `Admin > Models` 메뉴에서 개별 모델을 활성화/비활성화할 수 있습니다.
- 모델별로 '용도(Capability)'를 태깅할 수 있습니다 (예: `coding`, `chat`, `embedding`).

### 라우팅 정책 (Routing Policy)

작업 유형에 따라 어떤 모델을 사용할지 결정하는 규칙을 설정합니다.

- `Admin > Routing Policies` 메뉴로 이동합니다.
- **Rules 예시**:
  - `Chat` 요청 -> 속도가 빠른 `mistral:7b` 모델 사용
  - `Code Generation` 요청 -> 성능이 좋은 `codellama:13b` 또는 `deepseek-coder` 모델 사용
- 가중치(Cost, Performance, Availability) 기반으로 점수를 매겨 최적의 모델을 선택하도록 설정할 수 있습니다.

---

## 4. 보안 및 제한 설정 (Security & Limits)

### 보안 설정

- **Prompt Injection Protection**: 사용자 입력에 대한 인젝션 공격 탐지 활성화 여부를 설정합니다.
- **PII Masking**: 코드 생성 시 이메일, IP 등 민감 정보 마스킹 여부를 설정합니다.
- **Code Safety Check**: 생성된 코드에 취약점 패턴이 있는지 검사합니다.

### 사용량 제한 (Rate Limiting)

- 사용자별/IP별 분당 요청 횟수 제한 설정이 가능합니다.
- `Throttler` 설정을 통해 과도한 리소스 사용을 방지합니다.

---

## 5. 문제 해결 (Troubleshooting)

### 주요 이슈

#### Q: AI 응답이 오지 않고 타임아웃이 발생합니다.

- **확인**: 백엔드 서버에서 AI Provider(Ollama/vLLM) URL로 접속이 가능한지 확인하세요 (`curl` 테스트).
- **조치**: Docker 환경인 경우 `host.docker.internal` 또는 올바른 서비스 이름을 사용했는지 확인하세요. 모델이 로드되는 데 시간이 오래 걸릴 수 있으므로 타임아웃 설정을 늘려보세요.

#### Q: "No active model found" 에러가 발생합니다.

- **확인**: 관리자 페이지에서 활성화된(Active) 모델이 하나 이상 있는지 확인하세요.
- **조치**: Provider 연결을 확인하고 `Sync Models` 버튼을 눌러 모델 목록을 갱신하세요.

#### Q: 데이터베이스 연결 오류

- **확인**: `.env` 파일의 `DATABASE_URL`이 올바른지 확인하세요.
- **조치**: PostgreSQL 서비스가 실행 중인지 확인하고 로그를 점검하세요.

### 로그 확인

- 백엔드 로그: `docker-compose logs -f backend`
- 프론트엔드 로그: `docker-compose logs -f frontend`
