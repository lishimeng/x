package s3

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"testing"
)

func TestRegistry_FsFactory(t *testing.T) {
	root := Region(filepath.Join(t.TempDir(), "uploads"))
	reg := NewRegistry()
	reg.RegisterFsRoot(root)
	gw, err := NewGateway(reg.Factory())
	if err != nil {
		t.Fatal(err)
	}
	p, err := gw.Provider(Fs, root)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Save(context.Background(), "a/b.txt", bytes.NewReader([]byte("hi")))
	if err != nil {
		t.Fatal(err)
	}
	err = p.Open(context.Background(), "a/b.txt", func(r io.Reader) error {
		_, e := io.ReadAll(r)
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRegistry_FsRootNotRegistered(t *testing.T) {
	reg := NewRegistry()
	_, err := reg.Factory()(Fs, Region("/tmp/unregistered"))
	if err != ErrFsRootNotAllowed {
		t.Fatalf("want ErrFsRootNotAllowed, got %v", err)
	}
}
