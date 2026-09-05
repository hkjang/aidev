"""gate.py 회귀 테스트 — 모든 비정상 응답에서 진행이 차단되어야 한다.  실행: python3 -m pytest tests/  (또는 python3 tests/test_gate.py)"""
import json
import os
import subprocess
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, "..", "bin"))
import gate  # noqa: E402

SHA = "a" * 40


def run(name, status="completed", conclusion="success", sha=SHA, started="2026-09-05T00:00:00Z", rid=1):
    return {"id": rid, "name": name, "status": status, "conclusion": conclusion, "head_sha": sha, "started_at": started}


class CI(unittest.TestCase):
    def ok(self, data, **kw):
        return gate.evaluate_ci(data, SHA, **kw)[0]

    def test_all_success(self):
        self.assertTrue(self.ok({"check_runs": [run("CI"), run("lint", rid=2)]}))

    def test_failure_blocks(self):
        ok, state, _, _ = gate.evaluate_ci({"check_runs": [run("CI", conclusion="failure")]}, SHA)
        self.assertFalse(ok); self.assertEqual(state, "failed")

    def test_cancelled_and_timed_out_block(self):
        self.assertFalse(self.ok({"check_runs": [run("CI", conclusion="cancelled")]}))
        ok, state, _, _ = gate.evaluate_ci({"check_runs": [run("CI", conclusion="timed_out")]}, SHA)
        self.assertFalse(ok); self.assertEqual(state, "timed_out")

    def test_in_progress_blocks(self):
        ok, state, _, _ = gate.evaluate_ci({"check_runs": [run("CI", status="in_progress", conclusion=None)]}, SHA)
        self.assertFalse(ok); self.assertEqual(state, "pending")

    def test_queued_blocks(self):
        self.assertFalse(self.ok({"check_runs": [run("CI"), run("build", status="queued", conclusion=None, rid=2)]}))

    def test_empty_response_blocks_unless_policy(self):
        ok, state, _, _ = gate.evaluate_ci({"check_runs": []}, SHA)
        self.assertFalse(ok); self.assertEqual(state, "no-ci")
        self.assertTrue(self.ok({"check_runs": []}, allow_no_ci=True))

    def test_api_error_blocks(self):
        ok, state, _, _ = gate.evaluate_ci({"message": "Not Found", "documentation_url": "x"}, SHA)
        self.assertFalse(ok); self.assertEqual(state, "api-error")

    def test_parse_error_blocks(self):
        self.assertFalse(self.ok({"foo": 1}))
        self.assertFalse(self.ok({"check_runs": "nope"}))
        self.assertFalse(self.ok("garbage"))

    def test_other_commit_blocks(self):
        ok, state, _, _ = gate.evaluate_ci({"check_runs": [run("CI", sha="b" * 40)]}, SHA)
        self.assertFalse(ok); self.assertEqual(state, "sha-mismatch")

    def test_required_missing_blocks(self):
        ok, state, _, _ = gate.evaluate_ci({"check_runs": [run("lint")]}, SHA, required=["CI"])
        self.assertFalse(ok); self.assertEqual(state, "required-missing")

    def test_required_only_considers_required(self):
        # 필수 검사가 성공이면 부가 검사의 실패는 무시하지 않는다? — 정책: required 를 지정하면 그것만 본다
        self.assertTrue(self.ok({"check_runs": [run("CI"), run("optional", conclusion="failure", rid=2)]}, required=["CI"]))
        self.assertFalse(self.ok({"check_runs": [run("CI", conclusion="failure"), run("optional", rid=2)]}, required=["CI"]))

    def test_pagination_merges_pages(self):
        pages = [{"check_runs": [run("CI")]}, {"check_runs": [run("build", status="in_progress", conclusion=None, rid=2)]}]
        self.assertFalse(self.ok(pages))
        pages = [{"check_runs": [run("CI")]}, {"check_runs": [run("build", rid=2)]}]
        self.assertTrue(self.ok(pages))

    def test_rerun_uses_latest_attempt(self):
        old = run("CI", conclusion="failure", started="2026-09-05T00:00:00Z", rid=1)
        new = run("CI", conclusion="success", started="2026-09-05T01:00:00Z", rid=2)
        self.assertTrue(self.ok({"check_runs": [old, new]}))
        self.assertFalse(self.ok({"check_runs": [new | {"conclusion": "failure", "started_at": "2026-09-05T02:00:00Z", "id": 3}, old, new]}))

    def test_neutral_and_skipped_are_ok(self):
        self.assertTrue(self.ok({"check_runs": [run("CI"), run("docs", conclusion="skipped", rid=2), run("cov", conclusion="neutral", rid=3)]}))

    def test_action_required_blocks(self):
        self.assertFalse(self.ok({"check_runs": [run("CI", conclusion="action_required")]}))


class Review(unittest.TestCase):
    def test_only_valid_approve_passes(self):
        self.assertTrue(gate.evaluate_review({"verdict": "approve", "reasons": [], "risk": "low"})[0])
        self.assertFalse(gate.evaluate_review({"verdict": "reject", "reasons": ["x"]})[0])
        self.assertFalse(gate.evaluate_review({"verdict": "approved"})[0])
        self.assertFalse(gate.evaluate_review({"verdict": "APPROVE"})[0])
        self.assertFalse(gate.evaluate_review({})[0])
        self.assertFalse(gate.evaluate_review([])[0])
        self.assertFalse(gate.evaluate_review(None)[0])


class Release(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.mkdtemp()
        self.asset = os.path.join(self.tmp, "app-v1.tar.gz")
        open(self.asset, "wb").write(b"x" * 10)

    def test_valid(self):
        ok, st, _, d = gate.evaluate_release({"status": "released", "tag": "v1.2.3", "github_release": False, "assets": [self.asset]}, self.tmp)
        self.assertTrue(ok); self.assertEqual(d["assets_ok"], [os.path.realpath(self.asset)])

    def test_unknown_status_blocks(self):
        self.assertFalse(gate.evaluate_release({"status": "done"}, self.tmp)[0])
        self.assertFalse(gate.evaluate_release({"status": "skipped"}, self.tmp)[0])

    def test_asset_outside_dir_blocks(self):
        self.assertFalse(gate.evaluate_release({"status": "released", "tag": "v1", "assets": ["/etc/passwd"]}, self.tmp)[0])
        self.assertFalse(gate.evaluate_release({"status": "released", "tag": "v1", "assets": [self.tmp + "/../x.tar.gz"]}, self.tmp)[0])

    def test_missing_or_empty_asset_blocks(self):
        self.assertFalse(gate.evaluate_release({"status": "released", "tag": "v1", "assets": [self.tmp + "/nope.tgz"]}, self.tmp)[0])
        empty = os.path.join(self.tmp, "empty.tgz"); open(empty, "wb").close()
        self.assertFalse(gate.evaluate_release({"status": "released", "tag": "v1", "assets": [empty]}, self.tmp)[0])

    def test_bad_tag_or_types_block(self):
        self.assertFalse(gate.evaluate_release({"status": "released", "tag": "v1; rm -rf /", "assets": []}, self.tmp)[0])
        self.assertFalse(gate.evaluate_release({"status": "released", "tag": "v1", "github_release": "yes", "assets": []}, self.tmp)[0])
        self.assertFalse(gate.evaluate_release({"status": "released", "tag": "v1", "assets": "a.tgz"}, self.tmp)[0])


class Ideas(unittest.TestCase):
    def test_schema(self):
        good = [{"title": "t", "value": 3, "risk": 1, "size": "S", "status": "pending"}]
        self.assertTrue(gate.evaluate_ideas(good)[0])
        self.assertFalse(gate.evaluate_ideas({"title": "t"})[0])
        self.assertFalse(gate.evaluate_ideas([{"title": "", "value": 3, "risk": 1, "size": "S", "status": "pending"}])[0])
        self.assertFalse(gate.evaluate_ideas([{"title": "t", "value": 9, "risk": 1, "size": "S", "status": "pending"}])[0])
        self.assertFalse(gate.evaluate_ideas([{"title": "t", "value": 3, "risk": 1, "size": "XL", "status": "pending"}])[0])
        self.assertFalse(gate.evaluate_ideas([{"title": "t", "value": 3, "risk": 1, "size": "S", "status": "todo"}])[0])


class Secrets(unittest.TestCase):
    def test_detects(self):
        self.assertTrue(gate.scan_secrets("token: ghp_" + "A" * 36))
        self.assertTrue(gate.scan_secrets("AKIAABCDEFGHIJKLMNOP"))
        self.assertTrue(gate.scan_secrets("-----BEGIN RSA PRIVATE KEY-----"))
        self.assertTrue(gate.scan_secrets('password = "hunter2hunter2"'))
        self.assertTrue(gate.scan_secrets("postgres://user:pw@db.example.com/x"))
        self.assertTrue(gate.scan_secrets("host 10.0.3.7 down", internal=True))

    def test_clean(self):
        self.assertFalse(gate.scan_secrets("fix: handle empty list in parser; password field renamed"))
        self.assertFalse(gate.scan_secrets("see https://github.com/hkjang/aidev/pull/1"))
        self.assertFalse(gate.scan_secrets("host 10.0.3.7 down"))  # internal 옵션 없으면 사설 IP 는 통과


class Verify(unittest.TestCase):
    def test_states(self):
        self.assertTrue(gate.evaluate_verify({"source": "policy", "commands": [{"cmd": "go test ./...", "exit": 0}]})[0])
        self.assertFalse(gate.evaluate_verify({"source": "policy", "commands": [{"cmd": "go test ./...", "exit": 1}]})[0])
        self.assertFalse(gate.evaluate_verify({"source": "none", "commands": []})[0])
        self.assertFalse(gate.evaluate_verify({"source": "auto", "commands": [{"cmd": "x", "exit": None}]})[0])
        self.assertFalse(gate.evaluate_verify("x")[0])


class CLI(unittest.TestCase):
    def test_exit_codes_and_missing_files(self):
        g = os.path.join(HERE, "..", "bin", "gate.py")
        r = subprocess.run([sys.executable, g, "review", "/nonexistent/review.json"], capture_output=True, text=True)
        self.assertEqual(r.returncode, 1); self.assertEqual(json.loads(r.stdout)["state"], "missing")
        with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as f:
            f.write("{not json"); path = f.name
        r = subprocess.run([sys.executable, g, "ci", path, "--sha", SHA], capture_output=True, text=True)
        self.assertEqual(r.returncode, 1)
        with open(path, "w") as f:
            json.dump({"check_runs": [run("CI")]}, f)
        r = subprocess.run([sys.executable, g, "ci", path, "--sha", SHA], capture_output=True, text=True)
        self.assertEqual(r.returncode, 0); self.assertTrue(json.loads(r.stdout)["ok"])
        r = subprocess.run([sys.executable, g, "secrets", "-"], input="ghp_" + "B" * 36, capture_output=True, text=True)
        self.assertEqual(r.returncode, 1)


if __name__ == "__main__":
    unittest.main(verbosity=1)
