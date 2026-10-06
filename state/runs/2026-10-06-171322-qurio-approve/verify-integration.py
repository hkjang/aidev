import os, subprocess, time, pathlib, yaml
root=pathlib.Path.cwd()
run=pathlib.Path(__file__).parent
job=yaml.safe_load((root/'.github/workflows/release.yml').read_text())['jobs']['verify']
env=os.environ.copy()
env.update(job['env'])
containers=[]
def call(args, log, extra=None):
    with (run/log).open('w') as f:
        result=subprocess.run(args,env=extra or env,stdout=f,stderr=subprocess.STDOUT,timeout=1800)
    print(log, result.returncode,flush=True)
    if result.returncode: raise RuntimeError(log+' failed')
try:
    for kind,port in [('postgres',5432),('oracle',1521)]:
        service=job['services'][kind]
        name='qurio-v1414-verify-'+kind
        args=['docker','run','-d','--name',name,'-p','127.0.0.1::'+str(port)]
        if kind=='oracle': args+=['--shm-size','1g']
        for k,v in service['env'].items(): args+=['-e',k+'='+str(v)]
        args+=[service['image']]
        subprocess.run(args,check=True,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
        containers.append(name)
        assigned=subprocess.check_output(['docker','port',name,str(port)],text=True).strip().rsplit(':',1)[1]
        if kind=='postgres':
            for key in ['POSTGRES_DSN','QURIO_INTEGRATION_DSN','QURIO_TEST_POSTGRES_DSN']: env[key]=env[key].replace(':5432/',':'+assigned+'/')
        else:
            step=next(s for s in job['steps'] if s.get('name','').startswith('Verify live Oracle'))
            env.update({k:str(v) for k,v in step['env'].items()})
            env['QURIO_TEST_ORACLE_PORT']=assigned
        command=['pg_isready','-U','qurio'] if kind=='postgres' else ['healthcheck.sh']
        until=time.monotonic()+300
        while subprocess.run(['docker','exec',name]+command,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL).returncode:
            if time.monotonic()>until: raise RuntimeError(kind+' readiness timed out')
            time.sleep(3)
        print(kind+' ready',flush=True)
    env.pop('CGO_ENABLED',None)
    call(['go','test','-tags=integration','./cmd/qurio','-run','^TestIntegrationFreshInstallDatabase$','-count=1'],'fresh-db.log')
    call(['go','test','-tags=integration','./cmd/qurio','-run','^TestIntegrationDatabaseMigrations$','-count=1'],'migrations.log')
    oracle_env=env.copy(); oracle_env['CGO_ENABLED']='0'
    call(['go','test','-tags=integration','./internal/domain/dbexec','-run','^TestOracleLive','-count=1'],'oracle.log',oracle_env)
    call(['go','test','-race','-p=1','-tags=integration','./...'],'go-tests.log')
    call(['go','vet','-tags=integration','./...'],'go-vet.log')
finally:
    for name in reversed(containers): subprocess.run(['docker','rm','-f',name],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
