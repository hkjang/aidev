# fix-summary

- 문제: `web/src/tracking.tsx`의 `addOrigin()`에 이 PR이 새로 넣은 `setPreview(null)`이 격리 미리보기 경고의 '허용 목록에 추가' 버튼(같은 `addOrigin` 사용)에도 적용되어, 차단 출처를 하나 추가하면 `{preview && ...}` 패널 전체가 언마운트되고 남은 `previewViolations` 목록이 사라졌습니다. main의 `addOrigin`은 preview를 건드리지 않았고 커밋·문서·테스트 어디에도 근거가 없는 범위 이탈이었습니다.
- 수정: `setPreview(null)` 한 줄만 제거(커밋 845b6b2). 스니펫 주소 제안 기능(`trackingSnippetOrigins`, 제안 Alert, 가변 인자 `addOrigin`)은 그대로 유지했습니다.
- 검증: `npm --prefix web test` 91/91 통과(실패 0·건너뜀 0), `npm --prefix web run build`(tsc 포함) 성공. 빌드 산출물은 커밋하지 않았습니다.
