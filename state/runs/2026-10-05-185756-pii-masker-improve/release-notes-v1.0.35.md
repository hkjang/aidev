## 규칙이 걸린 PII를 한 글자도 가리지 않은 채 `masked_value`에 원문 그대로 보고하던 문제 수정

`internal/masking`의 자리 기준 마스킹 규칙은 모두 입력의 일부를 보이게 남기도록 만들어져 있고, 그중 숫자만 바꾸는 헬퍼들(`maskDigitsAfter`·`maskLastDigits`·`maskKeepTrailingDigits`·`maskDigitsRange`)과 `maskEveryEvenRune`은 보이는 구간 밖에 가릴 자리가 없으면 입력을 그대로 돌려줍니다. 그래서 주민등록번호 칸에 생년월일 6자리만 적힌 값(`"900101"`), 한 글자 이름(`"이"`·`"J"`), 숫자가 없는 플레이스홀더(`"확인불가"`·`"카드없음"`·`"미상"`·`"연락처없음"`·`"없음"`)는 `MaskValue`에서 원문 그대로 나왔습니다. 그 값은 `masking.FieldEntry.MaskedValue` → `service.summarizeFields`를 거쳐 **HTTP 응답의 `pii_summary[].masked_value`와 디스크의 `job.json`에 평문으로** 들어갔고, 그러면서 `resident_registration_number` 같은 규칙명을 함께 달아 "마스킹했다"고 보고했습니다. 가린 내용이 덜 보고되는 것보다 규칙명을 붙여 원문을 그대로 내보내는 쪽이 나쁘므로, 이제 그런 값은 전체를 가립니다.

- `MaskValue`에 "규칙 결과가 trimmed 입력과 같으면 `maskAllVisible`로 값 전체를 가린다"는 폴백을 한 지점에만 넣었습니다. 헬퍼를 개별로 손대지 않았으므로 "앞 6자리는 보인다" 같은 각 규칙의 자리 계약은 그대로입니다.
- 폴백은 `restoreSurroundingSpace` 앞에 있어 앞뒤 공백 복원 순서가 바뀌지 않습니다. `empty` 규칙은 그 전에 조기 반환하므로 빈 값과 공백만 있는 값은 지금까지처럼 원문을 그대로 돌려줍니다 — 가릴 것이 없는 값입니다.
- `maskAllVisible`은 공백이 아닌 룬만 `*`로 바꾸므로 룬 수가 보존됩니다. 덕분에 `ComputeMaskedRuneSpans`가 문서에 인쇄된 값 위로 가림 영역을 맞추는 정렬 계약이 유지됩니다.
- 이미 한 글자 이상 가려지던 값의 출력은 바뀌지 않습니다. 기존 `"홍*동"` 같은 결과는 그대로입니다.
- 도면 위 가림에서는, 폴백이 걸린 값의 가림 영역이 bounding box 폴리곤 전체에서 그 폴리곤의 축 정렬 경계 사각형으로 바뀝니다. 축 정렬 좌표에서는 동일하고 회전된 폴리곤에서는 덮는 면적이 오히려 늘기 때문에 새로 드러나는 영역은 없습니다.
