# vibe-coders 프록시 연동

## 의도

`vibe-code`는 `vibe-coders` 게이트웨이를 직접 기동하지 않습니다. `vibe-coders`가 이미 OpenAI 호환 프록시로 떠 있다고 보고, Vibe Code의 모델 호출 base URL을 `vibe-coders`로 바꿉니다.

이후 Vibe Code에서 발생하는 `/v1/chat/completions`, `/v1/models`, `/v1/embeddings` 호출은 `vibe-coders`를 통과하고, 집계는 `vibe-coders`가 자동으로 처리합니다.

## 기본값

- 프로젝트 경로: `../vibe-coders`
- 기본 base URL: `http://localhost:8080/v1`
- 기본 provider profile: `vibe-coders proxy`
- 기본 모델: `gpt-4.1-mini`

프로젝트 경로는 문서/점검용입니다. 프록시 호출 자체에는 base URL만 필요합니다.

## 사용 흐름

1. `vibe-coders` 프록시가 별도로 실행 중인지 확인합니다.
2. VS Code 명령 팔레트에서 `Vibe Code: vibe-coders 프록시 적용`을 실행합니다.
3. proxy API key를 입력합니다. 비우면 개발 기본값 `dev-proxy-key`를 사용합니다.
4. Vibe Code provider profile이 `openai` provider와 `vibe-coders` base URL로 저장/활성화됩니다.
5. 적용 전 활성 profile 이름은 복원용으로 기억됩니다.
6. 마지막 적용 시각, profile, base URL, model, provider header는 상태 보기용으로 기억됩니다.
7. 이후 Vibe Code 모델 호출은 `vibe-coders`를 지나가며 사용량/토큰/비용이 집계됩니다.

## 명령

- `Vibe Code: vibe-coders 프록시 적용`
  - `apiProvider = openai`
  - `openAiBaseUrl = vibe-code.vibeCodersBaseUrl`
  - `openAiModelId = vibe-code.vibeCodersDefaultModel`
  - `openAiApiKey = 입력한 proxy API key`
  - 선택적으로 `X-Proxy-Provider` 헤더 저장

- `Vibe Code: vibe-coders 프록시 점검`
  - 현재 활성 provider profile이 `vibe-coders` base URL을 향하는지 확인합니다.
  - `/health`, `/ready`만 확인합니다.
  - 모델 호출이 아니므로 사용량 집계를 증가시키지 않습니다.

- `Vibe Code: vibe-coders 프록시 상태 보기`
  - 현재 profile, base URL, model, `X-Proxy-Provider`, 이전 profile, 마지막 적용 시각을 Output 채널에 표시합니다.
  - 네트워크 요청이나 모델 호출을 만들지 않습니다.
  - 운영 중 “지금 Vibe Code 호출이 어디로 갈 예정인지”만 빠르게 확인할 때 사용합니다.

- `Vibe Code: 이전 프로바이더로 복원`
  - `vibe-coders 프록시 적용` 전 활성화되어 있던 provider profile로 되돌립니다.
  - API key나 provider secret을 복사하지 않고 기존 profile을 다시 활성화합니다.

- `Vibe Code: vibe-coders 프록시 설정 파일 생성`
  - 워크스페이스에 `.vibe-code/vibe-coders-proxy.json`을 생성합니다.
  - API key는 기록하지 않습니다.
  - base URL을 클립보드에 복사합니다.

## 설정

```json
{
  "vibe-code.vibeCodersBaseUrl": "http://localhost:8080/v1",
  "vibe-code.vibeCodersDefaultModel": "gpt-4.1-mini",
  "vibe-code.vibeCodersProfileName": "vibe-coders proxy",
  "vibe-code.vibeCodersProviderHeader": ""
}
```

`vibe-code.vibeCodersProviderHeader`를 비워두면 `vibe-coders`의 모델명 기반 provider 자동 라우팅을 사용합니다.

## 사용량 리포트와 대시보드

프록시 profile이 적용되어 있으면 profile의 proxy key를 bearer token으로 `GET /me/report?window=weekly|monthly`를 호출합니다. 이 엔드포인트는 vibe-coders가 API key 기준으로 집계한 개인 사용량(요청, 토큰, KRW 비용, 이전 구간 대비 증감, 성공률, 지연, 캐시 적중, 상위 모델, 절감 가능액)을 돌려줍니다.

- `Vibe Code: vibe-coders 사용량 리포트 보기`: 주간·월간 리포트를 Output에 출력합니다.
- `Vibe Code: vibe-coders 사용량 대시보드`: 같은 데이터를 웹뷰 패널로 엽니다. 새로고침 버튼으로 다시 조회합니다.
- 상태바 `VC: ACTIVE` 항목에 주간 비용·토큰이 붙습니다. 5분 캐시이며 `vibe-code.offlineMode = offline`이면 조회하지 않습니다.

리포트 조회는 `/v1/chat/completions` 호출이 아니므로 사용량 집계에 포함되지 않습니다. key가 없거나 프록시가 꺼져 있으면 실패 사유만 표시합니다.
