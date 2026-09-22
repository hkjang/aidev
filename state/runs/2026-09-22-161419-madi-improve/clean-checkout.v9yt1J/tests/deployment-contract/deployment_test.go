package deploymentcontract

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func read(t *testing.T, name string) []byte {
	t.Helper()
	data, e := os.ReadFile(filepath.Join("..", "..", name))
	if e != nil {
		t.Fatal(e)
	}
	return data
}
func TestDeploymentContracts(t *testing.T) {
	version := strings.TrimSpace(string(read(t, "VERSION")))
	var compose struct {
		Services map[string]struct {
			Image       string            `yaml:"image"`
			Pull        string            `yaml:"pull_policy"`
			Environment map[string]string `yaml:"environment"`
			ReadOnly    bool              `yaml:"read_only"`
			Tmpfs       []string          `yaml:"tmpfs"`
		} `yaml:"services"`
	}
	if e := yaml.Unmarshal(read(t, "compose.yaml"), &compose); e != nil {
		t.Fatal(e)
	}
	if len(compose.Services) != 1 {
		t.Fatal("release compose must contain only the service, not dependency images")
	}
	s := compose.Services["madi"]
	if s.Image != "madi:v"+version || s.Pull != "never" || !s.ReadOnly {
		t.Fatal("offline image/tag/read-only contract changed")
	}
	if len(s.Environment) != 4 {
		t.Fatal("service environment must contain exactly four bootstrap fields")
	}
	for _, key := range []string{"POSTGRES_DSN", "BOOTSTRAP_ADMIN", "BOOTSTRAP_ADMIN_PASSWORD", "ENCRYPTION_KEY"} {
		if s.Environment[key] == "" {
			t.Fatalf("missing %s", key)
		}
	}
	if len(s.Tmpfs) != 1 || !strings.Contains(s.Tmpfs[0], "size=2g") {
		t.Fatal("bounded staging space must cover supported imports/restores")
	}
	for _, name := range []string{"deploy/kubernetes.yaml", ".github/workflows/ci.yml", ".github/workflows/release.yml", ".github/workflows/pages.yml"} {
		decoder := yaml.NewDecoder(bytes.NewReader(read(t, name)))
		for {
			var document map[string]any
			e := decoder.Decode(&document)
			if e == io.EOF {
				break
			}
			if e != nil {
				t.Fatalf("%s: %v", name, e)
			}
		}
	}
	dockerfile := string(read(t, "Dockerfile"))
	goVersion := ""
	for _, line := range strings.Split(string(read(t, "go.mod")), "\n") {
		if fields := strings.Fields(line); len(fields) == 2 && fields[0] == "go" {
			goVersion = fields[1]
		}
	}
	if goVersion == "" || !strings.Contains(dockerfile, "FROM golang:"+goVersion+"-alpine") {
		t.Fatal("Docker builder must pin the same patched Go version as go.mod")
	}
	for _, part := range []string{"CGO_ENABLED=0", "-X main.version=", "USER 10001:10001", `ENTRYPOINT ["/usr/local/bin/madi"]`} {
		if !strings.Contains(dockerfile, part) {
			t.Fatalf("Dockerfile missing %s", part)
		}
	}
	release := string(read(t, ".github/workflows/release.yml"))
	for _, part := range []string{"go test -race", "scripts/verify-image.sh", `"dist/madi-${GITHUB_REF_NAME}.tar.gz"`, "--verify-tag"} {
		if !strings.Contains(release, part) {
			t.Fatalf("release verification missing %s", part)
		}
	}
	for _, name := range []string{".github/workflows/ci.yml", ".github/workflows/release.yml"} {
		workflow := string(read(t, name))
		for _, part := range []string{"scripts/licenses.mjs --check", "scripts/verify-browser.sh", "scripts/build-docs.mjs", "tests/docs.mjs", "golang.org/x/vuln/cmd/govulncheck@v1.7.0"} {
			if !strings.Contains(workflow, part) {
				t.Fatalf("%s missing final verification %s", name, part)
			}
		}
	}
	browser := string(read(t, "scripts/verify-browser.sh"))
	for _, suite := range []string{"document-async", "document-foundations", "browser", "navigation", "document-read-mode", "database-advanced", "operations", "automation-browser", "notification-browser", "inbound-capture-browser", "storage-browser", "migration-browser", "spaces", "graph", "templates", "tasks", "tasks-html", "inbox", "discussion", "knowledge", "canvas", "plugins", "enterprise-browser", "transfer-browser", "pwa"} {
		if !strings.Contains(browser, suite) {
			t.Fatalf("sequential browser regression missing %s", suite)
		}
	}
	if !strings.Contains(browser, "node tests/regression-shared.mjs") || !strings.Contains(browser, "MADI_BROWSER_STORAGE") {
		t.Fatal("browser regression must use the disposable service and API-configured test storage")
	}
}

func TestMainCIRequiresParallelOfflineImageVerification(t *testing.T) {
	var workflow struct {
		Jobs map[string]struct {
			Needs           any    `yaml:"needs"`
			If              string `yaml:"if"`
			ContinueOnError bool   `yaml:"continue-on-error"`
			Steps           []struct {
				Uses            string            `yaml:"uses"`
				Run             string            `yaml:"run"`
				If              string            `yaml:"if"`
				ContinueOnError bool              `yaml:"continue-on-error"`
				With            map[string]string `yaml:"with"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(read(t, ".github/workflows/ci.yml"), &workflow); err != nil {
		t.Fatal(err)
	}
	testJob, ok := workflow.Jobs["test"]
	if !ok || testJob.If != "" || testJob.ContinueOnError {
		t.Fatal("main CI must retain its complete required test job")
	}
	job, ok := workflow.Jobs["offline-image"]
	if !ok || job.Needs != nil || job.If != "" || job.ContinueOnError {
		t.Fatal("offline image verification must be an unconditional required parallel job")
	}
	checkout, goSetup, build, verify := -1, -1, -1, -1
	for n, step := range job.Steps {
		if step.If != "" || step.ContinueOnError {
			t.Fatal("offline image steps must not skip or ignore verification failures")
		}
		if strings.HasPrefix(step.Uses, "actions/checkout@") {
			checkout = n
			if step.With["ref"] != "" {
				t.Fatal("offline image must use the same event commit as the tests")
			}
		}
		if strings.HasPrefix(step.Uses, "actions/setup-go@") && step.With["go-version-file"] == "go.mod" {
			goSetup = n
		}
		if strings.TrimSpace(step.Run) == "docker build -t madi:ci ." {
			build = n
		}
		if strings.TrimSpace(step.Run) == "bash scripts/verify-image.sh madi:ci" {
			verify = n
		}
	}
	if checkout < 0 || goSetup <= checkout || build <= goSetup || verify <= build {
		t.Fatal("offline image job must check out, prepare Go, build and verify in order")
	}
}
