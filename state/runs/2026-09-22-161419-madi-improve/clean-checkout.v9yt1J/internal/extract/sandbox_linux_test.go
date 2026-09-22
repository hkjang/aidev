//go:build linux && (amd64 || arm64)

package extract

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "--sandbox-test" {
		if len(os.Args) != 4 {
			os.Exit(2)
		}
		self, _ := os.Executable()
		if e := enterSandbox(os.Args[2], self); e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(3)
		}
		outside := os.Args[3]
		if _, e := os.ReadFile(outside); e == nil {
			os.Exit(10)
		}
		if e := os.WriteFile(outside, []byte("changed"), 0600); e == nil {
			os.Exit(11)
		}
		if e := os.Truncate(outside, 0); e == nil {
			os.Exit(12)
		}
		if _, e := os.ReadFile(fmt.Sprintf("/proc/%d/environ", os.Getppid())); e == nil {
			os.Exit(13)
		}
		if fd, e := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, 0); e == nil {
			unix.Close(fd)
			os.Exit(14)
		}
		if fd, e := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM, 0); e == nil {
			unix.Close(fd)
			os.Exit(15)
		}
		if e := os.WriteFile(filepath.Join(os.Args[2], "output"), []byte("scoped result"), 0600); e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(16)
		}
		fmt.Println("filesystem/read-write-truncate/proc/network blocked; scoped output allowed")
		os.Exit(0)
	}
	if handled, code := RunWorker(os.Args[1:]); handled {
		os.Exit(code)
	}
	os.Exit(m.Run())
}

func TestAttachmentSandboxDeniedEscapes(t *testing.T) {
	abi, _, e := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET, 0, 0, unix.LANDLOCK_CREATE_RULESET_VERSION)
	if e != 0 || abi < 3 {
		t.Skipf("host cannot enable native extraction: Landlock ABI=%d error=%v", abi, e)
	}
	base := t.TempDir()
	dir, e1 := os.MkdirTemp(base, "madi-extract-")
	if e1 != nil {
		t.Fatal(e1)
	}
	outside := filepath.Join(base, "private")
	if e := os.WriteFile(outside, []byte("kept"), 0600); e != nil {
		t.Fatal(e)
	}
	self, e1 := os.Executable()
	if e1 != nil {
		t.Fatal(e1)
	}
	cmd := exec.Command(self, "--sandbox-test", dir, outside)
	cmd.Env = []string{"GOMAXPROCS=2"}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("sandbox: %v\n%s", err, output)
	}
	value, err := os.ReadFile(outside)
	if err != nil || string(value) != "kept" {
		t.Fatalf("outside mutated: %q %v", value, err)
	}
	if !strings.Contains(string(output), "scoped output allowed") {
		t.Fatalf("probe output: %s", output)
	}
	t.Logf("Landlock ABI%d + seccomp verified: %s", abi, output)
}

func TestAttachmentWorkerRejectsPathAndArgumentInjection(t *testing.T) {
	dir, e := os.MkdirTemp(t.TempDir(), "madi-extract-")
	if e != nil {
		t.Fatal(e)
	}
	if e = CopySource(dir, strings.NewReader("%PDF-1.4"), 50<<20); e != nil {
		t.Fatal(e)
	}
	for _, op := range []string{"/bin/sh", "pdf-info;id", "--help"} {
		if _, _, e := workerCommand(dir, op, 1); e == nil {
			t.Fatalf("operation accepted: %s", op)
		}
	}
	if _, _, e := workerCommand(dir, "pdf-text", 501); e == nil {
		t.Fatal("unbounded page accepted")
	}
	link := filepath.Join(t.TempDir(), "madi-extract-link")
	if e = os.Symlink(dir, link); e != nil {
		t.Fatal(e)
	}
	if _, _, e := workerCommand(link, "pdf-info", 0); e == nil {
		t.Fatal("symlink directory accepted")
	}
	if e = os.Chmod(dir, 0755); e != nil {
		t.Fatal(e)
	}
	if _, _, e := workerCommand(dir, "pdf-info", 0); e == nil {
		t.Fatal("public staging directory accepted")
	}
}
