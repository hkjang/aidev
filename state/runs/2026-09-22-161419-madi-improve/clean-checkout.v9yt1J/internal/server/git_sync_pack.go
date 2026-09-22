package server

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/packfile"
	"github.com/go-git/go-git/v5/storage/memory"
)

const gitSyncMaxPackBytes int64 = 64 << 20
const gitSyncMaxObjectBytes int64 = 50 << 20
const gitSyncMaxInflatedBytes int64 = 128 << 20
const gitSyncMaxObjects uint32 = 10000
const gitSyncMaxDeltas = 256

var errGitSyncLimit = errors.New("Git 저장소의 pack·객체·압축 해제 한도를 초과했습니다")

type gitContextSeeker struct {
	ctx context.Context
	io.ReadSeeker
}

func (r gitContextSeeker) Read(p []byte) (int, error) {
	if e := r.ctx.Err(); e != nil {
		return 0, e
	}
	return r.ReadSeeker.Read(p)
}
func (r gitContextSeeker) Seek(offset int64, whence int) (int64, error) {
	if e := r.ctx.Err(); e != nil {
		return 0, e
	}
	return r.ReadSeeker.Seek(offset, whence)
}

type gitLimitWriter struct {
	ctx       context.Context
	out       io.Writer
	remaining int64
}

func (w *gitLimitWriter) Write(p []byte) (int, error) {
	if e := w.ctx.Err(); e != nil {
		return 0, e
	}
	if int64(len(p)) > w.remaining {
		return 0, errGitSyncLimit
	}
	n, e := w.out.Write(p)
	w.remaining -= int64(n)
	return n, e
}

// Validate the declared AND actual inflated bytes before go-git allocates
// memory objects or applies deltas. v5.19.2 additionally verifies delta copy
// bounds and exact target length; this preflight caps that target allocation.
// An untrusted pack is never passed directly to the library's object parser.
func gitSyncPreflightPack(ctx context.Context, f *os.File) error {
	stat, e := f.Stat()
	if e != nil {
		return e
	}
	if stat.Size() > gitSyncMaxPackBytes {
		return errGitSyncLimit
	}
	if _, e = f.Seek(0, io.SeekStart); e != nil {
		return e
	}
	scanner := packfile.NewScanner(gitContextSeeker{ctx, f})
	_, count, e := scanner.Header()
	if e != nil {
		return errors.New("Git pack 헤더가 올바르지 않습니다")
	}
	if count > gitSyncMaxObjects {
		return errGitSyncLimit
	}
	var expanded int64
	deltas := 0
	for i := uint32(0); i < count; i++ {
		header, e := scanner.NextObjectHeader()
		if e != nil {
			return errors.New("Git 객체 헤더가 올바르지 않습니다")
		}
		if header.Length < 0 || header.Length > gitSyncMaxObjectBytes {
			return errGitSyncLimit
		}
		expanded += header.Length
		if expanded > gitSyncMaxInflatedBytes {
			return errGitSyncLimit
		}
		delta := header.Type == plumbing.OFSDeltaObject || header.Type == plumbing.REFDeltaObject
		if !delta && header.Type != plumbing.CommitObject && header.Type != plumbing.TreeObject && header.Type != plumbing.BlobObject && header.Type != plumbing.TagObject {
			return errors.New("지원하지 않는 Git 객체 종류입니다")
		}
		var buf bytes.Buffer
		var sink io.Writer = io.Discard
		if delta {
			deltas++
			if deltas > gitSyncMaxDeltas {
				return errGitSyncLimit
			}
			sink = &buf
		}
		writer := &gitLimitWriter{ctx: ctx, out: sink, remaining: header.Length}
		if _, _, e = scanner.NextObject(writer); e != nil {
			return errors.New("Git 객체 압축 데이터가 올바르지 않거나 한도를 초과했습니다")
		}
		if writer.remaining != 0 {
			return errors.New("Git 객체의 실제 크기가 선언과 다릅니다")
		}
		if delta {
			reader := bytes.NewReader(buf.Bytes())
			source, e := binary.ReadUvarint(reader)
			if e != nil || source > uint64(gitSyncMaxObjectBytes) {
				return errGitSyncLimit
			}
			target, e := binary.ReadUvarint(reader)
			if e != nil || target > uint64(gitSyncMaxObjectBytes) {
				return errGitSyncLimit
			}
			expanded += int64(target)
			if expanded > gitSyncMaxInflatedBytes {
				return errGitSyncLimit
			}
		}
	}
	if _, e = scanner.Checksum(); e != nil {
		return errors.New("Git pack 체크섬이 일치하지 않습니다")
	}
	_, e = f.Seek(0, io.SeekStart)
	return e
}

func gitSyncDecodePack(ctx context.Context, f *os.File) (*memory.Storage, error) {
	if e := gitSyncPreflightPack(ctx, f); e != nil {
		return nil, e
	}
	store := memory.NewStorage()
	parser, e := packfile.NewParserWithStorage(packfile.NewScanner(gitContextSeeker{ctx, f}), store)
	if e != nil {
		return nil, e
	}
	if _, e = parser.Parse(); e != nil {
		return nil, errors.New("검증된 Git 객체를 해석하지 못했습니다")
	}
	return store, nil
}
