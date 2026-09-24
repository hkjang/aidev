---
title: "aiportal-front — 자율 개선 이력"
description: "aiportal-front: 자율 개선 회차 81회, 릴리즈 0건. 최근 릴리즈 없음."
last_modified_at: 2026-09-24 12:13:28 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "SoftwareSourceCode",
 "name": "aiportal-front",
 "codeRepository": "https://github.com/hkjang/aiportal-front",
 "url": "https://hkjang.github.io/aidev/projects/aiportal-front/",
 "description": "aiportal-front: 자율 개선 회차 81회, 릴리즈 0건. 최근 릴리즈 없음.",
 "inLanguage": "ko",
 "maintainer": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-24T12:13:28+09:00"
}
</script>

# aiportal-front

<p class="tldr"><strong>요약.</strong> aiportal-front: 자율 개선 회차 81회, 릴리즈 0건. 최근 릴리즈 없음. <span class="pill pill-failed" title="14일: 릴리즈 0, 실패 0, 경고 20, 회귀 0">건강 D</span> <span class="meta">14일: 릴리즈 0, 실패 0, 경고 20, 회귀 0</span></p>

<ul class="stats"><li><b>81</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>0</b><span>배포 준비 완료</span></li><li><b>0</b><span>릴리즈 진행 중</span></li><li><b>36</b><span>병합 완료</span></li><li><b>6</b><span>검토 대기</span></li><li><b>1</b><span>검증 실패</span></li><li><b>35</b><span>변경 없음</span></li><li><b>3</b><span>실행 오류</span></li><li><b>$172.03</b><span>비용</span></li><li><b>7시간 9분</b><span>에이전트 시간</span></li></ul>

## 현황

<dl class="kv">
<dt>저장소</dt><dd><a href="https://github.com/hkjang/aiportal-front">https://github.com/hkjang/aiportal-front</a></dd>
<dt>마지막 회차</dt><dd>2026-09-24 11:16 KST — <span class="pill pill-merged">✅ 머지</span> fix-round: merged <a href="https://github.com/hkjang/aiportal-front/pull/38">PR #38</a>, <strong>release failed</strong></dd>
<dt>최근 릴리즈</dt><dd>failed — failed</dd>
<dt>사유</dt><dd>릴리즈 관례를 정할 근거가 없어 다음 버전을 결정할 수 없다(27회째 동일 교착). 이번 회차에 6개 항목을 직접 재실측함: (1) git tag --sort=-creatordate 출력 0건, 태그 총 0개 (2) package.json version=0.0.0, package-lock.json version 및 packages[&quot;&quot;].version=0.0.0 — 최초 커밋 ab700ba 의 package.json 도 0.0.0 이라 증가 이력이 전무 (3) CHANGELOG 파일 없음 (4) 릴리즈 노트 없음 — docs/RELEASE.md 는 스스로 &#x27;릴리즈 노트나 새 릴리즈 정책이 아니며 다음 버전·태그·릴리즈 커밋 관례를 정하지 않는다&#x27; 고 명시 (5) git log --oneline -60 에 릴리즈 커밋 메시지 양식 0건 (6) .github 디렉터리 부재, .gitlab-ci.yml 은 CI_COMMIT_TAG 참조 0건인 main/develop 브랜치 조건 배포 전용이며 태그 기반 버전 증가나 Release 생성을 정의하지 않음. 러너가 미리 가져온 GitHub Release 목록도 비어 있어 과거 릴리즈 자산 규칙도 없다(assets 는 빈 배열). 절차 5 의 skipped 조건(&#x27;태그도, 버전 파일도, 릴리즈 노트도 없음&#x27;)은 package.json 이라는 버전 파일이 실재하므로 충족하지 않고, 절차 2 의 &#x27;최근 릴리즈들의 증가 패턴&#x27; 은 릴리즈가 0건이라 따를 패턴 자체가 없다. 따라서 released 로 진행하려면 관례를 새로 발명해야 하는데 이는 사람의 일이므로 하지 않았다. 버전 파일·태그·CHANGELOG·릴리즈 노트·릴리즈 커밋을 일체 만들지 않았고 원격에 아무것도 보내지 않았으며 docs/RELEASE.md 도 수정하지 않았다(워크트리 clean 유지). 러너 지시(&#x27;워크플로 파일과 실패한 단계의 스크립트·테스트를 읽고 원인을 고치세요&#x27;)는 수행 대상이 없다: 이 저장소에 워크플로 파일이 없고(.github 부재) 실패 지점이 워크트리 밖의 외부 aidev bin/run.sh -&gt; release.json -&gt; bin/gate.py 이며, 게이트를 느슨하게 만드는 우회(version 필드 삭제 등)는 금지라 하지 않았다. 해소에 필요한 사람 입력 4가지(직전 26회차와 동일): (1) 첫 릴리즈 버전을 0.0.1 / 0.1.0 / 1.0.0 중 무엇으로 할지 — README 와 docs/01-시작하기.md 에 &#x27;0.0.1 / 2025-01-01 / 초기 릴리스&#x27; 기재가 있으나 실제 릴리스 증거는 미확인 (2) 태그 형식(v1.2.3 / 1.2.3)과 주석 태그 여부 (3) 릴리즈 노트의 위치·양식·언어(CHANGELOG.md 신설 여부 포함) (4) GitHub Release 사용 여부 — 현재 배포는 GitLab CI 브랜치 배포뿐이다.</dd>
<dt>수정 과제</dt><dd>⚠️ 오류 대응(자동 적재): 릴리즈 실패(). 릴리즈 관례를 정할 근거가 없어 다음 버전을 결정할 수 없다(27회째 동일 교착). 이번 회차에 6개 항목을 직접 재실측함: (1) git tag --sort=-creatordate 출력 0건, 태그 총 0개 (2) package.json version=0.0.0, package-lock.json version 및 packages[&quot;&quot;].version=0.0.0 — 최초 커밋 ab700ba 의 package.json 도 0.0.0 이라 증가 이력이 전무 (3) CHANGELOG 파일 없음 (4) 릴리즈 노트 없음 — docs/RELEASE.md 는 스스로 &#x27;릴리즈 노트나 새 릴리즈 정책이 아니며 다음 버전·태그·릴리즈 커밋 관례를 정하지 않는다&#x27; 고 명시 (5) git log --oneline -60 에 릴리즈 커밋 메시지 양식 0건 (6) .github 디렉터리 부재, .gitlab-ci.yml 은 CI_COMMIT_TAG 참조 0건인 main/develop 브랜치 조건 배포 전용이며 태그 기반 버전 증가나 Release 생성을 정의하지 않음. 러너가 미리 가져온 GitHub Release 목록도 비어 있어 과거 릴리즈 자산 규칙도 없다(assets 는 빈 배열). 절차 5 의 skipped 조건(&#x27;태그도, 버전 파일도, 릴리즈 노트도 없음&#x27;)은 package.json 이라는 버전 파일이 실재하므로 충족하지 않고, 절차 2 의 &#x27;최근 릴리즈들의 증가 패턴&#x27; 은 릴리즈가 0건이라 따를 패턴 자체가 없다. 따라서 released 로 진행하려면 관례를 새로 발명해야 하는데 이는 사람의 일이므로 하지 않았다. 버전 파일·태그·CHANGELOG·릴리즈 노트·릴리즈 커밋을 일체 만들지 않았고 원격에 아무것도 보내지 않았으며 docs/RELEASE.md 도 수정하지 않았다(워크트리 clean 유지). 러너 지시(&#x27;워크플로 파일과 실패한 단계의 스크립트·테스트를 읽고 원인을 고치세요&#x27;)는 수행 대상이 없다: 이 저장소에 워크플로 파일이 없고(.github 부재) 실패 지점이 워크트리 밖의 외부 aidev bin/run.sh -&gt; release.json -&gt; bin/gate.py 이며, 게이트를 느슨하게 만드는 우회(version 필드 삭제 등)는 금지라 하지 않았다. 해소에 필요한 사람 입력 4가지(직전 26회차와 동일): (1) 첫 릴리즈 버전을 0.0.1 / 0.1.0 / 1.0.0 중 무엇으로 할지 — README 와 docs/01-시작하기.md 에 &#x27;0.0.1 / 2025-01-01 / 초기 릴리스&#x27; 기재가 있으나 실제 릴리스 증거는 미확인 (2) 태그 형식(v1.2.3 / 1.2.3)과 주석 태그 여부 (3) 릴리즈 노트의 위치·양식·언어(CHANGELOG.md 신설 여부 포함) (4) GitHub Release 사용 여부 — 현재 배포는 GitLab CI 브랜치 배포뿐이다.</dd>
</dl>

## 회차 이력

<div class="table-wrap"><table class="rt" data-filter="1"><thead><tr><th>일시</th><th class="primary">프로젝트</th><th>결과</th></tr></thead><tbody><tr data-status="merged"><td data-label="일시">2026-09-24 11:16</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> fix-round: merged <a href="https://github.com/hkjang/aiportal-front/pull/38">PR #38</a>, <strong>release failed</strong><div class="meta">2파일 <span style="color:var(--good)">+389</span>/<span style="color:var(--bad)">−3</span> · 테스트 1 — fix: 위젯 설정 저장 응답 대기 중 편집·재저장으로 최신 편집이 유실되던 문제 수정</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-24 09:35</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> fix-round: merged <a href="https://github.com/hkjang/aiportal-front/pull/37">PR #37</a>, <strong>release failed</strong><div class="meta">2파일 <span style="color:var(--good)">+226</span>/<span style="color:var(--bad)">−1</span> · 테스트 1 — fix: 앱 공유 리스트 조회 실패가 &#x27;없습니다&#x27; 문구로 감춰지던 문제 수정</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-24 07:40</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> fix-round: merged <a href="https://github.com/hkjang/aiportal-front/pull/36">PR #36</a>, <strong>release failed</strong><div class="meta">2파일 <span style="color:var(--good)">+246</span>/<span style="color:var(--bad)">−2</span> · 테스트 1 — fix: 프롬프트 선택 팝업 검색이 배경 게시판 검색창을 읽던 문제 수정</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-24 06:48</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> fix-round: merged <a href="https://github.com/hkjang/aiportal-front/pull/35">PR #35</a>, <strong>release failed</strong><div class="meta">2파일 <span style="color:var(--good)">+236</span>/<span style="color:var(--bad)">−1</span> · 테스트 1 — fix: 요청 게시판 검색이 무동작이던 문제 수정</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-24 05:25</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> fix-round: merged <a href="https://github.com/hkjang/aiportal-front/pull/34">PR #34</a>, <strong>release failed</strong><div class="meta">4파일 <span style="color:var(--good)">+267</span>/<span style="color:var(--bad)">−1</span> · 테스트 1 — fix: 게시판 목록 3형제의 조회 실패가 &#x27;없습니다&#x27; 문구로 감춰지던 문제 수정</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-24 05:00</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> fix-round: merged <a href="https://github.com/hkjang/aiportal-front/pull/33">PR #33</a>, <strong>release failed</strong><div class="meta">4파일 <span style="color:var(--good)">+270</span>/<span style="color:var(--bad)">−3</span> · 테스트 1 — fix: 게시판 상세 조회 실패가 빈 자리표시자 화면으로 감춰지던 문제 수정</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-24 03:21</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> fix-round: merged <a href="https://github.com/hkjang/aiportal-front/pull/32">PR #32</a>, <strong>release failed</strong><div class="meta">2파일 <span style="color:var(--good)">+227</span>/<span style="color:var(--bad)">−0</span> · 테스트 1 — fix: 사이드메뉴 대화 이력 조회 실패가 &#x27;기록 없음&#x27; 으로 감춰지던 문제 수정</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-24 01:47</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> fix-round: merged <a href="https://github.com/hkjang/aiportal-front/pull/31">PR #31</a>, <strong>release failed</strong><div class="meta">2파일 <span style="color:var(--good)">+258</span>/<span style="color:var(--bad)">−0</span> · 테스트 1 — fix: 대화 삭제 실패 시 전역 Alert 과 토스트가 겹쳐 안내되던 문제 수정</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-24 00:43</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> fix-round: merged <a href="https://github.com/hkjang/aiportal-front/pull/30">PR #30</a>, <strong>release failed</strong><div class="meta">2파일 <span style="color:var(--good)">+217</span>/<span style="color:var(--bad)">−0</span> · 테스트 1 — fix: 심플봇 생성 실패 시 전역 Alert 과 토스트가 겹쳐 안내되던 문제 수정</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-23 22:07</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> fix-round: merged <a href="https://github.com/hkjang/aiportal-front/pull/29">PR #29</a>, <strong>release failed</strong><div class="meta">2파일 <span style="color:var(--good)">+290</span>/<span style="color:var(--bad)">−2</span> · 테스트 1 — fix: 심플봇 수정 팝업이 저장 실패에도 닫혀 수정 내용이 유실되던 문제 수정</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-23 21:26</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/28">PR #28</a>, <strong>release failed</strong><div class="meta">3파일 <span style="color:var(--good)">+214</span>/<span style="color:var(--bad)">−0</span> · 테스트 1 — fix: 심플봇 추천 프롬프트 생성 실패가 조용히 삼켜지던 문제 수정</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-23 20:39</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> fix-round: merged <a href="https://github.com/hkjang/aiportal-front/pull/27">PR #27</a>, <strong>release failed</strong><div class="meta">2파일 <span style="color:var(--good)">+191</span>/<span style="color:var(--bad)">−16</span> · 테스트 1 — fix: 앱 공유 등록이 공통 오류를 &#x27;권한 없음&#x27; 으로 잘못 안내하던 문제 수정</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-23 18:19</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> fix-round: merged <a href="https://github.com/hkjang/aiportal-front/pull/26">PR #26</a>, <strong>release failed</strong><div class="meta">2파일 <span style="color:var(--good)">+174</span>/<span style="color:var(--bad)">−3</span> · 테스트 1 — fix: 전역 Confirm 을 Escape 로 닫으면 취소 동작이 유실되던 문제 수정</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-23 16:49</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> fix-round: merged <a href="https://github.com/hkjang/aiportal-front/pull/25">PR #25</a>, <strong>release failed</strong><div class="meta">3파일 <span style="color:var(--good)">+145</span>/<span style="color:var(--bad)">−3</span> · 테스트 1 — fix: 전역 Alert 의 후속 동작이 Escape 로 닫으면 유실되던 문제 수정</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-23 14:46</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> fix-round: merged <a href="https://github.com/hkjang/aiportal-front/pull/24">PR #24</a>, <strong>release failed</strong><div class="meta">2파일 <span style="color:var(--good)">+181</span>/<span style="color:var(--bad)">−2</span> · 테스트 1 — fix: OCR 업로드 실패가 조용히 삼켜지던 문제 수정</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-23 12:20</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> fix-round: merged <a href="https://github.com/hkjang/aiportal-front/pull/23">PR #23</a>, <strong>release failed</strong><div class="meta">2파일 <span style="color:var(--good)">+158</span>/<span style="color:var(--bad)">−0</span> · 테스트 2 — test: 전역 로딩 스피너 계약과 법률 문서 링크 클릭 경로 고정</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-23 10:14</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> fix-round: merged <a href="https://github.com/hkjang/aiportal-front/pull/22">PR #22</a>, <strong>release failed</strong><div class="meta">8파일 <span style="color:var(--good)">+451</span>/<span style="color:var(--bad)">−3</span> · 테스트 2 — fix: 알림/확인 팝업 본문의 HTML 주입 차단 및 개행 렌더링 정정</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-22 23:27</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> fix-round: merged <a href="https://github.com/hkjang/aiportal-front/pull/21">PR #21</a>, <strong>release failed</strong><div class="meta">4파일 <span style="color:var(--good)">+21</span>/<span style="color:var(--bad)">−19</span> · <em>테스트 없음</em> — docs: 존재하지 않는 빌드·lint 명령과 끊긴 문서 링크 정정</div></td></tr><tr data-status="other"><td data-label="일시">2026-09-22 21:34</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> fix-round: review held, PR open <a href="https://github.com/hkjang/aiportal-front/pull/20">PR #20</a><div class="meta">2파일 <span style="color:var(--good)">+122</span>/<span style="color:var(--bad)">−2</span> · 테스트 1 — fix: 사이드메뉴 앱 목록 캐시가 배열이 아닐 때 목록 조회 실패 수정</div></td></tr><tr data-status="nochange"><td data-label="일시">2026-09-22 19:49</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-22 17:30</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-22 16:00</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-22 14:10</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-22 12:28</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-22 10:50</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-22 09:39</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-22 08:59</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-22 07:39</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-22 06:35</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-22 06:00</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-22 04:17</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-22 02:39</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-22 01:31</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-21 23:40</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-21 22:59</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-21 21:10</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-21 19:10</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-21 17:19</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-21 15:40</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-21 15:00</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-21 13:39</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-21 12:59</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-21 11:35</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-21 10:13</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-21 09:59</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-21 08:12</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-21 07:25</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-21 06:28</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-21 05:39</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-21 04:14</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-21 03:59</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-21 02:15</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-21 02:00</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="merged"><td data-label="일시">2026-09-21 00:26</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> fix-round: merged <a href="https://github.com/hkjang/aiportal-front/pull/19">PR #19</a>, <strong>release failed</strong><div class="meta">4파일 <span style="color:var(--good)">+98</span>/<span style="color:var(--bad)">−6</span> · <em>테스트 없음</em> — docs: 릴리즈 스킬 탐색 안내 및 버전 근거 정본화</div></td></tr><tr data-status="nochange"><td data-label="일시">2026-09-20 23:11</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> fix-round: no change</td></tr><tr data-status="merged"><td data-label="일시">2026-09-20 21:28</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/18">PR #18</a>, <strong>release failed</strong><div class="meta">2파일 <span style="color:var(--good)">+126</span>/<span style="color:var(--bad)">−10</span> · 테스트 1 — fix: 페이지 초기화 후 중복 목록 조회 제거</div></td></tr><tr data-status="other"><td data-label="일시">2026-09-20 09:40</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> review held, PR open <a href="https://github.com/hkjang/aiportal-front/pull/17">PR #17</a><div class="meta">4파일 <span style="color:var(--good)">+77</span>/<span style="color:var(--bad)">−11</span> · 테스트 1 — fix: serviceCode 비교를 대소문자 무시 공용 매처(isServiceCode)로 통일</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-19 04:41</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> fix-round: merged <a href="https://github.com/hkjang/aiportal-front/pull/16">PR #16</a>, release skipped<div class="meta">3파일 <span style="color:var(--good)">+79</span>/<span style="color:var(--bad)">−0</span> · 테스트 1 — fix: eventBus 가 쓰는 mitt 를 package.json 직접 의존성으로 선언</div></td></tr><tr data-status="other"><td data-label="일시">2026-09-18 16:23</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-failed" title="outcome=error">실행 오류</span> hold: budget</td></tr><tr data-status="merged"><td data-label="일시">2026-09-17 03:21</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/15">PR #15</a>, release skipped<div class="meta">3파일 <span style="color:var(--good)">+320</span>/<span style="color:var(--bad)">−23</span> · 테스트 1 — fix: 채팅 이동 데이터가 다른 세션 진입 시 이전 질문을 재전송하던 문제 및 레코드 정규화</div></td></tr><tr data-status="other"><td data-label="일시">2026-09-16 20:03</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-failed" title="outcome=error">실행 오류</span> hold: budget</td></tr><tr data-status="other"><td data-label="일시">2026-09-16 17:53</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-failed" title="outcome=error">실행 오류</span> hold: budget</td></tr><tr data-status="merged"><td data-label="일시">2026-09-10 19:15</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/14">PR #14</a>, release skipped<div class="meta">7파일 <span style="color:var(--good)">+340</span>/<span style="color:var(--bad)">−22</span> · 테스트 3 — fix: 열린 앱 저장 레코드 표기 불일치 및 임베딩 파일명 복원 공용화</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-10 11:23</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/13">PR #13</a>, release skipped<div class="meta">5파일 <span style="color:var(--good)">+83</span>/<span style="color:var(--bad)">−49</span> · 테스트 1 — fix: 대화저장 상세의 기본앱 비동기 처리 오류 및 봇 식별자 파싱 공용화</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-09 15:15</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/12">PR #12</a>, release skipped<div class="meta">3파일 <span style="color:var(--good)">+246</span>/<span style="color:var(--bad)">−31</span> · 테스트 1 — fix: 기본앱 조회 실패 처리 및 마운트 시 중복 API 호출 제거</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-09 07:00</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/11">PR #11</a>, release skipped<div class="meta">4파일 <span style="color:var(--good)">+296</span>/<span style="color:var(--bad)">−8</span> · 테스트 1 — fix: 공용 저장소 사용자 캐시 로그아웃 잔존 및 storage 예외 방어</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-09 00:11</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/10">PR #10</a>, release skipped<div class="meta">6파일 <span style="color:var(--good)">+485</span>/<span style="color:var(--bad)">−32</span> · 테스트 2 — fix: 인증/채팅세션 저장소 방어 및 탭 동기화 플래그 충돌 수정</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-08 21:07</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/9">PR #9</a> (approved), release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-08 21:06</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/8">PR #8</a> (approved), release skipped</td></tr><tr data-status="other"><td data-label="일시">2026-09-08 19:20</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> needs approval (risk=medium, files=5), PR open <a href="https://github.com/hkjang/aiportal-front/pull/9">PR #9</a><div class="meta">5파일 <span style="color:var(--good)">+408</span>/<span style="color:var(--bad)">−79</span> · 테스트 1 — fix: 노트북/업무도구 목록 캐시 사용자 간 잔존 및 조회 실패 수정</div></td></tr><tr data-status="other"><td data-label="일시">2026-09-08 16:11</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> needs approval (risk=medium, files=2), PR open <a href="https://github.com/hkjang/aiportal-front/pull/8">PR #8</a><div class="meta">2파일 <span style="color:var(--good)">+577</span>/<span style="color:var(--bad)">−52</span> · 테스트 1 — fix: OCR 사용 횟수 초기화 누락 및 상태 스토어 정합성 수정</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-07 10:39</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/6">PR #6</a> (approved), release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-07 10:37</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/5">PR #5</a> (approved), release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-07 08:59</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/7">PR #7</a>, release skipped<div class="meta">3파일 <span style="color:var(--good)">+330</span>/<span style="color:var(--bad)">−47</span> · 테스트 1 — fix: 로그아웃 후 사용자 정보 잔존 및 탭 간 동기화 누락 수정</div></td></tr><tr data-status="other"><td data-label="일시">2026-09-06 23:47</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-failed" title="outcome=verify-failed">검증 실패</span> verify failed: secrets in diff<div class="meta">2파일 <span style="color:var(--good)">+447</span>/<span style="color:var(--bad)">−60</span> · 테스트 1 — fix: 토큰 재발급 인터셉터 리프레시 락 누수 수정 및 단위 테스트 추가</div></td></tr><tr data-status="other"><td data-label="일시">2026-09-06 12:11</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> CI no-ci, PR open <a href="https://github.com/hkjang/aiportal-front/pull/6">PR #6</a><div class="meta">2파일 <span style="color:var(--good)">+370</span>/<span style="color:var(--bad)">−31</span> · 테스트 1 — fix: 첨부파일 용량 안내 문구 옵션 반영 및 업로드 오류 오판 수정 및 단위 테스트 추가</div></td></tr><tr data-status="other"><td data-label="일시">2026-09-06 02:03</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> CI no-ci, PR open <a href="https://github.com/hkjang/aiportal-front/pull/5">PR #5</a><div class="meta">2파일 <span style="color:var(--good)">+185</span>/<span style="color:var(--bad)">−8</span> · 테스트 1 — fix: 마크다운 렌더링 코드블록/표 래퍼 속성 유실 및 외부 링크 하드닝</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-05 00:31</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/4">PR #4</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-04 05:45</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/3">PR #3</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-03 22:04</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/2">PR #2</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-03 13:47</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/1">PR #1</a>, release skipped</td></tr></tbody></table></div>

## 비용·사용량

<div class="table-wrap"><table class="rt"><caption class="meta">최근 30세션</caption><thead><tr><th>시각</th><th class="primary">프로젝트</th><th>단계</th><th class="num">시간</th><th class="num">턴</th><th class="num">비용</th><th class="num">토큰 입력/출력</th><th>종료</th></tr></thead><tbody><tr data-status="other"><td data-label="시각">11:16</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">릴리즈</td><td data-label="시간" class="num">1분</td><td data-label="턴" class="num">13</td><td data-label="비용" class="num">$0.72</td><td data-label="토큰 입력/출력" class="num">437K / 7K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">11:14</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">비평</td><td data-label="시간" class="num">0분</td><td data-label="턴" class="num">1</td><td data-label="비용" class="num">$0.00</td><td data-label="토큰 입력/출력" class="num">436K / 2K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">11:13</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">수리</td><td data-label="시간" class="num">6분</td><td data-label="턴" class="num">47</td><td data-label="비용" class="num">$2.29</td><td data-label="토큰 입력/출력" class="num">2.1M / 24K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">11:07</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">비평</td><td data-label="시간" class="num">0분</td><td data-label="턴" class="num">1</td><td data-label="비용" class="num">$0.00</td><td data-label="토큰 입력/출력" class="num">339K / 2K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">11:06</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">6분</td><td data-label="턴" class="num">39</td><td data-label="비용" class="num">$2.92</td><td data-label="토큰 입력/출력" class="num">2.7M / 27K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">11:00</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">정찰</td><td data-label="시간" class="num">7분</td><td data-label="턴" class="num">27</td><td data-label="비용" class="num">$2.08</td><td data-label="토큰 입력/출력" class="num">1.2M / 26K</td><td data-label="종료">error_max_budget_usd</td></tr><tr data-status="other"><td data-label="시각">09:35</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">릴리즈</td><td data-label="시간" class="num">2분</td><td data-label="턴" class="num">14</td><td data-label="비용" class="num">$0.75</td><td data-label="토큰 입력/출력" class="num">410K / 7K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">09:33</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">비평</td><td data-label="시간" class="num">3분</td><td data-label="턴" class="num">21</td><td data-label="비용" class="num">$1.03</td><td data-label="토큰 입력/출력" class="num">728K / 9K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">09:30</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">6분</td><td data-label="턴" class="num">33</td><td data-label="비용" class="num">$2.33</td><td data-label="토큰 입력/출력" class="num">2.1M / 21K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">07:40</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">릴리즈</td><td data-label="시간" class="num">2분</td><td data-label="턴" class="num">13</td><td data-label="비용" class="num">$0.68</td><td data-label="토큰 입력/출력" class="num">384K / 7K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">07:37</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">비평</td><td data-label="시간" class="num">2분</td><td data-label="턴" class="num">22</td><td data-label="비용" class="num">$1.08</td><td data-label="토큰 입력/출력" class="num">751K / 9K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">07:35</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">6분</td><td data-label="턴" class="num">39</td><td data-label="비용" class="num">$2.84</td><td data-label="토큰 입력/출력" class="num">2.7M / 24K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">07:29</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">정찰</td><td data-label="시간" class="num">6분</td><td data-label="턴" class="num">23</td><td data-label="비용" class="num">$1.82</td><td data-label="토큰 입력/출력" class="num">1.1M / 23K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">06:48</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">릴리즈</td><td data-label="시간" class="num">2분</td><td data-label="턴" class="num">16</td><td data-label="비용" class="num">$0.74</td><td data-label="토큰 입력/출력" class="num">441K / 7K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">06:46</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">비평</td><td data-label="시간" class="num">4분</td><td data-label="턴" class="num">25</td><td data-label="비용" class="num">$1.25</td><td data-label="토큰 입력/출력" class="num">948K / 12K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">06:42</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">5분</td><td data-label="턴" class="num">29</td><td data-label="비용" class="num">$2.08</td><td data-label="토큰 입력/출력" class="num">1.8M / 18K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">06:38</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">정찰</td><td data-label="시간" class="num">5분</td><td data-label="턴" class="num">23</td><td data-label="비용" class="num">$1.68</td><td data-label="토큰 입력/출력" class="num">1.0M / 19K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">05:25</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">릴리즈</td><td data-label="시간" class="num">1분</td><td data-label="턴" class="num">13</td><td data-label="비용" class="num">$0.62</td><td data-label="토큰 입력/출력" class="num">339K / 5K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">05:23</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">비평</td><td data-label="시간" class="num">3분</td><td data-label="턴" class="num">25</td><td data-label="비용" class="num">$1.20</td><td data-label="토큰 입력/출력" class="num">872K / 11K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">05:20</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">8분</td><td data-label="턴" class="num">49</td><td data-label="비용" class="num">$3.30</td><td data-label="토큰 입력/출력" class="num">3.2M / 28K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">05:13</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">정찰</td><td data-label="시간" class="num">4분</td><td data-label="턴" class="num">22</td><td data-label="비용" class="num">$1.79</td><td data-label="토큰 입력/출력" class="num">1.1M / 21K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">05:00</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">릴리즈</td><td data-label="시간" class="num">1분</td><td data-label="턴" class="num">12</td><td data-label="비용" class="num">$0.73</td><td data-label="토큰 입력/출력" class="num">357K / 7K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">04:58</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">비평</td><td data-label="시간" class="num">3분</td><td data-label="턴" class="num">32</td><td data-label="비용" class="num">$1.50</td><td data-label="토큰 입력/출력" class="num">1.3M / 13K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">04:55</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">8분</td><td data-label="턴" class="num">49</td><td data-label="비용" class="num">$3.55</td><td data-label="토큰 입력/출력" class="num">3.8M / 28K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">04:48</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">정찰</td><td data-label="시간" class="num">4분</td><td data-label="턴" class="num">24</td><td data-label="비용" class="num">$1.56</td><td data-label="토큰 입력/출력" class="num">945K / 16K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">03:21</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">릴리즈</td><td data-label="시간" class="num">2분</td><td data-label="턴" class="num">15</td><td data-label="비용" class="num">$0.77</td><td data-label="토큰 입력/출력" class="num">397K / 8K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">03:19</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">비평</td><td data-label="시간" class="num">3분</td><td data-label="턴" class="num">21</td><td data-label="비용" class="num">$1.00</td><td data-label="토큰 입력/출력" class="num">727K / 9K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">03:16</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">7분</td><td data-label="턴" class="num">51</td><td data-label="비용" class="num">$3.65</td><td data-label="토큰 입력/출력" class="num">3.6M / 33K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">03:09</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">정찰</td><td data-label="시간" class="num">5분</td><td data-label="턴" class="num">30</td><td data-label="비용" class="num">$2.00</td><td data-label="토큰 입력/출력" class="num">1.6M / 18K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">01:47</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="단계">릴리즈</td><td data-label="시간" class="num">1분</td><td data-label="턴" class="num">12</td><td data-label="비용" class="num">$0.59</td><td data-label="토큰 입력/출력" class="num">291K / 5K</td><td data-label="종료">success</td></tr></tbody></table></div>

## 아이디어 백로그 — 대기 12 / 전체 13

<div class="table-wrap"><table class="rt" data-filter="1"><caption class="meta">에이전트가 회차마다 재평가한다. 가치 높고 위험 낮은 대기 항목이 다음 회차 후보다.</caption><thead><tr><th class="primary">아이디어</th><th>가치/위험/크기</th><th>상태</th><th>메모</th><th>갱신</th></tr></thead><tbody><tr data-status="nochange"><td data-label="아이디어" class="primary">릴리즈 절차 교착 자체를 사람에게 에스컬레이션</td><td data-label="가치/위험/크기">5/1/S</td><td data-label="상태">대기</td><td data-label="메모">26회 연속 동일 교착. 사람 입력 4가지(①첫 릴리즈 버전 0.0.1/0.1.0/1.0.0 ②태그 형식·주석 태그 여부 ③릴리즈 노트 위치·양식·언어 ④GitHub Release 사용 여부)만 정해지면 이후는 전례로 자동 진행 가능. GitLab CI 가 main 머지 시점에 이미 배포하므로 버전 릴리즈 없이도 서비스는 나간다 — 버전 체계 도입 여부 자체가 제품 판단이다.</td><td data-label="갱신">2026-09-24</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">[수정 과제] 릴리즈 버전 결정 입력 복구 — 승인 근거 확보 전까지 pending 유지</td><td data-label="가치/위험/크기">5/2/S</td><td data-label="상태">대기</td><td data-label="메모">26회째 진입 조건 미충족. 이번 회차에도 무변경(정찰 지시대로 재조사도 하지 않았다). 고칠 워크플로 파일이 저장소에 없고(.github 부재, .gitlab-ci.yml 은 CI_COMMIT_TAG 참조 0건인 브랜치 배포 전용) 실패 지점이 워크트리 밖(외부 aidev bin/run.sh → release.json → bin/gate.py). package.json 에 version 필드가 있어 skipped 조건도 미충족이며 version 필드 삭제는 게이트 완화라 금지.</td><td data-label="갱신">2026-09-24</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">globalLoading 참조 카운트 도입 (병렬 요청 중 스피너 조기 종료)</td><td data-label="가치/위험/크기">4/4/M</td><td data-label="상태">대기</td><td data-label="메모">loading.vue 의 자동 해제가 카운터를 되돌리지 않고 타이머 재시작 계약도 미확정이라 지금 넣으면 스피너가 영구히 안 꺼지는 회귀. 위 타이머 항목이 선행.</td><td data-label="갱신">2026-09-24</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">심플봇 두 팝업의 tryUpdateAppIcon / tryUpdateLog 가 예외를 catch(e){} 로 완전히 삼킴</td><td data-label="가치/위험/크기">3/2/S</td><td data-label="상태">대기</td><td data-label="메모">PopSimpleBot.vue:542,563 / PopSimpleBotUpdate.vue:649. &#x27;아이콘 실패가 앱 저장 전체의 실패인가&#x27; 라는 기대 계약이 미확인이고 simpleBotUpdateClose.spec.js 가 현 동작(&#x27;아이콘 실패는 성공 판정을 바꾸지 않는다&#x27;)을 고정하고 있어, 고치려면 계약을 &#x27;앱 저장은 성공, 아이콘 실패만 별도 토스트&#x27; 로 두고 기존 케이스부터 갱신해야 한다.</td><td data-label="갱신">2026-09-24</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">loading.vue 의 자동 해제 타이머가 이미 켜진 상태의 추가 startLoading() 을 반영하지 않음</td><td data-label="가치/위험/크기">3/3/M</td><td data-label="상태">대기</td><td data-label="메모">watch(() =&gt; props.modelValue, startTimer) 가 false→true 전이에서만 재시작한다. 기대 계약 확정이 선행이며 globalLoading 참조 카운트의 선행 조건. 이번 회차 미열람(이전 기록 기준 보존).</td><td data-label="갱신">2026-09-24</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">Alert/Confirm 이 showHeader 를 넘기지 않아 title 이 렌더링되지 않음</td><td data-label="가치/위험/크기">3/3/S</td><td data-label="상태">대기</td><td data-label="메모">Base.vue:50 showHeader 기본 false. 보이게 하면 title 자리에 Error 객체를 넘기는 호출부 3곳(SupportOcrDetail.vue:88 / SupportOcrDetail2.vue:90 / SupportSttDetail.vue:81)이 객체를 화면에 찍는다 — 기대 계약 확정이 선행.</td><td data-label="갱신">2026-09-24</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">openAlert 의 title 자리에 Error 객체를 넘기는 호출부 3곳 정리</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">showHeader 기본값이 false 라 현재 관측 가능한 동작 변화가 없다 — showHeader 계약이 확정될 때 함께 처리.</td><td data-label="갱신">2026-09-24</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">MainLayout.vue:107 의 @click=&quot;onWidgetSave&quot; 가 미정의 핸들러</td><td data-label="가치/위험/크기">2/2/S</td><td data-label="상태">대기</td><td data-label="메모">신규(이번 회차에서 분리). grep 상 onWidgetSave 는 어디에도 선언돼 있지 않아 PopWidgetSetting 의 emit(&#x27;click&#x27;) 이 아무 데도 닿지 않는다. 이번 수정으로 emit(&#x27;click&#x27;) 은 저장 성공 여부와 무관하게 먼저 나가는 형태가 그대로 남았다 — 핸들러를 정의할지 emit(&#x27;click&#x27;) 자체를 없앨지는 기대 계약 결정이 필요하고, 지금은 양쪽 다 관측 가능한 동작 변화가 0 이라 단독으로는 무효 변경이다.</td><td data-label="갱신">2026-09-24</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">isSuccess 검사가 없는 잔여 1곳 — HeaderAlarm.getList</td><td data-label="가치/위험/크기">2/2/M</td><td data-label="상태">대기</td><td data-label="메모">PopWidgetSetting 을 이번에 처리해 남은 것은 HeaderAlarm.getList(:37-67) 한 곳이다. &#x27;조용한 빈 알림 목록이 의도인가&#x27; 라는 기대 계약이 미확인이고, 알림 뱃지는 주기적으로 갱신되는 보조 UI 라 실패마다 토스트를 띄우면 오히려 시끄러울 수 있다 — 계약 확정이 선행.</td><td data-label="갱신">2026-09-24</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">onToggleFav 3곳의 bare catch 가 즐겨찾기 실패를 무음 처리</td><td data-label="가치/위험/크기">2/2/S</td><td data-label="상태">대기</td><td data-label="메모">FavoriteList:158 / SupportList:446 / MyAgentList:482 모두 catch (e) {}. FavoriteList 쪽은 showMore:false 라 더보기 버튼이 렌더되지 않아 도달 불가이고, 나머지 2곳도 별 아이콘이 원래 상태로 남아 &#x27;실패&#x27; 로 읽히므로 개선폭이 작다.</td><td data-label="갱신">2026-09-24</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">RequestDetail/RequestList 가 inf.library.* 를 호출한다 — 요청 게시판 전용 엔드포인트 부재</td><td data-label="가치/위험/크기">2/4/S</td><td data-label="상태">대기</td><td data-label="메모">조사 항목. interface.js 에 board(:93)/library(:106) 두 도메인만 있고 request 도메인 자체가 없다. &#x27;요청 게시판은 library 를 공유한다&#x27; 가 설계일 가능성이 높다. 백엔드가 별개 저장소라 이 저장소만으로 확정 불가 — 사람/백엔드 확인 없이 바꾸면 화면이 통째로 죽는다. 기존 스펙도 /library? 가 나가는 현 동작을 고정했다.</td><td data-label="갱신">2026-09-24</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">아무것도 연결하지 않는 v-model=&quot;searchKey&quot; 잔재 4곳 정리</td><td data-label="가치/위험/크기">1/1/S</td><td data-label="상태">대기</td><td data-label="메모">SearchBox 에 modelValue prop 이 없으므로 NoticeList:114 / LibraryList:112 / ShareList:93 / PopGlobalSearch:110 의 v-model 은 아무 배선도 만들지 않는다. 관측 가능한 동작 변화가 0 이라 단독으로는 운영자 지시(무효 변경 금지)에 걸린다. SearchBox 에 modelValue 계약을 실제로 넣는 결정이 날 때 함께 처리할 항목.</td><td data-label="갱신">2026-09-24</td></tr><tr data-status="released"><td data-label="아이디어" class="primary">위젯 설정 팝업(PopWidgetSetting)의 저장 실패가 성공처럼 닫히고 옛 값을 다시 그림</td><td data-label="가치/위험/크기">4/2/M</td><td data-label="상태">완료</td><td data-label="메모">saveWidgetSetting 에 isSuccess 검사가 없어 업무 실패에도 bus.emit(WIDGET_CHANGED) 가 나가고 catch (e) {} 가 예외를 완전히 삼켰으며, onClick 이 await 없이 close() 를 먼저 불러 저장 결과와 무관하게 닫혔다. 저장(팝업) 표준형(PopSimpleBotUpdate.onCreate:663-681)대로 실패는 토스트 + 닫지 않음, 성공 경로에서만 emit + close 로 맞췄다. 신규 스펙 6건(Red 5건) + 변이 7건. npm test 38파일 579테스트, build:dev 통과.</td><td data-label="갱신">2026-09-24</td></tr></tbody></table></div>

## 교훈 (깨졌던 변경)

- 2026-09-08 **demoted** — 자율화 단계 release → low-risk: 회귀(reverted) 2026-09-08T00:00:35+09:00

## 원장 (에이전트가 남긴 기록)

## 2026-09-03
- 선택: 공통 유틸 Vitest 단위 테스트 도입 및 파일 확장자 판별 오류 수정 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 자동화 테스트가 전무했던 저장소에 Vitest + jsdom 환경(`vitest.config.js`, `npm test` / `npm run test:watch`)을 추가하고 `common`, `page`, `chatFile`, `favToggle` 유틸에 대한 단위 테스트 61건을 작성했다. 테스트 작성 중 발견한 실제 버그 2건(`isPreviewablXlsx` 가 `['xlsx','xlsx']` 로 중복 비교해 `.xls` 를 인식하지 못하던 문제, 파일 판별 함수들이 `file` 이 null 일 때 `file.file_name` 접근으로 예외를 던지던 문제)을 고치고 회귀 테스트로 고정했다. 검증은 `npm test`(61건 통과)와 `npm run build:dev`(빌드 성공)로 수행했다. README 와 `docs/09-테스트가이드-총론.md` 에 실행 방법을 문서화했다.
- 보류 아이디어:
  - markdown.js 의 DOMPurify 설정/onclick 파싱 경로 XSS 하드닝 검토 (가치 4 / 위험 3 / M)
  - ESLint + Prettier 도입 (탭·스페이스 혼재로 초기 diff 노이즈가 커서 보류) (가치 3 / 위험 2 / M)
  - `parseUtcToKstDate` 가 로컬 타임존이 KST 일 때 +9h 를 이중 적용하는 문제 정리 (운영 표시값 영향 커서 보류) (가치 3 / 위험 4 / S)
  - `useFileAccept` 의 mode 값 불일치(`modeKey` 는 'ocr', 제외 확장자 분기는 'OCR') 수정 (가치 3 / 위험 3 / S)
  - `docs/07-개선사항-권장사항.md` 의 스토어 패턴 통일(useServiceStore → Pinia defineStore) (가치 3 / 위험 4 / L)

## 2026-09-03
- 선택: 파일 업로드 accept/용량 정책 오류 수정 및 단위 테스트 추가 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `useFileAccept` 에서 실사용 버그 4건을 고쳤다 — (1) `SupportOcr.vue` 가 넘기는 `"OCR"` 이 소문자 키 맵과 매칭되지 않아 `ocrFileLimit` 정책이 적용되지 않고 항상 기본 50MB 였던 문제, (2) 괄호 표기 확장자(`"엑셀(xlsx,xls)"`)를 파싱할 때 `"." + [".xlsx"].join()` 으로 점을 중복해 `..xlsx` 같은 잘못된 accept 토큰을 만들던 문제, (3) OCR 제외 확장자 필터가 괄호 표기 항목과 fallback 목록에는 적용되지 않던 문제, (4) 세 개 화면(`SupportOcr`, `PopSimpleBot`, `PopSimpleBotUpdate`)의 용량 초과 알림이 바이트 값을 "52428800MB" 처럼 MB 로 표기하던 문제. accept/용량 계산을 순수 함수(`parseExtensionEntry`, `buildAcceptString`, `buildFallbackAccept`, `resolveMaxSizeMb`)로 분리하고 회귀 테스트 18건을 추가했다. 검증은 `npm test`(총 79건 통과)와 `npm run build:dev`(빌드 성공)로 수행했다.
- 보류 아이디어:
  - markdown.js 의 DOMPurify 설정/onclick 파싱 경로 XSS 하드닝 검토 (가치 4 / 위험 3 / M)
  - ESLint + Prettier 도입 (탭·스페이스 혼재로 초기 diff 노이즈가 커서 보류) (가치 3 / 위험 2 / M)
  - `parseUtcToKstDate` 가 로컬 타임존이 KST 일 때 +9h 를 이중 적용하는 문제 정리 (운영 표시값 영향 커서 보류) (가치 3 / 위험 4 / S)
  - `buttonSettingPolicy.buildSettingActions` 모드별 액션 구성 단위 테스트 보강 (가치 2 / 위험 1 / S)
  - 빌드 산출물 단일 청크 6.2MB 문제(manualChunks 코드 스플리팅) 개선 (가치 3 / 위험 3 / M)

## 2026-09-04
- 선택: 앱 정보 조회(getAppInfo) 캐시/재조회 경로 오류 수정 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `src/utils/appList.js` 의 `getAppInfo` 에서 실사용 버그 3건을 고쳤다 — (1) 사이드메뉴 캐시 미존재 시 기본값이 `{}` 라 `appList.length === 0` 이 항상 false 가 되어 API 재조회를 건너뛰고 `{}.find` 에서 TypeError 가 나 `{}` 를 반환하던 문제(딥링크로 /chatmain?appId= 진입 시 앱 정보 유실), (2) `serviceMenu === 'all'` 재조회가 `serviceCode === 'all'` 로 필터링돼 항상 빈 목록을 반환, 사이드메뉴 상위 30개에 없는 앱은 절대 찾지 못하던 문제, (3) 재조회로 찾은 앱은 `writeOpenApp`/simple 모드 후처리를 건너뛰던 경로 불일치. 순수 함수(`toAppArray`, `pickServiceRows`, `flattenAppList`, `findAppInfo`)로 분리하고 `findAppInfo` 의 app_id 문자열 비교·빈 키·비배열 입력 방어를 추가했다. 검증은 `npm test`(총 100건 통과, 신규 21건)와 `npm run build:dev`(빌드 성공)로 수행했다.
- 보류 아이디어:
  - markdown.js 의 DOMPurify 설정/onclick 파싱 경로 XSS 하드닝 검토 (가치 4 / 위험 3 / M)
  - `useFileAttach` 의 알림 문구가 옵션(maxFileSize/maxTotalSize)을 무시하고 20MB/100MB 로 하드코딩된 문제 수정 (가치 3 / 위험 1 / S)
  - `useAppList.getFormattedAppList` 의 `String(app?.knowledge_info?.status) ?? ''` 가 문자열 "undefined" 를 만드는 문제 정리 (Sidemenu.vue 에도 동일 코드 중복) (가치 2 / 위험 1 / S)
  - `mitt` 이 `src/utils/eventBus.js` 에서 직접 import 되는데 package.json 직접 의존성에 없어 전이 의존성에 기대고 있는 문제 (가치 3 / 위험 1 / S)
  - 빌드 산출물 단일 청크 6.2MB 문제(manualChunks 코드 스플리팅) 개선 (가치 3 / 위험 3 / M)

## 2026-09-05
- 선택: 앱 이동(updateLogAndGo) 쿼리 유실·외부 링크 취약점 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 모든 앱 카드 클릭이 거치는 공통 네비게이션 경로 `src/utils/appUpdateLog.js` 에서 5건을 고쳤다 — (1) `buildTo` 가 `to` 를 문자열로 받으면 `extraQuery` 를 그대로 버려 img 모드 앱의 `?mode=img` 가 유실되던 문제(문자열도 `{path, query, hash}` 로 병합하도록 변경), (2) 같은 이유로 `nextTo.path` 가 undefined 라 `/chatmain` 판별에 실패해 `writeOpenApp` 이 호출되지 않고 챗 화면이 앱 컨텍스트를 잃던 문제(`toRoutePath` 도입), (3) `mode === 'link'` 의 `window.open(urlLink, '_blank')` 이 `noopener` 없이 열려 reverse tabnabbing 이 가능하고 `javascript:`/`data:` url_link 가 그대로 실행되던 문제(`isSafeExternalLink`/`openExternalLink` 로 http·https·mailto 만 허용, 차단 시 일반 이동으로 폴백), (4) iframe 모드도 동일하게 검증해 위험한 값은 빈 문자열로 넘기도록(ChatFrame 은 빈 값이면 "불러올 메신저가 없습니다" 표시) 처리, (5) `url_link` 가 문자열이 아닐 때 `.replace` 로 예외가 나던 문제. 순수 헬퍼(`toRoutePath`, `buildTo`, `resolveUserIdFromEmail`, `applyUserIdTemplate`, `isSafeExternalLink`, `openExternalLink`)로 분리하고 회귀 테스트 30건을 추가했다. 검증은 `npm test`(총 130건 통과, 신규 30건)와 `npm run build:dev`(빌드 성공)로 수행했다.
- 보류 아이디어:
  - markdown.js 의 DOMPurify 설정/onclick 파싱 경로 XSS 하드닝 검토 (transformAppAnchors 가 `&lt;` 를 실태그로 되돌림) (가치 4 / 위험 3 / M)
  - `useFileAttach` 알림 문구 하드코딩(20MB/100MB) 및 `msg.includes('10')` 로 서버 오류를 용량초과로 오판하는 문제 수정 (가치 3 / 위험 1 / S)
  - `useTableUtils.sortData` 가 모든 값을 `getNum` 으로 숫자 변환해 텍스트·날짜 컬럼 정렬이 동작하지 않는 문제 (현재 .vue 에서 미사용) (가치 2 / 위험 2 / S)
  - `mapAppToCard(item, idx, 'prompt')` 결과에 `to` 가 없어 GlobalSearch 프롬프트 카드 클릭이 무반응인 문제 (의도 확인 필요) (가치 3 / 위험 3 / S)
  - 빌드 산출물 단일 청크 6.2MB 문제(manualChunks 코드 스플리팅) 개선 (가치 3 / 위험 3 / M)
## 2026-09-06
- 선택: 마크다운 렌더링(parseMarkdown) 코드블록/표 래퍼 속성 유실 및 외부 링크 하드닝 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 챗 본문 렌더링 공통 경로 `src/utils/markdown.js` 에서 4건을 고쳤다 — (1) `transformAppAnchors` 의 code 태그 보호가 여는 태그를 버리고 `<code>${content}</code>` 로 복원해 `class="language-xx hljs"` 가 사라지고 highlight.js 테마(github-dark)의 `.hljs` 배경/기본색이 적용되지 않던 문제(매칭 전체를 보관·복원하도록 변경), (2) 불필요한 div 제거 로직이 `renderer.table` 이 만드는 `.table-wrap` 래퍼까지 걷어내 컬럼이 많은 표에서 가로 스크롤(`overflow-x:auto`)이 동작하지 않던 문제(table-wrap 래퍼만 마커로 보호 후 복원), (3) `target` 이 지정된 a 태그에 `rel` 이 없어 reverse tabnabbing 이 가능하던 문제(DOMPurify `afterSanitizeAttributes` 훅으로 `rel="noopener noreferrer"` 강제), (4) 본문에 `MARK_PLACEHOLDER_n__`/`DATE_PLACEHOLDER_n__` 문자열이 우연히 포함되면 복원 단계에서 undefined 구조분해로 TypeError 가 나거나 "undefined" 가 출력되던 문제. 검증은 `npm test`(총 155건 통과, 신규 25건 — 수정 전 코드에서 6건 실패함을 확인)와 `npm run build:dev`(빌드 성공)로 수행했다.
- 보류 아이디어: `useFileAttach` 알림 문구 하드코딩(20MB/100MB) 및 `msg.includes('10')` 로 서버 오류를 용량초과로 오판하는 문제 수정 (가치 3 / 위험 1 / S) · `parseUtcToKstDate` 가 로컬 타임존이 KST 일 때 +9h 를 이중 적용하는 문제 정리 (가치 3 / 위험 4 / S) · `mapAppToCard(item, idx, 'prompt')` 결과에 `to` 가 없어 GlobalSearch 프롬프트 카드 클릭이 무반응인 문제 (가치 3 / 위험 3 / S) · `mitt` 이 `eventBus.js` 에서 import 되는데 package.json 직접 의존성에 없어 전이 의존성에 기대는 문제 (가치 3 / 위험 1 / S) · 빌드 산출물 단일 청크 6.2MB 문제(manualChunks 코드 스플리팅) 개선 (가치 3 / 위험 3 / M)

## 2026-09-06
- 선택: useFileAttach 용량 안내 문구 옵션 반영 및 업로드 오류 오판 수정 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `src/utils/useFileAttach.js` 에서 6건을 고쳤다 — (1) 개당/전체 용량 초과 안내가 `maxFileSize`/`maxTotalSize` 옵션을 무시하고 "20MB"/"100MB" 로 하드코딩돼 있어 다른 정책을 주면 잘못된 숫자를 안내하던 문제(`formatSizeLimit` 로 옵션 값 기반 생성), (2) `msg.includes('10') || msg.includes('100')` 때문에 "10초 후 다시 시도해 주세요" 같은 무관한 서버 오류까지 용량 초과로 오판하던 문제(HTTP 413 + 용량 키워드 판별 `isSizeLimitError` 로 교체), (3) `err.response.data.message` 가 문자열이 아닐 때 `.includes` 에서 예외가 나던 문제, (4) `initialList` 를 그대로 ref 에 담아 `removeFile` 시 호출부 배열까지 변경되던 문제, (5) `removeFile` 의 인덱스 범위 미검증(음수 인덱스로 끝 항목이 삭제됨), (6) `.env` 같은 dotfile 을 확장자 "env" 로 잘못 판정하던 문제. 순수 헬퍼(`formatBytes`, `formatSizeLimit`, `getExt`, `isSizeLimitError`, `resolveErrorMessage`, `normalizeServerFiles`)로 분리하고 회귀 테스트 26건을 추가했다. 검증은 `npm test`(총 156건 통과, 신규 26건)와 `npm run build:dev`(빌드 성공)로 수행했다.
- 보류 아이디어: `parseUtcToKstDate` 가 로컬 타임존이 KST 일 때 +9h 를 이중 적용하는 문제 정리 (가치 3 / 위험 4 / S) · `mitt` 이 `eventBus.js` 에서 import 되는데 package.json 직접 의존성에 없어 전이 의존성에 기대는 문제 (가치 3 / 위험 1 / S) · `mapAppToCard(item, idx, 'prompt')` 결과에 `to` 가 없어 GlobalSearch 프롬프트 카드 클릭이 무반응인 문제 (가치 3 / 위험 3 / S) · `useDateRangeFilter` 가 endDate < startDate 만 보정하고 startDate > endDate 는 보정하지 않는 문제 + 미검증 컴포저블 테스트 공백 (가치 3 / 위험 2 / S) · 빌드 산출물 단일 청크 6.2MB 문제(manualChunks 코드 스플리팅) 개선 (가치 3 / 위험 3 / M)

## 2026-09-06
- 선택: 토큰 재발급 인터셉터 리프레시 락 누수 수정 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 모든 API 요청이 거치는 `src/api/common/interceptors.js` 에서 6건을 고쳤다 — (1) 리프레시 성공 경로가 `isRefreshing = false` 를 실행하지 않고 곧바로 `return instance(originalRequest)` 해서, 최초 토큰 재발급 이후 발생하는 모든 401 요청이 `if (isRefreshing)` 대기열에 들어간 뒤 아무도 깨워주지 않아 영원히 settle 되지 않던 문제(응답 200+`error.code===401` 경로와 응답 400+`code 401/403` 경로 양쪽 모두, 화면은 로딩 스피너에 멈춤), (2) `retryAllSubscribers` 가 `newAccessToken` 을 즉시 null 로 지워 "다른 API 에서 이미 리프레시됨" 재시도 최적화가 사실상 죽어 있던 문제, (3) 대기열 순회 중 재진입으로 구독이 유실될 수 있던 문제(먼저 비우고 순회), (4) `errCode === 401` 엄격 비교라 서버가 문자열 `"401"` 을 주면 재발급을 건너뛰던 문제(같은 파일 400 분기는 `== '401'` 느슨 비교라 내부 불일치 — `isUnauthorizedCode` 로 통일), (5) `error.config` 가 없는 에러에서 `error.config.url.includes('/app/simple')` 가 TypeError 를 던지던 BZ01 처리 경로, (6) `readAuth()` 결과와 `instance.defaults.headers.common` 미존재 시 예외 방어. 리프레시 종료를 `finishRefresh(ok)` 로 일원화하고 `isUnauthorizedCode`/`extractTokenDataFromResponse` 를 export 해 테스트 23건을 추가했다(수정 전 코드에서 6건 실패함을 확인). 검증은 `npm test`(총 153건 통과)와 `npm run build:dev`(빌드 성공)로 수행했다.
- 보류 아이디어: `mitt` 이 `eventBus.js` 에서 import 되는데 package.json 직접 의존성에 없어 전이 의존성에 기대는 문제 (가치 3 / 위험 1 / S) · `buildSettingActions` 의 support 모드에서 `isSimpleBotBool` 이 undefined 면 ButtonSetting 이 `show: undefined` 를 노출로 처리해 편집/워크플로우 액션이 동시에 뜨는 문제 (가치 3 / 위험 1 / S) · `globalLoading` 이 참조 카운트 없이 boolean 이라 병렬 요청 중 하나만 끝나도 스피너가 사라지는 문제 (가치 3 / 위험 3 / S) · `parseUtcToKstDate` 가 로컬 타임존이 KST 일 때 +9h 를 이중 적용하는 문제 정리 (가치 3 / 위험 4 / S) · 빌드 산출물 단일 청크 6.2MB 문제(manualChunks 코드 스플리팅) 개선 (가치 3 / 위험 3 / M)

## 2026-09-07
- 선택: 로그아웃 후 사용자 정보 잔존 및 탭 간 동기화 누락 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `src/storage/userStorage.js` 에서 6건을 고쳤다 — (1) `readUser()` 가 `userRef.value` 를 반환하는데 `clearUser()` 는 저장소만 비우고 `userRef` 를 그대로 둬서 로그아웃(HeaderSetting/`redirectToLogin`/인터셉터 401) 이후에도 이전 사용자의 이름·이메일·`is_admin` 이 계속 반환되던 문제, (2) 파일 상단 인라인 `if (!window._userStorageSyncInitialized)` 블록이 플래그를 먼저 세워서 아래 `setupUserStorageSync()` 가 항상 조기 반환 → 다른 탭 변경 시 현재 탭의 sessionStorage 가 전혀 갱신되지 않던 문제(다른 탭에서 로그아웃해도 이 탭을 새로고침하면 `readFromStorage` 가 sessionStorage 를 우선 읽어 로그아웃된 사용자가 되살아남), 리스너를 하나로 통합, (3) storage 이벤트의 `JSON.parse(e.newValue)` 미보호로 깨진 값에서 리스너가 예외를 던지던 문제, (4) 사용자 변경 통지가 `writeUser` 에서만 발생해 `clearUser`·다른 탭 변경이 구독자(Sidemenu)에게 전달되지 않던 문제와 구독자 하나의 예외가 나머지 통지를 끊던 문제, (5) `writeUser` 저장 형태를 순수 함수 `normalizeUserRecord` 로 분리, (6) `Sidemenu.vue` 가 `addUserChangeListener` 의 해제 함수를 버려 마운트마다 리스너가 누적되던 누수(+ 로그아웃 통지에 반응해 앱 목록을 재조회하지 않도록 `readUser()` 가드). 검증은 `npm test`(총 151건 통과, 신규 21건 — 수정 전 코드에서 11건 실패함을 확인)와 `npm run build:dev`(빌드 성공)로 수행했다.
- 보류 아이디어: `mitt` 이 `eventBus.js` 에서 import 되는데 package.json 직접 의존성에 없어 전이 의존성에 기대는 문제 (가치 3 / 위험 1 / S) · `globalLoading` 이 참조 카운트 없이 boolean 이라 병렬 요청 중 하나만 끝나도 스피너가 사라지는 문제 (가치 3 / 위험 3 / S) · `authStorage.readAuth` 가 localStorage 폴백 시 sessionStorage 를 복구하지 않고 `clearAuth` 도 통지가 없어 userStorage 와 동작이 어긋나는 문제 (가치 3 / 위험 2 / S) · `parseUtcToKstDate` 가 로컬 타임존이 KST 일 때 +9h 를 이중 적용하는 문제 정리 (가치 3 / 위험 4 / S) · 빌드 산출물 단일 청크 6.2MB 문제(manualChunks 코드 스플리팅) 개선 (가치 3 / 위험 3 / M)

## 2026-09-08
- 선택: OCR 사용 횟수 초기화 누락 및 상태 스토어 정합성 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 미테스트 상태였던 `src/storage/ocrStatusCheckStore.js` 에서 8건을 고쳤다 — (1) `checkOcrStatus` 가 `if (todayOcrList.length)` 가드 때문에 오늘 목록이 비면 `useCount` 갱신을 건너뛰어 어제 저장된 사용 횟수가 localStorage 에 그대로 남고 다음 날 첫 변환이 "동시 OCR 변환 사용 횟수를 초과했습니다" 로 막히던 문제(항상 재계산 + 저장 값에 `savedDate` 를 남겨 날짜가 바뀌면 로드 시점에 0 으로 복구), (2) `loadStatus()` 가 스토어 객체를 통째로 교체하면서 `completedItems` 키를 빠뜨려 Header/Home 마운트 때마다 `undefined` 가 되고 SupportOcrList·SupportOcrListCurrent 의 watch 대상이 사라지던 문제, (3) `getMaxUseOcr` 가 `res?.data?.body ?? 5` 로 비숫자 응답(객체 등)을 그대로 반환해 `useCount < NaN` 이 항상 false 가 되어 OCR 이 영구히 막히던 문제, (4) `canUseOcr` 가 로그인 정보 확인 전에 최대 횟수 API 를 호출하던 문제, (5) `startPolling(0)` 이 간격 0 의 setInterval 을 만들어 사실상 무한 루프가 되던 문제(최소 1초 하한 — 옛 코드로 테스트를 돌리면 실제로 힙 OOM 발생), (6) `calculateUseCount` 가 `group_id` 없는 항목들을 한 그룹으로 묶어 여러 요청을 1회로 세던 문제, (7) `checkOcrStatus` 가 API 실패 시 문서와 달리 `undefined` 를 반환하던 문제, (8) `todayOcrList`/저장 JSON 이 배열·객체가 아닐 때의 방어. 순수 헬퍼(`todayKey`, `toCount`, `normalizeStoredStatus`, `calculateUseCount`, `extractCompletedItems`)로 분리하고 테스트 43건을 추가했다(수정 전 코드에서 12건 실패 + 폴링 테스트는 OOM 으로 크래시함을 확인). 검증은 `npm test`(총 245건 통과)와 `npm run build:dev`(빌드 성공)로 수행했다.
- 보류 아이디어: `mitt` 이 `eventBus.js` 에서 import 되는데 package.json 직접 의존성에 없어 전이 의존성에 기대는 문제 (가치 3 / 위험 1 / S) · `authStorage.readAuth` 가 localStorage 폴백 시 sessionStorage 를 복구하지 않고 `clearAuth` 도 통지가 없어 userStorage 와 동작이 어긋나는 문제 (가치 3 / 위험 2 / S) · Header.vue 가 OCR 사용 여부와 무관하게 마운트 즉시 3초 폴링을 시작해 statusOcr API 를 계속 호출하는 문제 (가치 3 / 위험 3 / M) · `globalLoading` 이 참조 카운트 없이 boolean 이라 병렬 요청 중 하나만 끝나도 스피너가 사라지는 문제 (가치 3 / 위험 3 / S) · 빌드 산출물 단일 청크 6.2MB 문제(manualChunks 코드 스플리팅) 개선 (가치 3 / 위험 3 / M)

## 2026-09-08
- 선택: 노트북/업무도구 목록 캐시 사용자 간 잔존 및 조회 실패 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 미테스트 상태였던 `src/storage/myNoteBookStorage.js` 에서 7건을 고쳤다 — (1) `getMyNoteBookList` 이 `serviceCode == "myNoteBook"` 정확 일치로 찾는데 서버/다른 화면(MyAgentList.vue:307)은 `MyNotebook` 표기를 쓰고 있어 API 폴백 경로가 항상 빈 목록을 반환하고 매번 재조회하던 문제(대소문자 무시 비교로 통일), (2) `find` 로 첫 행만 써서 같은 serviceCode 가 여러 행으로 오면 나머지 appList 를 버리던 문제(flatMap 으로 합침), (3) `clearMyList` 가 export 만 되고 호출부가 전혀 없어 로그아웃(HeaderSetting.logout / common.js redirectToLogin) 후에도 이전 사용자의 노트북·업무도구 목록이 localStorage 에 남아 다음 사용자의 `checkAppExists`(Chat/Main.vue:508, Chat/Index.vue:710·1688)가 남의 앱에 true 를 반환하고 SupportImgList 가 남의 app_id 로 이동하던 문제(`clearMyNoteBookCache()` 추가 후 두 로그아웃 경로에 연결), (4) `readMyList`/`readWorkList` 의 `JSON.parse` 미보호로 캐시가 손상되면 `checkAppExists`/`modeCheckCount` 가 예외를 던져 챗 진입 흐름이 끊기던 문제, (5) 저장값이 배열이 아닐 때 `.some`/`.find` TypeError 및 `body.length` 접근 방어, (6) 로그인 정보 미확인 시 쿼리스트링에 문자열 `user_id=undefined` 가 실리던 문제, (7) `item.app_id === appId` 엄격 비교라 서버가 숫자 app_id 를 주면 라우트 쿼리(문자열)와 일치하지 않던 문제. 순수 헬퍼(`parseStoredList`, `extractAppList`)로 분리하고 `SupportImgList.vue` 의 `getAppInfo.app_id` (img 앱이 없으면 undefined 접근 TypeError)도 함께 고쳤다. 검증은 `npm test`(총 232건 통과, 신규 30건 — 수정 전 코드에 신규 스펙을 돌려 10건 실패함을 확인)와 `npm run build:dev`(빌드 성공)로 수행했다.
- 보류 아이디어: `mitt` 이 eventBus.js 에서 import 되는데 package.json 직접 의존성에 없어 전이 의존성에 기대는 문제 (가치 3 / 위험 1 / S) · `authStorage.readAuth` 가 localStorage 폴백 시 sessionStorage 를 복구하지 않고 `clearAuth` 도 통지가 없어 userStorage 와 동작이 어긋나는 문제 (가치 3 / 위험 2 / S) · Header.vue 가 OCR 사용 여부와 무관하게 마운트 즉시 3초 폴링을 시작해 statusOcr API 를 계속 호출하는 문제 (가치 3 / 위험 3 / M) · `globalLoading` 이 참조 카운트 없이 boolean 이라 병렬 요청 중 하나만 끝나도 스피너가 사라지는 문제 (가치 3 / 위험 3 / S) · 빌드 산출물 단일 청크 6.2MB 문제(manualChunks 코드 스플리팅) 개선 (가치 3 / 위험 3 / M)

## 2026-09-09
- 선택: 인증/채팅세션 저장소 방어 및 탭 동기화 플래그 충돌 수정 (가치 3 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 미테스트 상태였던 `src/storage/authStorage.js` 와 `src/storage/sessionChatStroage.js` 에서 7건을 고쳤다 — (1) `sessionChatStroage` 의 초기화 가드가 `window._userStorageSyncInitialized` 로 **userStorage 와 같은 플래그**를 써서, 먼저 로드된 모듈이 플래그를 세우면 나머지 모듈의 storage 리스너가 아예 등록되지 않던 문제(router/index.js 가 MainLayout 을 먼저 import 하므로 실제로는 sessionChat 쪽 탭 동기화가 죽어 있었고, import 순서가 바뀌면 2026-09-07 에 고친 userStorage 탭 동기화가 대신 죽는 잠재 회귀), 전용 플래그 `_sessionChatStorageSyncInitialized` 로 분리, (2) `authStorage` 가 sessionStorage/localStorage 접근을 전혀 감싸지 않아 쿠키 차단·시크릿 모드에서 `readAuth`/`getAccessToken` 이 예외를 던지던 문제 — 이 경로는 request 인터셉터를 포함한 모든 API 호출이 지나므로 앱 전체가 멈춘다(userStorage 와 동일하게 try/catch 로 방어), (3) `safeParse` 가 `'null'`/`'"str"'`/`'[1,2]'` 같은 비객체 JSON 을 그대로 돌려줘 `interceptors.js` 의 `readAuth().authority` 에서 예외가 나거나 `setAccessToken` 의 `{ ...prev }` 가 문자열 인덱스 키를 퍼뜨리던 문제(항상 평범한 객체로 정규화), (4) `writeAuth` 가 객체가 아닌 값을 받으면 토큰 4개를 전부 빈 문자열로 덮어 세션이 조용히 끊기던 문제, (5) `readAuth`/`readSessionChat` 이 localStorage 폴백 시 sessionStorage 를 복구하지 않아 매 호출마다 두 저장소를 읽던 문제(모듈 주석의 "sessionStorage 우선" 의도와 불일치), (6) `clearAuth`/`clearSessionChat` 의 storage 예외 미방어, (7) 로그아웃 경로(`HeaderSetting.logout`, `common.redirectToLogin`)가 `clearUser`/`clearAuth`/`clearMyNoteBookCache` 는 부르면서 `clearSessionChat` 은 빠뜨려 이전 사용자의 채팅 세션 id 가 localStorage 에 남던 문제(라우터 가드가 비-chat 경로에서만 지우므로 다음 사용자가 chat 딥링크로 진입하면 남의 세션을 불러올 수 있음). 검증은 `npm test`(총 308건 통과, 신규 33건 — 수정 전 코드에 신규 스펙을 돌려 11건 실패함을 확인)와 `npm run build:dev`(빌드 성공)로 수행했다.
- 보류 아이디어: `mitt` 이 eventBus.js 에서 import 되는데 package.json 직접 의존성에 없어 전이 의존성에 기대는 문제 (가치 3 / 위험 1 / S) · Header.vue 가 OCR 사용 여부와 무관하게 마운트 즉시 3초 폴링을 시작해 statusOcr API 를 계속 호출하는 문제 (가치 3 / 위험 3 / M) · `globalLoading` 이 참조 카운트 없이 boolean 이라 병렬 요청 중 하나만 끝나도 스피너가 사라지는 문제 (가치 3 / 위험 3 / S) · serviceCode 리터럴 표기 불일치(myNoteBook/MyNotebook/multiModal) 상수화 (가치 3 / 위험 2 / S) · 빌드 산출물 단일 청크 6.2MB 문제(manualChunks 코드 스플리팅) 개선 (가치 3 / 위험 3 / M)

## 2026-09-09
- 선택: 공용 저장소(commonStorage) 사용자 캐시 로그아웃 잔존 및 storage 예외 방어 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 미테스트 상태였던 `src/storage/commonStorage.js` 에서 6건을 고쳤다 — (1) 로그아웃 경로(`HeaderSetting.logout`, `common.redirectToLogin`)가 `clearUser`/`clearAuth`/`clearMyNoteBookCache`/`clearSessionChat` 은 부르면서 common 저장소는 그대로 둬, 이전 사용자의 기본앱(`defaultBot`)·서비스 목록(`serviceList`)·사이드메뉴 앱 목록(`sidemenuAppList`)·열린 앱/문서 상태(`openApp`/`openAppDoc`/`docState`)·개인 헤더 설정이 남아 다음 사용자에게 그대로 보이던 문제(레이아웃 상태 `sidebar_open`/`headerBar` 만 남기는 `clearCommonUserData()` 추가 후 두 로그아웃 경로에 연결 — 2026-09-08 의 myNoteBook 캐시 잔존과 같은 종류이며 이쪽이 상위 저장소), (2) localStorage 접근을 전혀 감싸지 않아 쿠키 차단·시크릿 모드에서 `commonStorage.get` 이 예외를 던져 MainLayout setup 과 스토어 초기화가 통째로 실패하던 문제(authStorage 와 동일한 `readRaw`/`writeRaw`/`removeRaw` 방어), (3) `safeParse` 가 `'null'`/`'"str"'`/`'[1,2]'` 를 그대로 돌려줘 `get(key)` 가 문자열 인덱스를 반환하거나 `remove(key)` 의 `delete` 에서 TypeError 가 나(useSettingStore.clearHeaderSetting·useServiceStore 의 clear 가 예외로 끊김) 문제, (4) `set()` 이 배열도 통과시켜 인덱스 키를 저장소에 퍼뜨리던 문제, (5) 용량 초과(SIDEMENU_APP_LIST + SERVICE_LIST 가 한 덩어리)·순환 참조 시 `setItem` 이 호출부로 예외를 던져 Sidemenu 앱 목록 저장에서 렌더가 끊기던 문제, (6) `remove()` 가 없는 key 에도 저장소를 다시 쓰던 불필요한 write. 검증은 `npm test`(총 332건 통과, 신규 24건 — 수정 전 코드에 신규 스펙을 돌려 13건 실패함을 확인)와 `npm run build:dev`(빌드 성공)로 수행했다.
- 보류 아이디어: `mitt` 이 eventBus.js 에서 import 되는데 package.json 직접 의존성에 없어 전이 의존성에 기대는 문제 (가치 3 / 위험 1 / S) · Header.vue 가 OCR 사용 여부와 무관하게 마운트 즉시 3초 폴링을 시작해 statusOcr API 를 계속 호출하는 문제 (가치 3 / 위험 3 / M) · `globalLoading` 이 참조 카운트 없이 boolean 이라 병렬 요청 중 하나만 끝나도 스피너가 사라지는 문제 (가치 3 / 위험 3 / S) · serviceCode 리터럴 표기 불일치(myNoteBook/MyNotebook/multiModal) 상수화 (가치 3 / 위험 2 / S) · `appDefaultStorage.getDefaultApp` 이 실패 시 빈 문자열을 반환하고 `res.data?.body[0]` 로 body 미존재 시 TypeError 를 내는 문제 정리 (가치 2 / 위험 1 / S)

## 2026-09-09
- 선택: 기본앱 조회(appDefaultStorage) 실패 처리 및 마운트 시 중복 API 호출 제거 (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: 미테스트 상태였던 `src/storage/appDefaultStorage.js` 에서 5건을 고쳤다 — (1) `getDefaultApp` 이 실패 시 `readDefaultApp` 의 계약(객체|null)과 다른 빈 문자열을 반환하던 문제를 `null` 로 정정, (2) `res.data?.body[0]` 가 body 미존재 시 TypeError 를 던지고(밖의 try/catch 에 삼켜져 "기본앱이 없습니다" 로만 보임) 서버가 배열 대신 객체 하나를 줄 때도 실패하던 문제를 순수 헬퍼 `extractDefaultApp` 으로 분리, (3) Sidemenu·Chat/Main·Chat/Index 가 같은 시점에 마운트되어 각각 `readDefaultApp` 을 부르는데 첫 진입에는 전부 캐시 미스라 `/app/default` 를 동시에 3번 호출하던 문제를 진행 중 요청 재사용(inflight)으로 1회로 합침, (4) 응답 대기 중 로그아웃·사용자 전환이 일어나면 이전 사용자의 기본앱이 common 캐시에 다시 써지던 문제(로그아웃은 `clearCommonUserData` 로 저장소만 비우고 모듈 메모리의 진행 중 요청은 못 지운다)를 완료 시점 user_id 재확인으로 차단, (5) 캐시에 객체가 아닌 값이 들어 있으면 그대로 반환하던 문제를 정규화 후 재조회로 변경. 함께 `Sidemenu.gotoChatMain` 이 기본앱이 없을 때 `String(undefined)` 로 `?appId=undefined` 를 붙여 이동하던 버그도 고쳤다. 검증은 `npm test`(총 347건 통과, 신규 15건 — 수정 전 코드에 신규 스펙을 돌려 10건 실패함을 확인)와 `npm run build:dev`(빌드 성공)로 수행했다.
- 보류 아이디어: `mitt` 이 eventBus.js 에서 import 되는데 package.json 직접 의존성에 없어 전이 의존성에 기대는 문제 (가치 3 / 위험 1 / S) · ChatStorageDetail 의 `computed(() => getDefaultBot())` 이 async 함수를 감싸 Promise 를 담고 있어 `isGeneralChat` 이 항상 false 인 문제 (가치 3 / 위험 2 / S) · Header.vue 가 OCR 사용 여부와 무관하게 마운트 즉시 3초 폴링을 시작해 statusOcr API 를 계속 호출하는 문제 (가치 3 / 위험 3 / M) · `globalLoading` 이 참조 카운트 없이 boolean 이라 병렬 요청 중 하나만 끝나도 스피너가 사라지는 문제 (가치 3 / 위험 3 / S) · serviceCode 리터럴 표기 불일치(myNoteBook/MyNotebook/multiModal) 상수화 (가치 3 / 위험 2 / S)

## 2026-09-10
- 선택: 대화저장 상세(ChatStorageDetail)의 기본앱 비동기 처리 오류 및 봇 식별자 파싱 공용화 (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `src/views/ChatStorage/ChatStorageDetail.vue` 가 async 함수 `readDefaultApp` 을 `computed(() => getDefaultBot())` 로 감싸 Promise 를 담고 있어, `pickBotIds(Promise)` 가 항상 빈 식별자를 돌려주고 `appId`/`prjId` 의 기본앱 폴백이 죽어 있던 문제를 고쳤다 — 일반챗봇 히스토리(라우트에 appId 쿼리가 없는 진입)에서 `pInf.chat.getHistorySession`(대화 불러오기)과 `inf.chat.feedbackAnswer`(피드백 저장)가 `app_id`/`project_id` 를 빈 문자열로 보내고, `isGeneralChat` 이 영원히 false 라 헤더가 항상 "전문가 AI와 대화중" 으로 표시됐다. `defaultBot` 을 ref 로 바꾸고 `initByRoute` 가 세션 조회 전에 `ensureDefaultBot()` 으로 선로딩하도록 했다(readDefaultApp 은 캐시·inflight 합치기가 있어 추가 호출이 생기지 않는다). 함께 Chat/Main·Chat/Index·ChatStorageDetail 에 각각 복사돼 있던 `pickBotIds` 를 `appDefaultStorage.pickBotIds` 로 합쳐(app_id/appId/appID·project_id/prj_id/prjId 표기 모두 처리, 비객체·Promise 방어) Main.vue 의 `app_id` 단일 표기 사본을 제거했다. 검증은 `npm test`(총 353건 통과, 신규 6건)와 `npm run build:dev`(빌드 성공, dist 는 커밋 전 삭제)로 수행했다. 다만 이 저장소에는 컴포넌트 테스트 환경이 없어 .vue 수정 자체는 순수 헬퍼 테스트와 코드 리뷰로만 확인했다.
- 보류 아이디어: mitt 이 package.json 직접 의존성에 없어 전이 의존성에 기대는 문제 (가치 3 / 위험 1 / S) · Header.vue 가 OCR 사용 여부와 무관하게 마운트 즉시 3초 폴링을 시작하는 문제 (가치 3 / 위험 3 / M) · globalLoading 이 참조 카운트 없이 boolean 이라 병렬 요청 중 하나만 끝나도 스피너가 사라지는 문제 (가치 3 / 위험 3 / S) · serviceCode 리터럴 표기 불일치(myNoteBook/MyNotebook/multiModal) 상수화 (가치 3 / 위험 2 / S) · 빌드 산출물 단일 청크 6.2MB 코드 스플리팅(manualChunks) (가치 3 / 위험 3 / M)

## 2026-09-10
- 선택: 열린 앱(openApp/openAppDoc) 저장 레코드 표기 불일치 정규화 및 임베딩 파일명 복원 공용화 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 미테스트 상태였던 `src/storage/useOpenAppStroage.js` 에서 3건을 고쳤다 — (1) `writeOpenApp` 이 두 가지 표기를 섞어 저장하던 문제: Sidemenu.toggleApp/gotoChatMain 과 appUpdateLog.updateLogAndGo 는 `{ appId, prjId, appName, _raw }` 로 쓰는데 `appList.getAppInfo:96` 은 서버 원본(`{ app_id, project_id, name, mode }`)을 그대로 저장하고, 읽는 쪽(Sidemenu.gotoChatMain:63, appList.getAppInfo:71)은 항상 `appId`/`_raw` 를 꺼내 쓴다. 원본이 한 번 저장되면 `openedAppInfo?.appId` 가 undefined 라 심플봇 진입 후 로고 클릭 시 임베딩 재확인(`clearOpenAppDoc` + `getUsed` + `writeOpenAppDoc`)이 통째로 건너뛰어져 Chat/Main.resetInfo 가 낡은 임베딩 파일 목록/실패 안내를 보여줬고, getAppInfo 의 캐시 히트 경로도 죽어 있었다 — `normalizeOpenAppRecord` 로 어느 표기로 들어와도 같은 형태로 저장하고 `readOpenApp` 도 읽는 시점에 정규화해 이미 원본 표기로 남아 있는 캐시까지 맞췄다(`_raw` 가 객체가 아니면 null 로 둬 캐시 히트 오인 방지). (2) `normalizeOpenAppDoc` 으로 `items` 를 항상 배열, `appId` 를 항상 문자열로 맞췄다 — 서버가 `knowledgeDetail_info` 를 주지 않으면 `writeOpenAppDoc({...undefined, appId: undefined})` 가 저장되고 라우트에 appId 가 없을 때 `undefined == undefined` 로 매칭돼 `Chat/Main.resetInfo` 의 `docinfo.items.map` 이 TypeError 를 던졌다. (3) `makeTimestampedFileName('|__|' 구분자)` 의 역변환이 Chat/Main.vue:508 과 PopSimpleBotUpdate.vue:209 에 각각 복사돼 있었는데, 구분자가 없는 파일명이면 `slice(0, -1)` 로 마지막 글자가 잘리고(`보고서.pdf` → `보고서.pd.pdf`) 확장자가 없으면 `slice(-1)` 이 타임스탬프 마지막 글자를 확장자로 붙였다 — `common.parseStoredFileName` 으로 합치고 `document`/`file_name` 누락 방어도 함께 넣었다. 검증은 `npm test`(총 382건 통과, 신규 29건 — 수정 전 코드에 신규 스펙을 돌려 30건 실패함을 확인)와 `npm run build:dev`(빌드 성공, dist 는 커밋 전 삭제)로 수행했다. 다만 이 저장소에는 컴포넌트 테스트 환경이 없어 .vue 수정 자체는 순수 헬퍼 테스트와 코드 리뷰로만 확인했다.
- 보류 아이디어: mitt 이 package.json 직접 의존성에 없어 전이 의존성에 기대는 문제 (가치 3 / 위험 1 / S) · getAppInfo 의 'all' 폴백이 SIDEMENU_APP_LIST 가 아닌 APP_LIST 에 캐싱해 매번 재조회하는 문제 (가치 2 / 위험 2 / S) · Header.vue 가 OCR 사용 여부와 무관하게 마운트 즉시 3초 폴링을 시작하는 문제 (가치 3 / 위험 3 / M) · globalLoading 이 참조 카운트 없이 boolean 이라 병렬 요청 중 하나만 끝나도 스피너가 사라지는 문제 (가치 3 / 위험 3 / S) · serviceCode 리터럴 표기 불일치(myNoteBook/MyNotebook/multiModal) 상수화 (가치 3 / 위험 2 / S)

## 2026-09-17
- 선택: 채팅 이동 데이터(chatTransitionStorage)가 다른 세션 진입 시 이전 질문을 재전송하던 문제 및 레코드 정규화 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: 보류 목록에 [2/1/S]로 있던 `chatTransitionStorage` 방어 보강을 재평가하다가 더 큰 실사용 버그를 찾았다 — `Chat/Main.sendChat` 이 `writeChatTransitionData({ mainQuestion, sessionId, appId… })` 로 저장하는 레코드를 `Chat/Index.vue` 가 **어느 세션용인지 확인하지 않고** 읽는다. 5 분 만료 전에 홈 위젯(RowListItem/CardListItem)·챗메인 히스토리 목록(Chat/Main.gotoChat)으로 다른 세션 S2 에 진입하면(이 경로들은 `removeChatTransitionData` 를 부르지 않음) `initByRoute` 의 `if (sid && transitionData?.mainQuestion)` 가 "새 질문 시작" 으로 오인해 `loadChatSession` 을 건너뛰고 화면을 비운 뒤, `onMounted` 가 저장된 이전 질문을 S2 세션으로 다시 전송했다(히스토리 대신 남의 세션에 옛 질문이 실림). `readChatTransitionData(match)` 에 세션/앱 매칭(`matchesTransitionTarget`, 빈 문자열은 "세션 없음" 으로 참여)을 추가하고 Index.vue 의 읽기 3곳(initByRoute·onMounted·mapHistorySession 의 previewUrl 복원)에 지금 진입한 `session_id`/`appId` 를 넘기도록 했다. 매칭 실패는 삭제가 아닌 무시라 원래 세션으로 돌아오면 여전히 쓸 수 있고, 새로고침 대응(같은 라우트 재진입)도 그대로다. 함께 timestamp 없는 레코드 영구 잔존·배열/문자열 JSON·비배열 sentFiles 를 `normalizeTransitionData`/`isTransitionExpired` 로 정리했다. 검증은 `npm test`(총 408건 통과, 신규 26건 — 수정 전 모듈에 신규 스펙을 돌려 17건 실패함을 확인)와 `npm run build:dev`(빌드 성공, dist 는 커밋 전 삭제)로 수행했다. 이 저장소에는 컴포넌트 테스트 환경이 없어 Index.vue 수정 자체는 순수 헬퍼 테스트와 코드 리뷰로만 확인했다.
- 보류 아이디어: mitt 이 package.json 직접 의존성에 없어 전이 의존성에 기대는 문제 (가치 3 / 위험 1 / S) · Header.vue 가 OCR 사용 여부와 무관하게 마운트 즉시 3초 폴링을 시작하는 문제 (가치 3 / 위험 3 / M) · globalLoading 이 참조 카운트 없이 boolean 이라 병렬 요청 중 하나만 끝나도 스피너가 사라지는 문제(호출부 142곳) (가치 3 / 위험 3 / S) · useAppList/Sidemenu 의 `String(app?.knowledge_info?.status) ?? ''` 가 status 미존재 시 문자열 "undefined" 를 만드는 문제 (가치 2 / 위험 1 / S) · serviceCode 리터럴 표기 불일치(myNoteBook/MyNotebook/multiModal) 상수화 (가치 3 / 위험 2 / S)

## 2026-09-19
- 선택: [수정 과제] 예산 홀드 오류 대응 판정 + `mitt` 를 package.json 직접 의존성으로 선언 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: **오류 대응 판정** — 직전 회차(2026-09-18-162350)의 `error` 는 `hold: budget`(회차 예산 $22 > 일일 잔여)으로 러너가 구현 단계를 시작하기 전에 보류한 것이며, 저장소 코드·테스트·워크플로의 실패가 아니다. 이 저장소에는 `.github/workflows` 가 없고(이번 회차 `ls .github` 로 재확인) 태그·릴리즈 관례도 없어 "같은 이유로 두 번 실패한 릴리즈 워크플로" 에 해당하는 파일·스크립트·테스트가 존재하지 않는다 — 예산 홀드는 저장소 외부(aidev 러너 예산 정책) 원인이며 코드 수정 대상이 없고, 느슨하게 만들 워크플로도 없다. 회차를 비우지 않기 위해 예산이 가장 적게 드는 S 과제를 수행했다: `src/utils/eventBus.js` 가 `import mitt from 'mitt'` 로 쓰고 10곳(Sidemenu/Home/Chat/MyAgentList/SupportList/PopWidgetSetting/appList/appUpdateLog/useAppList)이 의존하는데 `mitt` 가 package.json 어디에도 없고 lock 상 devDependency `vite-plugin-vue-devtools` → `@vue/devtools-kit` 의 전이 의존성으로만 설치되고 있었다(devtools 플러그인 제거·업그레이드나 hoisting 변경 시 `Rollup failed to resolve import "mitt"` 로 프로덕션 빌드가 깨질 수 있음). `dependencies` 에 `"mitt": "^3.0.1"` 을 추가하고 `npm install mitt@^3.0.1` 로 lock 을 갱신했다 — lock 변경은 최상위 `packages[""].dependencies` 의 한 줄뿐이고 설치 버전(3.0.1)·다른 패키지 변동 없음(npm 이 package.json 키를 재정렬한 것은 원래 순서로 되돌리고 `marked` 다음에 손으로 추가). `tests/unit/eventBus.spec.js` 를 신규 추가해 mock 없이 실제 `@/utils/eventBus.js` 를 import 하여 `bus` 가 실제 mitt 인스턴스(on/emit/off/all·와일드카드·싱글턴)로 동작함과 `BUS_EVENT` 상수를 검증한다(선언 누락 시 모듈 해석 실패로 잡히는 회귀 테스트). 검증은 `npm ci` → `npm test`(총 415건 통과, 신규 7건) → `npm run build:dev`(빌드 성공, dist 삭제) → `git diff --stat` 으로 변경이 package.json / package-lock.json / tests/unit/eventBus.spec.js 3개(+79줄)에 한정됨을 확인했다. 커밋 27e86f9.
- 보류 아이디어: useAppList/Sidemenu 의 `String(app?.knowledge_info?.status) ?? ''` 가 status 미존재 시 "undefined" 문자열을 만드는 문제 — `String(status ?? '')` + getFormattedAppList export/테스트 (가치 2 / 위험 1 / S) · Header.vue 가 OCR 사용 여부와 무관하게 마운트 즉시 3초 폴링을 시작하는 문제 (가치 3 / 위험 3 / M) · globalLoading 이 참조 카운트 없이 boolean 이라 병렬 요청 중 하나만 끝나도 스피너가 사라지는 문제(호출부 142곳 짝 확인 선행) (가치 3 / 위험 3 / S) · serviceCode 리터럴 표기 불일치(myNoteBook/MyNotebook/multiModal) 상수화 (가치 3 / 위험 2 / S) · 빌드 산출물 단일 청크 6.2MB 코드 스플리팅(manualChunks) (가치 3 / 위험 3 / M)

## 2026-09-20
- 선택: serviceCode 비교를 대소문자 무시 공용 매처(isServiceCode)로 통일 — PopWorkFlow / MyAgentList 가 같은 API 응답을 서로 다른 표기로 정확 일치 비교 (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: 같은 API(`inf.athena.appListGet`, `serviceMenu=workTools`) 응답의 `serviceCode` 를 `PopWorkFlow.getMyNoteBookProjectId` 는 `=== 'myNoteBook'`, `MyAgentList.getWorkToolsList` 는 `=== 'MyNotebook'` 으로 정확 일치 비교하고 있어 서버가 어느 표기를 주든 한쪽은 빈 결과를 얻는 상태였다(2026-09-08 에 `extractAppList` 만 대소문자 무시로 고쳐져 세 읽기 경로가 같은 값을 다르게 읽는 상태). `src/storage/myNoteBookStorage.js` 에 순수 매처 `isServiceCode(row, code)` 와 `SERVICE_CODE_MY_NOTE_BOOK`/`SERVICE_CODE_WORK_TOOLS` 를 export 하고, `extractAppList`·PopWorkFlow 의 `find`·MyAgentList 의 `noteRows` 필터가 모두 이 매처를 쓰도록 바꿨다(MyAgentList 의 중복 필터는 `noteRows.flatMap` 으로 축약, `.vue` 는 import 교체 최소 diff). 검증은 매처 스펙 6건을 먼저 추가해 수정 전 코드에서 `isServiceCode is not a function` 으로 6건 실패함을 확인 → 구현 후 `npm ci` → `npm test`(총 421건 통과, 기존 415 + 신규 6) → `npm run build:dev`(빌드 성공, dist 삭제) → `git diff --stat` 으로 변경이 4개 파일(+77/-11)에 한정됨을 확인했다. 커밋 839f98d. 컴포넌트 테스트 환경이 없어 .vue 두 곳은 매처 테스트와 코드 리뷰로만 확인했다.
- 보류 아이디어: Header.vue 가 OCR 사용 여부와 무관하게 마운트 즉시 3초 폴링을 시작하는 문제(대기 항목 없으면 중단 + 60초 유예, 서버 status semantics 미확인) (가치 3 / 위험 3 / M) · globalLoading 참조 카운트 부재로 병렬 요청 중 스피너 조기 종료(호출부 153곳 짝 감사 선행) (가치 3 / 위험 3 / S) · PopWorkFlow.getMyNoteBookProjectId 가 팝업을 열 때마다 appListGet 을 직접 호출 — myNoteBookStorage 캐시가 appList 만 저장하고 project_id 는 저장하지 않아 재사용 불가 (가치 2 / 위험 2 / S) · MyAgentList 의 receiveProjectId 는 menuName '나의 노트북' 정확 일치 행에서, PopWorkFlow 의 project_id 는 serviceCode 일치 첫 행에서 고르므로 두 경로가 project_id 를 다르게 고를 수 있음(서버 menuName 변동 시 receiveProjectId 만 null) (가치 2 / 위험 2 / S) · MyAgentList 의 'MyAvatar'/'etc' 비교도 매처로 통일(표기 불일치 증거 없음 — 서버 응답 확인 후) (가치 2 / 위험 2 / S)
- 과제서: 채택 — 과제서의 근거(PopWorkFlow.vue:118 'myNoteBook' / MyAgentList.vue:301,307 'MyNotebook' 정확 일치 비교)가 현 코드와 정확히 일치했고 수용 기준 1~3 을 모두 충족해 그대로 구현했다.

## 2026-09-20
- 선택: usePaging.resetPageAndGet의 페이지 초기화 후 중복 목록 조회 제거 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: 초기화 횟수를 page와 함께 관찰하여 직접 조회한 1페이지의 watcher 중복 호출을 없애고, 같은 tick에서 다른 페이지로 이동하는 조회 및 원래 값/Promise 반환을 유지했다. 실제 Vue ref/watch/nextTick/effectScope와 native Promise 테스트로 수정 전 5건 실패를 확인했고, npm ci 후 페이지 테스트 16건·전체 424건 및 build:dev가 통과했다. 회사 technology 스킬/Skill 도구는 발견하지 못했으며 별도 로컬 superpowers systematic-debugging·test-driven-development 지침을 대체 사용했다(회사 스킬 반환 형식 준수는 주장하지 않음).
- 보류 아이디어:
  - useAppList 비배열 SIDEMENU_APP_LIST 캐시 방어 (가치 3 / 위험 2 / 작업량 S)
  - 테스트 가이드의 없는 문서 링크 정리 (가치 2 / 위험 1 / 작업량 S)
  - OCR 미사용 시 Header 폴링 조정 (가치 3 / 위험 3 / 작업량 M; 서버 계약 확인 선행)
  - globalLoading 병렬 요청 참조 카운트 (가치 3 / 위험 3 / 작업량 S; 호출 짝 감사 선행)
- 과제서: 채택 — 실제 코드와 수정 전 Vue 테스트에서 직접 호출 및 watcher의 중복 조회를 확인했고 지정된 두 파일만 변경했다.

## 2026-09-20
- 선택: [수정 과제] Codex 폴백 스킬 경로·버전 증거 기록 및 aidev 담당 이관 (가치 5 / 위험 2 / 작업량 M)
- 결과: 변경없음
- 요약: 현재 작업 표면은 aiportal-front@dd65af7이며 외부 aidev 구현 회차로 이관되지 않아, 과제서의 명시적 외부 수정 금지에 따라 러너·앱 코드와 테스트를 수정하지 않았다. aidev/bin/run.sh:245~340, agents/registry.json, release-prompt.md, tests/test_sim.py 및 tests/sim/run_sim.sh를 읽어 Claude용 Skill 안내의 Codex 전달과 원문 경로 누락을 정적으로 재확인했고, release 2개·scout 3개 원문 파일의 읽기 가능 여부 및 package/lock 0.0.0 대 문서 0.0.1 모순도 확인했다. 이번 실행은 bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh 통과와 파일 확인뿐이며, gate 27건 통과는 앞선 정찰 증거이고 신규 폴백 재현·Agents·전체 sim·앱 test/build는 미실행이므로 수정 완료나 릴리즈 성공/skipped를 주장하지 않는다.
- 보류 아이디어:
  - Codex 엔진별 원문 경로 전달 및 quota 폴백 회귀: aidev 별도 구현 회차 필요; pending 유지.
  - 릴리즈 버전 근거 불일치: 실제 배포 기록과 증가 관례 확인 전 미해결.
  - useAppList 비배열 캐시 방어: 이번 우선 과제와 무관하여 보류.
  - globalLoading 병렬 요청 참조 카운트: 호출 짝 감사 전 보류.
- 과제서: 차선 — 지정 구현 파일은 외부 aidev 저장소에 있고 이번 회차에서 직접 수정하지 말라는 지시가 유지되어, 허용된 회차 기록으로 이관 근거를 남겼다.

### 이관 근거와 검증 한계
- Skill 호출 도구는 제공된 callable 도구 목록에서 찾지 못했으나 technology:completion-verification, technology:systematic-debugging, technology:test-driven-development의 실제 SKILL.md를 읽고 적용했다. 앞선 정찰 설명과 달리 systematic-debugging과 test-driven-development에는 Return contract 절이 존재한다.
- 재현: 실패 기록은 ../2026-09-20-211415-aiportal-front-improve/release.json의 failed. 이번 동적 quota 폴백 재현은 미실행이므로 정적 전달 결함과 실패 기록의 일치를 인과관계 회귀 검증 완료로 취급하지 않는다. 동일 원인 두 실패는 독립 확인하지 않았다.
- 원인 근거: run_agent가 dept_note를 prompt에 붙인 후 run_codex에 그대로 전달한다. run_codex는 registry/HEADCOUNT_DIR/SKILL.md 경로를 안내하지 않는다. tests/sim/bin에는 claude와 gh만 있고 codex 스텁이 없다.
- release 원문: /mnt/c/Users/USER/projects/headcount/plugins/marketing/skills/product-launch/SKILL.md 및 /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/release-and-deployment/SKILL.md — 모두 읽기 확인.
- scout 원문: /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md, /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md, /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md — registry의 3개 모두 읽기 확인.
- 수정 및 추가 테스트: 없음. 외부 범위 차단 때문에 Red/Green 및 수정 되돌림 검증 미실행. 이관 후 가짜 claude/codex로 release/scout 및 직접 review 호출, HEADCOUNT_DIR 재정의·공백·NO-HEADCOUNT·agents.headcount=false·활성 상태 누락 이름/경로 보고를 검증해야 한다. 실제 모델 호출 금지.
- 이관 후 검증: aidev 루트에서 bash -n bin/run.sh, PYTHONDONTWRITEBYTECODE=1 python3 tests/test_gate.py, 신규 등록 폴백 테스트 단독 실행, PYTHONDONTWRITEBYTECODE=1 python3 tests/test_sim.py Agents, PYTHONDONTWRITEBYTECODE=1 python3 tests/test_sim.py. 기존 gate 27건을 포함해 확인하되 이번 결과로 대체하지 않는다.
- README.md:438 및 docs/01-시작하기.md:359는 0.0.1(2025-01-01), package.json 및 lock 두 버전 필드는 0.0.0. release-prompt.md 절차 5는 버전 파일도 없는 경우만 skipped를 허용하므로 스킬 경로 복구와 별개 장애다.
- 앱 CI 잡 실패를 증명하는 로그는 없고 외부 에이전트 단계의 실패 기록이 있다. .gitlab-ci.yml·gate·릴리즈 규칙·버전·태그·headcount 원문은 수정하지 않았고 커밋·배포·원격 전송도 하지 않았다.

## 2026-09-21
- 선택: [수정 과제] 릴리즈 입력 복구 — 회사 스킬 탐색 안내와 버전 근거 정본화 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 수정 과제 — 릴리즈 입력 복구, 전체 릴리즈 미검증. AGENTS.md에 활성 세션이 요구한 스킬만 Skill 도구 우선·부재 시 실제 원문 읽기·누락 시 이름/경로/미확인 기록을 안내하고, 두 문서의 초기 릴리스 표를 docs/RELEASE.md로 통일해 기존 0.0.1/2025-01-01 기재와 실제 릴리스 미확인 상태, 현재 세 값 0.0.0 및 최초 lock 버전 필드 부재를 보존했다. 지정 Git/JSON 재조회와 실제 스킬 5개 읽기, 정본 명령 실행, 링크 대상 읽기, bash -n 및 git diff --check가 통과했으며 다음 버전 정책은 미해결이다.
- 보류 아이디어:
  - 외부 Codex 폴백 원문 경로 전달: 5/2/M, 외부 aidev 소유이며 동적 검증 필요.
  - useAppList 비배열 캐시 방어: 3/2/S, 이번 문서 과제와 무관하여 보류.
  - globalLoading 병렬 요청 참조 카운트: 3/3/S, 호출 짝 감사 선행.
  - 테스트 가이드의 없는 문서 링크 정리: 2/1/S, 별도 문서 과제로 보류.
- 과제서: 채택 — 실제 원문 접근 가능 여부와 Git/JSON 결과가 과제서에 일치하며 AGENTS 추가를 금지하는 저장소 규칙은 발견되지 않았다.

검증 및 스킬 반환 기록:
- 재현/원인: 수정 전 두 표의 단정적 0.0.1 기재와 현재 0.0.0, 최초 lock 필드 부재를 직접 조회했다. 실패 release.json과 외부 run.sh/release-prompt.md/test_sim.py의 run·HappyPath·Agents 및 sim/run_sim.sh를 읽어 Skill 안내에 원문 경로가 없음을 확인했다. 앱 CI 실패를 입증하는 로그는 없어 앱 빌드 결함으로 분류하지 않았다.
- 수정: 원문 탐색 진입점과 버전 근거 정본화, 문서 4개만 변경. 과제서 체크포인트 1~4 완료. CI·러너·headcount·버전·태그·원격 서비스는 변경하지 않았다.
- 실행: git rev-parse --is-shallow-repository → false; git tag --sort=-creatordate → 빈 출력; git log --oneline -60 및 파일 이력 조회 성공; 지정 Python → CURRENT 0.0.0 0.0.0 0.0.0, 최초 lock ABSENT ABSENT, 기존 두 표 확인, READ OK 5개. 정본의 두 bash 블록도 실제 실행하여 종료 0; 링크 대상 7개 읽기 성공; bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh 및 git diff --check 종료 0.
- 스킬: Skill 도구 없음. technology:completion-verification/systematic-debugging/test-driven-development의 실제 SKILL.md를 읽고 적용했다. 파일 읽기를 Skill 호출로 기록하지 않는다.
- 테스트 추가 없음: 실행 코드가 없는 문서 변경이므로 문구 문자열 회귀 테스트는 만들지 않았다. 앱 npm ci/test/build, 전체 sim, 실제 모델/releaser, 동적 quota 폴백은 미실행. Red/Green 및 수정 되돌림에 의한 릴리즈 인과 검증도 미실행이며 전체 장애 해결 완료로 보고하지 않는다.
- 잔여: 다음 버전 증가·커밋·태그 관례는 승인된 정책 또는 실제 릴리즈/배포 증거가 필요하다. 버전 파일 존재로 이력 전무 skipped 조건 불충족; 독립된 동일 실패 2회는 미확인.

## 2026-09-21
- 선택: [수정 과제] 릴리즈 버전 결정 입력 복구 — 승인 근거 확보 후에만 구현 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 수정 과제: 필수 정책 입력 부재로 차단, 코드 변경 없음. docs/RELEASE.md와 실제 Git/JSON, 독립 실패 두 건, 외부 release-prompt/run.sh/gate 및 테스트를 대조했지만 승인된 다음 버전·증가 단위·릴리즈 커밋·태그 사용/형식/종류·노트·자산 방식은 확보하지 못했다. 실제 최신 실패 JSON을 수정하지 않은 gate.py release에 전달하여 exit 1/ok=false/state=failed를 재현했으며, gate 테스트 27건(ResourceWarning 있음)·bash -n·git diff --check 통과는 전체 해결이 아니다.
- 보류 아이디어:
  - 릴리즈 결정 입력 복구: pending/BLOCKED, 승인 원문 또는 실제 릴리즈 기록 필요.
  - 외부 Codex 폴백 경로 전달: 외부 aidev 소유, 최신 실패는 원문 접근 이후 발생하여 이번 해법으로 재시도하지 않음.
  - useAppList 비배열 캐시 방어: 우선 배정과 무관하여 보류.
  - globalLoading 병렬 요청 참조 카운트: 호출 짝 감사 선행, 이번 변경 없음.
- 과제서: 채택 — 현재 근거가 과제서의 진입 차단 조건과 일치하여 대체 과제 없이 저장소 무변경과 차단 사유를 기록했다.

검증 및 스킬 반환 기록:
- Skill callable 도구 없음. /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/{completion-verification,systematic-debugging,test-driven-development}/SKILL.md 실제 원문 세 개를 읽고 적용했으며 Skill 호출 성공으로 기록하지 않는다.
- 재현/원인: release-prompt.md 절차 2는 과거 증가 패턴을 요구하지만, HEAD e938e8e의 전체 37커밋·로컬 태그 0개·현재 세 버전 0.0.0 및 패키지 이력에 해당 근거가 없다. 최초 lock 두 필드는 ABSENT. 정책의 autonomy=release/allow_merge_without_ci는 버전 정책을 제공하지 않는다.
- 수용 기준별 출처: 다음 버전/증가 단위/릴리즈 커밋 양식/태그 사용 여부·형식·종류/노트 방식/자산 방식 모두 승인 원문 또는 실제 릴리즈 기록 미확보. release-context.md의 GitHub Release 목록 공란·워크플로 없음은 승인된 미사용 정책이 아니다. 원격 현재 상태는 조회하지 않았다.
- 소비 경로: run.sh release_project → run_agent/run_codex → 실제 release.json → gate.py cmd_release/evaluate_release. failed를 차단하는 소비 단계는 재현했으나 모델 버전 결정 자체는 재실행하지 않았다. 최신 실패 기록은 스킬 원문 접근 성공 후 버전 근거 부족을 명시한다. 앱 CI 실패 증거는 없고 .gitlab-ci.yml은 브랜치 빌드·복사 배포다.
- 수정/테스트 추가: 없음. 필수 입력 부재로 구현에 진입하지 않아 Red/Green/수정 되돌림 인과 검증은 미실행. sim은 사전 v0.0.1 태그와 에이전트 대역을 사용하여 이번 실제 판단을 증명하지 못하므로 실행하지 않았다.
- 실행 명령과 전체 출력: 같은 회차 implementation-checks.json. PYTHONDONTWRITEBYTECODE=1 및 TMPDIR=이번 회차 assets로 gate 27건 실행. 이번 실행에서 경고에 나온 임시 자산 디렉터리는 삭제했다.
- 앱 npm ci/test/build, 전체 releaser, 원격 게시·배포 미실행. 코드·버전·태그·문서·외부 러너·게이트·실패 JSON 무변경, 커밋 없음. released/skipped/수정 완료로 판정하지 않는다.

## 2026-09-21
- 선택: [수정 과제] 릴리즈 실패의 필수 정책 입력 결손 해소 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 수정 과제는 BLOCKED이며 수정 미완료다. 새 승인 원문이나 실제 릴리즈 이력이 제공되지 않아 과제서의 명시적 중단 조건에 따라 구현·동일 재조사에 진입하지 않았고, 저장소와 외부 러너를 변경하지 않았다. 이번에는 요구 스킬 원문·AGENTS.md·docs/RELEASE.md·회차 기록을 읽고 작업 트리 무변경과 기록 형식을 확인했으며, 정찰의 gate 27건 및 실패 두 건 차단 결과는 이번 실행 결과가 아니다.
- 보류 아이디어:
  - 릴리즈 정책 입력 확보: pending/BLOCKED; 다음 버전·증가 단위·커밋·태그 사용/형식/종류·노트·자산 방식의 실제 출처 필요.
  - 외부 Codex 폴백 경로 전달: pending; 외부 aidev 소유이며 최신 실패 해법으로 재선정 금지.
  - useAppList 비배열 캐시 방어: pending; 이번 고정 과제의 대체 후보 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝과 실제 병렬 요청 검증 선행.
- 과제서: 채택 — 새 근거 없이는 구현·동일 재조사를 반복하지 말라는 진입 조건에 따라 차단 상태로 인계했다.

검증 및 스킬 반환 기록:
- callable Skill 도구 없음. /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/{completion-verification,systematic-debugging,test-driven-development}/SKILL.md 실제 원문을 읽고 적용했다. 파일 읽기는 Skill 호출 성공이 아니다.
- 재현/원인: 과제서의 필수 정책 입력 결손과 정찰 재현 결과를 인계받았다. 이번 독립 재현 및 새로운 원인 입증은 수행하지 않았으며, 앱 CI 결함이나 gate 결함으로 재분류하지 않았다.
- 수정/인과 검증: 없음. 수용 기준 1의 출처 미확보로 2·3 미착수. Red/Green 및 수정 되돌림 검증 미실행, 장애 해결 완료 주장 없음.
- 추가 테스트: 없음. 실행 코드 변경이 없고 반복 조사 금지에 따라 앱 npm ci/test/build, gate, sim, 실제 releaser를 재실행하지 않았다. 기존 정찰 결과는 scout-checks.json에 보존한다.
- 이번 확인: git status --short 빈 출력. 기록 작성 후 JSON 파싱·15개 기존 아이디어 보존·허용 상태 및 필수 필드·원장 단일 항목·구현 노트 8줄 이내 검사를 수행한다. 이는 릴리즈 통과 검증이 아니다.
- 버전·태그·정본 문서·실패 JSON·외부 게이트·원격 무변경, 커밋 없음. 원격 이력은 여전히 미확인이다. 차단은 사용자 과제서와 AGENTS.md의 새 관례 금지에 따른 것이며 새 승인 요청은 하지 않았다.

## 2026-09-21
- 선택: [수정 과제] 릴리즈 버전 결정 입력 복구 — 현재 배정 불가/BLOCKED (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 수정 과제는 BLOCKED이며 수정 미완료다. 이번 회차 과제서·기록에 새 승인 원문이나 실제 릴리즈 이력이 추가되지 않아 명시적 중단 조건에 따라 동일 조사·구현·gate 재실행 없이 차단 상태를 인계했다. 요구 스킬 원문과 AGENTS.md·docs/RELEASE.md·회차 기록을 읽고 기록 형식과 저장소 무변경만 확인했으며, 릴리즈 통과를 검증하지 않았다.
- 보류 아이디어:
  - 릴리즈 정책 입력 복구: pending/BLOCKED; 다음 버전·증가 단위·커밋·태그 사용/형식/종류·노트·자산 방식의 출처 미확보.
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 최신 실패 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 이번 고정 과제 대체 금지.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 병렬 요청 검증 선행.
- 과제서: 채택 — 새 근거 없이는 반복 조사나 구현을 하지 말라는 진입 조건에 따라 저장소 무변경과 차단 상태를 인계했다.

검증 및 스킬 반환 기록:
- 호출 가능한 Skill 도구 없음. /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md, /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md, /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md를 실제로 읽었다. 파일 읽기는 성공한 Skill 호출이 아니다.
- 재현/원인: 필수 정책 입력 결손과 정찰 재현 결과를 인계받았다. 이번 독립 재현·원인 입증·다른 원인 배제는 수행하지 않았으며 앱 CI 결함 또는 gate 결함으로 재분류하지 않는다.
- 수정/인과 검증: 없음. 수용 기준 1의 출처 미확보로 2·3 미착수. Red/Green 및 수정 되돌림 검증 미실행, 추가 테스트 없음.
- 의도적 미검증: 반복 실행 금지와 구현 미착수에 따라 앱 npm ci/test/build, gate 단위 테스트·release, sim, 실제 releaser를 실행하지 않았다. 정찰 결과는 scout-checks.json에 보존하며 이번 구현의 통과 근거로 사용하지 않는다. 원격 이력·실제 릴리즈 성공은 미확인이다.
- 이번 확인: git status --short 빈 출력. 기록 작성 후 Python으로 JSON 필수 필드·허용 값·17개 기존 아이디어 보존·원장 단일 항목·구현 노트 8줄 이내를 검사하고 git diff --check 및 git status --short를 실행한다. 이는 기록 검증이며 릴리즈 수용 기준 검증이 아니다.
- 저장소·버전·태그·정본·실패 JSON·외부 러너·게이트 무변경, 커밋·원격 전송·배포 없음. 새 아이디어 선정 절차는 정찰 과제서로 갈음하여 기존 17개를 보존한다.

## 2026-09-21
- 선택: [수정 과제] 릴리즈 버전 결정 입력 결손 해소 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 수정 과제는 필수 정책 출처가 제공되지 않아 수정 미완료/BLOCKED다. 과제서의 명시적 진입 조건에 따라 동일 조사·gate 재현·구현을 반복하지 않았으며 저장소 변경과 커밋은 없다. 요구 스킬 원문과 AGENTS.md·docs/RELEASE.md·회차 기록을 읽고 기록 형식과 작업 트리만 확인했으며, 릴리즈 해결 검증은 수행하지 않았다.
- 보류 아이디어:
  - 릴리즈 정책 입력 복구: pending/BLOCKED; 다음 버전·증가 단위·릴리즈 커밋·태그 사용/형식/종류·노트·자산 방식의 출처 필요.
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 최신 실패 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 이번 고정 과제 대체 금지.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 병렬 요청 검증 선행.
- 과제서: 채택 — 새 입력 없이는 반복 조사나 구현 없이 BLOCKED로 종료하라는 조건을 적용했다.

검증 및 스킬 반환 기록:
- callable Skill 도구 없음. /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md, /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md, /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md 원문을 실제 읽었다. 파일 읽기는 성공한 Skill 호출이 아니다.
- 재현·원인: 정찰의 정책 입력 결손 진단을 인계받았으며 이번 독립 재현·원인 입증·다른 원인 배제는 미수행이다. 앱 CI나 gate 결함으로 재분류하지 않는다.
- 수정·검증: 코드 수정·추가 테스트 없음. 수용 기준 1 출처 미확보로 2·3 미착수이며 Red/Green 및 되돌림 인과 검증은 수행하지 않았다.
- 의도적 미검증: 사용자 반복 실행 금지에 따라 npm ci/test/build, gate 단위 테스트·release, sim, 실제 releaser 미실행. 원격 이력 미확인. 과거 결과를 이번 통과 근거로 사용하지 않는다.
- 이번 기록 검사: Python JSON 파싱·필수 필드/허용 값·기존 19개 아이디어 보존·원장 단일 항목·구현 노트 8줄 이내 검사, git diff --check 및 git status --short. 릴리즈 통과 검증이 아니다.
- 버전·태그·정본 문서·실패 JSON·외부 러너·게이트 무변경, 커밋·배포·원격 쓰기 없음. 아이디어 선정은 과제서로 갈음하고 기존 19개를 보존했다.

## 2026-09-21
- 선택: [수정 과제] 릴리즈 필수 정책 입력 결손 해소 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 승인 정책 원문이나 실제 릴리즈 출처가 없어 수정 미완료/BLOCKED다. 과제서의 진입 조건에 따라 동일 조사·구현·gate 재실행 없이 종료하며 저장소 변경과 커밋은 없다. 요구 스킬 원문·AGENTS.md·docs/RELEASE.md·회차 기록을 읽고 기록 형식과 작업 트리만 검증하며 릴리즈 통과를 주장하지 않는다.
- 보류 아이디어:
  - 릴리즈 정책 입력 복구: pending/BLOCKED; 증가 단위·다음 버전·커밋·태그·노트·자산·검증 방식의 출처 필요.
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 최신 실패 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 이번 고정 과제의 대체 대상 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 병렬 요청 검증 선행.
- 과제서: 채택 — 신규 근거 없이는 구현·동일 재조사·gate 반복 없이 BLOCKED로 종료하라는 조건을 적용했다.

검증 및 스킬 반환 기록:
- 호출 가능한 Skill 도구 없음. /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md, /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md, /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md 원문을 실제 읽었다. 파일 읽기는 성공한 Skill 호출이 아니다.
- 재현·원인: 정찰의 정책 입력 결손 진단을 인계받았다. 이번 독립 재현·원인 입증·다른 원인 배제는 미수행이며 앱 CI 또는 gate 결함으로 재분류하지 않는다.
- 수정·인과 검증: 없음. 수용 기준 1 미충족으로 2·3 미착수. Red/Green 및 수정 되돌림 검증 미실행, 추가 테스트 없음.
- 의도적 미검증: 사용자 과제서의 반복 실행 금지에 따라 npm ci/test/build, gate, sim, 실제 releaser 미실행. 원격 이력 미확인. 과거 실행 결과는 이번 통과 근거가 아니다.
- 이번 기록 검증: Python으로 JSON 필수 필드·허용 값·기존 21개 아이디어 보존·원장 단일 항목·구현 노트 8줄 이내 검사; git diff --check 및 git status --short로 저장소 상태 확인. 이는 릴리즈 수용 기준 검증이 아니다.
- 저장소·버전·태그·정본 문서·실패 JSON·외부 러너·게이트 무변경, 커밋·배포·원격 전송 없음. 새 아이디어 선정은 정찰 과제서로 갈음했다.

## 2026-09-21
- 선택: [수정 과제] 릴리즈 버전 결정 근거 복구 (가치 5 / 위험 2 / 작업량 S)
- 결과: 정찰 완료, 수정 미완료/BLOCKED — 구현 가능한 앱 수정 배정 없음.
- 근거: non-shallow 37커밋·로컬 태그 0개·현재 버전 3곳 0.0.0, 기존 관례 출처 없음. 원격 조회는 git exit 128/gh exit 4 인증 실패로 미확인.
- 검증: Git/JSON 재조회·기록 형식 검사·git diff --check 및 상태 확인만 수행. 동일 릴리즈 통과·앱 test/build·gate·배포는 미실행.
- 재개 조건: 실제 승인 정책/과거 릴리즈 근거가 새로 제공된 경우만 재계획. 정본 재정리·판정 완화·임의 버전 증가·외부 수정으로 대체 금지.

## 2026-09-21
- 선택: [수정 과제] 릴리즈 버전 결정 입력 결손 해소 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 승인 정책이나 실제 릴리즈 출처가 제공되지 않아 수정 미완료/BLOCKED다. 과제서의 재개 조건에 따라 동일 조사·gate 재실행·외부 수정으로 대체하지 않았으며 저장소 변경과 커밋은 없다. 요구 스킬 원문·AGENTS.md·docs/RELEASE.md·회차 노트를 읽고 기록 형식 및 작업 트리만 검사했으며 릴리즈 해결 검증은 수행하지 않았다.
- 보류 아이디어:
  - 릴리즈 정책 입력 복구: pending/BLOCKED; 다음 버전·증가 단위·커밋·태그·노트·자산·검증 방식의 출처 필요.
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 최신 실패 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 이번 고정 과제 대체 금지.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 병렬 요청 검증 선행.
- 과제서: 채택 — 새 정책 근거 없이는 반복 조사·구현·gate 실행 없이 BLOCKED로 인계하라는 재개 조건을 적용했다.

검증 및 스킬 반환 기록:
- callable Skill 도구 없음. /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md, /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md, /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md 원문을 실제 읽었다. 파일 읽기는 성공한 Skill 호출이 아니다.
- 재현·원인: 정책 입력 결손 진단을 정찰에서 인계받았다. 이번 독립 재현·원인 입증·다른 원인 배제는 미수행이다. 앱 CI나 gate 결함으로 재분류하지 않는다.
- 수정·인과 검증: 코드 수정과 추가 테스트 없음. Red/Green 및 수정 되돌림 검증 미실행이다.
- 의도적 미검증: 사용자 재개 조건과 반복 실행 금지에 따라 워크플로·실패 스크립트 재조사, npm ci/test/build, gate, sim, 실제 releaser 미실행. 원격 이력과 릴리즈 성공 미확인. 과거 통과를 이번 증거로 사용하지 않는다.
- 이번 기록 검사: Python JSON 필수 필드·허용 값·기존 23개 아이디어 보존·원장 단일 항목·구현 노트 8줄 이내 검사 통과. git diff --check 통과, git status --short 빈 출력. 이는 기록 검증이며 릴리즈 수용 기준 검증이 아니다.
- 저장소·버전·태그·정본·실패 JSON·외부 러너·게이트 무변경, 커밋·배포·원격 전송 없음. 새 아이디어 선정은 정찰 과제서로 갈음하고 기존 23개를 보존했다.

## 2026-09-21
- 선택: [수정 과제] 릴리즈 버전 결정 실패의 정책 근거 결손 해소 (가치 5 / 위험 2 / 작업량 S)
- 결과: 정찰 완료, 수정 미완료/BLOCKED — 실행 가능한 앱 수정 배정 없음.
- 근거: 로컬 37커밋·태그 0개·현재 버전 세 곳 0.0.0, 외부 릴리즈 절차는 기존 관례 요구. 새 승인 정책/실제 관례 출처 없음.
- 검증: 로컬 Git/JSON·기록 형식·작업 트리 검사. 실제 릴리즈·gate·앱 test/build 미실행, 수정 후 동일 검증 통과 미확인.
- 재개: 새 정책 출처가 있을 때만 재계획. 동일 조사·문서 재정리·판정 완화·임의 버전·외부 수정 대체 금지.

## 2026-09-21
- 선택: [수정 과제] 릴리즈 버전 결정 실패 — 실행 가능한 수정 배정 성립 여부 판정 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 승인 정책이나 실제 릴리즈 출처가 제공되지 않아 수정 미완료/BLOCKED다. 과제서의 구현 진입 조건에 따라 앱 수정·동일 조사·gate 재실행 없이 인계하며 저장소 변경과 커밋은 없다. 요청 스킬 원문·AGENTS.md·docs/RELEASE.md·회차 기록을 읽고 결과 기록 형식과 작업 트리만 검사했으며, 동일 릴리즈 검증 통과는 미확인이다.
- 보류 아이디어:
  - 릴리즈 정책 입력 복구: pending; 증가 방식·커밋·태그·노트·자산·검증 방법을 뒷받침할 실제 출처 필요.
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 이번 실패 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 이번 고정 과제의 대체 대상 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 병렬 요청 검증 선행.
- 과제서: 채택 — 실행 가능한 앱 수정 배정이 없다는 판정을 수용하며 반복 무변경 조사를 새로운 구현 성과로 세지 않는다.

검증 및 스킬 반환 기록:
- 호출 가능한 Skill 도구가 없어 아래 원문을 실제 파일로 읽었다. 성공한 Skill 호출로 간주하지 않는다.
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md
- 재현·입증된 원인: 정책 입력 결손 진단을 정찰에서 인계받았다. 이번 독립 재현·원인 입증·다른 원인 배제는 미수행이며 앱 CI 또는 gate 결함으로 재분류하지 않는다.
- 수정·인과 검증: 코드 수정·추가 테스트 없음. 수용 기준 1 미충족으로 2·3 미착수이며 Red/Green 및 수정 되돌림 검증 미실행이다.
- 의도적 미검증: 사용자 과제서의 반복 실행 금지에 따라 워크플로·실패 스크립트 재조사, npm ci/test/build, gate, sim, 실제 releaser 미실행. 원격 릴리즈 이력도 미확인이다.
- 이번 기록 검사: Python JSON 필수 필드·허용 값·기존 27개 아이디어 보존·원장 단일 항목·구현 노트 8줄 이내 검사. git diff --check 및 git status --short로 저장소 상태 확인. 이는 릴리즈 수용 기준 검증이 아니다.
- 버전·태그·정본·실패 JSON·외부 러너·게이트 무변경. 커밋·배포·원격 전송 없음. 아이디어 선정은 정찰 과제서로 갈음하고 정찰에서 추가한 2개를 포함한 27개를 보존했다.

## 2026-09-21
- 선택: [수정 과제] 릴리즈 버전 결정 입력 결손 해소 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 승인 정책이나 실제 릴리즈 출처가 제공되지 않아 수정 미완료/BLOCKED다. 과제서의 재개 조건에 따라 반복 조사·외부 수정·무관 앱 개선으로 대체하지 않았으며 저장소 변경과 커밋은 없다. 요청 스킬 원문과 정본·인계 기록을 읽고 결과 파일 형식과 작업 트리만 검사했으며 동일 릴리즈 검증 통과는 미확인이다.
- 보류 아이디어:
  - 릴리즈 정책 입력 복구: pending; 증가 규칙·커밋·태그·노트·자산·실제 검증 명령의 출처 필요.
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 이번 실패 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 이번 고정 과제 대체 대상 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 병렬 요청 검증 필요.
- 과제서: 채택 — 신규 근거가 있을 때만 재개한다는 조건을 적용하며 반복 무변경 처리를 구현 성과로 세지 않는다.

검증 및 스킬 반환 기록:
- callable Skill 도구 없음. 다음 세 원문을 파일로 읽었다(성공한 Skill 호출 아님):
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md
- 재현·원인: 정책 입력 결손 진단은 인계받았으며 이번 독립 재현·원인 입증·다른 원인 배제는 미수행이다. 앱 CI 또는 GitHub 워크플로 2회 실패로 확정하지 않는다.
- 수정·인과 검증: 코드 수정·추가 테스트 없음. Red/Green 및 수정 되돌림 검증 미실행. 수용 기준 1 미충족으로 2·3 미착수다.
- 의도적 미검증: 과제서의 반복 실행 금지에 따라 워크플로·실패 스크립트 재조사, npm ci/test/build, gate, sim, 실제 releaser 미실행. 원격 이력과 릴리즈 성공 미확인이다.
- 기록 검사: Python으로 JSON 필수 필드·허용 값·기존 29개 보존·원장 단일 항목·구현 노트 8줄 이내 검사. git diff --check와 git status --short 검사. 기록 검사는 릴리즈 해결 증거가 아니다.
- 저장소·버전·태그·정본·실패 JSON·외부 러너·게이트 무변경, 커밋·배포·원격 전송 없음. 아이디어 선정은 고정 과제서로 갈음하며 기존 29개를 보존했다.

## 2026-09-21
- 선택: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 승인 정책이나 실제 릴리즈 출처가 제공되지 않아 수정 미완료이며 기존 pending을 유지한다. 과제서의 재개 조건에 따라 반복 조사·gate·sim·앱 테스트를 실행하지 않았고 저장소 수정과 커밋도 없다. 요청 스킬 원문·정본·회차 기록을 읽고 기록 형식과 작업 트리를 검사하며, 동일 릴리즈 검증 통과는 미확인이다.
- 보류 아이디어:
  - 릴리즈 정책 입력 복구: pending; 실제 정책·관례 출처 필요.
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 이번 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 고정 과제 대체 대상 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 요청 검증 필요.
- 과제서: 채택 — 실행 가능한 수정 배정이 없다는 판정과 신규 입력 이후 재개 조건을 따른다.

검증 및 스킬 반환 기록:
- 수정 과제: 미해결 상태 보존. 수용 기준 1 미충족, 2·3 미착수.
- callable Skill 도구 없음. 아래 원문을 실제 파일로 읽었으며 성공한 Skill 호출로 간주하지 않는다.
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md
- 재현·원인: 정책 입력 결손 진단을 인계받았다. 이번 독립 재현·원인 입증·다른 원인 배제는 미수행이며 앱 CI 또는 GitHub workflow 실패로 확정하지 않는다.
- 수정·인과 검증: 코드 수정·추가 테스트 없음. Red/Green·수정 되돌림 검증 미실행.
- 의도적 미검증: 사용자 과제서의 반복 실행 금지에 따라 워크플로·실패 스크립트 재조사, npm ci/test/build, gate, sim, 실제 releaser 미실행. 원격 이력도 미확인이다.
- 기록 검증 명령: python3 /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-125404-aiportal-front-improve/verify-records.py 및 Python 단일 원장·구현 노트 길이·31개 아이디어 보존 검사; 저장소 검사는 git diff --check, git status --short. 실행 결과는 이번 구현 노트에 남긴다. 릴리즈 수용 기준 검증이 아니다.
- 아이디어 선정은 정찰 과제서로 갈음하고 기존 29개와 정찰 신규 2개를 모두 보존했다. 버전·태그·정본·실패 JSON·외부 러너·gate 변경 및 커밋·배포·원격 전송 없음.

## 2026-09-21
- 선택: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 승인 정책이나 실제 릴리즈 출처가 전달되지 않아 수용 기준 1 미충족, 2·3 미착수이며 수정 및 동일 검증 통과는 미완료다. 과제서의 재개 계약에 따라 반복 조사·gate·앱 검증을 실행하지 않았고 저장소 수정과 커밋도 없다. 요청 스킬 원문과 정본·인계 자료를 읽고 회차 기록 형식과 작업 트리만 검사했으며 기존 pending을 유지한다.
- 보류 아이디어:
  - 릴리즈 정책 입력 복구: pending; 버전·커밋·태그·노트·자산·검증 명령을 결정할 출처 필요.
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 이번 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 고정 과제 대체 대상 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 요청 검증 필요.
- 과제서: 채택 — 신규 근거가 있을 때만 재개하며 반복 무변경 처리를 구현 성과로 세지 않는다.

검증 및 스킬 반환 기록:
- 호출 가능한 Skill 도구 없음. 다음 원문을 실제 파일로 읽었으며 성공한 Skill 호출로 간주하지 않는다.
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md
- 재현·원인: 정책 입력 결손 진단은 정찰에서 인계받았다. 이번 독립 재현·원인 입증·다른 원인 배제는 미수행이며 앱 CI나 GitHub Actions 실패로 확정하지 않는다.
- 수정·인과 검증: 코드 수정·추가 테스트 없음. Red/Green 및 수정 되돌림 검증 미실행이다.
- 의도적 미검증: 사용자 과제서의 반복 실행 금지에 따라 워크플로·실패 스크립트 재조사, npm ci/test/build, gate, sim, 실제 releaser 미실행. 원격 릴리즈 이력도 미확인이다.
- 기록 검사: python3 인라인 검사로 JSON 필수 필드·허용 값·기존 33개 제목/상태 보존·단일 원장 항목·구현 노트 8줄 이내를 확인한다. git diff --check 및 git status --short 결과는 구현 노트에 기록한다. 릴리즈 수용 기준 검증이 아니다.
- 아이디어 선정은 정찰 과제서로 갈음하고 정찰 신규 2개를 포함한 기존 33개를 보존했다. 버전·태그·정본·실패 JSON·외부 러너·gate 변경 및 커밋·배포·원격 전송 없음.

## 2026-09-21
- 선택: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 승인 정책이나 실제 릴리즈 출처가 인계되지 않아 수용 기준 1 미충족, 2·3 미착수이며 수정 및 동일 검증 통과는 미완료다. 과제서에 따라 반복 조사·gate·앱 검사를 실행하지 않았으며 저장소 수정과 커밋 없이 기존 pending을 유지한다. 요청 스킬 원문·정본·회차 인계를 읽고 기록 형식과 작업 트리만 검사했다.
- 보류 아이디어:
  - 릴리즈 정책 입력 복구: pending; 버전·커밋·태그·노트·자산·검증 절차의 승인 출처 필요.
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 이번 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 고정 과제 대체 대상 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 요청 검증 필요.
- 과제서: 채택 — 신규 출처 이후 재개 조건을 따르며 반복 무변경 처리를 구현 성과로 세지 않는다.

검증 및 스킬 반환 기록:
- 호출 가능한 Skill 도구 없음. 아래 세 원문을 파일로 읽었으며 성공한 Skill 호출로 간주하지 않는다.
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md
- 재현·원인: 정책 입력 결손 진단은 정찰에서 인계받았다. 이번 독립 재현·원인 입증·다른 원인 배제는 미수행이며 앱 CI나 GitHub Actions 실패로 확정하지 않는다.
- 수정·인과 검증: 코드 수정·추가 테스트 없음. Red/Green 및 수정 되돌림 검증 미실행.
- 의도적 미검증: 사용자 과제서의 반복 실행 금지에 따라 실패 경로 재조사, npm ci/test/build, gate, sim, 실제 releaser 미실행. 원격 릴리즈 이력도 미확인이다.
- 기록 검사: python3 인라인 검사로 JSON 필수 필드·허용 값·기존 35개 제목/상태 보존·단일 원장 항목·구현 노트 8줄 이내 확인. git diff --check exit 0, git status --short 출력 없음. 릴리즈 수용 기준 검증이 아니다.
- 아이디어 선정은 정찰 과제서로 갈음하고 정찰 신규 2개를 포함한 35개를 보존했다. 버전·태그·정본·실패 JSON·외부 러너·gate 변경 및 커밋·배포·원격 전송 없음.

## 2026-09-21
- 선택: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 승인 정책이나 실제 릴리즈 출처가 인계되지 않아 수용 기준 1 미충족, 2·3 미착수이며 수정 및 동일 검증 통과는 미완료다. 과제서에 따라 반복 조사·gate·앱 검사를 실행하지 않았으며 저장소 수정과 커밋 없이 기존 pending을 유지한다. 요청 스킬 원문·정본·회차 인계를 읽고 기록 형식과 작업 트리만 검사했다.
- 보류 아이디어:
  - 릴리즈 정책 입력 복구: pending; 버전·커밋·태그·노트·자산·검증 절차의 승인 출처 필요.
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 이번 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 고정 과제 대체 대상 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 요청 검증 필요.
- 과제서: 채택 — 신규 출처 이후 재개 조건을 따르며 반복 무변경 처리를 구현 성과로 세지 않는다.

검증 및 스킬 반환 기록:
- 호출 가능한 Skill 도구 없음. 아래 세 원문을 파일로 읽었으며 성공한 Skill 호출로 간주하지 않는다.
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md
- 재현·원인: 정책 입력 결손 진단은 정찰에서 인계받았다. 이번 독립 재현·원인 입증·다른 원인 배제는 미수행이며 앱 CI나 GitHub Actions 실패로 확정하지 않는다.
- 수정·인과 검증: 코드 수정·추가 테스트 없음. Red/Green 및 수정 되돌림 검증 미실행.
- 의도적 미검증: 사용자 과제서의 반복 실행 금지에 따라 실패 경로 재조사, npm ci/test/build, gate, sim, 실제 releaser 미실행. 원격 릴리즈 이력도 미확인이다.
- 기록 검사: python3 인라인 검사로 JSON 필수 필드·허용 값·기존 37개 제목/상태 보존·단일 원장 항목·구현 노트 8줄 이내 확인. git diff --check exit 0, git status --short 출력 없음. 릴리즈 수용 기준 검증이 아니다.
- 아이디어 선정은 정찰 과제서로 갈음하고 정찰 신규 2개를 포함한 37개를 보존했다. 버전·태그·정본·실패 JSON·외부 러너·gate 변경 및 커밋·배포·원격 전송 없음.

## 2026-09-21
- 선택: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 승인 정책이나 실제 릴리즈 출처가 인계되지 않아 수용 기준 1 미충족, 2·3 미착수이며 수정 및 동일 검증 통과는 미완료다. 과제서에 따라 반복 조사·gate·앱 검사를 실행하지 않았으며 저장소 수정과 커밋 없이 기존 pending을 유지한다. 요청 스킬 원문·정본·회차 인계를 읽고 기록 형식과 작업 트리만 검사했다.
- 보류 아이디어:
  - 릴리즈 정책 입력 복구: pending; 버전·커밋·태그·노트·자산·검증 절차의 승인 출처 필요.
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 이번 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 고정 과제 대체 대상 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 요청 검증 필요.
- 과제서: 채택 — 신규 출처 이후 재개 조건을 따르며 반복 무변경 처리를 구현 성과로 세지 않는다.

검증 및 스킬 반환 기록:
- 호출 가능한 Skill 도구 없음. 아래 세 원문을 파일로 읽었으며 성공한 Skill 호출로 간주하지 않는다.
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md
- 재현·원인: 정책 입력 결손 진단은 정찰에서 인계받았다. 이번 독립 재현·원인 입증·다른 원인 배제는 미수행이며 앱 CI나 GitHub Actions 실패로 확정하지 않는다.
- 수정·인과 검증: 코드 수정·추가 테스트 없음. Red/Green 및 수정 되돌림 검증 미실행.
- 의도적 미검증: 사용자 과제서의 반복 실행 금지에 따라 실패 경로 재조사, Git/JSON 릴리즈 이력 재조사, npm ci/test/build, gate, sim, 실제 releaser 미실행. 원격 릴리즈 이력도 미확인이다.
- 기록 검사: python3 인라인 검사로 JSON 필수 필드·허용 값·기존 39개 제목/상태 보존·단일 원장 항목·구현 노트 8줄 이내 확인. git diff --check exit 0, git status --short 출력 없음. 릴리즈 수용 기준 검증이 아니다.
- 아이디어 선정은 정찰 과제서로 갈음하고 정찰 신규 2개를 포함한 39개를 보존했다. 버전·태그·정본·실패 JSON·외부 러너·gate 변경 및 커밋·배포·원격 전송 없음.

## 2026-09-21
- 선택: [수정 과제] 릴리즈 근거 입력 복구 — 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 승인 정책이나 실제 릴리즈 출처가 인계되지 않아 수용 기준 1 미충족, 2·3 미착수이며 수정 및 동일 검증 통과는 미완료다. 과제서에 따라 반복 조사·gate·앱 검사를 실행하지 않았으며 저장소 수정과 커밋 없이 기존 pending을 유지한다. 요청 스킬 원문·정본·회차 인계를 읽고 기록 형식과 작업 트리만 검사했다.
- 보류 아이디어:
  - 릴리즈 정책 입력 복구: pending; 증가·커밋·태그·노트·자산·검증 명령의 유효한 출처 필요.
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 이번 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 고정 과제 대체 대상 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 요청 검증 필요.
- 과제서: 채택 — 신규 출처 이후 재개 조건을 따르며 반복 무변경 처리를 구현 성과로 세지 않는다.

검증 및 스킬 반환 기록:
- 호출 가능한 Skill 도구 없음. 아래 세 원문을 직접 읽었으며 성공한 Skill 호출로 간주하지 않는다.
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md
- 재현·원인: 정책 입력 결손 진단은 정찰에서 인계받았다. 이번 독립 재현·원인 입증·다른 원인 배제는 미수행이며 앱 CI나 GitHub Actions 실패로 확정하지 않는다. 원격 조회 인증 실패는 이력 부재를 증명하지 않는다.
- 수정·인과 검증: 코드 수정·추가 테스트 없음. Red/Green 및 수정 되돌림 검증 미실행.
- 의도적 미검증: 사용자 과제서에 따라 실패 경로 및 Git/JSON 릴리즈 이력 재조사, 원격 조회, npm ci/test/build, gate, sim, 실제 releaser 미실행. 릴리즈 수용 기준 검증은 미완료다.
- 아이디어 선정은 정찰 과제서로 갈음하고 정찰 신규 2개를 포함한 기존 41개 제목과 상태를 보존했다. 버전·태그·정본·실패 JSON·외부 러너·gate 변경 및 커밋·배포·원격 전송 없음.
- 기록 검사: python3 인라인 검사 통과(기존 41개 제목/상태 보존, JSON 필드·허용 값, 단일 원장 항목, 구현 노트 8줄 이내). git diff --check 및 git status --short는 exit 0/출력 없음. 이 결과는 릴리즈 수용 기준 통과가 아니다.

## 2026-09-21
- 선택: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 승인 정책이나 실제 릴리즈 출처가 인계되지 않아 수용 기준 1 미충족, 2·3 미착수이며 수정 및 동일 검증 통과는 미완료다. 과제서에 따라 반복 조사·gate·앱 검사를 실행하지 않았으며 저장소 수정과 커밋 없이 기존 pending을 유지한다. 요청 스킬 원문·정본·회차 인계를 읽고 기록 형식과 작업 트리만 검사했다.
- 보류 아이디어:
  - 릴리즈 정책 입력 복구: pending; 출처·적용 대상·증가 단위·커밋/태그·노트·자산·검증 절차 근거 필요.
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 이번 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 고정 과제 대체 대상 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 요청 검증 필요.
- 과제서: 채택 — 신규 출처 이후 재개 조건을 따르며 반복 무변경 처리를 구현 성과로 세지 않는다.

검증 및 스킬 반환 기록:
- 호출 가능한 Skill 도구 없음. 아래 세 원문을 직접 읽었으며 성공한 Skill 호출로 간주하지 않는다.
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md
- 재현·원인: 정책 입력 결손 진단은 정찰에서 인계받았다. 이번 독립 재현·원인 입증·다른 원인 배제는 미수행이며 앱 CI나 GitHub Actions 실패로 확정하지 않는다. 원격 인증 실패나 빈 목록은 원격 이력 부재를 증명하지 않는다.
- 수정·인과 검증: 코드 수정·추가 테스트 없음. Red/Green 및 수정 되돌림 검증 미실행.
- 의도적 미검증: 사용자 과제서에 따라 실패 경로 및 Git/JSON 릴리즈 이력 재조사, 원격 조회, npm ci/test/build, gate, sim, 실제 releaser 미실행. 릴리즈 수용 기준 검증은 미완료다.
- 아이디어 선정은 정찰 과제서로 갈음하고 정찰 신규 2개를 포함한 기존 43개 제목과 상태를 보존했다. 버전·태그·정본·실패 JSON·외부 러너·gate 변경 및 커밋·배포·원격 전송 없음.
- 기록 검사: python3 인라인 검사 통과(기존 43개 제목/상태 보존, JSON 필드·허용 값, 단일 원장 항목, 구현 노트 8줄 이내). git diff --check 및 git status --short는 출력 없이 exit 0. 릴리즈 수용 기준 통과가 아니다.

## 2026-09-21
- 선택: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 승인 정책이나 실제 릴리즈 출처가 인계되지 않아 수용 기준 1 미충족, 2·3 미착수이며 수정 및 동일 검증 통과는 미완료다. 과제서에 따라 반복 조사·gate·앱 검사를 실행하지 않았으며 저장소 수정과 커밋 없이 기존 pending을 유지한다. 요청 스킬 원문·정본·회차 인계를 읽고 기록 형식을 검사했다.
- 보류 아이디어:
  - 릴리즈 정책 입력 복구: pending; 적용 프로젝트·기준 SHA·출처·증가 단위·커밋/태그·노트·자산·검증 절차 근거 필요.
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 이번 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 고정 과제 대체 대상 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 요청 검증 필요.
- 과제서: 채택 — 신규 출처 이후 재개 조건을 따르며 반복 무변경 처리를 구현 성과로 세지 않는다.

검증 및 스킬 반환 기록:
- 호출 가능한 Skill 도구 없음. 아래 세 원문을 직접 읽었으며 성공한 Skill 호출로 간주하지 않는다.
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md
- 재현·원인: 정책 입력 결손 진단은 정찰에서 인계받았다. 이번 독립 재현·원인 입증·다른 원인 배제는 미수행이며 앱 CI나 GitHub Actions 실패로 확정하지 않는다. 원격 인증 실패나 빈 목록은 원격 이력 부재를 증명하지 않는다.
- 수정 전 결과(정찰 인계): 기존 실패 JSON 두 건 gate exit 1/failed/ok=false, 외부 Release 회귀 5개 통과(ResourceWarning). 이번 실행 결과가 아니며 수정 후 통과 증거도 아니다.
- 수정·인과 검증: 코드 수정·추가 테스트 없음. Red/Green 및 수정 되돌림 검증 미실행. 수정 후 동일 releaser/gate 통과는 미완료다.
- 의도적 미검증: 사용자 과제서에 따라 실패 경로 및 Git/JSON 릴리즈 이력 재조사, 원격 조회, npm ci/test/build, gate, sim, 실제 releaser 미실행.
- 아이디어 선정은 정찰 과제서로 갈음하고 정찰 신규 2개를 포함한 기존 45개 제목과 상태를 보존했다. 버전·태그·정본·실패 JSON·외부 러너·gate 변경 및 커밋·배포·원격 전송 없음.
- 기록 검사: python3 인라인 검사 통과(기존 45개 제목/상태 보존, JSON 필드·허용 값, 단일 원장 항목, 구현 노트 8줄 이내). git diff --check 및 git status --short는 출력 없이 exit 0. 릴리즈 수용 기준 통과가 아니다.

## 2026-09-21
- 선택: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 승인 정책이나 실제 릴리즈 출처가 인계되지 않아 수용 기준 1 미충족, 2·3 미착수이며 수정 및 동일 검증 통과는 미완료다. 과제서에 따라 반복 조사·gate·앱 검사를 실행하지 않았으며 저장소 수정과 커밋 없이 기존 pending을 유지한다. 요청 스킬 원문·정본·회차 인계를 읽고 기록 형식과 작업 트리만 검사했다.
- 보류 아이디어:
  - 릴리즈 정책 입력 복구: pending; 적용 프로젝트·기준 SHA·출처·증가 단위·커밋/태그·노트·자산·검증 절차 근거 필요.
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 이번 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 고정 과제 대체 대상 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 요청 검증 필요.
- 과제서: 채택 — 신규 출처 이후 재개 조건을 따르며 반복 무변경 처리를 구현 성과로 세지 않는다.

검증 및 스킬 반환 기록:
- 호출 가능한 Skill 도구 없음. 아래 세 원문을 직접 읽었으며 성공한 Skill 호출로 간주하지 않는다.
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md
- 재현·원인: 정책 입력 결손 진단은 정찰에서 인계받았다. 이번 독립 재현·원인 입증·다른 원인 배제는 미수행이며 앱 CI나 GitHub Actions 실패로 확정하지 않는다. 원격 인증 실패나 빈 목록은 원격 이력 부재를 증명하지 않는다.
- 수정 전 결과(정찰 인계): 기존 실패 JSON 두 건 gate exit 1/failed/ok=false, 외부 Release 회귀 5개 통과(ResourceWarning). 이번 실행 결과가 아니며 수정 후 통과 증거도 아니다.
- 수정·인과 검증: 코드 수정·추가 테스트 없음. Red/Green 및 수정 되돌림 검증 미실행. 수정 후 동일 releaser/gate 통과는 미완료다.
- 의도적 미검증: 사용자 과제서에 따라 실패 경로 및 Git/JSON 릴리즈 이력 재조사, 원격 조회, npm ci/test/build, gate, sim, 실제 releaser 미실행.
- 아이디어 선정은 정찰 과제서로 갈음하고 정찰 신규 2개를 포함한 기존 47개 제목과 상태를 보존했다. 버전·태그·정본·실패 JSON·외부 러너·gate 변경 및 커밋·배포·원격 전송 없음.
- 기록 검사: python3 인라인 검사 통과(기존 47개 제목/상태 보존, JSON 필드·허용 값, 단일 원장 항목, 구현 노트 8줄 이내). git diff --check 및 git status --short는 출력 없이 exit 0. 릴리즈 수용 기준 통과가 아니다.

## 2026-09-22
- 선택: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 승인 정책이나 실제 릴리즈 출처가 인계되지 않아 수용 기준 1 미충족, 2·3 미착수이며 수정 및 동일 검증 통과는 미완료다. 과제서에 따라 반복 조사·gate·앱 검사를 실행하지 않고 저장소 수정과 커밋 없이 기존 pending을 유지한다. 요청 스킬 원문·정본·회차 인계를 읽고 필수 기록을 작성했으며 기록 형식 검사는 릴리즈 해결 증거가 아니다.
- 보류 아이디어:
  - 릴리즈 정책 입력 복구: pending; 프로젝트·기준 SHA·출처·증가 단위·커밋/태그·노트·자산·검증 절차 근거 필요.
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 이번 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 고정 과제 대체 대상 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 요청 검증 필요.
- 과제서: 채택 — 신규 출처 이후 재개 조건을 따르며 반복 무변경 처리를 구현 성과로 세지 않는다.

검증 및 스킬 반환 기록:
- 호출 가능한 Skill 도구 없음. 다음 원문을 직접 읽었으며 성공한 Skill 호출로 간주하지 않는다.
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md
- 재현·원인: 정책 입력 결손 진단은 정찰 인계다. 이번 독립 재현·원인 입증·다른 원인 배제는 미수행이며 앱 CI나 GitHub Actions 실패로 확정하지 않는다. 원격 인증 실패나 빈 목록은 원격 이력 부재의 증거가 아니다.
- 수정 전 결과(정찰 인계): 기존 실패 JSON 두 건 gate exit 1/failed/ok=false, 외부 Release 회귀 5개 통과(ResourceWarning). 이번 구현 실행 결과나 수정 후 통과 증거가 아니다.
- 수정·인과 검증: 코드 수정·추가 테스트 없음. Red/Green 및 수정 되돌림 검증 미실행. 실제 releaser의 새 결과와 동일 게이트 통과 미완료.
- 의도적 미검증: 사용자 과제서에 따라 실패 경로 및 Git/JSON 릴리즈 이력 재조사, 원격 조회, npm ci/test/build, gate, sim, 실제 releaser 미실행.
- 아이디어 선정은 정찰 과제서로 갈음하고 정찰 신규 2개를 포함한 기존 49개 제목과 상태를 보존했다. 버전·태그·정본·실패 JSON·외부 러너·gate 변경 및 커밋·배포·원격 전송 없음.
- 기록 검사: python3 -B 인라인 검사 PASS(49개 제목/상태 보존, JSON 필드·허용 값, 단일 원장 항목, 구현 노트 8줄 이내). git diff --check 및 git status --short는 출력 없이 exit 0. 릴리즈 수용 기준 통과가 아니다.

## 2026-09-22
- 선택: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 승인 정책이나 실제 릴리즈 출처가 인계되지 않아 수용 기준 1 미충족, 2·3 미착수이며 수정 및 동일 검증 통과는 미완료다. 과제서에 따라 반복 조사·gate·앱 검사를 실행하지 않고 저장소 수정과 커밋 없이 기존 pending을 유지한다. 요청 스킬 원문·정본·회차 인계를 읽고 필수 기록을 작성했으며 기록 형식 검사는 릴리즈 해결 증거가 아니다.
- 보류 아이디어:
  - 릴리즈 정책 입력 복구: pending; 프로젝트·기준 SHA·출처·증가 단위·커밋/태그·노트·자산·검증 절차 근거 필요.
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 이번 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 고정 과제 대체 대상 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 요청 검증 필요.
- 과제서: 채택 — 신규 출처 이후 재개 조건을 따르며 반복 무변경 처리를 구현 성과로 세지 않는다.

검증 및 스킬 반환 기록:
- 호출 가능한 Skill 도구 없음. 다음 원문을 직접 읽었으며 성공한 Skill 호출로 간주하지 않는다.
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md
- 재현·원인: 정책 입력 결손 진단은 정찰 인계다. 이번 독립 재현·원인 입증·다른 원인 배제는 미수행이며 앱 CI나 GitHub Actions 실패로 확정하지 않는다. 원격 인증 실패나 빈 목록은 원격 이력 부재의 증거가 아니다.
- 수정 전 결과(정찰 인계): 기존 실패 JSON 두 건 gate exit 1/failed/ok=false, 외부 Release 회귀 5개 통과(ResourceWarning). 이번 구현 실행 결과나 수정 후 통과 증거가 아니다.
- 수정·인과 검증: 코드 수정·추가 테스트 없음. Red/Green 및 수정 되돌림 검증 미실행. 실제 releaser의 새 결과와 동일 게이트 통과 미완료.
- 의도적 미검증: 사용자 과제서에 따라 실패 경로 및 Git/JSON 릴리즈 이력 재조사, 원격 조회, npm ci/test/build, gate, sim, 실제 releaser 미실행.
- 아이디어 선정은 정찰 과제서로 갈음하고 정찰 신규 2개를 포함한 기존 51개 제목과 상태를 보존했다. 버전·태그·정본·실패 JSON·외부 러너·gate 변경 및 커밋·배포·원격 전송 없음.
- 기록 검사: python3 -B 인라인 검사 PASS(51개 제목/상태 보존, JSON 필드·허용 값, 단일 원장 항목, 구현 노트 8줄 이내). git diff --check 및 git status --short는 출력 없이 exit 0. 릴리즈 수용 기준 통과가 아니다.

## 2026-09-22
- 선택: [수정 과제] 릴리즈 버전 결정 입력 복구 (가치 5 / 위험 2 / 작업량 S)
- 결과: 정찰 완료 / 수정 미완료 / pending
- 요약: 적용 가능한 새 정책 또는 실제 릴리즈 출처를 확보하지 못했다. 동일 BLOCKED 구현 재배정 없이 필수 입력 계약을 인계한다. 회귀 5개 통과 및 기존 실패 두 건의 게이트 거부는 확인했으나 수정 후 동일 검증 통과가 아니다.
- 변경: 지정 회차의 정찰 기록만 작성. 저장소 코드·버전·태그·외부 실행기·게이트 무변경, 커밋 없음.

## 2026-09-22
- 선택: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 승인 정책이나 실제 릴리즈 출처가 인계되지 않아 수용 기준 1 미충족, 2·3 미착수이며 수정 및 동일 검증 통과는 미완료다. 과제서에 따라 반복 조사·gate·앱 검사를 실행하지 않고 저장소 수정과 커밋 없이 기존 pending을 유지한다. 요청 스킬 원문·정본·회차 인계를 읽고 필수 기록을 작성했으며 기록 형식 검사는 릴리즈 해결 증거가 아니다.
- 보류 아이디어:
  - 릴리즈 정책 입력 복구: pending; 프로젝트·기준 SHA·출처·증가 단위·커밋/태그·노트·자산·검증 절차 근거 필요.
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 이번 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 고정 과제 대체 대상 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 요청 검증 필요.
- 과제서: 채택 — 신규 출처 이후 재개 조건을 따르며 반복 무변경 처리를 구현 성과로 세지 않는다.

검증 및 스킬 반환 기록:
- 호출 가능한 Skill 도구 없음. 다음 원문을 직접 읽었으며 성공한 Skill 호출로 간주하지 않는다.
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md
- 재현·원인: 정책 입력 결손 진단은 정찰 인계다. 이번 독립 재현·원인 입증·다른 원인 배제는 미수행이며 앱 CI나 GitHub Actions 실패로 확정하지 않는다. 원격 인증 실패나 빈 목록은 원격 이력 부재의 증거가 아니다.
- 수정 전 결과(정찰 인계): 기존 실패 JSON 두 건 gate exit 1/failed/ok=false, 외부 Release 회귀 5개 통과. 이번 구현 실행 결과나 수정 후 통과 증거가 아니다.
- 수정·인과 검증: 코드 수정·추가 테스트 없음. Red/Green 및 수정 되돌림 검증 미실행. 실제 releaser의 새 결과와 동일 게이트 통과 미완료.
- 의도적 미검증: 사용자 과제서에 따라 실패 경로 및 Git/JSON 릴리즈 이력 재조사, 원격 조회, npm ci/test/build, gate, sim, 실제 releaser 미실행.
- 아이디어 선정은 정찰 과제서로 갈음하고 정찰 신규 2개를 포함한 기존 55개 제목과 상태를 보존했다. 버전·태그·정본·실패 JSON·외부 러너·gate 변경 및 커밋·배포·원격 전송 없음.

## 2026-09-22
- 선택: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 승인 정책이나 실제 릴리즈 출처가 없어 재개 조건을 충족하지 못했으며, 수정 및 동일 검증 통과는 미완료다. docs/RELEASE.md, 실제 Git/JSON, GitLab CI, 외부 release-prompt.md·run.sh·gate.py·Release 테스트 및 최신 과제서를 확인했지만 새 관례를 정당화할 근거를 확보하지 못했다. 저장소 변경·커밋 없이 필수 기록만 작성했으며 기록 형식 검사는 릴리즈 해결 증거가 아니다.
- 보류 아이디어:
  - 릴리즈 정책 입력 복구: pending; 프로젝트/SHA·출처·증가 단위·커밋/태그·노트·자산·검증 명령 필요.
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이고 최신 실패의 해결책이 아님.
  - useAppList 비배열 캐시 방어: pending; 고정 과제 대체 대상 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝과 런타임 검증 필요.
- 과제서: 채택 — 새 출처 확보 후 재개 조건을 유지하며 무변경 처리를 해결 성과로 세지 않는다.

검증 및 스킬 반환 기록:
- 호출 가능한 Skill 도구 없음. 아래 세 원문을 직접 읽어 적용했으며 성공한 Skill 호출로 간주하지 않는다.
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md
- 재현·원인: 이번 독립 실행 재현 및 인과 입증은 미수행. 읽은 실행 경로는 release_project → run_agent → release.json → cmd_release/evaluate_release이며 failed 결과를 게시 전에 차단한다. 정책 결손은 인계 진단이고 GitHub Actions/앱 CI 실패로 확정하지 않는다.
- 수정·검증: 코드 수정·추가 테스트 없음. Red/Green·수정 되돌림·실제 releaser 및 동일 게이트 통과 미실행. 새 출처 없는 반복 BLOCKED 실행을 피하라는 과제서에 따라 기존 실패 게이트·앱 test/build·원격 조회를 반복하지 않았다.
- 현재 확인: git log -5의 HEAD e938e8e, git tag 출력 없음, package.json 및 lock 두 버전 필드는 모두 0.0.0. 원격 이력은 미확인이다.
- 아이디어 선정은 고정 과제서로 갈음하여 기존 55개 제목/상태 보존. 무관한 새 아이디어를 추가하거나 대신 구현하지 않았다.

## 2026-09-22
- 선택: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 적용 정책이나 실제 릴리즈 출처가 인계되지 않아 수용 기준 1 미충족, 2·3 미착수이며 수정 및 동일 검증 통과는 미완료다. 요청 스킬 원문·정본·회차 brief/journal을 읽고 사용자 과제서에 따라 반복 조사·gate·앱 검사 없이 기존 pending을 유지했다. 저장소 수정과 커밋 없이 지정 회차 기록만 갱신했으며 기록 형식 검사는 릴리즈 해결 증거가 아니다.
- 보류 아이디어:
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 이번 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 고정 과제 대체 대상 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 요청 검증 필요.
  - README 빌드 명령 정합성: pending; 고정 과제 대체 대상 아님.
- 과제서: 채택 — 신규 출처 이후 재개 조건을 유지하며 무변경 처리를 해결 성과로 세지 않는다.

검증 및 스킬 반환 기록:
- 호출 가능한 Skill 도구 없음. 아래 원문을 직접 읽었으며 성공한 Skill 호출로 간주하지 않는다.
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md
- 재현·원인: 정책 입력 결손은 정찰 인계 진단이다. 이번 독립 실행 재현·원인 입증·다른 원인 배제는 미수행이며 앱 CI나 동일 GitHub step 두 번 실패로 확정하지 않는다.
- 수정 전 결과(정찰 인계): 기존 실패 JSON 두 건 gate exit 1/failed/ok=false, Release 회귀 5개 OK(ResourceWarning). 이번 구현 실행 결과나 수정 후 통과 증거가 아니다.
- 수정·인과 검증: 코드 수정·추가 테스트 없음. Red/Green·수정 되돌림·실제 releaser 및 동일 게이트 통과 미실행.
- 의도적 미검증: 신규 출처 없는 반복 실행 금지에 따라 Git/JSON 이력 재조사·원격 조회·npm ci/test/build·gate·sim·releaser를 실행하지 않았다. 재개에는 프로젝트/기준 SHA·원문 출처·증가 단위·커밋/태그·노트·자산·정확한 검사 명령이 필요하다.
- 아이디어 선정은 정찰 과제서로 갈음했다. 정찰 신규 2개를 포함한 기존 57개 제목/상태를 보존하며 무관한 후보를 추가하거나 대신 구현하지 않았다.

## 2026-09-22
- 선택: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 적용 정책이나 실제 릴리즈 출처가 인계되지 않아 수용 기준 1 미충족, 2·3 미착수이며 수정 및 동일 검증 통과는 미완료다. 요청 스킬 원문·docs/RELEASE.md·회차 brief/journal을 읽고 과제서에 따라 반복 조사·gate·앱 검사 없이 기존 pending을 유지했다. 저장소 수정·커밋 없이 지정 회차 기록만 갱신했으며 기록 검사는 릴리즈 해결 증거가 아니다.
- 보류 아이디어:
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 이번 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 고정 과제 대체 대상 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 요청 검증 필요.
  - README 빌드 명령 정합성: pending; 고정 과제 대체 대상 아님.
- 과제서: 채택 — 신규 출처 이후 재개 조건을 유지하며 무변경 처리를 해결 성과로 세지 않는다.

검증 및 스킬 반환 기록:
- 호출 가능한 Skill 도구 없음. 아래 원문을 직접 읽었으며 성공한 Skill 호출로 간주하지 않는다.
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md
- 재현·원인: 정책 입력 결손은 정찰 인계 진단이다. 이번 독립 실행 재현·원인 입증·다른 원인 배제는 미수행이며 앱 CI나 동일 GitHub step 두 번 실패로 확정하지 않는다.
- 수정 전 결과(정찰 인계): 기존 실패 JSON 두 건 gate exit 1/failed/ok=false, Release 회귀 5개 OK(ResourceWarning). 이번 구현 실행 결과나 수정 후 통과 증거가 아니다.
- 수정·인과 검증: 코드 수정·추가 테스트 없음. Red/Green·수정 되돌림·실제 releaser 및 동일 게이트 통과 미실행.
- 의도적 미검증: 신규 출처 없는 반복 실행 금지에 따라 Git/JSON 이력 재조사·원격 조회·npm ci/test/build·gate·sim·releaser를 실행하지 않았다. 재개에는 프로젝트/기준 SHA·원문 출처·증가 단위·커밋/태그·노트·자산·정확한 검사 명령이 필요하다.
- 아이디어 선정은 정찰 과제서로 갈음했다. 정찰 신규 2개를 포함한 기존 59개 제목/상태를 보존하며 무관한 후보를 추가하거나 대신 구현하지 않았다.

## 2026-09-22
- 유형: 수정 과제
- 선택: 릴리즈 버전 결정 입력 복구 (가치 5 / 위험 2 / 작업량 S)
- 상태: pending / 수정 미완료 / 신규 실행 배정 불가
- 근거: HEAD e938e8e, 현재 버전 세 필드 0.0.0, 로컬 태그 0개, 신규 적용 정책·실제 릴리즈 출처 없음.
- 검증: 기존 실패 JSON 두 건을 현재 gate CLI로 재현하여 각각 exit 1/failed/ok=false 확인. 수정 후 동일 검증 통과 아님.
- 변경: 정찰 brief/profile/ideas/journal/원장만 작성; 저장소 코드·커밋·태그·외부 전송 없음.
- 재개: 출처와 적용 프로젝트/SHA, 증가 단위·커밋/태그·노트·자산·검증 명령을 확보한 뒤에만 구현. 반복 무변경은 해결 성과가 아니다.

## 2026-09-22
- 선택: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 적용 정책이나 실제 릴리즈 출처가 인계되지 않아 수용 기준 1 미충족, 2·3 미착수이며 수정 및 동일 검증 통과는 미완료다. 요청 스킬 원문·docs/RELEASE.md·회차 brief/journal을 읽고 과제서에 따라 반복 조사·gate·앱 검사 없이 기존 pending을 유지했다. 저장소 수정·커밋 없이 지정 회차 기록만 갱신하며 이를 새 해결 성과로 세지 않는다.
- 보류 아이디어:
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 이번 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 고정 과제 대체 대상 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 요청 검증 필요.
  - README 빌드 명령 정합성: pending; 고정 과제 대체 대상 아님.
- 과제서: 채택 — 신규 출처 이후 재개 조건을 유지하며 무변경 처리를 해결 성과로 세지 않는다.

검증 및 스킬 반환 기록:
- 호출 가능한 Skill 도구 없음. 다음 원문을 직접 읽었으며 성공한 Skill 호출로 간주하지 않는다.
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md
- 재현·원인: 정책 입력 결손은 정찰 인계 진단이다. 이번 독립 실행 재현·인과 입증·다른 원인 배제는 미수행이며 앱 CI나 동일 GitHub step 두 번 실패로 확정하지 않는다.
- 수정 전 결과(정찰 인계): 기존 실패 JSON 두 건 gate exit 1/failed/ok=false, Release 회귀 5개 OK(ResourceWarning). 이번 구현 실행 결과나 수정 후 통과 증거가 아니다.
- 수정·인과 검증: 코드 수정·추가 테스트 없음. Red/Green·수정 되돌림·실제 releaser 및 동일 게이트 통과 미실행.
- 의도적 미검증: 신규 출처 없는 반복 실행 금지에 따라 Git/JSON 이력 재조사·원격 조회·npm ci/test/build·gate·sim·releaser를 실행하지 않았다. 재개에는 프로젝트/기준 SHA·원문 출처·증가 단위·커밋/태그·노트·자산·정확한 검사 명령이 필요하다.
- 아이디어 선정은 정찰 과제서로 갈음했다. 기존 63개 제목/상태를 보존하며 무관한 후보를 추가하거나 대신 구현하지 않았다.
- 기록 검증: Python으로 JSON 재읽기·63개 제목/상태 보존·필수 필드·원장 항목 1개·구현 노트 8줄 이내를 검사한다. 이 검사는 릴리즈 해결 증거가 아니다.

## 2026-09-22
- 유형: 수정 과제
- 선택: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 적용 정책·실제 릴리즈 출처가 인계되지 않아 수용 기준 1 미충족, 2·3 미착수이며 수정 및 동일 검증 통과는 미완료다. 요청 스킬 원문·docs/RELEASE.md·회차 brief/journal을 읽고 신규 입력 없는 반복 실행 금지에 따라 pending을 유지했다. 저장소 수정·커밋 없이 지정 회차 기록만 갱신했으며 새 해결 성과로 세지 않는다.
- 보류 아이디어:
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 이번 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 고정 과제 대체 대상 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 요청 검증 필요.
  - README 빌드 명령 정합성: pending; 고정 과제 대체 대상 아님.
- 과제서: 채택 — 재개 조건과 기존 pending을 유지하며 반복 무변경 구현을 신규 과제로 배정하지 않는다.

검증 및 스킬 반환 기록:
- 호출 가능한 Skill 도구 없음. 다음 원문을 직접 읽었으며 성공한 Skill 호출로 간주하지 않는다.
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md
- 재현·원인: 정책 입력 결손은 정찰 인계 진단이다. 이번 독립 실행 재현·인과 입증·다른 원인 배제는 미수행이며 앱 CI나 동일 GitHub step 두 번 실패로 확정하지 않는다.
- 수정 전 결과(인계): 기존 failed JSON 두 건 gate exit 1/failed/ok=false, Release 회귀 5개 OK(ResourceWarning). 이번 실행 결과나 수정 후 성공 증거가 아니다. 정찰의 Git/JSON·실행기·게이트·테스트 원문 조회 또한 정찰 실행 범위다.
- 수정·인과 검증: 코드 수정·추가 테스트 없음. Red/Green·수정 되돌림·실제 releaser 및 동일 게이트 통과 미실행.
- 의도적 미검증: 신규 출처 없는 반복 실행 금지에 따라 Git/JSON 이력 재조사·원격 조회·npm ci/test/build·gate·sim·releaser를 실행하지 않았다. 재개에는 프로젝트/기준 SHA·원문 출처·증가 단위·커밋/태그·노트·자산·정확한 검사 명령이 필요하다.
- 아이디어 선정은 정찰 과제서로 갈음했다. 정찰 신규 2개를 포함한 기존 65개 제목/상태를 보존하며 무관한 후보를 추가하거나 대신 구현하지 않았다.
- 기록 검증: python3 표준 라이브러리로 JSON 재읽기·65개 제목/상태 보존·필수 필드·원장 항목 1개·구현 노트 8줄 이내 검사. 릴리즈 해결 검증과 구분한다.

## 2026-09-22
- 유형: 수정 과제
- 선택: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 적용 정책·실제 릴리즈 출처가 인계되지 않아 수용 기준 1 미충족, 2·3 미착수이며 수정 및 동일 검증 통과는 미완료다. 요청 스킬 원문·docs/RELEASE.md·회차 brief/journal을 읽고 신규 입력 없는 반복 실행 금지에 따라 pending을 유지했다. 저장소 수정·커밋 없이 지정 회차 기록만 갱신하며 새 해결 성과로 세지 않는다.
- 보류 아이디어:
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 이번 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 고정 과제 대체 대상 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 요청 검증 필요.
  - README 빌드 명령 정합성: pending; 고정 과제 대체 대상 아님.
- 과제서: 채택 — 재개 조건과 기존 pending을 유지하며 반복 무변경 구현을 신규 과제로 배정하지 않는다.

검증 및 스킬 반환 기록:
- 호출 가능한 Skill 도구 없음. 다음 원문을 직접 읽었으며 성공한 Skill 호출로 간주하지 않는다.
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md
- 재현·원인: 정책 입력 결손은 정찰 인계 진단이다. 이번 독립 실행 재현·인과 입증·다른 원인 배제는 미수행이며 앱 CI나 동일 GitHub step 두 번 실패로 확정하지 않는다.
- 수정 전 결과(인계): 기존 failed JSON 두 건 gate exit 1/failed/ok=false, Release 회귀 5개 OK(ResourceWarning). 이번 실행 결과나 수정 후 성공 증거가 아니다.
- 수정·인과 검증: 코드 수정·추가 테스트 없음. Red/Green·수정 되돌림·실제 releaser 및 동일 게이트 통과 미실행.
- 의도적 미검증: 신규 출처 없는 반복 실행 금지에 따라 Git/JSON 이력 재조사·원격 조회·npm ci/test/build·gate·sim·releaser를 실행하지 않았다. 재개에는 프로젝트/기준 SHA·원문 출처·증가 단위·커밋/태그·노트·자산·정확한 검사 명령이 필요하다.
- 아이디어 선정은 정찰 과제서로 갈음했다. 정찰 신규 2개를 포함한 기존 67개 제목/상태를 보존하며 무관한 후보를 추가하거나 대신 구현하지 않았다.
- 기록 검증: python3 표준 라이브러리로 JSON 재읽기·67개 제목/상태 보존·필수 필드·원장 항목 1개·구현 노트 8줄 이내를 검사했다. 릴리즈 해결 검증과 구분한다.

## 2026-09-22
- 유형: 수정 과제
- 선택: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 적용 정책·실제 릴리즈 출처가 인계되지 않아 수용 기준 1 미충족, 2·3 미착수이며 수정 및 동일 검증 통과는 미완료다. 요청 스킬 원문·AGENTS.md·docs/RELEASE.md·회차 brief/journal을 읽고 신규 입력 없는 반복 실행 금지에 따라 pending을 유지했다. 저장소 수정·커밋 없이 지정 회차 기록만 갱신하며 새 해결 성과로 세지 않는다.
- 보류 아이디어:
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 이번 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 고정 과제 대체 대상 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 요청 검증 필요.
  - README 빌드 명령 정합성: pending; 고정 과제 대체 대상 아님.
- 과제서: 채택 — 재개 조건과 기존 pending을 유지하며 반복 무변경 구현을 신규 과제로 배정하지 않는다.

검증 및 스킬 반환 기록:
- 호출 가능한 Skill 도구 없음. 다음 원문을 직접 읽었으며 성공한 Skill 호출로 간주하지 않는다.
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md
- 재현·원인: 정책 입력 결손은 정찰 인계 진단이다. 이번 독립 실행 재현·인과 입증·다른 원인 배제는 미수행이며 앱 CI나 동일 GitHub step 두 번 실패로 확정하지 않는다.
- 수정 전 결과(인계): 기존 failed JSON 두 건 gate exit 1/failed/ok=false, Release 회귀 5개 OK(ResourceWarning). 이번 실행 결과나 수정 후 성공 증거가 아니다.
- 수정·인과 검증: 코드 수정·추가 테스트 없음. Red/Green·수정 되돌림·실제 releaser 및 동일 게이트 통과 미실행.
- 의도적 미검증: 신규 출처 없는 반복 실행 금지에 따라 Git/JSON 이력 재조사·실행기/게이트 재조사·원격 조회·npm ci/test/build·gate·sim·releaser를 실행하지 않았다. 재개에는 프로젝트/기준 SHA·원문 출처·증가 단위·커밋/태그·노트·자산·정확한 검사 명령이 필요하다.
- 아이디어 선정은 정찰 과제서로 갈음했다. 정찰 신규 2개를 포함한 기존 69개 제목/상태를 보존하며 무관한 후보를 추가하거나 대신 구현하지 않았다.
- 기록 검증: python3 표준 라이브러리로 JSON 재읽기·69개 제목/상태 보존·필수 필드·원장 항목 1개·구현 노트 8줄 이내를 검사했다. 릴리즈 해결 검증과 구분한다.

## 2026-09-22
- 유형: 수정 과제
- 선택: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음
- 요약: 신규 적용 정책·실제 릴리즈 출처가 인계되지 않아 수용 기준 1 미충족, 2·3 미착수이며 수정 및 동일 검증 통과는 미완료다. 요청 스킬 원문·AGENTS.md·docs/RELEASE.md·회차 brief/journal을 읽고 신규 입력 없는 반복 실행 금지에 따라 pending을 유지했다. 저장소 수정·커밋 없이 지정 회차 기록만 갱신하며 새 해결 성과로 세지 않는다.
- 보류 아이디어:
  - 외부 Codex 폴백 경로 전달: pending; 외부 소유이며 이번 해법으로 재선정하지 않음.
  - useAppList 비배열 캐시 방어: pending; 고정 과제 대체 대상 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사와 실제 요청 검증 필요.
  - README 빌드 명령 정합성: pending; 고정 과제 대체 대상 아님.
- 과제서: 채택 — 재개 조건과 기존 pending을 유지하며 반복 무변경 구현을 신규 과제로 배정하지 않는다.

검증 및 스킬 반환 기록:
- 호출 가능한 Skill 도구 없음. 다음 원문을 직접 읽었으며 성공한 Skill 호출로 간주하지 않는다.
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/completion-verification/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/systematic-debugging/SKILL.md
  - /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/test-driven-development/SKILL.md
- 재현·원인: 정책 입력 결손은 정찰 인계 진단이다. 이번 독립 실행 재현·인과 입증·다른 원인 배제는 미수행이며 앱 CI나 동일 GitHub step 두 번 실패로 확정하지 않는다.
- 수정 전 결과(인계): 기존 failed JSON 두 건 gate exit 1/failed/ok=false, Release 회귀 5개 OK(ResourceWarning). 이번 실행 결과나 수정 후 성공 증거가 아니다.
- 수정·인과 검증: 코드 수정·추가 테스트 없음. Red/Green·수정 되돌림·실제 releaser 및 동일 게이트 통과 미실행.
- 의도적 미검증: 신규 출처 없는 반복 실행 금지에 따라 Git/JSON 이력 재조사·실행기/게이트 재조사·원격 조회·npm ci/test/build·gate·sim·releaser를 실행하지 않았다. 재개에는 프로젝트/기준 SHA·원문 출처·증가 단위·커밋/태그·노트·자산·정확한 검사 명령이 필요하다.
- 아이디어 선정은 정찰 과제서로 갈음했다. 정찰 신규 2개를 포함한 기존 71개 제목/상태를 보존하며 무관한 후보를 추가하거나 대신 구현하지 않았다.
- 기록 검증: python3 표준 라이브러리로 JSON 재읽기·71개 제목/상태 보존·필수 필드·원장 항목 1개·구현 노트 8줄 이내를 검사했다. 릴리즈 해결 검증과 구분한다.

## 2026-09-22
- 유형: 수정 과제
- 선택: [수정 과제] 릴리즈 버전 결정 입력 복구 — 승인 정책 입력 없이는 배정 불가, 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 결과: 변경없음 (진입 조건 미충족)
- 요약: 과제서 수용 기준 1이 요구하는 승인된 증가 정책·태그 형식·릴리즈 커밋 양식·노트 위치가 brief.md·journal.md·운영자 지시 어디에도 출처와 함께 인계되지 않았다. 과제서 지시대로 반복 조사(원격 조회·gate·npm ci/test)를 재실행하지 않았고 저장소 파일·커밋·태그를 바꾸지 않았다(`git status --short` 0줄). 이 기록은 해결 성과로 세지 않으며 재개에는 프로젝트/기준 SHA·정책 원문 출처·증가 단위·커밋/태그 양식·노트 위치·자산 목록·정확한 검사 명령이 필요하다.
- 보류 아이디어:
  - [수정 과제] Codex 폴백에 headcount 스킬 실제 경로 전달 복구: pending; 외부 aidev 소유, 이번 회차 Skill 도구 3개 호출 성공.
  - useAppList 비배열 SIDEMENU_APP_LIST 캐시 방어: pending; 두 경로 end-to-end 검증 필요, 고정 과제 대체 아님.
  - globalLoading 병렬 요청 참조 카운트: pending; 호출 짝 감사 선행.
  - README 빌드 명령 정합성: pending; 과제서 차선 후보이나 진입 조건 미충족 회차에 같이 넣지 않음(범위 밖).
- 과제서: 채택 — 진입 조건 미충족 시 무변경 5줄 기록으로 끝내라는 과제서 지시를 그대로 따랐다.

## 2026-09-22
- 선택: useAppList의 비배열 sidemenuAppList 캐시 가드 (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `commonStorage`의 `sidemenuAppList`가 객체/문자열이면 `|| []`가 truthy를 걸러내지 못해 `getFormattedAppList`의 `list.map`에서 `TypeError: list.map is not a function`이 났고, 같은 값을 읽는 형제 경로(`src/utils/appList.js:77`)만 `toAppArray`로 방어돼 두 경로의 동작이 갈려 있었다. `fetchTopApps`와 `useAppList.updateList` 두 진입 경로를 기존 `toAppArray` 재사용으로 바꾸고(새 헬퍼 없음), 비배열·null·미설정 입력과 정상 배열의 매핑/limit 회귀를 함께 고정하는 `tests/unit/useAppList.spec.js`(9케이스)를 추가했다. 검증: 수정 전 새 스펙 3개 실패(Red, 원인 로그 `useAppList.js:13`) → 수정 후 `npm test` 20파일 433테스트 통과(기준 19/424), 가드를 한 곳씩 되돌릴 때마다 각각 2개씩 실패함을 확인해 인과를 증명했고, `npm run build:dev`도 통과 후 `dist/`를 지웠다.
- 보류 아이디어:
  - [수정 과제] 릴리즈 버전 결정 입력 복구: pending — release-prompt.md:21,22(2·3항)이 '이전 증가 패턴'을 요구하는데 태그 0개·릴리즈 커밋 0개이고, :24(5항)의 `skipped`는 '버전 파일도 없음'을 요구하는데 `package.json:3`에 `"version": "0.0.0"`이 있어 released도 skipped도 불가능한 구조적 교착이다. 관례 신설은 5항·AGENTS.md·docs/RELEASE.md:3,83이 모두 금지하므로 저장소 안에 합법적 수단이 없다. 이번 회차에 버전 파일·태그·CHANGELOG를 건드리지 않았다.
  - 릴리즈 절차 교착 자체를 사람에게 에스컬레이션: pending — 수정 지점이 외부 aidev 절차이고 사람의 결정이 필요하다.
  - globalLoading 병렬 요청 참조 카운트: pending — 호출 짝 감사와 실제 병렬 요청 재현이 선행되어야 한다.
  - README 문제 해결 절의 존재하지 않는 `npm run build`: pending — 3순위 문서 정합성 수정으로 남긴다(실제 스크립트는 `build:dev`/`build:core` 등 모드별).
- 과제서: 차선 — 진입 조건 1)이 정한 대로 승인된 증가 단위·태그 형식·커밋 양식·노트 위치가 어디에도 출처와 함께 없어 1순위가 성립하지 않으므로, 과제서가 '이번 회차의 기본 실행 대상'으로 지정한 차선을 구현했다.

## 2026-09-22
- 선택: README·docs 의 존재하지 않는 빌드/lint 명령과 끊긴 문서 링크 정정 (가치 2 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 과제서의 기본 실행 대상(globalLoading 참조 카운트)은 과제서가 스스로 정한 중단 조건에 걸려 차선으로 전환했다 — 31개 호출 파일을 전수 감사한 결과 `src/views/ChatStorage/ChatStorageDetail.vue:886-902 openLawViewer` 가 `startLoading()` 만 하고 어떤 경로에서도 `stopLoading()` 을 부르지 않으며, `src/views/Support/Ocr/SupportOcr.vue:203-223` 은 `stopLoading()` 이 `setTimeout` 안에만 있어 catch 경로에서 누락된다. 이 상태로 카운터를 넣으면 카운터가 0 으로 돌아오지 못해 스피너가 영구히 안 꺼지는, 현재 조기 종료보다 나쁜 회귀가 난다. 대신 차선 후보를 구현해 README 문제 해결절·docs/06 의 `npm run build`/`npm run lint`, docs/01·docs/06 의 `npm run build:prod` 를 실제 스크립트(`build:dev`, `npm test`, 모드별 `build:core`/`build:ofc`)로 바꾸고, docs/09 문서 표의 끊긴 링크 6건과 README 의 docs/07 경로 오타를 정리했다. 검증: package.json scripts 와 실제 파일 존재를 대조하는 python3 스크립트로 수정 전 끊긴 링크 5건·없는 스크립트 6건을 확인하고 수정 후 끊긴 링크 0건(남은 `build:prod` 1건은 docs/06:424 의 "GitLab CI 예제" 가상 블록으로 의도적 제외), `npm test` 19파일 424테스트 통과, `npm run build:dev` 통과 후 `dist/` 삭제.
- 보류 아이디어: [수정 과제] 릴리즈 버전 결정 입력 복구: pending — release-prompt.md:21,22 는 없는 증가 패턴을 요구하고 :24 의 skipped 는 package.json:3 의 "0.0.0" 때문에 불가한 교착. 진입 조건(승인된 증가 단위·태그 형식·커밋 양식·노트 위치) 미충족이라 버전 파일·태그·CHANGELOG 미변경.
  - 릴리즈 절차 교착을 사람에게 에스컬레이션: pending — 수정 지점이 외부 aidev 절차라 저장소 안에 수단이 없다.
  - globalLoading 호출부 누수 2건 선행 수정(ChatStorageDetail.vue:887, SupportOcr.vue:204): pending — 신규. 참조 카운트보다 먼저 고쳐야 하며, .vue 는 현재 vitest 설정(SFC 플러그인 없음)으로 테스트 불가라 증명 수단 확보가 선행 과제다.
  - globalLoading 병렬 요청 참조 카운트: pending — 위 누수 2건이 남아 있는 한 넣으면 안 된다(스피너 영구 유지).
- 과제서: 차선 — 과제서가 명시한 중단 조건("`finally` 없이 start 하는 호출부를 하나라도 발견하면 범위를 줄여 보고하라")에 해당하는 누수 2건을 실물로 확인해 지정된 차선 후보를 구현했다.

## 2026-09-23
- 선택: 알림/확인 팝업 본문의 HTML 주입 차단·개행 렌더링 정정 (vitest SFC 마운트 지원 포함) (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `Alert.vue`/`Confirm.vue` 는 본문을 `v-html="text"` 로 출력하는데 호출부 다수가 `openAlert(res?.data?.message || '...')` 로 서버 응답 메시지를 그대로 넘긴다. 서버 메시지의 태그가 실제 DOM 요소로 만들어지고(마운트 테스트로 `<script>` 요소 생성 확인), 반대로 본문의 `\n` 은 `.alert-text` 에 `white-space` 지정이 없어(morpheus-common.css:117) 줄바꿈으로 보이지 않았다. 본문을 이스케이프하고 개행과 기존 호출부의 `<br>` 표기만 `<br>` 로 되살리는 `src/utils/alertText.js` 를 추가해 두 컴포넌트에 연결했고, 증명 수단이 없던 문제를 먼저 해결하려 `vitest.config.js` 에 `@vitejs/plugin-vue` 를 연결하고 `@vue/test-utils` 를 devDep 으로 추가해 두 컴포넌트를 실제로 마운트하는 spec 을 작성했다. 검증: 기준선 `npm test` 19파일 424테스트 → 새 spec 2개로 Red 4건 확인(실제 DOM 에 `<script>` 생성·`br` 0개) → 수정 후 21파일 444테스트 통과. 컴포넌트 배선만 되돌리면 4건 실패, 개행 변환만 제거하면 4건 실패로 인과를 각각 증명했고, `npm run build:dev` 통과 후 `dist/` 를 삭제했다.
- 보류 아이디어: ChatStorageDetail.vue openLawViewer 의 고아 startLoading() 제거 — pending, 이제 SFC 마운트가 가능해져 증명 수단이 생겼다(단 이 뷰는 의존이 무거워 마운트 가능 여부 미확인).
  - SupportOcr.vue 업로드 실패 경로의 stopLoading 누락 — pending, 위 건과 함께 고쳐야 참조 카운트 도입이 가능하다.
  - Base.vue 의 `@:click` 오타로 배경 클릭 닫기가 동작하지 않음 — pending(신규), 바로 위 줄이 주석 처리된 정상 코드라 의도적 비활성일 수 있어 기대 계약 확인이 선행된다.
  - Alert/Confirm 이 `showHeader` 를 넘기지 않아 `title` 이 영구히 렌더링되지 않음 — pending(신규), `openAlert(msg, error)` 처럼 title 에 Error 객체를 넘기는 호출부 2곳도 이 때문에 증상이 감춰져 있다.
- 과제서: 없음(러너 09:59 판정) — 정찰 노트의 1순위(고아 startLoading 제거)는 SFC 마운트 없이는 증명할 수 없어 pending 으로 남기고, 정찰이 "1순위가 막히면 주저 말고 그쪽으로" 라고 지정한 차선(vitest SFC 지원)을 실제 버그 수정과 함께 구현했다.

## 2026-09-23
- 선택: 전역 로딩 스피너 계약과 법률 문서 링크 클릭 경로 테스트 고정 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 과제서 B(ChatStorageDetail.vue:887 고아 startLoading() 제거)는 근거가 지금 코드와 맞지 않아 구현하지 않았다 — `openLawViewer` 의 유일한 호출부 452행이 `handleContentClick` 의 try 안에 있고 finally(476-478)가 `stopLoading()` 을 부르므로 887행은 누수가 아니라 이미 true 인 상태를 다시 켜는 중복 호출이다(과제서는 호출부를 확인하지 않았다). 증명: 공유 모드(chat-share, 네트워크 없이 route.params.data 로 메시지 복원)로 실제 컴포넌트를 마운트해 법률 링크를 실제 DOM 클릭으로 통과시킨 뒤, ① finally 의 stopLoading() 만 제거하면 단정이 실패하고(테스트가 누수를 실제로 잡는다) ② 887행만 제거하면 2 테스트 모두 그대로 통과함(관측 가능한 동작 변화 0)을 각각 확인했다. 운영자 지시 "출력·동작이 바뀌지 않는 수정은 넣지 말 것" 에 따라 소스는 건드리지 않고 차선 후보인 계약 spec 을 구현했다. 검증: `npm ci` → `npm test` 기준선 21파일 444테스트 → 23파일 452테스트 통과, globalLoading 구현 변이 2건(문구 초기화 제거 / singleton 해제)으로 새 spec 이 각각 1건·2건 실패함을 확인해 무딘 테스트가 아님을 증명, `npm run build:dev` 통과 후 `dist/` 삭제.
- 보류 아이디어: [수정 과제] 릴리즈 버전 결정 입력 복구 — pending(9회째). HEAD 595c0c5 에서 재확인: `.github` 없음, git tag 0개, package.json:3 = 0.0.0/private:true. 진입 조건 1·2(승인된 증가 단위·태그 형식·커밋 양식·노트 위치)가 출처와 함께 인계되지 않아 버전 파일·태그·CHANGELOG·원격 전송을 일체 하지 않았고 판정을 skipped/released 로 낮추지 않았다.
  - 릴리즈 절차 교착을 사람에게 에스컬레이션 — pending; 수정 지점이 외부 aidev 절차라 저장소 안에 합법적 수단이 없다.
  - SupportOcr.vue:205 의 ocrParse 를 await 하지 않아 try/catch 가 비동기 실패를 못 잡음 — pending; stopLoading 누락 건과 묶어 함께 고쳐야 무효 변경이 되지 않는다.
  - globalLoading 참조 카운트 — pending; SupportOcr 누수가 남아 있고 loading.vue 의 60초 자동 해제가 카운터를 되돌리지 않아 선행 조건 미충족. 이번 spec 은 참조 카운트를 계약으로 고정하지 않았다.
  - ChatStorageDetail.vue:887 중복 startLoading() 제거 — rejected(누수 아님으로 확정). 동작 변화가 없어 단독 수정 대상이 아니며, 이번 회차 spec 이 실제 동작을 고정했다.
- 과제서: 차선 — B 의 근거(고아 startLoading → 60초 오버레이)가 호출부 확인 누락으로 지금 코드와 맞지 않음을 마운트 테스트와 되돌림 실험으로 확정하고, 과제서가 지정한 차선 후보(globalLoading 계약 spec)를 구현했다.

## 2026-09-23
- 선택: OCR 업로드 실패가 조용히 삼켜지는 문제 수정 — ocrParse 미-await 와 실패 경로 stopLoading 누락 (가치 4 / 위험 3 / 작업량 M)
- 결과: 성공
- 요약: `SupportOcr.vue:206` 의 `inf.tools.ocrParse.call(formData)` 는 저장소에서 유일하게 `await` 없는 `inf.tools.*` 호출이라 업로드가 실패해도 `catch` 에 도달하지 못했고, 사용자는 실패 알림 없이 2초 뒤 변환 목록으로 이동했으며 전역 스피너는 `loading.vue` 의 60초 자동 해제까지 남았다. 형제 `SupportStt.vue:131-155` 와 `docs/03-API-개발가이드.md:415` 가 정한 형태대로 `const res = await` + `isSuccess(res)` 실패 분기 + `catch` 의 `stopLoading()` 을 넣되, 성공 경로의 `setTimeout` 안 `stopLoading()` 은 그대로 둬 스피너가 이동 직전까지 켜져 있게 했다. 검증: 실제 컴포넌트를 마운트해 드롭→버튼 클릭까지 실제 DOM 으로 통과시키는 신규 스펙 6케이스가 수정 전 4건 실패(Red) → 수정 후 전부 통과했고, 세 변경분을 각각 하나씩 되돌리면 `await` 제거 시 3건·`catch stopLoading` 제거 시 2건·`isSuccess` 분기 제거 시 1건이 실패함을 확인해 인과를 증명했다. `npm test` 24파일 458테스트 통과(기준선 23/452), `npm run build:dev` 통과 후 `dist/` 삭제.
- 우선 과제(릴리즈): 진입 조건 미충족으로 무변경. 과제서 지시대로 재조사하지 않았고 버전 파일·태그·CHANGELOG·릴리즈 노트·원격 전송을 일체 하지 않았다. 승인된 증가 단위·시작 버전, 태그 형식, 릴리즈 커밋 양식, 노트 위치가 출처와 함께 인계되지 않았다. 판정을 skipped/released 로 낮추지 않는다.
- 보류 아이디어: [수정 과제] 릴리즈 버전 결정 입력 복구 — pending(10회째, 진입 조건 미충족).
  - 릴리즈 절차 교착을 사람에게 에스컬레이션 — pending; 수정 지점이 외부 aidev 절차라 저장소 안에 합법적 수단이 없다.
  - Alert/Confirm 이 showHeader 를 넘기지 않아 title 이 렌더링되지 않음 — pending; 이번 회차 차선 후보였으나 '보이게 하는 것' 이 기대 계약인지 확정이 선행.
  - globalLoading 참조 카운트 — pending; SupportOcr 누수는 이번에 해소됐으나 loading.vue 의 60초 자동 해제가 카운터를 되돌리지 않는 문제가 남아 선행 조건 미충족.
  - inf 의 나머지 네임스페이스(auth/chat/app) await 누락 1회성 감사 — pending; 이번엔 tools 만 확인했다.
- 과제서: 채택 — A(릴리즈)는 지시대로 무변경으로 두고, B(SupportOcr ocrParse 미-await)를 수용 기준 1~5 그대로 구현·검증했다.

## 2026-09-23
- 선택: 전역 Alert 의 후속 동작(callback)이 Escape 로 닫으면 유실되는 문제 수정 (가치 4 / 위험 3 / 작업량 M)
- 결과: 성공
- 요약: `openAlert(text, title, callback)` 의 callback 이 `App.vue:52` 에서 `@primary`(확인 버튼)에만 연결돼 있어, `Base.vue:25-30` 이 window keydown 으로 Escape 를 받아 `close()` 를 부르는 경로에서는 한 번도 실행되지 않았다 — 세션 만료 시 `redirectToLogin()`(interceptors.js:164,210,223 / common.js:587)과 수정 불가 팝업의 `onCancel()`(PopSimpleBotUpdate.vue:166,172)이 통째로 유실돼 사용자가 죽은 화면에 남았다. `globalAlert.js` 에 `closeAlert()`(callback 을 꺼내 null 로 비운 뒤 실행)를 추가하고 `App.vue` 를 `:model-value` + `@update:model-value="closeAlert"` 단일 경로로 바꿨다(확인 버튼도 `Alert.vue:18-21` 이 `close()` 로 수렴하므로 이중 실행이 없다). 검증: 실제 `App.vue` 를 pinia + RouterView 스텁으로 마운트해(과제서가 15분 안에 확인하라던 것 — 가능했다) 실제 `globalAlert.js` 싱글턴에 `openAlert(..., cb)` 를 호출하고 실제 DOM 이벤트(버튼 `click`, window `keydown{Escape}`)로 닫는 신규 스펙 5케이스가 수정 전 2건 실패(Red) → 수정 후 통과. 변이 4건으로 각 요소의 인과를 증명했다: ①`@primary` 전용 배선으로 되돌리면 3건 실패 ②callback 을 비우지 않으면 1건 실패 ③비우는 시점을 실행 뒤로 옮기면(clear-after) 재진입 케이스 1건 실패 ④`@primary` 와 `@update:model-value` 를 둘 다 연결하면 이중 실행으로 2건 실패. `npm test` 25파일 463테스트 통과(기준선 24/458), `npm run build:dev` 통과 후 `dist/` 삭제.
- 우선 과제(릴리즈): 진입 조건 미충족으로 무변경(11회째). 과제서 지시대로 재조사하지 않았고 버전 파일·태그·CHANGELOG·릴리즈 노트·릴리즈 커밋·원격 전송을 일체 하지 않았으며 `docs/RELEASE.md` 도 건드리지 않았다. 판정을 skipped/released 로 낮추지 않는다. 필요한 사람 입력: ①시작 버전(0.0.1 대 0.1.0)과 증가 단위 ②태그 형식·주석 태그 여부 ③릴리즈 커밋 메시지 양식 ④릴리즈 노트 위치·양식·언어 ⑤GitHub Release 사용 여부.
- 보류 아이디어: [수정 과제] 릴리즈 버전 결정 입력 복구 — pending(11회째, 진입 조건 미충족).
  - 릴리즈 절차 교착을 사람에게 에스컬레이션 — pending; 수정 지점이 외부 aidev 절차라 저장소 안에 합법적 수단이 없다.
  - Confirm 도 Escape 로 닫으면 cancelAction 이 실행되지 않음 — pending; 이번 Alert 수정과 같은 형태로 고칠 수 있으나 `useConfirmStore` 의 Promise 해제 방식 확인이 선행이라 이번 범위 밖으로 뒀다.
  - `src/utils/alerts.js` 죽은 중복 모듈 제거 — pending; 이번 회차에 재확인(`grep -rn "utils/alerts" src tests` = 0건, App.vue 는 `globalAlert.js` 만 쓴다). 동작 변화가 없어 1순위가 성립한 이번 회차에는 하지 않았다.
  - globalLoading 참조 카운트 — pending; loading.vue 의 60초 자동 해제가 카운터를 되돌리지 않는 문제가 남아 선행 조건 미충족.
- 과제서: 채택 — A(릴리즈)는 지시대로 무변경으로 두고, B 를 수용 기준 1~5 그대로 구현·검증했다. 다만 수용 기준 2 의 "1회성" 을 처음 쓴 형태(다음 openAlert 후 닫기)는 `openAlert` 가 callback 을 항상 덮어쓰므로 변이 실험에서 실패하지 않는 무딘 테스트임을 확인해, 실제로 판별되는 두 계약(닫힌 뒤 `alertCallback` 이 null 인지 직접 단정 / callback 이 새 Alert 을 열 때 새 callback 이 살아남는지)으로 바꿔 증명했다.

## 2026-09-23
- 선택: 전역 Confirm 을 Escape 로 닫으면 취소 동작(cancelAction)이 유실되는 문제 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `Base.vue:25-30` 이 window keydown(Escape) 에서 `update:modelValue` 로만 닫힘을 알리는데 `Confirm.vue:33` 이 이를 그대로 부모에 넘겨 `App.vue:51` 의 `v-model` 이 `confirmStore.visible` 만 false 로 만들었고, `@secondary` 에 걸린 `cancelAction` 은 한 번도 실행되지 않았다 — `Chat/Main.vue:910` / `Chat/Index.vue:307` 의 base64 붙여넣기 취소 처리(텍스트만 남기고 base64 제거)가 통째로 유실되며, 이 경로는 `Main.vue:836` 에서 `event.preventDefault()` 를 먼저 하므로 사용자가 붙여넣은 텍스트가 그대로 사라진다. Base 의 닫힘(Escape/배경 클릭/헤더 X)을 '취소' 로 처리하는 `onBaseClose` 를 두고, 동작이 새 팝업을 열어도 뒤따르는 닫힘 emit 이 그 팝업을 지우지 않도록 `onPrimary`/`onSecondary` 의 emit 순서를 닫힘 먼저로 바꿨다(App.vue 는 무변경). 검증: 실제 `App.vue` 를 pinia + RouterView 스텁으로 마운트해 실제 `useConfirmStore.open()` 을 호출하고 실제 DOM 이벤트(버튼 click, window keydown{Escape})로 닫는 신규 스펙 8케이스가 수정 전 5건 실패(Red) → 수정 후 전부 통과. 변이 2건으로 인과를 분리 증명했다: ①`onBaseClose` 배선만 되돌리면 Escape 관련 3건 실패 ②버튼 emit 순서만 되돌리면 재진입 3건 실패. `npm test` 26파일 471테스트 통과(기준선 25/463), `npm run build:dev` 통과 후 `dist/` 삭제.
- 우선 과제(릴리즈): 진입 조건 미충족으로 무변경(12회째). 이번 회차에 직접 재확인한 근거 — `git tag` 0개, `package.json` version=0.0.0, `.github` 디렉터리 없음. 버전 파일·태그·CHANGELOG·릴리즈 노트·릴리즈 커밋·원격 전송을 일체 하지 않았고 `docs/RELEASE.md` 도 건드리지 않았다. 판정을 skipped/released 로 낮추지 않는다. 필요한 사람 입력: ①시작 버전(0.0.1 대 0.1.0)과 증가 단위 ②태그 형식·주석 태그 여부 ③릴리즈 커밋 메시지 양식 ④릴리즈 노트 위치·양식·언어 ⑤GitHub Release 사용 여부. 이 저장소는 태그가 아니라 브랜치 머지로 배포되므로 버전 릴리즈 관례를 도입할지 자체가 먼저 결정돼야 한다.
- 보류 아이디어: [수정 과제] 릴리즈 버전 결정 입력 복구 — pending(12회째, 진입 조건 미충족).
  - 릴리즈 절차 교착을 사람에게 에스컬레이션 — pending; 수정 지점이 워크트리 밖(외부 aidev 절차)이라 저장소 안에 합법적 수단이 없다.
  - `Base.vue:46` 의 `@:click` 오타로 배경 클릭 닫기가 동작하지 않음 — pending; 이번 수정으로 Confirm 은 배경 클릭이 살아나도 취소가 정상 실행되는 상태가 됐으나, 오타 수정 자체는 Base 를 쓰는 모든 팝업에 영향이 있어 기대 계약 확인이 선행.
  - `src/utils/alerts.js` 죽은 중복 모듈 제거 — pending; 관측 가능한 동작 변화가 없어 1순위가 성립한 이번 회차에는 하지 않았다.
  - globalLoading 참조 카운트 — pending; `loading.vue` 의 60초 자동 해제가 카운터를 되돌리지 않는 문제가 남아 선행 조건 미충족.
- 과제서: 채택 — A(릴리즈)는 지시대로 재조사 없이 무변경으로 두고, 보류 목록의 1순위(Confirm 의 Escape 취소 유실)를 구현했다. 과제서가 선행이라던 "confirmStore 의 Promise 해제 방식 확인" 은 실제로는 Promise 가 없고 콜백 기반(`useConfirmStore.js:8-11`)임을 읽어 확인했으며, 그 과정에서 같은 계열의 재진입 결함(동작이 새 팝업을 열면 뒤따르는 닫힘 emit 이 지워버림)을 버튼 경로에서도 테스트로 잡아 함께 고쳤다.

## 2026-09-23
- 선택: 앱 공유 등록이 공통 오류('COM')를 '권한 없음' 으로 잘못 안내하던 문제 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `src/utils/shareAppService.js` 의 `createSharedApp` 이 `try/catch` 로 모든 예외를 잡아 `{ok:false, message:'요청 처리 중 오류가 발생했습니다.'}` 로 바꾸는 바람에, 호출부 4곳(`HeaderShare.vue:108`, `SupportList.vue:391`, `InfoSearchList.vue:293`, `MyAgentList.vue:425`)이 모두 가지고 있는 `catch (e) { if (e == 'COM') return; ... }` 분기가 한 번도 실행되지 않았다 — 공통 인터셉터가 이미 전역 Alert 을 띄우고 `Promise.reject('COM')` 한 오류(세션 만료 / BZ01 업무 오류, `interceptors.js:164,210,223,233`)까지 화면이 `if(!result.ok)` 로 흘러가 '권한이 없어 설치가 불가합니다. 관리자에게 문의해 주시기 바랍니다.' 를 덧붙여 안내했고, 통신 오류도 같은 문구로 잘못 안내됐다. 예외를 삼키지 않고 그대로 던지도록 바꿔 기존 분기를 살렸다(검증 실패 분기·성공 경로는 그대로). 검증: HTTP 전송(axios adapter)만 대역으로 바꾸고 **실제 `interceptors.js` → 실제 `inf.app.share.call` → 실제 `createSharedApp` → 실제 `HeaderShare.vue` 마운트**를 그대로 통과시키는 신규 스펙 6건이 수정 전 3건 실패(Red: BZ01 에 권한 토스트가 붙음 / 통신 오류에 권한 토스트가 붙음) → 수정 후 전부 통과했고, 수정만 되돌리면 같은 3건이 다시 실패함을 확인해 인과를 증명했다. 나머지 3건(전역 Alert 표시 / 진짜 403 실패의 권한 안내 유지 / 성공 안내)은 수정 전후 모두 통과해 범위가 좁음을 고정한다. `npm test` 27파일 477테스트 통과(기준선 26/471), `npm run build:dev` 통과 후 `dist/` 삭제.
- 우선 과제(릴리즈): 진입 조건 미충족으로 무변경(13회째). 이번 회차에 직접 재확인 — `git tag` 0개, `package.json` version=0.0.0, `.github` 디렉터리 없음, `docs/RELEASE.md` 는 근거 문서이지 릴리즈 노트가 아님. 과제서의 "워크플로 파일과 실패한 단계의 스크립트를 고치라" 는 지시는 이 저장소 안에서 수행할 수 없다 — 실패 지점이 워크트리 밖(외부 aidev `bin/run.sh` → `release.json` → `bin/gate.py`)이고, 저장소 안에서 통과시키는 유일한 수단은 버전 파일·태그·릴리즈 노트를 새로 만드는 것인데 이는 이 세션의 절대 규칙("릴리즈는 이 세션의 일이 아닙니다")과 "워크플로를 느슨하게 만들어 통과시키는 것은 금지" 양쪽에 걸린다. 따라서 버전 파일·태그·CHANGELOG·릴리즈 노트·릴리즈 커밋·원격 전송을 일체 하지 않았고 `docs/RELEASE.md` 도 수정하지 않았다. 판정을 skipped/released 로 낮추지 않는다. 필요한 사람 입력: ①버전 릴리즈 관례 도입 여부 자체(현 배포는 GitLab CI 의 브랜치 푸시 트리거) ②시작 버전(0.0.1 대 0.1.0)과 증가 단위 ③태그 형식·주석 태그 여부 ④릴리즈 커밋 메시지 양식 ⑤릴리즈 노트 위치·양식·언어 ⑥GitHub Release 사용 여부.
- 보류 아이디어: [수정 과제] 릴리즈 버전 결정 입력 복구 — pending(13회째, 진입 조건 미충족). 저장소 안에 합법적 수단이 없어 사람 입력 없이는 진행 불가.
  - `inf` 나머지 네임스페이스 await 누락 감사 — done(결함 없음). `inf.<도메인>.<이름>.call` 중 await 없는 3곳은 모두 정당하다: `useFileAccept.js:87-88` 은 `Promise.all` 안이고, `common.js:477` 의 `groupLogin` 은 form submit 후 즉시 resolve 하는 이동 경로다.
  - `Base.vue:46` 의 `@:click` 이 오타라는 판단 — rejected. Vue 3 런타임의 `patchEvent` 는 `on:click` 형태(`name[2] === ':'`)를 정상 이벤트명으로 파싱하므로 배경 클릭 닫기는 동작한다. 고칠 동작 변화가 없다.
  - `src/utils/alerts.js` 죽은 중복 모듈 제거 — pending; 관측 가능한 동작 변화가 없어 단독 가치가 낮다.
  - globalLoading 참조 카운트 — pending; `loading.vue` 의 타이머가 `modelValue` 의 false→true 전이에서만 재시작돼 이미 켜진 상태의 추가 `startLoading()` 을 반영하지 못하는 문제가 먼저 정리돼야 한다.
- 과제서: 차선 — A(릴리즈)는 저장소 안에서 합법적으로 수행 가능한 수단이 없어 지시대로 무변경으로 두고(게이트 완화 금지·릴리즈 금지 양쪽 규칙 준수), 보류 목록의 감사 항목을 실제로 수행하던 중 발견한 실제 결함(공유 등록의 'COM' 삼킴)을 구현·검증했다.

## 2026-09-23
- 선택: 심플봇 '추천 프롬프트' 생성 실패가 조용히 삼켜지던 문제 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `PopSimpleBot.vue:334` / `PopSimpleBotUpdate.vue:421` 의 `reload()` 는 `isSuccess(res)` 가 false 인 업무 실패 경로에 else 가 없어, `startLoopTyping` 이 찍던 안내 문구('추천 프롬프트를 가져오는중입니다.' 의 일부)가 입력란에 얼어붙은 채 남고 그대로 `buildCreateAppJson` 의 `payload.prompt`(PopSimpleBot.vue:505)로 전송됐다 — `interceptors.js:228-237` 은 HTTP 200 + 본문 code 불일치 응답을 그대로 통과시키므로 이 경로는 실제로 화면까지 도달한다. 같은 함수의 `catch` 는 이미 `rolePrompt.value = ""` 로 비우고 있어 '실패하면 비운다' 가 원 저자의 계약임이 코드에 남아 있었고, 두 컴포넌트의 `reload()` 는 문자 단위로 동일해 양쪽을 같이 고쳤다. 실패 응답에서 입력란을 비우고 `toast('추천 프롬프트를 가져오지 못했습니다.')` 로 알리게 했으며, 'COM' 이 아닌 예외도 같은 안내를 하도록 `catch` 를 보완했다(COM 은 인터셉터 Alert 과 겹치지 않게 기존 early return 뒤에 둠). 검증: HTTP 전송(axios adapter)만 대역으로 바꾸고 **실제 interceptors.js → 실제 `inf.app.promptCreate.call` → 실제 두 컴포넌트 마운트**를 통과시키는 신규 스펙 10건(2컴포넌트 × 5케이스)이 수정 전 6건 실패(Red: 입력란에 '추천 ' 잔존 / 안내 없음) → 수정 후 전부 통과. 150ms 타이핑 인터벌이 실제로 돌도록 응답을 deferred 로 잡아 500ms 흘려보낸 뒤 결말지었고(즉시 resolve 로는 Red 가 안 난다), 변이 4건으로 각 요소의 인과를 분리 증명했다: ①else 의 입력란 비우기 제거 → 2건 실패 ②else 의 toast 제거 → 2건 실패 ③catch 의 toast 제거 → 2건 실패 ④catch 의 toast 를 'COM' early return 앞으로 이동 → 이중 안내로 2건 실패. `npm test` 28파일 487테스트 통과(기준선 27/477), `npm run build:dev` 통과 후 `dist/` 삭제.
- 보류 아이디어: [수정 과제] 릴리즈 버전 결정 입력 복구 — pending(14회째, 진입 조건 미충족). 이번 회차 재확인: `git tag` 0개, `package.json` version=0.0.0/private:true, `.github` 없음. 실패 지점이 워크트리 밖(외부 aidev `bin/run.sh` → `release.json` → `bin/gate.py`)이라 저장소 안에 합법적 수단이 없다. 버전 파일·태그·CHANGELOG·릴리즈 노트·원격 전송을 일체 하지 않았고 판정을 skipped/released 로 낮추지 않았다.
  - 릴리즈 절차 교착을 사람에게 에스컬레이션 — pending; 수정 지점이 워크트리 밖이라 저장소 안에 합법적 수단이 없다.
  - `loading.vue` 의 자동 해제 타이머가 이미 켜진 상태의 추가 `startLoading()` 을 반영하지 않음 — pending; 어느 쪽이 기대 계약인지 확정이 선행이며 globalLoading 참조 카운트의 선행 조건이다.
  - `globalLoading` 참조 카운트 도입 — pending; 위 타이머 계약이 미확정이라 지금 넣으면 스피너가 영구히 안 꺼지는 회귀 위험.
  - `src/utils/alerts.js` 죽은 중복 모듈 제거 / `SupportSttList.vue` 의 미사용 Confirm import 제거 — pending; 관측 가능한 동작 변화가 없어 단독 가치가 낮다(운영자 지시상 무효 변경 금지).
- 과제서: 채택 — 정찰이 고른 항목(PopSimpleBot/PopSimpleBotUpdate 의 `reload()` else 누락)을 그대로 구현했다. 정찰이 "미확인" 으로 남긴 두 가지를 실제로 확인했다: ①`validatePromptCreate` 는 userId·name 외에 description 까지 요구하므로 테스트에서 `input#name` 과 `textarea#description` 을 실제 DOM 으로 채워야 한다 ②차선(ChatData.vue source_seq)은 이번에도 손대지 않았다.

## 2026-09-23
- 선택: 심플봇 수정 팝업이 저장 실패에도 팝업을 닫고 'update' 를 알려 수정 내용이 유실되는 문제 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `PopSimpleBotUpdate.vue` 의 `onCreate()` 는 `finally` 에서 `emit('update', true)` 와 `close()` 를 무조건 실행해, `inf.app.updateAppFile`/`inf.app.createApp` 이 업무 실패(HTTP 200 + `isSuccess` false)를 돌려주고 `toast('수정에 실패했습니다.')` 를 띄운 뒤에도 팝업이 닫혀 사용자가 입력한 이름·설명·프롬프트·첨부가 통째로 사라졌고, `App.vue:37-43` 이 호출자의 `updateCallback()`(목록 새로고침)을 돌려 성공한 것처럼 보였다. 형제 `PopSimpleBot.vue:566-604` 가 성공 경로 안에서만 `emit`+`close` 하는 형태 그대로 `emit('update', true)`/`close()` 를 `await tryUpdateAppIcon()` 뒤로 옮기고 `finally` 에는 `isSendLoading.value = false` 만 남겼으며, 저장소 표준형(`PopSimpleBotUpdate.vue:347-349`)대로 `catch` 의 토스트 앞에 `if (e == 'COM') return` 을 넣어 인터셉터의 전역 Alert 과 겹치지 않게 했다. 검증: HTTP 전송(axios adapter)만 대역으로 바꾸고 **실제 `interceptors.js` → 실제 `inf.app.*.call` → 실제 `PopSimpleBotUpdate.vue` 마운트**를 통과시키는 신규 스펙 10건(팝업을 열어 실제 `/app/{id}` 상세 응답으로 폼을 채운 뒤 실제 DOM 으로 '수정' 클릭)이 수정 전 6건 실패(Red: 실패인데 `update:modelValue` false·`update` emit / COM 중복 토스트 / `cancel` 2회) → 수정 후 전부 통과. 변이 3건으로 인과를 분리 증명했다: ①`emit`+`close` 를 `finally` 로 되돌리면 6건 실패 ②`if (e == 'COM') return` 만 제거하면 1건 실패 ③성공 경로 emit 을 남긴 채 `finally` 에도 emit 을 추가하면 중복으로 7건 실패. `npm test` 29파일 497테스트 통과(기준선 28/487), `npm run build:dev` 통과 후 `dist/` 삭제.
- 우선 과제(릴리즈): 진입 조건 미충족으로 무변경(15회째). 과제서 지시대로 재조사하지 않았고 버전 파일·태그·CHANGELOG·릴리즈 노트·릴리즈 커밋·원격 전송을 일체 하지 않았으며 `docs/RELEASE.md` 도 수정하지 않았다. 판정을 skipped/released 로 낮추지 않는다. 필요한 사람 입력: ①버전 릴리즈 관례 도입 여부 자체(현 배포는 GitLab CI 브랜치 푸시라 태그가 배포에 불필요) ②시작 버전(0.0.1 대 0.1.0)과 증가 단위 ③태그 형식·주석 태그 여부 ④릴리즈 커밋 메시지 양식 ⑤릴리즈 노트 위치·양식·언어 ⑥GitHub Release 사용 여부.
- 보류 아이디어: [수정 과제] 릴리즈 버전 결정 입력 복구 — pending(15회째, 진입 조건 미충족. 실패 지점이 워크트리 밖이라 저장소 안에 합법적 수단이 없다).
  - `PopSimpleBotUpdate.tryUpdateAppIcon` 이 `appEdit` 실패를 `catch(e){}` 로 완전히 삼킴 — pending(이번 차선 후보). 이번 스펙에 '아이콘 실패가 성공 판정을 바꾸지 않는다' 는 현 동작을 고정하는 케이스를 넣어 뒀으니, 계약을 '별도 토스트로 알리되 앱 수정 자체는 성공' 으로 바꿀 때 그 케이스부터 고치면 된다.
  - `loading.vue` 의 자동 해제 타이머가 이미 켜진 상태의 추가 `startLoading()` 을 반영하지 않음 — pending; 기대 계약 확정이 선행.
  - `globalLoading` 참조 카운트 도입 — pending; 위 타이머 계약이 미확정이라 지금 넣으면 스피너가 안 꺼지는 회귀 위험.
  - Alert/Confirm 의 `showHeader` 미전달 + `openAlert` title 자리에 Error 객체를 넘기는 호출부 3곳 — pending; '보이게 하는 것' 이 기대 계약인지 확정이 선행.
- 과제서: 채택 — A(릴리즈)는 지시대로 재조사 없이 무변경으로 두고, B 를 수용 기준 1~6 그대로 구현·검증했다. 과제서가 "`name`·`description` 을 실제 DOM 으로 채우라" 고 한 부분은 실제로는 부족했다 — `validateCreateApp()` 은 그 둘 외에 LLM·임베딩 모델, 프롬프트, **첨부 1개 이상**까지 요구하므로, 팝업을 닫힌 상태로 마운트한 뒤 열어 실제 `/app/{id}` 상세 조회 응답으로 폼 전체가 채워지게 하는 경로를 썼다(첨부는 `oldYn:'Y'` 기존 문서라 `/app/update/doc` 단계도 자연히 돈다).

## 2026-09-24
- 선택: 심플봇 생성 실패 시 전역 Alert 과 토스트가 겹쳐 안내되던 문제 수정 (가치 3 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: `PopSimpleBot.vue:599` 의 `onCreate` catch 에만 `if (e == 'COM') return` 가드가 없어, 인터셉터(`interceptors.js:228-236`)가 이미 전역 Alert 을 띄우고 `Promise.reject('COM')` 한 오류(세션 만료 / status≠200 + BZ01)에도 '앱생성에 실패하였습니다.' 토스트가 겹쳐 떴다 — 같은 파일의 다른 catch 3곳(261,295,365)과 형제 `PopSimpleBotUpdate.onCreate`(679) 는 모두 가드를 가져 두 경로가 같은 실패를 다르게 안내하던 비대칭이었다. 검증: HTTP 전송(axios adapter) 한 겹만 대역으로 바꾸고 실제 `interceptors.js` → 실제 `inf.app.*.call` → 실제 `PopSimpleBot.vue` 마운트를 통과시키는 신규 스펙 6건(팝업을 닫힌 상태로 마운트 후 열어 실제 `/app/model` 로 LLM·임베딩을 채우고, 이름·설명·프롬프트를 실제 DOM 으로, 첨부는 실제 file input `change` 이벤트로 올린 뒤 '만들기' 클릭)이 수정 전 1건 실패(Red: Alert 이 떠 있는데 토스트도 함께 들어옴) → 수정 후 전부 통과. 첫 케이스가 `/app/create/doc` 와 `/app/create` 요청이 실제로 나갔음을 단정해 폼 검증에 막혀 조용히 통과하는 형태가 아님을 고정했고, 변이 1건(가드를 토스트 뒤로 이동)을 6회 반복해 같은 1건이 항상 실패함을 확인했다. `npm test` 30파일 503테스트 통과(기준선 29/497), `npm run build:dev` 통과 후 `dist/` 삭제.
- 우선 과제(릴리즈): 진입 조건 미충족으로 무변경(17회째). 이번 회차 워크트리(HEAD 5d948b7)에서 직접 재확인 — `git tag` 0개, `package.json:3` version=0.0.0, `.github`/`CHANGELOG.md`/`VERSION`/`scripts`/`Makefile` 모두 없음. 과제서의 "워크플로 파일과 실패한 단계의 스크립트를 고치라" 는 이 저장소 안에서 수행 불가하다 — 실패 지점이 워크트리 밖(외부 aidev `bin/run.sh` → `release.json` → `bin/gate.py`)이고, 저장소 안에서 통과시키는 유일한 수단은 버전 파일·태그·릴리즈 노트를 새로 만드는 것인데 이는 이 세션의 절대 규칙("릴리즈는 이 세션의 일이 아닙니다")과 "워크플로를 느슨하게 만들어 통과시키는 것은 금지" 양쪽에 걸린다. 버전 파일·태그·CHANGELOG·릴리즈 노트·릴리즈 커밋·원격 전송을 일체 하지 않았고 `docs/RELEASE.md` 도 수정하지 않았다. 판정을 skipped/released 로 낮추지 않는다. 필요한 사람 입력: ①버전 릴리즈 관례 도입 여부 자체(현 배포는 GitLab CI 브랜치 푸시라 태그가 배포에 불필요) ②시작 버전(0.0.1 대 0.1.0)과 증가 단위 ③태그 형식·주석 태그 여부 ④릴리즈 커밋 메시지 양식 ⑤릴리즈 노트 위치·양식·언어 ⑥GitHub Release 사용 여부.
- 보류 아이디어: Sidemenu.vue:499 `onDelete` 의 catch 에 'COM' 가드 누락 — pending(다음 1순위). `pInf` 도 같은 `createInstance` 를 쓰므로 인터셉터가 동일하게 'COM' 을 던진다는 것을 이번에 확인했다(`pythonInterface.js:8`). 실패 경로에서 `openedId` 가 초기화되지 않는 문제도 같은 자리에 있다.
  - `gotoShare`(Sidemenu.vue:441 / Chat/Main.vue:544)의 'COM' 가드 누락 — rejected. 이번 회차에 읽어 확인: 두 함수 모두 API 호출 없이 클립보드 복사만 하므로 'COM' 이 발생할 수 없다.
  - [수정 과제] 릴리즈 버전 결정 입력 복구 — pending(17회째, 진입 조건 미충족. 저장소 안에 합법적 수단이 없다).
  - `PopSimpleBot.tryUpdateAppIcon`/`tryUpdateLog` 와 `PopSimpleBotUpdate.tryUpdateAppIcon` 이 `catch(e){}` 로 예외를 완전히 삼킴 — pending; '아이콘 실패가 전체 실패인가' 라는 기대 계약이 미확인이고 기존 스펙이 현 동작을 고정하고 있다.
  - `loading.vue` 자동 해제 타이머 계약 확정 → `globalLoading` 참조 카운트 — pending; 타이머 계약이 미확정이라 지금 넣으면 스피너가 안 꺼지는 회귀 위험.
- 과제서: 기각 — 정찰이 실패해 과제서가 없었다(러너 00:28 판정). 회차 노트의 정찰 노트에 남은 후보(PopSimpleBot.vue:599)를 직접 재검증해 골랐고, 정찰이 경고한 "Red 가 실제로 나는지 먼저 확인할 것" 을 지켰다.

## 2026-09-24
- 선택: 사이드메뉴 대화 삭제 실패 시 전역 Alert 과 토스트가 겹쳐 안내되던 문제 수정 (가치 3 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: `Sidemenu.vue:499` 의 `onDeleted` catch 에만 `if (e == 'COM') return` 가드가 없어, 인터셉터가 이미 전역 Alert 을 띄우고 `Promise.reject('COM')` 한 오류(세션 만료 / status≠200 + BZ01 / axios 오류 + BZ01)에도 '삭제 중 오류가 발생했습니다.' 토스트가 겹쳐 떴다 — 같은 파일의 `fetchAppList`(:233 `if (e !== 'COM')`)와 형제 `onSaved`(:540)는 모두 가드를 가진 비대칭이었다. `pInf` 가 `inf` 와 같은 `createInstance('/papi')` 를 쓰므로(`pythonInterface.js:8`) 인터셉터가 동일하게 'COM' 을 던지는 것을 실제 통과로 확인했다. 검증: HTTP 전송(axios adapter) 한 겹만 대역으로 바꾸고 실제 `interceptors.js` → 실제 `pInf.chat.*.call` → 실제 `Sidemenu.vue` 마운트를 통과시키는 신규 스펙 6건(이 저장소 첫 Sidemenu 마운트 하네스: `openSide:true` 마운트 → `/app/side/list` 로 앱 렌더 → 앱 펼치기 → `/history_list` 로 대화 렌더 → '채팅 설정' 클릭 → 드롭다운 '삭제' 클릭, 전부 실제 DOM 클릭)이 수정 전 2건 실패(Red: Alert 이 떠 있는데 토스트도 함께 들어옴) → 수정 후 전부 통과. 변이 1건(가드 줄만 제거)으로 같은 2건만 다시 실패함을 확인해 인과와 범위를 분리 증명했다. `npm test` 31파일 509테스트 통과(기준선 30/503), `npm run build:dev` 통과 후 `dist/` 삭제.
- 우선 과제(릴리즈): 진입 조건 미충족으로 무변경(18회째). 이번 회차 워크트리(HEAD 1a8cb85)에서 직접 재확인 — `git tag` 0개(`--sort=-creatordate` 출력 없음, `is-shallow-repository`=false), `package.json:3` version=0.0.0, `.github`/`CHANGELOG.md`/`VERSION`/`scripts`/`Makefile` 모두 없음, `.gitlab-ci.yml` 의 `CI_COMMIT_TAG` 참조 0건. 과제서의 "워크플로 파일과 실패한 단계의 스크립트를 고치라" 는 이 저장소 안에서 수행 불가하다 — 고칠 워크플로 파일이 저장소에 없고(`.github` 부재, GitLab CI 는 브랜치 푸시 배포 전용), 실패 지점이 워크트리 밖(외부 aidev `bin/run.sh` → `release.json` → `bin/gate.py`)이다. 저장소 안에서 통과시키는 유일한 수단은 버전 파일·태그·릴리즈 노트 관례를 새로 발명하는 것인데 이는 이 세션의 절대 규칙("릴리즈는 이 세션의 일이 아닙니다")·"워크플로를 느슨하게 만들어 통과시키는 것은 금지"·AGENTS.md("확인되지 않은 관례를 새로 만들거나 릴리즈 판정 조건을 완화하지 않는다") 셋 모두에 걸린다. 버전 파일·태그·CHANGELOG·릴리즈 노트·릴리즈 커밋·원격 전송을 일체 하지 않았고 `docs/RELEASE.md` 도 수정하지 않았다. 판정을 skipped/released 로 낮추지 않는다. 필요한 사람 입력: ①버전 릴리즈 관례 도입 여부 자체(현 배포는 GitLab CI 브랜치 푸시라 태그가 배포에 불필요) ②시작 버전(0.0.1 대 0.1.0)과 증가 단위 ③태그 형식·주석 태그 여부 ④릴리즈 커밋 메시지 양식 ⑤릴리즈 노트 위치·양식·언어 ⑥GitHub Release 사용 여부(현 원격 배포는 GitLab).
- 보류 아이디어: `Sidemenu.onDeleted` 의 `openedId` 초기화가 성공 경로에만 있는 비대칭 — rejected. 이번 회차에 실제로 확인: `PopAppChatSetting.DelCt`(:22-25)가 `'deleted'` 직후 `'close'` 를 emit 하고 템플릿의 `@close="openedId = null"`(`Sidemenu.vue:740`)이 이를 비우므로 성공·실패 모두 드롭다운은 이미 닫힌다 — `finally` 로 옮겨도 관측 가능한 동작 변화가 없어 운영자 지시상 무효 변경이다. 이를 고정하는 케이스('실패해도 설정 드롭다운은 닫힌다')를 스펙에 넣어 뒀다. 같은 이유로 `gotoShare:444`·`onSaved:543` 의 `finally` 도 중복이다.
  - [수정 과제] 릴리즈 버전 결정 입력 복구 — pending(18회째, 진입 조건 미충족. 저장소 안에 합법적 수단이 없다).
  - 릴리즈 절차 교착 자체를 사람에게 에스컬레이션 — pending; 수정 지점이 워크트리 밖이라 저장소 안에 합법적 수단이 없다.
  - `getHistoryList`(:416) 의 bare catch 가 설정 오류까지 삼켜 '최근 대화 기록이 없습니다' 로 보이게 함 — pending(신규). 이번 하네스 구축 중 실제로 당했다: `VITE_PYTHON_API_TARGET` 이 비면 `getPApi()` 가 던지는데(`api/index.js:16`) 그 예외가 통째로 삼켜져 빈 목록과 구분되지 않는다. 다만 '조회 실패를 사용자에게 알릴 것인가' 라는 기대 계약이 미확인(사이드메뉴는 보조 UI라 조용한 빈 목록이 의도일 수 있다).
  - 심플봇 두 팝업의 `tryUpdateAppIcon`/`tryUpdateLog` 가 `catch(e){}` 로 예외를 완전히 삼킴 — pending; '아이콘 실패가 전체 실패인가' 라는 기대 계약이 미확인이고 `simpleBotUpdateClose.spec.js` 가 현 동작을 고정하고 있다.
  - `loading.vue` 자동 해제 타이머 계약 확정 → `globalLoading` 참조 카운트 — pending; 타이머 계약이 미확정이라 지금 넣으면 스피너가 안 꺼지는 회귀 위험.
- 과제서: 채택 — 정찰이 고른 B(`Sidemenu.onDeleted`)를 구현했다. 다만 과제서가 묶어 지시한 두 변경 중 하나는 근거를 확인해 기각했다: 'COM' 가드 누락은 실재하는 결함이라 고쳤고, `openedId` 를 `finally` 로 옮기라는 지시는 `@close` 배선이 이미 닫고 있어 무효 변경이므로 하지 않았다(대신 그 사실을 테스트로 고정). 정찰이 경고한 `item.item.prj_id` TypeError 는 실제 emit 경로로만 돌려 피했고, 미확인이라던 앱 펼치기 DOM 은 `button.btn-chat-section` 임을 확인했다.

## 2026-09-24
- 선택: 사이드메뉴 대화 이력 조회 실패가 '최근 대화 기록이 없습니다' 로 감춰지던 문제 수정 (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `Sidemenu.vue:416` 의 `getHistoryList` 는 bare catch 로 모든 예외를 삼키고 `!isSuccess(res)` 업무 실패도 조용히 `historyMap[appId] = []` 로 만들어, 조회 실패가 Nodata 의 '최근 대화 기록이 없습니다'(`:756`) 라는 **사실 진술**과 구분되지 않았다 — 같은 파일의 형제 목록 조회 `fetchAppList`(`:208` 업무 실패 토스트 / `:233` `if (e !== 'COM')` 토스트)는 둘 다 알리고 있어 같은 실패를 두 경로가 다르게 처리하던 비대칭이다. `fetchAppList` 의 문구 양식을 그대로 따라 업무 실패는 `toast(res?.data?.message || '최근 대화 기록 조회에 실패하였습니다.')`, 공통 처리되지 않은 예외는 `toast('최근 대화 기록 조회 중 오류가 발생하였습니다.')` 로 안내하고, 인터셉터가 이미 전역 Alert 을 띄운 `'COM'` 만 조용히 넘기게 했다(`[]` 로 비우는 기존 동작은 유지). 검증: HTTP 전송(axios adapter) 한 겹만 대역으로 바꾸고 실제 `interceptors.js` → 실제 `pInf.chat.getHistoryList.call` → 실제 `Sidemenu.vue` 마운트 → 실제 DOM 으로 앱 펼치기까지 통과시키는 신규 스펙 7건이 수정 전 4건 실패(Red: 통신 오류·설정 오류·업무 실패 메시지·업무 실패 기본문구 모두 토스트 0건) → 수정 후 전부 통과. 나머지 3건(성공 렌더 / COM 은 Alert 만 / 실패 시 목록 비움)은 수정 전후 모두 통과해 범위가 좁음을 고정한다. 변이 3건으로 인과를 분리 증명: ①업무 실패 토스트 제거 → 해당 2건만 실패 ②catch 토스트 제거 → 해당 2건만 실패 ③`if (e !== 'COM')` 가드만 제거 → COM 중복 안내로 1건만 실패. `npm test` 32파일 516테스트 통과(기준선 31/509), `npm run build:dev` 통과 후 `dist/` 삭제.
- 우선 과제(릴리즈): 진입 조건 미충족으로 무변경(19회째). 과제서 지시대로 재조사하지 않았고 버전 파일·태그·CHANGELOG·릴리즈 노트·릴리즈 커밋·원격 전송을 일체 하지 않았으며 `docs/RELEASE.md` 도 수정하지 않았다. 판정을 skipped/released 로 낮추지 않는다. 고칠 워크플로 파일이 저장소에 없고 실패 지점이 워크트리 밖(외부 aidev `bin/run.sh` → `release.json` → `bin/gate.py`)이라 저장소 안에 합법적 수단이 없다. 필요한 사람 입력 6가지는 이전 회차와 동일(①관례 도입 여부 ②시작 버전·증가 단위 ③태그 형식 ④릴리즈 커밋 양식 ⑤노트 위치·양식·언어 ⑥GitHub Release 사용 여부).
- 보류 아이디어: 즐겨찾기 화면 ButtonSetting 의 `:actions` 누락 — **근거 정정 후 pending(재설계 필요)**. 과제서의 핵심 사실이 틀렸다: `FavoriteList.mapAppToCard:56` 이 `showMore:false` 를 박고 `SwiperList:showMore="item.showMore ?? props.showMore"` → `ButtonCard` 의 `IconBtn v-if="showMore"` 가 false 라 **더보기 버튼 자체가 렌더되지 않는다**(실제 마운트 스펙으로 확인: `button.btn-ico.ico-more` 0개, 카드 자체는 정상 렌더). 따라서 '빈 팝업이 뜬다' 는 재현되지 않고, `:actions`/`@action` 만 고치면 관측 가능한 동작 변화가 0 이라 운영자 지시(무효 변경 금지) 위반이다. 살리려면 `showMore` 도 켜야 하는데 이는 한 번도 보인 적 없는 UI 를 새로 노출하는 기능 추가이고, `git log` 가 `first commit` 하나뿐이라 의도를 확인할 수단이 없다(형제 `LibrarySearchList.vue:44` 도 `showMore:false` + ButtonSetting 없음으로 일관). 사람 입력 필요: 즐겨찾기 카드에 더보기를 노출할 것인가.
  - [수정 과제] 릴리즈 버전 결정 입력 복구 — pending(19회째, 진입 조건 미충족).
  - 릴리즈 절차 교착 자체를 사람에게 에스컬레이션 — pending; 수정 지점이 워크트리 밖이라 저장소 안에 합법적 수단이 없다.
  - `FavoriteList`/`SupportList`/`MyAgentList` 의 `onToggleFav` 3벌이 `createFavToggler` 를 쓰지 않는 중복 — rejected. `FavoriteList` 쪽은 위 근거대로 도달 자체가 불가능한 죽은 코드라 통합해도 관측 가능한 이득이 없다.
  - `onToggleFav` 3곳의 bare catch 가 즐겨찾기 실패를 무음 처리 — pending; 이번 `getHistoryList` 와 같은 형태이나 `FavoriteList` 쪽은 도달 불가, 나머지 둘은 별 아이콘이 원래 상태로 남아 '실패' 로 읽히므로 개선폭이 작다.
  - `NoticeDetail`/`LibraryDetail`/`RequestDetail` 의 `fetchBoardDetail` 이 `isSuccess(res)` 검사 없이 `res?.data?.body` 를 바로 읽고 catch 로 삼킴 — pending(신규). 3형제가 문자 단위로 같아 한 번에 고칠 수 있으나, 상세 화면의 빈 상태 기대 계약이 미확인.
  - 심플봇 두 팝업의 `tryUpdateAppIcon`/`tryUpdateLog` 가 `catch(e){}` 로 예외를 완전히 삼킴 — pending; 기대 계약 미확인이고 `simpleBotUpdateClose.spec.js` 가 현 동작을 고정하고 있다.
  - `loading.vue` 자동 해제 타이머 계약 확정 → `globalLoading` 참조 카운트 — pending; 타이머 계약이 미확정이라 지금 넣으면 스피너가 안 꺼지는 회귀 위험.
- 과제서: 차선 — A(릴리즈)는 지시대로 무변경. B(즐겨찾기 ButtonSetting)는 **근거가 지금 코드와 맞지 않아 기각**했다: 과제서가 인용한 바로 그 파일의 `mapAppToCard:56` 에 `showMore:false` 가 있어 더보기 버튼이 아예 렌더되지 않으므로 '빈 팝업' 은 재현되지 않고, `:actions` 만 붙이는 수정은 관측 가능한 변화가 0 이며 버튼까지 노출하는 것은 승인되지 않은 기능 추가다(실제 마운트 스펙으로 확인 후 그 탐색 스펙은 삭제). 그래서 과제서가 지정한 차선 후보(`getHistoryList` 의 bare catch)를 구현했고, 차선 후보가 '미확인' 이라던 기대 계약은 **같은 파일의 형제 `fetchAppList` 가 이미 확정해 둔 것**(업무 실패·비COM 예외는 토스트, COM 은 Alert 만)을 그대로 따르는 것으로 해결해 문구를 새로 발명하지 않았다.

## 2026-09-24
- 선택: 게시판 상세 3형제의 조회 실패가 '빈 자리표시자 화면' 으로 감춰지는 문제 수정 (가치 3 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `NoticeDetail.vue:57-66` / `LibraryDetail.vue:60-72` / `RequestDetail.vue:64-76` 의 `fetchBoardDetail` 은 `isSuccess(res)` 검사 없이 `res?.data?.body` 를 읽고 `catch` 로 예외를 통째로 삼켜, 조회 실패 시 `notice/library = null` 이 되면 화면 바인딩 computed 의 자리표시자('공지사항 제목' / '자료실 제목' / 'YYYY-MM-DD HH:MM:SS')가 대신 렌더돼 **실패가 '제목 없는 빈 게시글' 로 정상 렌더**됐다 — 인터셉터(`interceptors.js:228-236`)는 HTTP 200 + code 불일치 응답을 그대로 통과시키므로 이 경로는 실제로 화면까지 도달한다. 같은 저장소의 상세 조회 표준형 `SupportOcrDetail.getOcrs`(`:40-42` 업무 실패 `openAlert(res?.data?.message || 기본문구)` 후 return / `:55-56` `'COM'` 가드 뒤 `openAlert`)를 그대로 따라 문구·수단을 새로 발명하지 않고 세 화면에 같은 처리를 넣었다. 검증: HTTP 전송(axios adapter) 한 겹만 대역으로 바꾸고 실제 `interceptors.js` → 실제 `inf.board.detail.call`/`inf.library.detail.call` → 실제 컴포넌트 마운트(실제 `/notice|library|request/detail/:boardId` 메모리 라우터로 `boardId` 를 채움)를 통과시키는 신규 스펙 18건(3화면 × 6케이스)이 수정 전 9건 실패(Red: 업무 실패·기본문구·예외 세 케이스가 3화면 모두 Alert 0회) → 수정 후 전부 통과. 변이 4건으로 각 요소의 인과를 분리 증명: ①업무 실패 openAlert 제거 → 6건만 실패 ②catch openAlert 제거 → 3건만 실패 ③`'COM'` 가드 제거 → 인터셉터 Alert 과 겹쳐 3건만 실패 ④업무 실패 분기의 `null` 초기화 제거 → **18건 전부 통과(효과 없음이 증명돼 그 줄은 넣지 않았다)**. `npm test` 33파일 534테스트 통과(기준선 32/516), `npm run build:dev` 통과 후 `dist/` 삭제.
- 우선 과제(릴리즈): 진입 조건 미충족으로 무변경(20회째). 과제서 지시대로 재조사하지 않았고 버전 파일·태그·CHANGELOG·릴리즈 노트·릴리즈 커밋·원격 전송을 일체 하지 않았으며 `docs/RELEASE.md` 도 수정하지 않았다. 판정을 skipped/released 로 낮추지 않는다. 고칠 워크플로 파일이 저장소에 없고(`.github` 부재, `.gitlab-ci.yml` 은 `CI_COMMIT_TAG` 참조 0건인 브랜치 푸시 배포 전용) 실패 지점이 워크트리 밖(외부 aidev `bin/run.sh` → `release.json` → `bin/gate.py`)이라 저장소 안에 합법적 수단이 없다. 형제 저장소 `aiportal-front-admin` 의 `v0.1.x` 이식도 승인되지 않은 관례 이식이라 하지 않았다. **사람 입력 필요 4가지**: ①첫 릴리즈 버전(0.0.1 / 0.1.0 / 1.0.0) ②태그 형식과 주석 태그 여부 ③릴리즈 노트 위치·양식·언어 ④GitHub Release 사용 여부.
- 보류 아이디어: 목록 3형제(`NoticeList`/`LibraryList`/`RequestList`)의 조회 업무 실패가 조용히 빈 목록이 됨 — pending(다음 1순위 후보). 이번 상세 3형제 수정이 '상세는 openAlert' 라는 전례를 하나 더 굳혔으나, 목록은 '검색 결과 없음' 과 실패를 구분하는 기대 계약이 여전히 미확인이다.
  - [수정 과제] 릴리즈 버전 결정 입력 복구 — pending(20회째, 진입 조건 미충족).
  - 릴리즈 절차 교착 자체를 사람에게 에스컬레이션 — pending; 수정 지점이 워크트리 밖이라 저장소 안에 합법적 수단이 없다.
  - 심플봇 두 팝업의 `tryUpdateAppIcon`/`tryUpdateLog` 가 `catch(e){}` 로 예외를 완전히 삼킴 — pending(이번 차선 후보였음); 기대 계약 미확인이고 `simpleBotUpdateClose.spec.js` 가 현 동작을 고정하고 있다.
  - `RequestDetail` 이 `inf.library.detail` 을 호출한다(요청 게시판 전용 엔드포인트 부재) — pending(조사만). 이번에 그대로 두었고 테스트도 `/library/{id}` 가 나가는 현 동작을 고정했다. 백엔드가 별개 저장소라 이 저장소만으로 확정 불가.
  - `loading.vue` 자동 해제 타이머 계약 확정 → `globalLoading` 참조 카운트 — pending; 타이머 계약이 미확정이라 지금 넣으면 스피너가 안 꺼지는 회귀 위험.
- 과제서: 채택 — A(릴리즈)는 지시대로 재조사 없이 무변경, B 를 수용 기준 1~6 그대로 구현·검증했다. 다만 과제서가 "실패 시 기존 동작(`notice/library = null`)은 유지된다" 를 근거로 업무 실패 분기에 `null` 초기화를 넣으라는 형태로 읽힐 수 있었는데, 변이로 확인하니 `fetchBoardDetail` 은 `onMounted` 에서 한 번만 돌고 상태가 이미 `null` 이라 그 줄은 관측 가능한 효과가 0 이었다 — 운영자 지시(무효 변경 금지)에 따라 넣지 않고 형제 `SupportOcrDetail` 과 동일한 '알리고 return' 형태로 맞췄다(불변은 테스트로 고정). 과제서가 경고한 라우터 params 공백 함정은 첫 케이스에서 `/board/{id}`·`/library/{id}` 요청이 실제로 나갔음을 단정해 막았다.

## 2026-09-24
- 선택: 게시판 목록 3형제의 조회 실패가 '없습니다' 문구로 감춰지던 문제 수정 (가치 3 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `NoticeList.vue:60` / `LibraryList.vue:58` / `RequestList.vue:132` 의 `getBoardList` 는 `isSuccess(res)` 검사가 없어 HTTP 200 으로 도착한 업무 실패를 그대로 통과시키고(`interceptors.js:228-236` 은 status 200 응답을 거르지 않는다), `catch` 는 `'COM'` 가드 뒤에서 목록만 비웠다 — 그 결과 조회 실패가 Nodata 의 '공지사항이 없습니다'(`NoticeList:133`) / '등록된 자료가 없습니다'(`LibraryList:131`) / '조회 내용이 없습니다'(`RequestList:261`) 라는 **사실 진술**과 구분되지 않았다. 문구·수단은 같은 저장소의 목록 조회 표준형 `Sidemenu.fetchAppList`(`:208-212` 업무 실패 `toast(res?.data?.message || 기본문구)` 후 return / `:233-236` `if (e !== 'COM')` 토스트)를 그대로 따라 새로 발명하지 않았다(상세=`openAlert` / 목록=`toast` 라는 확정된 구분을 유지). 검증: HTTP 전송(axios adapter) 한 겹만 대역으로 바꾸고 실제 `interceptors.js` → 실제 `inf.board.list.call`/`inf.library.list.call` → 실제 컴포넌트 마운트(메모리 라우터로 `/notice`·`/library/list/003`·`/request` 를 실제로 push)를 통과시키는 신규 스펙 18건(3화면 × 6케이스)이 수정 전 12건 실패(Red: 업무 실패 메시지·기본 문구·예외·재조회 네 케이스가 3화면 모두 토스트 0건) → 수정 후 전부 통과. 변이 4건으로 각 줄의 인과를 분리 증명: ①업무 실패 토스트 제거 → 9건만 실패 ②catch 토스트 제거 → 3건만 실패 ③`'COM'` 가드 제거 → 인터셉터 Alert 과 겹쳐 3건만 실패 ④**업무 실패 분기의 `return` 만 제거 → 3건만 실패**(무효 변경이 아님을 증명). `npm test` 34파일 552테스트 통과(기준선 33/534), `npm run build:dev` 통과 후 `dist/` 삭제.
- 우선 과제(릴리즈): 진입 조건 미충족으로 무변경(22회째). 회차 노트의 정찰 지시("구현자는 재조사하지 말고 손대지 말 것")대로 재조사하지 않았고, 버전 파일·태그·CHANGELOG·릴리즈 노트·릴리즈 커밋·원격 전송을 일체 하지 않았으며 `docs/RELEASE.md` 도 수정하지 않았다. 판정을 skipped/released 로 낮추지 않는다. 고칠 워크플로 파일이 저장소에 없고 실패 지점이 워크트리 밖(외부 aidev `bin/run.sh` → `release.json` → `bin/gate.py`)이라 저장소 안에 합법적 수단이 없다. 필요한 사람 입력 4가지는 직전 회차와 동일(①첫 릴리즈 버전 ②태그 형식·주석 태그 여부 ③릴리즈 노트 위치·양식·언어 ④GitHub Release 사용 여부).
- 보류 아이디어: RequestList 의 검색어가 SearchBox 에서 전달되지 않아 요청 게시판 검색이 무동작 — pending(신규, 다음 1순위 후보). `SearchBox.vue` 에 `modelValue` prop 도 input 바인딩도 없어 `v-model="searchKey"` 가 아무것도 연결하지 않는데, Notice/Library 는 `getElementById('searchKey')` 로 우회하는 반면 RequestList 는 `searchId` 를 넘기지 않아 `searchKey` 가 영영 빈 문자열이다. 읽어서 확인했을 뿐 마운트로 재현하지는 않았다.
  - isSuccess 검사가 없는 잔여 3곳(ShareList / HeaderAlarm / PopWidgetSetting) — pending(신규). grep 으로 이 3곳만 남은 것을 확인했으나 보조 UI 의 기대 계약이 미확인이고 서로 닮지 않아 묶기 어렵다.
  - [수정 과제] 릴리즈 버전 결정 입력 복구 / 릴리즈 교착의 사람 에스컬레이션 — pending(22회째, 저장소 안에 합법적 수단 없음).
  - 심플봇 두 팝업의 `tryUpdateAppIcon`/`tryUpdateLog` 가 `catch(e){}` 로 예외를 완전히 삼킴 — pending; 기대 계약 미확인이고 `simpleBotUpdateClose.spec.js` 가 현 동작을 고정하고 있다.
  - `loading.vue` 자동 해제 타이머 계약 확정 → `globalLoading` 참조 카운트 — pending; 타이머 계약이 미확정이라 지금 넣으면 스피너가 안 꺼지는 회귀 위험.
- 과제서: 채택 — 러너는 정찰을 실패로 판정했으나(05:13) 회차 노트의 정찰 노트에 후보와 근거가 남아 있어 그대로 골랐고, 경고 3가지를 모두 지켰다: ①토스트를 `'COM'` 가드 **뒤**에 뒀다 ②업무 실패 분기에 목록 초기화를 넣지 않고 표준형대로 `return` 만 뒀으며, 그 `return` 이 무효 변경이 아님을 변이 ④로 증명했다 ③LibraryList 는 `/library/list/003` 을 실제로 push 하고 RequestList 는 `writeUser` 로 `creatorId` 를 채워, 요청이 실제로 나갔음을 첫 케이스에서 단정했다. 정찰이 '미확인' 이라던 RequestList 의 `isFirst`/`searchCount`/`excludedCount` 갱신 건너뜀은 **화면에 관측되지 않아** 케이스로 고정하지 않았다(실패 응답의 카운트를 기록하지 않는 것이 옳은 방향이기도 하다).

## 2026-09-24
- 선택: 요청 게시판(RequestList)의 검색이 무동작 — 검색어가 요청에 실리지 않던 문제 수정 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `SearchBox.vue` 는 `modelValue` prop 도 `input` 의 `:value`/`@input` 바인딩도 없어(`:3-20,30`) 호출부의 `v-model` 이 아무것도 연결하지 않는다 — 형제 4화면(`NoticeList:114`, `LibraryList:112`, `ShareList`, `PopGlobalSearch`)은 모두 `searchId` 를 넘기고 `onSearchClick` 에서 `document.getElementById(...)` 로 DOM 값을 직접 읽어 우회하는데 **`RequestList` 만** `searchId` 를 넘기지 않고(`:226`) `onSearchClick`(`:85-96`)이 `searchKey.value` 를 그대로 써, `searchKey` 가 영영 빈 문자열이고 `getBoardList` 의 `if (searchKey.value && ...)`(`:123-125`)가 never-true 였다(무엇을 입력하고 검색을 눌러도 `searchKey` 파라미터 없이 전체 목록이 다시 온다). 형제와 문자 단위로 같은 두 줄(`getElementById` → `trim()`)을 `onSearchClick` 첫머리에 넣고 `searchId="searchKey"` 를 넘겼으며, 아무것도 연결하지 않던 `v-model="searchKey"` 는 제거했다(동작 변화 0 인 정리). 주석 처리된 글자수 검증 블록과 `isFirst`/`searchCount`/`excludedCount` 카운트 로직, `inf.library.*` 엔드포인트, 공유 컴포넌트 `SearchBox.vue` 는 지시대로 손대지 않았다. 검증: HTTP 전송(axios adapter) 한 겹만 대역으로 바꾸고 실제 `interceptors.js` → 실제 `inf.library.list.call` → 실제 컴포넌트 마운트(`attachTo: document.body`, 메모리 라우터로 `/request` push, `writeUser` 로 `creatorId` 충족)를 통과시키는 신규 스펙 8건이 수정 전 5건 실패(Red: 버튼 클릭·Enter·trim·재검색·페이지 리셋 후 검색 모두 `searchKey=null`) → 수정 후 전부 통과. 나머지 3건(최초 조회 / 공백만 / 빈 값에는 파라미터 미전송)은 수정 전후 모두 통과해 `trim()` 계약이 안 깨졌음을 고정한다. 변이 2건으로 인과를 분리 증명: ①`getElementById` 두 줄 제거 → 같은 5건만 실패 ②`searchId="searchKey"` 제거 → 같은 5건만 실패(둘 다 형제 스펙 18건은 그대로 통과). `npm test` 35파일 560테스트 통과(기준선 34/552), `npm run build:dev` 통과 후 `dist/` 삭제.
- 우선 과제(릴리즈): 진입 조건 미충족으로 무변경(23회째). 과제서 지시("재조사하지 말고 무변경으로 두라")대로 재조사하지 않았고, 버전 파일·태그·CHANGELOG·릴리즈 노트·릴리즈 커밋·원격 전송을 일체 하지 않았으며 `docs/RELEASE.md` 도 수정하지 않았다. 판정을 skipped/released 로 낮추지 않는다. 고칠 워크플로 파일이 저장소에 없고(`.github` 부재, `.gitlab-ci.yml` 은 `CI_COMMIT_TAG` 참조 0건인 브랜치 푸시 배포 전용) 실패 지점이 워크트리 밖(외부 aidev `bin/run.sh` → `release.json` → `bin/gate.py`)이라 저장소 안에 합법적 수단이 없다. 필요한 사람 입력 4가지는 직전 회차와 동일(①첫 릴리즈 버전 0.0.1/0.1.0/1.0.0 ②태그 형식·주석 태그 여부 ③릴리즈 노트 위치·양식·언어 ④GitHub Release 사용 여부).
- 보류 아이디어: SearchBox 를 쓰는 나머지 5화면(SupportOcr/Stt/Img List, ChatStorageList, PopPromptSelect)의 검색 배선 실태 확인 — pending(다음 1순위 후보). 이번에도 열어 보지 않아 `onSearchClick` 이 DOM 을 읽는지 미확인이다.
  - isSuccess 검사가 없는 잔여 3곳(ShareList / HeaderAlarm / PopWidgetSetting) — pending; 보조 UI 의 기대 계약이 미확인이고 세 화면이 서로 닮지 않아 묶기 어렵다.
  - [수정 과제] 릴리즈 버전 결정 입력 복구 / 릴리즈 교착의 사람 에스컬레이션 — pending(23회째, 저장소 안에 합법적 수단 없음).
  - 심플봇 두 팝업의 `tryUpdateAppIcon`/`tryUpdateLog` 가 `catch(e){}` 로 예외를 완전히 삼킴 — pending; 기대 계약 미확인이고 `simpleBotUpdateClose.spec.js` 가 현 동작을 고정하고 있다.
  - `loading.vue` 자동 해제 타이머 계약 확정 → `globalLoading` 참조 카운트 — pending; 타이머 계약이 미확정이라 지금 넣으면 스피너가 안 꺼지는 회귀 위험.
- 과제서: 채택 — A(릴리즈)는 지시대로 재조사 없이 무변경, B 를 수용 기준 1~6 그대로 구현·검증했다. 과제서가 경고한 함정을 모두 지켰다: ①`attachTo: document.body` 로 마운트하고 케이스마다 `unmount` 해 같은 id `searchKey` 를 쓰는 형제 스펙과 섞이지 않게 했다 ②`SearchBox.vue` 는 건드리지 않고 호출부만 형제와 같게 맞췄다 ③무효 변경 금지 지적에 따라 변이 2건으로 넣은 줄이 실제로 필요함을 증명했다(`v-model` 제거는 동작 변화 0 임을 알고 하는 정리라 변이 대상에서 제외). 과제서가 '선택' 으로 남긴 `v-model` 은 제거 쪽을 택했다.

## 2026-09-24
- 선택: 프롬프트 선택 팝업(PopPromptSelect)의 검색이 배경 게시판 화면의 검색창을 읽던 문제 수정 — 중복 DOM id `searchKey` (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `SearchBox.vue` 는 `modelValue` prop 도 input 바인딩도 없어 호출부가 `searchId` 를 넘기고 `document.getElementById('<id>')` 로 값을 직접 읽는 것이 이 저장소의 관례인데, 전역 팝업 `PopPromptSelect` 만 게시판 4화면(`NoticeList:114`/`LibraryList:112`/`ShareList:93`/`RequestList:228`)이 이미 쓰는 일반 id `searchKey` 를 재사용했다(`:107` reader, `:148` 템플릿). 이 팝업은 `App.vue:49-62` 의 Teleport 안에서 열려 배경 route 화면과 DOM 을 공유하므로 `getElementById` 가 문서 순서상 먼저 오는 `#app` 쪽 게시판 input 을 돌려줬다. 형제 팝업·목록이 모두 고유 id(`totalSearchKey`/`ocrSearchKey`/`sttSearchKey`/`imgSearchKey`/`chatSearchKey`)를 쓰고 있어 이 한 곳만 비대칭이었고, 공유 컴포넌트를 넓히지 않고 호출부를 형제와 같게 맞추는 확정된 방향대로 두 줄만 `promptSearchKey` 로 바꿨다. 검증: HTTP 전송(axios adapter) 한 겹만 대역으로 바꾸고 실제 `interceptors.js` → 실제 `inf.app.getPrompt.call` → 실제 컴포넌트 마운트(두 wrapper 모두 `attachTo: document.body`, 게시판을 먼저 마운트해 현실의 `#app → teleport` 문서 순서를 재현, 메모리 라우터로 `/notice` push, `writeUser` 로 `user_id` 충족)를 통과시키는 신규 스펙 7건이 **수정 전 2건 실패(Red: 팝업에 `템플릿` 을 입력해도 나가는 `key_word` 가 `''`, 팝업이 비어 있고 게시판에 `공지` 가 남아 있으면 `key_word` 가 `공지`)** → 수정 후 전부 통과. 나머지 5건(팝업 단독의 최초 조회·검색어 전달·trim·공백만 입력·2페이지에서 검색 시 pageNum 1 리셋)은 수정 전후 모두 통과해 범위가 좁음을 고정한다. 변이 2건으로 두 줄이 모두 필요함을 분리 증명: ①템플릿 `searchId` 만 되돌림 → 4건 실패 ②`getElementById` 만 되돌림 → 5건 실패(둘 다 무효 변경이 아님). `npm test` 36파일 567테스트 통과(기준선 35/560 을 수정 전에 실제로 돌려 확정), `npm run build:dev` 통과 후 `dist/` 삭제.
- 우선 과제(릴리즈): 진입 조건 미충족으로 무변경(24회째). 과제서 지시("재조사하지 말고 손대지 말 것")대로 재조사하지 않았고, 버전 파일·태그·CHANGELOG·릴리즈 노트·릴리즈 커밋·원격 전송을 일체 하지 않았으며 `docs/RELEASE.md` 도 수정하지 않았다. 판정을 skipped/released 로 낮추지 않는다. 고칠 워크플로 파일이 저장소에 없고(`.github` 부재, `.gitlab-ci.yml` 은 `CI_COMMIT_TAG` 참조 0건인 브랜치 배포 전용) 실패 지점이 워크트리 밖(외부 aidev `bin/run.sh` → `release.json` → `bin/gate.py`)이라 저장소 안에 합법적 수단이 없다. 필요한 사람 입력 4가지는 직전 회차와 동일(①첫 릴리즈 버전 0.0.1/0.1.0/1.0.0 ②태그 형식·주석 태그 여부 ③릴리즈 노트 위치·양식·언어 ④GitHub Release 사용 여부).
- 보류 아이디어: isSuccess 검사가 없는 잔여 3곳(ShareList / HeaderAlarm / PopWidgetSetting) — pending(이번 차선 후보였음). ShareList 만 목록 표준형(toast)에 맞출 수 있고 보조 UI 2곳은 기대 계약 미확인이라 묶지 말 것.
  - [수정 과제] 릴리즈 버전 결정 입력 복구 / 릴리즈 교착의 사람 에스컬레이션 — pending(24회째, 저장소 안에 합법적 수단 없음).
  - 심플봇 두 팝업의 `tryUpdateAppIcon`/`tryUpdateLog` 가 `catch(e){}` 로 예외를 완전히 삼킴 — pending; 기대 계약 미확인이고 `simpleBotUpdateClose.spec.js` 가 현 동작을 고정하고 있다.
  - 아무것도 연결하지 않는 `v-model="searchKey"` 잔재 4곳(NoticeList:114 / LibraryList:112 / ShareList:93 / PopGlobalSearch:110) 정리 — pending; 동작 변화 0 이라 단독으로는 무효 변경. SearchBox 에 modelValue 계약을 넣는 결정이 날 때 함께.
  - `loading.vue` 자동 해제 타이머 계약 확정 → `globalLoading` 참조 카운트 — pending; 타이머 계약이 미확정이라 지금 넣으면 스피너가 안 꺼지는 회귀 위험.
- 과제서: 채택 — A(릴리즈)는 지시대로 재조사 없이 무변경, B 를 지정된 두 줄만 고쳐 구현했고 `SearchBox.vue` 와 게시판 4화면의 id 는 손대지 않았다. 다만 과제서의 사실 두 가지가 실제와 달라 하네스를 바로잡았다: ①`writeUser` 는 `normalizeUserRecord` 를 거치므로 입력 키가 `user_id` 가 아니라 `userId` 다(과제서대로 `{user_id:'tester'}` 를 넣으면 `user_id` 가 빈 문자열이 되어 요청이 아예 안 나간다) ②열림 watch 는 값이 **바뀔 때만** 돌아서 `modelValue: true` 로 마운트하면 최초 조회가 나가지 않는다 — 닫은 채 마운트하고 `setProps` 로 열어야 과제서가 말한 '최초 조회 1건' 이 재현된다. 수용 기준 5(변이)의 예측('템플릿 `searchId` 만 되돌리면 1)·2) 만 실패')도 실제로는 4건 실패였다: 한쪽만 되돌리면 읽는 id 와 심는 id 가 어긋나 팝업이 자기 입력란을 아예 못 찾기 때문이다. 인과 분리 목적(두 줄 모두 load-bearing)은 그대로 달성됐다.

## 2026-09-24
- 선택: 앱 공유 리스트(ShareList)의 조회 실패가 '없습니다' 문구로 감춰지던 문제 수정 (가치 3 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `src/views/Share/ShareList.vue:43-69` 의 `getShareList` 는 `isSuccess(res)` 검사가 없어 HTTP 200 으로 도착한 업무 실패를 그대로 통과시키고(`interceptors.js:228-236` 은 status 200 응답을 거르지 않는다), `catch` 는 `'COM'` 가드 뒤에서 목록만 비웠다 — 그 결과 조회 실패가 Nodata 의 '공유된 앱 리스트가 없습니다'(`:121`) 라는 사실 진술과 구분되지 않았고, 성공 후의 재조회가 실패하면 이미 받은 목록까지 빈 목록으로 덮였다. 문구·수단은 목록 조회 표준형(`Sidemenu.fetchAppList:208-212` 업무 실패 `toast(res?.data?.message || 기본문구)` 후 return / `:233-236` `if (e !== 'COM')` 토스트, 게시판 목록 3형제 ce28031)을 그대로 따라 새로 발명하지 않았다(상세=`openAlert` / 목록=`toast` 구분 유지). 보류 아이디어가 경고한 대로 기대 계약이 미확인인 보조 UI 2곳(`HeaderAlarm` / `PopWidgetSetting`)은 묶지 않고 손대지 않았다. 검증: HTTP 전송(axios adapter) 한 겹만 대역으로 바꾸고 실제 `interceptors.js` → 실제 `inf.app.shareList.call`(`/app-management/share`) → 실제 컴포넌트 마운트(`attachTo: document.body`, 메모리 라우터로 `/share/list` push, `writeUser({userId:'tester'})` 로 `userId` 파라미터 충족)를 통과시키는 신규 스펙 6건이 수정 전 4건 실패(Red: 업무 실패 메시지·기본 문구·예외·성공 후 재조회 실패가 모두 토스트 0건) → 수정 후 전부 통과. 변이 4건으로 각 줄의 인과를 분리 증명: ①업무 실패 토스트 제거 → 3건만 실패 ②**업무 실패 분기의 `return` 만 제거 → 1건 실패**(무효 변경이 아님) ③catch 토스트 제거 → 1건 실패 ④기존 `'COM'` 가드 제거 → 인터셉터 Alert 과 겹쳐 1건 실패. `npm test` 37파일 573테스트 통과(기준선 36/567 을 수정 전에 실제로 돌려 확정), `npm run build:dev` 통과 후 `dist/` 삭제.
- 우선 과제(릴리즈): 진입 조건 미충족으로 무변경(25회째). 과제서가 재실측으로 확정한 대로 이 저장소에는 따라 할 릴리즈 관례가 없어(태그 0개, 릴리즈 커밋 0건, package.json/lock 모두 `0.0.0` 으로 최초 커밋 이후 무변경, CHANGELOG·릴리즈 노트·`.github` 부재, `.gitlab-ci.yml` 은 `CI_COMMIT_TAG` 참조 0건인 브랜치 배포 전용) 버전 파일·태그·CHANGELOG·릴리즈 노트·릴리즈 커밋을 일체 만들지 않았고 `docs/RELEASE.md` 도 수정하지 않았다. 판정을 skipped/released 로 낮추지 않는다. 러너 지시("워크플로 파일과 실패한 단계의 스크립트·테스트를 읽고 원인을 고치세요")는 이 저장소에 워크플로 파일이 존재하지 않고 실패 지점이 워크트리 밖(외부 aidev `bin/run.sh` → `release.json` → `bin/gate.py`)이라 수행 대상이 없다 — 게이트를 느슨하게 만드는 것은 금지이므로 `version` 필드 삭제 같은 우회도 하지 않았다. 필요한 사람 입력 4가지는 직전 회차와 동일(①첫 릴리즈 버전 0.0.1/0.1.0/1.0.0 ②태그 형식·주석 태그 여부 ③릴리즈 노트 위치·양식·언어 ④GitHub Release 사용 여부).
- 보류 아이디어: isSuccess 검사가 없는 잔여 2곳(HeaderAlarm / PopWidgetSetting) — pending; 보조 UI 라 '조용한 빈 목록이 의도인가' 라는 기대 계약이 미확인이고 서로 닮지 않아 묶지 말 것.
  - [수정 과제] 릴리즈 버전 결정 입력 복구 / 릴리즈 교착의 사람 에스컬레이션 — pending(25회째, 저장소 안에 합법적 수단 없음).
  - 심플봇 두 팝업의 `tryUpdateAppIcon`/`tryUpdateLog` 가 `catch(e){}` 로 예외를 완전히 삼킴 — pending; 기대 계약 미확인이고 `simpleBotUpdateClose.spec.js` 가 현 동작을 고정하고 있다.
  - 아무것도 연결하지 않는 `v-model="searchKey"` 잔재 4곳(NoticeList:114 / LibraryList:112 / ShareList:93 / PopGlobalSearch:110) 정리 — pending; 동작 변화 0 이라 단독으로는 무효 변경. SearchBox 에 modelValue 계약을 넣는 결정이 날 때 함께.
  - `loading.vue` 자동 해제 타이머 계약 확정 → `globalLoading` 참조 카운트 — pending; 타이머 계약이 미확정이라 지금 넣으면 스피너가 안 꺼지는 회귀 위험.
- 과제서: 채택 — A(릴리즈)는 과제서 결론대로 무변경(재조사 없이, 위 근거는 과제서 재실측 결과를 인용). B 는 과제서가 남긴 pending 중 '묶지 말고 ShareList 만 떼어낼 것' 이라는 지시를 그대로 따라 ShareList 한 화면만 표준형에 맞췄다. 과제서가 예상하지 않았던 사실 하나를 추가로 확인해 케이스로 고정했다: ShareList 는 업무 실패 시 `res.data.body.appList` 가 없어 목록이 `[]` 로 덮이므로, 게시판 3형제와 달리 **성공 후 재조회 실패에서 이미 렌더된 행이 사라지는** 관측 가능한 손실이 있었다(변이 ②가 이 한 건만 잡아낸다).

## 2026-09-24
- 선택: 위젯 설정 팝업(PopWidgetSetting)의 저장 실패가 성공처럼 닫히고 옛 값을 다시 그리던 문제 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `src/components/common/popup/Global/PopWidgetSetting.vue` 의 `saveWidgetSetting`(:142-150)은 `isSuccess` 검사가 없어 HTTP 200 으로 도착한 업무 실패에도 `bus.emit(BUS_EVENT.WIDGET_CHANGED)` 를 그대로 보내고(`interceptors.js:228-236` 은 status 200 응답을 거르지 않는다), `catch (e) {}` 로 예외를 완전히 삼켰다. 게다가 `onClick`(:20-24)이 `saveWidgetSetting()` 을 기다리지 않고 `close()` 를 먼저 불러 저장 결과와 무관하게 팝업이 닫혔다 — 그 결과 저장 실패가 성공과 전혀 구분되지 않고, `Home.vue:448` 이 `WIDGET_CHANGED` 를 받아 **저장되지 않은 옛 위젯 구성을 성공한 것처럼 다시 렌더**했다(관측 가능한 손실). 문구·수단은 이 저장소의 저장(팝업) 실패 표준형(`PopSimpleBotUpdate.onCreate:663-681` — 실패 시 토스트 + 닫지 않음 + 부모에 알리지 않음, `catch` 는 `if (e == 'COM') return` 뒤 토스트)을 그대로 따랐고, 토스트 문구는 같은 저장소의 동작 실패 어법(`Sidemenu:534` '보관함 저장에 실패했습니다.')에 맞춰 새로 발명하지 않았다. 검증: HTTP 전송(axios adapter) 한 겹만 대역으로 바꾸고 실제 `interceptors.js` → 실제 `inf.widget.personal.call`(`PUT /widget/personal?<qs>`) → 실제 컴포넌트 마운트(`attachTo: document.body`, `vuedraggable` 실제 마운트, `writeUser({userId:'tester'})` 로 조회 파라미터 충족)를 통과시키는 신규 스펙 6건이 수정 전 5건 실패(Red: 업무 실패 토스트 0건 / 업무 실패인데 닫힘 / 예외 완전 무음 / 'COM' 인데 닫힘 / 응답 전에 이미 닫힘) → 수정 후 전부 통과. 변이 7건으로 각 줄의 인과를 분리 증명: ①업무 실패 블록 제거 → 2건 실패 ②**업무 실패 분기의 `return` 만 제거 → 2건 실패**(무효 변경 아님) ③catch 토스트 제거 → 1건 실패 ④`'COM'` 가드 제거 → 인터셉터 Alert 과 겹쳐 1건 실패 ⑤`onClick` 을 원래 모양(미대기 호출 후 `close()`)으로 되돌림 → 5건 실패 ⑥**`onClick` 의 `await` 만 제거 → 6건 전부 통과(효과 0 이 증명돼 `await`/`async` 를 넣지 않고 `close()` 삭제만 남겼다)** ⑦성공 경로의 `close()` 만 제거 → 2건 실패. `npm test` 38파일 579테스트 통과(기준선 37/573 을 수정 전에 실제로 돌려 확정), `npm run build:dev` 통과 후 `dist/` 삭제.
- 우선 과제(릴리즈): 진입 조건 미충족으로 무변경(26회째). 회차 노트의 정찰 지시("A(릴리즈)는 26회째 무변경이 정답이고 이번 회차에 6개 항목을 직접 재실측해 확정했다 — 구현자는 재조사도 하지 말 것")대로 재조사하지 않았고, 버전 파일·태그·CHANGELOG·릴리즈 노트·릴리즈 커밋·원격 전송을 일체 하지 않았으며 `docs/RELEASE.md` 도 수정하지 않았다. 판정을 skipped/released 로 낮추지 않는다. 러너 지시("워크플로 파일과 실패한 단계의 스크립트·테스트를 읽고 원인을 고치세요")는 이 저장소에 워크플로 파일이 없고(`.github` 부재, `.gitlab-ci.yml` 은 `CI_COMMIT_TAG` 참조 0건인 브랜치 배포 전용) 실패 지점이 워크트리 밖(외부 aidev `bin/run.sh` → `release.json` → `bin/gate.py`)이라 수행 대상이 없다 — 게이트를 느슨하게 만드는 우회(`version` 필드 삭제 등)는 금지이므로 하지 않았다. 필요한 사람 입력 4가지는 직전 회차와 동일(①첫 릴리즈 버전 0.0.1/0.1.0/1.0.0 ②태그 형식·주석 태그 여부 ③릴리즈 노트 위치·양식·언어 ④GitHub Release 사용 여부).
- 보류 아이디어: `MainLayout.vue:107` 의 `@click="onWidgetSave"` 가 미정의 핸들러 — pending(신규, 이번에 분리). `PopWidgetSetting` 의 `emit('click')` 도 이제 저장 성공 여부와 무관하게 먼저 나가므로, 핸들러를 정의할지 `emit('click')` 을 없앨지 결정이 필요하다. 지금은 양쪽 다 동작 변화 0 이라 단독으로는 무효 변경.
  - isSuccess 검사가 없는 잔여 1곳(`HeaderAlarm.getList:37-67`) — pending; 보조 UI 라 '조용한 빈 목록이 의도인가' 라는 기대 계약이 미확인이다.
  - [수정 과제] 릴리즈 버전 결정 입력 복구 / 릴리즈 교착의 사람 에스컬레이션 — pending(26회째, 저장소 안에 합법적 수단 없음).
  - 심플봇 두 팝업의 `tryUpdateAppIcon`/`tryUpdateLog` 가 `catch(e){}` 로 예외를 완전히 삼킴 — pending; 기대 계약 미확인이고 `simpleBotUpdateClose.spec.js` 가 현 동작을 고정하고 있다.
  - `loading.vue` 자동 해제 타이머 계약 확정 → `globalLoading` 참조 카운트 — pending; 타이머 계약이 미확정이라 지금 넣으면 스피너가 안 꺼지는 회귀 위험.
- 과제서: 채택 — 러너는 정찰을 실패로 판정했으나(11:00) 회차 노트의 정찰 노트에 후보와 근거가 남아 있어 그대로 골랐고, 경고 3가지를 모두 지켰다: ①`emit('click')` 은 건드리지 않았다(`MainLayout.vue:107` 의 미정의 핸들러 건은 별도 아이디어로 분리) ②토스트를 `'COM'` 가드 **뒤**에 뒀다(변이 ④로 겹침이 실제로 잡힘을 확인) ③`bus` 싱글턴은 케이스마다 `bus.off` + 카운터 리셋했다. 정찰이 '확신 없음' 으로 남긴 세 가지를 실측으로 해소했다: ①`vuedraggable` 은 jsdom 에서 그대로 마운트된다(성공 케이스가 Red 단계에서 이미 통과) ②`npm test` 기준선 37파일/573테스트를 수정 전에 실제로 돌려 확정했다 ③`Custom.vue` 의 `@primary` 는 `Custom` 이 `primary` 를 emit 하지 않아 죽은 배선이므로, 실제 저장 경로인 푸터 `MainBtns @click2` 버튼을 클릭해 구동했다. 정찰이 예상하지 않은 사실 하나를 추가로 확정했다: `onClick` 에 `await` 를 넣는 것은 변이 ⑥에서 효과 0 으로 증명돼(핸들러의 반환 프로미스를 아무도 기다리지 않는다) 넣지 않았고, `close()` 를 성공 경로로 옮기는 것만으로 다섯 케이스가 모두 고쳐진다.


[← 대시보드](https://hkjang.github.io/aidev/) · [교훈 모음](https://hkjang.github.io/aidev/lessons/)

{% endraw %}
