# golang/vulndb 수정 제안 — GO-2026-6452 에 `fixed: 2.11.0` 추가

> 저장소 밖 문서. 제출(outward-facing)은 운영자 결정. 아래 본문은 그대로 PR 에 붙여 쓸 수 있게 영어로 썼다.
> 2026-09-20 09:0x 확인 기준: `https://vuln.go.dev/ID/GO-2026-6452.json` → `modified 2026-09-16T18:00:43Z`, events `[{introduced:"0"}]` 뿐(`fixed` 없음).
> 원본 보고서 `data/reports/GO-2026-6452.yaml` 은 `versions:` 블록 자체가 없고 `vulnerable_at: 2.10.1` 만 있다.

## 대상 파일과 diff

`data/reports/GO-2026-6452.yaml`

```diff
 id: GO-2026-6452
 modules:
     - module: github.com/xuri/excelize/v2
+      versions:
+        - fixed: 2.11.0
       vulnerable_at: 2.10.1
       packages:
         - package: github.com/xuri/excelize/v2
           symbols:
             - xlsxC.getValueFrom
```

(`vulnreport fix GO-2026-6452` 로 derived_symbols·OSV 를 재생성한 뒤 커밋. `data/osv/GO-2026-6452.json` 이 함께 바뀐다.)

## PR 제목

`data/reports: add fixed version for GO-2026-6452`

## PR 본문 (그대로 사용)

```
data/reports: add fixed version 2.11.0 for GO-2026-6452

The report for GO-2026-6452 (CVE-2026-59162, GHSA-fx5j-qcqg-grpf,
"Panic via negative shared-string index in github.com/xuri/excelize")
has no `versions` block, so every version of github.com/xuri/excelize/v2
is treated as affected and `govulncheck` reports `Fixed in: N/A` even for
the release that contains the fix.

Evidence that the fix shipped in v2.11.0:

- The report's own `references` list the fix commit
  https://github.com/qax-os/excelize/commit/93f0b3caed37f21ef5079e3259c6c21dcfe68453
  (PR qax-os/excelize#2331) and the v2.11.0 release page
  https://github.com/qax-os/excelize/releases/tag/v2.11.0.
- `git merge-base --is-ancestor 93f0b3c v2.11.0` in qax-os/excelize
  succeeds; the upstream advisory GHSA-fx5j-qcqg-grpf lists
  "Patched versions: 2.11.0".
- The vulnerable symbol in this report is `xlsxC.getValueFrom`; at
  v2.11.0 it rejects negative shared-string indices (the in-memory
  path). Reproduced locally: a workbook whose cell A1 references shared
  string index -1 panics under v2.10.1 (`index out of range [-1]`) and
  is rejected without panic under v2.11.0.

Note: a separate, later fix f98df08 (GHSA-wcg2-648h-mhxq, in
`getFromStringItem`, the >16 MiB temp-file path) is not part of this
report's symbol set and is tracked separately; it does not change the
fixed version for GO-2026-6452.

Adding `fixed: 2.11.0` makes `govulncheck` stop flagging modules on
v2.11.0 and later.

Related open issues reporting the same gap: #6510, #6501, #6513, #6532.

Fixes #6510
```

## 운영자가 확인하면 좋을 것 (제출 전)

1. golang/vulndb 의 이슈 #6510·#6501·#6513·#6532 에 그 사이 답이나 병합된 수정이 없는지 (있으면 이 PR 은 불필요).
2. `go run ./cmd/vulnreport fix GO-2026-6452` 를 vulndb 클론에서 돌려 lint 가 통과하는지 — `versions` 블록의 위치는 `module:` 바로 아래(`vulnerable_at` 앞/뒤 무관, vulnreport 가 정렬).
3. 원한다면 temp-file 경로(f98df08)도 다루고 싶겠지만 그것은 이 보고서의 symbol 이 아니므로 **별도 보고서**(GHSA-wcg2-648h-mhxq) 요청으로 내야 한다 — 이 PR 에 섞지 말 것.

## 병합 뒤 SecCheck 에서 할 일

- 어떤 코드 변경도 필요 없음. `origin/main`(excelize v2.11.0)과 열린 PR #9/#10(`v2.11.1-0.20260918021423-0434413565bf`)·#11·#12·#13 모두 `security-ci` 의 실패 run 을 재실행(`gh run rerun <id> --failed`)만 하면 step 7 `Go vulnerability scan` 이 초록이 된다.
  - PR #13: run 35474271658 / PR #12: 35449769602 / PR #10: 35442392185 / PR #9: 35436507400 / PR #8: 35372823139
- vulndb 반영은 `vuln.go.dev` 갱신(보통 병합 뒤 수 시간)까지 기다린 뒤 `curl -s https://vuln.go.dev/ID/GO-2026-6452.json | jq '[.affected[].ranges[].events]'` 에 `{"fixed":"2.11.0"}` 이 보이는지 확인하고 재실행한다.
