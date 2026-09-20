import os, pathlib, subprocess, sys
outdir = pathlib.Path(__file__).parent
case = sys.argv[1]
env = os.environ.copy()
env.pop('TEST_POSTGRES_DSN', None)
if case in ('empty', 'spaces', 'whitespace', 'nbsp'):
    env['TEST_POSTGRES_DSN'] = {'nbsp':' ', 'empty':'', 'spaces':'   ', 'whitespace':' \t\r\n '}[case]
elif case == 'invalid':
    env['TEST_POSTGRES_DSN'] = 'postgres://invalid@127.0.0.1:1/postgres?sslmode=disable&connect_timeout=1'
elif case == 'valid':
    env['TEST_POSTGRES_DSN'] = os.environ['TEST_POSTGRES_DSN']
p = subprocess.run(['make', 'test'], env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
output = p.stdout
# No supplied DSN is persisted, including in existing Go failure messages.
dsn = env.get('TEST_POSTGRES_DSN', '')
(outdir / (case + '.log')).write_text(output.replace(dsn, '[DSN]') if dsn.strip() else output)
warn = [line for line in output.splitlines() if 'WARN' in line and 'TEST_POSTGRES_DSN' in line]
print(f'{case}: exit={p.returncode}, warnings={len(warn)}', flush=True)
if case in ('valid', 'invalid'):
    assert not warn, 'nonempty DSN must not warn'
else:
    assert len(warn) == 1, 'missing DSN must warn exactly once'
    assert 'PostgreSQL' in warn[0] and 'skipped' in warn[0]
    assert output.index(warn[0]) < output.index('cd backend && go test')
assert (p.returncode != 0) if case == 'invalid' else (p.returncode == 0)
if case != 'invalid':
    assert 'cd runner && go test' in output and 'cd web && npm test' in output
    assert '94 passed' in output
print('PASS', flush=True)
