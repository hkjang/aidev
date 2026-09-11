---
title: "cutover — 자율 개선 이력"
description: "cutover: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음."
last_modified_at: 2026-09-12 01:21:41 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "SoftwareSourceCode",
 "name": "cutover",
 "codeRepository": "https://github.com/hkjang/cutover",
 "url": "https://hkjang.github.io/aidev/projects/cutover/",
 "description": "cutover: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음.",
 "inLanguage": "ko",
 "maintainer": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-12T01:21:41+09:00"
}
</script>

# cutover

<p class="tldr"><strong>요약.</strong> cutover: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음. <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0">건강 B</span> <span class="meta">14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0</span></p>

<ul class="stats"><li><b>1</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>0</b><span>배포 준비 완료</span></li><li><b>0</b><span>릴리즈 진행 중</span></li><li><b>0</b><span>병합 완료</span></li><li><b>1</b><span>검토 대기</span></li><li><b>0</b><span>검증 실패</span></li><li><b>0</b><span>변경 없음</span></li><li><b>0</b><span>실행 오류</span></li><li><b>$6.11</b><span>비용</span></li><li><b>16분</b><span>에이전트 시간</span></li></ul>

## 현황

<dl class="kv">
<dt>저장소</dt><dd><a href="https://github.com/hkjang/cutover">https://github.com/hkjang/cutover</a></dd>
<dt>마지막 회차</dt><dd>2026-09-11 20:31 KST — <span class="pill pill-other">• 기타</span> CI no-ci, PR open <a href="https://github.com/hkjang/cutover/pull/1">PR #1</a></dd>
</dl>

## 회차 이력

<div class="table-wrap"><table class="rt"><thead><tr><th>일시</th><th class="primary">프로젝트</th><th>결과</th></tr></thead><tbody><tr data-status="other"><td data-label="일시">2026-09-11 20:31</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/cutover/">cutover</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> CI no-ci, PR open <a href="https://github.com/hkjang/cutover/pull/1">PR #1</a><div class="meta">24파일 <span style="color:var(--good)">+677</span>/<span style="color:var(--bad)">−192</span> · 테스트 1 — docs: 사용자·관리자 가이드를 실제 화면 캡처가 들어간 표준 형식으로 재작성하고 PDF 추가</div></td></tr></tbody></table></div>

## 비용·사용량

<div class="table-wrap"><table class="rt"><caption class="meta">최근 30세션</caption><thead><tr><th>시각</th><th class="primary">프로젝트</th><th>단계</th><th class="num">시간</th><th class="num">턴</th><th class="num">비용</th><th class="num">토큰 입력/출력</th><th>종료</th></tr></thead><tbody><tr data-status="other"><td data-label="시각">20:26</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/cutover/">cutover</a></td><td data-label="단계">review</td><td data-label="시간" class="num">1분</td><td data-label="턴" class="num">14</td><td data-label="비용" class="num">$0.71</td><td data-label="토큰 입력/출력" class="num">498K / 6K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">20:25</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/cutover/">cutover</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">14분</td><td data-label="턴" class="num">63</td><td data-label="비용" class="num">$5.40</td><td data-label="토큰 입력/출력" class="num">5.0M / 56K</td><td data-label="종료">success</td></tr></tbody></table></div>

## 아이디어 백로그 — 대기 5 / 전체 6

<div class="table-wrap"><table class="rt"><caption class="meta">에이전트가 회차마다 재평가한다. 가치 높고 위험 낮은 대기 항목이 다음 회차 후보다.</caption><thead><tr><th class="primary">아이디어</th><th>가치/위험/크기</th><th>상태</th><th>메모</th><th>갱신</th></tr></thead><tbody><tr data-status="nochange"><td data-label="아이디어" class="primary">운영 빌드 관리자 세션 쿠키의 Secure 속성을 설정 가능하게 (HTTP 사내망 배포에서 관리자 로그인 불가)</td><td data-label="가치/위험/크기">5/2/S</td><td data-label="상태">대기</td><td data-label="메모">lib/adminSession·login route 의 secure: NODE_ENV===&#x27;production&#x27; 때문에 Docker 이미지를 http://&lt;LAN IP&gt;:3000 으로 쓰면 브라우저가 쿠키를 버려 PUT 이 401, 새로고침 시 로그인 화면. 2026-09-11 실측 확인. 환경 변수(예: ADMIN_COOKIE_SECURE) 또는 x-forwarded-proto 기반으로 제어하고 e2e 추가.</td><td data-label="갱신">2026-09-11</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">삭제·상태 전파의 하위 탐색을 ID 접두사 대신 parentId 기반으로 변경</td><td data-label="가치/위험/크기">4/2/S</td><td data-label="상태">대기</td><td data-label="메모">lib/treeUtils deleteActivity 와 app/admin/page.tsx onStatusChange 가 id.startsWith(parent+&#x27;-&#x27;) 로 하위를 찾는다. 업로드한 JSON 의 자식 ID 가 접두사를 따르지 않으면 삭제 시 고아가 남아 validateActivityImport 400 으로 조용히 실패. 관리자 가이드 6장에 증상으로 기록됨.</td><td data-label="갱신">2026-09-11</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">관리자 콘솔 PUT 실패를 화면에 표시하고 삭제 확인창 중복 제거</td><td data-label="가치/위험/크기">3/1/S</td><td data-label="상태">대기</td><td data-label="메모">ActivityTree 삭제 버튼과 admin page onDeleteActivity 가 각각 confirm 을 띄워 두 번 뜨고, 두 번째는 루트 node.title 을 표시. PUT 응답(400/401)을 무시해 실패가 보이지 않음. 제목 100자 초과 시에도 조용히 닫힘.</td><td data-label="갱신">2026-09-11</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">validateActivityImport 단위 테스트 추가 (node --test)</td><td data-label="가치/위험/크기">3/1/S</td><td data-label="상태">대기</td><td data-label="메모">현재 검증 로직은 e2e 한 건으로만 간접 검증됨. 중복 ID·순환·level 불일치·50건 생략 메시지·진행중 변환을 빠른 단위 테스트로 고정.</td><td data-label="갱신">2026-09-11</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">미사용 코드 정리: lib/s3.ts, amplify.yml, *_bak/*_local 파일, reset-visitor API</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">lib/s3.ts 는 어떤 라우트도 import 하지 않음(_bak 만). visitorCount 는 증가 코드가 없어 reset-visitor 도 의미 없음. 정리 시 README 아키텍처 그림의 S3 언급도 함께 수정.</td><td data-label="갱신">2026-09-11</td></tr><tr data-status="released"><td data-label="아이디어" class="primary">사용자·관리자 가이드를 실제 화면 캡처가 들어간 표준 형식으로 완성 (캠페인 guides-2026-09)</td><td data-label="가치/위험/크기">5/1/M</td><td data-label="상태">완료</td><td data-label="메모">docs/USER_GUIDE.md·pdf, docs/ADMIN_GUIDE.md·pdf, docs/assets/guide/*.png 13장, scripts/guide-screenshots (npm run screenshots:guide). 커밋 18cfb20.</td><td data-label="갱신">2026-09-11</td></tr></tbody></table></div>

## 원장 (에이전트가 남긴 기록)

## 2026-09-11
- 선택: 사용자·관리자 가이드를 실제 화면 캡처가 들어간 표준 형식으로 완성 (가치 5 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: 기존 docs/USER_GUIDE.md·ADMIN_GUIDE.md(캡처 0장, 일부 내용이 코드와 불일치)를 GUIDE-STANDARD 구성으로 다시 쓰고, Playwright 기반 캡처 스크립트(scripts/guide-screenshots, 격리 데이터 파일 + 자체 dev 서버 + GUIDE_SHOT_* 전용 환경 변수)로 실제 화면 13장을 1440x900으로 찍어 docs/assets/guide/에 실었으며 공용 md2pdf로 두 PDF를 만들었다. 환경 변수 표·API 메서드·연동 규칙은 코드에서 읽어 작성했고, 운영 빌드(NODE_ENV=production)에서 Secure 쿠키 때문에 평문 HTTP LAN 주소로는 관리자 세션이 유지되지 않는 문제를 실측(LAN IP 401 / 127.0.0.1 200)으로 확인해 문서에 경고로 넣었다. 검증: eslint, tsc --noEmit, 기존 e2e 2건 통과, 캡처 스크립트 5건 통과, PDF 페이지를 렌더해 표지·표·그림 확인.
- 보류 아이디어: (1) 운영 빌드 관리자 세션 쿠키 Secure 속성을 환경 변수나 x-forwarded-proto 로 제어해 HTTP 사내망 배포에서도 관리자 로그인이 되게 한다 — 가치 5 / 위험 2 / S. (2) 삭제·상태 전파가 ID 접두사(`<부모ID>-`)에 의존해 업로드한 JSON 의 자식이 고아로 남아 삭제가 400 으로 조용히 실패한다 — parentId 기반 하위 탐색으로 바꾼다 — 가치 4 / 위험 2 / S. (3) 관리자 콘솔의 PUT 실패(400/401)를 화면에 표시하고 삭제 확인창이 두 번 뜨는 중복을 제거한다 — 가치 3 / 위험 1 / S. (4) 미사용 lib/s3.ts·amplify.yml·*_bak 파일과 사용되지 않는 reset-visitor API 정리 — 가치 2 / 위험 1 / S. (5) lib/activityData.ts validateActivityImport 단위 테스트(Node test runner) 추가 — 가치 3 / 위험 1 / S.


[← 대시보드](https://hkjang.github.io/aidev/) · [교훈 모음](https://hkjang.github.io/aidev/lessons/)

{% endraw %}
