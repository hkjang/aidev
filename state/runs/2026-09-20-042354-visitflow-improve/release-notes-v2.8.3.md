## What's Changed
* auto-improve: fix(import): restore phone numbers Excel stored as numbers by @hkjang in https://github.com/hkjang/visitflow/pull/19

- 방문자 CSV·XLSX 가져오기에서 엑셀이 휴대전화 칸을 숫자로 저장해 앞자리 0이
  빠진 값(`1012345678`)이 경고 없이 잘못된 번호로 저장되고, 지수 서식
  (`1.012345678E+09`)이면 경고만 남고 원인을 알 수 없던 문제를 수정.
  8~10자리 숫자만인 값에는 앞 `0`을 보완하고, 가수가 자릿수를 모두 담은 지수
  표기만 정수로 되돌린 뒤 같은 규칙을 적용한다. `1.01E+09`처럼 잘린 값과
  이미 `0`으로 시작하거나 하이픈·공백·`+82`가 있는 값은 그대로 두며 다른
  열은 손대지 않는다. `importPhone` 단위 테스트와 XLSX 숫자·지수 서식 행
  통합 테스트를 추가하고 사용자 가이드에 앞 0 보완을 적었다.

**Full Changelog**: https://github.com/hkjang/visitflow/compare/v2.8.2...v2.8.3
