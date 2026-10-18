#!/usr/bin/env python3
"""Assert the nine lines of docs/index_en.html that JS never overwrites are English.

Usage: python3 -I check_visible_en.py <index_en.html> <index.html> <head_index_en.html>

The nine causal checks (acceptance criteria 1 and 2) are the ones that must go
9 FAIL -> 0 FAIL.  Scope checks (criterion 3) are reported separately because
"136 != 127" fails trivially on the pre-fix file and proves nothing.
"""
import re
import sys

BAD = "Enterprise Telemetry"
HANGUL = re.compile(r"[가-힣ᄀ-ᇿ㄰-㆏]")
COMMENT = re.compile(r'^(?P<pre>.*<span class="term-comment">)(?P<body>[^<]*)(?P<post></span>.*)$')
CLI_LINES = [553, 556, 568, 573, 584, 588, 598, 602]
LANG_LINE = 166

fails = []
oks = []


def check(name, ok, detail=""):
    (oks if ok else fails).append(name)
    print(("OK   " if ok else "FAIL ") + name + ("" if ok else "  -- " + detail))


def lines(path):
    with open(path, encoding="utf-8") as fh:
        return fh.read().split("\n")


def main():
    en_path, ko_path, head_path = sys.argv[1:4]
    en, ko, head = lines(en_path), lines(ko_path), lines(head_path)

    # --- criterion 1: the Korean-switch button label is byte-identical to the KO page
    got, want = en[LANG_LINE - 1], ko[LANG_LINE - 1]
    check(
        "1 line 166 lang-btn byte-identical to docs/index.html:166",
        got.encode("utf-8") == want.encode("utf-8"),
        "got %r want %r" % (got, want),
    )

    # --- criterion 2: each cli-code comment is English, and only the comment text moved
    for n in CLI_LINES:
        name = "2 line %d term-comment English, tags/command untouched" % n
        m_en, m_head = COMMENT.match(en[n - 1]), COMMENT.match(head[n - 1])
        if not m_en or not m_head:
            check(name, False, "no single-line term-comment span on line %d" % n)
            continue
        body = m_en.group("body")
        problems = []
        if BAD in body:
            problems.append("corrupt string still in comment: %r" % body)
        if HANGUL.search(body):
            problems.append("Korean left in comment: %r" % body)
        if not body.startswith("# ") or len(body.strip()) < 4:
            problems.append("not a populated '# ...' comment: %r" % body)
        if "&" in body and not re.search(r"&(amp|lt|gt|quot|#\d+);", body):
            problems.append("raw & must be written &amp;: %r" % body)
        if m_en.group("pre") != m_head.group("pre"):
            problems.append("markup before the comment changed")
        if m_en.group("post") != m_head.group("post"):
            problems.append("markup after the comment changed")
        check(name, not problems, "; ".join(problems))

    print("\n%d check(s) failed, %d passed" % (len(fails), len(oks)))

    # --- criterion 3: scope.  Only meaningful once the nine lines are fixed.
    print("\n[scope] corrupt lines in docs/index_en.html: %d (head had %d)"
          % (sum(1 for l in en if BAD in l), sum(1 for l in head if BAD in l)))
    print("[scope] corrupt lines in docs/index.html: %d"
          % sum(1 for l in ko if BAD in l))
    head_set = {i + 1 for i, l in enumerate(head) if BAD in l}
    en_set = {i + 1 for i, l in enumerate(en) if BAD in l}
    expected = head_set - set(CLI_LINES) - {LANG_LINE}
    print("[scope] unexpectedly fixed lines (out of scope): %s"
          % sorted(expected - en_set))
    print("[scope] target lines still corrupt: %s"
          % sorted(en_set & (set(CLI_LINES) | {LANG_LINE})))

    return 1 if fails else 0


if __name__ == "__main__":
    sys.exit(main())
