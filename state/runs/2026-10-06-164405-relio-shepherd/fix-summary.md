# 수리 요약 — relio PR #44 (CI test=failure)

- 문제: CI `Frontend security audit` 단계(`npm audit --audit-level=high`)가 GHSA-68fv-2mgg-jv7q(source-map-js 1.0.0–1.2.1, high) 때문에 exit 1. **이 PR 과 무관** — `git diff origin/main...HEAD -- web` 이 비어 있고(Go 3파일+테스트만), origin/main 의 lockfile 에도 같은 1.2.1 이 박혀 있어 main 에서도 똑같이 실패한다. 상류 어드바이저리가 새로 공표되며 생긴 선존 실패다.
- 재현: `npm audit --audit-level=high` (web) → CI 로그와 동일한 1건 high, exit 1. 고친 뒤 같은 명령 exit 0.
- 고친 방법: `web/package-lock.json` 의 `node_modules/source-map-js` 만 1.2.1→1.2.2 (패치판, 레지스트리 integrity 대조). postcss 의 `^1.2.1` 범위를 그대로 만족하므로 lockfile 3줄 외에는 손대지 않았다. `npm audit fix --package-lock-only` 는 npm 10 이 npm 11 산 `libc` 필드 45줄을 지워서 쓰지 않고 손으로 바꿨다.
- 검증(전부 PASS): npm ci / audit / typecheck / build / test(27) · `go test -race ./...` 전체 · go vet · gofmt -l 공백 · check-env-contract.sh · check-static-assets.sh. 빌드 산출물 미커밋(dist 는 .gitignore).
- 판단 근거: 지적(CI red)은 실재하나 원인은 이 PR 이 아니다. 그래도 고친 이유는 머지 게이트가 실제로 막혀 있고 수정이 lockfile 한 패키지 범위 안의 보안 패치여서다. 테스트·단언·워크플로는 일절 건드리지 않았다. e00f244 의 본 변경은 그대로다.
