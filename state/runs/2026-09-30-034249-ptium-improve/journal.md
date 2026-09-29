# 회차 노트 2026-09-30-034249-ptium-improve — ptium
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:42] base pinned — main@64c8875
- [러너 03:42] autonomy release —

## 정찰 노트
- 고른 이유: 보류 목록의 「# 뒤 공백」후보를 임시 테스트로 **실제 실행**해 재현했더니 예상보다 나빴다 — `#1 우선순위는 출시입니다.` 는 제목이 되어 **문장이 본문에서 통째로 사라진다**(요점 0개 슬라이드). setext h2·4칸 코드 블록은 front matter·목록 계약이 선행이라 또 밀었고, XLSX 누적 예산·메일 후속은 M 이상이라 제쳤다.
- 추측 없이 확인한 것: 세 재현 출력과 `## 제목 ##` 의 덱 이름 오염, `.txt` 도 같은 리더를 탄다는 것(docs.go:81), baseline `go test ./internal/docs` = ok 2.741s. 임시 테스트 파일은 지웠고 작업 트리는 깨끗하다.
- 추측으로 적은 것: `docs/` 의 한국어 가이드가 `#` 규칙을 적고 있는지는 **미확인** — 구현자가 grep 한 번 하고 자리가 있을 때만 한 줄.
- 구현자가 조심할 것: `internal/deck/source.go:261` 의 `#` 는 덱 DSL 이라 같은 코드처럼 보여도 계약이 다르다. 거기를 같이 고치면 golden 회귀가 움직인다. 제목이 아니게 된 줄은 반드시 `default:` 로 떨어져 setext 앞보기와 `writer.point` 를 그대로 타야 하고, 미닫힘 펜스 재생 루프도 같은 `handle` 을 쓰므로 새 규칙이 거기에 적용되는 것을 테스트로 한 번 볼 것.
- 프로필은 다시 썼다: VERSION 1.69.50→1.69.52, base de8b1e3→64c8875, setext h1 는 이제 구현됨(80b1be5) 으로 갱신.
- [러너 03:47] scout done — 마크다운/텍스트 업로드에서 `#` 제목은 `#` 뒤에 공백이 있을 때만 — `#출시` 해시태그와 `#1 우선순위`가 �

## 구현 노트
- 무엇을·왜: `prose.go` 의 `case strings.HasPrefix(line, "#")` 를 새 `atxHeading`(1~6 해시 + 줄 끝/공백/탭)으로 바꿔 `#출시` 해시태그와 `#1 우선순위…` 문장이 슬라이드를 열지 않게 했다. 후자는 제목이 되면서 문장 자체가 덱에서 사라지던 것이라 이번 회차의 실제 값이다. 분기는 `switch` 앞의 `if` 로 꺼냈고, 제목이 아니게 된 줄은 `default:` 로 떨어져 setext 앞보기와 `writer.point` 를 그대로 탄다.
- 확신 없는 곳·검증 못 한 것: (1) 공백을 요구하면 `#제목` 이라고 쓰고 제목을 의도한 문서는 이제 요점이 된다 — 의도한 트레이드이고 주석·커밋에 적었지만 실사용 문서로 재어 보지는 못했다. (2) 웹·DB·실서버·Docker·릴리즈는 실행하지 않았다(변경이 Go `internal/docs` 한 파일 + 가이드 한 줄이라 `make test` 의 웹 단계는 최근 회차 관례대로 건너뜀). (3) `docs/` 의 **릴리즈 노트**들에는 옛 `#` 설명이 남아 있을 수 있으나 과거 기록이라 손대지 않았다.
- 일부러 안 한 것: `internal/deck/source.go:261`(덱 DSL 의 `#`, golden 회귀가 그 위에 있음), 닫는 `#` 시퀀스 떼기(`## 제목 ##` → 아직 제목이 `제목 ##` 이고 덱 이름까지 그렇다 — 같은 헬퍼 안에서 끝나므로 다음 회차 1순위로 `ideas.json` 에 올림), `writer.go`·`tables.go`·숫자 파서·`escapeLine`.
- 다음 역할이 조심할 것: `markdownhash_test.go` 6개 중 **2개는 계약 고정용**이라 고치기 전에도 통과한다(6단계 제목 전부 / 빈 제목 `#`·`#   ` 두 경우 — 뒤엣것은 리더가 줄을 TrimSpace 하므로 앞엣것과 같은 줄이 된다). red 였던 것은 나머지 4개. DB 불필요, `go test ./internal/docs` 만으로 돈다.
- [러너 03:51] brief accepted — 채택 — 재현 세 개·수용 기준·권장 구조(`atxHeading` 헬퍼 + `switch` 앞 `if` + `default:` 낙하)가 현재 코드와 정확히 맞아 지정
- [러너 03:51] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 판정 approve(risk low, blocking 없음). `atxHeading` 은 **순수한 좁히기**임을 읽어서 확인했다 — 새 제목 집합이 옛 집합의 부분집합이고, 겹치는 자리의 제목 텍스트는 `TrimSpace(line[hashes:])` == `TrimSpace(TrimLeft(line,"#"))` 로 동일하며, `switch` 앞으로 올린 것은 `atxHeading("")=false` 라 `case line==""` 를 가리지 않는다. 호출부가 이미 `TrimSpace` 한 줄을 넘긴다(prose.go:266).
- 실행한 것: `go build ./...` + `go test -count=1 ./...` 서버 전체 green(stamped_test 포함), `go test -count=1 -race ./internal/docs` ok 36.4s, `go vet` 무출력, `git grep '^#[^ \t#]'` 로 옛 규칙에 기대는 .md/.txt 픽스처 없음. 왕복도 봤다 — escapeLine(docs.go:131) `\#` ↔ unescapePayload(deck/source.go:586). 보안·법무 접점(인증·개인정보·의존성·마이그레이션) 없음.
- 못 본 것: 웹·DB·e2e·Docker·릴리즈 경로(diff 가 Go 한 파일 + 가이드 한 줄이라 실행하지 않음). 구현자가 의심한 「실사용 `#제목` 문서」도 실제 문서로 재지는 못했고 코드 경로로만 따졌다.
- 승인이어도 남는 우려(릴리즈 노트에 한 줄씩): (1) USER_GUIDE.md:276 은 「`#` 뒤에 공백이 있을 때만」이라 적지만 `#` 하나뿐인 줄은 여전히 빈 제목이다 — 코드·테스트는 일치하고 가이드만 거칠다. (2) `#제목` 문서는 장이 합쳐지고 옛 제목 줄도 요점이 되어, 이미 maximumSlides=30 에 닿아 있던 문서는 (계속) 장이 늘어 슬라이드가 잘릴 수 있다. 다만 writer.go:267 이 몇 장인지 알려 주므로 조용한 손실은 아니다.
- 다음 회차: `## 제목 ##` 의 닫는 `#` 가 제목과 덱 이름까지 오염시키는 것이 그대로 남아 있다 — 같은 `atxHeading` 안에서 끝나므로 `ideas.json` 1순위 배치가 타당하다.
- [러너 03:56] review approved — 리뷰 승인 (risk=low)
- [러너 03:56] pr created — https://github.com/hkjang/ptium/pull/39
- [러너 04:00] ci passed — 검사 1개 모두 success
- [러너 04:01] merge done — 70f65fe
- [러너 04:10] release published — v1.69.53
- [러너 04:10] gh-release created — GitHub Release v1.69.53
- [러너 04:10] manifest ok — ptium-1.69.53.tar.gz ptium-1.69.53.tar.gz.sha256 docker-compose.ptium-1.69.53.yml ptium-1.69.53.env.example load-ptium-1.69.53.ps1 load-ptium-1.69.53.sh ptium-1.69.53.kubernetes.yaml 
- [러너 04:10] assets uploaded — 7개
- [러너 04:10] assets verified — v1.69.53 자산 7개 (이전 v1.69.52: 7)
