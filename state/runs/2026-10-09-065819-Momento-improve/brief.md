- 과제: README 개발 명령의 누적 cd 때문에 make docker가 web에서 실행되는 문제를 고친다 (가치 3 / 위험 1 / 작업량 S)
- 왜: README.md 개발 블록을 루트에서 순서대로 실행하면 sdk → ../web으로 이동한 채 make docker를 호출해 `No rule to make target 'docker'`로 끝난다. SDK와 web 명령을 각각 서브셸로 감싸면 기존 설치·검증·빌드 명령을 유지하면서 마지막 Docker 빌드가 루트 Makefile을 찾는다.
- 수용 기준: 1) 루트에서 문서의 개발 블록을 순서대로 실행할 때 SDK 명령은 sdk, web 명령은 web, 마지막 make는 저장소 루트에서 실행된다. 2) npm·Go 명령의 인자와 순서, Docker 타깃·이미지 이름은 그대로이며 README 한 파일만 변경한다. 3) 수정한 블록의 bash 구문 검사와 아래 경로 실행 검증이 통과한다. 현행 블록은 마지막 make가 exit 2, 제안한 서브셸 블록은 exit 0인 대조 결과를 기록한다. 이것은 경로와 Makefile 해석 증거이며 npm/Go/Docker 전체 빌드 성공을 뜻하지 않는다.
- 건드릴 파일: README.md:98~112 `개발` 절(함수 없음) — 109행을 `(cd sdk && npm install && npm run typecheck && npm run build)`, 110행을 `(cd web && npm install && npm run lint && npm test && npm run build)`로 변경한다. 필요하면 블록 앞에 “저장소 루트에서 실행합니다.” 한 문장만 추가한다. 프로덕션 코드 0파일, 문서 1파일이다. Makefile의 docker 타깃은 읽기 근거이며 수정 대상이 아니다.
- 검증 명령: 아래 경로 검증 명령, `git diff --check`, `git diff -- README.md`. 정찰에서 `(cd web && npm test)`도 259/259 통과했지만 문서 변경만을 위해 새 영구 테스트를 만들 필요는 없다. 전체 CI 명령은 아래 별도 기재했다.
- 위험과 피할 것: auth·session·internal/database/migrations·.github/workflows·Makefile·package.json·잠금 파일·버전·생성 HTML/PDF를 변경하지 않는다. npm install→npm ci 변경, SDK npm test 추가, 실패 시 전체 블록 중단 정책, make verify 도입을 합치지 않는다. 실제 npm install은 잠금 파일을 바꿀 수 있으므로 정찰 검증과 혼동하지 않는다. 보호 대상 원문을 감사 details에 추가하지 않는다. 기존 보안 검사 누락과 의존성 하한 추측을 근거로 새로운 변경을 끼워 넣지 않는다.
- 차선 후보: Custom Dimension 이름·Property key 입력 칸에 기존 서버 규칙 helperText 안내 (가치 2 / 위험 1 / S) — 1순위가 구현 시작 시 이미 해결돼 있을 때만 사용한다. `AdminPage.tsx:DimensionsAdmin`의 3054~3071 TextField 두 곳에는 placeholder만 있다. `internal/httpapi/segments.go:saveDimension` 243~245와 `internal/segment/segment.go:PropertyKeyPattern` 56행이 정본이며 첫 글자는 영문 또는 밑줄, 이후 영문·숫자·밑줄·점·하이픈, 총 1~128자다. 기존 이름 소문자화·서버 trim·저장 버튼·payload는 그대로 두고 안내만 추가한다. 실제 앱 `/admin?section=dimensions`에서 두 helperText 표시와 기존 등록 흐름을 확인한다. 이전 회차의 오류 Alert 한국어화 자체를 다시 구현하지 않는다.

범위와 결정 근거

정찰 기준은 main@0beff4c(v0.34.61), 시작 시 작업 트리는 깨끗했다. README:109~111, Makefile:docker, sdk/package.json, web/package.json을 직접 읽었다. 현행 블록에서 SDK/web 경로는 맞지만 마지막 make만 web에서 실행된다. `make -n docker`는 실제 루트 Makefile의 docker build 명령을 출력하며 exit 0, `make -C web -n docker`는 exit 2다. 문서의 cd/서브셸 경계를 그대로 실행하는 경로 검증도 현행 exit 2, 메모리에서만 고친 후보 exit 0이었다. 저장소 파일은 수정하지 않았다.

대안은 (a) 두 npm 줄을 서브셸로 격리, (b) 마지막 줄 전에 cd .. 추가, (c) make -C .. docker 사용, (d) 현행 유지다. (a)를 선택한다: 문서에 이미 나뉜 SDK/web 단계가 각각 루트 기준이 되어 뒤 명령이 이전 cd에 의존하지 않는다. (b)는 최소 수정으로 유효하지만 이전 단계가 어느 경로에 남았는지에 계속 의존하고, (c)는 마지막 make만 맞추며 셸을 web에 남긴다. (d)는 실행으로 확인한 실패를 그대로 둔다. 스크립트나 Makefile 진입점 신설은 이 두 줄 수정에 비해 범위가 커 제외한다. 핵심 가정은 사용자가 루트에서 블록을 시작한다는 것이며, 첫 Go 명령과 make 모두 원래 이 전제를 갖는다.

실행 계획 (모든 단계 미착수, 인간 승인 체크포인트 없음)

1. 기준 확인: README 개발 블록과 `git diff -- README.md`를 읽고 아래 검증을 수정 전에 실행해 exit 2를 기록한다. 이미 exit 0이며 경로까지 맞으면 중복 변경하지 말고 차선으로 전환한 이유를 기록한다. 체크포인트: 구현자 자체 확인.
2. README의 두 줄만 위의 서브셸 형태로 수정한다. 아래 검증으로 실제 cd가 SDK → web → 루트를 가리키고 make dry-run이 exit 0인지 확인한다. `git diff --check`로 문서 공백 오류를 확인한다. 체크포인트: 구현자 자체 확인, 실패하면 과제서와 현실 차이를 기록하고 원인부터 확인한다.
3. `git diff -- README.md`로 범위 확인 후 변경 전후 결과를 journal에 기록한다. CI 실패가 별도로 보이면 이 문서 변경의 실패라고 단정하지 말고 실제 로그로 구분한다. 체크포인트: 다음 비평 단계로 인계, 정찰이 완료했다고 표시할 단계가 아니다.

경로 검증 명령 (저장소 루트에서 실행)

아래는 README에서 실제 블록을 읽어 bash -n을 돌린다. 경로 검증 부분은 비용·부작용이 있는 Go/npm 명령 대신 pwd를 실행하고 문서의 cd·서브셸 경계는 유지하며, 마지막에는 진짜 make -n docker를 실행한다. 컴파일러·패키지 설치·Docker 데몬 동작을 검증하지 않는다는 한계를 보고에 남긴다. 새 테스트 파일로 저장하지 않아도 된다.

```bash
python3 - <<'PY_CHECK'
from pathlib import Path
import re, subprocess
root = Path.cwd().resolve()
blocks = re.findall(r'```bash\n(.*?)```', Path('README.md').read_text(), re.S)
body = next(b for b in blocks if b.startswith('go test'))
subprocess.run(['bash', '-n'], input=body, text=True, check=True)
probe = []
for line in body.splitlines():
    if line.startswith(('cd ', '(cd ')):
        prefix = line.split(' && ')[0]
        probe.append(prefix + ' && pwd' + (')' if line.endswith(')') else ''))
    elif line.startswith('make docker'):
        probe.extend(['pwd', 'make -n docker'])
r = subprocess.run(['bash', '-c', '\n'.join(probe)], text=True, capture_output=True)
print(r.stdout, end='')
print(r.stderr, end='')
if r.returncode:
    raise SystemExit(r.returncode)
assert r.stdout.splitlines()[:3] == [str(root / 'sdk'), str(root / 'web'), str(root)]
PY_CHECK
git diff --check
```

CI와 검증 한계

정찰 실측: `(cd web && npm test)` exit 0, 259개 통과; 위 경로 검증과 dry-run 대조 완료. Go race/vet/govulncheck, npm audit, SDK 검사, npm 설치, web lint/build, 실제 Docker 이미지·DB 통합 검증은 이번 정찰에서 미실행이다. 문서 수정의 직접 검증과 전체 릴리즈 게이트를 혼동하지 않는다. 기존 회차의 verify-failed 때문에 전체 게이트를 수행할 때에는 make test 하나로 대신하지 않는다. 현재 `.github/workflows/ci.yml`의 명령은 다음과 같다(정찰이 통과했다고 주장하는 목록이 아님).

```bash
go test -race ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go run golang.org/x/vuln/cmd/govulncheck@latest ./cmd/... ./internal/...
(cd sdk && npm ci && npm audit && npm run typecheck && npm test && npm run build)
(cd web && npm ci && npm audit && npm run lint && npm test && npm run build)
docker build --build-arg VERSION=ci --build-arg COMMIT="$(git rev-parse --short=12 HEAD)" -t momento:ci .
```

CI는 Node 24와 PostgreSQL 17 서비스/테스트 DSN을 제공한다. 정찰 환경은 Node 22.23.1/npm 10.9.8이며 DB 통합은 미실행이다. 테스트 DSN 없이 Go 검사만 통과해도 DB 통합 성공으로 보고하지 않는다. 릴리즈 워크플로는 오프라인 아카이브 재적재·기동 smoke까지 수행하므로 Docker dry-run은 그 대체가 아니다.

작업량 산정과 여유

bottom-up: 기준·실패 확인 3~5분, README 수정·구문/경로 확인 3~5분, diff·결과 기록 4~8분으로 기본 10~18분이다. 알려진 불확실성인 기존 문서 변경 충돌·블록 추출 조정에 contingency 5~7분을 별도로 두어 총 15~25분으로 잡는다(중간 이상 확신의 판단 범위이며 통계적 신뢰구간이 아니다). 전역 CI/네트워크 대기·Docker 빌드 시간은 미측정 외부 게이트로 별도 기록한다. 45분 안 문서 작업 완료는 가능하다고 판단하나 전체 CI 완료 시간을 보장하지 않는다. management reserve는 0분이며 새 범위는 추가하지 않고 후속 아이디어로 남긴다. 유사 회차 실측 시간이 없어 두 번째 산정법의 수치 비교는 미확인이다. 범위·가정·위험을 분리하는 산정 근거는 요청된 estimating-and-contingency 스킬과 [GAO 비용 산정 가이드의 방법 개요](https://www.gao.gov/products/gao-20-195g)를 따르며, 위 분 단위 수치는 이 정찰의 판단이다.

기록 차이: 이전 회차 기록에는 describeDimensionError 완료·270개 테스트가 있으나 현재 고정 기준에는 함수가 없고 DimensionsAdmin:3122가 원문 Alert이며 실측 259개다. 완료된 작업을 반복 선택하지 않는다. ideas.json에는 이 차이를 남겼다.
