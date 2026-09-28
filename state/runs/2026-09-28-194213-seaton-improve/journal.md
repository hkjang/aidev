# 회차 노트 2026-09-28-194213-seaton-improve — seaton
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:42] base pinned — main@7502f82
- [러너 19:42] autonomy release — 

## 정찰 노트
- 조직 필터를 골랐다: 서버가 이미 `organizationId` 를 받는데 화면만 안 보내는 확정된 공백이라 프로덕션 2파일로 닫히고, 시드가 3조직·10명이라 E2E 로 곧장 증명된다. `rows.Err()` 삼킴(3/2/M)은 파일 10개 안팎이라 6파일 규칙에, 500건 total 은 서버 COUNT+501명이 필요해 45분에 걸린다.
- 확인한 것: employees.go:62,69(서버가 org 를 거름), EmployeesPage.tsx:58-61(화면이 안 보냄), server.go:82(조직 GET 은 인증만), seed.mjs:14-30(개발6·영업2·인사2), UsersPage.tsx:260 ↔ admin.spec.ts:150(MUI Select 의 inputProps aria-label 이 getByRole 로 실제 잡힘).
- 추측으로 적은 것: 조직이 0개인 설치에서 Select 가 어떻게 보이는지는 확인하지 않았다(과제서에 미확인으로 표기). 코드는 실행하지 않았고 테스트도 돌리지 않았다.
- 구현자가 조심할 것: 기존 두 Select 에 aria-label 을 더할 때 **보이는 텍스트를 바꾸면** employee-export.spec.ts:52-55 의 `filter({hasText:"전체 배정상태"})` 가 멎는다. 그리고 주소 반영(?organizationId=)·total 은 범위 밖으로 명시했으니 끌어들이지 말 것.
- 프로필은 기준이 v1.4.8 로 낡아 있어 v1.4.11(7502f82) 기준으로 새로 썼다(씨드 구성·MUI Select 접근 이름·employees 쿼리 계약을 새로 담음).
- [러너 19:46] scout done — 직원 화면에 조직 필터 노출 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 서버가 이미 거르는 `organizationId` 를 화면이 보내지 않던 공백을 닫았다. 프로덕션은 2파일 — `web/src/lib/employeeQuery.ts`(신규, 쿼리 키·빈 값 규칙 한 곳) + `EmployeesPage.tsx`(조직 Select·상태·`load` 네 번째 인자). 서버·SQL·마이그레이션·워크플로 미접촉.
- 확신 없는 곳: **조직이 0개인 설치**(시드 전)의 화면을 확인하지 못했다. 시드가 항상 조직 3개를 만들어 그 상태를 E2E 로 만들 수 없었고, `organizations.map` 이 빈 배열이면 `전체 조직` 한 항목만 남는다는 것까지만 코드로 읽었다. 조직 로드 실패 경로(`catch` → Select 만 빔)도 실제로 실패시켜 보지는 않았다 — 목록 조회와 독립인 것은 코드로만 확인했다.
- 일부러 하지 않은 것: 주소(`?organizationId=`) 반영과 `total`·페이지네이션. 과제서가 범위 밖으로 명시했고, 전자는 상태 네 개의 직렬화 규칙을, 후자는 서버 `COUNT` 와 501명을 새로 요구한다. 기존 두 Select 의 **보이는 텍스트는 바꾸지 않았다** — `employee-export.spec.ts:52-55` 가 `hasText:"전체 배정상태"` 로 집는다. USER_GUIDE 의 스크린샷 캡션도 그대로 뒀다(PNG 를 재촬영하지 않았으므로 캡션이 없는 필터를 말하면 거짓이 된다).
- 다음 역할이 조심할 것: `web/e2e/employee-filter.spec.ts` 3건은 **Docker 이미지 + PostgreSQL 16 + 시드**가 있어야 돈다. `getByRole("button", { name: "검색" })` 은 AppShell 의 "직원 빠른 검색" 까지 잡혀 strict 위반이므로 `exact: true` 가 필수다(그래서 붙였다).
- 준비 함정 하나: `/api/v1/health` 는 **없다**(404). 컨테이너 기동 대기를 그것으로 걸면 무한정 돈다 — `GET /` 의 200 으로 확인할 것. 이번에 여기서 10분을 잃었다.
- E2E 기준선: 전체 71건 중 69건 통과. 실패 2건은 `tracking.spec.ts:110`(Momento 프록시 수집기)·`mcp-oauth.spec.ts:247`(가짜 Keycloak) 뿐이고, 변경 전 이미지에서도 같은 2건이 같은 이유로 실패하는 것을 이번에 실제로 돌려 확인했다.
- [러너 20:20] brief accepted — 채택 — 과제서가 지목한 자리(employees.go:59-69 의 org 필터, EmployeesPage.tsx:58-61 의 누락, server.go:82 의 인증만 요구, seed.mjs 의 
- [러너 20:20] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: diff 전부(프런트 2 + 테스트 2 + 문서 2), employees.go:11-83(조직 GET 이 `items` 를 항상 싣고 organizationId 가 `$2` 로 바인딩됨), EmployeesPage.tsx 의 load 호출 8자리·counts·exportCsv 가 모두 `items` 에서 파생, e2e 의 combobox 로케이터 6곳, 사전순 선행 spec 이 직원을 만들지 않음(api-keys:68 은 403, seat-detail:69 는 복원 upsert). 직접 실행: vitest 8/8, `tsc -b`, prettier 통과.
- 못 본 것: E2E 3건은 돌리지 않았다(Docker + PG16 필요). 구현자의 변경 전 이미지 재현 출력(`조직 필터` combobox 대기 타임아웃 3건)을 증상 일치로 받아들였다. 조직 0개 설치도 실행으로는 확인 못 했다 — 서버가 빈 `items` 를 보장하므로 크래시 경로는 없다는 코드 확인까지만.
- 승인이어도 남는 우려: `employee-filter.spec.ts:38` 이 시드 인원 10을 상수로 못박아 seed.mjs 의 PEOPLE 이 늘면 먼저 깨진다. `docs/assets/guide/admin-employees.png` 에는 새 필터가 없어 다음 캡처 회차 대상이다. USER_GUIDE 의 "팀별로 나누면 500 제한에 걸리지 않습니다" 는 500명 넘는 팀에서 참이 아니다 — 릴리즈 노트에 500 상한·total 부재가 그대로임을 적어 둘 것.
- 보안·법무: 새 엔드포인트·권한 확대·비밀값·개인정보 신규 수집 없음. 관리자가 이미 보던 목록을 좁히는 읽기 전용 필터이므로 차단 사유 없음.
- [러너 20:24] review approved — 리뷰 승인 (risk=low)
- [러너 20:24] pr created — https://github.com/hkjang/seaton/pull/38
- [러너 20:29] ci passed — 검사 2개 모두 success
- [러너 20:29] merge done — 59574c9

## 릴리즈 노트
- v1.4.12 로 냈다. 이전 세 회차(v1.4.9·v1.4.10·v1.4.11)와 똑같은 꼴 — 패치 한 칸, 문서 버전 표기만 고치는 `docs: v1.4.12 기준으로 문서 정비` 커밋(6ae2417) + 같은 본문을 담은 주석 태그 `v1.4.12`. 손댄 파일 10개, 32/32 줄로 직전 세 릴리즈와 완전히 같은 모양이다.
- 버전 문자열이 박힌 곳은 README.md·docs/ADMIN_GUIDE.md·docs/USER_GUIDE.md·docs/ROADMAP_PLAN.md 네 곳뿐이고 모두 고쳤다(`grep 1.4.11` 잔여 0). `web/package.json` 의 `1.0.0` 은 릴리즈마다 올리지 않는 관례라 그대로 뒀고, 애플리케이션 버전은 Dockerfile 의 `ARG VERSION` 으로 빌드 때 주입되므로 코드에 박힌 값이 없다.
- 산출물: `scripts/build-docs.py ADMIN_GUIDE USER_GUIDE ROADMAP_PLAN` 으로 HTML 3개 + ROADMAP_PLAN.pdf, 가이드 PDF 2개는 README 가 지정한 공용 변환기(`aidev/tools/guide/md2pdf.mjs`, `--version v1.4.12`)로 구웠다. 09-28 기능 회차가 미뤄 둔 USER_GUIDE.pdf 가 이번에 조직 필터 설명을 담아 다시 구워졌다(2328777 → 2332038 바이트).
- 자산은 붙이지 않는다(`assets: []`). `release.yml` 이 `v*.*.*` 태그 푸시에 반응해 `seaton:v1.4.12` 를 빌드하고 `SeatOn-v1.4.12.tar.gz` 를 `gh release create --generate-notes --title "SeatOn v1.4.12"` 로 직접 올리므로 사람이 만들 자산이 없고, GitHub Release 도 워크플로가 만든다(`github_release: false`).
- 검증: `go vet`·`go test ./...` 통과, `npm ci`·vitest 150건 전부 통과, `tsc -b && vite build` 통과, `git diff --check` 깨끗. README 의 릴리즈 로컬 검증도 그대로 재현했다 — `bash scripts/release-image.sh 1.4.12` 로 linux/amd64 이미지를 빌드해 `SeatOn-v1.4.12.tar.gz`(47,628,395바이트) 를 만들고 `gzip -t` 통과를 본 뒤 지웠다(워크플로가 같은 것을 다시 만든다). 스크립트 파일 모드가 644 라 `./` 로는 `Permission denied` 가 나므로 `bash` 로 불러야 한다 — 다음 릴리즈 담당자가 같은 데서 멈추지 말 것.
- 비평이 남긴 숙제 하나는 릴리즈 본문에 담지 못했다: "조직 필터로 팀을 나누면 500명 상한에 걸리지 않는다" 는 USER_GUIDE 문장이 **500명 넘는 팀에서는 참이 아니고**, `total`·더 보기는 여전히 없다. 이 저장소의 Release 본문은 세 회차 모두 `--generate-notes` 자동 생성이어서 사람이 쓴 문단을 넣는 자리가 없다 — 관례를 깨지 않기로 하고 여기에 적어 둔다. 다음 회차의 '500건 상한에 total·더 보기' 아이디어가 이 문장의 근거를 실제로 만들어 줄 후보다.
- 런치 등급은 3단계(개선)로 봤다. 관리자가 이미 보던 목록을 좁히는 읽기 전용 필터이고 새 권한·새 엔드포인트·가격/패키징 변화가 없어, 사용자 가이드 한 문장과 릴리즈 노트 말고 대외 공지·영업 브리핑을 붙일 일이 아니다. 화면 캡처(`docs/assets/guide/admin-employees.png`)에 새 필터가 아직 없는 것이 이 등급에서 남는 유일한 부채이고, 다음 캡처 회차 대상이다.
- [러너 20:43] release published — v1.4.12
- [러너 20:45] assets verified — v1.4.12 자산 1개 (이전 v1.4.11: 1)
