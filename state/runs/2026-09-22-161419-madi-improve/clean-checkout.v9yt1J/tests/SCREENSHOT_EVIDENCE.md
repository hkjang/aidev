# 정상 캡처 게시 절차

`scripts/verify-browser.sh`는 별도 시험 DB와 비어 있는 8080 포트에서 25개 shared suite를 순차 실행합니다. `regression-shared.mjs`는 `test-results/regression-shared/screenshots/<suite>/`에 캡처하고 `report.json`에 다음 증거를 남깁니다.

- 실제 서비스가 시험 전후 제공한 번들 이름과 실행 batch ID
- 정확한 시작·종료 시각, 종료 코드·timeout·증거 수집 오류
- 이번 실행에서 새로 생성되거나 변경된 정상 이름 PNG의 크기·SHA-256·mtime·ctime

이전 디렉터리의 변경 없는 PNG는 이번 실행의 결과에 포함하지 않습니다. 번들이 실행 중 달라지거나 확인되지 않으면 suite 성공으로 기록하지 않습니다. `MADI_REGRESSION_SCREENSHOT_ROOT`는 별도 경로로 시험할 때 사용할 수 있지만, 공개 게시기는 아래 정규 경로만 허용합니다.

## 검토 후 게시

```sh
node --test tests/publish-verified-screenshots.test.mjs
node tests/publish-verified-screenshots.mjs --bundle main-VERIFIED_HASH.js --dry-run
node tests/publish-verified-screenshots.mjs --bundle main-VERIFIED_HASH.js
```

`--bundle`에는 실제 정상 실행 report의 번들을 명시합니다. 현재 체크아웃이나 오래된 report의 번들 값을 자동으로 새 값으로 바꾸지 않습니다. 같은 batch의 25개 suite가 모두 통과해야 하며, 이 중 정상 캡처가 없는 비동기·읽기 모드 시험도 성공 증거는 필요합니다. 일부 suite만 재실행한 report를 전체 새 batch의 성공으로 게시하지 않습니다.

`screenshot-evidence.mjs`의 suite별 명시 승인 목록만 게시합니다. `failure`, `error`, `debug`, `diagnostic`, `raw` 또는 임의의 새 파일명은 성공한 suite 디렉터리에 있어도 게시하지 않습니다. 새 정상 캡처를 추가하려면 실제 화면·비밀값 노출 여부를 확인하고 승인 목록에 추가하세요. 이 도구는 이미지 OCR이나 개인정보 검사를 수행하지 않으며, 정상 파일명만으로 화면 내용의 안전성까지 보장하지 않습니다.

게시 전에 모든 report·파일의 시각·해시·경로를 검증합니다. 심볼릭 링크, 시험 이후 변경된 파일, 잘못된 시간 구간은 거부합니다. 같은 파일명은 더 나중에 촬영된 검증 결과를 선택하고 충돌 해소 기록을 남깁니다. 동일 시각의 서로 다른 내용은 모호하므로 거부합니다. 기존 게시 증거가 더 최신이면 오래된 후보로 덮어쓰지 않습니다.

게시한 파일은 `docs/screenshots/`에 저장하고 `test-results/regression-shared/published-screenshots.json`에 원본 경로·suite·bundle·batch·시각·SHA-256과 제외/교체 이유를 기록합니다. 기존 다른 모듈 캡처는 건드리지 않고, 이번 report가 검증한 파일만 이번 번들의 증거로 취급합니다. 공개 갤러리와 coverage 생성은 이 단계가 끝난 뒤 수행합니다. 구 형식 report는 재사용하지 말고 현재 runner로 다시 검증하세요.
