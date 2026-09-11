---
title: "Momento — 자율 개선 이력"
description: "Momento: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음."
last_modified_at: 2026-09-11 20:03:21 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "SoftwareSourceCode",
 "name": "Momento",
 "codeRepository": "https://github.com/hkjang/Momento",
 "url": "https://hkjang.github.io/aidev/projects/Momento/",
 "description": "Momento: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음.",
 "inLanguage": "ko",
 "maintainer": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-11T20:03:21+09:00"
}
</script>

# Momento

<p class="tldr"><strong>요약.</strong> Momento: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음. <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 1, 경고 1, 회귀 0">건강 C</span> <span class="meta">14일: 릴리즈 0, 실패 1, 경고 1, 회귀 0</span></p>

<ul class="stats"><li><b>1</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>0</b><span>배포 준비 완료</span></li><li><b>0</b><span>릴리즈 진행 중</span></li><li><b>0</b><span>병합 완료</span></li><li><b>0</b><span>검토 대기</span></li><li><b>1</b><span>검증 실패</span></li><li><b>0</b><span>변경 없음</span></li><li><b>0</b><span>실행 오류</span></li><li><b>$13.11</b><span>비용</span></li><li><b>32분</b><span>에이전트 시간</span></li></ul>

## 현황

<dl class="kv">
<dt>저장소</dt><dd><a href="https://github.com/hkjang/Momento">https://github.com/hkjang/Momento</a></dd>
<dt>마지막 회차</dt><dd>2026-09-11 19:35 KST — <span class="pill pill-failed">❌ 실패</span> <strong>CI failed</strong>, PR open <a href="https://github.com/hkjang/Momento/pull/1">PR #1</a></dd>
</dl>

## 회차 이력

<div class="table-wrap"><table class="rt"><thead><tr><th>일시</th><th class="primary">프로젝트</th><th>결과</th></tr></thead><tbody><tr data-status="failed"><td data-label="일시">2026-09-11 19:35</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/Momento/">Momento</a></td><td data-label="결과"><span class="pill pill-failed" title="outcome=verify-failed">검증 실패</span> <strong>CI failed</strong>, PR open <a href="https://github.com/hkjang/Momento/pull/1">PR #1</a><div class="meta">64파일 <span style="color:var(--good)">+2321</span>/<span style="color:var(--bad)">−692</span> · <em>테스트 없음</em> — docs: 사용자·관리자 가이드를 실제 화면 캡처가 실린 완성본으로 다시 쓴다</div></td></tr></tbody></table></div>

## 비용·사용량

<div class="table-wrap"><table class="rt"><caption class="meta">최근 30세션</caption><thead><tr><th>시각</th><th class="primary">프로젝트</th><th>단계</th><th class="num">시간</th><th class="num">턴</th><th class="num">비용</th><th class="num">토큰 입력/출력</th><th>종료</th></tr></thead><tbody><tr data-status="other"><td data-label="시각">19:34</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/Momento/">Momento</a></td><td data-label="단계">review</td><td data-label="시간" class="num">3분</td><td data-label="턴" class="num">26</td><td data-label="비용" class="num">$1.52</td><td data-label="토큰 입력/출력" class="num">1.4M / 9K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">19:30</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/Momento/">Momento</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">29분</td><td data-label="턴" class="num">121</td><td data-label="비용" class="num">$11.59</td><td data-label="토큰 입력/출력" class="num">14.8M / 84K</td><td data-label="종료">success</td></tr></tbody></table></div>

## 아이디어 백로그 — 대기 5 / 전체 6

<div class="table-wrap"><table class="rt"><caption class="meta">에이전트가 회차마다 재평가한다. 가치 높고 위험 낮은 대기 항목이 다음 회차 후보다.</caption><thead><tr><th class="primary">아이디어</th><th>가치/위험/크기</th><th>상태</th><th>메모</th><th>갱신</th></tr></thead><tbody><tr data-status="nochange"><td data-label="아이디어" class="primary">User Explorer 타임라인의 365일 요청이 기본 쿼리 정책(180일)에 막힘</td><td data-label="가치/위험/크기">4/2/S</td><td data-label="상태">대기</td><td data-label="메모">UserExplorerPage.tsx 가 rangeQuery(365)로 고정 요청. 기본 max_exact_days=180 인 사이트에서는 추적 버튼을 누르면 RANGE_EXCEEDS_POLICY. 사이트 정책 한도(query-policy)를 읽어 min(365, 한도)로 요청하도록 수정</td><td data-label="갱신">2026-09-11</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">가이드 캡처 스크립트를 CI 에서 릴리즈 tarball 로 돌려 화면 어긋남 검사</td><td data-label="가치/위험/크기">3/2/M</td><td data-label="상태">대기</td><td data-label="메모">scripts/guide 를 release 워크플로 뒤에 붙여 캡처가 실패하면(셀렉터·문구 변경) 알린다. 그림 자동 갱신은 하지 않음</td><td data-label="갱신">2026-09-11</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">대량 과거 이벤트 반입 직후 aggregate maintenance deadlock(40P01) 실패</td><td data-label="가치/위험/크기">3/3/M</td><td data-label="상태">대기</td><td data-label="메모">시드로 70일치 late event 를 넣자 late_event 재집계 작업 17건이 deadlock detected 로 failed. Worker 의 일별 집계 upsert 와 재집계의 잠금 순서 정리 또는 40P01 재시도</td><td data-label="갱신">2026-09-11</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">compose.yml 에 릴리즈 이미지로 띄울 때 --no-build 가 필요하다는 주석</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">build: 가 있어 소스 트리에서 docker compose up 하면 적재한 릴리즈 이미지 대신 다시 빌드함. 관리자 가이드 2.3 에 문서화함</td><td data-label="갱신">2026-09-11</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">관리자 가이드 4.1 역할 표를 라우터 순회 테스트로 고정</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">admin/orgAdmin 미들웨어 대상 경로 목록을 테스트가 출력해 문서 표와 diff 하면 권한 변경이 문서에서 조용히 어긋나지 않음</td><td data-label="갱신">2026-09-11</td></tr><tr data-status="released"><td data-label="아이디어" class="primary">사용자·관리자 가이드를 실제 화면 캡처가 실린 완성본으로 재구성</td><td data-label="가치/위험/크기">5/1/M</td><td data-label="상태">완료</td><td data-label="메모">guides-2026-09 캠페인. 49장 캡처, 두 PDF, scripts/guide 시드·캡처 도구, 옛 HTML 가이드 정리</td><td data-label="갱신">2026-09-11</td></tr></tbody></table></div>

## 원장 (에이전트가 남긴 기록)

## 2026-09-11
- 선택: 사용자·관리자 가이드를 실제 화면 캡처가 실린 완성본으로 재구성 (가치 5 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: docs/USER_GUIDE.md·ADMIN_GUIDE.md 를 GUIDE-STANDARD 구성(처음 5분·화면별 사용법·막혔을 때 / 구성 요소·설치·환경 변수 전수 표·계정과 권한·운영·장애 대응·보안)으로 다시 쓰고, v0.34.39 를 로컬 Postgres 와 함께 실제로 띄워 seed.mjs 로 가짜 데이터(데모 포털, EMP0001, hong@example.com)를 넣은 뒤 headless Chrome 1440x900 으로 49장을 찍어 docs/assets/guide/ 에 실었다. 환경 변수 표는 internal/config·internal/database 에서, 역할 표는 라우터 미들웨어에서, 오류 문구는 writeError/queryError.ts 에서 읽었고, 옛 USER_GUIDE.html/ADMIN_GUIDE.html 은 삭제하고 docs/index*.html·README 링크를 새 문서로 옮겼다. 공용 md2pdf 로 두 PDF(47쪽/27쪽)를 생성해 표지·그림이 렌더링되는 것과 문서가 참조하는 그림이 모두 존재하는 것을 확인했다. scripts/guide 는 전용 환경 변수만 받고 로컬 호스트가 아니면 멈추며 --cleanup 으로 만든 것만 지운다; 비밀값은 글자로 적지 않았다.
- 보류 아이디어: User Explorer 타임라인이 365일을 요청해 기본 정책(180일)에서 RANGE_EXCEEDS_POLICY 로 막힘 — 사이트 정책 한도를 읽어 요청하도록 수정 (가치 4 / 위험 2 / S); 대량 과거 이벤트 반입 직후 aggregate maintenance 가 deadlock(40P01)으로 실패하는 재집계 작업 — 재시도 또는 잠금 순서 정리 (가치 3 / 위험 3 / M); compose.yml 의 build: 때문에 릴리즈 이미지로 띄울 때 --no-build 가 필요 — 문서화했으나 compose 파일에 주석 추가 (가치 2 / 위험 1 / S); 캡처 스크립트를 CI 에서 릴리즈 tarball 로 돌려 가이드 그림이 최신 화면과 어긋나는지 검사 (가치 3 / 위험 2 / M).


[← 대시보드](https://hkjang.github.io/aidev/) · [교훈 모음](https://hkjang.github.io/aidev/lessons/)

{% endraw %}
