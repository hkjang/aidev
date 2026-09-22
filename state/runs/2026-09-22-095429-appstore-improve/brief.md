- 과제: 즐겨찾기의 기기 간 동기화 오안내를 사용자 가이드·PDF·웹 요약에서 정정 (가치 3 / 위험 1 / 작업량 S)
- 왜: 현재 FavoritesProvider는 로그인 상태를 읽지 않고 appstore.favorites를 localStorage에만 저장하지만 사용자 가이드 4.4는 로그인하면 다른 기기에서도 같게 보인다고 안내한다. 저장 범위를 정확히 설명하면 기기를 바꾼 사용자가 즐겨찾기 누락을 장애로 오해하는 일을 줄인다.
- 수용 기준: 1) USER_GUIDE 3.6과 4.4가 모두 로그인 여부와 관계없이 현재 브라우저에 저장하며 다른 브라우저·기기로 자동 동기화되지 않는다고 설명하고, 브라우저 데이터 삭제 시 사라진다는 안내를 유지한다. 2) docs/guides/user/index.html의 같은 안내도 일치하고 docs/USER_GUIDE.pdf에 수정 문구가 반영된다. 기존 표지 v2.11.4·캡처·링크는 유지한다. 3) 문서 계약 검사 성공과 PDF 실제 텍스트 추출로 기존 동기화 권유가 사라졌음을 증명한다. 새 제품 테스트는 필요 없으며 PDF 해당 페이지의 한글·줄바꿈·이미지도 육안 확인한다.
- 건드릴 파일: docs/USER_GUIDE.md:3.6 즐겨찾기 및 4.4 자주 쓰는 앱 모아 두기 — 두 설명 수정; docs/USER_GUIDE.pdf — 같은 Markdown에서 재생성; docs/guides/user/index.html:59의 즐겨찾기 설명 — 로그인 여부와 무관함 명시. 근거로 읽은 web/src/features/apps/favorites.tsx:FavoritesProvider/toggle 및 web/src/pages/public-pages.tsx:AppsPage는 수정하지 않는다.
- 검증 명령: 저장소 루트에서 `./scripts/check-docs.sh`, `./scripts/check-env-contract.sh`, `git diff --check`. 아래 PDF 생성·검증 명령 참조.
- 위험과 피할 것: auth·session·migrations·.github/workflows·제품 코드·서버 /me/favorites 연결은 범위 밖. 즐겨찾기 페이지 개수/100개 제한은 별도 과제다. make screenshots/make guides/make build를 실행하거나 캡처·버전·ADMIN_GUIDE·internal/webui/dist를 함께 갱신하지 않는다. guides/appstore/USER_GUIDE.pdf는 외부 수집본이며 docs 정본과 현재도 바이트가 다르다. 이번에는 내장 가이드 전체 동기화를 하지 말고 별도 후보로 기록한다. PDF 변환기는 기존 출력이 있으면 Chrome 실패를 놓칠 가능성이 있으므로 반드시 새 임시 출력에 생성·검사한 뒤 교체한다. 단순 check-docs 성공은 PDF 내용 일치를 증명하지 않는다.
- 차선 후보: E2E public config override 기본값 덮어쓰기 순서 정리 — docs가 이미 정정됐거나 PDF 도구를 10분 안에 사용할 수 없을 때만 선택. web/e2e/mock-api.ts:installMockApi에서 기본값 뒤에 options.config를 펼치고, core.spec.ts의 기존 로그인 전용 설치 테스트가 config 옵션을 실제 사용하도록 해 false와 siteName 덮어쓰기를 검증한다. 별도 auth/session 응답 route는 유지한다. `npm --prefix web ci --no-audit --no-fund`, `npm --prefix web test`, `npm --prefix web run lint`, `npm --prefix web run build`, `CI=true npm --prefix web run test:e2e`로 desktop/mobile 전체 검증. 모바일 메뉴 이동이 필요하면 메뉴를 먼저 열 것.

범위와 접근 비교
- 선택: 실제 구현에 맞춰 문서 3개만 정정. 서비스 동작을 바꾸지 않고 이미 존재하는 사용자 오해를 제거한다.
- 대안: 서버 favorites API를 UI에 연결하면 실제 동기화가 가능하나 로그인 전후 병합·권한·삭제 정책이 필요해 45분 과제로 부적합하다.
- 대안: Markdown만 정정하면 가장 짧지만 배포 PDF와 웹 요약의 오안내가 남아 수용하지 않는다. 현상 유지는 사용자에게 틀린 사용법을 계속 제공한다.
- 가장 중요한 가정: 요청은 동기화 기능 추가가 아니라 현행 저장 방식의 정확한 안내다. FavoritesProvider에 인증/API 경로가 없음을 직접 읽어 확인했다.

실행 계획 (구현자 기록용, 모두 미착수; 사람 승인 체크포인트 없음)
1. 문구 정정: 위 3.6·4.4와 웹 요약을 일치시킨다. 권장 핵심 문장: “로그인 여부와 관계없이 즐겨찾기는 현재 브라우저에 저장되며, 다른 브라우저나 기기로 자동 동기화되지 않습니다.” 증명: `git diff -- docs/USER_GUIDE.md docs/guides/user/index.html`와 `./scripts/check-docs.sh`. 통과 후 2단계.
2. 기존 표지와 같은 인자로 새 PDF를 만든다. 아래 명령의 출력은 기존 파일과 다른 임시 파일이어야 한다. PDF 페이지를 추출해 두 설명이 반영됐는지 확인하고 해당 페이지를 렌더링해 검토한 뒤 docs/USER_GUIDE.pdf로 교체한다. 성공 후 3단계. 실행 환경이 다르면 근거와 계획을 수정하고 미검증 상태를 숨기지 않는다.
3. `./scripts/check-docs.sh`, `./scripts/check-env-contract.sh`, `git diff --check`, `git diff --stat`으로 범위와 계약 확인. PDF 텍스트 검증 결과·시각 확인 페이지·한계를 회차 노트에 기록한다. 전체 Go/E2E는 문서 정정의 필수 증명이 아니다.

PDF 생성 명령 (구현 단계에서만 실행; 정찰에서는 원본과 도구만 읽음)
```bash
node /mnt/c/Users/USER/projects/aidev/tools/guide/md2pdf.mjs docs/USER_GUIDE.md /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-095429-appstore-improve/USER_GUIDE.updated.pdf --project AppStore --title '사용자 가이드' --version v2.11.4
```
새 출력이 이미 있으면 고유한 이름을 사용한다. pypdf·fitz가 이 환경에 설치되어 있고 `/usr/bin/google-chrome` 및 변환기의 marked 의존성을 확인했다. pdftotext/pdfinfo는 없으므로 아래처럼 확인한다(수정 전 정찰에서 같은 방법으로 원본 27쪽, 오안내 15·25쪽 확인).
```bash
python3 - <<'PYVERIFY'
from pypdf import PdfReader
p = PdfReader('/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-095429-appstore-improve/USER_GUIDE.updated.pdf')
print('pages:', len(p.pages))
print(p.pages[0].extract_text())
for n, page in enumerate(p.pages, 1):
    text = page.extract_text()
    if '즐겨찾기' in text and ('기기' in text or '브라우저' in text):
        print(n, text)
all_text = ''.join((page.extract_text() or '') for page in p.pages)
compact = ''.join(all_text.split())
assert '다른기기에서도같게보이게하려면로그인한상태로담으세요' not in compact
assert compact.count('로그인여부와관계없이') >= 2
assert compact.count('자동동기화되지않습니다') >= 2
PYVERIFY
```
문구를 다르게 쓰면 단언을 실제 합의한 문장으로 조정하되 두 절을 모두 검증한다. fitz로 해당 페이지를 PNG로 렌더링해 이미지 도구로 확인할 수 있다. 정찰에서 PDF 재생성은 실행하지 않았으므로 결과와 새 쪽 수는 미확인이다.

작업량 근거와 예비 시간 (pmo 스킬 적용)
- bottom-up 추정: 문구와 범위 확인 5~7분 + PDF 생성/내용·시각 검사 8~12분 + 계약 검사/인수 기록 5~6분 = 기본 18~25분.
- 알려진 불확실성 예비 5~10분: Chrome 기동·폰트·PDF 줄바꿈. 총 23~35분, 중간 신뢰의 판단 범위이며 통계적 80% 구간을 주장하지 않는다. 관리 예비는 0분; 새 기능/도구 개조는 별도 범위다. 유사 회차에서 같은 변환기로 두 PDF를 생성한 기록은 있으나 소요 시간이 없어 정량 유사 추정은 불가하다.
- 10분 내 PDF 환경이 해결되지 않으면 차선의 소요를 다시 산정하고 이유를 기록한다. 두 과제를 동시에 하지 않는다.

정찰 증거와 스킬
- HEAD 019f7d1(v2.11.4), git log -30 확인. check-docs/check-env 각각 exit 0. 작업 트리 변경 없음. frontend node_modules 없음; 이번 정찰에서는 Vitest·Go·E2E를 실행하지 않았다.
- Skill 호출 도구는 제공되지 않았으므로 아래 실제 SKILL.md를 직접 읽어 절차를 적용했다. 반환 템플릿 강제는 없고 사용자 과제서 형식에 선택 비교·증명/체크포인트·추정 근거를 추가했다.
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md 및 references/sources.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md
