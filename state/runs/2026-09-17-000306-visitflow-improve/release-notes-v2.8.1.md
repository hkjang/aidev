## What's Changed
* auto-improve: fix(import): read 예/O style consent and find the header below a title row by @hkjang in https://github.com/hkjang/visitflow/pull/16

- 방문자 CSV·XLSX 가져오기가 `예`·`O`·`✓` 같은 한국어 동의 표기를 인정한다(X·빈 칸·미동의는 계속 거부).
- 제목 행이나 빈 행이 앞에 있는 템플릿도 앞 10행에서 `이름`·`휴대전화` 헤더를 찾아 읽고, 경고의 행 번호는 엑셀에서 보이는 번호를 유지한다. 별칭에 성명·휴대폰·연락처를 더했다.

**Full Changelog**: https://github.com/hkjang/visitflow/compare/v2.8.0...v2.8.1
