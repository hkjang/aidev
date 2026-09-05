"""전체 흐름 모의 실행 — 가짜 GitHub/에이전트로 러너를 돌려 차단·게시가 올바른지 확인한다.
실행: python3 tests/test_sim.py   (각 시나리오 10~40초; 실제 상태·원격은 건드리지 않는다)"""
import json
import os
import subprocess
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
SIM = os.path.join(HERE, "sim", "run_sim.sh")


def run(scenario):
    with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as f:
        json.dump(scenario, f)
        path = f.name
    out = subprocess.run(["bash", SIM, path], capture_output=True, text=True, timeout=900)
    os.unlink(path)
    lines = [l for l in out.stdout.splitlines() if l.startswith("{")]
    assert lines, f"no summary: {out.stdout}\n{out.stderr}"
    res = json.loads(lines[0])
    res["asset_lines"] = [json.loads(l) for l in lines[1:]]
    return res


BASE = {"ci": "success", "review": "approve", "release": {"tag": "v0.0.2", "assets": ["simproj-{tag}.tar.gz"], "github_release": True},
        "prev_release_assets": ["simproj-v0.0.1.tar.gz"]}


def scn(**kw):
    s = json.loads(json.dumps(BASE))
    for k, v in kw.items():
        if isinstance(v, dict) and isinstance(s.get(k), dict):
            s[k].update(v)
        else:
            s[k] = v
    return s


class HappyPath(unittest.TestCase):
    def test_full_flow_release_ready(self):
        r = run(scn())
        self.assertEqual(r["outcome"], "release-ready", r)
        self.assertEqual(r["stages"].get("verify"), "passed")
        self.assertEqual(r["stages"].get("review"), "approved")
        self.assertEqual(r["stages"].get("ci"), "passed")
        self.assertEqual(r["stages"].get("merge"), "done")
        self.assertIn("v0.0.2", r["remote_tags"])
        self.assertEqual(r["prs"][0]["state"], "MERGED")
        names = " ".join(a["names"] for a in r["asset_lines"] if a["release_assets"] == "v0.0.2")
        self.assertIn("simproj-v0.0.2.tar.gz", names)


class Blocking(unittest.TestCase):
    def test_ci_api_error_blocks_merge(self):
        r = run(scn(ci="api-error"))
        self.assertEqual(r["outcome"], "review-pending", r)
        self.assertEqual(r["prs"][0]["state"], "OPEN")
        self.assertIn("api-error", r["stages"].get("ci", ""))

    def test_ci_failure_blocks_merge(self):
        r = run(scn(ci="failure"))
        self.assertEqual(r["outcome"], "verify-failed", r)
        self.assertEqual(r["prs"][0]["state"], "OPEN")

    def test_ci_pending_blocks_merge(self):
        r = run(scn(ci="pending", args="--no-release"))
        self.assertEqual(r["prs"][0]["state"], "OPEN")
        self.assertNotEqual(r["stages"].get("merge"), "done")

    def test_no_ci_without_policy_blocks(self):
        r = run(scn(ci="none"))
        self.assertEqual(r["prs"][0]["state"], "OPEN")
        self.assertEqual(r["stages"].get("ci"), "no-ci")

    def test_no_ci_with_policy_allows(self):
        r = run(scn(ci="none", allow_merge_without_ci=True))
        self.assertEqual(r["stages"].get("merge"), "done", r)

    def test_review_missing_holds(self):
        r = run(scn(review="missing"))
        self.assertEqual(r["prs"][0]["state"], "OPEN")
        self.assertEqual(r["stages"].get("review"), "missing")

    def test_review_invalid_or_broken_holds(self):
        for mode in ("invalid", "broken"):
            r = run(scn(review=mode))
            self.assertEqual(r["prs"][0]["state"], "OPEN", mode)
            self.assertNotEqual(r["stages"].get("merge"), "done", mode)

    def test_review_reject_holds(self):
        r = run(scn(review="reject"))
        self.assertEqual(r["stages"].get("review"), "rejected")
        self.assertEqual(r["prs"][0]["state"], "OPEN")

    def test_runner_verify_failure_no_pr(self):
        r = run(scn(verify=["false"]))
        self.assertEqual(r["outcome"], "verify-failed")
        self.assertEqual(r["prs"], [])

    def test_secret_in_diff_no_pr(self):
        r = run(scn(improve={"secret": True}))
        self.assertEqual(r["outcome"], "verify-failed")
        self.assertIn("secrets", r["result"])
        self.assertEqual(r["prs"], [])

    def test_guarded_file_holds(self):
        r = run(scn(improve={"file": ".github/workflows/ci.yml", "content": "on: push"}))
        self.assertEqual(r["stages"].get("guard"), "held")
        self.assertEqual(r["prs"][0]["state"], "OPEN")

    def test_stop_merge(self):
        r = run(scn(stops=["merge"]))
        self.assertEqual(r["stages"].get("merge"), "stopped")
        self.assertEqual(r["prs"][0]["state"], "OPEN")

    def test_stop_all_no_round(self):
        r = run(scn(stops=["all"]))
        self.assertEqual(r["prs"], [])
        self.assertIsNone(r.get("outcome"))


class Autonomy(unittest.TestCase):
    def test_pr_level_never_merges(self):
        r = run(scn(autonomy="pr"))
        self.assertEqual(r["prs"][0]["state"], "OPEN")
        self.assertIn("needs approval", r["result"])

    def test_low_risk_merges_but_no_release(self):
        r = run(scn(autonomy="low-risk"))
        self.assertEqual(r["stages"].get("merge"), "done")
        self.assertEqual(r["stages"].get("release"), "skipped")
        self.assertNotIn("v0.0.2", r["remote_tags"])

    def test_low_risk_high_risk_review_holds(self):
        r = run(scn(autonomy="low-risk", review_risk="high"))
        self.assertNotEqual(r["stages"].get("merge"), "done")

    def test_analyze_discards_commits(self):
        r = run(scn(autonomy="analyze"))
        self.assertEqual(r["prs"], [])
        self.assertIn("analyze-only", r["result"])


class ReleaseSafety(unittest.TestCase):
    def test_tag_conflict_does_not_fail(self):
        r = run(scn(tag_exists=True))
        self.assertEqual(r["stages"].get("merge"), "done")
        self.assertIn("v0.0.2", r["remote_tags"])
        self.assertEqual(r["stages"].get("release"), "published")

    def test_asset_missing_is_manifest_failure(self):
        r = run(scn(release={"assets": []}))
        self.assertEqual(r["outcome"], "releasing", r)
        self.assertIn("ASSETS MISSING", r["result"])

    def test_asset_outside_out_dir_rejected(self):
        r = run(scn(release={"asset_outside": True}))
        self.assertIn(r["stages"].get("release"), ("bad-assets",))
        self.assertNotIn("v0.0.2", r["remote_tags"])

    def test_wrong_asset_name_fails_manifest(self):
        r = run(scn(release={"assets": ["other-{tag}.zip"]}))
        self.assertEqual(r["stages"].get("manifest"), "failed")

    def test_release_result_missing_blocks(self):
        r = run(scn(release={"status": "missing"}))
        self.assertEqual(r["stages"].get("release"), "missing")
        self.assertNotIn("v0.0.2", r["remote_tags"])

    def test_duplicate_pr_not_created(self):
        # 같은 브랜치의 PR 이 이미 있으면(재개 상황) 두 번째 PR 을 만들지 않는다
        r = run(scn(args="--no-release", ci="pending"))
        self.assertEqual(len(r["prs"]), 1)

    def test_transient_merge_error_is_retried(self):
        r = run(scn(merge_fail="network", args="--no-release"))
        self.assertEqual(r["stages"].get("merge"), "done", r)


if __name__ == "__main__":
    unittest.main(verbosity=2)
