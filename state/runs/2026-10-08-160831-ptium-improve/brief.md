- 과제: 마크다운/텍스트 목록 안 인용의 `>`를 요점에서 제거하기 (가치 2 / 위험 2 / 작업량 S)
- 왜: 현재 `Read("월간 보고서.md", []byte("# 제목\n\n- > 인용입니다.\n"))`는 요점을 `- \> 인용입니다.`로 만들고 빈 `- >`도 요점으로 남긴다. 목록 분기에서 기존 인용 헬퍼를 재사용하면 불필요한 마커와 빈 요점을 제거하면서 현재 슬라이드 구조와 출처를 유지할 수 있다.
- 수용 기준:
  1) 같은 파일명으로 읽은 `- > 인용입니다.`와 `- 인용입니다.`의 Document.Source가 같고 경고가 없다. `.md`, `.markdown`, `.txt` 모두 확인한다.
  2) 기존에 인식하는 목록 마커 뒤의 인용을 처리한다: `* >> 문장`, `+ > > 문장`, `• > 문장`, 탭이 낀 목록·인용은 손으로 인용 마커만 제거한 문서와 같다. `> - > 문장`도 한 요점이다.
  3) `- >`, `- >>`처럼 인용 마커만 남은 목록은 요점·경고를 만들지 않는다. 내용이 있는 문서에 넣어 비교하고, 다섯 요점 뒤에 넣어 불필요한 '(계속)' 슬라이드가 생기지 않음을 증명한다. 빈 항목 양쪽의 표는 기존 flush처럼 별개의 표로 남는다.
  4) 인용을 제거한 뒤에도 목록 내용은 요점이다. `- > # 소제목`은 `- # 소제목`과 같은 Source여야 하고 `- > | 이름 | 값 |`도 인용만 뗀 목록과 같아야 한다. `- > - 항목`에서 두 번째 목록 마커를 추가로 제거하지 않는다. handle 재귀로 제목·표·수평선으로 재해석하지 않는다.
  5) `- 매출 > 목표`, `- # 제목`, `-5% 감소`, 이스케이프된 `- \> 문자`는 현재 출력 그대로다. 완결된 펜스 안 목록 인용은 코드 블록 1개·해당 줄 수로만 경고하고, 미닫힘 펜스 재생에서는 새 목록 규칙이 적용된다.
  6) 기존 목록·인용·펜스·front matter·setext·golden 테스트를 보존한다. 새 테스트는 가능하면 동일 파일명의 두 Read 결과 전체 Source를 비교해 표지·제목·출처까지 지킨다. 가이드에는 지원되는 목록 인용의 예를 짧게 추가한다.
- 건드릴 파일:
  - `server/internal/docs/prose.go:readMarkdown` (205행, 목록 분기 281~284행) — flush 뒤 withoutListMarker로 얻은 point에 withoutQuoteMarker를 한 번 적용하고, 빈 결과는 쓰지 않는다. 이 헬퍼 내부가 이미 모든 선두 인용 마커를 제거한다(647행). 기존 writer.point도 빈 문자열을 무시하므로 별도 조건 유무는 구현자가 간결하게 결정한다.
  - `server/internal/docs/markdownquote_test.go` — 위 재현과 경계를 추가. 기존 TestAQuotedSentenceIsAPointWithoutItsMarker 등의 비교 패턴 재사용.
  - `docs/USER_GUIDE.md:276` — 기존 인용 설명 끝에 `- > 인용입니다.`도 마커 없이 요점이 된다는 짧은 문장만 추가.
  - 총 프로덕션 코드 1파일 + 테스트 1파일 + 문서 1파일. 새 라이브러리·API·설정·DB 변경 없음.
- 검증 명령:
  - 저장소 루트: `cd server && go test -count=1 ./internal/docs`
  - 저장소 루트: `cd server && go test -race ./...`
  - 저장소 루트: `cd server && go vet ./...`
  - 저장소 루트: `gofmt -l server/internal/docs/prose.go server/internal/docs/markdownquote_test.go` (출력 없음), `git diff --check`
  - 이번 정찰 baseline: docs 3.010초 통과, go vet 통과, git diff --check 통과. 전체 race 결과는 아래 검증 기록 참조. 웹 변경이 없어 npm 설치·웹 검증은 이번 과제에 불필요하다.
- 위험과 피할 것: withoutListMarker는 Word/PDF와 공유하므로 바꾸지 않는다. writer.go·docs.go:escapeLine·internal/deck/source.go·tables.go·숫자 파서·auth·migrations·workflows·릴리즈/버전 파일·의존성은 범위 밖. 목록 분기의 flush를 제거하면 표가 합쳐질 수 있다. 인용·목록의 일반적인 중첩 AST 파서를 만들거나 순서 있는 목록/들여쓰기 코드/인라인 구문 지원까지 넓히지 않는다. 현재 main@eba092b에는 과거 문서 분리 작업 cb30ae5의 새 절이 없다(276행은 여전히 긴 표 칸): 존재하지 않는 '아직 안 된다' 문장을 지우거나 그 실패 작업 전체를 재수행하지 않는다. 사용자 문자열에 새로운 한국어 조사 조합을 만들 필요도 없다.
- 차선 후보: 제품 /guide 가져오기 설명에 이미 지원되는 TSV·일반 텍스트를 명시하기 (가치 2 / 위험 1 / 작업량 S). `web/src/pages/GuidePage.tsx:GuidePage`의 import 절(115~135행)은 엑셀·CSV·워드·PDF·마크다운만 안내하며 TSV·TXT가 빠져 있다. 서버 `docs.go:Read`와 문서 표를 근거로 두 형식만 추가하고 `cd web && npm ci`, `npm run typecheck`, `npm run build`로 확인한다(정찰은 node_modules가 없어 웹 명령 미실행). 1순위가 이미 해결됐거나 계약상 부적절하다는 실제 증거가 있을 때만 선택한다.

실행 근거와 상태:
- 정찰 전용 `scout_probe_test.go`와 `scout_overlay.json`은 이 회차 출력 폴더에만 있다. 저장소 파일은 생성/수정하지 않았다.
- server 작업 디렉터리에서 `go test -overlay=/mnt/c/Users/USER/projects/aidev/state/runs/2026-10-08-160831-ptium-improve/scout_overlay.json -run TestScoutListQuoteObservation -v ./internal/docs` 실행으로 현재 결함을 확인했다.
- 관찰: `- > 인용입니다.` → `- \> 인용입니다.`, `* >> 인용입니다.` → `- \>> 인용입니다.`, `- >` → `- \>`, `- > # 소제목` → `- \> # 소제목`, `- 매출 > 목표` → `- 매출 > 목표`. 모두 경고 없음. 이 probe는 현재 결함이 있음을 확인하는 진단이므로 수정 뒤의 수용 테스트로 사용하지 않는다.

구현 순서(현재 모두 미착수, 사람 승인 대기 지점 없음):
1. 위 파일과 재현을 다시 확인하고 기존 docs 테스트를 실행한다. 증명: `cd server && go test -count=1 ./internal/docs`. 결과가 다른 경우 과제서를 갱신한 뒤 다음 단계로 간다.
2. 목록 분기와 회귀 테스트를 한 작업 단위로 바꾼다. 새 회귀 테스트가 수정 전 실패함을 확인하고, 수정 후 같은 docs 명령이 통과하는 상태에서 단계 완료로 기록한다. 기존 테스트 기대값 완화 금지.
3. 가이드의 한 문장만 갱신한다. 증명: `cd server && go test -count=1 ./internal/docs -run TestEveryFormatReadIsAFormatSaid`와 `git diff --check`; 이 테스트는 확장자 안내만 검사하므로 새 문장의 정확성은 수용 테스트/코드와 대조한다.
4. 전체 race·vet·gofmt·diff 검증 후 완료 표시한다. 무관한 baseline 실패는 따로 기록하고 이 과제로 고치지 않는다.

접근 비교와 선택:
- 선택: 목록 분기에서 기존 withoutQuoteMarker 조합. 새 상태나 의존성이 없고 프로덕션 1파일에서 사용자 결함을 해소한다.
- 보류: 완전한 Markdown 파서/AST 도입. 앞으로 중첩 표·펜스까지 다루기에는 유리하지만 현재 계약 변경과 다수 회귀를 동반하여 45분/소규모 제한 밖이다.
- 가능한 차선: 동작을 유지하고 한계를 문서화. 런타임 위험은 없으나 매 업로드의 불필요한 글자는 남으므로 이번에는 기능 수정을 선택한다.
- 핵심 가정: 이 리더에서는 목록의 역할(요점)을 유지하고 그 안의 인용 마커만 제거하는 것이 맞다. 일반 Markdown 전체 블록 의미를 구현한다는 주장은 하지 않는다.

작업량 근거:
- 방법: 위 파일·경계 사례를 기준으로 상향식 산정. 재현/테스트 설계 8~10분, 구현+회귀 통과 8~10분, 문서 2~3분, 전체 검증/검토 7~9분 = 기본 25~32분.
- 알려진 불확실성(표 flush, 최대 요점수 경계)의 예비 시간 5~8분을 별도로 두어 총 30~40분. 신뢰는 중간(정찰자의 판단 범위, 실측 확률이나 보장 아님). 관리 예비는 0분 배정: 발견한 새 범위는 다음 회차로 넘긴다.
- 10월 5일 인용 제거 성공 사례는 헬퍼·테스트 형식이 같은 유사 근거다. 당시 작업 소요시간이 없어 분 단위 유사 추정은 하지 않았고, 45분 안에 끝나는지의 범위 점검에만 사용했다.
- 근거·가정·검증·위험을 분리한 추정 방식은 pmo 스킬과 [GAO Cost Estimating Guide](https://www.gao.gov/products/gao-20-195g)의 추정 구성요소를 따른다. 위 분 단위 수치는 GAO 수치가 아니라 이번 정찰 추정이다.

검증 기록(정찰 완료):
- `go test -count=1 ./internal/docs`: exit 0, 3.010초.
- `go test -race ./...`: exit 0, 테스트 보유 25패키지 통과(모델 1패키지는 테스트 없음). docs 44.111초, export 37.518초. DB 서비스 연결 검증은 미확인.
- `go vet ./...`: exit 0. `git diff --check`: exit 0. `git status --short`: 출력 없음.
- 코드 수정/커밋 없음. 수용 기준의 개선 후 결과는 구현자가 확인해야 하며 정찰의 baseline 통과와 구분한다.

적용 스킬:
- [pmo:estimating-and-contingency](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md): 작업 분해, 근거/범위, 예비 시간 분리.
- [technology:implementation-planning](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md): 단계별 파일·검증·체크포인트·미착수 상태.
- [technology:solution-exploration](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md): 국소 수정/파서 도입/현행 유지 대안 비교와 가정 명시.
- 전용 Skill 도구가 노출되지 않아 실제 로컬 원문을 읽어 적용했다.
