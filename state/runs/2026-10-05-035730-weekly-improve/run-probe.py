import json, os, subprocess, urllib.parse
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
(r / "probe-overlay.json").write_text(json.dumps({"Replace": {str(Path.cwd() / "internal/app/scout_attachmentcleanup_test.go"): str(r / "attachmentcleanup_probe_test.go")}}))
result = subprocess.run(["go", "test", "-overlay", str(r / "probe-overlay.json"), "./internal/app", "-run", "TestScoutAttachmentCleanupDuringFirstUpload|TestARefusedAttachmentUploadStoresNoneOfTheImages", "-count=1", "-v", "-timeout", "60s"], env=env)
raise SystemExit(result.returncode)
