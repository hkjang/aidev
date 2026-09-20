import subprocess, json
from pathlib import Path
p=Path(__file__).parent
commands = [
("focused", "npm --prefix web run test:e2e -- core.spec.ts -g '공개 앱 즐겨찾기는|게시되지 않은 내 앱'"),
("e2e", "CI=true npm --prefix web run test:e2e"),
("react-final", "npm --prefix web test"),
("lint", "npm --prefix web run lint"),
("prettier", "cd web && npx prettier --check e2e/core.spec.ts"),
("offline", "./scripts/check-offline-assets.sh web/dist"),
("env", "./scripts/check-env-contract.sh"),
("docs", "./scripts/check-docs.sh"),
("diff", "git diff --check"),
]
results=[]
for name, command in commands:
    with (p / ("impl-"+name+".log")).open("w") as f:
        r=subprocess.run(command, shell=True, stdout=f, stderr=subprocess.STDOUT)
    results.append(dict(name=name, command=command, exit_code=r.returncode))
    (p/"impl-check-results.json").write_text(json.dumps(results, ensure_ascii=False, indent=2)+"\n")
    print(name, r.returncode, flush=True)
    if r.returncode: raise SystemExit(r.returncode)
