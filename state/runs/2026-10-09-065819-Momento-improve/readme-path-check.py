from pathlib import Path
import re, subprocess
root = Path.cwd().resolve()
blocks = re.findall(r'```bash\n(.*?)```', Path('README.md').read_text(), re.S)
body = next(b for b in blocks if b.startswith('go test'))
subprocess.run(['bash', '-n'], input=body, text=True, check=True)
probe = []
for line in body.splitlines():
    if line.startswith(('cd ', '(cd ')):
        prefix = line.split(' && ')[0]
        probe.append(prefix + ' && pwd' + (')' if line.endswith(')') else ''))
    elif line.startswith('make docker'):
        probe.extend(['pwd', 'make -n docker'])
r = subprocess.run(['bash', '-c', '\n'.join(probe)], text=True, capture_output=True)
print(r.stdout, end='')
print(r.stderr, end='')
if r.returncode:
    raise SystemExit(r.returncode)
assert r.stdout.splitlines()[:3] == [str(root / 'sdk'), str(root / 'web'), str(root)]
