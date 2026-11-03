// Package storage provides an interchangeable file store for uploaded drawings.
package storage

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Files interface {
	Put(io.Reader) (key, sha string, size int64, err error)
	Open(string) (io.ReadCloser, error)
	Delete(string) error
}

type Local struct{ root string }

func NewLocal(root string) (*Local, error) {
	if err := os.MkdirAll(root, 0750); err != nil {
		return nil, err
	}
	// Check write access at startup, not only on the first upload.
	f, err := os.CreateTemp(root, ".write-check-")
	if err != nil {
		return nil, err
	}
	name := f.Name()
	err = f.Close()
	_ = os.Remove(name)
	if err != nil {
		return nil, err
	}
	return &Local{root: root}, nil
}

func (l *Local) path(key string) (string, error) {
	parts := strings.Split(key, "/")
	if len(parts) != 3 || len(parts[0]) != 4 || len(parts[1]) != 2 || len(parts[2]) != 32 {
		return "", errors.New("invalid storage key")
	}
	for _, p := range parts {
		for _, c := range p {
			if (c < '0' || c > '9') && (c < 'a' || c > 'f' || len(p) != 32) {
				return "", errors.New("invalid storage key")
			}
		}
	}
	return filepath.Join(l.root, filepath.FromSlash(key)), nil
}
func (l *Local) Put(r io.Reader) (string, string, int64, error) {
	now := time.Now().UTC()
	dir := filepath.Join(l.root, fmt.Sprintf("%04d", now.Year()), fmt.Sprintf("%02d", now.Month()))
	if err := os.MkdirAll(dir, 0750); err != nil {
		return "", "", 0, err
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", "", 0, err
	}
	key := fmt.Sprintf("%04d/%02d/%s", now.Year(), now.Month(), hex.EncodeToString(b[:]))
	tmp, err := os.CreateTemp(dir, ".upload-")
	if err != nil {
		return "", "", 0, err
	}
	defer os.Remove(tmp.Name())
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, h), r)
	if err == nil {
		err = tmp.Sync()
	}
	if e := tmp.Close(); err == nil {
		err = e
	}
	if err != nil {
		return "", "", 0, err
	}
	dest, _ := l.path(key)
	if err = os.Rename(tmp.Name(), dest); err != nil {
		return "", "", 0, err
	}
	return key, hex.EncodeToString(h.Sum(nil)), n, nil
}
func (l *Local) Open(key string) (io.ReadCloser, error) {
	p, e := l.path(key)
	if e != nil {
		return nil, e
	}
	return os.Open(p) // #nosec G304 -- path() validates every component before joining with the configured root.
}
func (l *Local) Delete(key string) error {
	p, e := l.path(key)
	if e != nil {
		return e
	}
	return os.Remove(p)
}
