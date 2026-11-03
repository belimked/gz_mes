package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocal(t *testing.T) {
	root := t.TempDir()
	s, e := NewLocal(root)
	if e != nil {
		t.Fatal(e)
	}
	key, sha, n, e := s.Put(strings.NewReader("drawing content"))
	if e != nil {
		t.Fatal(e)
	}
	expected := sha256.Sum256([]byte("drawing content"))
	if sha != hex.EncodeToString(expected[:]) || n != 15 {
		t.Fatalf("hash/size %s %d", sha, n)
	}
	if _, e = os.Stat(filepath.Join(root, filepath.FromSlash(key))); e != nil {
		t.Fatal(e)
	}
	f, e := s.Open(key)
	if e != nil {
		t.Fatal(e)
	}
	b, e := io.ReadAll(f)
	_ = f.Close()
	if e != nil || string(b) != "drawing content" {
		t.Fatalf("read %q %v", b, e)
	}
	if _, e = s.Open("../../etc/passwd"); e == nil {
		t.Fatal("path traversal accepted")
	}
	if e = s.Delete(key); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Open(key); !os.IsNotExist(e) {
		t.Fatalf("deleted file: %v", e)
	}
}
