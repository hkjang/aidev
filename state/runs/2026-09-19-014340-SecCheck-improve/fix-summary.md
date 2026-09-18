# 수리 요약 (시도 2) — 커밋 693b733

- **문제 1 (Settings.tsx)**: 테스트 메일 주소 칸+버튼을 `.search-box` 로 감싸 `.search-box svg` 절대배치가 버튼의 `<Send>` 아이콘을 입력칸 왼쪽으로 끌어냈고 `.search-box .input` 이 35px 빈 여백을 만들었음 → 지적이 맞음. 래퍼를 기존 CSP-safe 유틸 `data-sx="sx-008"`(flex, gap 8px, 세로 가운데)로 바꾸고, flex 축소로 버튼 글자가 두 줄로 꺾이지 않게 `whiteSpace: nowrap` 을 줌.
- **문제 2 (가이드 그림)**: 실제로 재촬영해 보니 `admin-settings-mail.png` 는 뷰포트(900px) 캡처라 테스트 메일 줄이 접힌 부분 아래에 있어 옛 그림과 바이트 단위로 같았음 — 캡션이 말하는 "테스트 메일, 발송 기록" 이 애초에 그림에 없던 상태. `scripts/capture_all.js` 의 메일 탭만 `fullPage: true` 로 찍게 해(precheck 는 너비 1440 만 검사하고 full-page 를 허용) 새 주소 칸·버튼·발송 기록이 보이는 1440×1513 그림으로 교체했고, 아이콘이 버튼 안에 있는 것을 캡처에서 확인.
- 같은 그림을 싣는 features.md 때문에 ADMIN_GUIDE.pdf 외에 seccheck_features_guide.pdf·seccheck_complete_manual.pdf 도 `scripts/build_docs_pdf.sh` 로 재생성(precheck PDF 신선도 검사 요구).
- 검증: `bash scripts/precheck.sh` 통과(gofmt·vet·go test·tsc·vitest 11개·vite build·그림 너비·PDF 신선도·gitleaks). 캡처는 일회용 Postgres+서버 컨테이너(127.0.0.1:18080)에서 찍고 컨테이너는 정리함.
