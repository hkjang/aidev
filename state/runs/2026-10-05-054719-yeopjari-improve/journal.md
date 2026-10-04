# 회차 노트 2026-10-05-054719-yeopjari-improve — yeopjari
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:47] base pinned — main@32869c9
- [러너 05:47] autonomy release — 

## 정찰 노트
- EXIF·GPS 제거를 실제 JPEG와 sanitise로 검증하는 테스트 1파일 과제를 선택했다. DB가 필요한 연봉·CSRF 및 보호 경로 CI보다 45분 내 확실히 끝낼 수 있다.
- 연봉 공개 규칙은 기존 E2E에 있어 done 처리했다. 별도로 RLS assertion이 undefined도 성공 처리하는 공백을 새 후보로 남겼다.
- 합성 GPS JPEG의 축소 유무/full·thumb 정제를 읽기 전용 실행으로 확인했다. npm test는 의존성 부재로 exit127; 전체 검사와 외부 도구의 잠금파일 일치는 미확인이다.
- 이전 쿠키 수정은 pinned main에 보이지 않아 재선택하지 않았다. 구현자는 유효한 GPS fixture부터 입증하고 타입에 없는 exifBuffer·대역·프로덕션 범위 확장을 피할 것.
- [러너 05:53] scout done — 사진 정제의 EXIF·GPS 제거를 실제 JPEG 회귀 테스트로 보장 (가치 4 / 위험 1 / 작업량 M)

## 구현 노트
- image.test.ts 신규 1파일(프로덕션 0개), 커밋 14f79f7: 실제 JPEG의 EXIF/GPS·COM 제거 회귀 테스트 7건 추가.
- 입력의 정상 decode·TIFF GPS IFD 포인터/태그/RATIONAL·주석을 먼저 검증; maxLongEdge 32/16 × full/thumb의 크기·decode·SOS 이전 헤더 제거 확인.
- 실패 재현 없음: 기존 구현은 정상이며 첫 입력 테스트부터 통과. 원본 EXIF와 COM 각각의 거부를 toThrow 대조군으로 검증했고 결함 수정으로 주장하지 않음.
- 검증: Node v22.23.1, npm ci exit 0(취약점 0; eslint 지원 종료 경고), npm test -- packages/core/src/__tests__/image.test.ts 입력 1건→완성 7건 통과.
- 검증: npm run check exit 0(린트·core/server/web 타입 검사·17파일 180건), npm run build exit 0; 빌드 산출물은 기존 dist ignore 적용으로 커밋 제외.
- 검증 못 한 곳: HTTP 업로드·DB·저장소 E2E와 릴리즈. 증명 범위는 저장 직전 실제 sanitise 함수이며 DB/외부 서비스 불필요.
- 일부러 제외: 프로덕션/코덱/품질 루프/의존성/API/릴리즈 변경, 픽셀·전체 JPEG 바이트 스냅샷. 후속 역할은 헤더 검사가 SOS 이전에 한정됨에 유의.
- [러너 05:57] brief accepted — 채택 — 현재 sanitise와 mediaRoutes가 정찰 근거와 일치하고 지정된 테스트 1파일만으로 수용 기준을 충족했다.
- [러너 05:57] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- approve / low / blocking 없음: main...HEAD는 image.test.ts 1파일이며 프로덕션·의존성·권한·외부 상태 변경 없음. 요청한 세 부서 스킬은 전용 도구 부재로 로컬 SKILL.md를 직접 읽고 적용.
- 실제 sanitise 호출, 합성 GPS TIFF 포인터·RATIONAL·COM, 축소 유무 × full/thumb decode·크기·메타데이터 제거 및 거부 대조군 확인. 기존 정상 동작을 고정하는 테스트 추가이므로 수정 전 실패 없음은 결함이 아님.
- npm run check 재실행 exit 0(17파일 180건), git diff --check 성공. 소스 수정 없음. 개인정보 신규 처리나 구체적 보안 공격 경로 없음.
- 남는 범위: SOS 이전 헤더만 검사하므로 코덱 교체 시 재검토 필요. mediaRoutes 저장 연결은 소스로 확인했지만 HTTP·DB/RLS·저장소 E2E 및 빌드·릴리즈는 재실행하지 않음.
- [러너 05:59] review approved — 리뷰 승인 (risk=low)
- [러너 05:59] pr created — https://github.com/hkjang/yeopjari/pull/2
- [러너 06:03] ci no-ci — 이 커밋에 검사가 없음 (정책 allow_merge_without_ci 가 없으면 차단)
