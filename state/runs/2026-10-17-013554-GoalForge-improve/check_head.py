#!/usr/bin/env python3
"""Acceptance checks for the index_en.html <head> restoration (run with python3 -I).

Encodes the five acceptance criteria from the scout brief as assertions so the
same command can be run before and after the edit. Usage:

    python3 -I check_head.py <path-to-docs/index_en.html>
"""
import json
import re
import sys

HANGUL = re.compile(r'[가-힣]')
CORRUPT = "Enterprise Telemetry"
HEAD_LINES = 134

path = sys.argv[1]
with open(path, encoding='utf-8') as fh:
    lines = fh.read().split('\n')
head = lines[:HEAD_LINES]
whole = '\n'.join(lines)

fails = []


def check(name, ok, detail=''):
    print(('PASS  ' if ok else 'FAIL  ') + name + (('  -- ' + detail) if detail else ''))
    if not ok:
        fails.append(name)


# 1) no corrupted filler left in the head, except line 127.
# Line 127 is the dead `lang.indexOf(...)` clause inside #auto-lang-detector.
# Both language buttons carry data-lang ("ko"/"en"), so `lang` is always exactly
# "ko" or "en" and that clause is unreachable. The brief's canonical repair is to
# restore index.html's `lang.indexOf('한')`, but that would put Hangul (U+D55C)
# back inside lines 1-134 and break criterion 2. Dead JS is invisible to crawlers,
# so it is left byte-for-byte alone and excluded here rather than "fixed".
DEAD_LINE = 127
hits = [i + 1 for i, l in enumerate(head) if CORRUPT in l]
check('1 no "%s" in head outside the dead line %d' % (CORRUPT, DEAD_LINE),
      hits in ([], [DEAD_LINE]), 'lines %s' % hits)

# 2) no Hangul left in the head
han = [i + 1 for i, l in enumerate(head) if HANGUL.search(l)]
check('2 no Hangul in head', not han, 'lines %s' % han)

# 3) self-referential canonical / og:url / twitter:url, and en_US locale
EN_URL = 'https://hkjang.github.io/goalforge/index_en.html'
for label, pat in (
    ('canonical', r'<link rel="canonical" href="([^"]*)"'),
    ('og:url', r'<meta property="og:url" content="([^"]*)"'),
    ('twitter:url', r'<meta name="twitter:url" content="([^"]*)"'),
):
    m = re.search(pat, '\n'.join(head))
    got = m.group(1) if m else None
    check('3 %s -> index_en.html' % label, got == EN_URL, 'got %r' % got)
m = re.search(r'<meta property="og:locale" content="([^"]*)"', '\n'.join(head))
check('3 og:locale en_US', m and m.group(1) == 'en_US', 'got %r' % (m and m.group(1)))

# 4) head JSON-LD parses, and the FAQPage questions are English
block = re.search(r'application/ld\+json">(.*?)</script>', whole, re.S)
try:
    data = json.loads(block.group(1))
    parsed = True
except Exception as exc:  # noqa: BLE001
    data, parsed = None, False
    check('4 JSON-LD parses', False, repr(exc))
if parsed:
    check('4 JSON-LD parses', True)
    faq = [n for n in data['@graph'] if n.get('@type') == 'FAQPage'][0]
    qs = faq['mainEntity']
    check('4 FAQPage has 4 questions', len(qs) == 4, 'got %d' % len(qs))
    blob = json.dumps(data, ensure_ascii=False)
    check('4 JSON-LD has no Hangul', not HANGUL.search(blob))
    check('4 JSON-LD has no "%s"' % CORRUPT, CORRUPT not in blob)
    for i, q in enumerate(qs, 1):
        name = q['name']
        text = q['acceptedAnswer']['text']
        ok = not HANGUL.search(name) and not HANGUL.search(text) \
            and CORRUPT not in name and CORRUPT not in text
        check('4 Q%d English' % i, ok, name[:60])

# whole-file corruption count must only go down (reported, not asserted)
print('\ninfo  whole-file "%s" occurrences: %d'
      % (CORRUPT, sum(1 for l in lines if CORRUPT in l)))
print('info  title: %s' % re.search(r'<title>(.*?)</title>', whole).group(1)[:90])

print('\n%d check(s) failed' % len(fails))
sys.exit(1 if fails else 0)
