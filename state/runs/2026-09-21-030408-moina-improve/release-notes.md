## MOINA 오프라인 이미지

- 파일: `moina-v0.1.35.tar.gz`
- 이미지: `moina:v0.1.35`
- 플랫폼: `linux/amd64`

태그 워크플로가 서비스 이미지를 빌드·검증하고 tar.gz 하나를 게시하며, 최종 GitHub Release 본문에 실제 산출물의 SHA256을 기록합니다.

## 변경 사항

- 미디어 업로드의 multipart 본문 한도 초과를 `413 media_too_large`로 일관되게 반환합니다. ContentLength가 알려진 요청과 알 수 없는 요청 모두 적용됩니다.
- 다른 multipart 파싱 오류는 기존 `400 invalid_media`를 유지하며 OpenAPI 설명을 보강했습니다.
- 본문·파일 한도, MIME, 파일 이름, 저장 및 인증 정책은 유지합니다.

## 검증

- make fmt, make check, 전체 화면 캡처를 요구하는 Pages QA, 백엔드 빌드 통과.
- 임시 PostgreSQL을 연결한 go test -race ./..., go vet ./..., staticcheck 통과. 테스트 skip 없음.
- 프런트엔드 테스트 250개, ESLint(오류 0, 기존 경고 39), v0.1.35 프로덕션 빌드 통과.
- 이미지 빌드·패키징·런타임 및 브라우저 smoke는 태그 워크플로에서 수행합니다.
- 회사 스킬 marketing:product-launch 및 technology:release-and-deployment는 도구·로컬 경로에 없어 고유 절차와 반환 형식은 확인하지 못했습니다.
