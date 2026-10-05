# v0.76.6 — 이름 없는 공간은 읽는 사람의 말로 불립니다

공간에 이름을 아직 붙이지 않았으면 umm 이 대신 하나를 불러 줍니다. 그 대신 부르는 이름이 화면
코드에 **영어 그대로** 박혀 있었으므로, 한국어로 umm 을 쓰는 사람은 캔버스 머리글에서 `My Space`
를 봤고 그 공간을 내려받으면 파일 이름에도 `My Space` 가 따라왔습니다. 번역 사전에는 이미
`'내 공간'` 항목이 있었는데도, 그 한 값만 번역을 지나가지 않았습니다.

같은 자리에서 **공백뿐인 이름**도 통과했습니다. 이름이 `"   "` 인 공간은 "이름이 있다" 로 세어져
머리글이 비어 버리고, 내려받은 파일은 이름 자리가 공백인 `umm-   .md` 가 됐습니다.

```
    한국어로 쓰는 사람이 이름 없는 공간을 열고 내려받으면

            전                            후
    ━━━━━━━━━━━━━━━━━━━━━━━    ━━━━━━━━━━━━━━━━━━━━━━
    머리글:  My Space           머리글:  내 공간
    백업:    umm-My Space.md    백업:    umm-내 공간.md
    차례:    umm-My Space-차례.md        umm-내 공간-차례.md
    그림:    umm-My Space.png            umm-내 공간.png
    PDF:     umm-My Space.pdf            umm-내 공간.pdf

    이름이 공백뿐인 공간(`"   "`)이면

    머리글:  (빈칸)              머리글:  내 공간
    백업:    umm-   .md          백업:    umm-내 공간.md
```

## 한 값에서 머리글과 파일 이름이 함께 나옵니다

고친 자리는 `CanvasPage` 의 **한 줄**입니다. 그 줄의 결과는 머리글 한 곳, 내려받기 파일 이름 네
곳(백업 `.md` · 차례 `.md` · 그림 `.png` · `.pdf`), 자식 컴포넌트에 넘기는 이름 한 곳에 **함께**
쓰입니다. 그래서 머리글만 한국어로 만들고 파일 이름을 두고 올 수가 없었습니다 — 여섯 곳이 한
값에서 나오므로, 그 값 하나를 고치면 여섯 곳이 같이 고쳐집니다.

```ts
export function spaceDisplayName(spaces: readonly Space[], activeSpace: string | undefined, fallback: string): string {
  const name = spaces.find((space) => space.id === activeSpace)?.name.trim();
  return name ? name : fallback;
}
```

`?.name ||` 이 `?.name.trim()` 이 되면서 공백뿐인 이름은 "없는 이름" 으로 세어집니다. 다듬은 결과가
비어 있지 않으면 **사람이 지은 이름을 그대로** 보이고, 비어 있으면 — 이름이 없거나, 공백뿐이거나,
고른 공간이 아직 목록에 없거나 — 대신 부를 이름이 나갑니다.

## 대신 부를 이름만 번역하고, 사람이 지은 이름은 건드리지 않습니다

이 모듈은 **안에서 번역을 부르지 않습니다.** 대신 부를 이름을 인자로 받습니다. 부르는 쪽이
`spaceDisplayName(spaces, activeSpace, t('내 공간'))` 으로 넘깁니다.

번역 사전의 키는 한국어 원문입니다. 그래서 사전에 든 말을 **자기 공간 이름으로 지은 사람**이
있습니다 — `생각 공간` 은 영어 사전에 `Thought space` 로 올라 있는 키입니다. 함수 안에서 이름을
번역했다면 그 사람의 공간은 영어 화면에서 `Thought space` 로 바뀌어 불렸을 것입니다. 사람이 지은
이름은 번역할 말이 아니라 그 사람의 말이고, 번역하는 것은 **우리가 붙인** 대신 부를 이름뿐입니다.

영어로 읽는 사람의 머리글은 `My Space` 에서 `My space` 로, 사전에 적힌 대로 바뀝니다. 같은 뜻의
같은 자리이고, 이제 두 언어가 같은 한 항목에서 나옵니다.

## 달라지지 않는 것

**이미 지어진 이름은 하나도 바뀌지 않습니다.** 이 함수는 보여 줄 이름을 고르기만 하고 저장된 것을
읽지도 쓰지도 않습니다. 서버도, API 도, 데이터도 v0.76.5 와 같습니다.

왼쪽 사이드바와 모바일 탭의 이름표(`AppLayout`)는 **건드리지 않았습니다.** 그쪽 모바일 탭은 이름표의
첫 단어만 써서 좁은 화면에 넣으므로, 같은 손질을 거기까지 끌고 가면 탭이 `내` 한 글자가 됩니다 —
라벨을 어떻게 줄일지가 먼저 정해져야 하는, 이 수정과 다른 과제입니다. 내려받기 네 곳 가운데 PDF 만
파일 이름에서 위험한 문자를 털어내고 나머지 셋은 그러지 않는 차이도 그대로 남아 있습니다. 이번
릴리즈는 **이름을 고르는 한 값**만 고칩니다.

## 검증

시험은 손으로 만든 사전이나 가짜 번역 함수를 쓰지 않습니다. 실제 `i18n/translate` 의
`setLocale`/`translate`, 실제 영어 사전, 실제 `Space` 타입을 그대로 들여와 씁니다.

```
✓ 사람이 지은 이름을 그대로 보인다                     제품 계획 → 제품 계획
✓ 이름이 비었으면 대신 부를 이름                       "" → 내 공간
✓ 공백뿐인 이름도 대신 부를 이름                       "   " → 내 공간
✓ 고른 공간이 목록에 아직 없으면 대신 부를 이름        없는 id · 빈 목록 → 내 공간
✓ 사전 키와 겹치는 이름은 영어 화면에서도 그대로       생각 공간 → 생각 공간 (en)
✓ 대신 부를 이름은 한국어 독자에게 내 공간             translate('내 공간') = 내 공간 (ko)
✓ 대신 부를 이름은 영어 독자에게 My space              translate('내 공간') = My space (en)
```

고치기 전에 같은 시험이 실패하는 것을 먼저 확인했습니다. 고친 모듈에 전의 식
(`?.name || 'My Space'`)을 글자 그대로 되돌려 넣으면 **3개 실패 / 4개 통과** 입니다 —
`expected 'My Space' to be '내 공간'`(이름이 빈 공간), `expected '   ' to be '내 공간'`(공백뿐인
이름), `expected 'My Space' to be '내 공간'`(고른 공간이 목록에 없음). 같은 실행에서 나머지 4개 —
사람이 지은 이름, 사전 키와 겹치는 이름, 두 언어의 대신 부를 이름 — 는 통과합니다. 새 시험이
"전부 거절" 로 통과하는 것이 아니라는 증거입니다.

```
✓ npm test --silent                      EXIT=0   21개 파일 225개 시험 (기준선 20/218)
✓ 프로덕션 모듈만 되돌리면               3개 실패 / 4개 통과
✓ npm run typecheck                      EXIT=0
✓ npm run lint (oxlint + Prettier)       EXIT=0   All matched files use Prettier code style!
✓ node web/scripts/check-i18n.mjs        EXIT=0   1060개 키
✓ npm run build · npm run verify:pwa     EXIT=0   150개 build asset
✓ npm run test:offline-queue             EXIT=0
✓ npm audit --audit-level=high           EXIT=0   high 이상 0건
```

---

**전체 검증**: `gofmt -l` 무출력 · `go vet ./...` · `go build ./cmd/...` ·
`go test ./... -count=1` 15개 패키지 통과(이 기계에 `POSTGRES_DSN` 이 없어 DB integration 은
건너뛰었습니다 — 돌았다고 적지 않습니다. 이번 변경에 Go 코드는 0줄입니다) ·
`scripts/check-version.sh` · `release.yml` 과 같은 `docker build --build-arg VERSION=0.76.6 .`
