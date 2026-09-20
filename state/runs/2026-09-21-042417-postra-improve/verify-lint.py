import hashlib, pathlib, subprocess, tempfile, tarfile, io
root = pathlib.Path.cwd()
run = pathlib.Path(__file__).parent

def snapshot(base):
    return {str(p.relative_to(base)): hashlib.sha256(p.read_bytes()).hexdigest() for p in base.rglob('*') if p.is_file() and '.git' not in p.relative_to(base).parts}

def check(base, label, args, success=True, contains=None):
    before = snapshot(base)
    result = subprocess.run(args, cwd=base, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    (run / (label + '.log')).write_text(result.stdout)
    assert (result.returncode == 0) == success, (label, result.returncode, result.stdout[-3000:])
    assert contains is None or contains in result.stdout, (label, result.stdout[-3000:])
    assert snapshot(base) == before, label + ': files changed'
    print(label, 'exit', result.returncode, 'files unchanged', flush=True)

status = subprocess.check_output(['git', 'status', '--porcelain'], cwd=root)
for target in ['lint-format', 'lint-security', 'lint']:
    check(root, target, ['make', target])
with tempfile.TemporaryDirectory(prefix='lint-fixtures-', dir=run) as temp:
    copy = pathlib.Path(temp)
    archive = subprocess.check_output(['git', 'archive', 'HEAD'], cwd=root)
    with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
        tar.extractall(copy, filter='data')
    (copy / 'Makefile').write_bytes((root / 'Makefile').read_bytes())
    fixture = copy / 'internal/lintfixture'
    fixture.mkdir()
    source = fixture / 'fixture.go'
    source.write_text('package lintfixture\nfunc example( ){ }\n')
    check(copy, 'format-violation', ['make', 'lint-format'], False, 'internal/lintfixture/fixture.go')
    check(copy, 'lint-format-violation', ['make', 'lint'], False, 'internal/lintfixture/fixture.go')
    source.write_text('package lintfixture\nfunc broken(\n')
    check(copy, 'format-syntax-error', ['make', 'lint-format'], False, 'expected')
    source.write_text('package lintfixture\n\nimport "os"\n\nfunc UnsafeWrite(path string) error {\n\treturn os.WriteFile(path, []byte("fixture"), 0777)\n}\n')
    subprocess.run(['gofmt', '-w', str(source)], check=True)
    check(copy, 'security-violation', ['make', 'lint-security'], False, 'G306')
    check(copy, 'lint-security-violation', ['make', 'lint'], False, 'G306')
    source.write_text('package lintfixture\nfunc broken(\n')
    check(copy, 'security-load-error', ['make', 'lint-security'], False)
assert subprocess.check_output(['git', 'status', '--porcelain'], cwd=root) == status
print('Temporary fixtures removed; original git status unchanged', flush=True)
