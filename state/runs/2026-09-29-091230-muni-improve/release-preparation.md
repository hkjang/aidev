# v0.50.0 릴리즈 준비

- 스킬: marketing:product-launch, technology:release-and-deployment를 로컬 headcount SKILL.md에서 읽고 적용. 전용 Skill 도구는 제공되지 않음.
- Tier 3: 워크스페이스 ZIP 사용자 대상 오류 수정, 릴리즈 노트만 준비. 외부 발송 없음.
- 최신 로컬 태그와 제공된 GitHub Release는 v0.48.0. VERSION과 기존 노트는 v0.49.0이므로 기존 마이너 증가 관례를 이어 v0.50.0 선택.
- 버전 원천 VERSION 갱신. frontend package.json/package-lock.json의 0.1.0은 독립 private 패키지 버전, Compose/Kubernetes의 v0.1.0은 기존 설치 예제이므로 관례대로 유지. 노트에 운영 이미지 변경 명시. main.version 기본값 dev는 빌드 인자로 대체.
- release.yml: 태그 푸시로 linux/amd64 이미지 빌드, docker save+gzip, manifest 태그 확인, GitHub Release 및 muni-v0.50.0.tar.gz 생성. 따라서 github_release=false, assets=[].
- sync-release-notes.yml은 main 변경 시 이미 존재하는 Release만 갱신함. 태그 생성과 경합할 수 있어 러너가 보존된 노트 적용 여부를 확인해야 함.
- 빌드 검증 이미지는 로컬 검사용이며 배포하지 않음. 운영 배포와 검증된 CI 산출물 승격은 후속 워크플로/배포 담당자 책임.
- 독립 비평 역할의 실제 ZIP 라우트 검증 승인 기록을 제공받음. 신규 외부 사용자의 수동 체험은 수행하지 않음.
- 중단 기준: health/readiness 실패 또는 1999/2000/2001 경계의 문서 수·생략 안내 불일치. 배포 담당자가 이전 보관 이미지로 롤백. 스키마 변경 없음.
- 배포 후 첫 주 담당자가 ZIP 문의를 검토: 정확히 2000건의 잘못된 생략 안내 0건, 초과 안내 유지. 운영 관찰은 아직 수행되지 않음.
