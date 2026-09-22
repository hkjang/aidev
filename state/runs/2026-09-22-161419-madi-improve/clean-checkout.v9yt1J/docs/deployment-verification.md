# 서비스 이미지 검증 기록과 재현 절차

이미지 생성만으로 폐쇄망 배포 성공을 판단하지 않습니다. 압축 파일을 다시 불러온 뒤 외부 통신이 차단된 Docker 내부 네트워크에서 실제 데이터 왕복을 확인합니다. PostgreSQL은 검증 인프라이며 서비스 릴리즈 아카이브에 포함하지 않습니다.

## v0.2.0 게시·검증 결과

2026-09-10 01:19:09 KST에 [v0.2.0 정식 릴리즈](https://github.com/hkjang/madi/releases/tag/v0.2.0)가 게시됐습니다. 태그 소스는 `aff3ee1f6d48f895b0d1461dafdf2711638eb0ce`이며 main CI와 [태그 릴리즈 작업 34369441095](https://github.com/hkjang/madi/actions/runs/34369441095)의 전체 3개 작업이 모두 통과했습니다. 호환성 6조합·전체 Go·shared25 재검사, 이미지 저장·재반입·폐쇄망 실행 후 서비스 이미지 아카이브 하나만 게시했습니다. 두 차례 이전 main CI 실패와 후보별 로컬 결과는 아래에 보존합니다.

| 범위 | 확인된 상태 |
| --- | --- |
| 태그 소스·웹 번들 | `v0.2.0` → `aff3ee1f6d48f895b0d1461dafdf2711638eb0ce`, `main-BPw-YbqO.js`. By2·이전 커밋 결과는 해당 소스의 기록으로 별도 보존 |
| 문서 UI 경합 전체 | BPw에서 CI 기본 조건의 `TestBrowserDocumentRaces -race -count=5` 13.97초·13.82초·13.54초·13.47초·13.78초, Go 패키지 69.648초 통과. 저장 후 목록 갱신의 지연·실패·후속 저장·화면 이탈 4경계 단독 시험 5.10초·패키지 6.161초 통과 |
| DB 편집 응답 확인 회귀 | 실제 DB commit 후 응답을 보류하는 영구 시험과 실제 200·version·읽기 셀·포커스·GET 정본 확인. 기존 고급 DB 검사 포함 PG18 3회 패키지 53.373초, PG17 1회 패키지 20.096초 통과. 제품 코드·timeout·기존 검증 완화 없음 |
| 관련 UI 교차·탐색 회귀 | AISelection 32.35초·DocumentFoundations 35.81초·UXStructure 60.62초·Worksets 21.23초, 패키지 151.096초 통과. 탐색 3회 126.997초 및 명령 경계·끝슬래시 11.644초 통과 |
| OIDC 계정 보호 회귀 | 실제 PostgreSQL `-race` 4.359초 통과 |
| 백엔드 교차 회귀 12개 | E11·OIDC·블록 조회·설정 동시성/CAS 등 117.824초 통과 |
| 현재 전체 스키마의 운영 백업·복원 | 최신 SSO 설정을 포함한 `TestNativeBackupCurrentFullSchema` 시험 2.96초·Go 패키지 4.022초 통과 |
| 대표 호환성 6개 조합 | 동일 aff3ee1f의 [main 호환성 CI](https://github.com/hkjang/madi/actions/runs/34363815488)에서 PG17.11·18.6 × Chromium·Firefox·WebKit 통과, Go 패키지 46.305초·46.326초. 태그 재검사도 6조합 통과, PG17·18 패키지 47.390초·89.407초. [호환성 가이드](compatibility-guide.md)에 각 실행과 이전 By2 진단을 구분 |
| 공유 서버 브라우저 25개 묶음 | 태그 릴리즈의 BPw 단일 배치 전부 통과·실행 전후 번들 일치·315.417초. main CI 315.618초·로컬 BPw 8차 327.867초·이전 By2 7차 409.427초는 별도 보존 |
| 수정 후 전체 Go·독립 브라우저 검사 | [main CI 34363815607](https://github.com/hkjang/madi/actions/runs/34363815607) 서버 Go-race 2,393.809초 통과. 태그 릴리즈에서도 전체 Go·35개 독립 브라우저 옵션 통과, 서버 2,372.699초·추출 검증 2.045초·배포 계약 1.024초 |
| 문서 조회 예산 후속 회귀 | 후보 권한 조회 최적화 후 PG17 전체 관련 회귀 23.514초·PG18 25.193초 통과. 기존 방식과 전체 JSON·순서·현재 권한 144개 조합 일치. PG17 경계·시간 초과·만료 시험 3회 14.576초 통과. 2초 요청 예산·1,900ms SQL 제한 유지 |
| 공개 캡처 검증 목록 | 동일 BPw 배치 정상 157장 게시·중복 4건 해결·미승인 0·95경로 캡처 연결 통과. [현재 검증 목록](screenshots/verification-v0.2.0-shared.json)은 이 157장만 기술하며 전체 갤러리를 같은 배치로 인증하지 않음. [이전 By2 목록](screenshots/verification-v0.2.0-By2nz__J.json)은 당시 시각·해시로 별도 보존. 로그인·OIDC 설정 화면 직접 시각 확인 |
| 최종 로컬 정적·의존성 검사 | BPw `go vet` 통과, 배포 계약 Go-race 1.066초·Node 26개 2.670초 통과. 이전 gofmt·라이선스 423개·웹 production 취약점 0·Go 호출 경로/가져오는 패키지 0의 근거는 보존하며 미호출 의존 모듈 취약점 1건은 별도 한계로 유지 |
| 로컬 문서 생성·정적 브라우저 검사 | 실제 게시 결과 반영 후 49매뉴얼·412화면·54사이트맵 재생성, 54페이지·537링크·390px·구조화 데이터·갤러리 필터·외부 자산 0 통과 (`.local/v020-post-release-final-docs.log`). 이전 BPw 535링크·By2 534링크·초기 531링크 기록도 보존 |
| 공개 Pages | 동일 aff3ee1f의 [Pages 34363815423](https://github.com/hkjang/madi/actions/runs/34363815423) 통과. 공개 홈·SSO 가이드·BPw 배포 기록·목록 HTTP 200, 목록의 157개 PNG 모두 실제 크기·SHA-256 일치 |
| CI 서비스 이미지 | 동일 aff3ee1f의 [offline-image 작업 102507290796](https://github.com/hkjang/madi/actions/runs/34363815607/job/102507290796) 4분 58초 통과. 대응 소스·실제 PDF/선택 OCR·복원·내부망 실행 확인. `madi:ci`의 빌드·실행 결과이며 tar 저장·재반입·릴리즈 통과는 아님 |
| 공개 v0.2.0 릴리즈 | 정식 게시 완료, draft·prerelease 아님. 자산은 `madi-v0.2.0.tar.gz` 1개. 게시 시각 `2026-09-09T16:19:09Z` |

### 실제 게시 파일

| 항목 | 확정 값 |
| --- | --- |
| 릴리즈 | [v0.2.0](https://github.com/hkjang/madi/releases/tag/v0.2.0), ID `385673963` |
| 단일 자산 | [madi-v0.2.0.tar.gz](https://github.com/hkjang/madi/releases/download/v0.2.0/madi-v0.2.0.tar.gz) |
| 크기 | 558,764,847바이트 |
| GitHub 자산 SHA-256 | `f949e323849192fd33f485bf7bc14b3807fc85c7eb1014ce9379cff68ba89ef7` |
| 원격 재반입 검증 | 이미지 저장 후 재로딩 성공, Docker 내부망에서 실제 PDF·선택 OCR·복원 포함 검증 통과 |
| 독립 실제 다운로드 | 558,764,847바이트·스트리밍 SHA-256이 GitHub digest 및 릴리즈 본문과 모두 일치, `gzip -t` 통과 |
| 다운로드 파일의 Docker manifest | 항목 1개, `RepoTags` 정확히 `["madi:v0.2.0"]` |

이 해시는 실제 게시 아카이브의 값이며 아래 CI 이미지 ID나 과거 v0.1.0 후보 해시와 다릅니다. 독립 다운로드 근거는 `.local/release-v020-verified.e2U3QZ/verification.json`입니다. 이 로컬 확인은 파일 무결성·내용 검증이며 Docker 재반입·폐쇄망 실행은 위 원격 릴리즈 작업의 실제 결과로 구분합니다.

태그 릴리즈의 shared25 원본은 단일 배치 `9082ed9a-cefd-497e-86dd-c5dc266ca66f`입니다. 시작 `2026-09-09T16:07:13.305Z`·완료 `2026-09-09T16:12:28.722Z`, 구간315.417초에 25개 모두 통과했고 evidence v2·전후 BPw·timeout 0을 확인했습니다. 원본 캡처161개의 크기·해시·시간 범위도 일치했습니다. 보고서는 `.local/release-actions-34369441095.YA50cF/release-verification/test-results/regression-shared/report.json`, SHA-256은 `f0c147418278d6e12e2ac3da1d70652e128b78302ab0f4d4dbc8672ab8ad9a18`입니다. 이 원격 캡처를 공개157장 목록의 새로운 검증 시각으로 사용하지 않습니다.

최신 원격 main CI의 shared25는 evidence v2 단일 배치 `d3809649-8a54-45e4-aee8-25469524bdcd`입니다. `2026-09-09T15:11:40.038Z`부터 `2026-09-09T15:16:55.656Z`까지 315.618초에 25개 모두 통과했고 실행 전후 BPw 번들이 일치했습니다. 전체 Go와 shared25 근거는 `.local/main-ci-34363815607-passed.log` 및 해당 CI 아티팩트에 보존합니다. 이 배치를 로컬 캡처의 새 검증 시각으로 사용하지 않습니다.

로컬 shared25 근거는 `test-results/regression-shared/report.json`의 BPw 배치 `9ae61b64-ed0c-40e6-9193-4f65482e3410`와 `.local/shared25-final-v020-8.log`입니다. 최초 시작 `2026-09-09T14:16:34.595Z`부터 마지막 완료 `2026-09-09T14:22:02.462Z`까지 327.867초이며, 25개 전부 통과·종료 코드 0·실행 전후 `main-BPw-YbqO.js`가 일치합니다. 두 배치의 시간은 각각 첫 시작~마지막 완료 구간이며 서버 준비 시간을 포함한 총 실행시간이나 서비스 응답시간 보증이 아닙니다. 공개 목록은 이 로컬 배치의 157장만 기술합니다. 이전 By2 배치 `53905e19-80e8-4867-a2d0-53778b501242`의 시작 `2026-09-09T11:57:35.782Z`·완료 `2026-09-09T12:04:25.209Z`·409.427초는 `.local/shared25-final-v020-7.log`와 별도 By2 목록에 보존하며, 기존 `verification.json`도 역사 자료로 유지합니다.

이전 전체 Go 4차 `.local/upgrade-final-go-race-4.jsonl`은 최상위 388개 통과·UXStructure 1개 실패·별도 opt-in 10개 건너뜀, 서버 패키지 1,798.478초였습니다. 추적에서 실제 경로 이탈 타이머가 원문 선택 기록을 덮어쓰는 오류와 높이 변화에 결합된 복원 오류를 확인해 수정했습니다. 후속 실제 URL·로드된 문서 명령 경계와 끝슬래시 guard도 적용했습니다. 이후 main CI 전체 통과를 확인했지만 과거 실패와 당시 실행 시간은 그대로 보존합니다.

문서 경합 전체의 최신 근거는 `test-results/document-save-races-final-ci2.log`이며, 강제 응답 보류·실패 4경계는 `test-results/document-save-refresh-positive.log`입니다. 이전 By2 2회 통과의 `test-results/navigation-command-races-ci-final.log`와 고부하·진단 계측 조건의 저장 5초 대기 실패도 보존합니다. 새 5회 통과를 모든 부하에서의 5초 지연 보장으로 해석하지 않습니다. PG17 WebKit의 동일 출처 프로필 PUT 취소 2건도 [호환성 가이드](compatibility-guide.md)에 남겼습니다. 이전 로컬 문서 검사 `.local/v020-By2nz__J-final-docs.log`와 `.local/v020-docs-final.log`는 새 서비스 이미지 검증을 대신하지 않습니다.

main CI의 실패는 `TestPostgresDocumentQueryBoundsTimeoutAndExpiredSession`에서 정상 문서 2,100개 중 최대 후보 2,000개를 조회하는 경로가 2초 예산을 넘어 HTTP 408을 반환한 사례입니다. 이 실행에서는 35개 독립 브라우저 옵션의 실패는 없었지만 전체 Go 단계가 실패해 뒤의 shared25와 이미지 단계는 건너뛰었습니다. 기존 로컬 shared25 통과와 원격 전체 실패를 구분해 보존합니다. CI 구성은 동일 커밋의 `test`와 `offline-image`를 독립 병렬 작업으로 나눠 이미지 문제도 먼저 확인하도록 변경했으며, 전체 CI 성공에는 두 작업 모두의 성공이 필요합니다. 기존 시험의 시간 제한·옵션·이미지 보안 검사는 완화하지 않았습니다.

CI의 408 자체는 로컬에서 재현되지 않았습니다. 다만 PG17에서 재귀 권한 함수의 반복 비용을 확인해 상위 문서·공간이 없는 후보에 기존 검색과 동등한 현재 사용자·워크스페이스 권한 검사를 적용했습니다. 상속 권한과 최종 ACL·원문 해시·세션·정보보호 재검증은 그대로입니다. 같은 2,100문서 진단의 후보 SQL은 PG17 통계 갱신 전/후 850.679/1,029.556ms에서 24.857/25.339ms, PG18 352.349/327.163ms에서 25.668/25.166ms로 감소했습니다. PG17 실제 정상 요청 3회는 252/235/254ms였고 강제 SQL 잠금은 기존 약 1.910초에 408, 만료 세션은 403을 반환했습니다. 이는 해당 표본의 측정이며 일반 운영 SLA나 원래 CI 실패의 직접 재현을 뜻하지 않습니다. 실행 계획의 custom/generic 모드도 비교했지만 운영 모드를 강제하지 않습니다.

두 번째 main CI의 `TestBrowserDatabaseEditing`은 수량 셀 입력의 `locator.fill` 30초 대기 시간 초과로 실패했습니다(시험 35.78초, `tests/database-editing.mjs:110`). `TestBrowserDocumentRaces`는 태그 변경 후 저장 중 상태가 저장됨으로 바뀌기를 기다리는 5초 조건에서 실패했습니다(시험 12.27초, `tests/document-races.mjs:225`). 근거는 `.local/main-ci-34356027987-failed.log`입니다. 이 실행의 후속 shared25는 건너뛰었고, [별도 호환성 작업](https://github.com/hkjang/madi/actions/runs/34356028022)과 [Pages 작업](https://github.com/hkjang/madi/actions/runs/34356027953)은 통과했지만 전체 CI 실패를 상쇄하지 않습니다.

후속 추적에서 문서 저장은 실제 canonical PUT 완료 뒤 `await reload()`의 목록 GET이 지연되어 `saving=true`가 남는 제품 오류로 확인했습니다. 목록 갱신을 오류 처리하는 백그라운드 작업으로 분리하는 15줄 수정과 현재 경로·정본·후속 저장 guard를 적용했습니다. 목록 응답 보류, 503 실패, 새 PUT 진행 중 이전 갱신 완료, 화면 이탈 뒤 늦은 오류의 4경계와 CI 기본 조건 전체 5회가 BPw 번들에서 통과했습니다. DB 편집은 활성 select의 option에 이미 ‘완료’ 문구가 있어 시험의 `toContainText`가 ACK 전에 통과한 확인 경계 오류였습니다. 실제 handler의 DB commit 뒤 ACK를 보류하는 회귀를 영구화하고 실제 응답 200·version·읽기 셀·포커스·GET 정본을 기다리도록 시험만 수정했습니다. PG18 3회 및 PG17 근거는 `.local/database-editing-after-ack-fix-count3.log`, `.local/database-editing-after-ack-fix-pg17.log`입니다. timeout과 원래 검증은 완화하지 않았습니다. CI·릴리즈의 `go test -race`에 추가한 `-failfast`는 실패 후 조기 종료를 위한 것이며 성공 시 전체 모듈·35개 옵션을 검사하는 관문은 유지합니다. 수정 후 원격 main CI와 태그 릴리즈의 전체 재검사도 통과했습니다.

[이전 독립 이미지 작업](https://github.com/hkjang/madi/actions/runs/34356027987/job/102480798995)은 당시 커밋에서 4분 48초에 통과했습니다. `.local/main-ci-34356027987-offline-image.log`에서 대응 소스·69개 APK/52개 소스 origin·모델 해시, 실제 PDF 텍스트·명시 선택한 영어/한국어 OCR·위치 조각·본문 검색·원본 불변, 논리 복원·첨부 복구·이전 세션 무효화·자동 외부 작업 중지·파생 추출 본문 폐기를 확인했습니다. CPU 2개·메모리 4GiB·read-only·비루트·내부망·네 런타임 변수 제한도 유지했습니다. 당시 이미지 ID는 `sha256:8f64ff6c431af4ca5def8cbc86616f554b62f568314a8327544789c306be26a4`이며 배포용 tar.gz의 SHA-256이 아닙니다.

aff3ee1f의 main offline-image 작업은 4분 58초에 같은 대응 소스·PDF/선택 OCR·복원·폐쇄망 실행 검사를 통과했습니다. 실제 로그는 `.local/main-ci-34363815607-offline-image.log`, CI 이미지 ID는 `sha256:ff56c043593fb805284f5a05f4088c3256fdcbe62a66d60790e76b1d7b26ad58`입니다. 이 값은 빌드·실행 검사의 이미지 ID이지 아카이브 해시가 아닙니다. 이후 태그 릴리즈는 UTC `16:12:28~16:18:32` 빌드·저장, `16:18:35~16:18:40` 재로딩, `16:18:40~16:18:53` 폐쇄망 실행 단계가 모두 성공했고, 게시 파일 값은 위 표에 별도로 기록했습니다.

이전 후보의 기능·이미지 시험과 실패 후 수정 기록은 [고도화 검증 장부](https://github.com/hkjang/madi/blob/main/OPERATIONS_UPGRADE.md)에 구분합니다. 아래 v0.1.0 표의 크기·해시는 과거 로컬 후보 값이며 v0.2.0 반입 검증에 사용할 수 없습니다.

## 자동 검증

```sh
# 인터넷 연결 빌드 환경에서 실행합니다. 외부 GitHub 게시 작업은 하지 않습니다.
bash scripts/release-image.sh
gzip -t dist/madi-v0.2.0.tar.gz
gzip -dc dist/madi-v0.2.0.tar.gz | docker load
bash scripts/verify-image.sh madi:v0.2.0
```

`release-image.sh`는 `VERSION`을 검증하여 `madi:v버전` 하나를 저장합니다. 임시 아카이브 압축 검증 후에만 `dist/madi-v버전.tar.gz`로 교체하며 SHA-256을 출력합니다. GitHub 업로드는 별도의 태그 릴리즈 작업에서 모든 검증이 통과한 뒤 수행합니다. 사용자 첨부 릴리즈 자산은 서비스 이미지 압축 파일 하나입니다.

검증 도구에는 Docker, Go, Bash, OpenSSL, gzip이 필요합니다. Go로 만든 작은 검증 클라이언트는 **별도 테스트 컨테이너에만** 읽기 전용 마운트되며 서비스 이미지에 추가되지 않습니다. 검증용 PostgreSQL 이미지는 연결망에서 미리 준비할 수 있습니다. 스크립트는 생성한 컨테이너·전용 네트워크·익명 볼륨만 정리하고 기존 서비스에는 접근하지 않습니다.

## 확인 항목

- 서비스에 전달하는 설정은 네 개: `POSTGRES_DSN`, `BOOTSTRAP_ADMIN`, `BOOTSTRAP_ADMIN_PASSWORD`, `ENCRYPTION_KEY`.
- `--internal` 네트워크, UID/GID 10001, 읽기 전용 루트, 전체 capability 제거, 권한 상승 금지.
- 서비스 CPU 2개·메모리 4GiB 제한과 최대 2GiB 임시 공간.
- readiness, 버전 정보, 로그인 화면과 번들 JavaScript·CSS·한국어 폰트 로드.
- Bootstrap 관리자 로그인, 워크스페이스·문서 생성/수정, 첨부파일 바이트 일치.
- 실제 PostgreSQL 작업 큐가 생성한 Markdown 내보내기와 원문 바이트 일치.
- 같은 이미지에 포함된 런타임 대응 소스·라이선스·해시 목록과 영어·한국어 OCR 모델.
- 실제 PDF 텍스트와 사용자가 선택한 빈 페이지의 영어·한국어 OCR, 위치·본문 검색·원본 파일 불변.
- 전체 논리 백업 후 문서 변경→복원, 문서·첨부 복구, 이전 로그인 쿠키 무효화.
- 복원 후 자동 작업 정책이 중지되어 과거 외부 작업을 자동 재실행하지 않는지 확인.

위 목록은 자동 검증이 수행하는 항목입니다. 새 버전의 실행 로그 없이 이전 버전의 통과 결과를 그대로 적용하지 않습니다.

복원된 관리자 계정이 현재 요청자와 일치하면 응답에서 새 세션을 발급할 수 있습니다. 검증은 새 세션과 별개로 **복원 이전 쿠키가 401로 거부되는지** 확인합니다.

## PostgreSQL 운영 백업 별도 검증

논리 ZIP 복원과 별도로 현재 전체 스키마의 `pg_dump`·`pg_restore` 경로도 검증합니다. 운영 DB와 분리된 `_test` 데이터베이스 URI, 데이터베이스 생성 권한, 해당 PostgreSQL 서버 버전 이상인 `pg_dump` 클라이언트가 필요합니다.

```sh
MADI_TEST_POSTGRES_DSN='postgres://tester@127.0.0.1:5432/madi_test?sslmode=disable' \
  MADI_NATIVE_BACKUP_FULL=1 \
  go test ./internal/server -run '^TestNativeBackupCurrentFullSchema$' -count=1 -v
```

이 opt-in 시험은 임의 이름의 두 데이터베이스를 만들고 현재 전체 migration을 적용합니다. 서비스 작업자를 시작하지 않은 상태에서 운영 백업 스크립트로 저장·복원하여 Markdown·첨부 바이트·암호화 설정의 일치를 검사합니다. 복원 세션 제거, Git·커넥터·알림·IMAP·Runbook·공개 링크·내보내기·지원 진단·OTLP 실행 정책 중지와 검색 재색인 대기도 확인합니다. 종료 시 해당 시험이 생성한 두 DB만 삭제하며 기존 서비스에는 접근하지 않습니다. 2026-09-09 PostgreSQL 18.6에서 최신 SSO 설정을 포함한 전체 스키마의 백업·복원 시험이 2.96초, Go 패키지 실행이 4.022초에 통과했습니다. 이 결과는 Docker 재반입 시험이나 실제 운영 데이터의 복구 시간을 대신하지 않습니다.

영구 정책·처리 이력·근거 메타데이터와 재생성 가능한 색인을 구분합니다. 검색·벡터·첨부 추출 본문의 파생 테이블은 논리 백업 대상에서 제외하며, native 백업에 들어 있더라도 복원 후 폐기하거나 무효화합니다. 외부 제공자로 보내는 색인은 과거 동의만으로 자동 전송하지 않고 새 명시 동의가 필요합니다.

운영 스크립트의 자동 첨부 백업 대상은 기본 로컬 저장소입니다. 추가 로컬 provider나 S3/MinIO 객체가 있으면 불완전한 백업을 만들지 않고 거부합니다. 이 경우 모든 provider 루트·버킷·접두사를 포함하는 조직의 일관된 스냅샷 절차를 사용하거나, 크기 제한 내에서 모든 provider 파일을 모으는 madi 논리 백업을 사용하세요. `ENCRYPTION_KEY`는 두 백업 방식 모두 별도 비밀 저장소에 보관합니다.

## 과거 v0.1.0 로컬 이미지 검증

2026-09-08 WSL2 Linux/amd64, CPU 4개·메모리 약 15.6GiB의 공유 개발 호스트에서 Docker 29.8.0 rootless/overlay2와 PostgreSQL 17 컨테이너로 당시 제공한 기동·문서·첨부·내보내기·복구 항목을 실행했습니다. 초기 `v0.1.0` 후보는 약 65MiB 이미지였으며, JavaScript 2개·폰트 124개와 시험 데이터 복구가 통과했습니다. 최종 릴리즈의 크기·해시는 최종 소스 재빌드 결과를 사용해야 합니다.

당시 화면 수정과 Go 1.26.8 보안 패치를 포함한 **v0.1.0 로컬 검증 후보**는 다음과 같습니다. `madi:v0.1.0` 한 개만 포함한 아카이브를 만들고 기존 검증 이미지를 삭제한 뒤 다시 불러와 당시의 전체 이미지 시험을 재통과했습니다. v0.2.0에서 추가된 PDF/OCR·폰트 대응 소스 검증은 이 과거 결과에 포함되지 않습니다.

| 항목 | 로컬 검증 후보 |
| --- | --- |
| 웹 번들 | `main-B2_oEw5_.js` |
| 이미지 크기 | 69,347,594 bytes |
| 아카이브 | `madi-v0.1.0.tar.gz` |
| 아카이브 크기 | 25,699,223 bytes |
| SHA-256 | `6a4aa2539cc1fefe6f827c0df666dd68b5734f15396ade1c7a1c169730ef3dde` |

**이 해시를 실제 GitHub 릴리즈 파일의 해시로 간주하면 안 됩니다.** 릴리즈 작업은 검증한 소스를 다시 빌드하므로 이미지 생성 메타데이터 등에 따라 압축 파일 해시가 달라질 수 있습니다. 워크플로는 이미지 재반입과 폐쇄망 시험을 통과한 실제 게시 파일의 크기·SHA-256을 릴리즈 노트 본문에 자동 기록합니다. 운영망 반입 시에는 해당 릴리즈 본문의 값을 대조하세요. 체크섬 파일이나 PostgreSQL 이미지를 추가 릴리즈 자산으로 첨부하지 않습니다.

기존 개발 호스트 Docker Desktop의 API 오류 때문에 검증 전용 rootless 엔진을 사용했으며, 기존 데몬·컨테이너·WSL은 재시작하지 않았습니다. 공식 패키지를 전용 경로에 추출하고 해시를 대조했으며, namespace 초기화에 사용한 두 UID 매핑 helper의 setuid 비트는 초기화 후 해제했습니다. 이는 해당 개발 호스트의 검증 절차일 뿐, 운영 서비스에 rootless helper나 추가 환경변수를 요구하지 않습니다. 설치 조건은 [Docker rootless 문서](https://docs.docker.com/engine/security/rootless/)를 참고하고, 수동 정적 엔진은 자동 보안 업데이트가 없는 테스트용 선택임에 유의하세요. [Docker 정적 바이너리 안내](https://docs.docker.com/engine/install/binaries/).

이 결과는 단일 복제본 기능·복구 검증입니다. 대규모 성능, 실제 사내 인증서 체인, 조직별 Keycloak/AD·외부 시스템 제품 버전의 수용 시험을 대체하지 않습니다. [규모 측정 가이드](scale-verification.md)에서 부하 시험의 별도 결과와 제한을 확인하세요.
