# v0.1.1

브라우저 오류 알림이 같은 화면의 다른 오류를 삼키지 않습니다.

## 고친 것

- 브라우저에서 올라온 오류 보고가 모두 평범한 `new Error` 로 넘어가 알림 파이프라인의
  `code` 가 늘 `'internal'` 이 되었고, 중복 묶음 서명이 `CLIENT <경로> internal` 하나로
  뭉쳤습니다. 그 결과 한 화면에서 처음 터진 오류가 묶음 창 동안 **그 화면의 다른 모든 오류
  메일을 막았습니다.**
- 서명 조립을 순수 함수 `alertSignature(method, path, code, extra='')` 한 곳으로 모으고,
  브라우저 오류는 스크럽된 메시지와 오류 종류로 만든 고정 8자 hex 지문
  (`clientErrorFingerprint`, FNV-1a) 을 서명 끝에 붙입니다. 이제 같은 화면의 서로 다른
  오류는 서로 다른 묶음으로 쌓이고, 진짜 중복만 한 행으로 합쳐집니다.
- `extra` 가 빈 문자열이면 서명은 이전과 바이트 단위로 같습니다 — 운영 DB 에 이미 억제되어
  있던 서버 측 오류가 이 배포로 한꺼번에 메일로 쏟아지지 않습니다.
- 지문은 서명에만 들어가고 DB `path` 칼럼에는 섞이지 않습니다. 폴딩된 경로를 변수 하나로
  묶어 `path` 칼럼과 메일이 서명과 어긋날 수 없게 했습니다.

사용자에게 보이는 동작 변화는 없습니다. 운영자가 받는 오류 메일의 묶음 단위만 달라집니다.

## 버전이 올라간 곳

`VERSION`, 루트와 workspace 의 `package.json` (`package-lock.json` 포함),
`wrangler.toml` 의 `YOUR_VERSION`, `scripts/ci-env.mjs`. 이 값들이 어긋나면
`scripts/deploy-pages.sh` 가 배포 전에 막습니다.

## 검증

- `npm ci` — 수정한 `package-lock.json` 과 `package.json` 이 맞는지 확인됨
- `npm run check` — lint, 타입, 단위 테스트 39개 통과
- `npm run build` — 웹·서버 빌드 통과
- `VERSION` 과 `wrangler.toml` 의 `YOUR_VERSION` 일치 확인 (`deploy-pages.sh` 와 같은 검사)
- e2e 전체는 **이 기계에서 돌리지 못했습니다.** `scripts/e2e.mjs` 가 가짜 서버 포트
  18998·18997 을 하드코딩하고 있고, 같은 기계의 다른 세션이 두 포트와 5440 포트의 개발
  DB 를 이미 쓰고 있었습니다. 구현 단계에서 이 트리로 160 passed / 0 failed 를 받았고,
  CI (`.github/workflows/check.yml`) 가 push 에서 PostgreSQL 16 위로 다시 돌립니다.

## 배포

이 저장소는 태그와 GitHub Release 를 쓰지 않습니다. 운영 배포는 사람이
`bash scripts/deploy-pages.sh` 로 수동 실행합니다 (Cloudflare Pages + Supabase).
