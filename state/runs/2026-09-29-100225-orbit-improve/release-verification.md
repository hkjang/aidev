# Orbit v0.7.5 릴리즈 검증

- Tier 3: 기존 AI 사용자 대상 오류 처리 개선. 별도 캠페인이나 외부 알림 없이 기존 GitHub Release 노트로 전달한다.
- 없는 사람·타인 소유 사람·관계 없는 사람은 SSE 시작 전 동일한 404 not_found JSON을 받는다. 정상 스트림 및 기존 400/503/500 계약은 유지한다.
- 최근 세 릴리즈는 VERSION만 변경한 chore(release): v0.7.x 커밋과 Orbit v0.7.x 주석 태그를 사용한다. web/package.json과 lock의 0.1.0은 독립 프런트엔드 패키지 값으로 기존 관례를 유지한다. Go 실행 버전과 OpenAPI 버전은 빌드 주입 값을 사용한다.
- 저장소 내 CHANGELOG나 릴리즈 노트 파일은 없으며, GitHub 워크플로가 PR 기반 노트를 생성한다. release-notes.md는 동일 형식의 인계용 사본이다.
- 격리 PostgreSQL 16에서 go test -race -count=1 -v ./... 통과. AI 하위 시험 9개 모두 실행 및 통과. go vet ./..., go build ./... 통과.
- 잠금 파일 기반 npm 설치, 웹 16개 파일 121개 시험, TypeScript/Vite 빌드 통과.
- linux/amd64 Docker 이미지 빌드 통과. 로컬 이미지는 검증용이며 배포 자산이 아니다. 태그 워크플로가 최종 커밋에서 이미지를 빌드하고 아카이브 manifest의 태그를 확인한 뒤 Release에 업로드한다.
- 자산은 워크플로 생성 대상 orbit-v0.7.5.tar.gz 하나이므로 로컬 인계 assets는 빈 배열이며 github_release는 false이다.
- 스키마 변경 없음. 실제 운영 배포·외부 AI·신규 사용자 브라우저 확인은 이 로컬 릴리즈 범위에서 실행하지 않았다. 앞선 독립 비평의 실제 DB 검증 결과도 확인했다.
- 운영 후속 확인 담당: hkjang. 배포 후 첫 24시간 AI 오류 응답과 지원 문의 확인. 조회 불가 세 경우에서 500 또는 제공자 호출이 한 건이라도 재현되거나 정상 SSE가 실패하면 확대를 중단하고 이전 v0.7.4 이미지로 복귀한다. 운영 관측은 아직 수행하지 않았다.
- 원격 푸시, GitHub Release 생성, 업로드는 외부 러너가 수행한다.
