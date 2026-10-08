import json, os, subprocess, sys, urllib.parse
info=json.loads(subprocess.check_output(["docker","inspect","weekly-test-pg"]))[0]
v=dict(x.split("=",1) for x in info["Config"]["Env"] if "=" in x)
user=v.get("POSTGRES_USER","postgres")
port=info["NetworkSettings"]["Ports"]["5432/tcp"][0]["HostPort"]
db="weekly_release_0319_"+str(os.getpid())
q=urllib.parse.quote
env=os.environ.copy()
env.pop("TMPDIR",None)
env["WEEKLY_TEST_POSTGRES_DSN"]="postgres://"+q(user,safe="")+":"+q(v["POSTGRES_PASSWORD"],safe="")+"@127.0.0.1:"+port+"/"+db+"?sslmode=disable"
subprocess.run(["docker","exec","weekly-test-pg","createdb","-U",user,db],check=True)
try:
 result=subprocess.run(sys.argv[1:],env=env,timeout=1600)
finally:
 subprocess.run(["docker","exec","weekly-test-pg","dropdb","-U",user,db],check=True)
sys.exit(result.returncode)
