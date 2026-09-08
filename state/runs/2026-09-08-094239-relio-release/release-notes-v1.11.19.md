## Relio v1.11.19 — 릴리즈가 한 번 깨진 뒤로 이미지가 더 나오지 않던 문제

태그는 v1.11.18 까지 붙어 있는데 오프라인 반입용 Docker Image 는 v1.11.14 가 마지막이었습니다. 릴리즈 워크플로의 "이전 릴리즈에서 업그레이드" 단계가 **직전 태그**를 직전 릴리즈로 여겼기 때문에, 한 번 릴리즈가 실패해 태그만 남으면 그 뒤의 모든 릴리즈가 존재하지 않는 자산을 내려받으려다 1초 만에 실패했습니다. 이번 릴리즈는 업그레이드 출처를 **실제로 발행된 가장 최신 릴리즈**로 바꿔, 깨진 릴리즈 하나가 다음 릴리즈들을 계속 막는 고리를 끊습니다.

## 1. 원인

### 1-1. 직전 태그는 직전 릴리즈가 아니다

업그레이드 검증 단계는 이렇게 이전 릴리즈를 골랐습니다.

```bash
previous_tag="$(git tag --sort=-v:refname | awk -v current="$GITHUB_REF_NAME" '$0 != current { print; exit }')"
if [ -n "$previous_tag" ]; then
  previous_asset="relio-${previous_tag}.tar.gz"
  gh release download "$previous_tag" --repo "$GITHUB_REPOSITORY" --pattern "$previous_asset"
  ...
fi
```

태그는 워크플로가 시작되는 시점에 이미 존재합니다. 그런데 GitHub Release 는 워크플로의 **마지막 단계**인 "Publish GitHub Release" 가 만듭니다. 그 앞의 어느 단계에서든 멈추면 릴리즈도 이미지도 없는 태그가 남습니다. 다음 릴리즈는 그 태그를 이전 릴리즈로 보고 `relio-<태그>.tar.gz` 를 요청하지만, 그런 자산은 존재한 적이 없습니다.

### 1-2. 그 실패가 다음 릴리즈를 다시 실패시킨다

업그레이드 단계가 실패하면 그 뒤의 "Publish GitHub Release" 도 실행되지 않습니다. 즉 **실패가 다시 릴리즈 없는 태그를 만들어** 다음 릴리즈에 같은 함정을 물려줍니다.

| 태그 | 멈춘 지점 | 남은 것 |
| --- | --- | --- |
| v1.11.15 | Test source | 릴리즈 없는 태그 |
| v1.11.16 | Verify upgrade (`relio-v1.11.15.tar.gz` 없음) | 릴리즈 없는 태그 |
| v1.11.17 | Verify upgrade (`relio-v1.11.16.tar.gz` 없음) | 릴리즈 없는 태그 |
| v1.11.18 | Verify upgrade (`relio-v1.11.17.tar.gz` 없음) | 릴리즈 없는 태그 |

처음 한 번은 소스 테스트 실패라는 정당한 이유였지만, 그 뒤 세 번은 고쳐진 코드가 릴리즈되지 못한 것뿐입니다. 태그 네 개가 릴리즈 없이 남았고, 오프라인망에 반입할 수 있는 마지막 이미지는 v1.11.14 에 멈춰 있었습니다.

### 1-3. 정렬 순서상 더 나중 릴리즈가 뽑힐 수도 있었다

옛 선택은 "현재 태그가 아닌 가장 최신 태그" 였습니다. 오래된 태그로 워크플로를 다시 돌리면 자기보다 **나중** 릴리즈에서 업그레이드하는, 방향이 뒤집힌 검증이 됩니다.

## 2. 수정

### 2-1. 발행된 릴리즈를 찾을 때까지 거슬러 올라간다

`scripts/previous-release-tag.sh` 가 현재 태그보다 **오래된** 태그만 최신순으로 훑으면서, `gh release view` 로 릴리즈가 실제로 존재하고 `relio-<태그>.tar.gz` 자산까지 달려 있는 첫 태그를 고릅니다.

```bash
older_tags="$(git tag --sort=-v:refname | awk -v current="$current_tag" 'found { print } $0 == current { found = 1 }')"
```

현재 태그보다 위로 정렬되는 것은 애초에 후보에 들어가지 않으므로, 나중 릴리즈가 업그레이드 출처가 되는 일은 없습니다.

### 2-2. 느슨해진 것은 없다

건너뛰는 경우는 두 가지, "태그는 있는데 릴리즈가 없다" 와 "릴리즈는 있는데 이미지 자산이 없다" 뿐입니다. 그 외의 실패는 조용히 더 거슬러 올라가지 않고 릴리즈를 멈춥니다.

```bash
case "$output" in
*"release not found"* | *"Not Found"* | *"HTTP 404"*) return 1 ;;
esac
printf '%s\n' "$output" >&2
return 2
```

API 가 503 을 돌려주는 것과 릴리즈가 없는 것은 다른 사실입니다. 전자를 후자로 읽으면 "업그레이드할 이전 릴리즈가 없다" 는 잘못된 결론으로 검증을 통째로 건너뛰게 되므로, 이때는 실패로 처리합니다. 결과가 빈 값인 경우는 여전히 하나, **발행된 릴리즈가 하나도 없는 첫 릴리즈** 뿐입니다.

워크플로는 이제 어느 이미지에서 업그레이드했는지도 로그에 남깁니다.

```bash
previous_tag="$(./scripts/previous-release-tag.sh "$GITHUB_REF_NAME")"
if [ -z "$previous_tag" ]; then
  echo "No earlier release was ever published, so there is no image to upgrade from."
  exit 0
fi
echo "Upgrading from the newest published release: $previous_tag"
```

## 3. 회귀 방지

`scripts/previous-release-tag-test.sh` 를 CI 의 "Verify the release upgrades from a published release" 단계에 추가했습니다. `gh` 를 태그별 파일 하나로 답하는 스텁으로 대신해 실제 릴리즈 이력을 재현하고, 8가지를 확인합니다.

- 직전 태그가 발행된 릴리즈면 그것이 이전 릴리즈인지.
- 릴리즈 없는 태그는 건너뛰고 실제로 발행된 최신 태그를 고르는지.
- 이번에 처음 깨진 릴리즈(v1.11.16)의 업그레이드 출처도 제대로 찾아지는지.
- 이미지 자산이 없는 릴리즈는 건너뛰는지.
- 나중 릴리즈가 업그레이드 출처로 뽑히지 않는지.
- 첫 릴리즈는 업그레이드할 대상이 없다고 답하는지.
- 릴리즈를 읽을 수 없으면(예: 503) 탐색을 멈추고 실패하는지.
- 체크아웃에 없는 태그는 거절하는지.

두 스크립트는 CI 가 직접 실행하므로 실행 권한(100755)으로 커밋되어 있습니다. 100644 로 들어가면 그 단계가 exit 126 으로 멈춥니다.

## 4. 적용

마이그레이션은 없습니다. Application 코드는 바뀌지 않았고, 이번 릴리즈에서 달라지는 것은 릴리즈 파이프라인뿐입니다. 이 릴리즈의 업그레이드 검증은 v1.11.14 이미지에서 v1.11.19 이미지로 수행됩니다 — v1.11.15 부터 v1.11.18 까지는 태그만 있고 릴리즈가 없기 때문입니다. 따라서 v1.11.19 자산 하나에 v1.11.15 이후의 모든 수정이 함께 들어 있습니다.
