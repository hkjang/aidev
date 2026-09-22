package extract

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const workerFlag = "--madi-attachment-worker"

type cappedOutput struct {
	data  []byte
	limit int
}

func (c *cappedOutput) Write(p []byte) (int, error) {
	if len(c.data)+len(p) > c.limit {
		return 0, ErrLimit
	}
	c.data = append(c.data, p...)
	return len(p), nil
}

// Run starts the same service executable with a fixed internal operation. No
// administrator setting or uploaded filename can become a program or flag.
func Run(ctx context.Context, jobDir, operation string, page int) ([]byte, error) {
	self, e := os.Executable()
	if e != nil {
		return nil, e
	}
	return runWorker(ctx, self, jobDir, operation, page)
}

func runWorker(ctx context.Context, self, jobDir, operation string, page int) ([]byte, error) {
	if _, _, e := workerCommand(jobDir, operation, page); e != nil {
		return nil, e
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, self, workerFlag, operation, jobDir, strconv.Itoa(page))
	// Explicit allowlist: database DSN, encryption key, bootstrap credentials,
	// cloud tokens, proxy settings and user HOME are never inherited.
	cmd.Env = []string{"LANG=C.UTF-8", "LC_ALL=C.UTF-8", "TZ=UTC", "OMP_THREAD_LIMIT=1", "OMP_NUM_THREADS=1", "GOMEMLIMIT=256MiB", "GOMAXPROCS=2"}
	cmd.Dir = jobDir
	cmd.WaitDelay = 2 * time.Second
	output := &cappedOutput{limit: 16 << 20}
	diagnostic := &cappedOutput{limit: 16 << 10}
	cmd.Stdout = output
	cmd.Stderr = diagnostic
	if e := cmd.Run(); e != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		// Upstream errors can contain PDF metadata or local paths. Keep them out
		// of API responses and audit logs; the policy UI exposes a fixed reason.
		return nil, errors.New("격리된 첨부 추출에 실패했습니다. 형식·암호화 여부·자원 제한·격리 진단을 확인하세요")
	}
	return output.data, nil
}

// RunWorker must be called at the very beginning of main, before environment
// validation or opening PostgreSQL. The service environment is not required.
func RunWorker(args []string) (bool, int) {
	if len(args) == 0 || args[0] != workerFlag {
		return false, 0
	}
	if len(args) != 4 {
		return true, 2
	}
	page, e := strconv.Atoi(args[3])
	if e != nil {
		return true, 2
	}
	program, argv, e := workerCommand(args[2], args[1], page)
	if e != nil {
		return true, 2
	}
	if e = enterSandbox(args[2], program); e != nil {
		fmt.Fprintln(os.Stderr, "attachment sandbox unavailable")
		return true, 3
	}
	if e = replaceProcess(program, argv, []string{"LANG=C.UTF-8", "LC_ALL=C.UTF-8", "TZ=UTC", "OMP_THREAD_LIMIT=1", "OMP_NUM_THREADS=1", "HOME=" + args[2], "TMPDIR=" + args[2]}); e != nil {
		return true, 4
	}
	return true, 0
}

func workerCommand(jobDir, operation string, page int) (string, []string, error) {
	if !filepath.IsAbs(jobDir) || filepath.Clean(jobDir) != jobDir || !strings.HasPrefix(filepath.Base(jobDir), "madi-extract-") {
		return "", nil, errors.New("추출 임시 경로가 올바르지 않습니다")
	}
	info, e := os.Lstat(jobDir)
	if e != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		return "", nil, errors.New("추출 임시 경로는 전용 0700 디렉터리여야 합니다")
	}
	real, e := filepath.EvalSymlinks(jobDir)
	if e != nil || real != jobDir {
		return "", nil, errors.New("추출 임시 경로에 심볼릭 링크를 사용할 수 없습니다")
	}
	source := filepath.Join(jobDir, "source")
	stat, e := os.Lstat(source)
	if e != nil || !stat.Mode().IsRegular() || stat.Size() > 50<<20 {
		return "", nil, errors.New("추출 원본 파일을 확인하세요")
	}
	if page < 0 || page > 500 {
		return "", nil, ErrLimit
	}
	switch operation {
	case "pdf-info":
		return "/usr/bin/pdfinfo", []string{"pdfinfo", "-enc", "UTF-8", source}, nil
	case "pdf-text":
		if page < 1 {
			return "", nil, ErrLimit
		}
		n := strconv.Itoa(page)
		return "/usr/bin/pdftotext", []string{"pdftotext", "-f", n, "-l", n, "-bbox-layout", "-enc", "UTF-8", source, "-"}, nil
	case "pdf-render":
		if page < 1 {
			return "", nil, ErrLimit
		}
		n := strconv.Itoa(page)
		return "/usr/bin/pdftoppm", []string{"pdftoppm", "-f", n, "-l", n, "-singlefile", "-scale-to", "2400", "-png", source, filepath.Join(jobDir, "page")}, nil
	case "ocr":
		image := filepath.Join(jobDir, "page.png")
		stat, e := os.Lstat(image)
		if e != nil || !stat.Mode().IsRegular() || stat.Size() > 32<<20 {
			return "", nil, ErrLimit
		}
		// The bundled kor model requests chi_tra as an auxiliary language by
		// default. Limit auxiliary loading to our bundled English data as well;
		// otherwise it logs a missing external model even though kor succeeds.
		return "/usr/bin/tesseract", []string{"tesseract", image, "stdout", "-l", "eng+kor", "--oem", "1", "--psm", "3", "-c", "tessedit_load_sublangs=eng", "tsv"}, nil
	default:
		return "", nil, errors.New("지원하지 않는 추출 작업입니다")
	}
}

// CopySource limits source bytes before an untrusted native parser starts.
func CopySource(dir string, source io.Reader, max int64) error {
	f, e := os.OpenFile(filepath.Join(dir, "source"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	defer f.Close()
	n, e := io.Copy(f, io.LimitReader(source, max+1))
	if e != nil {
		return e
	}
	if n > max {
		return ErrLimit
	}
	return f.Sync()
}
