---
title: "seaton — 자율 개선 이력"
description: "seaton: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음."
last_modified_at: 2026-09-12 08:28:03 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "SoftwareSourceCode",
 "name": "seaton",
 "codeRepository": "https://github.com/hkjang/seaton",
 "url": "https://hkjang.github.io/aidev/projects/seaton/",
 "description": "seaton: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음.",
 "inLanguage": "ko",
 "maintainer": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-12T08:28:03+09:00"
}
</script>

# seaton

<p class="tldr"><strong>요약.</strong> seaton: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음. <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0">건강 B</span> <span class="meta">14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0</span></p>

<ul class="stats"><li><b>1</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>0</b><span>배포 준비 완료</span></li><li><b>0</b><span>릴리즈 진행 중</span></li><li><b>0</b><span>병합 완료</span></li><li><b>1</b><span>검토 대기</span></li><li><b>0</b><span>검증 실패</span></li><li><b>0</b><span>변경 없음</span></li><li><b>0</b><span>실행 오류</span></li><li><b>$11.88</b><span>비용</span></li><li><b>22분</b><span>에이전트 시간</span></li></ul>

## 현황

<dl class="kv">
<dt>저장소</dt><dd><a href="https://github.com/hkjang/seaton">https://github.com/hkjang/seaton</a></dd>
<dt>마지막 회차</dt><dd>2026-09-11 21:02 KST — <span class="pill pill-other">• 기타</span> review held, PR open <a href="https://github.com/hkjang/seaton/pull/25">PR #25</a></dd>
</dl>

## 회차 이력

<div class="table-wrap"><table class="rt"><thead><tr><th>일시</th><th class="primary">프로젝트</th><th>결과</th></tr></thead><tbody><tr data-status="other"><td data-label="일시">2026-09-11 21:02</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/seaton/">seaton</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> review held, PR open <a href="https://github.com/hkjang/seaton/pull/25">PR #25</a><div class="meta">31파일 <span style="color:var(--good)">+1214</span>/<span style="color:var(--bad)">−402</span> · <em>테스트 없음</em> — docs: 사용자·관리자 가이드를 실제 화면 캡처와 함께 표준 구성으로 정비</div></td></tr></tbody></table></div>

## 비용·사용량

<div class="table-wrap"><table class="rt"><caption class="meta">최근 30세션</caption><thead><tr><th>시각</th><th class="primary">프로젝트</th><th>단계</th><th class="num">시간</th><th class="num">턴</th><th class="num">비용</th><th class="num">토큰 입력/출력</th><th>종료</th></tr></thead><tbody><tr data-status="other"><td data-label="시각">21:02</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/seaton/">seaton</a></td><td data-label="단계">review</td><td data-label="시간" class="num">3분</td><td data-label="턴" class="num">22</td><td data-label="비용" class="num">$1.03</td><td data-label="토큰 입력/출력" class="num">883K / 9K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">20:59</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/seaton/">seaton</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">18분</td><td data-label="턴" class="num">105</td><td data-label="비용" class="num">$10.85</td><td data-label="토큰 입력/출력" class="num">13.9M / 67K</td><td data-label="종료">success</td></tr></tbody></table></div>

## 아이디어 백로그 — 대기 5 / 전체 6

<div class="table-wrap"><table class="rt"><caption class="meta">에이전트가 회차마다 재평가한다. 가치 높고 위험 낮은 대기 항목이 다음 회차 후보다.</caption><thead><tr><th class="primary">아이디어</th><th>가치/위험/크기</th><th>상태</th><th>메모</th><th>갱신</th></tr></thead><tbody><tr data-status="nochange"><td data-label="아이디어" class="primary">사용자 권한 화면에 계정 비활성화 토글 추가</td><td data-label="가치/위험/크기">3/2/S</td><td data-label="상태">대기</td><td data-label="메모">PATCH /api/v1/users/{id} active 는 있으나 UsersPage 에 UI 가 없어 관리자 가이드가 API 를 안내함</td><td data-label="갱신">2026-09-11</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">좌석 상세 편집 창에서 조직 구역 지정</td><td data-label="가치/위험/크기">3/2/S</td><td data-label="상태">대기</td><td data-label="메모">seats PATCH organizationId 는 있으나 화면은 처리필요의 영역 맞춤으로만 바꿀 수 있음</td><td data-label="갱신">2026-09-11</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">좌석맵에서 배정 해제 단추</td><td data-label="가치/위험/크기">3/2/S</td><td data-label="상태">대기</td><td data-label="메모">DELETE /api/v1/seat-assignments/{seatID} 를 쓰는 UI 가 없어 퇴직자 일괄 해제 외에는 API 로만 가능</td><td data-label="갱신">2026-09-11</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">build-docs.py 파서 단위 테스트</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">이미지 블록·번호 목록 이어 세기(start) 회귀 방지</td><td data-label="갱신">2026-09-11</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">CI 에 guide-shots 스모크 추가</td><td data-label="가치/위험/크기">2/2/S</td><td data-label="상태">대기</td><td data-label="메모">e2e 잡 뒤에 guide-shots.mjs 를 돌려 캡처 스크립트가 셀렉터 변경으로 깨지지 않는지 확인. 산출물은 커밋하지 않음</td><td data-label="갱신">2026-09-11</td></tr><tr data-status="released"><td data-label="아이디어" class="primary">사용자·관리자 가이드를 실제 화면 캡처와 함께 GUIDE-STANDARD 구성으로 정비</td><td data-label="가치/위험/크기">5/1/M</td><td data-label="상태">완료</td><td data-label="메모">guide-shots.mjs 22장 캡처, USER/ADMIN_GUIDE md+pdf(md2pdf.mjs), build-docs.py 그림 지원</td><td data-label="갱신">2026-09-11</td></tr></tbody></table></div>

## 원장 (에이전트가 남긴 기록)

## 2026-09-11
- 선택: 사용자·관리자 가이드를 실제 화면 캡처와 함께 GUIDE-STANDARD 구성으로 정비 (가치 5 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: 릴리즈와 같은 방법(docker build → postgres + compose 환경변수)으로 SeatOn v1.4.0을 띄우고 e2e 시드로 가짜 데이터를 채운 뒤, 새 `web/e2e/guide-shots.mjs`(Playwright, 1440x900, 캡처 전용 환경변수·로컬 주소 가드·API 키 원문 가림과 폐기)로 22장을 찍어 `docs/assets/guide/`에 실었다. USER_GUIDE는 처음 5분·화면별 사용법·자주 하는 작업·막혔을 때(서버·프런트 실제 문구)·용어로, ADMIN_GUIDE는 구성 요소·설치·환경변수 3개와 설정 키 32개 전수 표·계정과 권한·운영(healthz/readyz·로그·백업·복구·업그레이드)·장애 대응(실제 로그 문구)·보안으로 다시 썼고 기존 엔진 실측 절은 유지했다. PDF는 공용 md2pdf.mjs로 굽고(21쪽/21쪽, 표지·표·코드·그림 확인) build-docs.py는 두 가이드의 HTML만 만들며 그림·번호 목록 이어 세기를 지원하도록 고쳤다. 검증: tsc, vitest 76건, go test/vet, Playwright login/admin/api-keys 11건 통과. 부서 관리자 역할은 코드상 직원과 동일함을 그대로 적었고 화면에 계정 생성·비활성화가 없다는 점도 API 경로와 함께 명시했다.
- 보류 아이디어: 사용자 권한 화면에 계정 비활성화 토글 추가(API는 있으나 UI 없음) / 좌석 상세 편집 창에서 조직 구역 지정(현재 API·처리필요로만 가능) / 좌석맵에서 배정 해제 단추(현재 DELETE API·퇴직자 해제만) / CI에 guide-shots 스모크(캡처 스크립트가 깨지지 않는지) 추가 / build-docs.py 파서 단위 테스트(이미지·목록 번호 이어 세기)


[← 대시보드](https://hkjang.github.io/aidev/) · [교훈 모음](https://hkjang.github.io/aidev/lessons/)

{% endraw %}
