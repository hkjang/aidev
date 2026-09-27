# 수리 요약 — madi PR #13

- 문제(확인됨): 이 브랜치는 `deploy/runtime-apk.lock:31` 에서 libexpat 을 2.8.4-r0 → 2.8.5-r0 으로 재정렬하는데, `RELEASE_NOTES.md:13` 은 재정렬 대상을 `ca-certificates`·`ca-certificates-bundle`·`tzdata` 로만 열거해 공지가 실제 이미지 변경과 어긋났다. `.github/workflows/release.yml:132` 이 이 파일을 그대로 게시 본문으로 복사하므로 v0.4.0 공지에 그대로 나간다.
- 고친 것: `RELEASE_NOTES.md:13` 한 문장에 `libexpat` 2.8.5-r0(CVE-2026-93990 수정)을 함께 열거. 문서 1줄만 변경했고 락 파일은 건드리지 않았다.
- 근거 검증(재사용 아님, 직접 확인): 로컬 `madi:ci` 이미지에서 `/usr/share/madi/sources/aports/expat/41a9605.../recipe/APKBUILD` secfixes 에 `2.8.5-r0: CVE-2026-93990` 확인, 설치 DB `V:2.8.5-r0` 이 락 31줄과 일치 확인.
- 재검증: `go test -count=1 ./tests/deployment-contract` ok, `node --test tests/natural-date.mjs tests/silent-sso.mjs` 2/2 pass, `git diff --check` 깨끗. 다른 *.md 에 락 버전 열거 없음(grep 재확인).
- 커밋: cb624b5 (ad5084a 위 새 커밋, rebase/amend 없음, push 안 함).
