# 회차 노트 2026-10-04-104220-Momento-improve — Momento
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:42] base pinned — main@ec7cb66
- [러너 10:42] autonomy release — 

## 정찰 노트
- 배정된 우선 과제(「CI failed, PR #23, 릴리즈 워크플로 두 번 실패」)는 **이미 해결돼 있었다**: f539c2f 가 고치고 PR #23 은 3d18b32 로 머지, ec7cb66(v0.34.55)의 체크 4개 전부 success, 두 락파일에 braces/micromatch 0건. release.json 의 `github_release:false` 는 v0.34.40 이후 모든 릴리즈가 같은 값이라 실패 신호가 아니다 — 「두 번 실패」를 뒷받침하는 증거는 로컬에 없었다. 그래서 금지된 완화(`--audit-level`) 대신 1순위 보류 아이디어를 집었다.
- 고른 것: 망 구분 추가 Alert 한국어화. 사용자(v0.34.54)·사이트(v0.34.55)가 같은 틀로 연속 통과했고 파일 3개로 끝난다. 「숨긴 열 검색」·「CSV format」·「rowKey 중복」을 제친 이유는 셋 다 '의도일 수 있음/관측 없음' 이 다섯~여덟 회차째 그대로라서다(두 개는 rejected 로 내렸다).
- 과제서의 추측 하나: `cidr` 컬럼 타입이 `10.0.0.5/24` 를 거절해 500 이 될 가능성 — **재현하지 않았다.** 그래서 기준 1 의 표기 안내를 재현 불필요한 `INVALID_CIDR`(400) 쪽에만 달게 적었다.
- 구현자가 조심할 것: `network_ranges` 에 UNIQUE 가 없으므로(001_initial.sql:64-70) 500 을 "중복" 이라 설명하면 거짓이다. 공용 상수 세 문장은 재사용만 하고 수정하지 말 것(기존 13건이 글자 그대로 단언한다). `.ts` 확장자는 값 import 에도 필요하다.
- 이 세션은 `npm ci`·`gh` 가 둘 다 막혀 **명령을 하나도 실행하지 못했다** — 전부 읽기로만 확인했다. 프로필(2026-10-01)은 읽어 본 범위에서 현재 코드와 어긋나지 않아 새로 쓰지 않았다.
- [러너 10:48] scout done — 망 구분(네트워크) 추가 실패를 서버의 영문 문장·pgx 원문 대신 한국어로 안내한다 (가치 3 / 위험 1 / 작업

## 구현 노트
- 배정된 우선 과제는 이미 해결돼 있었다(f539c2f → PR #23 머지 → v0.34.55 전 체크 success; 두 락파일에 braces·micromatch 0건을 직접 grep 으로 확인). 고칠 원인이 없어 과제서 1순위를 했다: `adminErrors.ts` 에 `describeNetworkError` 신설 + `AdminPage.tsx:3244-3246` Alert 을 `NetworkErrorAlert` 로 교체 (프로덕션 2 + 테스트 1).
- **확신 없는 곳·검증 못 한 것**: (1) `release.yml` 런 이력을 **못 봤다** — `gh` 가 이 세션도 미인증이다. 배정이 말한 「두 번 실패」가 거기에 실재하면 이번 회차는 그것을 다루지 않았다. (2) `network_ranges.cidr` 가 Postgres `cidr` 타입이라 `net.ParseCIDR` 이 통과시킨 `10.0.0.5/24` 를 거절하는지 **DB 로 재현하지 않았다**(Postgres 없음, DSN 비었음). 그래서 500 문구를 중립으로 뒀고, 그 꼴의 원문은 '믿을 수 없는 입력' 으로만 테스트에 넣어 본문 유출 여부만 묶었다 — 그 시나리오가 실재한다는 주장은 어디에도 하지 않았다.
- **일부러 하지 않은 것**: `npm audit` 게이트 완화(금지) · `.github/workflows/*` · `internal/httpapi/admin.go`(서버 검사가 정본) · `describeUserError`·`describeSiteError`·`AdminNoticeAlert`·공용 상수 세 문장(기존 13건이 글자 그대로 단언한다 — 재사용만 했다) · 나머지 Alert 들(핸들러 단위로 쪼갠 선례 유지). CIDR 의 화면 선검증도 뺐다 — 이번 과제는 문구만이다. ideas.json 에 신규 후보로 적었다.
- **과제서와 다르게 한 것 하나**: 기준 1 의 `INVALID_CIDR` 안내를 과제서 예시인 "넷마스크 경계" 대신 **비트 수 표기**로 적었다. `net.ParseCIDR` 은 넷마스크 오른쪽 비트를 통과시키므로 그 400 이 경계 때문에 날 수 없고, 그렇게 쓰면 실제로 틀린 것(비트 수 누락·범위 초과)을 못 가리킨다.
- **다음 역할이 조심할 것**: 새 테스트 8건은 DB 도 브라우저도 필요 없다(`cd web && node --test test/adminErrors.test.mjs`, 199/199). 브라우저 확인에 쓴 하네스는 /tmp 에만 있고 커밋에 없다 — 재현하려면 `/api/v1/me` 의 role 을 **`super_admin`** 으로 주고(`owner` 는 유효한 역할이 아니다) `/api/v1/sites` 에 사이트 하나를 꼭 넣어야 한다(비면 관리 센터가 「사이트를 생성하세요」 만 띄워 섹션을 안 그린다); 이름 칸 라벨은 「망 구분」 이 아니라 **「망 구분 명」** 이다. `web/dist` 는 지웠고 `git status` 로 3파일만 담긴 것을 확인했다. `npx prettier --check` 는 게이트로 쓰지 말 것.
- [러너 10:57] brief accepted — 채택 — 인용한 행 번호(AdminPage.tsx:3244-3246 Alert·3250 disabled·78 import·3343-3345 SiteErrorAlert, admin.go:800·809·813·823, adminErrors.ts:2
- [러너 10:57] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 3파일 diff 전체, admin.go:800-826(코드 셋 — INVALID_PAYLOAD·INVALID_CIDR·NETWORK_CREATE_FAILED 가 전부라는 주장 확인), 001_initial.sql 의 UNIQUE 부재, client.ts:81-85 의 APIError 꼴, AdminPage.tsx 의 CIDR placeholder(10.20.30.0/24 — 안내 예시와 일치), describeSiteError 선례. 직접 실행: `npm run lint` 클린, `npm test` 199/199, `git status` 클린(dist·하네스 없음).
- 기준 1 통과: 원장의 `- 실패 재현:` 에 항등 스텁에서 새 8건 중 5건이 떨어진 실제 출력이 있고 기존 13건은 스텁에서도 통과한다 — 테스트가 이번 변경 경로만 짚는다. 브라우저 쪽 복원 재실행(4건 실패 → 5/5)도 증상과 맞는다.
- 못 본 것: release.yml 런 이력(gh 미인증 — 배정의 「두 번 실패」 는 이 세션도 확인 불가), Postgres `cidr` 거절 재현(DB 없음). 둘 다 이번 변경의 동작이 의존하지 않는다.
- 남는 우려(차단 아님): 403 FORBIDDEN 영문이 default 로 새는 경로가 그대로다 — 선례와 동일한 공백이지만 커밋 제목이 그 경로까지 포함하는 것처럼 읽힌다. 「망 대역」/「망 구분」 용어 혼재는 기존 화면을 따른 것.
- security·legal 차단 없음: 새 엔드포인트·인가 변경·개인정보 수집 없음이고, pgx 원문 노출은 본문→detail 로 좁아졌다(화면은 orgAdmin 전용).
- [러너 11:00] review approved — 리뷰 승인 (risk=low)
- [러너 11:00] pr created — https://github.com/hkjang/Momento/pull/24
- [러너 11:06] ci passed — 검사 1개 모두 success
- [러너 11:06] merge done — 49b6d2c
- [러너 11:24] release published — v0.34.56
- [러너 11:26] assets verified — v0.34.56 자산 2개 (이전 v0.34.55: 2)
