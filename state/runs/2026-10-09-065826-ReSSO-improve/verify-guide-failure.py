import os
from pathlib import Path
import subprocess
import sys

run = Path(__file__).parent
label = sys.argv[1]
bad_certs = run / "nonexistent-certs"
assert not bad_certs.exists()
for container in ("resso-test-pg", "resso-test-ldap", "resso-test-ldaps"):
    state = subprocess.check_output(["docker", "inspect", "-f", "{{.State.Status}}", container], text=True).strip()
    assert state == "running", (container, state)
block = Path("docs/user-federation.md").read_text().split("```bash\n", 1)[1].split("```", 1)[0]
# Observe the follow-up without running tests or the old destructive cleanup.
lines = block.splitlines()
test_index = next(i for i, line in enumerate(lines) if line.strip() == "go test ./internal/...")
probe = "\n".join(lines[:test_index] + ["  printf 'FOLLOWUP_RAN\\n'"]) + "\n"
(run / (label + "-probe.sh")).write_text(probe)
result = subprocess.run(["bash", "--noprofile", "--norc", "-c", probe], env={**os.environ, "RESSO_TEST_CERT_DIR": str(bad_certs)}, text=True, capture_output=True)
(run / (label + ".out")).write_text(result.stdout)
(run / (label + ".err")).write_text(result.stderr)
(run / (label + ".status")).write_text(str(result.returncode) + "\n")
assert "ca.crt is not a readable regular file" in result.stderr, result.stderr
if result.returncode == 0 or result.stdout:
    print(f"FAIL: preparation example exited {result.returncode}; expected nonzero and no follow-up")
    print(result.stdout.strip())
    sys.exit(1)
print(f"PASS: preparation example exited {result.returncode}; no follow-up; missing CA diagnostic confirmed")
