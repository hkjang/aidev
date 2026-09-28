# v2.11.10 릴리즈 인계

적용 스킬: marketing:product-launch 및 technology:release-and-deployment의 로컬 SKILL.md.
전용 Skill 도구는 제공되지 않아 원본을 직접 읽었다. 별도 정형 반환 형식은 없었다.
Tier 3: 2페이지 이후의 카탈로그 이용자가 보기 전환 중 위치를 잃는 문제의 수정으로,
릴리즈 노트만 준비한다. 별도 홍보·가격·패키지 변경은 없다.

기존 릴리즈는 chore(release): AppStore vX.Y.Z 커밋과 한국어 주석 태그를 사용하며,
별도 CHANGELOG 파일은 없다. v2.11.7~9와 동일한 패치 증가로 v2.11.10을 선택했다.
release.yml이 태그 푸시 뒤 영어 본문, SHA-256 및 Docker tar.gz 하나를 만든다.
따라서 github_release=false, assets=[]이며 Docker 자산은 로컬에서 중복 생성하지 않는다.

실제 배포는 수행하지 않았다. 배포 담당자는 동일 CI 산출물을 먼저 한 인스턴스에서
검증하고 확대한다. 새 버전의 health 실패 또는 보기 전환 시 page/표시 앱 불일치가
1건이라도 재현되면 배포 담당자가 확대를 중단하고 v2.11.9 이미지로 복구한다.
배포 후 첫 24시간 내 desktop/mobile 각각 2페이지 보기 전환 성공 1회 이상과
관련 문의를 확인한다. 이는 인계 기준이며 실제 사용자 채택이나 배포 성공은 미측정이다.
외부 사용자 첫 사용 검증은 무인 로컬 세션에서 수행하지 못했으며, 별도 비평 역할의
검토와 HTTP fixture 기반 전체 E2E만 확보했다. 실제 DB/Keycloak 검증은 미실행,
DSN 없는 DB 통합 테스트는 skip이다. 이미지 build/load/smoke는 태그 CI의 후속 gate다.
