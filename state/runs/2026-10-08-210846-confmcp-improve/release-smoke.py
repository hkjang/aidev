import json, os, re, subprocess, time
from pathlib import Path
from urllib.parse import quote

def capture(args):
    return subprocess.check_output(args, stderr=subprocess.PIPE).decode().strip()

info=json.loads(capture(['docker','inspect','confmcp-dev-pg']))[0]
pgenv=dict(v.split('=',1) for v in info['Config']['Env'] if '=' in v)
user=pgenv['POSTGRES_USER']
db='confmcp_release_011_smoke'
container='confmcp-release-011-smoke'
image='confmcp:v0.1.1'
commit=capture(['git','rev-parse','--short','HEAD'])
fixture=Path('scripts/dev.sh').read_text()
env=os.environ.copy()
for key in ['BOOTSTRAP_ADMIN','BOOTSTRAP_ADMIN_PASSWORD','ENCRYPTION_KEY']:
    env[key]=re.search(r'\b'+key+r'=([^\s]+)',fixture).group(1)
env['DATABASE_URL']='postgres://'+quote(user,safe='')+':'+quote(pgenv['POSTGRES_PASSWORD'],safe='')+'@127.0.0.1:5432/'+db+'?sslmode=disable'
subprocess.run(['docker','exec','confmcp-dev-pg','createdb','-U',user,db],check=True,stdout=subprocess.DEVNULL)
try:
    args=['docker','run','-d','--name',container,'--network','container:confmcp-dev-pg']
    for key in ['DATABASE_URL','BOOTSTRAP_ADMIN','BOOTSTRAP_ADMIN_PASSWORD','ENCRYPTION_KEY']:
        args+=['-e',key]
    args+=[image]
    subprocess.run(args,env=env,check=True,stdout=subprocess.DEVNULL,stderr=subprocess.PIPE)
    for attempt in range(60):
        result=subprocess.run(['docker','exec',container,'curl','-fsS','http://127.0.0.1:8080/healthz'],stdout=subprocess.PIPE,stderr=subprocess.DEVNULL)
        if result.returncode==0: break
        time.sleep(1)
    else: raise RuntimeError('healthz readiness timed out')
    for path in ['/healthz','/readyz','/api/config','/']:
        status=capture(['docker','exec',container,'curl','-fsS','-o','/dev/null','-w','%{http_code}','http://127.0.0.1:8080'+path])
        assert status=='200',(path,status)
        print(path,status,flush=True)
    version=json.loads(capture(['docker','exec',container,'curl','-fsS','http://127.0.0.1:8080/api/version']))
    assert version['version']=='0.1.1', version
    assert version['commit']==commit, version
    assert version['buildDate'] not in ('unknown',''), version
    label=capture(['docker','image','inspect',image,'--format','{{index .Config.Labels "org.opencontainers.image.version"}}'])
    assert label=='0.1.1',label
    headers=capture(['docker','exec',container,'curl','-fsS','-D','-','-o','/dev/null','http://127.0.0.1:8080/.well-known/oauth-protected-resource/mcp'])
    assert 'x-confmcp-version: 0.1.1' in headers.lower()
    html=capture(['docker','exec',container,'curl','-fsS','http://127.0.0.1:8080/'])
    for asset in re.findall(r'(?:src|href)="(/assets/[^\"]+)"',html):
        status=capture(['docker','exec',container,'curl','-fsS','-o','/dev/null','-w','%{http_code}','http://127.0.0.1:8080'+asset])
        assert status=='200'
        print('embedded asset',asset,status,flush=True)
    print('version metadata:',json.dumps(version),flush=True)
    print('image label and OAuth version header: PASS',flush=True)
finally:
    subprocess.run(['docker','rm','-f',container],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
    subprocess.run(['docker','exec','confmcp-dev-pg','dropdb','-U',user,db],check=True,stdout=subprocess.DEVNULL)
