---
title: "SecCheck — 자율 개선 이력"
description: "SecCheck: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음."
last_modified_at: 2026-09-11 20:31:33 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "SoftwareSourceCode",
 "name": "SecCheck",
 "codeRepository": "https://github.com/hkjang/SecCheck",
 "url": "https://hkjang.github.io/aidev/projects/SecCheck/",
 "description": "SecCheck: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음.",
 "inLanguage": "ko",
 "maintainer": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-11T20:31:33+09:00"
}
</script>

# SecCheck

<p class="tldr"><strong>요약.</strong> SecCheck: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음. <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0">건강 B</span> <span class="meta">14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0</span></p>

<ul class="stats"><li><b>1</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>0</b><span>배포 준비 완료</span></li><li><b>0</b><span>릴리즈 진행 중</span></li><li><b>0</b><span>병합 완료</span></li><li><b>0</b><span>검토 대기</span></li><li><b>1</b><span>검증 실패</span></li><li><b>0</b><span>변경 없음</span></li><li><b>0</b><span>실행 오류</span></li><li><b>$12.66</b><span>비용</span></li><li><b>22분</b><span>에이전트 시간</span></li></ul>

## 현황

<dl class="kv">
<dt>저장소</dt><dd><a href="https://github.com/hkjang/SecCheck">https://github.com/hkjang/SecCheck</a></dd>
<dt>마지막 회차</dt><dd>2026-09-11 20:03 KST — <span class="pill pill-other">• 기타</span> verify failed: 실패한 검증: [ -d node_modules ] || npm ci --no-audit --no-fund (exit 1)</dd>
</dl>

## 회차 이력

<div class="table-wrap"><table class="rt"><thead><tr><th>일시</th><th class="primary">프로젝트</th><th>결과</th></tr></thead><tbody><tr data-status="other"><td data-label="일시">2026-09-11 20:03</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/SecCheck/">SecCheck</a></td><td data-label="결과"><span class="pill pill-failed" title="outcome=verify-failed">검증 실패</span> verify failed: 실패한 검증: [ -d node_modules ] || npm ci --no-audit --no-fund (exit 1)<div class="meta">75파일 <span style="color:var(--good)">+1048</span>/<span style="color:var(--bad)">−369</span> · <em>테스트 없음</em> — Put the real screens into the guides people are handed</div></td></tr></tbody></table></div>

## 비용·사용량

<div class="table-wrap"><table class="rt"><caption class="meta">최근 30세션</caption><thead><tr><th>시각</th><th class="primary">프로젝트</th><th>단계</th><th class="num">시간</th><th class="num">턴</th><th class="num">비용</th><th class="num">토큰 입력/출력</th><th>종료</th></tr></thead><tbody><tr data-status="other"><td data-label="시각">20:03</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/SecCheck/">SecCheck</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">22분</td><td data-label="턴" class="num">110</td><td data-label="비용" class="num">$12.66</td><td data-label="토큰 입력/출력" class="num">16.7M / 81K</td><td data-label="종료">success</td></tr></tbody></table></div>

## 아이디어 백로그 — 대기 5 / 전체 6

<div class="table-wrap"><table class="rt"><caption class="meta">에이전트가 회차마다 재평가한다. 가치 높고 위험 낮은 대기 항목이 다음 회차 후보다.</caption><thead><tr><th class="primary">아이디어</th><th>가치/위험/크기</th><th>상태</th><th>메모</th><th>갱신</th></tr></thead><tbody><tr data-status="nochange"><td data-label="아이디어" class="primary">승인 프로세스를 켠 상태의 반려·승인 대기·결재 화면을 추가 촬영해 승인자 절 보강</td><td data-label="가치/위험/크기">3/1/S</td><td data-label="상태">대기</td><td data-label="메모">capture_all.js 에 approval_enabled 를 읽고-복원하는 단계와 approve/reject 시드를 더하면 된다. 현재 사용자 가이드 4-4 절은 캡처 없이 글로만 설명.</td><td data-label="갱신">2026-09-11</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">가이드 PDF 생성을 릴리즈 워크플로에 붙여 md 와 pdf 가 어긋나지 않게 하기</td><td data-label="가치/위험/크기">3/2/M</td><td data-label="상태">대기</td><td data-label="메모">현재는 손으로 md2pdf 를 돌려야 한다. 캡처는 실제 서버가 필요하므로 PDF 변환만 CI 에 넣고 캡처는 수동으로 두는 절충이 현실적.</td><td data-label="갱신">2026-09-11</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">docs/features.md 와 seccheck_features_guide.pdf 를 새 캡처 파일명 기준으로 재생성</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">features.md 의 이미지 경로는 새 이름으로 바꿨으나 PDF 는 옛 캡처로 만든 것이 그대로 남아 있다. scripts/generate_pdf.js 로 재생성.</td><td data-label="갱신">2026-09-11</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">scripts/generate_pdf.js 를 공용 md2pdf 로 대체해 저장소 전용 변환기 제거</td><td data-label="가치/위험/크기">2/2/S</td><td data-label="상태">대기</td><td data-label="메모">표준은 저장소마다 변환기를 두지 말라고 한다. 남은 features·api·architecture·complete manual 4종을 공용 도구로 옮기면 generate_pdf.js 와 web 의존 puppeteer 경로를 지울 수 있다.</td><td data-label="갱신">2026-09-11</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">캡처 스크립트에 시드 데이터 정리 옵션 추가</td><td data-label="가치/위험/크기">2/2/S</td><td data-label="상태">대기</td><td data-label="메모">현재는 사용자·심의를 만들고 지우지 않는다(버려도 되는 설치 전제). 심의 취소 API 와 사용자 비활성화로 되돌리는 --cleanup 을 두면 반복 실행이 쉬워진다.</td><td data-label="갱신">2026-09-11</td></tr><tr data-status="released"><td data-label="아이디어" class="primary">사용자·관리자 가이드를 GUIDE-STANDARD 구조로 재작성하고 실제 화면 캡처 34장을 싣기</td><td data-label="가치/위험/크기">5/2/M</td><td data-label="상태">완료</td><td data-label="메모">docs/USER_GUIDE.md·pdf, docs/ADMIN_GUIDE.md·pdf 생성. capture_all.js 를 env 필수·loopback 가드·설정 복원 방식으로 고쳐 docs/screenshots 에 34장 촬영. 옛 가이드는 대체 안내로 새 문서를 가리킴.</td><td data-label="갱신">2026-09-11</td></tr></tbody></table></div>

## 원장 (에이전트가 남긴 기록)

## 2026-09-11
- 선택: 사용자·관리자 가이드를 GUIDE-STANDARD 구조로 재작성하고 실제 화면 캡처 34장을 싣기 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: docs/USER_GUIDE.md·pdf, docs/ADMIN_GUIDE.md·pdf 를 표준 구성(처음 5분·화면별·자주 하는 작업·막혔을 때·용어 / 구성 요소·설치·환경 변수 전수 표·역할·운영·장애 대응·보안)으로 새로 썼고, scripts/capture_all.js 를 전용 환경 변수(URL·계정·시드 비밀번호) 필수화 + loopback 가드 + workflow·security 설정 읽고-복원 방식으로 고쳐 로컬 Postgres + go build 서버에 데모 데이터를 시드한 뒤 Playwright Chromium 1440x900 으로 docs/screenshots/*.png 34장을 찍었다. 옛 user-guide.md·admin-guide.md 는 "대체되었습니다" 안내로 새 문서를 가리키고, README·docs/README.md·docs/index.html·features.md 의 링크와 캡처 파일명을 새 이름으로 바꾸고 옛 PDF 2종은 지웠다. 검증: 캡처 스크립트 실행 성공(설정 복원 확인), md2pdf 로 PDF 2종 생성(23·29쪽, 이미지 15·18장), 참조 이미지 전수 존재 확인, node --check, CI 와 같은 gitleaks 스캔 "no leaks found". 환경 변수 표는 internal/app/config.go 와 selftest.go 에서, 설정 기본값은 마이그레이션 SQL 에서, API 메서드는 server.go 라우트 등록에서 읽어 적었다.
- 보류 아이디어: 캡처 스크립트로 반려·승인 대기·승인자 결재 화면(approval_enabled 켠 상태)까지 추가 촬영해 승인자 절 보강 (가치 3 / 위험 1 / S); docs/features.md·seccheck_features_guide.pdf 도 새 캡처 파일명 기준으로 재생성 (가치 2 / 위험 1 / S); scripts/generate_pdf.js 를 공용 md2pdf 로 대체해 저장소 전용 변환기 제거 (가치 2 / 위험 2 / S); GUIDE PDF 생성을 CI 릴리즈 단계에 붙여 md 와 pdf 가 어긋나지 않게 하기 (가치 3 / 위험 2 / M); 캡처 스크립트에 시드 데이터 정리(생성한 사용자·심의 삭제) 옵션 추가 (가치 2 / 위험 2 / S)


[← 대시보드](https://hkjang.github.io/aidev/) · [교훈 모음](https://hkjang.github.io/aidev/lessons/)

{% endraw %}
