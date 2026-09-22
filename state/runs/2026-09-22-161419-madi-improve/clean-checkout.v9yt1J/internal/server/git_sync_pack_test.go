package server

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/sha1"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/packfile"
	"github.com/go-git/go-git/v5/storage/memory"
)

func gitPackTestFile(t *testing.T, data []byte) *os.File {
	t.Helper()
	name := filepath.Join(t.TempDir(), "bounded.pack")
	if e := os.WriteFile(name, data, 0600); e != nil {
		t.Fatal(e)
	}
	f, e := os.Open(name)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { f.Close() })
	return f
}
func gitMalformedTestPack(kind byte, declared int64, body []byte, count uint32) []byte {
	var out bytes.Buffer
	out.WriteString("PACK")
	binary.Write(&out, binary.BigEndian, uint32(2))
	binary.Write(&out, binary.BigEndian, count)
	first := byte(declared&15) | (kind << 4)
	declared >>= 4
	if declared > 0 {
		first |= 128
	}
	out.WriteByte(first)
	for declared > 0 {
		b := byte(declared & 127)
		declared >>= 7
		if declared > 0 {
			b |= 128
		}
		out.WriteByte(b)
	}
	if kind == 7 {
		out.Write(make([]byte, 20))
	}
	z := zlib.NewWriter(&out)
	z.Write(body)
	z.Close()
	sum := sha1.Sum(out.Bytes())
	out.Write(sum[:])
	return out.Bytes()
}

func TestGitSyncPackBoundedInflationAndRoundtrip(t *testing.T) {
	store := memory.NewStorage()
	obj := store.NewEncodedObject()
	obj.SetType(plumbing.BlobObject)
	data := []byte("# 안전한 Markdown\n\nGit roundtrip\n")
	obj.SetSize(int64(len(data)))
	writer, _ := obj.Writer()
	writer.Write(data)
	writer.Close()
	hash, e := store.SetEncodedObject(obj)
	if e != nil {
		t.Fatal(e)
	}
	var pack bytes.Buffer
	if _, e = packfile.NewEncoder(&pack, store, false).Encode([]plumbing.Hash{hash}, 0); e != nil {
		t.Fatal(e)
	}
	decoded, e := gitSyncDecodePack(context.Background(), gitPackTestFile(t, pack.Bytes()))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = decoded.EncodedObject(plumbing.BlobObject, hash); e != nil {
		t.Fatal("valid pack lost object", e)
	}
	var delta bytes.Buffer
	var buf [10]byte
	n := binary.PutUvarint(buf[:], 1)
	delta.Write(buf[:n])
	n = binary.PutUvarint(buf[:], uint64(gitSyncMaxObjectBytes+1))
	delta.Write(buf[:n])
	tests := map[string][]byte{
		"declared_oversized": gitMalformedTestPack(3, gitSyncMaxObjectBytes+1, []byte("small"), 1),
		"inflated_overrun":   gitMalformedTestPack(3, 1, bytes.Repeat([]byte("a"), 1<<20), 1),
		"object_count":       gitMalformedTestPack(3, 1, []byte("a"), gitSyncMaxObjects+1),
		"delta_target":       gitMalformedTestPack(7, int64(delta.Len()), delta.Bytes(), 1),
	}
	corrupt := append([]byte{}, pack.Bytes()...)
	corrupt[len(corrupt)-1] ^= 1
	tests["checksum"] = corrupt
	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			if _, e := gitSyncDecodePack(context.Background(), gitPackTestFile(t, data)); e == nil {
				t.Fatal("unbounded/malformed pack accepted")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := gitSyncDecodePack(ctx, gitPackTestFile(t, pack.Bytes())); e == nil {
		t.Fatal("cancelled parse accepted")
	}
}
