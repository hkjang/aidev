import sys
import pymupdf
pdf = pymupdf.open(sys.argv[1])
links = [(i + 1, link) for i, page in enumerate(pdf) for link in page.get_links()]
broken = [(page, link) for page, link in links if '/work/' in str(link)]
assert not broken, f'Build-container links are inaccessible to readers: {broken}'
text = ' '.join(' '.join(page.get_text().split()) for page in pdf)
assert 'docs/security.md' in text, 'Missing offline source-document reference'
assert '세 가지 키 계층' in text, 'Missing target section name'
print(f'PASS: {len(pdf)} pages, {len(links)} links; no build-container targets; offline file and section visible')
