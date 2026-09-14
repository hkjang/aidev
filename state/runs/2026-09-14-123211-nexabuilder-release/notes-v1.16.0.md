
[`v1.16.0`](https://github.com/hkjang/nexabuilder/releases/tag/v1.16.0)
은 메뉴 권한 거버넌스 + 사이드바 개인화 릴리즈 (#131~#161) 이며,
Spring Boot 4.1 / Java 21 / jQuery 4 로 런타임을 올렸습니다.

### 런타임 / 빌드 (#1~#17)

- **Spring Boot 4.1.1 + Spring Framework 7 / Security 7** (#16). Jackson
  은 `spring-boot-jackson2` 호환 모듈과 `withJsonConverter` 로 기존
  Jackson 2 `ObjectMapper` 를 그대로 씀 — Jackson 3 전환은 별도 회차.
  MockMvc / Liquibase / Security 테스트 자동설정이 Boot 4 의 모듈로
  옮겨감. `JsonWireUsesJackson2Test` 가 전선 규칙을 고정.
- **Java 21 (Temurin) + Gradle wrapper 8.14.3** (#11). 빌드·실행
  이미지, 툴체인, CI `setup-java` 를 21 로 통일. 사설 저장소에서
  뜨지 않던 CodeQL 잡은 저장소가 공개일 때만 돌도록 조건 부여.
- **foojay 툴체인 리졸버** — JDK 21 이 없는 머신에서도 Gradle 이
  Temurin 21 을 `~/.gradle/jdks` 에 내려받아 빌드됨.
- **jQuery 4.0.0 + jquery-migrate 4.0.2** (#15). jQWidgets(webjar 23.1.0
  이 최신) 가 jQuery 4 가 없앤 `isFunction`·`trim` 등을 400회 넘게
  부르므로 migrate 를 jQuery 와 `jqx-all` 사이에 둠. 스크립트 순서를
  고정하는 테스트 동반.
- **webjar 경로에서 버전 제거** (#14). `webjars-locator-lite` 로 요청
  시점에 풀고 템플릿·JS 28곳의 버전 표기를 삭제. 조용히 404 나던
  ECharts `<script>` 와 qrcodejs 경로를 고침. 템플릿에서 webjar
  참조를 긁어 하나씩 요청하는 회귀 테스트 추가.
- **의존성** — OpenPDF 3.0.5 (`org.openpdf.text` 패키지 이동, #12),
  guava 33.7.1 (#9), commons-lang3 3.20.0 (#7), actions/checkout v6 ·
  setup-java v5 · upload-artifact v7 (#1~#3).
- **AGPLv3 라이선스** 명시, 브랜딩 로고 · 파비콘, `FUNDING.yml`.

### 사이드바 개인화 (#131~#139, #145)

- **자주 사용 자동 핀** — 최근 30일 클릭 Top 5 메뉴를 사이드바
  상단 "⭐ 자주 사용" 섹션으로 (#132). **수동 즐겨찾기** — 마이그레이션
  077 `nexa_user_menu_pin`, 최대 20개, 별 버튼으로 토글하고 HTML5
  drag-and-drop 으로 순서 변경 (#134, #137).
- **빠른 검색 팔레트** — `Cmd/Ctrl+Shift+P` 로 사이드바 메뉴를 fuzzy
  검색 (#135). 개인 클릭 횟수(`/api/v1/me/menu-clicks/weights`)로
  결과 가중 정렬 (#138).
- **전체 접기/펼치기 단축키** — `Ctrl/Cmd+Shift+[` / `]` (#133).
- **클릭 로그 보존 기간** — 매일 03:45 정리, 기본 180일
  (`nexa.menu.click-log.retention-days`), 사용 통계 탭에서 즉시 정리
  (#131).
- **모바일 off-canvas drawer** — 슬라이드 전환, ESC 닫기, 폭 769px
  이상으로 넓히면 자동 닫힘 (#145).
- **ARIA 접근성** — 사이드바 nav / 섹션 토글 / 핀 버튼 / 팔레트
  combobox·listbox 에 role·aria 속성 정비 (#139).

### 메뉴 권한 거버넌스 (#140~#144, #146~#161)

- **매트릭스 CSV 내보내기/가져오기** — 메뉴 × 역할 매트릭스를 CSV
  로 내려받고 (#141), Excel 에서 고쳐 다시 올리면 dry-run 미리보기
  → 자동 백업 스냅샷 → 적용 (#156).
- **변경 이력과 되돌리기** — "최근 변경" 탭에 MENU_PERMISSION 감사
  20건과 행별 되돌리기 (#148), 선택 일괄 되돌리기 (#151).
- **권한 스냅샷** — 마이그레이션 078, 이름 붙인 복원 지점 생성·복원·
  삭제 (복원 전 자동 백업) (#150), 두 스냅샷 diff 뷰어 (#153),
  `nexa.menu.snapshot.auto-create` 로 일일 자동 스냅샷 + 보존 개수
  (#154).
- **What-if 시뮬레이션** — 부여/제거 전에 메뉴 diff 와 영향 사용자
  미리보기 (#143).
- **알림과 연동** — 권한 변경 시 `EntityChangedEvent` 로 웹훅 발행
  (#142), 영향 사용자에게 opt-in 알림 (#136), 마이그레이션 079
  `nexa_menu_notify_channel` 로 담당자 채널 라우팅 (#152) + 채널
  테스트 발송 버튼 (#155).
- **점검 탭** — 고아 권한 행 스캔·정리 (#149), 공개 메뉴 / 빈 역할 /
  전역 부여 메뉴 커버리지 리포트 (#157), "왜 이 사용자에게 이 메뉴가
  안 보이나" 접근 진단 (#158), 메뉴 트리 구조 진단(고아 자식·순환·
  비활성 부모) (#160), 문제 수 배지 (#161).
- **통계** — 일별 추이 / 워크스페이스 분포 / 24×7 히트맵 ECharts
  (#140), 권한 변경 활동 추이와 변경자 Top 10 (#159), 미사용 메뉴
  보관 후보 + 일괄 비활성화 (#144).
- **역할 관리** — 메뉴 집합 포함 관계로 역할 상속 Hasse 다이어그램
  렌더 (#147). **아이콘 선택기** — 메뉴 폼에 FontAwesome 아이콘 그리드
  모달 (#146).

### 수정

- **`/admin/analytics` 화면이 다시 그려짐** (#17). JS 주석의 `[[` 를
  Thymeleaf 가 인라인 표현식으로 읽어 응답이 중간에 끊기던 문제.
  경로 변수 없는 모든 화면 라우트를 열어 보는
  `PagesRenderIntegrationTest` 로 재발 방지.
- OTLP endpoint 검증, 로컬 rate limiter 비활성화, 메뉴 권한 drift
  감지 갱신.

**Full Changelog**: https://github.com/hkjang/nexabuilder/compare/v1.15.0...v1.16.0
