import os, pathlib, subprocess, sys, yaml
run = pathlib.Path('/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-29-081231-moina-improve')
steps = yaml.safe_load(pathlib.Path('.github/workflows/ci.yml').read_text())['jobs']['image']['steps']
summary = run / 'workflow-summary.md'
summary.write_text('existing summary\n')
for step in steps:
    if step.get('name') == 'Summarize e2e failures':
        assert step['if'] == 'failure()'
        assert steps.index(step) < next(i for i,s in enumerate(steps) if s.get('name') == 'Upload failure diagnostics')
        subprocess.run(['bash','-e','-c',step['run']], check=True, env={**os.environ, 'GITHUB_STEP_SUMMARY':str(summary),'MOINA_E2E_OUTPUT':str(run / 'missing-results')})
value = summary.read_text()
assert value.startswith('existing summary\n'), 'existing summary was overwritten'
assert 'e2e 결과 JSON이 없습니다' in value, 'FAIL: image job failure did not append missing-results diagnostics'
print('PASS: CI summary command appends missing-results diagnostics and preserves existing summary')
