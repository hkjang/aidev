import json, os, subprocess, sys, urllib.parse
from pathlib import Path
base = Path(__file__).resolve().parent
info = json.loads(subprocess.check_output(["docker", "inspect", "weekly-test-pg"]))[0]
v = dict(x.split("=", 1) for x in info["Config"]["Env"] if "=" in x)
port = info["NetworkSettings"]["Ports"]["5432/tcp"][0]["HostPort"]
q = urllib.parse.quote
env = os.environ.copy()
env["WEEKLY_TEST_POSTGRES_DSN"] = "postgres://" + q(v.get("POSTGRES_USER", "postgres"), safe="") + ":" + q(v["POSTGRES_PASSWORD"], safe="") + "@127.0.0.1:" + port + "/" + q(v.get("POSTGRES_DB", v.get("POSTGRES_USER", "postgres")), safe="") + "?sslmode=disable"
command = sys.argv[1:] or ["go", "test", "-overlay", str(base / "overlay.json"), "./internal/app", "-run", "^(TestScoutUploadOrderQueryOverflow|TestASentSortOrderIsStoredAsSent)$", "-count=1", "-v"]
sys.exit(subprocess.run(command, env=env, timeout=360).returncode)
