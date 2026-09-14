## KCB Cutover Dashboard v1.5.0

### 신규 기능
- **관리자 화면에서 붙이는 방문 추적 스크립트**: 관리자 콘솔(`/admin`)의 **방문 추적 스크립트** 카드에서 Momento(사내 수집기)·GA4·GTM·Matomo·직접 입력 스니펫을 켜고 끌 수 있습니다. **기본값은 꺼짐**이며, 설정은 `data/tracking.json`에 저장되어 재배포 없이 바뀝니다.
  - Momento는 같은 오리진 프록시(`/momento/*`)를 기본으로 사용해 외부 출처가 정책에 등장하지 않고, 방문자 PC에서 수집기에 직접 닿지 않는 망에서도 동작
  - 직접 입력 스니펫은 8KB 제한, `<script>` 태그만 삽입
  - `allowed_hosts`로 스니펫에서 자동으로 읽지 못한 출처를 손으로 추가, 카드 하단에 정책에 더해지는 출처 미리보기
  - 관리 화면 포함 여부(`include_admin`), 삽입 위치(`head` / `body`) 선택
  - 같은 설정을 API로도 조작 가능: `GET/PUT /api/tracking`, `GET/DELETE /api/tracking/violations` (관리자 세션 필요)
- **정책이 막은 출처 확인과 한 번에 허용**: 추적이 켜진 동안 브라우저의 CSP 위반 신고를 `POST /api/csp-report`로 받아 출처·지시어를 기록(서로 다른 출처 최대 100개, 메모리 보관)하고, 관리 화면에서 **허용 목록에 추가** 한 번으로 `allowed_hosts`에 반영합니다.

### 보안
- **모든 화면에 nonce 기반 콘텐츠 보안 정책(CSP) 적용** (`proxy.ts`): 스크립트는 앱 오리진과 요청별 nonce가 붙은 것만 허용(`script-src 'self' 'nonce-…'`, `'unsafe-inline'` 없음). 추적을 켜면 스니펫에서 읽은 http(s) 출처만 `script-src`·`connect-src`·`img-src`에 더해지고, 끄면 원래 정책으로 되돌아갑니다.
- 비화면 경로(`/api/*`, `/momento/*`)는 `default-src 'none'`으로 더 좁게 제한

### 문서
- `docs/ADMIN_GUIDE.md` 8장 「방문 추적 스크립트 설정」 추가: CSP와 nonce 동작, 설정 항목 표, Momento 연결 절차, 막힌 출처 허용 방법
- `README.md`: 방문 추적 기능·CSP 설명, `cutover-data` 볼륨에 `tracking.json` 보관 안내, `npm run test:unit` 추가

### 변경 파일
- `proxy.ts`: 요청별 nonce 생성과 CSP 헤더 적용
- `lib/tracking/core.ts`, `lib/tracking/store.ts`, `lib/tracking/violations.ts`: 설정 저장·검증, provider별 스니펫 렌더링, 출처 추출·정책 조립, 차단 출처 고리 버퍼
- `app/api/tracking/route.ts`, `app/api/tracking/violations/route.ts`, `app/api/csp-report/route.ts`, `app/momento/[...path]/route.ts`
- `components/TrackingSettings.tsx`, `components/TrackingSnippet.tsx`, `app/layout.tsx`, `app/admin/page.tsx`
- `lib/tracking/core.test.ts`, `lib/tracking/violations.test.ts` (단위 테스트 25건), `e2e/tracking.spec.ts` (e2e 3건)

### 검증
- Docker 멀티스테이지 빌드 내 Next.js 프로덕션 빌드(`npm run build`) 통과
- 단위 테스트 `npm run test:unit` 25건 통과
- 빌드한 이미지 기동 후 `/api/activities` 200 응답 및 `/` 응답에 `Content-Security-Policy: … script-src 'self' 'nonce-…'` 헤더 확인

### 업그레이드
데이터 파일 형식은 이전 버전과 같으므로 `cutover-data` 볼륨을 그대로 물려줍니다. 방문 추적은 기본 꺼짐이라 업그레이드만으로 화면 동작이 달라지지 않습니다.

```bash
docker load -i cutover-v1.5.0.tar.gz
docker stop cutover-app && docker rm cutover-app
docker run -d --name cutover-app --restart always -p 3000:3000 \
  -e ADMIN_PASSWORD='<관리자-비밀번호>' -e USER_PASSWORD='<사용자-비밀번호>' -e AUTH_SECRET='<기존과-같은-값>' \
  -v cutover-data:/app/data cutover:v1.5.0
```

### 오프라인 배포
첨부된 `cutover-v1.5.0.tar.gz`는 `docker build -t cutover:v1.5.0 . && docker save cutover:v1.5.0 | gzip` 로 만든 이미지입니다. 폐쇄망 서버 반입·실행 절차는 `docs/ADMIN_GUIDE.md` 2장을 참고하십시오.
