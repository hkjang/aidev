# PostgreSQL·브라우저 호환성 검증

게시된 버전 `0.2.0`의 대표 사용자 흐름을 PostgreSQL 17·18과 Playwright Chromium·Firefox·WebKit의 6개 조합에서 검증했습니다. main CI와 태그 릴리즈의 재검사·이미지 재반입·폐쇄망 검증을 통과했습니다. 이 결과는 전체 기능을 모든 OS에서 인증했다는 뜻이 아닙니다. 실제 macOS/iOS Safari, Windows/macOS 한글 IME, 터치 키보드는 별도 실기기 검증 대상입니다.

## main·태그 호환성 검증

커밋 `aff3ee1f6d48f895b0d1461dafdf2711638eb0ce`·번들 `main-BPw-YbqO.js`의 [호환성 CI 34363815488](https://github.com/hkjang/madi/actions/runs/34363815488)에서 6개 조합이 모두 통과했습니다. JavaScript 오류·HTTP 5xx·외부 페이지 요청은 0건이었으며, `requestfailed` 이벤트는 이번 원격 검사에서 수집하지 않았으므로 모든 네트워크 실패가 0건이라고 표현하지 않습니다. 같은 SHA의 main 전체 Go·35개 독립 브라우저 옵션과 shared25도 통과했습니다. 이후 [태그 릴리즈 34369441095](https://github.com/hkjang/madi/actions/runs/34369441095)의 호환성6조합과 나머지 릴리즈 검증도 통과하여 v0.2.0 단일 이미지 아카이브가 게시됐습니다. 아래 수치와 JSON은 해당 main 호환성 CI의 원본이며 태그 재실행 시간으로 바꾸지 않습니다.

| PostgreSQL | Chromium 153.0.8010.12 | Firefox 155.0 | WebKit 26.6 | Go `-race` 패키지 실행 |
| --- | --- | --- | --- | --- |
| 17.11 | 통과 · 9.568초 | 통과 · 12.798초 | 통과 · 18.365초 | 통과 · 46.305초 |
| 18.6 | 통과 · 8.462초 | 통과 · 14.072초 | 통과 · 17.765초 | 통과 · 46.326초 |

CI 아티팩트의 원본 JSON 내용은 [PG17 CI 결과](benchmarks/compatibility-pg17-v0.2.0-ci34363815488-aff3ee1f.json), [PG18 CI 결과](benchmarks/compatibility-pg18-v0.2.0-ci34363815488-aff3ee1f.json)에 변경 없이 보존했습니다. `verified_at`은 각각 `2026-09-09T14:30:31.232Z`, `2026-09-09T14:32:32.967Z`입니다. 원격 환경은 Linux/amd64·Go 1.26.8·논리 CPU 4개, 커널 `6.17.0-1022-azure`이며 두 작업 모두 private sysroot 없이 실행했습니다. 아래 로컬 환경·이전 수치·캡처를 이 CI 결과로 덮어쓰지 않습니다. 최신 전체 상태는 [배포 검증 기록](deployment-verification.md)에서 확인하세요.

태그 릴리즈의 별도 6조합 재검사도 모두 통과했으며 PostgreSQL17·18의 Go 패키지 시간은 각각 47.390초·89.407초였습니다. 이는 위 main CI의 46.305초·46.326초와 다른 실행입니다. 두 실행의 시간을 성능 개선·저하나 운영 응답시간 보증으로 해석하지 않습니다.

## 이전 By2 로컬 재검증

2026년 9월 9일 후보 웹 번들 `main-By2nz__J.js`를 기준으로 PostgreSQL 17·18의 격리 스키마에서 세 엔진을 각각 순차 재실행했습니다. 당시 6개 조합 모두 통과했으며 JavaScript 오류·HTTP 5xx·페이지 외부 요청은 0건입니다. 같은 번들의 shared25 7차와 문서 경합 전체 2회도 별도 로컬 검사에서 통과했습니다. 이 수치는 아래 원본 보고서의 해당 후보 결과입니다.

| PostgreSQL | Chromium 153.0.8010.12 | Firefox 155.0 | WebKit 26.6 | Go `-race` 시험 / 패키지 실행 |
| --- | --- | --- | --- | --- |
| 17.11 | 통과 · 16.205초 | 통과 · 30.672초 | 통과 · 25.486초 | 통과 · 78.30초 / 79.435초 |
| 18.6 | 통과 · 10.636초 | 통과 · 14.248초 | 통과 · 12.920초 | 통과 · 42.09초 / 43.160초 |

당시 실제 보고서를 [PostgreSQL 17 By2 후보 결과](benchmarks/compatibility-pg17-v0.2.0-By2nz__J.json), [PostgreSQL 18 By2 후보 결과](benchmarks/compatibility-pg18-v0.2.0-By2nz__J.json)에 별도로 보존했습니다. `verified_at` 값은 각각 `2026-09-09T11:57:51.849Z`, `2026-09-09T11:59:35.939Z`입니다. Go 시험·패키지 실행 시간은 `.local/compatibility-pg17-latest-By2nz__J.log`와 PG18 로그의 마지막 PASS·`ok` 행을 대조한 값입니다.

네트워크 진단에는 PG17 WebKit의 동일 출처 `PUT /api/v1/profile` 요청 취소가 2건 있습니다(`Load request cancelled`). 따라서 모든 네트워크 실패가 0건이라고 표현하지 않습니다. 원본은 `test-results/compatibility-latest-By2nz__J/pg17/webkit/diagnostics.json`에 보존했고, 각 조합의 진단과 기능 검증 결과를 구분합니다.

## 이전 B2 후보 재검사

앞선 `main-B2VqUvBo.js`의 PG17·18 재검사도 6개 조합이 통과했으며 Go 패키지 시간은 각각 63.534초·38.349초였습니다. 당시 [PostgreSQL 17 보고서](benchmarks/compatibility-pg17-v0.2.0-final.json), [PostgreSQL 18 보고서](benchmarks/compatibility-pg18-v0.2.0-final.json)는 원본 그대로 유지합니다. 파일명의 `final`은 당시 후보 명칭이며 후속 결과로 덮어쓰거나 현재 최종 릴리즈 통과 근거로 해석하지 않습니다. B2·CBN·Brs 실행 원본도 별도 시험 결과 경로에 보존합니다.

## 이전 첫 검사 기록

아래는 같은 날 처음 완료한 6개 조합의 기록입니다. 당시 두 PostgreSQL 시험은 같은 호스트에서 병렬 수행했고, 각 PostgreSQL의 세 엔진은 순차 실행했습니다. 이전 보고서를 최신 번들의 결과로 덮어쓰지 않습니다.

| PostgreSQL | Chromium 153.0.8010.12 | Firefox 155.0 | WebKit 26.6 | Go `-race` 시험 |
| --- | --- | --- | --- | --- |
| 17.11 | 통과 · 13.093초 | 통과 · 20.011초 | 통과 · 15.117초 | 통과 · 51.45초 |
| 18.6 | 통과 · 12.951초 | 통과 · 19.111초 | 통과 · 15.730초 | 통과 · 51.11초 |

이전 원본 보고서는 [PostgreSQL 17 첫 검사 결과](benchmarks/compatibility-pg17.json), [PostgreSQL 18 첫 검사 결과](benchmarks/compatibility-pg18.json)에 그대로 보존했습니다. 두 표의 시간은 각 격리 표본의 관측값이며 서비스 응답시간 보증이나 부하 성능 수치가 아닙니다. 호스트의 동시 작업 조건도 달라 두 실행의 시간을 성능 개선 수치로 비교하지 않습니다.

이전 로컬 시험 환경은 Linux/amd64, Ubuntu 26.04 사용자 공간, WSL2 커널 `6.6.87.2-microsoft-standard-WSL2`, Go `1.26.8`, Node `26.7.0`, Intel i7-8700의 논리 CPU 4개·메모리 약 15.62GiB입니다. CI는 별도로 Node 22와 PostgreSQL 17/18 서비스 컨테이너를 사용하며, 실제 원격 실행 결과와 엔진 버전을 아티팩트로 보존합니다. 로컬 결과와 원격 결과의 환경·시간·수집 범위를 구분합니다.

## 무엇을 확인했나요?

- 실제 로그인 폼과 후보 `VERSION`에 일치하는 로그인·프로필 버전 표시
- 문서의 Markdown 원문 편집, 한글·이모지 저장 후 API 정본과 새로고침 결과 비교
- 한글 태그 추가·보존, 합성 composition 이벤트의 조합 중 Enter가 미완성 태그를 만들지 않는지 확인
- 데이터베이스 항목의 단일 선택과 체크박스 다중 선택·해제, 저장 후 정본 값 비교
- 모달 안 Tab 초점, Escape·닫기 버튼 후 원래 버튼으로 초점 복귀
- 프로필 글꼴의 native select 저장·새로고침 반영
- 390×844 화면의 문서·데이터베이스, 가로 넘침과 모달 경계, 모바일 선택 입력

한글 시험은 브라우저의 Unicode 입력과 합성 조합 이벤트를 검사합니다. OS 입력기 자체의 조합·변환·후보 창까지 자동화한 시험은 아닙니다. 작은 화면은 뷰포트 변경이며 iPhone·Android 하드웨어 검증과 구분합니다. Playwright의 각 버전은 특정 엔진 바이너리를 사용하므로, 엔진 버전과 실제 Chrome·Firefox·Safari 배포 제품의 버전을 혼동하지 않습니다. [Playwright 브라우저 안내](https://playwright.dev/docs/browsers)

아래 공개 화면은 `main-By2nz__J.js`·PostgreSQL 18.6 재검사의 실제 캡처입니다. Chromium 한글 원문, Firefox 선택 입력, WebKit 모바일 선택 화면을 시각 검토한 뒤 원본 PNG 그대로 반영했습니다. 검증 시점은 공개 폴더로 복사된 파일의 생성·수정 시각이 아니라 `test-results/compatibility-latest-By2nz__J/pg18/`의 원본 보고서 시각과 엔진별 캡처 경로를 기준으로 확인합니다.

[shared25 공개 검증 목록](screenshots/verification-v0.2.0-shared.json)은 이 호환성 시험 및 원격 CI와 별개인 로컬 BPw 배치 `9ae61b64-ed0c-40e6-9193-4f65482e3410`의 정상 157장 근거입니다. [이전 By2 목록](screenshots/verification-v0.2.0-By2nz__J.json)도 당시 시각·해시로 보존합니다. 전체 갤러리나 여기의 별도 호환성 캡처까지 모두 같은 배치에서 검증했다는 뜻은 아니며, 기존 `verification.json`의 과거 기록도 유지합니다.

![Chromium에서 한글 원문과 태그 저장](screenshots/compatibility-chromium-korean.png)

![Firefox에서 단일·다중 선택 입력](screenshots/compatibility-firefox-select.png)

![WebKit 390px 화면의 선택 입력](screenshots/compatibility-webkit-mobile.png)

## 재현 방법

시험 코드는 `internal/server/compatibility_browser_test.go`와 `tests/compatibility-browser.mjs`입니다. 기본 Go 회귀에서는 건너뛰고 명시적으로 활성화합니다. 전용 시험 DB에만 연결하세요. 기존 통합 시험 도우미가 임의 이름의 격리 스키마를 만들고 종료 시 해당 스키마만 정리합니다. 실행 중인 개발 서버의 8080 포트를 사용하지 않습니다.

```sh
npm --prefix web ci
npm --prefix web run build
npm --prefix tests ci
node tests/node_modules/playwright/cli.js install --with-deps chromium firefox webkit

MADI_TEST_POSTGRES_DSN='postgres://madi:test-password@127.0.0.1:5432/madi_test?sslmode=disable' \
MADI_BROWSER_COMPATIBILITY=1 \
MADI_COMPAT_PG_MAJOR=17 \
MADI_COMPAT_BROWSERS=chromium,firefox,webkit \
go test -race ./internal/server -run '^TestBrowserCompatibility$' -count=1 -timeout=8m
```

PostgreSQL 18의 전용 DB에서도 DSN과 `MADI_COMPAT_PG_MAJOR=18`을 바꿔 반복합니다. 연결된 실제 메이저 버전이 요청 값과 다르면 즉시 실패합니다. 위 변수는 개발·CI 시험 전용이며 제품 런타임 환경변수 네 개 계약을 바꾸지 않습니다. 브라우저와 빌드 의존성 다운로드는 연결된 빌드 환경에서 수행합니다.

기본 결과 경로는 `internal/server/test-results/compatibility/pg17/` 또는 `pg18/`입니다. `report.json`에 실제 PostgreSQL·엔진·Go 버전과 환경·항목별 결과를 기록하고 엔진별 PNG를 보관합니다. 실패 캡처는 진단용이며 공개 갤러리에 포함하지 않습니다. `.github/workflows/compatibility.yml`은 동일 시험을 PG17/18 행렬로 실행하고 릴리즈가 해당 검증을 선행 조건으로 사용합니다.

## 로컬 격리와 한계

이번 PG17은 시스템 패키지를 설치하거나 기존 DB를 재시작하지 않고, 공식 PGDG 저장소의 서명된 메타데이터와 패키지 SHA-256을 검증한 후 전용 0700 임시 디렉터리에 추출했습니다. PostgreSQL 공식 Ubuntu 저장소는 이 배포판을 지원합니다. [공식 PGDG 설치 안내](https://www.postgresql.org/download/linux/ubuntu/)

실제 서버 패키지는 `postgresql-17 17.11-1.pgdg26.04+2`이고 SHA-256은 `4b2abeefb22fdfb35b2d3dffd26305c07e61f7f9cc22007197819ccac2044585`입니다. 시험 클러스터는 loopback에만 바인딩했고 완료 후 종료와 포트 해제를 확인했습니다. 기존 PG18·Docker Desktop·WSL 설정은 변경하지 않았습니다.

로컬 WebKit은 호스트 라이브러리 부족으로 공식 Ubuntu 패키지를 전용 sysroot에 추출해 사용했습니다. `MADI_COMPAT_WEBKIT_BUNDLE` 시험 옵션은 설치된 동일 Playwright MiniBrowser 바이너리와 해당 라이브러리를 직접 지정하며, 결과의 `private_dependency_sysroot`로 구분합니다. 바이너리·브라우저 보안 옵션을 수정하지 않았습니다. 일반 CI는 `install --with-deps`로 준비하므로 이 우회 옵션이 필요하지 않습니다. 이 로컬 조합이 다른 배포판의 모든 라이브러리 조합을 보증하지는 않습니다.

PDF/OCR·외부 모델·Git·SSO 등 선택 기능 전체를 이 작은 행렬에서 재시험하지는 않습니다. 기능별 보안·프로토콜 시험과 [첨부 본문 검증](attachment-extraction-guide.md), [오프라인 배포 검증](deployment-verification.md)을 함께 확인하세요.
