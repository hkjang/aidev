HEAD 1855dcb의 CI 실패는 로컬에서 재현되지 않아 코드·테스트·워크플로 수정 및 커밋을 하지 않았다(push 없음).
Node 20.19.2에서 npm ci → npm run check → node --check dist/extension.js && node --check dist/extension.core.js 모두 종료 0: typecheck, 11파일/146 tests, build 통과(Node 22.23.1 check도 통과).
저장된 CI 결과는 3초 만의 failure와 annotations_count=2만 제공한다. gh 기본 인증 없음, 기존 Windows gh 설정/API 조회도 HTTP 404로 원격 실패 로그·annotation 원문을 확보하지 못했다.
원인은 미확정이며 코드가 틀렸다는 근거도 없어 임의 수리하지 않았다. 다음 역할은 해당 check-run 109246301265의 두 annotation과 실패 단계 원문을 접근 가능한 인증으로 확인해야 한다.
기존 runtime assets 누락 빌드 경고 및 npm moderate 2건은 그대로이며 VSIX/Windows/Extension Host는 미검증; 새 테스트 추가·원복 재실패 검증은 재현되는 결함이 없어 수행하지 않았다.
