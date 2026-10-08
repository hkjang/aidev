from pathlib import Path
from html.parser import HTMLParser
import sqlite3
import subprocess
import time

class CodeParser(HTMLParser):
    def __init__(self):
        super().__init__()
        self.inside = False
        self.items = []
    def handle_starttag(self, tag, attrs):
        if tag == "code":
            self.inside = True
            self.items.append("")
    def handle_endtag(self, tag):
        if tag == "code":
            self.inside = False
    def handle_data(self, data):
        if self.inside:
            self.items[-1] += data

parser = CodeParser()
parser.feed((Path(__file__).parent / "pages-site/RELEASE_NOTES_v0.2.45.html").read_text())
predicate, = [x for x in parser.items if x.startswith("WHERE resource_id LIKE")]
query = "WITH samples(resource_id) AS (VALUES ('{abc}'), ('{'), ('{}'), ('abc'), (''), ('x{abc}'), (NULL)) SELECT resource_id FROM samples " + predicate + " ORDER BY resource_id"
rows = sqlite3.connect(":memory:").execute(query).fetchall()
assert rows == [("{",), ("{abc}",), ("{}",)], rows
print("SQLite: rendered SQL selects exactly the three brace-prefixed values.")
container = "invenqor-pages-sql-20261008-115213"
subprocess.run(["docker", "run", "--rm", "-d", "--name", container, "--network", "none", "-e", "POSTGRES_HOST_AUTH_METHOD=trust", "postgres:17-alpine"], check=True, stdout=subprocess.DEVNULL)
try:
    for attempt in range(30):
        if subprocess.run(["docker", "exec", container, "pg_isready", "-h", "127.0.0.1", "-U", "postgres"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL).returncode == 0:
            break
        time.sleep(1)
    else:
        raise RuntimeError("PostgreSQL did not become ready")
    result = subprocess.run(["docker", "exec", "-i", container, "psql", "-h", "127.0.0.1", "-U", "postgres", "-At", "-v", "ON_ERROR_STOP=1"], input=query + ";", text=True, capture_output=True, check=True)
    assert result.stdout.splitlines() == ["{", "{abc}", "{}"], result.stdout
    print("PostgreSQL: the same rendered SQL selects exactly the three brace-prefixed values.")
finally:
    subprocess.run(["docker", "rm", "-f", container], check=True, stdout=subprocess.DEVNULL)
