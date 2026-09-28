import json, os, pathlib, subprocess, yaml
run=pathlib.Path('/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-29-081231-moina-improve')
repo=pathlib.Path.cwd()
steps=yaml.safe_load((repo/'.github/workflows/ci.yml').read_text())['jobs']['image']['steps']
startup=next(s['run'] for s in steps if s.get('name')=='Start ephemeral PostgreSQL and moina').replace('moina-ci','moina-restore-0929')
version=(repo/'VERSION').read_text().strip()
env={**os.environ,'MOINA_VERSION':version}
def cleanup():
 subprocess.run(['docker','rm','-f','-v','moina-restore-0929-app','moina-restore-0929-postgres'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
 subprocess.run(['docker','network','rm','moina-restore-0929'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
def start():
 with (run/'startup.log').open('a') as f: subprocess.run(['bash','-c',startup],env=env,stdout=f,stderr=subprocess.STDOUT,check=True)
def browser(label,command,extra={}):
 target=run/label; target.mkdir(exist_ok=True)
 args=['docker','run','--rm','--network','host','--user',f'{os.getuid()}:{os.getgid()}','-e','HOME=/tmp','-v',f'{repo}/e2e:/work/e2e','-v',f'{target}:/work/e2e/test-results','-w','/work/e2e']
 settings={'MOINA_E2E_BASE_URL':'http://127.0.0.1:18080','MOINA_E2E_USERNAME':'ci-admin','MOINA_E2E_PASSWORD':'ci-password-12345','MOINA_E2E_VERSION':version,**extra}
 # The startup block's name substitution also affects its public test password.
 settings['MOINA_E2E_PASSWORD']='ci-password-12345'
 for k,v in settings.items(): args+=['-e',f'{k}={v}']
 args+=['mcr.microsoft.com/playwright:v1.62.1-noble','npm',*command]
 with (run/f'{label}.log').open('w') as f: result=subprocess.run(args,stdout=f,stderr=subprocess.STDOUT)
 print(f'{label}: exit {result.returncode}',flush=True)
 return result.returncode
try:
 start()
 assert browser('normal-results',['test'])==0, 'normal e2e failed; inspect normal-results.log'
 normal=run/'normal-results'
 for file in ['visual/visual-regression.json','accessibility-regression.json','browser-smoke.json']:
  assert json.loads((normal/file).read_text())['ok'] is True
 result=subprocess.run(['node','scripts/e2e-failure-summary.mjs',str(normal)],capture_output=True,text=True,check=True)
 (run/'normal-summary.md').write_text(result.stdout)
 assert 'e2e 결과 3개가 모두 통과로 기록' in result.stdout
 print('normal summary: all three passed',flush=True)
 cleanup(); start()
 visual_exit=browser('failure-results',['run','test:visual'],{'MOINA_VISUAL_MAX_DIFF_RATIO':'0'})
 visual=json.loads((run/'failure-results/visual/visual-regression.json').read_text())
 assert visual_exit!=0 and not visual['ok'] and visual['failures'], 'visual failure not observed'
 smoke_exit=browser('failure-results',['run','test:smoke'],{'MOINA_E2E_VERSION':'v0.0.0-invalid'})
 smoke=json.loads((run/'failure-results/browser-smoke.json').read_text())
 assert smoke_exit!=0 and not smoke['ok'] and 'browser-smoke.mjs:' in smoke['error']
 # Keep the passing accessibility producer result beside the two actual failures.
 (run/'failure-results/accessibility-regression.json').write_bytes((normal/'accessibility-regression.json').read_bytes())
 result=subprocess.run(['node','scripts/e2e-failure-summary.mjs',str(run/'failure-results')],capture_output=True,text=True,check=True)
 with (run/'failure-summary.md').open('a') as f: f.write(result.stdout)
 assert 'browser-smoke.mjs:' in result.stdout and 'locator' in result.stdout.lower()
 assert '### 접근성' not in result.stdout
 ratios={x['id']:x['diffRatio'] for x in visual['results']}
 largest=sorted(visual['failures'],key=lambda x:ratios[x['id']],reverse=True)[0]
 assert largest['id'] in result.stdout and largest['diff'] in result.stdout and f"{ratios[largest['id']]*100:.3f}%" in result.stdout
 print(f"PASS: actual failures summarized; visual {len(visual['failures'])}/{len(visual['results'])}; smoke locator and script frame; passed accessibility omitted",flush=True)
finally: cleanup()
