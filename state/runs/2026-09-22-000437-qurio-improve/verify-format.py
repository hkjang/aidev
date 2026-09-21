import hashlib
import shutil
import subprocess
import tempfile
from pathlib import Path

repo = Path.cwd()
roots = ("cmd", "internal", "migrations", "scripts")
def hashes(base):
    return {str(p.relative_to(base)): hashlib.sha256(p.read_bytes()).hexdigest()
            for root in roots for p in (base / root).rglob("*") if p.is_file()}
def check(base, target, failure=False, path=None):
    before = hashes(base)
    result = subprocess.run(["make", "-C", str(base), target], capture_output=True, text=True)
    output = result.stdout + result.stderr
    assert hashes(base) == before, "files modified"
    assert (result.returncode != 0) == failure, output
    if path:
        assert path in output, output
    if failure and target == "lint":
        assert "go vet" not in output and "npm --prefix" not in output, output
        assert "check-go-format" in output, output
    print(f"PASS {target} failure={failure} path={path}: SHA-256 unchanged")
    if failure:
        print(output.strip())
    return output

with tempfile.TemporaryDirectory(prefix="qurio-format-") as temp:
    base = Path(temp)
    for name in ("Makefile", "VERSION"):
        shutil.copy2(repo / name, base / name)
    for root in roots:
        shutil.copytree(repo / root, base / root)
    check(base, "check-go-format")
    for root in roots:
        for name, content in (
            ("format_fixture.go", "package fixture;func example( ){ }\n"),
            ("format_fixture_test.go", "package fixture;func example( ){ }\n"),
            ("format_fixture_integration_test.go", "//go:build integration\n\npackage fixture;func example( ){ }\n"),
        ):
            relative = f"{root}/{name}"
            fixture = base / relative
            fixture.write_text(content)
            output = check(base, "check-go-format", True, relative)
            assert "gofmt -w" in output, output
            check(base, "lint", True, relative)
            fixture.unlink()
        relative = f"{root}/syntax_fixture.go"
        fixture = base / relative
        fixture.write_text("package fixture\nfunc broken( {\n")
        check(base, "check-go-format", True, relative)
        check(base, "lint", True, relative)
        fixture.unlink()
    check(base, "check-go-format")
