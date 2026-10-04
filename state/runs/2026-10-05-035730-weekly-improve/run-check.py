import json, os, subprocess, sys, urllib.parse
from pathlib import Path
r = Path(__file__).resolve().parent
info = json.loads(subprocess.check_output(["docker", "inspect", "weekly-test-pg"]))[0]
settings = dict(x.split("=", 1) for x in info["Config"]["Env"] if "=" in x)
quote = lambda v: urllib.parse.quote(v, safe="")
user = quote(settings.get("POSTGRES_USER", "postgres"))
password = quote(settings["POSTGRES_PASSWORD"])
database = quote(settings.get("POSTGRES_DB", "postgres"))
env = os.environ.copy()
env["WEEKLY_TEST_POSTGRES_DSN"] = f"postgres://{user}:{password}@127.0.0.1:15434/{database}?sslmode=disable"
with (r / sys.argv[1]).open("w") as log:
    command = sys.argv[2:]
    print("COMMAND:", " ".join(command), file=log, flush=True)
    process = subprocess.Popen(command, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    for line in process.stdout:
        print(line, end="", flush=True)
        log.write(line)
        log.flush()
    code = process.wait()
    print(f"EXIT={code}", flush=True)
    print(f"EXIT={code}", file=log)
raise SystemExit(code)
