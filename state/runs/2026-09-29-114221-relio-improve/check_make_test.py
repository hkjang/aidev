import subprocess
output = subprocess.check_output(["make", "-n", "test"], text=True)
print(output, end="")
assert "cd web && npm ci && npm run typecheck && npm test" in output, "make test omits frontend regression tests"
