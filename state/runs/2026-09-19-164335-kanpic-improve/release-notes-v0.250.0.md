# kanpic v0.250.0

# v0.250.0

## 일괄 등록의 파일 고르기가 탭으로 가른 명단도 보여 줍니다

v0.245.0 부터 서버의 `parseUserCSV` 는 쉼표뿐 아니라 **탭**과 **세미콜론**으로
가른 명단도 읽습니다. 그런데 관리자 화면 **일괄 등록…** 대화상자의 파일 고르기는
`accept=".csv,text/csv"` 그대로여서, 엑셀의 '텍스트(탭으로 분리)' 로 저장한
`.txt` 나 AD 에서 뽑은 `.tsv` 명단은 고르기 창에 아예 보이지 않았습니다. 서버는
받을 준비가 되어 있었지만 그 파일은 대화상자에 **붙여 넣기로만** 들어올 수
있었고, v0.245.0 노트도 그렇게 안내했습니다.

이제 파일 고르기가 `.csv`·`.tsv`·`.txt`(와 그 MIME `text/csv`·
`text/tab-separated-values`·`text/plain`)를 보여 줍니다. 고른 파일을 읽는 쪽
(`readFileText`)은 확장자를 보지 않으므로 그 밖에 바뀌는 것은 없습니다 —
인코딩 판별(v0.244.0)과 구분자 판별(v0.245.0)은 서버가 지금까지처럼 합니다.
`.txt` 가 열리는 만큼 고정폭·공백으로 가른 명단도 골라지지만, 그것은
지금까지처럼 "머리글 줄에 `user_id` 열이 있어야 합니다" 로 되돌아옵니다.

## 릴리즈 워크플로 "두 번 실패" 는 저장소 문제가 아니었습니다

수정 과제로 올라온 "릴리즈 워크플로가 같은 이유로 두 번 실패" 를 확인했습니다.
두 회차 모두 자동 개선 러너가 자기 하루 예산 상한에 걸려 구현 단계에 들어가기
전에 멈춘 것을 실패로 적은 것이고, 그 사이의 v0.249.0 릴리즈는 정상이었습니다.
그래도 `.github/workflows/release.yml` 의 단계(`docker build` → `docker save |
gzip` → `scripts/release-notes.sh`)를 로컬에서 그대로 재현해 모두 통과하는 것을
확인했습니다. 워크플로와 스크립트는 손대지 않았습니다.

## 확인

- `web/src/components/UserImportDialog.test.tsx` — 고르기 목록에 `.csv`·`.tsv`·
  `.txt` 가 있고, 고른 `.tsv` 의 내용이 그대로 입력 칸에 들어오는지 봅니다.
  고치기 전 코드에서 실제로 깨지는 것을 확인했습니다.

`npm run lint`·`npm test`(71 파일 499개 통과)·`npm run build`,
`gofmt`·`go vet`·`go build ./...`, `scripts/check-release-docs.sh`,
`scripts/check-commit-identities.sh` 를 돌렸습니다. Go 코드는 손대지 않았습니다.

## 올리실 때

웹 화면의 파일 고르기 속성 하나만 바뀌었습니다. 마이그레이션과 새 설정은 없고
필수 런타임 환경 변수는 계속 `POSTGRES_DSN` 하나입니다. v0.249.0 에서
올라오시는 분은 이미지만 바꿔 다시 띄우시면 됩니다.

## 오프라인 설치

이 릴리즈는 웹 자산과 서버 바이너리를 포함한 Docker 이미지 아카이브로 제공됩니다.

```bash
sha256sum -c kanpic-v0.250.0.tar.gz.sha256
gzip -dc kanpic-v0.250.0.tar.gz | docker load
docker run --rm -p 8080:8080 \
  -e POSTGRES_DSN='postgres://kanpic:password@postgres.internal:5432/kanpic?sslmode=require' \
  kanpic:v0.250.0
```

- 필수 런타임 환경 변수: `POSTGRES_DSN`
- Redis 및 외부 인터넷 연결: 불필요
- 관리자 초기 로그인을 보호하려면 `BOOTSTRAP_ADMIN_ID`와 `BOOTSTRAP_ADMIN_PASSWORD`를 함께 지정합니다.
- 무결성 검증 파일: `kanpic-v0.250.0.tar.gz.sha256`
